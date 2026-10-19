---
name: automation-pilot-executions
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: >
  Runs SAP Automation Pilot commands and operates their executions. Use when a user wants to trigger or monitor a run, inspect its input, output, or logs, diagnose a failed or stalled execution, or analyze execution history. Do not use to repair a command definition or a schedule that failed to trigger.
---

# SAP Automation Pilot Executions

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
