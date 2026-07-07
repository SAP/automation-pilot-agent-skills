---
name: automation-pilot-executions-api
description: Monitor and manage SAP Automation Pilot executions via API. Use when triggering commands, checking execution status, retrieving logs, aborting or pausing executions, or troubleshooting failed runs.
---

# SAP Automation Pilot Executions API Management

This skill manages command executions in SAP Automation Pilot - triggering, monitoring, controlling, and troubleshooting executions.

## Prerequisites

1. Set the following **required** environment variables:

```bash
export AUTOPI_HOSTNAME="emea.autopilot.cloud.sap"
export AUTOPI_USERNAME="your-username"
export AUTOPI_PASSWORD="your-password"
export AUTOPI_DEFAULT_CATALOG="mycommands-<<<TENANT_ID>>>"
```

2. Ensure `curl` is available in your environment.

---

# Execution Lifecycle

## Statuses

| Status | Description | Active or Terminal |
|--------|-------------|-------------------|
| `RUNNING` | Execution is currently running. During delays (initial delay, retry backoff, repeat interval) the execution stays RUNNING while individual executors show their delay reason. | Active |
| `PAUSED` | Manually paused — must be resumed to continue. | Active |
| `INPUT_REQUIRED` | Waiting for a user choice (Confirm action required). | Active |
| `FAILED` | Failed due to an error or failed validation. Can be retried or reset. | Active |
| `FINISHED` | Completed successfully. | **Terminal** |
| `ABORTED` | Canceled by the user. | **Terminal** |
| `SUSPENDED` | Canceled by Automation Pilot (runtime limit exceeded, tenant execution limit reached, or execution/output size limit exceeded). | **Terminal** |

**Terminal** statuses (FINISHED, ABORTED, SUSPENDED) mean the execution is complete and no further actions can be applied. Completed executions can be retriggered and deleted.

**Active** statuses (RUNNING, PAUSED, INPUT_REQUIRED, FAILED) mean the execution is still live and can be acted upon.

## Actions

Each action is only valid from specific statuses — applying an action from the wrong status returns HTTP 409.

| Action | Valid from | Result |
|--------|-----------|--------|
| **PAUSE** | `RUNNING` | → `PAUSED` |
| **RESUME** | `PAUSED` | → `RUNNING` |
| **RETRY** | `FAILED` | → `RUNNING`. Re-attempts the execution, preserving progress for provided composite commands. |
| **RESET** | `FAILED`, `PAUSED`, `INPUT_REQUIRED` | → `RUNNING` from a chosen previous child command. Not available for direct executions of provided commands. For provided composite commands, starts the whole command from the beginning (unlike RETRY which preserves progress). |
| **ABORT** | `PAUSED`, `FAILED` | → `ABORTED` |
| **COMMENT** | Any | Adds a comment to the execution's Action Log (max 10 comments per execution). |

```bash
EXEC_ID="execution-uuid-here"

# Pause
curl -s -X POST -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"type": "PAUSE", "reason": "Blocking on manual verification"}' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/actions"

# Resume
curl -s -X POST -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"type": "RESUME", "reason": "Resuming after verification"}' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/actions"

# Retry
curl -s -X POST -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"type": "RETRY", "reason": "Retrying after transient failure"}' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/actions"

# Abort
curl -s -X POST -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"type": "ABORT", "reason": "Superseded by a newer run"}' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/actions"
```

---

# Triggering Executions

**Note:** To retrieve logs after execution, include `feature:logs` in the `tags` of the trigger request. This tag is opt-in and cannot be added retroactively — an execution triggered without it will return no logs from `GET /executions/{id}/logs`.

## Trigger a Command

Add `"feature:dryRun": ""` to the tags to execute without making actual changes.

```bash
# Basic trigger (opt-in feature:logs tag enables log retrieval)
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "commandId": "my-catalog:MyCommand:1",
    "inputValues": {
      "param1": "value1",
      "param2": "value2"
    },
    "tags": {"feature:logs": ""}
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions"

# With dry run mode (adds feature:dryRun tag)
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "commandId": "my-catalog:MyCommand:1",
    "inputValues": {"param1": "value1"},
    "tags": {"feature:logs": "", "feature:dryRun": ""}
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions"
```

**All `inputValues` entries must be strings on the wire**, regardless of the declared parameter type. The runtime converts each value to its declared type inside the executor:

```json
"inputValues": {
  "appName": "my-app",
  "instances": "3",
  "enabled": "true",
  "regions": "[\"cf-eu10\",\"cf-us10\"]",
  "config": "{\"maxRetries\":5}"
}
```

---

# Monitoring Executions

## List Executions

```bash
# List all (up to limit)
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions?limit=100"
# Filter by status
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions?status=FAILED&limit=50"
# Filter by command
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions?commandId=my-catalog:MyCommand:1"
# Filter by time (epoch milliseconds)
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions?startedAfter=1707000000000"
```

## Get Execution Details

```bash
EXEC_ID="execution-uuid-here"
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID"
```

## Get Execution Summary

```bash
# Get counts by status
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/summary"
```

## Get Input/Output

```bash
# Get execution input
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/input"
# Get execution output (only for FINISHED)
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/output"
```

---

# Controlling Executions

All actions use the same endpoint: `POST /api/v1/executions/$EXEC_ID/actions`

