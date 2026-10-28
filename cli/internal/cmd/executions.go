package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
	"github.com/SAP/automation-pilot-agent-skills/cli/internal/domain"
)

func executionsResource() *resource {
	return &resource{
		name:    "executions",
		summary: "trigger, monitor, and control executions",
		verbs: []*verb{
			{"list", "list executions with filters", "GET /executions", execList},
			{"get", "get one execution", "GET /executions/{id}", execGet},
			{"summary", "execution counts by status", "GET /executions/summary", execSummary},
			{"input", "resolved input values", "GET /executions/{id}/input", execInput},
			{"output", "execution output", "GET /executions/{id}/output", execOutput},
			{"logs", "execution logs (paged)", "GET /executions/{id}/logs", execLogs},
			{"trigger", "trigger a command", "POST /executions", execTrigger},
			{"action", "request a public execution action", "POST /executions/{id}/actions", execAction},
			{"bulk-action", "act on up to 50 executions; exit 7 on item failures", "POST /executions/actions", execBulkAction},
			{"actions", "action history", "GET /executions/{id}/actions", execActions},
			{"delete", "delete an execution", "DELETE /executions/{id}", execDelete},
		},
	}
}

func execList(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "executions list")
	var limit string
	var status, commandID, execID, tag, originatorID stringList
	var filterMode string
	var startedAfter, startedBefore string
	var full bool
	fs.StringVar(&limit, "limit", "", "max results")
	fs.Var(&status, "status", "filter by status (repeatable), e.g. FAILED RUNNING PAUSED")
	fs.Var(&commandID, "command-id", "filter by command id (repeatable)")
	fs.Var(&execID, "execution-id", "filter by execution id (repeatable)")
	fs.Var(&originatorID, "originator-id", "filter by initiator (repeatable)")
	fs.StringVar(&filterMode, "filter-mode", "", "identifier matching: exact (default) or contains")
	fs.Var(&tag, "tag", "filter by tag key=value (repeatable)")
	fs.StringVar(&startedAfter, "started-after", "", "epoch-ms, or a relative window like 24h")
	fs.StringVar(&startedBefore, "started-before", "", "epoch-ms, or a relative window like 24h")
	fs.BoolVar(&full, "full", false, "return complete objects instead of the metadata summary")
	setUsage(fs, "GET /executions", "autopi executions list [--status S ...] [--limit N] [--started-after 24h]",
		"autopi executions list --status FAILED --started-after 24h --limit 50")
	if err := parseNoArgs(fs, args); err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	q := url.Values{}
	if limit != "" {
		if err := integerFlag("limit", limit, 1); err != nil {
			return err
		}
		q.Set("limit", limit)
	}
	for _, s := range status {
		if !knownStatus(s) {
			return usagef("invalid execution status %q", s)
		}
		q.Add("status", s)
	}
	for _, id := range commandID {
		q.Add("commandId", id)
	}
	for _, id := range execID {
		q.Add("executionId", id)
	}
	for _, id := range originatorID {
		q.Add("originatorId", id)
	}
	if filterMode != "" {
		q.Set("filterMode", filterMode)
	}
	addTagFilters(q, tag)
	if startedAfter != "" {
		ms, err := epochMillis(startedAfter)
		if err != nil {
			return err
		}
		q.Set("startedAfter", ms)
	}
	if startedBefore != "" {
		ms, err := epochMillis(startedBefore)
		if err != nil {
			return err
		}
		q.Set("startedBefore", ms)
	}
	if q.Get("startedAfter") != "" && q.Get("startedBefore") != "" {
		a, _ := strconv.ParseInt(q.Get("startedAfter"), 10, 64)
		b, _ := strconv.ParseInt(q.Get("startedBefore"), 10, 64)
		if a >= b {
			return usagef("--started-after must precede --started-before")
		}
	}
	return execute(c, cfg, client.Request{Method: "GET", Path: "executions", Query: q, Summarize: !full}, false)
}

