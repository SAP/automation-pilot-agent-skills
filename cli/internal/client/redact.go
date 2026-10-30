package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/domain"
)

func protectedInput(path string) bool {
	p, err := url.Parse(path)
	if err != nil {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(strings.Trim(p.Path, "/"), "api/v1/"), "/")
	return len(parts) > 0 && (parts[0] == "inputs" || len(parts) > 1 && parts[0] == "bulk" && parts[1] == "inputs")
}

// MCP uses POST for both schema reads and tool execution. Its explicit protocol
// responses have full-read semantics: masking field names would corrupt JSON
// Schema and tool outputs; negotiated SSE framing must also survive unchanged.
// REST MCP-server management remains covered by mutation protection.
func mcpProtocol(req Request) bool {
	if req.Method != "POST" {
		return false
	}
	u, err := url.Parse(req.Path)
	if err != nil {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(strings.Trim(u.Path, "/"), "api/v1/"), "/")
	if len(parts) != 2 || parts[0] != "mcp" || parts[1] == "" {
		return false
	}
	value, err := decodeJSON(req.Body)
	if err != nil {
		return false
	}
	m, ok := value.(map[string]any)
	if !ok {
		return false
	}
	method, ok := m["method"].(string)
	return ok && method != "" && m["jsonrpc"] == "2.0"
}

func decodeJSON(body []byte) (any, error) {
	var value any
	d := json.NewDecoder(bytes.NewReader(body))
	d.UseNumber()
	if err := d.Decode(&value); err != nil {
		return nil, err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("expected one JSON value")
	}
	return value, nil
}

// inputMetadata is a keep-list: unexpected shapes and messages cannot echo secrets.
func inputMetadata(body []byte) []byte {
	v, err := decodeJSON(body)
	if err != nil {
		return nil
	}
	var project func(any) any
	project = func(v any) any {
		switch value := v.(type) {
		case []any:
			out := make([]any, len(value))
			for i, item := range value {
				out[i] = project(item)
			}
			return out
		case map[string]any:
			out := map[string]any{}
			for _, key := range []string{"id", "catalog", "name", "version"} {
				if field, ok := value[key]; ok {
					switch field.(type) {
					case string, json.Number:
						out[key] = field
					}
				}
			}
			for _, key := range []string{"inputs", "results"} {
				if field, ok := value[key]; ok {
					out[key] = project(field)
				}
			}
			redact(out)
			return out
		default:
			return nil
		}
	}
	b, _ := json.Marshal(project(v))
	return b
}

func redact(value any) {
	redactDepth(value, 0)
}

func redactDepth(value any, depth int) {
	switch v := value.(type) {
	case []any:
		for i, item := range v {
			v[i] = redactEncodedValue(item, depth)
			redactDepth(v[i], depth+1)
		}
	case map[string]any:
		if sensitive, _ := v["sensitive"].(bool); sensitive {
			delete(v, "defaultValue")
		}
		for key, item := range v {
			// Schema maps carry credential names as keys. Only known descriptor
			// types and typed metadata may bypass masking a credential-named object.
			// Arbitrary payloads can also contain fields called "keys" or "type".
			if key == "keys" || key == "inputKeys" || key == "outputKeys" {
				if schemas, ok := item.(map[string]any); ok {
					for name, raw := range schemas {
						if schema, ok := raw.(map[string]any); ok && keySchemaType(schema["type"]) {
							if sensitive, _ := schema["sensitive"].(bool); sensitive || domain.SecretKey(name) || schema["type"] == "secret" {
								schema = keySchemaMetadata(schema)
								schemas[name] = schema
							}
							redactDepth(schema, depth+1)
						} else {
							schemas[name] = "***"
						}
					}
					continue
				}
			}
			if domain.SecretKey(key) {
				v[key] = "***"
			} else if key == "values" || key == "inputValues" || key == "outputValues" {
				v[key] = "***"
			} else {
				lower := strings.ToLower(key)
				if text, ok := item.(string); ok && (strings.HasSuffix(lower, "url") || strings.HasSuffix(lower, "uri")) {
					v[key] = redactURL(text)
					continue
				}
				v[key] = redactEncodedValue(item, depth)
				redactDepth(v[key], depth+1)
			}
		}
	}
}

func keySchemaType(value any) bool {
	switch value {
	case "boolean", "number", "string", "object", "array", "secret":
		return true
	}
	return false
}

// Keep contract metadata for sensitive keys, never literal values or unknown
// fields. Type checks prevent a structured credential from posing as metadata.
func keySchemaMetadata(schema map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range schema {
		keep := false
		switch key {
		case "type", "description":
			_, keep = value.(string)
		case "required", "sensitive":
			_, keep = value.(bool)
		case "minSize", "maxSize", "minValue", "maxValue":
			_, keep = value.(json.Number)
		case "defaultValueFromInput":
			if reference, ok := value.(map[string]any); ok {
				safe := map[string]any{}
				for _, field := range []string{"inputReference", "inputKey"} {
					if text, ok := reference[field].(string); ok {
						safe[field] = text
					}
				}
				out[key] = safe
			}
		case "allowedValuesFromInputKeys", "suggestedValuesFromInputKeys":
			if names, ok := value.([]any); ok {
				keep = true
				for _, name := range names {
					if _, ok := name.(string); !ok {
						keep = false
					}
				}
			}
		}
		if keep {
			out[key] = value
		}
	}
	return out
}

