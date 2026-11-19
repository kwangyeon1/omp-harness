// The manager reads declared state; only leaf scripts inspect installed products.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type Item struct {
	ID    string            `json:"id"`
	Name  string            `json:"name"`
	Kind  string            `json:"kind"`
	Entry map[string]string `json:"entry"`
	Help  string            `json:"help"`
}
type Catalog struct {
	Items []Item `json:"items"`
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("invalid JSON in %s: %w", path, err)
	}
	return nil
}
func dataDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "omp-harness")
	}
	if base := os.Getenv("XDG_STATE_HOME"); base != "" {
		return filepath.Join(base, "omp-harness")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "omp-harness")
}
func statePath() string {
	if p := os.Getenv("OMP_HARNESS_STATE_FILE"); p != "" {
		return p
	}
	return filepath.Join(dataDir(), "state.json")
}
func readState(path string) (map[string]bool, error) {
	// Reject non-boolean values, including null, without silently resetting state.
	raw := map[string]json.RawMessage{}
	if err := readJSON(path, &raw); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	if raw == nil {
		return nil, errors.New("state must be a JSON object")
	}
	state := map[string]bool{}
	for id, v := range raw {
		s := strings.TrimSpace(string(v))
		if s != "true" && s != "false" {
			return nil, fmt.Errorf("state %s must be boolean", id)
		}
		state[id] = s == "true"
	}
	return state, nil
}
func setState(path, id string, installed bool) error {
	if id == "" {
		return errors.New("item ID is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("state is locked; retry after the other script finishes: %w", err)
	}
	lock.Close()
	defer os.Remove(path + ".lock")
	state, err := readState(path)
	if err != nil {
		return err
	}
	state[id] = installed
	data, _ := json.MarshalIndent(state, "", "  ")
	return atomicWrite(path, append(data, '\n'), 0600)
}
func loadCatalog(root string) (Catalog, error) {
	var c Catalog
	err := readJSON(filepath.Join(root, "catalog.json"), &c)
	seen := map[string]bool{}
	for _, item := range c.Items {
		if item.ID == "" || seen[item.ID] {
			return c, errors.New("catalog IDs must be unique and nonempty")
		}
		seen[item.ID] = true
	}
	return c, err
}
func show(root string) error {
	c, err := loadCatalog(root)
	if err != nil {
		return err
	}
	s, err := readState(statePath())
	if err != nil {
		return err
	}
	fmt.Println("Recorded installation state (JSON only):")
	for _, item := range c.Items {
		status := "not installed"
		if s[item.ID] {
			status = "installed"
		}
		fmt.Printf("%s | %s | %s\n", item.ID, status, item.Name)
	}
	return nil
}
func runItem(root, id string, args ...string) error {
	c, err := loadCatalog(root)
	if err != nil {
		return err
	}
	for _, item := range c.Items {
		if item.ID == id {
			entry := item.Entry[runtime.GOOS]
			if entry == "" {
				return errors.New("item has no entry for this platform")
			}
			path, err := safePath(root, entry)
			if err != nil {
				return err
			}
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				if strings.EqualFold(filepath.Ext(path), ".ps1") {
					cmd = exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", path)
					cmd.Args = append(cmd.Args, args...)
				} else {
					// cmd.exe reparses command strings. Reject expansion/control characters.
					for _, value := range append([]string{path}, args...) {
						if strings.ContainsAny(value, "\"%&|<>^!\r\n") {
							return errors.New("CMD entry/arguments contain unsupported metacharacters; use a PowerShell entry")
						}
					}
					cmd = exec.Command("cmd.exe", "/d", "/c", "call", path)
					cmd.Args = append(cmd.Args, args...)
				}
			} else {
				cmd = exec.Command("sh", path)
				cmd.Args = append(cmd.Args, args...)
			}
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			state, err := filepath.Abs(statePath())
			if err != nil {
				return err
			}
			settings := os.Getenv("OMP_HARNESS_SETTINGS_FILE")
			if settings == "" {
				settings = filepath.Join(dataDir(), "settings.json")
			}
			settings, err = filepath.Abs(settings)
			if err != nil {
				return err
			}
			cmd.Env = append(os.Environ(), "OMP_HARNESS_MANAGER="+exe, "OMP_HARNESS_STATE_FILE="+state, "OMP_HARNESS_ITEM_ID="+id, "OMP_HARNESS_DEFAULTS_FILE="+filepath.Join(root, "defaults", "settings.json"), "OMP_HARNESS_SETTINGS_FILE="+settings)
			cmd.Dir = root
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			// Never infer installed=true from a zero exit code: cancellation also exits zero.
			return cmd.Run()
		}
	}
	return fmt.Errorf("unknown item: %s", id)
}

// Read only the menu choice: buffering stdin here steals answers from the leaf.
func readMenuLine(reader io.Reader) (string, error) {
	var line strings.Builder
	var char [1]byte
	for {
		n, err := reader.Read(char[:])
		if n > 0 {
			if char[0] == '\n' {
				return line.String(), nil
			}
			line.WriteByte(char[0])
		}
		if err != nil {
			if err == io.EOF && line.Len() > 0 {
				return line.String(), nil
			}
			return "", err
		}
	}
}
func menu(root string) error {
	for {
		if err := show(root); err != nil {
			return err
		}
		fmt.Print("Item ID to run (Enter to exit): ")
		line, err := readMenuLine(os.Stdin)
		if err != nil {
			return nil
		}
		id := strings.TrimSpace(line)
		if id == "" {
			return nil
		}
		if err := itemHelp(root, id); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		fmt.Print("Arguments as JSON array (Enter = defaults, e.g. [\"v18.8.9\"]): ")
		line, err = readMenuLine(os.Stdin)
		if err != nil {
			return nil
		}
		args, err := parseArguments(line)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		fmt.Printf("Run %s %q? [y/N]: ", id, args)
		confirm, err := readMenuLine(os.Stdin)
		if err != nil {
			return nil
		}
		if !strings.EqualFold(strings.TrimSpace(confirm), "y") {
			continue
		}
		if err := runItem(root, id, args...); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
func dispatch(root string, args []string) error {
	if len(args) == 0 {
		return menu(root)
	}
	switch args[0] {
	case "--help", "-h":
		fmt.Println(usage)
		return nil
	case "help":
		if len(args) == 2 {
			return itemHelp(root, args[1])
		}
	case "list":
		return show(root)
	case "run":
		if len(args) >= 2 {
			return runItem(root, args[1], args[2:]...)
		}
	case "snapshot":
		if len(args) == 5 && args[1] == "omp" {
			return snapshotOMP(args[2], args[3], args[4])
		}
		if len(args) == 5 && args[1] == "module" {
			return snapshotModule(args[2], args[3], args[4])
		}
	case "restore":
		return restoreDispatch(root, args[1:])
	case "state":
		if len(args) == 4 && args[1] == "set" && (args[3] == "true" || args[3] == "false") {
			b, _ := strconv.ParseBool(args[3])
			return setState(statePath(), args[2], b)
		}
	case "bundle":
		if len(args) >= 3 && args[1] == "apply" {
			selected := args[3:]
			return applyBundle(root, args[2], selected)
		}
		if len(args) == 2 && args[1] == "remove" {
			return removeBundle(root)
		}
	}
	return errors.New(usage)
}
func main() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	root := filepath.Dir(filepath.Dir(exe))
	root, args, err := selectProject(root, os.Args[1:])
	if err == nil {
		err = dispatch(root, args)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
