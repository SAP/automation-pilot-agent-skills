package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/SAP/automation-pilot-agent-skills/ci/evals/internal/redact"
	"os"
	"path/filepath"
	"strings"
	"time"

	live "github.com/SAP/automation-pilot-agent-skills/ci/evals/e2e"
)

type e2eAcceptanceRunner interface {
	Execute(context.Context, string, string) error
	Results() live.Acceptance
}
type e2eHooks struct {
	PublicFiles func(context.Context, runOptions) (map[string]string, error)
	Generate    func(context.Context, runOptions, map[string]string, string, string) (agentOutput, error)
	Open        func(context.Context, live.Options) (e2eAcceptanceRunner, error)
}

type e2eRunResult struct {
	Generation       int             `json:"generation"`
	Mode             string          `json:"mode"`
	Status           string          `json:"status"`
	Reason           string          `json:"reason,omitempty"`
	Model            string          `json:"resolved_model,omitempty"`
	Usage            map[string]any  `json:"agent_usage,omitempty"`
	CostUSD          float64         `json:"cost_usd"`
	CostKnown        bool            `json:"cost_known"`
	BudgetChargedUSD float64         `json:"budget_charged_usd"`
	ElapsedSeconds   float64         `json:"elapsed_seconds"`
	Directory        string          `json:"directory"`
	Acceptance       live.Acceptance `json:"acceptance"`
	Repairs          []e2eRunResult  `json:"repairs,omitempty"`
}

func includesE2E(suite string) bool { return suite == "full" || suite == "e2e" }

func validateCampaignSelection(suite, skills, tiers, caseID, mode string) error {
	if includesE2E(suite) && (skills != "" || tiers != "" || caseID != "") {
		return fmt.Errorf("%s runs a complete campaign and does not accept --skill, --tier or --case; use smoketest/minimal or the bank-only regression alias for filtered tasks", suite)
	}
	if mode != "agent" && mode != "reference" {
		return fmt.Errorf("--e2e-mode must be agent or reference")
	}
	if suite == "full" && mode != "agent" {
		return fmt.Errorf("full includes agent-generated E2E; use --suite e2e --e2e-mode reference for reference acceptance")
	}
	return nil
}

func e2eStatus(a live.Acceptance) string {
	if len(a.Scenarios) == 0 || len(a.Required) == 0 {
		return "UNVERIFIED"
	}
	result := "PASS"
	for _, id := range append(append([]string{}, a.Required...), a.Scenarios...) {
		check, ok := a.Checks[id]
		if !ok || check.Status == "BLOCKED" || check.Status == "UNRUN" {
			if result != "FAIL" {
				result = "UNVERIFIED"
			}
			continue
		}
		if check.Status != "PASS" {
			result = "FAIL"
		}
	}
	for _, check := range a.Checks {
		if check.Status == "BLOCKED" && result != "FAIL" {
			result = "UNVERIFIED"
		}
		if check.Status == "FAIL" {
			return "FAIL"
		}
	}
	return result
}

