package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func transportFailure(t *testing.T) error {
	t.Helper()
	program, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("test requires sh")
	}
	return exec.Command(program, "-c", "exit 3").Run()
}

func TestTransportReadRetryRetainsBothReceipts(t *testing.T) {
	failure := transportFailure(t)
	h := &harness{dir: t.TempDir(), cli: "unused", ctx: context.Background()}
	h.runCommand = func(cmd *exec.Cmd) error {
		if h.calls == 1 {
			fmt.Fprintln(cmd.Stderr, "request failed: context deadline exceeded")
			return failure
		}
		fmt.Fprintln(cmd.Stdout, `{"status":"FINISHED"}`)
		return nil
	}
	response, err := h.call("execution-state", []string{"executions", "get", "owned-id"}, nil)
	if err != nil || h.calls != 2 || object(response)["status"] != "FINISHED" {
		t.Fatal(response, err, h.calls)
	}
	for name, code := range map[string]int{"0001-execution-state.json": 3, "0002-execution-state-read-retry.json": 0} {
		var receipt map[string]any
		raw, err := os.ReadFile(filepath.Join(h.dir, "evidence", name))
		if err != nil || json.Unmarshal(raw, &receipt) != nil || receipt["exitCode"] != float64(code) {
			t.Fatal(name, err, receipt)
		}
	}
}

func TestTransportFailureNeverReplaysMutationsOrUnknownOperations(t *testing.T) {
	failure := transportFailure(t)
	for _, args := range [][]string{{"commands", "create"}, {"executions", "trigger"}, {"executions", "action", "owned-id"}, {"scheduled-executions", "disable", "owned-id"}, {"mcp", "call"}, {"unknown", "get"}} {
		h := &harness{dir: t.TempDir(), cli: "unused"}
		h.runCommand = func(cmd *exec.Cmd) error { return failure }
		if _, err := h.call("operation", args, nil); err == nil || h.calls != 1 {
			t.Fatal(args, err, h.calls)
		}
	}
}

func TestReadTransportRetryIsBoundedAndHonorsCancellation(t *testing.T) {
	failure := transportFailure(t)
	for _, canceled := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		if canceled {
			cancel()
		}
		h := &harness{dir: t.TempDir(), cli: "unused", ctx: ctx}
		h.runCommand = func(cmd *exec.Cmd) error { return failure }
		_, err := h.call("state", []string{"executions", "get", "owned-id"}, nil)
		cancel()
		want := 2
		if canceled {
			want = 1
		}
		if err == nil || h.calls != want {
			t.Fatal(canceled, err, h.calls)
		}
	}
}
