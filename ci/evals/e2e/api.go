package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Options describe one immutable candidate and its journal. The connection is
// always read by autopi from AUTOMATION_PILOT_HOSTNAME and AUTOMATION_PILOT_TOKEN.
type Options struct {
	Repository, Directory, Definitions, Tenant string
	Root, Launcher                             string
	Reference                                  bool
}
type Check struct {
	Status         string  `json:"status"`
	Detail         string  `json:"detail"`
	ExecutionID    string  `json:"execution_id,omitempty"`
	ElapsedSeconds float64 `json:"elapsed_seconds,omitempty"`
}
type Acceptance struct {
	Checks    map[string]Check `json:"checks"`
	Scenarios []string         `json:"scenarios"`
	Required  []string         `json:"required"`
}
type Runner struct {
	h           *harness
	cleanupOnly bool
}

func (h *harness) definitionPath(file string) string {
	dir := h.definitions
	if dir == "" {
		dir = filepath.Join(h.root, "definitions")
	}
	return filepath.Join(dir, file)
}

func Open(ctx context.Context, opts Options) (*Runner, error) { return open(ctx, opts, false) }
func open(ctx context.Context, opts Options, cleanupOnly bool) (*Runner, error) {
	root := opts.Root
	if root == "" {
		root = filepath.Join(opts.Repository, "ci/evals/e2e")
	}
	if !opts.Reference && opts.Definitions == "" {
		return nil, fmt.Errorf("generated E2E requires candidate definitions")
	}
	h := &harness{root: root, dir: opts.Directory, cli: filepath.Join(opts.Repository, "skills/automation-pilot-cli/scripts/autopi"), definitions: opts.Definitions, ctx: ctx, quietReport: true}
	h.repository = opts.Repository
	if opts.Launcher != "" {
		h.cli = opts.Launcher
	}
	if os.PathSeparator == 92 {
		h.cli += ".cmd"
	}
	if err := read(filepath.Join(root, "suite.json"), &h.spec); err != nil {
		return nil, err
	}
	if !opts.Reference {
		h.spec.Negative = nil
		if err := validateCandidateScope(root, opts.Definitions, h.spec); err != nil {
			return nil, err
		}
	}
	err := read(filepath.Join(h.dir, "state.json"), &h.state)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if os.IsNotExist(err) {
		tenant := opts.Tenant
		if tenant == "" {
			tenant = "TENANT"
		}
		sum := sha256.Sum256([]byte(h.dir))
		suffix := hex.EncodeToString(sum[:])[:8]
		name := "e2e-" + suffix
		h.state = state{Run: name, Tenant: tenant, Variables: map[string]string{"RUN": name, "CORE_PREFIX": "ec" + suffix, "OPS_PREFIX": "eo" + suffix, "CORE": "ec" + suffix + "-" + tenant, "OPS": "eo" + suffix + "-" + tenant}, Resources: map[string]string{}, Attempts: map[string]string{}, Results: map[string]verdict{}, Hashes: map[string]string{}}
	}
	if opts.Tenant != "" && h.state.Tenant != opts.Tenant {
		return nil, fmt.Errorf("saved E2E tenant differs from configured identity; no writes performed")
	}
	for _, r := range h.spec.Resources {
		if r.Resource != "commands" && r.Resource != "inputs" {
			continue
		}
		var d map[string]any
		if err := read(h.definitionPath(r.File), &d); err != nil {
			return nil, err
		}
		h.names = append(h.names, str(d["name"]))
	}
	entries, _ := os.ReadDir(filepath.Join(h.dir, "evidence"))
	for _, f := range entries {
		n, _ := strconv.Atoi(strings.SplitN(f.Name(), "-", 2)[0])
		if n > h.calls {
			h.calls = n
		}
	}
	if err := h.captureSources(); err != nil {
		return nil, err
	}
	if err := h.save(); err != nil {
		return nil, err
	}
	if !cleanupOnly {
		if err := write(filepath.Join(h.dir, "runner-options.json"), opts); err != nil {
			return nil, err
		}
	}
	return &Runner{h: h, cleanupOnly: cleanupOnly}, nil
}

