package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestPackagedPluginIsCompleteAndRejectsDamagedAssets(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if _, e := os.Stat("skills"); e == nil {
		root, err = os.Getwd()
	}
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "plugin")
	if err := stage(root, dest, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "cli")); !os.IsNotExist(err) {
		t.Fatal("runtime package contains contributor source")
	}
	if _, err := os.Stat(filepath.Join(dest, "ci")); !os.IsNotExist(err) {
		t.Fatal("runtime package contains evaluation or maintenance files")
	}
	buildCommands := regexp.MustCompile(`\bgo\s+(run|build|test|install|mod|env|version)\b|ci/cli/build\.sh|\bmake build\b`)
	if err := filepath.WalkDir(filepath.Join(dest, "skills", "automation-pilot-cli"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".go" || entry.Name() == "go.mod" || entry.Name() == "go.sum" {
			t.Errorf("CLI skill ships build source: %s", path)
		}
		if filepath.Ext(path) == ".md" {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if buildCommands.Match(data) {
				t.Errorf("CLI skill tells agents to use Go/build tools: %s", path)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(dest, ".claude-plugin", "plugin.json"))
	if err != nil || bytes.Contains(manifest, []byte("experimental")) {
		t.Fatal("runtime manifest contains development-only configuration", err)
	}
	evaluation := filepath.Join(t.TempDir(), "evaluation")
	if err := stage(root, evaluation, true); err != nil {
		t.Fatal(err)
	}
	manifest, err = os.ReadFile(filepath.Join(evaluation, ".claude-plugin", "plugin.json"))
	if err != nil || !bytes.Contains(manifest, []byte("ci/evals/agent-skills")) {
		t.Fatal("evaluation fixture missing cases", err)
	}
	if _, err := os.Stat(filepath.Join(evaluation, "ci", "evals", "agent-skills", "local-authoring", "prompt.md")); err != nil {
		t.Fatal(err)
	}
	first, second := filepath.Join(t.TempDir(), "first.zip"), filepath.Join(t.TempDir(), "second.zip")
	if err := archive(dest, first); err != nil {
		t.Fatal(err)
	}
	if err := archive(dest, second); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(first)
	b, _ := os.ReadFile(second)
	if !bytes.Equal(a, b) {
		t.Fatal("plugin archive is not reproducible")
	}
	self := filepath.Join(dest, "self.zip")
	if err := archive(dest, self); err == nil {
		t.Fatal("archive accepted its own output as input")
	}
	if _, err := os.Stat(self); !os.IsNotExist(err) {
		t.Fatal("rejected archive created an output file")
	}
	nested := filepath.Join(dest, "skills", "nested")
	if err := stage(dest, nested, false); err == nil {
		t.Fatal("recursive staging was accepted")
	}
	if _, err := os.Stat(nested); !os.IsNotExist(err) {
		t.Fatal("recursive staging changed source")
	}
	resource := filepath.Join(dest, "skills", "automation-pilot-cli", "references", "capabilities.md")
	if err := os.Remove(resource); err != nil {
		t.Fatal(err)
	}
	if err := check(dest); err == nil {
		t.Fatal("missing skill resource was accepted")
	}
}
