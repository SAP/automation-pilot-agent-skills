---
name: automation-pilot-command-authoring
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: >
  Authors and maintains SAP Automation Pilot commands and composite automations on the tenant. Use when a user wants to create a reusable command, build an orchestration flow (HTTP calls, scripts, polling, batching, conditionals), or change an existing command's contract, executor flow, expressions, or error handling. Also use for production-readiness reviews and local-only command definitions.
---

# SAP Automation Pilot Command Authoring

Before API work, read the [shared CLI operating rules](../automation-pilot-cli/SKILL.md)
for setup, authorization, secret handling, and response safety.

## Goal

Turn an operational intent into a well-formed Automation Pilot command and create or update it on the tenant. A command is a JSON definition with an input/output contract. SAP built-ins may be atomic, but a tenant command authored through this skill has a `configuration` that orchestrates one or more executors. Design that definition, write its expressions, create or update it on the tenant, and prove it with a run.

## Before you start

You need two things from the user:

- The **operational intent**: what the command does, for whom, and the success criteria.
- The **caller-facing contract**: the input keys the caller supplies and the output keys it exposes.

Resolve gaps in intent, scope, or success criteria before authoring. If the user ran the planning skill, author from its specification. Ask about an omitted choice only when it changes the contract or behavior; otherwise author from the documented defaults and state the assumption.

Select the requested mode first: **review** reads and reports findings with file paths and fixes;
**local authoring** writes or edits files and runs lint; **tenant authoring** additionally deploys
and verifies. Reviews do not deploy or trigger. Preserve the existing ID, version, contract,
executor controls, descriptions, tags, and unrelated fields when updating. Fetch the current full
definition before a tenant update; compare only the requested changes and exclude server-owned
fields from the write. Use settled targets and prior authorization.

Then follow the five-step workflow below. The reference material is indexed at the end of this document. Each step points to the one it needs as you reach it.

## Workflow

### Step 1: Design before JSON

Pin down, in this order:

1. **Contract:** the input keys the caller supplies (name, type, required, sensitive, allowed values) and the output keys the command exposes. Design the contract for the caller rather than the underlying APIs.
2. **Shape:** reuse an existing SAP built-in directly when one atomic operation fits; author a tenant command as a composite with a `configuration` that orchestrates one or more executors. Use multiple executors for data flow, conditions, polling, or batching.
3. **Flow:** the ordered executors in a composite, what each consumes and produces, and where data crosses between them. Choose the reference by user goal: [call an API](references/commands/http.md), [run custom code or a CLI](references/commands/scripting.md), [query or change HANA data](references/commands/sql.md), or [process a collection](references/commands/iteration.md). Read the matching example when one exists. Before tenant deployment, read composed executor and Input contracts live (see "Resolve everything live"). Local-only work records any unavailable contracts.
4. **Resilience:** transient failures that need `autoRetry`, asynchronous operations that need `repeat` polling, conditional steps (`when`), postconditions (`validate`), and `errorMessages` written for the caller. The [pattern cookbook](references/patterns.md) has the composition shapes for these.

Done when contract, shape, flow, and resilience are decided. Tenant authoring also requires
live executor/Input contracts; local-only work records unavailable contracts as assumptions.

### Step 2: Write the command

Read the [command model](references/command-model.md) for the caller-facing contract and top-level JSON, then the [composition controls](references/composition-controls.md) for `configuration`, executors, conditions, retries, polling, errors, and delays. The facts that bite hardest if missed:

