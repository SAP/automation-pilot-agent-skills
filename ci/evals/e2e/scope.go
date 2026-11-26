package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// Validate declared dependencies before any create. Dynamic target selection is
// excluded: this suite uses fixed native commands and its own declared resources.
func validateCandidateScope(root, definitions string, spec suite) error {
	commands, inputs := map[string]bool{}, map[string]bool{}
	candidates, contracts := map[string]map[string]any{}, map[string]map[string]any{}
	for _, resource := range spec.Resources {
		path := filepath.Join(definitions, resource.File)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular candidate file", resource.File)
		}
		var candidate, contract map[string]any
		if err := read(path, &candidate); err != nil {
			return err
		}
		if err := read(filepath.Join(root, "definitions", resource.File), &contract); err != nil {
			return err
		}
		candidates[resource.File] = candidate
		contracts[resource.File] = contract
		for _, key := range []string{"name", "catalog", "technicalName"} {
			if required, ok := contract[key]; ok && !reflect.DeepEqual(candidate[key], required) {
				return fmt.Errorf("%s must preserve public %s role/namespace", resource.File, key)
			}
		}
		id := str(contract["catalog"]) + ":" + str(contract["name"]) + ":1"
		if resource.Resource == "commands" {
			commands[id] = true
		}
		if resource.Resource == "inputs" {
			inputs[id] = true
		}
	}
	allowed := map[string]bool{}
	for id := range commands {
		allowed[id] = true
	}
	for _, id := range NativeDependencies() {
		allowed[id] = true
	}
	var inspectCommand func(any) error
	inspectCommand = func(value any) error {
		switch v := value.(type) {
		case map[string]any:
			for key, item := range v {
				if key == "execute" || key == "command" || key == "commandId" {
					if target, ok := item.(string); ok && !allowed[target] {
						return fmt.Errorf("command target is outside the declared E2E dependency graph")
					}
				}
				if err := inspectCommand(item); err != nil {
					return err
				}
			}
		case []any:
			for _, item := range v {
				if err := inspectCommand(item); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, resource := range spec.Resources {
		candidate, contract := candidates[resource.File], contracts[resource.File]
		switch resource.Resource {
		case "commands":
			if err := inspectCommand(candidate["configuration"]); err != nil {
				return fmt.Errorf("%s: %w", resource.File, err)
			}
		case "inputs":
			for key, expected := range object(contract["values"]) {
				if object(object(contract["keys"])[key])["sensitive"] == true {
					continue
				}
				if !equivalentValue(object(candidate["values"])[key], expected) {
					return fmt.Errorf("%s: required fixture value %s changed", resource.File, key)
				}
			}
		case "mcp-servers":
			tools, _ := candidate["mcpTools"].([]any)
			for _, tool := range tools {
				t := object(tool)
				if !commands[str(t["commandId"])] {
					return fmt.Errorf("%s: MCP tool targets an undeclared command", resource.File)
				}
				refs, _ := t["inputReferences"].([]any)
				for _, ref := range refs {
					if id, ok := ref.(string); !ok || !inputs[id] {
						return fmt.Errorf("%s: MCP binding targets an undeclared Input", resource.File)
					}
				}
			}
		}
	}
	return nil
}
func equivalentValue(a, b any) bool {
	left, lok := a.(string)
	right, rok := b.(string)
	if lok && rok {
		var x, y any
		if json.Unmarshal([]byte(left), &x) == nil && json.Unmarshal([]byte(right), &y) == nil {
			return reflect.DeepEqual(x, y)
		}
		return strings.TrimSpace(left) == strings.TrimSpace(right)
	}
	return reflect.DeepEqual(a, b)
}
