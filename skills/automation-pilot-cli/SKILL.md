---
name: automation-pilot-cli
description: The autopi CLI — the single tool every Automation Pilot skill uses to call the SAP Automation Pilot REST API. Use whenever you need to authenticate, set the regional host, or run any API call (catalogs, commands, inputs, mcp-servers, a2a-adapters, executions, scheduled-executions, triggers) in place of curl. Also provides `autopi lint` to validate command/input JSON. Prebuilt, cross-platform (macOS/Linux/Windows), zero runtime dependencies.
---

# autopi — the Automation Pilot CLI

`autopi` is a single, dependency-free command-line tool that drives the entire SAP
Automation Pilot REST API. It replaces every hand-written `curl` block across the
Automation Pilot skills. **Every skill that touches the API calls `autopi`** — this
file is the one place authentication, hostnames, and the command surface are defined.

The API is uniform: one base URL (`https://$AUTOPI_HOSTNAME/api/v1/...`), HTTP Basic
auth, standard REST verbs. `autopi` maps that onto resource subcommands plus a raw `api`
escape hatch for anything not yet wrapped.

## Setup (do this once per session)

Set the following environment variables. `autopi` reads them directly — the password is
never written to logs, echoed, or embedded in URLs.

```bash
export AUTOPI_HOSTNAME="emea.autopilot.cloud.sap"
export AUTOPI_USERNAME="your-username"
export AUTOPI_PASSWORD="your-password"
```

**Supported regional hosts:**

| Host | Region |
|------|--------|
| `emea.autopilot.cloud.sap` | Europe (default) |
| `amer.autopilot.cloud.sap` | Americas |
| `apac.autopilot.cloud.sap` | Asia Pacific |
| `aus.autopilot.cloud.sap`  | Australia |
| `ksa.autopilot.cloud.sap`  | Saudi Arabia |

Each flag has an env var and vice-versa: `--host`/`AUTOPI_HOSTNAME`,
`--user`/`AUTOPI_USERNAME`, `--password`/`AUTOPI_PASSWORD`.
A flag always wins over the env var. Prefer the env var for the password.

There is **no default catalog**. `--catalog` is an explicit filter on the `list`/`ids`
verbs only, and it has no env var — name the catalog every time you list or author
content. Never assume a catalog: propose a catalog id and confirm it with the user.

## Running it

Invoke the platform selector in this skill's `scripts/` directory — it detects the OS
and architecture and execs the matching prebuilt static binary bundled alongside it. No
build step, no network, no `curl`, no `jq` required by the tool itself.

```bash
# macOS / Linux
<path-to-this-skill>/scripts/autopi version

# Windows
<path-to-this-skill>\scripts\autopi.cmd version
```

Add `scripts/` to your `PATH`, or alias `autopi`, so the examples below work verbatim.
If a prebuilt binary for your platform is missing, rebuild all of them with
`cli/build.sh` (needs a Go toolchain — build-time only). See
[`references/build.md`](references/build.md).

## The command surface

```
autopi <resource> <verb> [flags]
autopi api <METHOD> <path> [--data @file|json]   # raw escape hatch
autopi lint <files...>                           # validate command/input JSON
```

| Resource | Verbs | Endpoint family |
|----------|-------|-----------------|
| `connection` | info | `/server-info`, `/user-information`, `/tenant` |
| `catalogs` | list / get / create / update / delete | `/catalogs` |
| `commands` | list / ids / get / **contract** / create / update / delete / release / deprecate / restore / bulk-create / export / deploy | `/commands`, `/bulk/commands` |
| `inputs` | list / get / create / update / **patch** / delete / release / deprecate / restore / bulk-create / deploy | `/inputs`, `/bulk/inputs` |
| `mcp-servers` | list / get / create / update / delete / deploy | `/mcp-servers` |
| `a2a-adapters` | list / get / create / update / delete / deploy | `/a2a-adapters` |
| `executions` | list / get / summary / input / output / logs / trigger / action / actions / delete | `/executions` |
| `scheduled-executions` | list / get / create / update / delete / time-zones | `/scheduled-executions` |
| `triggers` | generic-event / ans-event | `/triggers` |

**The CLI's own help is the source of truth for flags.** Don't memorize flags — ask it:

```bash
autopi help                       # overview (resources, env, exit codes)
autopi <resource> --help          # verbs for a resource
autopi <resource> <verb> --help   # the exact flags, the endpoint it calls, an example
```

### Common examples

```bash
# Verify connectivity and permissions (no secrets printed)
autopi connection info

# Discover
autopi catalogs list --provided
autopi commands list --catalog applm-sapcp --include-issues
autopi commands get applm-sapcp:RestartCfApp:1
autopi commands contract applm-sapcp:RestartCfApp:1   # contract only (no configuration)

# Deploy content (upsert = GET then PUT or POST; stays in DRAFT)
autopi commands deploy -f MyCommand.command.json
for f in ./commands/*.command.json; do autopi commands deploy -f "$f"; done
autopi commands export applm-sapcp:RestartCfApp:1 -o RestartCfApp.command.json

# Rotate one credential value without replacing the whole input (merge-patch)
autopi inputs patch my-cat:JiraCredentials:1 --data '{"values":{"password":"<new>"}}'

# Run and watch
autopi executions trigger --command-id my-cat:MyCommand:1 --input region=cf-eu10 --dry-run
autopi executions summary
autopi executions logs <execution-id> --page 0 --max-page-size 100

# Anything not wrapped — the escape hatch (auth + base URL injected)
autopi api GET 'commands?catalog=applm-sapcp'
```

