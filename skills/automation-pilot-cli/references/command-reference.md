# autopi command reference

The binary's own `--help` is the authoritative, always-current source. This page mirrors
it for quick scanning. Run `autopi <resource> <verb> --help` for the live flag list, the
endpoint a verb calls, and an example.

## Connection configuration

Set `AUTOMATION_PILOT_HOSTNAME` to the regional hostname (without a scheme or path) and
`AUTOMATION_PILOT_TOKEN` to the token without the `Bearer ` prefix. Connection
configuration is environment-only. Legacy `AUTOPI_*` variables are unsupported.

## Operational flags

| Flag | Env var | Meaning |
|------|---------|---------|
| `--catalog` | — | catalog-scoped `commands`/`inputs` list/ids verbs only; no default |
| `--output json\|raw` | — | pretty JSON (default) or verbatim body |
| `--timeout 30s` | — | request deadline (default 30 seconds) |
| `--print-request` | — | render a redacted offline request; do not send |
| `--insecure` | — | skip TLS verification (escape hatch) |

Body-bearing verbs (`create`/`update`/`patch`/`deploy`/`bulk-create`/`trigger`/schedule
create+update/`triggers`) accept `-f FILE` / `--file FILE`, or `--data JSON` (also
`--data @file` curl-style). GET-style verbs accept `--status-only` to print just the HTTP
code.

Every `list` verb (and `executions list`) defaults to a **metadata summary** — heavy
blocks (`configuration`/`executors`/`inputKeys`/`outputKeys`/`keys`/`values`/`mcpTools`/
`skills`/`schedule`/…) are stripped, leaving `id`/`name`/`catalog`/`description`/`version`/
`tags`/`enabled`/`commandId`. Pass `--full` for complete objects. `get <id>` is always
full — **except `inputs get`, which omits the secret `values` block unless `--full`** —
and `commands ids` is already minimal and has no `--full`.

## connection

| Verb | Endpoint | Returns |
|------|----------|---------|
| `info` | `GET /server-info` + `GET /user-information` + `GET /tenant` | non-secret `{connected, url, tenantId, username, permissions, plan}` |

`connection info` is the connectivity/identity check: it composes the three reads into one
summary and never prints the credential. Use it to confirm the configured tenant and inspect
`permissions` against the requested task; MCP/A2A writes require `GenAI`.

## catalogs

| Verb | Endpoint | Key flags |
|------|----------|-----------|
| `list` | `GET /catalogs` | `--own` \| `--provided`, `--tag k=v` (repeatable), `--full` |
| `get <id>` | `GET /catalogs/{id}` | `--status-only` |
| `create` | `POST /catalogs` | `-f` / `--data` |
| `update <id>` | `PUT /catalogs/{id}` | `-f` / `--data` |
| `delete <id>` | `DELETE /catalogs/{id}` | — (catalog must be empty) |

## commands

| Verb | Endpoint | Key flags |
|------|----------|-----------|
| `list` | `GET /commands` | `--catalog`, `--tag k=v`, `--include-issues`, `--full` |
| `ids` | `GET /commands/ids` | `--catalog`, `--tag k=v` |
| `get <id>` | `GET /commands/{id}` | `--status-only` |
| `contract <id>` | `GET /commands/{id}` | projects to `id`/`catalog`/`name`/`description`/`version`/`inputKeys`/`outputKeys`/`tags` (strips `configuration`) |
| `create` | `POST /commands` | `-f` / `--data` |
| `update <id>` | `PUT /commands/{id}` | `-f` / `--data` |
| `delete <id>` | `DELETE /commands/{id}` | — |
| `release <id>` | `PUT /commands/{id}/release` | ⚠️ user must explicitly request |
| `deprecate <id>` | `PUT /commands/{id}/deprecate` | — |
| `restore <id>` | `PUT /commands/{id}/restore` | — |
| `bulk-create` | `POST /bulk/commands` | `-f` / `--data` — body `{"commands":[...]}` |
| `export <id>` | `GET /commands/{id}` | `-o FILE` (default stdout) |
| `deploy` | `GET` then `PUT`\|`POST /commands` | `-f FILE`, `--id ID` (upsert; id derived from file) |

IDs are `catalog:name:version`. Colons in IDs are passed through to the path unencoded,
as the API expects.

## inputs

Same verbs as `commands` (`list`/`get`/`create`/`update`/`delete`/`release`/`deprecate`/
`restore`/`bulk-create`/`deploy`), **plus**:

| Verb | Endpoint | Key flags |
|------|----------|-----------|
| `get <id>` | `GET /inputs/{id}` | `--full` (include the secret `values` block; omitted by default), `--status-only` |
| `patch <id>` | `PATCH /inputs/{id}` | `-f` / `--data` — sends `Content-Type: application/merge-patch+json` |

`inputs get` omits the `values` block by default so stored credentials stay out of the
view (stored secrets cannot be read back regardless); pass `--full` to include it.
`patch` is for rotating individual values without replacing the whole input — e.g.
`autopi inputs patch my-cat:JiraCredentials:1 -f - < protected-rotation.json`.
`bulk-create` body is `{"inputs":[...]}`.

## mcp-servers

| Verb | Endpoint |
|------|----------|
| `list` | `GET /mcp-servers` |
| `get <id>` | `GET /mcp-servers/{id}` |
| `create` | `POST /mcp-servers` |
| `update <id>` | `PUT /mcp-servers/{id}` |
| `delete <id>` | `DELETE /mcp-servers/{id}` |
| `deploy` | `GET` then `PUT`\|`POST /mcp-servers` (upsert) |

