package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type engineJournal struct {
	CandidateDigests map[string]string `json:"candidate_sha256,omitempty"`
	Tenant           string            `json:"tenant"`
	Catalog          string            `json:"catalog"`
	Commands         []string          `json:"commands"`
	Executions       []string          `json:"executions"`
	Pending          string            `json:"pending_mutation,omitempty"`
	CleanupErrors    []string          `json:"cleanup_errors,omitempty"`
}
type engineVerifier struct {
	mu                         sync.Mutex
	callOverride               func(context.Context, []string, any) (any, error)
	repository, runDir, tenant string
	journal                    engineJournal
}

type engineHTTPError struct {
	Status int
	Err    error
}

func (e *engineHTTPError) Error() string { return fmt.Sprintf("engine HTTP %d", e.Status) }
func (e *engineHTTPError) Unwrap() error { return e.Err }

func (v *engineVerifier) save() error {
	return writeJSON(filepath.Join(v.runDir, "engine-journal.json"), v.journal)
}
func (v *engineVerifier) call(ctx context.Context, args []string, body any) (any, error) {
	if v.callOverride != nil {
		return v.callOverride(ctx, args, body)
	}
	command := exec.CommandContext(ctx, filepath.Join(v.repository, "skills/automation-pilot-cli/scripts/autopi"), args...)
	containProcess(command)
	if body != nil {
		command.Stdin = bytes.NewReader(mustJSON(body))
	}
	raw, err := command.Output()
	if err != nil {
		var failure *exec.ExitError
		if errors.As(err, &failure) {
			match := regexp.MustCompile("HTTP ([0-9]{3})").FindSubmatch(failure.Stderr)
			if len(match) > 1 {
				status, _ := strconv.Atoi(string(match[1]))
				return nil, &engineHTTPError{Status: status, Err: err}
			}
		}
		return nil, fmt.Errorf("autopi %s: %w", strings.Join(args[:min(2, len(args))], " "), err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	var out any
	if json.Unmarshal(raw, &out) != nil {
		return nil, fmt.Errorf("invalid CLI result")
	}
	return out, nil
}

// The connection comes exclusively from the Automation Pilot environment.
// expected is an optional identity assertion, never a destination override.
func (v *engineVerifier) resolveTenant(ctx context.Context, expected string) error {
	for _, name := range []string{"AUTOMATION_PILOT_HOSTNAME", "AUTOMATION_PILOT_TOKEN"} {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			return fmt.Errorf("engine connection requires %s", name)
		}
	}
	identity, err := v.call(ctx, []string{"connection", "info"}, nil)
	if err != nil {
		return err
	}
	info, _ := identity.(map[string]any)
	tenant, _ := info["tenantId"].(string)
	if info["connected"] != true || strings.TrimSpace(tenant) == "" {
		return fmt.Errorf("configured Automation Pilot connection did not establish tenant identity")
	}
	if expected != "" && tenant != expected {
		return fmt.Errorf("configured tenant differs from the tenant identity assertion")
	}
	v.tenant = tenant
	return nil
}

func (v *engineVerifier) initialize(ctx context.Context) error {
	if err := v.resolveTenant(ctx, v.tenant); err != nil {
		return err
	}
	var err error
	if _, err := os.Stat(filepath.Join(v.runDir, "engine-journal.json")); err == nil {
		return fmt.Errorf("engine journal already exists; reconcile and clean up instead of replaying creates")
	}
	for _, id := range []string{"utils-sapcp:Void:1", "utils-sapcp:ForEach:2"} {
		if _, err = v.call(ctx, []string{"commands", "contract", id}, nil); err != nil {
			return fmt.Errorf("required native contract unavailable: %w", err)
		}
	}
	v.journal = engineJournal{Tenant: v.tenant}
	technical := "ev" + bytesDigest([]byte(v.runDir))[:8]
	v.journal.Pending = "create catalog " + technical
	if err = v.save(); err != nil {
		return err
	}
	result, err := v.call(ctx, []string{"catalogs", "create", "-f", "-"}, map[string]any{"technicalName": technical, "name": "Skill expression evaluation", "description": "Run-owned expression probes; no external system operations", "tags": map[string]string{"evaluation": filepath.Base(v.runDir)}})
	if err != nil {
		return fmt.Errorf("catalog creation requires reconciliation: %w", err)
	}
	object, _ := result.(map[string]any)
	v.journal.Catalog, _ = object["id"].(string)
	if v.journal.Catalog == "" {
		return fmt.Errorf("catalog create omitted ID; reconcile pending mutation")
	}
	v.journal.Pending = ""
	return v.save()
}
func (v *engineVerifier) evaluate(ctx context.Context, c evalCase, artifact any) (map[string]any, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.journal.Pending != "" {
		return nil, fmt.Errorf("unreconciled engine mutation: %s", v.journal.Pending)
	}
	if c.Engine == nil {
		return nil, fmt.Errorf("case has no engine contract")
	}
	expression, _ := artifact.(map[string]any)
	expr, _ := expression["expression"].(string)
	if c.Engine.Kind != "command" && (!strings.HasPrefix(expr, "$(") || !strings.HasSuffix(expr, ")") || len(expr) > 16384) {
		return nil, fmt.Errorf("submit JSON with a bounded Automation Pilot expression")
	}
	name := "Probe" + digest([]any{c.ID, len(v.journal.Commands)})[:12]
	id := v.journal.Catalog + ":" + name + ":1"
	command := map[string]any{"id": id, "name": name, "catalog": v.journal.Catalog, "version": 1, "description": "Evaluate a submitted expression against synthetic payloads", "tags": map[string]string{"evaluation": filepath.Base(v.runDir)}, "inputKeys": map[string]any{"payload": map[string]any{"type": "object", "required": true, "sensitive": false, "description": "Synthetic partition encoded as a JSON object"}}, "outputKeys": map[string]any{"result": map[string]any{"type": c.Engine.ResultType, "sensitive": false, "description": "Result of the submitted expression"}}, "configuration": map[string]any{"values": []any{}, "listeners": []any{}, "output": map[string]string{"result": "$(.evaluate.output.message)"}, "executors": []any{map[string]any{"execute": "utils-sapcp:Void:1", "alias": "evaluate", "input": map[string]string{"message": expr}, "when": nil, "validate": nil, "repeat": nil, "autoRetry": nil, "initialDelay": nil, "pause": nil, "description": nil, "progressMessage": nil, "errorMessages": []any{}}}}}
	if c.Engine.Kind == "command" {
		if err := validateNativeCandidate(artifact); err != nil {
			return nil, err
		}
		command = jsonValue(artifact).(map[string]any)
		command["id"] = id
		command["catalog"] = v.journal.Catalog
		command["name"] = name
	}
	if v.journal.CandidateDigests == nil {
		v.journal.CandidateDigests = map[string]string{}
	}
	v.journal.CandidateDigests[id+"/submitted"] = digest(artifact)
	v.journal.CandidateDigests[id+"/deployed"] = digest(command)
	// Only a trusted Void probe is authored by the evaluator. Candidate JSON cannot
	// inject executors, scripts, URLs, listeners, or test-scope changes.
	candidateDir, err := os.MkdirTemp("", "autopi-probe-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(candidateDir)
	candidatePath := filepath.Join(candidateDir, name+".command.json")
	if err = writeJSON(candidatePath, command); err != nil {
		return nil, err
	}
	if _, err := v.call(ctx, []string{"lint", "--kind", "command", "--output", "json", candidatePath}, nil); err != nil {
		return nil, err
	}
	v.journal.Pending = "create command " + id
	if err := v.save(); err != nil {
		return nil, err
	}
	if _, err := v.call(ctx, []string{"commands", "create", "-f", "-"}, command); err != nil {
		var failure *engineHTTPError
		if errors.As(err, &failure) && failure.Status >= 400 && failure.Status < 500 {
			_, present, inspectErr := v.inspectRecorded(ctx, "commands", id)
			if inspectErr == nil && !present {
				v.journal.Pending = ""
				if saveErr := v.save(); saveErr != nil {
					return nil, saveErr
				}
				if failure.Status == 400 || failure.Status == 422 {
					rejected := map[string]any{}
					for _, p := range c.Engine.Partitions {
						rejected[p.ID] = map[string]any{"candidate_rejected_http_status": failure.Status, "verified_absent": true}
					}
					return rejected, fmt.Errorf("candidate rejected by engine: HTTP %d; exact command verified absent", failure.Status)
				}
			}
		}
		return nil, err
	}
	v.journal.Commands = append(v.journal.Commands, id)
	v.journal.Pending = ""
	if err := v.save(); err != nil {
		return nil, err
	}
	outputs := map[string]any{}
	for _, part := range c.Engine.Partitions {
		v.journal.Pending = "trigger " + id + " partition " + part.ID
		if err := v.save(); err != nil {
			return nil, err
		}
		result, err := v.call(ctx, []string{"executions", "trigger", "-f", "-"}, map[string]any{"commandId": id, "inputValues": part.Values, "tags": map[string]string{"evaluation": filepath.Base(v.runDir), "case": c.ID}})
		if err != nil {
			return nil, fmt.Errorf("ambiguous trigger retained in journal: %w", err)
		}
		execution, _ := result.(map[string]any)
		eid, _ := execution["id"].(string)
		if eid == "" {
			eid, _ = execution["executionId"].(string)
		}
		if eid == "" {
			return nil, fmt.Errorf("trigger omitted execution ID")
		}
		v.journal.Executions = append(v.journal.Executions, eid)
		v.journal.Pending = ""
		if err = v.save(); err != nil {
			return nil, err
		}
		waitCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		for {
			state, err := v.call(waitCtx, []string{"executions", "get", eid}, nil)
			if err != nil {
				cancel()
				return nil, err
			}
			obj, _ := state.(map[string]any)
			status, _ := obj["status"].(string)
			if status == "FINISHED" {
				break
			}
			if status == "FAILED" || status == "ABORTED" {
				cancel()
				outputs[part.ID] = map[string]any{"unexpected_terminal_status": status}
				break
			}
			select {
			case <-waitCtx.Done():
				cancel()
				return nil, fmt.Errorf("probe deadline reached for %s", eid)
			case <-time.After(time.Second):
			}
		}
		cancel()
		if _, failed := outputs[part.ID]; failed {
			continue
		}
		result, err = v.call(ctx, []string{"executions", "output", eid}, nil)
		if err != nil {
			return nil, err
		}
		envelope, _ := result.(map[string]any)
		out, _ := envelope["values"].(map[string]any)
		outputKey := c.Engine.OutputKey
		if outputKey == "" {
			outputKey = "result"
		}
		raw, present := out[outputKey]
		if !present {
			outputs[part.ID] = map[string]any{"missing_persisted_output": true}
			continue
		}
		if c.Engine.ResultType != "string" {
			value, ok := raw.(string)
			if !ok {
				outputs[part.ID] = map[string]any{"invalid_wire_output": raw}
				continue
			}
			var parsed any
			if json.Unmarshal([]byte(value), &parsed) != nil {
				outputs[part.ID] = map[string]any{"invalid_wire_output": value}
				continue
			}
			raw = parsed
		}
		outputs[part.ID] = raw
	}
	return outputs, nil
}
func validateNativeCandidate(artifact any) error {
	object, ok := artifact.(map[string]any)
	if !ok {
		return fmt.Errorf("native candidate must be a command object")
	}
	config, ok := object["configuration"].(map[string]any)
	if !ok {
		return fmt.Errorf("native candidate has no configuration")
	}
	if values, _ := config["values"].([]any); len(values) > 0 {
		return fmt.Errorf("engine acceptance excludes external Input bindings")
	}
	if listeners, _ := config["listeners"].([]any); len(listeners) > 0 {
		return fmt.Errorf("engine acceptance excludes listeners")
	}
	steps, _ := config["executors"].([]any)
	if len(steps) == 0 || len(steps) > 8 {
		return fmt.Errorf("native probe executor bound exceeded")
	}
	for _, value := range steps {
		step, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid native step")
		}
		if step["execute"] != "utils-sapcp:Void:1" && step["execute"] != "utils-sapcp:ForEach:2" {
			return fmt.Errorf("native acceptance permits only Void and sequential ForEach")
		}
		if step["execute"] == "utils-sapcp:ForEach:2" {
			input, _ := step["input"].(map[string]any)
			if input["command"] != "utils-sapcp:Void:1" {
				return fmt.Errorf("iterator child must be fixed to Void")
			}
		}
		for _, field := range []string{"autoRetry", "repeat", "initialDelay", "pause"} {
			if step[field] != nil {
				return fmt.Errorf("native acceptance excludes delays, repeats, retries and pauses")
			}
		}
	}
	return nil
}
func (v *engineVerifier) cleanup(ctx context.Context) error {
	if v.journal.Tenant == "" {
		return fmt.Errorf("engine journal has no tenant identity; cleanup cannot verify ownership")
	}
	if err := v.resolveTenant(ctx, v.journal.Tenant); err != nil {
		return err
	}
	if v.journal.Pending != "" {
		return fmt.Errorf("manual reconciliation required before cleanup: %s", v.journal.Pending)
	}
	if v.journal.Catalog == "" {
		return nil
	}
	var err error
	failed := []string{}
	remainingExecutions := []string{}
	for _, id := range v.journal.Executions {
		state, present, err := v.inspectRecorded(ctx, "executions", id)
		if err != nil {
			failed = append(failed, "inspect execution "+id)
			remainingExecutions = append(remainingExecutions, id)
			continue
		}
		if !present {
			continue
		}
		status, _ := state["status"].(string)
		if status != "FINISHED" && status != "FAILED" && status != "ABORTED" {
			v.journal.Pending = "abort execution " + id
			if err = v.save(); err != nil {
				return err
			}
			if _, err = v.call(ctx, []string{"executions", "action", id, "--type", "ABORT", "--reason", "bounded evaluation cleanup"}, nil); err != nil {
				return fmt.Errorf("abort requires reconciliation; journal retained")
			}
			v.journal.Pending = ""
			if err = v.save(); err != nil {
				return err
			}
			for attempt := 0; attempt < 20; attempt++ {
				state, _, err = v.inspectRecorded(ctx, "executions", id)
				if err != nil {
					break
				}
				status, _ = state["status"].(string)
				if status == "ABORTED" || status == "FINISHED" || status == "FAILED" {
					break
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Second):
				}
			}
			if err != nil || !(status == "ABORTED" || status == "FINISHED" || status == "FAILED") {
				failed = append(failed, "execution still active "+id)
				remainingExecutions = append(remainingExecutions, id)
				continue
			}
		}
		if _, err = v.call(ctx, []string{"executions", "delete", id}, nil); err != nil {
			failed = append(failed, "delete execution "+id)
			remainingExecutions = append(remainingExecutions, id)
		}
	}
	v.journal.Executions = remainingExecutions
	if err = v.save(); err != nil {
		return err
	}
	if len(failed) == 0 {
		remainingCommands := []string{}
		for i := len(v.journal.Commands) - 1; i >= 0; i-- {
			id := v.journal.Commands[i]
			_, present, inspectErr := v.inspectRecorded(ctx, "commands", id)
			if inspectErr != nil {
				failed = append(failed, "inspect command "+id)
				remainingCommands = append(remainingCommands, id)
				continue
			}
			if !present {
				continue
			}
			if _, err = v.call(ctx, []string{"commands", "delete", id}, nil); err != nil {
				failed = append(failed, "delete command "+id)
				remainingCommands = append(remainingCommands, id)
			}
		}
		v.journal.Commands = remainingCommands
		if err = v.save(); err != nil {
			return err
		}
	}
	if len(failed) == 0 {
		_, present, inspectErr := v.inspectRecorded(ctx, "catalogs", v.journal.Catalog)
		if inspectErr != nil {
			failed = append(failed, "inspect owned catalog")
		} else if present {
			if _, err = v.call(ctx, []string{"catalogs", "delete", v.journal.Catalog}, nil); err != nil {
				failed = append(failed, "delete owned empty catalog")
			}
		}
		if len(failed) == 0 {
			v.journal.Catalog = ""
		}
	}
	v.journal.CleanupErrors = failed
	_ = v.save()
	if len(failed) > 0 {
		return fmt.Errorf("engine cleanup incomplete; retained journal")
	}
	return nil
}
func (v *engineVerifier) inspectRecorded(ctx context.Context, resource, id string) (map[string]any, bool, error) {
	status, err := v.call(ctx, []string{resource, "get", id, "--status-only"}, nil)
	if err != nil {
		return nil, false, err
	}
	code, ok := status.(float64)
	if !ok {
		return nil, false, fmt.Errorf("invalid status-only response")
	}
	if code == 404 {
		return nil, false, nil
	}
	if code != 200 {
		return nil, false, fmt.Errorf("inspection returned HTTP %.0f", code)
	}
	value, err := v.call(ctx, []string{resource, "get", id}, nil)
	if err != nil {
		return nil, false, err
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, false, fmt.Errorf("invalid read-back")
	}
	return object, true, nil
}
