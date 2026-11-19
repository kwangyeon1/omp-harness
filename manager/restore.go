package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"unicode/utf8"
)

// Saved environments own portable files, not OS-specific installations.
type SavedItem struct {
	HomeAliases []string          `json:"homeAliases,omitempty"`
	HomeOrigin  string            `json:"homeOrigin,omitempty"`
	Kind        string            `json:"kind"`
	Release     string            `json:"release"`
	Files       map[string]string `json:"files"`
}
type Environment struct {
	Version int                  `json:"version"`
	Items   map[string]SavedItem `json:"items"`
}

var itemID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

func loadEnvironment(root string) (Environment, error) {
	var e Environment
	err := readJSON(filepath.Join(root, "environment.json"), &e)
	if err != nil {
		return e, err
	}
	if e.Version != 1 || len(e.Items) == 0 {
		return e, errors.New("unsupported or empty environment manifest")
	}
	for id, item := range e.Items {
		if !itemID.MatchString(id) || (item.Kind != "omp" && item.Kind != "module") || len(item.Files) == 0 || (item.Kind == "omp" && (id != "omp" || !releaseTag.MatchString(item.Release))) {
			return e, errors.New("invalid environment item")
		}
	}
	return e, nil
}
func excluded(name string) bool {
	name = strings.ToLower(name)
	switch name {
	case ".git", "node_modules", ".npmrc", ".netrc", ".pnpm-store", ".venv", "__pycache__":
		return true
	}
	for _, prefix := range []string{".env", "auth.json", "credentials.", ".credentials."} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	if strings.Contains(name, ".db") || strings.Contains(name, ".sqlite") {
		return true
	}
	for _, suffix := range []string{".db", ".db-wal", ".db-shm", ".db-journal", ".sqlite", ".sqlite-wal", ".sqlite-shm", ".sqlite3", ".key", ".pem"} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}
