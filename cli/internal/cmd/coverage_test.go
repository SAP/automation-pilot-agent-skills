package cmd

import (
	"github.com/SAP/automation-pilot-agent-skills/cli/internal/contract"
	"io"
	"net/http"
	"testing"
)

// Every operation in the pinned Swagger must reach its wrapper's actual request.
// Deprecated parameters are intentionally raw-only; MCP/connection are separate APIs.
func TestContractFeaturesAllSwaggerOperationsHaveWorkingWrappers(t *testing.T) {
	input := `{"name":"Hello","catalog":"c","version":1,"keys":{},"values":{}}`
	catalog := `{"name":"Demo","technicalName":"Demo"}`
	webhook := `{"name":"Event","type":"GENERIC","commandId":"c:Hello:1"}`
	schedule := `{"commandId":"c:Hello:1","schedule":{"hourly":{"minutes":[0]}}}`
	ans := `{"eventType":"test","subject":"test","body":"test","category":"ALERT","severity":"INFO","resource":{"resourceName":"fixture","resourceType":"test"}}`
	cases := map[string][]string{
		"listCatalogs": {"catalogs", "list"}, "getCatalog": {"catalogs", "get", "id"}, "createCatalog": {"catalogs", "create", "--data", catalog}, "updateCatalog": {"catalogs", "update", "id", "--data", `{"name":"Demo"}`}, "deleteCatalog": {"catalogs", "delete", "id"},
		"listCommands": {"commands", "list"}, "getCommandIds": {"commands", "ids"}, "getCommand": {"commands", "get", "c:Hello:1"}, "createCommand": {"commands", "create", "--data", validCommand}, "updateCommand": {"commands", "update", "c:Hello:1", "--data", validCommand}, "deleteCommand": {"commands", "delete", "c:Hello:1"},
		"releaseCommand": {"commands", "release", "c:Hello:1"}, "deprecateCommand": {"commands", "deprecate", "c:Hello:1"}, "restoreCommand": {"commands", "restore", "c:Hello:1"}, "createCommands": {"commands", "bulk-create", "--data", `{"commands":[` + validCommand + `]}`},
		"listInputs": {"inputs", "list"}, "getInput": {"inputs", "get", "c:Hello:1"}, "createInput": {"inputs", "create", "--data", input}, "updateInput": {"inputs", "update", "c:Hello:1", "--data", input}, "patchInput": {"inputs", "patch", "c:Hello:1", "--data", `{"description":"changed"}`}, "deleteInput": {"inputs", "delete", "c:Hello:1"},
		"releaseInput": {"inputs", "release", "c:Hello:1"}, "deprecateInput": {"inputs", "deprecate", "c:Hello:1"}, "restoreInput": {"inputs", "restore", "c:Hello:1"}, "createInputs": {"inputs", "bulk-create", "--data", `{"inputs":[` + input + `]}`},
		"listWebhooks": {"webhooks", "list"}, "getWebhook": {"webhooks", "get", "id"}, "createWebhook": {"webhooks", "create", "--data", webhook}, "updateWebhook": {"webhooks", "update", "id", "--data", webhook}, "deleteWebhook": {"webhooks", "delete", "id"}, "executeWebhook": {"webhooks", "trigger", "id"},
		"listExecutions": {"executions", "list"}, "triggerExecution": {"executions", "trigger", "--command-id", "c:Hello:1"}, "getExecution": {"executions", "get", "run"}, "deleteExecution": {"executions", "delete", "run"}, "getInputExecution": {"executions", "input", "run"}, "getOutput": {"executions", "output", "run"},
		"listActions": {"executions", "actions", "run"}, "requestAction": {"executions", "action", "run", "--type", "PAUSE", "--reason", "test"}, "requestBulkAction": {"executions", "bulk-action", "--execution-id", "run", "--type", "PAUSE", "--reason", "test"},
		"triggerAnsEventExecution": {"triggers", "ans-event", "--command-reference", "c:Hello:1", "--data", ans}, "triggerGenericEventExecution": {"triggers", "generic-event", "--command-reference", "c:Hello:1"},
		"getExecutionLog": {"executions", "logs", "run", "--executor", "step"}, "listExecutionLogs": {"executions", "logs", "run"}, "getExecutionSummary": {"executions", "summary"},
		"listScheduledExecutions": {"scheduled-executions", "list"}, "createScheduledExecution": {"scheduled-executions", "create", "--data", schedule}, "getScheduledExecution": {"scheduled-executions", "get", "id"}, "updateScheduledExecution": {"scheduled-executions", "update", "id", "--data", schedule}, "deleteScheduledExecution": {"scheduled-executions", "delete", "id"}, "listTimeZones": {"scheduled-executions", "time-zones"},
	}
	// The Content and Executions documents both use getInput as an operationId.
	for _, op := range contract.Data.Operations {
		t.Run(op.Method+op.Path, func(t *testing.T) {
			args := cases[op.OperationID]
			if op.OperationID == "getInput" && op.Path == "/executions/{executionId}/input" {
				args = cases["getInputExecution"]
			}
			if args == nil {
				t.Fatalf("uncovered operation %s", op.OperationID)
			}
			calls := 0
			code, _, stderr := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				actual, _ := contract.Find(r.Method, r.URL.EscapedPath())
				if actual == nil || actual.Path != op.Path {
					t.Errorf("expected %s %s, got %s %s", op.Method, op.Path, r.Method, r.URL)
				}
				io.WriteString(w, `{"id":"run","results":[{"executionId":"run","succeeded":true}]}`)
			}, "", args...)
			if code != 0 || calls != 1 {
				t.Fatalf("code=%d calls=%d err=%s", code, calls, stderr)
			}
		})
	}
}
