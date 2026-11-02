package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// These tests exercise the shipped launchers and binaries in disposable bundles.
// Requests use synthetic credentials and --print-request, so no API is contacted.
type launcherBundle struct {
	dir, target, binary, sums string
}

func newLauncherBundle(t *testing.T) launcherBundle {
	t.Helper()
	arch := runtime.GOARCH
	if runtime.GOOS == "windows" {
		arch = "amd64" // The Windows launcher always selects this bundled target.
	}
	switch runtime.GOOS + "-" + arch {
	case "darwin-amd64", "darwin-arm64", "linux-amd64", "linux-arm64", "windows-amd64":
	default:
		t.Skip("no bundled launcher target for this platform")
	}
	b := launcherBundle{
		dir:    filepath.Join(t.TempDir(), "bundle with spaces & symbols!"),
		target: runtime.GOOS + "-" + arch + "/autopi",
	}
	if runtime.GOOS == "windows" {
		b.target += ".exe"
	}
	b.binary = filepath.Join(b.dir, "bin", filepath.FromSlash(b.target))
	b.sums = filepath.Join(b.dir, "bin", "SHA256SUMS")
	if err := os.MkdirAll(filepath.Dir(b.binary), 0755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join("..", "skills", "automation-pilot-cli", "scripts")
	for _, name := range []string{"autopi", "autopi.cmd", "verify-autopi.ps1", "bin/SHA256SUMS", "bin/" + b.target} {
		path := filepath.Join(source, filepath.FromSlash(name))
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		writeLauncherFile(t, filepath.Join(b.dir, filepath.FromSlash(name)), data, info.Mode().Perm())
	}
	return b
}

func writeLauncherFile(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
}

type launcherResult struct {
	code           int
	stdout, stderr string
}

func (b launcherBundle) run(t *testing.T, env []string, args ...string) launcherResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	name := "autopi"
	if runtime.GOOS == "windows" {
		name += ".cmd"
	}
	command := launcherCommand(ctx, filepath.Join(b.dir, name), args)
	command.Env = append(os.Environ(), env...)
	command.Env = append(command.Env,
		"AUTOMATION_PILOT_HOSTNAME=cli-test.invalid",
		"AUTOMATION_PILOT_TOKEN=synthetic-test-token")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	if ctx.Err() != nil {
		t.Fatalf("launcher timed out: %v; stderr=%s", ctx.Err(), &stderr)
	}
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		t.Fatalf("could not start launcher: %v", err)
	}
	return launcherResult{command.ProcessState.ExitCode(), stdout.String(), stderr.String()}
}

func requireLauncherRejection(t *testing.T, result launcherResult) {
	t.Helper()
	if result.code == 0 || result.stdout != "" || !strings.Contains(strings.ToLower(result.stderr), "verification failed") {
		t.Fatalf("expected verification failure: code=%d stdout=%q stderr=%q", result.code, result.stdout, result.stderr)
	}
}

func TestLauncherVerifiedBinaryRuns(t *testing.T) {
	result := newLauncherBundle(t).run(t, nil, "version")
	if result.code != 0 || !strings.HasPrefix(result.stdout, "autopi ") {
		t.Fatalf("code=%d stdout=%q stderr=%q", result.code, result.stdout, result.stderr)
	}
}

func TestLauncherRejectsUnverifiedBundles(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, launcherBundle)
	}{
		{"missing checksum list", func(t *testing.T, b launcherBundle) {
			if err := os.Remove(b.sums); err != nil {
				t.Fatal(err)
			}
		}},
		{"missing target entry", func(t *testing.T, b launcherBundle) {
			data, err := os.ReadFile(b.sums)
			if err != nil {
				t.Fatal(err)
			}
			var kept []string
			for _, line := range strings.SplitAfter(string(data), "\n") {
				if !strings.HasSuffix(strings.TrimSpace(line), b.target) {
					kept = append(kept, line)
				}
			}
			writeLauncherFile(t, b.sums, []byte(strings.Join(kept, "")), 0644)
		}},
		{"duplicate target entry", func(t *testing.T, b launcherBundle) {
			data, err := os.ReadFile(b.sums)
			if err != nil {
				t.Fatal(err)
			}
			for _, line := range strings.SplitAfter(string(data), "\n") {
				if strings.HasSuffix(strings.TrimSpace(line), b.target) {
					writeLauncherFile(t, b.sums, append(data, line...), 0644)
					return
				}
			}
			t.Fatal("fixture lacks the selected checksum entry")
		}},
		{"malformed checksum", func(t *testing.T, b launcherBundle) {
			writeLauncherFile(t, b.sums, []byte("not-a-sha256  "+b.target+"\n"), 0644)
		}},
		{"wrong checksum", func(t *testing.T, b launcherBundle) {
			writeLauncherFile(t, b.sums, []byte(strings.Repeat("0", 64)+"  "+b.target+"\n"), 0644)
		}},
		{"tampered binary", func(t *testing.T, b launcherBundle) {
			data, err := os.ReadFile(b.binary)
			if err != nil {
				t.Fatal(err)
			}
			writeLauncherFile(t, b.binary, append(data, "corrupted-after-release"...), 0755)
		}},
		{"missing binary", func(t *testing.T, b launcherBundle) {
			if err := os.Remove(b.binary); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b := newLauncherBundle(t)
			test.mutate(t, b)
			requireLauncherRejection(t, b.run(t, nil, "version"))
		})
	}
}

func TestLauncherPreservesArgumentsAndExitStatus(t *testing.T) {
	b := newLauncherBundle(t)
	want := map[string]string{"message": "spaces & ampersand ! punctuation"}
	payload, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	// JSON supplies embedded quotes and backslashes. The header supplies literal
	// shell metacharacters (json.Marshal escapes '&' in the payload).
	header := "X-Message: " + want["message"]
	result := b.run(t, nil, "api", "POST", "checks", "--data", string(payload), "--header", header, "--print-request")
	head, body, ok := strings.Cut(strings.ReplaceAll(result.stdout, "\r\n", "\n"), "\n\n")
	if result.code != 0 || !ok {
		t.Fatalf("code=%d stdout=%q stderr=%q", result.code, result.stdout, result.stderr)
	}
	if !strings.Contains("\n"+head+"\n", "\n"+header+"\n") {
		t.Fatalf("forwarded headers = %q, want %q", head, header)
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) || got["message"] != want["message"] {
		t.Fatalf("forwarded payload = %v, want %v", got, want)
	}
	result = b.run(t, nil, "version", "unexpected")
	if result.code != 1 {
		t.Fatalf("expected CLI exit status 1: code=%d stdout=%q stderr=%q", result.code, result.stdout, result.stderr)
	}
}
