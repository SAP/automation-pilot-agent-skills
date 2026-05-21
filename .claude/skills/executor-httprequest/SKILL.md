---
name: executor-httprequest
description: Master the HTTP executor for Automation Pilot commands. Covers HTTP executor parameters, expression sanitization, timeout/retry configuration, response transformers, and HTTP-specific error handling. Use when building commands that make HTTP/REST API calls.
---

# HTTP Executor Guide

The HTTP executor (`http-sapcp:HttpRequest:1`) is the most commonly used executor in Automation Pilot. This guide covers everything you need to build robust HTTP-based commands.

## Parameters Reference

| Parameter | Type | Required | Default | Notes |
|-----------|------|----------|---------|-------|
| `url` | string | ✅ | - | Use `toUrlEncoded` for dynamic parts |
| `method` | string | ✅ | - | GET, POST, PUT, PATCH, DELETE |
| `headers` | object | ❌ | - | JSON object: `{"Content-Type": "application/json"}` |
| `body` | string | ❌ | - | Use `toEscapedJson` for dynamic JSON values |
| `timeout` | number | ❌ | 3 | Seconds (1-90, recommend ≤10) |
| `responseBodyTransformer` | string | ❌ | - | `toObject`, `toString`, `toNumber`, `.nested.path` |
| `user` | string | ❌ | - | Basic auth username OR OAuth client ID |
| `password` | string | ❌ | - | Basic auth password OR OAuth client secret |
| `tokenUrl` | string | ❌ | - | OAuth token endpoint |
| `refreshToken` | string | ❌ | - | OAuth refresh token flow |
| `clientId` | string | ❌ | - | OAuth client ID (with refreshToken) |
| `clientSecret` | string | ❌ | - | OAuth client secret (with refreshToken) |

---

## ⚠️ Valid Expression Functions

**CRITICAL:** Only use expression functions that exist in the codebase. Common functions:
- **URL/JSON Sanitization**: `toUrlEncoded`, `toEscapedJson` (MANDATORY for dynamic values)
- **Response Parsing**: `toObject`, `toNumber`, `toString`
- **Header Access**: `getCaseInsensitive` (MANDATORY for headers - case-insensitive)
- **Validation**: `length`, `contains`, `isGuid`, `valueIn`
- **Array/Object**: `filter`, `map`, `select`, `any`, `all`, `keys`, `values`

**Do not** invent functions like `type`, `isBool`, `isString` - they don't exist in Automation Pilot.

Only use functions listed in `references/expressions.md` or the official SAP documentation.

---

## Expression Sanitization (CRITICAL)

### URL Parameters - Always Use toUrlEncoded

**❌ WRONG - Unsafe for special characters:**
```json
{
  "url": "https://api.example.com/users/$(.execution.input.userId)"
}
```

**✅ CORRECT - Safe for all characters:**
```json
{
  "url": "https://api.example.com/users/$(.execution.input.userId | toUrlEncoded)"
}
```

**Why:** User IDs might contain `/`, `?`, `&`, `=`, spaces, or Unicode characters that break URLs.

**Common URL Patterns:**
```json
// Path parameter
"url": "https://api.cf.eu10.hana.ondemand.com/v3/spaces/$(.execution.input.spaceId | toUrlEncoded)"

// Query parameter (single)
"url": "https://api.example.com/search?q=$(.execution.input.query | toUrlEncoded)"

// Query parameter (multiple)
"url": "https://api.example.com/items?name=$(.execution.input.name | toUrlEncoded)&type=$(.execution.input.type | toUrlEncoded)"

// Dynamic subdomain
"url": "https://$(.execution.input.tenantId | toUrlEncoded).api.example.com/resource"
```

---

### JSON Body Values - Always Use toEscapedJson

**❌ WRONG - Breaks on quotes, newlines, backslashes:**
```json
{
  "body": "{\"name\": \"$(.execution.input.name)\", \"description\": \"$(.execution.input.description)\"}"
}
```

**✅ CORRECT - Handles all special characters:**
```json
{
  "body": "{\"name\": \"$(.execution.input.name | toEscapedJson)\", \"description\": \"$(.execution.input.description | toEscapedJson)\"}"
}
```

