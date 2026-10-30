// The evaluation harness invokes autopi; it contains no Automation Pilot HTTP client.
package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type resource struct{ File, Resource string }
type assertion struct {
	Path     string `json:"path"`
	Eq       any    `json:"eq"`
	Length   *int   `json:"length"`
	Nonempty bool   `json:"nonempty"`
}
type scenario struct {
	ID, Command, Status, MCP, Tool, FailedExecutor, Skipped string
	Values                                                  map[string]string
	References                                              []string
	Arguments                                               map[string]any
	Assertions                                              []assertion
	Children                                                int
	Listener, Rotate, ExpectAsync                           bool
}
type negative struct {
	ID, Source, Edit string
	FindingContains  string
	Patch            map[string]any
	ExpectedExit     int
}
type suite struct {
	Resources              []resource
	Cases                  []scenario
	Negative               []negative
	ContractExpected       map[string]map[string][]string
	MCPLifecycle, Schedule bool
}
type verdict struct {
	Status         string  `json:"status"`
	Detail         string  `json:"detail"`
	ExecutionID    string  `json:"executionId,omitempty"`
	ElapsedSeconds float64 `json:"elapsedSeconds,omitempty"`
}
type mcpRestoration struct {
	Server     string
	Definition map[string]any
}
type state struct {
	Run, Tenant     string
	Catalog         string
	Namespace       string
	Variables       map[string]string
	Resources       map[string]string
	Attempts        map[string]string
	Results         map[string]verdict
	Hashes          map[string]string
	Revision        string
	EvidenceSources []string
	PendingMCP      *mcpRestoration `json:",omitempty"`
}
type harness struct {
	root, cli, dir string
	spec           suite
	state          state
	calls          int
	names          []string
	runCommand     func(*exec.Cmd) error // optional process runner for offline failure tests
	sourceDigest   string
	featureRound   string
	quietReport    bool
}
type callError struct {
	code int
	text string
}

