package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDeployOnlyCreatesAfter404(t *testing.T) {
	for _, status := range []int{200, 404, 401, 403, 429, 500, 503, 307} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var methods []string
			newServer(t, func(w http.ResponseWriter, r *http.Request) {
				methods = append(methods, r.Method)
				if r.Method == "GET" {
					if r.URL.Path != "/api/v1/inputs/c:Cred:1" {
						t.Errorf("derived ID path=%s", r.URL.Path)
					}
					w.WriteHeader(status)
					io.WriteString(w, `{"id":"c:Cred:1"}`)
					return
				}
				io.WriteString(w, `{"id":"c:Cred:1"}`)
			})
			_, _, code := runCmd("inputs", "deploy", "--data", `{"catalog":"c","name":"Cred","version":1,"keys":{},"values":{}}`, "--insecure")
			want := []string{"GET"}
			if status == 200 {
				want = append(want, "PUT")
			} else if status == 404 {
				want = append(want, "POST")
			}
			if !reflect.DeepEqual(methods, want) {
				t.Fatalf("methods=%v want=%v", methods, want)
			}
			if (code == 0) != (status == 200 || status == 404) {
				t.Fatalf("exit=%d", code)
			}
		})
	}
}

func TestDeployDoesNotHideTransportOrCreateConflict(t *testing.T) {
	for _, transport := range []bool{false, true} {
		var methods []string
		newServer(t, func(w http.ResponseWriter, r *http.Request) {
			methods = append(methods, r.Method)
			if r.Method == "GET" {
				if transport {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
						return
					}
					conn.Close()
					return
				}
				w.WriteHeader(404)
				return
			}
			w.WriteHeader(409)
		})
		_, _, code := runCmd("commands", "deploy", "--data", validCommand, "--insecure")
		if transport && (code != exitNetwork || len(methods) != 1) {
			t.Fatalf("transport code=%d methods=%v", code, methods)
		}
		if !transport && (code != exitHTTP4xx || len(methods) != 2) {
			t.Fatalf("conflict code=%d methods=%v", code, methods)
		}
	}
}

func TestTriggerWireAndReferenceBuilder(t *testing.T) {
	var payload map[string]any
	newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		io.WriteString(w, `{"id":"run"}`)
	})
	_, _, code := runCmd("executions", "trigger", "--command-id", "c:C:1", "--input", "a=b", "--input-json", `{"n":"9007199254740993","array":"[1,2]"}`, "--input-reference", "c:I:1", "--logs", "--insecure")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if !reflect.DeepEqual(payload["inputValues"], map[string]any{"a": "b", "n": "9007199254740993", "array": "[1,2]"}) {
		t.Fatalf("wire values=%v", payload)
	}
	if !reflect.DeepEqual(payload["inputReferences"], []any{"c:I:1"}) {
		t.Fatal("reference lost")
	}
}

func TestInvalidArgumentsNeverSend(t *testing.T) {
	sent := 0
	newServer(t, func(w http.ResponseWriter, r *http.Request) { sent++ })
	cases := [][]string{
		{"commands", "delete", "first", "second"}, {"commands", "list", "unexpected"},
		{"commands", "create", "unexpected", "--data", "{}"}, {"commands", "get", "x", "--catalog", "c"},
		{"api", "POST", "inputs", "unexpected", "--data", "{}"},
		{"executions", "trigger", "--data", `{"commandId":"c:C:1"}`, "--command-id", "c:Other:1"},
		{"executions", "trigger", "--data", `{"commandId":"c:C:1"}`, "--logs=false"},
		{"executions", "trigger", "--data", `{"commandId":"c:C:1"}`, "--dry-run"},
		{"executions", "trigger", "--data", `{"commandId":"c:C:1"}`, "--dry-run=false", "--print-request"},
		{"commands", "delete", "x", "--dry-run"},
		{"executions", "trigger", "--command-id", "c:C:1", "--input-json", `{"n":42}`},
		{"executions", "trigger", "--command-id", "c:C:1", "--input-json", `{"n":null}`},
		{"executions", "trigger", "--command-id", "c:C:1", "--input-reference", "c:I:1", "--input-reference", "c:I:1"},
		{"executions", "logs", "x", "--executor", "step", "--page", "0"},
		{"executions", "logs", "x", "--page", "-1"}, {"executions", "list", "--limit", "0"},
		{"executions", "action", "x", "--type", "UNKNOWN"}, {"executions", "list", "--status", "UNKNOWN"},
		{"catalogs", "list", "--own", "--provided"}, {"commands", "get", "x", "--timeout", "0"},
		{"commands", "deploy", "--id", "c:Other:1", "--data", `{"catalog":"c","name":"C","version":1}`},
		{"commands", "update", "x", "--data", "{}", "--if-match", ""},
	}
	for _, args := range cases {
		_, _, code := runCmd(append(args, "--insecure")...)
		if code != exitUsage {
			t.Errorf("%v exit=%d", args, code)
		}
	}
	if sent != 0 {
		t.Fatalf("%d invalid requests sent", sent)
	}
}

