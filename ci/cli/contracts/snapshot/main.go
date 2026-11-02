// snapshot pins local Swagger contracts and their source digests.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type loader struct {
	root      string
	documents map[string]any
	digests   map[string]string
}

var keys = map[string]bool{}

func init() {
	for _, k := range strings.Fields("type format required properties items additionalProperties minimum maximum minLength maxLength minItems maxItems minProperties maxProperties uniqueItems enum pattern nullable readOnly default collectionFormat name in") {
		keys[k] = true
	}
}
func (l *loader) read(path string) (any, error) {
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(l.root, path)
	if err != nil {
		return nil, err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("schema reference escapes Swagger directory: %s", rel)
	}
	if v, ok := l.documents[path]; ok {
		return v, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var v any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err = d.Decode(&v); err != nil {
		return nil, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("schema must contain one JSON document: %s", rel)
	}
	sum := sha256.Sum256(raw)
	l.digests[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
	l.documents[path] = v
	return v, nil
}
func (l *loader) resolve(v any, path string, trail map[string]bool) (any, error) {
	switch node := v.(type) {
	case []any:
		out := make([]any, len(node))
		for i, item := range node {
			var err error
			out[i], err = l.resolve(item, path, trail)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	case map[string]any:
		if ref, ok := node["$ref"].(string); ok {
			filename, fragment, _ := strings.Cut(ref, "#")
			refpath := path
			if filename != "" {
				refpath = filepath.Join(filepath.Dir(path), filename)
			}
			var err error
			refpath, err = filepath.EvalSymlinks(refpath)
			if err != nil {
				return nil, err
			}
			marker := refpath + "#" + fragment
			if trail[marker] {
				return nil, fmt.Errorf("recursive schema: %s", ref)
			}
			target, err := l.read(refpath)
			if err != nil {
				return nil, err
			}
			if fragment != "" {
				if !strings.HasPrefix(fragment, "/") {
					return nil, fmt.Errorf("unsupported schema fragment: %s", fragment)
				}
				for _, key := range strings.Split(strings.TrimPrefix(fragment, "/"), "/") {
					key = strings.ReplaceAll(strings.ReplaceAll(key, "~1", "/"), "~0", "~")
					m, ok := target.(map[string]any)
					if !ok {
						return nil, fmt.Errorf("invalid schema pointer: %s", ref)
					}
					target, ok = m[key]
					if !ok {
						return nil, fmt.Errorf("missing schema pointer: %s", ref)
					}
				}
			}
			next := map[string]bool{}
			for k, v := range trail {
				next[k] = v
			}
			next[marker] = true
			resolved, err := l.resolve(target, refpath, next)
			if err != nil {
				return nil, err
			}
			result, ok := resolved.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("schema reference is not an object: %s", ref)
			}
			siblings := map[string]any{}
			for k, v := range node {
				if k != "$ref" {
					siblings[k] = v
				}
			}
			extra, err := l.resolve(siblings, path, trail)
			if err != nil {
				return nil, err
			}
			for k, v := range extra.(map[string]any) {
				result[k] = v
			}
			return result, nil
		}
		out := map[string]any{}
		for k, value := range node {
			if !keys[k] {
				continue
			}
			if k == "required" {
				if _, ok := value.([]any); !ok {
					continue
				}
			}
			if k == "properties" {
				props, ok := value.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("properties must be an object")
				}
				mapped := map[string]any{}
				for name, prop := range props {
					resolved, err := l.resolve(prop, path, trail)
					if err != nil {
						return nil, err
					}
					mapped[name] = resolved
				}
				out[k] = mapped
			} else {
				resolved, err := l.resolve(value, path, trail)
				if err != nil {
					return nil, err
				}
				out[k] = resolved
			}
		}
		return out, nil
	default:
		return v, nil
	}
}
func object(v any) map[string]any { m, _ := v.(map[string]any); return m }
func generate(folder string) (map[string]any, error) {
	folder, err := filepath.Abs(folder)
	if err != nil {
		return nil, err
	}
	folder, err = filepath.EvalSymlinks(folder)
	if err != nil {
		return nil, err
	}
	l := loader{folder, map[string]any{}, map[string]string{}}
	ops := []map[string]any{}
	for _, filename := range []string{"ContentAPI.json", "ExecutionsAPI.json", "ScheduledExecutionsAPI.json"} {
		path := filepath.Join(folder, filename)
		doc, err := l.read(path)
		if err != nil {
			return nil, err
		}
		for route, verbs := range object(object(doc)["paths"]) {
			for method, value := range object(verbs) {
				if !map[string]bool{"get": true, "put": true, "post": true, "patch": true, "delete": true}[method] {
					continue
				}
				op := object(value)
				if _, ok := op["operationId"].(string); !ok {
					return nil, fmt.Errorf("missing operationId: %s %s", method, route)
				}
				params := []any{}
				values, _ := op["parameters"].([]any)
				for _, v := range values {
					param := object(v)
					normalized, err := l.resolve(param, path, nil)
					if err != nil {
						return nil, err
					}
					n := object(normalized)
					required, exists := param["required"]
					if !exists {
						required = false
					}
					n["required"] = required
					if schema, ok := param["schema"]; ok {
						n["schema"], err = l.resolve(schema, path, nil)
						if err != nil {
							return nil, err
						}
					}
					params = append(params, n)
				}
				responses := map[string]any{}
				for status, response := range object(op["responses"]) {
					headers := []string{}
					for k := range object(object(response)["headers"]) {
						headers = append(headers, k)
					}
					sort.Strings(headers)
					responses[status] = map[string]any{"headers": headers}
				}
				ops = append(ops, map[string]any{"method": strings.ToUpper(method), "path": route, "operationId": op["operationId"], "parameters": params, "responses": responses})
			}
		}
	}
	sort.Slice(ops, func(i, j int) bool {
		a, b := ops[i], ops[j]
		if a["path"] == b["path"] {
			return a["method"].(string) < b["method"].(string)
		}
		return a["path"].(string) < b["path"].(string)
	})
	kinds := map[string]any{}
	for kind, name := range map[string]string{"command": "command", "input": "input", "catalog": "catalog", "catalog-update": "catalogUpdateRequest", "bulk-commands": "createCommandsRequest", "bulk-inputs": "createInputsRequest", "execution-request": "executionRequest", "action": "action", "bulk-action": "bulkActionRequest", "scheduled-execution": "scheduledExecution", "webhook": "webhook", "replacement": "replacement"} {
		path := filepath.Join(folder, "definitions", name+".json")
		v, err := l.read(path)
		if err != nil {
			return nil, err
		}
		kinds[kind], err = l.resolve(v, path, nil)
		if err != nil {
			return nil, err
		}
	}
	revision := "unknown"
	if raw, err := exec.Command("git", "-C", folder, "rev-parse", "HEAD").Output(); err == nil {
		revision = strings.TrimSpace(string(raw))
	}
	return map[string]any{"revision": revision, "files": l.digests, "operations": ops, "kinds": kinds}, nil
}
func main() {
	check := flag.Bool("check", false, "fail if the pinned snapshot differs")
	target := flag.String("output", "cli/internal/contract/snapshot.json", "snapshot path, relative to current directory")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: go run ci/cli/contracts/snapshot/main.go [--check] [--output FILE] SWAGGER_DIR")
		os.Exit(1)
	}
	result, err := generate(flag.Arg(0))
	fatal(err)
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	fatal(enc.Encode(result))
	if *check {
		existing, err := os.ReadFile(*target)
		fatal(err)
		if !bytes.Equal(existing, out.Bytes()) {
			fatal(fmt.Errorf("contract snapshot differs; review and regenerate it"))
		}
	} else {
		fatal(os.MkdirAll(filepath.Dir(*target), 0755))
		fatal(os.WriteFile(*target, out.Bytes(), 0644))
	}
	fmt.Printf("Contract snapshot: %d operations, source digests recorded\n", len(result["operations"].([]map[string]any)))
}
func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
