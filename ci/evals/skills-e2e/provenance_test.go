package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvidenceDistinguishesChangedSkillsAndIgnoresRunOutput(t *testing.T) {
	repository := t.TempDir()
	for _, dir := range []string{"skills/fixture", "ci/evals/skills-e2e/runs/local"} {
		if err := os.MkdirAll(filepath.Join(repository, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	file := filepath.Join(repository, "skills/fixture/SKILL.md")
	if err := os.WriteFile(file, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	before, _, err := evidenceSources(repository)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "ci/evals/skills-e2e/runs/local/evidence.json"), []byte("private run"), 0600); err != nil {
		t.Fatal(err)
	}
	after, _, err := evidenceSources(repository)
	if err != nil || before != after {
		t.Fatal("run output changed source identity", err)
	}
	if err := os.WriteFile(file, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	after, _, err = evidenceSources(repository)
	if err != nil || before == after {
		t.Fatal("skill change missing from evidence identity", err)
	}
}
