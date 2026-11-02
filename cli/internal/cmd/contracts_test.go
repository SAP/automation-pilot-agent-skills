package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type memoryTransport func(*http.Request) (*http.Response, error)

func (f memoryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func memoryArgs(t *testing.T, handler http.HandlerFunc, stdin string, args ...string) (int, string, string) {
	t.Helper()
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "fixture.example")
	t.Setenv("AUTOMATION_PILOT_TOKEN", "fixture-token")
	var out, stderr bytes.Buffer
	c := &cmdContext{stdout: &out, stderr: &stderr, stdin: strings.NewReader(stdin), version: "test", transport: memoryTransport(func(r *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		handler(rec, r)
		res := rec.Result()
		res.Request = r
		return res, nil
	})}
	code := dispatch(c, args)
	return code, out.String(), stderr.String()
}

const validCommand = `{"name":"Hello","catalog":"c","version":1,"inputKeys":{},"outputKeys":{},"configuration":{"executors":[{"alias":"call","execute":"utils-sapcp:Void:1","input":{}}]}}`

func TestRemovedAdapterCommandsRejectBeforeTransport(t *testing.T) {
	for _, verb := range []string{"list", "get", "create", "update", "delete", "deploy", "--help"} {
		t.Run(verb, func(t *testing.T) {
			calls := 0
			code, _, stderr := memoryArgs(t, func(http.ResponseWriter, *http.Request) { calls++ }, "", "a2a-adapters", verb)
			if code != exitUsage || calls != 0 || !strings.Contains(stderr, `unknown command "a2a-adapters"`) {
				t.Fatalf("code=%d calls=%d stderr=%s", code, calls, stderr)
			}
		})
	}
	for _, args := range [][]string{{"help"}, {"surface"}} {
		code, out, stderr := memoryArgs(t, func(http.ResponseWriter, *http.Request) {
			t.Fatal("local help contacted the transport")
		}, "", args...)
		if code != exitOK || strings.Contains(strings.ToLower(out), "a2a") {
			t.Fatalf("code=%d out=%s stderr=%s", code, out, stderr)
		}
	}
}

func TestContractFeaturesETagSelectorsAndConditionalDelete(t *testing.T) {
	calls := 0
	handler := func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/mcp-servers/server-id" {
			t.Error(r.URL.Path)
		}
		if r.Method == "DELETE" {
			if r.Header.Get("If-Match") != `"v1"` {
				t.Error("delete lost ETag")
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		io.WriteString(w, `{"id":"server-id"}`)
	}
	code, _, stderr := memoryArgs(t, handler, "", "mcp-servers", "get", "server-id", "--show-etag", "--status-only")
	if code != exitUsage || calls != 0 {
		t.Fatalf("conflicting selectors: code=%d calls=%d stderr=%s", code, calls, stderr)
	}
	code, _, stderr = memoryArgs(t, handler, "", "mcp-servers", "get", "server-id", "--show-etag")
	if code == exitOK || calls != 1 {
		t.Fatalf("missing ETag: code=%d calls=%d stderr=%s", code, calls, stderr)
	}
	code, _, stderr = memoryArgs(t, handler, "", "mcp-servers", "delete", "server-id", "--if-match", `"v1"`)
	if code != exitOK || calls != 2 {
		t.Fatalf("conditional delete: code=%d calls=%d stderr=%s", code, calls, stderr)
	}
}

func TestContractFeaturesRejectBeforeTransport(t *testing.T) {
	for _, args := range [][]string{
		{"executions", "action", "id", "--type", "PAUSE"},
		{"executions", "action", "id", "--type", "RESET", "--reason", "repair"},
		{"executions", "action", "id", "--type", "SUSPEND", "--reason", "repair"},
		{"executions", "list", "--limit", "1001"},
		{"executions", "logs", "id", "--page", "10001"},
		{"executions", "trigger", "--command-id", "c:C:1", "--wait-seconds", "61"},
		{"executions", "trigger", "--command-id", "c:C:1", "--wait-seconds", "10", "--timeout", "10s"},
		{"inputs", "patch", "c:I:1", "--data", `{"values":{"secret":null}}`},
		{"catalogs", "create", "--data", `{"name":"Demo","technicalName":"bad-name"}`},
		{"commands", "deploy", "--data", `{"catalog":"c","name":"Bad","version":1}`},
		{"inputs", "deploy", "--data", `{"catalog":"c","name":"Bad","version":1,"keys":{}}`, "--print-request"},
		{"commands", "get", "c:Hello:1", "--show-etag", "--response-envelope"},
		{"catalogs", "update", "id", "--data", `{"name":"Demo"}`, "--if-match", "revision"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			calls := 0
			code, _, stderr := memoryArgs(t, func(http.ResponseWriter, *http.Request) { calls++ }, "", args...)
			if code != 1 || calls != 0 {
				t.Fatalf("code=%d calls=%d stderr=%s", code, calls, stderr)
			}
		})
	}
}

