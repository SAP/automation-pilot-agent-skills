package main

import (
	"context"
	"path/filepath"
	"testing"
)

func TestCapturedReviewPreservesOriginalRunAndNeverLaunchesCandidate(t *testing.T) {
	c := findCase(t, "checksum-recovery")
	b := bank{Version: 1, Cases: []evalCase{c}}
	source := t.TempDir()
	out := filepath.Join(t.TempDir(), "reviewed")
	e := c.Reference
	quote := quotationLines(e)[0]
	judgments := []semanticJudgment{}
	for _, criterion := range c.Criteria {
		judgments = append(judgments, semanticJudgment{ID: criterion.ID, Pass: true, Quote: quote, Reason: "valid recovery recommendation"})
	}
	output := map[string]any{"structured_output": semanticReport{Judgments: judgments}, "total_cost_usd": 0.01, "usage": map[string]any{"input_tokens": 17}}
	program := fakeCLI(t, "cat >/dev/null"+"\n"+"cat <<'RESULT'"+"\n"+string(mustJSON(output))+"\n"+"RESULT"+"\n")
	h := hostConfig{Kind: "claude", Program: program, Isolation: "native"}
	original := runReport{Version: 2, ID: "original", BankDigest: digest(b), Results: []caseResult{{ID: c.ID, CaseDigest: digest(c), Generation: 1, Primary: c.Primary, Tier: c.Tier, Status: "FAIL", EvidenceFile: "evidence.json"}}}
	original.Results[0].GraderUsage = map[string]any{"input_tokens": float64(5)}
	for name, value := range map[string]any{"bank.snapshot.json": b, "report.json": original, "evidence.json": e} {
		if err := writeJSON(filepath.Join(source, name), value); err != nil {
			t.Fatal(err)
		}
	}
	if err := reviewCapturedRun(context.Background(), b, "unused-repository", source, out, "", h, 1, 1); err != nil {
		t.Fatal(err)
	}
	var after runReport
	if err := readJSON(filepath.Join(source, "report.json"), &after); err != nil {
		t.Fatal(err)
	}
	if after.Results[0].Status != "FAIL" {
		t.Fatal("original report overwritten")
	}
	if err := readJSON(filepath.Join(out, "report.json"), &after); err != nil {
		t.Fatal(err)
	}
	if after.Results[0].Status != "PASS" || after.Results[0].OriginalStatus != "FAIL" || after.OriginalReport == "" {
		t.Fatal("review provenance lost", after)
	}
	if after.Results[0].GraderUsage["input_tokens"] != float64(22) || after.GraderDigest != gradingProtocolDigest() {
		t.Fatal("review usage or grader identity lost", after)
	}
}
