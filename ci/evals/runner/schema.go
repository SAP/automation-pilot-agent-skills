package main

import (
	"encoding/json"
	"reflect"
	"strings"
)

func schemaType(t reflect.Type) map[string]any {
	if t == reflect.TypeOf(json.RawMessage{}) {
		return map[string]any{}
	}
	if t.Kind() == reflect.Pointer {
		return schemaType(t.Elem())
	}
	switch t.Kind() {
	case reflect.Interface:
		return map[string]any{}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice:
		return map[string]any{"anyOf": []any{map[string]any{"type": "array", "items": schemaType(t.Elem())}, map[string]any{"type": "null"}}}
	case reflect.Map:
		return map[string]any{"anyOf": []any{map[string]any{"type": "object", "additionalProperties": schemaType(t.Elem())}, map[string]any{"type": "null"}}}
	case reflect.Struct:
		props := map[string]any{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := strings.Split(f.Tag.Get("json"), ",")
			if tag[0] == "" || tag[0] == "-" {
				continue
			}
			s := schemaType(f.Type)
			if f.Type.Kind() == reflect.Pointer {
				s = map[string]any{"anyOf": []any{s, map[string]any{"type": "null"}}}
			}
			if tag[0] == "tier" {
				s["enum"] = []string{"local", "fixture", "engine"}
			}
			if tag[0] == "split" {
				s["enum"] = []string{"development", "holdout"}
			}
			if tag[0] == "suites" {
				s = map[string]any{"anyOf": []any{map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": suiteNames}, "uniqueItems": true}, map[string]any{"type": "null"}}}
			}
			props[tag[0]] = s
			if len(tag) == 1 {
				required = append(required, tag[0])
			}
		}
		return map[string]any{"type": "object", "additionalProperties": false, "properties": props, "required": required}
	default:
		return map[string]any{}
	}
}
func bankSchema() map[string]any {
	schema := schemaType(reflect.TypeOf(bank{}))
	schema["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	schema["title"] = "Automation Pilot evaluator-only task bank"
	return schema
}
