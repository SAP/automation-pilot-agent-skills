# Composition controls

Read this when assembling a tenant-authored command's configuration, loading reusable Inputs, or
adding conditions, retries, polling, custom errors, or delays. The skill workflow loads the
caller-facing command model, worked patterns, and executor-specific references directly when the
current design needs them.

Resolve each executor and Input contract live; snippets define shapes, not tenant availability.

## Contents

- [Configuration](#configuration)
- [Reusable values](#reusable-values)
- [Executors](#executors)
- [Dynamic expressions](#dynamic-expressions)
- [Condition blocks](#condition-blocks)
- [Supported delays](#supported-delays)
- [Retries](#retries)
- [Polling](#polling)
- [Custom errors](#custom-errors)
- [Initial delay](#initial-delay)

## Configuration

```json
"configuration": {
  "values": [],
  "output": {},
  "executors": [],
  "listeners": []
}
```

- `values` loads reusable Input data under local aliases.
- `output` maps the completed command's declared output keys.
- `executors` contains ordered child-command steps.
- `listeners` contains whole-execution status handlers; use `[]` when none are needed.

## Reusable values

```json
"values": [
  {
    "alias": "regionData",
    "valueFrom": {
      "inputReference": "metadata-sapcp:CfRegionData:1",
      "inputKey": "$(.execution.input.region)"
    }
  },
  {
    "alias": "credentials",
    "valueFrom": {
      "inputReference": "welcome-<<<TENANT_ID>>>:MyCredentials:1",
      "inputKey": null
    }
  }
]
```

`inputKey` has three forms:

- A dynamic expression selects a key at runtime.
- `null` loads the complete Input as an object, reached as `$(.alias.field)`.
- A plain string loads one named key.

Resolve every `inputReference` with `autopi inputs get` and confirm the key's type and
sensitivity before using it.

## Executors

```json
{
  "execute": "catalog:Command:version",
  "alias": "performOperation",
  "input": {},
  "description": null,
  "progressMessage": null,
  "initialDelay": null,
  "pause": null,
  "when": null,
  "validate": null,
  "autoRetry": null,
  "repeat": null,
  "errorMessages": []
}
```

Keep the complete executor control-field shape explicit. A later step reaches an earlier output as
`$(.earlierAlias.output.key)`. Read the executor contract live before writing its `input` map.

## Dynamic expressions

The Dynamic Expressions skill owns expression syntax, scope, limits, and debugging. Give it the
target field, expected result type, and an execution-shaped fixture containing the parent input and
all prior outputs available at that point. Insert the returned expression unchanged, then prove it
through a complete command execution.

## Condition blocks

`when`, `validate`, `autoRetry.when`, `repeat.until`, and `errorMessages[].when` share this shape:

```json
{
  "semantic": "OR",
  "conditions": [
    {
      "semantic": "OR",
      "cases": [
        {
          "expression": "$(.step.output.status)",
          "operator": "EQUALS",
          "semantic": "OR",
          "values": ["200", "201"]
        }
      ]
    }
  ]
}
```

Every `values` entry is a string. Resolve supported operators from a live example or current product
documentation rather than extending the set by analogy. The inner case `semantic` controls how
multiple values combine: for example, `CONTAINS` plus `OR` means any value may match, while a
negative membership check normally needs `AND` to exclude every listed value.

## Supported delays

The pinned API schema accepts `5s`, `10s`, `15s`, `30s`, `1m`, `2m`, and `3m` for
`autoRetry.delay`, `repeat.delay`, and `initialDelay.interval`. Use one of these values;
longer retry/poll budgets come from the count and delay together. Distinguish these
discrete executor delays from other duration fields such as a child's await deadline.

## Retries

Use `autoRetry` only for a transient, repeat-safe operation:

```json
"autoRetry": {
  "maxCount": 3,
  "delay": "5s",
  "logic": "INCREMENTAL",
  "applyOnValidation": false,
  "when": { "semantic": "OR", "conditions": [] }
}
```

`logic` accepts `FIXED` or `INCREMENTAL` on the tenant tested on 2026-10-26. Resolve other
values from service evidence before use; `CONSTANT` was rejected.

The executor must expose the result that the retry condition inspects. For HTTP, a non-success
status or timeout can fail before `autoRetry.when` runs; apply the HTTP observability rules loaded
by the skill workflow before branching on those outcomes. If accepted transient status codes
are converted into failures by `validate`, set `applyOnValidation: true` so those failures
participate in retry. Keep `when` restricted to transient codes; permanent 401/403 failures
should fail immediately. Verify the engine ordering using the template acceptance scenarios.

## Polling

Use `repeat` when the same executor must run until an observed condition becomes true:

```json
"repeat": {
  "maxCount": 60,
  "delay": "15s",
  "failOnMaxCount": true,
  "until": { "semantic": "OR", "conditions": [] }
}
```

Define both the success condition and the terminal failure path. Use the pattern cookbook for
fixed-count and wall-clock-budget variants.

## Custom errors

`validate` determines whether the executor result is acceptable. `errorMessages` selects
a caller-facing explanation for an existing failure; it does not establish a postcondition.
Use an explicit matching condition for each message:

```json
"errorMessages": [
  {
    "message": "The upstream service rejected the request. Check the supplied credentials.",
    "when": {
      "semantic": "OR",
      "conditions": [{
        "semantic": "OR",
        "cases": [{
          "expression": "$(.step.output.status)",
          "operator": "EQUALS", "semantic": "OR", "values": ["401", "403"]
        }]
      }]
    }
  }
]
```

Messages should help the caller act without exposing credentials or secret-bearing response data.
For HTTP calls that may echo secrets, use the executor's body-omission control and an appropriate
sensitive-output variant.

## Initial delay

`{"interval":"30s","when":null}` delays an executor unconditionally. Add a condition block to
delay only while a prerequisite state holds.

