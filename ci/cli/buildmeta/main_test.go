package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceDigestIncludesRelocatedBuildTools(t *testing.T) {
	repository := t.TempDir()
	module := filepath.Join(repository, "cli")
	files := []string{
		"cli/main.go", "cli/internal/contract/snapshot.json", "cli/internal/embedded.txt", "cli/.go-version", "cli/Makefile",
		"ci/cli/build.sh", "ci/cli/buildmeta/main.go", "ci/cli/contracts/snapshot/main.go",
		"ci/cli/contracts/helpdocs/main.go",
		"skills/automation-pilot-cli/scripts/autopi",
		"skills/automation-pilot-cli/scripts/autopi.cmd",
		"skills/automation-pilot-cli/scripts/verify-autopi.ps1",
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
	writeFile(".github/workflows/validate.yml", "validation")
	writeFile(".claude-plugin/plugin.json", "plugin")
	writeFile("ci/evals/agent-skills/case/prompt.md", "evaluation")
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
	validationBefore, err := inputDigest(module, true)
	if err != nil {
		t.Fatal(err)
	}
	writeFile("cli/main_test.go", "test-only change")
	validationAfter, err := inputDigest(module, true)
	if err != nil || validationAfter == validationBefore {
		t.Fatal("test change missing from validation digest", err)
	}
	writeFile("ci/.local/review.md", "private report")
	validationWithPrivate, err := inputDigest(module, true)
	if err != nil || validationWithPrivate != validationAfter {
		t.Fatal("private notes changed validation provenance", err)
	}
	for _, name := range []string{"cli/dist/scratch.go", "ci/links/main.go", "ci/evals/skills-e2e/main.go", "ci/evals/skills-e2e/runs/local/state.json", "skills/automation-pilot-cli/scripts/bin/build-manifest.json", "skills/automation-pilot-cli/scripts/bin.stage.fixture/build-manifest.json"} {
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

func TestManifestBindsChecksumsToShippedPaths(t *testing.T) {
	for _, scenario := range []string{"valid", "different checksum list", "missing checksum list", "duplicate checksum entry", "alternate binary path", "parent traversal", "invalid hash", "different source"} {
		t.Run(scenario, func(t *testing.T) {
			bin := t.TempDir()
			m := manifest{SourceSHA256: "source-digest"}
			var sums strings.Builder
			for _, target := range targets {
				a := artifact{target, artifactPath(target), strings.Repeat("a", 64)}
				m.Artifacts = append(m.Artifacts, a)
				fmt.Fprintf(&sums, "%s  %s\n", a.SHA256, a.Path)
			}
			sumText := sums.String()
			switch scenario {
			case "different checksum list":
				sumText = strings.Replace(sumText, strings.Repeat("a", 64), strings.Repeat("b", 64), 1)
			case "duplicate checksum entry":
				sumText += strings.Split(sumText, "\n")[0] + "\n"
			case "alternate binary path":
				m.Artifacts[0].Path = "elsewhere/autopi"
			case "parent traversal":
				m.Artifacts[0].Path = "../darwin-arm64/autopi"
			case "invalid hash":
				m.Artifacts[0].SHA256 = "not-a-hash"
			case "different source":
				m.SourceSHA256 = "another-source"
			}
			b, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(bin, "build-manifest.json"), b, 0600); err != nil {
				t.Fatal(err)
			}
			if scenario != "missing checksum list" {
				if err := os.WriteFile(filepath.Join(bin, "SHA256SUMS"), []byte(sumText), 0600); err != nil {
					t.Fatal(err)
				}
			}
			_, err = loadManifest(bin, "source-digest")
			if (err == nil) != (scenario == "valid") {
				t.Fatalf("unexpected verification result: %v", err)
			}
		})
	}
}
