// Package contract validates requests against the pinned, reviewed Swagger snapshot.
package contract

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

//go:embed snapshot.json
var snapshot []byte

type Schema map[string]any
type Parameter struct {
	Name      string
	In        string
	Required  bool
	Schema    Schema
	Type      string
	Minimum   *float64
	Maximum   *float64
	MinItems  *int
	MaxItems  *int
	MaxLength *int
	Enum      []any
	Items     Schema
}
type Operation struct {
	Method      string
	Path        string
	OperationID string
	Parameters  []Parameter
	Responses   map[string]struct{ Headers []string }
}
type Snapshot struct {
	Revision   string
	Files      map[string]string
	Operations []Operation
	Kinds      map[string]Schema
}

var Data Snapshot

func init() {
	if err := json.Unmarshal(snapshot, &Data); err != nil {
		panic(err)
	}
	// Versioned service/schema exceptions are documented in references/api-contract.md.
	for _, s := range Data.Kinds {
		normalize(s)
	}
	for i := range Data.Operations {
		for j := range Data.Operations[i].Parameters {
			p := &Data.Operations[i].Parameters[j]
			normalize(p.Schema)
			// The snapshot's UUID-sized limit excludes HTTP entity-tag quotes.
			// Live ETags are opaque header values (quoted UUIDs are 38 bytes).
			// Preserve the complete header and let the service match its version.
			if p.In == "header" && strings.EqualFold(p.Name, "If-Match") {
				p.MaxLength = nil
			}
		}
	}
}

func normalize(s Schema) {
	if s == nil {
		return
	}
	props, _ := s["properties"].(map[string]any)
	// executionStatusListener has a stale undefined required property. The service
	// converter requires a status and command reference instead.
	if props["status"] != nil && props["execute"] != nil && props["when"] != nil && props["properties"] == nil {
		s["required"] = []any{"status", "execute"}
		if status, ok := props["status"].(map[string]any); ok {
			status["enum"] = []any{"PAUSED", "INPUT_REQUIRED", "FINISHED", "ABORTED", "SUSPENDED", "FAILED"}
		}
	}
	if _, ok := props["schedule"]; ok {
		if required, ok := s["required"].([]any); ok {
			filtered := []any{}
			for _, name := range required {
				if name != "entityVersion" {
					filtered = append(filtered, name)
				}
			}
			s["required"] = filtered
		}
	}
	if _, ok := props["eventKey"]; ok {
		if name, ok := props["name"].(map[string]any); ok {
			name["readOnly"] = false
			name["pattern"] = "^[a-zA-Z0-9]+$"
			name["minLength"] = float64(1)
		}
	}
	if s["type"] == "object" {
		if values, ok := s["enum"].([]any); ok && len(values) > 0 {
			if _, ok := values[0].(string); ok {
				s["type"] = "string"
			}
		}
	}
	for _, v := range props {
		if m, ok := v.(map[string]any); ok {
			normalize(m)
		}
	}
	for _, key := range []string{"items", "additionalProperties"} {
		if m, ok := s[key].(map[string]any); ok {
			normalize(m)
		}
	}
}

type Finding struct {
	Path    string
	Message string
}
type InvalidRequest struct{ Findings []Finding }

func (e *InvalidRequest) Error() string {
	if len(e.Findings) == 0 {
		return "invalid API request"
	}
	f := e.Findings[0]
	return "invalid API request: " + f.Path + ": " + f.Message
}

