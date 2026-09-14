---
name: automation-pilot-executor-executescript
description: Master the ExecuteScript executor for Automation Pilot commands. Covers script execution parameters, Base64 encoding, timeout/exit code handling, stdin/parameters/environment patterns, and language-specific commands (Python, Node.js, PowerShell). Use when building commands that execute shell scripts or custom code.
---

# Script Execution Guide

SAP Automation Pilot provides dedicated commands for each script language. Choose the right command for your language — do not use `ExecuteScript` as a bash wrapper to run Python, Node.js, or PowerShell.

## Command Overview

| Command | ID | Use for |
|---------|-----|---------|
| `ExecuteScript:1` | `scripts-sapcp:ExecuteScript:1` | Bash (legacy, Base64 script) |
| `ExecuteScript:2` | `scripts-sapcp:ExecuteScript:2` | Bash (recommended, plain text) |
| `ExecutePythonScript:1` | `scripts-sapcp:ExecutePythonScript:1` | Python scripts |
| `ExecuteNodeJsScript:1` | `scripts-sapcp:ExecuteNodeJsScript:1` | Node.js scripts |
| `ExecutePowerShellScript:1` | `scripts-sapcp:ExecutePowerShellScript:1` | PowerShell scripts |

Each command also has a `Sensitive*` variant (e.g. `SensitiveExecutePythonScript:1`) that marks the output as sensitive so it is never logged.

---

## ExecuteScript (Bash)

### Version Differences (CRITICAL)

**Version 1** (`scripts-sapcp:ExecuteScript:1`): `script` and `stdin` must be Base64-encoded  
**Version 2** (`scripts-sapcp:ExecuteScript:2`): Plain text — always prefer this

| Parameter | Type | Required | Default | Notes |
|-----------|------|----------|---------|-------|
| `script` | string | ✅ | - | V1: Base64 encoded / V2: Plain text |
| `stdin` | string | ❌ | - | Sensitive. V1: Base64 / V2: Plain text. Multiple values: `\n` separator |
| `parameters` | array | ❌ | `[]` | Passed as `$1`, `$2`, `$3`... in order |
| `environment` | object | ❌ | `{}` | Available as shell environment variables |
| `timeout` | number | ❌ | `30` | Range: 15–600s |
| `successExitCodes` | array | ❌ | `["0"]` | Use `["x"]` to accept any exit code |

**Output keys:** `exitCode` (number), `result` (array of output lines, last 64 KB)

### Example

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

---

## ExecutePythonScript

`scripts-sapcp:ExecutePythonScript:1` — executes a Python script directly. The `script` field takes plain Python source code.

| Parameter | Type | Required | Default | Notes |
|-----------|------|----------|---------|-------|
| `script` | string | ✅ | - | Plain Python source code |
| `packages` | array | ❌ | `[]` | pip packages installed before execution |
| `parameters` | array | ❌ | `[]` | Passed as positional args to the script |
| `stdin` | string | ❌ | - | Sensitive. Passed to script's standard input |
| `environment` | object | ❌ | `{}` | Environment variables |
| `timeout` | number | ❌ | `60` | Range: 15–600s |
| `successExitCodes` | array | ❌ | `["0"]` | |

**Output keys:** `exitCode` (number), `result` (array of output lines)

### Example

