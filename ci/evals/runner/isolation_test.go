package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestEvaluatedProcessCannotReadOracleOrWriteRuntime(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("evaluated-process isolation is supported and acceptance-tested on Linux only")
	}
	parent := t.TempDir()
	root := filepath.Join(parent, "agent")
	for _, name := range []string{"workspace", "plugin"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	private := filepath.Join(parent, "hidden-oracle.txt")
	canary := "PRIVATE-ORACLE-CANARY"
	if err := os.WriteFile(private, []byte(canary), 0600); err != nil {
		t.Fatal(err)
	}
	visible := filepath.Join(root, "workspace", "public.txt")
	if err := os.WriteFile(visible, []byte("public fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	runtimeAsset := filepath.Join(root, "plugin", "SKILL.md")
	if err := os.WriteFile(runtimeAsset, []byte("runtime"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd, err := isolatedCommand(ctx, root, "/bin/cat", []string{visible})
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "public fixture" {
		t.Fatalf("public read failed: %v %s", err, out)
	}
	cmd, err = isolatedCommand(ctx, root, "/bin/cat", []string{private})
	if err != nil {
		t.Fatal(err)
	}
	out, err = cmd.CombinedOutput()
	if err == nil || strings.Contains(string(out), canary) {
		t.Fatal("oracle readable", err, string(out))
	}
	cmd, err = isolatedCommand(ctx, root, "/bin/sh", []string{"-c", "printf compromised > \"$1\"", "fixture", runtimeAsset})
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Run(); err == nil {
		t.Fatal("runtime mutable")
	}
	original, err := os.ReadFile(runtimeAsset)
	if err != nil || string(original) != "runtime" {
		t.Fatal("runtime changed")
	}
}
func TestCandidateSymlinksAreRejected(t *testing.T) {
	root := t.TempDir()
	if err := os.Symlink("/private/oracle", filepath.Join(root, "submission.json")); err != nil {
		t.Skip(err)
	}
	if _, err := snapshotWorkspace(root); err == nil {
		t.Fatal("symlink accepted")
	}
}