func TestContractFeaturesWaitAndBulkOutcomes(t *testing.T) {
	for _, status := range []int{200, 202} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			code, out, stderr := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Prefer") != "wait=60" {
					t.Error("wait header missing")
				}
				w.Header().Set("Preference-Applied", "wait=60")
				w.WriteHeader(status)
				io.WriteString(w, `{"id":"run","status":"FINISHED"}`)
			}, "", "executions", "trigger", "--command-id", "c:C:1", "--wait-seconds", "60", "--response-envelope")
			var envelope struct {
				Status  int
				Headers map[string]string
			}
			if json.Unmarshal([]byte(out), &envelope) != nil || code != 0 || envelope.Status != status || envelope.Headers["Preference-Applied"] != "wait=60" {
				t.Fatalf("code=%d out=%s err=%s", code, out, stderr)
			}
		})
	}
	for _, failed := range []bool{false, true} {
		t.Run("bulk", func(t *testing.T) {
			body := `{"executionIds":["first","first"],"type":"ABORT","reason":"superseded"}`
			code, out, stderr := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/executions/actions" {
					t.Error(r.URL.Path)
				}
				json.NewEncoder(w).Encode(map[string]any{"summary": map[string]int{"total": 2}, "results": []any{map[string]any{"executionId": "first", "succeeded": true}, map[string]any{"executionId": "first", "succeeded": !failed}}})
			}, body, "executions", "bulk-action", "-f", "-")
			want := 0
			if failed {
				want = 7
			}
			if code != want || !strings.Contains(out, "results") {
				t.Fatalf("code=%d out=%s err=%s", code, out, stderr)
			}
		})
	}
}

func TestContractFeaturesConditionalLifecycleAndDeploy(t *testing.T) {
	code, _, stderr := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-Match") != `"rev"` {
			t.Error("ETag changed")
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["id"] != "my-cat:Replacement:1" {
			t.Error(body)
		}
		w.WriteHeader(204)
	}, "", "commands", "deprecate", "c:Hello:1", "--replacement", "my-cat:Replacement:1", "--if-match", `"rev"`)
	if code != 0 {
		t.Fatal(stderr)
	}
	calls := 0
	code, _, stderr = memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method == "GET" {
			w.Header().Set("ETag", `"rev"`)
			io.WriteString(w, validCommand)
			return
		}
		if r.Header.Get("If-Match") != `"rev"` {
			t.Error("deploy lost ETag")
		}
		w.WriteHeader(412)
	}, validCommand, "commands", "deploy", "-f", "-")
	if code != 4 || calls != 2 {
		t.Fatalf("code=%d calls=%d err=%s", code, calls, stderr)
	}
}

func TestContractFeaturesBulkResponseCompleteness(t *testing.T) {
	for _, response := range []string{
		`{"results":[{"executionId":"first","succeeded":true}]}`,
		`{"results":[{"executionId":"second","succeeded":true},{"executionId":"first","succeeded":true}]}`,
		`{"results":[{"executionId":"first","succeeded":true},{"executionId":"second"}]}`,
	} {
		calls := 0
		code, out, _ := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			io.WriteString(w, response)
		}, "", "executions", "bulk-action", "--execution-id", "first", "--execution-id", "second", "--type", "ABORT", "--reason", "test")
		if code == 0 || calls != 1 || !strings.Contains(out, "results") {
			t.Fatalf("incomplete results accepted or lost: code=%d calls=%d out=%s", code, calls, out)
		}
	}
}

