package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var releaseTag = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`)
var versionOutput = regexp.MustCompile(`^(?:omp[ /])?v?([0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?)$`)

func transientHomeName(name string) bool {
	switch strings.ToLower(name) {
	case "logs", "sessions", "terminal-sessions", "custom-session-files", "blobs", "cache", ".cache", "run", "natives", "bin", "stats", "history", "install-id", "project_registries.json":
		return true
	}
	return strings.HasSuffix(strings.ToLower(name), ".log")
}

func binaryVersion(binary string) (string, error) {
	cmd := exec.Command(binary, "--version")
	cmd.Env = append(os.Environ(), "BUN_BE_BUN=")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("OMP --version failed: %w", err)
	}
	match := versionOutput.FindStringSubmatch(strings.TrimSpace(string(out)))
	if match == nil {
		return "", errors.New("OMP must report a stable release version (for example 18.8.9)")
	}
	return "v" + match[1], nil
}

// Save the real config root (~/.omp), not a binary or an npm runtime tree.
func snapshotOMP(binary, home, environment string) error {
	binary, err := filepath.Abs(binary)
	if err != nil {
		return err
	}
	release, err := binaryVersion(binary)
	if err != nil {
		return err
	}
	inputHome, err := filepath.Abs(home)
	if err != nil {
		return err
	}
	home, err = canonicalPath(home)
	if err != nil {
		return err
	}
	info, err := os.Stat(filepath.Join(home, "agent"))
	if err != nil || !info.IsDir() {
		return errors.New("OMP_HOME must be the config root containing agent/ (normally ~/.omp); consolidate custom/XDG directories first")
	}
	item := SavedItem{Kind: "omp", Release: release, HomeOrigin: home}
	if inputHome != home {
		item.HomeAliases = []string{inputHome}
	}
	return saveItem(environment, "omp", item, []string{home}, func(stage string) error {
		if err := copyPortable(home, filepath.Join(stage, "home"), true); err != nil {
			return err
		}
		if err := snapshotPlugins(home, stage); err != nil {
			return err
		}
		if err := checkDependencies(stage, func(dir string, locked bool) error {
			if locked {
				return nil
			}
			return runCommand(dir, append(os.Environ(), "BUN_BE_BUN=1"), binary, "install", "--lockfile-only", "--ignore-scripts")
		}); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(stage, "home-marker"), []byte("OMP config root restored to runtime/home\n"), 0600)
	})
}

// Dependency manifests are restored with their own Bun lock, without resetting
// OMP's enabled features/settings through repeated `omp plugin install` calls.
func checkDependencies(home string, install func(string, bool) error) error {
	return filepath.WalkDir(home, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "package.json" {
			return nil
		}
		var manifest struct {
			Dependencies         map[string]string `json:"dependencies"`
			DevDependencies      map[string]string `json:"devDependencies"`
			OptionalDependencies map[string]string `json:"optionalDependencies"`
		}
		if err := readJSON(p, &manifest); err != nil {
			return err
		}
		if len(manifest.Dependencies)+len(manifest.DevDependencies)+len(manifest.OptionalDependencies) == 0 {
			return nil
		}
		dir := filepath.Dir(p)
		locked := false
		for _, name := range []string{"bun.lock", "bun.lockb"} {
			if info, err := os.Stat(filepath.Join(dir, name)); err == nil && info.Mode().IsRegular() {
				locked = true
			}
		}
		rel, err := filepath.Rel(home, dir)
		if err != nil {
			return err
		}
		portablePath := filepath.ToSlash(rel)
		capturedPlugin := strings.HasPrefix(portablePath, "linked-plugins/") || strings.Contains(portablePath, "/plugins/cache/plugins/")
		if !locked && (!capturedPlugin || install == nil) {
			return fmt.Errorf("dependency restore needs bun.lock or bun.lockb beside %s", p)
		}
		if install != nil {
			return install(dir, locked)
		}
		return nil
	})
}

func restoredEnv(dest string) ([]string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	// OMP joins PI_CONFIG_DIR to the user's home; an absolute value is not a
	// portable override. On Windows both paths must be on the same volume.
	relative, err := filepath.Rel(userHome, filepath.Join(dest, "home"))
	if err != nil {
		return nil, fmt.Errorf("choose a restore destination on the same volume as the user home: %w", err)
	}
	return append(os.Environ(), "PI_CONFIG_DIR="+relative,
		"PI_CODING_AGENT_DIR="+filepath.Join(dest, "home", "agent"),
		"OMP_PROFILE=", "PI_PROFILE=", "XDG_DATA_HOME=", "XDG_STATE_HOME=", "XDG_CACHE_HOME=",
		"BUN_BE_BUN=", "OMP_HARNESS_MANAGER=", "OMP_HARNESS_ITEM_ID=",
		"PI_INSTALL_DIR="+filepath.Join(dest, "bin")), nil
}

func prepareOMP(project, dest string, item SavedItem) error {
	home := filepath.Join(dest, "home")
	if err := os.MkdirAll(filepath.Join(home, "agent"), 0700); err != nil {
		return err
	}
	if err := checkDependencies(dest, nil); err != nil {
		return err
	}
	env, err := restoredEnv(dest)
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		path, err := safePath(project, "install-omp.bat")
		if err != nil {
			return err
		}
		if strings.ContainsAny(path, "\"%&|<>^!\r\n") {
			return errors.New("unsupported CMD character in installer path")
		}
		err = runCommand(project, env, "cmd.exe", "/d", "/c", "call", path, item.Release)
		if err != nil {
			return err
		}
	} else {
		if err = runCommand(project, env, "sh", filepath.Join(project, "install-omp.sh"), item.Release); err != nil {
			return err
		}
	}
	binary := filepath.Join(dest, "bin", "omp")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	actual, err := binaryVersion(binary)
	if err != nil {
		return err
	}
	if actual != item.Release {
		return fmt.Errorf("installed OMP version %s does not match saved %s", actual, item.Release)
	}
	if err = checkDependencies(dest, func(dir string, locked bool) error {
		return runCommand(dir, append(env, "BUN_BE_BUN=1"), binary, "install", "--frozen-lockfile")
	}); err != nil {
		return err
	}
	if err := restorePluginLinks(dest); err != nil {
		return err
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(userHome, home)
	if err != nil {
		return err
	}
	encoded, _ := json.Marshal(relative)
	if err = os.WriteFile(filepath.Join(dest, "restore-config-path.json"), encoded, 0600); err != nil {
		return err
	}
	// Store a literal single-quoted value; no user-controlled shell interpolation.
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
	shell := `#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
export PI_CONFIG_DIR=` + quote(relative) + `
export PI_CODING_AGENT_DIR="$root/home/agent"
unset OMP_PROFILE PI_PROFILE XDG_DATA_HOME XDG_STATE_HOME XDG_CACHE_HOME BUN_BE_BUN
exec "$root/bin/omp" "$@"
`
	if err = os.WriteFile(filepath.Join(dest, "start-restored.sh"), []byte(shell), 0700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dest, "start-restored.ps1"), []byte(`$ErrorActionPreference = 'Stop'
$env:PI_CONFIG_DIR = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'restore-config-path.json') -Raw | ConvertFrom-Json
$env:PI_CODING_AGENT_DIR = Join-Path $PSScriptRoot 'home/agent'
foreach ($name in @('OMP_PROFILE', 'PI_PROFILE', 'XDG_DATA_HOME', 'XDG_STATE_HOME', 'XDG_CACHE_HOME', 'BUN_BE_BUN')) { Remove-Item "Env:$name" -ErrorAction SilentlyContinue }
& (Join-Path $PSScriptRoot 'bin/omp.exe') @args
exit $LASTEXITCODE
`), 0600)
}

// Relocate complete path prefixes, never similarly named neighbouring folders.
func relocateJSON(value any, old, next string) any {
	return relocateJSONPaths(value, []string{old}, next)
}

func relocateJSONPaths(value any, origins []string, next string) any {
	switch v := value.(type) {
	case string:
		normalized := strings.ReplaceAll(v, "\\", "/")
		for _, old := range origins {
			previous := strings.TrimRight(strings.ReplaceAll(old, "\\", "/"), "/")
			windowsPath := (len(previous) > 1 && previous[1] == ':') || strings.HasPrefix(previous, "//")
			for _, prefix := range []string{"", "file:", "link:"} {
				root := prefix + previous
				if previous == "" || len(normalized) < len(root) {
					continue
				}
				candidate := normalized[:len(root)]
				matches := candidate == root || (windowsPath && strings.EqualFold(candidate, root))
				suffix := normalized[len(root):]
				if matches && (suffix == "" || strings.HasPrefix(suffix, "/")) {
					return prefix + next + filepath.FromSlash(suffix)
				}
			}
		}
	case []any:
		for i, entry := range v {
			v[i] = relocateJSONPaths(entry, origins, next)
		}
	case map[string]any:
		for key, entry := range v {
			v[key] = relocateJSONPaths(entry, origins, next)
		}
	}
	return value
}
