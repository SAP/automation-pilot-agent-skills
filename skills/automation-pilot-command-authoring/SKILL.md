---
name: automation-pilot-command-authoring
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: >
  Authors and maintains SAP Automation Pilot commands and composite automations on the tenant. Use when a user wants to create a reusable command, build an orchestration flow (HTTP calls, scripts, polling, batching, conditionals), or change an existing command's contract, executor flow, expressions, or error handling.
---

# SAP Automation Pilot Command Authoring

## Goal

Turn an operational intent into a well-formed Automation Pilot command and create or update it on the tenant. A command is a JSON definition with an input/output contract. SAP built-ins may be atomic, but a tenant command authored through this skill has a `configuration` that orchestrates one or more executors. Design that definition, write its expressions, create or update it on the tenant, and prove it with a run.

## Before you start

You need two things from the user:

- The **operational intent**: what the command does, for whom, and the success criteria.
- The **caller-facing contract**: the input keys the caller supplies and the output keys it exposes.

Resolve gaps in intent, scope, or success criteria before authoring. If the user ran the planning skill, author from its specification. Ask about an omitted choice only when it changes the contract or behavior; otherwise author from the documented defaults and state the assumption.

Then follow the five-step workflow below. The reference material is indexed at the end of this document. Each step points to the one it needs as you reach it.

## Workflow

### Step 1: Design before JSON

Pin down, in this order:

