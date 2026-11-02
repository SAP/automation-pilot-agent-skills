package lint

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type MCPServer struct {
	Name     string    `json:"name"`
	MCPTools []MCPTool `json:"mcpTools"`
}

type MCPTool struct {
	Name            string   `json:"name"`
	CommandID       string   `json:"commandId"`
	InputReferences []string `json:"inputReferences"`
	Enabled         *bool    `json:"enabled"`
	ReadOnly        *bool    `json:"readOnlyHint"`
	Destructive     *bool    `json:"destructiveHint"`
	Idempotent      *bool    `json:"idempotentHint"`
	OpenWorld       *bool    `json:"openWorldHint"`
}

type BindingContract struct {
	InputKeys map[string]InputKey `json:"inputKeys"`
	Keys      map[string]InputKey `json:"keys"`
}

// BindingMetadata contains contracts only; stored Input values are never used.
type BindingMetadata struct {
	Commands map[string]BindingContract `json:"commands"`
	Inputs   map[string]BindingContract `json:"inputs"`
}

type ToolContract struct {
	Tool         string   `json:"tool"`
	InjectedKeys []string `json:"injectedKeys"`
	CallerKeys   []string `json:"callerKeys"`
	RequiredKeys []string `json:"requiredCallerKeys"`
}

type MCPReport struct {
	Findings []Finding      `json:"findings"`
	Tools    []ToolContract `json:"tools"`
}

var toolName = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

func exactContentID(id string) bool {
	return validCommandID(id) && !strings.ContainsAny(id, "/\\ \t\r\n?#")
}

func decodeMCP(file string, body []byte) (MCPServer, []Finding) {
	var server MCPServer
	if json.Unmarshal(body, &server) != nil {
		return server, []Finding{{High, file, "", "MCP definition has invalid field types"}}
	}
	var out []Finding
	if !kebabCase.MatchString(server.Name) {
		out = append(out, Finding{High, file, "name", "server name must be kebab-case"})
	}
	if server.MCPTools == nil || len(server.MCPTools) > 50 {
		out = append(out, Finding{High, file, "mcpTools", "mcpTools must be an array of at most 50 tools"})
	}
	names := map[string]bool{}
	for i, tool := range server.MCPTools {
		prefix := fmt.Sprintf("mcpTools[%d]", i)
		if !toolName.MatchString(tool.Name) || names[tool.Name] || tool.Name == "get_execution" {
			out = append(out, Finding{High, file, prefix + ".name", "tool name must be unique snake_case; get_execution is reserved"})
		}
		names[tool.Name] = true
		if !exactContentID(tool.CommandID) {
			out = append(out, Finding{High, file, prefix + ".commandId", "backing command needs an exact catalog:name:version ID"})
		}
		for _, identity := range tool.InputReferences {
			if !exactContentID(identity) {
				out = append(out, Finding{High, file, prefix + ".inputReferences", "bindings need exact catalog:name:version Input IDs"})
			}
		}
	}
	return server, out
}

func CheckMCPDefinition(file string, body []byte) []Finding {
	_, findings := decodeMCP(file, body)
	return findings
}

