package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func write(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}
func TestStateOnlyAndExplicitRecording(t *testing.T) {
	root := filepath.Join(t.TempDir(), "entry path with spaces")
	state := filepath.Join(t.TempDir(), "state.json")
	t.Setenv("OMP_HARNESS_STATE_FILE", state)
	// No product binary is installed; true in JSON must still display as installed.
	write(t, filepath.Join(root, "catalog.json"), `{"items":[{"id":"missing-product","name":"Test","entry":{}}]}`)
	write(t, state, `{"missing-product":true,"other":false}`)
	if err := show(root); err != nil {
		t.Fatal(err)
	}
	if err := setState(state, "other", true); err != nil {
		t.Fatal(err)
	}
	got, err := readState(state)
	if err != nil || !got["missing-product"] || !got["other"] {
		t.Fatalf("%v %v", got, err)
	}
	write(t, state, `{"other":null}`)
	if _, err := readState(state); err == nil {
		t.Fatal("accepted invalid state")
	}
	before, _ := os.ReadFile(state)
	if err := setState(state, "new", true); err == nil {
		t.Fatal("overwrote corrupt state")
	}
	after, _ := os.ReadFile(state)
	if string(before) != string(after) {
		t.Fatal("state changed")
	}
}
func TestChildSuccessCancelAndFailureNeverInferState(t *testing.T) {
	root := filepath.Join(t.TempDir(), "entry path with spaces")
	state := filepath.Join(t.TempDir(), "state.json")
	t.Setenv("OMP_HARNESS_STATE_FILE", state)
	for _, status := range []string{"0", "1"} {
		name := "cancel.sh"
		body := "#!/bin/sh\nexit " + status + "\n"
		if runtime.GOOS == "windows" {
			name = "cancel.cmd"
			body = "@echo off\r\nexit /b " + status + "\r\n"
		}
		write(t, filepath.Join(root, name), body)
		c := Catalog{Items: []Item{{ID: "test", Name: "test", Entry: map[string]string{runtime.GOOS: name}}}}
		data, _ := json.Marshal(c)
		write(t, filepath.Join(root, "catalog.json"), string(data))
		write(t, state, `{"test":false,"existing":true}`)
		err := runItem(root, "test")
		if (err != nil) != (status == "1") {
			t.Fatalf("exit %s: %v", status, err)
		}
		s, _ := readState(state)
		if s["test"] || !s["existing"] {
			t.Fatal("manager inferred state from exit")
		}
	}
}
func makeBundle(t *testing.T, root, version string, parts map[string]map[string]string) {
	t.Helper()
	b := Bundle{Version: version, Components: map[string]Component{}}
	for id, files := range parts {
		c := Component{Version: version, Files: map[string]string{}}
		for p, body := range files {
			write(t, filepath.Join(root, p), body)
			hash := sha256.Sum256([]byte(body))
			c.Files[p] = hex.EncodeToString(hash[:])
		}
		b.Components[id] = c
	}
	data, _ := json.Marshal(b)
	write(t, filepath.Join(root, "bundle.json"), string(data))
}
func TestPartialFullAndPreservation(t *testing.T) {
	old, source := t.TempDir(), t.TempDir()
	dest := filepath.Join(t.TempDir(), "installed with spaces")
	makeBundle(t, old, "1.0.0", map[string]map[string]string{"core": {"defaults/settings.json": "old-default"}, "plugin-a": {"plugins/a/old.txt": "old"}})
	if err := applyBundle(old, dest, nil); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dest, "user-settings.json"), "keep")
	state := filepath.Join(t.TempDir(), "state.json")
	write(t, state, `{"plugin-a":true}`)
	t.Setenv("OMP_HARNESS_STATE_FILE", state)
	makeBundle(t, source, "2.0.0", map[string]map[string]string{"core": {"defaults/settings.json": "new-default"}, "plugin-a": {"plugins/a/new.txt": "new"}})
	if err := applyBundle(source, dest, []string{"plugin-a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "plugins/a/old.txt")); !os.IsNotExist(err) {
		t.Fatal("stale file remains")
	}
	data, _ := os.ReadFile(filepath.Join(dest, "defaults/settings.json"))
	if string(data) != "old-default" {
		t.Fatal("unselected component changed")
	}
	b, _ := loadBundle(dest)
	if b.Components["core"].Version != "1.0.0" || b.Components["plugin-a"].Version != "2.0.0" {
		t.Fatal("component versions incorrect")
	}
	if err := applyBundle(source, dest, nil); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(filepath.Join(dest, "defaults/settings.json"))
	if string(data) != "new-default" {
		t.Fatal("default not updated")
	}
	if err := removeBundle(dest); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(filepath.Join(dest, "user-settings.json"))
	if string(data) != "keep" {
		t.Fatal("user file lost")
	}
	data, _ = os.ReadFile(state)
	if string(data) != `{"plugin-a":true}` {
		t.Fatal("recorded state changed")
	}
}
func TestInvalidBundleDoesNotTouchDestination(t *testing.T) {
	source, dest := t.TempDir(), t.TempDir()
	write(t, filepath.Join(dest, ".omp-harness-package"), "omp-harness-script-installer\n")
	write(t, filepath.Join(dest, "README.md"), "keep")
	makeBundle(t, source, "1.0.0", map[string]map[string]string{"core": {"README.md": "new"}})
	write(t, filepath.Join(source, "README.md"), "tampered")
	if err := applyBundle(source, dest, nil); err == nil {
		t.Fatal("accepted bad hash")
	}
	data, _ := os.ReadFile(filepath.Join(dest, "README.md"))
	if string(data) != "keep" {
		t.Fatal("destination changed")
	}
	for _, p := range []string{"../escape", "plugins/../../escape", "/absolute", "plugins/a\\b", "state.json", "settings.json"} {
		makeBundle(t, source, "1.0.0", map[string]map[string]string{"bad": {"README.md": "new"}})
		data, _ := os.ReadFile(filepath.Join(source, "bundle.json"))
		data = []byte(strings.ReplaceAll(string(data), "README.md", strings.ReplaceAll(p, "\\", "\\\\")))
		write(t, filepath.Join(source, "bundle.json"), string(data))
		if _, err := loadBundle(source); err == nil {
			t.Fatalf("accepted %s", p)
		}
	}
}
func TestSymlinkDestinationRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privilege varies on Windows")
	}
	source, dest, outside := t.TempDir(), t.TempDir(), t.TempDir()
	makeBundle(t, source, "1.0.0", map[string]map[string]string{"plugin": {"plugins/file": "payload"}})
	write(t, filepath.Join(dest, ".omp-harness-package"), "omp-harness-script-installer\n")
	if err := os.Symlink(outside, filepath.Join(dest, "plugins")); err != nil {
		t.Fatal(err)
	}
	if err := applyBundle(source, dest, nil); err == nil {
		t.Fatal("followed symlink")
	}
	if _, err := os.Stat(filepath.Join(outside, "file")); !os.IsNotExist(err) {
		t.Fatal("wrote outside destination")
	}
}

