---
name: autopi-debug
description: Debug and troubleshoot SAP Automation Pilot execution failures. Use when executions fail, need to investigate errors, check execution health (pass/fail summary), or understand error patterns. Provides error pattern matching with suggested fixes.
---

# SAP Automation Pilot Debugging & Troubleshooting

Debug failed Executions, check recent execution health, and investigate error patterns. Use this skill when things go wrong — not for normal operations.

## Quick Diagnostics

### Check Recent Executions Summary

```bash
# Standard mode (shows available commands on load)
source .env && source .claude/skills/autopi-debugger/scripts/autopi-debug-config.sh

# Quiet mode (suppress help banner - recommended for scripting)
source .env && export AUTOPI_DEBUG_QUIET=1 && source .claude/skills/autopi-debugger/scripts/autopi-debug-config.sh

# Quick summary of last N executions
autopi_debug_summary 10

# List only failed executions
autopi_debug_failures 10

# List only successful executions  
autopi_debug_successes 10
```

### Investigate a Specific Execution

```bash
source .env && source .claude/skills/autopi-debugger/scripts/autopi-debug-config.sh

EXEC_ID="your-execution-id"

# Full diagnostic report
autopi_debug_diagnose "$EXEC_ID"

# Just the error message
autopi_debug_error "$EXEC_ID"

# Execution logs
autopi_debug_logs "$EXEC_ID"
```

---

## Error Pattern Reference

### 🔴 "Parameter 'X' is required but not provided"

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

### 🔴 "Missing valid combination of input values for authentication"

**Full Error**: `"Missing valid combination of input values for authentication. Please select a valid option: 1) 'clientCert' for X509 2) 'user' & 'password' for Basic authentication"`

**Context**: Typically occurs with `email-sapcp:SendEmail:1` command.

**Root Cause**: The SMTP server requires authentication. Internal mail relays like `mail.sap.corp` that work from on-premise networks require authentication when accessed from cloud services (Automation Pilot runs in the cloud).

**Fix Options**:
1. Provide `user` and `password` for SMTP authentication
2. Provide `clientCert` for X509 certificate authentication
3. Use a different SMTP server that allows unauthenticated relay
4. Use SAP Alert Notification Service (ANS) instead of direct SMTP

---

### 🔴 "The following input keys can only have default values from input, because they are marked as sensitive"

**Context**: Occurs when deploying a command.

**Root Cause**: Sensitive input keys (like passwords) cannot have hardcoded `defaultValue`. This is a security feature.

**Fix**: Remove `defaultValue` from sensitive fields. Use `defaultValueFromInput` to reference an Input object instead, or leave no default.

**Wrong**:
```json
"password": {
  "type": "string",
  "sensitive": true,
  "defaultValue": "secret123"  // NOT ALLOWED
}
```

**Correct**:
```json
"password": {
  "type": "string", 
  "sensitive": true
  // No defaultValue for sensitive fields
}
```

---

### 🔴 "Command not found" or 404 on execution trigger

**Root Cause Options**:
1. Command ID is wrong (check catalog, name, version)
2. Command exists but is not released (still in draft state)
3. Command was deleted

**Fix**: 
```bash
# Check if command exists
autopi_api GET "/commands/catalog:CommandName:1"

# Release the command if it's a draft
curl -X PUT -u "$USER:$PASS" "https://$HOST/api/v1/commands/catalog:CommandName:1/release"
```

---

### 🔴 Execution stuck in RUNNING

**Possible Causes**:
1. `Delay:1` step is waiting (check `progressMessage` for "Waiting X minutes")
2. External HTTP call is timing out
3. Polling loop (`repeat`) hasn't met exit condition
4. Execution is paused or waiting for user input

**Diagnosis**:
```bash
# Check current state
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions/$EXEC_ID" | jq '{status, progressMessage, currentExecutorPath}'

# Check if waiting for input
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions/$EXEC_ID" | jq '.userChoice'
```

**Fix Options**:
1. Wait for delay/polling to complete
2. Abort if stuck: `curl -X POST -d '{"action":"abort"}' .../executions/$EXEC_ID/actions`

