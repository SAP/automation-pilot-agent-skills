package cmd

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
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
			{"action", "PAUSE/RESUME/RETRY/ABORT/RESET/COMMENT", "POST /executions/{id}/actions", execAction},
			{"actions", "action history", "GET /executions/{id}/actions", execActions},
			{"delete", "delete an execution", "DELETE /executions/{id}", execDelete},
		},
	}
}

func execList(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "executions list")
	var limit string
	var status, commandID, execID, tag stringList
	var startedAfter, startedBefore string
	var full bool
	fs.StringVar(&limit, "limit", "", "max results")
	fs.Var(&status, "status", "filter by status (repeatable), e.g. FAILED RUNNING PAUSED")
	fs.Var(&commandID, "command-id", "filter by command id (repeatable)")
	fs.Var(&execID, "execution-id", "filter by execution id (repeatable)")
	fs.Var(&tag, "tag", "filter by tag key:value (repeatable)")
	fs.StringVar(&startedAfter, "started-after", "", "epoch-ms, or a relative window like 24h")
	fs.StringVar(&startedBefore, "started-before", "", "epoch-ms, or a relative window like 24h")
	fs.BoolVar(&full, "full", false, "return complete objects instead of the metadata summary")
	setUsage(fs, "GET /executions", "autopi executions list [--status S ...] [--limit N] [--started-after 24h]",
		"autopi executions list --status FAILED --started-after 24h --limit 50")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	q := url.Values{}
	if limit != "" {
		q.Set("limit", limit)
	}
	for _, s := range status {
		q.Add("status", s)
	}
	for _, id := range commandID {
		q.Add("commandId", id)
	}
	for _, id := range execID {
		q.Add("executionId", id)
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
	return execute(c, cfg, client.Request{Method: "GET", Path: "executions", Query: q, Summarize: !full}, false)
}

// epochMillis accepts a literal epoch-ms string or a Go duration like "24h" and
// returns the corresponding epoch-ms (now - duration) as a string. This removes
// the shell `$(($(date +%s)...))` arithmetic the skills used.
func epochMillis(v string) (string, error) {
	if n, err := strconv.ParseInt(v, 10, 64); err == nil {
		return strconv.FormatInt(n, 10), nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return "", usagef("invalid time %q (want epoch-ms or a duration like 24h)", v)
	}
	return strconv.FormatInt(time.Now().Add(-d).UnixMilli(), 10), nil
}

func execGet(c *cmdContext, args []string) error {
	return getVerb(c, args, "executions get", "GET /executions/{id}",
		"autopi executions get abc123", "execution id", "executions")
}

func execSummary(c *cmdContext, args []string) error {
	return listVerb(c, args, "executions summary", "GET /executions/summary",
		"autopi executions summary", "executions/summary", false)
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
	fs.StringVar(&page, "page", "", "page number (0-based)")
	fs.StringVar(&maxPageSize, "max-page-size", "", "page size")
	fs.StringVar(&executor, "executor", "", "fetch logs for a specific executor path")
	setUsage(fs, "GET /executions/{id}/logs[/{executor}]",
		"autopi executions logs <id> [--page 0 --max-page-size 20 | --executor PATH]",
		"autopi executions logs abc123 --page 0 --max-page-size 20")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
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
		return execute(c, cfg, client.Request{Method: "GET", Path: client.JoinPath("executions", id, "logs", executor)}, false)
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
	var input, tag stringList
	var dryRun, withLogs bool
	fs.StringVar(&commandID, "command-id", "", "command id to trigger")
	fs.Var(&input, "input", "input value key=value (repeatable; sent as a string)")
	fs.StringVar(&inputJSON, "input-json", "", "inputValues as a JSON object (merged over --input)")
	fs.Var(&tag, "tag", "tag key:value (repeatable)")
	fs.BoolVar(&dryRun, "dry-run", false, "add the feature:dryRun tag")
	fs.BoolVar(&withLogs, "logs", false, "add the feature:logs tag")
	fs.StringVar(&file, "f", "", "send a prebuilt JSON body instead of building one")
	fs.StringVar(&file, "file", "", "send a prebuilt JSON body instead of building one")
	fs.StringVar(&data, "data", "", "inline JSON body, or @file")
	setUsage(fs, "POST /executions",
		"autopi executions trigger (--command-id ID --input k=v ... | -f FILE)",
		"autopi executions trigger --command-id my-cat:MyCommand:1 --input appName=my-app --logs")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}

	var body []byte
	if file != "" || data != "" {
		body, err = resolveBody(file, data)
		if err != nil {
			return err
		}
	} else {
		if commandID == "" {
			return usagef("--command-id is required (or pass a full body with -f/--data)")
		}
		payload := map[string]any{"commandId": commandID}
		values := map[string]string{}
		for _, kv := range input {
			k, v, ok := strings.Cut(kv, "=")
			if !ok {
				return usagef("invalid --input %q (want key=value)", kv)
			}
			values[k] = v // strings on the wire; runtime converts to the declared type
		}
		iv := toAny(values)
		if inputJSON != "" {
			var extra map[string]any
			if err := json.Unmarshal([]byte(inputJSON), &extra); err != nil {
				return usagef("invalid --input-json: %v", err)
			}
			for k, v := range extra {
				iv[k] = v
			}
		}
		if len(iv) > 0 {
			payload["inputValues"] = iv
		}
		tags := map[string]string{}
		for _, t := range tag {
			k, v, _ := strings.Cut(t, ":")
			tags[k] = v
		}
		if withLogs {
			tags["feature:logs"] = ""
		}
		if dryRun {
			tags["feature:dryRun"] = ""
		}
		if len(tags) > 0 {
			payload["tags"] = tags
		}
		body, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}
	return execute(c, cfg, client.Request{Method: "POST", Path: "executions", Body: body}, false)
}

func toAny(m map[string]string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func execAction(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "executions action")
	var typ, reason, value string
	fs.StringVar(&typ, "type", "", "PAUSE|RESUME|RETRY|ABORT|RESET|COMMENT")
	fs.StringVar(&reason, "reason", "", "human-readable reason")
	fs.StringVar(&value, "value", "", "target (e.g. step alias for RESET)")
	setUsage(fs, "POST /executions/{id}/actions",
		"autopi executions action <id> --type TYPE [--reason TEXT] [--value V]",
		"autopi executions action abc123 --type ABORT --reason 'superseded'")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	id, err := requireID(pos, "execution id")
	if err != nil {
		return err
	}
	if typ == "" {
		return usagef("--type is required")
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	payload := map[string]string{"type": typ}
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
	return execute(c, cfg, client.Request{Method: "POST", Path: client.JoinPath("executions", id, "actions"), Body: body}, false)
}
