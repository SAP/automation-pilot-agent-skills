---
name: automation-pilot-executions-api
description: Monitor and manage SAP Automation Pilot executions via API. Use when triggering commands, checking execution status, retrieving logs, aborting or pausing executions, or troubleshooting failed runs.
---

# SAP Automation Pilot Executions API Management

This skill manages command executions in SAP Automation Pilot - triggering, monitoring, controlling, and troubleshooting executions.

## API access

All calls use the `autopi` CLI — see [`../automation-pilot-cli/SKILL.md`](../automation-pilot-cli/SKILL.md) for setup (the `AUTOPI_*` environment variables, supported regional hosts, install, and the full command surface). Add the CLI's `scripts/` to your `PATH` so `autopi` runs directly.

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

All lifecycle operations go through the single `executions action <id> --type <TYPE>`
verb — there is **no** standalone `executions abort` / `pause` / `resume` subcommand.
To abort, use `executions action <id> --type ABORT` (not `executions abort`).

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
autopi executions action $EXEC_ID --type PAUSE --reason 'Blocking on manual verification'

# Resume
autopi executions action $EXEC_ID --type RESUME --reason 'Resuming after verification'

# Retry
autopi executions action $EXEC_ID --type RETRY --reason 'Retrying after transient failure'

# Abort
autopi executions action $EXEC_ID --type ABORT --reason 'Superseded by a newer run'
```

---

# Triggering Executions

**Note:** To retrieve logs after execution, include `feature:logs` in the `tags` of the trigger request. This tag is opt-in and cannot be added retroactively — an execution triggered without it will return no logs from `GET /executions/{id}/logs`.

## Trigger a Command

Add `"feature:dryRun": ""` to the tags to execute without making actual changes.

```bash
# Basic trigger (opt-in feature:logs tag enables log retrieval)
autopi executions trigger --command-id my-catalog:MyCommand:1 \
  --input param1=value1 \
  --input param2=value2 \
  --logs

# With dry run mode (adds feature:dryRun tag)
autopi executions trigger --command-id my-catalog:MyCommand:1 \
  --input param1=value1 \
  --logs \
  --dry-run
```

**All `inputValues` entries must be strings on the wire**, regardless of the declared
parameter type — see
[`../automation-pilot-cli/SKILL.md`](../automation-pilot-cli/SKILL.md#values-are-always-strings-on-the-wire)
for the canonical rule and type→wire table. For example:

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

Lists return metadata only; add `--full` for complete execution objects.

```bash
# List all (up to limit)
autopi executions list --limit 100
# Filter by status
autopi executions list --status FAILED --limit 50
# Filter by command
autopi executions list --command-id my-catalog:MyCommand:1
# Filter by time (epoch milliseconds)
autopi executions list --started-after 1707000000000
```

## Get Execution Details

```bash
EXEC_ID="execution-uuid-here"
autopi executions get $EXEC_ID
```

## Get Execution Summary

```bash
# Get counts by status
autopi executions summary
```

## Get Input/Output

```bash
# Get execution input
autopi executions input $EXEC_ID
# Get execution output (only for FINISHED)
autopi executions output $EXEC_ID
```

---

# Controlling Executions

All actions use the same endpoint: `POST /api/v1/executions/$EXEC_ID/actions`

See the Actions table in the Execution Lifecycle section for valid statuses and expected outcomes.

```bash
EXEC_ID="execution-uuid-here"

# Reset — resets to a previous child command (provide the executor path via --value)
autopi executions action $EXEC_ID --type RESET --value stepAlias --reason 'Retry after fixing the input'

# Comment — add a note to the Action Log (max 10 per execution)
autopi executions action $EXEC_ID --type COMMENT --reason 'Escalated to L2 on-call'

# List action history
autopi executions actions $EXEC_ID
```

---

# Viewing Logs

## Get Execution Logs

```bash
EXEC_ID="execution-uuid-here"

# Get paginated logs
autopi executions logs $EXEC_ID --page 0 --max-page-size 20
# Get logs for specific executor
EXECUTOR_PATH="step1"
autopi executions logs $EXEC_ID --executor $EXECUTOR_PATH
```

---

# Troubleshooting

## Quick Troubleshooting

```bash
EXEC_ID="your-execution-id"

# 1. Get status and error
autopi executions get $EXEC_ID

# 2. Check input used
autopi executions input $EXEC_ID

# 3. Check recent logs
autopi executions logs $EXEC_ID --page 0 --max-page-size 10

# 4. Check output (if FINISHED)
autopi executions output $EXEC_ID
```

## Common Issues

### Execution Stuck in RUNNING

1. Check logs for the current step
2. Look for external dependencies (HTTP timeouts, etc.)
3. Consider aborting and retrying

```bash
autopi executions logs $EXEC_ID --page 0 --max-page-size 20

autopi executions action $EXEC_ID --type ABORT --reason 'Aborting stuck execution'
```

### Execution Failed

1. Get execution details to see error
2. Check logs for the failed step
3. Review input parameters

```bash
autopi executions get $EXEC_ID
autopi executions input $EXEC_ID
```

### Input Required

The execution is waiting for manual input:

```bash
autopi executions get $EXEC_ID
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
autopi executions list --status FAILED --started-after 24h --limit 50
# Multiple statuses
autopi executions list --status RUNNING --status PAUSED
```

---

# Delete Executions

Only completed executions can be deleted (not RUNNING).

```bash
EXEC_ID="execution-uuid-here"
autopi executions delete $EXEC_ID
```

---

# Event Triggers

The command to run is passed as `--command-reference <id>` (sent as a query
parameter — the API rejects it inside the body). The file/`--data` payload is the
**event object only**, not a wrapper containing `commandReference`.

## Trigger via Generic Event

```bash
# event.json is the event payload itself, e.g.:
# { "type": "alert", "message": "High CPU usage detected" }
autopi triggers generic-event --command-reference my-catalog:MyCommand:1 -f event.json

# or inline:
autopi triggers generic-event --command-reference my-catalog:MyCommand:1 \
  --data '{"type":"alert","message":"High CPU usage detected"}'
```

## Trigger via ANS Event

```bash
# ans-event.json is the ANS event payload itself (no commandReference wrapper):
autopi triggers ans-event --command-reference my-catalog:MyCommand:1 -f ans-event.json
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
