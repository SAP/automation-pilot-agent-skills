---
name: automation-pilot-content-management-via-api
description: Manage SAP Automation Pilot content (commands, catalogs, inputs) via Content API. Use when importing or exporting commands, listing catalogs, or managing automation content programmatically.
---

# SAP Automation Pilot Content API Management

This skill manages catalogs, commands, and inputs in SAP Automation Pilot using the Content API.

## Command Release Policy

⚠️ Commands deploy in DRAFT state. **Never release automatically** — only when the user explicitly requests it and the command has been tested. See `../automation-pilot-command-generation/SKILL.md` for the full policy.

## Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Commands | PascalCase | `RestartCfApp`, `GetHanaInstance` |
| Inputs | PascalCase | `BtpCredentials`, `JiraConfig` |
| Catalogs | kebab-case | `my-automations-xxx` |

Names must not contain spaces (causes API failures with URL encoding).

## Prerequisites

1. Set the following **required** environment variables:

```bash
export AUTOPI_HOSTNAME="emea.autopilot.cloud.sap"
export AUTOPI_USERNAME="your-username"
export AUTOPI_PASSWORD="your-password"
export AUTOPI_DEFAULT_CATALOG="mycommands-<<<TENANT_ID>>>"
```

**Supported hostnames**:
- `emea.autopilot.cloud.sap` (default - Europe)
- `aus.autopilot.cloud.sap` (Australia)
- `apac.autopilot.cloud.sap` (Asia Pacific)
- `amer.autopilot.cloud.sap` (Americas)
- `ksa.autopilot.cloud.sap` (Saudi Arabia)

2. Ensure `curl` is available in your environment.

---

## Values Are Always Strings on the Wire

When creating or updating inputs via the API, every entry in `values` and `defaultValue` must be a **JSON string**, regardless of the declared `type` in `keys`.

| Declared `type` | Wire representation |
|-----------------|---------------------|
| `string` | `"my-app"` |
| `number` | `"42"` |
| `boolean` | `"true"` |
| `array` | `"[\"a\",\"b\"]"` |
| `object` | `"{\"url\":\"...\",\"user\":\"...\"}"` |

This also applies to `inputValues` on trigger and schedule endpoints — every value must be a string on the wire. The runtime converts it to the declared type inside the executor.

---

# Catalogs

## List Catalogs

```bash
# List all catalogs owned by the tenant
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/catalogs?own=true"
# List catalogs provided by SAP
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/catalogs?provided=true"
# List all catalogs (owned + provided)
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/catalogs"
# List catalogs filtered by tag
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/catalogs?tag=environment:production"
```

## Get Catalog by ID

```bash
CATALOG_ID="mycatalog-xxx"
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/catalogs/$CATALOG_ID"
```

## Create Catalog

```bash
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-catalog",
    "description": "My custom catalog for automation commands",
    "tags": {
      "environment": "development"
    }
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/catalogs"
```

## Update Catalog

```bash
CATALOG_ID="mycatalog-xxx"
curl -s -X PUT \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated description",
    "tags": {
      "environment": "production"
    }
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/catalogs/$CATALOG_ID"
```

## Delete Catalog

Note: Catalog must be empty (no commands or inputs) before deletion.

```bash
CATALOG_ID="mycatalog-xxx"
curl -s -X DELETE -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/catalogs/$CATALOG_ID"
```

---

# Commands

## List Commands

```bash
# List all commands in a catalog
CATALOG="mycatalog-xxx"
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/commands?catalog=$CATALOG"
# List commands with design-time issues
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/commands?catalog=$CATALOG&includeIssues=true"
# List commands by tag
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/commands?tag=type:monitoring"
# List only command IDs (lightweight)
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/commands/ids?catalog=$CATALOG"
```

## Get Command by ID

Command ID format: `catalog:name:version`

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/commands/$COMMAND_ID"
```

## Create/Upload Command

Upload a command from a `.command.json` file:

```bash
COMMAND_FILE="path/to/MyCommand.command.json"

# Create the command
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d @"$COMMAND_FILE" \
  "https://$AUTOPI_HOSTNAME/api/v1/commands"
