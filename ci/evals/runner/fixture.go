package main

import (
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
)

// A fixture models API state and scripted diagnostics. It never executes a
// generated Command or evaluates an Automation Pilot expression.
type fixtureTenant struct {
	mu          sync.Mutex
	spec        fixtureSpec
	resources   map[string]any
	requests    []requestRecord
	routeCounts map[string]int
	writes      map[string]int
	versions    map[string]int
	token       string
	server      *httptest.Server
}

func startFixture(spec fixtureSpec, token string) *fixtureTenant {
	if spec.Resources == nil {
		spec.Resources = map[string]any{}
	}
	f := &fixtureTenant{spec: spec, resources: jsonValue(spec.Resources).(map[string]any), routeCounts: map[string]int{}, writes: map[string]int{}, versions: map[string]int{}, token: token}
	f.server = httptest.NewTLSServer(http.HandlerFunc(f.serve))
	return f
}
func (f *fixtureTenant) close() { f.server.Close() }
func (f *fixtureTenant) writeCA(path string) error {
	return os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: f.server.Certificate().Raw}), 0600)
}
func (f *fixtureTenant) snapshot() ([]requestRecord, map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]requestRecord{}, f.requests...), jsonValue(f.resources).(map[string]any)
}
func (f *fixtureTenant) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	req := requestRecord{Sequence: len(f.requests) + 1, Method: r.Method, Path: path, Query: r.URL.Query(), Headers: map[string]string{}}
	for _, h := range []string{"If-Match", "Content-Type"} {
		if value := r.Header.Get(h); value != "" {
			req.Headers[h] = value
		}
	}
	if r.Body != nil {
		b, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
		if err != nil {
			http.Error(w, "invalid body", 400)
			return
		}
		if len(b) > 0 && json.Unmarshal(b, &req.Body) != nil {
			req.Status = 400
			f.requests = append(f.requests, req)
			http.Error(w, "invalid JSON", 400)
			return
		}
	}
	respond := func(status int, body any) {
		req.ResponseBody = body
		req.Status = status
		f.requests = append(f.requests, req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}
	if r.Header.Get("Authorization") != "Bearer "+f.token {
		respond(401, map[string]any{"error": "fixture authentication required"})
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		allowed := false
		for _, scope := range f.spec.Writes {
			if scope.Method == r.Method && scope.Path == path && f.writes[r.Method+" "+path] < scope.Max {
				allowed = true
				break
			}
		}
		// MCP discovery is a read-only protocol operation despite its HTTP method.
		if body, ok := req.Body.(map[string]any); ok && strings.HasPrefix(path, "mcp/") {
			method, _ := body["method"].(string)
			if method == "initialize" || method == "tools/list" || method == "notifications/initialized" {
				allowed = true
			}
		}
		if !allowed {
			req.Unauthorized = true
			respond(403, map[string]any{"error": "outside case write scope"})
			return
		}
		f.writes[r.Method+" "+path]++
	}
	key := r.Method + " " + path
	for _, route := range f.spec.Routes {
		if route.Method == r.Method && route.Path == path {
			count := f.routeCounts[key]
			f.routeCounts[key]++
			if count >= len(route.Responses) {
				count = len(route.Responses) - 1
			}
			if count < 0 {
				respond(500, map[string]any{"error": "empty response sequence"})
				return
			}
			res := route.Responses[count]
			for k, v := range res.Headers {
				w.Header().Set(k, v)
			}
			respond(res.Status, res.Body)
			return
		}
	}
	if r.Method == "GET" {
		switch path {
		case "server-info":
			respond(200, map[string]any{"autoPiTenantUrl": f.server.URL})
			return
		case "user-information":
			respond(200, map[string]any{"username": "fixture-user", "permissions": []string{"Read", "Write", "Execute"}, "appPlan": "standard"})
			return
		case "tenant":
			respond(200, map[string]any{"id": f.spec.Tenant})
			return
		case "scheduled-executions/time-zones":
			respond(200, []string{"UTC", "Europe/Sofia", "America/New_York"})
			return
		}
		if value, ok := f.resources[path]; ok {
			w.Header().Set("ETag", fmt.Sprintf("\"v%d\"", f.versions[path]+1))
			respond(200, publicResource(path, value))
			return
		}
		if contains([]string{"commands", "inputs", "catalogs", "mcp-servers", "executions", "scheduled-executions", "commands/ids"}, path) {
			collection := strings.TrimSuffix(path, "/ids")
			items := []any{}
			for _, key := range sortedKeys(f.resources) {
				if !strings.HasPrefix(key, collection+"/") || strings.Count(key, "/") != 1 {
					continue
				}
				item, ok := f.resources[key].(map[string]any)
				if !ok {
					continue
				}
				if catalog := r.URL.Query().Get("catalog"); catalog != "" && item["catalog"] != catalog {
					continue
				}
				matches := true
				tags, _ := item["tags"].(map[string]any)
				for _, filter := range r.URL.Query()["tag"] {
					pair := strings.SplitN(filter, "=", 2)
					v, exists := tags[pair[0]]
					if !exists || (len(pair) == 2 && v != pair[1]) {
						matches = false
					}
				}
				if !matches {
					continue
				}
				if path == "commands/ids" {
					items = append(items, item["id"])
				} else {
					items = append(items, publicResource(key, item))
				}
			}
			respond(200, items)
			return
		}
		respond(404, map[string]any{"error": "fixture resource not found"})
		return
	}
	if strings.HasPrefix(path, "mcp/") && r.Method == "POST" {
		f.mcp(w, req, respond)
		return
	}
	body, ok := req.Body.(map[string]any)
	switch r.Method {
	case "PUT", "PATCH":
		current, exists := f.resources[path]
		if !exists {
			respond(404, nil)
			return
		}
		if strings.HasPrefix(path, "catalogs/") && r.Header.Get("If-Match") != "" {
			respond(400, map[string]any{"error": "catalog If-Match unsupported"})
			return
		}
		etag := fmt.Sprintf("\"v%d\"", f.versions[path]+1)
		if h := r.Header.Get("If-Match"); h != "" && h != etag {
			respond(412, nil)
			return
		}
		if !ok {
			respond(400, nil)
			return
		}
		var next any = body
		if r.Method == "PATCH" {
			next = mergePatch(current, body)
		}
		if obj, yes := next.(map[string]any); yes {
			old, _ := current.(map[string]any)
			for _, field := range []string{"id", "catalog", "version", "owner"} {
				if _, exists := obj[field]; !exists && old[field] != nil {
					obj[field] = old[field]
				}
			}
		}
		f.resources[path] = next
		f.versions[path]++
		respond(200, publicResource(path, next))
	case "POST":
		if !ok {
			respond(400, nil)
			return
		}
		id, _ := body["id"].(string)
		if id == "" {
			name, _ := body["name"].(string)
			catalog, _ := body["catalog"].(string)
			if path == "catalogs" {
				technical, _ := body["technicalName"].(string)
				id = technical + "-" + f.spec.Tenant
			} else if path == "commands" || path == "inputs" {
				id = fmt.Sprintf("%s:%s:1", catalog, name)
			} else {
				id = "fixture-created"
			}
		}
		if _, exists := f.resources[path+"/"+id]; exists {
			respond(409, nil)
			return
		}
		body["id"] = id
		f.resources[path+"/"+id] = body
		respond(201, publicResource(path, body))
	case "DELETE":
		if _, exists := f.resources[path]; !exists {
			respond(404, nil)
			return
		}
		delete(f.resources, path)
		respond(204, nil)
	default:
		respond(405, nil)
	}
}
func publicResource(path string, value any) any {
	if strings.HasPrefix(path, "inputs/") {
		object, _ := jsonValue(value).(map[string]any)
		delete(object, "values")
		return object
	}
	return value
}
func mergePatch(current, patch any) any {
	p, ok := patch.(map[string]any)
	if !ok {
		return jsonValue(patch)
	}
	c, _ := jsonValue(current).(map[string]any)
	if c == nil {
		c = map[string]any{}
	}
	for k, v := range p {
		if v == nil {
			delete(c, k)
		} else {
			c[k] = mergePatch(c[k], v)
		}
	}
	return c
}
func (f *fixtureTenant) mcp(w http.ResponseWriter, req requestRecord, respond func(int, any)) {
	body, _ := req.Body.(map[string]any)
	method, _ := body["method"].(string)
	if method == "notifications/initialized" {
		respond(202, nil)
		return
	}
	result := any(map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]any{"name": "fixture", "version": "1"}})
	if method == "tools/list" {
		id := strings.TrimPrefix(req.Path, "mcp/")
		server, ok := f.resources["mcp-servers/"+id].(map[string]any)
		if !ok {
			respond(404, nil)
			return
		}
		tools := []any{}
		if server["enabled"] == false {
			respond(404, nil)
			return
		}
		entries, _ := server["mcpTools"].([]any)
		for _, entry := range entries {
			tool, _ := entry.(map[string]any)
			if tool["enabled"] == false {
				continue
			}
			commandID, _ := tool["commandId"].(string)
			command, _ := f.resources["commands/"+commandID].(map[string]any)
			keys, _ := command["inputKeys"].(map[string]any)
			props := map[string]any{}
			required := []string{}
			injected := map[string]bool{}
			refs, _ := tool["inputReferences"].([]any)
			for _, ref := range refs {
				input, _ := f.resources["inputs/"+fmt.Sprint(ref)].(map[string]any)
				iks, _ := input["keys"].(map[string]any)
				for key := range iks {
					injected[key] = true
				}
			}
			for key, value := range keys {
				if injected[key] {
					continue
				}
				contract, _ := value.(map[string]any)
				props[key] = map[string]any{"type": contract["type"], "description": contract["description"]}
				if contract["required"] == true {
					required = append(required, key)
				}
			}
			sort.Strings(required)
			annotations := map[string]any{}
			for _, hint := range []string{"readOnlyHint", "destructiveHint", "idempotentHint", "openWorldHint"} {
				if value, ok := tool[hint]; ok {
					annotations[hint] = value
				}
			}
			tools = append(tools, map[string]any{"name": tool["name"], "description": command["description"], "annotations": annotations, "inputSchema": map[string]any{"type": "object", "properties": props, "required": required}})
		}
		tools = append(tools, map[string]any{"name": "get_execution", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"executionId": map[string]any{"type": "string"}}, "required": []string{"executionId"}}})
		result = map[string]any{"tools": tools}
	}
	respond(200, map[string]any{"jsonrpc": "2.0", "id": body["id"], "result": result})
}
