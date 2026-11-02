package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveReferencesAndRejectCyclesOrEscape(t *testing.T) {
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "schema.json")
	if err := os.WriteFile(path, []byte(`{"definitions":{"a/b":{"type":"string","description":"omit"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	l := loader{root, map[string]any{}, map[string]string{}}
	got, err := l.resolve(map[string]any{"$ref": "schema.json#/definitions/a~1b", "maxLength": json.Number("38")}, path, nil)
	if err != nil || object(got)["type"] != "string" || object(got)["maxLength"] != json.Number("38") {
		t.Fatalf("reference/override failed: %v %v", got, err)
	}
	if _, ok := object(got)["description"]; ok {
		t.Fatal("documentation leaked into schema")
	}
	if len(l.digests) != 1 {
		t.Fatal("source digest absent")
	}
	if err := os.WriteFile(filepath.Join(root, "cycle.json"), []byte(`{"$ref":"cycle.json"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := l.resolve(map[string]any{"$ref": "cycle.json"}, path, nil); err == nil {
		t.Fatal("recursive schema accepted")
	}
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(root, outside)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.resolve(map[string]any{"$ref": rel}, path, nil); err == nil {
		t.Fatal("out-of-tree schema accepted")
	}
}
