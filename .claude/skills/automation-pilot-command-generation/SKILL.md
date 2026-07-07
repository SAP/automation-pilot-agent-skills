---
name: automation-pilot-command-generation
description: Generate SAP Automation Pilot commands with dynamic expressions, jq transformations, and composite workflows. Use when creating commands, building orchestration flows, or working with Automation Pilot expressions and script execution (Bash, Python, Node.js, PowerShell).
---

# SAP Automation Pilot Command Development

This skill provides guidance for creating SAP Automation Pilot commands by composing existing reference commands from the content library.

## Command Release Policy

Deployed commands start in DRAFT state. Do not release automatically — only release when:
1. The command has been tested and works correctly
2. The user explicitly requests release

Draft state allows safe iteration without affecting production.

## Overview

SAP Automation Pilot commands are JSON definitions that automate operations on SAP BTP. Commands can be:
- **Atomic**: Execute a single operation (e.g., `HttpRequest`, `SendEmail`)
- **Composite**: Orchestrate multiple steps using executors

### Reference Materials

- **`examples/`** — Pattern examples (HTTP, ForEach, polling workflows)
- **`references/patterns.md`** — Common implementation patterns
- **`references/expressions.md`** — Expression language reference

### Discovering Available Executors

Use the **catalog-explorer** skill to discover available executors via API:

```bash
# List available catalogs
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/catalogs?own=false"

# Find commands in a catalog
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/commands?catalog=applm-sapcp"

# Get full command definition (inputs, outputs, executors)
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/commands/applm-sapcp:RestartCfApp:1"
```

This ensures you always use valid executor names and correct parameters, even for newly added catalogs.

## Dry Run

`dryRun` is always present on every executor. Set it to `null` unless you have a specific reason to provide mock output.

```json
"dryRun": null
```

The `dryRun: {output: {...}}` form (with mock values) is only needed if a downstream executor references this executor's output AND you want dry-run mode to propagate realistic values through the chain. In practice, `null` is the standard.

## IMPORTANT: Catalog Naming & ForEach Version

See **`references/catalogs.md`** for the complete catalog reference. Key rules:

- **Generated commands**: Use `<<<TENANT_ID>>>` suffix (e.g., `mycommands-<<<TENANT_ID>>>:MyCommand:1`)
- **Built-in SAP commands**: Use `sapcp` suffix (e.g., `http-sapcp:HttpRequest:1`)
- **ForEach**: Always use `ForEach:2`, never `ForEach:1` (deprecated)

## Command Structure

### Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Command name | PascalCase | `RestartCfApp`, `GetHanaInstance` |
| Input name | PascalCase | `BtpCredentials`, `JiraConfig` |
| Input/output keys | camelCase | `resourceName`, `subAccount` |
| Aliases | camelCase | `getResource`, `createInstance` |
| Catalog ID | kebab-case | `mycommands-xxx` |

Names must not contain spaces (causes API issues). PascalCase matches SAP's built-in commands.

### Basic Command Definition

```json
{
  "id": "mycommands-<<<TENANT_ID>>>:CommandName:1",
  "catalog": "mycommands-<<<TENANT_ID>>>",
  "name": "CommandName",
  "description": "What the command does",
  "version": 1,
  "inputKeys": { },
  "outputKeys": { },
  "configuration": null,
  "tags": { }
}
```

### Input Keys

Define command parameters:

```json
"inputKeys": {
  "paramName": {
    "type": "string",           // string, number, boolean, array, object
    "description": "Description",
    "required": true,
    "sensitive": false,         // true masks in logs
    "defaultValue": "value",
    "minValue": 1,              // for numbers
    "maxValue": 100
  }
}
```

Additional optional fields exist: `allowedValues` (fixed set of valid options), `suggestedValues` (hints shown in UI), `allowedValuesFromInputKeys` and `suggestedValuesFromInputKeys` (dynamic lists from an input reference). Omit them when not needed.

**`defaultValue` is always a JSON string on the wire** regardless of the declared `type`. For non-string types, stringify the JSON: `"defaultValue": "5"` for a number, `"defaultValue": "true"` for a boolean, `"defaultValue": "[\"a\",\"b\"]"` for an array, `"defaultValue": "{\"key\":\"val\"}"` for an object.

**Region inputs** must always use `allowedValuesFromInputKeys` to constrain the value to the SAP-provided region list:
- Cloud Foundry: `"allowedValuesFromInputKeys": ["metadata-sapcp:CfRegionData:1"]`
- Neo: `"allowedValuesFromInputKeys": ["metadata-sapcp:NeoRegionData:1"]`

This ensures only valid Automation Pilot regions can be entered.

### Output Keys

Define command outputs:

```json
"outputKeys": {
  "result": {
    "type": "string",
    "description": "The result",
    "sensitive": false
  }
}
```

