package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type Component struct {
	Version string            `json:"version"`
	Files   map[string]string `json:"files"`
}
type Bundle struct {
	Version    string               `json:"version"`
	Components map[string]Component `json:"components"`
}

func safePath(root, rel string) (string, error) {
	if rel == "" || strings.ContainsAny(rel, "\\:\x00") || strings.HasPrefix(rel, "/") {
		return "", fmt.Errorf("invalid bundle path: %s", rel)
	}
	for _, p := range strings.Split(rel, "/") {
		if p == "" || p == "." || p == ".." {
			return "", fmt.Errorf("invalid bundle path: %s", rel)
		}
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	path := absolute
	for _, part := range strings.Split(rel, "/") {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlinks are not managed: %s", path)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	return path, nil
}
func managedName(name string) bool {
	for _, prefix := range []string{"manager/", "scripts/", "plugins/", "defaults/", "docs/"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	switch name {
	case "restore.sh", "restore.bat", "restore-tui.sh", "restore-tui.bat", "catalog.json", "manage.sh", "manage.cmd", "launch-installer.bat", "uninstall.sh", "VERSION", "README.md", "install-omp.sh", "install-omp.bat", "install-omp-tui.sh", "install-omp-tui.bat", "install-omp-tui.ps1":
		return true
	}
	return false
}
func loadBundle(root string) (Bundle, error) {
	var b Bundle
	if err := readJSON(filepath.Join(root, "bundle.json"), &b); err != nil {
		return b, err
	}
	if b.Version == "" || len(b.Components) == 0 {
		return b, errors.New("empty bundle manifest")
	}
	seen := map[string]bool{}
	for _, c := range b.Components {
		for path, hash := range c.Files {
			if !managedName(path) || seen[strings.ToLower(path)] {
				return b, fmt.Errorf("invalid or duplicate ownership: %s", path)
			}
			seen[strings.ToLower(path)] = true
			if _, err := safePath(root, path); err != nil {
				return b, err
			}
			if raw, err := hex.DecodeString(hash); err != nil || len(raw) != 32 {
				return b, fmt.Errorf("invalid sha256 for %s", path)
			}
		}
	}
	return b, nil
}
func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".omp-write-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err = temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Chmod(mode); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

type previousFile struct {
	data   []byte
	mode   os.FileMode
	exists bool
}

func applyBundle(source, destination string, selected []string) error {
	source, _ = filepath.Abs(source)
	destination, _ = filepath.Abs(destination)
	// Run the updater from the newly extracted bundle, outside the destination.
	exe, _ := os.Executable()
	rel, _ := filepath.Rel(destination, exe)
	if source == destination || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
		return errors.New("run bundle apply from a separate extracted bundle, not the installed manager")
	}
	if info, err := os.Lstat(destination); err == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
		return errors.New("destination must be a real directory")
	}
	next, err := loadBundle(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destination, 0755); err != nil {
		return err
	}
	unlock, err := lockBundle(destination)
	if err != nil {
		return err
	}
	defer unlock()
	old, err := loadBundle(destination)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if old.Components == nil {
		entries, e := os.ReadDir(destination)
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		marker, _ := os.ReadFile(filepath.Join(destination, ".omp-harness-package"))
		if len(entries) > 1 && strings.TrimSpace(string(marker)) != "omp-harness-script-installer" {
			return errors.New("choose an empty destination or an existing package installation")
		}
		old = Bundle{Components: map[string]Component{}}
	}
	legacy := old.Version == ""
	if legacy && len(selected) > 0 {
		return errors.New("first installation or legacy migration requires a full bundle apply")
	}
	full := len(selected) == 0
	if full {
		for id := range next.Components {
			selected = append(selected, id)
		}
		for id := range old.Components {
			if _, ok := next.Components[id]; !ok {
				selected = append(selected, id)
			}
		}
	}
	wanted := map[string]bool{}
	for _, id := range selected {
		if _, ok := next.Components[id]; !ok && !full {
			return fmt.Errorf("unknown component: %s", id)
		}
		wanted[id] = true
	}
	merged := Bundle{Version: old.Version, Components: map[string]Component{}}
	for id, c := range old.Components {
		merged.Components[id] = c
	}
	if full {
		merged.Version = next.Version
	}
	files := map[string][]byte{}
	modes := map[string]os.FileMode{}
	remove := map[string]bool{}
	for id := range wanted {
		for p := range old.Components[id].Files {
			remove[p] = true
		}
		c, ok := next.Components[id]
		if !ok {
			delete(merged.Components, id)
			continue
		}
		for p, expected := range c.Files {
			// A component may not overwrite files owned by an unselected component.
			for oldID, oldC := range old.Components {
				if !wanted[oldID] {
					for oldPath := range oldC.Files {
						if strings.EqualFold(oldPath, p) {
							return fmt.Errorf("ownership conflict: %s", p)
						}
					}
				}
			}
			path, e := safePath(source, p)
			if e != nil {
				return e
			}
			data, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			hash := sha256.Sum256(data)
			if hex.EncodeToString(hash[:]) != strings.ToLower(expected) {
				return fmt.Errorf("checksum mismatch: %s", p)
			}
			files[p] = data
			modes[p] = 0644
			if strings.HasSuffix(p, ".sh") || strings.HasPrefix(p, "manager/manage-") {
				modes[p] = 0755
			}
			delete(remove, p)
		}
		merged.Components[id] = c
	}
	for p := range files {
		delete(remove, p)
	}
	// All selected input is validated before any destination file is changed.
	manifest, _ := json.MarshalIndent(merged, "", "  ")
	files["bundle.json"] = append(manifest, '\n')
	modes["bundle.json"] = 0644
	files[".omp-harness-package"] = []byte("omp-harness-script-installer\n")
	modes[".omp-harness-package"] = 0644
	backups := map[string]previousFile{}
	paths := map[string]bool{}
	for p := range files {
		paths[p] = true
	}
	for p := range remove {
		paths[p] = true
	}
	order := []string{}
	for p := range paths {
		path, e := safePath(destination, p)
		if e != nil {
			return e
		}
		info, e := os.Stat(path)
		previous := previousFile{}
		if e == nil {
			if !legacy && managedName(p) {
				owned := false
				for _, c := range old.Components {
					for oldPath := range c.Files {
						if strings.EqualFold(oldPath, p) {
							owned = true
						}
					}
				}
				if !owned {
					return fmt.Errorf("refusing to overwrite a user file: %s", p)
				}
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("not a regular file: %s", p)
			}
			previous.data, e = os.ReadFile(path)
			if e != nil {
				return e
			}
			previous.mode = info.Mode().Perm()
			previous.exists = true
		} else if !errors.Is(e, os.ErrNotExist) {
			return e
		}
		backups[p] = previous
		order = append(order, p)
	}
	sort.Strings(order)
	changed := []string{}
	rollback := func(cause error) error {
		for i := len(changed) - 1; i >= 0; i-- {
			p := changed[i]
			prev := backups[p]
			path := filepath.Join(destination, filepath.FromSlash(p))
			var e error
			if prev.exists {
				e = atomicWrite(path, prev.data, prev.mode)
			} else {
				e = os.Remove(path)
				if errors.Is(e, os.ErrNotExist) {
					e = nil
				}
			}
			if e != nil {
				cause = errors.Join(cause, fmt.Errorf("restore %s: %w", p, e))
			}
		}
		return cause
	}
	for _, p := range order {
		path := filepath.Join(destination, filepath.FromSlash(p))
		if remove[p] {
			err = os.Remove(path)
			if errors.Is(err, os.ErrNotExist) {
				err = nil
			}
		} else {
			err = atomicWrite(path, files[p], modes[p])
		}
		if err != nil {
			return rollback(err)
		}
		changed = append(changed, p)
	}
	fmt.Printf("Bundle files applied to %s. No item scripts were run.\n", destination)
	return nil
}
func lockBundle(root string) (func(), error) {
	path := filepath.Join(root, ".bundle.lock")
	lock, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("bundle operation locked: %w", err)
	}
	lock.Close()
	var once sync.Once
	return func() { once.Do(func() { os.Remove(path) }) }, nil
}
func removeBundle(root string) error {
	unlock, err := lockBundle(root)
	if err != nil {
		return err
	}
	defer unlock()
	b, err := loadBundle(root)
	if err != nil {
		return err
	}
	for _, c := range b.Components {
		for p := range c.Files {
			path, e := safePath(root, p)
			if e != nil {
				return e
			}
			if e = os.Remove(path); e != nil && !errors.Is(e, os.ErrNotExist) {
				return e
			}
		}
	}
	for _, p := range []string{"bundle.json", ".omp-harness-package"} {
		if err := os.Remove(filepath.Join(root, p)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	unlock()
	// rmdir only: never recursively delete user files in managed directories.
	dirs := []string{}
	filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e == nil && d.IsDir() {
			dirs = append(dirs, p)
		}
		return nil
	})
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	for _, p := range dirs {
		os.Remove(p)
	}
	fmt.Println("Package removed; user state, settings and installed products retained.")
	return nil
}
