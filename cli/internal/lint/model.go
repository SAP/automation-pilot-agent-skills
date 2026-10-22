// Package lint implements the mechanical checks that the command-review skill
// previously described in prose: structure, naming, sensitive-flag, and a few
// expression/HTTP heuristics. It runs offline on command/input/catalog JSON.
package lint

import (
	"encoding/json"
	"fmt"
)

// Severity ranks a finding.
type Severity string

const (
	Critical Severity = "CRITICAL"
	High     Severity = "HIGH"
	Medium   Severity = "MEDIUM"
)

// Finding is a single lint result.
type Finding struct {
	Severity Severity `json:"severity"`
	File     string   `json:"file"`
	Path     string   `json:"path,omitempty"`
	Message  string   `json:"message"`
}

// Kind is the detected artifact type.
type Kind string

const (
	KindCommand Kind = "command"
	KindInput   Kind = "input"
	KindCatalog Kind = "catalog"
	KindUnknown Kind = "unknown"
)

// Doc is a loosely-typed view of a content JSON file; Raw keeps the original
// bytes for text heuristics.
//
// Both Keys (input definitions) and InputKeys (command input schemas) are
// JSON objects keyed by the key name — e.g. "keys": {"password": {...}} — not
// arrays, so they unmarshal into maps. The map key is the authoritative key
// name; InputKey carries only the per-key attributes.
type Doc struct {
	Name       string                     `json:"name"`
	Keys       map[string]InputKey        `json:"keys"` // input-definition keys
	OutputKeys map[string]InputKey        `json:"outputKeys"`
	InputKeys  map[string]InputKey        `json:"inputKeys"` // command input schema
	Tags       map[string]json.RawMessage `json:"tags"`      // flat tag map (keys are all we lint)
	Raw        json.RawMessage            `json:"-"`
	Fields     map[string]any             `json:"-"`
}

// InputKey is one key of an input definition or command input schema. The key
// name is the map key under which this value is stored, not a field here.
type InputKey struct {
	Type         string `json:"type"`
	Sensitive    *bool  `json:"sensitive"`
	DefaultValue any    `json:"defaultValue"`
}

// Parse unmarshals the bytes into a Doc and classifies the artifact kind from
// its fields. Callers that know the filename should prefer the extension (see
// kindFromName) and fall back to this classification.
func Parse(b []byte) (*Doc, Kind, error) {
	var d Doc
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, KindUnknown, fmt.Errorf("invalid JSON: %w", err)
	}
	_ = json.Unmarshal(b, &d.Fields)
	if d.Fields == nil {
		return nil, KindUnknown, fmt.Errorf("artifact must be a JSON object")
	}
	d.Raw = append(json.RawMessage(nil), b...)
	return &d, classify(d.Fields), nil
}

func classify(f map[string]any) Kind {
	if f == nil {
		return KindUnknown
	}
	// Commands carry inputKeys/outputKeys/configuration/executors/steps.
	// Check these before the catalog fallback: a command also has a "name",
	// so testing name first would misclassify every command as a catalog.
	for _, k := range []string{"inputKeys", "outputKeys", "configuration", "executors", "steps"} {
		if _, ok := f[k]; ok {
			return KindCommand
		}
	}
	// Inputs carry a "keys" map (and typically "values").
	if _, ok := f["keys"]; ok {
		return KindInput
	}
	// Catalogs have name + description but none of the above.
	if _, ok := f["name"]; ok {
		return KindCatalog
	}
	return KindUnknown
}
