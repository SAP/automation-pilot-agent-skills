package main

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestCleanupRejectsUnownedOrTruncatedExecutionInventory(t *testing.T) {
	for _, tc := range []struct {
		name      string
		inventory any
		want      string
	}{
		{"foreign-command", []any{map[string]any{"id": "foreign-run", "commandId": "other:Command:1", "status": "RUNNING"}}, "unrecorded command"},
		{"truncated", make([]any, 1000), "may be truncated"},
		{"wrong-shape", map[string]any{}, "unexpected response shape"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := failureHarness(t)
			h.state.Tenant = "Expected"
			h.runCommand = func(c *exec.Cmd) error {
				if c.Args[1] == "connection" {
					return json.NewEncoder(c.Stdout).Encode(map[string]any{"tenantId": "Expected", "permissions": []string{"Read", "Write", "Execute"}})
				}
				if c.Args[1] == "executions" && c.Args[2] == "list" {
					return json.NewEncoder(c.Stdout).Encode(tc.inventory)
				}
				t.Fatalf("unexpected mutation or request: %v", c.Args)
				return nil
			}
			if err := h.cleanup(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q rejection, got %v", tc.want, err)
			}
		})
	}
}

func TestCleanupRetainsReusedCatalogWithoutOwnershipReceipt(t *testing.T) {
	h := failureHarness(t)
	h.state.Tenant = "Expected"
	h.state.Catalog = "reused-catalog"
	h.runCommand = func(c *exec.Cmd) error {
		if c.Args[1] == "connection" {
			return json.NewEncoder(c.Stdout).Encode(map[string]any{"tenantId": "Expected", "permissions": []string{"Read", "Write", "Execute"}})
		}
		if c.Args[1] == "executions" && c.Args[2] == "list" {
			return json.NewEncoder(c.Stdout).Encode([]any{})
		}
		t.Fatalf("reused catalog should remain untouched: %v", c.Args)
		return nil
	}
	if err := h.cleanup(); err != nil {
		t.Fatal(err)
	}
}
