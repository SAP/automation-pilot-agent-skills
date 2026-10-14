package lint

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	pascalCase = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	kebabCase  = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	// credential-ish key names that must be marked sensitive
	secretKey = regexp.MustCompile(`(?i)(password|secret|token|credential|apikey|api_key|privatekey|private_key|passphrase)`)
	// jq functions that do not exist in the expression language
	badFuncs = regexp.MustCompile(`\|\s*(isBool|isString|isNil|type)\b`)
	// captures the string value of a "url" field (JSON-escaped, inner quotes escaped)
	urlValue = regexp.MustCompile(`"url"\s*:\s*"((?:[^"\\]|\\.)*)"`)
	// captures duration fields (autoRetry/repeat delay, initialDelay interval)
	durationField = regexp.MustCompile(`"(delay|interval)"\s*:\s*"([^"]*)"`)
)

// allowedDurations is the server's enum for autoRetry.delay / repeat.delay /
// initialDelay.interval. Any other literal value is rejected at deploy time with
// HTTP 400 ("supported values are: 5s, 10s, 15s, 30s, 1m, 2m, 3m").
var allowedDurations = map[string]bool{
	"5s": true, "10s": true, "15s": true, "30s": true,
	"1m": true, "2m": true, "3m": true,
}

// CheckFile lints one file's bytes. name is the display path.
func CheckFile(name string, b []byte) []Finding {
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
		out = append(out, checkDurations(name, b)...)
	}

	out = append(out, checkExpressions(name, b)...)
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
		isSecret := secretKey.MatchString(keyName)
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
		if k.Sensitive != nil && *k.Sensitive && k.DefaultValue != nil && k.DefaultValue != "" {
			out = append(out, Finding{Critical, name, path, "sensitive key " + q(keyName) + " must not have a hardcoded defaultValue"})
		}
	}
	return out
}

// checkExpressions runs light text heuristics for unsafe expression usage.
func checkExpressions(name string, b []byte) []Finding {
	var out []Finding
	text := string(b)
	if badFuncs.MatchString(text) {
		out = append(out, Finding{Medium, name, "", "uses a jq function that does not exist in the expression language (type/isBool/isString/isNil)"})
	}
	// URL interpolation without encoding. Only flag a url value that splices an
	// expression into surrounding literal text — e.g. "https://h/$(.input.x)" —
	// where the interpolated segment needs | toUrlEncoded. A url whose entire
	// value is a single expression (e.g. "$(.input.endpoint)") has nothing to
	// encode and must not be flagged.
	for _, m := range urlValue.FindAllStringSubmatch(text, -1) {
		v := m[1]
		if !strings.Contains(v, "$(.") || strings.Contains(v, "toUrlEncoded") {
			continue
		}
		if isSingleExpression(v) {
			continue
		}
		out = append(out, Finding{Medium, name, "url", "URL interpolates $(.) into literal text without | toUrlEncoded"})
	}
	return out
}

// isSingleExpression reports whether s is exactly one $(...) expression with no
// surrounding literal text — i.e. the whole url value is the expression.
func isSingleExpression(s string) bool {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "$(") || !strings.HasSuffix(s, ")") {
		return false
	}
	// A second "$(" means there is more than one expression (so literal text or
	// concatenation sits between them) — not a single wrapping expression.
	return strings.Count(s, "$(") == 1
}

// checkDurations flags autoRetry.delay / repeat.delay / initialDelay.interval
// literals that are not in the server's enum — the API rejects anything else
// with HTTP 400 at deploy, which no earlier check would catch. A dynamic
// expression value is skipped (it can only be resolved at runtime).
func checkDurations(name string, b []byte) []Finding {
	var out []Finding
	seen := map[string]bool{}
	for _, m := range durationField.FindAllStringSubmatch(string(b), -1) {
		field, val := m[1], m[2]
		if strings.Contains(val, "$(") || allowedDurations[val] {
			continue
		}
		key := field + "=" + val
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Finding{High, name, field,
			field + " " + q(val) + " is not an allowed interval (use 5s, 10s, 15s, 30s, 1m, 2m, 3m)"})
	}
	return out
}

func q(s string) string { return "\"" + s + "\"" }