## Composite Commands

Commands with `configuration` orchestrate multiple steps:

```json
"configuration": {
  "values": [],      // Pre-computed values from input references
  "output": {},      // Final output mapping
  "executors": [],   // Steps to execute
  "listeners": []    // Event handlers
}
```

### Input References (values)

Load reusable credentials or metadata at execution time. Three forms of `inputKey`:

```json
"values": [
  {
    "alias": "regionData",
    "valueFrom": {
      "inputReference": "metadata-sapcp:CfRegionData:1",
      "inputKey": "$(.execution.input.region)"
    }
  },
  {
    "alias": "fullInputObject",
    "valueFrom": {
      "inputReference": "mycatalog-<<<TENANT_ID>>>:MyCredentials:1",
      "inputKey": null
    }
  },
  {
    "alias": "ServiceAccountPassword",
    "valueFrom": {
      "inputReference": "mycatalog-<<<TENANT_ID>>>:MyCredentials:1",
      "inputKey": "password"
    }
  }
]
```

- **Expression** (`"$(.execution.input.region)"`) — dynamically selects which key to load based on input
- **`null`** — loads the entire input reference as an object (access its fields via `$(.alias.fieldName)`)
- **Plain string** (`"password"`) — loads a single named key directly

### Executors

Each executor runs a command with mapped inputs.

```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "alias": "getResource",
  "input": {
    "url": "$(.regionData.cfApiUrl)/v3/apps",
    "method": "GET",
    "user": "$(.execution.input.user)",
    "password": "$(.execution.input.password)",
    "tokenUrl": "$(.regionData.uaaTokenUrl)",
    "clientId": "cf",
    "timeout": "20"
  },
  "description": null,
  "progressMessage": null,
  "initialDelay": null,
  "pause": null,
  "when": null,
  "validate": null,
  "autoRetry": null,
  "repeat": null,
  "errorMessages": [],
  "dryRun": null
}
```

## Dynamic Expressions

Expressions use jq 1.6 syntax wrapped in `$()`. Access data with:
- `.execution.input.key` - Input values
- `.stepAlias.output.key` - Previous step outputs
- `.regionData.field` - Values from input references
- `$` - Global scope (use inside pipes)

See **[`references/expressions.md`](references/expressions.md)** for the full expression reference — patterns, string/array/object operations, type conversions, and utilities.

## IMPORTANT: Expression Complexity Limits

Expressions have a complexity limit. If you get "Expression contains too many elements" error, break complex object construction into intermediate `Void` steps:

Too complex — avoid this:

```json
"output": {
  "counts": "$({\"a\": .x, \"b\": .y, \"c\": .z, \"d\": .w, \"e\": .v, ...})"
}
```

Use a `Void` step instead:

```json
"executors": [
  {
    "execute": "utils-sapcp:Void:1",
    "alias": "buildCounts",
    "input": {
      "message": "{\"a\": $(.x), \"b\": $(.y), \"c\": $(.z)}"
    }
  }
],
"output": {
  "counts": "$(.buildCounts.output.message | toObject)"
}
```

**Rule of thumb**: If building an object with more than 5-6 fields from different step outputs, use a `Void` step.

## Conditional Execution (when)

Skip steps based on conditions:

```json
"when": {
  "semantic": "OR",
  "conditions": [
    {
      "semantic": "OR",
      "cases": [
        {
          "expression": "$(.execution.input.skipStep)",
          "operator": "EQUALS",
          "semantic": "OR",
          "values": ["false"]
        }
      ]
    }
  ]
}
```

Operators: `EQUALS`, `NOT_EQUALS`, `CONTAINS`, `NOT_CONTAINS`, `STARTS_WITH`, `ENDS_WITH`

## Validation

Assert conditions on step output:

```json
"validate": {
  "semantic": "OR",
  "conditions": [
    {
      "semantic": "OR",
      "cases": [
        {
          "expression": "$(.step.output.status)",
          "operator": "EQUALS",
          "semantic": "OR",
          "values": ["200", "201"]
        }
      ]
    }
  ]
}
```

## Auto Retry

Retry on transient failures. Valid `logic` values: `FIXED` (constant delay) or `INCREMENTAL` (increasing delay).

```json
"autoRetry": {
  "maxCount": 3,
  "delay": "5s",
  "logic": "FIXED",
  "applyOnValidation": false,
  "when": {
    "semantic": "OR",
    "conditions": [
      {
        "semantic": "OR",
        "cases": [
          {
            "expression": "$([408, 429, 500, 502, 503, 504, -1] | filter(. == $.step.output.status) | length)",
            "operator": "EQUALS",
            "semantic": "OR",
            "values": ["1"]
          }
        ]
      }
    ]
  }
}
```

