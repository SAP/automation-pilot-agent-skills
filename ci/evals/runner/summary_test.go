package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummarySeparatesCasesRepairsAndRunFailures(t *testing.T) {
	report := runReport{ID: "saved", Results: []caseResult{
		{ID: "pass", Primary: "authoring", Tier: "local", Generation: 1, Status: "PASS", Review: "reviewed", CostKnown: true, CostUSD: 1, AgentCostUSD: .8, GraderCostUSD: .2, AgentUsage: map[string]any{"input_tokens": float64(20)}},
		{ID: "fail", Primary: "authoring", Tier: "local", Generation: 1, Status: "FAIL", CostUSD: .3, GraderCostUSD: .3, Repairs: []caseResult{{Generation: 1, Status: "PASS", CostKnown: true, CostUSD: .5, AgentCostUSD: .5}}},
		{ID: "excluded", Primary: "discovery", Status: "UNRUN", Reason: "excluded by selection"},
		{ID: "waiting", Primary: "discovery", Tier: "fixture", Generation: 1, Status: "UNRUN", Reason: "budget exhausted"},
		{ID: "engine-cleanup", Status: "EVALUATOR_ERROR", Reason: "pending mutation"},
	}}
	s := summarizeReport(report)
	if s.Selected != 3 || s.Excluded != 1 || s.Pending != 2 || s.Counts["PASS"] != 1 || s.Counts["UNRUN"] != 1 || s.Counts["EVALUATOR_ERROR"] != 0 || s.Repairs["PASS"] != 1 || len(s.RunErrors) != 1 {
		t.Fatalf("incorrect denominators: %+v", s)
	}
	if s.Cost != 1.8 || s.AgentCost != 1.3 || s.GraderCost != .5 || s.Unpriced != 2 {
		t.Fatalf("incorrect accounting: %+v", s)
	}
	if _, ok := s.Skills[""]; ok {
		t.Fatal("cleanup was counted as a skill")
	}
	text := reportMarkdown(report)
	for _, wanted := range []string{"Incomplete verification", "Provisional", "Separate repair outcomes", "engine-cleanup", "$1.80"} {
		if !strings.Contains(text, wanted) {
			t.Fatal("missing report detail", wanted)
		}
	}
}

func TestReportExportIsOfflineAndDoesNotOverwriteEvidence(t *testing.T) {
	root := t.TempDir()
	report := runReport{ID: "existing", Results: []caseResult{{ID: "case", Primary: "skill", Generation: 1, Status: "EVALUATOR_ERROR", Reason: "provider failed"}}}
	path := filepath.Join(root, "report.json")
	if err := writeJSON(path, report); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	// No CLI or bank is present. The report command must not try to resolve one.
	t.Setenv("PATH", t.TempDir())
	summary := filepath.Join(root, "summary.md")
	if err := dispatch([]string{"report", "--run", root, "--out", summary}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(summary)
	if err != nil || !strings.Contains(string(body), "provider failed") {
		t.Fatal(string(body), err)
	}
	if err := dispatch([]string{"report", "--run", root, "--out", path}); err == nil {
		t.Fatal("saved JSON overwritten")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("original report changed")
	}
}

func TestReviewedSummaryExplainsCumulativeCost(t *testing.T) {
	text := reportMarkdown(runReport{OriginalReport: "original/report.json"})
	if !strings.Contains(text, "do not add the original report total again") || !strings.Contains(text, "Historical grader usage may omit earlier attempts") {
		t.Fatal(text)
	}
}
