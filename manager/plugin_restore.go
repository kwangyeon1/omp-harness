package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Paths in this file are relative to the portable runtime, on either OS.
type pluginLink struct {
	Root   string `json:"root"`
	Name   string `json:"name"`
	Target string `json:"target"`
}

var packageName = regexp.MustCompile(`^(@[A-Za-z0-9_~-][A-Za-z0-9._~-]*/)?[A-Za-z0-9_~-][A-Za-z0-9._~-]*$`)
var pluginRoot = regexp.MustCompile(`^(profiles/[A-Za-z0-9_-][A-Za-z0-9._-]*/)?plugins$`)

func optionalJSON(path string, value any) error {
	err := readJSON(path, value)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func pluginRoots(home string) ([]string, error) {
	roots := []string{"plugins"}
	profiles, err := os.ReadDir(filepath.Join(home, "profiles"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, p := range profiles {
		if p.IsDir() {
			roots = append(roots, "profiles/"+p.Name()+"/plugins")
		}
	}
	return roots, nil
}

// Snapshot only cache entries referenced by the user's registry. Preserve the
// registry and lock verbatim so disabled plugins and their settings survive.
func snapshotPlugins(home, stage string) error {
	roots, err := pluginRoots(home)
	if err != nil {
		return err
	}
	var links []pluginLink
	for _, root := range roots {
		dir, err := safePath(home, root)
		if err != nil {
			return err
		}
		var installed struct {
			Plugins map[string][]struct{ InstallPath string }
		}
		if err := optionalJSON(filepath.Join(dir, "installed_plugins.json"), &installed); err != nil {
			return err
		}
		market := map[string]string{}
		for id, entries := range installed.Plugins {
			for _, entry := range entries {
				src, rel, err := registeredCachePath(dir, entry.InstallPath, "plugins")
				if err != nil {
					return fmt.Errorf("%s: %w", id, err)
				}
				target := "home/" + root + "/" + rel
				if err := copyPortable(src, filepath.Join(stage, filepath.FromSlash(target)), false); err != nil {
					return err
				}
				var pkg struct{ Name string }
				if err := optionalJSON(filepath.Join(src, "package.json"), &pkg); err != nil {
					return err
				}
				if pkg.Name == "" {
					pkg.Name, _, _ = strings.Cut(id, "@")
				}
				if !packageName.MatchString(pkg.Name) {
					return fmt.Errorf("invalid marketplace package name %q", pkg.Name)
				}
				if _, exists := market[pkg.Name]; exists {
					return fmt.Errorf("ambiguous marketplace package %s", pkg.Name)
				}
				market[pkg.Name] = target
				links = append(links, pluginLink{root, pkg.Name, target})
			}
		}
		var catalogs struct {
			Marketplaces []struct{ CatalogPath string }
		}
		if err := optionalJSON(filepath.Join(filepath.Dir(dir), "marketplaces.json"), &catalogs); err != nil {
			return err
		}
		for _, entry := range catalogs.Marketplaces {
			src, rel, err := registeredCachePath(dir, entry.CatalogPath, "marketplaces")
			if err != nil {
				return err
			}
			if err := copyRegular(src, filepath.Join(stage, "home", root, rel)); err != nil {
				return err
			}
		}

		var lock struct{ Plugins map[string]json.RawMessage }
		var manifest struct{ Dependencies map[string]string }
		if err := optionalJSON(filepath.Join(dir, "omp-plugins.lock.json"), &lock); err != nil {
			return err
		}
		if err := optionalJSON(filepath.Join(dir, "package.json"), &manifest); err != nil {
			return err
		}
		for name := range lock.Plugins {
			if _, ok := market[name]; ok {
				continue
			}
			if _, ok := manifest.Dependencies[name]; ok {
				continue
			}
			if !packageName.MatchString(name) {
				return fmt.Errorf("invalid linked package name %q", name)
			}
			path := filepath.Join(dir, "node_modules", filepath.FromSlash(name))
			// Readlink also understands Windows directory junctions, which Go
			// does not always label with ModeSymlink in Lstat.
			targetPath, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("read registered plugin link %s: %w", name, err)
			}
			if !filepath.IsAbs(targetPath) {
				targetPath = filepath.Join(filepath.Dir(path), targetPath)
			}
			source, err := filepath.EvalSymlinks(targetPath)
			if err != nil {
				return err
			}
			if inside(source, stage) || inside(filepath.Dir(stage), source) {
				return fmt.Errorf("snapshot overlaps linked plugin %s", name)
			}
			var pkg struct{ Name string }
			if err := readJSON(filepath.Join(source, "package.json"), &pkg); err != nil {
				return err
			}
			if pkg.Name != name {
				return fmt.Errorf("linked plugin %s has package name %q", name, pkg.Name)
			}
			target := "linked-plugins/" + root + "/" + name
			if err := copyPortable(source, filepath.Join(stage, target), false); err != nil {
				return err
			}
			links = append(links, pluginLink{root, name, target})
		}
	}
	if len(links) == 0 {
		return nil
	}
	data, err := json.MarshalIndent(links, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(stage, "plugin-links.json"), data, 0600)
}

func registeredCachePath(root, path, kind string) (string, string, error) {
	if !filepath.IsAbs(path) {
		return "", "", fmt.Errorf("invalid marketplace cache path %q", path)
	}
	cache := filepath.Join(root, "cache", kind)
	cache, err := canonicalPath(cache)
	if err != nil {
		return "", "", err
	}
	source, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(cache, source)
	if err != nil || !filepath.IsLocal(rel) || rel == "." {
		return "", "", fmt.Errorf("marketplace path outside %s: %s", cache, path)
	}
	return source, "cache/" + kind + "/" + filepath.ToSlash(rel), nil
}

func restorePluginLinks(dest string) error {
	var links []pluginLink
	if err := optionalJSON(filepath.Join(dest, "plugin-links.json"), &links); err != nil {
		return err
	}
	for _, link := range links {
		if !pluginRoot.MatchString(link.Root) || !packageName.MatchString(link.Name) {
			return errors.New("invalid plugin link registration")
		}
		if link.Target != "linked-plugins/"+link.Root+"/"+link.Name && !strings.HasPrefix(link.Target, "home/"+link.Root+"/cache/plugins/") {
			return errors.New("invalid plugin link target")
		}
		target, err := safePath(dest, link.Target)
		if err != nil {
			return err
		}
		info, err := os.Stat(target)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("missing plugin directory: %s", target)
		}
		path, err := safePath(dest, "home/"+link.Root+"/node_modules/"+link.Name)
		if err != nil {
			return err
		}
		if err := makePluginLink(target, path); err != nil {
			return fmt.Errorf("register %s: %w", link.Name, err)
		}
	}
	return nil
}

func makePluginLink(target, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		// Junctions work for ordinary Windows users without Developer Mode.
		env := append(os.Environ(), "OMP_LINK_PATH="+path, "OMP_LINK_SOURCE="+target)
		return runCommand(filepath.Dir(path), env, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "$ErrorActionPreference='Stop'; New-Item -ItemType Junction -Path $env:OMP_LINK_PATH -Target $env:OMP_LINK_SOURCE | Out-Null")
	}
	return os.Symlink(target, path)
}
