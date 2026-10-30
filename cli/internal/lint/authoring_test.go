package lint

import (
	"encoding/json"
	"testing"
)

func TestCatalogPrefixBoundaryAndInference(t *testing.T) {
	for _, technical := range []any{"wxeval", "abcdefghij", "weathereval", "", nil, 42} {
		body, _ := json.Marshal(map[string]any{"name": "catalog", "technicalName": technical})
		for _, filename := range []string{"catalog.catalog.json", "catalog.json"} {
			findings := CheckFile(filename, body)
			valid := technical == "wxeval" || technical == "abcdefghij"
			if hasMessage(findings, High, "1-10 characters") == valid {
				t.Fatalf("%s prefix=%v: %v", filename, technical, findings)
			}
		}
	}
	if fs := CheckFileMode("catalog.catalog.json", []byte(`{"name":"catalog"}`), true); len(fs) != 0 {
		t.Fatalf("catalog read-back rejected: %v", fs)
	}
}

func TestRetryLogicServiceEnum(t *testing.T) {
	for _, logic := range []any{"FIXED", "INCREMENTAL", "CONSTANT", 1, nil} {
		doc := validCommand()
		ex := doc["configuration"].(map[string]any)["executors"].([]any)[0].(map[string]any)
		ex["autoRetry"] = map[string]any{"logic": logic, "delay": "5s"}
		valid := logic == "FIXED" || logic == "INCREMENTAL" || logic == nil
		if hasMessage(findingsFor(t, doc), High, "retry logic") == valid {
			t.Fatalf("logic=%v", logic)
		}
	}
}

func TestSerializedCollectionBoundsWarning(t *testing.T) {
	for _, kind := range []string{"array", "object", "string"} {
		doc := validCommand()
		doc["inputKeys"] = map[string]any{"items": map[string]any{"type": kind, "sensitive": false, "maxSize": 5}}
		if hasMessage(findingsFor(t, doc), Medium, "serialized JSON") != (kind != "string") {
			t.Fatalf("kind=%s", kind)
		}
		body, _ := json.Marshal(map[string]any{"name": "Fixture", "catalog": "c", "version": 1, "keys": doc["inputKeys"]})
		if hasMessage(CheckFile("Fixture.input.json", body), Medium, "serialized JSON") != (kind != "string") {
			t.Fatalf("Input kind=%s", kind)
		}
	}
}

func TestOwnCounterInputAndPriorOutput(t *testing.T) {
	for _, value := range []string{"$(.call.retriesCount)", "$(.call.repetitionsCount)", "$(.previous.retriesCount)", "$(.call.current.item)", ".call.retriesCount"} {
		doc := validCommand()
		ex := doc["configuration"].(map[string]any)["executors"].([]any)[0].(map[string]any)
		ex["input"] = map[string]any{"message": value}
		want := value == "$(.call.retriesCount)" || value == "$(.call.repetitionsCount)"
		if hasMessage(findingsFor(t, doc), Medium, "own retry/repeat") != want {
			t.Fatalf("value=%s", value)
		}
	}
}