**Why:** Descriptions might contain quotes (`"`), newlines (`\n`), backslashes (`\`), or Unicode that breaks JSON.

**Examples:**
```json
"body": "{\"message\": \"$(.execution.input.message | toEscapedJson)\"}"
"body": "{\"name\": \"$(.execution.input.name | toEscapedJson)\", \"count\": $(.execution.input.count), \"enabled\": $(.execution.input.enabled)}"
```
**Rule:** Strings use `toEscapedJson`. Numbers/booleans: no escaping, no quotes.

---

### Response Headers - Use getCaseInsensitive

**❌ WRONG - Case-sensitive, fragile:**
```json
{
  "expression": "$(.apiCall.output.headers.Content-Type)"
}
```

**✅ CORRECT - Case-insensitive, robust:**
```json
{
  "expression": "$(.apiCall.output.headers | getCaseInsensitive(\"content-type\"))"
}
```

**Why:** HTTP headers are case-insensitive per RFC 7230. Server might return `Content-Type`, `content-type`, or `CONTENT-TYPE`.

**Header Access Patterns:**
```json
// Get content type
"expression": "$(.apiCall.output.headers | getCaseInsensitive(\"content-type\"))"

// Get location header (from 201 Created)
"expression": "$(.apiCall.output.headers | getCaseInsensitive(\"location\"))"

// Get custom header
"expression": "$(.apiCall.output.headers | getCaseInsensitive(\"x-request-id\"))"

// Check if header exists (check length, not isNil which doesn't exist)
"expression": "$(.apiCall.output.headers | getCaseInsensitive(\"x-api-version\") | length)"
// Then use operator: GREATER_THAN, values: ["0"]
```

---

## Response Transformers

| Transformer | Example | Result |
|-------------|---------|--------|
| `toObject` | `{"name":"test"}` → `{name: "test"}` | Parse JSON |
| `toObject.data.id` | `{"data":{"id":"123"}}` → `"123"` | Extract nested |
| `toObject.resources[0].guid` | `{"resources":[{"guid":"abc"}]}` → `"abc"` | Extract from array |
| `toString` | `123` → `"123"` | Convert to string |
| `toNumber` | `"123"` → `123` | Convert to number |
| (none) | Raw response | No transformation |

---

## Timeout Configuration

### Default Rules

**Recommended:** ≤ 10 seconds
**Default:** 5 seconds (if not specified)

```json
{
  "timeout": "5"  // Seconds as STRING, NOT milliseconds
}
```

### When to Adjust Timeout

| Scenario | Recommended Timeout | Reason |
|----------|---------------------|--------|
| Simple GET requests | 3-5s | Fast API responses |
| POST/PUT operations | 10s | Processing time needed |
| Async triggers (returns immediately) | 5-10s | Just triggers, doesn't wait |
| File uploads | 30s | Large data transfer |
| Long-running operations (polling) | 5-10s | Each poll should be fast |
| Complex operations (imports, exports) | 60-90s | Heavy processing |

**Note:** For truly long-running operations, use polling with `repeat` instead of increasing timeout (see Example 3). The maximum allowed timeout is 90 seconds.

---

## Retry Logic (autoRetry)

> **📘 UNIVERSAL PROPERTY**: `autoRetry` works with **ALL executor types** (HTTP, Script, ForEach, etc.), not just HTTP requests. The examples below use HTTP, but the same configuration applies to any executor.

### Retry Configuration Structure

```json
"autoRetry": {
  "maxCount": 3,
  "delay": "5s",
  "logic": "INCREMENTAL",
  "applyOnValidation": false,
  "when": {
    "semantic": "OR",
    "conditions": [{
      "semantic": "OR",
      "cases": [{
        "expression": "$([408, 429, 500, 502, 503, 504, -1] | filter(. == $.aliasName.output.status) | length)",
        "operator": "EQUALS",
        "semantic": "OR",
        "values": ["1"]
      }]
    }]
  }
}
```

**Fields:**
- `maxCount`: Maximum retry attempts (number)
- `delay`: Wait between retries (string with unit, e.g., "5s")
- `logic`: `INCREMENTAL` | `FIXED` (only these two exist)
- `applyOnValidation`: If `true`, also retries when `validate` fails. Default: `false`
- `when`: Condition for when to retry (typically checks status codes)

**⚠️ CRITICAL:** Use `"logic"` NOT `"delayType"`. Use `"when"` block with filter expression, NOT `"statusCodes"` array.

### Delay Types

| Type | Behavior | Example (5s base, 3 retries) | Best For |
|------|----------|------------------------------|----------|
| `INCREMENTAL` | Adds delay each time | 5s, 10s, 15s | General use, backoff |
| `FIXED` | Same delay each time | 5s, 5s, 5s | Rate limiting, polling |

### Status Codes by HTTP Method

| Method | Retry Status Codes | Reason |
|--------|-------------------|--------|
| GET, DELETE, HEAD | 408, 429, 500, 502, 503, 504, -1 | Idempotent - safe to retry |
| POST, PUT, PATCH | 429, 502, 503, 504 | Non-idempotent - no 500 (might have partially succeeded) |

**Status codes:** 408=timeout, 429=rate limit, 500/502/503/504=server errors, -1=network error

---

## Authentication

**Basic:**
```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "input": {
    "url": "https://api.example.com/resource",
    "method": "GET",
    "user": "$(.execution.input.user)",
    "password": "$(.execution.input.password)"
  },
  "alias": "apiCall"
}
```

**OAuth (user/password):**
```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "input": {
    "url": "https://api.cf.eu10.hana.ondemand.com/v3/spaces",
    "method": "GET",
    "user": "$(.execution.input.user)",
    "password": "$(.execution.input.password)",
    "tokenUrl": "https://uaa.cf.eu10.hana.ondemand.com/oauth/token"
  },
  "alias": "listSpaces"
}
```

**OAuth (refresh token):**
```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "input": {
    "url": "https://api.cf.eu10.hana.ondemand.com/v3/spaces",
    "method": "GET",
    "refreshToken": "$(.execution.input.refreshToken)",
    "clientId": "$(.execution.input.clientId)",
    "clientSecret": "$(.execution.input.clientSecret)",
    "tokenUrl": "https://uaa.cf.eu10.hana.ondemand.com/oauth/token"
  },
  "alias": "listSpaces"
}
```

**Service Key:**
```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "input": {
    "url": "$(.execution.input.serviceKey.endpoints.service_manager_url)/v1/service_instances",
    "method": "GET",
    "user": "$(.execution.input.serviceKey.uaa.clientid)",
    "password": "$(.execution.input.serviceKey.uaa.clientsecret)",
    "tokenUrl": "$(.execution.input.serviceKey.uaa.url)/oauth/token"
  },
  "alias": "listInstances"
}
```

**Bearer Token (Custom Headers):**
```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "input": {
    "url": "https://api.example.com/resource",
    "method": "GET",
    "headers": "{\"Authorization\": \"Bearer $(.execution.input.apiToken)\", \"X-API-Key\": \"$(.execution.input.apiKey)\"}"
  },
  "alias": "apiCall"
}
```

---

## HTTP Error Handling

> **📘 UNIVERSAL PROPERTY**: `errorMessages` works with **ALL executor types**, not just HTTP. Use it to provide context-specific error messages for any command failure. The HTTP status code examples below are specific to HTTP, but the structure applies to all executors.

### Error Message Structure

```json
"errorMessages": [
  {
    "message": "Dynamic error message with '$(.execution.input.field)' in single quotes",
    "when": {
      "semantic": "OR",
      "conditions": [
        {
          "semantic": "OR",
          "cases": [
            {
              "expression": "$(.executor.output.status)",
              "operator": "EQUALS",
              "semantic": "OR",
              "values": ["404"]
            }
          ]
        }
      ]
    }
  }
]
```

### Common HTTP Error Patterns

| Status | Message Pattern | When Clause |
|--------|----------------|-------------|
| **401** | `"Authentication failed. Check credentials for user '$(.execution.input.user)'"` | `status EQUALS 401` |
| **403** | `"User '$(.execution.input.user)' does not have permission to access resource '$(.execution.input.resourceName)'"` | `status EQUALS 403` |
| **404** | `"Resource '$(.execution.input.resourceName)' does not exist"` | `status EQUALS 404` |
| **409** | `"Resource '$(.execution.input.name)' already exists in space '$(.execution.input.space)'"` | `status EQUALS 409` |
| **422** | `"Invalid parameters: $(.createResource.output.result \| toObject.error)"` | `status EQUALS 422` |
| **500** | `"API returned internal server error. Operation may need to be retried"` | `status EQUALS 500` |
| **-1, 0** | `"Network error or timeout. Check connectivity to '$(.execution.input.apiUrl)'"` | `status IN ["-1", "0"]` |
| **≥400** | `"Operation failed: $(.apiCall.output.result \| toObject.error.message)"` | `status GREATER_THAN_OR_EQUAL 400` |

**Note:** See complete examples section for full JSON error message structures.

---

## Complete Examples

### Example 1: Simple GET with Retry

```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "input": {
    "url": "https://api.cf.eu10.hana.ondemand.com/v3/spaces/$(.execution.input.spaceId | toUrlEncoded)",
    "method": "GET",
    "user": "$(.execution.input.user)",
    "password": "$(.execution.input.password)",
    "tokenUrl": "https://uaa.cf.eu10.hana.ondemand.com/oauth/token",
    "timeout": "5",
    "responseBodyTransformer": "toObject"
  },
  "alias": "getSpace",
  "description": null,
  "progressMessage": null,
  "initialDelay": null,
  "pause": null,
  "when": null,
  "validate": null,
  "autoRetry": {
    "maxCount": 3,
    "delay": "5s",
    "logic": "INCREMENTAL",
    "when": {
      "semantic": "OR",
      "conditions": [{
        "semantic": "OR",
        "cases": [{
          "expression": "$([408, 429, 500, 502, 503, 504, -1] | filter(. == $.getSpace.output.status) | length)",
          "operator": "EQUALS",
          "semantic": "OR",
          "values": ["1"]
        }]
      }]
    }
  },
  "repeat": null,
  "errorMessages": [
    {
      "message": "Space '$(.execution.input.spaceId)' does not exist",
      "when": {
        "semantic": "OR",
        "conditions": [
          {
            "semantic": "OR",
            "cases": [
              {
                "expression": "$(.getSpace.output.status)",
                "operator": "EQUALS",
                "semantic": "OR",
                "values": ["404"]
              }
            ]
          }
        ]
      }
    },
    {
      "message": "Authentication failed for user '$(.execution.input.user)'",
      "when": {
        "semantic": "OR",
        "conditions": [
          {
            "semantic": "OR",
            "cases": [
              {
                "expression": "$(.getSpace.output.status)",
                "operator": "EQUALS",
                "semantic": "OR",
                "values": ["401"]
              }
            ]
          }
        ]
      }
    }
  ],
  "dryRun": null
}
```

**Key Points:**
- ✅ URL parameter uses `toUrlEncoded`
- ✅ OAuth authentication with tokenUrl
- ✅ Timeout set to 5 seconds
- ✅ Response parsed with `toObject`
- ✅ Retry on transient errors (GET is idempotent)
- ✅ Specific error messages for 404 and 401

---

### Example 2: POST with JSON Body

```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "input": {
    "url": "https://api.cf.eu10.hana.ondemand.com/v3/service_instances",
    "method": "POST",
    "user": "$(.execution.input.user)",
    "password": "$(.execution.input.password)",
    "tokenUrl": "https://uaa.cf.eu10.hana.ondemand.com/oauth/token",
    "headers": "{\"Content-Type\": \"application/json\"}",
    "body": "{\"type\": \"managed\", \"name\": \"$(.execution.input.name | toEscapedJson)\", \"relationships\": {\"space\": {\"data\": {\"guid\": \"$(.execution.input.spaceGuid | toEscapedJson)\"}}}, \"parameters\": $(.execution.input.parameters)}",
    "timeout": "10",
    "responseBodyTransformer": "toObject.guid"
  },
  "alias": "createInstance",
  "description": null,
  "progressMessage": null,
  "initialDelay": null,
  "pause": null,
  "when": null,
  "validate": null,
  "autoRetry": {
    "maxCount": 3,
    "delay": "5s",
    "logic": "INCREMENTAL",
    "when": {
      "semantic": "OR",
      "conditions": [{
        "semantic": "OR",
        "cases": [{
          "expression": "$([429, 502, 503, 504] | filter(. == $.createInstance.output.status) | length)",
          "operator": "EQUALS",
          "semantic": "OR",
          "values": ["1"]
        }]
      }]
    }
  },
  "repeat": null,
  "errorMessages": [
    {
      "message": "Service instance '$(.execution.input.name)' already exists",
      "when": {
        "semantic": "OR",
        "conditions": [
          {
            "semantic": "OR",
            "cases": [
              {
                "expression": "$(.createInstance.output.status)",
                "operator": "EQUALS",
                "semantic": "OR",
                "values": ["409"]
              }
            ]
          }
        ]
      }
    },
    {
      "message": "Invalid parameters: $(.createInstance.output.result | toObject.errors[0].detail)",
      "when": {
        "semantic": "OR",
        "conditions": [
          {
            "semantic": "OR",
            "cases": [
              {
                "expression": "$(.createInstance.output.status)",
                "operator": "EQUALS",
                "semantic": "OR",
                "values": ["422"]
              }
            ]
          }
        ]
      }
    }
  ],
  "dryRun": null
}
```

**Key Points:**
- ✅ POST method for creation
- ✅ JSON body with `toEscapedJson` for string values
- ✅ Parameters object passed without escaping (already JSON)
- ✅ Content-Type header set
- ✅ Response transformer extracts GUID directly
- ✅ Retry only on infrastructure errors (POST not idempotent)
- ✅ Specific errors for 409 (conflict) and 422 (validation)

---

### Example 3: Multi-Step with Dynamic Lookup

```json
{
  "executors": [
    {
      "execute": "http-sapcp:HttpRequest:1",
      "input": {
        "url": "$(.execution.input.serviceKey.endpoints.service_manager_url)/v1/instances/$(.execution.input.instanceId | toUrlEncoded)",
        "method": "PATCH",
        "user": "$(.execution.input.serviceKey.uaa.clientid)",
        "password": "$(.execution.input.serviceKey.uaa.clientsecret)",
        "tokenUrl": "$(.execution.input.serviceKey.uaa.url)/oauth/token",
        "headers": "{\"Content-Type\": \"application/json\"}",
        "body": "{\"parameters\": $(.execution.input.parameters)}",
        "timeout": "10",
        "responseBodyTransformer": "toObject"
      },
      "alias": "trigger",
      "description": null,
      "progressMessage": null,
      "initialDelay": null,
      "pause": null,
      "when": null,
      "validate": null,
      "autoRetry": {
        "maxCount": 3,
        "delay": "5s",
        "logic": "INCREMENTAL",
        "when": {
          "semantic": "OR",
          "conditions": [{
            "semantic": "OR",
            "cases": [{
              "expression": "$([429, 502, 503, 504] | filter(. == $.trigger.output.status) | length)",
              "operator": "EQUALS",
              "semantic": "OR",
              "values": ["1"]
            }]
          }]
        }
      },
      "repeat": null,
      "errorMessages": [],
      "dryRun": null
    },
    {
      "execute": "http-sapcp:HttpRequest:1",
      "input": {
        "url": "$(.execution.input.serviceKey.endpoints.service_manager_url)/v1/operations/$(.trigger.output.result.id | toUrlEncoded)",
        "method": "GET",
        "user": "$(.execution.input.serviceKey.uaa.clientid)",
        "password": "$(.execution.input.serviceKey.uaa.clientsecret)",
        "tokenUrl": "$(.execution.input.serviceKey.uaa.url)/oauth/token",
        "timeout": "5",
        "responseBodyTransformer": "toObject"
      },
      "alias": "poll",
      "description": null,
      "progressMessage": null,
      "initialDelay": "5s",
      "pause": null,
      "when": null,
      "validate": null,
      "autoRetry": null,
      "repeat": {
        "maxCount": 60,
        "delay": "10s",
        "until": {
          "semantic": "OR",
          "conditions": [{
            "semantic": "OR",
            "cases": [{
              "expression": "$(.poll.output.result.state)",
              "operator": "IN",
              "semantic": "OR",
              "values": ["succeeded", "failed"]
            }]
          }]
        }
      },
      "errorMessages": [{
        "message": "Operation failed: $(.poll.output.result.errors[0].description)",
        "when": {
          "semantic": "OR",
          "conditions": [{
            "semantic": "OR",
            "cases": [{
              "expression": "$(.poll.output.result.state)",
              "operator": "EQUALS",
              "semantic": "OR",
              "values": ["failed"]
            }]
          }]
        }
      }],
      "dryRun": null
    }
  ]
}
```

**Key Points:**
- ✅ Step 1: Trigger async update (PATCH)
- ✅ Step 2: Poll operation status with repeat
- ✅ Initial delay before first poll (5s)
- ✅ Poll every 10s up to 60 times (10 minutes max)
- ✅ Extract operation ID from trigger response
- ✅ Check for completion states (succeeded/failed)
- ✅ Extract error details if operation failed

---

## Related Resources

- **AGENTS.md** - Section 4 (Expression Reference) for basic sanitization examples
- **code-review SKILL** - HTTP Best Practices section for validation rules
- **oq-testing SKILL** - Test HTTP commands with comprehensive coverage