func TestContractFeaturesANSBooleanMappingAndFilteredIDs(t *testing.T) {
	for _, value := range []string{"false", "true"} {
		code, _, stderr := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("disableResourcePropertiesMapping") != value || r.URL.Query().Get("disableRegionPropertiesMapping") != value {
				t.Error(r.URL)
			}
			w.WriteHeader(202)
			io.WriteString(w, `{"id":"run"}`)
		}, "", "triggers", "ans-event", "--command-reference", "c:C:1", "--data", `{"eventType":"test","subject":"test","body":"test","category":"ALERT","severity":"INFO","resource":{"resourceName":"fixture","resourceType":"test"}}`, "--disable-resource-properties-mapping="+value, "--disable-region-properties-mapping="+value)
		if code != 0 {
			t.Fatal(stderr)
		}
	}
	code, out, stderr := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/commands" || r.URL.Query().Get("tag") != "team=ops" {
			t.Error(r.URL)
		}
		io.WriteString(w, `[{"id":"c:C:1","configuration":{"secret":"hidden"}}]`)
	}, "", "commands", "ids", "--tag", "team=ops")
	if code != 0 || !strings.Contains(out, "c:C:1") || strings.Contains(out, "hidden") {
		t.Fatalf("code=%d out=%s err=%s", code, out, stderr)
	}
}

func TestContractFeaturesEventFiltersAndWebhooks(t *testing.T) {
	code, _, stderr := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("mapEventToInputKey") != "event" || len(r.URL.Query()["inputReference"]) != 2 || r.Header.Get("Content-Type") != "text/plain" {
			t.Error(r.URL, r.Header)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "raw event" {
			t.Error(string(body))
		}
		w.WriteHeader(202)
		io.WriteString(w, `{"id":"run"}`)
	}, "raw event", "triggers", "generic-event", "--command-reference", "c:C:1", "--input-reference", "c:I:1", "--input-reference", "c:J:1", "--map-event-to-input-key", "event", "--content-type", "text/plain", "-f", "-")
	if code != 0 {
		t.Fatal(stderr)
	}
	code, _, stderr = memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			body, _ := io.ReadAll(r.Body)
			if len(body) != 0 {
				t.Error("unexpected webhook event")
			}
		}
		w.WriteHeader(201)
		io.WriteString(w, `{"id":"run"}`)
	}, "", "webhooks", "trigger", "webhook-id")
	if code != 0 {
		t.Fatal(stderr)
	}
	code, _, stderr = memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("originatorId") != "operator" || q.Get("filterMode") != "contains" {
			t.Error(q)
		}
		io.WriteString(w, `[]`)
	}, "", "executions", "list", "--originator-id", "operator", "--filter-mode", "contains")
	if code != 0 {
		t.Fatal(stderr)
	}
}

func TestContractFeaturesScheduleToggleAndCatalogForce(t *testing.T) {
	calls := 0
	code, _, stderr := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method == "GET" {
			w.Header().Set("ETag", "rev")
			io.WriteString(w, `{"commandId":"c:C:1","schedule":{"hourly":{"minutes":[0]}},"enabled":true,"state":{},"description":"preserve"}`)
			return
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["enabled"] != false || body["description"] != "preserve" || body["state"] != nil || r.Header.Get("If-Match") != "rev" {
			t.Error(body, r.Header)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(body)
	}, "", "scheduled-executions", "disable", "schedule-id")
	if code != 0 || calls != 2 {
		t.Fatalf("code=%d calls=%d err=%s", code, calls, stderr)
	}
	code, out, stderr := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("force") != "true" {
			t.Error(r.URL)
		}
		io.WriteString(w, `{"deletedCommandsCount":2,"deletedInputsCount":1}`)
	}, "", "catalogs", "delete", "catalog-id", "--force")
	if code != 0 || !strings.Contains(out, "deletedCommandsCount") {
		t.Fatalf("code=%d out=%s err=%s", code, out, stderr)
	}
}

func TestContractFeaturesSafeETagAndInputEnvelope(t *testing.T) {
	for _, args := range [][]string{{"inputs", "get", "c:I:1", "--show-etag"}, {"inputs", "create", "--response-envelope", "--data", `{"name":"I","catalog":"c","version":1,"keys":{},"values":{}}`}} {
		code, out, stderr := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("ETag", `"rev"`)
			w.Header().Set("Set-Cookie", "private-cookie")
			io.WriteString(w, `{"id":"c:I:1","values":{"password":"never-expose"}}`)
		}, "", args...)
		if code != 0 || strings.Contains(out, "never-expose") || strings.Contains(out, "private-cookie") {
			t.Fatalf("code=%d out=%s err=%s", code, out, stderr)
		}
	}
}