- **Naming.** Command and Input names use PascalCase; input/output keys and executor aliases use camelCase. Names cannot contain spaces. There is no default catalog: use the catalog and target already selected by the user; propose one only when no target is settled, and resolve that choice before a tenant write. Reserve the `-sapcp` suffix for SAP built-ins.
- **Everything-is-a-string on the wire.** An input key's `defaultValue`, every execution-time input value, and every `values` entry inside a condition block is a JSON string regardless of declared type: `"5"`, `"true"`, `"[\"a\",\"b\"]"`. Condition `values` are `["200"]`, `["true"]`, `["1"]`. A `number`/`object`/`array` output also serializes back as a string.
- **Expressions.** Use the `automation-pilot-dynamic-expressions` skill for every non-trivial expression. Supply its intended field, expected result type, and a non-sensitive execution-shaped fixture containing the parent input and prior executor outputs.
- **Keep the executor shape explicit.** Include the applicable control fields (`when`, `validate`, `autoRetry`, `repeat`, `errorMessages`, `initialDelay`, `pause`, `progressMessage`, `description`) with `null` or `[]` when unset; the examples show the shape.

Done when the JSON expresses the designed contract, follows the naming and engine rules, and carries the explicit executor shape.

### Step 3: Descriptions and sensitivity

Read [descriptions and sensitivity](references/descriptions-and-sensitivity.md). Write the command `description` in terms of its caller-visible behavior. For its standard parameters, use the matching mandated sentence; for other keys, describe the caller-visible meaning and constraints.

Mark every secret-bearing input or output key `sensitive: true`. Use a `Sensitive*` executor variant when the step's output can contain sensitive data; a sensitive input alone does not require the sensitive variant when the ordinary executor keeps that input protected and its output is non-sensitive.

Done when the command and every key carry a caller-facing description, mandated strings are used verbatim, and every secret-bearing key is `sensitive: true`.

### Step 4: Validate, then deploy when requested

Run `autopi lint <Name>.command.json` before every deployment. Fix CRITICAL/HIGH findings;
resolve or explain lower-severity heuristic findings with evidence. For reviews, return prioritized
findings and proposals here. For local authoring, return the validated file and remaining live
contract assumptions here.

For tenant authoring, use the exact target covered by the user's request. Create with
`autopi commands create`, replace with `autopi commands update`, or use `autopi commands deploy`
for a 404-only upsert. Preview the redacted request when it helps review. Ask for final approval
only if deployment is outside the existing authorization, after the definition is ready. Delete
only an exact target explicitly authorized by the user, with dependency impact understood.

When a validation execution needs diagnostic evidence, use the `automation-pilot-tags` skill to
apply execution logging, then hand the command to the executions skill.

Deploy dependencies first: catalog, reusable Inputs, leaf commands, then parent commands.
Stop at a failed prerequisite or blocking read-back issue; dependent definitions and execution
requests wait until it is repaired. Prefer sequential deployment for dependent resources.

Done when the command exists at the confirmed ID and its prerequisites are verified.

### Step 5: Verify

Read the command back with `autopi commands get` and compare its contract, executor wiring, resilience blocks, descriptions, sensitivity flags, and `issues` array with the design. `autopi commands get` returns the validation `issues` for the command; `autopi commands list --include-issues` can include them, but use the full get for verification. When execution is requested and authorized, exercise the command with representative inputs: schema validation cannot catch expression, ordering, or integration failures, so only a run proves them.

Done when no blocking validation issues remain, informational warnings are explained, and an
authorized representative run produces the intended output. If execution was not requested,
report deployment/read-back validation and the specific runtime checks still needed.

## Resolve everything live

**Before using any composed executor or Input's name, keys, or version, read its contract from the tenant.** Catalog contents change, so an example or a remembered contract never confirms what the tenant exposes now; infer the version from the live read, never from a name.