## Repeat (Polling)

Poll until condition is met:

```json
"repeat": {
  "maxCount": 100,
  "delay": "15s",
  "failOnMaxCount": true,
  "until": {
    "semantic": "OR",
    "conditions": [
      {
        "semantic": "OR",
        "cases": [
          {
            "expression": "$(.step.output.body | toObject.state)",
            "operator": "EQUALS",
            "semantic": "OR",
            "values": ["COMPLETE"]
          }
        ]
      }
    ]
  }
}
```

## Error Messages

Provide custom error messages:

```json
"errorMessages": [
  {
    "message": "Operation failed: $(.step.output.body | toObject.error)",
    "when": {
      "semantic": "OR",
      "conditions": [
        {
          "semantic": "OR",
          "cases": [
            {
              "expression": "$(.step.output.status)",
              "operator": "NOT_EQUALS",
              "semantic": "OR",
              "values": ["200"]
            }
          ]
        }
      ]
    }
  }
]
```

## Development Workflow

1. **Identify requirements** - Inputs, outputs, steps needed
2. **Find reference commands** - Use catalog-explorer skill or check `references/catalogs.md`
3. **Design the flow** - Map out executors and data transformations
4. **Write expressions** - Use jq syntax for data manipulation
5. **Add error handling** - validate, autoRetry, errorMessages
6. **Test incrementally** - Verify each step works (use `feature:dryRun` tag)

## Mandatory Description Patterns

⚠️ **When creating commands, use these exact description patterns for standard parameters.** No variations or creative rewording allowed!

See the complete authoritative list in **[`references/description-patterns.md`](references/description-patterns.md)**.

## Examples

### Example 1: HTTP health check command

User: "Create a command that checks if an API endpoint is healthy"

1. Create a composite command with one `http-sapcp:HttpRequest:1` executor
2. Add `url` and `expectedStatus` as input keys
3. Add `validate` to assert response status matches expected
4. Add `autoRetry` for transient failures (429, 500, 502, 503, 504)
5. Include `dryRun` with a mock 200 response
6. Use `<<<TENANT_ID>>>` suffix for the catalog

### Example 2: Batch processing with ForEach

User: "Build a command that processes a list of items in parallel batches of 5"

1. Create a main composite command with an `items` input key (array type)
2. Create a sub-command for processing a single item
3. Use `utils-sapcp:ForEach:2` (never `ForEach:1`) with `batchSize: "5"`
4. Pass items via the `inputs` parameter
5. Include `dryRun` on both the ForEach executor and the sub-command executors

### Example 3: Trigger-and-poll workflow

User: "Create a command that starts an async operation and polls until it completes"

1. Create a composite command with a trigger step (`HttpRequest` PATCH/POST)
2. Add a polling step with `repeat` configuration
3. Set `repeat.until` to check for completion states (e.g., `succeeded`, `failed`)
4. Set `repeat.maxCount` and `repeat.delay` for timeout protection
5. Set `failOnMaxCount: true` so the command fails if polling exceeds the limit
6. Extract operation ID from the trigger response and pass to the poll URL

---

## Troubleshooting

**Error:** "Expression contains too many elements"
**Cause:** A single expression builds an object with too many fields (>5-6 from different step outputs).
**Solution:** Break into intermediate `utils-sapcp:Void:1` steps to build partial objects, then combine.

**Error:** Command references `ForEach:1`
**Cause:** Using the deprecated version.
**Solution:** Always use `utils-sapcp:ForEach:2`. Version 1 is deprecated and must never be used.

**Error:** Catalog suffix uses `-sapcp` for a generated command
**Cause:** Applying the wrong suffix convention.
**Solution:** Generated commands use `<<<TENANT_ID>>>` suffix. Only SAP-provided built-in commands use `-sapcp`.

**Error:** Missing `dryRun` field on an executor
**Cause:** Executor defined without a `dryRun` field.
**Solution:** Always include `"dryRun": null` unless mock output propagation through a chain is needed.

---

## Additional Resources

### Reference Files

For detailed patterns and complete expression reference:
- **`references/expressions.md`** - Complete dynamic expressions guide
- **`references/patterns.md`** - Common command patterns
- **`references/catalogs.md`** - Available commands by catalog
- **`references/supported-data-manipulation-expressions.pdf`** - Official SAP documentation

### Example Files

Working command examples in `examples/` (note PascalCase naming):
- **`CreateResourceWithServiceKey.command.json`** - POST with service key / clientCert auth, autoRetry, errorMessages, and conditional initialDelay
- **`GetResourceWithRetry.command.json`** - HTTP GET with OAuth, retry, validate, and errorMessages
- **`WaitForOperation.command.json`** - Polling pattern with repeat
- **`ProcessAppsBatch.command.json`** - Batch processing with ForEach
