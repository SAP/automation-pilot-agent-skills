---
name: automation-pilot-executions
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: >
  Runs SAP Automation Pilot commands and operates their executions. Use to trigger/monitor runs or webhooks, inspect execution data, diagnose failures, analyze history, request single/bulk actions or operator choices, and delete execution history. Do not use to repair command definitions or schedules that failed to trigger.
---

# SAP Automation Pilot Executions

Before API work, read the [shared CLI operating rules](../automation-pilot-cli/SKILL.md)
for setup, authorization, secret handling, and response safety.

## Goal

Run the intended command once, follow the resulting execution accurately, and turn execution evidence into a clear outcome or diagnosis.

## Required inputs

For a new run:
- The command to run (resolvable to command, catalog, and version).
- Values or reusable Automation Pilot Inputs for every required command input.

For an existing run:
- An execution ID, or filters that identify the intended executions.

Optional:
- Tags and execution features.
- A monitoring deadline or history time window.

Ask about omitted choices that change the target, supplied input, or scope. Otherwise use the command contract and supported product defaults.

## Workflow

### Triggering a command

#### Step 1: Resolve the command and inputs

Resolve an unambiguous command ID and read its required and optional input keys, types, constraints, and sensitivity. Use `autopi commands contract` when an exact ID is known; use `autopi catalogs list` and `autopi commands list` to find or disambiguate it. Find reusable Inputs with `autopi inputs list` and confirm their non-secret key metadata with `autopi inputs get`. If no command fits, hand the unresolved requirement to command authoring. If a reusable configuration or credential Input is required but none fits, hand its contract to Inputs for creation before triggering.

Done when one exact command ID is selected, its live contract is read, and every required input is accounted for by a direct value or reusable Input.

#### Step 2: Assemble the execution request

Read [triggering](references/triggering.md), bind every required input, and apply its encoding and duplicate-run safeguards. Keep sensitive values out of direct values and user-facing output.

Before adding or overriding tags, use the `automation-pilot-tags` skill. Add diagnostic logging
before the run when troubleshooting evidence may be needed.

Done when every bound value matches the live type and wire format, sensitive values avoid direct bindings, tags are valid, and the duplicate-run decision is explicit.

#### Step 3: Trigger once

Call `autopi executions trigger` once and capture the returned execution ID. A trigger runs the command for real; the user's request must cover the resolved target and scope.

Done when the execution ID is captured, or an ambiguous trigger response has been investigated without issuing a second trigger.

#### Step 4: Monitor when requested

Continue with the monitoring workflow when the user asked for the result or for the run to be followed. Otherwise report the execution ID.

Done when the requested terminal or attention state is reached, the monitoring deadline is reached, or the user receives the execution ID for an intentionally asynchronous handoff.

### Monitoring or inspecting an execution

Read [monitoring and inspection](references/monitoring.md). When inspecting resolved input, also
read [triggering](references/triggering.md) directly before interpreting its serialized values. Use
`autopi executions get` as the source of truth, and follow the monitoring reference's stop,
attention, and stuck-run rules. Read resolved input or output only when it bears on the request; use
`autopi executions logs` for one relevant executor path. Never reproduce sensitive values.

Done when the reported state is current and the requested input, output, progress, or attention evidence is included without unrelated or sensitive data.

### Troubleshooting an execution

Read [monitoring and inspection](references/monitoring.md) for state and response semantics, then read [execution troubleshooting](references/troubleshooting.md) for the evidence sequence and failure catalog. Use `autopi commands get` only when the command implementation bears on the failure. Account for the top-level error, executor path and states, resolved input, available log, and command definition before concluding the cause.

Apply the reference's cause classification. Correct and re-trigger run inputs only when the user authorizes the new run and repeating the command is safe. Hand command-definition changes to command authoring, focused expression defects to Dynamic Expressions, and missed schedule triggers to scheduling.

Done when the top-level state, failed executor path, relevant input/output/log evidence, cause class, and next owner or action are each accounted for.

### Controlling or deleting an execution

Read `autopi executions get <id>` and `autopi executions actions <id>` first. Apply only the
action the user requested to that exact ID: `autopi executions action <id> --type
PAUSE|RESUME|RETRY|ABORT|RESET|CHOOSE|SKIP|FAIL|COMMENT`, with a required non-secret `--reason`
and `--value` for RESET/CHOOSE/SKIP. Pass the fetched execution ETag with --if-match
when protecting the action against concurrent changes. State eligibility is enforced by the service; do
not guess a reset path or user choice. RETRY and RESET may repeat side effects, so inspect
completed steps and idempotency before performing the authorized repeat. Monitoring alone
does not authorize a repeat, abort, or Confirm choice. On a timeout, inspect action history
and current state before resending. Read state and actions back to verify the result.

Delete with `autopi executions delete <id>` only when deletion of that history is requested
and the exact target is settled. Verify a 404; other failures do not establish deletion.

An HTTP 202 action response means acceptance, not completion; the CLI exits 0. Verify the
requested state effect with `autopi executions get <id>` and action history.

Done when the requested action is recorded and its state effect is verified, or deletion
is verified, without inventing a human choice.

For an authorized batch, resolve the exact IDs and request `executions bulk-action`
with PAUSE/RESUME/ABORT/RETRY and a reason. A batch holds at most 50 IDs; retain
ordered per-item outcomes. Exit 7 means individual failures despite HTTP 200.
Inspect each failed target before requesting recovery; successes are already processed.
For webhook execution, read the [webhook reference](../automation-pilot-cli/references/webhooks.md),
trigger the settled webhook once, and follow the returned execution ID.

### Browsing and analyzing history

Read [execution history](references/history.md). When filtering or interpreting executions by tag,
also use the `automation-pilot-tags` skill. Translate the request into the narrowest useful filters,
and call `autopi executions list`. Report a short verdict and the executions that support
it. Fetch individual execution details only when the requested explanation requires fields absent
from the list response.

Done when the filters and time range are explicit, the evidence set supports the verdict, and any excluded or unavailable data that could change it is disclosed.

## Error handling

Report tool errors without inventing execution state. A failed trigger request does not prove that Automation Pilot created no execution; follow the ambiguity procedure in the triggering reference before considering another run.

## Definition of done

The requested run or history set is unambiguous, sensitive values stayed hidden, and the result is grounded in execution state. A new run has a captured execution ID. A monitored run ends with its outcome or latest state. A diagnosis names the evidence, cause class, and next action.
