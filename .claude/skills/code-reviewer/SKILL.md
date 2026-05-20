---
name: code-review
description: Review production commands, inputs, and catalogs for Automation Pilot. Validates file structure, naming conventions, security requirements, mandatory patterns, and best practices. Use when reviewing .command.json, .input.json, or .catalog.json files.
---

# Code Review

Validate production commands meet structural, security, and quality standards.

## AI Assistant Guidelines

⚠️ **CRITICAL - When reviewing as an AI assistant:**
- **DO NOT suggest or make any changes** to expressions, response body transformers, or execution properties unless there is a clear error
- **Review comments should be added ONLY** when there is a concern, warning, or an issue. When something is fine, do not add comments or suggest changes
- **Never write code comments** in the properties fields of the commands
- **Do not add newline at end of a file** - files should end without trailing newline
- **Flag violations immediately** - critical issues (file structure, security, naming) must be caught

⚠️ **JSON FILES DO NOT SUPPORT COMMENTS:**
- **NEVER suggest adding `//` or `/* */` comments** - JSON syntax doesn't allow comments
- **NEVER suggest documenting status codes inline** - this would break the JSON
- **Standard retry status codes are well-known** - do not ask to document them:
  - `-1` = Network/connection failure (standard in this codebase)
  - `408` = Request Timeout
  - `429` = Too Many Requests  
  - `500, 502, 503, 504` = Server errors
- **If documentation is needed**, it belongs in SKILL.md files, not in JSON

❌ **WRONG - Never suggest this:**
```json
// Consider defining these in a comment
"expression": "$([408, 429, -1] | filter(...))"
```

✅ **CORRECT - This pattern is standard and needs no comment:**
```json
"expression": "$([408, 429, 500, 502, 503, 504, -1] | filter(. == $.<httpExecutorAlias>.output.status) | length)"
```

**Standard patterns that should NOT be flagged:**
- `semantic: "AND"` / `semantic: "OR"` conditions - these are clear, not "confusing"
- `$({...} | toObject)` for building JSON objects - this IS the cleanest approach
- Empty body checks combined with error message checks - valid error handling
- NO trailing newlines in JSON files - this is our convention (contrary to POSIX)

## Critical File Structure Rules

### File Naming (CRITICAL - Build Fails If Wrong)

**File name MUST match JSON `"name"` field EXACTLY:**

```
✅ CORRECT
File: CreateDirectory.command.json
JSON: "name": "CreateDirectory"

❌ WRONG (Build will fail!)
File: CreateDirectory.command.json
JSON: "name": "CreateDir"
```

**File Extensions:**
- Commands: `.command.json`
- Inputs: `.input.json`
- Catalogs: `.catalog.json`

**File Placement:**

| Type | Location | Example |
|------|----------|---------|
| Production command | `content/public/catalogs/[catalog]/` | `catalogs/http/HttpRequest.command.json` |
| Canary command | `content/canary/` | `canary/SensitiveHttpRequest.command.json` |
| Internal command | `content/internal/catalogs/[catalog]/` | `internal/catalogs/cld/GetCldSystem.command.json` |

**Critical:** ALL files (.command.json, .input.json, .catalog.json) MUST be inside `content/` folder.

### Naming Conventions

| Element | Convention | Example | Max Length |
|---------|-----------|---------|------------|
| Command names | PascalCase | `CreateDirectory`, `HttpRequest` | 32 chars |
| Input/Output keys | camelCase | `resourceName`, `subAccount` | 32 chars |
| Aliases | camelCase | `getResource`, `createInstance` | 32 chars |
| Catalog IDs | lowercase | `http-sapcp`, `cf-sapcp` | 32 chars |

**❌ Common Violations:**
- Name exceeds 32 characters (e.g., `testDelayedVoidWithDynamicMessage` = 35 chars)
- Wrong case (e.g., `create_directory` instead of `CreateDirectory`)
- File name doesn't match JSON name

### Data Types and Constraints

