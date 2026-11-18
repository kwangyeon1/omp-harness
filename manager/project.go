package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const usage = `manage [--project PATH] [COMMAND]
No command: repeatable installation menu.
Commands: list | help ID | run ID [ARG ...] | state set ID true/false
          bundle apply DEST [COMPONENT ...] | bundle remove
          snapshot omp BINARY OMP_HOME ENVIRONMENT
          snapshot module ID DIRECTORY ENVIRONMENT
          restore [ENVIRONMENT] | restore list ENVIRONMENT
          restore run ENVIRONMENT ID NEW_DESTINATION
Projects must contain catalog.json. Only run trusted project scripts.
Clone saved environments with Git yourself, then open the restore menu.`

func parseArguments(line string) ([]string, error) {
	if strings.TrimSpace(line) == "" {
		return nil, nil
	}
	var args []string
	if err := json.Unmarshal([]byte(line), &args); err != nil || args == nil {
		return nil, errors.New("arguments must be a JSON array of strings")
	}
	// JSON null elements must not silently become empty arguments.
	var raw []json.RawMessage
	_ = json.Unmarshal([]byte(line), &raw)
	for _, v := range raw {
		if string(v) == "null" {
			return nil, errors.New("null argument is not a string")
		}
	}
	return args, nil
}
func itemHelp(root, id string) error {
	c, err := loadCatalog(root)
	if err != nil {
		return err
	}
	for _, item := range c.Items {
		if item.ID == id {
			fmt.Printf("%s: %s\n%s\nEntries: %v\n", item.ID, item.Name, item.Help, item.Entry)
			return nil
		}
	}
	return fmt.Errorf("unknown item: %s", id)
}
func selectProject(root string, args []string) (string, []string, error) {
	if len(args) > 0 && args[0] == "--project" {
		if len(args) < 2 || args[1] == "" {
			return root, nil, errors.New("--project requires a path")
		}
		absolute, err := filepath.Abs(args[1])
		if err != nil {
			return root, nil, err
		}
		if _, err := loadCatalog(absolute); err != nil {
			return root, nil, err
		}
		root = absolute
		args = args[2:]
	}
	if len(args) > 0 && strings.HasPrefix(args[0], "--") && args[0] != "--help" {
		return root, nil, errors.New("unsupported option: " + args[0])
	}
	return root, args, nil
}
