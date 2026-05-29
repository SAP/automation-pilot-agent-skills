---
name: automation-pilot-executor-executescript
description: Master the ExecuteScript executor for Automation Pilot commands. Covers script execution parameters, Base64 encoding, timeout/exit code handling, stdin/parameters/environment patterns, and language-specific wrappers (Python, Node.js, PowerShell). Use when building commands that execute shell scripts or custom code.
version: 1.0.0
---

# ExecuteScript Executor Guide

The ExecuteScript executor (`scripts-sapcp:ExecuteScript:1/2`) runs shell scripts, Python, Node.js, and PowerShell in Automation Pilot.

---

## Version Differences (CRITICAL)

**Version 1** (`scripts-sapcp:ExecuteScript:1`): Requires Base64 for `script` and `stdin`
**Version 2** (`scripts-sapcp:ExecuteScript:2`): Plain text (recommended)

```json
// V1: Base64 required
{"script": "ZWNobyAiSGVsbG8i", "stdin": "$(.execution.input.stdin | toBase64)"}

// V2: Plain text
{"script": "echo \"Hello\"", "stdin": "$(.execution.input.stdin)"}
```

---

| Parameter | Type | Required | Default | Notes |
|-----------|------|----------|---------|-------|
| `script` | string | ✅ | - | V1: Base64 / V2: Plain text |
| `stdin` | string (sensitive) | ❌ | - | V1: Base64 / V2: Plain text. Always sensitive. Multiple values: `\n` separator |
| `parameters` | array | ❌ | `[]` | Passed as `$1`, `$2`, `$3`... in order |
| `environment` | object | ❌ | `{}` | Available as shell variables |
| `timeout` | number | ❌ | `30` | Range: 15-600s. Exit code `124` if exceeded |
| `successExitCodes` | array | ❌ | `["0"]` | Use `["x"]` to accept all codes |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success (default) |
| `124` | **Timeout exceeded** |
| `-1` | **Insufficient resources/quota** |
| `137` | Killed by system (SIGKILL) |
| Custom | Use `successExitCodes: ["0", "137"]` or `["x"]` (all codes) |

---

## Language Wrappers

**Note:** Language wrappers ALWAYS use Base64 for the script parameter:
```json
{"parameters": "$([.execution.input.script | toBase64] + (.execution.input.parameters // []))"}
```

### Python (ExecutePythonScript)

```json
{
  "execute": "scripts-sapcp:ExecuteScript:2",
  "input": {
    "parameters": "$([.execution.input.script | toBase64] + (.execution.input.parameters // []))",
    "script": "$([\"#!/usr/bin/env bash\", \"set -e\"] + 
      ($.execution.input.packages | map(\"pip install \\(.) -qq\")) + 
      [\"echo $1 | base64 --decode > script.py\", \"shift\", \"cat - | python3 script.py $@\"] 
      | join(\"\\n\"))",
    "stdin": "$(.execution.input.stdin)",
    "timeout": "$(.execution.input.timeout)"
  }
}
```

**Pattern:** Install packages → Decode Base64 script → Execute with Python3

### Node.js (ExecuteNodeJsScript)

```json
{
  "execute": "scripts-sapcp:ExecuteScript:2",
  "input": {
    "parameters": "$([.execution.input.script | toBase64] + (.execution.input.parameters // []))",
    "script": "#!/usr/bin/env bash\nset -e\nnpm install <packages>\necho $1 | base64 --decode > script.js\nshift\ncat - | node script.js $@",
    "timeout": "$(.execution.input.timeout)"
  }
}
```

### PowerShell (ExecutePowerShellScript)

```json
{
  "execute": "scripts-sapcp:ExecuteScript:2",
  "input": {
    "parameters": "$([.execution.input.script | toBase64] + (.execution.input.parameters // []))",
    "script": "#!/usr/bin/env bash\nset -e\npwsh -Command \"Install-Module -Name <module> -Force\"\necho $1 | base64 --decode > script.ps1\nshift\ncat - | pwsh script.ps1 $@"
  }
}
```

---

## Execution Workflow (Internal)

ExecuteScript uses a 5-step async workflow:

1. **CreateTenantSandbox**: Create isolated environment
2. **CreateScriptSecret:2**: Store script/stdin/params/environment
3. **StartScript:1**: Initiate execution
4. **PollScript:1**: Poll every 10s (max 2160 times = 6 hours) until status NOT_EQUALS "RUNNING"
5. **FinalizeScript:1**: Retrieve exit code and output

**AutoRetry:** Steps 2-4 retry on status < 100 or ≥ 500 (maxCount: 10, delay: 5s, INCREMENTAL)

---

## Output Handling

**Structure:**
```json
{
  "exitCode": 0,           // 124 = timeout, -1 = no resources
  "output": ["line1", ...]  // Limited to last 64 KB, trimmed
}
```

**64 KB Limit:** Large outputs trimmed from beginning (tail behavior)

---

## Examples

### Basic Bash

```json
{
  "execute": "scripts-sapcp:ExecuteScript:2",
  "input": {
    "script": "echo \"$1 $2\"",
    "parameters": "[\"Hello\", \"World\"]",
    "timeout": "30"
  },
  "alias": "runScript",
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

### Stdin (Sensitive Data)

```json
{
  "execute": "scripts-sapcp:ExecuteScript:2",
  "input": {
    "script": "read password; echo \"Received ${#password} chars\"",
    "stdin": "$(.execution.input.password)",
    "timeout": "30"
  },
  "alias": "processPassword",
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

### Python with Packages

```json
{
  "execute": "scripts-sapcp:ExecutePythonScript:1",
  "input": {
    "script": "import requests\\nprint(requests.__version__)",
    "packages": "[\"requests\"]",
    "timeout": "60"
  },
  "alias": "runPython",
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

---

## Common Anti-Patterns

| ❌ Wrong | ✅ Correct |
|---------|-----------|
| V1: `"script": "echo test"` | V1: `"script": "$(.execution.input.script \| toBase64)"` |
| V2: `"script": "$(.execution.input.script \| toBase64)"` | V2: `"script": "$(.execution.input.script)"` |
| `"stdin": {"sensitive": false}` | `"stdin": {"sensitive": true}` |
| `"timeout": 5` (< min 15) | `"timeout": "30"` |
| `"successExitCodes": "0"` | `"successExitCodes": "[\"0\"]"` |
| Parameters in wrong order | Match script's `$1 $2 $3` order |
