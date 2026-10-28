package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWireOutputsPreserveFalseAndDecodeNestedCollections(t *testing.T) {
	values := map[string]any{"summary": `{"flag":false,"status":"200","posts":[{"id":1},{"id":11}]}`, "count": "0"}
	if valueAt(values, "summary.flag") != false {
		t.Fatal("false was lost")
	}
	if valueAt(values, "summary.posts.1.id") != float64(11) {
		t.Fatal("nested serialized output was not decoded")
	}
	if valueAt(values, "count") != float64(0) {
		t.Fatal("zero was lost")
	}
	if valueAt(values, "summary.status") != "200" {
		t.Fatal("a nested JSON string was incorrectly coerced to a number")
	}
	if valueAt(values, "summary.posts.10.id") != nil {
		t.Fatal("missing array entry should remain missing")
	}
}

func failureHarness(t *testing.T) *harness {
	t.Helper()
	root := t.TempDir()
	return &harness{root: root, cli: "offline-fixture", dir: filepath.Join(root, "run"), state: state{
		Run: "offline", Variables: map[string]string{}, Resources: map[string]string{},
		Attempts: map[string]string{}, Results: map[string]verdict{}, Hashes: map[string]string{},
	}}
}

func TestVerdictPersistenceFailureDoesNotClaimPass(t *testing.T) {
	h := failureHarness(t)
	if err := os.WriteFile(h.dir, []byte("directory obstruction"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := h.record("check", "PASS", "example"); err == nil {
		t.Fatal("storage failure was ignored")
	}
	if _, recorded := h.state.Results["check"]; recorded {
		t.Fatal("an unpersisted pass was retained")
	}
}

func TestSchedulePersistenceFailuresStopSubmissionAndPreserveAmbiguity(t *testing.T) {
	for _, stage := range []string{"prediction", "submitted", "accepted"} {
		t.Run(stage, func(t *testing.T) {
			h := failureHarness(t)
			if err := h.save(); err != nil {
				t.Fatal(err)
			}
			obstruct := func(path string) {
				if err := os.Mkdir(path+".tmp", 0700); err != nil {
					t.Fatal(err)
				}
			}
			if stage == "prediction" {
				obstruct(filepath.Join(h.dir, "schedule-prediction.json"))
			}
			if stage == "submitted" {
				obstruct(filepath.Join(h.dir, "state.json"))
			}
			creates := 0
			h.runCommand = func(command *exec.Cmd) error {
				switch command.Args[2] {
				case "time-zones":
					return json.NewEncoder(command.Stdout).Encode([]string{"Europe/Sofia"})
				case "create":
					creates++
					if stage == "accepted" {
						obstruct(filepath.Join(h.dir, "state.json"))
					}
					return json.NewEncoder(command.Stdout).Encode(map[string]string{"id": "schedule-id"})
				case "disable":
					if stage != "accepted" || command.Args[3] != "schedule-id" {
						t.Fatalf("unexpected schedule cleanup: %v", command.Args)
					}
					return nil
				}
				t.Fatalf("unexpected CLI operation after persistence failure: %v", command.Args)
				return nil
			}
			err := h.schedule()
			if err == nil {
				t.Fatal("storage failure was ignored")
			}
			if stage != "accepted" {
				if creates != 0 {
					t.Fatal("schedule created without durable submission evidence")
				}
				if h.state.Attempts["schedule"] != "" {
					t.Fatal("a request that was never submitted left an ambiguous marker")
				}
				return
			}
			if creates != 1 || !strings.Contains(err.Error(), "schedule-id") || !strings.Contains(err.Error(), "reconcile") {
				t.Fatalf("acceptance evidence lost: creates=%d err=%v", creates, err)
			}
			var saved state
			if err := read(filepath.Join(h.dir, "state.json"), &saved); err != nil {
				t.Fatal(err)
			}
			if saved.Attempts["schedule"] != "submitted" || saved.Resources["schedule"] != "" {
				t.Fatal("durable ambiguous-submission marker was not preserved")
			}
			h.state = saved
			if err := h.schedule(); err == nil || !strings.Contains(err.Error(), "ambiguous") || creates != 1 {
				t.Fatalf("resume did not stop duplicate creation: %v", err)
			}
		})
	}
}

func TestLifecycleFailuresRestoreOriginalDefinition(t *testing.T) {
	for _, failure := range []string{"rpc", "missing-list", "visible-tool", "restoration"} {
		t.Run(failure, func(t *testing.T) {
			h := failureHarness(t)
			h.state.Resources["public.mcp.json"] = "server-id"
			original := map[string]any{"name": "PublicServer", "enabled": true, "mcpTools": []any{
				map[string]any{"name": "get_post_with_fallback", "enabled": true},
				map[string]any{"name": "another_tool", "enabled": true},
			}}
			if err := write(filepath.Join(h.root, "definitions", "public.mcp.json"), original); err != nil {
				t.Fatal(err)
			}
			updates := 0
			h.runCommand = func(command *exec.Cmd) error {
				if command.Args[1] == "api" {
					switch failure {
					case "missing-list":
						return json.NewEncoder(command.Stdout).Encode(map[string]any{"result": map[string]any{}})
					case "visible-tool":
						return json.NewEncoder(command.Stdout).Encode(map[string]any{"result": map[string]any{"tools": []any{map[string]any{"name": "get_post_with_fallback"}}}})
					}
					return errors.New("simulated RPC failure")
				}
				updates++
				var definition map[string]any
				if err := json.NewDecoder(command.Stdin).Decode(&definition); err != nil {
					t.Fatal(err)
				}
				if updates == 1 {
					if object(definition["mcpTools"].([]any)[0])["enabled"] != false {
						t.Fatal("fixture tool was not disabled")
					}
				} else if !subset(original, definition) {
					t.Fatal("failure cleanup did not restore the complete original definition")
				}
				if updates > 1 && failure == "restoration" {
					return errors.New("simulated restoration failure")
				}
				return json.NewEncoder(command.Stdout).Encode(map[string]string{"id": "server-id"})
			}
			err := h.lifecycle()
			if err == nil || updates != 2 || len(h.state.Results) != 0 {
				t.Fatalf("failure/cleanup evidence invalid: updates=%d results=%v err=%v", updates, h.state.Results, err)
			}
			if failure == "restoration" && (!strings.Contains(err.Error(), "disabled-tool-list") || !strings.Contains(err.Error(), "restoration failed")) {
				t.Fatalf("original or restoration failure was hidden: %v", err)
			}
			var saved state
			if err := read(filepath.Join(h.dir, "state.json"), &saved); err != nil {
				t.Fatal(err)
			}
			if (saved.PendingMCP != nil) != (failure == "restoration") {
				t.Fatalf("restoration intent was incorrectly retained or cleared: %+v", saved.PendingMCP)
			}
		})
	}
}

func TestInterruptedLifecycleRestoresBeforeSchemaAndOnlyAfterTenantValidation(t *testing.T) {
	for _, failure := range []string{"none", "tenant", "restore", "clear-state"} {
		t.Run(failure, func(t *testing.T) {
			h := failureHarness(t)
			h.state.Tenant = "intended"
			h.state.Resources["public.mcp.json"] = "server-id"
			original := map[string]any{"name": "SavedServer", "enabled": true, "mcpTools": []any{map[string]any{"name": "tool", "enabled": true}}}
			if err := h.prepareMCPRestoration("server-id", original); err != nil {
				t.Fatal(err)
			}
			// A new process has only the saved state, not a local deferred cleanup.
			h.state = state{}
			if err := read(filepath.Join(h.dir, "state.json"), &h.state); err != nil {
				t.Fatal(err)
			}
			restored, schemaCalled := false, false
			h.runCommand = func(command *exec.Cmd) error {
				switch command.Args[1] {
				case "connection":
					tenant := "intended"
					if failure == "tenant" {
						tenant = "different"
					}
					return json.NewEncoder(command.Stdout).Encode(map[string]any{"tenantId": tenant, "permissions": []string{"Read", "Write", "Execute"}})
				case "mcp-servers":
					if failure == "tenant" {
						t.Fatal("restored before tenant validation")
					}
					var definition map[string]any
					if err := json.NewDecoder(command.Stdin).Decode(&definition); err != nil {
						t.Fatal(err)
					}
					if command.Args[2] != "update" || command.Args[3] != "server-id" || !subset(original, definition) || !subset(definition, original) {
						t.Fatal("restart did not restore the exact saved definition")
					}
					restored = true
					if failure == "restore" {
						return errors.New("restore rejected")
					}
					if failure == "clear-state" {
						if err := os.Mkdir(filepath.Join(h.dir, "state.json.tmp"), 0700); err != nil {
							t.Fatal(err)
						}
					}
					return nil
				case "api":
					if !restored || h.state.PendingMCP != nil {
						t.Fatal("schema checked before durable restoration")
					}
					schemaCalled = true
					return errors.New("stop after recovery")
				}
				t.Fatalf("unexpected command %v", command.Args)
				return nil
			}
			if err := h.run(); err == nil {
				t.Fatal("expected injected error")
			}
			var saved state
			if err := read(filepath.Join(h.dir, "state.json"), &saved); err != nil {
				t.Fatal(err)
			}
			if schemaCalled != (failure == "none") || (saved.PendingMCP == nil) != (failure == "none") || (h.state.PendingMCP == nil) != (failure == "none") {
				t.Fatalf("invalid recovery state: failure=%s schema=%v saved=%+v", failure, schemaCalled, saved.PendingMCP)
			}
		})
	}
}

func TestLifecycleRestorationMustPersistBeforeMutation(t *testing.T) {
	h := failureHarness(t)
	h.state.Resources["public.mcp.json"] = "server-id"
	original := map[string]any{"mcpTools": []any{map[string]any{"name": "get_post_with_fallback", "enabled": true}}}
	if err := write(filepath.Join(h.root, "definitions", "public.mcp.json"), original); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(h.dir, []byte("obstruction"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := h.lifecycle(); err == nil || h.calls != 0 || h.state.PendingMCP != nil {
		t.Fatalf("mutation proceeded without saved restoration: calls=%d err=%v", h.calls, err)
	}
}

func TestFailedScheduleReadbackUsesConditionalDisableAndRetainsBothErrors(t *testing.T) {
	for _, cleanupFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "cleanup", true: "cleanup-fails"}[cleanupFailure], func(t *testing.T) {
			h := failureHarness(t)
			h.state.Resources["schedule"] = "schedule-id"
			if err := write(filepath.Join(h.dir, "schedule-prediction.json"), map[string]any{"definition": map[string]any{"enabled": true}}); err != nil {
				t.Fatal(err)
			}
			disables := 0
			h.runCommand = func(command *exec.Cmd) error {
				switch command.Args[2] {
				case "time-zones":
					return json.NewEncoder(command.Stdout).Encode([]string{"Europe/Sofia"})
				case "get":
					return errors.New("readback failed")
				case "disable":
					if command.Args[3] != "schedule-id" {
						t.Fatal("disabled a different schedule")
					}
					disables++
					if cleanupFailure {
						return errors.New("disable rejected")
					}
					return nil
				}
				t.Fatalf("unexpected CLI operation: %v", command.Args)
				return nil
			}
			err := h.schedule()
			if err == nil || !strings.Contains(err.Error(), "schedule-readback") || disables != 1 {
				t.Fatalf("failed schedule was not disabled once: %v", err)
			}
			if cleanupFailure && !strings.Contains(err.Error(), "cannot disable schedule schedule-id") {
				t.Fatalf("cleanup error was hidden: %v", err)
			}
		})
	}
}

func TestMCPSuccessfulTransportCannotHideFailedExecution(t *testing.T) {
	var response any
	if err := json.Unmarshal([]byte(`{"result":{"isError":false,"content":[{"type":"text","text":"{\"executionId\":\"test-run\",\"status\":\"FAILED\"}"}]}}`), &response); err != nil {
		t.Fatal(err)
	}
	if str(execution(response)["status"]) != "FAILED" {
		t.Fatal("transport envelope hid terminal failure")
	}
}

func TestReadbackDetectsChangedContractWhileAllowingServiceFields(t *testing.T) {
	expected := map[string]any{"name": "Example", "inputKeys": map[string]any{"id": map[string]any{"type": "number", "sensitive": false}}}
	actual := map[string]any{"name": "Example", "createdAt": "service-owned", "inputKeys": map[string]any{"id": map[string]any{"type": "number", "sensitive": false, "defaultValue": nil}}}
	if !subset(expected, actual) {
		t.Fatal("service-owned extras should not invalidate submitted fields")
	}
	object(object(actual["inputKeys"])["id"])["sensitive"] = true
	if subset(expected, actual) {
		t.Fatal("changed sensitivity was accepted")
	}
}

func TestLocallyBuiltScheduleMatchesDecodedJSONReadback(t *testing.T) {
	expected := map[string]any{"inputReferences": []string{"catalog:Input:1"}, "tags": map[string]string{"case": "once"}, "schedule": map[string]any{"once": map[string]int{"month": 1, "day": 15, "hour": 12, "minute": 30}}}
	b, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	var actual any
	if err = json.Unmarshal(b, &actual); err != nil {
		t.Fatal(err)
	}
	if !subset(expected, actual) {
		t.Fatal("typed Go payload was falsely rejected against equivalent JSON")
	}
	object(object(object(actual)["schedule"])["once"])["minute"] = float64(45)
	if subset(expected, actual) {
		t.Fatal("changed trigger minute was accepted")
	}
}

func TestAmbiguousSubmissionCannotBeTriggeredAgain(t *testing.T) {
	h := harness{cli: "must-never-be-invoked", dir: t.TempDir(), state: state{Attempts: map[string]string{"once": "submitted"}, Results: map[string]verdict{}}}
	if err := h.runCase(scenario{ID: "once"}); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity rejection, got %v", err)
	}
	if h.calls != 0 {
		t.Fatal("ambiguity generated another CLI call")
	}
}

