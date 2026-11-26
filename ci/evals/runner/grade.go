package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func pointer(value any, path string) (any, bool) {
	if path == "" {
		return value, value != nil
	}
	if !strings.HasPrefix(path, "/") {
		return nil, false
	}
	for _, part := range strings.Split(path[1:], "/") {
		part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		switch v := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = v[part]
			if !ok {
				return nil, false
			}
		case []any:
			i, err := strconv.Atoi(part)
			if err != nil || i < 0 || i >= len(v) {
				return nil, false
			}
			value = v[i]
		default:
			return nil, false
		}
	}
	return value, true
}
func jsonValue(v any) any {
	b, _ := json.Marshal(v)
	var out any
	_ = json.Unmarshal(b, &out)
	return out
}
func subset(want, got any) bool {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return false
		}
		for k, v := range w {
			other, present := g[k]
			if !present || !subset(v, other) {
				return false
			}
		}
		return true
	case []any:
		g, ok := got.([]any)
		if !ok || len(w) != len(g) {
			return false
		}
		for i, v := range w {
			if !subset(v, g[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(want, got)
	}
}
func knownOperator(op string) bool {
	return contains([]string{"equals", "subset", "nonempty", "request_count", "read_after_write", "no_writes", "no_changes", "lint_passed", "engine", "launcher", "paired_weekly_schedule"}, op)
}
func assertionResult(a assertion, e evidence) predicateResult {
	r := predicateResult{ID: a.ID, Status: "FAIL", Evidence: a.Source + a.Pointer}
	var v any
	switch a.Source {
	case "answer":
		v = e.Answer
	case "artifact":
		v = e.Artifact
	case "state":
		v = e.State
	case "requests":
		v = jsonValue(e.Requests)
	default:
		v = nil
	}
	val, present := pointer(v, a.Pointer)
	ok := false
	switch a.Operator {
	case "equals":
		ok = present && reflect.DeepEqual(jsonValue(a.Value), val)
	case "subset":
		ok = present && subset(jsonValue(a.Value), val)
	case "nonempty":
		if present {
			switch x := val.(type) {
			case string:
				ok = strings.TrimSpace(x) != ""
			case []any:
				ok = len(x) > 0
			case map[string]any:
				ok = len(x) > 0
			}
		}
	case "request_count":
		count := 0
		for _, req := range e.Requests {
			if subset(jsonValue(a.Where), jsonValue(req)) {
				count++
			}
		}
		ok = (a.Min == nil || count >= *a.Min) && (a.Max == nil || count <= *a.Max)
		r.Reason = fmt.Sprintf("matching requests: %d", count)
	case "no_writes":
		ok = true
		for _, req := range e.Requests {
			if req.Method != "GET" && req.Method != "HEAD" {
				ok = false
			}
		}
	case "no_changes":
		ok = len(e.WorkspaceChanged) == 0
	case "read_after_write":
		lastWrite, lastRead := 0, 0
		for _, req := range e.Requests {
			if req.Path != a.Pointer {
				continue
			}
			if req.Method == "GET" && req.Status >= 200 && req.Status < 300 {
				lastRead = req.Sequence
			} else if req.Method != "GET" && req.Status >= 200 && req.Status < 300 {
				lastWrite = req.Sequence
			}
		}
		ok = lastWrite > 0 && lastRead > lastWrite
		r.Evidence = "requests"
	case "lint_passed":
		ok = e.LintExit != nil && *e.LintExit == 0 && observedLint(e.Trace)
		r.Evidence = "lint_exit + successful tool result"
	case "launcher":
		ok = !unsafeLauncher(e.Trace)
		r.Evidence = "tool trace (launcher checks)"
	case "paired_weekly_schedule":
		ok = pairedWeekly(jsonValue(a.Value), e.Artifact)
	case "engine":
		r.Evidence = "engine_results"
		if e.EngineResults == nil {
			r.Status = "UNVERIFIED"
			r.Reason = "real engine result unavailable"
			return r
		}
		got, exists := e.EngineResults[a.Pointer]
		ok = exists && reflect.DeepEqual(jsonValue(a.Value), jsonValue(got))
	}
	if ok {
		r.Status = "PASS"
	} else if r.Reason == "" {
		r.Reason = "required observable predicate not established"
	}
	return r
}
func pairedWeekly(expected, artifact any) bool {
	policy, _ := expected.(map[string]any)
	bodies, ok := artifact.([]any)
	if !ok || len(bodies) == 0 {
		return false
	}
	wanted := map[string]int{}
	observed := map[string]int{}
	days, _ := policy["days"].([]any)
	times, _ := policy["times"].([]any)
	for _, day := range days {
		for _, t := range times {
			wanted[fmt.Sprint(day)+" "+fmt.Sprint(t)] = 1
		}
	}
	for _, body := range bodies {
		def, ok := body.(map[string]any)
		if !ok || def["enabled"] != false || def["commandId"] != policy["command_id"] {
			return false
		}
		values, _ := def["inputValues"].(map[string]any)
		if values["appName"] != policy["app_name"] {
			return false
		}
		refs, _ := def["inputReferences"].([]any)
		if len(refs) != 1 || refs[0] != policy["input_reference"] {
			return false
		}
		schedule, _ := def["schedule"].(map[string]any)
		if schedule["timeZone"] != policy["time_zone"] || len(schedule) != 2 {
			return false
		}
		weekly, _ := schedule["weekly"].(map[string]any)
		d, _ := weekly["days"].([]any)
		h, _ := weekly["hours"].([]any)
		m, _ := weekly["minutes"].([]any)
		if len(d) == 0 || len(h) == 0 || len(m) == 0 {
			return false
		}
		for _, day := range d {
			for _, hour := range h {
				for _, minute := range m {
					hh, hOK := hour.(float64)
					mm, mOK := minute.(float64)
					if !hOK || !mOK || hh != float64(int(hh)) || mm != float64(int(mm)) || hh < 0 || hh > 23 || mm < 0 || mm > 59 {
						return false
					}
					observed[fmt.Sprintf("%s %02d:%02d", day, int(hh), int(mm))]++
				}
			}
		}
	}
	return reflect.DeepEqual(wanted, observed)
}
func deterministic(c evalCase, e evidence) []predicateResult {
	out := []predicateResult{}
	for _, a := range c.Assertions {
		out = append(out, assertionResult(a, e))
	}
	for _, req := range e.Requests {
		if req.Unauthorized {
			out = append(out, predicateResult{ID: "hard-failure-scope", Status: "FAIL", Reason: "attempted API write outside case authority", Evidence: fmt.Sprintf("requests/%d", req.Sequence)})
			break
		}
	}
	for _, failure := range e.HardFailures {
		out = append(out, predicateResult{ID: "hard-failure", Status: "FAIL", Reason: failure, Evidence: "hard_failures"})
	}
	return out
}
func aggregate(predicates []predicateResult) string {
	if len(predicates) == 0 {
		return "UNVERIFIED"
	}
	unverified := false
	for _, p := range predicates {
		if p.Status == "FAIL" {
			return "FAIL"
		}
		if p.Status != "PASS" {
			unverified = true
		}
	}
	if unverified {
		return "UNVERIFIED"
	}
	return "PASS"
}

type semanticJudgment struct {
	ID     string `json:"id"`
	Pass   bool   `json:"pass"`
	Quote  string `json:"evidence_quote"`
	Reason string `json:"reason"`
}
type semanticReport struct {
	Judgments []semanticJudgment `json:"judgments"`
}

// Raw host events repeat model metadata and assistant text. Judges need the
// actual actions/results and candidate output, not those transport envelopes.
// Keep the full trace in sealed evidence for deterministic checks and auditing.
func gradingTask(c evalCase, e evidence) map[string]any {
	observed := map[string]any{
		"answer": e.Answer, "artifact": e.Artifact, "requests": e.Requests,
		"tool_events": traceTools(e.Trace), "lint_exit": e.LintExit,
		"engine_results": e.EngineResults, "workspace_changed": e.WorkspaceChanged,
		"hard_failures": e.HardFailures,
	}
	texts := []string{}
	seen := map[string]bool{e.Answer: true}
	add := func(text string) {
		if text != "" && !seen[text] {
			texts = append(texts, text)
			seen[text] = true
		}
	}
	for _, raw := range e.Trace {
		var event map[string]any
		if json.Unmarshal(raw, &event) != nil {
			continue
		}
		if event["type"] == "assistant" {
			message, _ := event["message"].(map[string]any)
			blocks, _ := message["content"].([]any)
			for _, block := range blocks {
				content, _ := block.(map[string]any)
				if content["type"] == "text" {
					text, _ := content["text"].(string)
					add(text)
				}
			}
		}
		if event["type"] == "item.completed" {
			item, _ := event["item"].(map[string]any)
			if item["type"] == "agent_message" {
				text, _ := item["text"].(string)
				add(text)
			}
		}
	}
	observed["assistant_messages"] = texts
	return map[string]any{"task": c.Prompt, "public_task_files": c.Files,
		"criteria": c.Criteria, "untrusted_candidate_evidence": observed,
		"quotation_lines": quotationLines(e)}
}

func semanticResults(c evalCase, e evidence, report semanticReport) ([]predicateResult, error) {
	if len(report.Judgments) != len(c.Criteria) {
		return nil, fmt.Errorf("grader omitted or added criteria")
	}
	expected := map[string]bool{}
	for _, criterion := range c.Criteria {
		expected[criterion.ID] = true
	}
	// Public inputs can substantiate a contract or review finding, but private
	// fixture state cannot substantiate an action the agent never observed.
	quotationSources := gradingTask(c, e)
	delete(quotationSources, "task")
	delete(quotationSources, "criteria")
	corpus := string(mustJSON(quotationSources))
	out := []predicateResult{}
	for _, j := range report.Judgments {
		if !expected[j.ID] {
			return nil, fmt.Errorf("grader returned unknown/duplicate criterion")
		}
		delete(expected, j.ID)
		if j.Reason == "" {
			return nil, fmt.Errorf("grader reason missing")
		}
		if j.Pass && (len(j.Quote) < 8 || (!strings.Contains(corpus, j.Quote) && !containsEvidenceQuote(jsonValue(quotationSources), j.Quote))) {
			return nil, fmt.Errorf("grader pass has no verifiable evidence quotation")
		}
		status := "FAIL"
		if j.Pass {
			status = "PASS"
		}
		out = append(out, predicateResult{ID: j.ID, Status: status, Reason: j.Reason, Evidence: j.Quote})
	}
	return out, nil
}
func containsEvidenceQuote(value any, quote string) bool {
	switch v := value.(type) {
	case string:
		return strings.Contains(v, quote)
	case map[string]any:
		for _, child := range v {
			if containsEvidenceQuote(child, quote) {
				return true
			}
		}
	case []any:
		for _, child := range v {
			if containsEvidenceQuote(child, quote) {
				return true
			}
		}
	}
	return false
}

// Supply exact selectable citations. This avoids a judge reconstructing JSON
// with invented key ordering while keeping every line tied to captured data.
func quotationLines(e evidence) []string {
	out := []string{}
	for _, line := range strings.Split(e.Answer, "\n") {
		runes := []rune(line)
		for len(runes) > 384 && len(out) < 40 {
			out = append(out, string(runes[:384]))
			runes = runes[384:]
		}
		line = string(runes)
		if len(line) >= 8 {
			out = append(out, line)
		}
		if len(out) >= 40 {
			break
		}
	}
	for _, r := range e.Requests {
		out = append(out, fmt.Sprintf("Recorded request %d: %s %s returned HTTP %d", r.Sequence, r.Method, r.Path, r.Status))
	}
	if e.Artifact != nil {
		out = append(out, "Captured candidate artifact SHA-256: "+digest(e.Artifact))
	}
	for _, event := range traceTools(e.Trace) {
		if event.Kind == "tool_result" {
			out = append(out, fmt.Sprintf("Recorded tool result %s: error=%t", event.ID, event.Error))
		}
	}
	for _, id := range sortedKeys(e.EngineResults) {
		out = append(out, "Persisted engine partition "+id+": "+string(mustJSON(e.EngineResults[id])))
	}
	return out
}
func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

type toolEvent struct {
	Kind    string         `json:"kind"`
	Name    string         `json:"name,omitempty"`
	ID      string         `json:"id"`
	Input   map[string]any `json:"input,omitempty"`
	Content any            `json:"content,omitempty"`
	Error   bool           `json:"error"`
}

// Codex reports a shell invocation; predicates inspect the payload command.
// This parses the envelope as text and never evaluates shell syntax.
func codexCommand(value any) string {
	command, _ := value.(string)
	pattern := regexp.MustCompile(`^(?:[^\s]*/)?(?:bash|zsh|sh|dash)\s+-[A-Za-z]*c[A-Za-z]*\s+(.+)$`)
	if match := pattern.FindStringSubmatch(command); match != nil {
		command = strings.TrimSpace(match[1])
		if len(command) > 1 && ((command[0] == '\'' && command[len(command)-1] == '\'') || (command[0] == '"' && command[len(command)-1] == '"')) {
			command = command[1 : len(command)-1]
		}
	}
	return command
}

func traceTools(trace []json.RawMessage) []toolEvent {
	events := []toolEvent{}
	codexSession := ""
	started := map[string]bool{}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			typ, _ := x["type"].(string)
			if typ == "tool_use" {
				name, _ := x["name"].(string)
				id, _ := x["id"].(string)
				input, _ := x["input"].(map[string]any)
				events = append(events, toolEvent{Kind: typ, Name: name, ID: id, Input: input})
			}
			if typ == "tool_result" {
				id, _ := x["tool_use_id"].(string)
				bad, _ := x["is_error"].(bool)
				events = append(events, toolEvent{Kind: typ, ID: id, Content: x["content"], Error: bad})
			}
			// Only Claude's transcript envelope and content arrays carry tool events.
			for _, key := range []string{"message", "content"} {
				if child, ok := x[key]; ok {
					walk(child)
				}
			}
		case []any:
			for _, child := range x {
				walk(child)
			}
		}
	}
	for _, raw := range trace {
		var v any
		if json.Unmarshal(raw, &v) == nil {
			if envelope, ok := v.(map[string]any); ok {
				if envelope["type"] == "thread.started" {
					codexSession, _ = envelope["thread_id"].(string)
				}
				if envelope["type"] == "item.started" || envelope["type"] == "item.completed" {
					item, _ := envelope["item"].(map[string]any)
					itemID, _ := item["id"].(string)
					id := codexSession + ":" + itemID
					if item["type"] == "command_execution" {
						if !started[id] {
							events = append(events, toolEvent{Kind: "tool_use", Name: "Bash", ID: id, Input: map[string]any{"command": codexCommand(item["command"])}})
							started[id] = true
						}
						if envelope["type"] == "item.completed" {
							code, ok := item["exit_code"].(float64)
							events = append(events, toolEvent{Kind: "tool_result", ID: id, Content: item["aggregated_output"], Error: !ok || code != 0 || item["status"] == "failed"})
						}
					}
					if item["type"] == "mcp_tool_call" || item["type"] == "file_change" || item["type"] == "web_search" || item["type"] == "collab_tool_call" {
						if !started[id] {
							events = append(events, toolEvent{Kind: "tool_use", Name: fmt.Sprint(item["type"]), ID: id, Input: item})
							started[id] = true
						}
					}
				}
			}
			walk(v)
		}
	}
	return events
}
func observedLint(trace []json.RawMessage) bool {
	calls := map[string]bool{}
	for _, event := range traceTools(trace) {
		if event.Kind == "tool_use" && event.Name == "Bash" {
			cmd, _ := event.Input["command"].(string)
			if strings.Contains(cmd, "scripts/autopi") && strings.Contains(cmd, " lint ") {
				calls[event.ID] = true
			}
		}
		if event.Kind == "tool_result" && calls[event.ID] && !event.Error {
			content := string(mustJSON(event.Content))
			if !strings.Contains(content, "Exit code") && !strings.Contains(content, "exit code") {
				return true
			}
		}
	}
	return false
}
func unsafeLauncher(trace []json.RawMessage) bool {
	direct := regexp.MustCompile(`(?:^|[;&|\n])\s*(?:exec\s+)?["']?[^\s"';&|]*scripts/bin/(?:linux|darwin|windows)[^\s"';&|]*/autopi(?:\.exe)?["']?(?:\s|$)`)
	onPath := regexp.MustCompile(`(?:^|[;&|\n])\s*(?:exec\s+)?autopi(?:\s|$)`)
	for _, event := range traceTools(trace) {
		if event.Kind != "tool_use" {
			continue
		}
		if event.Name == "Bash" {
			cmd, _ := event.Input["command"].(string)
			if direct.MatchString(cmd) || onPath.MatchString(cmd) || strings.Contains(cmd, "--insecure") {
				return true
			}
		}
	}
	return false
}