func TestDryRunAddsExecutionTag(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		wantTag bool
	}{
		{"flag", []string{"--command-id", "c:C:1", "--dry-run"}, true},
		{"preview disabled", []string{"--command-id", "c:C:1", "--dry-run", "--print-request=false"}, true},
		{"explicit tag", []string{"--command-id", "c:C:1", "--tag", "feature:dryRun="}, true},
		{"prebuilt", []string{"--data", `{"commandId":"c:C:1","tags":{"feature:dryRun":""}}`}, true},
		{"disabled", []string{"--command-id", "c:C:1", "--dry-run=false"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sent := 0
			newServer(t, func(w http.ResponseWriter, r *http.Request) {
				sent++
				if r.Method != "POST" || r.URL.Path != "/api/v1/executions" {
					t.Errorf("request=%s %s", r.Method, r.URL.Path)
				}
				var payload struct {
					CommandID string            `json:"commandId"`
					Tags      map[string]string `json:"tags"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				value, present := payload.Tags["feature:dryRun"]
				if payload.CommandID != "c:C:1" || present != tc.wantTag || value != "" {
					t.Errorf("unexpected execution payload: %+v", payload)
				}
				io.WriteString(w, `{"id":"run"}`)
			})
			args := append([]string{"executions", "trigger"}, tc.args...)
			_, errb, code := runCmd(append(args, "--insecure")...)
			if code != 0 || sent != 1 {
				t.Fatalf("code=%d requests=%d stderr=%s", code, sent, errb)
			}
		})
	}
}

func TestPrintRequestAlwaysOfflineAndEndOptions(t *testing.T) {
	sent := 0
	newServer(t, func(w http.ResponseWriter, r *http.Request) { sent++ })
	for _, args := range [][]string{
		{"executions", "trigger", "--command-id", "c:C:1", "--print-request"},
		{"executions", "trigger", "--data", `{"commandId":"c:C:1"}`, "--print-request"},
		{"commands", "delete", "--print-request", "--", "--odd-id"},
	} {
		out, errb, code := runCmd(args...)
		if code != 0 || out == "" {
			t.Fatalf("%v code=%d stderr=%s", args, code, errb)
		}
	}
	out, errb, code := runCmd("executions", "trigger", "--command-id", "c:C:1", "--dry-run", "--print-request")
	if code != 0 || !strings.Contains(out, `"feature:dryRun": ""`) {
		t.Fatalf("execution tag preview failed: code=%d stdout=%s stderr=%s", code, out, errb)
	}
	if sent != 0 {
		t.Fatal("request preview sent requests")
	}
	_, _, code = runCmd("commands", "delete", "first", "--", "--print-request")
	if code != exitUsage {
		t.Fatal("flags parsed after --")
	}
}

func TestStdinAndLintInterface(t *testing.T) {
	var out, errb bytes.Buffer
	c := &cmdContext{stdin: strings.NewReader(`{"values":{"password":"synthetic-secret"}}`), stdout: &out, stderr: &errb}
	if dispatch(c, []string{"inputs", "patch", "c:I:1", "-f", "-", "--print-request"}) != 0 || strings.Contains(out.String(), "synthetic-secret") {
		t.Fatal("stdin preview failed or leaked")
	}
	path := filepath.Join(t.TempDir(), "Cred.input.json")
	if err := os.WriteFile(path, []byte(`{"name":"Cred","catalog":"c","version":1,"keys":{},"values":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	result, _, code := runCmd("lint", path, "--output", "json")
	if code != 0 || strings.TrimSpace(result) != "[]" {
		t.Fatalf("lint result=%s code=%d", result, code)
	}
	for _, args := range [][]string{{"lint", "--help"}, {"lint", path, "--output", "yaml"}} {
		_, _, code := runCmd(args...)
		want := 0
		if len(args) > 2 {
			want = exitUsage
		}
		if code != want {
			t.Fatalf("%v code=%d", args, code)
		}
	}
}

func TestExportPreservesPriorFileOnBadResponse(t *testing.T) {
	for _, body := range []string{`{"message":"not a command"}`, `{"name":"C","configuration":{}}`} {
		newServer(t, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) })
		dir := t.TempDir()
		path := filepath.Join(dir, "C.command.json")
		os.WriteFile(path, []byte("original"), 0600)
		_, _, code := runCmd("commands", "export", "c:C:1", "-o", path, "--insecure")
		data, _ := os.ReadFile(path)
		if strings.Contains(body, "message") {
			if code == 0 || string(data) != "original" {
				t.Fatal("bad export replaced file")
			}
		} else {
			if code != 0 || string(data) != body {
				t.Fatal("valid export failed")
			}
		}
		files, _ := os.ReadDir(dir)
		if len(files) != 1 {
			t.Fatal("temporary export leaked")
		}
	}
}
