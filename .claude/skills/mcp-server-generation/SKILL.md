---
name: create-sap-automation-pilot-mcp-server
description: This skill should be used when the user asks to "create MCP server", "build MCP server definition", "generate MCP tools", "create autopi MCP config", "add MCP tool", "configure MCP server for automation pilot", "define MCP server", "create tool definition", or discusses SAP Automation Pilot MCP server configuration and tool definitions.
version: 1.0.0
---

# SAP Automation Pilot MCP Server Definition Generator

This skill guides the creation of MCP server definitions for SAP Automation Pilot. MCP server definitions are JSON files that expose Automation Pilot commands as MCP tools, allowing AI assistants to invoke them directly.

Reference examples are available in `autopi-mcp-servers/` at the project root.

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

The `autopi-mcp-servers/` directory at the project root contains 7 production examples:

| File | Tools | Complexity | Good Example Of |
|------|-------|-----------|-----------------|
| `ans-event-producer.json` | 1 | Simple | Minimal single-tool server |
| `btp-resource-discovery.json` | 5 | Simple | All read-only tools, single credential |
| `incident-management.json` | 3 | Medium | Mixed catalogs, disabled tool |
| `application-logs-and-metrics.json` | 4 | Medium | Multiple credential types in one server |
| `cloud-landscape-directory.json` | 6 | Medium | Rich instructions with markdown |
| `cloud-transport-management.json` | 14 | Complex | Full CRUD + destructive ops, openWorldHint variations |
| `hana-cloud-lifecycle-management.json` | 14 | Complex | Lifecycle management, idempotent operations, destructive operations |

Always read and reference these examples when generating new definitions. Use them to validate that your output follows the established patterns.