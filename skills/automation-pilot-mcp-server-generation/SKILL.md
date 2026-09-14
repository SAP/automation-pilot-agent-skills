---
name: automation-pilot-mcp-server-generation
description: Create, configure, and deploy SAP Automation Pilot MCP server definitions and tool configurations. Use when building MCP servers, defining tools, setting up Automation Pilot MCP configs, or deploying MCP server definitions via API.
---

# SAP Automation Pilot MCP Server Definition Generator

This skill guides the creation of MCP server definitions for SAP Automation Pilot. MCP server definitions are JSON files that expose Automation Pilot commands as MCP tools, allowing AI assistants to invoke them directly.

---

## JSON Schema

### Top-Level Structure

Every MCP server definition follows this structure:

```json
{
  "name": "kebab-case-server-name",
  "enabled": true,
  "instructions": "<Natural-language description for the AI>",
  "mcpTools": [ ...tool objects... ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Kebab-case identifier (e.g., `cf-app-management`). Must match filename without `.json`. No spaces. |
| `enabled` | boolean | Activates/deactivates the entire server |
| `instructions` | string | Natural-language guidance for the AI on what this server does and when to use it |
| `mcpTools` | array | Array of tool definitions (1 or more) |

### Tool Object Structure

Each entry in `mcpTools` uses exactly this schema — all fields are required:

```json
{
  "commandId": "<catalog>-<suffix>:<CommandName>:<version>",
  "name": "<snake_case_tool_name>",
  "enabled": true,
  "inputReferences": [
    "<catalog>-<<<TENANT_ID>>>:<InputName>:<version>"
  ],
  "tags": {},
  "title": "<Title Case Label>",
  "destructiveHint": false,
  "idempotentHint": false,
  "openWorldHint": true,
  "readOnlyHint": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| `commandId` | string | Reference to an Automation Pilot command: `<catalog>-<suffix>:<CommandName>:<version>` |
| `name` | string | Snake_case identifier exposed to the MCP protocol (e.g., `list_orgs`, `create_snapshot`) |
| `enabled` | boolean | Enable/disable individual tool (set `false` to include but not activate) |
| `inputReferences` | string[] | Array of credential/input references injected at runtime. Use `[]` if credentials are embedded in the command. |
| `tags` | object | Always `{}` (reserved for future use) |
| `title` | string | Title Case human-readable label (e.g., `"List Transport Nodes"`) |
| `destructiveHint` | boolean | Signals the tool deletes or permanently alters state |
| `idempotentHint` | boolean | Signals the tool is safe to call repeatedly with the same result |
| `openWorldHint` | boolean | Signals the tool discovers across a landscape vs. targeting a known resource |
| `readOnlyHint` | boolean | Signals the tool only reads data, never modifies state |

---

## Command ID Format

Command IDs follow the pattern: `<catalog>-<suffix>:<CommandName>:<version>`

### Suffix Rules

| Suffix | Usage | Example |
|--------|-------|---------|
| `-sapcp` | SAP-provided (shared/public) commands | `cf-sapcp:ListCfOrgs:1` |
| `-<<<TENANT_ID>>>` | Tenant-specific (custom) commands | `hanalm-<<<TENANT_ID>>>:GetHanaCloudInstance:1` |

The `<<<TENANT_ID>>>` is a placeholder substituted at deployment time with the actual tenant identifier.

### Known Catalog Prefixes

| Prefix | Domain |
|--------|--------|
| `ans` | Alert Notification Service |
| `applm` | Application Lifecycle Management |
| `calmhm` | Cloud ALM Health Monitoring |
| `capops` | Custom operations (tenant-specific) |
| `cf` | Cloud Foundry |
| `cld` | Cloud Landscape Directory |
| `ctms` | Cloud Transport Management |
| `hanalm` | HANA Cloud Lifecycle Management |
| `jira` | Jira Integration |
| `monitoring` | Monitoring |
| `welcome` | Welcome/onboarding (credential inputs) |

---

## Input References (Credential Injection)

Input references inject pre-configured credentials into the command at runtime. Format: `<catalog>-<<<TENANT_ID>>>:<InputName>:<version>`

### Known Credential Types

| Input Reference | Domain | Used By |
|----------------|--------|---------|
| `capops-<<<TENANT_ID>>>:BtpCredentials:1` | BTP / Cloud Foundry | BTP Resource Discovery, App Logs & Metrics |
| `capops-<<<TENANT_ID>>>:AnsCredentials:1` | Alert Notification Service | ANS Event Producer, App Logs & Metrics |
| `capops-<<<TENANT_ID>>>:JiraDestination:1` | Jira | Incident Management |
| `welcome-<<<TENANT_ID>>>:CldDefaultValues:1` | Cloud Landscape Directory | Cloud Landscape Directory |
| `welcome-<<<TENANT_ID>>>:CtmsCredentials:1` | Cloud Transport Management | Cloud Transport Management |
| `welcome-<<<TENANT_ID>>>:ServiceManager:1` | SAP Service Manager | HANA Cloud Lifecycle Management |

### Rules

- Most tools have exactly **one** input reference for their credential type
- Use **empty array `[]`** when credentials are embedded directly in the command definition (e.g., custom commands with hardcoded config)
- A single server can mix credential types if tools span different services (e.g., App Logs & Metrics uses both `AnsCredentials` and `BtpCredentials`)
- The same credential reference can be reused across multiple tools in a server

---

## Hint Boolean Rules

The four hint booleans form a semantic contract. Follow these rules strictly:

### `readOnlyHint`

- **`true`** for all `list_*`, `get_*`, `fetch_*` operations that only read data
- **`false`** for all write, create, update, delete, start, stop operations
- If `readOnlyHint: true` then `destructiveHint` MUST be `false`

### `destructiveHint`

- **`true`** ONLY for operations that delete or permanently alter state:
  - Delete operations (`delete_*`, `remove_*`)
  - Irreversible state changes (`revert_to_snapshot`, `upgrade_instance`)
- **`false`** for everything else (reads, creates, updates, start/stop)
- If `destructiveHint: true` then `readOnlyHint` MUST be `false`

### `idempotentHint`

- **`true`** ONLY for write operations that are safe to repeat:
  - Idempotent updates (`update_transport_node`, `update_transport_route`)
  - Idempotent state transitions (`start_instance`, `stop_instance`, `restart_instance`, `enable_capabilities`)
- **`false`** for read operations (idempotent by nature but not tagged)
- **`false`** for creates, deletes, and non-idempotent mutations

### `openWorldHint`

- **`true`** for operations that discover or act across a landscape/collection (most tools)
- **`false`** for operations on a specific, known resource (e.g., get/create/update/delete a single node by ID)

### Quick Reference Matrix

| Operation Type | readOnly | destructive | idempotent | openWorld |
|---------------|----------|-------------|------------|-----------|
| List/discovery | `true` | `false` | `false` | `true` |
| Get single resource | `true` | `false` | `false` | `true` or `false` |
| Create | `false` | `false` | `false` | `false` |
| Update (idempotent) | `false` | `false` | `true` | `false` |
| Update (non-idempotent) | `false` | `false` | `false` | `false` or `true` |
| Delete | `false` | `true` | `false` | `false` |
| Start/Stop/Restart | `false` | `false` | `true` | `true` |
| Send/Trigger event | `false` | `false` | `false` | `true` |

---

## Tool Naming Conventions

- Use **snake_case** for all tool names
- Start with a **verb prefix** that matches the operation:

| Prefix | Usage |
|--------|-------|
| `list_` | List/enumerate collections |
| `get_` | Retrieve a single resource |
| `fetch_` | Retrieve data from external sources |
| `create_` | Create a new resource |
| `update_` | Modify an existing resource |
| `delete_` | Remove a resource |
| `start_` | Start/activate a resource |
| `stop_` | Stop/deactivate a resource |
| `restart_` | Restart a resource |
| `enable_` | Enable a feature/capability |
| `send_` | Send a message/event |
| `add_` | Add to an existing resource |
| `import_` | Import resources |
| `forward_` | Forward/promote resources |
| `reset_` | Reset resource state |
| `remove_` | Remove from a collection |
| `revert_` | Revert to a previous state |
| `upgrade_` | Upgrade version |

- Keep names concise but descriptive (e.g., `list_orgs` not `list_cloud_foundry_organizations`)
- The `title` field should be the Title Case equivalent of the name (e.g., `list_orgs` → `"List Orgs"`)

---

## Instructions Field Guidelines

The `instructions` field tells the AI when and how to use this server.

### Simple servers (1-5 tools, single domain)

Use a single sentence:

```
"MCP server that can be used to send events through the Alert Notification service. Use it to notify your team about operational incidents."
```

### Complex servers (6+ tools, multiple capabilities)

Use markdown with bullet points organized by capability:

```
"MCP server for managing SAP HANA Cloud database instances through the SAP Service Manager API.\n\nUse this server to:\n- **Instance Lifecycle**: Start, stop, restart instances\n- **Snapshots**: Create, list, delete, and revert to storage snapshots\n- **Configuration**: Update instance size and enable capabilities\n- **Upgrades**: Upgrade instances to newer versions"
```

### Guidelines

- Start with what the server is ("MCP server for...")
- Mention the underlying service or API
- For complex servers, group capabilities with bold section headers
- Mention key parameters users typically need (e.g., "Most commands require an instanceId")
- Use `\n` for newlines within the JSON string value

---

## Validation Checklist

Before finalizing a generated MCP server definition, verify:

- [ ] `name` matches the intended filename (without `.json`)
- [ ] `enabled` is set appropriately (`true` for active, `false` for draft)
- [ ] `instructions` clearly describes when to use the server
- [ ] Every tool has all 10 required fields
- [ ] `commandId` follows the `<catalog>-<suffix>:<Name>:<version>` format
- [ ] `name` is snake_case and starts with an appropriate verb
- [ ] `title` is Title Case and matches the tool name semantically
- [ ] `inputReferences` uses the correct credential type for the domain (or `[]` if embedded)
- [ ] `tags` is `{}` (always empty)
- [ ] Hint booleans follow the semantic contract:
  - `readOnlyHint: true` → `destructiveHint: false`
  - `destructiveHint: true` → `readOnlyHint: false`
  - `idempotentHint: true` only on safe-to-repeat write operations
- [ ] No duplicate tool names within the server
- [ ] The output is valid JSON

---

## Reference Examples

### Minimal single-tool server (read-only)

```json
{
  "name": "ans-event-producer",
  "enabled": true,
  "instructions": "Use this server to send Alert Notification Service events. Call send_event when you need to publish an alert.",
  "mcpTools": [
    {
      "commandId": "ans-sapcp:SendEvent:1",
      "name": "send_event",
      "enabled": true,
      "inputReferences": ["ans-<<<TENANT_ID>>>:AnsCredentials:1"],
      "tags": {},
      "title": "Send ANS Event",
      "destructiveHint": false,
      "idempotentHint": false,
      "openWorldHint": true,
      "readOnlyHint": false
    }
  ]
}
```

### Multi-tool read-only server

```json
{
  "name": "btp-resource-discovery",
  "enabled": true,
  "instructions": "Use this server to discover BTP resources. All tools are read-only — they list or retrieve data and never modify state.",
  "mcpTools": [
    {
      "commandId": "cf-sapcp:ListCfOrgs:1",
      "name": "list_orgs",
      "enabled": true,
      "inputReferences": ["cf-<<<TENANT_ID>>>:CfCredentials:1"],
      "tags": {},
      "title": "List CF Organizations",
      "destructiveHint": false,
      "idempotentHint": true,
      "openWorldHint": true,
      "readOnlyHint": true
    },
    {
      "commandId": "cf-sapcp:ListCfSpaces:1",
      "name": "list_spaces",
      "enabled": true,
      "inputReferences": ["cf-<<<TENANT_ID>>>:CfCredentials:1"],
      "tags": {},
      "title": "List CF Spaces",
      "destructiveHint": false,
      "idempotentHint": true,
      "openWorldHint": true,
      "readOnlyHint": true
    },
    {
      "commandId": "cf-sapcp:ListCfApps:1",
      "name": "list_apps",
      "enabled": true,
      "inputReferences": ["cf-<<<TENANT_ID>>>:CfCredentials:1"],
      "tags": {},
      "title": "List CF Applications",
      "destructiveHint": false,
      "idempotentHint": true,
      "openWorldHint": true,
      "readOnlyHint": true
    }
  ]
}
```

---

## Examples

### Example 1: Create a simple read-only MCP server

User: "Create an MCP server for listing Cloud Foundry resources"

1. Define server with `name`, `enabled: true`, and clear `instructions`
2. Create tools for `list_orgs`, `list_spaces`, `list_apps`
3. Set `readOnlyHint: true`, `destructiveHint: false` for all tools
4. Use `cf-sapcp` catalog prefix for built-in CF commands
5. Add `BtpCredentials` input reference for authentication
6. Validate against the checklist

### Example 2: Create a full lifecycle MCP server

User: "Build an MCP server for managing HANA Cloud instances"

1. Create tools: `list_instances`, `get_instance`, `start_instance`, `stop_instance`, `restart_instance`, `delete_instance`
2. Apply correct hints: `readOnlyHint: true` for list/get, `idempotentHint: true` for start/stop/restart, `destructiveHint: true` for delete
3. Use `hanalm-<<<TENANT_ID>>>` catalog prefix
4. Write rich `instructions` with markdown grouping capabilities by category

### Example 3: Create a minimal single-tool event sender

User: "Create a minimal MCP server with one tool to send ANS alerts"

1. Create minimal server with single tool: `send_event`
2. Set `readOnlyHint: false`, `destructiveHint: false`, `openWorldHint: true`
3. Add `AnsCredentials` input reference
4. Write simple one-line `instructions`
5. Use the "Minimal single-tool server" inline example above as a structural guide

---

# Deploying MCP Servers via API

## Prerequisites

Set the following **required** environment variables:

```bash
export AUTOPI_HOSTNAME="emea.autopilot.cloud.sap"
export AUTOPI_USERNAME="your-username"
export AUTOPI_PASSWORD="your-password"
```

For the full list of supported hostnames (emea, aus, apac, amer, ksa), see `../automation-pilot-content-management-via-api/SKILL.md` → Prerequisites.

Ensure `curl` is available in your environment.

The API requires the `GenAI` permission for all write operations.

## List MCP Servers

```bash
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/mcp-servers"
```

## Get MCP Server by ID

MCP servers have a system-generated `id` field (e.g. `T000414R2-0000001779286543418-1-1`) returned in the create/list responses. This is distinct from the human-readable `name` field. Always use the `id` for GET, update, and delete operations.

```bash
# First, find the ID from the list response
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/mcp-servers" | jq '[.[] | {name, id}]'

# Then fetch by ID
MCP_SERVER_ID="T000414R2-0000001779286543418-1-1"
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/mcp-servers/$MCP_SERVER_ID"
```

## Create MCP Server

```bash
MCP_SERVER_FILE="path/to/my-server.json"
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d @"$MCP_SERVER_FILE" \
  "https://$AUTOPI_HOSTNAME/api/v1/mcp-servers"
```

## Update MCP Server

Use the system-generated `id` (not the `name`) in the URL.

```bash
MCP_SERVER_ID="<id-from-list-response>"
MCP_SERVER_FILE="path/to/updated-server.json"

curl -s -X PUT \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d @"$MCP_SERVER_FILE" \
  "https://$AUTOPI_HOSTNAME/api/v1/mcp-servers/$MCP_SERVER_ID"
```

## Delete MCP Server

```bash
MCP_SERVER_ID="<id-from-list-response>"

curl -s -X DELETE \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/mcp-servers/$MCP_SERVER_ID"
```

## Deploy MCP Server (Upsert Pattern)

Check for existence first — create if new, update if already deployed. Use the system-generated `id` returned from a previous create or list call:

```bash
MCP_SERVER_FILE="my-server.json"
MCP_SERVER_ID="<id-from-prior-create-or-list>"

STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/mcp-servers/$MCP_SERVER_ID")

if [[ "$STATUS" == "200" ]]; then
  echo "Updating existing MCP server..."
  curl -s -X PUT \
    -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
    -H "Content-Type: application/json" \
    -d @"$MCP_SERVER_FILE" \
    "https://$AUTOPI_HOSTNAME/api/v1/mcp-servers/$MCP_SERVER_ID"
else
  echo "Creating new MCP server..."
  curl -s -X POST \
    -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
    -H "Content-Type: application/json" \
    -d @"$MCP_SERVER_FILE" \
    "https://$AUTOPI_HOSTNAME/api/v1/mcp-servers"
fi
```

---

## Troubleshooting

**Error:** Duplicate tool names in server
**Cause:** Two tools have the same `name` field value.
**Solution:** Each tool name must be unique within a server. Use verb prefixes (`list_`, `get_`, `create_`) to differentiate tools that operate on the same resource type.

**Error:** Conflicting hint booleans
**Cause:** `readOnlyHint: true` and `destructiveHint: true` set on the same tool.
**Solution:** These are mutually exclusive — a read-only tool cannot be destructive. Refer to the Quick Reference Matrix to determine the correct combination for the operation type.

**Error:** Invalid `commandId` format
**Cause:** Command ID doesn't follow the `<catalog>-<suffix>:<Name>:<version>` pattern — missing separator, wrong suffix, or missing version.
**Solution:** Verify catalog prefix, suffix (`-sapcp` for SAP-provided, `-<<<TENANT_ID>>>` for custom), PascalCase command name, and integer version number.

**Error:** MCP server tool not activating
**Cause:** Individual tool `enabled` field set to `false`.
**Solution:** Set `enabled: true` on each tool that should be active. Use `false` only for tools intentionally included but not yet ready for use.