See the Actions table in the Execution Lifecycle section for valid statuses and expected outcomes.

```bash
EXEC_ID="execution-uuid-here"

# Reset — resets to a previous child command (provide the executor path via "value")
curl -s -X POST -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"type": "RESET", "value": "stepAlias", "reason": "Retry after fixing the input"}' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/actions"

# Comment — add a note to the Action Log (max 10 per execution)
curl -s -X POST -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"type": "COMMENT", "reason": "Escalated to L2 on-call"}' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/actions"

# List action history
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/actions"
```

---

# Viewing Logs

## Get Execution Logs

```bash
EXEC_ID="execution-uuid-here"

# Get paginated logs
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/logs?page=0&maxPageSize=20"
# Get logs for specific executor
EXECUTOR_PATH="step1"
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/logs/$EXECUTOR_PATH"
```

---

# Troubleshooting

## Quick Troubleshooting

```bash
EXEC_ID="your-execution-id"

# 1. Get status and error
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID"

# 2. Check input used
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/input"

# 3. Check recent logs
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/logs?page=0&maxPageSize=10"

# 4. Check output (if FINISHED)
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/output"
```

## Common Issues

### Execution Stuck in RUNNING

1. Check logs for the current step
2. Look for external dependencies (HTTP timeouts, etc.)
3. Consider aborting and retrying

```bash
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/logs?page=0&maxPageSize=20"

curl -s -X POST -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"type": "ABORT", "reason": "Aborting stuck execution"}' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/actions"
```

### Execution Failed

1. Get execution details to see error
2. Check logs for the failed step
3. Review input parameters

```bash
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID"
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/input"
```

### Input Required

The execution is waiting for manual input:

```bash
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID"
# Inspect the response — check the suspendedStep field
```

### No Logs Available

**Cause:** The execution was triggered without the `feature:logs` tag, so logging was not enabled.
**Solution:** Always include `"feature:logs": ""` in the `tags` object when triggering. Logs cannot be retrieved retroactively for executions that were triggered without this tag. Re-trigger the execution with the tag included.

---

# Filtering & Pagination

## Query Parameters for List Executions

| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | array | Filter by status (RUNNING, FAILED, FINISHED, etc.) |
| `commandId` | array | Filter by command ID |
| `executionId` | array | Filter by execution ID |
| `startedAfter` | integer | Unix epoch milliseconds |
| `startedBefore` | integer | Unix epoch milliseconds |
| `tag` | array | Key-value pairs |
| `limit` | integer | Max results (1-1000, default 1000) |

## Examples

```bash
# Last 24 hours, failed only
YESTERDAY=$(($(date +%s) * 1000 - 86400000))
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions?status=FAILED&startedAfter=$YESTERDAY&limit=50"
# Multiple statuses
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions?status=RUNNING&status=PAUSED"
```

---

# Delete Executions

Only completed executions can be deleted (not RUNNING).

```bash
EXEC_ID="execution-uuid-here"
curl -s -X DELETE -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID"
```

---

# Event Triggers

## Trigger via Generic Event

```bash
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "commandReference": "my-catalog:MyCommand:1",
    "event": {
      "type": "alert",
      "message": "High CPU usage detected"
    }
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/triggers/generic-event"
```

## Trigger via ANS Event

```bash
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "commandReference": "my-catalog:MyCommand:1",
    "event": {...ANS event payload...}
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/triggers/ans-event"
```

---

# Error Handling

## HTTP Status Codes

| Code | Description |
|------|-------------|
| `200` | Success |
| `202` | Accepted (trigger/action) |
| `204` | No Content (delete) |
| `400` | Bad Request |
| `401` | Unauthorized |
| `403` | Forbidden |
| `404` | Not Found |
| `409` | Conflict (invalid state) |
| `413` | Payload Too Large |
| `429` | Rate Limited |

## Permissions Required

| Operation | Permission |
|-----------|-----------|
| List/Get executions | `Read` |
| Delete executions | `Write` |
| Trigger/Actions | `Execute` |

---

# Examples

## Example 1: Trigger and monitor a command

User: "Run my health check command"

1. POST to `/api/v1/executions` with `commandId`, input values, and `"tags": {"feature:logs": ""}`
2. Capture the execution ID from the response
3. GET `/api/v1/executions/$EXEC_ID` and poll until `status` is no longer `RUNNING`
4. Report the final status and output

## Example 2: Troubleshoot a failed execution

User: "My execution failed, what went wrong?"

1. GET `/api/v1/executions/$EXEC_ID` — extract `status`, `error`, `progressMessage`
2. GET `/api/v1/executions/$EXEC_ID/input` — verify the parameters that were used
3. GET `/api/v1/executions/$EXEC_ID/logs?page=0&maxPageSize=20` — find the failed step
4. Report the root cause from the error message and log output

## Example 3: Find and abort stuck executions

User: "Are there any stuck executions? Abort them."

1. GET `/api/v1/executions?status=RUNNING` to list all currently running executions
2. For each one, check `startedAt` to identify those running unexpectedly long
3. For each stuck execution, POST `{"type": "ABORT", "reason": "Stuck execution"}` to `/api/v1/executions/$EXEC_ID/actions`
4. Confirm each abort returned HTTP 202
