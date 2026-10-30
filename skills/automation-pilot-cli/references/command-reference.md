# autopi command reference

Generated from the bundled CLI. Run
`autopi <resource> <verb> --help` for usage, defaults, and examples.

For wire types, limits, permissions, HTTP outcomes, and schema exceptions, read
[API contracts](api-contract.md). For workflows, read [recipes](recipes.md).

## Contents

- [Shared options](#shared-options)
- [connection](#connection)
- [catalogs](#catalogs)
- [commands](#commands)
- [inputs](#inputs)
- [webhooks](#webhooks)
- [mcp-servers](#mcp-servers)
- [executions](#executions)
- [scheduled-executions](#scheduled-executions)
- [triggers](#triggers)
- [Raw API and validation](#raw-api-and-validation)

## Shared options

| Option | Behavior |
|---|---|
| `--output json/raw` | Pretty JSON or verbatim body |
| `--response-envelope` | JSON status, selected headers, and body; mutually exclusive with raw/status/ETag selectors |
| `--print-request` | Redacted offline preview; no API calls |
| `--timeout` | Total request deadline, default 30s |
| `--read-retries N` | GET/HEAD 429/503 only, 0–5 retries within the same deadline |
| `--insecure` | Explicit TLS-verification escape hatch |

Bodies support `-f FILE`, `-f -`, `--data JSON`, `--data @file`, and `--data @-`.
Bearer authentication is the only supported mode; connection credentials are environment-only.
See the [CLI entrypoint](../SKILL.md) for setup and response protection.

## connection

| Verb | Calls | Specific flags |
|---|---|---|
| `preflight` | local only (no API calls) |  |
| `info` | GET /server-info + /user-information + /tenant |  |

## catalogs

| Verb | Calls | Specific flags |
|---|---|---|
| `list` | GET /catalogs | `--full`, `--own`, `--provided`, `--tag` |
| `get` | GET /catalogs/{id} | `--status-only` |
| `create` | POST /catalogs | `--data`, `-f`, `--file` |
| `update` | PUT /catalogs/{id} | `--data`, `-f`, `--file` |
| `delete` | DELETE /catalogs/{id} | `--force` |

## commands

| Verb | Calls | Specific flags |
|---|---|---|
| `list` | GET /commands | `--catalog`, `--full`, `--include-issues`, `--tag` |
| `ids` | GET /commands/ids | `--catalog`, `--tag` |
| `get` | GET /commands/{id} | `--include-issues`, `--show-etag`, `--status-only` |
| `contract` | GET /commands/{id} |  |
| `create` | POST /commands | `--data`, `-f`, `--file` |
| `update` | PUT /commands/{id} | `--data`, `-f`, `--file`, `--if-match` |
| `delete` | DELETE /commands/{id} | `--if-match` |
| `release` | PUT /commands/{id}/release | `--if-match` |
| `deprecate` | PUT /commands/{id}/deprecate | `--if-match`, `--replacement` |
| `restore` | PUT /commands/{id}/restore | `--if-match` |
| `bulk-create` | POST /bulk/commands | `--data`, `-f`, `--file` |
| `export` | GET /commands/{id} | `-o`, `--output-file` |
| `deploy` | GET then PUT\|POST /commands | `--data`, `-f`, `--file`, `--id`, `--if-match` |

## inputs

| Verb | Calls | Specific flags |
|---|---|---|
| `list` | GET /inputs | `--catalog`, `--full`, `--tag` |
| `get` | GET /inputs/{id} | `--full`, `--show-etag`, `--status-only` |
| `create` | POST /inputs | `--data`, `-f`, `--file` |
| `update` | PUT /inputs/{id} | `--data`, `-f`, `--file`, `--if-match` |
| `patch` | PATCH /inputs/{id} | `--data`, `-f`, `--file`, `--if-match` |
| `delete` | DELETE /inputs/{id} | `--if-match` |
| `release` | PUT /inputs/{id}/release | `--if-match` |
| `deprecate` | PUT /inputs/{id}/deprecate | `--if-match`, `--replacement` |
| `restore` | PUT /inputs/{id}/restore | `--if-match` |
| `bulk-create` | POST /bulk/inputs | `--data`, `-f`, `--file` |
| `deploy` | GET then PUT\|POST /inputs | `--data`, `-f`, `--file`, `--id`, `--if-match` |

## webhooks

| Verb | Calls | Specific flags |
|---|---|---|
| `list` | GET /webhooks | `--full` |
| `get` | GET /webhooks/{id} | `--show-etag`, `--status-only` |
| `create` | POST /webhooks | `--data`, `-f`, `--file` |
| `update` | PUT /webhooks/{id} | `--data`, `-f`, `--file`, `--if-match` |
| `delete` | DELETE /webhooks/{id} | `--if-match` |
| `trigger` | POST /webhooks/{id}/trigger | `--content-type`, `--data`, `-f`, `--file` |
| `validate` | GET /commands/{id}, /inputs/{id}; no mutations | `-f`, `--file`, `--metadata` |

## mcp-servers

| Verb | Calls | Specific flags |
|---|---|---|
| `list` | GET /mcp-servers | `--full` |
| `get` | GET /mcp-servers/{id} | `--show-etag`, `--status-only` |
| `create` | POST /mcp-servers | `--data`, `-f`, `--file` |
| `update` | PUT /mcp-servers/{id} | `--data`, `-f`, `--file`, `--if-match` |
| `delete` | DELETE /mcp-servers/{id} | `--if-match` |
| `deploy` | GET then PUT\|POST /mcp-servers | `--data`, `-f`, `--file`, `--id`, `--if-match` |
| `validate` | GET /commands/{id}, /inputs/{id}; no API calls with --metadata | `--client-schema`, `-f`, `--file`, `--metadata` |

## executions

| Verb | Calls | Specific flags |
|---|---|---|
| `list` | GET /executions | `--command-id`, `--execution-id`, `--filter-mode`, `--full`, `--limit`, `--originator-id`, `--started-after`, `--started-before`, `--status`, `--tag` |
| `get` | GET /executions/{id} | `--show-etag`, `--status-only` |
| `summary` | GET /executions/summary | `--started-after`, `--started-before` |
| `input` | GET /executions/{id}/input |  |
| `output` | GET /executions/{id}/output |  |
| `logs` | GET /executions/{id}/logs[/{executor}] | `--executor`, `--max-page-size`, `--messages`, `--page` |
| `trigger` | POST /executions | `--command-id`, `--correlation-id`, `--data`, `--dry-run`, `-f`, `--file`, `--input`, `--input-json`, `--input-reference`, `--logs`, `--tag`, `--wait-seconds` |
| `action` | POST /executions/{id}/actions | `--correlation-id`, `--data`, `-f`, `--file`, `--if-match`, `--reason`, `--type`, `--value` |
| `bulk-action` | POST /executions/actions | `--data`, `--execution-id`, `-f`, `--file`, `--reason`, `--type` |
| `actions` | GET /executions/{id}/actions |  |
| `delete` | DELETE /executions/{id} | `--if-match` |

## scheduled-executions

| Verb | Calls | Specific flags |
|---|---|---|
| `list` | GET /scheduled-executions | `--full` |
| `get` | GET /scheduled-executions/{id} | `--show-etag`, `--status-only` |
| `create` | POST /scheduled-executions | `--data`, `-f`, `--file` |
| `update` | PUT /scheduled-executions/{id} | `--data`, `-f`, `--file`, `--if-match` |
| `delete` | DELETE /scheduled-executions/{id} | `--if-match` |
| `enable` | GET then PUT /scheduled-executions/{id} |  |
| `disable` | GET then PUT /scheduled-executions/{id} |  |
| `time-zones` | GET /scheduled-executions/time-zones |  |

## triggers

| Verb | Calls | Specific flags |
|---|---|---|
| `generic-event` | POST /triggers/generic-event | `--command-reference`, `--content-type`, `--correlation-id`, `--data`, `-f`, `--file`, `--input-reference`, `--map-event-to-input-key`, `--tag` |
| `ans-event` | POST /triggers/ans-event | `--command-reference`, `--correlation-id`, `--data`, `--disable-region-properties-mapping`, `--disable-resource-properties-mapping`, `-f`, `--file`, `--input-reference`, `--map-event-to-input-key`, `--resource-group`, `--resource-instance`, `--resource-name`, `--sub-account`, `--tag` |

## Raw API and validation

```text
autopi api METHOD PATH [-f FILE|- | --data JSON|@file|@-] [--header K:V] [--content-type TYPE]
autopi lint [--kind KIND] [--mode authoring|api] [--read-back] [--output text|json] FILE...
autopi policy
autopi surface
```

Raw API preserves the configured authentication and documented output protection,
and bypasses the pinned request validator. Custom Authorization is rejected.
Lint separates API request requirements from naming/sensitivity/expression policy.
Exit 6 means lint findings; exit 7 means one or more bulk action items failed.
