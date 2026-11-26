package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func fakeCLI(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake CLI uses a POSIX shell")
	}
	path := filepath.Join(t.TempDir(), "fake-cli")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nset -eu\n"+body), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}
func testHostRoot(t *testing.T, h hostConfig) string {
	t.Helper()
	root := t.TempDir()
	if err := prepareHostRuntime(root, h, true); err != nil {
		t.Fatal(err)
	}
	return root
}
func TestHostDefaultsAndExplicitCheaperModel(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())
	h := hostConfig{Kind: "claude", Isolation: "native"}
	args := claudeArgs(h, "/case", "with-skills", 1)
	for _, forbidden := range []string{"--bare", "--restricted", "--setting-sources", "--model", "--effort"} {
		for _, arg := range args {
			if arg == forbidden {
				t.Fatal("default configuration suppressed:", arg)
			}
		}
	}
	h.Kind = "codex"
	args, err := codexArgs(h, "/case", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"--ignore-user-config", "--ignore-rules", "--model", "--dangerously-bypass-approvals-and-sandbox"} {
		for _, arg := range args {
			if arg == forbidden {
				t.Fatal("default configuration suppressed:", arg)
			}
		}
	}
	h.Model = "smaller-provider-model"
	h.Effort = "low"
	args, err = codexArgs(h, "/case", false)
	if err != nil || !strings.Contains(strings.Join(args, " "), "--model smaller-provider-model") || !strings.Contains(strings.Join(args, " "), `model_reasoning_effort="low"`) {
		t.Fatal(args, err)
	}
	h.Kind = "claude"
	args = claudeArgs(h, "/case", "without-skills", 1)
	if !strings.Contains(strings.Join(args, " "), "--model smaller-provider-model") || !strings.Contains(strings.Join(args, " "), "--disable-slash-commands") {
		t.Fatal(args)
	}
}
func TestNativeEnvironmentRetainsLoginProviderAndStripsTenant(t *testing.T) {
	t.Setenv("HOME", "/saved-home")
	t.Setenv("PATH", "/configured/bin:/usr/bin")
	t.Setenv("CODEX_HOME", "/saved-codex")
	t.Setenv("CLAUDE_CONFIG_DIR", "/saved-claude")
	t.Setenv("OPENAI_BASE_URL", "https://configured-provider.invalid")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("AUTOMATION_PILOT_TOKEN", "real-tenant-secret")
	t.Setenv("AUTOPI_TOKEN", "legacy-secret")
	env := strings.Join(agentEnvironment("/case", "synthetic.invalid", "synthetic-token", ""), "\n")
	for _, want := range []string{"HOME=/saved-home", "PATH=/configured/bin:/usr/bin", "CODEX_HOME=/saved-codex", "CLAUDE_CONFIG_DIR=/saved-claude", "OPENAI_BASE_URL=https://configured-provider.invalid", "AUTOMATION_PILOT_TOKEN=synthetic-token"} {
		if !strings.Contains(env, want) {
			t.Fatal("missing configured environment", want)
		}
	}
	if strings.Contains(env, "real-tenant-secret") || strings.Contains(env, "legacy-secret") {
		t.Fatal("real tenant credentials forwarded")
	}
}
func TestCodexMCPIsDisabledWithoutReplacingProvider(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	raw := `model="configured-model"
model_provider="company-provider"
[mcp_servers."company.api"]
command="internal-server"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	args, err := codexArgs(hostConfig{Kind: "codex"}, "/case", false)
	joined := strings.Join(args, " ")
	if err != nil || !strings.Contains(joined, `mcp_servers = {`) || !strings.Contains(joined, `company.api`) || !strings.Contains(joined, `enabled = false`) || strings.Contains(joined, "model_provider=") || strings.Contains(joined, "--model") {
		t.Fatal(args, err)
	}
}
func TestClaudeSavedLoginAndScriptedReplies(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	capture := filepath.Join(t.TempDir(), "turns.jsonl")
	t.Setenv("EVAL_TEST_CAPTURE", capture)
	program := fakeCLI(t, `n=0
while IFS= read -r turn; do
 printf '%s\n' "$turn" >> "$EVAL_TEST_CAPTURE"
 n=$((n + 1))
 if [ "$n" = 1 ]; then
  printf '%s\n' '{"type":"system","model":"configured-default"}' '{"type":"assistant","message":{"content":[{"type":"text","text":"Which region?"}]}}' '{"type":"result","result":"Which region?","total_cost_usd":0.01,"usage":{"input_tokens":20}}'
 else
  printf '%s\n' '{"type":"assistant","message":{"content":[{"type":"text","text":"Done with selected region."}]}}' '{"type":"result","result":"Done with selected region.","total_cost_usd":0.02,"usage":{"input_tokens":40}}'
 fi
done
`)
	h := hostConfig{Kind: "claude", Program: program, Isolation: "native"}
	root := testHostRoot(t, h)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := evalCase{Prompt: "Do the task.", MaxTurns: 5, Replies: []userReply{{QuestionPattern: `Which region\?`, Text: "Europe"}}}
	out := launchAgent(ctx, c, root, h, "with-skills", "", "", "", 1)
	if out.Err != nil || out.Evidence.Answer != "Done with selected region." || out.Model != "configured-default" || !out.CostKnown || out.Cost != 0.02 {
		t.Fatal(out)
	}
	raw, err := os.ReadFile(capture)
	if err != nil || !strings.Contains(string(raw), "Europe") || len(out.Evidence.Trace) != 5 {
		t.Fatal(string(raw), out.Err, len(out.Evidence.Trace))
	}
}
func TestCodexStreamsCommandsUsageAndEphemeralReplies(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())
	capture := filepath.Join(t.TempDir(), "public-dialogue")
	t.Setenv("EVAL_TEST_CAPTURE", capture)
	program := fakeCLI(t, `cat > "$EVAL_TEST_CAPTURE"
if [ ! -f .continued ]; then
 printf x > .continued
 printf '%s\n' '{"type":"thread.started","thread_id":"first"}' '{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"/case/plugin/skills/automation-pilot-cli/scripts/autopi lint candidate.command.json"}}' '{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"/case/plugin/skills/automation-pilot-cli/scripts/autopi lint candidate.command.json","aggregated_output":"No lint findings","exit_code":0,"status":"completed"}}' '{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"Which region?"}}' '{"type":"turn.completed","usage":{"input_tokens":10,"output_tokens":5}}'
else
 printf '%s\n' '{"type":"thread.started","thread_id":"second"}' '{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"Finished in Europe."}}' '{"type":"turn.completed","usage":{"input_tokens":15,"output_tokens":6}}'
fi
`)
	h := hostConfig{Kind: "codex", Program: program, Isolation: "native"}
	root := testHostRoot(t, h)
	c := evalCase{Prompt: "Original task", MaxTurns: 5, Replies: []userReply{{QuestionPattern: `Which region\?`, Text: "Europe"}}}
	out := launchAgent(context.Background(), c, root, h, "with-skills", "", "", "", 1)
	if out.Err != nil || out.Evidence.Answer != "Finished in Europe." || out.CostKnown || out.Usage["input_tokens"] != float64(25) || !observedLint(out.Evidence.Trace) {
		t.Fatal(out)
	}
	raw, err := os.ReadFile(capture)
	if err != nil || !strings.Contains(string(raw), "Original task") || !strings.Contains(string(raw), "Previous assistant response:\nWhich region?") || !strings.Contains(string(raw), "User reply:\nEurope") {
		t.Fatal(string(raw), err)
	}
	tools := traceTools(out.Evidence.Trace)
	if len(tools) != 2 || tools[0].ID != "first:item_1" || tools[1].Error {
		t.Fatal(tools)
	}
}
func TestStructuredGraderWithBothHosts(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())
	t.Setenv("ANTHROPIC_API_KEY", "")
	c := evalCase{Prompt: "State observed evidence", Criteria: []criterion{{ID: "criterion"}}}
	body := `{"judgments":[{"id":"criterion","pass":true,"evidence_quote":"observable evidence","reason":"observed"}]}`
	for _, kind := range []string{"claude", "codex"} {
		t.Run(kind, func(t *testing.T) {
			output := `{"structured_output":` + body + `,"total_cost_usd":0.03,"modelUsage":{"configured-grader":{}}}`
			if kind == "codex" {
				output = `{"type":"item.completed","item":{"id":"answer","type":"agent_message","text":` + string(mustJSON(body)) + `}}` + "\n" + `{"type":"turn.completed","usage":{"input_tokens":30,"output_tokens":15}}`
			}
			program := fakeCLI(t, "cat >/dev/null\ncat <<'RESULT'\n"+output+"\nRESULT\n")
			report, meta := externalGrade(context.Background(), hostConfig{Kind: kind, Program: program, Isolation: "native"}, c, evidence{Answer: "observable evidence"}, 0.25)
			if meta.Err != nil || len(report.Judgments) != 1 || !report.Judgments[0].Pass || meta.CostKnown != (kind == "claude") {
				t.Fatal(report, meta)
			}
		})
	}
}
func TestCodexFailedCommandCannotEstablishLint(t *testing.T) {
	event := json.RawMessage(`{"type":"item.completed","item":{"id":"cmd","type":"command_execution","command":"scripts/autopi lint candidate.command.json","aggregated_output":"","exit_code":1,"status":"failed"}}`)
	if observedLint([]json.RawMessage{event}) {
		t.Fatal("failed command established lint")
	}
}
func TestCodexRuntimeUsesWorkspaceSkillsAndKeepsSiblingAssets(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"automation-pilot-cli", "automation-pilot-plan"} {
		path := filepath.Join(root, "plugin/skills", name)
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte("skill"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "plugin", "README.md"), []byte("plugin context"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := prepareHostRuntime(root, hostConfig{Kind: "codex", Isolation: "native"}, false); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"skills/automation-pilot-cli/SKILL.md", "skills/automation-pilot-plan/SKILL.md", "README.md"} {
		if _, err := os.Stat(filepath.Join(root, "workspace/.agents", name)); err != nil {
			t.Fatal(err)
		}
	}
}

// Opt-in, bounded integration test. Ordinary Go tests never call a paid model.
func TestConfiguredCLIIntegration(t *testing.T) {
	kind := os.Getenv("AUTOPI_EVAL_LIVE_HOST_PROBE")
	if kind == "" {
		t.Skip("set AUTOPI_EVAL_LIVE_HOST_PROBE=claude|codex to use a configured live CLI")
	}
	h, err := resolveHost(kind, "", "", "", "native")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	repository, _ := filepath.Abs("../../..")
	if err = stageRuntime(context.Background(), repository, root, "with-skills"); err != nil {
		t.Fatal(err)
	}
	if err = prepareHostRuntime(root, h, false); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := evalCase{Prompt: "Use the Automation Pilot CLI skill. Run only the verified launcher with version, then report the version. This is a local request; do not connect to a tenant, inspect other installations, or modify files.", MaxTurns: 15}
	out := launchAgent(ctx, c, root, h, "with-skills", "", "", "", 1)
	if out.Err != nil {
		// Display runtime failures only; never display keys, provider URLs or auth files.
		paths, _ := filepath.Glob(filepath.Join(root, "*stderr*"))
		for _, path := range paths {
			raw, _ := os.ReadFile(path)
			for _, line := range strings.Split(string(raw), "\n") {
				if strings.Contains(line, "panicked") || strings.Contains(line, "panic:") || strings.Contains(line, "unexpected argument") {
					t.Log(line)
				}
			}
		}
		t.Fatal(out.Err)
	}
	if out.Evidence.Answer == "" || len(traceTools(out.Evidence.Trace)) == 0 {
		t.Fatal("missing observable skill/CLI evidence")
	}
	c.Criteria = []criterion{{ID: "reported-version", Requirement: "The answer reports the version observed from the verified CLI launcher."}}
	report, meta := externalGrade(ctx, h, c, out.Evidence, 0.25)
	if meta.Err != nil {
		t.Fatal(meta.Err)
	}
	scores, err := semanticResults(c, out.Evidence, report)
	if err != nil || aggregate(scores) != "PASS" {
		t.Fatal(scores, err)
	}
	t.Log("configured CLI agent and grader passed", kind)
}

func TestClaudeChunksAreOneAssistantTurn(t *testing.T) {
	program := fakeCLI(t, `IFS= read -r prompt
printf '%s\n' '{"type":"assistant","message":{"id":"same-message","content":[{"type":"text","text":"Checking"}]}}' '{"type":"assistant","message":{"id":"same-message","content":[{"type":"tool_use","id":"tool","name":"Read","input":{"file_path":"SKILL.md"}}]}}' '{"type":"result","result":"Done","total_cost_usd":0.01}'
cat >/dev/null
`)
	h := hostConfig{Kind: "claude", Program: program, Isolation: "native"}
	out := launchAgent(context.Background(), evalCase{Prompt: "task", MaxTurns: 1}, testHostRoot(t, h), h, "with-skills", "", "", "", 1)
	if out.Err != nil || out.Evidence.Answer != "Done" {
		t.Fatal(out)
	}
}
func TestCodexGraderHasOneSandboxSelection(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())
	args, err := codexArgs(hostConfig{Kind: "codex"}, "/case", true)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for i, arg := range args {
		if arg == "-s" {
			count++
			if args[i+1] != "read-only" {
				t.Fatal(args)
			}
		}
	}
	if count != 1 {
		t.Fatal("Codex rejects repeated sandbox flags", args)
	}
}
func TestCodexShellEnvelopeDoesNotHideUnsafeLauncher(t *testing.T) {
	for _, command := range []string{`/bin/zsh -lc 'autopi version'`, `/bin/bash -c '/plugin/skills/automation-pilot-cli/scripts/bin/linux-amd64/autopi help'`} {
		raw := mustJSON(map[string]any{"type": "item.started", "item": map[string]any{"id": "tool", "type": "command_execution", "command": command}})
		if !unsafeLauncher([]json.RawMessage{raw}) {
			t.Fatal("unsafe launcher hidden in shell envelope", command)
		}
	}
}
func TestUnpricedSessionsReserveBudgetAndZeroDisablesRunLimit(t *testing.T) {
	options := runOptions{Budget: 2.5, PerCase: 1, PerGrade: 0.25}
	if !canStartSession(1.25, options) || canStartSession(1.26, options) {
		t.Fatal("reservation limit is incorrect")
	}
	options.Budget = 0
	if !validBudget(0) || !canStartSession(10000, options) {
		t.Fatal("zero budget did not disable the run reservation limit")
	}
}
func TestComparisonRefusesChangedConfigurationAndIsolation(t *testing.T) {
	baseline := runReport{Model: "fixed", GraderModel: "grader", Host: "host", Ablation: "with-skills", BankDigest: "bank", ConfigDigest: "before", Isolation: "native"}
	path := filepath.Join(t.TempDir(), "baseline.json")
	if err := writeJSON(path, baseline); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"config", "isolation", "grader"} {
		candidate := baseline
		switch field {
		case "config":
			candidate.ConfigDigest = "after"
		case "isolation":
			candidate.Isolation = "bubblewrap"
		case "grader":
			candidate.GraderHost = "other-host"
		}
		if printReport(candidate, path) == nil {
			t.Fatal("incomparable report accepted", field)
		}
	}
}

func TestCodexRecoverableErrorDoesNotReplayOrAbortTurn(t *testing.T) {
	program := fakeCLI(t, `cat >/dev/null
printf '%s\n' '{"type":"error","message":"Reconnecting after a transient rate limit"}' '{"type":"item.completed","item":{"id":"a","type":"agent_message","text":"Completed after reconnect"}}' '{"type":"turn.completed","usage":{"input_tokens":1,"output_tokens":1}}'
`)
	h := hostConfig{Kind: "codex", Program: program, Isolation: "native"}
	out := runCodex(context.Background(), testHostRoot(t, h), h, nil, "task", "", "", "", 3)
	if out.Err != nil || out.Evidence.Answer != "Completed after reconnect" {
		t.Fatal(out.Err, out.Evidence.Answer)
	}
}

func TestTransientToolIDsDoNotAlterConfigurationIdentity(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	h := hostConfig{Kind: "claude"}
	t.Setenv("CODEX_TOOL_CALL_ID", "first-tool")
	t.Setenv("CODEX_SESSION_ID", "first-session")
	before := hostConfigDigest(h)
	t.Setenv("CODEX_TOOL_CALL_ID", "second-tool")
	t.Setenv("CODEX_SESSION_ID", "second-session")
	if hostConfigDigest(h) != before {
		t.Fatal("transient tool identity contaminated configuration")
	}
}
