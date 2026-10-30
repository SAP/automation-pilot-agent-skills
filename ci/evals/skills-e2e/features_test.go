package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFeatureMutationResumptionDoesNotReplay(t *testing.T) {
	h := failureHarness(t)
	calls := 0
	h.runCommand = func(c *exec.Cmd) error { calls++; _, err := fmt.Fprint(c.Stdout, "{\"id\":\"created\"}"); return err }
	p := probe{h}
	for range 2 {
		v := p.call("create-once", []string{"commands", "create", "-f", "-"}, map[string]any{})
		if object(v)["id"] != "created" {
			t.Fatal(v)
		}
	}
	if calls != 1 {
		t.Fatalf("mutation replayed %d times", calls)
	}
}
func TestFeatureUnresolvedMutationStopsBeforeCLI(t *testing.T) {
	h := failureHarness(t)
	h.state.Attempts["feature-trigger-once"] = "submitted"
	h.runCommand = func(c *exec.Cmd) error { t.Fatal("ambiguous trigger replayed"); return nil }
	p := probe{h}
	defer func() {
		if _, ok := recover().(probeFailure); !ok {
			t.Fatal("expected ambiguity failure")
		}
	}()
	p.call("trigger-once", []string{"executions", "trigger", "-f", "-"}, map[string]any{})
}
func TestFeatureChangedMutationCannotReuseReceipt(t *testing.T) {
	h := failureHarness(t)
	calls := 0
	h.runCommand = func(c *exec.Cmd) error { calls++; _, err := fmt.Fprint(c.Stdout, "{\"id\":\"created\"}"); return err }
	p := probe{h}
	p.call("create", []string{"commands", "create", "-f", "-"}, map[string]any{"name": "First"})
	defer func() {
		if _, ok := recover().(probeFailure); !ok {
			t.Fatal("changed mutation accepted")
		}
		if calls != 1 {
			t.Fatal("changed mutation sent")
		}
	}()
	p.call("create", []string{"commands", "create", "-f", "-"}, map[string]any{"name": "Second"})
}
func TestFeatureEvidenceWriteFailurePreservesUnresolvedSubmission(t *testing.T) {
	h := failureHarness(t)
	h.runCommand = func(c *exec.Cmd) error {
		if err := write(filepath.Join(h.dir, "mutations"), map[string]any{}); err != nil {
			return err
		}
		_, err := fmt.Fprint(c.Stdout, "{\"id\":\"accepted\"}")
		return err
	}
	// An invalid mutation-cache path fails after the request and evidence write.
	p := probe{h}
	defer func() {
		if _, ok := recover().(probeFailure); !ok {
			t.Fatal("expected persistence failure")
		}
		if h.state.Attempts["feature-create-once"] != "submitted" {
			t.Fatal("ambiguity lost")
		}
	}()
	p.call("create-once", []string{"commands", "create", "-f", "-"}, map[string]any{})
}