**Validate input/output key definitions:**
- Correct **data types** used: array, string, object, number, boolean
- Optional input keys have **default** values when appropriate
- **allowedValues** specified when there's a fixed set of valid options
- **suggestedValues** provided when there's a recommended default

**Example:**
```json
{
  "timeout": {
    "type": "number",
    "sensitive": false,
    "required": false,
    "minSize": null,
    "maxSize": null,
    "minValue": null,
    "maxValue": null,
    "allowedValues": ["1", "2", "5", "10"],
    "allowedValuesFromInputKeys": null,
    "suggestedValues": ["5"],
    "suggestedValuesFromInputKeys": null,
    "defaultValue": "5",
    "defaultValueFromInput": null,
    "description": "Request timeout in seconds"
  }
}
```

## Security Requirements (CRITICAL)

### Mandatory Sensitive Flags

These fields MUST have `"sensitive": true`:

```json
"password": {
  "type": "string",
  "sensitive": true,  // ⚠️ REQUIRED
  "required": true,
  "description": "The password for the specified technical user or the client secret for the specified OAuth 2.0 client ID to be used for authentication. Related input keys: 'user' or 'tokenUrl'"
}
```

**Fields requiring sensitive flag:**
- `password`
- `clientSecret`
- `refreshToken`
- `serviceKey`
- Any field containing credentials, tokens, or secrets

### Input Data References (valueFrom Pattern)

Sensitive data should be stored in Input definitions and referenced:

**Pattern 1: Load Entire Input (when you need multiple fields)**
```json
"values": [{
  "alias": "TechnicalUser",
  "valueFrom": {
    "inputReference": "OQ-<<<TENANT_ID>>>:TechnicalUser:1",
    "inputKey": null  // null = load all fields
  }
}]
// Access: $(.TechnicalUser.user), $(.TechnicalUser.password)
```

**Pattern 2: Load Specific Field**
```json
"values": [{
  "alias": "Password",
  "valueFrom": {
    "inputReference": "OQ-<<<TENANT_ID>>>:ServiceAccount:1",
    "inputKey": "password"  // Load only this field
  }
}]
// Access: $(.Password)
```

**Pattern 3: Direct Input (production commands)**
```json
"inputKeys": {
  "password": {
    "type": "string",
    "sensitive": true,
    "required": true
  }
}
// Access in executors: $(.execution.input.password)
```

**Pattern 4: Dynamic Metadata Lookup**
```json
"values": [{
  "alias": "regionData",
  "valueFrom": {
    "inputReference": "metadata-sapcp:CfRegionData:1",
    "inputKey": "$(.execution.input.region)"  // Dynamic key
  }
}]
// Access: $(.regionData.cfApiUrl), $(.regionData.uaaTokenUrl)
```

## Mandatory Description Patterns

⚠️ **All descriptions of the following parameters must be one of these exact sentences or start with them.** No variations, alternatives, or creative rewording allowed for these Input and Output keys!

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

**When reviewing descriptions, verify:**
- Authentication parameters use exact patterns from the list above
- Region parameters include examples (cf-eu10, neo-eu1, etc.)
- Output parameters like `status` include example values
- No creative rewording of mandatory patterns

### Description Rules

**DO NOT:**
- End with period unless complete sentence
- Use forbidden verbs: abort, terminate, kill, disable
- Use generic descriptions like "The password."
- Skip examples when helpful

**DO:**
- Start with capital letter
- Include examples where applicable
- Be specific and descriptive
- Use exact mandatory patterns for auth/region

**Examples of Good vs Bad Descriptions:**

| ✅ Good | ❌ Bad | Issue |
|---------|--------|-------|
| `"Password used for authentication"` | `"The password."` | Ends with punctuation, too generic |
| `"Technical name of the Cloud Foundry region"` | `"Region name"` | Doesn't follow mandatory pattern |
| `"Status of the Jenkins build execution"` | `"Build status."` | Ends with punctuation, too vague |

