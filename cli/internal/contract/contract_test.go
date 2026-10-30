package contract

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func TestRequestBoundsAndPublicActionRules(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body string
		query                    url.Values
		valid                    bool
	}{
		{"limit boundary", "GET", "executions", "", url.Values{"limit": {"1000"}}, true},
		{"limit too high", "GET", "executions", "", url.Values{"limit": {"1001"}}, false},
		{"zero log page size", "GET", "executions/id/logs", "", url.Values{"maxPageSize": {"0"}}, true},
		{"large log page", "GET", "executions/id/logs", "", url.Values{"maxPageSize": {"21"}}, false},
		{"too many statuses", "GET", "executions", "", url.Values{"status": {"RUNNING", "RUNNING", "RUNNING", "RUNNING", "RUNNING", "RUNNING", "RUNNING", "RUNNING", "RUNNING"}}, false},
		{"partial identifier filter", "GET", "executions", "", url.Values{"filterMode": {"contains"}, "originatorId": {"operator"}}, true},
		{"invalid mode", "GET", "executions", "", url.Values{"filterMode": {"prefix"}}, false},
		{"missing reason", "POST", "executions/id/actions", `{"type":"PAUSE"}`, nil, false},
		{"missing reset value", "POST", "executions/id/actions", `{"type":"RESET","reason":"repair"}`, nil, false},
		{"operator choice", "POST", "executions/id/actions", `{"type":"CHOOSE","reason":"continue","value":"yes"}`, nil, true},
		{"internal action", "POST", "executions/id/actions", `{"type":"SUSPEND","reason":"test"}`, nil, false},
		{"generic no event", "POST", "triggers/generic-event", "", url.Values{"commandReference": {"c:C:1"}}, true},
		{"generic arbitrary bytes", "POST", "triggers/generic-event", "plain event", url.Values{"commandReference": {"c:C:1"}}, true},
		{"required ANS event", "POST", "triggers/ans-event", "", url.Values{"commandReference": {"c:C:1"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRequest(tc.method, tc.path, tc.query, nil, []byte(tc.body))
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}

func TestBulkActionSizeBoundaries(t *testing.T) {
	for _, count := range []int{0, 1, 50, 51} {
		ids := make([]string, count)
		for i := range ids {
			ids[i] = "same"
		}
		body, _ := json.Marshal(map[string]any{"executionIds": ids, "type": "ABORT", "reason": "test"})
		valid := len(ValidateKind("bulk-action", body, false)) == 0
		if valid != (count >= 1 && count <= 50) {
			t.Fatalf("count=%d valid=%v", count, valid)
		}
	}
}

func TestListenerSchemaCorrection(t *testing.T) {
	base := `{"name":"ListenerCommand","catalog":"c","version":1,"inputKeys":{},"outputKeys":{},"configuration":{"executors":[],"listeners":[LISTENER]}}`
	for _, tc := range []struct {
		listener string
		valid    bool
	}{
		{`{"status":"FINISHED","execute":"c:Audit:1","input":{}}`, true},
		{`{"execute":"c:Audit:1"}`, false},
		{`{"status":"UNKNOWN","execute":"c:Audit:1"}`, false},
	} {
		findings := ValidateKind("command", []byte(strings.Replace(base, "LISTENER", tc.listener, 1)), false)
		if (len(findings) == 0) != tc.valid {
			t.Fatalf("listener=%s findings=%v", tc.listener, findings)
		}
	}
}

func TestScheduleAndMergePatchServiceExceptions(t *testing.T) {
	for _, tc := range []struct {
		kind, body string
		valid      bool
	}{
		{"scheduled-execution", `{"commandId":"c:C:1","schedule":{"once":{"month":10,"day":28,"hour":12,"minute":30}}}`, true},
		{"scheduled-execution", `{"commandId":"c:C:1","schedule":{"once":{"month":10,"day":28,"hour":12,"minute":31}}}`, false},
		{"scheduled-execution", `{"commandId":"c:C:1","schedule":{"hourly":{"minutes":[0,0]}}}`, false},
		{"scheduled-execution", `{"commandId":"c:C:1","schedule":{"hourly":{"minutes":[0]},"daily":{"hours":[12],"minutes":[0]}}}`, false},
		{"scheduled-execution", `{"commandId":"c:C:1","schedule":{"hourly":{"minutes":[0,5]},"deadlineInMinutes":10}}`, false},
		{"scheduled-execution", `{"commandId":"c:C:1","schedule":{"hourly":{"minutes":[0]}}}`, true},
		{"scheduled-execution", `{"commandId":"c:C:1","schedule":{"timeZone":null,"hourly":{"minutes":[0]}}}`, false},
		{"webhook", `{"name":"Monitor","commandId":"c:C:1","type":"GENERIC"}`, true},
		{"webhook", `{"name":"bad-name","commandId":"c:C:1","type":"GENERIC"}`, false},
		{"input-patch", `{"description":"updated"}`, true},
		{"input-patch", `{"values":{"password":"fixture"}}`, true},
		{"input-patch", `{"values":{"password":null}}`, false},
		{"input-patch", `{"keys":{"obsolete":null}}`, true},
		{"input-patch", `{"keys":{"newKey":{"type":"string"}}}`, false},
		{"input-patch", `{}`, false},
		{"input-patch", `{"name":"Renamed"}`, false},
		{"input", `{"name":"Config","catalog":"c","version":1,"keys":{}}`, false},
		{"catalog", `{"name":"Demo catalog","technicalName":"bad-name!"}`, false},
		{"catalog-update", `{"name":"Demo catalog"}`, true},
	} {
		t.Run(tc.kind+tc.body, func(t *testing.T) {
			findings := ValidateKind(tc.kind, []byte(tc.body), false)
			if (len(findings) == 0) != tc.valid {
				t.Fatalf("valid=%v findings=%v", tc.valid, findings)
			}
		})
	}
}

func TestLiteralRoutesAndBatchSemantics(t *testing.T) {
	op, _ := Find("GET", "/api/v1/executions/summary")
	if op == nil || op.OperationID != "getExecutionSummary" {
		t.Fatal("summary matched ID route")
	}
	body := `{"executionIds":["same","same"],"type":"ABORT","reason":"review"}`
	if findings := ValidateKind("bulk-action", []byte(body), false); len(findings) != 0 {
		t.Fatal(findings)
	}
	bad := strings.Replace(body, "ABORT", "RESET", 1)
	if len(ValidateKind("bulk-action", []byte(bad), false)) == 0 {
		t.Fatal("bulk accepted a single-only action")
	}
	if len(Data.Operations) != 51 {
		t.Fatalf("snapshot has %d operations", len(Data.Operations))
	}
	for _, operation := range Data.Operations {
		for _, p := range operation.Parameters {
			if p.In == "body" && p.Schema == nil {
				t.Fatalf("missing body schema: %s", operation.OperationID)
			}
		}
	}
}
