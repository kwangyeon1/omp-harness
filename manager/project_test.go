package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestArgumentParsing(t *testing.T) {
	for _, bad := range []string{`null`, `[null]`, `[1]`, `"x"`, `["unfinished]`} {
		if _, err := parseArguments(bad); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	want := []string{"path with spaces", "", "$(touch nope)", "한글"}
	data, _ := json.Marshal(want)
	got, err := parseArguments(string(data))
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("%v %v", got, err)
	}
}
func fixtureProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if runtime.GOOS == "windows" {
		write(t, filepath.Join(root, "record.ps1"), `$args | ConvertTo-Json -Compress | Add-Content -Encoding UTF8 -LiteralPath $env:TEST_RESULT
if ($args[0] -eq 'fail') { exit 7 }
`)
	} else {
		write(t, filepath.Join(root, "record.sh"), "printf '%s\\n' \"$@\" >> \"$TEST_RESULT\"\n[ \"${1-}\" != fail ]\n")
	}
	write(t, filepath.Join(root, "catalog.json"), `{"items":[{"id":"one","entry":{"linux":"record.sh","windows":"record.ps1"}},{"id":"two","entry":{"linux":"record.sh","windows":"record.ps1"}}]}`)
	return root
}
func TestDirectAndMenuSameLeafContinueAfterFailure(t *testing.T) {
	root := fixtureProject(t)
	result := filepath.Join(t.TempDir(), "result")
	t.Setenv("TEST_RESULT", result)
	t.Setenv("OMP_HARNESS_STATE_FILE", filepath.Join(t.TempDir(), "state.json"))
	if err := dispatch(root, []string{"run", "one", "path with spaces", "한글"}); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(t.TempDir(), "input")
	write(t, input, "one\n[\"cancelled\"]\nn\none\n[\"fail\"]\ny\ntwo\n[\"after failure\",\"path with spaces\"]\ny\n\n")
	f, err := os.Open(input)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	previous := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = previous }()
	if err := menu(root); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(result)
	for _, want := range []string{"path with spaces", "한글", "fail", "after failure"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("missing %s: %s", want, data)
		}
	}
	if strings.Contains(string(data), "cancelled") {
		t.Fatal("cancel executed")
	}
}
func TestLocalProjectAndRemovedGitOptions(t *testing.T) {
	root := fixtureProject(t)
	got, rest, err := selectProject("unused", []string{"--project", root, "list"})
	if err != nil || got != root || len(rest) != 1 {
		t.Fatalf("%s %v %v", got, rest, err)
	}
	for _, args := range [][]string{{"--project"}, {"--checkout", "x"}, {"--project", root, "--git", "x"}, {"--project", root, "--project", root}} {
		if _, _, err := selectProject(root, args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
