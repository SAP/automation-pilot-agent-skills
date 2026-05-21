---
name: manage-sap-automation-pilot-commands-catalogs-and-inputs-via-api
description: This skill should be used when the user asks to "list autopi catalogs", "import command","manage commands","upload command to autopi", "deploy command", "create input", "trigger webhook", "list commands", "list inputs", "manage automation pilot content", "export command", "import command", "list MCP servers", "create MCP server", "deploy MCP server", "update MCP server", "delete MCP server", "export MCP server", "manage MCP servers", or discusses SAP Automation Pilot Content API operations for catalogs, commands, inputs, webhooks, or MCP servers.
version: 1.2.0
---

# SAP Automation Pilot Content API Management

This skill manages catalogs, commands, inputs, webhooks, and MCP servers in SAP Automation Pilot using the Content API.

## Command Release Policy

Commands deploy in DRAFT state by default. Do not release automatically — only release when:
1. The command has been tested and verified working
2. The user explicitly requests release

Draft state allows safe testing without affecting production.

## Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Commands | PascalCase | `RestartCfApp`, `GetHanaInstance` |
| Inputs | PascalCase | `BtpCredentials`, `JiraConfig` |
| Catalogs | kebab-case | `my-automations-xxx` |
| MCP Servers | kebab-case | `cf-app-management` |

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

2. Ensure `curl` and `jq` are available in your environment.

---

# Executable Scripts

Reusable shell scripts are available in `scripts/` directory. These scripts handle config loading, error handling, and provide a CLI interface.

## Quick Reference

| Script | Description |
|--------|-------------|
| `autopi-catalogs.sh` | List, get, create, delete catalogs |
| `autopi-commands.sh` | List, get, deploy, delete, release commands |
| `autopi-inputs.sh` | List, get, create, update, delete inputs |
| `autopi-webhooks.sh` | List, get, create, trigger webhooks |
| `autopi-mcp-servers.sh` | List, get, create, update, delete MCP servers |
| `autopi-deploy.sh` | Smart deploy (create or update) a command |
| `autopi-deploy-mcp-server.sh` | Smart deploy (create or update) an MCP server |
| `autopi-export.sh` | Export commands/inputs/MCP servers to files |
| `autopi-sync.sh` | Batch sync all commands from a directory |

## Usage Examples

```bash
# List catalogs
./scripts/autopi-catalogs.sh list --own

# List commands in a catalog
./scripts/autopi-commands.sh list my-catalog-xxx

# Deploy a command (creates if new, updates if exists)
./scripts/autopi-deploy.sh MyCommand.command.json

# Export a command to file
./scripts/autopi-export.sh command my-catalog:MyCmd:1 output.json

# Export all commands from a catalog
./scripts/autopi-export.sh catalog my-catalog-xxx ./exported/

# Sync all commands from a directory
./scripts/autopi-sync.sh ./commands/

# Trigger a webhook
./scripts/autopi-webhooks.sh trigger my-webhook-id '{"message":"hello"}'

# List MCP servers
./scripts/autopi-mcp-servers.sh list

# Deploy an MCP server (creates if new, updates if exists)
./scripts/autopi-deploy-mcp-server.sh my-server.json

# Export all MCP servers to a directory
./scripts/autopi-export.sh mcp-servers ./exported-servers/
```

## Script Details

### autopi-commands.sh

```bash
autopi-commands.sh list [catalog]              # List commands
autopi-commands.sh ids [catalog]               # List command IDs only
autopi-commands.sh get <command-id>            # Get command details
autopi-commands.sh deploy <file>               # Deploy command (upsert)
autopi-commands.sh delete <command-id>         # Delete command
autopi-commands.sh release <command-id>        # Release draft
autopi-commands.sh deprecate <command-id>      # Mark deprecated
autopi-commands.sh restore <command-id>        # Restore deprecated
```

### autopi-inputs.sh

```bash
autopi-inputs.sh list [catalog]                # List inputs
autopi-inputs.sh get <input-id>                # Get input details
autopi-inputs.sh create <file>                 # Create from file
autopi-inputs.sh update <id> <file>            # Full update
autopi-inputs.sh patch <id> '<json>'           # Partial update
autopi-inputs.sh delete <input-id>             # Delete input
```

