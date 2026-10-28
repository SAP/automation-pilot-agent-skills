---
name: automation-pilot-scheduling
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: >
  Schedules SAP Automation Pilot commands for one-time or recurring execution and manages existing schedules. Use when a user wants to create, inspect, explain, update, enable, disable, or delete a scheduled execution, or diagnose why a schedule did not trigger. Do not use to troubleshoot a command run that the schedule successfully created.
---

# SAP Automation Pilot Scheduling

Before API work, read the [shared CLI operating rules](../automation-pilot-cli/SKILL.md)
for setup, authorization, secret handling, and response safety.

## Goal

Bind a command and its inputs to trigger times, and keep that binding correct over its lifetime.

## Required inputs

- The command to run (resolvable to command, catalog, and version).
- When it runs, and whether once or recurring.

Optional:
- Time zone.
- Command input bindings (direct schedule values or reusable Automation Pilot Inputs).
- Tags and execution features.

Ask about omitted choices that change the command or its timing; otherwise use supported product defaults.

## Workflow

### Creating a schedule

#### Step 1: Resolve the command and its inputs

Resolve the command, catalog, and version with `autopi catalogs list`, `autopi commands list`, and `autopi commands contract`. Read the command's required and optional inputs, types, and sensitivity. If no command fits, hand the unresolved requirement to command authoring.

Done when one exact command ID and version are selected, its live input contract is read, and every required key has a planned binding.

#### Step 2: Bind execution values

Bind reusable Input IDs through `inputReferences` and non-sensitive schedule-specific values through `inputValues`. Find reusable Inputs with `autopi inputs list` and confirm their metadata with `autopi inputs get`. Bind sensitive values through a sensitive Input and keep stored values hidden. If no compatible Input exists, hand the required key contract to Inputs for creation before scheduling.

Done when every required key has exactly one compatible binding, each reusable Input is confirmed live, and no sensitive value is inlined.

#### Step 3: Assemble the schedule

Translate the request into one schedule type under `schedule.<type>` and apply these rules:

- Set exactly one schedule type: `once`, `hourly`, `daily`, `weekly`, `monthly`, or `yearly`.
- Include the required `commandId` and `schedule` fields.
- Generate a concise, non-empty top-level description for every new schedule from the resolved command's purpose. You may reference schedule mechanics and input values at a high level, but omit their specific details and exact values. Preserve explicit wording only when the user supplies it.
- Use an IANA `timeZone`; it defaults to `Etc/UTC`. Set it for local-time schedules and tell the user when their request omitted it.
- Use non-empty arrays with unique values for recurring `minutes`, `hours`, `days`, and `months`; only `once` uses singular integers.
- Use multiples of five from 0 through 55 for minutes. This also applies to `once.minute`.
- Treat values in different fields as a cartesian product, not pairwise times.
- Express intervals by listing exact values. Automation Pilot has no interval primitive or cron string, so some cadences cannot be represented.
- Treat the configured time as the start of a trigger window. Allow for delayed starts and enough time before the next occurrence; overlapping windows can drop an occurrence.

Load exactly one matching type reference and adapt its payload:

- Read [`once`](references/schedule-types/once.md) for one execution at the next matching month, day, and time.
- Read [`hourly`](references/schedule-types/hourly.md) for every-hour or regular sub-hourly execution.
- Read [`daily`](references/schedule-types/daily.md) for execution every day at listed times.
- Read [`weekly`](references/schedule-types/weekly.md) for execution on listed weekdays.
- Read [`monthly`](references/schedule-types/monthly.md) for execution on listed days of every month.
- Read [`yearly`](references/schedule-types/yearly.md) for listed months and days, including multi-month cadences.

Use the `automation-pilot-tags` skill to attach tags or execution features.

Apply the selected type's field rules and edge cases. Show at least the next three predicted occurrences with local date/time and UTC offset,
including the next relevant DST or month-end edge. Confirm `timeZone` with `autopi
scheduled-executions time-zones`. Fields form a cartesian product: paired times such as
09:00 and 17:30 need separate schedules to avoid also scheduling 09:30 and 17:00.
For a one-time schedule, show the full next calendar date before creating it. Label these
as calendar predictions; repository docs do not prove how the service skips or duplicates
DST times. Use UTC or an unambiguous local time when the request requires a guarantee.

Done when exactly one schedule type is present and its recurrence, time zone, description, inputs, enabled state, tags, and relevant date/DST edge cases match the request.

#### Step 4: Write and verify

Run `autopi lint --kind scheduled-execution --mode api FILE` before writing; it checks
wire types, recurrence fields, minute granularity, and deadline bounds. The stale Swagger
entityVersion requirement is excluded using service-source evidence; use ETags for updates.
Call `autopi scheduled-executions create` once with the assembled body. Then read it back with `autopi scheduled-executions get` and compare its command, description, recurrence, time, time zone, input bindings, enabled state, and tags with the request.

Done when the created schedule ID is captured and the complete read-back matches the intended command, timing, bindings, state, and tags.

### Changing a schedule

Resolve an indirect reference with list when needed. Read the current schedule and its
ETag together with `get --response-envelope`. Build the replacement from that response's
body, change only the requested writable properties, validate, and
send the complete writable object with update --if-match ETAG. Preserve untouched fields;
omit server-managed id/state. On 412, re-read and reconcile. Schedule writes need Execute.
Read the result and verify it.

Done when the requested fields changed, every untouched field is preserved, and the read-back confirms the complete replacement object.

### Enabling, disabling, deleting

Use `scheduled-executions disable` or `enable` to change enabled state through a
read-modify-conditional PUT preserving other writable fields. Direct update can also
set enabled. Delete removes a schedule permanently and requires an unambiguous target;
pass its ETag with --if-match. If "stop this schedule" could mean pause or delete,
resolve that consequential choice. These writes need Execute.

Done when the exact target and requested lifecycle action are unambiguous and the resulting state or confirmed deletion is verified.

### Explaining a schedule

Resolve an indirect reference with `autopi scheduled-executions list` when needed, then read it with `autopi scheduled-executions get`. Report its recurrence, configured time zone, state, and relevant command, input, or tag details in user terms. Keep sensitive values hidden and preserve a named time zone instead of converting it to UTC.

Done when the explanation covers the effective recurrence, named time zone, state, command, non-sensitive bindings, and behavior-affecting tags.

### Schedule lifecycle

Recurring schedules (`hourly`, `daily`, `weekly`, `monthly`, and `yearly`) have no end date or occurrence count. They run until disabled or deleted; Automation Pilot has no native "run N times" or "run until date" option.

A one-time schedule fires at the next occurrence of its month/day/time, becomes `COMPLETED`, and does not run again. Automation Pilot retains it by default. If the product has Auto-Delete Schedule enabled, Automation Pilot deletes it after the configured retention period.

## Error handling

Report tool errors without inventing missing state.

For a missed trigger, inspect the schedule's existence, state, configured time, time-zone interpretation, date edge cases, errors, and resolved input metadata. Investigate the execution only when the schedule created one.

## Definition of done

The exact schedule and command are identified, every required input is bound compatibly, and the
effective timing is expressed in the requested time zone. A create or update is read back and
matches the intended complete definition. A lifecycle action is verified. An explanation or
missed-trigger diagnosis accounts for the schedule evidence without exposing sensitive values.
