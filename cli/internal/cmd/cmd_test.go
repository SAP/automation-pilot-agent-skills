package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newServer starts a TLS test server recording the last request it saw and
// returns the server plus a context whose host points at it.
func newServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, func() []string) {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)
	host := strings.TrimPrefix(srv.URL, "https://")
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", host)
	t.Setenv("AUTOMATION_PILOT_TOKEN", "test-token")
	return srv, nil
}

func runCmd(args ...string) (string, string, int) {
	var out, errb bytes.Buffer
	c := &cmdContext{stdout: &out, stderr: &errb, version: "test"}
	code := dispatch(c, args)
	return out.String(), errb.String(), code
}

// TestEveryVerbFlagSet dispatches every registered verb under --print-request so
// a verb-local flag that clashes with a global (which panics at flagset build
// time) can never ship silently again. We don't assert output here — only that
// no verb panics while constructing its flagset.
func TestEveryVerbFlagSet(t *testing.T) {
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "h")
	t.Setenv("AUTOMATION_PILOT_TOKEN", "test-token")
	for _, r := range registry() {
		for _, v := range r.verbs {
			name := r.name + " " + v.name
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						t.Errorf("%s panicked: %v", name, rec)
					}
				}()
				// Dummy id + inline body so verbs needing either still build
				// their flagset and reach the --print-request path.
				runCmd(r.name, v.name, "dummy:Id:1", "--data", "{}", "--print-request")
			}()
		}
	}
}

func TestRepeatableStatusQuery(t *testing.T) {
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "h")
	out, _, code := runCmd("executions", "list", "--status", "FAILED", "--status", "RUNNING", "--print-request")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out, "status=FAILED&status=RUNNING") {
		t.Fatalf("want repeated status keys, got:\n%s", out)
	}
}

// TestTagFilterUsesEquals verifies the --tag list filter is forwarded as the
// server's key=value form. The CLI now accepts `key=value` directly, which
// lets tag keys containing colons (e.g. `feature:logs`) pass through intact.
// A key-only filter (no `=`) is forwarded as the bare key.
func TestTagFilterUsesEquals(t *testing.T) {
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "h")
	out, _, code := runCmd("executions", "list", "--tag", "skilltest=1019", "--tag", "keyonly", "--print-request")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out, "tag=skilltest%3D1019") {
		t.Fatalf("want tag=skilltest=1019 (encoded), got:\n%s", out)
	}
	if !strings.Contains(out, "tag=keyonly") || strings.Contains(out, "keyonly%3D") {
		t.Fatalf("key-only tag should pass through unchanged, got:\n%s", out)
	}
	// Colon-in-key: feature:logs=true must not be split on the colon.
	out2, _, code2 := runCmd("executions", "list", "--tag", "feature:logs=true", "--print-request")
	if code2 != 0 {
		t.Fatalf("exit = %d", code2)
	}
	want := "tag=feature%3Alogs%3Dtrue"
	if !strings.Contains(out2, want) {
		t.Fatalf("want %s (feature:logs=true encoded), got:\n%s", want, out2)
	}
}

func TestMergePatchContentType(t *testing.T) {
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "h")
	out, _, _ := runCmd("inputs", "patch", "cat:Cred:1", "--data", `{"values":{"p":"x"}}`, "--print-request")
	if !strings.Contains(out, "Content-Type: application/merge-patch+json") {
		t.Fatalf("want merge-patch content type, got:\n%s", out)
	}
}

func TestTriggerBuildsBody(t *testing.T) {
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "h")
	out, _, _ := runCmd("executions", "trigger", "--command-id", "c:C:1", "--input", "a=b", "--logs", "--print-request")
	for _, want := range []string{`"commandId": "c:C:1"`, `"inputValues": "***"`, `"feature:logs": ""`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestConfigErrorExit(t *testing.T) {
	// No credentials and a real (non-print) call -> exit 2.
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "")
	t.Setenv("AUTOMATION_PILOT_TOKEN", "")
	_, errb, code := runCmd("catalogs", "list")
	if code != exitConfig {
		t.Fatalf("exit = %d, want %d (%s)", code, exitConfig, errb)
	}
}

