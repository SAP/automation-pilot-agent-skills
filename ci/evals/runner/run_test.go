package main

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoKeyStillLaunchesHostAndLocalSelectionDoesNotContactEngine(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	repository, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(t.TempDir(), "missing-key-run")
	b := testBank(t)
	local := selectCases(b, "pilot", "", "local")[0]
	h := hostConfig{Kind: "claude", Program: "/usr/bin/true", Isolation: "native"}
	report, err := runBank(context.Background(), b, runOptions{Repository: repository, Directory: directory, Suite: "pilot", Case: local.ID, AgentHost: h, GraderHost: h, Ablation: "with-skills", Tenant: "must-not-contact", Runs: 1, Budget: 10, PerCase: 1, PerGrade: 0.25})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 160 {
		t.Fatal("incomplete coverage disappeared", len(report.Results))
	}
	selected := 0
	for _, r := range report.Results {
		if r.Status == "PASS" || r.CostUSD != 0 {
			t.Fatal("unrun result scored or billed")
		}
		if r.Generation > 0 {
			selected++
			if r.Status != "EVALUATOR_ERROR" || r.EvidenceFile == "" {
				t.Fatal("CLI was not launched without an API key", r)
			}
		}
	}
	if selected != 1 {
		t.Fatal(selected)
	}
	if _, err = os.Stat(filepath.Join(directory, "engine-journal.json")); !os.IsNotExist(err) {
		t.Fatal("engine operation initialized for a local-only selection")
	}
	if _, err = os.Stat(filepath.Join(directory, "report.json")); err != nil {
		t.Fatal(err)
	}
}
func TestFinitePositiveCostsOnly(t *testing.T) {
	for _, value := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if validCost(value) {
			t.Fatal("unsafe budget", value)
		}
	}
	if !validCost(10) {
		t.Fatal("valid budget rejected")
	}
}

func TestCodexStartupDiagnosticsAreSavedInReport(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())
	t.Setenv("EVAL_TEST_API_KEY", "environment-secret")
	program := fakeCLI(t, "cat >/dev/null\nprintf '%s\\n' 'Error: invalid transport in mcp_servers.node_repl; api_key=environment-secret' >&2\nexit 1\n")
	h := hostConfig{Kind: "codex", Program: program, Isolation: "native"}
	repository, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(t.TempDir(), "startup-error")
	report, err := runBank(context.Background(), testBank(t), runOptions{Repository: repository, Directory: directory, Suite: "smoketest", Case: "proposal-scope", AgentHost: h, GraderHost: h, Ablation: "with-skills", Runs: 1, PerCase: 1, PerGrade: .25})
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range report.Results {
		if result.Generation == 0 {
			continue
		}
		if result.Status != "EVALUATOR_ERROR" || !strings.Contains(result.Reason, "invalid transport") || strings.Contains(result.Reason, "environment-secret") || result.ToolCalls != 0 {
			t.Fatal("incorrect persisted startup result", result)
		}
	}
	raw, err := os.ReadFile(filepath.Join(directory, "report.json"))
	if err != nil || !strings.Contains(string(raw), "invalid transport") || strings.Contains(string(raw), "environment-secret") {
		t.Fatal("diagnostics missing or unredacted in saved report", err)
	}
}
func TestReportComparisonRefusesDifferentBanks(t *testing.T) {
	baseline := runReport{Model: "fixed", GraderModel: "grader", Host: "host", Ablation: "with-skills", BankDigest: "before"}
	path := filepath.Join(t.TempDir(), "baseline.json")
	if err := writeJSON(path, baseline); err != nil {
		t.Fatal(err)
	}
	candidate := baseline
	candidate.BankDigest = "after"
	if printReport(candidate, path) == nil {
		t.Fatal("incomparable bank accepted")
	}
}
