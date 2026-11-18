package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The test executable doubles as a portable upstream binary fixture. Actual
// snapshot/restore/installer processes and launchers still run on each OS.
func TestMain(m *testing.M) {
	if os.Getenv("OMP_TEST_BINARY") == "1" {
		if len(os.Args) > 1 && os.Args[1] == "--version" {
			fmt.Println("omp/18.8.9")
			os.Exit(0)
		}
		log := os.Getenv("OMP_TEST_LOG")
		cwd, _ := os.Getwd()
		payload, _ := json.Marshal(map[string]any{"args": os.Args[1:], "cwd": cwd, "agent": os.Getenv("PI_CODING_AGENT_DIR"), "config": os.Getenv("PI_CONFIG_DIR"), "bun": os.Getenv("BUN_BE_BUN"), "profile": os.Getenv("OMP_PROFILE"), "xdg": os.Getenv("XDG_DATA_HOME")})
		f, err := os.OpenFile(log, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			os.Exit(8)
		}
		f.Write(append(payload, '\n'))
		f.Close()
		if os.Getenv("OMP_TEST_FAIL_DEPS") == "1" && os.Getenv("BUN_BE_BUN") == "1" {
			os.Exit(9)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func gitTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v", out, err)
	}
}
func savedOMP(t *testing.T) (string, string) {
	t.Helper()
	home := filepath.Join(t.TempDir(), "PC A home")
	saved := filepath.Join(t.TempDir(), "saved environment")
	write(t, filepath.Join(home, "agent/extensions/custom.ts"), "user-created feature")
	config, _ := json.Marshal(map[string]string{"root": home, "extension": filepath.Join(home, "agent/extensions/custom.ts")})
	write(t, filepath.Join(home, "agent/settings.json"), string(config))
	write(t, filepath.Join(home, "plugins/package.json"), `{"dependencies":{"fixture":"1.0.0"}}`)
	write(t, filepath.Join(home, "plugins/bun.lock"), "fixture lock\n")
	write(t, filepath.Join(home, "plugins/omp-plugins.lock.json"), `{"plugins":{"fixture":{"enabled":false}},"settings":{"fixture":{"color":"blue"}}}`)
	for _, rel := range []string{"agent/auth.json", "agent/agent.db", "agent/agent.db-wal", "agent/history.db", "agent/sessions/private.jsonl", "agent/cache/private", "profiles/work/agent/auth.json.bak", "agent/.env.local", "plugins/node_modules/ignored.js", "agent/secret-placeholder.key"} {
		write(t, filepath.Join(home, rel), "do not publish")
	}
	t.Setenv("OMP_TEST_BINARY", "1")
	binary, _ := os.Executable()
	if err := snapshotOMP(binary, home, saved); err != nil {
		t.Fatal(err)
	}
	return home, saved
}
func fixtureInstaller(t *testing.T) string {
	t.Helper()
	project := t.TempDir()
	binary, _ := os.Executable()
	t.Setenv("OMP_TEST_BINARY_SOURCE", binary)
	write(t, filepath.Join(project, "install-omp.sh"), `set -eu
[ "$1" = v18.8.9 ]
[ -z "$OMP_HARNESS_MANAGER" ]
mkdir -p "$PI_INSTALL_DIR"
cp "$OMP_TEST_BINARY_SOURCE" "$PI_INSTALL_DIR/omp"
chmod +x "$PI_INSTALL_DIR/omp"
`)
	write(t, filepath.Join(project, "install-omp.bat"), `@echo off
if not "%~1"=="v18.8.9" exit /b 6
if defined OMP_HARNESS_MANAGER exit /b 7
mkdir "%PI_INSTALL_DIR%"
copy /y "%OMP_TEST_BINARY_SOURCE%" "%PI_INSTALL_DIR%\omp.exe" >nul
exit /b %errorlevel%
`)
	return project
}
func TestOMPSnapshotGitRoundTripAndIntegrity(t *testing.T) {
	_, saved := savedOMP(t)
	e, err := loadEnvironment(saved)
	if err != nil {
		t.Fatal(err)
	}
	for name := range e.Items["omp"].Files {
		if strings.Contains(name, "auth.json") || strings.Contains(name, ".db") || strings.Contains(name, "private") || strings.Contains(name, "ignored") || strings.Contains(name, ".env") || strings.HasSuffix(name, ".key") {
			t.Fatal("secret/cache captured", name)
		}
	}
	gitTest(t, saved, "init")
	gitTest(t, saved, "add", "-f", ".")
	gitTest(t, saved, "-c", "user.name=Fixture", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "-m", "saved")
	cloned := filepath.Join(t.TempDir(), "PC B clone")
	if out, err := exec.Command("git", "clone", "--", saved, cloned).CombinedOutput(); err != nil {
		t.Fatalf("%s %v", out, err)
	}
	if err := verifyItem(cloned, "omp", e.Items["omp"]); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(cloned, "omp/home/agent/extensions/custom.ts"), "tampered")
	dest := filepath.Join(t.TempDir(), "dest")
	if err := restoreItem(t.TempDir(), cloned, "omp", dest); err == nil {
		t.Fatal("accepted corruption")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("created destination before verification")
	}
}
func TestOMPRestoreUsesPinnedBinaryHomeAndLocks(t *testing.T) {
	_, saved := savedOMP(t)
	project := fixtureInstaller(t)
	dest := filepath.Join(t.TempDir(), "new PC runtime")
	// Windows TEMP can use an 8.3 alias (RUNNER~1); restoration intentionally
	// expands existing ancestors before building the launcher and rebasing paths.
	dest, err := canonicalPath(dest)
	if err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(t.TempDir(), "commands")
	t.Setenv("OMP_TEST_LOG", log)
	t.Setenv("OMP_PROFILE", "unrelated")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("OMP_HARNESS_MANAGER", "must not record installation during restore")
	if err := restoreItem(project, saved, "omp", dest); err != nil {
		t.Fatal(err)
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(dest, "start-restored.ps1"), "argument with spaces")
	} else {
		cmd = exec.Command("sh", filepath.Join(dest, "start-restored.sh"), "argument with spaces")
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v", out, err)
	}
	data, _ := os.ReadFile(log)
	rows := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(rows) != 2 {
		t.Fatalf("missing dependency/launcher invocation: %s", data)
	}
	var dep, launch struct {
		Args                             []string
		Agent, Config, Bun, Profile, Xdg string
	}
	json.Unmarshal([]byte(rows[0]), &dep)
	json.Unmarshal([]byte(rows[1]), &launch)
	if strings.Join(dep.Args, " ") != "install --frozen-lockfile" || dep.Bun != "1" {
		t.Fatal("dependencies not locked", dep)
	}
	if len(launch.Args) != 1 || launch.Args[0] != "argument with spaces" || launch.Agent != filepath.Join(dest, "home/agent") || launch.Profile != "" || launch.Xdg != "" || launch.Bun != "" {
		t.Fatal("launcher context incorrect", launch)
	}
	var config map[string]string
	if err := readJSON(filepath.Join(dest, "home/agent/settings.json"), &config); err != nil || config["root"] != filepath.Join(dest, "home") {
		t.Fatal("home path not rebased", config, err)
	}
	if _, err := os.Stat(filepath.Join(dest, "restored.json")); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(filepath.Join(saved, "omp/home/plugins/omp-plugins.lock.json"))
	restored, _ := os.ReadFile(filepath.Join(dest, "home/plugins/omp-plugins.lock.json"))
	if string(original) != string(restored) {
		t.Fatal("plugin enablement/settings changed")
	}
	binary := filepath.Join(dest, "bin/omp")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if err := snapshotOMP(binary, filepath.Join(dest, "home"), filepath.Join(t.TempDir(), "next")); err != nil {
		t.Fatal("cannot resnapshot", err)
	}
	if err := restoreItem(project, saved, "omp", dest); err == nil {
		t.Fatal("overwrote destination")
	}
}
func TestOMPFailedDependenciesKeepFilesWithoutSuccess(t *testing.T) {
	_, saved := savedOMP(t)
	project := fixtureInstaller(t)
	t.Setenv("OMP_TEST_LOG", filepath.Join(t.TempDir(), "log"))
	t.Setenv("OMP_TEST_FAIL_DEPS", "1")
	dest := filepath.Join(t.TempDir(), "failed")
	if err := restoreItem(project, saved, "omp", dest); err == nil {
		t.Fatal("ignored failed dependency install")
	}
	if _, err := os.Stat(filepath.Join(dest, "restored.json")); !os.IsNotExist(err) {
		t.Fatal("failure marked complete")
	}
	if _, err := os.Stat(filepath.Join(dest, "home/agent/extensions/custom.ts")); err != nil {
		t.Fatal("failure files lost")
	}
}
func TestOMPRejectsUnrestorableDependencies(t *testing.T) {
	home, saved := savedOMP(t)
	os.Remove(filepath.Join(home, "plugins/bun.lock"))
	binary, _ := os.Executable()
	if err := snapshotOMP(binary, home, filepath.Join(t.TempDir(), "no lock")); err == nil {
		t.Fatal("accepted missing dependency lock")
	}
	if err := snapshotOMP(binary, home, saved); err == nil {
		t.Fatal("overwrote snapshot")
	}
	write(t, filepath.Join(home, "plugins/bun.lock"), "lock")
	write(t, filepath.Join(home, "plugins/omp-plugins.lock.json"), `{"plugins":{"external-link":{}}}`)
	if err := snapshotOMP(binary, home, filepath.Join(t.TempDir(), "external")); err == nil {
		t.Fatal("silently dropped linked plugin")
	}
}
func TestRestorePathsAndSymlinkContainment(t *testing.T) {
	home, saved := savedOMP(t)
	e, _ := loadEnvironment(saved)
	item := e.Items["omp"]
	item.Files["../../escape"] = "bad"
	if err := verifyItem(saved, "omp", item); err == nil {
		t.Fatal("accepted traversal")
	}
	if runtime.GOOS == "windows" {
		return
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(home, alias); err != nil {
		t.Fatal(err)
	}
	binary, _ := os.Executable()
	if err := snapshotOMP(binary, home, filepath.Join(alias, "nested")); err == nil {
		t.Fatal("snapshot recursed through symlink")
	}
	if err := os.Symlink(saved, alias+"saved"); err != nil {
		t.Fatal(err)
	}
	if err := restoreItem(t.TempDir(), saved, "omp", filepath.Join(alias+"saved", "nested")); err == nil {
		t.Fatal("restored into snapshot through symlink")
	}
	if err := os.Symlink("/etc/passwd", filepath.Join(home, "agent/external")); err != nil {
		t.Fatal(err)
	}
	if err := snapshotOMP(binary, home, filepath.Join(t.TempDir(), "unsafe")); err == nil {
		t.Fatal("copied external symlink")
	}
}
func TestModuleRestoreMenuCancellationFailureAndRepeat(t *testing.T) {
	source := t.TempDir()
	env := filepath.Join(t.TempDir(), "environment")
	write(t, filepath.Join(source, "restore.sh"), "test -f custom.txt\nexit 0\n")
	write(t, filepath.Join(source, "restore.ps1"), "if (!(Test-Path custom.txt)) { exit 3 }; exit 0\n")
	write(t, filepath.Join(source, "custom.txt"), "custom module data")
	write(t, filepath.Join(source, "restored.json"), "old success marker")
	if err := snapshotModule("extra", source, env); err != nil {
		t.Fatal(err)
	}
	failed := t.TempDir()
	write(t, filepath.Join(failed, "restore.sh"), "exit 4\n")
	write(t, filepath.Join(failed, "restore.ps1"), "exit 4\n")
	if err := snapshotModule("failure", failed, env); err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	cancel := filepath.Join(base, "cancel")
	bad := filepath.Join(base, "bad")
	good := filepath.Join(base, "good")
	file := filepath.Join(t.TempDir(), "answers")
	write(t, file, "extra\n"+cancel+"\nn\nfailure\n"+bad+"\ny\nextra\n"+good+"\ny\n\n")
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	prev := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = prev }()
	if err := restoreMenu(t.TempDir(), env); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cancel); !os.IsNotExist(err) {
		t.Fatal("cancel ran")
	}
	if _, err := os.Stat(filepath.Join(bad, "restored.json")); !os.IsNotExist(err) {
		t.Fatal("failure marked complete")
	}
	if data, err := os.ReadFile(filepath.Join(good, "custom.txt")); err != nil || string(data) != "custom module data" {
		t.Fatal("next module failed", err)
	}
	if err := snapshotModule("extra", source, env); err == nil {
		t.Fatal("overwrote saved item")
	}
}
func TestSnapshotHashesAllowGitTextLineEndings(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yml")
	write(t, p, "one: 1\ntwo: 2\n")
	a, _ := hashFile(p)
	write(t, p, "one: 1\r\ntwo: 2\r\n")
	b, _ := hashFile(p)
	if a != b {
		t.Fatal("Git line ending conversion invalidates portable text")
	}
	write(t, p, "one: 2\r\ntwo: 2\r\n")
	c, _ := hashFile(p)
	if c == a {
		t.Fatal("content change not detected")
	}
}