```

Create command inline:

```bash
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MyCommand",
    "catalog": "mycatalog-xxx",
    "description": "My custom command",
    "version": 1,
    "inputKeys": {
      "message": {
        "type": "string",
        "description": "Message to process",
        "required": true
      }
    },
    "outputKeys": {
      "result": {
        "type": "string",
        "description": "Processing result"
      }
    },
    "configuration": null,
    "tags": {}
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/commands"
```

## Update Command

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
COMMAND_FILE="path/to/MyCommand.command.json"

curl -s -X PUT \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d @"$COMMAND_FILE" \
  "https://$AUTOPI_HOSTNAME/api/v1/commands/$COMMAND_ID"
```

## Delete Command

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
curl -s -X DELETE -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/commands/$COMMAND_ID"
```

## Release Command

⚠️ **USE ONLY IF THE USER EXPLICITLY REQUESTED IT**

Release a draft command:

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
curl -s -X PUT -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/commands/$COMMAND_ID/release"
```

## Deprecate Command

Mark a command as deprecated:

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
curl -s -X PUT -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/commands/$COMMAND_ID/deprecate"
```

## Restore Command

Restore a deprecated command:

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
curl -s -X PUT -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/commands/$COMMAND_ID/restore"
```

## Bulk Create Commands

Upload multiple commands at once:

```bash
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "commands": [
      {"name": "Cmd1", "catalog": "mycatalog-xxx", ...},
      {"name": "Cmd2", "catalog": "mycatalog-xxx", ...}
    ]
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/bulk/commands"
```

---

# Inputs

## List Inputs

```bash
# List inputs in a catalog
CATALOG="mycatalog-xxx"
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/inputs?catalog=$CATALOG"
# List inputs by tag
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/inputs?tag=type:credentials"
```

## Get Input by ID

Input ID format: `catalog:name:version`

```bash
INPUT_ID="mycatalog-xxx:MyInput:1"
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/inputs/$INPUT_ID"
```

## Create Input

```bash
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MyCredentials",
    "catalog": "mycatalog-xxx",
    "description": "Service credentials",
    "version": 1,
    "keys": {
      "username": {
        "type": "string",
        "description": "Service username",
        "sensitive": false
      },
      "password": {
        "type": "string",
        "description": "Service password",
        "sensitive": true
      }
    },
    "values": {
      "username": "<your-username>",
      "password": "<your-password>"
    },
    "tags": {
      "type": "credentials"
    }
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/inputs"
```

## Update Input (Full)

```bash
INPUT_ID="mycatalog-xxx:MyInput:1"
curl -s -X PUT \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MyCredentials",
    "catalog": "mycatalog-xxx",
    "description": "Updated credentials",
    "version": 1,
    "keys": {...},
    "values": {...}
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/inputs/$INPUT_ID"
```

## Update Input (Partial - Values Only)

Update only specific values without replacing the entire input:

```bash
INPUT_ID="mycatalog-xxx:MyInput:1"
curl -s -X PATCH \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/merge-patch+json" \
  -d '{
    "values": {
      "password": "<your-new-password>"
    }
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/inputs/$INPUT_ID"
```

## Delete Input

```bash
INPUT_ID="mycatalog-xxx:MyInput:1"
curl -s -X DELETE -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/inputs/$INPUT_ID"
```

## Release Input

⚠️ **USE ONLY IF THE USER EXPLICITLY REQUESTED IT**

```bash
INPUT_ID="mycatalog-xxx:MyInput:1"
curl -s -X PUT -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/inputs/$INPUT_ID/release"
```

## Deprecate Input

```bash
INPUT_ID="mycatalog-xxx:MyInput:1"
curl -s -X PUT -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/inputs/$INPUT_ID/deprecate"
```

## Restore Input

```bash
INPUT_ID="mycatalog-xxx:MyInput:1"
curl -s -X PUT -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/inputs/$INPUT_ID/restore"
```

## Bulk Create Inputs

```bash
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "inputs": [
      {"name": "Input1", "catalog": "mycatalog-xxx", ...},
      {"name": "Input2", "catalog": "mycatalog-xxx", ...}
    ]
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/bulk/inputs"
```

---

# Working with Inputs

Inputs are **reusable parameter sets** stored in Automation Pilot. They allow you to save credentials, configurations, or commonly-used values that can be referenced when triggering command executions.

## Input Structure

