# Iteration: ForEach & ParallelForEach

Run a sub-command once per item of a list. Use when a step must repeat over a resolved set
(apps in a space, rows from a query, IDs from an API) rather than once.

## Contents

- [When to use](#when-to-use)
- [Executor IDs and variants](#executor-ids--variants)
- [Shared inputs](#shared-inputs)
- [Parallel-only inputs](#parallelforeach1-only-inputs)
- [Outputs](#outputs--how-to-read-them)
- [Minimal snippets](#minimal-snippets)
- [Gotchas](#gotchas)

## When to use

- When order matters, later items depend on earlier side effects, or you want one execution, use
  `utils-sapcp:ForEach:2`. It runs the sub-command sequentially, inside the current execution.
- When items are independent and you need throughput and per-child execution IDs, use
  `utils-sapcp:ParallelForEach:1`. It triggers separate child executions in batches and polls them.

## Executor IDs & variants

| Executor | Model |
|----------|-------|
| `utils-sapcp:ForEach:2` | Sequential, in-execution. Use `:2`, not the deprecated `:1`. |
| `utils-sapcp:ParallelForEach:1` | Parallel child executions, batched + polled |

## Shared inputs

Required: `command` and `inputs`.

| Field | Notes |
|-------|-------|
| `command` | ID of the sub-command. Must be a literal, not a dynamic expression. |
| `inputs` | Expression resolving to an array, one entry per iteration. |
| `inputMapping` | Stringified JSON object; every value must be a quoted string (`"key":"$(expr)"`). An unquoted value is rejected at deploy (`"must be a valid JSON object with static keys"`). A quoted `$(expr)` resolving to an object or array is passed to the sub-command typed, so no `toEscapedJson` is needed. Default `{}`. |

Reach the current element with the ForEach step's own alias:
`$(.<thisAlias>.current.item)` (object items: `$(.<thisAlias>.current.item.name)`). Do not
use the list step's alias.

## `ParallelForEach:1`-only inputs

| Field | Default | Notes |
|-------|---------|-------|
| `batchSize` | `5` | Child executions per batch (**1–25**). |
| `mode` | `trigger, poll, and collect` | `trigger and poll` waits for each batch to reach an expected status; the default also collects outputs. |
| `statuses` | `["FINISHED"]` | Child end-states counted as success for polling. |
| `deadline` | `5` | Max minutes to poll before failing (**1–10080**). |
| `tags` | `{}` | Tags stamped on each child execution for later queries. |

## Outputs & how to read them

- `ForEach:2` outputs `outputs`: an array of each run's output, in input order. Collect it with `$(.<alias>.output.outputs)`.
- `ParallelForEach:1` outputs `outputs` (per-item results, in order; `null` when `mode` does not collect)
  and `executionsMetadata`: an array of `{execution, status, batchIndex}` giving each child's ID.

## Minimal snippets

Sequential (`ForEach:2`):

```json
{
  "execute": "utils-sapcp:ForEach:2",
  "alias": "processItems",
  "input": {
    "command": "utils-sapcp:Void:1",
    "inputs": "$(.listItems.output.ids)",
    "inputMapping": "{\"message\":\"$(.processItems.current.item)\"}"
  },
  "when": null, "validate": null, "autoRetry": null, "repeat": null,
  "description": null, "progressMessage": null, "initialDelay": null, "pause": null,
  "errorMessages": []
}
```

Parallel (`ParallelForEach:1`):

```json
{
  "execute": "utils-sapcp:ParallelForEach:1",
  "alias": "processItems",
  "input": {
    "command": "utils-sapcp:Void:1",
    "inputs": "$(.listItems.output.ids)",
    "inputMapping": "{\"message\":\"$(.processItems.current.item)\"}",
    "batchSize": "5",
    "mode": "trigger, poll, and collect",
    "statuses": "[\"FINISHED\"]",
    "deadline": "30"
  },
  "when": null, "validate": null, "autoRetry": null, "repeat": null,
  "description": null, "progressMessage": null, "initialDelay": null, "pause": null,
  "errorMessages": []
}
```

## Gotchas

- `command` cannot be a dynamic expression. It is a fixed command ID.
- `inputMapping` is a stringified JSON object, and `inputs` must resolve to an array.
- Quote every mapping value, including object and array ones. `"payload":"$(.<thisAlias>.current.item.payload)"` delivers the object to the sub-command typed. Writing it unquoted (`"payload":$(...)`) breaks static-JSON validity and is rejected at deploy. Do not reach for `toEscapedJson` here; the quotes alone are enough.
- Reach the element with the ForEach's own alias (`$(.<thisAlias>.current.item)`), never the
  list step's alias. This is a frequent copy-paste error.
- Guard the step with `when` on `$(.listStep.output.X | length) GREATER_THAN "0"` so an empty
  list is skipped rather than erroring. If the parent exposes iterator results, map skipped output
  to an empty array with `$(.<alias>.output.outputs // [])`; use the same fallback for
  `executionsMetadata` from `ParallelForEach`.
- The sub-command named in `command` must already exist and be resolved live before use.
