---
name: automation-pilot-executions-api
description: Monitor and manage SAP Automation Pilot executions via API. Use when triggering commands, checking execution status, retrieving logs, aborting or pausing executions, or troubleshooting failed runs.
version: 1.0.0
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

2. Ensure `curl` and `jq` are available in your environment.

---

# Execution Lifecycle

## Statuses

| Status | Description |
|--------|-------------|
| `RUNNING` | Execution in progress |
| `FINISHED` | Completed successfully |
| `FAILED` | Completed with errors |
| `ABORTED` | Manually aborted |
| `PAUSED` | Temporarily paused |
| `SUSPENDED` | Suspended state |
| `INPUT_REQUIRED` | Waiting for input |

## Status Flow

```
        ┌─────────┐
        │ RUNNING │
        └────┬────┘
             │
    ┌────────┼────────┬──────────┐
    ▼        ▼        ▼          ▼
┌────────┐ ┌────┐ ┌───────┐ ┌────────┐
│FINISHED│ │FAIL│ │ABORTED│ │ PAUSED │
└────────┘ └────┘ └───────┘ └───┬────┘
                                │
                                ▼
                            ┌───────┐
                            │RUNNING│ (resume)
                            └───────┘
```

---

# Triggering Executions

**Note:** All executions triggered via the API automatically include the `feature:logs` tag, which enables detailed execution logging.

## Dry Run Mode

Add the `feature:dryRun` tag to execute without making actual changes:

## Trigger a Command

```bash
# Basic trigger (includes feature:logs tag automatically)
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "commandId": "my-catalog:MyCommand:1",
    "input": {
      "param1": "value1",
      "param2": "value2"
    },
    "tags": {"feature:logs": ""}
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions" | jq .

# With dry run mode (adds feature:dryRun tag)
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "commandId": "my-catalog:MyCommand:1",
    "input": {"param1": "value1"},
    "tags": {"feature:logs": "", "feature:dryRun": ""}
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions" | jq .
```

---

# Monitoring Executions

## List Executions

```bash
# List all (up to limit)
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions?limit=100" | jq .

# Filter by status
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions?status=FAILED&limit=50" | jq .

# Filter by command
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions?commandId=my-catalog:MyCommand:1" | jq .

# Filter by time (epoch milliseconds)
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions?startedAfter=1707000000000" | jq .
```

## Get Execution Details

```bash
EXEC_ID="execution-uuid-here"
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID" | jq .
```

## Get Execution Summary

```bash
# Get counts by status
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/summary" | jq .
```

## Get Input/Output

```bash
# Get execution input
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/input" | jq .

# Get execution output (only for FINISHED)
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/output" | jq .
```

---

# Controlling Executions

## Abort Execution

```bash
EXEC_ID="execution-uuid-here"
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"action": "abort"}' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/actions" | jq .
```

## Pause Execution

```bash
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"action": "pause"}' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/actions" | jq .
```

## Resume Execution

```bash
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"action": "resume"}' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/actions" | jq .
```

## List Actions

```bash
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/actions" | jq .
```

---

# Viewing Logs

## Get Execution Logs

```bash
EXEC_ID="execution-uuid-here"

# Get paginated logs
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/logs?page=0&maxPageSize=20" | jq .

# Get logs for specific executor
EXECUTOR_PATH="step1"
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/logs/$EXECUTOR_PATH" | jq .
```

---

# Troubleshooting

## Quick Troubleshooting

```bash
EXEC_ID="your-execution-id"

# 1. Get status and error
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID" | \
  jq '{status, error, progressMessage, commandId}'

# 2. Check input used
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/input" | jq .

# 3. Check recent logs
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/logs?page=0&maxPageSize=10" | jq '.logs'

# 4. Check output (if FINISHED)
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/output" | jq .
```

## Common Issues

### Execution Stuck in RUNNING

1. Check logs for the current step
2. Look for external dependencies (HTTP timeouts, etc.)
3. Consider aborting and retrying

```bash
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/logs?page=0&maxPageSize=20" | jq '.logs'

curl -s -X POST -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"action": "abort"}' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/actions" | jq .
```

### Execution Failed

1. Get execution details to see error
2. Check logs for the failed step
3. Review input parameters

```bash
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID" | jq '{status, error}'
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/input" | jq .
```

### Input Required

The execution is waiting for manual input:

```bash
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID" | jq '.suspendedStep'
```

### HTTP 409 on Abort or Pause

**Cause:** The execution is no longer in a state that allows that action — it already finished, failed, or was aborted before your request arrived.
**Solution:** Check the current status first, then only send the action if the execution is still in a controllable state:

```bash
STATUS=$(curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID" | jq -r '.status')
echo "Current status: $STATUS"
# Only abort if RUNNING or PAUSED
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
  "https://$AUTOPI_HOSTNAME/api/v1/executions?status=FAILED&startedAfter=$YESTERDAY&limit=50" | jq .

# Multiple statuses
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions?status=RUNNING&status=PAUSED" | jq .
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
  "https://$AUTOPI_HOSTNAME/api/v1/triggers/generic-event" | jq .
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
  "https://$AUTOPI_HOSTNAME/api/v1/triggers/ans-event" | jq .
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
| `412` | Precondition Failed (ETag mismatch) |
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
3. For each stuck execution, POST `{"action": "abort"}` to `/api/v1/executions/$EXEC_ID/actions`
4. Confirm each abort returned HTTP 202