Write operations require the `GenAI` permission.

## a2a-adapters

| Verb | Endpoint | Key flags |
|------|----------|-----------|
| `list` | `GET /a2a-adapters` | — |
| `get <id>` | `GET /a2a-adapters/{id}` | `--show-etag` (print ETag header), `--status-only` |
| `create` | `POST /a2a-adapters` | `-f` / `--data` |
| `update <id>` | `PUT /a2a-adapters/{id}` | `-f` / `--data`, `--if-match ETAG` (optimistic lock → 412 on mismatch) |
| `delete <id>` | `DELETE /a2a-adapters/{id}` | — |
| `deploy` | `GET` then `PUT`\|`POST /a2a-adapters` | `-f`, `--id` (upsert; no If-Match) |

Write operations require the `GenAI` permission.

## executions

| Verb | Endpoint | Key flags |
|------|----------|-----------|
| `list` | `GET /executions` | `--status` (repeatable), `--command-id` (repeatable), `--execution-id` (repeatable), `--tag` (repeatable), `--limit N`, `--started-after`, `--started-before` (epoch-ms or relative `24h`), `--full` |
| `get <id>` | `GET /executions/{id}` | `--status-only` |
| `summary` | `GET /executions/summary` | — |
| `input <id>` | `GET /executions/{id}/input` | — |
| `output <id>` | `GET /executions/{id}/output` | — |
| `logs <id>` | `GET /executions/{id}/logs[/{executor}]` | `--page`, `--max-page-size`, `--executor PATH` |
| `trigger` | `POST /executions` | `--command-id`, `--input k=v` (repeatable), `--input-json JSON`, `--input-reference ID` (repeatable), `--tag k=v`, `--logs`, or `-f FILE` |
| `action <id>` | `POST /executions/{id}/actions` | `--type PAUSE\|RESUME\|RETRY\|ABORT\|RESET\|COMMENT`, `--reason`, `--value` |
| `actions <id>` | `GET /executions/{id}/actions` | — |
| `delete <id>` | `DELETE /executions/{id}` | — |

`trigger` builds the body for you: `--input` and `--input-json` require string wire values,
`--input-reference` binds reusable Inputs, and `--logs` adds `feature:logs`. Pass `-f`
to send a prebuilt body instead; prebuilt bodies cannot be combined with builder modifiers.

## scheduled-executions

| Verb | Endpoint |
|------|----------|
| `list` | `GET /scheduled-executions` |
| `get <id>` | `GET /scheduled-executions/{id}` |
| `create` | `POST /scheduled-executions` |
| `update <id>` | `PUT /scheduled-executions/{id}` |
| `delete <id>` | `DELETE /scheduled-executions/{id}` |
| `time-zones` | `GET /scheduled-executions/time-zones` |

## triggers

`commandReference` is sent as a query parameter (`--command-reference ID`); the body
(`-f`/`--data`) is the event payload itself, not a wrapper.

| Verb | Endpoint | Flags |
|------|----------|-------|
| `generic-event` | `POST /triggers/generic-event?commandReference=<id>` | `--command-reference ID` (required), `-f FILE` / `--data JSON` |
| `ans-event` | `POST /triggers/ans-event?commandReference=<id>` | `--command-reference ID` (required), `-f FILE` / `--data JSON` |

## api (raw escape hatch)

```
autopi api <METHOD> <path> [-f FILE|- | --data @file|@-|json] [--header 'K: V'] [--content-type CT] [--status-only]
```

The path is appended to `https://<host>/api/v1` (a leading `/api/v1` is tolerated). Bearer
auth from `AUTOMATION_PILOT_TOKEN` is injected; honors `--print-request` and `--output`.
Custom `Authorization` headers are rejected; `--header` is for other headers. Use it
for any endpoint or query shape not covered by a resource verb.

## lint

```
autopi lint [--output text|json] <files...>
```

Validates command/input/catalog JSON. Exits 6 when any finding is reported. See the
`automation-pilot-command-authoring` skill.

Checks include: filename ≠ `name`, non-PascalCase/kebab-case names, length limits,
required authored structure, string wire values, input/output sensitivity, hardcoded defaults on sensitive keys,
non-existent expression functions, URL substitutions without the appropriate encoding,
`autoRetry.delay` / `repeat.delay` / `initialDelay.interval` values outside the server
enum (`5s`, `10s`, `15s`, `30s`, `1m`, `2m`, `3m`), and invalid tags — a `feature:` tag
outside the allow-list or a reserved `autopi:` service tag in authored content (both
rejected by the server at deploy with `400 tags are invalid`).

Prebuilt trigger bodies cannot be combined with builder input, command, reference, tag, or logging
flags. Both body forms support offline previews. All body-bearing verbs accept `-f -` or
`--data @-` for stdin. Exact positional arity and `--` end-of-options are enforced.

`autopi lint FILE --output json` accepts flags before or after files, returns `[]` for no
findings, and exits 6 for any findings. Use `--read-back` only for retrieved definitions.
`autopi policy` returns the shared feature-tag values and composition-duration snapshot.

Update and delete verbs support a non-empty `--if-match ETAG` conditional header.

`autopi surface` returns resource names and verbs from the CLI registry for tool discovery
and complete help smoke checks.