func checkE2EReferences(ctx context.Context, repository string) error {
	dir, err := os.MkdirTemp("", "autopi-e2e-offline-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	r, err := live.Open(ctx, live.Options{Repository: repository, Directory: dir, Reference: true})
	if err != nil {
		return err
	}
	return r.Execute(ctx, "check", "")
}

func e2ePublicFiles(ctx context.Context, options runOptions) (map[string]string, error) {
	root := options.E2EAssets
	if root == "" {
		root = filepath.Join(options.Repository, "ci/evals/e2e")
	}
	task, err := live.PublicTaskFromRoot(root)
	if err != nil {
		return nil, err
	}
	repository := options.Repository
	if options.RuntimeSnapshot != "" {
		repository = filepath.Join(options.RuntimeSnapshot, "plugin")
	}
	verifier := engineVerifier{repository: repository}
	contracts := map[string]any{}
	for _, id := range live.NativeDependencies() {
		contract, err := verifier.call(ctx, []string{"commands", "contract", id}, nil)
		if err != nil {
			return nil, fmt.Errorf("E2E dependency %s: %w", id, err)
		}
		contracts[id] = contract
	}
	return map[string]string{"requirements.json": string(mustJSON(task)), "native-contracts.json": string(mustJSON(contracts))}, nil
}

func generateE2E(ctx context.Context, options runOptions, files map[string]string, target string, feedback string) (agentOutput, error) {
	root, err := os.MkdirTemp("", "autopi-e2e-agent-*")
	if err != nil {
		return agentOutput{}, err
	}
	defer os.RemoveAll(root)
	if err = copyTree(filepath.Join(options.RuntimeSnapshot, "plugin"), filepath.Join(root, "plugin")); err != nil {
		return agentOutput{}, err
	}
	for _, dir := range []string{"workspace", "config", "tmp", "home"} {
		if err = os.MkdirAll(filepath.Join(root, dir), 0700); err != nil {
			return agentOutput{}, err
		}
	}
	if err = prepareHostRuntime(root, options.AgentHost, false); err != nil {
		return agentOutput{}, err
	}
	for name, body := range files {
		if err = os.WriteFile(filepath.Join(root, "workspace", name), []byte(body), 0600); err != nil {
			return agentOutput{}, err
		}
	}
	prompt := e2eGenerationPrompt + feedback
	candidate := evalCase{ID: "e2e-complete-workflow", Prompt: prompt, MaxTurns: 80, MaxSeconds: int(options.E2ETimeout.Seconds())}
	runtimeBefore, err := snapshotWorkspace(filepath.Join(root, "plugin"))
	if err != nil {
		return agentOutput{}, err
	}
	workspaceBefore, err := snapshotWorkspace(filepath.Join(root, "workspace"))
	if err != nil {
		return agentOutput{}, err
	}
	output := launchAgent(ctx, candidate, root, options.AgentHost, options.Ablation, "", "", "", options.E2EBudget)
	runtimeAfter, err := snapshotWorkspace(filepath.Join(root, "plugin"))
	if err != nil || len(changedFiles(runtimeBefore, runtimeAfter)) > 0 {
		output.Evidence.HardFailures = append(output.Evidence.HardFailures, "installed runtime assets were changed")
	}
	workspaceAfter, err := snapshotWorkspace(filepath.Join(root, "workspace"))
	if err != nil {
		output.Evidence.HardFailures = append(output.Evidence.HardFailures, "cannot verify E2E workspace")
	}
	output.Evidence.WorkspaceChanged = changedFiles(workspaceBefore, workspaceAfter)
	for _, name := range output.Evidence.WorkspaceChanged {
		if strings.HasPrefix(name, ".agents/") {
			output.Evidence.HardFailures = append(output.Evidence.HardFailures, "installed Codex skill assets were changed")
			break
		}
	}
	return output, saveE2EGeneration(root, target, files, output)
}

// Seal and redact evidence before persisting any candidate file. Rejected
// submissions are retained in redacted form and never reach the tenant.
func saveE2EGeneration(root, target string, files map[string]string, output agentOutput) error {
	if err := os.MkdirAll(target, 0700); err != nil {
		return err
	}
	secrets := sensitiveEnvironmentValues()
	evidence := output.Evidence
	bodies := map[string][]byte{}
	var failure error
	if output.Err != nil {
		failure = output.Err
	}
	if len(evidence.HardFailures) > 0 {
		failure = fmt.Errorf("E2E candidate violated runtime integrity checks")
	}
	var requirements struct{ Resources []struct{ File string } }
	if err := json.Unmarshal([]byte(files["requirements.json"]), &requirements); err != nil && failure == nil {
		failure = err
	}
	workspace := filepath.Join(root, "workspace")
	definitions := filepath.Join(workspace, "definitions")
	if info, err := os.Lstat(definitions); err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		failure = fmt.Errorf("definitions must be a regular workspace directory")
	}
	if failure == nil {
		for _, role := range requirements.Resources {
			if !safeRelative(role.File) || strings.ContainsAny(role.File, "/\\") {
				failure = fmt.Errorf("invalid required artifact filename")
				break
			}
			path := filepath.Join(definitions, role.File)
			info, err := os.Lstat(path)
			if err != nil {
				failure = fmt.Errorf("missing E2E definition %s", role.File)
				break
			}
			if !info.Mode().IsRegular() || info.Size() > 2<<20 {
				failure = fmt.Errorf("invalid E2E definition %s", role.File)
				break
			}
			body, err := os.ReadFile(path)
			if err != nil {
				failure = err
				break
			}
			if !json.Valid(body) {
				failure = fmt.Errorf("invalid JSON definition %s", role.File)
				break
			}
			bodies[role.File] = body
		}
	}
	observed := string(mustJSON(evidence))
	for _, body := range bodies {
		observed += string(body)
	}
	if redact.Contains(observed, secrets) {
		evidence.HardFailures = append(evidence.HardFailures, "secret disclosed in E2E candidate evidence")
		failure = fmt.Errorf("secret disclosed in E2E candidate evidence; redacted submission rejected")
	}
	raw := scrub(string(mustJSON(evidence)), secrets)
	if err := json.Unmarshal([]byte(raw), &evidence); err != nil {
		return fmt.Errorf("cannot redact E2E evidence")
	}
	if err := writeJSON(filepath.Join(target, "agent-evidence.json"), evidence); err != nil {
		return err
	}
	for name, body := range bodies {
		if err := os.MkdirAll(filepath.Join(target, "definitions"), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(target, "definitions", name), []byte(scrub(string(body), secrets)), 0600); err != nil {
			return err
		}
	}
	return failure

}