- A **command executor** resolves with `autopi commands contract` (contract only) or `autopi commands get` (full definition plus validation `issues`).
- A reusable **Input** (anything in a key's `allowedValuesFromInputKeys` or a `configuration.values[].inputReference`, such as `metadata-sapcp:CfRegionData:1`) resolves with `autopi inputs get`. An Input ID 404s on `autopi commands get`.

Prefer a fitting existing command over rebuilding the same operation from HTTP or script primitives; the discovery skill finds those.

## References and examples

**Reference material.** Load the row matching the current decision; do not load every reference:

| User goal | Read |
|---|---|
| Define or change the caller-facing contract, naming, serialization, or catalog placement | [command model](references/command-model.md) |
| Assemble executors, reusable values, conditions, retries, polling, errors, or delays | [composition controls](references/composition-controls.md) |
| Describe keys or decide sensitivity | [descriptions and sensitivity](references/descriptions-and-sensitivity.md) |
| Poll, retry, branch, validate, listen for status, or chain steps | [pattern cookbook](references/patterns.md) |
| Call an API | [HTTP](references/commands/http.md) |
| Run custom code or a CLI | [scripting](references/commands/scripting.md) |
| Query or change HANA data | [SQL](references/commands/sql.md) |
| Process a collection sequentially or in parallel | [iteration](references/commands/iteration.md) |
| Diagnose deployment validation or runtime authoring symptoms | [troubleshooting](references/troubleshooting.md) |
| Apply tags, inheritance, retention, priority, or execution logging | Use the `automation-pilot-tags` skill |

For decisions that span rows, load every matching reference directly from this table. In particular,
an HTTP retry, poll, validation, or status branch needs the composition controls, pattern cookbook,
and HTTP references; a complete composite definition needs both the command model and composition
controls. When troubleshooting identifies an owning topic, return to this table and load that
reference directly. Reference files are terminal leaves; all cross-topic routing happens here.

**Examples.** Open the complete template matching the flow; each still needs tenant-ID substitution and live contract verification. Read the
[template acceptance scenarios](references/template-validation.md) before relying on retry,
polling, or collection behavior:

| Flow | Template |
|------|----------|
| Sequential in-execution batching | [`ProcessAppsBatch.command.json`](examples/ProcessAppsBatch.command.json) |
| Caller-supplied collection with count/member validation before iteration | [`ProcessValidatedItems.command.json`](examples/ProcessValidatedItems.command.json) |
| Parallel child executions | [`ProcessItemsInParallel.command.json`](examples/ProcessItemsInParallel.command.json) |
| Polling an asynchronous operation | [`WaitForOperation.command.json`](examples/WaitForOperation.command.json) |
| Resolving an identifier and chaining the result | [`ResolveAndGetResource.command.json`](examples/ResolveAndGetResource.command.json) |
| Retrying transient HTTP failures | [`GetResourceWithRetry.command.json`](examples/GetResourceWithRetry.command.json) |

## Handoffs

Sequence: **plan → discovery → Inputs when needed → authoring → executions / scheduling**.

- Ask **discovery** to find an existing command or reusable building blocks before you write one. Reuse a fitting command as a direct executor or embedded sub-command; write new behavior only for the remaining gap.
- Ask **Inputs** to create or rotate a reusable configuration or credential set; authoring only binds its verified ID.
- Use **dynamic expressions** to craft and verify every non-trivial expression.
- Send a finished command and any failed run to **executions**; send recurring runs to **scheduling**.

## Error handling

- **Schema or contract problem** (validation error on deploy, or blocking `issues`): read the `issues` array with `autopi commands get` and fix the named key or executor.
- **Expression failure at runtime:** reproduce it in the dynamic-expressions skill against an execution-shaped fixture, then `autopi commands update` the same command.
- **Integration failure at runtime:** read the failed run back through the executions skill, fix the definition here, and update the same command.
- Read [command-authoring troubleshooting](references/troubleshooting.md) for symptom → cause → owner,
  then load every owning reference it names directly from the reference table above.

## Definition of done

The command expresses the intended contract and behavior, reuses content that fits, follows the naming and engine rules, uses mandated descriptions, and marks secrets as sensitive. Completion follows the requested mode: review findings, a locally validated file, or an authorized
tenant deployment with read-back. Blocking `issues` are fixed; informational warnings are
reported with their meaning. Runtime behavior is claimed only for representative runs actually
performed within the user's authorization. Report what it does and how to run it, and point to executions for triggering and scheduling for recurring runs.
