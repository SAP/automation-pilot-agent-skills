package main

import (
	"context"
	"fmt"
	"github.com/SAP/automation-pilot-agent-skills/ci/evals/internal/redact"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func copyTree(from, to string) error {
	return filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in isolated assets: %s", path)
		}
		rel, _ := filepath.Rel(from, path)
		target := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("nonregular isolated asset")
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}
func copyFile(from, to string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(to), 0700); err != nil {
		return err
	}
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	return err
}
func isolatedCommand(ctx context.Context, root, program string, args []string) (*exec.Cmd, error) {
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	program, err = filepath.EvalSymlinks(program)
	if err != nil {
		return nil, err
	}
	switch runtime.GOOS {
	case "linux":
		sandbox, err := exec.LookPath("bwrap")
		if err != nil {
			return nil, fmt.Errorf("Linux isolation requires bubblewrap (bwrap)")
		}
		argv := []string{"--die-with-parent", "--new-session", "--unshare-all", "--share-net", "--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp"}
		for _, dir := range []string{"/usr", "/bin", "/sbin", "/lib", "/lib64", "/etc/ssl", "/etc/resolv.conf", "/etc/hosts", "/etc/nsswitch.conf"} {
			if _, err := os.Stat(dir); err == nil {
				argv = append(argv, "--ro-bind", dir, dir)
			}
		}
		argv = append(argv, "--bind", root, root, "--ro-bind", filepath.Join(root, "plugin"), filepath.Join(root, "plugin"))
		if _, err := os.Stat(filepath.Join(root, "workspace/.agents")); err == nil {
			dir := filepath.Join(root, "workspace/.agents")
			argv = append(argv, "--ro-bind", dir, dir)
		}
		argv = append(argv, "--ro-bind", program, program, "--chdir", filepath.Join(root, "workspace"), "--", program)
		argv = append(argv, args...)
		return exec.CommandContext(ctx, sandbox, argv...), nil
	default:
		return nil, fmt.Errorf("evaluated sessions require Linux with bubblewrap; offline check, mine, grade and report remain available on %s", runtime.GOOS)
	}
}
func agentEnvironment(root, hostname, token, ca string) []string {
	return hostEnvironment(root, "native", hostname, token, ca)
}
func hostEnvironment(root, isolation, hostname, token, ca string) []string {
	env := []string{}
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if strings.HasPrefix(key, "AUTOMATION_PILOT_") || strings.HasPrefix(key, "AUTOPI_") || key == "TMPDIR" || key == "CLAUDECODE" || key == "CLAUDE_CODE_ENTRYPOINT" {
			continue
		}
		if isolation == "bubblewrap" && (key == "HOME" || key == "CLAUDE_CONFIG_DIR" || key == "CODEX_HOME") {
			continue
		}
		if ca != "" && key == "SSL_CERT_FILE" {
			continue
		}
		env = append(env, value)
	}
	env = append(env, "TMPDIR="+filepath.Join(root, "tmp"), "CLAUDE_CODE_DISABLE_AUTO_MEMORY=1")
	if isolation == "bubblewrap" {
		env = append(env, "HOME="+filepath.Join(root, "home"), "CLAUDE_CONFIG_DIR="+filepath.Join(root, "home/.claude"), "CODEX_HOME="+filepath.Join(root, "home/.codex"))
	}
	if hostname != "" {
		env = append(env, "AUTOMATION_PILOT_HOSTNAME="+hostname, "AUTOMATION_PILOT_TOKEN="+token)
	}
	if ca != "" {
		env = append(env, "SSL_CERT_FILE="+ca)
	}
	return env
}
func hostCommand(ctx context.Context, root string, h hostConfig, args []string) (*exec.Cmd, error) {
	if h.Isolation == "bubblewrap" {
		return isolatedCommand(ctx, root, h.Program, args)
	}
	return exec.CommandContext(ctx, h.Program, args...), nil
}
func prepareHostRuntime(root string, h hostConfig, grader bool) error {
	for _, name := range []string{"workspace", "config", "tmp", "home", "plugin"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0700); err != nil {
			return err
		}
	}
	if h.Kind == "codex" && !grader {
		// Standard workspace discovery; copy the complete plugin to preserve all
		// sibling references and verified launchers without global installation.
		if err := copyTree(filepath.Join(root, "plugin"), filepath.Join(root, "workspace/.agents")); err != nil {
			return err
		}
	}
	if h.Isolation != "bubblewrap" {
		return nil
	}
	home, _ := os.UserHomeDir()
	configDir := os.Getenv("CLAUDE_CONFIG_DIR")
	if configDir == "" {
		configDir = filepath.Join(home, ".claude")
	}
	// Copy only runtime configuration and authentication, never the eval bank,
	// user skill installations, conversation history or project workspaces.
	paths := map[string]string{}
	if h.Kind == "claude" {
		for _, name := range []string{"settings.json", ".credentials.json"} {
			paths[filepath.Join(configDir, name)] = filepath.Join(root, "home/.claude", name)
		}
		paths[filepath.Join(home, ".claude.json")] = filepath.Join(root, "home/.claude.json")
	} else {
		for _, name := range []string{"config.toml", "auth.json"} {
			paths[filepath.Join(codexConfigDir(), name)] = filepath.Join(root, "home/.codex", name)
		}
	}
	for from, to := range paths {
		if _, err := os.Stat(from); os.IsNotExist(err) {
			continue
		}
		if err := copyFile(from, to, 0600); err != nil {
			return fmt.Errorf("cannot stage %s runtime configuration", h.Kind)
		}
	}
	return nil
}
func stageRuntime(ctx context.Context, repository, root, ablation string) error {
	cmd := exec.CommandContext(ctx, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", filepath.Join(repository, "ci/plugin/main.go"), "stage", repository, filepath.Join(root, "plugin"))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("runtime staging: %w: %s", err, out)
	}
	for _, name := range []string{"workspace", "config", "tmp", "home"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0700); err != nil {
			return err
		}
	}
	if ablation == "without-skills" {
		// Both conditions retain the same binaries and launcher at the same path.
		// Only skill guidance, examples and reference documents are removed.
		return filepath.WalkDir(filepath.Join(root, "plugin", "skills"), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "scripts" {
					return filepath.SkipDir
				}
				return nil
			}
			return os.Remove(path)
		})
	}
	return nil
}
func snapshotWorkspace(root string) (map[string]string, error) {
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("candidate workspace contains symlink")
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		files[filepath.ToSlash(rel)] = bytesDigest(b)
		return nil
	})
	return files, err
}
func changedFiles(before, after map[string]string) []string {
	out := []string{}
	all := map[string]bool{}
	for k := range before {
		all[k] = true
	}
	for k := range after {
		all[k] = true
	}
	for _, k := range sortedKeys(all) {
		if before[k] != after[k] {
			out = append(out, k)
		}
	}
	return out
}