Output is pretty-printed JSON by default; `--output raw` emits the body verbatim for
piping into `jq`. `jq` stays an optional external tool for shaping output — it is never
required to make a call.

## List output is summarized by default

To keep listings from flooding the context, every `list` (and `executions list`) trims
the heavy blocks — `configuration`, `executors`, `steps`, `inputKeys`, `outputKeys`,
`keys`, `values`, `mcpTools`, `skills`, `schedule`, etc. — and returns only the metadata
a listing is for: `id`, `name`, `catalog`, `description`, `version`, `tags`, `enabled`,
`commandId`. Pass `--full` to get the complete objects:

```bash
autopi commands list --catalog applm-sapcp              # metadata summary (default)
autopi commands list --catalog applm-sapcp --full       # complete objects
autopi commands list --catalog applm-sapcp --full --output raw | jq '.[].executors'
```

`get <id>` is **always** full — the summary applies to list views only — **except
`inputs get`, which omits the secret `values` block unless you pass `--full`** (stored
secrets can never be read back regardless). `commands ids` is already minimal (IDs only)
and has no `--full`. This is purely a view concern: `--full` changes *which fields* come
back, orthogonal to `--output json|raw` formatting.

## Values Are Always Strings on the Wire

The Automation Pilot API is **string-typed for input and output values**. When creating
or updating inputs, every entry in `values` and `defaultValue` must be a **JSON string**
regardless of the declared `type` in `keys`; outputs come back as strings too:

| Declared `type` | Wire representation |
|-----------------|---------------------|
| `string` | `"my-app"` |
| `number` | `"42"` |
| `boolean` | `"true"` |
| `array` | `"[\"a\",\"b\"]"` |
| `object` | `"{\"url\":\"...\",\"user\":\"...\"}"` |

This also applies to `inputValues` on trigger and schedule endpoints. Convert inside
expressions with `toNumber` / `toObject` / `toString` when you need the logical type.
This is an API-wide behavior, so it is documented **once here**; the content, executions,
and command-generation skills point to this section rather than restating it.

## Dry-run / request preview

`--print-request` renders the exact METHOD, URL, headers, and body **without sending**
and **without needing credentials**. The `Authorization` header is masked as `Basic ***`.
Use it to preview a call or to confirm a body is shaped correctly.

```bash
autopi inputs patch my-cat:JiraCredentials:1 --data '{"values":{"password":"x"}}' --print-request
# PATCH https://emea.autopilot.cloud.sap/api/v1/inputs/my-cat:JiraCredentials:1
# Authorization: Basic ***
# Content-Type: application/merge-patch+json
# { "values": { "password": "x" } }
```

## Existence checks

`--status-only` prints just the HTTP status code and never errors on a non-2xx response
— the replacement for `curl -o /dev/null -w "%{http_code}"`:

```bash
autopi commands get my-cat:MyCommand:1 --status-only   # -> 200 or 404
```

## Exit codes

`autopi` distinguishes transport failures from HTTP errors so callers can branch:

| Code | Meaning |
|------|---------|
| 0 | success |
| 1 | usage error (bad flags/arguments) |
| 2 | config error (missing host/user/password) |
| 3 | network / TLS / timeout — no HTTP response |
| 4 | HTTP 4xx |
| 5 | HTTP 5xx |
| 6 | `lint` reported findings |

Full table and troubleshooting: [`references/exit-codes.md`](references/exit-codes.md).

## Validation (`autopi lint`)

Mechanical checks on command/input/catalog JSON before deploying — naming conventions,
the `sensitive` flag on credential keys, and expression-safety heuristics. `--output json`
for machine consumption. Exits 6 when anything is reported. Used by the
`automation-pilot-command-authoring` skill before deploying.

```bash
autopi lint ./commands/*.command.json
autopi lint --output json ./MyInput.input.json
```

## References

- [`references/command-reference.md`](references/command-reference.md) — every resource, verb, and flag.
- [`references/exit-codes.md`](references/exit-codes.md) — exit codes and troubleshooting.
- [`references/build.md`](references/build.md) — rebuilding the binaries from source.

## Troubleshooting

- **Exit 2 "config error"** — a required env var is unset. Check `AUTOPI_HOSTNAME`,
  `AUTOPI_USERNAME`, `AUTOPI_PASSWORD`. `autopi ... --print-request` works without them.
- **Exit 4 with HTTP 401** — credentials wrong for that host. Verify user/password match
  the `AUTOPI_HOSTNAME` region.
- **Exit 4 with HTTP 403** — the user lacks permission (`Write` for create/update/delete).
- **Exit 4 with HTTP 409 on create** — the resource exists; use `deploy` (upsert) instead
  of `create`.
- **Exit 3** — network/TLS/timeout. Check the host and connectivity; `--timeout 60s` to
  extend, `--insecure` only as a deliberate escape hatch.
- **"no prebuilt binary for <os>-<arch>"** — rebuild with `cli/build.sh` (see
  [`references/build.md`](references/build.md)).