### autopi-deploy.sh (Smart Deploy)

Validates JSON, checks if command exists, creates or updates accordingly, and reports any issues.

```bash
./scripts/autopi-deploy.sh MyCommand.command.json
# Output:
# [INFO] Deploying: MyCommand
# [INFO]   ID: my-catalog:MyCommand:1
# [INFO]   Catalog: my-catalog
# [INFO] Command exists, updating...
# [OK] Deployed: my-catalog:MyCommand:1
```

### autopi-mcp-servers.sh

```bash
autopi-mcp-servers.sh list                            # List all MCP servers
autopi-mcp-servers.sh get <mcp-server-id>             # Get MCP server details
autopi-mcp-servers.sh create <file.json>              # Create MCP server from file
autopi-mcp-servers.sh update <mcp-server-id> <file>   # Update MCP server (auto-fetches ETag)
autopi-mcp-servers.sh delete <mcp-server-id>          # Delete MCP server (auto-fetches ETag)
```

### autopi-deploy-mcp-server.sh (Smart Deploy)

Validates JSON, checks if MCP server exists, creates or updates accordingly. Handles ETag automatically for updates.

```bash
./scripts/autopi-deploy-mcp-server.sh my-server.json
# Output:
# [INFO] Deploying MCP server: BTP Resource Discovery
# [INFO]   Tools: 5
# [INFO]   Enabled: true
# [INFO] Creating new MCP server...
# [OK] Deployed: BTP Resource Discovery (5 tools)
```

---

## Configuration Loading

All operations require the following environment variables to be set:

```bash
# Check required environment variables
if [[ -z "$AUTOPI_HOSTNAME" || -z "$AUTOPI_USERNAME" || -z "$AUTOPI_PASSWORD" || -z "$AUTOPI_DEFAULT_CATALOG" ]]; then
  echo "Error: Missing required environment variables"
  echo "Please set: AUTOPI_HOSTNAME, AUTOPI_USERNAME, AUTOPI_PASSWORD, AUTOPI_DEFAULT_CATALOG"
  exit 1
fi

HOST="$AUTOPI_HOSTNAME"
USER="$AUTOPI_USERNAME"
PASS="$AUTOPI_PASSWORD"
DEFAULT_CATALOG="$AUTOPI_DEFAULT_CATALOG"
```

---

# Catalogs

## List Catalogs

```bash
# List all catalogs owned by the tenant
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/catalogs?own=true" | jq .

# List catalogs provided by SAP
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/catalogs?provided=true" | jq .

# List all catalogs (owned + provided)
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/catalogs" | jq .

# List catalogs filtered by tag
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/catalogs?tag=environment:production" | jq .
```

## Get Catalog by ID

```bash
CATALOG_ID="mycatalog-xxx"
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/catalogs/$CATALOG_ID" | jq .
```

## Create Catalog

```bash
curl -s -X POST \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-catalog",
    "description": "My custom catalog for automation commands",
    "tags": {
      "environment": "development"
    }
  }' \
  "https://$HOST/api/v1/catalogs" | jq .
```

## Update Catalog

```bash
CATALOG_ID="mycatalog-xxx"
curl -s -X PUT \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated description",
    "tags": {
      "environment": "production"
    }
  }' \
  "https://$HOST/api/v1/catalogs/$CATALOG_ID" | jq .
```

## Delete Catalog

Note: Catalog must be empty (no commands or inputs) before deletion.

```bash
CATALOG_ID="mycatalog-xxx"
curl -s -X DELETE -u "$USER:$PASS" "https://$HOST/api/v1/catalogs/$CATALOG_ID"
```

---

# Commands

## List Commands

```bash
# List all commands in a catalog
CATALOG="mycatalog-xxx"
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/commands?catalog=$CATALOG" | jq .

# List commands with design-time issues
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/commands?catalog=$CATALOG&includeIssues=true" | jq .

# List commands by tag
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/commands?tag=type:monitoring" | jq .

# List only command IDs (lightweight)
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/commands/ids?catalog=$CATALOG" | jq .
```

