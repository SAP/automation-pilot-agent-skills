---
name: automation-pilot-debugger
description: Debug and troubleshoot SAP Automation Pilot execution failures. Use when executions fail, need to investigate errors, check execution health (pass/fail summary), or understand error patterns. Provides error pattern matching with suggested fixes.
---

# SAP Automation Pilot Debugging & Troubleshooting

Debug failed Executions, check recent execution health, and investigate error patterns. Use this skill when things go wrong — not for normal operations.

## Quick Diagnostics

### Check Recent Executions Summary

```bash
# Last 10 executions summary
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions?limit=10"

# Last 5 failed executions
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions?status=FAILED&limit=5"

# Last 5 successful executions
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions?status=FINISHED&limit=5"
```

### Investigate a Specific Execution

```bash
EXEC_ID="your-execution-id"

curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID"

curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/logs"
```

---

## Error Pattern Reference

### "Parameter 'X' is required but not provided"

**Error**: API returns 400 with message like `"Parameter 'smtpHost' is required but not provided"` even when the parameter IS provided in the JSON payload.

**Root Cause**: Known issue (possibly canary environment specific or permission-related) where the execution API fails to recognize required input parameters passed at trigger time.

**Workaround**:
1. **Avoid required input parameters** - make all inputs optional with default values, OR
2. **Hardcode values in command definition** - put the values directly in the executor input mappings instead of using `$(.execution.input.paramName)`
3. Commands with NO required inputs (empty `inputKeys: {}` or all `required: false`) work correctly

**Example - Before (fails)**:
```json
"inputKeys": {
  "smtpHost": { "type": "string", "required": true }
},
"executors": [{
  "input": { "host": "$(.execution.input.smtpHost)" }
}]
```

**Example - After (works)**:
```json
"inputKeys": {},
"executors": [{
  "input": { "host": "mail.example.com" }
}]
```

---

### "Missing valid combination of input values for authentication"

**Full Error**: `"Missing valid combination of input values for authentication. Please select a valid option: 1) 'clientCert' for X509 2) 'user' & 'password' for Basic authentication"`

**Context**: Typically occurs with `email-sapcp:SendEmail:1` command.

**Root Cause**: The SMTP server requires authentication. Internal mail relays like `mail.sap.corp` that work from on-premise networks require authentication when accessed from cloud services (Automation Pilot runs in the cloud).

**Fix Options**:
1. Provide `user` and `password` for SMTP authentication
2. Provide `clientCert` for X509 certificate authentication
3. Use a different SMTP server that allows unauthenticated relay
4. Use SAP Alert Notification Service (ANS) instead of direct SMTP

---

### "The following input keys can only have default values from input, because they are marked as sensitive"

**Context**: Occurs when deploying a command.

**Root Cause**: Sensitive input keys (like passwords) cannot have hardcoded `defaultValue`. This is a security feature.

**Fix**: Remove `defaultValue` from sensitive fields. Use `defaultValueFromInput` to reference an Input object instead, or leave no default.

**Wrong**:
```json
"password": {
  "type": "string",
  "sensitive": true,
  "defaultValue": "<your-password>"
}
```

**Correct**:
```json
"password": {
  "type": "string",
  "sensitive": true
}
```

---

### "Command not found" or 404 on execution trigger

**Root Cause Options**:
1. Command ID is wrong (check catalog, name, version)
2. Command exists but is not released (still in draft state)
3. Command was deleted

**Fix**: 
```bash
# Check if command exists
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/commands/catalog:CommandName:1"
```

⚠️ If the command is in draft state, release it only after verifying it works correctly and only if the user explicitly requests it:
```bash
curl -s -X PUT -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/commands/catalog:CommandName:1/release"
```

---

### Execution stuck in RUNNING

**Possible Causes**:
1. `Delay:1` step is waiting (check `progressMessage` for "Waiting X minutes")
2. External HTTP call is timing out
3. Polling loop (`repeat`) hasn't met exit condition
4. Execution is paused or waiting for user input

```bash
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID"
```

**Fix Options**:
1. Wait for delay/polling to complete
2. Abort if stuck:
```bash
curl -s -X POST -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"type": "ABORT", "reason": "Aborting stuck execution"}' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/actions"
```

---

### HTTP executor returns unexpected status

**Common Issues**:
- 401/403: Authentication failed - check credentials, token expiry
- 404: Resource not found - verify URL and resource exists
- 429: Rate limited - add retry logic with backoff
- 500/502/503: Server error - retry with `autoRetry` configuration
- -1 or 0: No response / timeout - check connectivity, increase timeout

---

### Expression evaluation errors

**Symptoms**: Error mentions jq, expression, or shows `$(.something.output)` in error.

**Common Causes**:
1. Previous step failed, so output doesn't exist
2. JSON parsing failed (`toObject` on non-JSON string)
3. Null value in expression chain
4. Array index out of bounds

**Fix**: Add null checks with `// "default"` operator:
```
$(.step.output.body | toObject.field // "default")
```

---

## Debugging Workflow

### Step 1: Get Execution Status and Error Details
```bash
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID"
```

### Step 2: Check Execution Logs
```bash
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/logs?maxPageSize=20"
```

### Step 3: Check Input That Was Used
```bash
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/input"
```

### Step 4: Match Error to Pattern
Look up the error message in the Error Pattern Reference above.

### Step 5: Verify Command Definition
```bash
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" "https://$AUTOPI_HOSTNAME/api/v1/commands/$COMMAND_ID"
```