// CheckMCPBindings verifies injection from metadata and optionally the captured client schema.
// Findings and summaries contain field names, never stored values or command defaults.
func CheckMCPBindings(file string, body []byte, metadata BindingMetadata, client []byte) MCPReport {
	server, findings := decodeMCP(file, body)
	report := MCPReport{Findings: findings, Tools: []ToolContract{}}
	if len(findings) != 0 {
		return report
	}
	add := func(severity Severity, path, message string) {
		report.Findings = append(report.Findings, Finding{severity, file, path, message})
	}
	expected := map[string]MCPTool{}
	contracts := map[string]ToolContract{}
	for _, tool := range server.MCPTools {
		command, ok := metadata.Commands[tool.CommandID]
		if !ok || command.InputKeys == nil {
			add(High, tool.Name+".commandId", "backing command contract is missing")
			continue
		}
		injected := map[string]bool{}
		for _, identity := range tool.InputReferences {
			input, ok := metadata.Inputs[identity]
			if !ok || input.Keys == nil {
				add(High, tool.Name+".inputReferences", "Input metadata is missing for "+identity)
				continue
			}
			for _, key := range sortedInputKeys(input.Keys) {
				schema := input.Keys[key]
				target, matched := command.InputKeys[key]
				if !matched {
					add(Medium, tool.Name+"."+key, "unused Input key in "+identity)
					continue
				}
				if injected[key] {
					add(High, tool.Name+"."+key, "overlapping Input bindings")
				}
				if schema.Type != target.Type {
					add(High, tool.Name+"."+key, "Input type differs from command type")
				}
				if flagTrue(schema.Sensitive) != flagTrue(target.Sensitive) {
					add(Critical, tool.Name+"."+key, "Input sensitivity differs from command sensitivity")
				}
				injected[key] = true
			}
		}
		summary := ToolContract{Tool: tool.Name, InjectedKeys: []string{}, CallerKeys: []string{}, RequiredKeys: []string{}}
		for _, key := range sortedInputKeys(command.InputKeys) {
			schema := command.InputKeys[key]
			if injected[key] {
				summary.InjectedKeys = append(summary.InjectedKeys, key)
				continue
			}
			summary.CallerKeys = append(summary.CallerKeys, key)
			if flagTrue(schema.Sensitive) {
				add(Critical, tool.Name+"."+key, "bind sensitive caller key through a reusable Input")
			}
			if schema.Required && schema.DefaultValue == nil && schema.DefaultValueFromInput == nil {
				summary.RequiredKeys = append(summary.RequiredKeys, key)
			}
		}
		report.Tools = append(report.Tools, summary)
		contracts[tool.Name] = summary
		if tool.Enabled == nil || *tool.Enabled {
			expected[tool.Name] = tool
		}
	}
	if len(client) != 0 {
		report.Findings = append(report.Findings, checkClientSchema(file, client, expected, contracts, metadata)...)
	}
	if report.Findings == nil {
		report.Findings = []Finding{}
	}
	return report
}

func flagTrue(value *bool) bool { return value != nil && *value }

func sortedInputKeys(keys map[string]InputKey) []string {
	names := make([]string, 0, len(keys))
	for key := range keys {
		names = append(names, key)
	}
	sort.Strings(names)
	return names
}

type clientProperty struct {
	Type  any             `json:"type"`
	Items json.RawMessage `json:"items"`
}

type clientTool struct {
	Name        string `json:"name"`
	InputSchema struct {
		Type       string                    `json:"type"`
		Properties map[string]clientProperty `json:"properties"`
		Required   []string                  `json:"required"`
	} `json:"inputSchema"`
	Annotations map[string]any `json:"annotations"`
}

func clientTools(body []byte) ([]clientTool, error) {
	var envelope struct {
		Body     json.RawMessage `json:"body"`
		ExitCode *int            `json:"exitCode"`
		Error    json.RawMessage `json:"error"`
		Result   json.RawMessage `json:"result"`
		Tools    []clientTool    `json:"tools"`
	}
	if json.Unmarshal(body, &envelope) != nil || (envelope.ExitCode != nil && *envelope.ExitCode != 0) {
		return nil, fmt.Errorf("client schema records an invalid or failed request")
	}
	if len(envelope.Body) != 0 && !bytes.Equal(bytes.TrimSpace(envelope.Body), []byte("null")) {
		if json.Unmarshal(envelope.Body, &envelope) != nil {
			return nil, fmt.Errorf("client schema has an invalid evidence body")
		}
	}
	if len(envelope.Error) != 0 && !bytes.Equal(bytes.TrimSpace(envelope.Error), []byte("null")) && !bytes.Equal(bytes.TrimSpace(envelope.Error), []byte(`""`)) {
		return nil, fmt.Errorf("client schema contains a JSON-RPC error")
	}
	if len(envelope.Result) != 0 {
		if json.Unmarshal(envelope.Result, &envelope) != nil {
			return nil, fmt.Errorf("client schema has an invalid tools/list result")
		}
	}
	if envelope.Tools == nil {
		return nil, fmt.Errorf("client schema has no tools/list array")
	}
	return envelope.Tools, nil
}