func TestJSONRelocationDoesNotChangeSiblingPrefixes(t *testing.T) {
	value := map[string]any{"inside": `C:\old\omp\agent\extensions\custom.ts`, "sibling": `C:\old\omp-backup\file`, "nested": []any{"file:C:/old/omp/plugins/custom"}}
	next := filepath.Join(t.TempDir(), "new home")
	got := relocateJSON(value, `C:\old\omp`, next).(map[string]any)
	if got["inside"] != filepath.Join(next, "agent/extensions/custom.ts") || got["sibling"] != `C:\old\omp-backup\file` || got["nested"].([]any)[0] != "file:"+filepath.Join(next, "plugins/custom") {
		t.Fatal(got)
	}
}
func TestEmptyHomeAndSnapshotLock(t *testing.T) {
	t.Setenv("OMP_TEST_BINARY", "1")
	binary, _ := os.Executable()
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, "agent"), 0700); err != nil {
		t.Fatal(err)
	}
	saved := filepath.Join(t.TempDir(), "saved")
	if err := snapshotOMP(binary, home, saved); err != nil {
		t.Fatal(err)
	}
	module := t.TempDir()
	write(t, filepath.Join(module, "restore.sh"), "exit 0")
	write(t, filepath.Join(saved, ".snapshot.lock"), "busy")
	if err := snapshotModule("extra", module, saved); err == nil {
		t.Fatal("ignored snapshot lock")
	}
}
