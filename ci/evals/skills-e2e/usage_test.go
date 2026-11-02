package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelUsageDeltaContainsOnlyCounters(t *testing.T) {
	h := failureHarness(t)
	log := filepath.Join(t.TempDir(), "session.jsonl")
	data := `{"timestamp":"2026-10-30T18:30:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":60,"output_tokens":10}}}}
{"timestamp":"2026-10-30T18:31:00Z","type":"response_item","payload":{"text":"private conversation sentinel"}}
{"timestamp":"2026-10-30T18:32:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":400,"cached_input_tokens":260,"output_tokens":40}}}}
`
	if err := os.WriteFile(log, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	if err := h.measureModelUsage(log, "2026-10-30T18:30:59Z"); err != nil {
		t.Fatal(err)
	}
	var result struct{ UsageDelta map[string]int64 }
	artifact := filepath.Join(h.dir, "model-usage.json")
	if err := read(artifact, &result); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]int64{"input_tokens": 300, "cached_input_tokens": 200, "uncached_input_tokens": 100, "output_tokens": 30} {
		if got := result.UsageDelta[key]; got != want {
			t.Fatalf("%s: got %d, want %d", key, got, want)
		}
	}
	content, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "sentinel") || strings.Contains(string(content), log) {
		t.Fatal("measurement copied non-counter session data")
	}
}

func TestModelUsageRequiresBracketingCounters(t *testing.T) {
	h := failureHarness(t)
	log := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(log, []byte(`{"type":"event_msg","payload":{"type":"other"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := h.measureModelUsage(log, "2026-10-30T18:30:59Z"); err == nil {
		t.Fatal("missing counters reported as measured usage")
	}
}
