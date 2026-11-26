package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSingleLauncherUsesEvaluationRootAndPreservesArguments(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX launcher; use WSL")
	}
	root := filepath.Join(t.TempDir(), "repository with spaces & punctuation!")
	for _, dir := range []string{"ci/evals", "tools"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile("../evals.sh")
	if err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(root, "ci/evals/evals.sh")
	if err := os.WriteFile(launcher, data, 0755); err != nil {
		t.Fatal(err)
	}
	stub := filepath.Join(root, "tools/go")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\npwd\nprintf '%s\\n' \"$@\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, suite := range []string{"smoketest", "minimal", "full", "e2e"} {
		command := exec.Command("sh", launcher, "run", "--suite", suite, "--model", "provider/model with spaces")
		command.Dir = t.TempDir()
		command.Env = append(os.Environ(), "PATH="+filepath.Dir(stub)+string(os.PathListSeparator)+os.Getenv("PATH"))
		output, err := command.CombinedOutput()
		want := filepath.Join(root, "ci/evals") + "\nrun\n./runner\nrun\n--suite\n" + suite + "\n--model\nprovider/model with spaces\n"
		if err != nil || string(output) != want {
			t.Fatalf("%v output=%q want=%q", err, output, want)
		}
	}
	if !strings.Contains(string(data), "exec go run ./runner") {
		t.Fatal("launcher does not preserve evaluator exit status")
	}
}
