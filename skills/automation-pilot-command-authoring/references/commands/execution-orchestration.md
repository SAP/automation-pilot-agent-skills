# Execution Orchestration

Start, await, and inspect separate executions from inside a parent execution. Use when
a step must run another command as its own execution, to fan out async work or hand off to a
long-running flow the parent should not block on.

For running a sub-command inside the current execution, prefer a plain sub-command executor.

Resolve these executor contracts and the child command with `autopi commands contract` before
tenant use. The tables are reference snapshots. For local-only work, record unavailable
contracts and output representations as assumptions.

## Contents

- [When to use](#when-to-use)
- [TriggerExecution](#triggerexecution)
- [AwaitExecution](#awaitexecution)
- [GetExecutionStatus](#getexecutionstatus)
- [GetExecutionInput / GetExecutionOutput](#getexecutioninput--getexecutionoutput)
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
| `values` | no | Inputs for the child, matching its input contract; serialize the object as a JSON string. |
| `tags` | no | Tags stamped on the child; serialize the object as a JSON string. |

**Output**: `execution` (string): the child ID. Read it as `$(.trigger.output.execution)` and feed it
to the commands below.

Triggering repeats the child's real effects. Keep parent retry/repeat controls off the trigger
unless launching another child is intended and repeat-safe. If a trigger response is ambiguous,
inspect existing execution evidence before launching a replacement.

## AwaitExecution

`utils-sapcp:AwaitExecution:1` polls a target execution until it reaches an expected status or the
`deadline` runs out.

**Inputs**

| Field | Required | Default | Notes |
|-------|----------|---------|-------|
| `execution` | yes | — | Target execution ID. |
| `statuses` | no | `["FINISHED"]` | States that stop the poll; pass the array as a JSON string. Include failure/cancellation states when the parent must handle them immediately. |
| `deadline` | no | `5` | Minutes to poll before the step fails (**1–10080**). |
| `succeedOnNotFound` | no | `false` | `true` → a missing target gives `found:false` instead of failing. |

**Outputs**: `found` (boolean) and `status` (string, e.g. `FINISHED`).

**Resolving is not child success.** The step stops as soon as the child reaches any status in
`statuses`, so with the default only `FINISHED` ends the poll. States outside the accepted set
leave the wait running until its deadline. To handle failure yourself, include the relevant
failure/cancellation states and branch on
the returned `status`. `FAILED` is an attention state that can be retried; `ABORTED` and
`SUSPENDED` are completed cancellations. A passive wait may also need to return for `PAUSED`
or `INPUT_REQUIRED`, rather than consume its deadline while a human action is required.

After awaiting, validate the observed status or route it to an explicit handling branch. A
successful await means the target reached an accepted state; only `FINISHED` establishes
successful child completion. Verify the target tenant accepts the intended status set.

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

**Output**: `input` (GetExecutionInput) or `output` (GetExecutionOutput), declared as an object.
Structured values can cross command boundaries as JSON strings. For a captured string-valued
object, parse before selecting fields, for example
`$(.childOutput.output.output | toObject.message)`. Preserve a representative fixture's actual
nesting and representation; the declared object type alone does not prove a direct nested path.
Sensitive values may be omitted or masked on reads; use reusable Inputs for secret bindings.

`GetExecutionOutput` fails if the target has not produced output yet. Gate it behind an
`AwaitExecution` (or `GetExecutionStatus` + `when`) and read complete output on `FINISHED`.
Cancellation can leave partial output; `FAILED` may leave none. Inspect availability before
attempting a partial-output read instead of treating every returned state as success.

## Minimal snippets

Trigger-and-await, then read the child's output:

```json
[
{
    "execute": "utils-sapcp:TriggerExecution:1",
    "alias": "trigger",
    "input": {
        "command": "utils-sapcp:Void:1",
        "values": "{}"
    },
    "description": "Start the child once",
    "progressMessage": null, "initialDelay": null, "pause": null,
    "when": null, "validate": null, "autoRetry": null, "repeat": null,
    "errorMessages": []
},
{
    "execute": "utils-sapcp:AwaitExecution:1",
    "alias": "await",
    "input": {
        "execution": "$(.trigger.output.execution)",
        "statuses": "[\"FINISHED\",\"FAILED\",\"ABORTED\",\"SUSPENDED\"]",
        "deadline": "30"
    },
    "description": "Wait and require successful child completion",
    "progressMessage": null, "initialDelay": null, "pause": null, "when": null,
    "validate": {
        "semantic": "AND",
        "conditions": [{
            "semantic": "AND",
            "cases": [{
                "expression": "$(.await.output.status)",
                "operator": "EQUALS", "semantic": "OR", "values": ["FINISHED"]
            }]
        }]
    },
    "autoRetry": null, "repeat": null, "errorMessages": []
},
{
    "execute": "utils-sapcp:GetExecutionOutput:1",
    "alias": "childOutput",
    "input": {
        "execution": "$(.trigger.output.execution)"
    },
    "description": "Read output after successful completion",
    "progressMessage": null, "initialDelay": null, "pause": null,
    "when": null, "validate": null, "autoRetry": null, "repeat": null,
    "errorMessages": []
}
]
```

The `validate` on `await` prevents the output-read step from running after failure or
cancellation. Replace `Void` and its values with the resolved child contract. The authoring
skill indexes a complete command template and acceptance scenarios for this pattern.