func TestLegacyOnlyConfigErrorExit(t *testing.T) {
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "")
	t.Setenv("AUTOMATION_PILOT_TOKEN", "")
	t.Setenv("AUTOPI_HOSTNAME", "legacy.example")
	t.Setenv("AUTOPI_USERNAME", "legacy-user")
	t.Setenv("AUTOPI_PASSWORD", "private-legacy-password")
	_, errb, code := runCmd("catalogs", "list")
	if code != exitConfig {
		t.Fatalf("legacy-only configuration: exit=%d, want %d", code, exitConfig)
	}
	for _, want := range []string{"AUTOMATION_PILOT_HOSTNAME", "AUTOMATION_PILOT_TOKEN"} {
		if !strings.Contains(errb, want) {
			t.Errorf("config error missing %s", want)
		}
	}
	if strings.Contains(errb, "private-legacy-password") {
		t.Fatal("config error exposed legacy password")
	}
}

func TestConnectionFlagsRejected(t *testing.T) {
	for _, args := range [][]string{
		{"catalogs", "list"},
		{"api", "GET", "catalogs"},
	} {
		for _, flag := range []string{"--host", "--user", "--password", "--token"} {
			cmdArgs := append(append([]string{}, args...), flag, "private-override", "--print-request")
			out, errb, code := runCmd(cmdArgs...)
			if code != exitUsage {
				t.Fatalf("%s: exit=%d, want usage error", flag, code)
			}
			if strings.Contains(out+errb, "private-override") {
				t.Fatalf("%s error exposed its value", flag)
			}
		}
	}
}

func TestAPIRejectsAuthorizationOverride(t *testing.T) {
	for _, header := range []string{
		"Authorization: Bearer private-override",
		"authorization: Basic private-override",
		" AUTHORIZATION : Bearer private-override",
		"aUtHoRiZaTiOn: Bearer private-override",
		"Authorization",
	} {
		for _, preview := range []bool{false, true} {
			args := []string{"api", "GET", "catalogs", "--header", header}
			if preview {
				args = append(args, "--print-request")
			}
			out, errb, code := runCmd(args...)
			if code != exitUsage {
				t.Fatalf("custom Authorization accepted (preview=%t): exit=%d", preview, code)
			}
			if out != "" || strings.Contains(errb, "private-override") {
				t.Fatal("rejected Authorization exposed request or credential")
			}
		}
	}
}

func TestTimeoutFlagWithoutEnvironmentDefault(t *testing.T) {
	t.Setenv("AUTOPI_TIMEOUT", "120s")
	for _, tc := range []struct {
		flag string
		want time.Duration
	}{
		{"", 30 * time.Second},
		{"45s", 45 * time.Second},
		{"60", 60 * time.Second},
	} {
		cfg, err := (&globalFlags{timeout: tc.flag}).config()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Timeout != tc.want {
			t.Errorf("--timeout %q: got %s, want %s", tc.flag, cfg.Timeout, tc.want)
		}
	}
}

func TestStatusOnlyAgainstServer(t *testing.T) {
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/commands/cat:Missing:1" {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(200)
	})
	out, _, code := runCmd("commands", "get", "cat:Missing:1", "--status-only", "--insecure")
	if code != 0 {
		t.Fatalf("status-only should not error on 404, exit=%d", code)
	}
	if strings.TrimSpace(out) != "404" {
		t.Fatalf("want 404, got %q", out)
	}
}

