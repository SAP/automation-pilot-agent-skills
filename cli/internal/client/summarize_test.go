package client

import (
	"encoding/json"
	"testing"
)

// decode is a small helper to turn Summarize output back into generic JSON.
func decode(t *testing.T, b []byte) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("output is not JSON: %v (%s)", err, b)
	}
	return v
}

func TestSummarizeArrayStripsHeavyKeepsMetadata(t *testing.T) {
	in := []byte(`[
	  {"id":"c:A:1","name":"A","catalog":"c","description":"d","version":1,"tags":{},"inputKeys":{"x":{}},"outputKeys":{},"configuration":{"a":1},"executors":[{"id":"e"}]},
	  {"id":"c:B:1","name":"B","configuration":{},"steps":[1,2,3]}
	]`)
	arr, ok := decode(t, Summarize(in)).([]any)
	if !ok || len(arr) != 2 {
		t.Fatalf("want 2-element array, got %T", arr)
	}
	first := arr[0].(map[string]any)
	for _, keep := range []string{"id", "name", "catalog", "description", "version", "tags"} {
		if _, ok := first[keep]; !ok {
			t.Errorf("metadata key %q was stripped", keep)
		}
	}
	for _, drop := range []string{"inputKeys", "outputKeys", "configuration", "executors"} {
		if _, ok := first[drop]; ok {
			t.Errorf("heavy key %q survived", drop)
		}
	}
	second := arr[1].(map[string]any)
	if _, ok := second["steps"]; ok {
		t.Error("heavy key \"steps\" survived in element 2")
	}
}

func TestSummarizeSingleObject(t *testing.T) {
	in := []byte(`{"name":"A","keys":{"password":{}},"values":{"password":"secret"}}`)
	obj := decode(t, Summarize(in)).(map[string]any)
	if _, ok := obj["name"]; !ok {
		t.Error("name was stripped")
	}
	if _, ok := obj["keys"]; ok {
		t.Error("keys survived")
	}
	if _, ok := obj["values"]; ok {
		t.Error("values survived (would leak secret values)")
	}
}

func TestSummarizeNonJSONPassthrough(t *testing.T) {
	in := []byte("not json at all")
	if got := Summarize(in); string(got) != string(in) {
		t.Fatalf("non-JSON should pass through unchanged, got %q", got)
	}
}

func TestSummarizeEmptyBody(t *testing.T) {
	if got := Summarize(nil); len(got) != 0 {
		t.Fatalf("nil body should pass through, got %q", got)
	}
}

func TestSummarizeScalarPassthrough(t *testing.T) {
	// A top-level JSON scalar (e.g. a count) is not a resource list; leave it.
	in := []byte(`42`)
	if got := Summarize(in); string(got) != "42" {
		t.Fatalf("scalar should pass through, got %q", got)
	}
}