func KnownKind(kind string) bool { _, ok := Data.Kinds[kind]; return ok || kind == "input-patch" }
func Kinds() []string {
	names := []string{"input-patch"}
	for k := range Data.Kinds {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func decode(body []byte) (any, error) {
	var v any
	d := json.NewDecoder(bytes.NewReader(body))
	d.UseNumber()
	if err := d.Decode(&v); err != nil {
		return nil, fmt.Errorf("expected a JSON object")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("expected exactly one JSON value")
	}
	return v, nil
}

func ValidateKind(kind string, body []byte, readBack bool) []Finding {
	s, ok := Data.Kinds[kind]
	if !ok && kind != "input-patch" {
		return []Finding{{"", "unknown API artifact kind"}}
	}
	v, err := decode(body)
	if err != nil {
		return []Finding{{"", err.Error()}}
	}
	if kind == "input-patch" {
		return validatePatch(v)
	}
	out := []Finding{}
	validate(s, v, "", readBack, 0, &out)
	semantic(kind, v, &out)
	return out
}

func validatePatch(v any) []Finding {
	m, ok := v.(map[string]any)
	if !ok {
		return []Finding{{"", "merge patch must be a JSON object"}}
	}
	out := []Finding{}
	props, _ := Data.Kinds["input"]["properties"].(map[string]any)
	if len(m) == 0 {
		add(&out, "", "Input patch must not be empty")
	}
	allowed := map[string]bool{"description": true, "keys": true, "values": true, "tags": true}
	for _, key := range keys(m) {
		if !allowed[key] {
			add(&out, key, "Input patch supports description, keys, values, and tags only")
			continue
		}
		val := m[key]
		if val == nil {
			if key == "values" {
				add(&out, key, "values cannot be null")
			}
			continue
		}
		raw := props[key]
		s := raw.(map[string]any)
		if key == "values" {
			values, ok := val.(map[string]any)
			if !ok {
				add(&out, key, "must be an object of strings")
				continue
			}
			for _, name := range keys(values) {
				if _, ok := values[name].(string); !ok {
					add(&out, key+"."+name, "must be a string; null values are rejected")
				}
			}
		} else if key == "keys" {
			descriptors, ok := val.(map[string]any)
			if !ok {
				add(&out, key, "must be an object")
				continue
			}
			values, _ := m["values"].(map[string]any)
			schema, _ := s["additionalProperties"].(map[string]any)
			for _, name := range keys(descriptors) {
				if descriptors[name] == nil {
					continue
				}
				validate(schema, descriptors[name], key+"."+name, false, 0, &out)
				if _, ok := values[name]; !ok {
					add(&out, key+"."+name, "non-null key descriptor requires a corresponding value")
				}
			}
		} else {
			validate(s, val, key, false, 0, &out)
		}
	}
	return out
}

func Find(method, path string) (*Operation, map[string]string) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, nil
	}
	path = strings.TrimPrefix(strings.Trim(u.EscapedPath(), "/"), "api/v1/")
	actual := strings.Split(path, "/")
	// Prefer literals such as /executions/summary to identifier routes.
	for _, dynamic := range []bool{false, true} {
		for i := range Data.Operations {
			op := &Data.Operations[i]
			if op.Method != method || strings.Contains(op.Path, "{") != dynamic {
				continue
			}
			expected := strings.Split(strings.Trim(op.Path, "/"), "/")
			if len(expected) != len(actual) {
				continue
			}
			params := map[string]string{}
			match := true
			for j, part := range expected {
				decoded, err := url.PathUnescape(actual[j])
				if err != nil {
					match = false
					break
				}
				if strings.HasPrefix(part, "{") {
					params[strings.Trim(part, "{}")] = decoded
				} else if part != decoded {
					match = false
					break
				}
			}
			if match {
				return op, params
			}
		}
	}
	return nil, nil
}

func ValidateRequest(method, path string, query url.Values, headers map[string]string, body []byte) error {
	op, params := Find(method, path)
	if op == nil {
		return nil
	}
	out := []Finding{}
	for _, p := range op.Parameters {
		if p.In == "body" {
			if len(body) == 0 {
				if p.Required {
					add(&out, "body", "is required")
				}
				continue
			}
			if p.Schema["type"] == "string" && op.OperationID != "patchInput" {
				continue
			}
			v, err := decode(body)
			if err != nil {
				add(&out, "body", err.Error())
				continue
			}
			if op.OperationID == "patchInput" {
				out = append(out, validatePatch(v)...)
			} else {
				validate(p.Schema, v, "body", false, 0, &out)
				semantic(kindFor(op.OperationID), v, &out)
			}
			continue
		}
		var values []string
		switch p.In {
		case "path":
			values = []string{params[p.Name]}
		case "query":
			values = query[p.Name]
		case "header":
			for k, v := range headers {
				if strings.EqualFold(k, p.Name) {
					values = []string{v}
				}
			}
		}
		if len(values) == 0 {
			if p.Required {
				add(&out, p.Name, "is required")
			}
			continue
		}
		s := Schema{"type": p.Type}
		if p.Minimum != nil {
			s["minimum"] = *p.Minimum
		}
		if p.Maximum != nil {
			s["maximum"] = *p.Maximum
		}
		if p.MaxLength != nil {
			s["maxLength"] = float64(*p.MaxLength)
		}
		if len(p.Enum) > 0 {
			s["enum"] = p.Enum
		}
		if p.Type == "array" {
			s["items"] = map[string]any(p.Items)
			if p.MinItems != nil {
				s["minItems"] = float64(*p.MinItems)
			}
			if p.MaxItems != nil {
				s["maxItems"] = float64(*p.MaxItems)
			}
			entries := make([]any, len(values))
			for i, v := range values {
				entries[i] = v
			}
			validate(s, entries, p.Name, false, 0, &out)
		} else {
			var v any = values[0]
			switch p.Type {
			case "integer", "number":
				v = json.Number(values[0])
			case "boolean":
				b, err := strconv.ParseBool(values[0])
				if err != nil {
					add(&out, p.Name, "must be a boolean")
					continue
				}
				v = b
			}
			validate(s, v, p.Name, false, 0, &out)
		}
	}
	if len(out) > 0 {
		return &InvalidRequest{out}
	}
	return nil
}

