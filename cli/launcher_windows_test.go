package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func launcherCommand(ctx context.Context, launcher string, args []string) *exec.Cmd {
	command := exec.CommandContext(ctx, filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe"))
	// cmd.exe needs its own outer quoting for a batch path containing spaces.
	// EscapeArg preserves JSON for the native binary; the fixtures keep shell
	// metacharacters in a separate quoted header. Delayed expansion stays off.
	line := `cmd.exe /d /v:off /s /c ""` + launcher + `"`
	for _, arg := range args {
		line += " " + syscall.EscapeArg(arg)
	}
	command.SysProcAttr = &syscall.SysProcAttr{CmdLine: line + `"`}
	return command
}

func TestLauncherRequiresSuccessfulVerifierExit(t *testing.T) {
	for _, status := range []string{"1", "-1"} {
		t.Run(status, func(t *testing.T) {
			b := newLauncherBundle(t)
			writeLauncherFile(t, filepath.Join(b.dir, "verify-autopi.ps1"), []byte("exit "+status+"\r\n"), 0644)
			result := b.run(t, []string{"ERRORLEVEL=0"}, "version")
			if result.code == 0 || result.stdout != "" {
				t.Fatalf("failed verifier did not produce a failure status: code=%d stdout=%q stderr=%q", result.code, result.stdout, result.stderr)
			}
		})
	}
}

func TestLauncherVerificationWithoutPowerShellModules(t *testing.T) {
	b := newLauncherBundle(t)
	verifier := filepath.Join(b.dir, "verify-autopi.ps1")
	data, err := os.ReadFile(verifier)
	if err != nil {
		t.Fatal(err)
	}
	// Reproduce unavailable utility cmdlets regardless of the runner's module paths.
	data = append([]byte("$PSModuleAutoLoadingPreference = 'None'\r\n"), data...)
	writeLauncherFile(t, verifier, data, 0644)
	env := []string{"PSModulePath=" + t.TempDir()}
	result := b.run(t, env, "version")
	if result.code != 0 || !strings.HasPrefix(result.stdout, "autopi ") || result.stderr != "" {
		t.Fatalf("verification depends on PowerShell modules: code=%d stdout=%q stderr=%q", result.code, result.stdout, result.stderr)
	}
	writeLauncherFile(t, b.binary, []byte("tampered"), 0755)
	requireLauncherRejection(t, b.run(t, env, "version"))
}

func TestLauncherPreservesExitStatusWithShadowedErrorLevel(t *testing.T) {
	b := newLauncherBundle(t)
	for _, shadow := range []string{"0", "99"} {
		t.Run(shadow, func(t *testing.T) {
			env := []string{"ERRORLEVEL=" + shadow}
			result := b.run(t, env, "version")
			if result.code != 0 || !strings.HasPrefix(result.stdout, "autopi ") {
				t.Fatalf("valid CLI call: code=%d stdout=%q stderr=%q", result.code, result.stdout, result.stderr)
			}
			result = b.run(t, env, "version", "unexpected")
			if result.code != 1 {
				t.Fatalf("CLI usage error lost: code=%d stdout=%q stderr=%q", result.code, result.stdout, result.stderr)
			}
		})
	}
}
