package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrozenE2ESourcesStayOutsideGoPackageDiscovery(t *testing.T) {
	program, err := exec.LookPath("go")
	if err != nil {
		t.Skip("Go is required for package discovery")
	}
	root := t.TempDir()
	assets := frozenE2EAssets(filepath.Join(root, "run-data"))
	if err := os.MkdirAll(assets, 0700); err != nil {
		t.Fatal(err)
	}
	for path, text := range map[string]string{
		filepath.Join(root, "go.mod"):      "module example.invalid/source-discovery\n\ngo 1.22\n",
		filepath.Join(root, "root.go"):     "package discovery\n",
		filepath.Join(assets, "frozen.go"): "package e2e\n",
	} {
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(program, "list", "./...")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off", "GOTOOLCHAIN=local")
	output, err := cmd.CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "example.invalid/source-discovery" {
		t.Fatalf("private frozen sources entered package discovery: %s (%v)", output, err)
	}
}