## Get Command by ID

Command ID format: `catalog:name:version`

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/commands/$COMMAND_ID" | jq .
```

## Create/Upload Command

Upload a command from a `.command.json` file:

```bash
COMMAND_FILE="path/to/MyCommand.command.json"

# Validate JSON first
if ! jq empty "$COMMAND_FILE" 2>/dev/null; then
  echo "Error: Invalid JSON in $COMMAND_FILE"
  exit 1
fi

# Create the command
curl -s -X POST \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d @"$COMMAND_FILE" \
  "https://$HOST/api/v1/commands" | jq .
```

Create command inline:

```bash
curl -s -X POST \
  -u "$USER:$PASS" \
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
  "https://$HOST/api/v1/commands" | jq .
```

## Update Command

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
COMMAND_FILE="path/to/MyCommand.command.json"

curl -s -X PUT \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d @"$COMMAND_FILE" \
  "https://$HOST/api/v1/commands/$COMMAND_ID" | jq .
```

## Delete Command

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
curl -s -X DELETE -u "$USER:$PASS" "https://$HOST/api/v1/commands/$COMMAND_ID"
```

## Release Command

!!! USE ONLY IF THE USER EXPLICITLY REQUESTED IT 

Release a draft command:

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
curl -s -X PUT -u "$USER:$PASS" "https://$HOST/api/v1/commands/$COMMAND_ID/release" | jq .
```

## Deprecate Command

Mark a command as deprecated:

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
curl -s -X PUT -u "$USER:$PASS" "https://$HOST/api/v1/commands/$COMMAND_ID/deprecate" | jq .
```

## Restore Command

Restore a deprecated command:

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
curl -s -X PUT -u "$USER:$PASS" "https://$HOST/api/v1/commands/$COMMAND_ID/restore" | jq .
```

## Bulk Create Commands

Upload multiple commands at once:

```bash
curl -s -X POST \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d '{
    "commands": [
      {"name": "Cmd1", "catalog": "mycatalog-xxx", ...},
      {"name": "Cmd2", "catalog": "mycatalog-xxx", ...}
    ]
  }' \
  "https://$HOST/api/v1/bulk/commands" | jq .
```

---

# Inputs

## List Inputs

```bash
# List inputs in a catalog
CATALOG="mycatalog-xxx"
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/inputs?catalog=$CATALOG" | jq .

# List inputs by tag
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/inputs?tag=type:credentials" | jq .
```

## Get Input by ID

Input ID format: `catalog:name:version`

```bash
INPUT_ID="mycatalog-xxx:MyInput:1"
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/inputs/$INPUT_ID" | jq .
```

## Create Input

```bash
curl -s -X POST \
  -u "$USER:$PASS" \
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
      "username": "admin",
      "password": "secret123"
    },
    "tags": {
      "type": "credentials"
    }
  }' \
  "https://$HOST/api/v1/inputs" | jq .
```

## Update Input (Full)

```bash
INPUT_ID="mycatalog-xxx:MyInput:1"
curl -s -X PUT \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MyCredentials",
    "catalog": "mycatalog-xxx",
    "description": "Updated credentials",
    "version": 1,
    "keys": {...},
    "values": {...}
  }' \
  "https://$HOST/api/v1/inputs/$INPUT_ID" | jq .
```

## Update Input (Partial - Values Only)

Update only specific values without replacing the entire input:

```bash
INPUT_ID="mycatalog-xxx:MyInput:1"
curl -s -X PATCH \
  -u "$USER:$PASS" \
  -H "Content-Type: application/merge-patch+json" \
  -d '{
    "values": {
      "password": "newSecret456"
    }
  }' \
  "https://$HOST/api/v1/inputs/$INPUT_ID" | jq .
```

## Delete Input

```bash
INPUT_ID="mycatalog-xxx:MyInput:1"
curl -s -X DELETE -u "$USER:$PASS" "https://$HOST/api/v1/inputs/$INPUT_ID"
```

