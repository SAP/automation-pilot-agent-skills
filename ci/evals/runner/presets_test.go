package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPresetCountsNestingAndPrimaryCoverage(t *testing.T) {
	b := testBank(t)
	for suite, count := range map[string]int{"smoketest": 12, "minimal": 20, "full": 160, "pilot": 20, "smoke": 40, "all": 160, "regression": 160, "release": 160} {
		if got := len(selectCases(b, suite, "", "")); got != count {
			t.Fatalf("%s: got %d, want %d", suite, got, count)
		}
	}
	ids := func(suite string) map[string]bool {
		out := map[string]bool{}
		for _, c := range selectCases(b, suite, "", "") {
			out[c.ID] = true
		}
		return out
	}
	minimal, full, pilot := ids("minimal"), ids("full"), ids("pilot")
	for _, id := range []string{
		"catalog-update-preserve-metadata", "checksum-recovery", "connection-tenant-mismatch", "discover-cf-start-completion",
		"hana-noop-side-effect", "http-timeout-success-policy", "iflow-empty-result-fallback", "input-secret-value-rotation",
		"local-authoring", "mcp-jsonrpc-result-validation", "mcp-session-cleanup-listeners", "native-iterator-wrapper",
		"plan-single-instance-availability", "preserve-policy-false", "proposal-scope", "publish-mcp-bound-tool",
		"recover-cf-accepted-run", "schedule-paired-weekday-times", "sm-write-only-parameters", "tag-one-run-priority-logging",
	} {
		if !minimal[id] {
			t.Fatal("original pilot case replaced", id)
		}
	}
	core := ids("smoketest")
	for _, id := range []string{
		"local-authoring", "connection-tenant-mismatch", "proposal-scope", "discover-cf-start-completion",
		"catalog-update-preserve-metadata", "tag-one-run-priority-logging", "input-secret-value-rotation",
		"schedule-paired-weekday-times", "publish-mcp-bound-tool", "recover-cf-accepted-run",
		"preserve-policy-false", "http-timeout-success-policy",
	} {
		if !core[id] {
			t.Fatal("planned core case replaced", id)
		}
	}
	for id := range minimal {
		if !pilot[id] || !full[id] {
			t.Fatal("minimal membership changed", id)
		}
	}
	counts, tiers := map[string]int{}, map[string]int{}
	for _, c := range selectCases(b, "smoketest", "", "") {
		if !minimal[c.ID] {
			t.Fatal("smoketest is not nested", c.ID)
		}
		counts[c.Primary]++
		tiers[c.Tier]++
	}
	if len(counts) != 12 || tiers["local"] != 4 || tiers["fixture"] != 7 || tiers["engine"] != 1 {
		t.Fatal(counts, tiers)
	}
	for skill, n := range counts {
		if n != 1 {
			t.Fatal("duplicate primary skill", skill, n)
		}
	}
	for suite, count := range map[string]int{"smoketest": 11, "minimal": 18, "full": 136} {
		if len(selectCases(b, suite, "", "local,fixture")) != count {
			t.Fatal("filter count", suite)
		}
	}
	if len(selectedCases(b, "smoketest", "automation-pilot-cli", "local", "local-authoring")) != 1 {
		t.Fatal("case intersection lost")
	}
	if len(selectedCases(b, "smoketest", "", "fixture", "local-authoring")) != 0 {
		t.Fatal("case escaped tier filter")
	}
}

func TestPresetDefaultsAndGenerationOverrides(t *testing.T) {
	if defaultSuite != "smoketest" {
		t.Fatal("default no longer uses the smallest preset")
	}
	for _, suite := range suiteNames {
		want := 1
		if suite == "release" {
			want = 3
		}
		if runGenerations(suite, 1, false) != want || runGenerations(suite, 1, true) != 1 || runGenerations(suite, 2, true) != 2 {
			t.Fatal("repetition semantics", suite)
		}
	}
}

func TestInvalidSelectionsFailBeforeHostResolutionAndFileCreation(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	for _, args := range [][]string{
		{"run", "--suite", "typo", "--bank", "missing-bank.json"},
		{"list", "--suite", "typo", "--bank", "missing-bank.json"},
		{"calibrate", "--suite", "typo", "--bank", "missing-bank.json"},
		{"run", "--suite", "smoketest", "--case", "checksum-recovery"},
	} {
		dir := filepath.Join(t.TempDir(), "must-not-exist")
		args = append(args, "--out", dir, "--bank", "../cases/bank.json")
		err := dispatch(args)
		if err == nil || (!strings.Contains(err.Error(), "unknown suite") && !strings.Contains(err.Error(), "selection matches no cases")) {
			t.Fatal(args, err)
		}
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatal("invalid selection created output")
		}
	}
}

func TestRunBankDefaultPresetWithLocalFilterNeedsNoTenant(t *testing.T) {
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "")
	t.Setenv("AUTOMATION_PILOT_TOKEN", "")
	repo, _ := filepath.Abs("../../..")
	h := hostConfig{Kind: "claude", Program: "/usr/bin/true", Isolation: "native"}
	r, err := runBank(context.Background(), testBank(t), runOptions{Repository: repo, Directory: filepath.Join(t.TempDir(), "default"), Case: "local-authoring", Tiers: "local", AgentHost: h, GraderHost: h, Ablation: "with-skills", Runs: 1, PerCase: 1, PerGrade: .25})
	if err != nil || r.Suite != "smoketest" || r.Tenant != "" {
		t.Fatal(r.Suite, r.Tenant, err)
	}
	if summarizeReport(r).Selected != 1 {
		t.Fatal("default selection ignored filters")
	}
}