func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }

// Login profiles and saved shell snapshots can restore a real tenant token.
// Pin the evaluator's staged launcher after its checksum verification so every
// allowed CLI operation uses the case's synthetic connection (or no connection).
func pinCaseConnection(root, hostname, token, ca string) error {
	for _, base := range []string{"plugin", "workspace/.agents"} {
		path := filepath.Join(root, base, "skills/automation-pilot-cli/scripts/autopi")
		raw, err := os.ReadFile(path)
		if os.IsNotExist(err) && base != "plugin" {
			continue
		}
		if err != nil {
			return err
		}
		original := "exec \"$bin\" \"$@\""
		if strings.Count(string(raw), original) != 1 {
			return fmt.Errorf("unexpected verified launcher shape")
		}
		connection := "exec env AUTOMATION_PILOT_HOSTNAME=" + shellQuote(hostname) + " AUTOMATION_PILOT_TOKEN=" + shellQuote(token) + " SSL_CERT_FILE=" + shellQuote(ca) + " \"$bin\" \"$@\""
		if err = os.WriteFile(path, []byte(strings.Replace(string(raw), original, connection, 1)), 0700); err != nil {
			return err
		}
	}
	return nil
}
func publicPrompt(c evalCase, root string) string {
	launcher := filepath.Join(root, "plugin/skills/automation-pilot-cli/scripts/autopi")
	return c.Prompt + "\n\nTask environment: the installed plugin is at " + filepath.Join(root, "plugin") + ". All Automation Pilot CLI calls use its verified launcher at " + launcher + ". Public task files are in the current workspace. This path and CLI availability are identical in the skill and no-skill conditions. Use only public task files and installed skill assets; do not search for evaluation definitions or acceptance data. Do not create nested agents.\n"
}
func scrub(text string, secrets []string) string { return redact.Text(text, secrets) }
func sensitiveEnvironmentValues() []string       { return redact.EnvironmentSecrets() }
