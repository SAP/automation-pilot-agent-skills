---
name: manage-sap-automation-pilot-executions-via-api
description: This skill should be used when the user asks to "list executions", "trigger command execution", "check execution status", "abort execution", "pause execution", "get execution logs", "troubleshoot execution", "monitor autopi executions", "execution failed", "execution output", or discusses SAP Automation Pilot execution management, monitoring, and troubleshooting.
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

# Executable Scripts

Reusable shell scripts are available in `scripts/` directory.

## Quick Reference

| Script | Description |
|--------|-------------|
| `autopi-executions.sh` | List, get, delete executions |
| `autopi-exec-trigger.sh` | Trigger new command executions |
| `autopi-exec-actions.sh` | Abort, pause, resume executions |
| `autopi-exec-logs.sh` | View execution logs |
| `autopi-exec-status.sh` | Quick status check & summary |
| `autopi-exec-troubleshoot.sh` | Combined troubleshooting helper |

## Usage Examples

```bash
# List recent executions
./scripts/autopi-executions.sh list --limit 10

# List failed executions
./scripts/autopi-exec-status.sh --failed

# Trigger a command execution
./scripts/autopi-exec-trigger.sh "my-catalog:MyCommand:1" '{"param": "value"}'

# Check execution status
./scripts/autopi-exec-status.sh <execution-id>

# Abort a running execution
./scripts/autopi-exec-actions.sh abort <execution-id>

# View execution logs
./scripts/autopi-exec-logs.sh <execution-id>

# Full troubleshooting report
./scripts/autopi-exec-troubleshoot.sh <execution-id>
```

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

**Note:** All executions triggered via the scripts automatically include the `feature:logs` tag, which enables detailed execution logging.

## Dry Run Mode

Use `--dry-run` to execute in dry run mode (no actual changes will be made):

```bash
# Trigger in dry run mode
./scripts/autopi-exec-trigger.sh "my-catalog:MyCommand:1" '{"param": "value"}' --dry-run
```

## Trigger a Command

```bash
# Basic trigger (includes feature:logs tag automatically)
curl -s -X POST \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d '{
    "commandId": "my-catalog:MyCommand:1",
    "input": {
      "param1": "value1",
      "param2": "value2"
    },
    "tags": {"feature:logs": ""}
  }' \
  "https://$HOST/api/v1/executions" | jq .

# Using the script (feature:logs tag included automatically)
./scripts/autopi-exec-trigger.sh "my-catalog:MyCommand:1" '{"param1": "value1"}'

# With dry run mode (adds feature:dryRun tag)
./scripts/autopi-exec-trigger.sh "my-catalog:MyCommand:1" '{"param1": "value1"}' --dry-run
```

## Trigger with Input File

```bash
# Create input file
echo '{"param1": "value1", "param2": "value2"}' > input.json

# Trigger using file
./scripts/autopi-exec-trigger.sh "my-catalog:MyCommand:1" --input-file input.json
```

---

# Monitoring Executions

## List Executions

```bash
# List all (up to limit)
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions?limit=100" | jq .

# Filter by status
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions?status=FAILED&limit=50" | jq .

# Filter by command
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions?commandId=my-catalog:MyCommand:1" | jq .

# Filter by time (epoch milliseconds)
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions?startedAfter=1707000000000" | jq .
```

## Get Execution Details

```bash
EXEC_ID="execution-uuid-here"
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions/$EXEC_ID" | jq .
```

## Get Execution Summary

```bash
# Get counts by status
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions/summary" | jq .
```

## Get Input/Output

```bash
# Get execution input
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions/$EXEC_ID/input" | jq .

# Get execution output (only for FINISHED)
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions/$EXEC_ID/output" | jq .
```

---

# Controlling Executions

## Abort Execution

```bash
EXEC_ID="execution-uuid-here"
curl -s -X POST \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d '{"action": "abort"}' \
  "https://$HOST/api/v1/executions/$EXEC_ID/actions" | jq .

# Using script
./scripts/autopi-exec-actions.sh abort $EXEC_ID
```

## Pause Execution

```bash
curl -s -X POST \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d '{"action": "pause"}' \
  "https://$HOST/api/v1/executions/$EXEC_ID/actions" | jq .

# Using script
./scripts/autopi-exec-actions.sh pause $EXEC_ID
```

## Resume Execution

```bash
curl -s -X POST \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d '{"action": "resume"}' \
  "https://$HOST/api/v1/executions/$EXEC_ID/actions" | jq .

# Using script
./scripts/autopi-exec-actions.sh resume $EXEC_ID
```

## List Actions

```bash
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions/$EXEC_ID/actions" | jq .
```

---

# Viewing Logs

## Get Execution Logs

```bash
EXEC_ID="execution-uuid-here"

# Get paginated logs
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions/$EXEC_ID/logs?page=0&maxPageSize=20" | jq .

# Get logs for specific executor
EXECUTOR_PATH="step1"
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions/$EXEC_ID/logs/$EXECUTOR_PATH" | jq .

# Using script
./scripts/autopi-exec-logs.sh $EXEC_ID
./scripts/autopi-exec-logs.sh $EXEC_ID --page 1
./scripts/autopi-exec-logs.sh $EXEC_ID step1
```

---

# Troubleshooting

## Quick Troubleshooting Script

```bash
# Get comprehensive troubleshooting info
./scripts/autopi-exec-troubleshoot.sh <execution-id>

# Shows:
# - Status and timing
# - Command that was executed
# - Input parameters
# - Output (if finished)
# - Error messages (if failed)
# - Recent logs
# - Suggested next steps
```

## Common Issues

### Execution Stuck in RUNNING

1. Check logs for the current step
2. Look for external dependencies (HTTP timeouts, etc.)
3. Consider aborting and retrying

```bash
./scripts/autopi-exec-logs.sh $EXEC_ID
./scripts/autopi-exec-actions.sh abort $EXEC_ID
```

### Execution Failed

1. Get execution details to see error
2. Check logs for the failed step
3. Review input parameters

```bash
./scripts/autopi-exec-troubleshoot.sh $EXEC_ID
```

### Input Required

The execution is waiting for manual input:

```bash
# Check what input is required
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions/$EXEC_ID" | jq '.suspendedStep'
```

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
curl -s -u "$USER:$PASS" \
  "https://$HOST/api/v1/executions?status=FAILED&startedAfter=$YESTERDAY&limit=50" | jq .

# Multiple statuses
curl -s -u "$USER:$PASS" \
  "https://$HOST/api/v1/executions?status=RUNNING&status=PAUSED" | jq .
```

---

# Delete Executions

Only completed executions can be deleted (not RUNNING).

```bash
EXEC_ID="execution-uuid-here"
curl -s -X DELETE -u "$USER:$PASS" "https://$HOST/api/v1/executions/$EXEC_ID"
```

---

# Event Triggers

## Trigger via Generic Event

```bash
curl -s -X POST \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d '{
    "commandReference": "my-catalog:MyCommand:1",
    "event": {
      "type": "alert",
      "message": "High CPU usage detected"
    }
  }' \
  "https://$HOST/api/v1/triggers/generic-event" | jq .
```

## Trigger via ANS Event

```bash
curl -s -X POST \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d '{
    "commandReference": "my-catalog:MyCommand:1",
    "event": {...ANS event payload...}
  }' \
  "https://$HOST/api/v1/triggers/ans-event" | jq .
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
