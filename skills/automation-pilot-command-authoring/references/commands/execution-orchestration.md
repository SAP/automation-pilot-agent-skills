# Execution Orchestration

Start, await, and inspect separate executions from inside a parent execution. Use when
a step must run another command as its own execution, to fan out async work or hand off to a
long-running flow the parent should not block on.

For running a sub-command inside the current execution, prefer a plain sub-command executor.

## Contents

- [When to use](#when-to-use)
- [TriggerExecution](#triggerexecution)
- [AwaitExecution](#awaitexecution)
- [GetExecutionStatus](#getexecutionstatus)
- [GetExecutionInput / GetExecutionOutput](#getexecutioninput--getexecutionoutput)
- [Child lineage](#child-lineage)
- [Minimal snippets](#minimal-snippets)

## When to use

- To start one child now without blocking, use `TriggerExecution:1`. It returns the child ID immediately.
- To start now, then block until it ends, use `TriggerExecution:1` followed by `AwaitExecution:1` on the returned ID.
- To read an existing execution's status, input, or output, use `GetExecutionStatus:1`, `GetExecutionInput:1`, or `GetExecutionOutput:1`, targeting any execution ID the parent holds.

## TriggerExecution

`utils-sapcp:TriggerExecution:1` starts a command as its own child execution and returns at once,
without waiting for it. Follow with `AwaitExecution` to block on the result.

**Inputs**

| Field | Required | Notes |
|-------|----------|-------|
| `command` | yes | ID of the command to run. Must be a literal, not a dynamic expression. |
| `values` | no | Inputs for the child, matching its input contract (object). |
| `tags` | no | Tags stamped on the child (object). |

**Output**: `execution` (string): the child ID. Read it as `$(.trigger.output.execution)` and feed it
to the commands below.

## AwaitExecution

`utils-sapcp:AwaitExecution:1` polls a target execution until it reaches an expected status or the
`deadline` runs out.

**Inputs**

| Field | Required | Default | Notes |
|-------|----------|---------|-------|
| `execution` | yes | — | Target execution ID. |
| `statuses` | no | `["FINISHED"]` | End-states that stop the poll. Add `"FAILED"`/`"ABORTED"` to stop on any terminal state, not only success. |
| `deadline` | no | `5` | Minutes to poll before the step fails (**1–10080**). |
| `succeedOnNotFound` | no | `false` | `true` → a missing target gives `found:false` instead of failing. |

**Outputs**: `found` (boolean) and `status` (string, e.g. `FINISHED`).

**Resolving is not child success.** The step stops as soon as the child reaches any status in
`statuses`, so with the default only `FINISHED` ends the poll. Other terminal states run out the
`deadline` and fail the step. To handle failure yourself, include every terminal state and branch on
the returned `status`.

## GetExecutionStatus

`utils-sapcp:GetExecutionStatus:1` is a point-in-time status read. Unlike `AwaitExecution` it does not
wait.

**Inputs**

| Field | Required | Default | Notes |
|-------|----------|---------|-------|
| `execution` | yes | — | Target execution ID. |
| `succeedOnNotFound` | no | `false` | `true` → a missing target gives `found:false` instead of failing. |

**Outputs**: `found` (boolean) and `status` (string).

## GetExecutionInput / GetExecutionOutput

`utils-sapcp:GetExecutionInput:1` and `utils-sapcp:GetExecutionOutput:1` read the input or output of
a target execution.

**Input**: `execution` (string, required): target execution ID.

**Output**: `input` (GetExecutionInput) or `output` (GetExecutionOutput), a typed object. Read a
field directly: `$(.childOutput.output.output.message)`. Sensitive keys come back blank, so never
route a secret through these.

`GetExecutionOutput` fails if the target has not produced output yet. Gate it behind an
`AwaitExecution` (or `GetExecutionStatus` + `when`) so the child is terminal first.

## Minimal snippets

Trigger-and-await, then read the child's output:

```json
{
    "execute": "utils-sapcp:TriggerExecution:1",
    "alias": "trigger",
    "input": {
        "command": "utils-sapcp:Void:1",
        "values": "{\"message\":\"$(.execution.input.message)\"}"
    }
},
{
    "execute": "utils-sapcp:AwaitExecution:1",
    "alias": "await",
    "input": {
        "execution": "$(.trigger.output.execution)",
        "statuses": "[\"FINISHED\",\"ABORTED\"]",
        "deadline": "30"
    }
},
{
    "execute": "utils-sapcp:GetExecutionOutput:1",
    "alias": "childOutput",
    "input": {
        "execution": "$(.trigger.output.execution)"
    }
}
```

The child output is a typed object: `$(.childOutput.output.output.message)`.
