package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFailedClaudeGradeKeepsStructuredReasonAndAccounting(t *testing.T) {
	t.Setenv("EVAL_TEST_API_KEY", "environment-secret")
	event := map[string]any{"is_error": true, "subtype": "error_max_budget_usd", "errors": []any{"Budget exceeded; credential environment-secret; Authorization: Bearer private-bearer; token=private-token"}, "total_cost_usd": .12, "usage": map[string]any{"input_tokens": 300}, "modelUsage": map[string]any{"configured-grader": map[string]any{}}}
	program := fakeCLI(t, "cat >/dev/null\ncat <<'RESULT'\n"+string(mustJSON(event))+"\nRESULT\nexit 1\n")
	c := evalCase{Criteria: []criterion{{ID: "one"}}}
	_, meta := externalGrade(context.Background(), hostConfig{Kind: "claude", Program: program, Isolation: "native"}, c, evidence{}, .25)
	if meta.Err == nil || !strings.Contains(meta.Err.Error(), "Budget exceeded") || !strings.Contains(meta.Err.Error(), "error_max_budget_usd") || meta.Cost != .12 || !meta.CostKnown || meta.Usage["input_tokens"] != float64(300) || meta.Model != "configured-grader" {
		t.Fatalf("lost structured error/accounting: %+v", meta)
	}
	for _, secret := range []string{"environment-secret", "private-bearer", "private-token"} {
		if strings.Contains(meta.Err.Error(), secret) {
			t.Fatal("credential in report error")
		}
	}
}

func TestNonzeroClaudeExitCannotPassEvenWithStructuredAnswer(t *testing.T) {
	event := map[string]any{"structured_output": semanticReport{}, "total_cost_usd": .03}
	program := fakeCLI(t, "cat >/dev/null\ncat <<'RESULT'\n"+string(mustJSON(event))+"\nRESULT\nexit 1\n")
	_, meta := externalGrade(context.Background(), hostConfig{Kind: "claude", Program: program, Isolation: "native"}, evalCase{Criteria: []criterion{{ID: "one"}}}, evidence{}, .25)
	if meta.Err == nil || meta.Cost != .03 || !meta.CostKnown {
		t.Fatal(meta)
	}
}

func TestGraderCitationRetryAccumulatesUsageAndUsesSmallSchema(t *testing.T) {
	dir := t.TempDir()
	count, schema := filepath.Join(dir, "first"), filepath.Join(dir, "schema")
	first := map[string]any{"structured_output": semanticReport{Judgments: []semanticJudgment{{ID: "one", Pass: true, Quote: "invented quotation", Reason: "observed"}}}, "total_cost_usd": .02, "usage": map[string]any{"input_tokens": 10, "output_tokens": 3, "cache_creation": map[string]any{"ephemeral_5m_input_tokens": 2}}}
	second := map[string]any{"structured_output": semanticReport{Judgments: []semanticJudgment{{ID: "one", Pass: true, Quote: "observed candidate answer", Reason: "observed"}}}, "total_cost_usd": .03, "usage": map[string]any{"input_tokens": 20, "output_tokens": 4, "cache_creation": map[string]any{"ephemeral_5m_input_tokens": 5}}}
	program := fakeCLI(t, "cat >/dev/null\nfor arg do printf '%s\n' \"$arg\"; done > \""+schema+"\"\nif [ -f \""+count+"\" ]; then\ncat <<'RESULT'\n"+string(mustJSON(second))+"\nRESULT\nelse\n: > \""+count+"\"\ncat <<'RESULT'\n"+string(mustJSON(first))+"\nRESULT\nfi\n")
	c := evalCase{Criteria: []criterion{{ID: "one"}}}
	report, meta := externalGrade(context.Background(), hostConfig{Kind: "claude", Program: program, Isolation: "native"}, c, evidence{Answer: "observed candidate answer"}, .25)
	if meta.Err != nil || len(report.Judgments) != 1 || meta.Cost != .05 || meta.Usage["input_tokens"] != float64(30) || meta.Usage["output_tokens"] != float64(7) || meta.Usage["cache_creation"].(map[string]any)["ephemeral_5m_input_tokens"] != float64(7) {
		t.Fatal(meta)
	}
	args, _ := os.ReadFile(schema)
	if strings.Contains(string(args), `"enum"`) || strings.Contains(string(args), "observed candidate answer") {
		t.Fatal("evidence duplicated into command-line schema")
	}
}

