package lint

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/domain"
)

var (
	pascalCase      = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	kebabCase       = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	controlPath     = regexp.MustCompile(`^(configuration\.)?executors\[\d+\]\.(autoRetry|repeat|initialDelay)$`)
	schemaDefault   = regexp.MustCompile(`^(keys|inputKeys|outputKeys)\.[^.]+\.defaultValue$`)
	executorInput   = regexp.MustCompile(`^(configuration\.)?executors\[\d+\]\.input$`)
	conditionValues = regexp.MustCompile(`^(configuration\.)?executors\[\d+\]\.(when|validate|repeat\.until|autoRetry\.when|initialDelay\.when|errorMessages\[\d+\]\.when)\.conditions\[\d+\]\.cases\[\d+\]\.values$`)
	badFuncs        = regexp.MustCompile(`\|\s*(isBool|isString|isNil)\b`)
)

// CheckFile lints one file's bytes. name is the display path.
func CheckFile(name string, b []byte) []Finding {
	return CheckFileMode(name, b, false)
}

// CheckFileMode distinguishes authored content from retrieved atomic definitions.
func CheckFileMode(name string, b []byte, readBack bool) []Finding {
	var out []Finding
	doc, fieldKind, err := Parse(b)
	if err != nil {
		return []Finding{{Severity: Critical, File: name, Message: err.Error()}}
	}

	// The filename extension (.command.json/.input.json/.catalog.json) is the
	// authoritative kind signal; fall back to field-based classification only
	// when the name carries no recognized extension.
	kind := kindFromName(name)
	if kind == KindUnknown {
		kind = fieldKind
	}

	out = append(out, checkStructure(name, kind, doc.Fields, readBack)...)

	// Structure: filename stem (before the first dot) should match the name.
	stem := strings.SplitN(filepath.Base(name), ".", 2)[0]
	if doc.Name != "" && stem != "" && !strings.EqualFold(stem, doc.Name) {
		out = append(out, Finding{High, name, "name",
			"filename stem " + q(stem) + " does not match JSON name " + q(doc.Name)})
	}

	switch kind {
	case KindCommand, KindInput:
		if doc.Name != "" && !pascalCase.MatchString(doc.Name) {
			out = append(out, Finding{High, name, "name", "name " + q(doc.Name) + " should be PascalCase"})
		}
		if len(doc.Name) > 32 {
			out = append(out, Finding{Medium, name, "name", "name exceeds 32 characters"})
		}
	case KindCatalog:
		if doc.Name != "" && !kebabCase.MatchString(doc.Name) {
			out = append(out, Finding{Medium, name, "name", "catalog name " + q(doc.Name) + " should be kebab-case"})
		}
	}

	// Credential/sensitive-flag rules apply to input-definition keys and to
	// command input schemas alike.
	switch kind {
	case KindInput:
		out = append(out, checkKeys(name, "keys", doc.Keys)...)
	case KindCommand:
		out = append(out, checkKeys(name, "inputKeys", doc.InputKeys)...)
		out = append(out, checkControlDurations(name, doc.Fields)...)
	}

	out = append(out, checkTagsMode(name, doc.Tags, readBack)...)
	out = append(out, checkWireValues(name, doc.Fields)...)
	out = append(out, checkExpressions(name, doc.Fields)...)
	if kind == KindCommand {
		out = append(out, checkKeys(name, "outputKeys", doc.OutputKeys)...)
	}
	return out
}

// kindFromName classifies by the compound file extension used across the repo.
func kindFromName(name string) Kind {
	base := filepath.Base(name)
	switch {
	case strings.HasSuffix(base, ".command.json"):
		return KindCommand
	case strings.HasSuffix(base, ".input.json"):
		return KindInput
	case strings.HasSuffix(base, ".catalog.json"):
		return KindCatalog
	}
	return KindUnknown
}

// checkKeys enforces the sensitive-flag rules over a map of keys. field is the
// JSON field the map came from ("keys" or "inputKeys"), used for finding paths.
// Keys are visited in sorted order so findings are deterministic.
func checkKeys(name, field string, keys map[string]InputKey) []Finding {
	var out []Finding
	names := make([]string, 0, len(keys))
	for keyName := range keys {
		names = append(names, keyName)
	}
	sort.Strings(names)
	for _, keyName := range names {
		k := keys[keyName]
		path := field + "." + keyName
		isSecret := domain.SecretKey(keyName)
		switch {
		case k.Sensitive == nil:
			sev := Medium
			msg := "key " + q(keyName) + " has no explicit \"sensitive\" flag"
			if isSecret {
				sev = Critical
				msg = "credential key " + q(keyName) + " must set \"sensitive\": true"
			}
			out = append(out, Finding{sev, name, path, msg})
		case isSecret && !*k.Sensitive:
			out = append(out, Finding{Critical, name, path, "credential key " + q(keyName) + " must set \"sensitive\": true"})
		}
		if text, ok := k.DefaultValue.(string); k.Sensitive != nil && *k.Sensitive && k.DefaultValue != nil && k.DefaultValue != "" && (!ok || !strings.Contains(text, "$(")) {
			out = append(out, Finding{Critical, name, path, "sensitive key " + q(keyName) + " must not have a hardcoded defaultValue"})
		}
	}
	return out
}