## Release Input

!!! USE ONLY IF THE USER EXPLICITLY REQUESTED IT 

```bash
INPUT_ID="mycatalog-xxx:MyInput:1"
curl -s -X PUT -u "$USER:$PASS" "https://$HOST/api/v1/inputs/$INPUT_ID/release" | jq .
```

## Deprecate Input

```bash
INPUT_ID="mycatalog-xxx:MyInput:1"
curl -s -X PUT -u "$USER:$PASS" "https://$HOST/api/v1/inputs/$INPUT_ID/deprecate" | jq .
```

## Restore Input

```bash
INPUT_ID="mycatalog-xxx:MyInput:1"
curl -s -X PUT -u "$USER:$PASS" "https://$HOST/api/v1/inputs/$INPUT_ID/restore" | jq .
```

## Bulk Create Inputs

```bash
curl -s -X POST \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d '{
    "inputs": [
      {"name": "Input1", "catalog": "mycatalog-xxx", ...},
      {"name": "Input2", "catalog": "mycatalog-xxx", ...}
    ]
  }' \
  "https://$HOST/api/v1/bulk/inputs" | jq .
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
    "password": "secret-value"
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
    "serviceKey": {
      "url": "https://hana-instance.hana.cloud.sap",
      "user": "DBADMIN",
      "password": "secret",
      "certificate": "..."
    }
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
    "kubeconfig": {
      "apiVersion": "v1",
      "kind": "Config",
      "clusters": [...],
      "users": [...],
      "contexts": [...]
    }
  }
}
```

## Using Inputs in Executions

When triggering a command execution, reference an input:

```bash
curl -s -X POST \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d '{
    "commandId": "applm-sapcp:RestartCfApp:1",
    "input": "mycatalog-xxx:CfCredentials-EU10:1",
    "additionalValues": {
      "appName": "my-application",
      "subAccount": "my-org"
    }
  }' \
  "https://$HOST/api/v1/executions" | jq .
```

- **input**: References a stored input by ID
- **additionalValues**: Override or add values not in the input

## Best Practices

1. **Separate inputs by environment** - Create distinct inputs for dev/staging/prod
2. **Mark sensitive values** - Always set `"sensitive": true` for passwords, tokens, and keys
3. **Use descriptive names** - Include region/environment in the name (e.g., `CfCredentials-EU10-Prod`)
4. **Keep inputs minimal** - Store only reusable values; command-specific values go in `additionalValues`

---

# Webhooks

## List Webhooks

```bash
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/webhooks" | jq .
```

## Get Webhook by ID

```bash
WEBHOOK_ID="my-webhook-id"
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/webhooks/$WEBHOOK_ID" | jq .
```

## Create Webhook

```bash
curl -s -X POST \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MyWebhook",
    "commandId": "mycatalog-xxx:MyCommand:1",
    "description": "Webhook to trigger MyCommand",
    "inputMapping": {
      "message": "$.event.message"
    }
  }' \
  "https://$HOST/api/v1/webhooks" | jq .
```

## Update Webhook

```bash
WEBHOOK_ID="my-webhook-id"
curl -s -X PUT \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MyWebhook",
    "commandId": "mycatalog-xxx:MyCommand:1",
    "description": "Updated webhook description",
    "inputMapping": {...}
  }' \
  "https://$HOST/api/v1/webhooks/$WEBHOOK_ID" | jq .
```

## Delete Webhook

```bash
WEBHOOK_ID="my-webhook-id"
curl -s -X DELETE -u "$USER:$PASS" "https://$HOST/api/v1/webhooks/$WEBHOOK_ID"
```

## Trigger/Execute Webhook

Execute a webhook to start a command execution:

```bash
WEBHOOK_ID="my-webhook-id"

# Trigger with event data
curl -s -X POST \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d '{
    "event": {
      "message": "Hello from webhook",
      "timestamp": "2024-01-15T10:30:00Z"
    }
  }' \
  "https://$HOST/api/v1/webhooks/$WEBHOOK_ID/trigger" | jq .

# Trigger without event data
curl -s -X POST -u "$USER:$PASS" "https://$HOST/api/v1/webhooks/$WEBHOOK_ID/trigger" | jq .
```

