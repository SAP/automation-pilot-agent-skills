# MCP Server Model

The exact JSON schema and field rules for an Automation Pilot MCP server definition.

## Shape

```json
{
  "name": "btp-cf-app-lifecycle-ops",
  "enabled": true,
  "instructions": "<guidance the client reads>",
  "mcpTools": [
    {
      "commandId": "applm-sapcp:StopCfApp:1",
      "name": "stop_app",
      "enabled": true,
      "inputReferences": ["welcome-<<<TENANT_ID>>>:CloudFoundryCredentials:1"],
      "title": "Stop Cloud Foundry Application",
      "readOnlyHint": false,
      "destructiveHint": true,
      "idempotentHint": true,
      "sync": false,
      "waitTimeSeconds": null
    }
  ]
}
```

## Server fields

| Field | Notes |
|-------|-------|
| `id` | System-generated (`T002267R4-0000001786127368271-1-1`), returned on create/list. Address get/update/delete by `id`, not `name`. Omit when creating. |
| `name` | Kebab-case, no spaces. **Immutable** after creation. |
| `enabled` | Automation Pilot creates the server as enabled and live. `false` takes it offline without deleting it. |
| `instructions` | Tells the AI agent what the server exposes, when to use each tool, how the tools relate, and which safety or workflow constraints apply. Write it as an operating guide for the client. |
| `mcpTools` | Up to 50 tools. `autopi mcp-servers update` replaces the whole array, so preserve unchanged entries. |

## Tool fields

| Field | Notes |
|-------|-------|
| `commandId` | `<catalog>:<CommandName>:<version>`. The command's own description and input-key descriptions become the tool's schema shown to the client, so a well-described command makes a well-described tool. |
| `name` | snake_case, verb-prefixed, unique per server. **Immutable after creation.** To use a different name, add a replacement tool and remove the old one; clients bound to the old name will break. |
| `enabled` | `false` keeps the tool in the definition but hides it from the client. |
| `inputReferences` | Reusable Input IDs (`<catalog>:<InputName>:<version>`; use the selected catalog). Automation Pilot injects them at call time and removes the command keys they cover from the client-facing schema. Use `[]` if the command needs no injected values. |
| `title` | Title Case label matching `name`. |
| `readOnlyHint` / `destructiveHint` / `idempotentHint` | Set these advisory fields based on the command's behavior. A read-only operation can also be idempotent; destructive describes side effects. These are advisory hints, not an authorization or enforcement mechanism.|
| `sync` / `waitTimeSeconds` | `sync: true` plus a `waitTimeSeconds` bound (max `60`) waits and returns the result; on timeout it falls back to async. `sync: false` plus `null` returns an execution ID. Use sync for fast commands. |