// Container values can be JSON-encoded strings on the wire, including in arrays.
// A malformed or too-deep encoded container cannot be safely inspected.
func redactEncodedValue(value any, depth int) any {
	text, ok := value.(string)
	if !ok {
		return value
	}
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return value
	}
	if depth >= 32 {
		return "***"
	}
	decoded, err := decodeJSON([]byte(text))
	if err != nil {
		return "***"
	}
	redactDepth(decoded, depth+1)
	safe, err := json.Marshal(decoded)
	if err != nil {
		return "***"
	}
	return string(safe)
}

func redactBody(body []byte) []byte {
	v, err := decodeJSON(body)
	if err != nil {
		return []byte("<body omitted: non-JSON payload>")
	}
	v = redactEncodedValue(v, 0)
	redact(v)
	b, _ := json.Marshal(v)
	return b
}

func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<url omitted: invalid URL>"
	}
	if u.User != nil {
		u.User = url.User("***")
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "<url omitted: invalid query>"
	}
	for key := range q {
		if domain.SecretKey(key) {
			q.Set(key, "***")
		}
	}
	if u.RawQuery != "" {
		u.RawQuery = q.Encode()
	}
	// OAuth responses and SPA routes can put credentials in fragment parameters.
	// Omit the whole fragment on a secret or parse failure, retaining ordinary anchors.
	fragments := []string{u.EscapedFragment()}
	if _, query, ok := strings.Cut(fragments[0], "?"); ok {
		// A '?' may also occur in an OAuth state value. Inspect both forms so
		// selecting a SPA query never discards a preceding credential parameter.
		fragments = append(fragments, query)
	}
	for _, fragment := range fragments {
		if !strings.Contains(fragment, "=") {
			continue
		}
		params, err := url.ParseQuery(fragment)
		omit := err != nil
		for key := range params {
			omit = omit || domain.SecretKey(key)
		}
		if omit {
			u.Fragment, u.RawFragment = "***", ""
			break
		}
	}
	return u.String()
}

// ProjectObject applies a strict projection without disclosing unexpected payloads.
func ProjectObject(body []byte, keys ...string) ([]byte, error) {
	v, err := decodeJSON(body)
	if err != nil {
		return nil, fmt.Errorf("expected a JSON object response")
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected a JSON object response")
	}
	out := map[string]any{}
	for _, key := range keys {
		if field, ok := m[key]; ok {
			out[key] = field
		}
	}
	return json.Marshal(out)
}

// InputContract projects metadata and redacts sensitive schema defaults.
func InputContract(body []byte) ([]byte, error) {
	return redactedObject(body, "id", "catalog", "name", "description", "version", "keys", "tags")
}

// CommandContract protects sensitive defaults while preserving caller metadata.
func CommandContract(body []byte) ([]byte, error) {
	return redactedObject(body, "id", "catalog", "name", "description", "version", "inputKeys", "outputKeys", "tags")
}

func redactedObject(body []byte, keys ...string) ([]byte, error) {
	b, err := ProjectObject(body, keys...)
	if err != nil {
		return nil, err
	}
	return redactBody(b), nil
}

// InputList exposes only known list metadata. Unexpected response envelopes or
// metadata types fail closed instead of passing potentially secret-bearing data.
func InputList(body []byte) ([]byte, error) {
	v, err := decodeJSON(body)
	if err != nil {
		return nil, fmt.Errorf("expected an array of Input metadata objects")
	}
	items, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("expected an array of Input metadata objects")
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected an array of Input metadata objects")
		}
		metadata := map[string]any{}
		for _, key := range []string{"id", "catalog", "name", "description", "owner", "version", "tags"} {
			value, present := object[key]
			if !present {
				continue
			}
			if value != nil {
				switch key {
				case "version":
					if _, ok := value.(json.Number); !ok {
						return nil, fmt.Errorf("unexpected Input metadata field type")
					}
				case "tags":
					tags, ok := value.(map[string]any)
					if !ok {
						return nil, fmt.Errorf("unexpected Input metadata field type")
					}
					for _, tag := range tags {
						if _, ok := tag.(string); !ok {
							return nil, fmt.Errorf("unexpected Input metadata field type")
						}
					}
				default:
					if _, ok := value.(string); !ok {
						return nil, fmt.Errorf("unexpected Input metadata field type")
					}
				}
			}
			metadata[key] = value
		}
		redact(metadata)
		out = append(out, metadata)
	}
	return json.Marshal(out)
}
