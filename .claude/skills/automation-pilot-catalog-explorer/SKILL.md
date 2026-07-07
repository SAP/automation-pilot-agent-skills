---
name: automation-pilot-catalog-explorer
description: Discover available commands and catalogs in SAP Automation Pilot via API. Use when you need to find what executors/commands exist, get command definitions, or explore a catalog before generating new commands. Essential for complex command generation involving multiple services.
---

# Catalog Explorer

Discover available commands in SAP Automation Pilot dynamically via API.

## Prerequisites

Environment variables must be set:

```bash
export AUTOPI_HOSTNAME="emea.autopilot.cloud.sap"
export AUTOPI_USERNAME="your-username"
export AUTOPI_PASSWORD="your-password"
```

## API Operations

### List All Catalogs

Discover what catalogs are available in the tenant.

```bash
# List all catalogs (SAP-provided + custom)
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/catalogs"

# List only SAP-provided catalogs (built-in)
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/catalogs?provided=true"

# List only custom/tenant catalogs
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/catalogs?own=true"
```

### List Commands in a Catalog

```bash
CATALOG="applm-sapcp"
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/commands?catalog=$CATALOG"
```

### Get Command Definition

```bash
COMMAND_ID="applm-sapcp:RestartCfApp:1"
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/commands/$COMMAND_ID"
```

### Search Across Catalogs

Find commands matching a pattern.

```bash
# List all SAP-provided catalog IDs
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/catalogs?own=false"

# Search specific catalogs for commands matching a keyword
for catalog in applm-sapcp sm-sapcp cf-sapcp; do
  echo "=== $catalog ==="
  curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
    "https://$AUTOPI_HOSTNAME/api/v1/commands?catalog=$catalog"
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