func TestContractFeaturesWebhookBindingValidation(t *testing.T) {
	dir := t.TempDir()
	definition := filepath.Join(dir, "webhook.json")
	metadata := filepath.Join(dir, "metadata.json")
	os.WriteFile(definition, []byte(`{"name":"Event","type":"GENERIC","commandId":"c:C:1","eventKey":"event","inputReferences":["c:I:1"]}`), 0600)
	os.WriteFile(metadata, []byte(`{"commands":{"c:C:1":{"inputKeys":{"event":{"type":"string","required":true,"sensitive":false},"password":{"type":"string","required":true,"sensitive":true}}}},"inputs":{"c:I:1":{"keys":{"password":{"type":"string","sensitive":true}}}}}`), 0600)
	calls := 0
	code, out, stderr := memoryArgs(t, func(http.ResponseWriter, *http.Request) { calls++ }, "", "webhooks", "validate", "-f", definition, "--metadata", metadata)
	if code != 0 || calls != 0 || !strings.Contains(out, `"valid": true`) {
		t.Fatalf("code=%d out=%s err=%s", code, out, stderr)
	}
}

func TestContractFeaturesWebhookInlineValueTypes(t *testing.T) {
	for _, tc := range []struct {
		kind, value string
		valid       bool
	}{
		{"number", "+00042", true},
		{"number", ".5", true},
		{"number", "1.", true},
		{"number", "1e5000", true},
		{"number", "90071992547409931234567890.123", true},
		{"number", "synthetic-private-value", false},
		{"number", "1x", false},
		{"number", "NaN", false},
		{"number", "Infinity", false},
		{"number", " 42", false},
		{"number", "1e2147483648", false},
		{"boolean", "TRUE", true},
		{"boolean", "False", true},
		{"boolean", "1", false},
		{"boolean", " true", false},
		{"array", `[1e5000,{"flag":true}]`, true},
		{"array", `[]`, true},
		{"array", `{"value":"synthetic-private-value"}`, false},
		{"array", `null`, false},
		{"object", `{"count":1e5000}`, true},
		{"object", `{}`, true},
		{"object", `["synthetic-private-value"]`, false},
		{"object", `null`, false},
		{"string", "$(input.value)", true},
		{"number", "$(input.value)", false},
	} {
		t.Run(tc.kind+tc.value, func(t *testing.T) {
			dir := t.TempDir()
			metadata := map[string]any{"commands": map[string]any{"c:Command:1": map[string]any{"inputKeys": map[string]any{"argument": map[string]any{"type": tc.kind, "required": true, "sensitive": false}}}}, "inputs": map[string]any{}}
			encoded, err := json.Marshal(metadata)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "contracts.json")
			if err = os.WriteFile(path, encoded, 0600); err != nil {
				t.Fatal(err)
			}
			body, err := json.Marshal(map[string]any{"name": "Hook", "type": "GENERIC", "commandId": "c:Command:1", "inputValues": map[string]string{"argument": tc.value}})
			if err != nil {
				t.Fatal(err)
			}
			code, out, stderr := memoryArgs(t, func(http.ResponseWriter, *http.Request) { t.Fatal("offline validation contacted the API") }, string(body), "webhooks", "validate", "-f", "-", "--metadata", path)
			want := exitLint
			if tc.valid {
				want = exitOK
			}
			if code != want || strings.Contains(out+stderr, "synthetic-private-value") {
				t.Fatalf("code=%d want=%d out=%s stderr=%s", code, want, out, stderr)
			}
		})
	}
}

func TestContractFeaturesEmptyWebhookMetadataStopsBeforeTransport(t *testing.T) {
	code, _, stderr := memoryArgs(t, func(http.ResponseWriter, *http.Request) { t.Fatal("explicit offline metadata fell back to the API") }, "", "webhooks", "validate", "-f", "-", "--metadata", "")
	if code != exitUsage || !strings.Contains(stderr, "--metadata needs a contract file") {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
}