// epochMillis accepts a literal epoch-ms string or a Go duration like "24h" and
// returns the corresponding epoch-ms (now - duration) as a string. This removes
// the shell `$(($(date +%s)...))` arithmetic the skills used.
func epochMillis(v string) (string, error) {
	if n, err := strconv.ParseInt(v, 10, 64); err == nil {
		if n < 0 {
			return "", usagef("epoch milliseconds must be non-negative")
		}
		return strconv.FormatInt(n, 10), nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return "", usagef("invalid time %q (want epoch-ms or a duration like 24h)", v)
	}
	if d <= 0 {
		return "", usagef("relative time duration must be positive")
	}
	return strconv.FormatInt(time.Now().Add(-d).UnixMilli(), 10), nil
}

func execGet(c *cmdContext, args []string) error {
	return getVerb(c, args, "executions get", "GET /executions/{id}",
		"autopi executions get abc123", "execution id", "executions")
}

func execSummary(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "executions summary")
	var after, before string
	fs.StringVar(&after, "started-after", "", "epoch-ms or relative window, e.g. 24h")
	fs.StringVar(&before, "started-before", "", "epoch-ms or relative window")
	setUsage(fs, "GET /executions/summary", "autopi executions summary [--started-after 24h]", "autopi executions summary --started-after 24h")
	if err := parseNoArgs(fs, args); err != nil {
		return err
	}
	q, err := timeFilters(after, before)
	if err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	return execute(c, cfg, client.Request{Method: "GET", Path: "executions/summary", Query: q}, false)
}

func timeFilters(after, before string) (url.Values, error) {
	q := url.Values{}
	for key, value := range map[string]string{"startedAfter": after, "startedBefore": before} {
		if value != "" {
			ms, err := epochMillis(value)
			if err != nil {
				return nil, err
			}
			q.Set(key, ms)
		}
	}
	if q.Get("startedAfter") != "" && q.Get("startedBefore") != "" {
		a, _ := strconv.ParseInt(q.Get("startedAfter"), 10, 64)
		b, _ := strconv.ParseInt(q.Get("startedBefore"), 10, 64)
		if a >= b {
			return nil, usagef("--started-after must precede --started-before")
		}
	}
	return q, nil
}

func execInput(c *cmdContext, args []string) error {
	return subResourceGet(c, args, "executions input", "input", "GET /executions/{id}/input",
		"autopi executions input abc123", "execution id", "executions")
}

func execOutput(c *cmdContext, args []string) error {
	return subResourceGet(c, args, "executions output", "output", "GET /executions/{id}/output",
		"autopi executions output abc123", "execution id", "executions")
}

func execActions(c *cmdContext, args []string) error {
	return subResourceGet(c, args, "executions actions", "actions", "GET /executions/{id}/actions",
		"autopi executions actions abc123", "execution id", "executions")
}

func execDelete(c *cmdContext, args []string) error {
	return deleteVerb(c, args, "executions delete", "DELETE /executions/{id}",
		"autopi executions delete abc123", "execution id", "executions")
}

// subResourceGet implements GET {collection}/{id}/{sub}.
func subResourceGet(c *cmdContext, args []string, name, sub, endpoint, example, idLabel, collection string) error {
	fs, g := newFlagSet(c, name)
	setUsage(fs, endpoint, name+" <id>", example)
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	id, err := requireID(pos, idLabel)
	if err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	return execute(c, cfg, client.Request{Method: "GET", Path: client.JoinPath(collection, id, sub)}, false)
}

func execLogs(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "executions logs")
	var page, maxPageSize, executor string
	var messages stringList
	fs.StringVar(&page, "page", "", "page number (0-based)")
	fs.StringVar(&maxPageSize, "max-page-size", "", "page size")
	fs.StringVar(&executor, "executor", "", "fetch logs for a specific executor path")
	fs.Var(&messages, "messages", "log message type (repeatable; requires --executor)")
	setUsage(fs, "GET /executions/{id}/logs[/{executor}]",
		"autopi executions logs <id> [--page 0 --max-page-size 20 | --executor PATH]",
		"autopi executions logs abc123 --page 0 --max-page-size 20")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if executor != "" && (page != "" || maxPageSize != "") {
		return usagef("--executor cannot be combined with paging flags")
	}
	if len(messages) > 0 && executor == "" {
		return usagef("--messages requires --executor")
	}
	if page != "" {
		if err := integerFlag("page", page, 0); err != nil {
			return err
		}
	}
	if maxPageSize != "" {
		if err := integerFlag("max-page-size", maxPageSize, 0); err != nil {
			return err
		}
	}
	id, err := requireID(pos, "execution id")
	if err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	if executor != "" {
		q := url.Values{}
		for _, message := range messages {
			q.Add("messages", message)
		}
		return execute(c, cfg, client.Request{Method: "GET", Path: client.JoinPath("executions", id, "logs", executor), Query: q}, false)
	}
	q := url.Values{}
	if page != "" {
		q.Set("page", page)
	}
	if maxPageSize != "" {
		q.Set("maxPageSize", maxPageSize)
	}
	return execute(c, cfg, client.Request{Method: "GET", Path: client.JoinPath("executions", id, "logs"), Query: q}, false)
}

