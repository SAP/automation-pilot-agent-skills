package lint

import (
	"encoding/json"
	"strings"
	"testing"
)

func bindingFixture(t *testing.T) ([]byte, BindingMetadata, []byte) {
	t.Helper()
	body := []byte(`{"name":"resource-lookup","mcpTools":[{"name":"find_resource","commandId":"ops-TENANT:FindResource:1","inputReferences":["ops-TENANT:ApiConfiguration:1"],"readOnlyHint":true,"openWorldHint":true}]}`)
	var metadata BindingMetadata
	err := json.Unmarshal([]byte(`{"commands":{"ops-TENANT:FindResource:1":{"inputKeys":{"resourceName":{"type":"string","required":true},"apiConfig":{"type":"object","required":true},"apiToken":{"type":"string","required":true,"sensitive":true}}}},"inputs":{"ops-TENANT:ApiConfiguration:1":{"keys":{"apiConfig":{"type":"object"},"apiToken":{"type":"string","sensitive":true}},"values":{"apiToken":"fixture-secret-value"}}}}`), &metadata)
	if err != nil {
		t.Fatal(err)
	}
	client := []byte(`{"result":{"tools":[{"name":"find_resource","inputSchema":{"type":"object","properties":{"resourceName":{"type":"string"}},"required":["resourceName"]},"annotations":{"readOnlyHint":true,"openWorldHint":true}},{"name":"get_execution","inputSchema":{"type":"object"}}]}}`)
	return body, metadata, client
}

func TestMCPBoundClientContractAndHiddenValues(t *testing.T) {
	body, metadata, client := bindingFixture(t)
	report := CheckMCPBindings("server.mcp.json", body, metadata, client)
	if len(report.Findings) != 0 {
		t.Fatalf("valid binding: %v", report.Findings)
	}
	if !sameNames(report.Tools[0].CallerKeys, []string{"resourceName"}) {
		t.Fatalf("caller keys: %v", report.Tools)
	}
	encoded, _ := json.Marshal(report)
	if strings.Contains(string(encoded), "fixture-secret-value") {
		t.Fatal("Input values leaked")
	}
	wrapped, _ := json.Marshal(map[string]any{"exitCode": 0, "body": json.RawMessage(client)})
	if report := CheckMCPBindings("server.mcp.json", body, metadata, wrapped); len(report.Findings) != 0 {
		t.Fatalf("evidence envelope: %v", report.Findings)
	}
}

func TestMCPBindingMismatchAndMissingMetadata(t *testing.T) {
	for _, scenario := range []string{"overlap", "type", "sensitivity", "missingCommand", "missingInput", "unbound", "unused"} {
		t.Run(scenario, func(t *testing.T) {
			body, metadata, _ := bindingFixture(t)
			const inputID = "ops-TENANT:ApiConfiguration:1"
			switch scenario {
			case "overlap":
				body = []byte(strings.Replace(string(body), `"ops-TENANT:ApiConfiguration:1"]`, `"ops-TENANT:ApiConfiguration:1","ops-TENANT:ApiConfiguration:1"]`, 1))
			case "type":
				key := metadata.Inputs[inputID].Keys["apiToken"]
				key.Type = "number"
				metadata.Inputs[inputID].Keys["apiToken"] = key
			case "sensitivity":
				key := metadata.Inputs[inputID].Keys["apiToken"]
				key.Sensitive = nil
				metadata.Inputs[inputID].Keys["apiToken"] = key
			case "missingCommand":
				metadata.Commands = nil
			case "missingInput":
				metadata.Inputs = nil
			case "unbound":
				body = []byte(strings.Replace(string(body), `["ops-TENANT:ApiConfiguration:1"]`, `[]`, 1))
			case "unused":
				metadata.Inputs[inputID].Keys["unused"] = InputKey{Type: "string"}
			}
			report := CheckMCPBindings("server.mcp.json", body, metadata, nil)
			if len(report.Findings) == 0 {
				t.Fatal("mismatch not reported")
			}
			if scenario == "unused" && report.Findings[0].Severity != Medium {
				t.Fatal("unused key should be advisory")
			}
		})
	}
}

func TestMCPClientLeakageRequirednessTypeAndHints(t *testing.T) {
	for _, scenario := range []string{"leak", "required", "type", "hints", "disabled", "duplicate", "error", "failedEvidence"} {
		t.Run(scenario, func(t *testing.T) {
			body, metadata, client := bindingFixture(t)
			switch scenario {
			case "leak":
				client = []byte(strings.Replace(string(client), `"properties":{"resourceName":{"type":"string"}}`, `"properties":{"resourceName":{"type":"string"},"apiToken":{"type":"string"}}`, 1))
			case "required":
				client = []byte(strings.Replace(string(client), `"required":["resourceName"]`, `"required":[]`, 1))
			case "type":
				client = []byte(strings.Replace(string(client), `"resourceName":{"type":"string"}`, `"resourceName":{"type":"number"}`, 1))
			case "hints":
				client = []byte(strings.Replace(string(client), `"openWorldHint":true`, `"openWorldHint":false`, 1))
			case "disabled":
				body = []byte(strings.Replace(string(body), `"name":"find_resource"`, `"name":"find_resource","enabled":false`, 1))
			case "duplicate":
				client = []byte(strings.Replace(string(client), `"name":"get_execution"`, `"name":"find_resource"`, 1))
			case "error":
				client = []byte(`{"error":{"message":"fixture-secret-value"}}`)
			case "failedEvidence":
				client = []byte(`{"exitCode":4,"body":{"result":{"tools":[]}}}`)
			}
			report := CheckMCPBindings("server.mcp.json", body, metadata, client)
			if len(report.Findings) == 0 {
				t.Fatal("client mismatch not reported")
			}
			encoded, _ := json.Marshal(report)
			if strings.Contains(string(encoded), "fixture-secret-value") {
				t.Fatal("error body leaked")
			}
		})
	}
}

func TestMCPStructureAndReservedPollingTool(t *testing.T) {
	for _, body := range []string{`null`, `{"name":"server","mcpTools":null}`, `{"name":"server","mcpTools":[{"name":"get_execution","commandId":"c:A:1"}]}`, `{"name":"server","mcpTools":[{"name":"find_resource","commandId":"../commands:A:1"}]}`} {
		if len(CheckFile("server.mcp.json", []byte(body))) == 0 {
			t.Fatalf("invalid definition accepted: %s", body)
		}
	}
	if fs := CheckFile("server.mcp.json", []byte(`{"name":"server","mcpTools":[]}`)); len(fs) != 0 {
		t.Fatalf("empty server rejected: %v", fs)
	}
}