```json
{
  "name": "MyCfCredentials",
  "catalog": "mycatalog-xxx",
  "description": "CF credentials for eu10 region",
  "version": 1,
  "keys": {
    "region": { "type": "string", "sensitive": false },
    "user": { "type": "string", "sensitive": false },
    "password": { "type": "string", "sensitive": true }
  },
  "values": {
    "region": "cf-eu10",
    "user": "technical-user",
    "password": "<your-password>"
  },
  "tags": {}
}
```

- **keys**: Defines the schema (type and sensitivity) for each value
- **values**: The actual stored values
- **sensitive: true**: Value is masked in logs and UI

## Common Input Patterns

### CF Credentials (for applm-sapcp, cf-sapcp commands)

```json
{
  "name": "CfCredentials-EU10",
  "catalog": "mycatalog-xxx",
  "description": "Cloud Foundry credentials for EU10",
  "version": 1,
  "keys": {
    "region": { "type": "string", "sensitive": false },
    "user": { "type": "string", "sensitive": false },
    "password": { "type": "string", "sensitive": true }
  },
  "values": {
    "region": "cf-eu10",
    "user": "cf-technical-user@example.com",
    "password": "your-password"
  }
}
```

### Service Key (for sm-sapcp, aicore-sapcp commands)

```json
{
  "name": "HanaCloudServiceKey",
  "catalog": "mycatalog-xxx",
  "description": "HANA Cloud service binding credentials",
  "version": 1,
  "keys": {
    "serviceKey": { "type": "object", "sensitive": true }
  },
  "values": {
    "serviceKey": "{\"url\":\"https://hana-instance.hana.cloud.sap\",\"user\":\"DBADMIN\",\"password\":\"<your-password>\",\"certificate\":\"...\"}"
  }
}
```

### API Token (for jira-sapcp, github-sapcp commands)

```json
{
  "name": "JiraCredentials",
  "catalog": "mycatalog-xxx",
  "description": "JIRA API credentials",
  "version": 1,
  "keys": {
    "host": { "type": "string", "sensitive": false },
    "user": { "type": "string", "sensitive": false },
    "password": { "type": "string", "sensitive": true }
  },
  "values": {
    "host": "https://jira.example.com",
    "user": "automation@example.com",
    "password": "api-token-here"
  }
}
```

### Kubernetes Config (for kubernetes-sapcp commands)

```json
{
  "name": "K8sClusterConfig",
  "catalog": "mycatalog-xxx",
  "description": "Kubernetes cluster kubeconfig",
  "version": 1,
  "keys": {
    "kubeconfig": { "type": "object", "sensitive": true }
  },
  "values": {
    "kubeconfig": "{\"apiVersion\":\"v1\",\"kind\":\"Config\",\"clusters\":[...],\"users\":[...],\"contexts\":[...]}"
  }
}
```

## Best Practices

1. **Separate inputs by environment** - Create distinct inputs for dev/staging/prod
2. **Mark sensitive values** - Always set `"sensitive": true` for passwords, tokens, and keys
3. **Use descriptive names** - Include region/environment in the name (e.g., `CfCredentials-EU10-Prod`)
4. **Keep inputs minimal** - Store only reusable values; command-specific values go in `inputValues` at trigger time

---

# Utility Patterns

## Export Command to File

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
OUTPUT_FILE="MyCommand.command.json"

curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/commands/$COMMAND_ID" > "$OUTPUT_FILE"

echo "Command exported to $OUTPUT_FILE"
```

## Upload All Commands from Directory

```bash
DIRECTORY="./commands"

