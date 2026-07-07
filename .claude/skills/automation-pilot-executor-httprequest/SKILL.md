---
name: automation-pilot-executor-httprequest
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
| `timeout` | number | ❌ | 5 | Seconds (1-90, recommend ≤10) |
| `responseBodyTransformer` | string | ❌ | - | `toObject`, `toString`, `toNumber`, `.nested.path` |
| `user` | string | ❌ | - | Basic auth username OR OAuth client ID |
| `password` | string | ❌ | - | Basic auth password OR OAuth client secret |
| `tokenUrl` | string | ❌ | - | OAuth token endpoint |
| `clientId` | string | ❌ | - | OAuth client ID for client credentials flow |
| `clientSecret` | string | ❌ | - | OAuth client secret for client credentials flow |
| `clientCert` | string | ❌ | - | X509 certificate + key (PEM) for mTLS/cert-based OAuth; use with `certurl`-based `tokenUrl` |
| `refreshToken` | string | ❌ | - | OAuth refresh token; when provided, `user`/`password` are ignored |
| `successResponseCodes` | string | ❌ | - | JSON array of HTTP status codes treated as success, e.g. `"[\"503\"]"`. Overrides default success check. |

---

## ⚠️ Valid Expression Functions

**CRITICAL:** Only use expression functions that exist in the codebase. Common functions:
- **URL/JSON Sanitization**: `toUrlEncoded`, `toEscapedJson` (MANDATORY for dynamic values)
- **Response Parsing**: `toObject`, `toNumber`, `toString`
- **Header Access**: `getCaseInsensitive` (MANDATORY for headers - case-insensitive)
- **Validation**: `length`, `contains`, `isGuid`, `valueIn`
- **Array/Object**: `filter`, `map`, `select`, `any`, `all`, `keys`, `values`

**Do not** invent functions like `type`, `isBool`, `isString` - they don't exist in Automation Pilot.

Only use functions listed in `../automation-pilot-command-generation/references/expressions.md` or the official SAP documentation.

---

## Expression Sanitization (CRITICAL)

### URL Parameters - Always Use toUrlEncoded

```json
"url": "$(.execution.input.serviceKey.endpoints.service_url)/v1/resources/$(.execution.input.resourceId | toUrlEncoded)"
```

Dynamic URL path/query parts must be wrapped in `toUrlEncoded`. Without it, special characters break the URL.

**Patterns:**
```json
"url": "https://example.com/v1/items/$(.execution.input.name | toUrlEncoded)"
"url": "https://example.com/v1/search?q=$(.execution.input.query | toUrlEncoded)"
"url": "https://example.com/v1/items?name=$(.execution.input.name | toUrlEncoded)&type=$(.execution.input.type | toUrlEncoded)"
```

---

### JSON Body Values - Always Use toEscapedJson

String values embedded in a JSON body must use `toEscapedJson`. Numbers and booleans: no escaping, no quotes.

```json
"body": "{\"name\": \"$(.execution.input.name | toEscapedJson)\", \"count\": $(.execution.input.count), \"enabled\": $(.execution.input.enabled)}"
```

---

### Response Headers - Use getCaseInsensitive

HTTP headers in the response must be accessed with `getCaseInsensitive` — header names are case-insensitive.

```json
"expression": "$(.apiCall.output.headers | getCaseInsensitive(\"content-type\"))"
"expression": "$(.apiCall.output.headers | getCaseInsensitive(\"location\"))"
"expression": "$(.apiCall.output.headers | getCaseInsensitive(\"x-api-version\") | length)"
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
  "timeout": "5"
}
```

Note: value is a STRING (seconds), not a number and not milliseconds.

### When to Adjust Timeout

| Scenario | Recommended Timeout |
|----------|---------------------|
| Simple GET | 3-5s |
| POST/PUT | 10s |
| Long-running async trigger | 5-10s |
| File upload / heavy processing | 60-90s |

Maximum allowed: **90 seconds**. For operations that take longer, use `repeat` polling instead.

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

### Unconditional Retry

Set `"when": null` to retry on any failure, regardless of status code:

```json
"autoRetry": {
  "maxCount": 3,
  "delay": "5s",
  "logic": "FIXED",
  "applyOnValidation": false,
  "when": null
}
```

### Delay Types

| Type | Behavior | Example (5s base, 3 retries) | Best For |
|------|----------|------------------------------|----------|
| `INCREMENTAL` | Adds delay each time | 5s, 10s, 15s | General use, backoff |
| `FIXED` | Same delay each time | 5s, 5s, 5s | Rate limiting, polling |

### Status Codes for autoRetry

| Method | Retry on |
|--------|----------|
| GET, DELETE, HEAD | 408, 429, 500, 502, 503, 504, -1 |
| POST, PUT, PATCH | 429, 502, 503, 504 |

`-1` = network error / no response.

---

## Authentication

**User/Password with OAuth:**
```json
{
  "url": "$(.regionData.cfApiUrl)/v3/spaces",
  "method": "GET",
  "user": "$(.execution.input.user)",
  "password": "$(.execution.input.password)",
  "tokenUrl": "$(.regionData.uaaTokenUrl)"
}
```

**Service Key (client credentials):**
```json
{
  "url": "$(.execution.input.serviceKey.endpoints.service_url)/v1/resources",
  "method": "GET",
  "clientId": "$(.execution.input.serviceKey.uaa.clientid)",
  "clientSecret": "$(.execution.input.serviceKey.uaa.clientsecret)",
  "tokenUrl": "$(.execution.input.serviceKey.uaa.url)/oauth/token"
}
```

**Service Key with mTLS (clientCert):**
```json
{
  "url": "$(.execution.input.serviceKey.endpoints.service_url)/v1/resources",
  "method": "GET",
  "clientCert": "$(.execution.input.serviceKey.uaa.certificate + .execution.input.serviceKey.uaa.key)",
  "clientId": "$(.execution.input.serviceKey.uaa.clientid)",
  "clientSecret": "$(.execution.input.serviceKey.uaa.clientsecret)",
  "tokenUrl": "$((if (.execution.input.serviceKey.uaa.certificate == null) and (.execution.input.serviceKey.uaa.key == null) then .execution.input.serviceKey.uaa.url else .execution.input.serviceKey.uaa.certurl end) + \"/oauth/token\")"
}
```

**Refresh token:**
```json
{
  "user": "$(.execution.input.user)",
  "password": "$(.execution.input.password)",
  "refreshToken": "$(.execution.input.refreshToken)",
  "tokenUrl": "$(.regionData.uaaTokenUrl)"
}
```

---

## Error Handling

> **📘 UNIVERSAL PROPERTY**: `errorMessages` works with **ALL executor types**, not just HTTP.

### Error Message Structure

```json
"errorMessages": [
  {
    "message": "Resource '$(.execution.input.resourceName)' does not exist",
    "when": {
      "semantic": "OR",
      "conditions": [
        {
          "semantic": "OR",
          "cases": [
            {
              "expression": "$(.aliasName.output.status)",
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

**Error body access:** Use `toObject.error.message` for single-error APIs, `toObject.errors[0].detail` for array-style APIs (e.g. CF API v3). Always check the actual response shape.