func (r *Runner) Execute(ctx context.Context, phase, only string) error {
	if r.cleanupOnly && phase != "cleanup" {
		return fmt.Errorf("resumed journals are cleanup-only; no deployment replay")
	}
	r.h.ctx = ctx
	var err error
	switch phase {
	case "check":
		err = r.h.check()
	case "deploy":
		err = r.h.deploy()
	case "run":
		err = r.h.run()
	case "features":
		err = r.h.features(only)
	case "measure":
		err = r.h.measure()
	case "cleanup":
		err = r.h.cleanup()
	default:
		return fmt.Errorf("unknown E2E phase %s", phase)
	}
	if err != nil {
		status := "FAIL"
		if blocked(err) {
			status = "BLOCKED"
		}
		if recordErr := r.h.record(phase+"-setup", status, err.Error()); recordErr != nil {
			return fmt.Errorf("%w; record failed: %v", err, recordErr)
		}
	}
	if reportErr := r.h.report(); reportErr != nil {
		return fmt.Errorf("phase error %v; report: %w", err, reportErr)
	}
	return err
}

func (r *Runner) Results() Acceptance {
	out := Acceptance{Checks: map[string]Check{}, Required: []string{"authored-lint-first", "bindings-public", "bindings-control", "schema-public", "schema-control", "mcp-tool-lifecycle", "mcp-server-lifecycle", "schedule-once"}}
	for _, resource := range r.h.spec.Resources {
		out.Required = append(out.Required, "deploy-"+resource.File)
	}
	for _, c := range r.h.spec.Cases {
		out.Scenarios = append(out.Scenarios, c.ID)
	}
	for id, v := range r.h.state.Results {
		out.Checks[id] = Check{v.Status, v.Detail, v.ExecutionID, v.ElapsedSeconds}
	}
	return out
}

// PublicTask exposes required interfaces and results, never executor implementations.
func PublicTask(repository string) (map[string]any, error) {
	return PublicTaskFromRoot(filepath.Join(repository, "ci/evals/e2e"))
}
func PublicTaskFromRoot(root string) (map[string]any, error) {
	var spec suite
	if err := read(filepath.Join(root, "suite.json"), &spec); err != nil {
		return nil, err
	}
	roles := []map[string]any{}
	for _, r := range spec.Resources {
		var d map[string]any
		if err := read(filepath.Join(root, "definitions", r.File), &d); err != nil {
			return nil, err
		}
		contract := map[string]any{"file": r.File, "resource": r.Resource}
		for _, key := range []string{"name", "catalog", "technicalName", "inputKeys", "outputKeys", "keys", "values"} {
			if v, ok := d[key]; ok {
				contract[key] = v
			}
		}
		roles = append(roles, contract)
	}
	return map[string]any{"resources": roles, "scenarios": spec.Cases, "mcpBusinessKeys": spec.ContractExpected, "schedule": spec.Schedule, "mcpLifecycle": spec.MCPLifecycle}, nil
}

func NativeDependencies() []string {
	return []string{"utils-sapcp:Void:1", "utils-sapcp:ForEach:2", "utils-sapcp:ParallelForEach:1", "http-sapcp:HttpRequest:1", "scripts-sapcp:ExecutePythonScript:1", "store-sapcp:GetValue:1", "store-sapcp:PutValue:1", "store-sapcp:DeleteValue:1"}
}

// Resume reuses persisted candidate paths and the journal; it never generates or deploys.
func Resume(ctx context.Context, repository, directory, tenant string) (*Runner, error) {
	// Cleanup relies on recorded IDs and fixed harness metadata. Missing or
	// moved candidate files must not strand already accepted tenant resources.
	opts := Options{Repository: repository, Directory: directory, Tenant: tenant, Reference: true}
	r, err := open(ctx, opts, true)
	if err != nil {
		return nil, err
	}
	r.cleanupOnly = true
	return r, nil
}
