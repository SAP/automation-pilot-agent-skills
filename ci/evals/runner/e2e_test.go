package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	live "github.com/SAP/automation-pilot-agent-skills/ci/evals/e2e"
	"github.com/SAP/automation-pilot-agent-skills/ci/evals/internal/redact"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeE2EHarness struct {
	phases  []string
	fail    string
	results live.Acceptance
}

func (h *fakeE2EHarness) Execute(_ context.Context, phase, _ string) error {
	h.phases = append(h.phases, phase)
	if phase == h.fail {
		return errors.New("deliberate " + phase + " failure")
	}
	return nil
}
func (h *fakeE2EHarness) Results() live.Acceptance { return h.results }
func passingAcceptance() live.Acceptance {
	return live.Acceptance{Scenarios: []string{"scenario"}, Required: []string{"deploy"}, Checks: map[string]live.Check{"scenario": {Status: "PASS"}, "deploy": {Status: "PASS"}}}
}

func TestE2EPhasesAndReferenceModeNeedNoAgent(t *testing.T) {
	h := &fakeE2EHarness{results: passingAcceptance()}
	hooks := &e2eHooks{Generate: func(context.Context, runOptions, map[string]string, string, string) (agentOutput, error) {
		t.Fatal("reference mode launched an agent")
		return agentOutput{}, nil
	}, Open: func(_ context.Context, o live.Options) (e2eAcceptanceRunner, error) {
		if !o.Reference || o.Tenant != "fixture" {
			t.Fatal(o)
		}
		return h, nil
	}}
	r := executeE2E(context.Background(), runOptions{Directory: t.TempDir(), Tenant: "fixture", E2EMode: "reference", E2ETimeout: time.Minute, E2EHooks: hooks}, 1)
	if r.Status != "PASS" || r.CostUSD != 0 || !r.CostKnown || strings.Join(h.phases, ",") != "deploy,run,features,measure,cleanup" {
		t.Fatalf("%+v phases=%v", r, h.phases)
	}
}

func TestE2EAlwaysCleansUpAfterDeploymentAndCleanupFailureIsAnError(t *testing.T) {
	for _, phase := range []string{"deploy", "run", "features", "measure", "cleanup"} {
		t.Run(phase, func(t *testing.T) {
			h := &fakeE2EHarness{fail: phase, results: passingAcceptance()}
			r := executeE2E(context.Background(), runOptions{Directory: t.TempDir(), Tenant: "fixture", E2EMode: "reference", E2ETimeout: time.Minute, E2EHooks: &e2eHooks{Open: func(context.Context, live.Options) (e2eAcceptanceRunner, error) { return h, nil }}}, 1)
			if h.phases[len(h.phases)-1] != "cleanup" || r.Status == "PASS" {
				t.Fatalf("%+v phases=%v", r, h.phases)
			}
			if phase == "cleanup" && r.Status != "EVALUATOR_ERROR" {
				t.Fatal("cleanup failure was hidden", r.Status)
			}
		})
	}
}

func TestE2ERepairsUseFreshCandidatesAndPreserveFirstFailure(t *testing.T) {
	opens, generated := 0, 0
	directories := []string{}
	hooks := &e2eHooks{
		PublicFiles: func(context.Context, runOptions) (map[string]string, error) {
			return map[string]string{"requirements.json": "{}"}, nil
		},
		Generate: func(_ context.Context, _ runOptions, _ map[string]string, dir, feedback string) (agentOutput, error) {
			generated++
			directories = append(directories, dir)
			if generated == 2 && !strings.Contains(feedback, "first candidate failed") {
				t.Fatal("missing repair feedback")
			}
			return agentOutput{Model: "fake-model", Cost: .1, CostKnown: true}, nil
		},
		Open: func(_ context.Context, o live.Options) (e2eAcceptanceRunner, error) {
			opens++
			a := passingAcceptance()
			if opens == 1 {
				a.Checks["scenario"] = live.Check{Status: "FAIL", Detail: "deliberate assertion failure"}
			}
			if o.Reference {
				t.Fatal("generated candidate marked reference")
			}
			return &fakeE2EHarness{results: a}, nil
		},
	}
	r := executeE2E(context.Background(), runOptions{Directory: t.TempDir(), E2EMode: "agent", E2EBudget: 2, E2ETimeout: time.Minute, E2EHooks: hooks, Repair: 1}, 1)
	if r.Status != "FAIL" || len(r.Repairs) != 1 || r.Repairs[0].Status != "PASS" || opens != 2 || generated != 2 || directories[0] == directories[1] {
		t.Fatalf("%+v opens=%d generated=%d dirs=%v", r, opens, generated, directories)
	}
}

