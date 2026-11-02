package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
	"github.com/SAP/automation-pilot-agent-skills/cli/internal/lint"
)

func TestNamedResourceIDsCannotChangeRouting(t *testing.T) {
	ids := []string{
		"victim?force=true", "victim#fragment", "../victim", "victim/child",
		`victim\child`, "victim;parameter=value", ".", "..",
		"victim%3fforce=true", "victim%253fforce=true", "%2e%2e%2finputs",
		"victim%2Fchild", "victim%00", "victim%", "victim\nother", "victim\tother",
	}
	operations := [][]string{
		{"catalogs", "delete"}, {"commands", "get"}, {"inputs", "get"},
		{"mcp-servers", "get"}, {"executions", "get"},
		{"scheduled-executions", "enable"}, {"webhooks", "trigger"},
	}
	for _, operation := range operations {
		for _, id := range ids {
			t.Run(strings.Join(operation, "/")+"/"+id, func(t *testing.T) {
				for _, preview := range []bool{false, true} {
					args := append(append([]string{}, operation...), id)
					if preview {
						args = append(args, "--print-request")
					}
					calls := 0
					code, out, stderr := memoryArgs(t, func(http.ResponseWriter, *http.Request) { calls++ }, "", args...)
					if code != exitUsage || calls != 0 || out != "" {
						t.Fatalf("preview=%v code=%d calls=%d stdout=%q stderr=%q", preview, code, calls, out, stderr)
					}
				}
			})
		}
	}
}

func TestDeployValidatesExplicitAndEmbeddedResourceIDs(t *testing.T) {
	for _, id := range []string{"server?force=true", "server%3Fforce=true", "../server"} {
		for _, embedded := range []bool{false, true} {
			body := map[string]any{"name": "test-server", "mcpTools": []any{}}
			args := []string{"mcp-servers", "deploy", "-f", "-"}
			if embedded {
				body["id"] = id
			} else {
				args = append(args, "--id", id)
			}
			raw, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			code, out, stderr := memoryArgs(t, func(http.ResponseWriter, *http.Request) { calls++ }, string(raw), args...)
			if code != exitUsage || calls != 0 || out != "" {
				t.Fatalf("id=%q embedded=%v code=%d calls=%d stdout=%q stderr=%q", id, embedded, code, calls, out, stderr)
			}
		}
	}
}

func TestBindingReadsValidateResourceIDs(t *testing.T) {
	for _, resource := range []string{"command", "input"} {
		calls := 0
		cl := client.New(client.Config{Host: "fixture.example", Token: "fixture-token",
			Transport: memoryTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{"inputKeys":{}}`))}, nil
			})}, io.Discard)
		tool := lint.MCPTool{CommandID: "c:Command:1"}
		wantCalls := 0
		if resource == "command" {
			tool.CommandID = "c:Command%3F:1"
		} else {
			tool.InputReferences = []string{"c:Input%2F:1"}
			wantCalls = 1
		}
		_, err := fetchBindingMetadata(cl, lint.MCPServer{MCPTools: []lint.MCPTool{tool}})
		if err == nil || calls != wantCalls {
			t.Fatalf("%s: err=%v calls=%d", resource, err, calls)
		}
	}
}

func TestExecutorPathsRejectRoutingSyntax(t *testing.T) {
	for _, executor := range []string{"../input", "parent/../input", "parent//child", "/",
		"parent%2Fchild", "parent/%2e%2e/input", "parent?messages=PRIVATE", "parent#child",
		`parent\child`, "parent;parameter=value"} {
		calls := 0
		code, out, stderr := memoryArgs(t, func(http.ResponseWriter, *http.Request) { calls++ }, "",
			"executions", "logs", "execution-id", "--executor", executor)
		if code != exitUsage || calls != 0 || out != "" {
			t.Fatalf("executor=%q code=%d calls=%d stdout=%q stderr=%q", executor, code, calls, out, stderr)
		}
	}
}

func TestNamedPathsAndIntentionalQueriesRemainSupported(t *testing.T) {
	tests := []struct {
		args  []string
		path  string
		query string
	}{
		{[]string{"commands", "get", "c:Read.Resource:1"}, "/api/v1/commands/c:Read.Resource:1", ""},
		{[]string{"catalogs", "delete", "victim", "--force"}, "/api/v1/catalogs/victim", "force=true"},
		{[]string{"api", "DELETE", "catalogs/victim?force=true"}, "/api/v1/catalogs/victim", "force=true"},
		{[]string{"executions", "logs", "execution-id", "--executor", "parent[0]/child"}, "/api/v1/executions/execution-id/logs/parent[0]/child", ""},
	}
	for _, tc := range tests {
		calls := 0
		code, _, stderr := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.URL.Path != tc.path || r.URL.RawQuery != tc.query {
				t.Errorf("unexpected request URL %s", r.URL)
			}
			w.WriteHeader(http.StatusNoContent)
		}, "", tc.args...)
		if code != exitOK || calls != 1 {
			t.Fatalf("args=%v code=%d calls=%d stderr=%s", tc.args, code, calls, stderr)
		}
	}
}