func TestReportSeparatesBlockedDeploymentFromUnexecutedCases(t *testing.T) {
	h := harness{dir: t.TempDir(), spec: suite{Cases: []scenario{{ID: "one"}, {ID: "two"}}}, state: state{Run: "test", Results: map[string]verdict{"setup": {Status: "BLOCKED", Detail: "network unavailable"}}}}
	if err := h.report(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(h.dir, "REPORT.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "0 passed, 0 failed, 1 blocked") || !strings.Contains(string(b), "2 of 2") {
		t.Fatal("pending cases were incorrectly scored")
	}
}

func TestNamespaceRewritesCandidateReferencesAndPreservesBuiltins(t *testing.T) {
	h := harness{names: []string{"FetchPost", "EvalApiConfiguration"}, state: state{Namespace: "RoundTwo", Variables: map[string]string{"CORE": "evaluation-TENANT"}}}
	if got := h.replace("<CORE>:FetchPost:1"); got != "evaluation-TENANT:RoundTwoFetchPost:1" {
		t.Fatalf("reference: %s", got)
	}
	if got := h.replace("utils-sapcp:Void:1"); got != "utils-sapcp:Void:1" {
		t.Fatalf("builtin changed: %s", got)
	}
	h.root = t.TempDir()
	if err := write(filepath.Join(h.root, "definitions", "FetchPost.command.json"), map[string]any{"name": "FetchPost", "catalog": "<CORE>", "id": "<CORE>:FetchPost:1"}); err != nil {
		t.Fatal(err)
	}
	d, err := h.definition("FetchPost.command.json")
	if err != nil {
		t.Fatal(err)
	}
	if str(d["name"]) != "RoundTwoFetchPost" || str(d["id"]) != "evaluation-TENANT:RoundTwoFetchPost:1" {
		t.Fatal("name and identity diverged")
	}
}

func TestEvaluationTenantRequiresExplicitLiveTarget(t *testing.T) {
	for _, mode := range []string{"deploy", "run"} {
		for _, requested := range []string{"", " "} {
			if _, err := evaluationTenant(mode, requested, "SAVED_TENANT"); err == nil {
				t.Fatalf("%s accepted an omitted tenant with saved state", mode)
			}
		}
		if got, err := evaluationTenant(mode, "REQUESTED_TENANT", "SAVED_TENANT"); err != nil || got != "REQUESTED_TENANT" {
			t.Fatalf("explicit %s target: %q, %v", mode, got, err)
		}
	}
	for _, mode := range []string{"check", "report"} {
		if got, err := evaluationTenant(mode, "", ""); err != nil || got != "TENANT" {
			t.Fatalf("offline %s target: %q, %v", mode, got, err)
		}
		if got, err := evaluationTenant(mode, "", "SAVED_TENANT"); err != nil || got != "SAVED_TENANT" {
			t.Fatalf("saved %s target: %q, %v", mode, got, err)
		}
	}
}
