package e2e

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicTaskContainsContractsButNoReferenceImplementations(t *testing.T) {
	repo, _ := filepath.Abs("../../..")
	task, err := PublicTask(repo)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(task)
	if strings.Contains(string(data), "\"executors\":") || strings.Contains(string(data), "\"autoRetry\":") {
		t.Fatal("reference implementation exposed")
	}
	roles := task["resources"].([]map[string]any)
	for _, role := range roles {
		if _, ok := role["configuration"]; ok {
			t.Fatal("executor configuration exposed")
		}
	}
	if len(roles) != 19 || len(task["scenarios"].([]scenario)) != 29 {
		t.Fatal("missing public requirements")
	}
}
func TestGeneratedDefinitionsUseCandidateDirectoryAndSkipReferenceMutationCalibration(t *testing.T) {
	repo, _ := filepath.Abs("../../..")
	dir := t.TempDir()
	defs := filepath.Join(dir, "definitions")
	if err := os.Mkdir(defs, 0700); err != nil {
		t.Fatal(err)
	}
	files, _ := os.ReadDir("definitions")
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join("definitions", file.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(defs, file.Name()), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	r, err := Open(context.Background(), Options{Repository: repo, Directory: filepath.Join(dir, "run"), Definitions: defs, Tenant: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	if r.h.definitionPath("FetchPost.command.json") != filepath.Join(defs, "FetchPost.command.json") || len(r.h.spec.Negative) != 0 {
		t.Fatal("candidate checked against supplied implementation")
	}
	if len(r.Results().Scenarios) != 29 {
		t.Fatal("acceptance assertions omitted")
	}
	if _, err := Open(context.Background(), Options{Repository: repo, Directory: filepath.Join(dir, "run"), Definitions: defs, Tenant: "other-tenant"}); err == nil {
		t.Fatal("cross-tenant cleanup or resume accepted")
	}
}

func TestReferenceDefinitionsPassCurrentBundledLintAndBindingChecks(t *testing.T) {
	repo, _ := filepath.Abs("../../..")
	dir := t.TempDir()
	r, err := Open(context.Background(), Options{Repository: repo, Directory: dir, Reference: true})
	if err != nil {
		t.Fatal(err)
	}
	if err = r.Execute(context.Background(), "check", ""); err != nil {
		entries, _ := os.ReadDir(filepath.Join(dir, "evidence"))
		for _, entry := range entries {
			if strings.Contains(entry.Name(), "authored-lint") {
				data, _ := os.ReadFile(filepath.Join(dir, "evidence", entry.Name()))
				t.Log(string(data))
			}
		}
		t.Fatal(err)
	}
}

func copyReferenceCandidates(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	entries, err := os.ReadDir("definitions")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join("definitions", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, entry.Name()), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}
func TestCandidateDependencyAndEndpointChangesFailBeforeJournalCreation(t *testing.T) {
	repo, _ := filepath.Abs("../../..")
	for _, kind := range []string{"executor", "iterator", "listener", "mcp", "endpoint"} {
		t.Run(kind, func(t *testing.T) {
			defs := copyReferenceCandidates(t)
			file := "HttpControl.command.json"
			if kind == "iterator" {
				file = "ComparePosts.command.json"
			}
			if kind == "mcp" {
				file = "public.mcp.json"
			}
			if kind == "endpoint" {
				file = "ApiConfiguration.input.json"
			}
			var candidate map[string]any
			if err := read(filepath.Join(defs, file), &candidate); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "executor":
				object(object(candidate["configuration"])["executors"].([]any)[0])["execute"] = "unrelated-tenant:Command:1"
			case "iterator":
				object(object(object(candidate["configuration"])["executors"].([]any)[0])["input"])["command"] = "unrelated-tenant:Command:1"
			case "listener":
				object(object(candidate["configuration"])["listeners"].([]any)[0])["execute"] = "unrelated-tenant:Command:1"
			case "mcp":
				object(candidate["mcpTools"].([]any)[0])["commandId"] = "unrelated-tenant:Command:1"
			case "endpoint":
				for key := range object(candidate["values"]) {
					object(candidate["values"])[key] = "https://outside.invalid"
					break
				}
			}
			if err := write(filepath.Join(defs, file), candidate); err != nil {
				t.Fatal(err)
			}
			journal := filepath.Join(t.TempDir(), "not-created")
			if _, err := Open(context.Background(), Options{Repository: repo, Directory: journal, Definitions: defs, Tenant: "fixture"}); err == nil {
				t.Fatal("out-of-scope candidate accepted")
			}
			if _, err := os.Stat(filepath.Join(journal, "state.json")); !os.IsNotExist(err) {
				t.Fatal("invalid candidate created state")
			}
		})
	}
}
func TestCleanupResumeDoesNotRequireOriginalCandidateFiles(t *testing.T) {
	repo, _ := filepath.Abs("../../..")
	defs := copyReferenceCandidates(t)
	dir := t.TempDir()
	if _, err := Open(context.Background(), Options{Repository: repo, Directory: dir, Definitions: defs, Tenant: "fixture"}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(defs); err != nil {
		t.Fatal(err)
	}
	r, err := Resume(context.Background(), repo, dir, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if !r.cleanupOnly {
		t.Fatal("resume can perform new work")
	}
	if err := r.Execute(context.Background(), "deploy", ""); err == nil {
		t.Fatal("cleanup resume replayed deployment")
	}
}
func TestAPIEvidenceRedactsKnownEnvironmentSecrets(t *testing.T) {
	secret := "review-account-token"
	t.Setenv("REVIEW_API_KEY", secret)
	h := failureHarness(t)
	h.runCommand = func(command *exec.Cmd) error {
		_, err := command.Stdout.Write([]byte(`{"token":"` + secret + `","id":"safe-id"}`))
		return err
	}
	response, err := h.call("synthetic-read", []string{"connection", "info"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(response)
	if strings.Contains(string(raw), secret) || !strings.Contains(string(raw), "safe-id") {
		t.Fatal("API response was not redacted")
	}
	entries, _ := os.ReadDir(filepath.Join(h.dir, "evidence"))
	for _, entry := range entries {
		data, _ := os.ReadFile(filepath.Join(h.dir, "evidence", entry.Name()))
		if strings.Contains(string(data), secret) {
			t.Fatal("API evidence contains a secret")
		}
	}
}
