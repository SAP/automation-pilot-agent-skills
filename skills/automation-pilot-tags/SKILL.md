---
name: automation-pilot-tags
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: >
  Validates, applies, and interprets SAP Automation Pilot tags. Use when selecting regular or feature tags, configuring priority, retention, auto-abort, or execution logging, filtering resources by tag, or interpreting tag inheritance, origin, and execution lineage from service tags.
---

# Tags in SAP Automation Pilot

Tags attach metadata to commands, inputs, executions, and scheduled executions. The API represents them as a flat string-to-string map. Use an empty JSON string for a tag with no value:

```json
{
  "team": "sre",
  "feature:logs": ""
}
```

Choose tags for a stated purpose. Regular tags identify and group resources, feature tags change runtime behavior, and service tags describe state that Automation Pilot owns. Do not put credentials or other secrets in tags.

## Categories and ownership

| Category | Prefix | User-writable | Purpose |
|---|---|---|---|
| Regular | none | Yes | Free-form metadata for grouping, filtering, and selection |
| Feature | `feature:` | Yes, from the allow-list below | Enables optional product behavior |
| Service | `autopi:` | No | Platform-managed lifecycle, origin, status, and lineage metadata |

Use the canonical lowercase prefixes. Service-prefix matching is case-insensitive, so a case variation does not bypass the restriction. Do not create, update, or copy an `autopi:` tag in a user-authored payload. Automation Pilot strips source service tags during triggering and stamps service tags for the new execution.

## Feature tags

Automation Pilot validates feature tags against a fixed allow-list. All supported feature tags
apply to commands, executions, and scheduled executions.

| Tag | Value and behavior |
|---|---|
| `feature:priority` | `high`, `medium`, or `low`. When executions exceed the count limit, Automation Pilot deletes lower-priority executions first. |
| `feature:autoAbort` | Integer hours from `1` through `168`. Automation Pilot aborts a failed execution that remains unmodified for that period. |
| `feature:retentionPolicy` | `archive` or `delete`. By default, Automation Pilot archives completed executions. |
| `feature:retentionPeriod` | Integer hours. For `archive`: `1`-`672`; for `delete`: `0`-`672`. With deletion, `0` means delete at completion. The default retention period is 28 days. |
| `feature:logs` | Empty string. Captures detailed executor logs. Enable logging before the execution starts; you cannot add it after the run begins. Automation Pilot retains logs for 72 hours after completion. |

Commands pass their applicable feature tags to executions. Schedules can override inherited command features for the executions they trigger. A direct trigger can override an inherited command feature in the same way.

For a run that may need troubleshooting, add `feature:logs` before triggering it.

## Regular tags

Regular tags carry user-defined metadata such as environment, team, workflow, or instance:

```json
{
  "environment": "production",
  "team": "sre",
  "workflow": "weekly-restart"
}
```

Validation rules:

- Key: required, `1`-`128` characters, not blank, and must not contain `=`.
- Value: optional and at most `256` characters. Use `""` for a tag without a value.
- Keys and values may use ASCII letters, digits, spaces, and `_ . : / = + - @`; the key's `=` restriction still applies.
- Automation Pilot reserves the `autopi:` and `feature:` prefixes; feature tags must match the allow-list.

Regular tags can drive API filtering and custom UI columns. Choose stable keys and values for filters and automation.

## Inheritance and precedence

Think of command tags as defaults:

- A direct execution starts with the command's regular and feature tags. Automation Pilot adds tags from the trigger request and replaces command tags that share a key.
- A scheduled execution starts with the command's regular and feature tags. Automation Pilot adds the schedule's tags and replaces command tags that share a key.
- A child execution receives the tags supplied by its caller. Automation Pilot adds the service tags that describe its parent and execution lineage.

Automation Pilot does not copy service tags from the source. It creates `autopi:` tags for each new execution.

Put a tag on the command when it should apply to each run, on the schedule when it should apply to that recurring run, or in the trigger request for one run. Preserve existing schedule tags during updates unless the user asks to remove or replace them.

## Service tags worth recognizing

Use service tags to interpret and filter returned resources. Users do not author these tags. Common examples include:

| Tag | Meaning |
|---|---|
| `autopi:scheduledExecution` | Contains the scheduled-execution ID that triggered this execution. Use it to trace a run back to its schedule. |
| `autopi:userAgent` | Contains the HTTP `User-Agent` for an API-triggered execution by a technical user, truncated to 256 characters. Use it to identify the calling client. |
| `autopi:mcpServerId` | Contains the MCP server ID when an MCP tool call triggered the execution. Use it to identify the originating server. |
| `autopi:mcpToolName` | Contains the MCP tool name when an MCP tool call triggered the execution. Use it with `autopi:mcpServerId` to identify the originating operation. |
| `autopi:statusListenerOf` | Contains the ID of the execution whose status change triggered this execution. Use it to trace status-listener chains. |
| `autopi:childExecutionOf` | Contains the parent execution ID for a sub-execution created by `TriggerExecution` or `ParallelForEach`. |
| `autopi:parentExecutorPath` | Contains the full path of the executor in the parent that created the sub-execution, for example `parentAlias.childAlias.triggerStep`. |

Use `automation-pilot:list_executions`' `tag` filter to scope history or analytics by inherited regular tags, feature tags, or platform-stamped service tags.