for file in "$DIRECTORY"/*.command.json; do
  echo "Uploading: $file"
  curl -s -X POST \
    -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
    -H "Content-Type: application/json" \
    -d @"$file" \
    "https://$AUTOPI_HOSTNAME/api/v1/commands"
done
```

## Check if Command Exists

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/commands/$COMMAND_ID")

if [[ "$STATUS" == "200" ]]; then
  echo "Command exists"
elif [[ "$STATUS" == "404" ]]; then
  echo "Command not found"
else
  echo "Error: HTTP $STATUS"
fi
```

## Create or Update Command (Upsert Pattern)

```bash
COMMAND_FILE="MyCommand.command.json"
COMMAND_ID="<catalog:name:version-from-file>"

# Check if exists
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/commands/$COMMAND_ID")

if [[ "$STATUS" == "200" ]]; then
  echo "Updating existing command..."
  curl -s -X PUT \
    -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
    -H "Content-Type: application/json" \
    -d @"$COMMAND_FILE" \
    "https://$AUTOPI_HOSTNAME/api/v1/commands/$COMMAND_ID"
else
  echo "Creating new command..."
  curl -s -X POST \
    -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
    -H "Content-Type: application/json" \
    -d @"$COMMAND_FILE" \
    "https://$AUTOPI_HOSTNAME/api/v1/commands"
fi
```

---

# Error Handling

The API returns standard HTTP status codes:

| Code | Description |
|------|-------------|
| `200` | Success (GET, PUT) |
| `201` | Created (POST) |
| `204` | No Content (DELETE) |
| `400` | Bad request - invalid JSON or missing required fields |
| `401` | Unauthorized - invalid credentials |
| `403` | Forbidden - insufficient permissions |
| `404` | Not found - resource doesn't exist |
| `409` | Conflict - resource already exists or invalid state |
| `429` | Too many requests - rate limited |
| `500` | Internal server error |
| `503` | Service unavailable |

Always check the response for error messages:

```bash
RESPONSE=$(curl -s -w "\n%{http_code}" -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/commands")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [[ "$HTTP_CODE" != "200" && "$HTTP_CODE" != "201" ]]; then
  echo "Error ($HTTP_CODE): $BODY"
  exit 1
fi

echo "$BODY"
```

---

# Required Permissions

| Operation | Required Permission |
|-----------|-------------------|
| List/Get resources | `Read` |
| Create/Update/Delete resources | `Write` |

---

# Mandatory Description Patterns

⚠️ **When creating or updating commands/inputs, use these exact description patterns for standard parameters.**

See the complete authoritative list in **[`../automation-pilot-command-generation/references/description-patterns.md`](../automation-pilot-command-generation/references/description-patterns.md)**.

---

# Examples

## Example 1: Deploy a command from a JSON file

User: "Deploy this command to Automation Pilot"

1. Check if the command already exists by reading its ID from the file and doing a GET
2. If it exists: PUT to `/api/v1/commands/$COMMAND_ID`; if not: POST to `/api/v1/commands`
3. Report success with the command ID from the response

## Example 2: Export all commands from a catalog

User: "Export all commands from my catalog to files"

1. GET `/api/v1/commands?catalog=$AUTOPI_DEFAULT_CATALOG` to list all command IDs
2. For each ID, GET `/api/v1/commands/$COMMAND_ID` and write the response to `<CommandName>.command.json`
3. Report how many files were exported and where

## Example 3: Rotate credentials in an input

User: "Update the password in my JiraCredentials input"

1. PATCH `/api/v1/inputs/mycatalog-xxx:JiraCredentials:1` with `Content-Type: application/merge-patch+json`
2. Send only `{"values": {"password": "<new-value>"}}` — do not replace the entire input
3. Confirm the update succeeded (HTTP 200)

---

# Troubleshooting

**Error:** HTTP 401 Unauthorized
**Cause:** Invalid credentials.
**Solution:** Verify `AUTOPI_USERNAME` and `AUTOPI_PASSWORD` are correct for the target hostname.

**Error:** HTTP 403 Forbidden
**Cause:** The user lacks the required permission for the operation.
**Solution:** Check the Required Permissions table — everything needs `Write`.

**Error:** HTTP 409 Conflict on command or input create
**Cause:** A resource with the same ID already exists.
**Solution:** Use the Upsert Pattern — GET first to check existence, then PUT if found or POST if not.

**Error:** HTTP 400 Bad Request on command create
**Cause:** Invalid JSON or a missing required field in the command definition.
**Solution:** Check the response body for the specific field that failed.

**Error:** HTTP 404 on command update
**Cause:** The command ID in the request URL doesn't match any existing command.
**Solution:** Verify the ID format is `catalog:name:version` and that the catalog and command exist. Use GET first to confirm.

---

⚠️ **CAUTION: DO NOT RELEASE COMMANDS & INPUTS IF NOT EXPLICITLY REQUESTED BY THE USER**