func execTrigger(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "executions trigger")
	var commandID, inputJSON, file, data string
	var correlationID string
	var waitSeconds int
	var input, tag, inputReferences stringList
	var dryRun, withLogs bool
	fs.StringVar(&commandID, "command-id", "", "command id to trigger")
	fs.StringVar(&correlationID, "correlation-id", "", "correlation ID for the execution")
	fs.IntVar(&waitSeconds, "wait-seconds", 0, "server wait for completion (0-60 seconds); HTTP 202 may still be returned")
	fs.Var(&input, "input", "input value key=value (repeatable; sent as a string)")
	fs.StringVar(&inputJSON, "input-json", "", "string-valued inputValues JSON object (merged over --input)")
	fs.Var(&inputReferences, "input-reference", "reusable Input ID (repeatable)")
	fs.Var(&tag, "tag", "tag key=value (repeatable)")
	fs.BoolVar(&dryRun, "dry-run", false, "add the feature:dryRun tag")
	fs.BoolVar(&withLogs, "logs", false, "add the feature:logs tag")
	fs.StringVar(&file, "f", "", "send a prebuilt JSON body instead of building one")
	fs.StringVar(&file, "file", "", "send a prebuilt JSON body instead of building one")
	fs.StringVar(&data, "data", "", "inline JSON body, or @file")
	setUsage(fs, "POST /executions",
		"autopi executions trigger (--command-id ID --input k=v ... | -f FILE)",
		"autopi executions trigger --command-id my-cat:MyCommand:1 --input appName=my-app --logs")
	if err := parseNoArgs(fs, args); err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}

	var body []byte
	if waitSeconds < 0 || waitSeconds > 60 {
		return usagef("--wait-seconds must be between 0 and 60")
	}
	headers := map[string]string{}
	if flagSet(fs, "wait-seconds") {
		headers["Prefer"] = fmt.Sprintf("wait=%d", waitSeconds)
		wait := time.Duration(waitSeconds) * time.Second
		if g.timeout != "" && cfg.Timeout <= wait {
			return usagef("--timeout must exceed --wait-seconds")
		}
		if g.timeout == "" && cfg.Timeout <= wait {
			cfg.Timeout = wait + 5*time.Second
		}
	}
	if file != "" || data != "" {
		if flagSet(fs, "command-id") || flagSet(fs, "correlation-id") || flagSet(fs, "input-json") || flagSet(fs, "input") || flagSet(fs, "tag") || flagSet(fs, "input-reference") || flagSet(fs, "logs") || flagSet(fs, "dry-run") {
			return usagef("prebuilt body cannot be combined with command/input/reference/tag/log/dry-run modifiers")
		}
		body, err = resolveBody(c, file, data)
		if err != nil {
			return err
		}
		var payload map[string]json.RawMessage
		if json.Unmarshal(body, &payload) != nil || payload == nil {
			return usagef("trigger body must be a JSON object")
		}
		var id string
		if json.Unmarshal(payload["commandId"], &id) != nil || id == "" {
			return usagef("trigger body requires commandId")
		}
		if raw, ok := payload["inputReferences"]; ok {
			var refs []string
			if json.Unmarshal(raw, &refs) != nil || refs == nil {
				return usagef("inputReferences must be an array of Input IDs")
			}
			if err := validReferences(refs); err != nil {
				return err
			}
		}
		if raw, ok := payload["inputValues"]; ok {
			if _, err := stringValues(raw); err != nil {
				return err
			}
		}
		if raw, ok := payload["tags"]; ok {
			tags, err := stringValues(raw)
			if err != nil {
				return err
			}
			if err := domain.ValidateTags(tags, false); err != nil {
				return usagef("invalid trigger tags: %v", err)
			}
		}
	} else {
		if commandID == "" {
			return usagef("--command-id is required (or pass a full body with -f/--data)")
		}
		payload := map[string]any{"commandId": commandID}
		if correlationID != "" {
			payload["correlationId"] = correlationID
		}
		values := map[string]string{}
		for _, kv := range input {
			k, v, ok := strings.Cut(kv, "=")
			if !ok || k == "" {
				return usagef("invalid --input (want non-empty key=value)")
			}
			values[k] = v // strings on the wire; runtime converts to the declared type
		}
		iv := values
		if inputJSON != "" {
			extra, err := stringValues([]byte(inputJSON))
			if err != nil {
				return err
			}
			for k, v := range extra {
				iv[k] = v
			}
		}
		if err := validReferences(inputReferences); err != nil {
			return err
		}
		if len(inputReferences) > 0 {
			payload["inputReferences"] = inputReferences
		}
		if len(iv) > 0 {
			payload["inputValues"] = iv
		}
		tags := map[string]string{}
		for _, t := range tag {
			k, v, _ := strings.Cut(t, "=")
			tags[k] = v
		}
		if withLogs {
			tags["feature:logs"] = ""
		}
		if dryRun {
			tags["feature:dryRun"] = ""
		}
		if err := domain.ValidateTags(tags, false); err != nil {
			return usagef("invalid trigger tags: %v", err)
		}
		if len(tags) > 0 {
			payload["tags"] = tags
		}
		body, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}
	return execute(c, cfg, client.Request{Method: "POST", Path: "executions", Body: body, Headers: headers}, false)
}