---

# MCP Servers

MCP servers expose Automation Pilot commands as MCP tools for AI assistants. The API is secured by Basic Authentication and requires the `GenAI` permission for write operations. Update and delete operations use ETag-based optimistic concurrency via the `If-Match` header.

## List MCP Servers

```bash
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/mcp-servers" | jq .
```

## Get MCP Server by ID

The MCP server ID is its name (e.g., `"BTP Resource Discovery"`).

```bash
MCP_SERVER_ID="BTP Resource Discovery"
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/mcp-servers/$MCP_SERVER_ID" | jq .
```

The response includes an `ETag` header needed for update/delete operations.

## Create MCP Server

```bash
MCP_SERVER_FILE="path/to/my-server.json"

# Validate JSON first
if ! jq empty "$MCP_SERVER_FILE" 2>/dev/null; then
  echo "Error: Invalid JSON in $MCP_SERVER_FILE"
  exit 1
fi

curl -s -X POST \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -d @"$MCP_SERVER_FILE" \
  "https://$HOST/api/v1/mcp-servers" | jq .
```

## Update MCP Server

Update requires the `If-Match` header with the current ETag value:

```bash
MCP_SERVER_ID="BTP Resource Discovery"
MCP_SERVER_FILE="path/to/updated-server.json"

# Get current ETag
ETAG=$(curl -s -I -u "$USER:$PASS" \
  "https://$HOST/api/v1/mcp-servers/$MCP_SERVER_ID" | \
  grep -i "^etag:" | awk '{print $2}' | tr -d '\r\n')

# Update with If-Match
curl -s -X PUT \
  -u "$USER:$PASS" \
  -H "Content-Type: application/json" \
  -H "If-Match: $ETAG" \
  -d @"$MCP_SERVER_FILE" \
  "https://$HOST/api/v1/mcp-servers/$MCP_SERVER_ID" | jq .
```

If the ETag does not match (HTTP 412), the server was modified since you last read it. Fetch the latest version and retry.

## Delete MCP Server

Delete also requires the `If-Match` header:

```bash
MCP_SERVER_ID="My Old Server"

# Get current ETag
ETAG=$(curl -s -I -u "$USER:$PASS" \
  "https://$HOST/api/v1/mcp-servers/$MCP_SERVER_ID" | \
  grep -i "^etag:" | awk '{print $2}' | tr -d '\r\n')

curl -s -X DELETE \
  -u "$USER:$PASS" \
  -H "If-Match: $ETAG" \
  "https://$HOST/api/v1/mcp-servers/$MCP_SERVER_ID"
```

## Deploy MCP Server (Upsert Pattern)

```bash
MCP_SERVER_FILE="my-server.json"
MCP_SERVER_ID=$(jq -r '.name' "$MCP_SERVER_FILE")

# Check if exists
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -u "$USER:$PASS" \
  "https://$HOST/api/v1/mcp-servers/$MCP_SERVER_ID")

if [[ "$STATUS" == "200" ]]; then
  echo "Updating existing MCP server..."
  ETAG=$(curl -s -I -u "$USER:$PASS" \
    "https://$HOST/api/v1/mcp-servers/$MCP_SERVER_ID" | \
    grep -i "^etag:" | awk '{print $2}' | tr -d '\r\n')
  curl -s -X PUT \
    -u "$USER:$PASS" \
    -H "Content-Type: application/json" \
    -H "If-Match: $ETAG" \
    -d @"$MCP_SERVER_FILE" \
    "https://$HOST/api/v1/mcp-servers/$MCP_SERVER_ID" | jq .
else
  echo "Creating new MCP server..."
  curl -s -X POST \
    -u "$USER:$PASS" \
    -H "Content-Type: application/json" \
    -d @"$MCP_SERVER_FILE" \
    "https://$HOST/api/v1/mcp-servers" | jq .
fi
```

---

# Utility Patterns

## Export Command to File

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
OUTPUT_FILE="MyCommand.command.json"