func TestFullAlwaysSchedulesE2EAndBudgetStopsAreVisible(t *testing.T) {
	repo, _ := filepath.Abs("../../..")
	for _, budget := range []float64{0, .25} {
		t.Run(fmt.Sprintf("budget-%g", budget), func(t *testing.T) {
			calls := 0
			b := testBank(t)
			b.Cases = selectCases(b, "smoketest", "", "local")[:1]
			h := hostConfig{Kind: "claude", Program: "/usr/bin/true", Isolation: "native"}
			dir := filepath.Join(t.TempDir(), "campaign")
			r, err := runBank(context.Background(), b, runOptions{Repository: repo, Directory: dir, Suite: "full", Runs: 1, AgentHost: h, GraderHost: h, Ablation: "with-skills", PerCase: 1, PerGrade: .25, E2EBudget: 2, Budget: budget, TenantLookup: func(context.Context, string) (string, error) { return "fixture", nil }, E2ERun: func(_ context.Context, o runOptions, g int) e2eRunResult {
				calls++
				if o.Tenant != "fixture" {
					t.Fatal(o.Tenant)
				}
				return e2eRunResult{Generation: g, Mode: "agent", Status: "PASS", Acceptance: passingAcceptance()}
			}})
			if err != nil || len(r.E2E) != 1 || r.E2EDigest == "" {
				t.Fatal(r, err)
			}
			if budget == 0 && (calls != 1 || r.E2E[0].Status != "PASS") {
				t.Fatal("full omitted E2E", r.E2E, calls)
			}
			if budget > 0 && (calls != 0 || r.E2E[0].Status != "UNRUN") {
				t.Fatal("budget silently removed E2E", r.E2E, calls)
			}
			if _, err := os.Stat(filepath.Join(dir, "report.json")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFullCannotBeFilteredOrReplacedByReferenceAcceptance(t *testing.T) {
	for _, args := range [][]string{{"full", "", "local", "", "agent"}, {"full", "", "", "proposal-scope", "agent"}, {"full", "automation-pilot-cli", "", "", "agent"}, {"full", "", "", "", "reference"}, {"e2e", "", "fixture", "", "agent"}} {
		if validateCampaignSelection(args[0], args[1], args[2], args[3], args[4]) == nil {
			t.Fatal("incomplete campaign accepted", args)
		}
	}
	if !includesE2E("full") || !includesE2E("e2e") || includesE2E("smoketest") || includesE2E("minimal") {
		t.Fatal("wrong preset integration")
	}
}

func TestE2EIncompleteAndFailureVerdictsAndAccounting(t *testing.T) {
	a := passingAcceptance()
	delete(a.Checks, "scenario")
	if e2eStatus(a) != "UNVERIFIED" {
		t.Fatal("missing verification passed")
	}
	a = passingAcceptance()
	a.Checks["scenario"] = live.Check{Status: "BLOCKED"}
	if e2eStatus(a) != "UNVERIFIED" {
		t.Fatal("blocked execution passed")
	}
	a.Checks["scenario"] = live.Check{Status: "FAIL"}
	if e2eStatus(a) != "FAIL" {
		t.Fatal("failed assertion passed")
	}
	r := runReport{E2E: []e2eRunResult{{Generation: 1, Mode: "agent", Status: "FAIL", CostUSD: .2, CostKnown: true, Usage: map[string]any{"input_tokens": float64(100)}, Acceptance: a, Repairs: []e2eRunResult{{Status: "PASS", CostUSD: .3, CostKnown: true, Usage: map[string]any{"input_tokens": float64(50)}}}}}}
	summary := summarizeReport(r)
	if summary.Selected != 0 || summary.Cost != .5 || summary.AgentUsage["input_tokens"] != float64(150) {
		t.Fatal("E2E costs or task totals incorrect", summary)
	}
	markdown := reportMarkdown(r)
	if !strings.Contains(markdown, "E2E workflow results are separate") || !strings.Contains(markdown, "Separate generation 1 repair: PASS") {
		t.Fatal(markdown)
	}
}

func TestE2ECandidateSecretsAreRedactedAndRejectedBeforeDeployment(t *testing.T) {
	secret := "review-credential-with-quote\"and-slash\\"
	t.Setenv("REVIEW_API_KEY", secret)
	root := t.TempDir()
	source := filepath.Join(root, "workspace/definitions")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	body := mustJSON(map[string]string{"name": "Example", "password": secret})
	if err := os.WriteFile(filepath.Join(source, "Example.command.json"), body, 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "sealed")
	output := agentOutput{Evidence: evidence{Answer: "observed " + secret}}
	err := saveE2EGeneration(root, target, map[string]string{"requirements.json": `{"resources":[{"file":"Example.command.json"}]}`}, output)
	if err == nil || !strings.Contains(err.Error(), "secret disclosed") {
		t.Fatal("secret disclosure accepted")
	}
	for _, name := range []string{"agent-evidence.json", "definitions/Example.command.json"} {
		data, err := os.ReadFile(filepath.Join(target, name))
		if err != nil {
			t.Fatal(err)
		}
		if !json.Valid(data) || redact.Contains(string(data), []string{secret}) {
			t.Fatal("private data persisted")
		}
	}
}

func TestE2ECandidateCannotUseSymlinkedDefinitionDirectory(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "workspace"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "workspace/definitions")); err != nil {
		t.Skip(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "Example.command.json"), []byte(`{"name":"outside"}`), 0600); err != nil {
		t.Fatal(err)
	}
	err := saveE2EGeneration(root, t.TempDir(), map[string]string{"requirements.json": `{"resources":[{"file":"Example.command.json"}]}`}, agentOutput{})
	if err == nil || !strings.Contains(err.Error(), "regular workspace directory") {
		t.Fatal("external artifact accepted")
	}
}

func TestLaterE2EGenerationsStopAfterUnresolvedCleanup(t *testing.T) {
	repo, _ := filepath.Abs("../../..")
	calls := 0
	report, err := runBank(context.Background(), bank{}, runOptions{
		Repository: repo, Directory: filepath.Join(t.TempDir(), "campaign"), Suite: "e2e", E2EMode: "reference", Runs: 2, Ablation: "with-skills", PerCase: 1, PerGrade: .25,
		TenantLookup: func(context.Context, string) (string, error) { return "fixture", nil },
		E2ERun: func(_ context.Context, _ runOptions, g int) e2eRunResult {
			calls++
			return e2eRunResult{Generation: g, Mode: "reference", Status: "EVALUATOR_ERROR", Acceptance: live.Acceptance{Checks: map[string]live.Check{"cleanup-setup": {Status: "FAIL"}}}}
		},
	})
	if err != nil || calls != 1 || len(report.E2E) != 2 || report.E2E[1].Status != "UNRUN" || !strings.Contains(report.E2E[1].Reason, "cleanup incomplete") {
		t.Fatal(report.E2E, calls, err)
	}
}
