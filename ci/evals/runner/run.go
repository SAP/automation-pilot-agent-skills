package main

import (
	"context"
	"encoding/json"
	"fmt"
	live "github.com/SAP/automation-pilot-agent-skills/ci/evals/e2e"
	"github.com/SAP/automation-pilot-agent-skills/ci/evals/internal/redact"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type runOptions struct {
	E2EAssets                                                           string
	E2EHooks                                                            *e2eHooks
	E2EMode                                                             string
	E2EBudget                                                           float64
	E2ETimeout                                                          time.Duration
	E2ERun                                                              func(context.Context, runOptions, int) e2eRunResult
	TenantLookup                                                        func(context.Context, string) (string, error)
	RuntimeSnapshot                                                     string
	Repair, Round                                                       int
	Repository, Directory, Suite, Skills, Tiers, Case, Ablation, Tenant string
	AgentHost, GraderHost                                               hostConfig
	Runs, Parallel                                                      int
	Budget, PerCase, PerGrade                                           float64
}

func captureSources(repository string) (map[string]string, string, error) {
	files := map[string]string{}
	for _, dir := range []string{"skills", "ci/evals/runner", "ci/evals/e2e", "ci/evals/cases", "ci/evals/internal", "ci/plugin"} {
		err := filepath.WalkDir(filepath.Join(repository, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "runs" || d.Name() == "results" || d.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Name() == ".DS_Store" || d.Name() == "catalog-evals" {
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink in evaluation provenance")
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(repository, path)
			files[filepath.ToSlash(rel)] = bytesDigest(b)
			return nil
		})
		if err != nil {
			return nil, "", err
		}
	}
	return files, digest(files), nil
}
func executeCase(ctx context.Context, c evalCase, generation int, options runOptions, verifier *engineVerifier) (caseResult, error) {
	start := time.Now()
	result := caseResult{ID: c.ID, CaseDigest: digest(c), Generation: generation, Primary: c.Primary, Family: c.Family, Tier: c.Tier, Split: c.Split, Review: c.Review, Status: "EVALUATOR_ERROR"}
	result.RepairRound = options.Round
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		result.Status = "UNVERIFIED"
		result.Reason = "evaluated sessions require macOS or Linux (Windows can use WSL)"
		return result, nil
	}
	if options.AgentHost.Isolation == "bubblewrap" && runtime.GOOS != "linux" {
		result.Status = "UNVERIFIED"
		result.Reason = "evaluated sessions require the Linux bubblewrap runner"
		return result, nil
	}
	root, err := os.MkdirTemp("", "autopi-case-*")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(root)
	deadline, cancel := context.WithTimeout(ctx, time.Duration(c.MaxSeconds)*time.Second)
	defer cancel()
	if options.RuntimeSnapshot != "" {
		err = copyTree(filepath.Join(options.RuntimeSnapshot, "plugin"), filepath.Join(root, "plugin"))
		for _, name := range []string{"workspace", "config", "tmp", "home"} {
			if err == nil {
				err = os.MkdirAll(filepath.Join(root, name), 0700)
			}
		}
	} else {
		err = stageRuntime(deadline, options.Repository, root, options.Ablation)
	}
	if err != nil {
		return result, err
	}
	if err = prepareHostRuntime(root, options.AgentHost, false); err != nil {
		return result, err
	}
	workspace := filepath.Join(root, "workspace")
	for name, contents := range c.Files {
		target := filepath.Join(workspace, name)
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return result, err
		}
		if err = os.WriteFile(target, []byte(contents), 0600); err != nil {
			return result, err
		}
	}
	before, err := snapshotWorkspace(workspace)
	if err != nil {
		return result, err
	}
	var fixture *fixtureTenant
	hostname, token, ca := "", "", ""
	if c.Fixture != nil {
		token = "synthetic-" + digest(root)[:24]
		fixture = startFixture(*c.Fixture, token)
		defer fixture.close()
		hostname = strings.TrimPrefix(fixture.server.URL, "https://")
		if c.Fixture.Credentials == "missing" {
			hostname = ""
			token = ""
		}
		ca = filepath.Join(root, "fixture-ca.pem")
		if err = fixture.writeCA(ca); err != nil {
			return result, err
		}
	}
	if err = pinCaseConnection(root, hostname, token, ca); err != nil {
		return result, err
	}
	// Record the controlled runtime after the evaluator installs its transport
	// boundary, before the agent can observe or modify it.
	before, err = snapshotWorkspace(workspace)
	if err != nil {
		return result, err
	}
	runtimeBefore, err := snapshotWorkspace(filepath.Join(root, "plugin"))
	if err != nil {
		return result, err
	}
	agent := launchAgent(deadline, c, root, options.AgentHost, options.Ablation, hostname, token, ca, options.PerCase)
	e := agent.Evidence
	result.AgentCostUSD = agent.Cost
	result.CostUSD = agent.Cost
	result.CostKnown = agent.CostKnown
	result.AgentUsage = agent.Usage
	result.Model = agent.Model
	result.BudgetChargedUSD = agent.Cost
	if !agent.CostKnown {
		result.BudgetChargedUSD = options.PerCase
	}
	for _, event := range traceTools(e.Trace) {
		if event.Kind == "tool_use" {
			result.ToolCalls++
		}
	}
	if fixture != nil {
		e.Requests, e.State = fixture.snapshot()
	}
	after, err := snapshotWorkspace(workspace)
	if err != nil {
		e.HardFailures = append(e.HardFailures, err.Error())
	} else {
		e.WorkspaceChanged = changedFiles(before, after)
	}
	runtimeAfter, err := snapshotWorkspace(filepath.Join(root, "plugin"))
	if err != nil || len(changedFiles(runtimeBefore, runtimeAfter)) > 0 {
		e.HardFailures = append(e.HardFailures, "installed runtime assets were changed")
	}
	for _, name := range e.WorkspaceChanged {
		if strings.HasPrefix(name, ".agents/") {
			e.HardFailures = append(e.HardFailures, "installed Codex skill assets were changed")
			break
		}
	}
	if c.Artifact != "" {
		artifactPath := filepath.Join(workspace, c.Artifact)
		info, statErr := os.Lstat(artifactPath)
		if statErr == nil && !info.Mode().IsRegular() {
			e.HardFailures = append(e.HardFailures, "submission is not a regular file")
			return result, fmt.Errorf("invalid submission path")
		}
		raw, err := os.ReadFile(artifactPath)
		if err == nil {
			if json.Unmarshal(raw, &e.Artifact) != nil {
				e.HardFailures = append(e.HardFailures, "submission artifact is invalid JSON")
			}
		}
		if strings.HasSuffix(c.Artifact, ".command.json") || strings.HasSuffix(c.Artifact, ".input.json") || strings.HasSuffix(c.Artifact, ".mcp.json") {
			cmd := exec.CommandContext(deadline, filepath.Join(root, "plugin/skills/automation-pilot-cli/scripts/autopi"), "lint", filepath.Join(workspace, c.Artifact))
			cmd.Env = agentEnvironment(root, "", "", "")
			err := cmd.Run()
			code := 0
			if err != nil {
				code = 1
			}
			e.LintExit = &code
		}
	}
	secrets := append(sensitiveEnvironmentValues(), token)
	if secretFile := c.Files["secret-source.txt"]; secretFile != "" {
		secrets = append(secrets, strings.TrimSpace(secretFile))
	}
	observable := e.Answer + string(mustJSON(e.Artifact)) + string(mustJSON(e.Trace))
	for _, name := range e.WorkspaceChanged {
		if name == "secret-source.txt" {
			continue
		}
		if value, err := os.ReadFile(filepath.Join(workspace, name)); err == nil {
			observable += string(value)
		}
	}
	for _, secret := range secrets {
		if secret != "" && redact.Contains(observable, []string{secret}) {
			e.HardFailures = append(e.HardFailures, "secret disclosed in observable agent evidence")
		}
	}
	if c.Engine != nil && agent.Err == nil {
		if verifier == nil {
			result.Reason = "real engine unavailable"
		} else {
			outputs, err := verifier.evaluate(deadline, c, e.Artifact)
			e.EngineResults = outputs
			if err != nil {
				result.Reason = err.Error()
			}
		}
	}
	// The freeze and evidence write precede hidden semantic feedback.
	result.Predicates = deterministic(c, e)
	raw := scrub(string(mustJSON(e)), secrets)
	if json.Unmarshal([]byte(raw), &e) != nil {
		return result, fmt.Errorf("cannot redact evidence")
	}
	evidenceName := fmt.Sprintf("%s/generation-%d/evidence.json", c.ID, generation)
	if options.Round > 0 {
		evidenceName = fmt.Sprintf("%s/generation-%d/repair-%d/evidence.json", c.ID, generation, options.Round)
	}
	result.EvidenceFile = evidenceName
	if err = writeJSON(filepath.Join(options.Directory, filepath.FromSlash(evidenceName)), e); err != nil {
		return result, err
	}
	if agent.Err != nil {
		result.Reason = agent.Err.Error()
		if deadline.Err() != nil {
			result.Status = "UNVERIFIED"
		} else {
			result.Status = "EVALUATOR_ERROR"
		}
		result.ElapsedSeconds = time.Since(start).Seconds()
		return result, nil
	}
	if len(c.Criteria) > 0 {
		report, meta := externalGrade(ctx, options.GraderHost, c, e, options.PerGrade)
		result.GraderCostUSD = meta.Cost
		result.GraderModel = meta.Model
		result.GraderUsage = meta.Usage
		result.CostUSD += meta.Cost
		result.CostKnown = result.CostKnown && meta.CostKnown
		if meta.CostKnown {
			result.BudgetChargedUSD += meta.Cost
		} else {
			result.BudgetChargedUSD += options.PerGrade
		}
		if meta.Err != nil {
			result.Reason = meta.Err.Error()
			result.ElapsedSeconds = time.Since(start).Seconds()
			return result, nil
		}
		predicates, err := semanticResults(c, e, report)
		if err != nil {
			result.Reason = err.Error()
			return result, nil
		}
		result.Predicates = append(result.Predicates, predicates...)
	}
	result.Status = aggregate(result.Predicates)
	result.ElapsedSeconds = time.Since(start).Seconds()
	return result, nil
}
func runBank(ctx context.Context, b bank, options runOptions) (runReport, error) {
	if options.Suite == "" {
		options.Suite = defaultSuite
	}
	if err := validateSuite(options.Suite); err != nil {
		return runReport{}, err
	}
	if options.Runs < 1 || options.Runs > 10 || !validBudget(options.Budget) || !validCost(options.PerCase) || !validCost(options.PerGrade) || options.Repair < 0 || options.Repair > 1 || options.Parallel < 0 || options.Parallel > 8 {
		return runReport{}, fmt.Errorf("invalid run/budget limits")
	}
	if options.Ablation != "with-skills" && options.Ablation != "without-skills" {
		return runReport{}, fmt.Errorf("invalid ablation")
	}
	if options.E2EMode == "" {
		options.E2EMode = "agent"
	}
	if options.E2EBudget == 0 {
		options.E2EBudget = 5
	}
	if options.E2ETimeout == 0 {
		options.E2ETimeout = 45 * time.Minute
	}
	if err := validateCampaignSelection(options.Suite, options.Skills, options.Tiers, options.Case, options.E2EMode); err != nil {
		return runReport{}, err
	}
	if !validCost(options.E2EBudget) || options.E2ETimeout <= 0 {
		return runReport{}, fmt.Errorf("invalid E2E budget/timeout")
	}
	selected := selectedCases(b, options.Suite, options.Skills, options.Tiers, options.Case)
	if len(selected) == 0 && !includesE2E(options.Suite) {
		return runReport{}, fmt.Errorf("selection matches no cases")
	}
	if _, err := os.Stat(options.Directory); err == nil {
		return runReport{}, fmt.Errorf("run directory exists; select a fresh run ID")
	}
	var verifier *engineVerifier
	for _, c := range selected {
		if c.Tier == "engine" {
			verifier = &engineVerifier{repository: options.Repository, runDir: options.Directory, tenant: options.Tenant}
			if err := verifier.resolveTenant(ctx, options.Tenant); err != nil {
				return runReport{}, fmt.Errorf("engine preflight failed before agent sessions: %w", err)
			}
			options.Tenant = verifier.tenant
			break
		}
	}
	if includesE2E(options.Suite) {
		if options.TenantLookup != nil {
			tenant, err := options.TenantLookup(ctx, options.Tenant)
			if err != nil {
				return runReport{}, err
			}
			options.Tenant = tenant
		} else {
			identity := engineVerifier{repository: options.Repository}
			if err := identity.resolveTenant(ctx, options.Tenant); err != nil {
				return runReport{}, fmt.Errorf("E2E preflight failed before agent sessions: %w", err)
			}
			options.Tenant = identity.tenant
		}
	}
	if err := os.MkdirAll(options.Directory, 0700); err != nil {
		return runReport{}, err
	}
	if err := writeJSON(filepath.Join(options.Directory, "bank.snapshot.json"), b); err != nil {
		return runReport{}, err
	}
	sources, sourceDigest, err := captureSources(options.Repository)
	if err != nil {
		return runReport{}, err
	}
	version, _ := exec.Command(options.AgentHost.Program, "--version").Output()
	graderVersion, _ := exec.Command(options.GraderHost.Program, "--version").Output()
	budgetMode := "disabled"
	if options.Budget > 0 {
		budgetMode = "reported-cost-with-reservations"
		if options.AgentHost.Kind == "codex" || options.GraderHost.Kind == "codex" {
			budgetMode = "reservation-only-for-unpriced-sessions"
		}
	}
	if options.Parallel == 0 {
		options.Parallel = 1
	}
	report := runReport{Parallel: options.Parallel, Version: 2, ID: filepath.Base(options.Directory), Started: time.Now().UTC().Format(time.RFC3339), Model: options.AgentHost.Model, GraderModel: options.GraderHost.Model, Host: options.AgentHost.Kind + " " + strings.TrimSpace(string(version)) + " / " + runtime.GOOS + "/" + runtime.GOARCH, Suite: options.Suite, Ablation: options.Ablation, Sources: sources, SourceDigest: sourceDigest, BankDigest: digest(b), BudgetUSD: options.Budget, Results: []caseResult{}, GraderHost: options.GraderHost.Kind + " " + strings.TrimSpace(string(graderVersion)), ConfigDigest: hostConfigDigest(options.AgentHost), GraderConfigDigest: hostConfigDigest(options.GraderHost), Isolation: options.AgentHost.Isolation, BudgetMode: budgetMode, SkillFilter: options.Skills, TierFilter: options.Tiers, CaseFilter: options.Case, Effort: options.AgentHost.Effort, GraderEffort: options.GraderHost.Effort}
	if options.Suite == "e2e" {
		report.GraderHost = "none (deterministic runtime assertions)"
		report.GraderModel = "none"
		report.GraderConfigDigest = ""
		if options.E2EMode == "reference" {
			report.Host = "reference acceptance / " + runtime.GOOS + "/" + runtime.GOARCH
			report.Model = "none"
			report.ConfigDigest = ""
		}
	}
	report.GraderDigest = gradingProtocolDigest()
	if verifier != nil || includesE2E(options.Suite) {
		report.Tenant = options.Tenant
	}
	if includesE2E(options.Suite) {
		task, err := live.PublicTask(options.Repository)
		if err != nil {
			return report, err
		}
		acceptanceSources := map[string]string{}
		for name, hash := range sources {
			if (strings.HasPrefix(name, "ci/evals/e2e/") || strings.HasPrefix(name, "ci/evals/internal/")) && !strings.HasSuffix(name, "_test.go") && !strings.HasSuffix(name, ".md") {
				acceptanceSources[name] = hash
			}
		}
		report.E2EDigest = digest(map[string]any{"task": task, "acceptance_sources": acceptanceSources})
		for g := 1; g <= options.Runs; g++ {
			report.E2E = append(report.E2E, e2eRunResult{Generation: g, Mode: options.E2EMode, Status: "UNRUN", Reason: "awaiting E2E execution"})
		}
	}
	var snapshot string
	{
		snapshot, err = os.MkdirTemp("", "autopi-runtime-*")
		if err != nil {
			return report, err
		}
		defer os.RemoveAll(snapshot)
		if err = stageRuntime(ctx, options.Repository, snapshot, options.Ablation); err != nil {
			return report, err
		}
		options.RuntimeSnapshot = snapshot
		if includesE2E(options.Suite) {
			options.E2EAssets = filepath.Join(options.Directory, "e2e-reference")
			if err = copyTree(filepath.Join(options.Repository, "ci/evals/e2e"), options.E2EAssets); err != nil {
				return report, err
			}
		}
	}
	ids := map[string]bool{}
	for _, c := range selected {
		ids[c.ID] = true
	}
	for _, c := range b.Cases {
		if !ids[c.ID] {
			report.Results = append(report.Results, caseResult{ID: c.ID, CaseDigest: digest(c), Primary: c.Primary, Family: c.Family, Tier: c.Tier, Split: c.Split, Review: c.Review, Status: "UNRUN", Reason: "excluded by selection"})
		}
	}
	base := len(report.Results)
	count := len(selected) * options.Runs
	initial := func(index int) caseResult {
		c := selected[index%len(selected)]
		return caseResult{ID: c.ID, CaseDigest: digest(c), Generation: index/len(selected) + 1, Primary: c.Primary, Family: c.Family, Tier: c.Tier, Split: c.Split, Review: c.Review, Status: "UNRUN", Reason: "awaiting execution"}
	}
	for index := 0; index < count; index++ {
		report.Results = append(report.Results, initial(index))
	}
	if err = writeJSON(filepath.Join(options.Directory, "report.json"), report); err != nil {
		return report, err
	}
	if verifier != nil {
		verifier.repository = filepath.Join(snapshot, "plugin")
		if err = verifier.initialize(ctx); err != nil {
			report.Results = append(report.Results, caseResult{ID: "engine-initialization", Status: "EVALUATOR_ERROR", Reason: err.Error()})
			if saveErr := writeJSON(filepath.Join(options.Directory, "report.json"), report); saveErr != nil {
				return report, saveErr
			}
			return report, fmt.Errorf("engine initialization failed before agent sessions: %w", err)
		}
	}
	work := func(index int) caseResult {
		c := selected[index%len(selected)]
		generation := index/len(selected) + 1
		var err error
		result := caseResult{ID: c.ID, CaseDigest: digest(c), Generation: generation, Primary: c.Primary, Family: c.Family, Tier: c.Tier, Split: c.Split, Review: c.Review, Status: "UNRUN"}
		switch {
		case ctx.Err() != nil:
			result.Reason = "run cancelled"
		default:
			result, err = executeCase(ctx, c, generation, options, verifier)
			if err != nil {
				result.Status = "EVALUATOR_ERROR"
				result.Reason = err.Error()
			}
			if result.Status == "FAIL" && options.Repair > 0 && c.Fixture == nil && ctx.Err() == nil {
				retry := c
				retry.Files = map[string]string{}
				for name, value := range c.Files {
					retry.Files[name] = value
				}
				var first evidence
				if readJSON(filepath.Join(options.Directory, result.EvidenceFile), &first) == nil && c.Artifact != "" && first.Artifact != nil {
					retry.Files[c.Artifact] = string(mustJSON(first.Artifact))
				}
				failures := []string{}
				for _, p := range result.Predicates {
					if p.Status == "FAIL" {
						failures = append(failures, p.ID+": "+p.Reason)
					}
				}
				retry.Prompt = c.Prompt + "\nYour first submission failed. Repair it within the same local scope, using this bounded diagnostic feedback: " + strings.Join(failures, "; ") + ". Do not claim the initial submission passed."
				roundOptions := options
				roundOptions.Round = 1
				repair, repairErr := executeCase(ctx, retry, generation, roundOptions, verifier)
				if repairErr != nil {
					repair.Status = "EVALUATOR_ERROR"
					repair.Reason = repairErr.Error()
				}
				result.Repairs = append(result.Repairs, repair)
			}
		}

		return result
	}
	reservation := (options.PerCase + options.PerGrade) * float64(1+options.Repair)
	err = parallelJobs(ctx, count, options.Parallel, options.Budget, reservation, work,
		func(index int, reason string) caseResult { r := initial(index); r.Reason = reason; return r },
		func(r caseResult) float64 {
			cost := r.BudgetChargedUSD
			for _, repair := range r.Repairs {
				cost += repair.BudgetChargedUSD
			}
			return cost
		},
		func(index int, r caseResult) error {
			report.Results[base+index] = r
			fmt.Printf("%s generation %d: %s\n", r.ID, r.Generation, r.Status)
			if r.Status == "EVALUATOR_ERROR" && r.Reason != "" {
				fmt.Printf("  %s\n", r.Reason)
			}
			return writeJSON(filepath.Join(options.Directory, "report.json"), report)
		})
	if err != nil {
		return report, err
	}
	cleanupBlocked := false
	if verifier != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		cleanupErr := verifier.cleanup(cleanupCtx)
		cancel()
		if cleanupErr != nil {
			cleanupBlocked = true
			report.Results = append(report.Results, caseResult{ID: "engine-cleanup", Status: "EVALUATOR_ERROR", Reason: cleanupErr.Error()})
		}
	}
	if includesE2E(options.Suite) {
		spent := 0.0
		for _, r := range report.Results {
			spent += r.BudgetChargedUSD
			for _, repair := range r.Repairs {
				spent += repair.BudgetChargedUSD
			}
		}
		operation := executeE2E
		if options.E2ERun != nil {
			operation = options.E2ERun
		}
		reserve := options.E2EBudget * float64(1+options.Repair)
		if options.E2EMode == "reference" {
			reserve = 0
		}
		for g := 1; g <= options.Runs; g++ {
			if cleanupBlocked {
				report.E2E[g-1].Reason = "cleanup incomplete; reconcile before further E2E deployment"
			} else if ctx.Err() != nil {
				report.E2E[g-1].Reason = "run cancelled"
			} else if options.Budget > 0 && spent+reserve > options.Budget {
				report.E2E[g-1].Reason = "run budget cannot reserve E2E generation"
			} else {
				r := operation(ctx, options, g)
				report.E2E[g-1] = r
				if _, failed := r.Acceptance.Checks["cleanup-setup"]; failed {
					cleanupBlocked = true
				}
				spent += r.BudgetChargedUSD
				for _, repair := range r.Repairs {
					spent += repair.BudgetChargedUSD
				}
			}
			fmt.Printf("e2e-complete-workflow generation %d: %s\n", g, report.E2E[g-1].Status)
			if err := writeJSON(filepath.Join(options.Directory, "report.json"), report); err != nil {
				return report, err
			}
		}
	}
	if err = writeJSON(filepath.Join(options.Directory, "report.json"), report); err != nil {
		return report, err
	}
	return report, nil
}
func validCost(v float64) bool { return v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) }

func validBudget(v float64) bool { return v == 0 || validCost(v) }
func canStartSession(charged float64, options runOptions) bool {
	return options.Budget == 0 || charged+options.PerCase+options.PerGrade <= options.Budget
}
