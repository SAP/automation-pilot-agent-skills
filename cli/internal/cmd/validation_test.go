package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPValidateOfflineSchemaLeak(t *testing.T) {
	fixture := filepath.Join("testdata", "mcp-validation")
	args := []string{"mcp-servers", "validate", "-f", filepath.Join(fixture, "server.mcp.json"), "--metadata", filepath.Join(fixture, "metadata.json")}
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "")
	t.Setenv("AUTOMATION_PILOT_TOKEN", "")
	output, errb, code := runCmd(args...)
	if code != 0 || !strings.Contains(output, `"resourceName"`) {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, output, errb)
	}
	output, errb, code = runCmd(append(args, "--client-schema", filepath.Join(fixture, "tools-list.json"))...)
	if code != exitLint || !strings.Contains(output, "unexpected keys [apiToken]") {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, output, errb)
	}
}

func TestMCPValidateLiveReadsOnlyAndCaches(t *testing.T) {
	requests := 0
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected request: %s", r.Method)
		}
		switch r.URL.Path {
		case "/api/v1/commands/c:FindResource:1":
			io.WriteString(w, `{"inputKeys":{"resourceName":{"type":"string","required":true},"apiToken":{"type":"string","sensitive":true}}}`)
		case "/api/v1/inputs/c:Configuration:1":
			io.WriteString(w, `{"keys":{"apiToken":{"type":"string","sensitive":true}},"values":{"apiToken":"fixture-secret-value"}}`)
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
		}
	})
	file := filepath.Join(t.TempDir(), "server.mcp.json")
	if err := os.WriteFile(file, []byte(`{"name":"resource-lookup","mcpTools":[{"name":"find_resource","commandId":"c:FindResource:1","inputReferences":["c:Configuration:1"]},{"name":"find_other","commandId":"c:FindResource:1","inputReferences":["c:Configuration:1"]}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	out, errb, code := runCmd("mcp-servers", "validate", "-f", file, "--insecure")
	if code != 0 || requests != 2 {
		t.Fatalf("code=%d requests=%d stderr=%s", code, requests, errb)
	}
	if strings.Contains(out, "fixture-secret-value") || strings.Contains(errb, "fixture-secret-value") {
		t.Fatal("Input values leaked")
	}
}

func TestMCPValidateFailedReadAndBadInputsDoNotWrite(t *testing.T) {
	requests := 0
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != "GET" {
			t.Fatal("validation sent a mutation")
		}
		w.WriteHeader(403)
		io.WriteString(w, `{"error":"fixture-secret-value"}`)
	})
	fixture := filepath.Join("testdata", "mcp-validation", "server.mcp.json")
	for _, args := range [][]string{
		{"mcp-servers", "validate", "-f", fixture, "--metadata", ""},
		{"mcp-servers", "validate", "-f", fixture, "--client-schema", "missing-file.json"},
		{"mcp-servers", "validate", "-f", fixture, "--print-request"},
	} {
		if _, _, code := runCmd(args...); code != exitUsage {
			t.Fatalf("%v exit=%d", args, code)
		}
	}
	if requests != 0 {
		t.Fatal("invalid prerequisite caused reads")
	}
	out, errb, code := runCmd("mcp-servers", "validate", "-f", fixture, "--insecure")
	if code != exitHTTP4xx || requests != 1 || strings.Contains(out+errb, "fixture-secret-value") {
		t.Fatalf("code=%d reads=%d stdout=%s stderr=%s", code, requests, out, errb)
	}
}

func TestLintNativeAndOptionalPythonChecks(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "CallApi.command.json")
	doc := map[string]any{"name": "CallApi", "catalog": "c", "version": 1, "inputKeys": map[string]any{}, "outputKeys": map[string]any{}, "configuration": map[string]any{"executors": []any{map[string]any{"alias": "call", "execute": "scripts-sapcp:ExecutePythonScript:1", "input": map[string]any{"script": "broken = [x for x in [1] if True else []"}, "autoRetry": map[string]any{"logic": "CONSTANT", "delay": "5s"}}}}}
	encoded, _ := json.Marshal(doc)
	if err := os.WriteFile(file, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	out, _, code := runCmd("lint", file, "--output", "json")
	if code != exitLint || !strings.Contains(out, "retry logic") {
		t.Fatalf("native lint code=%d out=%s", code, out)
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("optional syntax test needs Python 3")
	}
	out, errb, code := runCmd("lint", file, "--python", python, "--output", "json")
	if code != exitLint || !strings.Contains(out, "invalid Python syntax") || strings.Contains(out, "broken =") {
		t.Fatalf("code=%d out=%s stderr=%s", code, out, errb)
	}
	if _, _, code := runCmd("lint", file, "--python", filepath.Join(dir, "missing-python")); code != exitUsage {
		t.Fatal("missing selected parser accepted")
	}
}
