package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
)

func TestCodexMCPOverridePreservesLiteralNamesWithoutCredentials(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	names := []string{"node_repl", "company.api", "quoted\"name"}
	servers := map[string]any{}
	for _, name := range names {
		servers[name] = map[string]any{"command": "unused-server", "env": map[string]any{"API_KEY": "config-only-secret"}}
	}
	raw, err := toml.Marshal(map[string]any{"model_provider": "saved-provider", "mcp_servers": servers})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, grader := range []bool{false, true} {
		args, err := codexArgs(hostConfig{Kind: "codex"}, "/case", grader)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for i, arg := range args {
			if arg != "-c" || !strings.HasPrefix(args[i+1], "mcp_servers") {
				continue
			}
			found = true
			if strings.HasPrefix(args[i+1], "mcp_servers.") || strings.Contains(args[i+1], "config-only-secret") || strings.Contains(args[i+1], "unused-server") {
				t.Fatal("unsafe MCP override", args[i+1])
			}
			var parsed map[string]any
			if err := toml.Unmarshal([]byte(args[i+1]), &parsed); err != nil {
				t.Fatal(err)
			}
			disabled := parsed["mcp_servers"].(map[string]any)
			if len(disabled) != len(names) {
				t.Fatal("server names changed", disabled)
			}
			for _, name := range names {
				server, ok := disabled[name].(map[string]any)
				if !ok || len(server) != 1 || server["enabled"] != false {
					t.Fatal("literal server name lost", name, disabled)
				}
			}
		}
		if !found {
			t.Fatal("MCP override missing")
		}
	}
}

// Local config rendering only: this never starts an eval or samples a model.
// Opt in because the debug command is unavailable in older Codex versions.
func TestCodexOfflineConfigurationParsing(t *testing.T) {
	if os.Getenv("AUTOPI_EVAL_OFFLINE_CODEX_PROBE") != "1" {
		t.Skip("set AUTOPI_EVAL_OFFLINE_CODEX_PROBE=1 for an installed-CLI offline config check")
	}
	program, err := exec.LookPath("codex")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	marker := filepath.Join(dir, "mcp-started")
	mcp := fakeCLI(t, "touch "+shellQuote(marker)+"\nexit 1\n")
	servers := map[string]any{}
	for _, name := range []string{"node_repl", "company.api", "quoted\"name"} {
		servers[name] = map[string]any{"command": mcp}
	}
	raw, err := toml.Marshal(map[string]any{"mcp_servers": servers})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, grader := range []bool{false, true} {
		h := hostConfig{Kind: "codex", Program: program, Isolation: "native"}
		root := testHostRoot(t, h)
		args, err := codexArgs(h, root, grader)
		if err != nil {
			t.Fatal(err)
		}
		// Reuse the exact config/sandbox overrides while replacing exec with
		// prompt rendering. Discard rendered private instructions from logs.
		debugArgs := []string{}
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-c" || args[i] == "-a" || args[i] == "-s" || args[i] == "-C" {
				debugArgs = append(debugArgs, args[i], args[i+1])
				i++
			}
		}
		debugArgs = append(debugArgs, "debug", "prompt-input")
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		cmd := exec.CommandContext(ctx, program, debugArgs...)
		cmd.Env = hostEnvironment(root, "native", "", "", "")
		containProcess(cmd)
		_, err = cmd.Output()
		cancel()
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				t.Fatal(processError("Codex offline config", err, redactedDiagnostic(string(exit.Stderr))))
			}
			t.Fatal(err)
		}
		if _, err := os.Stat(marker); !os.IsNotExist(err) {
			t.Fatal("configured MCP server was launched", err)
		}
	}
}

func TestCodexStartupStderrSurvivesWithSecretsRedacted(t *testing.T) {
	t.Setenv("EVAL_TEST_API_KEY", "environment-secret")
	program := fakeCLI(t, "cat >/dev/null\ncat <<'DIAGNOSTIC' >&2\nError: invalid transport in mcp_servers.node_repl\nenvironment-secret synthetic-secret\nAuthorization: Bearer config-bearer\napi_key=config-key token=config-token\nhttps://alice:config-password@example.invalid/\nDIAGNOSTIC\nexit 1\n")
	h := hostConfig{Kind: "codex", Program: program, Isolation: "native"}
	root := testHostRoot(t, h)
	out := runCodex(context.Background(), root, h, nil, "public prompt", "", "synthetic-secret", "", 5)
	if out.Err == nil || !strings.Contains(out.Err.Error(), "invalid transport") || len(out.Evidence.Trace) != 0 {
		t.Fatal("startup error detail lost", out.Err)
	}
	for _, secret := range []string{"environment-secret", "synthetic-secret", "config-bearer", "config-key", "config-token", "config-password"} {
		if strings.Contains(out.Err.Error(), secret) {
			t.Fatal("secret in persistent diagnostics", secret)
		}
	}
}

func TestProcessDiagnosticBoundsNoiseAndRetainsFinalError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stderr")
	if err := os.WriteFile(path, []byte(strings.Repeat("startup noise\n", 5000)+"Error: invalid transport\n"), 0600); err != nil {
		t.Fatal(err)
	}
	detail := processDiagnostic(path)
	if len(detail) > 2003 || !strings.HasSuffix(detail, "Error: invalid transport") {
		t.Fatal("diagnostic unbounded or final error lost")
	}
}
