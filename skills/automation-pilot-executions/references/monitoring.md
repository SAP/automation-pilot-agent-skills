# Monitoring and Inspecting Executions

Read this when following a run, interpreting its state, or inspecting its input, output, or executor logs.

## Status lifecycle

| Status | Meaning | Monitoring action |
|---|---|---|
| `RUNNING` | Executing now. Initial delays, retry backoff, and repeat intervals remain `RUNNING`. | Continue while progress or expected waiting is visible. |
| `PAUSED` | A user paused the run in the product. | Report attention; use CLI `action --type RESUME` when the user requests resumption. |
| `INPUT_REQUIRED` | A Confirm step is waiting for a choice. | Hand back with the requested choice; responding requires the product or raw API. |
| `FAILED` | An error or failed validation stopped the run. The product can retry or reset it. | Stop polling and diagnose or report. |
| `FINISHED` | The command completed and its output is available. | Stop and report. |
| `ABORTED` | A user or Auto-Abort cancelled the run. | Stop and report. |
| `SUSPENDED` | Automation Pilot cancelled the run after a runtime, tenant-execution, or size limit. | Stop; treat it as a platform-enforced cancellation, not a command error. |

`FINISHED`, `ABORTED`, and `SUSPENDED` are completed statuses. `FAILED` remains active and can be retried, reset, skipped, or aborted, but passive monitoring should stop so the failure can be diagnosed or handed back. `PAUSED` and `INPUT_REQUIRED` also need attention rather than unchanged polling.

## Polling

Unless the user gives a different monitoring deadline:

- Wait 2 seconds before the first `autopi executions get` call, then poll every 2 seconds through 30 seconds.
- Poll every 10 seconds from 30 seconds through 5 minutes.
- Poll every 30 seconds after 5 minutes.
- After 10 minutes of foreground monitoring, return the execution ID and latest state unless the user asked to wait longer.

Stop on `FINISHED`, `ABORTED`, `SUSPENDED`, or `FAILED`, and hand back on `PAUSED` or `INPUT_REQUIRED`. After three consecutive polling-call failures, report the last error and execution ID. A polling failure does not authorize a replacement run.

Before calling a `RUNNING` execution stuck, inspect `progressMessage`, `currentExecutorPath`, and the per-executor states and timestamps. Expected initial delay, retry backoff, an external timeout, or a repeat interval can explain a period without visible progress.

## Execution details

`autopi executions get` returns summary fields plus the top-level `error`, `userChoice`, `currentExecutorPath`, `outputValues`, and `executors`. Each executor includes its alias, command ID, children, state, and timestamps. Use this structure to identify the last executor that ran and the first one that did not complete.

Pull the heavier payloads only when they bear on the request:

- `autopi executions input <id>` returns the resolved input the run used.
  Decode its string values with the triggering rules loaded directly by the skill.
  Prefer key metadata and presence; avoid fetching secret payloads when metadata answers the question. Never reproduce sensitive values.
- `autopi executions output <id>` returns the persisted endpoint response; declared command outputs are under `values`. The summary's `outputValues` may still be null, so use the fetched output object as the source of truth. `FINISHED` has complete output; `ABORTED` and `SUSPENDED` may have partial output. A `FAILED` run may have no output.
- `autopi executions logs <id>` takes an executor path via `--executor`. Use `currentExecutorPath` or the relevant path from the execution details; the root executor uses `""`. Logs exist only when logging was enabled before the run and remain available for the product's retention window.