const e2eGenerationPrompt = "Build a complete Automation Pilot automation locally from requirements.json. Write every required resource in definitions/ using the listed filenames and public names; these identify required roles, while executor implementations are your choice. Keep <RUN>, <CORE>, <OPS>, <CORE_PREFIX> and <OPS_PREFIX> placeholders for evaluator deployment. Read the installed skills and native-contracts.json, validate with the verified CLI, and leave a concise explanation. For local validation render separate temporary copies with synthetic eval catalog IDs and a lowercase run label; retain the required placeholders in submitted definitions. The evaluator performs authoritative materialized lint and binding checks. Do not contact a tenant, deploy, execute external services, or read credentials. The evaluator will deploy and execute the definitions.\nUse JSONPlaceholder for read-only post/user enrichment and httpbingo.org for real status sequences. Commands must cover enrichment, portfolio processing, ordered and parallel iteration, fallback, validation, retries, permanent errors, bounded repeat polling, listeners, initial delay and sensitive Input bindings. HTTP tests must obtain actual status codes from HttpRequest; per-parent runtime-store counters select fixture URLs and must be cleared on failure/abort. Preserve the named executor roles required by failedExecutor assertions. Bind configuration, workload bounds, scheduled arguments and a generated fake credential through the declared Inputs. Expose exactly the listed business parameters through public/control MCP tools. No real credential may reach a public endpoint. Satisfy all listed scenarios plus MCP disable/restore and once-schedule lineage checks. Keep dynamic expressions compatible with declared wire types. The evaluator creates the once schedule and verifies delivery; do not create a schedule artifact.\n"