---

### 🔴 HTTP executor returns unexpected status

**Common Issues**:
- 401/403: Authentication failed - check credentials, token expiry
- 404: Resource not found - verify URL and resource exists
- 429: Rate limited - add retry logic with backoff
- 500/502/503: Server error - retry with `autoRetry` configuration
- -1 or 0: No response / timeout - check connectivity, increase timeout

---

### 🔴 Expression evaluation errors

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

### Step 1: Get Execution Status
```bash
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions/$EXEC_ID" | jq '{status, error, progressMessage}'
```

### Step 2: If FAILED, Get Error Details
```bash
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions/$EXEC_ID" | jq '.error'
```

### Step 3: Check Execution Logs
```bash
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions/$EXEC_ID/logs?maxPageSize=50" | jq '.logs'
```

### Step 4: Check Input That Was Used
```bash
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/executions/$EXEC_ID/input" | jq '.values'
```

### Step 5: Match Error to Pattern
Look up the error message in the Error Pattern Reference above.

### Step 6: Verify Command Definition
```bash
curl -s -u "$USER:$PASS" "https://$HOST/api/v1/commands/$COMMAND_ID" | jq '.configuration.executors'
```

---

## Known Issues & Environment Notes

### Canary Environment (canary.autopilot.cloud.sap)

- The "required parameter not provided" bug has been observed here
- May have different behavior than production environments
- Use for testing, not production workloads

### SMTP from Cloud

- Internal SAP mail relays require authentication from cloud
- `mail.sap.corp` does NOT work without credentials from Automation Pilot
- Consider using ANS (Alert Notification Service) for notifications instead

### Command Release Requirement

- Newly deployed commands are in DRAFT state
- Must call `/commands/{id}/release` before execution
- Check for `autopi:released` tag to verify release status

---

## Quick Reference Commands

```bash
# Load debug functions
source .env && source .claude/skills/autopi-debugger/scripts/autopi-debug-config.sh

# === DIAGNOSTICS ===
autopi_debug_summary 10              # Last 10 executions summary
autopi_debug_failures 5              # Last 5 failures
autopi_debug_diagnose $EXEC_ID       # Full diagnosis of one execution

# === RAW API ===
# Get execution
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID" | jq .

# Get error only
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID" | jq '.error'

# Get logs
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/logs" | jq '.logs'

# Abort stuck execution
curl -s -X POST -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"action":"abort"}' \
  "https://$AUTOPI_HOSTNAME/api/v1/executions/$EXEC_ID/actions"
```

---

## Lessons Learned Log

This section captures debugging insights discovered over time.

### 2026-05-20: Required Parameters API Bug

**Scenario**: Creating command `GenerateGuidAndEmail:1` with required inputs `smtpHost`, `smtpPort`, `recipientEmail`, `senderEmail`.

**Issue**: Triggering execution with valid JSON payload always returned `400: Parameter 'smtpHost' is required but not provided`.

**Investigation**:
- Verified JSON was valid
- Tried different content types
- Tried string vs number types
- Confirmed same payload structure works for commands with NO required inputs

**Resolution**: Hardcoded all values directly in the command executor inputs instead of using input parameters. Commands with `inputKeys: {}` execute successfully.

**Hypothesis**: Possible permission issue with the technical user, or canary environment bug.

### 2026-05-20: SMTP Authentication from Cloud

**Scenario**: Using `email-sapcp:SendEmail:1` with `mail.sap.corp` as host.

**Issue**: Execution failed with authentication error after successfully completing GUID generation and delay steps.

**Resolution**: `mail.sap.corp` requires authentication when accessed from cloud. Need SMTP credentials or use ANS instead.

### 2026-05-20: Quiet Mode Added

**Scenario**: Using `autopi_debug_summary` to check execution health.

**Issue**: The "Debug functions loaded" help text was noisy when just wanting quick results.

**Resolution**: Added `AUTOPI_DEBUG_QUIET=1` environment variable to suppress the help banner. Use for scripting or when you just want results without the preamble.
