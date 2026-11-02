package lint

import (
	"encoding/json"
	"testing"
)

func validCommand() map[string]any {
	return map[string]any{"name": "CallApi", "catalog": "c", "version": 1, "inputKeys": map[string]any{}, "outputKeys": map[string]any{}, "configuration": map[string]any{"executors": []any{map[string]any{"alias": "call", "execute": "http-sapcp:HttpRequest:1", "input": map[string]any{}}}}}
}
func findingsFor(t *testing.T, doc map[string]any) []Finding {
	t.Helper()
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return CheckFile("CallApi.command.json", b)
}

func TestRequiredStructureAndReadBackMode(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `[]`, `{"name":"CallApi","catalog":"c","version":1,"configuration":null}`} {
		if len(CheckFile("CallApi.command.json", []byte(body))) == 0 {
			t.Fatalf("accepted %s", body)
		}
	}
	body := []byte(`{"name":"CallApi","catalog":"c","version":1,"inputKeys":{},"outputKeys":{},"configuration":null,"tags":{"autopi:status":"released"}}`)
	if fs := CheckFileMode("CallApi.command.json", body, true); len(fs) != 0 {
		t.Fatalf("atomic read-back rejected: %v", fs)
	}
	if fs := findingsFor(t, validCommand()); len(fs) != 0 {
		t.Fatalf("valid authored command rejected: %v", fs)
	}
}
func TestOutputSensitivityAndMetadataNames(t *testing.T) {
	doc := validCommand()
	doc["inputKeys"] = map[string]any{"tokenUrl": map[string]any{"type": "string", "sensitive": false}}
	if fs := findingsFor(t, doc); len(fs) != 0 {
		t.Fatalf("tokenUrl false positive: %v", fs)
	}
	for _, name := range []string{"accessToken", "authorizationHeader", "serviceKey", "clientCert"} {
		doc["outputKeys"] = map[string]any{name: map[string]any{"type": "string", "sensitive": false}}
		if fs := findingsFor(t, doc); !hasMessage(fs, Critical, "must set") {
			t.Fatalf("missed %s: %v", name, fs)
		}
	}
}
func TestWireAndContextualValidation(t *testing.T) {
	doc := validCommand()
	ex := doc["configuration"].(map[string]any)["executors"].([]any)[0].(map[string]any)
	ex["input"] = map[string]any{"interval": "1h", "delay": "3s", "script": "echo '$(.x | type)'"}
	doc["description"] = "Documentation: .x | type; interval 1h"
	if fs := findingsFor(t, doc); len(fs) != 0 {
		t.Fatalf("data/prose mistaken for control: %v", fs)
	}
	ex["autoRetry"] = map[string]any{"delay": "3s"}
	if fs := findingsFor(t, doc); !hasMessage(fs, High, "not an allowed interval") {
		t.Fatal("invalid control duration missed")
	}
	ex["autoRetry"] = map[string]any{"delay": 5}
	if fs := findingsFor(t, doc); !hasMessage(fs, High, "duration string") {
		t.Fatal("native control duration missed")
	}
	ex["input"] = map[string]any{"n": 42}
	if fs := findingsFor(t, doc); !hasMessage(fs, High, "wire value") {
		t.Fatal("native executor value missed")
	}
	doc["inputKeys"] = map[string]any{"n": map[string]any{"type": "number", "sensitive": false, "defaultValue": 42}}
	if fs := findingsFor(t, doc); !hasMessage(fs, High, "defaultValue must") {
		t.Fatal("native default missed")
	}
	doc = validCommand()
	doc["outputKeys"] = map[string]any{"weird": map[string]any{"type": map[string]any{}}}
	if fs := findingsFor(t, doc); len(fs) == 0 {
		t.Fatal("object type not rejected")
	}
}
func TestEveryURLSubstitutionChecked(t *testing.T) {
	doc := validCommand()
	ex := doc["configuration"].(map[string]any)["executors"].([]any)[0].(map[string]any)
	ex["input"] = map[string]any{"url": "https://h/$(.execution.input.name | toPercentEncoded)?q=$(.execution.input.query)"}
	if fs := findingsFor(t, doc); !hasMessage(fs, Medium, "URL substitution") {
		t.Fatal("partial encoding accepted")
	}
	ex["input"] = map[string]any{"url": "https://h/$(.execution.input.name | toPercentEncoded)?q=$(.execution.input.query | toUrlEncoded)"}
	if fs := findingsFor(t, doc); len(fs) != 0 {
		t.Fatalf("fully encoded URL flagged: %v", fs)
	}
}
func TestTagValuesAndInputWire(t *testing.T) {
	for _, tags := range []string{`{"feature:priority":"urgent"}`, `{"feature:autoAbort":"169"}`, `{"feature:logs":"true"}`, `{"feature:retentionPeriod":"0"}`, `{"team":42}`} {
		if fs := CheckFile("Cred.input.json", []byte(`{"name":"Cred","catalog":"c","version":1,"keys":{},"tags":`+tags+`}`)); len(fs) == 0 {
			t.Fatalf("invalid tags accepted %s", tags)
		}
	}
	if fs := CheckFile("Cred.input.json", []byte(`{"name":"Cred","catalog":"c","version":1,"keys":{},"values":{"n":42}}`)); !hasMessage(fs, High, "wire value") {
		t.Fatal("Input native value missed")
	}
}

func TestDocumentedTypeFunctionIsAllowed(t *testing.T) {
	doc := validCommand()
	ex := doc["configuration"].(map[string]any)["executors"].([]any)[0].(map[string]any)
	ex["input"] = map[string]any{"value": "$(.execution.input.x | type)"}
	if fs := findingsFor(t, doc); len(fs) != 0 {
		t.Fatalf("documented type filter rejected: %v", fs)
	}
	ex["input"] = map[string]any{"value": "$(.execution.input.x | isBool)"}
	if fs := findingsFor(t, doc); !hasMessage(fs, Medium, "does not exist") {
		t.Fatal("unsupported filter accepted")
	}
}