## Expression Sanitization (CRITICAL)

**Validation Rules:**

| Element | Required Sanitization | Validation Check |
|---------|----------------------|------------------|
| URL parameters | `toUrlEncoded` | Flag any URL with `$(.` that lacks `\| toUrlEncoded` |
| JSON string values | `toEscapedJson` | Flag JSON body strings with `$(.` that lack `\| toEscapedJson` |
| HTTP headers | `getCaseInsensitive` | Flag direct header access (e.g., `.headers.Content-Type`) |
| Response transformers | `toObject`, `toNumber`, `toString` | Verify transformer present for JSON parsing |

**Non-Existent Functions (DO NOT USE):**

| ❌ Wrong | ✅ Correct |
|---------|-----------|
| `\| type` (doesn't exist) | Use `NOT_EQUALS ""` to check existence |
| `\| isBool` / `\| isString` (don't exist) | Check value directly with `EQUALS` |
| `\| isNil` (doesn't exist) | Use `\| length` with `EQUALS "0"` |
| Invented expression functions | Verify with `grep -r "\| functionName" content/**/*.command.json` |

## HTTP Best Practices (Validation Rules)

| Element | Rule | Check |
|---------|------|-------|
| **Timeout** | ≤ 10 seconds | Flag timeout > 10 (unless justified) |
| **Retry (GET/DELETE)** | Status codes: 408, 429, 500, 502, 503, 504, -1 | Verify autoRetry present for idempotent ops |
| **Retry (POST/PATCH)** | Status codes: 429, 502, 503, 504 only | Flag 500 in POST retry (non-idempotent) |
| **Error Messages** | Dynamic values in single quotes | Flag static messages like "Resource not found" |
| **Error Structure** | `when.semantic.OR.conditions.cases` | Verify proper error condition structure |

**For HTTP implementation details:** See [executor-httprequest SKILL](./executor-httprequest/SKILL.md)

## Validation Examples

### Example 1: Input Definition with Sensitive Data

**Focus:** Sensitive flags, mixed sensitive/non-sensitive keys

```json
{
  "id": "OQ-<<<TENANT_ID>>>:TechnicalUser:1",
  "name": "TechnicalUser",
  "description": "Technical user credentials for testing",
  "catalog": "OQ-<<<TENANT_ID>>>",
  "version": 1,
  "keys": {
    "password": {
      "type": "string",
      "sensitive": true,
      "description": null
    },
    "user": {
      "type": "string",
      "sensitive": false,
      "description": null
    },
    "mail": {
      "type": "string",
      "sensitive": false,
      "description": null
    }
  },
  "values": {
    "password": "",
    "user": "P2004056162",
    "mail": "technical.user@example.com"
  }
}
```

**Key Points:**
- ✅ `password` marked as `sensitive: true`
- ✅ Non-sensitive fields explicitly marked `sensitive: false`
- ✅ Each key has explicit sensitive flag
- ✅ Values remain empty/placeholder (filled at runtime)

---

### Example 2: Error Handling Patterns

**Focus:** Multiple error conditions, authentication failures

```json
{
  "executors": [{
    "execute": "http-sapcp:HttpRequest:1",
    "input": {
      "url": "https://api.example.com/resource",
      "method": "POST"
    },
    "alias": "createResource",
    "description": null,
    "progressMessage": null,
    "initialDelay": null,
    "pause": null,
    "when": null,
    "validate": null,
    "autoRetry": null,
    "repeat": null,
    "errorMessages": [
      {
        "message": "Resource '$(.execution.input.name)' already exists",
        "when": {
          "semantic": "OR",
          "conditions": [{
            "semantic": "OR",
            "cases": [{
              "expression": "$(.createResource.output.status)",
              "operator": "EQUALS",
              "semantic": "OR",
              "values": ["409"]
            }]
          }]
        }
      },
      {
        "message": "Authentication failed. Check user '$(.execution.input.user)' and password",
        "when": {
          "semantic": "OR",
          "conditions": [{
            "semantic": "OR",
            "cases": [{
              "expression": "$(.createResource.output.status)",
              "operator": "EQUALS",
              "semantic": "OR",
              "values": ["401", "403"]
            }]
          }]
        }
      },
      {
        "message": "Invalid input: $(.createResource.output.body | toObject.error)",
        "when": {
          "semantic": "OR",
          "conditions": [{
            "semantic": "OR",
            "cases": [{
              "expression": "$(.createResource.output.status)",
              "operator": "EQUALS",
              "semantic": "OR",
              "values": ["400"]
            }]
          }]
        }
      }
    ],
    "dryRun": null
  }]
}
```

**Key Points:**
- ✅ Multiple error conditions for different status codes
- ✅ Dynamic values in single quotes
- ✅ Specific error messages per scenario
- ✅ Extract error details from response body

## Validation Checklist

Validate in this order:

### 1. File Structure (CRITICAL - Build Fails)
- File name matches `"name"` field exactly (PascalCase for commands/inputs)
- File extension: `.command.json`, `.input.json`, or `.catalog.json`
- File location: `content/public/catalogs/[catalog]/` (production)
- All names ≤ 32 characters
- Files inside `content/` folder (build fails if outside)

### 2. Naming Conventions (CRITICAL)
- Commands/Inputs: PascalCase (CreateDirectory, HttpRequest)
- Input/output keys: camelCase (resourceName, subAccount)
- Aliases: camelCase (getResource, createInstance)
- Catalog IDs: lowercase (http-sapcp, cf-sapcp)

### 3. Security (CRITICAL)
- `password` → `"sensitive": true`
- `clientSecret` → `"sensitive": true`
- `refreshToken` → `"sensitive": true`
- `serviceKey` → `"sensitive": true`
- Any credentials/tokens → `"sensitive": true`

### 4. Mandatory Descriptions (HIGH PRIORITY)
- `password` → Exact pattern: "The password for the specified technical user..."
- `user` → Exact pattern: "The name of a technical user or OAuth 2.0 client ID..."
- `region` → Exact pattern: "The technical name of the [CF/Neo] region. Example: ..."
- No forbidden verbs (abort, terminate, kill, disable)
- Proper punctuation (period only for complete sentences)

### 5. Expression Sanitization (HIGH PRIORITY)
- See [Expression Sanitization](#expression-sanitization-critical) section above for detailed rules
- Key checks: `toUrlEncoded`, `toEscapedJson`, `getCaseInsensitive`, avoid non-existent functions

### 6. HTTP Best Practices (MEDIUM PRIORITY)
- See [HTTP Best Practices](#http-best-practices-validation-rules) section above for detailed rules
- Key checks: timeout ≤ 10s, proper retry codes, dynamic error messages

### 7. Data Types (MEDIUM PRIORITY)
- Correct types: string, number, boolean, array, object
- Optional keys have `default` values
- `allowedValues` specified when applicable
- `suggestedValues` provided when helpful

### 8. Deprecated Commands (HIGH PRIORITY)
- No deprecated commands in new content
- Check for tag: `"tags": {"autopi:deprecated": ""}`
- Replace deprecated commands with substitutes
- Verify substitute specified when deprecating

### Quick Violation Flags
```
❌ CRITICAL: File name ≠ JSON "name" → Build fails
❌ CRITICAL: Name > 32 chars → Build fails
❌ CRITICAL: Missing sensitive flag → Security breach
❌ HIGH: Generic descriptions → Fails review
❌ HIGH: No URL encoding (toUrlEncoded) → Breaks with special chars
❌ HIGH: No JSON escaping (toEscapedJson) → Breaks with quotes
❌ MEDIUM: Timeout > 10s unjustified → Performance issue
❌ MEDIUM: Generic error messages → Poor UX
```

## Related Skills

- **oq-testing** - For creating and reviewing OQ test files (.OQ.command.json)
- Use code-review for production commands, oq-testing for test files
