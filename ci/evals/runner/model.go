package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type bank struct {
	Version int        `json:"schema_version"`
	Cases   []evalCase `json:"cases"`
}
type evalCase struct {
	ID         string            `json:"id"`
	Version    int               `json:"version"`
	Family     string            `json:"family"`
	Primary    string            `json:"primary_skill"`
	Secondary  []string          `json:"secondary_skills,omitempty"`
	Domain     string            `json:"domain"`
	Origin     string            `json:"origin"`
	Mode       string            `json:"mode"`
	Tier       string            `json:"tier"`
	Difficulty string            `json:"difficulty"`
	Split      string            `json:"split"`
	Suites     []string          `json:"suites"`
	Prompt     string            `json:"prompt"`
	Files      map[string]string `json:"public_files"`
	Artifact   string            `json:"artifact,omitempty"`
	Sources    []source          `json:"sources"`
	Criteria   []criterion       `json:"criteria"`
	Assertions []assertion       `json:"assertions"`
	Fixture    *fixtureSpec      `json:"fixture,omitempty"`
	Engine     *engineSpec       `json:"engine,omitempty"`
	Reference  evidence          `json:"reference"`
	Mutants    []mutant          `json:"mutants"`
	Review     string            `json:"review_status"`
	MaxTurns   int               `json:"max_turns"`
	MaxSeconds int               `json:"max_seconds"`
	Replies    []userReply       `json:"scripted_user_replies,omitempty"`
}
type source struct {
	Repository string `json:"repository"`
	Revision   string `json:"revision"`
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
	Evidence   string `json:"evidence"`
}
type criterion struct {
	ID          string `json:"id"`
	Requirement string `json:"requirement"`
}
type assertion struct {
	ID       string         `json:"id"`
	Source   string         `json:"source"`
	Pointer  string         `json:"pointer,omitempty"`
	Operator string         `json:"operator"`
	Value    any            `json:"value,omitempty"`
	Where    map[string]any `json:"where,omitempty"`
	Min      *int           `json:"min,omitempty"`
	Max      *int           `json:"max,omitempty"`
}
type engineSpec struct {
	Kind                string      `json:"kind,omitempty"`
	OutputKey           string      `json:"output_key,omitempty"`
	ResultType          string      `json:"result_type"`
	ReferenceExpression string      `json:"reference_expression"`
	Partitions          []partition `json:"partitions"`
}
type partition struct {
	ID       string            `json:"id"`
	Values   map[string]string `json:"input_values"`
	Expected any               `json:"expected"`
}
type fixtureSpec struct {
	Credentials string         `json:"credentials,omitempty"`
	Tenant      string         `json:"tenant"`
	Resources   map[string]any `json:"resources"`
	Routes      []route        `json:"routes,omitempty"`
	Writes      []writeScope   `json:"allowed_writes"`
}
type route struct {
	Method    string     `json:"method"`
	Path      string     `json:"path"`
	Responses []response `json:"responses"`
}
type response struct {
	Status  int               `json:"status"`
	Body    any               `json:"body,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}
type writeScope struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Max    int    `json:"max_calls"`
}
type userReply struct {
	QuestionPattern string `json:"question_pattern"`
	Text            string `json:"text"`
}
type requestRecord struct {
	ResponseBody any                 `json:"response_body,omitempty"`
	Sequence     int                 `json:"sequence"`
	Method       string              `json:"method"`
	Path         string              `json:"path"`
	Query        map[string][]string `json:"query,omitempty"`
	Body         any                 `json:"body,omitempty"`
	Headers      map[string]string   `json:"headers,omitempty"`
	Status       int                 `json:"status"`
	Unauthorized bool                `json:"unauthorized,omitempty"`
}
type evidence struct {
	Answer           string            `json:"answer"`
	Artifact         any               `json:"artifact,omitempty"`
	Requests         []requestRecord   `json:"requests"`
	State            map[string]any    `json:"state"`
	Trace            []json.RawMessage `json:"trace"`
	LintExit         *int              `json:"lint_exit,omitempty"`
	EngineResults    map[string]any    `json:"engine_results,omitempty"`
	WorkspaceChanged []string          `json:"workspace_changed,omitempty"`
	HardFailures     []string          `json:"hard_failures,omitempty"`
}
type mutant struct {
	Name     string   `json:"name"`
	Violates []string `json:"violates"`
	Evidence evidence `json:"evidence"`
}
type predicateResult struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
	Evidence string `json:"evidence,omitempty"`
}
type caseResult struct {
	OriginalStatus   string            `json:"original_status,omitempty"`
	CostKnown        bool              `json:"cost_known"`
	GraderModel      string            `json:"resolved_grader_model,omitempty"`
	GraderUsage      map[string]any    `json:"grader_usage,omitempty"`
	RepairRound      int               `json:"repair_round,omitempty"`
	AgentUsage       map[string]any    `json:"agent_usage,omitempty"`
	ToolCalls        int               `json:"tool_calls"`
	Model            string            `json:"resolved_model,omitempty"`
	BudgetChargedUSD float64           `json:"budget_charged_usd"`
	ID               string            `json:"id"`
	CaseDigest       string            `json:"case_digest"`
	Generation       int               `json:"generation"`
	Primary          string            `json:"primary_skill"`
	Family           string            `json:"family"`
	Tier             string            `json:"tier"`
	Split            string            `json:"split"`
	Review           string            `json:"review_status"`
	Status           string            `json:"status"`
	Reason           string            `json:"reason,omitempty"`
	Predicates       []predicateResult `json:"predicates"`
	CostUSD          float64           `json:"cost_usd"`
	AgentCostUSD     float64           `json:"agent_cost_usd"`
	GraderCostUSD    float64           `json:"grader_cost_usd"`
	ElapsedSeconds   float64           `json:"elapsed_seconds"`
	EvidenceFile     string            `json:"evidence_file,omitempty"`
	Repairs          []caseResult      `json:"repairs,omitempty"`
}
type runReport struct {
	E2E                []e2eRunResult    `json:"e2e,omitempty"`
	E2EDigest          string            `json:"e2e_digest,omitempty"`
	Tenant             string            `json:"tenant_id,omitempty"`
	GraderDigest       string            `json:"grader_digest,omitempty"`
	OriginalReport     string            `json:"original_report,omitempty"`
	ReviewedAt         string            `json:"reviewed_at,omitempty"`
	Parallel           int               `json:"parallel"`
	CaseFilter         string            `json:"case_filter,omitempty"`
	Effort             string            `json:"effort,omitempty"`
	GraderEffort       string            `json:"grader_effort,omitempty"`
	GraderHost         string            `json:"grader_host,omitempty"`
	ConfigDigest       string            `json:"config_digest,omitempty"`
	GraderConfigDigest string            `json:"grader_config_digest,omitempty"`
	Isolation          string            `json:"isolation,omitempty"`
	BudgetMode         string            `json:"budget_mode,omitempty"`
	SkillFilter        string            `json:"skill_filter,omitempty"`
	TierFilter         string            `json:"tier_filter,omitempty"`
	Version            int               `json:"schema_version"`
	ID                 string            `json:"run_id"`
	Started            string            `json:"started_at"`
	Model              string            `json:"model"`
	GraderModel        string            `json:"grader_model"`
	Host               string            `json:"host"`
	Suite              string            `json:"suite"`
	Ablation           string            `json:"ablation"`
	Sources            map[string]string `json:"source_sha256"`
	SourceDigest       string            `json:"source_digest"`
	BankDigest         string            `json:"bank_digest"`
	BudgetUSD          float64           `json:"budget_usd"`
	Results            []caseResult      `json:"results"`
}

func readJSON(path string, value any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 32<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return fmt.Errorf("%s: trailing JSON", path)
	}
	return nil
}
func writeJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".eval-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(append(b, '\n'))
	}
	if e := tmp.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
func digest(v any) string         { b, _ := json.Marshal(v); return bytesDigest(b) }
func bytesDigest(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

var identity = regexp.MustCompile(`^[a-z][a-z0-9-]{2,79}$`)

func safeRelative(path string) bool {
	return path != "" && !filepath.IsAbs(path) && filepath.Clean(path) == path && path != ".." && !strings.HasPrefix(path, ".."+string(filepath.Separator)) && !strings.ContainsAny(path, "\\\x00")
}
func loadBank(path string) (bank, error) {
	var b bank
	err := readJSON(path, &b)
	if err == nil {
		err = validateBank(b)
	}
	return b, err
}
func validateBank(b bank) error {
	if b.Version != 1 || len(b.Cases) == 0 {
		return fmt.Errorf("unsupported or empty bank")
	}
	ids := map[string]bool{}
	families := map[string]string{}
	for _, c := range b.Cases {
		seenSuites := map[string]bool{}
		for _, suite := range c.Suites {
			if err := validateSuite(suite); err != nil {
				return fmt.Errorf("%s: %w", c.ID, err)
			}
			if seenSuites[suite] {
				return fmt.Errorf("%s: duplicate suite %s", c.ID, suite)
			}
			seenSuites[suite] = true
		}
		if !identity.MatchString(c.ID) || ids[c.ID] {
			return fmt.Errorf("invalid or duplicate case ID %q", c.ID)
		}
		ids[c.ID] = true
		if !identity.MatchString(c.Family) || c.Version < 1 || !strings.HasPrefix(c.Primary, "automation-pilot-") {
			return fmt.Errorf("%s: invalid identity metadata", c.ID)
		}
		if c.Tier != "local" && c.Tier != "fixture" && c.Tier != "engine" {
			return fmt.Errorf("%s: invalid tier", c.ID)
		}
		if c.Split != "development" && c.Split != "holdout" {
			return fmt.Errorf("%s: invalid split", c.ID)
		}
		if prior := families[c.Family]; prior != "" && prior != c.Split {
			return fmt.Errorf("%s: family crosses development/holdout boundary", c.ID)
		}
		families[c.Family] = c.Split
		if len(strings.Fields(c.Prompt)) < 30 || len(c.Sources) == 0 || c.MaxTurns < 1 || c.MaxSeconds < 1 {
			return fmt.Errorf("%s: incomplete prompt/provenance/budget", c.ID)
		}
		for name := range c.Files {
			if !safeRelative(name) || strings.HasPrefix(name, ".claude") || strings.HasPrefix(name, "plugin/") {
				return fmt.Errorf("%s: unsafe public path %s", c.ID, name)
			}
		}
		if c.Artifact != "" && !safeRelative(c.Artifact) {
			return fmt.Errorf("%s: unsafe artifact", c.ID)
		}
		if c.Tier == "fixture" && c.Fixture == nil {
			return fmt.Errorf("%s: fixture missing", c.ID)
		}
		if c.Fixture != nil {
			if c.Fixture.Credentials != "" && c.Fixture.Credentials != "missing" {
				return fmt.Errorf("%s: unknown credential fixture mode", c.ID)
			}
			for _, route := range c.Fixture.Routes {
				if !safeRelative(route.Path) || len(route.Responses) == 0 {
					return fmt.Errorf("%s: invalid route fixture", c.ID)
				}
				for _, response := range route.Responses {
					if response.Status < 100 || response.Status > 599 {
						return fmt.Errorf("%s: invalid fixture HTTP status", c.ID)
					}
				}
			}
			for _, scope := range c.Fixture.Writes {
				if !safeRelative(scope.Path) || scope.Max < 1 {
					return fmt.Errorf("%s: invalid write scope", c.ID)
				}
			}
		}
		if c.Tier == "engine" && (c.Engine == nil || len(c.Engine.Partitions) < 2) {
			return fmt.Errorf("%s: engine partitions missing", c.ID)
		}
		if c.Engine != nil && !contains([]string{"string", "number", "boolean", "object", "array"}, c.Engine.ResultType) {
			return fmt.Errorf("%s: invalid engine output type", c.ID)
		}
		names := map[string]bool{}
		for _, a := range c.Assertions {
			if names[a.ID] || a.ID == "" {
				return fmt.Errorf("%s: duplicate assertion", c.ID)
			}
			names[a.ID] = true
			if !knownOperator(a.Operator) {
				return fmt.Errorf("%s: unknown operator %s", c.ID, a.Operator)
			}
		}
		for _, a := range c.Criteria {
			if names[a.ID] || a.ID == "" || a.Requirement == "" {
				return fmt.Errorf("%s: invalid criterion", c.ID)
			}
			names[a.ID] = true
		}
		if len(names) < 3 || len(c.Mutants) < 2 {
			return fmt.Errorf("%s: insufficient requirements or grader challenges", c.ID)
		}
		for _, m := range c.Mutants {
			if m.Name == "" || len(m.Violates) == 0 {
				return fmt.Errorf("%s: mutant lacks intended failure", c.ID)
			}
			for _, id := range m.Violates {
				if !names[id] {
					return fmt.Errorf("%s: unknown mutant criterion %s", c.ID, id)
				}
			}
		}
		for _, s := range c.Sources {
			if !safeRelative(s.Path) || len(s.SHA256) != 64 || s.Revision == "" || s.Evidence == "" {
				return fmt.Errorf("%s: incomplete source", c.ID)
			}
		}
		for _, reply := range c.Replies {
			if _, err := regexp.Compile(reply.QuestionPattern); err != nil || reply.Text == "" {
				return fmt.Errorf("%s: invalid user reply", c.ID)
			}
		}
	}
	return nil
}
func selectCases(b bank, suite, skills, tiers string) []evalCase {
	if validateSuite(suite) != nil || suite == "e2e" {
		return nil
	}
	if suite == "minimal" {
		suite = "pilot"
	}
	has := func(csv, value string) bool { return csv == "" || contains(strings.Split(csv, ","), value) }
	var out []evalCase
	for _, c := range b.Cases {
		if (suite == "full" || suite == "all" || suite == "regression" || suite == "release" || contains(c.Suites, suite)) && has(skills, c.Primary) && has(tiers, c.Tier) {
			out = append(out, c)
		}
	}
	return out
}
func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
func sortedKeys[V any](v map[string]V) []string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func stamp() string { return time.Now().UTC().Format("20060102T150405.000000000Z") }