func TestHTTPErrorExitCode(t *testing.T) {
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"message":"forbidden"}`))
	})
	_, _, code := runCmd("catalogs", "list", "--insecure")
	if code != exitHTTP4xx {
		t.Fatalf("exit = %d, want %d", code, exitHTTP4xx)
	}
}

func TestServerRequestShape(t *testing.T) {
	var gotMethod, gotPath, gotAuth, gotCT string
	var gotBody []byte
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = readAll(r)
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	})
	_, _, code := runCmd("commands", "create", "--data", validCommand, "--insecure")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if gotMethod != "POST" || gotPath != "/api/v1/commands" {
		t.Fatalf("got %s %s", gotMethod, gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("unexpected Bearer auth: %q", gotAuth)
	}
	if gotCT != "application/json" {
		t.Fatalf("content-type = %q", gotCT)
	}
	if string(gotBody) != validCommand {
		t.Fatalf("body = %q", gotBody)
	}
}

// readAll drains a request body (small helper to avoid importing io in two places).
func readAll(r *http.Request) ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r.Body)
	return buf.Bytes(), err
}

// commandArray is a stub list body shaped like a real commands listing: each
// element carries both metadata and the heavy blocks a summary must strip.
const commandArray = `[
  {"id":"c:A:1","name":"A","catalog":"c","description":"d","version":1,"tags":{},"inputKeys":{"x":{"type":"string"}},"outputKeys":{},"configuration":{"a":1},"executors":[{"id":"e"}]},
  {"id":"c:B:1","name":"B","catalog":"c","configuration":{},"executors":[]}
]`

func TestListSummarizedByDefault(t *testing.T) {
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(commandArray))
	})
	out, _, code := runCmd("commands", "list", "--catalog", "c", "--insecure")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	for _, keep := range []string{`"id"`, `"name"`, `"description"`} {
		if !strings.Contains(out, keep) {
			t.Errorf("default list dropped metadata %s:\n%s", keep, out)
		}
	}
	for _, drop := range []string{"inputKeys", "outputKeys", "configuration", "executors"} {
		if strings.Contains(out, drop) {
			t.Errorf("default list leaked heavy key %q:\n%s", drop, out)
		}
	}
}

func TestListFullKeepsHeavyBlocks(t *testing.T) {
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(commandArray))
	})
	out, _, code := runCmd("commands", "list", "--catalog", "c", "--full", "--insecure")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	for _, want := range []string{"inputKeys", "configuration", "executors"} {
		if !strings.Contains(out, want) {
			t.Errorf("--full dropped heavy key %q:\n%s", want, out)
		}
	}
}

func TestGetNotSummarized(t *testing.T) {
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"id":"c:A:1","name":"A","configuration":{"a":1},"executors":[{"id":"e"}]}`))
	})
	out, _, code := runCmd("commands", "get", "c:A:1", "--insecure")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	for _, want := range []string{"configuration", "executors"} {
		if !strings.Contains(out, want) {
			t.Errorf("get should return full object, missing %q:\n%s", want, out)
		}
	}
}

func TestIdsNotSummarized(t *testing.T) {
	// `commands ids` has no --full flag; passing it must be rejected rather than
	// silently accepted, proving the flag was never registered on that verb.
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`["c:A:1","c:B:1"]`))
	})
	_, _, code := runCmd("commands", "ids", "--catalog", "c", "--full", "--insecure")
	if code == 0 {
		t.Fatal("`commands ids --full` should fail: ids has no --full flag")
	}
}

// TestTriggerCommandReferenceInQuery pins Finding 2: generic-event must send
// commandReference as a query parameter (the API rejects it in the body) with
// the event payload as the request body.
func TestTriggerCommandReferenceInQuery(t *testing.T) {
	var gotPath, gotQuery string
	var gotBody []byte
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotBody, _ = readAll(r)
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"RUNNING"}`))
	})
	_, _, code := runCmd("triggers", "generic-event", "--command-reference", "c:C:1",
		"--data", `{"type":"alert"}`, "--insecure")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if gotPath != "/api/v1/triggers/generic-event" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotQuery != "commandReference=c%3AC%3A1" {
		t.Fatalf("want commandReference in query, got %q", gotQuery)
	}
	if string(gotBody) != `{"type":"alert"}` {
		t.Fatalf("body should be the event payload, got %q", gotBody)
	}
}

func TestTriggerRequiresCommandReference(t *testing.T) {
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "h")
	_, _, code := runCmd("triggers", "generic-event", "--data", `{"type":"alert"}`, "--print-request")
	if code != exitUsage {
		t.Fatalf("missing --command-reference should be a usage error, exit=%d", code)
	}
}

// TestIdsFilteredByCatalog pins Finding 3: /commands/ids ignores ?catalog=
// server-side, so the CLI filters the id list by the "<catalog>:" prefix.
func TestIdsFilteredByCatalog(t *testing.T) {
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`["c:A:1","c:B:1","other:Z:1"]`))
	})
	out, _, code := runCmd("commands", "ids", "--catalog", "c", "--insecure")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if !strings.Contains(out, "c:A:1") || !strings.Contains(out, "c:B:1") {
		t.Errorf("dropped in-catalog ids:\n%s", out)
	}
	if strings.Contains(out, "other:Z:1") {
		t.Errorf("leaked out-of-catalog id:\n%s", out)
	}
}

// TestDeployCreatesWithoutID pins Finding 4: a file with no top-level id (e.g.
// an mcp-server on first deploy) must create (POST) rather than erroring.
func TestDeployCreatesWithoutID(t *testing.T) {
	var gotMethod, gotPath string
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{"id":"gen-123"}`))
	})
	_, _, code := runCmd("mcp-servers", "deploy", "--data", `{"name":"srv","mcpTools":[]}`, "--insecure")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if gotMethod != "POST" || gotPath != "/api/v1/mcp-servers" {
		t.Fatalf("id-less deploy should POST /mcp-servers, got %s %s", gotMethod, gotPath)
	}
}