func executeE2E(ctx context.Context, options runOptions, generation int) (result e2eRunResult) {
	start := time.Now()
	result = e2eRunResult{Generation: generation, Mode: options.E2EMode, Status: "EVALUATOR_ERROR", Directory: filepath.Join("e2e", fmt.Sprintf("generation-%d", generation))}
	defer func() { result.ElapsedSeconds = time.Since(start).Seconds() }()
	dir := filepath.Join(options.Directory, result.Directory)
	if err := os.MkdirAll(dir, 0700); err != nil {
		result.Reason = err.Error()
		return
	}
	deadline, cancel := context.WithTimeout(ctx, options.E2ETimeout)
	defer cancel()
	publicFiles := e2ePublicFiles
	generate := generateE2E
	open := func(ctx context.Context, opts live.Options) (e2eAcceptanceRunner, error) { return live.Open(ctx, opts) }
	if h := options.E2EHooks; h != nil {
		if h.PublicFiles != nil {
			publicFiles = h.PublicFiles
		}
		if h.Generate != nil {
			generate = h.Generate
		}
		if h.Open != nil {
			open = h.Open
		}
	}
	files := map[string]string{}
	if options.E2EMode == "agent" {
		var err error
		files, err = publicFiles(deadline, options)
		if err != nil {
			result.Status = "UNVERIFIED"
			result.Reason = err.Error()
			return
		}
	}
	attempt := func(candidateDir string, feedback string) e2eRunResult {
		r := e2eRunResult{Generation: generation, Mode: options.E2EMode, Status: "EVALUATOR_ERROR"}
		r.Directory, _ = filepath.Rel(options.Directory, candidateDir)
		definitions := ""
		if options.E2EMode == "agent" {
			output, err := generate(deadline, options, files, candidateDir, feedback)
			r.Model = output.Model
			r.Usage = output.Usage
			r.CostUSD = output.Cost
			r.CostKnown = output.CostKnown
			r.BudgetChargedUSD = output.Cost
			if !output.CostKnown {
				r.BudgetChargedUSD = options.E2EBudget
			}
			if err != nil {
				r.Reason = err.Error()
				if output.Err == nil {
					r.Status = "FAIL"
				}
				return r
			}
			definitions = filepath.Join(candidateDir, "definitions")
		} else {
			r.CostKnown = true
		}
		runtimeDir := filepath.Join(candidateDir, "runtime")
		harness, err := open(deadline, live.Options{Repository: options.Repository, Directory: runtimeDir, Definitions: definitions, Tenant: options.Tenant, Reference: options.E2EMode == "reference", Root: options.E2EAssets, Launcher: filepath.Join(options.RuntimeSnapshot, "plugin/skills/automation-pilot-cli/scripts/autopi")})
		if err != nil {
			r.Status = "FAIL"
			r.Reason = err.Error()
			return r
		}
		for _, phase := range []string{"deploy", "run", "features", "measure"} {
			if err = harness.Execute(deadline, phase, ""); err != nil {
				r.Reason = phase + ": " + err.Error()
				break
			}
		}
		r.Acceptance = harness.Results()
		r.Status = e2eStatus(r.Acceptance)
		if err != nil && r.Status == "PASS" {
			r.Status = "EVALUATOR_ERROR"
		}
		if deadline.Err() != nil {
			r.Status = "UNVERIFIED"
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		cleanupErr := harness.Execute(cleanupCtx, "cleanup", "")
		cleanupCancel()
		r.Acceptance = harness.Results()
		if cleanupErr != nil {
			r.Status = "EVALUATOR_ERROR"
			r.Reason += "; cleanup: " + cleanupErr.Error()
		}
		return r
	}
	first := attempt(filepath.Join(dir, "first"), "")
	first.Directory = result.Directory + "/first"
	result = first
	if result.Status == "FAIL" && options.Repair == 1 && options.E2EMode == "agent" && deadline.Err() == nil {
		feedback := "\nThe first candidate failed acceptance. Build a fresh complete candidate using this bounded feedback. Do not claim the first attempt passed: " + result.Reason + "\n" + string(mustJSON(result.Acceptance))
		result.Repairs = append(result.Repairs, attempt(filepath.Join(dir, "repair-1"), feedback))
	}
	return
}

func cleanupE2E(ctx context.Context, repository, directory, expectedTenant string) error {
	verifier := engineVerifier{repository: repository}
	if err := verifier.resolveTenant(ctx, expectedTenant); err != nil {
		return err
	}
	return filepath.WalkDir(filepath.Join(directory, "e2e"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in E2E cleanup journal")
		}
		if d.Name() != "state.json" || d.IsDir() {
			return nil
		}
		r, err := live.Resume(ctx, repository, filepath.Dir(path), verifier.tenant)
		if err != nil {
			return err
		}
		return r.Execute(ctx, "cleanup", "")
	})
}