1. **Contract:** the input keys the caller supplies (name, type, required, sensitive, allowed values) and the output keys the command exposes. Design the contract for the caller rather than the underlying APIs.
2. **Shape:** reuse an existing SAP built-in directly when one atomic operation fits; author a tenant command as a composite with a `configuration` that orchestrates one or more executors. Use multiple executors for data flow, conditions, polling, or batching.
3. **Flow:** the ordered executors in a composite, what each consumes and produces, and where data crosses between them. Load the executor reference for each step from the [reference table](#references-and-examples) below, and read the matching example when one exists. Trust those references for the built-ins they document; for any other executor or Input, read its contract live before you commit to it (see "Resolve everything live").
4. **Resilience:** transient failures that need `autoRetry`, asynchronous operations that need `repeat` polling, conditional steps (`when`), postconditions (`validate`), and `errorMessages` written for the caller. The [pattern cookbook](references/patterns.md) has the composition shapes for these.

For each prior-executor field used by an executor input, `when`, `validate`, or final output mapping, record the evidence for its path. A command reference or live child contract proves the top-level output key; a sanitized representative execution output proves nested paths and types. If representative output is unavailable, record the path as an assumption for Step 5.

Done when the contract, shape, flow, and resilience are each decided, every executor or Input *not* covered by a command reference has had its contract read live, and every prior-executor path is backed by the required evidence or recorded as a verification assumption.

### Step 2: Write the command

Read the [command model](references/command-model.md) for the caller-facing contract and top-level JSON, then the [composition controls](references/composition-controls.md) for `configuration`, executors, conditions, retries, polling, errors, and delays. The facts that bite hardest if missed:

- **Naming.** Command and Input names use PascalCase; input/output keys and executor aliases use camelCase. Names cannot contain spaces. Put generated commands in the tenant's default `welcome-<<<TENANT_ID>>>` catalog unless the user names another existing custom catalog; reserve the `-sapcp` suffix for SAP built-ins.
- **Wire strings.** An input key's `defaultValue`, every execution-time input value, and every `values` entry inside a condition block is a JSON string regardless of declared type: `"5"`, `"true"`, `"[\"a\",\"b\"]"`. Condition `values` are `["200"]`, `["true"]`, `["1"]`.
- **Expressions.** Use the `automation-pilot-dynamic-expressions` skill for every non-trivial expression. Supply its intended field, expected result type, and a non-sensitive execution-shaped fixture containing the parent input and prior executor outputs. Preserve the evidenced sibling relationships, nesting, declared types, and string-encoded structured values; field names do not prove nesting.
- **Keep the executor shape explicit.** Include the applicable control fields (`when`, `validate`, `autoRetry`, `repeat`, `errorMessages`, `initialDelay`, `pause`, `progressMessage`, `description`) with `null` or `[]` when unset; the examples show the shape.

Done when the JSON expresses the designed contract, follows the naming and engine rules, and carries the explicit executor shape.

### Step 3: Descriptions and sensitivity

Read [descriptions and sensitivity](references/descriptions-and-sensitivity.md). Write the command `description` in terms of its caller-visible behavior. For its standard parameters, use the matching mandated sentence; for other keys, describe the caller-visible meaning and constraints.

Mark every secret-bearing input or output key `sensitive: true`. Use a `Sensitive*` executor variant when the step's output can contain sensitive data; a sensitive input alone does not require the sensitive variant when the ordinary executor keeps that input protected and its output is non-sensitive.

Done when the command and every key carry a caller-facing description, mandated strings are used verbatim, and every secret-bearing key is `sensitive: true`.

### Step 4: Create on the tenant

Creation mutates the tenant. Confirm the target command ID with the user first. Create with `automation-pilot:create_command` or replace an existing version with `automation-pilot:update_command`. `automation-pilot:delete_command` is irreversible, so confirm the exact ID with the user and use it only to clean up a command you created, never one you did not author.

When a validation execution needs diagnostic evidence, use the `automation-pilot-tags` skill to
apply execution logging, then hand the command to the executions skill.

Done when the command exists at the confirmed ID.

### Step 5: Verify

Read the command back with `automation-pilot:get_command` and compare its contract, executor wiring, resilience blocks, descriptions, sensitivity flags, and `issues` array with the design. Then exercise each caller-visible condition branch with safe representative inputs, and a value from each input partition the command folds into one output. Trace an unsafe branch, or a partition with no safe example, against sanitized captured output when available.

Done when `issues` is empty, a representative run produces the intended output, every safe caller-visible branch and documented input partition has been exercised, and the report distinguishes cases verified by a live run, traced only against captured output, or left unverified for lack of safe evidence.

## Resolve everything live

**The built-ins documented in the [command references](#references-and-examples) (HTTP, scripting, SQL, iteration, execution orchestration, human-in-the-loop) are authoritative. Trust their IDs, versions, keys, and usage as written, with no live read.** For anything else, meaning any other catalog command used as an executor or any reusable Input, read its contract from the tenant before using its name, keys, or version. Catalog contents change, so an example or a remembered contract never confirms what the tenant exposes now; infer the version from the live read, never from a name.

- A **command executor** resolves with `automation-pilot:get_command_contract` (contract only) or `automation-pilot:get_command` (full definition plus validation `issues`).
- A reusable **Input** (anything in a key's `allowedValuesFromInputKeys` or a `configuration.values[].inputReference`, such as `metadata-sapcp:CfRegionData:1`) resolves with `automation-pilot:get_input`. An Input ID 404s on `get_command`.

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
| Start, await, or inspect a separate execution (async/complex flows) | [execution orchestration](references/commands/execution-orchestration.md) |
| Pause for a human decision or free-text input (HITL) | [human-in-the-loop](references/commands/human-in-the-loop.md) |
| Diagnose deployment validation or runtime authoring symptoms | [troubleshooting](references/troubleshooting.md) |
| Apply tags, inheritance, retention, priority, or execution logging | Use the `automation-pilot-tags` skill |

For decisions that span rows, load every matching reference directly from this table. In particular,
an HTTP retry, poll, validation, or status branch needs the composition controls, pattern cookbook,
and HTTP references; a complete composite definition needs both the command model and composition
controls. When troubleshooting identifies an owning topic, return to this table and load that
reference directly. Reference files are terminal leaves; all cross-topic routing happens here.

**Examples.** Open the complete template matching the flow; each still needs tenant-ID substitution and live contract verification:

| Flow | Template |
|------|----------|
| Sequential in-execution batching | [`ProcessAppsBatch.command.json`](assets/ProcessAppsBatch.command.json) |
| Parallel child executions | [`ProcessItemsInParallel.command.json`](assets/ProcessItemsInParallel.command.json) |
| Polling an asynchronous operation | [`WaitForOperation.command.json`](assets/WaitForOperation.command.json) |
| Resolving an identifier and chaining the result | [`ResolveAndGetResource.command.json`](assets/ResolveAndGetResource.command.json) |
| Retrying transient HTTP failures | [`GetResourceWithRetry.command.json`](assets/GetResourceWithRetry.command.json) |

## Handoffs

Sequence: **plan → discovery → Inputs when needed → authoring → executions / scheduling**.

- Ask **discovery** to find an existing command or reusable building blocks before you write one. Reuse a fitting command as a direct executor or embedded sub-command; write new behavior only for the remaining gap.
- Ask **Inputs** to create or rotate a reusable configuration or credential set; authoring only binds its verified ID.
- Use **dynamic expressions** to craft and verify every non-trivial expression.
- Send a finished command and any failed run to **executions**; send recurring runs to **scheduling**.

## Error handling

- **Schema or contract problem** (validation error on deploy, or non-empty `issues`): read the `issues` array with `automation-pilot:get_command` and fix the named key or executor.
- **Expression failure at runtime:** reproduce it in the dynamic-expressions skill against an execution-shaped fixture, then `automation-pilot:update_command` the same command.
- **Integration failure at runtime:** read the failed run back through the executions skill, fix the definition here, and update the same command.
- Read [command-authoring troubleshooting](references/troubleshooting.md) for symptom → cause → owner,
  then load every owning reference it names directly from the reference table above.

## Definition of done

The command expresses the intended contract and behavior, reuses content that fits, follows the naming and engine rules, uses mandated descriptions, and marks secrets as sensitive. It is created on the tenant, read back with an empty `issues` array, and tested with representative inputs. Report what it does and how to run it, and point to executions for triggering and scheduling for recurring runs.
