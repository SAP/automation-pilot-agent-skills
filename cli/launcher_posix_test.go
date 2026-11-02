//go:build !windows

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func launcherCommand(ctx context.Context, launcher string, args []string) *exec.Cmd {
	return exec.CommandContext(ctx, launcher, args...)
}

func TestLauncherRequiresChecksumUtility(t *testing.T) {
	b := newLauncherBundle(t)
	utilities := t.TempDir()
	for _, name := range []string{"dirname", "uname", "awk"} {
		path, err := exec.LookPath(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(path, filepath.Join(utilities, name)); err != nil {
			t.Fatal(err)
		}
	}
	requireLauncherRejection(t, b.run(t, []string{"PATH=" + utilities}, "version"))
}