// walk visits fields deterministically, preserving the path and value type.
func walk(value any, path string, visit func(string, string, any)) {
	switch v := value.(type) {
	case map[string]any:
		names := make([]string, 0, len(v))
		for key := range v {
			names = append(names, key)
		}
		sort.Strings(names)
		for _, key := range names {
			p := key
			if path != "" {
				p = path + "." + key
			}
			visit(p, key, v[key])
			walk(v[key], p, visit)
		}
	case []any:
		for i, item := range v {
			walk(item, fmt.Sprintf("%s[%d]", path, i), visit)
		}
	}
}

func checkStructure(file string, kind Kind, fields map[string]any, readBack bool) []Finding {
	var out []Finding
	add := func(path, msg string) { out = append(out, Finding{High, file, path, msg}) }
	if name, ok := fields["name"].(string); !ok || strings.TrimSpace(name) == "" {
		add("name", "non-empty name is required")
	}
	if kind == KindUnknown {
		add("", "unrecognized artifact kind; use a command/input/catalog extension")
	}
	if kind == KindCommand || kind == KindInput {
		if catalog, ok := fields["catalog"].(string); !ok || catalog == "" {
			add("catalog", "catalog is required")
		}
		if version, ok := fields["version"].(float64); !ok || version < 1 || version != float64(int(version)) {
			add("version", "positive integer version is required")
		}
	}
	if kind == KindCommand {
		for _, key := range []string{"inputKeys", "outputKeys"} {
			if _, ok := fields[key].(map[string]any); !ok {
				add(key, "command contract requires a key definition object")
			}
		}
		config, ok := fields["configuration"].(map[string]any)
		if !ok {
			if !(readBack && fields["configuration"] == nil) {
				add("configuration", "tenant-authored command requires a configuration object")
			}
		} else {
			executors, ok := config["executors"].([]any)
			if !ok || len(executors) == 0 {
				add("configuration.executors", "a composite requires non-empty executors")
			}
			aliases := map[string]bool{}
			for i, raw := range executors {
				p := fmt.Sprintf("configuration.executors[%d]", i)
				ex, ok := raw.(map[string]any)
				if !ok {
					add(p, "executor must be an object")
					continue
				}
				alias, ok := ex["alias"].(string)
				if !ok || alias == "" || aliases[alias] {
					add(p+".alias", "executor alias must be non-empty and unique")
				}
				aliases[alias] = true
				if command, ok := ex["execute"].(string); !ok || !validCommandID(command) {
					add(p+".execute", "executor requires an exact catalog:name:version command ID")
				}
				if input, present := ex["input"]; present {
					if _, ok := input.(map[string]any); !ok {
						add(p+".input", "executor input must be an object")
					}
				}
			}
		}
	}
	if kind == KindInput {
		keys, ok := fields["keys"].(map[string]any)
		if !ok {
			add("keys", "Input keys must be an object")
		} else if len(keys) > 32 {
			add("keys", "Input supports at most 32 keys")
		}
	}
	walk(fields, "", func(path, key string, value any) {
		if path != "keys" && path != "inputKeys" && path != "outputKeys" {
			return
		}
		keys, ok := value.(map[string]any)
		if !ok {
			add(path, "key definitions must be an object")
			return
		}
		for _, name := range sortedKeys(keys) {
			schema, ok := keys[name].(map[string]any)
			if !ok {
				add(path+"."+name, "key definition must be an object")
				continue
			}
			typ, _ := schema["type"].(string)
			switch typ {
			case "string", "number", "boolean", "array", "object":
			default:
				add(path+"."+name+".type", "unsupported or missing key type")
			}
		}
	})
	return out
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func checkWireValues(file string, fields map[string]any) []Finding {
	var out []Finding
	add := func(path, message string) { out = append(out, Finding{High, file, path, message}) }
	walk(fields, "", func(path, key string, value any) {
		if schemaDefault.MatchString(path) && value != nil {
			if _, ok := value.(string); !ok {
				add(path, "defaultValue must be a JSON string")
			}
			return
		}
		if path == "values" || path == "inputValues" || path == "configuration.output" || executorInput.MatchString(path) {
			// configuration.values is an Input-binding array, not a wire value map.
			values, ok := value.(map[string]any)
			if !ok {
				add(path, "wire values must be a JSON object of strings")
				return
			}
			for _, k := range sortedKeys(values) {
				if _, ok := values[k].(string); !ok {
					add(path+"."+k, "wire value must be a JSON string")
				}
			}
		}
		if conditionValues.MatchString(path) {
			values, ok := value.([]any)
			if !ok {
				add(path, "condition values must be an array of JSON strings")
				return
			}
			for _, v := range values {
				if _, ok := v.(string); !ok {
					add(path, "condition values must be JSON strings")
					break
				}
			}
		}
	})
	return out
}

// Expressions returns balanced $() substitutions; quotes and escapes may contain parentheses.
func Expressions(value string) []string {
	var out []string
	for start := 0; start < len(value); {
		offset := strings.Index(value[start:], "$(")
		if offset < 0 {
			break
		}
		i := start + offset + 2
		begin := i
		depth := 1
		quoted := false
		escape := false
		for ; i < len(value); i++ {
			ch := value[i]
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '"' {
				quoted = !quoted
				continue
			}
			if quoted {
				continue
			}
			if ch == '(' {
				depth++
			}
			if ch == ')' {
				depth--
				if depth == 0 {
					out = append(out, value[begin:i])
					i++
					break
				}
			}
		}
		start = i
	}
	return out
}

func checkExpressions(file string, fields map[string]any) []Finding {
	var out []Finding
	walk(fields, "", func(path, key string, value any) {
		text, ok := value.(string)
		if !ok || key == "description" || key == "instructions" || key == "script" {
			return
		}
		expressions := Expressions(text)
		if key == "responseBodyTransformer" {
			expressions = append(expressions, text)
		}
		for _, expr := range expressions {
			if badFuncs.MatchString(expr) {
				out = append(out, Finding{Medium, file, path, "uses a jq function that does not exist in the expression language (isBool/isString/isNil)"})
				break
			}
		}
		if key != "url" || isSingleExpression(text) {
			return
		}
		for _, expr := range Expressions(text) {
			expr = strings.TrimSpace(expr)
			// A resolved API base URL is not a path/query segment; its contract supplies the URL.
			if strings.HasSuffix(strings.ToLower(expr), "url") || strings.HasSuffix(strings.ToLower(expr), "uri") {
				continue
			}
			if strings.Contains(expr, "toUrlEncoded") || strings.Contains(expr, "toPercentEncoded") {
				continue
			}
			if strings.HasPrefix(expr, ".") || strings.HasPrefix(expr, "$.") {
				out = append(out, Finding{Medium, file, path, "URL substitution needs | toUrlEncoded for query values or | toPercentEncoded for path segments"})
			}
		}
	})
	return out
}

func isSingleExpression(value string) bool {
	s := strings.TrimSpace(value)
	return strings.HasPrefix(s, "$(") && strings.HasSuffix(s, ")") && len(Expressions(s)) == 1 && len(Expressions(s)[0])+3 == len(s)
}

func checkControlDurations(file string, fields map[string]any) []Finding {
	var out []Finding
	walk(fields, "", func(path, key string, value any) {
		field := ""
		switch key {
		case "autoRetry", "repeat":
			field = "delay"
		case "initialDelay":
			field = "interval"
		default:
			return
		}
		if !controlPath.MatchString(path) {
			return
		}
		control, ok := value.(map[string]any)
		if !ok {
			return
		}
		literal, ok := control[field].(string)
		if !ok {
			out = append(out, Finding{High, file, path + "." + field, field + " must be a duration string"})
			return
		}
		if strings.Contains(literal, "$(") || domain.ValidDuration(literal) {
			return
		}
		out = append(out, Finding{High, file, path + "." + field, field + " " + q(literal) + " is not an allowed interval (use " + strings.Join(domain.Durations, ", ") + ")"})
	})
	return out
}

func checkTagsMode(file string, tags map[string]json.RawMessage, readBack bool) []Finding {
	var out []Finding
	values := map[string]string{}
	names := make([]string, 0, len(tags))
	for k := range tags {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, key := range names {
		var value string
		if len(tags[key]) == 0 || tags[key][0] != '"' || json.Unmarshal(tags[key], &value) != nil {
			out = append(out, Finding{High, file, "tags." + key, "tag value must be a JSON string"})
			continue
		}
		values[key] = value
	}
	if err := domain.ValidateTags(values, readBack); err != nil {
		sev := High
		if strings.Contains(err.Error(), "reserved") {
			sev = Critical
		}
		out = append(out, Finding{sev, file, "tags", err.Error()})
	}
	return out
}

func q(s string) string { return strconv.Quote(s) }

func validCommandID(id string) bool {
	parts := strings.Split(id, ":")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" {
		return false
	}
	version, err := strconv.Atoi(parts[2])
	return err == nil && version > 0
}
