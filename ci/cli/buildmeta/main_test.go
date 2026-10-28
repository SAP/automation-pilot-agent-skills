package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSourceDigestIncludesRelocatedBuildTools(t *testing.T) {
	repository := t.TempDir()
	module := filepath.Join(repository, "cli")
	files := []string{
		"cli/main.go", "cli/internal/contract/snapshot.json", "cli/.go-version", "cli/Makefile",
		"ci/cli/build.sh", "ci/cli/buildmeta/main.go", "ci/cli/contracts/snapshot.py",
		"ci/cli/contracts/helpdocs.py",
	}
	writeFile := func(name, content string) {
		t.Helper()
		path := filepath.Join(repository, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range files {
		writeFile(name, "original")
	}
	before, err := sourceDigest(module)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			writeFile(name, "changed")
			after, err := sourceDigest(module)
			if err != nil || after == before {
				t.Fatalf("changed build input was omitted from provenance: %v", err)
			}
			writeFile(name, "original")
		})
	}
	for _, name := range []string{"cli/dist/scratch.go", "ci/check_links.py", "ci/evals/skills-e2e/main.go", "ci/evals/skills-e2e/runs/local/state.json"} {
		writeFile(name, "unrelated to binary build")
	}
	after, err := sourceDigest(module)
	if err != nil || after != before {
		t.Fatalf("evaluation or unrelated maintenance files changed CLI provenance: %v", err)
	}
	// A partial checkout must fail rather than silently omit external build tooling.
	partial := filepath.Join(t.TempDir(), "cli")
	if err := os.Mkdir(partial, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := sourceDigest(partial); err == nil {
		t.Fatal("missing build tooling directory was silently ignored")
	}
}
