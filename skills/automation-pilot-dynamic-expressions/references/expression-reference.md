# Dynamic Expression Reference

Core forms, scope, engine behavior, and limits for SAP Automation Pilot dynamic expressions.

## Expression forms

- A dynamic expression is jq-compatible syntax wrapped in `$(` and `)`, for example `$(.execution.input.enabled)`. The text inside must be a valid jq filter.
- A single `$()` substitution holds one expression. Surrounding literal text belongs outside the `$()` pair.
- A command value may contain several substitutions, for example `https://$(.execution.input.host)/api/v1/resource/$(.execution.input.resourceId | toPercentEncoded)`, and may embed jq string interpolation inside an expression such as `$("https://host?q=\(.value | toUrlEncoded)")`. Reason about each embedded expression separately, then about the assembled value.
- An HTTP executor's `responseBodyTransformer` receives the same expression syntax without the surrounding `$()` wrapper. It is stored as a normal string, so it can also contain `$()` substitutions resolved from execution data before the transformer runs against the response body, for example `toArray | filter(. == $(.execution.input.expectedNumber))`.

## Scope and execution-shaped data

Treat the execution-shaped fixture as the expression root. Reproduce the relevant command paths in it:

```json
{
  "execution": {
    "input": { "resourceName": "demo" },
    "metadata": { "executionId": "example-id" },
    "error": { "originalMessage": "sample upstream error" }
  },
  "getResource": {
    "executed": true,
    "retriesCount": 0,
    "repetitionsCount": 0,
    "output": {
      "status": "200",
      "body": "{\"items\":[{\"name\":\"demo\"}]}"
    }
  }
}
```

- `.` is the current pipeline scope. `$` reaches the global root. Because `.` is rebound by `|`, `map`, `filter`, `select`, and similar, use `$` (for example `$.execution.input.name`) to reach execution data from inside those. Dropping the `$` — `filter(.name == .execution.input.itemName)` — looks for that path on each element and silently finds nothing; a common cause of a predicate matching nothing.
- Command data paths:
  - `.execution.input.<key>` — command input values.
  - `.execution.metadata.<key>` — execution metadata (`executionId`, `commandId`, `startTime`, `tenantId`, tags, and others).
  - `.execution.error.originalMessage` — the original error of a failed execution, before any custom error message overwrites it.
  - `.<alias>.output.<key>` — an earlier executor's output value.
  - `.<httpAlias>.output.status` — HTTP status code uses the documented output wire representation; apply `toNumber` before numeric membership or comparisons.
  - `.<alias>.executed`, `.<alias>.retriesCount`, `.<alias>.repetitionsCount` — whether an executor ran, was retried, or was repeated.
- Only earlier executor outputs are available. A conditionally skipped executor may not expose the expected output — its path resolves to `null`. Default it with `.alias.output.key // fallback`; this already covers the skipped case, so do not wrap it in `if .alias.executed then (.alias.output.key // fallback) else fallback end`, which restates what `//` does.
- Object and array values can cross executor or command output boundaries as JSON-encoded strings. Apply `toObject` (object strings) or `toArray` (array strings) before selecting inside them. Selecting directly — `.childStep.output.obj.someKey` on an unparsed string — returns `null`, and picking the wrong converter (`toObject` on an array string) yields the same silent `null`.

## Scope restrictions

The engine does not let an expression return a whole container through its global path; select the specific field instead. Enforcement comes in two forms:

- Bare `$(.)` and `$($)` are **hard-rejected at authoring** with "Using '.'/'$' is not allowed at that location."
- The deeper whole-container paths (`.execution`, `.execution.input`, `.execution.metadata`, a whole executor `.<alias>`, or a whole executor output `.<alias>.output`) are accepted at authoring but **silently resolve to `null` at runtime**. The result is empty, with no error, so an expression that leans on one of these produces a silent empty rather than a visible failure.

Either way, reach the specific field you need (`.execution.input.<key>`, `.<alias>.output.<key>`) rather than a container.

## Engine behavior that affects correctness

- Accessing non-existing data returns `null`. An impossible transformation (for example `1 * [1]` or `1 | toObject`) also returns `null`, not a visible error. The engine serializes no result or `null` as empty output, so a silent empty is not proof of a working `null`-handling path.
- Multiple results are concatenated into one text value: `$(1, 2, 3)` produces `123`, and `$(.array[])` concatenates its elements. Wrap the stream in `[ ... ]` when the intended result is an array.
- A scalar result serializes as text; an array or object serializes as JSON text. There is no separate result-type indicator, so reason about the type from the elements used, not from the serialized shape.

## Limits

- A single command value can hold at most **50** dynamic expressions.
- Each expression has a maximum element count. Exceeding it fails with `Expression contains too many elements: $(...)`. Split a large expression into several short ones (for example separate `utils-sapcp:Void:1` steps, or per-field interpolations into a literal JSON envelope).
- Chained parent/child transformations have a data-source limit. Exceeding it fails with `Expression includes too much data from other sources`.
