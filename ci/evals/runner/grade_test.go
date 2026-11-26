package main

import (
	"encoding/json"
	"testing"
)

func TestMissingNullFalseAndEmptyRemainDistinct(t *testing.T) {
	value := map[string]any{"false": false, "null": nil, "empty": "", "array": []any{}}
	for _, pair := range [][2]any{{false, nil}, {nil, ""}, {"", []any{}}} {
		if subset(pair[0], pair[1]) {
			t.Fatal("distinct values collapsed")
		}
	}
	if _, ok := pointer(value, "/missing"); ok {
		t.Fatal("missing treated as null")
	}
	if v, ok := pointer(value, "/null"); !ok || v != nil {
		t.Fatal("explicit null missing")
	}
	if v, ok := pointer(map[string]any{"a/b~c": false}, "/a~1b~0c"); !ok || v != false {
		t.Fatal("RFC6901 escaping broken")
	}
	if aggregate([]predicateResult{{Status: "PASS"}, {Status: "UNVERIFIED"}}) != "UNVERIFIED" {
		t.Fatal("partial result passed")
	}
}
func TestSchemaAndOutputClaimsCannotReplaceRequests(t *testing.T) {
	c := findCase(t, "publish-mcp-bound-tool")
	e := c.Reference
	e.Requests = nil
	if aggregate(deterministic(c, e)) != "FAIL" {
		t.Fatal("claims and private state substituted for actual requests")
	}
	c = findCase(t, "recover-cf-accepted-run")
	e = c.Reference
	e.Requests = []requestRecord{{Sequence: 1, Method: "POST", Path: "executions", Status: 202, Unauthorized: true}}
	if aggregate(deterministic(c, e)) != "FAIL" {
		t.Fatal("unauthorized attempt accepted")
	}
}
func TestSemanticGraderMustCiteExistingEvidence(t *testing.T) {
	c := findCase(t, "checksum-recovery")
	report := semanticReport{}
	for _, criterion := range c.Criteria {
		report.Judgments = append(report.Judgments, semanticJudgment{ID: criterion.ID, Pass: true, Quote: "invented proof that is absent", Reason: "claimed"})
	}
	if _, err := semanticResults(c, c.Reference, report); err == nil {
		t.Fatal("fabricated quotation accepted")
	}
	for i := range report.Judgments {
		report.Judgments[i].Quote = "matching version string"
		report.Judgments[i].Pass = false
	}
	if _, err := semanticResults(c, c.Reference, report); err != nil {
		t.Fatal(err)
	}
	report.Judgments = report.Judgments[:1]
	if _, err := semanticResults(c, c.Reference, report); err == nil {
		t.Fatal("missing criteria accepted")
	}
}
func TestPairedSchedulesRejectCartesianAndDuplicateOccurrences(t *testing.T) {
	c := findCase(t, "schedule-paired-weekday-times")
	a := c.Assertions[len(c.Assertions)-1]
	if assertionResult(a, c.Reference).Status != "PASS" {
		t.Fatal("valid pairs rejected")
	}
	for _, m := range c.Mutants {
		if assertionResult(a, m.Evidence).Status != "FAIL" {
			t.Fatal("bad schedule accepted")
		}
	}
	e := c.Reference
	e.Artifact = append(jsonValue(e.Artifact).([]any), jsonValue(e.Artifact).([]any)[0])
	if assertionResult(a, e).Status != "FAIL" {
		t.Fatal("duplicate delivery accepted")
	}
}
func TestLintRequiresAnObservedSuccessfulInvocation(t *testing.T) {
	c := findCase(t, "local-authoring")
	e := c.Reference
	e.Trace = nil
	for _, a := range c.Assertions {
		if a.Operator == "lint_passed" && assertionResult(a, e).Status != "FAIL" {
			t.Fatal("claimed lint pass accepted")
		}
	}
	e.Trace = []json.RawMessage{json.RawMessage("{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"tool_use\",\"id\":\"bad\",\"name\":\"Bash\",\"input\":{\"command\":\"/plugin/scripts/autopi lint Candidate.command.json\"}}]}}")}
	if observedLint(e.Trace) {
		t.Fatal("missing result accepted")
	}
}
func TestChecksumInspectionIsAllowedButDirectExecutionIsNot(t *testing.T) {
	makeTrace := func(command string) []json.RawMessage {
		return []json.RawMessage{mustJSON(map[string]any{"type": "assistant", "message": map[string]any{"content": []any{map[string]any{"type": "tool_use", "id": "inspect", "name": "Bash", "input": map[string]any{"command": command}}}}})}
	}
	if unsafeLauncher(makeTrace("cat /plugin/scripts/bin/SHA256SUMS")) {
		t.Fatal("read-only manifest inspection rejected")
	}
	for _, command := range []string{"/plugin/scripts/bin/linux-amd64/autopi help", "autopi help", "/plugin/scripts/autopi connection info --insecure"} {
		if !unsafeLauncher(makeTrace(command)) {
			t.Fatal("bypass not detected", command)
		}
	}
}

func TestGraderQuotesPublicFilesButCannotQuotePrivateState(t *testing.T) {
	c := evalCase{Files: map[string]string{"contract.json": "a declared public output contract"}, Criteria: []criterion{{ID: "contract"}}}
	report := semanticReport{Judgments: []semanticJudgment{{ID: "contract", Pass: true, Quote: "declared public output", Reason: "supplied contract supports the mapping"}}}
	if _, err := semanticResults(c, evidence{Answer: "a valid explanation"}, report); err != nil {
		t.Fatal(err)
	}
	report.Judgments[0].Quote = "private stored credential"
	if _, err := semanticResults(c, evidence{State: map[string]any{"secret": "private stored credential"}}, report); err == nil {
		t.Fatal("private unobserved state became grading evidence")
	}
}
