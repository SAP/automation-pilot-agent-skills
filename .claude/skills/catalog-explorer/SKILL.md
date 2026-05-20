---
name: catalog-explorer
description: Discover available commands and catalogs in SAP Automation Pilot via API. Use when you need to find what executors/commands exist, get command definitions, or explore a catalog before generating new commands. Essential for complex command generation involving multiple services.
---

# Catalog Explorer

Discover available commands in SAP Automation Pilot dynamically via API.

## Prerequisites

Environment variables must be set:

```bash
source .env  # Loads AUTOPI_HOSTNAME, AUTOPI_USERNAME, AUTOPI_PASSWORD
```

## API Operations

### List All Catalogs

Discover what catalogs are available in the tenant.

```bash
source .claude/skills/catalog-explorer/scripts/autopi-catalog-config.sh

# List all catalogs (SAP-provided + custom)
autopi_api GET "/catalogs" | jq '.data[] | {id, name, description}'

# List only SAP-provided catalogs (built-in)
autopi_api GET "/catalogs?own=false" | jq '.data[] | {id, name}'

# List only custom/tenant catalogs
autopi_api GET "/catalogs?own=true" | jq '.data[] | {id, name}'
```

### List Commands in a Catalog

Find available commands within a specific catalog.

```bash
source .claude/skills/catalog-explorer/scripts/autopi-catalog-config.sh

# List all commands in a catalog
CATALOG="applm-sapcp"
autopi_api GET "/commands?catalog=$CATALOG" | jq '.data[] | {id, name, description}'

# Search for commands by name pattern
autopi_api GET "/commands?catalog=$CATALOG" | jq '.data[] | select(.name | test("Restart"; "i")) | {id, name}'

# Get command count
autopi_api GET "/commands?catalog=$CATALOG" | jq '.data | length'
```

### Get Command Definition

Fetch the full definition of a specific command (inputs, outputs, executors).

```bash
source .claude/skills/catalog-explorer/scripts/autopi-catalog-config.sh

# Get full command definition
COMMAND_ID="applm-sapcp:RestartCfApp:1"
autopi_api GET "/commands/$COMMAND_ID" | jq .

# Get just input keys
autopi_api GET "/commands/$COMMAND_ID" | jq '.inputKeys'

# Get just output keys
autopi_api GET "/commands/$COMMAND_ID" | jq '.outputKeys'

# Get executors (the workflow steps)
autopi_api GET "/commands/$COMMAND_ID" | jq '.executors'
```

### Search Across Catalogs

Find commands matching a pattern.

```bash
source .claude/skills/catalog-explorer/scripts/autopi-catalog-config.sh

# First, list all catalogs
autopi_api GET "/catalogs?own=false" | jq -r '.data[].id'

# Then search specific catalogs for commands
for catalog in applm-sapcp sm-sapcp cf-sapcp; do
  echo "=== $catalog ==="
  autopi_api GET "/commands?catalog=$catalog" | jq -r '.data[] | select(.name | test("Service"; "i")) | .id'
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

- **command-generation** — Uses this skill for executor discovery
- **content-management-via-api** — Deploy generated commands
- **executions-api** — Run and monitor commands
