package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func mineInventory(root, out string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	catalogRoot := filepath.Join(root, "content/public/catalogs")
	testRoot := filepath.Join(root, "content/public/tests")
	revision, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		return err
	}
	files := map[string]string{}
	commands := []map[string]any{}
	testReferences := map[string][]string{}
	catalogs, inputs, tests, oq := 0, 0, 0, 0
	families := map[string]bool{}
	ids := regexp.MustCompile(`[A-Za-z0-9_-]+:[A-Za-z0-9_]+:[0-9]+`)
	for _, directory := range []string{catalogRoot, testRoot} {
		err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink in content source")
			}
			if d.IsDir() || !strings.HasSuffix(path, ".json") {
				return nil
			}
			if directory == testRoot && !strings.HasSuffix(path, ".command.json") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var object map[string]any
			if json.Unmarshal(raw, &object) != nil {
				return fmt.Errorf("invalid source JSON: %s", path)
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			files[rel] = bytesDigest(raw)
			if directory == testRoot {
				tests++
				if strings.HasSuffix(path, "OQ.command.json") || strings.HasSuffix(path, "OQSuite.command.json") {
					oq++
				}
				found := map[string]bool{}
				normalized := strings.ReplaceAll(string(raw), "-<<<TENANT_ID>>>", "-sapcp")
				for _, id := range ids.FindAllString(normalized, -1) {
					found[id] = true
				}
				testReferences[rel] = sortedKeys(found)
				return nil
			}
			switch {
			case strings.HasSuffix(path, ".catalog.json"):
				catalogs++
			case strings.HasSuffix(path, ".input.json"):
				inputs++ // Values are never exported.
			case strings.HasSuffix(path, ".command.json"):
				id, _ := object["id"].(string)
				tags, _ := object["tags"].(map[string]any)
				_, hidden := tags["autopi:hidden"]
				_, deprecated := tags["autopi:deprecated"]
				configuration, _ := object["configuration"].(map[string]any)
				steps, _ := configuration["executors"].([]any)
				controls := map[string]int{}
				dependencies := map[string]bool{}
				for _, value := range steps {
					step, _ := value.(map[string]any)
					for _, name := range []string{"when", "validate", "autoRetry", "repeat", "initialDelay"} {
						if step[name] != nil {
							controls[name]++
						}
					}
					for _, field := range []string{"execute", "command"} {
						if ref, ok := step[field].(string); ok {
							dependencies[strings.ReplaceAll(ref, "-<<<TENANT_ID>>>", "-sapcp")] = true
						}
					}
					if in, ok := step["input"].(map[string]any); ok {
						if ref, ok := in["command"].(string); ok {
							dependencies[strings.ReplaceAll(ref, "-<<<TENANT_ID>>>", "-sapcp")] = true
						}
					}
				}
				families[filepath.Base(filepath.Dir(path))+":"+fmt.Sprint(object["name"])] = true
				commands = append(commands, map[string]any{"path": rel, "sha256": files[rel], "id": id, "normalized_id": strings.ReplaceAll(id, "-<<<TENANT_ID>>>", "-sapcp"), "description": object["description"], "hidden": hidden, "deprecated": deprecated, "executor_count": len(steps), "controls": controls, "dependencies": sortedKeys(dependencies)})
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	for _, command := range commands {
		matches := []string{}
		for _, path := range sortedKeys(testReferences) {
			if contains(testReferences[path], fmt.Sprint(command["normalized_id"])) {
				matches = append(matches, path)
			}
		}
		command["literal_test_reference_paths"] = matches
	}
	return writeJSON(out, map[string]any{"schema_version": 1, "source_revision": strings.TrimSpace(string(revision)), "source_sha256": files, "source_digest": digest(files), "counts": map[string]int{"catalogs": catalogs, "commands": len(commands), "command_families": len(families), "inputs": inputs, "public_test_commands": tests, "OQ_named_files": oq}, "commands": commands, "limitations": []string{"Literal test references do not prove test coverage or successful runtime.", "Hidden and deprecated sources require explicit selection review.", "No stored Input values are exported."}})
}
