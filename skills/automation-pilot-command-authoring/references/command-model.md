# Command model

Read this when defining or changing a command's caller-facing contract or top-level JSON shape.
It owns command naming, input/output key structure, wire serialization, and catalog placement.
The skill workflow loads composition controls separately when a command has a configuration.

Resolve every referenced command and Input on the target tenant before deployment.

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
through `automation-pilot:create_command` must be composite and provide a configuration object,
even when it wraps one built-in command.

## Naming and catalog placement

| Element | Convention | Example |
|---|---|---|
| Command or Input name | PascalCase | `RestartCfApp`, `BtpCredentials` |
| Input/output key | camelCase | `resourceName`, `subAccount` |
| Executor alias | camelCase | `getResource`, `createInstance` |
| Default tenant catalog | `welcome-` plus tenant suffix | `welcome-<<<TENANT_ID>>>` |

Author commands in a tenant catalog. Use the tenant's `welcome-<<<TENANT_ID>>>` catalog unless the
user names another tenant catalog. Reserve the `-sapcp` suffix for SAP-provided catalogs.

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
them with `automation-pilot:get_input`.

Authentication keys such as `user`, `password`, or `refreshToken` can be optional when callers may
supply credentials through reusable Inputs. A missing default on such an optional key can produce
an informational validation issue; assess issue severity instead of inventing a default secret.

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

