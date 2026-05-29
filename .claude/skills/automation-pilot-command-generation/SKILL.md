---
name: automation-pilot-command-generation
description: Generate SAP Automation Pilot commands with dynamic expressions, jq transformations, and composite workflows. Use when creating commands, building orchestration flows, or working with Automation Pilot expressions.
version: 1.2.0
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
  "https://$AUTOPI_HOSTNAME/api/v1/catalogs?own=false" | jq '.data[] | {id, name}'

# Find commands in a catalog
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/commands?catalog=applm-sapcp" | jq '.data[] | {id, name}'

# Get full command definition (inputs, outputs, executors)
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/commands/applm-sapcp:RestartCfApp:1" | jq .
```

This ensures you always use valid executor names and correct parameters, even for newly added catalogs.

## IMPORTANT: Dry Run Configuration (MANDATORY)

**Every executor MUST include a `dryRun` configuration.** This enables safe testing of commands without making actual changes when the execution is triggered with the `feature:dryRun` tag.

The `dryRun.output` contains **sample values for ALL output keys** that the executed command would produce. When running in dry run mode, these mock values are returned instead of actually executing the command.

```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "alias": "getResource",
  "input": {
    "url": "https://api.example.com/resource",
    "method": "GET"
  },
  "dryRun": {
    "output": {
      "status": "200",
      "body": "{\"id\": \"mock-id-123\", \"name\": \"Mock Resource\"}",
      "headers": "{\"content-type\": \"application/json\"}",
      "method": "GET",
      "url": "https://api.example.com/resource",
      "time": "100",
      "size": "50"
    }
  }
}
```

### How to Define Dry Run Output

1. **Identify the executed command's output keys** - Check what outputs the command produces (e.g., `HttpRequest:1` outputs: `status`, `body`, `headers`, `method`, `url`, `time`, `size`)
2. **Provide sample values for ALL output keys** - Every output key must have a mock value
3. **Use realistic values** - Values should represent what a successful execution would return
4. **Ensure downstream compatibility** - If other executors reference this output (e.g., `$(.getResource.output.body)`), the mock values must work with those expressions

### Common Executor Output Keys

**`http-sapcp:HttpRequest:1`:**
```json
"dryRun": {
  "output": {
    "status": "200",
    "body": "{\"result\": \"success\"}",
    "headers": "{\"content-type\": \"application/json\"}",
    "method": "GET",
    "url": "https://api.example.com",
    "time": "100",
    "size": "50"
  }
}
```

**`utils-sapcp:Void:1`:**
```json
"dryRun": {
  "output": {
    "message": "Dry run: skipped actual operation"
  }
}
```

**`scripts-sapcp:ExecuteScript:2`:**
```json
"dryRun": {
  "output": {
    "output": "[\"Mock script output line 1\", \"Mock output line 2\"]",
    "exitCode": "0"
  }
}
```

**`kubernetes-sapcp:ListK8sResources:1`:**
```json
"dryRun": {
  "output": {
    "status": "200",
    "body": "{\"apiVersion\": \"v1\", \"kind\": \"PodList\", \"items\": []}"
  }
}
```

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
    "maxValue": 100,
    "allowedValuesFromInputKeys": ["metadata-sapcp:CfRegionData:1"]
  }
}
```

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

Load reusable metadata like region configurations:

```json
"values": [
  {
    "alias": "regionData",
    "valueFrom": {
      "inputReference": "metadata-sapcp:CfRegionData:1",
      "inputKey": "$(.execution.input.region)"
    }
  }
]
```

### Executors