```json
{
  "execute": "scripts-sapcp:ExecutePythonScript:1",
  "input": {
    "script": "import requests\nprint(requests.__version__)",
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

## ExecuteNodeJsScript

`scripts-sapcp:ExecuteNodeJsScript:1` — executes a Node.js script directly. The `script` field takes plain JavaScript source code.

| Parameter | Type | Required | Default | Notes |
|-----------|------|----------|---------|-------|
| `script` | string | ✅ | - | Plain JavaScript source code |
| `packages` | array | ❌ | `[]` | npm packages installed before execution |
| `npmrc` | string | ❌ | - | Sensitive. `.npmrc` content for private registry auth |
| `parameters` | array | ❌ | `[]` | Passed as positional args to the script |
| `stdin` | string | ❌ | - | Sensitive. Passed to script's standard input |
| `environment` | object | ❌ | `{}` | Environment variables |
| `timeout` | number | ❌ | `30` | Range: 15–600s |
| `successExitCodes` | array | ❌ | `["0"]` | |

**Output keys:** `exitCode` (number), `result` (array of output lines)

### Example

```json
{
  "execute": "scripts-sapcp:ExecuteNodeJsScript:1",
  "input": {
    "script": "const axios = require('axios');\nconsole.log(axios.VERSION);",
    "packages": "[\"axios\"]",
    "timeout": "30"
  },
  "alias": "runNode",
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

## ExecutePowerShellScript

`scripts-sapcp:ExecutePowerShellScript:1` — executes a PowerShell script directly. The `script` field takes plain PowerShell source code.

| Parameter | Type | Required | Default | Notes |
|-----------|------|----------|---------|-------|
| `script` | string | ✅ | - | Plain PowerShell source code |
| `modules` | array | ❌ | `[]` | PowerShell modules installed before execution |
| `parameters` | array | ❌ | `[]` | Passed as positional args to the script |
| `stdin` | string | ❌ | - | Sensitive. Passed to script's standard input |
| `environment` | object | ❌ | `{}` | Environment variables |
| `timeout` | number | ❌ | `60` | Range: 15–600s |
| `successExitCodes` | array | ❌ | `["0"]` | |

**Output keys:** `exitCode` (number), `result` (array of output lines)

### Example

```json
{
  "execute": "scripts-sapcp:ExecutePowerShellScript:1",
  "input": {
    "script": "Write-Output 'Hello from PowerShell'",
    "timeout": "60"
  },
  "alias": "runPowerShell",
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

## Sensitive Variants

Each command has a `Sensitive*` counterpart that marks the `result` output as sensitive (never logged or displayed):

| Standard | Sensitive variant |
|----------|-------------------|
| `scripts-sapcp:ExecuteScript:2` | `scripts-sapcp:SensitiveExecuteScript:2` |
| `scripts-sapcp:ExecutePythonScript:1` | `scripts-sapcp:SensitiveExecutePythonScript:1` |
| `scripts-sapcp:ExecuteNodeJsScript:1` | `scripts-sapcp:SensitiveExecuteNodeJsScript:1` |
| `scripts-sapcp:ExecutePowerShellScript:1` | `scripts-sapcp:SensitiveExecutePowerShellScript:1` |

Use sensitive variants when the script output contains credentials, tokens, or other secrets.

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success (default) |
| `-1` | Insufficient resources/quota |
| `137` | Killed by system (SIGKILL) |
| Custom | Use `successExitCodes: ["0", "1"]` or `["x"]` to accept any code |

Note: `ExecuteScript` returns exit code `124` on timeout. The dedicated language commands enforce the timeout differently — they stop execution but the exit code may vary.

---

## Common Anti-Patterns

| ❌ Wrong | ✅ Correct |
|---------|-----------|
| Using `ExecuteScript:2` with a bash wrapper to run Python | Use `ExecutePythonScript:1` directly |
| Using `ExecuteScript:2` with a bash wrapper to run Node.js | Use `ExecuteNodeJsScript:1` directly |
| V1: plain text `"script": "echo test"` (not encoded) | V1: hardcode literal Base64, or encode at runtime: `"script": "$(.execution.input.script \| toBase64)"` |
| V2: `"script": "$(.execution.input.script \| toBase64)"` | V2: `"script": "$(.execution.input.script)"` |
| `"timeout": 5` (below minimum of 15) | `"timeout": "30"` |
| `"successExitCodes": "0"` (string, not array) | `"successExitCodes": "[\"0\"]"` |
| Using `output` key on script result | Use `result` key — script commands output to `result`, not `output` |
| Standard command when output contains secrets | Use the `Sensitive*` variant |
