# Catalog model

Read this when constructing a custom catalog create/update payload or planning deletion.
These limits come from the CLI's pinned API contract; resolve service discrepancies from
current evidence.

## Fields

| Field | Creation | Rules |
|---|---|---|
| `technicalName` | Required | Immutable; 1–10 ASCII letters/digits. Used with the tenant identifier to generate the catalog ID. |
| `name` | Required | Display name; 1–64 ASCII letters/digits, spaces, hyphens, or underscores. |
| `description` | Optional | At most 512 characters; describe purpose and lifecycle. |
| `documentationUrl` | Optional | HTTP(S) URL, empty string, or null; at most 2000 characters. |
| `tags` | Optional | String-valued object; at most 50 entries. |
| `id`, `owner` | Server-owned | Read-back identity and ownership; omit from write payloads. |

## Create

```json
{
  "technicalName": "ops",
  "name": "Operations",
  "description": "Reusable operational commands and configuration for the platform team"
}
```

Validate with `autopi lint --kind catalog catalog.json`;
`autopi catalogs create -f catalog.json --print-request` previews the request offline.
A conflict does not authorize another technical name: inspect the matching existing
catalog and reconcile its purpose with the request.

## Update

Catalog updates require `name`. Build the replacement from a full current read, retaining
editable `description`, `documentationUrl`, and `tags` unless their change is requested.
Omit server-owned fields and the immutable `technicalName`. Validate with
`autopi lint --kind catalog-update --mode api catalog-update.json` before the PUT.

```json
{
  "name": "Platform Operations",
  "description": "Reusable operational commands and configuration for the platform team"
}
```

The example assumes the current catalog has no additional authored URL or tags. Preserve
those fields when present. Catalog If-Match is unsupported by the current service rules;
read back after updating and reconcile concurrent changes from the latest state.

## Deletion

Inspect `autopi commands list --catalog <id>` and `autopi inputs list --catalog <id>` to
establish the contained resources. Resolve dependency impact through the catalog skill's
deletion workflow before selecting a cascade; contained-resource inventory alone does
not establish the absence of dependents.

- `autopi catalogs delete <id>` deletes an empty catalog and returns HTTP 204.
- `autopi catalogs delete <id> --force` cascades to contained Commands and Inputs and
  returns deleted counts. Use it when removal of those exact contents is authorized.

Inspect the deletion result and verify `autopi catalogs get <id>` returns 404. Other
errors or timeouts do not establish deletion. SAP-provided catalogs belong to the
provider; use the owned catalog inventory to resolve a custom-catalog target.