func copyRegular(from, to string) error {
	info, err := os.Lstat(from)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("only regular portable files supported: %s", from)
	}
	if err = os.MkdirAll(filepath.Dir(to), 0700); err != nil {
		return err
	}
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	mode := os.FileMode(0600)
	if info.Mode()&0111 != 0 {
		mode = 0700
	}
	out, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func copyPortable(from, to string, home bool) error {
	info, err := os.Lstat(from)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("portable input must be a real directory")
	}
	if err := os.MkdirAll(to, 0700); err != nil {
		return err
	}
	return filepath.WalkDir(from, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, p)
		if rel == "." {
			return nil
		}
		skip := excluded(d.Name()) || (!home && rel == "restored.json")
		if home && transientHomeName(d.Name()) {
			skip = true
		}
		if skip {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		dest, err := safePath(to, filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		return copyRegular(p, dest)
	})
}
func inside(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Resolve existing ancestors too, so a symlink cannot bypass containment checks.
func canonicalPath(p string) (string, error) {
	p, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	var missing []string
	for {
		_, err = os.Lstat(p)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		missing = append(missing, filepath.Base(p))
		p = filepath.Dir(p)
	}
	p, err = filepath.EvalSymlinks(p)
	if err != nil {
		return "", err
	}
	for i := len(missing) - 1; i >= 0; i-- {
		p = filepath.Join(p, missing[i])
	}
	return p, nil
}
func hashFile(p string) (string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	// Git may convert line endings on a different OS. Text hashes use LF;
	// binary hashes remain byte-exact.
	if utf8.Valid(b) && !bytes.ContainsRune(b, 0) {
		b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// Stage a new item and publish its manifest only when all files have been copied.
// Existing items are deliberately never overwritten: snapshots are explicit versions.
func saveItem(environment, id string, item SavedItem, inputs []string, fill func(string) error) error {
	if !itemID.MatchString(id) {
		return errors.New("invalid item ID")
	}
	root, err := canonicalPath(environment)
	if err != nil {
		return err
	}
	for _, input := range inputs {
		p, err := filepath.Abs(input)
		if err != nil {
			return err
		}
		p, err = filepath.EvalSymlinks(p)
		if err != nil {
			return err
		}
		if inside(p, root) || inside(root, p) {
			return errors.New("snapshot and input directories must not contain each other")
		}
	}
	if _, err = safePath(root, "environment.json"); err != nil {
		return err
	}
	if err = os.MkdirAll(root, 0700); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(root, ".snapshot.lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	lock.Close()
	defer os.Remove(filepath.Join(root, ".snapshot.lock"))
	e := Environment{Version: 1, Items: map[string]SavedItem{}}
	if _, err = os.Stat(filepath.Join(root, "environment.json")); err == nil {
		e, err = loadEnvironment(root)
		if err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if _, ok := e.Items[id]; ok {
		return errors.New("item already saved; use a new snapshot directory")
	}
	target := filepath.Join(root, id)
	if _, err = os.Lstat(target); !os.IsNotExist(err) {
		return errors.New("item path already exists")
	}
	stage, err := os.MkdirTemp(root, ".snapshot-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if err = fill(stage); err != nil {
		return err
	}
	item.Files = map[string]string{}
	err = filepath.WalkDir(stage, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(stage, p)
		hash, err := hashFile(p)
		if err != nil {
			return err
		}
		item.Files[filepath.ToSlash(rel)] = hash
		return nil
	})
	if err != nil {
		return err
	}
	if len(item.Files) == 0 {
		return errors.New("no portable files found")
	}
	if err = os.Rename(stage, target); err != nil {
		return err
	}
	e.Items[id] = item
	data, _ := json.MarshalIndent(e, "", "  ")
	if err = atomicWrite(filepath.Join(root, "environment.json"), append(data, '\n'), 0600); err != nil {
		os.RemoveAll(target)
		return err
	}
	fmt.Println("Saved", id, "in", root, "— review files before committing; custom configs may contain secrets or external paths.")
	return nil
}
func snapshotModule(id, directory, environment string) error {
	if id == "omp" {
		return errors.New("omp is reserved; use snapshot omp")
	}
	if _, err := os.Stat(filepath.Join(directory, "restore.sh")); err != nil {
		if _, err := os.Stat(filepath.Join(directory, "restore.ps1")); err != nil {
			return errors.New("module needs restore.sh and/or restore.ps1")
		}
	}
	return saveItem(environment, id, SavedItem{Kind: "module"}, []string{directory}, func(stage string) error { return copyPortable(directory, stage, false) })
}
func verifyItem(root, id string, item SavedItem) error {
	seen := map[string]bool{}
	for rel, want := range item.Files {
		key := strings.ToLower(rel)
		if key == "restored.json" || seen[key] {
			return errors.New("reserved or case-colliding snapshot path: " + rel)
		}
		seen[key] = true
		if item.Kind == "omp" && rel != "home-marker" && rel != "plugin-links.json" && !strings.HasPrefix(rel, "linked-plugins/") && !strings.HasPrefix(rel, "home/") {
			return errors.New("OMP snapshot files must be under home/")
		}
		if hash, err := hex.DecodeString(want); err != nil || len(hash) != 32 {
			return errors.New("invalid snapshot hash")
		}
		p, err := safePath(root, id+"/"+rel)
		if err != nil {
			return err
		}
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("nonregular saved file")
		}
		got, err := hashFile(p)
		if err != nil {
			return err
		}
		if got != want {
			return fmt.Errorf("snapshot hash mismatch: %s", rel)
		}
	}
	return nil
}
func runCommand(dir string, env []string, name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Dir = dir
	c.Env = env
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
func restoreItem(project, environment, id, destination string) error {
	root, err := canonicalPath(environment)
	if err != nil {
		return err
	}
	e, err := loadEnvironment(root)
	if err != nil {
		return err
	}
	item, ok := e.Items[id]
	if !ok {
		return errors.New("unknown saved item: " + id)
	}
	if err = verifyItem(root, id, item); err != nil {
		return err
	}
	dest, err := canonicalPath(destination)
	if err != nil {
		return err
	}
	if inside(root, dest) || inside(dest, root) {
		return errors.New("restore destination must be outside the snapshot")
	}
	if _, err = safePath(dest, "restored.json"); err != nil {
		return err
	}
	if _, err = os.Lstat(dest); !os.IsNotExist(err) {
		return errors.New("destination exists; choose a new directory to preserve existing files")
	}
	if item.Kind == "module" {
		entry := "restore.sh"
		if runtime.GOOS == "windows" {
			entry = "restore.ps1"
		}
		if _, ok := item.Files[entry]; !ok {
			return errors.New("module has no restore entry for this OS")
		}
	}
	if err = os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return err
	}
	if err = os.Mkdir(dest, 0700); err != nil {
		return err
	}
	for rel := range item.Files {
		from, _ := safePath(root, id+"/"+rel)
		to, err := safePath(dest, rel)
		if err != nil {
			return err
		}
		if err = copyRegular(from, to); err != nil {
			return err
		}
	}
	if item.Kind == "omp" {
		err = rebaseHome(dest, item)
		if err == nil {
			err = prepareOMP(project, dest, item)
		}
	} else if runtime.GOOS == "windows" {
		err = runCommand(dest, os.Environ(), "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(dest, "restore.ps1"), dest)
	} else {
		err = runCommand(dest, os.Environ(), "sh", filepath.Join(dest, "restore.sh"), dest)
	}
	if err != nil {
		return fmt.Errorf("restore failed; partial files preserved at %s: %w", dest, err)
	}
	data, _ := json.MarshalIndent(item, "", "  ")
	if err = os.WriteFile(filepath.Join(dest, "restored.json"), data, 0600); err != nil {
		return err
	}
	fmt.Println("Restored", id, "at", dest)
	return nil
}
func showEnvironment(root string) error {
	e, err := loadEnvironment(root)
	if err != nil {
		return err
	}
	fmt.Println("Saved environment:", root)
	ids := make([]string, 0, len(e.Items))
	for id := range e.Items {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		item := e.Items[id]
		fmt.Printf("%s | %s | %d files | %s\n", id, item.Kind, len(item.Files), item.Release)
	}
	return nil
}

// Separate restore menu follows the existing select -> parameters -> confirm -> repeat flow.
func restoreMenu(project, environment string) error {
	if environment == "" {
		fmt.Print("Saved environment directory: ")
		var err error
		environment, err = readMenuLine(os.Stdin)
		if err != nil {
			return err
		}
		environment = strings.TrimSpace(environment)
	}
	for {
		if err := showEnvironment(environment); err != nil {
			return err
		}
		fmt.Print("Item ID to restore (Enter to exit): ")
		id, err := readMenuLine(os.Stdin)
		if err != nil {
			return nil
		}
		id = strings.TrimSpace(id)
		if id == "" {
			return nil
		}
		fmt.Print("New runtime directory: ")
		dest, err := readMenuLine(os.Stdin)
		if err != nil {
			return nil
		}
		dest = strings.TrimSpace(dest)
		if dest == "" {
			continue
		}
		fmt.Printf("Restore %s to %s? [y/N]: ", id, dest)
		answer, err := readMenuLine(os.Stdin)
		if err != nil {
			return nil
		}
		if !strings.EqualFold(strings.TrimSpace(answer), "y") {
			continue
		}
		if err = restoreItem(project, environment, id, dest); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
func restoreDispatch(project string, args []string) error {
	if len(args) == 0 {
		return restoreMenu(project, "")
	}
	if len(args) == 1 {
		if args[0] == "--help" {
			fmt.Println("restore-tui [ENVIRONMENT] | restore list ENVIRONMENT | restore run ENVIRONMENT ID NEW_DESTINATION")
			return nil
		}
		return restoreMenu(project, args[0])
	}
	if len(args) == 2 && args[0] == "list" {
		return showEnvironment(args[1])
	}
	if len(args) == 4 && args[0] == "run" {
		return restoreItem(project, args[1], args[2], args[3])
	}
	return errors.New("usage: restore [ENVIRONMENT] | restore list ENVIRONMENT | restore run ENVIRONMENT ID NEW_DESTINATION")
}

// Only known former roots in UTF-8 configuration are relocated. Other external
// paths remain the user's responsibility and are documented for review.
func rebaseHome(dest string, item SavedItem) error {
	return filepath.WalkDir(filepath.Join(dest, "home"), func(p string, d os.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		text := string(data)
		if ext == ".json" {
			var value any
			decoder := json.NewDecoder(bytes.NewReader(data))
			decoder.UseNumber()
			if err := decoder.Decode(&value); err != nil {
				return fmt.Errorf("invalid saved JSON %s: %w", p, err)
			}
			before, _ := json.Marshal(value)
			value = relocateJSONPaths(value, append([]string{item.HomeOrigin}, item.HomeAliases...), filepath.Join(dest, "home"))
			updated, err := json.MarshalIndent(value, "", "  ")
			if err != nil {
				return err
			}
			// Preserve exact original formatting when no path changed.
			after, _ := json.Marshal(value)
			if bytes.Equal(before, after) {
				return nil
			}
			return os.WriteFile(p, append(updated, '\n'), 0600)
		}
		replacements := []string{}
		for _, old := range append([]string{item.HomeOrigin}, item.HomeAliases...) {
			newPath := filepath.Join(dest, "home")
			if old == "" {
				continue
			}
			replacements = append(replacements, old, filepath.ToSlash(newPath), strings.ReplaceAll(old, "\\", "/"), filepath.ToSlash(newPath))
		}
		text = strings.NewReplacer(replacements...).Replace(text)
		return os.WriteFile(p, []byte(text), 0600)
	})
}
