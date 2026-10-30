package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Cleanup removes only IDs recorded by this run. Reused catalogs are retained;
// a catalog created externally by this run needs its saved HTTP 201 receipt.
func (h *harness) cleanup() (err error) {
	if err = h.connect(); err != nil {
		return err
	}
	p := probe{h}
	defer func() {
		if r := recover(); r != nil {
			if f, ok := r.(probeFailure); ok {
				err = f.err
			} else {
				panic(r)
			}
		}
	}()
	byKind := map[string]map[string]bool{}
	add := func(kind, id string) {
		if id == "" {
			return
		}
		if byKind[kind] == nil {
			byKind[kind] = map[string]bool{}
		}
		byKind[kind][id] = true
	}
	for _, r := range h.spec.Resources {
		if r.Resource != "catalogs" || h.state.Catalog == "" {
			add(r.Resource, h.state.Resources[r.File])
		}
	}
	add("scheduled-executions", h.state.Resources["schedule"])
	for key, id := range h.state.Resources {
		parts := strings.Split(key, "|")
		if len(parts) == 3 && parts[0] == "feature-resource" {
			add(parts[1], id)
		}
	}
	var receipt map[string]any
	if e := read(filepath.Join(h.dir, "catalog-created.json"), &receipt); e == nil {
		body := object(receipt["body"])
		if receipt["status"] == float64(201) && str(object(body["tags"])["evaluation"]) == h.state.Run && str(body["id"]) == h.state.Catalog {
			add("catalogs", h.state.Catalog)
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	// Disable and remove entry points first so no schedule or webhook can create
	// another run while execution and content cleanup is in progress.
	// get --response-envelope returns exit 4 for 404; use status-only in cleanup
	// before deletion so successful previous cleanup is safely resumable.
	remove := func(kind, id string) {
		safe := strings.NewReplacer(":", "_", "/", "_").Replace(id)
		label := "cleanup-" + kind + "-" + safe
		v := p.call(label+"-status", []string{kind, "get", id, "--status-only"}, nil)
		if v == float64(404) || strings.TrimSpace(fmt.Sprint(v)) == "404" {
			return
		}
		p.require(v == float64(200) || strings.TrimSpace(fmt.Sprint(v)) == "200", "Cannot establish resource state before cleanup: "+id)
		p.call(label, []string{kind, "delete", id}, nil)
		v = p.call(label+"-absent", []string{kind, "get", id, "--status-only"}, nil)
		p.require(v == float64(404) || strings.TrimSpace(fmt.Sprint(v)) == "404", "Deletion absence not established: "+id)
	}
	for _, kind := range []string{"scheduled-executions", "webhooks", "mcp-servers"} {
		ids := []string{}
		for id := range byKind[kind] {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			remove(kind, id)
		}
	}
	// Keep completed execution evidence before deletion; FAILED is active and must
	// be aborted first. No unrelated execution is selected by a broad status query.
	v := p.call("cleanup-execution-inventory", []string{"executions", "list", "--tag", "evaluation=" + h.state.Run, "--limit", "1000", "--full"}, nil)
	executions, validInventory := v.([]any)
	p.require(validInventory, "Execution inventory has an unexpected response shape")
	p.require(len(executions) < 1000, "Cleanup inventory may be truncated")
	parents := map[string]bool{}
	parentFile := filepath.Join(h.dir, "cleanup-store-parents.json")
	if e := read(parentFile, &parents); e != nil && !os.IsNotExist(e) {
		return e
	}
	for _, item := range executions {
		m := object(item)
		if str(m["commandId"]) == h.state.Resources["HttpControl.command.json"] {
			parents[str(m["id"])] = true
		}
	}
	p.must(write(parentFile, parents))
	for _, item := range executions {
		m := object(item)
		id := str(m["id"])
		command := str(m["commandId"])
		p.require(byKind["commands"][command], "Execution with run tag belongs to an unrecorded command; inspect before deleting: "+id)
		if _, storeDone := h.state.Results["cleanup-store"]; !storeDone && command == p.id("CleanupStore") {
			continue
		}
		status := str(m["status"])
		if status == "RUNNING" {
			p.action("cleanup-pause-"+id, id, "PAUSE", "")
			p.wait("cleanup-paused-"+id, id, "PAUSED")
			status = "PAUSED"
		}
		if status != "FINISHED" && status != "ABORTED" && status != "SUSPENDED" {
			p.action("cleanup-abort-"+id, id, "ABORT", "")
			p.wait("cleanup-aborted-"+id, id, "ABORTED")
		}
		remove("executions", id)
	}
	// Remove only the exact listener markers and counters belonging to this run.
	// Persist the parent inventory before deleting executions, for resumable cleanup.
	if _, done := h.state.Results["cleanup-store"]; !done && len(parents) > 0 {
		for _, id := range []string{"store-sapcp:DeleteValue:1", "store-sapcp:GetValue:1"} {
			p.call("cleanup-contract-"+strings.Split(id, ":")[1], []string{"commands", "contract", id}, nil)
		}
		d := p.command("CleanupStore", []any{step("remove", "store-sapcp:DeleteValue:1", map[string]any{"key": "$(.execution.input.key)"}), step("inspect", "store-sapcp:GetValue:1", map[string]any{"key": "$(.execution.input.key)", "failOnMissingKey": "false"})}, map[string]any{"empty": "$(.inspect.output.value == null or .inspect.output.value == \"\")"})
		d["inputKeys"] = map[string]any{"key": map[string]any{"type": "string", "description": "Exact test-only runtime storage key.", "sensitive": false, "required": true}}
		id := p.create("commands", "cleanup-store-command", d)
		add("commands", id)
		ids := []string{}
		for parent := range parents {
			ids = append(ids, parent)
		}
		sort.Strings(ids)
		for _, parent := range ids {
			for _, kind := range []string{"counter", "event"} {
				label := "cleanup-store-" + kind + "-" + parent
				eid := p.trigger(label, id, map[string]any{"key": "skills-eval/" + h.state.Run + "/" + kind + "/" + parent}, nil)
				p.wait(label, eid, "FINISHED")
				p.require(p.output(label, eid)["empty"] == "true", "Store cleanup verification failed")
			}
		}
		p.must(h.record("cleanup-store", "PASS", "Exact test counter and listener keys removed and checked for empty values"))
		// Fresh cleanup executions are also scoped and recorded; remove their history.
		for key, eid := range h.state.Resources {
			if strings.HasPrefix(key, "feature-execution|cleanup-store-") {
				remove("executions", eid)
			}
		}
	}
	remaining, valid := p.call("cleanup-executions-final", []string{"executions", "list", "--tag", "evaluation=" + h.state.Run, "--limit", "1000"}, nil).([]any)
	p.require(valid && len(remaining) == 0, "Run executions remain after cleanup")
	for _, kind := range []string{"commands", "inputs"} {
		ids := []string{}
		for id := range byKind[kind] {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			remove(kind, id)
		}
	}
	for id := range byKind["catalogs"] {
		status := p.call("cleanup-catalog-status-"+id, []string{"catalogs", "get", id, "--status-only"}, nil)
		if status == float64(404) {
			continue
		}
		for _, kind := range []string{"commands", "inputs"} {
			items, valid := p.call("cleanup-empty-"+kind+"-"+id, []string{kind, "list", "--catalog", id}, nil).([]any)
			p.require(valid && len(items) == 0, "Catalog has unrecorded content or an unknown inventory shape; refusing deletion")
		}
		remove("catalogs", id)
	}
	return h.record("cleanup", "PASS", "Recorded schedules, webhooks, MCP servers, executions, Commands, Inputs and owned empty catalogs removed; absence verified")
}
