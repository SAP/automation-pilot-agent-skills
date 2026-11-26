package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	live "github.com/SAP/automation-pilot-agent-skills/ci/evals/e2e"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

func main() {
	if err := dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func dispatch(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Println("evals.sh check|list|run|grade|calibrate|calibrate-engine|calibrate-engine-mutants|review|refresh-fixtures|report|mine|cleanup|schema\nPresets: smoketest (12, default), minimal (20), full (160 + agent-generated E2E), e2e (complete workflow). Legacy suite names remain supported.\nRun from ci/evals, or use sh ci/evals/evals.sh. Each command accepts --help.")
		return nil
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	bankPath := fs.String("bank", "cases/bank.json", "versioned evaluator-only bank")
	suite := fs.String("suite", defaultSuite, "smoketest (12), minimal (20), full (160 + E2E), e2e; legacy pilot, smoke, regression, release, all")
	skill := fs.String("skill", "", "comma-separated primary skill filter")
	tier := fs.String("tier", "", "comma-separated tier filter")
	sourceRoot := fs.String("source", "", "autopi-content checkout for mining/source verification")
	repository := fs.String("repository", "../..", "skills repository")
	out := fs.String("out", "", "output path; report writes a fresh Markdown summary without model calls")
	runID := fs.String("run", "", "fresh run ID for run; existing report directory for report/cleanup")
	caseID := fs.String("case", "", "single case ID for run or grade")
	evidencePath := fs.String("evidence", "", "private captured evidence JSON for grade")
	model := fs.String("model", "", "agent model override; empty uses the CLI default; choose a smaller supported model for cheaper runs")
	graderModel := fs.String("grader-model", "", "semantic grader model override; empty uses the grader CLI default")
	agent := fs.String("agent", "auto", "auto (prefer installed Claude), claude, or codex")
	agentBin := fs.String("agent-bin", "", "optional executable path for the selected agent")
	grader := fs.String("grader", "", "claude or codex; empty uses the agent host")
	graderBin := fs.String("grader-bin", "", "optional grader executable path")
	effort := fs.String("effort", "", "agent reasoning effort override; empty uses the CLI default")
	graderEffort := fs.String("grader-effort", "", "grader reasoning effort override")
	isolation := fs.String("isolation", "native", "native CLI sandbox/permissions, or Linux bubblewrap filesystem isolation")
	ablation := fs.String("ablation", "with-skills", "with-skills or without-skills (same CLI in both)")
	runs := fs.Int("runs", 1, "independent fresh generations; release defaults to three")
	parallel := fs.Int("parallel", 1, "independent concurrent case/grader sessions (1-8); engine mutations remain sequential")
	repair := fs.Int("repair", 0, "optional bounded repair round (0 or 1); first results remain unchanged")
	budget := fs.Float64("budget-usd", 0, "run reservation limit (0 disables); Codex does not report USD cost")
	perCase := fs.Float64("case-budget-usd", 1, "reserved ceiling per agent session")
	perGrade := fs.Float64("grader-budget-usd", 0.25, "reserved ceiling per semantic-grade session")
	tenant := fs.String("tenant", "", "optional identity assertion; engine tenant is resolved from AUTOMATION_PILOT_HOSTNAME and AUTOMATION_PILOT_TOKEN")
	compare := fs.String("compare", "", "baseline report.json path")
	e2eMode := fs.String("e2e-mode", "agent", "agent-generated E2E, or reference (standalone e2e only; no model sessions)")
	e2eBudget := fs.Float64("e2e-budget-usd", 5, "reserved ceiling per E2E generation/repair session")
	e2eTimeout := fs.Duration("e2e-timeout", 45*time.Minute, "total E2E generation and acceptance deadline; cleanup has a separate bound")
	resume := fs.String("resume", "", "calibrate only: reuse successful matching challenges from an existing calibration report")
	if err := fs.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	if err := validateSuite(*suite); err != nil {
		return err
	}
	if args[0] == "run" || args[0] == "list" {
		if err := validateCampaignSelection(*suite, *skill, *tier, *caseID, *e2eMode); err != nil {
			return err
		}
	}
	if *resume != "" && args[0] != "calibrate" {
		return fmt.Errorf("--resume is supported only by calibrate; tenant operations are never resumed blindly")
	}
	repo, err := filepath.Abs(*repository)
	if err != nil {
		return err
	}
	if args[0] == "mine" {
		if *sourceRoot == "" {
			return fmt.Errorf("mine requires --source")
		}
		if *out == "" {
			*out = "mined-inventory.json"
		}
		return mineInventory(*sourceRoot, *out)
	}
	if args[0] == "schema" {
		if *out == "" {
			*out = "cases/bank.schema.json"
		}
		return writeJSON(*out, bankSchema())
	}
	if args[0] == "report" {
		if *runID == "" {
			return fmt.Errorf("report requires --run directory")
		}
		var report runReport
		if err = readJSON(filepath.Join(*runID, "report.json"), &report); err != nil {
			return err
		}
		if err = printReport(report, *compare); err != nil {
			return err
		}
		if *out != "" {
			return writeReportSummary(*out, report)
		}
		return nil
	}
	if args[0] == "cleanup" {
		if *runID == "" {
			return fmt.Errorf("cleanup requires --run directory")
		}
		cleaned := false
		var failures []string
		var journal engineJournal
		err = readJSON(filepath.Join(*runID, "engine-journal.json"), &journal)
		if err == nil {
			cleaned = true
			if *tenant != "" && journal.Tenant != *tenant {
				return fmt.Errorf("journal tenant mismatch")
			}
			v := engineVerifier{repository: repo, runDir: *runID, tenant: *tenant, journal: journal}
			if err := v.cleanup(context.Background()); err != nil {
				failures = append(failures, err.Error())
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		if _, err := os.Stat(filepath.Join(*runID, "e2e")); err == nil {
			cleaned = true
			if err := cleanupE2E(context.Background(), repo, *runID, *tenant); err != nil {
				failures = append(failures, err.Error())
			}
		}
		if _, err := os.Stat(filepath.Join(*runID, "state.json")); err == nil {
			cleaned = true
			v := engineVerifier{repository: repo}
			if err := v.resolveTenant(context.Background(), *tenant); err != nil {
				return err
			}
			r, err := live.Resume(context.Background(), repo, *runID, v.tenant)
			if err != nil {
				return err
			}
			if err = r.Execute(context.Background(), "cleanup", ""); err != nil {
				failures = append(failures, err.Error())
			}
		}
		if !cleaned {
			return fmt.Errorf("no engine or E2E cleanup journals in selected run")
		}
		if len(failures) > 0 {
			return fmt.Errorf("cleanup incomplete: %s", strings.Join(failures, "; "))
		}
		return nil
	}
	b, err := loadBank(*bankPath)
	if err != nil {
		return err
	}
	switch args[0] {
	case "review":
		if *runID == "" || *out == "" {
			return fmt.Errorf("review requires --run existing-directory and --out fresh-directory")
		}
		_, h, err := resolveHosts(*agent, *agentBin, *grader, *graderBin, *model, *graderModel, *effort, *graderEffort, *isolation)
		if err != nil {
			return err
		}
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		return reviewCapturedRun(ctx, b, repo, *runID, *out, *tenant, h, *perGrade, *parallel)
	case "refresh-fixtures":
		return refreshFixtureReferences(b, *out)
	case "check":
		if err := checkReferenceArtifacts(context.Background(), b, repo); err != nil {
			return err
		}
		if err := checkBank(b, *sourceRoot, repo); err != nil {
			return err
		}
		if err := checkE2EReferences(context.Background(), repo); err != nil {
			return err
		}
		fmt.Println("E2E reference offline checks: PASS. Full also requires agent-generated E2E execution; no live verification occurred.")
		return nil
	case "calibrate-engine-mutants":
		directory := *out
		if directory == "" {
			directory = filepath.Join("runs", "engine-mutants-"+stamp())
		}
		return calibrateEngineMutants(context.Background(), b, *suite, repo, *tenant, directory)
	case "calibrate-engine":
		directory := *out
		if directory == "" {
			directory = filepath.Join("runs", "engine-calibration-"+stamp())
		}
		return calibrateEngine(context.Background(), b, *suite, repo, *tenant, directory)
	case "list":
		if includesE2E(*suite) {
			fmt.Println("e2e-complete-workflow\tall skills\treal tenant\t29 core scenarios + lifecycle/features\t" + *e2eMode + "")
		}
		for _, c := range selectedCases(b, *suite, *skill, *tier, *caseID) {
			fmt.Printf("%s\t%s\t%s\t%s\t%s\n", c.ID, c.Primary, c.Tier, c.Split, c.Review)
		}
		return nil
	case "run":
		if !includesE2E(*suite) && len(selectedCases(b, *suite, *skill, *tier, *caseID)) == 0 {
			return fmt.Errorf("selection matches no cases")
		}
		if *runID == "" {
			*runID = stamp()
		}
		if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,79}$`).MatchString(*runID) {
			return fmt.Errorf("invalid run ID")
		}
		if !safeRelative(*runID) || strings.Contains(*runID, "/") {
			return fmt.Errorf("run ID must be a single safe component")
		}
		explicitRuns := false
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "runs" {
				explicitRuns = true
			}
		})
		*runs = runGenerations(*suite, *runs, explicitRuns)
		var agentHost, graderHost hostConfig
		if *suite == "e2e" && *e2eMode == "agent" {
			agentHost, err = resolveHost(*agent, *agentBin, *model, *effort, *isolation)
			if err != nil {
				return err
			}
			graderHost = agentHost
		} else if *suite != "e2e" || *e2eMode != "reference" {
			agentHost, graderHost, err = resolveHosts(*agent, *agentBin, *grader, *graderBin, *model, *graderModel, *effort, *graderEffort, *isolation)
			if err != nil {
				return err
			}
		}
		if err != nil {
			return err
		}
		directory := filepath.Join("runs", *runID)
		if *out != "" {
			directory = *out
		}
		directory, err = filepath.Abs(directory)
		if err != nil {
			return err
		}
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		report, err := runBank(ctx, b, runOptions{Repository: repo, Directory: directory, Suite: *suite, Skills: *skill, Tiers: *tier, AgentHost: agentHost, GraderHost: graderHost, Case: *caseID, Ablation: *ablation, Tenant: *tenant, Runs: *runs, Parallel: *parallel, Budget: *budget, PerCase: *perCase, PerGrade: *perGrade, Repair: *repair, E2EMode: *e2eMode, E2EBudget: *e2eBudget, E2ETimeout: *e2eTimeout})
		if err != nil {
			return err
		}
		if err = printReport(report, *compare); err != nil {
			return err
		}
		for _, r := range report.E2E {
			if r.Status != "PASS" {
				return fmt.Errorf("E2E %s; inspect %s/report.json", r.Status, directory)
			}
		}
		for _, r := range report.Results {
			if r.Status == "FAIL" || r.Status == "EVALUATOR_ERROR" {
				return fmt.Errorf("evaluation failures; inspect %s/report.json", directory)
			}
			if r.Generation > 0 && r.Status != "PASS" {
				return fmt.Errorf("evaluation incomplete; inspect %s/report.json", directory)
			}
		}
		return nil
	case "grade":
		if *caseID == "" || *evidencePath == "" {
			return fmt.Errorf("grade requires --case and --evidence")
		}
		var e evidence
		if err = readJSON(*evidencePath, &e); err != nil {
			return err
		}
		for _, c := range b.Cases {
			if c.ID == *caseID {
				predicates := deterministic(c, e)
				if len(c.Criteria) > 0 {
					_, h, err := resolveHosts(*agent, *agentBin, *grader, *graderBin, *model, *graderModel, *effort, *graderEffort, *isolation)
					if err != nil {
						return err
					}
					judgments, meta := externalGrade(context.Background(), h, c, e, *perGrade)
					if meta.Err != nil {
						return meta.Err
					}
					scored, err := semanticResults(c, e, judgments)
					if err != nil {
						return err
					}
					predicates = append(predicates, scored...)
				}
				return json.NewEncoder(os.Stdout).Encode(map[string]any{"status": aggregate(predicates), "predicates": predicates})
			}
		}
		return fmt.Errorf("unknown case")
	case "calibrate":
		if *caseID != "" {
			filtered := []evalCase{}
			for _, c := range selectCases(b, *suite, *skill, *tier) {
				if c.ID == *caseID {
					filtered = append(filtered, c)
				}
			}
			if len(filtered) == 0 {
				return fmt.Errorf("calibration case selection is empty")
			}
			b.Cases = filtered
		}
		_, h, err := resolveHosts(*agent, *agentBin, *grader, *graderBin, *model, *graderModel, *effort, *graderEffort, *isolation)
		if err != nil {
			return err
		}
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		return calibrateBankResumed(ctx, b, *suite, h, *budget, *perGrade, *out, *parallel, *resume)
	default:
		return fmt.Errorf("unknown command %s", args[0])
	}
}
func checkBank(b bank, sourceRoot, repository string) error {
	counts := map[string]int{}
	tiers := map[string]int{}
	splits := map[string]int{}
	suites := map[string]int{}
	prompts := map[string]bool{}
	semantic := 0
	deterministicChallenges := 0
	for _, c := range b.Cases {
		counts[c.Primary]++
		tiers[c.Tier]++
		splits[c.Split]++
		for _, s := range c.Suites {
			suites[s]++
		}
		p := digest(strings.Fields(c.Prompt))
		if prompts[p] {
			return fmt.Errorf("duplicate prompt")
		}
		prompts[p] = true
		for _, s := range c.Sources {
			if sourceRoot == "" || s.Repository != "autopi-content" {
				continue
			}
			root := repository
			if s.Repository == "autopi-content" {
				root = sourceRoot
			}
			raw, err := os.ReadFile(filepath.Join(root, s.Path))
			if err != nil {
				return err
			}
			if bytesDigest(raw) != s.SHA256 {
				return fmt.Errorf("%s: source snapshot changed: %s (bank remains pinned; review before refreshing)", c.ID, s.Path)
			}
		}
		reference := deterministic(c, c.Reference)
		if aggregate(reference) == "FAIL" {
			return fmt.Errorf("%s: reference evidence fails deterministic grader", c.ID)
		}
		if len(c.Criteria) > 0 {
			semantic++
		}
		for _, m := range c.Mutants {
			scores := deterministic(c, m.Evidence)
			hit := false
			for _, p := range scores {
				if p.Status == "FAIL" && contains(m.Violates, p.ID) {
					hit = true
				}
			}
			if hit {
				deterministicChallenges++
			} else {
				needsSemantic := false
				for _, criterion := range c.Criteria {
					if contains(m.Violates, criterion.ID) {
						needsSemantic = true
					}
				}
				if !needsSemantic {
					return fmt.Errorf("%s: mutant %s does not fail its declared deterministic assertion", c.ID, m.Name)
				}
			}
		}
	}
	for _, suite := range suiteNames {
		suites[suite] = len(selectCases(b, suite, "", ""))
	}
	result := map[string]any{"bank_digest": digest(b), "cases": len(b.Cases), "primary_skills": counts, "tiers": tiers, "splits": splits, "suites": suites, "deterministic_mutant_challenges_passed": deterministicChallenges, "cases_requiring_semantic_calibration": semantic, "note": "Structural and deterministic evidence-fixture checks do not constitute fresh-agent calibration, domain sign-off, or real engine execution."}
	return json.NewEncoder(os.Stdout).Encode(result)
}
func calibrateBank(ctx context.Context, b bank, suite string, h hostConfig, budget, perGrade float64, out string) error {
	return calibrateBankParallel(ctx, b, suite, h, budget, perGrade, out, 1)
}
func calibrateBankParallel(ctx context.Context, b bank, suite string, h hostConfig, budget, perGrade float64, out string, parallel int) error {
	return calibrateBankResumed(ctx, b, suite, h, budget, perGrade, out, parallel, "")
}
func calibrateBankResumed(ctx context.Context, b bank, suite string, h hostConfig, budget, perGrade float64, out string, parallel int, resume string) error {
	if err := validateSuite(suite); err != nil {
		return err
	}
	if len(selectCases(b, suite, "", "")) == 0 {
		return fmt.Errorf("calibration selection matches no cases")
	}
	if !validBudget(budget) || !validCost(perGrade) || parallel < 1 || parallel > 8 {
		return fmt.Errorf("invalid calibration limits")
	}
	if out == "" {
		out = filepath.Join("runs", "calibration-"+stamp(), "calibration.json")
	}
	if _, err := os.Stat(out); err == nil {
		return fmt.Errorf("calibration output exists; choose a fresh --out")
	}
	reusable := map[string]map[string]any{}
	for _, path := range strings.Split(resume, ",") {
		prior, err := calibrationReuse(b, h, path)
		if err != nil {
			return err
		}
		for key, row := range prior {
			reusable[key] = row
		}
	}
	var err error
	if err = writeJSON(strings.TrimSuffix(out, ".json")+".bank.json", b); err != nil {
		return err
	}
	type job struct {
		c         evalCase
		candidate mutant
	}
	jobs := []job{}
	for _, c := range selectCases(b, suite, "", "") {
		if len(c.Criteria) == 0 {
			continue
		}
		candidates := append([]mutant{{Name: "correct-reference", Evidence: c.Reference}}, c.Mutants...)
		for _, candidate := range candidates {
			jobs = append(jobs, job{c, candidate})
		}
	}
	results := make([]map[string]any, len(jobs))
	for index, j := range jobs {
		results[index] = map[string]any{"case": j.c.ID, "candidate": j.candidate.Name, "status": "UNRUN", "reason": "awaiting calibration"}
	}
	cost := 0.0
	save := func() error {
		return writeJSON(out, map[string]any{"model": h.Model, "host": h.Kind, "config_digest": hostConfigDigest(h), "grader_digest": gradingProtocolDigest(), "isolation": h.Isolation, "parallel": parallel, "bank_digest": digest(b), "budget_charged_usd": cost, "results": results})
	}
	if err := save(); err != nil {
		return err
	}
	type outcome struct {
		row  map[string]any
		cost float64
	}
	work := func(index int) outcome {
		j := jobs[index]
		c, candidate := j.c, j.candidate
		if row, ok := reusable[c.ID+"/"+candidate.Name]; ok {
			return outcome{row, 0}
		}
		judgments, meta := externalGrade(ctx, h, c, candidate.Evidence, perGrade)
		charged := meta.Cost
		if !meta.CostKnown {
			charged = perGrade
		}
		status, reason := "PASS", ""
		var scores []predicateResult
		err := meta.Err
		if err == nil {
			scores, err = semanticResults(c, candidate.Evidence, judgments)
			if err == nil {
				if candidate.Name == "correct-reference" {
					if aggregate(scores) != "PASS" {
						status = "FAIL"
						reason = "correct candidate rejected"
					}
				} else {
					hit := false
					for _, p := range scores {
						if p.Status == "FAIL" && contains(candidate.Violates, p.ID) {
							hit = true
						}
					}
					if !hit {
						for _, p := range deterministic(c, candidate.Evidence) {
							if p.Status == "FAIL" && contains(candidate.Violates, p.ID) {
								hit = true
							}
						}
					}
					if !hit {
						status = "FAIL"
						reason = "mutant accepted"
					}
				}
			}
		}
		if err != nil {
			status = "EVALUATOR_ERROR"
			reason = err.Error()
		}
		return outcome{map[string]any{"case": c.ID, "candidate": candidate.Name, "case_digest": digest(c), "candidate_digest": digest(candidate), "status": status, "reason": reason, "predicates": scores, "judgments": judgments, "resolved_model": meta.Model, "usage": meta.Usage, "cost_known": meta.CostKnown}, charged}
	}
	err = parallelJobs(ctx, len(jobs), parallel, budget, perGrade, work,
		func(index int, reason string) outcome {
			j := jobs[index]
			return outcome{row: map[string]any{"case": j.c.ID, "candidate": j.candidate.Name, "status": "UNRUN", "reason": reason}}
		},
		func(value outcome) float64 { return value.cost },
		func(index int, value outcome) error {
			results[index] = value.row
			cost += value.cost
			fmt.Printf("%s %s calibration: %s\n", value.row["case"], value.row["candidate"], value.row["status"])
			return save()
		})
	if err != nil {
		return err
	}
	fmt.Println(out)
	for _, r := range results {
		if r["status"] != "PASS" {
			return fmt.Errorf("calibration found grader failures")
		}
	}
	return nil
}

func calibrationReuse(b bank, h hostConfig, path string) (map[string]map[string]any, error) {
	out := map[string]map[string]any{}
	if path == "" {
		return out, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var previous map[string]any
	if err = json.Unmarshal(raw, &previous); err != nil {
		return nil, err
	}
	if previous["host"] != h.Kind || previous["model"] != h.Model || previous["isolation"] != h.Isolation || previous["config_digest"] != hostConfigDigest(h) {
		return nil, fmt.Errorf("calibration host/model/configuration differs; cannot reuse judgments")
	}
	if previous["grader_digest"] != gradingProtocolDigest() {
		return nil, fmt.Errorf("calibration grader implementation changed or was not recorded; cannot reuse judgments")
	}
	old, err := loadBank(strings.TrimSuffix(path, ".json") + ".bank.json")
	if err != nil {
		return nil, fmt.Errorf("calibration reuse requires its saved bank snapshot: %w", err)
	}
	if previous["bank_digest"] != digest(old) {
		return nil, fmt.Errorf("calibration bank snapshot digest mismatch")
	}
	cases := map[string]evalCase{}
	for _, c := range b.Cases {
		cases[c.ID] = c
	}
	prior := map[string]evalCase{}
	for _, c := range old.Cases {
		prior[c.ID] = c
	}
	rows, _ := previous["results"].([]any)
	for _, value := range rows {
		row, _ := value.(map[string]any)
		id, _ := row["case"].(string)
		name, _ := row["candidate"].(string)
		c, ok := cases[id]
		oldCase := prior[id]
		if !ok || row["status"] != "PASS" || digest(c) != digest(oldCase) {
			continue
		}
		var candidates = []mutant{{Name: "correct-reference", Evidence: c.Reference}}
		candidates = append(candidates, c.Mutants...)
		for _, candidate := range candidates {
			if candidate.Name == name {
				valid := name == "correct-reference"
				if !valid {
					scores, _ := row["predicates"].([]any)
					for _, value := range scores {
						p, _ := value.(map[string]any)
						pid, _ := p["id"].(string)
						if p["status"] == "FAIL" && contains(candidate.Violates, pid) {
							valid = true
						}
					}
					for _, p := range deterministic(c, candidate.Evidence) {
						if p.Status == "FAIL" && contains(candidate.Violates, p.ID) {
							valid = true
						}
					}
				}
				if valid {
					row["reused_from"] = path
					row["case_digest"] = digest(c)
					row["candidate_digest"] = digest(candidate)
					out[id+"/"+name] = row
				}
			}
		}
	}
	return out, nil
}
func printReport(report runReport, baselinePath string) error {
	summary := summarizeReport(report)
	fmt.Printf("Selected case-generations: %d; excluded cases: %d\n", summary.Selected, summary.Excluded)
	fmt.Printf("Case outcomes: %s\n", outcomeCounts(summary.Counts))
	if len(summary.RunErrors) > 0 {
		fmt.Printf("Run-level errors (separate from cases): %v\n", summary.RunErrors)
	}
	if summary.Counts["EVALUATOR_ERROR"]+summary.Counts["UNVERIFIED"]+summary.Counts["UNRUN"] > 0 || len(summary.RunErrors) > 0 {
		fmt.Println("Verification is incomplete; these results do not establish an overall skill-quality score.")
	}
	displayModel := report.Model
	if displayModel == "" {
		displayModel = "configured default"
	}
	fmt.Printf("Run %s: %s / %s / %s\n", report.ID, report.Host, displayModel, report.Ablation)
	fmt.Printf("Reported USD cost $%.4f (unpriced sessions excluded); rows with incomplete cost data: %d\n", summary.Cost, summary.Unpriced+summary.E2EUnpriced)
	fmt.Printf("Isolation: %s; run budget: %s\n", report.Isolation, report.BudgetMode)
	if report.OriginalReport != "" {
		fmt.Println("Review totals include reused candidate sessions and subsequent grading; do not add the original campaign total again.")
	}
	if len(summary.Repairs) > 0 {
		fmt.Printf("Separate repaired outcomes: %s\n", outcomeCounts(summary.Repairs))
	}
	for _, name := range sortedKeys(summary.Skills) {
		fmt.Printf("%s: %s\n", name, outcomeCounts(summary.Skills[name]))
	}
	printE2E(report.E2E)
	if summary.Pending > 0 {
		fmt.Println("Case review/calibration is pending: treat these scores as provisional.")
	}
	if baselinePath == "" {
		return nil
	}
	var baseline runReport
	if err := readJSON(baselinePath, &baseline); err != nil {
		return err
	}
	if baseline.E2EDigest != report.E2EDigest {
		return fmt.Errorf("baseline E2E suite differs; comparison refused")
	}
	if baseline.Model != report.Model || baseline.GraderModel != report.GraderModel || baseline.Host != report.Host || baseline.Ablation != report.Ablation || baseline.BankDigest != report.BankDigest || baseline.GraderHost != report.GraderHost || baseline.ConfigDigest != report.ConfigDigest || baseline.GraderConfigDigest != report.GraderConfigDigest || baseline.Isolation != report.Isolation || baseline.Suite != report.Suite || baseline.SkillFilter != report.SkillFilter || baseline.TierFilter != report.TierFilter || baseline.CaseFilter != report.CaseFilter || baseline.Effort != report.Effort || baseline.GraderEffort != report.GraderEffort {
		return fmt.Errorf("baseline model/host/configuration/isolation/selection/ablation/bank differs; comparison refused")
	}
	if baseline.GraderDigest != report.GraderDigest {
		return fmt.Errorf("baseline grader implementation differs or was not recorded; comparison refused")
	}
	if err := compareE2E(baseline.E2E, report.E2E); err != nil {
		return err
	}
	group := func(results []caseResult) map[string][]caseResult {
		out := map[string][]caseResult{}
		for _, r := range results {
			if r.Generation > 0 {
				out[r.ID] = append(out[r.ID], r)
			}
		}
		return out
	}
	before, after := group(baseline.Results), group(report.Results)
	ids := sortedKeys(after)
	sort.Strings(ids)
	fmt.Println("Case\tBaseline passes / generations\tCandidate passes / generations")
	for _, id := range ids {
		a, ok := before[id]
		if !ok {
			continue
		}
		b := after[id]
		if a[0].CaseDigest != b[0].CaseDigest {
			continue
		}
		if report.Model == "" && (a[0].Model == "" || b[0].Model == "") {
			return fmt.Errorf("resolved model unavailable for %s; pin --model for comparisons", id)
		}
		if report.GraderModel == "" && (a[0].GraderModel == "" || b[0].GraderModel == "") && len(b[0].GraderUsage) > 0 {
			return fmt.Errorf("resolved grader model unavailable for %s; pin --grader-model for comparisons", id)
		}
		if a[0].GraderModel != "" && b[0].GraderModel != "" && a[0].GraderModel != b[0].GraderModel {
			return fmt.Errorf("resolved grader model differs for %s; comparison refused", id)
		}
		if a[0].Model != "" && b[0].Model != "" && a[0].Model != b[0].Model {
			return fmt.Errorf("resolved model differs for %s; comparison refused", id)
		}
		pass := func(rs []caseResult) int {
			n := 0
			for _, r := range rs {
				if r.Status == "PASS" {
					n++
				}
			}
			return n
		}
		fmt.Printf("%s\t%d/%d\t%d/%d\n", id, pass(a), len(a), pass(b), len(b))
	}
	return nil
}
