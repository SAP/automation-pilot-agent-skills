package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type hostConfig struct {
	Kind, Program, Model, Effort, Isolation string
}

func resolveHosts(agent, agentBin, grader, graderBin, model, graderModel, effort, graderEffort, isolation string) (hostConfig, hostConfig, error) {
	a, err := resolveHost(agent, agentBin, model, effort, isolation)
	if err != nil {
		return a, hostConfig{}, err
	}
	if grader == "" {
		grader = a.Kind
		if graderBin == "" {
			graderBin = a.Program
		}
	}
	g, err := resolveHost(grader, graderBin, graderModel, graderEffort, isolation)
	return a, g, err
}

func resolveHost(kind, program, model, effort, isolation string) (hostConfig, error) {
	if isolation != "native" && isolation != "bubblewrap" {
		return hostConfig{}, fmt.Errorf("--isolation must be native or bubblewrap")
	}
	if kind == "auto" {
		if program != "" {
			return hostConfig{}, fmt.Errorf("--agent-bin requires an explicit --agent claude or codex")
		}
		for _, candidate := range []string{"claude", "codex"} {
			if _, err := exec.LookPath(candidate); err == nil {
				kind = candidate
				break
			}
		}
	}
	if kind != "claude" && kind != "codex" {
		return hostConfig{}, fmt.Errorf("choose --agent claude or codex; its CLI must be installed and logged in")
	}
	if program == "" {
		program = kind
	}
	path, err := exec.LookPath(program)
	if err != nil {
		return hostConfig{}, fmt.Errorf("%s executable unavailable: %w", kind, err)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return hostConfig{}, err
	}
	return hostConfig{Kind: kind, Program: path, Model: model, Effort: effort, Isolation: isolation}, nil
}

func codexConfigDir() string {
	if dir := os.Getenv("CODEX_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex")
}

func readCodexConfig() (map[string]any, error) {
	path := filepath.Join(codexConfigDir(), "config.toml")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read Codex config")
	}
	var config map[string]any
	if toml.Unmarshal(raw, &config) != nil {
		return nil, fmt.Errorf("cannot parse Codex config")
	}
	return config, nil
}

func codexArgs(h hostConfig, root string, grader bool) ([]string, error) {
	sandbox := "workspace-write"
	if grader {
		sandbox = "read-only"
	}
	args := []string{"-a", "never", "exec", "--json", "--ephemeral", "--skip-git-repo-check", "--color", "never", "-C", filepath.Join(root, "workspace"), "-s", sandbox, "-c", "sandbox_workspace_write.network_access=true"}
	// Keep authentication, provider and model settings. Bound evaluation tools.
	for _, setting := range []string{"agents.enabled=false", "features.multi_agent=false", "features.multi_agent_v2=false", "features.shell_snapshot=false", "shell_environment_policy.experimental_use_profile=false", "web_search=\"disabled\""} {
		args = append(args, "-c", setting)
	}
	discovery := "false"
	if grader {
		discovery = "true"
	}
	args = append(args, "-c", "features.skip_host_skill_discovery="+discovery)
	config, err := readCodexConfig()
	if err != nil {
		return nil, err
	}
	if servers, ok := config["mcp_servers"].(map[string]any); ok {
		disabled := map[string]any{}
		for _, name := range sortedKeys(servers) {
			disabled[name] = map[string]any{"enabled": false}
		}
		if len(disabled) > 0 {
			// Codex splits dotted override paths literally: quotes become part
			// of the server name. Encode names inside a TOML table instead.
			// Only enabled flags enter argv, never server URLs or credentials.
			var override strings.Builder
			if err := toml.NewEncoder(&override).SetTablesInline(true).Encode(map[string]any{"mcp_servers": disabled}); err != nil {
				return nil, fmt.Errorf("cannot encode disabled Codex MCP servers: %w", err)
			}
			args = append(args, "-c", strings.TrimSpace(override.String()))
		}
	}
	if h.Model != "" {
		args = append(args, "--model", h.Model)
	}
	if h.Effort != "" {
		args = append(args, "-c", "model_reasoning_effort="+string(mustJSON(h.Effort)))
	}
	if grader {
		// The judge receives evidence as data and needs no execution or skill tools.
		args = append(args, "-c", "features.shell_tool=false", "-c", "features.unified_exec=false", "-c", "features.code_mode=false", "--output-schema", filepath.Join(root, "grade.schema.json"))
	}
	return args, nil
}

func hostConfigDigest(h hostConfig) string {
	files := map[string]string{}
	home, _ := os.UserHomeDir()
	paths := []string{filepath.Join(home, ".claude", "settings.json"), filepath.Join(home, ".claude", "CLAUDE.md")}
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		paths = []string{filepath.Join(dir, "settings.json"), filepath.Join(dir, "CLAUDE.md")}
	}
	if h.Kind == "codex" {
		paths = []string{filepath.Join(codexConfigDir(), "config.toml"), filepath.Join(codexConfigDir(), "AGENTS.md")}
	}
	for _, path := range paths {
		if raw, err := os.ReadFile(path); err == nil {
			files[path] = bytesDigest(raw)
		}
	}
	// Only hashes enter reports; credentials and configuration text never do.
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if key == "CLAUDECODE" || key == "CLAUDE_CODE_ENTRYPOINT" || key == "CODEX_THREAD_ID" || key == "CODEX_SESSION_ID" || key == "CODEX_TOOL_CALL_ID" || key == "CODEX_INTERNAL_ORIGINATOR_OVERRIDE" || key == "CODEX_SANDBOX" {
			continue
		}
		if strings.HasPrefix(key, "ANTHROPIC_") || strings.HasPrefix(key, "CLAUDE_") || strings.HasPrefix(key, "CODEX_") || strings.HasPrefix(key, "OPENAI_") {
			files["env:"+key] = bytesDigest([]byte(value))
		}
	}
	return digest(files)
}
