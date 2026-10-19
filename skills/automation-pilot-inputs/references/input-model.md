# Input Model

Read this before creating, replacing, or deleting an Automation Pilot Input.

## Definition shape

```json
{
  "catalog": "welcome-<<<TENANT_ID>>>",
  "name": "CloudFoundryCredentials",
  "description": "Credentials used for Cloud Foundry operations.",
  "version": 1,
  "keys": {
    "user": {
      "type": "string",
      "description": "The ID of the technical user.",
      "sensitive": false
    },
    "password": {
      "type": "string",
      "description": "The password of the technical user. Related input keys: 'user'",
      "sensitive": true
    }
  },
  "values": {
    "user": "service-user",
    "password": "<secret supplied through an approved path>"
  },
  "tags": {
    "environment": "production"
  }
}
```

Automation Pilot creates a new Input at version 1. An Input belongs to exactly one catalog and its name/version pair must be unique there. Use a tenant-owned catalog; reserve `-sapcp` catalogs for SAP-provided content. An Input supports at most 32 keys.

Key names may contain English letters, digits, dashes, and underscores; use the repository's camelCase convention for authored keys. Supported types are `string`, `number`, `boolean`, `array`, and `object`.

## Wire values

Every entry in `values` is a JSON string, regardless of the declared type:

| Declared type | Stored wire value |
|---|---|
| `string` | `"eu10"` |
| `number` | `"42"` |
| `boolean` | `"true"` |
| `array` | `"[\"a\",\"b\"]"` |
| `object` | `"{\"url\":\"https://example.invalid\"}"` |

The service validates the string against the declared type. Do not double-encode strings or send native JSON numbers, booleans, arrays, or objects in `values`.

`autopi inputs get` deliberately removes the complete `values` object, not only sensitive entries. Input write tools remove `values` recursively from successful responses and suppress server error payloads entirely because an error message could echo a submitted secret. The HTTP status remains available for diagnosis. A read-back can verify the contract but cannot prove the stored value. Prefer merge patches because a full replacement assembled from `autopi inputs get` would silently omit every stored value.

## Sensitivity

Set `sensitive: true` for passwords, client secrets, tokens, private keys, certificates with private material, service keys, and structures containing any of them. Sensitivity protects storage and display but does not make descriptions or tags safe places for secrets.

Personal Inputs cannot be used to build commands or as bindings for triggers and scheduled executions, and service accounts cannot own them. Create reusable tenant Inputs for agent-managed workflows.

## Unsupported product capabilities

Creating a new version, cloning, importing/exporting, and content-connector backup are product capabilities not exposed by the `autopi` CLI. State that limitation instead of emulating one with an unverified endpoint.

## Patch shapes

Rotate selected values without replacing the rest:

```json
{
  "values": {
    "password": "<new secret>"
  }
}
```

Change visible metadata:

```json
{
  "description": "Credentials used by production Cloud Foundry operations.",
  "tags": {
    "environment": "production",
    "owner": "platform-team"
  }
}
```

JSON Merge Patch recursively updates included object members and preserves omitted siblings. Setting a member to `null` removes it. When deleting or restructuring keys or tags, assemble the patch deliberately and verify the visible map afterward. Stored value siblings remain unreadable, so a successful patch response is the available evidence that an omitted value was preserved.