Each executor runs a command with mapped inputs. **Every executor MUST include a `dryRun` configuration.**

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
  "when": null,
  "validate": null,
  "autoRetry": null,
  "repeat": null,
  "errorMessages": [],
  "dryRun": {
    "output": {
      "status": "200",
      "body": "{\"resources\": [], \"pagination\": {\"total_results\": 0}}",
      "headers": "{\"content-type\": \"application/json\"}",
      "method": "GET",
      "url": "https://api.cf.example.com/v3/apps",
      "time": "150",
      "size": "100"
    }
  }
}
```

## Dynamic Expressions

Expressions use jq 1.6 syntax wrapped in `$()`. Access data with:
- `.execution.input.key` - Input values
- `.stepAlias.output.key` - Previous step outputs
- `.regionData.field` - Values from input references
- `$` - Global scope (use inside pipes)

### Essential Expressions

| Pattern | Description |
|---------|-------------|
| `$(.execution.input.name)` | Access input |
| `$(.step.output.body \| toObject)` | Parse JSON response |
| `$(.step.output.body \| toObject.items[0].id)` | Extract nested value |
| `$(if .cond then "a" else "b" end)` | Conditional |
| `$(.arr \| map(.field))` | Transform array |
| `$(.str \| toUrlEncoded)` | URL encode |
| `$(.obj \| toObject.key // "default")` | Fallback value |

### String Operations

- `toUpperCase`, `toLowerCase`, `strip`
- `split(".")`, `join(",")`
- `gsub("old"; "new")` - Replace all
- `"\(.var) text"` - Interpolation

### Array/Object Operations

- `filter(condition)`, `select(condition)`
- `map(transform)`, `sort`, `sortBy(.field)`
- `unique`, `uniqueBy(.field)`
- `keys`, `values`, `length`
- `toEntries`, `fromEntries`

### Type Conversions

- `toObject`, `toArray`, `toString`, `toNumber`, `toBoolean`
- `toBase64`, `fromBase64`
- `toUrlEncoded`, `fromUrlEncoded`
- `toMd5`, `toSha256`
- `fromYaml`, `toYaml`

### Utilities

- `now` - Unix timestamp
- `guid`, `guidShort` - Generate UUIDs
- `isGuid` - Validate GUID format

## IMPORTANT: Expression Complexity Limits

Expressions have a complexity limit. If you get "Expression contains too many elements" error, break complex object construction into intermediate `Void` steps:

```json
// WRONG - too many elements in one expression
"output": {
  "counts": "$({\"a\": .x, \"b\": .y, \"c\": .z, \"d\": .w, \"e\": .v, ...})"
}

// CORRECT - use Void step to build complex objects
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
5. **Add dryRun config** - Define mock outputs for each executor (MANDATORY)
6. **Add error handling** - Validate, autoRetry, errorMessages
7. **Test incrementally** - Verify each step works (use `--dry-run` first)

## Mandatory Description Patterns

⚠️ **When creating commands, use these exact description patterns for standard parameters.** No variations or creative rewording allowed!

### 📥 Input Parameters

**Authentication:**
- **password**
  - description
    - The password for the specified technical user or the client secret for the specified OAuth 2.0 client ID to be used for authentication. Related input keys: 'user' or 'tokenUrl'
    - The password of a service account with [permissions]. Related input keys: 'user'
    - The password of the provided user
  - type - string | sensitive 🔒
- **user**
  - description
    - The name of a technical user or an OAuth 2.0 client ID to be used for authentication. Related input keys: 'password' or 'tokenUrl'
    - The ID (username) of a service account with [permissions]. Related input keys: 'password'
    - The user ID or the email of a Cloud Foundry user to be used for authentication
    - The username of the [service/system]
  - type - string
- **refreshToken**
  - description
    - An OAuth 2.0 refresh token to be used for authentication. If 'refreshToken' is passed, 'user' and 'password' will be ignored. Related input keys: 'clientId', 'clientSecret', 'tokenUrl'
    - An OAuth 2.0 refresh token which will be used to get a new access token via the Refresh Token grant type. Related input keys: 'clientId', 'clientSecret', 'tokenUrl'
  - type - string | sensitive 🔒

**Region & Organization:**
- **region**
  - description
    - The technical name of the Cloud Foundry region. Example: cf-eu10, cf-eu10-002
    - The technical name of the Neo region. Example: neo-eu1, neo-us1
  - type - string
  - Allowed values should be used
- **subAccount**
  - description
    - The name or the ID of the Cloud Foundry organization. Examples: my-org-name-1, 0ffeb410-5f78-0000-af5c-5b26baf46623
  - type - string

**Resources & Services:**
- **resourceName**
  - description
    - The technical name of the [resource type]. Example: [examples]
  - type - string
- **serviceKey**
  - description
    - A service key for [service]
  - type - object | sensitive 🔒
- **name**
  - description
    - The name of the [object]
  - type - string

### 📤 Output Parameters

**Status & Response:**
- **status**
  - description
    - The status of the [object]
    - The status code of the response. Examples: 200, 301, 404, 503
    - The status code of the response. Examples: 200, 301, 404, 503. In case of no response or a timeout, the status codes may be 0 and -1.
  - types - number, string, object
- **responseCode**
  - description
    - The HTTP response code
  - type - number
- **state**
  - description
    - The state of the [object/entity]. Examples: [examples]
  - type - string

**Data Collections:**
- **resourceInstancesStates**
  - description
    - An array of all application instances after command execution
  - type - array
- **output**
  - description
    - The original response from [service/API]
  - types - array, string, object
- **result**
  - description
    - The result of the [operation]
  - types - array, string, object

**System Operations:**
- **exitCode**
  - description
    - The exit code returned by the script execution
  - type - number
- **instanceId**
  - description
    - The ID of the [instance/object]
  - type - string

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

**Error:** Missing `dryRun` configuration on an executor
**Cause:** Executor defined without a `dryRun` block.
**Solution:** Every executor MUST include `dryRun.output` with sample values for ALL output keys of the executed command.

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
- **`GetResourceWithRetry.command.json`** - HTTP request with retry and validation
- **`WaitForOperation.command.json`** - Polling pattern with repeat
- **`ProcessAppsBatch.command.json`** - Batch processing with ForEach
