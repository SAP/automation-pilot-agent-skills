# Command model

Read this when defining or changing a command's caller-facing contract or top-level JSON shape.
It owns command naming, input/output key structure, wire serialization, and catalog placement.
The skill workflow loads composition controls separately when a command has a configuration.

Resolve every referenced command and Input on the target tenant before deployment.

## Contents

- [Top-level shape](#top-level-shape)
- [Naming and catalog placement](#naming-and-catalog-placement)
- [Input keys](#input-keys)
- [Output keys](#output-keys)

## Top-level shape

```json
{
  "id": "welcome-<<<TENANT_ID>>>:CommandName:1",
  "catalog": "welcome-<<<TENANT_ID>>>",
  "name": "CommandName",
  "description": "What the command does, in the caller's terms",
  "version": 1,
  "inputKeys": {},
  "outputKeys": {},
  "configuration": {},
  "tags": {}
}
```

SAP built-ins may read back as atomic commands with `configuration: null`. A tenant command created
through `autopi commands create` must be composite and provide a configuration object,
even when it wraps one built-in command.

## Naming and catalog placement

| Element | Convention | Example |
|---|---|---|
| Command or Input name | PascalCase | `RestartCfApp`, `BtpCredentials` |
| Input/output key | camelCase | `resourceName`, `subAccount` |
| Executor alias | camelCase | `getResource`, `createInstance` |

Author commands in the tenant catalog already settled with the user. There is no default catalog; resolve a missing target before a tenant write. Reserve the `-sapcp` suffix for SAP-provided catalogs.

Use the returned catalog ID in commands and Inputs rather than inferring it from a display
name. Custom catalog creation and its naming limits belong to the Catalogs skill.
The [SAP catalog guide](https://help.sap.com/docs/automation-pilot/automation-pilot/catalog)
documents ownership, dependencies, and import behavior. Imports can overwrite existing content
and omit sensitive exported values; verify prerequisites and supply secrets from their approved
source rather than treating an export as a complete credential backup.

## Input keys

```json
"inputKeys": {
  "paramName": {
    "type": "string",
    "description": "Caller-facing meaning and constraints",
    "required": true,
    "sensitive": false,
    "defaultValue": "value",
    "minValue": null,
    "maxValue": null,
    "minSize": null,
    "maxSize": null,
    "allowedValues": null,
    "allowedValuesFromInputKeys": null,
    "suggestedValues": null,
    "suggestedValuesFromInputKeys": null,
    "defaultValueFromInput": null
  }
}
```

Supported types are `string`, `number`, `boolean`, `array`, and `object`. Read a live command or
Input contract before assuming which constraints are present.

### Wire serialization

`defaultValue` is always a JSON string on the wire, regardless of declared type: `"5"`,
`"true"`, `"[\"a\",\"b\"]"`, or `"{\"k\":\"v\"}"`. The same boundary applies to direct
execution input values and condition-block `values` entries.

For a region selector, resolve the relevant reusable Input live, then use it from
`allowedValuesFromInputKeys`. Common SAP-provided IDs are `metadata-sapcp:CfRegionData:1` and
`metadata-sapcp:NeoRegionData:1`; treat those IDs as a product snapshot until the tenant confirms
them with `autopi inputs get`.

Authentication keys such as `user`, `password`, or `refreshToken` can be optional when callers may
supply credentials through reusable Inputs. A missing default on such an optional key can produce
an informational validation issue; assess issue severity instead of inventing a default secret.

### Collection bounds

On the tenant tested on 2026-10-26, `minSize` and `maxSize` on array inputs constrained the
serialized JSON text: three city names failed `maxSize: 5`. MCP schemas also emitted
`minLength`/`maxLength` with unconstrained `items`, rather than element-count limits.
Enforce collection counts and member shapes in an explicit validation step before fan-out or
API calls. Use size fields only for an intentional serialized-size limit verified on the tenant.

## Output keys

```json
"outputKeys": {
  "result": {
    "type": "string",
    "description": "Caller-facing meaning and shape",
    "sensitive": false
  }
}
```

An `object` or `array` output crosses a command boundary as a JSON-encoded string. A consuming
command parses it with `toObject` or `toArray` before reading nested values. An executor skipped by
a condition exposes no output, so final mappings that read conditionally executed steps must supply
an explicit fallback of the declared type.

