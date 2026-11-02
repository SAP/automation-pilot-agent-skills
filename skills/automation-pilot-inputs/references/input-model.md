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

For a new Input, include `"version": 1` explicitly in the definition. `autopi lint` requires
a positive integer version. An Input belongs to exactly one catalog and its name/version
pair must be unique there. Use a tenant-owned catalog; reserve `-sapcp` catalogs for
SAP-provided content. An Input supports at most 32 keys.

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

`autopi inputs get` deliberately removes the complete `values` object, not only sensitive entries. Input write tools return only identity metadata (including nested bulk results) from successful responses and suppress server error payloads entirely because an error message could echo a submitted secret. For HTTP 400, stderr still reports `error: HTTP 400: Bad Request` and the CLI exits 4; the server's detailed error reason is hidden. A read-back can verify the contract but cannot prove the stored value. Prefer merge patches because a full replacement assembled from `autopi inputs get` would silently omit every stored value.

## Sensitivity

Set `sensitive: true` for passwords, client secrets, tokens, private keys, certificates with private material, service keys, and structures containing any of them. Sensitivity protects storage and display but does not make descriptions or tags safe places for secrets.

Personal Inputs cannot be used to build commands or as bindings for triggers and scheduled executions, and service accounts cannot own them. Create reusable tenant Inputs for agent-managed workflows.

## Bulk import and other product capabilities

`autopi inputs bulk-create -f FILE` imports an envelope with an `inputs` array of full
definitions. It supports `overwriteExisting`, `skipExisting`, and
`keepExistingSensitiveValues`. Choose overwrite or skip deliberately; the contract
does not define their precedence when both are true or promise transactional import.
Use `keepExistingSensitiveValues: true` when preserving existing stored secrets, and
provide required `values` in each full definition. Validate with
`autopi lint --kind bulk-inputs --mode api FILE`, and transmit secret-bearing imports
through protected files/stdin.

Dedicated version-creation, cloning, Input export, and content-connector backup verbs
are unavailable. Creating a definition with a chosen name/version uses the normal
create contract; do not invent special endpoints for these product capabilities.

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

Input PATCH accepts only description, keys, values, and tags in a nonempty JSON object.
It preserves omitted siblings. Values and individual stored values cannot be null.
A null key descriptor deletes a key; a non-null key descriptor requires a corresponding
string value in the patch. Assemble key/tag removals deliberately and verify visible
metadata afterward. Stored value siblings remain unreadable, so successful write
responses provide the evidence for hidden value changes. Use the ETag from the same
metadata read with `--if-match`; a 412 requires another read and reconciliation.