// TestConnectionInfoComposesSummary verifies `connection info` makes the three
// GETs and composes a non-secret summary (mirroring MCP get_connection_info).
func TestConnectionInfoComposesSummary(t *testing.T) {
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("connection info request missing Bearer token")
		}
		w.WriteHeader(200)
		switch {
		case strings.HasSuffix(r.URL.Path, "/server-info"):
			w.Write([]byte(`{"autoPiTenantUrl":"https://t.example"}`))
		case strings.HasSuffix(r.URL.Path, "/user-information"):
			w.Write([]byte(`{"username":"svc","permissions":["Read","Write","Execute"],"appPlan":"STANDARD"}`))
		case strings.HasSuffix(r.URL.Path, "/tenant"):
			w.Write([]byte(`{"id":"T001"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	out, _, code := runCmd("connection", "info", "--insecure")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if strings.Contains(out, "test-token") {
		t.Fatal("connection summary exposed token")
	}
	for _, want := range []string{`"connected": true`, `"url": "https://t.example"`,
		`"tenantId": "T001"`, `"username": "svc"`, `"Execute"`, `"plan": "STANDARD"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// TestCommandContractStripsConfiguration verifies `commands contract` keeps the
// contract fields (inputKeys/outputKeys) but drops configuration/executors.
func TestCommandContractStripsConfiguration(t *testing.T) {
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"id":"c:A:1","name":"A","catalog":"c","description":"d","version":1,"tags":{},"inputKeys":{"x":{"type":"string"}},"outputKeys":{},"configuration":{"a":1},"executors":[{"id":"e"}]}`))
	})
	out, _, code := runCmd("commands", "contract", "c:A:1", "--insecure")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	for _, keep := range []string{"inputKeys", "outputKeys", `"id"`, `"name"`} {
		if !strings.Contains(out, keep) {
			t.Errorf("contract dropped %q:\n%s", keep, out)
		}
	}
	for _, drop := range []string{"configuration", "executors"} {
		if strings.Contains(out, drop) {
			t.Errorf("contract leaked %q:\n%s", drop, out)
		}
	}
}

// TestInputGetStripsValues verifies `inputs get` omits the secret values block
// by default, and includes it under --full (mirroring MCP get_input).
func TestInputGetStripsValues(t *testing.T) {
	const body = `{"id":"c:Cred:1","name":"Cred","keys":{"password":{"type":"string","sensitive":true}},"values":{"password":"s3cret"}}`
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(body))
	})
	out, _, code := runCmd("inputs", "get", "c:Cred:1", "--insecure")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if strings.Contains(out, "values") || strings.Contains(out, "s3cret") {
		t.Errorf("default inputs get leaked values:\n%s", out)
	}
	if !strings.Contains(out, `"keys"`) {
		t.Errorf("default inputs get should keep non-secret keys block:\n%s", out)
	}
	outFull, _, code := runCmd("inputs", "get", "c:Cred:1", "--full", "--insecure")
	if code != 0 {
		t.Fatalf("exit=%d (--full)", code)
	}
	if !strings.Contains(outFull, "values") {
		t.Errorf("--full should include values:\n%s", outFull)
	}
}
