---
name: automation-pilot-dynamic-expressions
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: >
  Crafts, explains, and debugs SAP Automation Pilot dynamic expressions against representative input data. Use when a user wants to write a $(...) expression or responseBodyTransformer pipeline, diagnose an expression that returns an unexpected or empty result, or build a non-trivial expression while authoring or repairing a command. Do not use for jq unrelated to Automation Pilot.
---

# SAP Automation Pilot Dynamic Expressions

## Goal

Produce an Automation Pilot dynamic expression, or a `responseBodyTransformer` pipeline, that is correct for its placement and reasoned against representative, non-sensitive data.

Automation Pilot evaluates expressions with a custom jq implementation that adds product-specific functions, restrictions, and output behavior and deviates from stock jq in places. Local jq is not proof that an expression behaves the same in Automation Pilot.

## Required inputs

- Where the expression is placed (command value, output value, HTTP `responseBodyTransformer`, or another field).
- The value and type it must produce.
- The data available at that point: parent input, metadata, and earlier executor aliases and outputs, with their declared output types and any string-encoded structured values.

An expression cannot read a later executor, and a conditionally skipped executor may have no output. When Command Authoring delegates expression work, it owns the accuracy of the execution-shaped fixture and the placement requirements.

Ask about omitted choices that change the value or its type; otherwise reason from the documented behavior.

## Reference

An expression is jq-compatible syntax wrapped in `$(` and `)`, evaluated against an execution-shaped root. Reach data by its path:

- `.execution.input.<key>`, `.execution.metadata.<key>`, `.execution.error.originalMessage` — command input, metadata, and the original failure message.
- `.<alias>.output.<key>`, `.<alias>.executed` / `.retriesCount` / `.repetitionsCount` — an earlier executor's output and run state (later executors are not readable).
- `.` is the current pipeline scope and is rebound by `|`, `map`, `filter`, `select`; use `$` (for example `$.execution.input.name`) to reach the root from inside those.

Never return a whole container (`.execution`, a bare `.<alias>`, `.<alias>.output`): the engine resolves it to a silent `null`. Reach the specific field.

Read the [expression reference](references/expression-reference.md) for the full fixture shape, expression forms, engine behavior, scope restrictions, and limits before writing a non-trivial expression or diagnosing a failure.

Load the matching element reference when the expression needs functions from that group, and adapt its documented spellings and gotchas:

- [conversions and serialization](references/elements/conversions.md) — `toObject`, `toArray`, `toNumber`, `fromXml`, `toMinifiedJsonString`, and the encodings and hashes.
- [strings](references/elements/strings.md) — case, trimming, slicing, interpolation, `split`/`join`, and regular expressions.
- [collections](references/elements/collections.md) — `map`, `filter`, `select`, `sort`, `reduce`, grouping, and the `minBy`/`maxBy` analysis functions.
- [objects and safety](references/elements/objects.md) — key access, `getCaseInsensitive`, `has`/`in`, `//`, and the `null`-handling forms.
- [time and identifiers](references/elements/time-and-identifiers.md) — `now`, `nowMillis`, `toDate`, `fromDate`, `compareVersions`, and GUIDs.
- [math and control flow](references/elements/math-and-control-flow.md) — arithmetic, `if`/`then`/`else`, variables, the comma, and `range`.

Read [engine gotchas](references/engine-gotchas.md) before relying on any non-obvious behavior, and whenever an expression produces an unexpected or empty result.

## Workflow

### Step 1: Establish the expression context

Confirm the placement, the required value and type, and the available data from Required inputs. Build the smallest representative JSON fixture that preserves the relevant paths and types, using the shape in the [expression reference](references/expression-reference.md). Replace credentials, tokens, personal data, and production payloads with fake values.

Done when the fixture includes the placement, expected type, all referenced paths, each referenced output's declared type, and any string-encoded structured values that affect the expression.

### Step 2: Craft the smallest expression

Write the smallest expression that produces the intended value. Use only documented Automation Pilot elements and their exact spellings; load the matching element reference above. Match each conversion to the value's current type or the receiving field's documented contract. When a value crosses an output key, apply the [output-key type boundary](references/expression-reference.md#output-key-type-boundary).

Match the wrapper syntax to the placement: a `$()` substitution, a multi-substitution command value, or a bare `responseBodyTransformer` ([expression forms](references/expression-reference.md#expression-forms)). For a value with several substitutions, reason about each separately, then about the assembled value.

Done when the expression uses documented elements, preserves the expected result type, uses wrapper syntax that matches its placement, and every explicit conversion is required by the value's current type or the receiving field's documented contract.

### Step 3: Trace it by hand

Trace the expression against the fixture and predict the serialized output, then compare it semantically with the expected value. A result that is no value or `null` serializes as empty output, so do not read a silent empty as proof of a working `null`-handling path.

Reason through every case the [engine behavior](references/expression-reference.md#engine-behavior-that-affects-correctness) and [scope restrictions](references/expression-reference.md#scope-restrictions) raise that can change this expression's behavior. If the trace exposes a problem, change the expression according to the evidence; do not substitute stock jq behavior for the Automation Pilot behavior in the reference. Reason about result type from the elements used, not from the serialized shape.

Done when the representative case and every relevant edge case has a predicted output and any discovered mismatch has been corrected.

### Step 4: Hand off

Return the exact expression, its intended placement, the fixture shape, the expected result type, and the predicted representative and edge-case outcomes. Do not deploy or execute a command unless the user also requested command work.

Command Authoring owns inserting the expression, preserving its reasoned context, validating the complete definition, and deploying. Execution troubleshooting owns collecting failed-run evidence; expression failures come here for focused reproduction and correction before Command Authoring updates the definition.

Done when the hand-off includes enough context for the caller to insert the expression without changing its meaning.

## Definition of done

The expression uses documented Automation Pilot syntax, references data available at its placement, and produces the intended value and representation for a representative fixture when traced by hand, including the relevant edge cases.

## Error handling

Report the symptom in terms of the engine behavior that produces it ([engine gotchas](references/engine-gotchas.md)): a silent `null` from missing data or an impossible conversion, accidental text concatenation from a multiple-result stream, a limit error, or a scope restriction. Diagnose against the fixture rather than inventing runtime state. When the failure is a real execution, request the failed-run data from execution troubleshooting and reproduce it against the fixture here.