func e2eIncomplete(results []e2eRunResult) bool {
	for _, r := range results {
		if r.Status == "EVALUATOR_ERROR" || r.Status == "UNVERIFIED" || r.Status == "UNRUN" {
			return true
		}
	}
	return false
}
func e2eCoreCounts(r e2eRunResult) map[string]int {
	counts := map[string]int{}
	for _, id := range r.Acceptance.Scenarios {
		c, ok := r.Acceptance.Checks[id]
		status := c.Status
		if !ok {
			status = "UNRUN"
		}
		counts[status]++
	}
	return counts
}
func printE2E(results []e2eRunResult) {
	for _, r := range results {
		counts := e2eCoreCounts(r)
		fmt.Printf("E2E generation %d (%s): %s; core scenarios PASS %d / %d; evidence %s\n", r.Generation, r.Mode, r.Status, counts["PASS"], len(r.Acceptance.Scenarios), r.Directory)
		if r.Reason != "" {
			fmt.Printf("  %s\n", r.Reason)
		}
		for _, repair := range r.Repairs {
			fmt.Printf("  Separate E2E repair: %s\n", repair.Status)
		}
	}
}
func e2eMarkdown(results []e2eRunResult) string {
	if len(results) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("\nE2E workflow results are separate from independent task scores. Agent mode uses generated definitions; reference mode verifies supplied definitions and does not establish agent authoring quality.\n\n| Generation | Mode | First attempt | Core scenarios passed | Evidence directory |\n|---|---|---|---:|---|\n")
	for _, r := range results {
		counts := e2eCoreCounts(r)
		fmt.Fprintf(&out, "| %d | %s | %s | %d / %d | %s |\n", r.Generation, r.Mode, r.Status, counts["PASS"], len(r.Acceptance.Scenarios), markdownCell(r.Directory))
	}
	for _, r := range results {
		if r.Reason != "" {
			fmt.Fprintf(&out, "\nGeneration %d: %s\n", r.Generation, markdownCell(r.Reason))
		}
		for _, id := range sortedKeys(r.Acceptance.Checks) {
			c := r.Acceptance.Checks[id]
			if c.Status != "PASS" {
				fmt.Fprintf(&out, "\n- %s: %s — %s\n", markdownCell(id), c.Status, markdownCell(c.Detail))
			}
		}
		for _, repair := range r.Repairs {
			fmt.Fprintf(&out, "\nSeparate generation %d repair: %s; evidence %s.\n", r.Generation, repair.Status, markdownCell(repair.Directory))
		}
	}
	return out.String()
}
func compareE2E(before, after []e2eRunResult) error {
	if len(before) != len(after) {
		return fmt.Errorf("baseline E2E generations differ")
	}
	for i, r := range after {
		if r.Mode != before[i].Mode || r.Generation != before[i].Generation {
			return fmt.Errorf("baseline E2E mode/generation differs")
		}
		if r.Mode == "agent" && (r.Model == "" || before[i].Model == "" || r.Model != before[i].Model) {
			return fmt.Errorf("baseline resolved E2E model differs or is unavailable")
		}
		fmt.Printf("E2E generation %d: baseline %s -> candidate %s\n", r.Generation, before[i].Status, r.Status)
	}
	return nil
}
