package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func fixtureRequest(t *testing.T, f *fixtureTenant, method, path string, body any, headers map[string]string) (int, any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(mustJSON(body))
	}
	r, err := http.NewRequest(method, f.server.URL+"/api/v1/"+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Authorization", "Bearer "+f.token)
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	res, err := f.server.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if len(bytes.TrimSpace(raw)) > 0 {
		if err = json.Unmarshal(raw, &value); err != nil {
			t.Fatal(err)
		}
	}
	return res.StatusCode, value
}
func TestVerifiedCLIAgainstFixtureWithNormalTLS(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("fixture requires the POSIX launcher")
	}
	f := startFixture(fixtureSpec{Tenant: "fixture-tenant", Resources: map[string]any{}}, "synthetic-cli-token")
	defer f.close()
	dir := t.TempDir()
	ca := filepath.Join(dir, "ca.pem")
	if err := f.writeCA(ca); err != nil {
		t.Fatal(err)
	}
	launcher, err := filepath.Abs("../../../skills/automation-pilot-cli/scripts/autopi")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(launcher, "connection", "info")
	cmd.Env = append(os.Environ(), "AUTOMATION_PILOT_HOSTNAME="+strings.TrimPrefix(f.server.URL, "https://"), "AUTOMATION_PILOT_TOKEN=synthetic-cli-token", "SSL_CERT_FILE="+ca)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("verified CLI with fixture CA: %v\n%s", err, out)
	}
	var identity map[string]any
	if json.Unmarshal(out, &identity) != nil || identity["tenantId"] != "fixture-tenant" {
		t.Fatal("fixture identity lost")
	}
}
func TestFixturePreservesSecretsAndRejectsUnexpectedCalls(t *testing.T) {
	c := findCase(t, "input-secret-value-rotation")
	f := startFixture(*c.Fixture, "fixture-token")
	defer f.close()
	path := "inputs/eval-fixture:ApiConfiguration:1"
	status, value := fixtureRequest(t, f, "GET", path, nil, nil)
	if status != 200 || strings.Contains(string(mustJSON(value)), "SYNTHETIC-OLD") {
		t.Fatal("stored secret exposed")
	}
	status, _ = fixtureRequest(t, f, "GET", "unsupported-collection", nil, nil)
	if status != 404 {
		t.Fatal("unknown route returned generic success")
	}
	status, _ = fixtureRequest(t, f, "PATCH", path, map[string]any{"values": map[string]any{"clientSecret": "replacement"}}, map[string]string{"If-Match": "\"wrong\""})
	if status != 412 {
		t.Fatal(status)
	}
	status, _ = fixtureRequest(t, f, "DELETE", "catalogs/unrelated", nil, nil)
	if status != 403 {
		t.Fatal("unauthorized write accepted")
	}
	requests, _ := f.snapshot()
	if requests[len(requests)-1].Unauthorized != true {
		t.Fatal("denied attempt not captured")
	}
	for _, r := range requests {
		if _, exists := r.Headers["Authorization"]; exists {
			t.Fatal("authorization recorded")
		}
	}
}
func TestReplayAllFixtureReferenceActions(t *testing.T) {
	for _, c := range testBank(t).Cases {
		if c.Fixture == nil {
			continue
		}
		t.Run(c.ID, func(t *testing.T) {
			f := startFixture(*c.Fixture, "fixture-token")
			defer f.close()
			for _, r := range c.Reference.Requests {
				status, _ := fixtureRequest(t, f, r.Method, r.Path, r.Body, r.Headers)
				if status != r.Status {
					t.Fatalf("%s %s: got %d want %d", r.Method, r.Path, status, r.Status)
				}
			}
			requests, state := f.snapshot()
			e := c.Reference
			e.Requests = requests
			e.State = state
			for _, a := range c.Assertions {
				if a.Source == "requests" || a.Source == "state" {
					p := assertionResult(a, e)
					if p.Status != "PASS" {
						t.Fatalf("%s: %+v", a.ID, p)
					}
				}
			}
		})
	}
}
func TestMCPFixtureClientSchemaValidatesThroughActualCLI(t *testing.T) {
	c := findCase(t, "publish-mcp-bound-tool")
	f := startFixture(*c.Fixture, "fixture-token")
	defer f.close()
	for _, r := range c.Reference.Requests {
		fixtureRequest(t, f, r.Method, r.Path, r.Body, r.Headers)
	}
	_, schema := fixtureRequest(t, f, "POST", "mcp/fixture-created", map[string]any{"jsonrpc": "2.0", "id": 3, "method": "tools/list", "params": map[string]any{}}, nil)
	dir := t.TempDir()
	definition := c.Reference.State["mcp-servers/fixture-created"]
	metadata := map[string]any{"commands": map[string]any{}, "inputs": map[string]any{}}
	for path, obj := range c.Fixture.Resources {
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 && (parts[0] == "commands" || parts[0] == "inputs") {
			metadata[parts[0]].(map[string]any)[parts[1]] = obj
		}
	}
	for name, value := range map[string]any{"server.mcp.json": definition, "metadata.json": metadata, "tools-list.json": schema} {
		if err := writeJSON(filepath.Join(dir, name), value); err != nil {
			t.Fatal(err)
		}
	}
	cli, err := filepath.Abs("../../../skills/automation-pilot-cli/scripts/autopi")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(cli, "mcp-servers", "validate", "-f", filepath.Join(dir, "server.mcp.json"), "--metadata", filepath.Join(dir, "metadata.json"), "--client-schema", filepath.Join(dir, "tools-list.json"))
	cmd.Env = append(os.Environ(), "AUTOMATION_PILOT_HOSTNAME=", "AUTOMATION_PILOT_TOKEN=")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("actual CLI rejected fixture schema: %v\n%s", err, out)
	}
}

func TestPinnedLauncherIgnoresPoisonedShellConnection(t *testing.T) {
	root := t.TempDir()
	source, err := filepath.Abs("../../../skills/automation-pilot-cli/scripts")
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "plugin/skills/automation-pilot-cli/scripts")
	if err = copyTree(source, target); err != nil {
		t.Fatal(err)
	}
	f := startFixture(fixtureSpec{Tenant: "synthetic-only", Resources: map[string]any{}}, "fixture-only-token")
	defer f.close()
	ca := filepath.Join(root, "ca.pem")
	if err = f.writeCA(ca); err != nil {
		t.Fatal(err)
	}
	if err = pinCaseConnection(root, strings.TrimPrefix(f.server.URL, "https://"), "fixture-only-token", ca); err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(target, "autopi")
	cmd := exec.Command("/bin/sh", "-c", `export AUTOMATION_PILOT_HOSTNAME=127.0.0.1:1 AUTOMATION_PILOT_TOKEN=poisoned SSL_CERT_FILE=/missing; exec "$1" connection info`, "fixture", launcher)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pinned launcher: %v %s", err, out)
	}
	var identity map[string]any
	if json.Unmarshal(out, &identity) != nil || identity["tenantId"] != "synthetic-only" {
		t.Fatal("shell connection escaped fixture", string(out))
	}
	requests, _ := f.snapshot()
	if len(requests) != 3 {
		t.Fatal("fixture identity reads missing", requests)
	}
}
