package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
				t.Fatalf("failed verifier allowed CLI execution: code=%d stdout=%q stderr=%q", result.code, result.stdout, result.stderr)
			}
		})
	}
}
