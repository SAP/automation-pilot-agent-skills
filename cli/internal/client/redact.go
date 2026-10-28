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
		for _, item := range v {
			redactDepth(item, depth+1)
		}
	case map[string]any:
		if sensitive, _ := v["sensitive"].(bool); sensitive {
			delete(v, "defaultValue")
		}
		for key, item := range v {
			if domain.SecretKey(key) {
				if schema, ok := item.(map[string]any); ok && schema["type"] != nil {
					delete(schema, "defaultValue")
					redactDepth(schema, depth+1)
				} else {
					v[key] = "***"
				}
			} else if key == "values" || key == "inputValues" || key == "outputValues" {
				v[key] = "***"
			} else {
				// Header/object/array values are often JSON strings on the wire.
				if text, ok := item.(string); ok && depth < 32 {
					if decoded, err := decodeJSON([]byte(text)); err == nil {
						switch decoded.(type) {
						case map[string]any, []any:
							redactDepth(decoded, depth+1)
							safe, _ := json.Marshal(decoded)
							v[key] = string(safe)
						}
					}
				}
				redactDepth(item, depth+1)
			}
		}
	}
}

func redactBody(body []byte) []byte {
	v, err := decodeJSON(body)
	if err != nil {
		return []byte("<body omitted: non-JSON payload>")
	}
	redact(v)
	b, _ := json.Marshal(v)
	return b
}

func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	for key := range q {
		if domain.SecretKey(key) {
			q.Set(key, "***")
		}
	}
	if u.RawQuery != "" {
		u.RawQuery = q.Encode()
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
	b, err := ProjectObject(body, "id", "catalog", "name", "description", "version", "keys", "tags")
	if err != nil {
		return nil, err
	}
	return redactBody(b), nil
}
