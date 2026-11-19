package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPluginsRestoreWithoutSourcePC(t *testing.T) {
	a := t.TempDir()
	home := filepath.Join(a, "config")
	root := filepath.Join(home, "profiles/work/plugins")
	local := filepath.Join(a, "local")
	write(t, filepath.Join(home, "agent/settings.json"), "{}")
	write(t, filepath.Join(local, "package.json"), `{"name":"@team/local","version":"1.2.0"}`)
	write(t, filepath.Join(local, "feature.ts"), "local source edit")
	write(t, filepath.Join(local, ".env"), "private")
	if err := makePluginLink(local, filepath.Join(root, "node_modules/@team/local")); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "cache/plugins/lab___demo___2")
	write(t, filepath.Join(cache, "skills/demo/SKILL.md"), "marketplace source")
	entry, _ := json.Marshal(map[string]any{"version": 2, "plugins": map[string]any{"demo@lab": []any{map[string]any{"version": "2", "enabled": false, "installPath": cache}}}})
	write(t, filepath.Join(root, "installed_plugins.json"), string(entry))
	lock := `{"plugins":{"@team/local":{"enabled":true},"demo":{"enabled":false}},"settings":{"demo":{"color":"blue"}}}`
	write(t, filepath.Join(root, "omp-plugins.lock.json"), lock)
	t.Setenv("OMP_TEST_BINARY", "1")
	exe, _ := os.Executable()
	env := filepath.Join(t.TempDir(), "saved")
	if err := snapshotOMP(exe, home, env); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(a, a+"-offline"); err != nil {
		t.Fatal(err)
	}
	defer os.Rename(a+"-offline", a)
	dest, err := canonicalPath(filepath.Join(t.TempDir(), "B"))
	if err != nil {
		t.Fatal(err)
	}
	if err := restoreItem(fixtureInstaller(t), env, "omp", dest); err != nil {
		t.Fatal(err)
	}
	bRoot := filepath.Join(dest, "home/profiles/work/plugins")
	for rel, want := range map[string]string{"node_modules/@team/local/feature.ts": "local source edit", "node_modules/demo/skills/demo/SKILL.md": "marketplace source", "omp-plugins.lock.json": lock} {
		got, err := os.ReadFile(filepath.Join(bRoot, rel))
		if err != nil || string(got) != want {
			t.Fatalf("%s: %s %v", rel, got, err)
		}
	}
	if _, err := os.Stat(filepath.Join(bRoot, "node_modules/@team/local/.env")); !os.IsNotExist(err) {
		t.Fatal("included .env")
	}
	var reg struct {
		Plugins map[string][]struct {
			InstallPath string
			Enabled     bool
		}
	}
	if err := readJSON(filepath.Join(bRoot, "installed_plugins.json"), &reg); err != nil {
		t.Fatal(err)
	}
	installed := reg.Plugins["demo@lab"][0]
	if installed.Enabled || installed.InstallPath != filepath.Join(bRoot, "cache/plugins/lab___demo___2") {
		t.Fatalf("incorrect registry: %#v", installed)
	}
	if err := snapshotOMP(exe, filepath.Join(dest, "home"), filepath.Join(t.TempDir(), "next")); err != nil {
		t.Fatal(err)
	}
}