func TestCompactGradeRetainsActionsButExcludesHostMetadataAndPrivateState(t *testing.T) {
	e := evidence{Answer: "final answer", State: map[string]any{"secret": "private unobserved state"}, Trace: []json.RawMessage{
		mustJSON(map[string]any{"type": "system", "debug": strings.Repeat("provider metadata ", 1000)}),
		mustJSON(map[string]any{"type": "assistant", "message": map[string]any{"content": []any{map[string]any{"type": "text", "text": "intermediate decision"}, map[string]any{"type": "tool_use", "id": "one", "name": "Bash", "input": map[string]any{"command": "scripts/autopi lint candidate.json"}}}}}),
		mustJSON(map[string]any{"type": "user", "message": map[string]any{"content": []any{map[string]any{"type": "tool_result", "tool_use_id": "one", "content": "No lint findings"}}}}),
	}}
	c := evalCase{Files: map[string]string{"contract": "public contract"}, Criteria: []criterion{{ID: "one"}}}
	text := string(mustJSON(gradingTask(c, e)))
	for _, wanted := range []string{"final answer", "intermediate decision", "scripts/autopi lint", "No lint findings", "public contract"} {
		if !strings.Contains(text, wanted) {
			t.Fatal("lost observable evidence", wanted)
		}
	}
	if strings.Contains(text, "provider metadata") || strings.Contains(text, "private unobserved state") || len(text)*4 >= len(mustJSON(e)) {
		t.Fatal("metadata/private state reached grader")
	}
	for _, quote := range []string{"intermediate decision", "No lint findings", "provider metadata"} {
		_, err := semanticResults(c, e, semanticReport{Judgments: []semanticJudgment{{ID: "one", Pass: true, Quote: quote, Reason: "observed"}}})
		if (err != nil) != (quote == "provider metadata") {
			t.Fatal("citation evidence boundary failed", quote, err)
		}
	}
}

func TestCalibrationReuseRejectsChangedOrMissingGraderImplementation(t *testing.T) {
	c := findCase(t, "checksum-recovery")
	b := bank{Version: 1, Cases: []evalCase{c}}
	h := hostConfig{Kind: "claude", Isolation: "native"}
	path := filepath.Join(t.TempDir(), "calibration.json")
	if err := writeJSON(strings.TrimSuffix(path, ".json")+".bank.json", b); err != nil {
		t.Fatal(err)
	}
	for _, fingerprint := range []string{"", "old-implementation", gradingProtocolDigest()} {
		previous := map[string]any{"host": h.Kind, "model": h.Model, "grader_effort": h.Effort, "isolation": h.Isolation, "config_digest": hostConfigDigest(h), "grader_digest": fingerprint, "bank_digest": digest(b), "results": []any{map[string]any{"case": c.ID, "candidate": "correct-reference", "status": "PASS"}}}
		if err := writeJSON(path, previous); err != nil {
			t.Fatal(err)
		}
		rows, err := calibrationReuse(b, h, path)
		if fingerprint == gradingProtocolDigest() {
			if err != nil || len(rows) != 1 {
				t.Fatal(rows, err)
			}
		} else if err == nil || !strings.Contains(err.Error(), "grader implementation") {
			t.Fatal("stale grader results reused", err)
		}
	}
}

func TestCalibrationReuseRequiresRecordedMatchingEffort(t *testing.T) {
	c := findCase(t, "checksum-recovery")
	b := bank{Version: 1, Cases: []evalCase{c}}
	h := hostConfig{Kind: "claude", Model: "sonnet", Effort: "high", Isolation: "native"}
	path := filepath.Join(t.TempDir(), "calibration.json")
	if err := writeJSON(strings.TrimSuffix(path, ".json")+".bank.json", b); err != nil {
		t.Fatal(err)
	}
	for _, recorded := range []any{nil, "", "low", "high"} {
		previous := map[string]any{"host": h.Kind, "model": h.Model, "isolation": h.Isolation, "config_digest": hostConfigDigest(h), "grader_digest": gradingProtocolDigest(), "bank_digest": digest(b), "results": []any{map[string]any{"case": c.ID, "candidate": "correct-reference", "status": "PASS"}}}
		if recorded != nil {
			previous["grader_effort"] = recorded
		}
		if err := writeJSON(path, previous); err != nil {
			t.Fatal(err)
		}
		rows, err := calibrationReuse(b, h, path)
		if recorded == "high" {
			if err != nil || len(rows) != 1 {
				t.Fatal(rows, err)
			}
		} else if err == nil || !strings.Contains(err.Error(), "effort") {
			t.Fatal("unrecorded or different effort reused", recorded, err)
		}
	}
}