func (e *callError) Error() string { return e.text }
func blocked(err error) bool {
	var e *callError
	return errors.As(err, &e) && (e.code == 2 || e.code == 3 || e.code == 5 ||
		(e.code == 4 && (strings.Contains(e.text, "HTTP 401") || strings.Contains(e.text, "HTTP 403"))))
}
func read(path string, value any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return json.Unmarshal(b, value)
}
func write(path string, value any) error {
	b, e := json.MarshalIndent(value, "", "  ")
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	// Replace atomically: an interrupted process must not lose accepted execution IDs.
	tmp := path + ".tmp"
	if e = os.WriteFile(tmp, append(b, '\n'), 0600); e != nil {
		return e
	}
	return os.Rename(tmp, path)
}
func object(value any) map[string]any { v, _ := value.(map[string]any); return v }
func str(value any) string            { v, _ := value.(string); return v }
func (h *harness) save() error        { return write(filepath.Join(h.dir, "state.json"), h.state) }
func (h *harness) submit(key string) error {
	h.state.Attempts[key] = "submitted"
	if err := h.save(); err != nil {
		delete(h.state.Attempts, key) // No request was sent; later saves must preserve that fact.
		return err
	}
	return nil
}
func (h *harness) replace(s string) string {
	keys := make([]string, 0, len(h.state.Variables))
	for k := range h.state.Variables {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		s = strings.ReplaceAll(s, "<"+k+">", h.state.Variables[k])
	}
	if h.state.Namespace != "" {
		for _, name := range h.names {
			s = strings.ReplaceAll(s, ":"+name+":1", ":"+h.state.Namespace+name+":1")
		}
	}
	return s
}
func (h *harness) definition(file string) (map[string]any, error) {
	b, e := os.ReadFile(filepath.Join(h.root, "definitions", file))
	if e != nil {
		return nil, e
	}
	var m map[string]any
	e = json.Unmarshal(b, &m)
	if e == nil {
		h.materialize(m)
	}
	if e == nil && h.state.Namespace != "" && (strings.HasSuffix(file, ".command.json") || strings.HasSuffix(file, ".input.json")) {
		m["name"] = h.state.Namespace + str(m["name"])
	}
	return m, e
}
func (h *harness) materialize(value any) any {
	switch v := value.(type) {
	case string:
		return h.replace(v)
	case map[string]any:
		for key, item := range v {
			v[key] = h.materialize(item)
		}
	case []any:
		for index, item := range v {
			v[index] = h.materialize(item)
		}
	}
	return value
}
func (h *harness) call(label string, args []string, body any) (any, error) {
	h.calls++
	var input []byte
	var e error
	if body != nil {
		input, e = json.Marshal(body)
		if e != nil {
			return nil, e
		}
	}
	command := exec.Command(h.cli, args...)
	command.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	started := time.Now()
	if h.runCommand != nil {
		e = h.runCommand(command)
	} else {
		e = runCLI(command)
	}
	code := 0
	if e != nil {
		code = 1
		var x *exec.ExitError
		if errors.As(e, &x) {
			code = x.ExitCode()
		}
	}
	var response any
	if stdout.Len() > 0 && json.Unmarshal(stdout.Bytes(), &response) != nil {
		response = stdout.String()
	}
	// Neither argv nor stdin is persisted. Input APIs use autopi's metadata projection.
	record := map[string]any{"operation": label, "exitCode": code, "response": response, "recordedAt": time.Now().UTC().Format(time.RFC3339)}
	record["sourceDigest"] = h.sourceDigest
	record["elapsedMilliseconds"] = time.Since(started).Milliseconds()
	record["stdoutBytes"] = stdout.Len()
	record["stderrBytes"] = stderr.Len()
	record["requestBodyBytes"] = len(input)
	if code != 0 {
		record["error"] = strings.TrimSpace(stderr.String())
	}
	if we := write(filepath.Join(h.dir, "evidence", fmt.Sprintf("%04d-%s.json", h.calls, label)), record); we != nil {
		return nil, we
	}
	if e != nil {
		return response, &callError{code, fmt.Sprintf("%s failed (exit %d): %s", label, code, strings.TrimSpace(stderr.String()))}
	}
	return response, nil
}
func (h *harness) record(id, status, detail string) error {
	previous, existed := h.state.Results[id]
	h.state.Results[id] = verdict{Status: status, Detail: detail}
	if e := h.save(); e != nil {
		if existed {
			h.state.Results[id] = previous
		} else {
			delete(h.state.Results, id)
		}
		return fmt.Errorf("cannot persist verdict %s: %w", id, e)
	}
	fmt.Printf("%s %s: %s\n", status, id, detail)
	return nil
}
func (h *harness) connect() error {
	v, e := h.call("connection", []string{"connection", "info", "--timeout", "15s"}, nil)
	if e != nil {
		return e
	}
	if str(object(v)["tenantId"]) != h.state.Tenant {
		return fmt.Errorf("configured tenant differs from --tenant; no writes performed")
	}
	perms := fmt.Sprint(object(v)["permissions"])
	for _, p := range []string{"Read", "Write", "Execute"} {
		if !strings.Contains(perms, p) {
			return fmt.Errorf("configured account lacks %s", p)
		}
	}
	return nil
}
func (h *harness) metadata() (map[string]any, error) {
	commands, inputs := map[string]any{}, map[string]any{}
	for _, r := range h.spec.Resources {
		d, e := h.definition(r.File)
		if e != nil {
			return nil, e
		}
		id := str(d["catalog"]) + ":" + str(d["name"]) + ":1"
		if r.Resource == "commands" {
			commands[id] = map[string]any{"inputKeys": d["inputKeys"]}
		}
		if r.Resource == "inputs" {
			inputs[id] = map[string]any{"keys": d["keys"]}
		}
	}
	return map[string]any{"commands": commands, "inputs": inputs}, nil
}
func (h *harness) check() error {
	metadata, e := h.metadata()
	if e != nil {
		return e
	}
	metadataPath := filepath.Join(h.dir, "contracts-offline.json")
	if e = write(metadataPath, metadata); e != nil {
		return e
	}
	files := []string{}
	for _, r := range h.spec.Resources {
		d, e := h.definition(r.File)
		if e != nil {
			return e
		}
		b, _ := json.Marshal(d)
		sum := sha256.Sum256(b)
		h.state.Hashes[r.File] = hex.EncodeToString(sum[:])
		suffix := map[string]string{"commands": ".command.json", "inputs": ".input.json", "catalogs": ".catalog.json", "mcp-servers": ".mcp.json"}[r.Resource]
		file := filepath.Join(h.dir, "candidate", str(d["name"])+suffix)
		if e = write(file, d); e != nil {
			return e
		}
		files = append(files, file)
		first := filepath.Join(h.dir, "candidate-first", filepath.Base(file))
		if _, err := os.Stat(first); os.IsNotExist(err) {
			if e = write(first, d); e != nil {
				return e
			}
		}
	}
	_, e = h.call("authored-lint", append([]string{"lint", "--output", "json"}, files...), nil)
	if e != nil {
		if _, recorded := h.state.Results["authored-lint-first"]; !recorded {
			if e := h.record("authored-lint-first", "FAIL", e.Error()); e != nil {
				return e
			}
		}
		if e := h.record("authored-lint", "FAIL", e.Error()); e != nil {
			return e
		}
		return e
	}
	if e := h.record("authored-lint", "PASS", "All materialized authored definitions passed autopi lint"); e != nil {
		return e
	}
	if _, recorded := h.state.Results["authored-lint-first"]; !recorded {
		if e := h.record("authored-lint-first", "PASS", "First materialized candidate passed autopi lint"); e != nil {
			return e
		}
	}
	for _, which := range []string{"public", "control"} {
		d, _ := h.definition(which + ".mcp.json")
		file := filepath.Join(h.dir, "candidate", str(d["name"])+".mcp.json")
		_, e = h.call("bindings-"+which, []string{"mcp-servers", "validate", "-f", file, "--metadata", metadataPath}, nil)
		if e != nil {
			return e
		}
		if e := h.record("bindings-"+which, "PASS", "Offline bindings match command types and sensitivity"); e != nil {
			return e
		}
	}
	for _, n := range h.spec.Negative {
		d, e := h.definition(n.Source)
		if e != nil {
			return e
		}
		for k, v := range n.Patch {
			d[k] = v
		}
		executors, _ := object(d["configuration"])["executors"].([]any)
		switch n.Edit {
		case "retry-enum":
			object(object(executors[1])["autoRetry"])["logic"] = "CONSTANT"
		case "wire-number":
			object(d["values"])["maxItems"] = 5
		case "service-tag":
			object(d["tags"])["autopi:injected"] = "forbidden"
		case "binding-overlap":
			tools := d["mcpTools"].([]any)
			object(tools[0])["inputReferences"] = []string{h.replace("<CORE>:EvalApiConfiguration:1"), h.replace("<CORE>:EvalApiConfiguration:1")}
		case "unbound-secret":
			tools := d["mcpTools"].([]any)
			object(tools[1])["inputReferences"] = []string{}
		}
		file := filepath.Join(h.dir, "negative", str(d["name"])+".json")
		if e = write(file, d); e != nil {
			return e
		}
		args := []string{"lint", file}
		if strings.HasPrefix(n.Edit, "binding") || n.Edit == "unbound-secret" {
			args = []string{"mcp-servers", "validate", "-f", file, "--metadata", metadataPath}
		}
		v, e := h.call(n.ID, args, nil)
		var ce *callError
		if !errors.As(e, &ce) || ce.code != n.ExpectedExit {
			return fmt.Errorf("negative case %s expected exit %d, got %v", n.ID, n.ExpectedExit, e)
		}
		if v == nil {
			return fmt.Errorf("negative case %s returned no finding evidence", n.ID)
		}
		findingJSON, _ := json.Marshal(v)
		if n.FindingContains != "" && !strings.Contains(string(findingJSON), n.FindingContains) {
			return fmt.Errorf("negative case %s failed for a different reason than its intended defect", n.ID)
		}
		if e := h.record(n.ID, "PASS", "autopi rejected the deliberate defect with expected findings"); e != nil {
			return e
		}
	}
	return h.save()
}
func subset(expected, actual any) bool {
	// Locally built payloads can contain []string or map[string]int; API JSON
	// decodes those as []any / map[string]any with float64 numbers.
	b, err := json.Marshal(expected)
	if err != nil {
		return false
	}
	var normalized any
	if json.Unmarshal(b, &normalized) != nil {
		return false
	}
	return subsetJSON(normalized, actual)
}
func subsetJSON(expected, actual any) bool {
	if expected == nil {
		return actual == nil
	}
	if m, ok := expected.(map[string]any); ok {
		a := object(actual)
		if a == nil {
			return false
		}
		for k, v := range m {
			if !subsetJSON(v, a[k]) {
				return false
			}
		}
		return true
	}
	if arr, ok := expected.([]any); ok {
		a, ok := actual.([]any)
		if !ok || len(a) != len(arr) {
			return false
		}
		for i, v := range arr {
			if !subsetJSON(v, a[i]) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(expected, actual)
}
func (h *harness) deploy() error {
	if e := h.check(); e != nil {
		return e
	}
	if e := h.connect(); e != nil {
		return e
	}
	// Resolve all external executor contracts before the first write.
	for _, id := range []string{"utils-sapcp:Void:1", "utils-sapcp:ForEach:2", "utils-sapcp:ParallelForEach:1", "http-sapcp:HttpRequest:1", "scripts-sapcp:ExecutePythonScript:1", "store-sapcp:GetValue:1", "store-sapcp:PutValue:1", "store-sapcp:DeleteValue:1"} {
		if _, e := h.call("contract-"+strings.Split(id, ":")[1], []string{"commands", "contract", id, "--timeout", "20s"}, nil); e != nil {
			return e
		}
	}
	for _, r := range h.spec.Resources {
		d, e := h.definition(r.File)
		if e != nil {
			return e
		}
		if r.Resource == "catalogs" && h.state.Catalog != "" {
			if _, e := h.call("reuse-evaluation-catalog", []string{"catalogs", "get", h.state.Catalog, "--timeout", "20s"}, nil); e != nil {
				return e
			}
			h.state.Resources[r.File] = h.state.Catalog
			if e := h.record("deploy-"+r.File, "REUSED", "Existing dedicated evaluation catalog reused; catalog creation not scored"); e != nil {
				return e
			}
			continue
		}
		identity := h.state.Resources[r.File]
		if identity == "" {
			if h.state.Attempts["create-"+r.File] != "" {
				return fmt.Errorf("earlier create outcome for %s needs reconciliation; refusing a duplicate", r.File)
			}
			payload := d
			if r.File == "BindingFixture.input.json" {
				b := make([]byte, 24)
				if _, e = rand.Read(b); e != nil {
					return e
				}
				object(payload["values"])["apiToken"] = hex.EncodeToString(b)
			}
			if e = h.submit("create-" + r.File); e != nil {
				return e
			}
			v, e := h.call("create-"+strings.Split(r.File, ".")[0], []string{r.Resource, "create", "-f", "-", "--timeout", "30s"}, payload)
			if e != nil {
				return e
			}
			identity = str(object(v)["id"])
			if identity == "" {
				return fmt.Errorf("create %s returned no identity; reconcile before retrying", r.File)
			}
			h.state.Resources[r.File] = identity
			h.state.Attempts["create-"+r.File] = "accepted"
			if e = h.save(); e != nil {
				return e
			}
			if r.Resource == "catalogs" {
				symbol := "CORE"
				if strings.HasPrefix(r.File, "ops") {
					symbol = "OPS"
				}
				if identity != h.state.Variables[symbol] {
					return fmt.Errorf("catalog ID differs from predicted ID; inspect before composing dependencies")
				}
			}
		}
		v, e := h.call("readback-"+strings.Split(r.File, ".")[0], []string{r.Resource, "get", identity, "--timeout", "20s"}, nil)
		if e != nil {
			return e
		}
		if r.Resource == "inputs" {
			delete(d, "values")
		}
		if r.Resource == "catalogs" {
			delete(d, "technicalName")
		}
		if !subset(d, v) {
			return fmt.Errorf("read-back differs from authored fields for %s", r.File)
		}
		if issues, ok := object(v)["issues"].([]any); ok && len(issues) > 0 {
			return fmt.Errorf("command %s has validation issues; inspect evidence", r.File)
		}
		if e := h.record("deploy-"+r.File, "PASS", "Created and verified exact authored fields"); e != nil {
			return e
		}
	}
	return nil
}
func valueAt(value any, path string) any {
	for _, part := range strings.Split(path, ".") {
		if s, ok := value.(string); ok {
			var decoded any
			if json.Unmarshal([]byte(s), &decoded) == nil {
				value = decoded
			}
		}
		if m, ok := value.(map[string]any); ok {
			value = m[part]
		} else if a, ok := value.([]any); ok {
			i, e := strconv.Atoi(part)
			if e != nil || i < 0 || i >= len(a) {
				return nil
			}
			value = a[i]
		} else {
			return nil
		}
	}
	// Only the top-level output value is on the string wire. A nested JSON string
	// such as a status "200" must retain its logical type after the object is parsed.
	if s, ok := value.(string); ok && !strings.Contains(path, ".") {
		var v any
		if json.Unmarshal([]byte(s), &v) == nil {
			return v
		}
	}
	return value
}
func execution(v any) map[string]any {
	m := object(v)
	if result, ok := m["result"]; ok {
		m = object(result)
	}
	if s := object(m["structuredContent"]); s != nil {
		return s
	}
	if arr, ok := m["content"].([]any); ok {
		for _, v := range arr {
			if text := str(object(v)["text"]); text != "" {
				var parsed map[string]any
				if json.Unmarshal([]byte(text), &parsed) == nil {
					return parsed
				}
			}
		}
	}
	return m
}
func (h *harness) rpc(label, server, method string, params any) (any, error) {
	return h.call(label, []string{"api", "POST", "mcp/" + server, "-f", "-", "--timeout", "75s", "--header", "Accept: application/json, text/event-stream"}, map[string]any{"jsonrpc": "2.0", "id": label, "method": method, "params": params})
}
func (h *harness) poll(id string) (map[string]any, error) {
	deadline := time.Now().Add(10 * time.Minute)
	failures := 0
	for time.Now().Before(deadline) {
		v, e := h.call("state-"+id, []string{"executions", "get", id, "--timeout", "20s"}, nil)
		if e != nil {
			failures++
			if failures >= 3 {
				return nil, e
			}
			time.Sleep(2 * time.Second)
			continue
		}
		failures = 0
		m := object(v)
		if str(m["status"]) != "RUNNING" {
			return m, nil
		}
		time.Sleep(5 * time.Second)
	}
	return nil, fmt.Errorf("execution %s exceeded ten-minute monitoring deadline; do not retrigger", id)
}
func (h *harness) schema(which string) error {
	server := h.state.Resources[which+".mcp.json"]
	if server == "" {
		return fmt.Errorf("%s MCP server is not deployed", which)
	}
	if _, e := h.rpc("initialize-"+which, server, "initialize", map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "skill-evaluation", "version": "2"}}); e != nil {
		return e
	}
	v, e := h.rpc("tools-list-"+which, server, "tools/list", map[string]any{})
	if e != nil {
		return e
	}
	file := filepath.Join(h.dir, "client-"+which+".json")
	if e = write(file, v); e != nil {
		return e
	}
	d, _ := h.definition(which + ".mcp.json")
	def := filepath.Join(h.dir, "candidate", str(d["name"])+".mcp.json")
	metadata, _ := h.metadata()
	mf := filepath.Join(h.dir, "contracts-offline.json")
	if e := write(mf, metadata); e != nil {
		return e
	}
	report, ve := h.call("client-contract-"+which, []string{"mcp-servers", "validate", "-f", def, "--metadata", mf, "--client-schema", file}, nil)
	if ve != nil {
		var ce *callError
		if !errors.As(ve, &ce) || ce.code != 6 {
			return ve
		}
		findings, _ := object(report)["findings"].([]any)
		for _, f := range findings {
			if str(object(f)["severity"]) != "MEDIUM" {
				return fmt.Errorf("client schema has blocking findings")
			}
		}
	}
	tools, _ := object(object(v)["result"])["tools"].([]any)
	seen := map[string]bool{}
	for _, t := range tools {
		m := object(t)
		name := str(m["name"])
		seen[name] = true
		if name == "get_execution" {
			continue
		}
		want, ok := h.spec.ContractExpected[which][name]
		if !ok {
			return fmt.Errorf("unexpected client tool %s", name)
		}
		props := object(object(m["inputSchema"])["properties"])
		keys := []string{}
		for k := range props {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		sort.Strings(want)
		if len(keys) != len(want) {
			return fmt.Errorf("tool %s exposed unexpected caller keys", name)
		}
		for i := range keys {
			if keys[i] != want[i] {
				return fmt.Errorf("tool %s contract differs", name)
			}
		}
	}
	for name := range h.spec.ContractExpected[which] {
		if !seen[name] {
			return fmt.Errorf("missing client tool %s", name)
		}
	}
	if !seen["get_execution"] {
		return fmt.Errorf("missing generated execution polling tool")
	}
	if e := h.record("schema-"+which, "PASS", "Live tools/list exposes precisely the expected business keys; MEDIUM collection advisories are guarded by command validation"); e != nil {
		return e
	}
	return nil
}
func (h *harness) listener(parent string, status string) error {
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		v, e := h.call("listener-search", []string{"executions", "list", "--tag", "autopi:statusListenerOf=" + parent, "--limit", "10", "--full", "--timeout", "20s"}, nil)
		if e != nil {
			return e
		}
		arr, _ := v.([]any)
		for _, row := range arr {
			m := object(row)
			if str(m["commandId"]) != h.replace("<OPS>:AuditListener:1") {
				continue
			}
			s, e := h.poll(str(m["id"]))
			if e != nil {
				return e
			}
			if str(s["status"]) != "FINISHED" {
				return fmt.Errorf("status listener failed for %s", parent)
			}
			v, e := h.call("event-read", []string{"executions", "trigger", "--command-id", "store-sapcp:GetValue:1", "--input", "key=skills-eval/" + h.state.Run + "/event/" + parent, "--timeout", "20s"}, nil)
			if e != nil {
				return e
			}
			event, e := h.poll(str(object(v)["id"]))
			if e != nil {
				return e
			}
			if str(event["status"]) != "FINISHED" {
				return fmt.Errorf("event marker unavailable")
			}
			o, e := h.call("event-output", []string{"executions", "output", str(event["id"]), "--timeout", "20s"}, nil)
			if e != nil {
				return e
			}
			if valueAt(object(o)["values"], "value.outcome") != status {
				return fmt.Errorf("event marker has wrong terminal outcome")
			}
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("status listener was not observed for %s", parent)
}
func (h *harness) runCase(c scenario) error {
	if result, done := h.state.Results[c.ID]; done {
		// A blocked poll can resume the captured execution, never resubmit it.
		if result.Status != "BLOCKED" || h.state.Attempts[c.ID] == "" || h.state.Attempts[c.ID] == "submitted" {
			return nil
		}
	}
	start := time.Now()
	identity := h.state.Attempts[c.ID]
	if identity == "submitted" {
		return fmt.Errorf("case %s has an ambiguous submission; refusing a duplicate", c.ID)
	}
	if identity == "" {
		if c.Rotate {
			payload := map[string]any{"values": map[string]any{}}
			b := make([]byte, 24)
			if _, e := rand.Read(b); e != nil {
				return e
			}
			object(payload["values"])["apiToken"] = hex.EncodeToString(b)
			if _, e := h.call("rotate-fake-binding", []string{"inputs", "patch", h.state.Resources["BindingFixture.input.json"], "-f", "-", "--timeout", "20s"}, payload); e != nil {
				return e
			}
		}
		if e := h.submit(c.ID); e != nil {
			return e
		}
		var v any
		var e error
		if c.MCP != "" {
			v, e = h.rpc(c.ID+"-call", h.state.Resources[c.MCP+".mcp.json"], "tools/call", map[string]any{"name": c.Tool, "arguments": c.Arguments})
		} else {
			refs := []string{}
			for _, r := range c.References {
				refs = append(refs, h.replace(r))
			}
			v, e = h.call(c.ID+"-trigger", []string{"executions", "trigger", "-f", "-", "--timeout", "30s"}, map[string]any{"commandId": h.replace(c.Command), "inputValues": c.Values, "inputReferences": refs, "tags": map[string]string{"evaluation": h.state.Run, "case": c.ID, "feature:logs": "", "feature:priority": "high"}})
		}
		if e != nil {
			return e
		}
		m := execution(v)
		identity = str(m["id"])
		if identity == "" {
			identity = str(m["executionId"])
		}
		if identity == "" {
			return fmt.Errorf("case %s returned no execution ID; inspect before retrying", c.ID)
		}
		h.state.Attempts[c.ID] = identity
		if e = h.save(); e != nil {
			return e
		}
		if c.ExpectAsync && str(m["status"]) == "FINISHED" {
			return fmt.Errorf("sync fallback expected an active execution")
		}
	}
	s, e := h.poll(identity)
	if e != nil {
		return e
	}
	if str(s["status"]) != c.Status {
		return fmt.Errorf("expected %s, observed %s at %s: %s", c.Status, str(s["status"]), str(s["currentExecutorPath"]), str(s["error"]))
	}
	if c.FailedExecutor != "" && !strings.HasPrefix(str(s["currentExecutorPath"]), c.FailedExecutor) {
		return fmt.Errorf("failure occurred outside expected executor %s", c.FailedExecutor)
	}
	if c.MCP == "" {
		tags := object(s["tags"])
		if str(tags["evaluation"]) != h.state.Run || str(tags["case"]) != c.ID || str(tags["feature:priority"]) != "high" {
			return fmt.Errorf("direct execution lost its scope or feature override tags")
		}
		if _, present := tags["feature:logs"]; !present {
			return fmt.Errorf("direct execution lost requested logging")
		}
	}
	if c.Skipped != "" {
		found := false
		arr, _ := s["executors"].([]any)
		for _, x := range arr {
			m := object(x)
			if str(m["alias"]) == c.Skipped && str(m["status"]) == "SKIPPED_CONDITION" {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("expected skipped executor %s", c.Skipped)
		}
	}
	if c.Status == "FINISHED" {
		v, e := h.call(c.ID+"-output", []string{"executions", "output", identity, "--timeout", "20s"}, nil)
		if e != nil {
			return e
		}
		values := object(v)["values"]
		for _, a := range c.Assertions {
			actual := valueAt(values, a.Path)
			if a.Length != nil {
				arr, ok := actual.([]any)
				if !ok || len(arr) != *a.Length {
					return fmt.Errorf("%s length assertion failed", a.Path)
				}
			} else if a.Nonempty {
				if str(actual) == "" {
					return fmt.Errorf("%s must be nonempty", a.Path)
				}
			} else if !reflect.DeepEqual(actual, a.Eq) {
				return fmt.Errorf("%s: expected %v, got %v", a.Path, a.Eq, actual)
			}
		}
		if c.Children > 0 {
			arr, _ := valueAt(values, "summary.executions").([]any)
			for _, x := range arr {
				m := object(x)
				if str(m["status"]) != "FINISHED" {
					return fmt.Errorf("parallel child did not finish")
				}
				childID := str(m["execution"])
				child, e := h.call("child-lineage-"+childID, []string{"executions", "get", childID, "--timeout", "20s"}, nil)
				if e != nil {
					return e
				}
				childTags := object(object(child)["tags"])
				if str(childTags["autopi:childExecutionOf"]) != identity || str(childTags["autopi:parentExecutorPath"]) != "enrichPosts" {
					return fmt.Errorf("parallel child %s has incorrect parent lineage", childID)
				}
				if str(childTags["role"]) != "post-child" {
					return fmt.Errorf("parallel child %s lost its explicit role tag", childID)
				}
			}
		}
	}
	if c.MCP != "" {
		tags := object(s["tags"])
		if str(tags["autopi:mcpServerId"]) != h.state.Resources[c.MCP+".mcp.json"] || str(tags["autopi:mcpToolName"]) != c.Tool {
			return fmt.Errorf("MCP execution lineage differs")
		}
		polled, e := h.rpc(c.ID+"-mcp-poll", h.state.Resources[c.MCP+".mcp.json"], "tools/call", map[string]any{"name": "get_execution", "arguments": map[string]any{"executionId": identity}})
		if e != nil {
			return e
		}
		if str(execution(polled)["status"]) != c.Status {
			return fmt.Errorf("generated MCP polling tool reported a different terminal status")
		}
	}
	if c.Listener {
		if e := h.listener(identity, c.Status); e != nil {
			return e
		}
		if _, e := h.call(c.ID+"-logs", []string{"executions", "logs", identity, "--executor", map[bool]string{true: "retryProbe", false: "pollProbe"}[strings.Contains(c.ID, "retry")], "--timeout", "20s"}, nil); e != nil {
			return e
		}
	}
	h.state.Results[c.ID] = verdict{Status: "PASS", Detail: "Expected terminal state, semantic assertions, and requested lineage matched", ExecutionID: identity, ElapsedSeconds: time.Since(start).Seconds()}
	if e := h.save(); e != nil {
		delete(h.state.Results, c.ID)
		return e
	}
	fmt.Printf("PASS %s (%s)\n", c.ID, identity)
	return nil
}
func (h *harness) prepareMCPRestoration(server string, definition map[string]any) error {
	if h.state.PendingMCP != nil {
		return fmt.Errorf("pending MCP restoration must finish before another lifecycle mutation")
	}
	h.state.PendingMCP = &mcpRestoration{Server: server, Definition: definition}
	if err := h.save(); err != nil {
		h.state.PendingMCP = nil
		return fmt.Errorf("cannot persist MCP restoration before mutation: %w", err)
	}
	return nil
}
func (h *harness) restoreMCP(label string) error {
	pending := h.state.PendingMCP
	if pending == nil {
		return nil
	}
	if pending.Server == "" || pending.Server != h.state.Resources["public.mcp.json"] || len(pending.Definition) == 0 {
		return fmt.Errorf("pending MCP restoration does not identify this run's public server; reconcile state before resuming")
	}
	if _, err := h.call(label, []string{"mcp-servers", "update", pending.Server, "-f", "-", "--timeout", "20s"}, pending.Definition); err != nil {
		return fmt.Errorf("MCP lifecycle restoration failed; inspect server %s before resuming: %w", pending.Server, err)
	}
	h.state.PendingMCP = nil
	if err := h.save(); err != nil {
		h.state.PendingMCP = pending // Retain recovery intent until its completion is durable.
		return fmt.Errorf("MCP server %s restored but recovery state could not be cleared: %w", pending.Server, err)
	}
	return nil
}
func (h *harness) lifecycle() (err error) {
	server := h.state.Resources["public.mcp.json"]
	original, err := h.definition("public.mcp.json")
	if err != nil {
		return err
	}
	snapshot, err := json.Marshal(original)
	if err != nil {
		return err
	}
	var d map[string]any
	if err = json.Unmarshal(snapshot, &d); err != nil {
		return err
	}
	update := func(label string, definition map[string]any) error {
		_, e := h.call(label, []string{"mcp-servers", "update", server, "-f", "-", "--timeout", "20s"}, definition)
		return e
	}
	restorePending := false
	defer func() {
		if restorePending {
			if restoreErr := h.restoreMCP("restore-lifecycle-after-failure"); restoreErr != nil {
				err = errors.Join(err, restoreErr)
			}
		}
	}()
	tools, _ := d["mcpTools"].([]any)
	found := false
	for _, tool := range tools {
		if str(object(tool)["name"]) == "get_post_with_fallback" {
			object(tool)["enabled"] = false
			found = true
		}
	}
	if !found {
		return fmt.Errorf("lifecycle tool is absent from the candidate definition")
	}
	if err = h.prepareMCPRestoration(server, original); err != nil {
		return err
	}
	restorePending = true
	if err = update("disable-tool", d); err != nil {
		return err
	}
	v, err := h.rpc("disabled-tool-list", server, "tools/list", map[string]any{})
	if err != nil {
		return err
	}
	arr, _ := object(object(v)["result"])["tools"].([]any)
	if arr == nil {
		return fmt.Errorf("disabled tools/list returned no tool list")
	}
	for _, x := range arr {
		if str(object(x)["name"]) == "get_post_with_fallback" {
			return fmt.Errorf("disabled tool remains visible")
		}
	}
	restorePending = false
	if err = h.restoreMCP("restore-tools"); err != nil {
		return err
	}
	if err = h.schema("public"); err != nil {
		return err
	}
	if err = h.record("mcp-tool-lifecycle", "PASS", "Disabled tool disappeared; complete original tool array restored"); err != nil {
		return err
	}
	if err = json.Unmarshal(snapshot, &d); err != nil {
		return err
	}
	d["enabled"] = false
	if err = h.prepareMCPRestoration(server, original); err != nil {
		return err
	}
	restorePending = true
	if err = update("disable-server", d); err != nil {
		return err
	}
	v, err = h.call("disabled-server-readback", []string{"mcp-servers", "get", server, "--timeout", "20s"}, nil)
	if err != nil {
		return err
	}
	if object(v)["enabled"] != false {
		return fmt.Errorf("server did not disable")
	}
	restorePending = false
	if err = h.restoreMCP("restore-server"); err != nil {
		return err
	}
	if err = h.schema("public"); err != nil {
		return err
	}
	return h.record("mcp-server-lifecycle", "PASS", "Server disabled and restored with successful client readiness")
}
func (h *harness) schedule() (err error) {
	if _, done := h.state.Results["schedule-once"]; done {
		return nil
	}
	identity := h.state.Resources["schedule"]
	defer func() {
		if err != nil && identity != "" {
			if _, cleanupErr := h.call("disable-failed-schedule", []string{"scheduled-executions", "disable", identity, "--timeout", "20s"}, nil); cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("cannot disable schedule %s; inspect before resuming: %w", identity, cleanupErr))
			}
		}
	}()
	zone, e := time.LoadLocation("Europe/Sofia")
	if e != nil {
		return e
	}
	zones, e := h.call("schedule-timezones", []string{"scheduled-executions", "time-zones", "--timeout", "20s"}, nil)
	if e != nil {
		return e
	}
	zb, _ := json.Marshal(zones)
	if !bytes.Contains(zb, []byte("Europe/Sofia")) {
		return fmt.Errorf("Europe/Sofia not supported by service")
	}
	if identity == "" {
		if h.state.Attempts["schedule"] != "" {
			return fmt.Errorf("ambiguous schedule create must be reconciled")
		}
		next := time.Now().Add(90 * time.Second).Truncate(5 * time.Minute).Add(5 * time.Minute).In(zone)
		body := map[string]any{"commandId": h.replace("<CORE>:FetchPost:1"), "description": "Run the isolated public content snapshot once to evaluate Input bindings and delivery.", "enabled": true, "inputReferences": []string{h.replace("<CORE>:EvalApiConfiguration:1"), h.replace("<CORE>:EvalScheduledArguments:1")}, "inputValues": map[string]string{}, "tags": map[string]string{"evaluation": h.state.Run, "case": "schedule-once", "feature:logs": ""}, "schedule": map[string]any{"timeZone": "Europe/Sofia", "once": map[string]int{"month": int(next.Month()), "day": next.Day(), "hour": next.Hour(), "minute": next.Minute()}}}
		if e := write(filepath.Join(h.dir, "schedule-prediction.json"), map[string]any{"local": next.Format(time.RFC3339), "utc": next.UTC().Format(time.RFC3339), "definition": body}); e != nil {
			return e
		}
		if e := h.submit("schedule"); e != nil {
			return e
		}
		v, e := h.call("create-schedule", []string{"scheduled-executions", "create", "-f", "-", "--timeout", "20s"}, body)
		if e != nil {
			return e
		}
		identity = str(object(v)["id"])
		if identity == "" {
			return fmt.Errorf("schedule returned no ID")
		}
		h.state.Resources["schedule"] = identity
		h.state.Attempts["schedule"] = "accepted"
		if e := h.save(); e != nil {
			return fmt.Errorf("schedule %s was accepted but its ID could not be persisted; reconcile before resuming: %w", identity, e)
		}
	}
	var prediction map[string]any
	if e := read(filepath.Join(h.dir, "schedule-prediction.json"), &prediction); e != nil {
		return e
	}
	readback, e := h.call("schedule-readback", []string{"scheduled-executions", "get", identity, "--timeout", "20s"}, nil)
	if e != nil {
		return e
	}
	if !subset(prediction["definition"], readback) {
		return fmt.Errorf("schedule read-back differs")
	}
	deadline := time.Now().Add(9 * time.Minute)
	for time.Now().Before(deadline) {
		v, e := h.call("schedule-executions", []string{"executions", "list", "--tag", "autopi:scheduledExecution=" + identity, "--limit", "10", "--full", "--timeout", "20s"}, nil)
		if e != nil {
			return e
		}
		arr, _ := v.([]any)
		if len(arr) > 0 {
			if len(arr) != 1 {
				return fmt.Errorf("once schedule created more than one execution")
			}
			m := object(arr[0])
			id := str(m["id"])
			s, e := h.poll(id)
			if e != nil {
				return e
			}
			if str(s["status"]) != "FINISHED" {
				return fmt.Errorf("scheduled execution did not finish")
			}
			o, e := h.call("schedule-output", []string{"executions", "output", id, "--timeout", "20s"}, nil)
			if e != nil {
				return e
			}
			if valueAt(object(o)["values"], "post.id") != float64(1) {
				return fmt.Errorf("scheduled Input value differs")
			}
			v, e = h.call("schedule-final", []string{"scheduled-executions", "get", identity, "--timeout", "20s"}, nil)
			if e != nil {
				return e
			}
			if str(object(object(v)["state"])["status"]) != "COMPLETED" {
				// Schedule state can lag the FINISHED execution and briefly be null.
				// Keep the captured execution; wait within the original delivery bound.
				time.Sleep(5 * time.Second)
				continue
			}
			if e := h.record("schedule-once", "PASS", "Scheduled execution finished with correct Input and schedule lineage"); e != nil {
				return e
			}
			return nil
		}
		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("schedule delivery timed out")
}
func (h *harness) run() error {
	if e := h.connect(); e != nil {
		return e
	}
	if e := h.restoreMCP("restore-interrupted-lifecycle"); e != nil {
		return e
	}
	for _, r := range h.spec.Resources {
		if h.state.Resources[r.File] == "" {
			return fmt.Errorf("suite is not deployed: %s", r.File)
		}
	}
	for _, which := range []string{"public", "control"} {
		if e := h.schema(which); e != nil {
			return e
		}
	}
	for _, c := range h.spec.Cases {
		if e := h.runCase(c); e != nil {
			status := "FAIL"
			if blocked(e) {
				status = "BLOCKED"
			}
			if e := h.record(c.ID, status, e.Error()); e != nil {
				return e
			}
			if blocked(e) {
				return e
			}
		}
	}
	if h.spec.MCPLifecycle {
		_, toolsDone := h.state.Results["mcp-tool-lifecycle"]
		_, serverDone := h.state.Results["mcp-server-lifecycle"]
		if !toolsDone || !serverDone {
			if e := h.lifecycle(); e != nil {
				return e
			}
		}
	}
	if h.spec.Schedule {
		if e := h.schedule(); e != nil {
			return e
		}
	}
	return nil
}
func (h *harness) report() error {
	pass, fail, block := 0, 0, 0
	keys := []string{}
	for k := range h.state.Results {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	fmt.Fprintf(&b, "# Skill evaluation: %s\n\nGit revision: `%s`. Intended tenant: `%s`.\n\n", h.state.Run, h.state.Revision, h.state.Tenant)
	b.WriteString("| Check | Verdict | Evidence |\n|---|---|---|\n")
	for _, id := range keys {
		v := h.state.Results[id]
		if id == "authored-lint-first" {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", id, v.Status, v.Detail)
			continue // First-attempt observation is not another acceptance case.
		}
		switch v.Status {
		case "PASS":
			pass++
		case "FAIL":
			fail++
		case "BLOCKED":
			block++
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", id, v.Status, strings.NewReplacer("|", "/", "\n", " ", "\r", " ").Replace(v.Detail))
	}
	pending := 0
	for _, c := range h.spec.Cases {
		if _, ok := h.state.Results[c.ID]; !ok {
			pending++
		}
	}
	fmt.Fprintf(&b, "\nRecorded: %d passed, %d failed, %d blocked. Runtime scenarios without a recorded result: %d of %d.\n\n", pass, fail, block, pending, len(h.spec.Cases))
	additional := 0
	for _, id := range []string{"schema-public", "schema-control", "mcp-tool-lifecycle", "mcp-server-lifecycle", "schedule-once"} {
		if _, recorded := h.state.Results[id]; !recorded {
			additional++
		}
	}
	fmt.Fprintf(&b, "Additional live schema/lifecycle/schedule checks without results: %d of 5. Deployment read-backs expected: %d.\n\n", additional, len(h.spec.Resources))
	b.WriteString("Local lint/binding/negative results do not establish deployed command correctness. Live cases require captured execution IDs and semantic output assertions. Pending cases receive no pass. This is one candidate run, without a no-skill baseline or independent model replicates.\n")
	fmt.Fprintf(&b, "\nSource snapshots: %s. Per-call evidence identifies the applicable snapshot.\n", strings.Join(h.state.EvidenceSources, ", "))
	if len(h.state.EvidenceSources) > 1 {
		b.WriteString("This run spans changed or legacy-unrecorded tooling/skills. Earlier passes do not validate the current snapshot; recovery preserves already accepted execution IDs.\n")
	}
	if e := os.WriteFile(filepath.Join(h.dir, "REPORT.md"), []byte(b.String()), 0600); e != nil {
		return e
	}
	if h.quietReport {
		fmt.Printf("Recorded results: %d passed, %d failed, %d blocked. Full evidence: %s\n", pass, fail, block, filepath.Join(h.dir, "REPORT.md"))
	} else {
		fmt.Print(b.String())
	}
	return nil
}
func evaluationTenant(mode, requested, saved string) (string, error) {
	if mode == "deploy" || mode == "run" || mode == "features" || mode == "cleanup" || mode == "measure" {
		if strings.TrimSpace(requested) == "" {
			return "", fmt.Errorf("--tenant is required for %s; specify the intended tenant explicitly", mode)
		}
	}
	if requested != "" {
		return requested, nil
	}
	if saved != "" {
		return saved, nil
	}
	return "TENANT", nil // Synthetic identity for offline checks and empty reports.
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run . check|deploy|run|features|measure|cleanup|report --tenant TENANT --run RUN")
		os.Exit(1)
	}
	mode := os.Args[1]
	flags := flag.NewFlagSet(mode, flag.ExitOnError)
	tenant := flags.String("tenant", "", "intended tenant; required for deploy/run and verified before API writes")
	run := flags.String("run", "local", "new run identifier, 1-24 lowercase alphanumeric characters")
	catalog := flags.String("catalog", "", "existing dedicated evaluation catalog, for tenants at their catalog quota")
	namespace := flags.String("namespace", "", "PascalCase prefix for disjoint candidate Commands and Inputs in a reused catalog")
	only := flags.String("only", "", "optional feature group: content, expressions, actions, schedules, events, scripts, orchestration")
	round := flags.String("round", "", "PascalCase feature repair round; preserves original verdicts and resources")
	sessionLog := flags.String("session-log", "", "optional Codex JSONL log, read only for usage counters")
	since := flags.String("since", "", "RFC3339 start time for model usage measurement")
	_ = flags.Parse(os.Args[2:])
	if !regexp.MustCompile(`^[a-z][a-z0-9]{0,23}$`).MatchString(*run) {
		fmt.Fprintln(os.Stderr, "invalid run identifier")
		os.Exit(1)
	}
	root, e := os.Getwd()
	if e != nil {
		panic(e)
	}
	cli := filepath.Join(root, "..", "..", "..", "skills", "automation-pilot-cli", "scripts", "autopi")
	if os.PathSeparator == '\\' {
		cli += ".cmd"
	}
	h := harness{root: root, cli: cli, dir: filepath.Join(root, "runs", *run)}
	h.quietReport = mode != "report"
	if *round != "" && !regexp.MustCompile("^[A-Z][A-Za-z0-9]{0,7}$").MatchString(*round) {
		fmt.Fprintln(os.Stderr, "invalid feature round")
		os.Exit(1)
	}
	h.featureRound = *round
	if e = read(filepath.Join(root, "suite.json"), &h.spec); e != nil {
		panic(e)
	}
	statePath := filepath.Join(h.dir, "state.json")
	e = read(statePath, &h.state)
	if e != nil && !os.IsNotExist(e) {
		panic(e)
	}
	effectiveTenant, tenantErr := evaluationTenant(mode, *tenant, h.state.Tenant)
	if tenantErr != nil {
		fmt.Fprintln(os.Stderr, tenantErr)
		os.Exit(1)
	}
	*tenant = effectiveTenant
	if os.IsNotExist(e) {
		sum := sha256.Sum256([]byte(*run))
		suffix := hex.EncodeToString(sum[:])[:4]
		h.state = state{Run: *run, Tenant: *tenant, Variables: map[string]string{"RUN": *run, "CORE_PREFIX": "evcore" + suffix, "OPS_PREFIX": "evops" + suffix, "CORE": "evcore" + suffix + "-" + *tenant, "OPS": "evops" + suffix + "-" + *tenant}, Resources: map[string]string{}, Attempts: map[string]string{}, Results: map[string]verdict{}, Hashes: map[string]string{}}
		command := exec.Command("git", "rev-parse", "HEAD")
		if b, e := command.Output(); e == nil {
			h.state.Revision = strings.TrimSpace(string(b))
		}
	}
	if h.state.Tenant != *tenant {
		fmt.Fprintln(os.Stderr, "run was initialized for a different intended tenant")
		os.Exit(1)
	}
	if *catalog != "" {
		if h.state.Catalog != "" && h.state.Catalog != *catalog {
			fmt.Fprintln(os.Stderr, "run already targets another catalog")
			os.Exit(1)
		}
		if len(h.state.Resources) != 0 && h.state.Catalog == "" {
			fmt.Fprintln(os.Stderr, "cannot change catalog after resource creation")
			os.Exit(1)
		}
		h.state.Catalog = *catalog
		h.state.Variables["CORE"] = *catalog
		h.state.Variables["OPS"] = *catalog
	}
	if *namespace != "" {
		if !regexp.MustCompile(`^[A-Z][A-Za-z0-9]{0,23}$`).MatchString(*namespace) {
			fmt.Fprintln(os.Stderr, "invalid resource namespace")
			os.Exit(1)
		}
		if len(h.state.Resources) != 0 && h.state.Namespace != *namespace {
			fmt.Fprintln(os.Stderr, "cannot change namespace after creation")
			os.Exit(1)
		}
		h.state.Namespace = *namespace
	}
	for _, r := range h.spec.Resources {
		if r.Resource != "commands" && r.Resource != "inputs" {
			continue
		}
		var d map[string]any
		if err := read(filepath.Join(root, "definitions", r.File), &d); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		h.names = append(h.names, str(d["name"]))
	}
	// Preserve evidence filenames across process restarts.
	entries, _ := os.ReadDir(filepath.Join(h.dir, "evidence"))
	for _, f := range entries {
		parts := strings.SplitN(f.Name(), "-", 2)
		n, _ := strconv.Atoi(parts[0])
		if n > h.calls {
			h.calls = n
		}
	}
	if mode != "report" {
		if e = h.captureSources(); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
	}
	if e = h.save(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	switch mode {
	case "check":
		e = h.check()
	case "deploy":
		e = h.deploy()
	case "run":
		e = h.run()
	case "features":
		e = h.features(*only)
	case "measure":
		e = h.measure()
		if e == nil && *sessionLog != "" {
			e = h.measureModelUsage(*sessionLog, *since)
		}
	case "cleanup":
		e = h.cleanup()
	case "report":
		e = h.report()
	default:
		e = fmt.Errorf("unknown mode %s", mode)
	}
	if e != nil {
		status := "FAIL"
		if blocked(e) {
			status = "BLOCKED"
		}
		fmt.Fprintln(os.Stderr, e)
		if recordErr := h.record(mode+"-setup", status, e.Error()); recordErr != nil {
			fmt.Fprintln(os.Stderr, recordErr)
		}
		if reportErr := h.report(); reportErr != nil {
			fmt.Fprintln(os.Stderr, reportErr)
		}
		os.Exit(1)
	}
	if mode != "report" {
		if e = h.report(); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
	}
}