func stringValues(raw []byte) (map[string]string, error) {
	var values map[string]json.RawMessage
	if json.Unmarshal(raw, &values) != nil || values == nil {
		return nil, usagef("input values must be a string-valued JSON object")
	}
	out := map[string]string{}
	for k, v := range values {
		var s string
		if k == "" || len(v) == 0 || v[0] != '"' || json.Unmarshal(v, &s) != nil {
			return nil, usagef("value for key %q must be a JSON string", k)
		}
		out[k] = s
	}
	return out, nil
}

func integerFlag(name, value string, min int) error {
	n, err := strconv.Atoi(value)
	if err != nil || n < min {
		return usagef("--%s must be an integer >= %d", name, min)
	}
	return nil
}
func knownStatus(value string) bool {
	switch value {
	case "RUNNING", "PAUSED", "INPUT_REQUIRED", "FAILED", "FINISHED", "ABORTED", "SUSPENDED":
		return true
	}
	return false
}

func execAction(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "executions action")
	var typ, reason, value, correlationID, ifMatch, file, data string
	fs.StringVar(&typ, "type", "", "PAUSE|RESUME|RETRY|ABORT|RESET|CHOOSE|SKIP|FAIL|COMMENT")
	fs.StringVar(&reason, "reason", "", "human-readable reason")
	fs.StringVar(&value, "value", "", "target (e.g. step alias for RESET)")
	fs.StringVar(&correlationID, "correlation-id", "", "action correlation ID")
	fs.StringVar(&ifMatch, "if-match", "", "conditional execution ETag")
	fs.StringVar(&file, "f", "", "prebuilt action JSON file; - for stdin")
	fs.StringVar(&file, "file", "", "prebuilt action JSON file")
	fs.StringVar(&data, "data", "", "prebuilt action JSON or @file")
	setUsage(fs, "POST /executions/{id}/actions",
		"autopi executions action <id> --type TYPE --reason TEXT [--value V] [--if-match ETAG]",
		"autopi executions action abc123 --type ABORT --reason 'superseded'")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	id, err := requireID(pos, "execution id")
	if err != nil {
		return err
	}
	if typ == "" && file == "" && data == "" {
		return usagef("--type is required")
	}
	switch typ {
	case "", "PAUSE", "RESUME", "RETRY", "ABORT", "RESET", "COMMENT", "CHOOSE", "SKIP", "FAIL":
	default:
		return usagef("invalid action type %q", typ)
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	payload := map[string]string{"type": typ}
	if correlationID != "" {
		payload["correlationId"] = correlationID
	}
	if reason != "" {
		payload["reason"] = reason
	}
	if value != "" {
		payload["value"] = value
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if file != "" || data != "" {
		if flagSet(fs, "type") || flagSet(fs, "reason") || flagSet(fs, "value") || flagSet(fs, "correlation-id") {
			return usagef("prebuilt action cannot be combined with builder fields")
		}
		body, err = resolveBody(c, file, data)
		if err != nil {
			return err
		}
	}
	if flagSet(fs, "if-match") && ifMatch == "" {
		return usagef("--if-match requires a non-empty ETag")
	}
	headers := map[string]string{}
	if ifMatch != "" {
		headers["If-Match"] = ifMatch
	}
	return execute(c, cfg, client.Request{Method: "POST", Path: client.JoinPath("executions", id, "actions"), Body: body, Headers: headers}, false)
}

func execBulkAction(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "executions bulk-action")
	var ids stringList
	var typ, reason, file, data string
	fs.Var(&ids, "execution-id", "execution ID (repeatable, 1-50; duplicates retain result entries)")
	fs.StringVar(&typ, "type", "", "PAUSE|RESUME|ABORT|RETRY")
	fs.StringVar(&reason, "reason", "", "required action reason")
	fs.StringVar(&file, "f", "", "prebuilt JSON body; - for stdin")
	fs.StringVar(&file, "file", "", "prebuilt JSON body")
	fs.StringVar(&data, "data", "", "JSON body or @file")
	setUsage(fs, "POST /executions/actions", "autopi executions bulk-action --execution-id ID ... --type TYPE --reason TEXT", "autopi executions bulk-action -f bulk-action.json")
	if err := parseNoArgs(fs, args); err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	var body []byte
	if file != "" || data != "" {
		if flagSet(fs, "execution-id") || flagSet(fs, "type") || flagSet(fs, "reason") {
			return usagef("prebuilt bulk action cannot be combined with builder fields")
		}
		body, err = resolveBody(c, file, data)
	} else {
		body, err = json.Marshal(map[string]any{"executionIds": ids, "type": typ, "reason": reason})
	}
	if err != nil {
		return err
	}
	cl := c.client(cfg)
	res, err := fetch(c, cl, client.Request{Method: "POST", Path: "executions/actions", Body: body})
	if err != nil {
		return err
	}
	if res == nil {
		return nil
	}
	var request struct{ ExecutionIDs []string }
	if err := json.Unmarshal(body, &request); err != nil {
		return err
	}
	var response struct {
		Results []struct {
			ExecutionID string
			Succeeded   *bool
		}
	}
	if err := cl.Render(c.stdout, res); err != nil {
		return err
	}
	if err := json.Unmarshal(res.Body, &response); err != nil || len(response.Results) != len(request.ExecutionIDs) {
		return fmt.Errorf("invalid bulk action response; inspect execution state before retrying")
	}
	failed := 0
	for i, result := range response.Results {
		if result.ExecutionID != request.ExecutionIDs[i] {
			return fmt.Errorf("bulk action result order or execution ID differs; inspect execution state before retrying")
		}
		if result.Succeeded == nil {
			return fmt.Errorf("bulk action result is missing succeeded")
		}
		if !*result.Succeeded {
			failed++
		}
	}
	if failed > 0 {
		return &partialFailure{failed}
	}
	return nil
}

func validReferences(refs []string) error {
	seen := map[string]bool{}
	for _, id := range refs {
		parts := strings.Split(id, ":")
		if len(parts) != 3 || parts[0] == "" || parts[1] == "" || seen[id] {
			return usagef("inputReferences require unique catalog:name:version IDs")
		}
		if err := integerFlag("input-reference version", parts[2], 1); err != nil {
			return err
		}
		seen[id] = true
	}
	return nil
}