func checkClientSchema(file string, body []byte, expected map[string]MCPTool, contracts map[string]ToolContract, metadata BindingMetadata) []Finding {
	var out []Finding
	add := func(path, message string) { out = append(out, Finding{High, file, path, message}) }
	tools, err := clientTools(body)
	if err != nil {
		add("clientSchema", err.Error())
		return out
	}
	actual := map[string]clientTool{}
	for _, tool := range tools {
		if _, exists := actual[tool.Name]; exists {
			add("clientSchema", "duplicate client tool name")
		}
		actual[tool.Name] = tool
	}
	wanted := []string{"get_execution"}
	for name := range expected {
		wanted = append(wanted, name)
	}
	actualNames := make([]string, 0, len(actual))
	for name := range actual {
		actualNames = append(actualNames, name)
	}
	if !sameNames(wanted, actualNames) {
		add("clientSchema", "enabled tool set or generated get_execution differs")
	}
	sort.Strings(wanted)
	for _, name := range wanted {
		tool, ok := expected[name]
		if !ok {
			continue
		}
		visible, ok := actual[name]
		if !ok {
			continue
		}
		summary := contracts[name]
		schema := visible.InputSchema
		if schema.Type != "object" || schema.Properties == nil {
			add(name+".inputSchema", "expected an object schema")
			continue
		}
		properties := make([]string, 0, len(schema.Properties))
		for key := range schema.Properties {
			properties = append(properties, key)
		}
		if !sameNames(summary.CallerKeys, properties) {
			add(name+".inputSchema", fmt.Sprintf("missing caller keys %v; unexpected keys %v", difference(summary.CallerKeys, properties), difference(properties, summary.CallerKeys)))
		}
		if !sameNames(summary.RequiredKeys, schema.Required) {
			add(name+".inputSchema", "required caller keys differ")
		}
		for _, key := range summary.CallerKeys {
			property, exists := schema.Properties[key]
			if !exists {
				continue
			}
			kind := metadata.Commands[tool.CommandID].InputKeys[key].Type
			if !schemaTypeMatches(property.Type, kind) {
				add(name+"."+key, "client type differs from command type")
			}
			items := bytes.TrimSpace(property.Items)
			if kind == "array" && (len(items) == 0 || bytes.Equal(items, []byte("{}")) || bytes.Equal(items, []byte("null"))) {
				out = append(out, Finding{Medium, file, name + "." + key, "client items are unconstrained; use a command-side guard or flat wrapper"})
			}
		}
		for _, hint := range []struct {
			name  string
			value *bool
		}{
			{"readOnlyHint", tool.ReadOnly}, {"destructiveHint", tool.Destructive},
			{"idempotentHint", tool.Idempotent}, {"openWorldHint", tool.OpenWorld},
		} {
			value := hint.value
			if value != nil {
				actualValue, exists := visible.Annotations[hint.name].(bool)
				if !exists || actualValue != *value {
					add(name+"."+hint.name, "client hint differs from definition")
				}
			}
		}
	}
	return out
}

func schemaTypeMatches(value any, expected string) bool {
	if value == expected {
		return true
	}
	if values, ok := value.([]any); ok {
		for _, kind := range values {
			if kind == expected {
				return true
			}
		}
	}
	return false
}

func sameNames(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	a = append([]string(nil), a...)
	b = append([]string(nil), b...)
	sort.Strings(a)
	sort.Strings(b)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func difference(a, b []string) []string {
	seen := map[string]bool{}
	for _, value := range b {
		seen[value] = true
	}
	result := []string{}
	for _, value := range a {
		if !seen[value] {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
