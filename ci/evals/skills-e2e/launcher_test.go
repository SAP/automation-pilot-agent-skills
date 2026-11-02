package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHarnessUsesVerifiedLauncher(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "installed plugin & spaces!")
	target := runtime.GOOS + "-" + runtime.GOARCH + "/autopi"
	launcher := "autopi"
	if runtime.GOOS == "windows" {
		target, launcher = "windows-amd64/autopi.exe", "autopi.cmd"
	}
	for _, name := range []string{"autopi", "autopi.cmd", "verify-autopi.ps1", "bin/SHA256SUMS", "bin/" + target} {
		source := filepath.Join("..", "..", "..", "skills", "automation-pilot-cli", "scripts", filepath.FromSlash(name))
		data, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0755); err != nil {
			t.Fatal(err)
		}
	}
	run := func() (string, string, error) {
		command := exec.Command(filepath.Join(dir, launcher), "api", "POST", "probe", "-f", "-", "--header", "X-Probe: a&b!", "--print-request")
		command.Stdin = strings.NewReader(`{"ordinary":"kept","password":"synthetic-secret"}`)
		command.Env = append(os.Environ(), "AUTOMATION_PILOT_HOSTNAME=eval.invalid", "AUTOMATION_PILOT_TOKEN=synthetic-token")
		var out, errout bytes.Buffer
		command.Stdout, command.Stderr = &out, &errout
		err := runCLI(command)
		return out.String(), errout.String(), err
	}
	out, errout, err := run()
	if err != nil || !strings.Contains(out, "a&b!") || !strings.Contains(out, "kept") || strings.Contains(out, "synthetic-secret") {
		t.Fatalf("launcher forwarding: %v stdout=%s stderr=%s", err, out, errout)
	}
	if err := os.WriteFile(filepath.Join(dir, "bin", filepath.FromSlash(target)), []byte("tampered"), 0755); err != nil {
		t.Fatal(err)
	}
	out, errout, err = run()
	if err == nil || out != "" || !strings.Contains(errout, "verification failed") {
		t.Fatalf("unverified binary reached harness: %v stdout=%s stderr=%s", err, out, errout)
	}
}
