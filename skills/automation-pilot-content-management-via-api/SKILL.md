---
name: automation-pilot-content-management-via-api
description: Manage SAP Automation Pilot content (commands, catalogs, inputs) via the Content API. Use when importing or exporting commands, listing catalogs, or managing automation content programmatically.
---

# SAP Automation Pilot Content API Management

This skill manages catalogs, commands, and inputs in SAP Automation Pilot using the
Content API, driven by the `autopi` CLI.

**API access:** all calls use the `autopi` CLI — see
[`../automation-pilot-cli/SKILL.md`](../automation-pilot-cli/SKILL.md) for setup
(the `AUTOPI_*` environment variables, supported regional hosts, install, and the full
command surface). Add the CLI's `scripts/` to your `PATH` so `autopi` runs directly.

## Command Release Policy

⚠️ Commands deploy in DRAFT state. **Never release automatically** — only when the user
explicitly requests it and the command has been tested. See
`../automation-pilot-command-generation/SKILL.md` for the full policy.

## Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Commands | PascalCase | `RestartCfApp`, `GetHanaInstance` |
| Inputs | PascalCase | `BtpCredentials`, `JiraConfig` |
| Catalogs | kebab-case | `my-automations-xxx` |

Names must not contain spaces (causes API failures with URL encoding).

## Values Are Always Strings on the Wire

Every entry in `values`, `defaultValue`, and `inputValues` must be a JSON **string** on
the wire, regardless of the declared `type`. See the canonical rule and the
type→wire table in
[`../automation-pilot-cli/SKILL.md`](../automation-pilot-cli/SKILL.md#values-are-always-strings-on-the-wire).

---

# Catalogs

Lists return metadata only; add `--full` for complete objects. `get` is always full.

```bash
# List catalogs
autopi catalogs list --own                       # owned by the tenant
autopi catalogs list --provided                  # provided by SAP
autopi catalogs list                             # all
autopi catalogs list --tag environment:production

# Get, create, update, delete
autopi catalogs get mycatalog-xxx
autopi catalogs create -f catalog.json
autopi catalogs update mycatalog-xxx -f catalog.json
autopi catalogs delete mycatalog-xxx             # catalog must be empty
```

Create/update body shape:

```json
{
  "technicalName": "my-catalog",
  "name": "my-catalog",
  "description": "My custom catalog for automation commands",
  "tags": { "environment": "development" }
}
```

`technicalName` is **required** (omitting it returns
`400 Violation in field 'technicalName': must not be null`). The resulting catalog
id is `<technicalName>-<tenant>` (e.g. `my-catalog-xxx`) — use that id for
`get`/`update`/`delete` and as the `--catalog` for commands/inputs.

---

# Commands

Command ID format: `catalog:name:version` (e.g. `mycatalog-xxx:MyCommand:1`).

```bash
# List
autopi commands list --catalog mycatalog-xxx
autopi commands list --catalog mycatalog-xxx --include-issues   # with design-time issues
autopi commands list --tag type:monitoring
autopi commands ids --catalog mycatalog-xxx                     # IDs only (lightweight)

# Get
autopi commands get mycatalog-xxx:MyCommand:1

# Create / update from a file
autopi commands create -f path/to/MyCommand.command.json
autopi commands update mycatalog-xxx:MyCommand:1 -f path/to/MyCommand.command.json

# Create inline
autopi commands create --data '{
  "name": "MyCommand", "catalog": "mycatalog-xxx", "version": 1,
  "description": "My custom command",
  "inputKeys": {"message": {"type": "string", "description": "Message to process", "required": true}},
  "outputKeys": {"result": {"type": "string", "description": "Processing result"}},
  "configuration": null, "tags": {}
}'

# Delete
autopi commands delete mycatalog-xxx:MyCommand:1

# Lifecycle
autopi commands release mycatalog-xxx:MyCommand:1     # ⚠️ ONLY IF THE USER EXPLICITLY REQUESTED IT
autopi commands deprecate mycatalog-xxx:MyCommand:1
autopi commands restore mycatalog-xxx:MyCommand:1

# Bulk create — body is {"commands": [ {...}, {...} ]}
autopi commands bulk-create -f commands.json
```

---

# Inputs

Input ID format: `catalog:name:version`. Inputs are **reusable parameter sets** (saved
credentials, configurations, common values) referenced when triggering executions. See
[`references/input-patterns.md`](references/input-patterns.md) for the input structure
and ready-to-use patterns (CF credentials, service keys, API tokens, kubeconfig) and
best practices.

```bash
# List
autopi inputs list --catalog mycatalog-xxx
autopi inputs list --tag type:credentials

# Get
autopi inputs get mycatalog-xxx:MyInput:1

# Create / full update from a file (or --data for inline)
autopi inputs create -f MyInput.input.json
autopi inputs update mycatalog-xxx:MyInput:1 -f MyInput.input.json

# Partial update — rotate individual values only (merge-patch)
autopi inputs patch mycatalog-xxx:MyInput:1 --data '{"values": {"password": "<your-new-password>"}}'

# Delete
autopi inputs delete mycatalog-xxx:MyInput:1

# Lifecycle
autopi inputs release mycatalog-xxx:MyInput:1     # ⚠️ ONLY IF THE USER EXPLICITLY REQUESTED IT
autopi inputs deprecate mycatalog-xxx:MyInput:1
autopi inputs restore mycatalog-xxx:MyInput:1

# Bulk create — body is {"inputs": [ {...}, {...} ]}
autopi inputs bulk-create -f inputs.json
```

`autopi inputs patch` sends `Content-Type: application/merge-patch+json` for you — send
only the keys you want to change, never the whole input.

---

# Utility Patterns

```bash
# Export a command to a file
autopi commands export mycatalog-xxx:MyCommand:1 -o MyCommand.command.json

# Upload every command in a directory (deploy = upsert, so re-runs are safe)
for f in ./commands/*.command.json; do autopi commands deploy -f "$f"; done

# Check if a command exists (prints 200 or 404, never errors)
autopi commands get mycatalog-xxx:MyCommand:1 --status-only

# Create or update a single command (upsert: GET then PUT or POST)
autopi commands deploy -f MyCommand.command.json
```

`deploy` derives the ID from the file's own `id`/`name`+`catalog`+`version`; pass
`--id` to override.

---

# Error Handling

`autopi` exits with a distinct code per failure class (0 ok, 2 config, 3 network, 4 HTTP
4xx, 5 HTTP 5xx) and prints the API's error body before failing. The full exit-code and
HTTP-status reference lives in
[`../automation-pilot-cli/references/exit-codes.md`](../automation-pilot-cli/references/exit-codes.md).

| Operation | Required Permission |
|-----------|-------------------|
| List/Get resources | `Read` |
| Create/Update/Delete resources | `Write` |

---

# Mandatory Description Patterns

⚠️ **When creating or updating commands/inputs, use these exact description patterns for
standard parameters.** See the authoritative list in
**[`../automation-pilot-command-generation/references/description-patterns.md`](../automation-pilot-command-generation/references/description-patterns.md)**.

---

# Examples & Troubleshooting

Worked end-to-end examples (deploy from file, export a whole catalog, rotate credentials)
and content-specific troubleshooting are in [`references/examples.md`](references/examples.md).

---

⚠️ **CAUTION: DO NOT RELEASE COMMANDS & INPUTS IF NOT EXPLICITLY REQUESTED BY THE USER**