func kindFor(operation string) string {
	switch operation {
	case "triggerExecution":
		return "execution-request"
	case "requestAction":
		return "action"
	case "requestBulkAction":
		return "bulk-action"
	case "createScheduledExecution", "updateScheduledExecution":
		return "scheduled-execution"
	}
	return ""
}

func semantic(kind string, v any, out *[]Finding) {
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	if kind == "action" {
		typ, _ := m["type"].(string)
		if typ == "FAIL" {
			add(out, "type", "FAIL is not available for public execution states")
		}
		if typ == "SUSPEND" || typ == "EXECUTE_STATUS_LISTENER" {
			add(out, "type", "internal action is not exposed by the public API")
		}
		if typ == "RESET" || typ == "CHOOSE" || typ == "SKIP" {
			if value, ok := m["value"].(string); !ok || strings.TrimSpace(value) == "" {
				add(out, "value", "is required for RESET, CHOOSE, and SKIP")
			}
		}
		if typ == "SKIP" {
			text, _ := m["value"].(string)
			payload, err := decode([]byte(text))
			object, ok := payload.(map[string]any)
			if err != nil || !ok {
				add(out, "value", "SKIP requires a JSON object serialized into a string")
			} else {
				path, ok := object["executorPath"].(string)
				if !ok || strings.TrimSpace(path) == "" {
					add(out, "value.executorPath", "is required")
				}
				if values, present := object["outputValues"]; present {
					validate(Schema{"type": "object", "additionalProperties": map[string]any{"type": "string"}}, values, "value.outputValues", false, 0, out)
				}
				if target, present := object["target"]; present {
					if _, ok := target.(string); !ok {
						add(out, "value.target", "must be a string")
					}
				}
			}
		}
	}
	if kind == "action" || kind == "bulk-action" {
		if reason, ok := m["reason"].(string); !ok || strings.TrimSpace(reason) == "" {
			add(out, "reason", "a non-empty reason is required")
		}
	}
	if kind == "bulk-action" {
		typ, _ := m["type"].(string)
		if typ != "PAUSE" && typ != "RESUME" && typ != "ABORT" && typ != "RETRY" {
			add(out, "type", "bulk actions support PAUSE, RESUME, ABORT, and RETRY")
		}
	}
	if kind == "scheduled-execution" {
		s, ok := m["schedule"].(map[string]any)
		if !ok {
			return
		}
		if zone, present := s["timeZone"]; present && zone == nil {
			add(out, "schedule.timeZone", "must not be null")
		}
		count := 0
		for _, key := range []string{"once", "hourly", "daily", "weekly", "monthly", "yearly"} {
			if s[key] != nil {
				count++
			}
		}
		if count != 1 {
			add(out, "schedule", "requires exactly one recurrence type")
		}
		deadline := float64(5)
		if raw, present := s["deadlineInMinutes"]; present {
			value, ok := number(raw)
			if !ok {
				add(out, "schedule.deadlineInMinutes", "must not be null")
			} else {
				deadline = value
			}
		}
		for _, key := range []string{"once", "hourly", "daily", "weekly", "monthly", "yearly"} {
			block, ok := s[key].(map[string]any)
			if !ok {
				continue
			}
			if key == "once" {
				if minute, ok := number(block["minute"]); ok && int(minute)%5 != 0 {
					add(out, "schedule.once.minute", "must be a multiple of five")
				}
				continue
			}
			raw, _ := block["minutes"].([]any)
			minutes := []float64{}
			for _, value := range raw {
				if minute, ok := number(value); ok {
					minutes = append(minutes, minute)
					if int(minute)%5 != 0 {
						add(out, "schedule."+key+".minutes", "minutes must be multiples of five")
					}
				}
			}
			sort.Float64s(minutes)
			window := float64(60)
			for i, minute := range minutes {
				next := minutes[(i+1)%len(minutes)]
				gap := next - minute
				if i == len(minutes)-1 {
					gap += 60
				}
				if gap < window {
					window = gap
				}
			}
			if deadline > window {
				add(out, "schedule.deadlineInMinutes", "cannot exceed the shortest cyclic minute window")
			}
		}
	}
}

func add(out *[]Finding, path, message string) {
	if len(*out) < 64 {
		*out = append(*out, Finding{path, message})
	}
}
func keys(m map[string]any) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
func number(v any) (float64, bool) {
	switch n := v.(type) {
	case json.Number:
		f, err := n.Float64()
		return f, err == nil && !math.IsInf(f, 0) && !math.IsNaN(f)
	case float64:
		return n, true
	}
	return 0, false
}
func bound(s Schema, key string) (float64, bool) { v, ok := s[key].(float64); return v, ok }

