---
name: automation-pilot-catalog-explorer
description: Discover available commands and catalogs in SAP Automation Pilot via API. Use when you need to find what executors/commands exist, get command definitions, or explore a catalog before generating new commands. Essential for complex command generation involving multiple services.
---

# Catalog Explorer

Discover available commands in SAP Automation Pilot dynamically via API.

## API access

All calls use the `autopi` CLI — see [`../automation-pilot-cli/SKILL.md`](../automation-pilot-cli/SKILL.md) for setup (the `AUTOPI_*` environment variables, supported regional hosts, install, and the full command surface). Add the CLI's `scripts/` to your `PATH` so `autopi` runs directly.

## API Operations

### List All Catalogs

Discover what catalogs are available in the tenant.

```bash
# List all catalogs (SAP-provided + custom)
autopi catalogs list

# List only SAP-provided catalogs (built-in)
autopi catalogs list --provided

# List only custom/tenant catalogs
autopi catalogs list --own
```

### List Commands in a Catalog

Lists return lightweight metadata (name, id, catalog, description, tags, …) — ideal for
discovery. Use `commands get` (always full) for a command's executors/inputs/outputs, or
`commands list --full` for complete objects.

```bash
CATALOG="applm-sapcp"
autopi commands list --catalog $CATALOG
```

### Get Command Definition

```bash
COMMAND_ID="applm-sapcp:RestartCfApp:1"
autopi commands get $COMMAND_ID
```

### Search Across Catalogs

Find commands matching a pattern.

```bash
# List all SAP-provided catalog IDs
autopi catalogs list --provided

# Search specific catalogs for commands matching a keyword
for catalog in applm-sapcp sm-sapcp cf-sapcp; do
  echo "=== $catalog ==="
  autopi commands list --catalog "$catalog"
done
```

## Workflow: Complex Command Generation

When generating a command that involves multiple services:

```
1. User: "Create a command that restarts CF apps and checks SM bindings"

2. Discover relevant catalogs:
   - Query: GET /commands?catalog=applm-sapcp
   - Query: GET /commands?catalog=sm-sapcp

3. Find specific commands:
   - Found: applm-sapcp:RestartCfApp:1
   - Found: sm-sapcp:GetServiceBinding:1

4. Fetch definitions to understand parameters:
   - GET /commands/applm-sapcp:RestartCfApp:1
   - GET /commands/sm-sapcp:GetServiceBinding:1

5. Generate composite command using discovered executors
```

## Error Handling

| Error | Meaning | Action |
|-------|---------|--------|
| 401 Unauthorized | Invalid credentials | Check AUTOPI_USERNAME/PASSWORD in .env |
| 404 Not Found | Catalog/command doesn't exist | Use discovery to find correct ID |
| Empty response | No commands in catalog | Catalog may be empty or restricted |

## Related Skills

- **automation-pilot-command-generation** — Uses this skill for executor discovery
- **automation-pilot-content-management-via-api** — Deploy generated commands
- **automation-pilot-executions-api** — Run and monitor commands