func TestBundleLockAndUserCollision(t *testing.T) {
	source, dest := t.TempDir(), t.TempDir()
	makeBundle(t, source, "1", map[string]map[string]string{"core": {"README.md": "old"}})
	if err := applyBundle(source, dest, []string{"core"}); err == nil {
		t.Fatal("partial initial install accepted")
	}
	if err := applyBundle(source, dest, nil); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dest, ".bundle.lock"), "busy")
	if err := applyBundle(source, dest, nil); err == nil {
		t.Fatal("ignored lock")
	}
	if err := removeBundle(dest); err == nil {
		t.Fatal("removal ignored lock")
	}
	os.Remove(filepath.Join(dest, ".bundle.lock"))
	write(t, filepath.Join(dest, "plugins/custom.txt"), "user data")
	makeBundle(t, source, "2", map[string]map[string]string{"core": {"README.md": "new"}, "plugin": {"plugins/custom.txt": "package data"}})
	if err := applyBundle(source, dest, nil); err == nil {
		t.Fatal("overwrote user collision")
	}
	data, _ := os.ReadFile(filepath.Join(dest, "plugins/custom.txt"))
	if string(data) != "user data" {
		t.Fatal("user file changed")
	}
	data, _ = os.ReadFile(filepath.Join(dest, "README.md"))
	if string(data) != "old" {
		t.Fatal("modified destination before collision check")
	}
}