curl -s -u "$USER:$PASS" \
  "https://$HOST/api/v1/commands/$COMMAND_ID" | jq . > "$OUTPUT_FILE"

echo "Command exported to $OUTPUT_FILE"
```

## Upload All Commands from Directory

```bash
DIRECTORY="./commands"

for file in "$DIRECTORY"/*.command.json; do
  echo "Uploading: $file"
  RESPONSE=$(curl -s -X POST \
    -u "$USER:$PASS" \
    -H "Content-Type: application/json" \
    -d @"$file" \
    "https://$HOST/api/v1/commands")
  echo "$RESPONSE" | jq -r '.id // .message'
done
```

## Check if Command Exists

```bash
COMMAND_ID="mycatalog-xxx:MyCommand:1"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -u "$USER:$PASS" \
  "https://$HOST/api/v1/commands/$COMMAND_ID")

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
COMMAND_ID=$(jq -r '.id' "$COMMAND_FILE")

# Check if exists
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -u "$USER:$PASS" \
  "https://$HOST/api/v1/commands/$COMMAND_ID")

if [[ "$STATUS" == "200" ]]; then
  echo "Updating existing command..."
  curl -s -X PUT \
    -u "$USER:$PASS" \
    -H "Content-Type: application/json" \
    -d @"$COMMAND_FILE" \
    "https://$HOST/api/v1/commands/$COMMAND_ID" | jq .
else
  echo "Creating new command..."
  curl -s -X POST \
    -u "$USER:$PASS" \
    -H "Content-Type: application/json" \
    -d @"$COMMAND_FILE" \
    "https://$HOST/api/v1/commands" | jq .
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
RESPONSE=$(curl -s -w "\n%{http_code}" -u "$USER:$PASS" "https://$HOST/api/v1/commands")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [[ "$HTTP_CODE" != "200" && "$HTTP_CODE" != "201" ]]; then
  echo "Error ($HTTP_CODE): $(echo "$BODY" | jq -r '.message // .')"
  exit 1
fi

echo "$BODY" | jq .
```

---

# Required Permissions

| Operation | Required Permission |
|-----------|-------------------|
| List/Get resources | `Read` |
| Create/Update/Delete resources | `Write` |
| Trigger webhooks | `Execute` |
| Create/Update/Delete MCP servers | `GenAI` |

---

# Mandatory Description Patterns

⚠️ **When creating or updating commands/inputs, use these exact description patterns for standard parameters.**

## 📥 Input Parameters

**Authentication:**
- **password** - The password for the specified technical user or the client secret for the specified OAuth 2.0 client ID to be used for authentication. Related input keys: 'user' or 'tokenUrl'
- **user** - The name of a technical user or an OAuth 2.0 client ID to be used for authentication. Related input keys: 'password' or 'tokenUrl'
- **refreshToken** - An OAuth 2.0 refresh token to be used for authentication. If 'refreshToken' is passed, 'user' and 'password' will be ignored. Related input keys: 'clientId', 'clientSecret', 'tokenUrl'

**Region & Organization:**
- **region** - The technical name of the Cloud Foundry region. Example: cf-eu10, cf-eu10-002
- **subAccount** - The name or the ID of the Cloud Foundry organization. Examples: my-org-name-1, 0ffeb410-5f78-0000-af5c-5b26baf46623

**Resources & Services:**
- **resourceName** - The technical name of the [resource type]. Example: [examples]
- **serviceKey** - A service key for [service]
- **name** - The name of the [object]

## 📤 Output Parameters

**Status & Response:**
- **status** - The status code of the response. Examples: 200, 301, 404, 503
- **responseCode** - The HTTP response code
- **state** - The state of the [object/entity]. Examples: [examples]

**Data Collections:**
- **output** - The original response from [service/API]
- **result** - The result of the [operation]

**System Operations:**
- **exitCode** - The exit code returned by the script execution
- **instanceId** - The ID of the [instance/object]

---

!!! CAUTION: DO NOT RELEASE COMMANDS & INPUTS IF NOT EXPLICITLY REQUESTED BY THE USER