func validate(s Schema, v any, path string, readBack bool, depth int, out *[]Finding) {
	if depth > 64 {
		add(out, path, "schema nesting limit exceeded")
		return
	}
	if v == nil {
		if readBack {
			return
		}
		if nullable, _ := s["nullable"].(bool); nullable {
			return
		}
		add(out, path, "must not be null")
		return
	}
	if readOnly, _ := s["readOnly"].(bool); readOnly {
		return
	}
	typ, _ := s["type"].(string)
	switch typ {
	case "object":
		m, ok := v.(map[string]any)
		if !ok {
			add(out, path, "must be an object")
			return
		}
		props, _ := s["properties"].(map[string]any)
		required := map[string]bool{}
		if entries, ok := s["required"].([]any); ok {
			for _, entry := range entries {
				if k, ok := entry.(string); ok {
					required[k] = true
				}
			}
		}
		for _, k := range keysBool(required) {
			raw, _ := props[k].(map[string]any)
			ro, _ := raw["readOnly"].(bool)
			if (readBack && (k == "values" || k == "configuration" || k == "technicalName")) || ro {
				continue
			}
			if _, ok := m[k]; !ok {
				add(out, join(path, k), "is required")
			}
		}
		for _, k := range keys(m) {
			value := m[k]
			raw, known := props[k]
			var child Schema
			if known {
				child, _ = raw.(map[string]any)
				if value == nil && !required[k] {
					continue
				}
			} else if ap, ok := s["additionalProperties"].(map[string]any); ok {
				child = ap
			} else {
				if s["additionalProperties"] == false {
					add(out, join(path, k), "unknown property")
				}
				continue
			}
			validate(child, value, join(path, k), readBack, depth+1, out)
		}
		if max, ok := bound(s, "maxProperties"); ok && float64(len(m)) > max {
			add(out, path, fmt.Sprintf("must have at most %.0f properties", max))
		}
	case "array":
		a, ok := v.([]any)
		if !ok {
			add(out, path, "must be an array")
			return
		}
		for _, limit := range []struct {
			key   string
			lower bool
		}{{"minItems", true}, {"maxItems", false}} {
			if n, ok := bound(s, limit.key); ok && ((limit.lower && float64(len(a)) < n) || (!limit.lower && float64(len(a)) > n)) {
				add(out, path, fmt.Sprintf("violates %s %.0f", limit.key, n))
			}
		}
		seen := map[string]bool{}
		unique, _ := s["uniqueItems"].(bool)
		item, _ := s["items"].(map[string]any)
		for i, value := range a {
			validate(item, value, fmt.Sprintf("%s[%d]", path, i), readBack, depth+1, out)
			if unique {
				raw, _ := json.Marshal(value)
				if seen[string(raw)] {
					add(out, path, "items must be unique")
				}
				seen[string(raw)] = true
			}
		}
	case "string":
		text, ok := v.(string)
		if !ok {
			add(out, path, "must be a JSON string")
			return
		}
		length := float64(utf8.RuneCountInString(text))
		for _, limit := range []struct {
			key   string
			lower bool
		}{{"minLength", true}, {"maxLength", false}} {
			if n, ok := bound(s, limit.key); ok && ((limit.lower && length < n) || (!limit.lower && length > n)) {
				add(out, path, fmt.Sprintf("violates %s %.0f", limit.key, n))
			}
		}
		if pattern, ok := s["pattern"].(string); ok {
			r, err := regexp.Compile(pattern)
			if err != nil {
				add(out, path, "unsupported schema pattern")
			} else if !r.MatchString(text) {
				add(out, path, "does not match the API pattern")
			}
		}
	case "integer", "number":
		n, ok := number(v)
		if !ok || (typ == "integer" && math.Trunc(n) != n) {
			add(out, path, "must be a "+typ)
			return
		}
		if min, ok := bound(s, "minimum"); ok && n < min {
			add(out, path, fmt.Sprintf("must be >= %g", min))
		}
		if max, ok := bound(s, "maximum"); ok && n > max {
			add(out, path, fmt.Sprintf("must be <= %g", max))
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			add(out, path, "must be a boolean")
			return
		}
	}
	if values, ok := s["enum"].([]any); ok {
		found := false
		for _, allowed := range values {
			if fmt.Sprint(v) == fmt.Sprint(allowed) {
				found = true
			}
		}
		if !found {
			add(out, path, "is outside the API enum")
		}
	}
}
func keysBool(m map[string]bool) []string {
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}
