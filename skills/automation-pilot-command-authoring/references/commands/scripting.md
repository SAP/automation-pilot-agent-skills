# Script Execution

Run a shell, Python, Node.js, or PowerShell script inside a command step. Apply the authoring
skill's **Native commands first** rule before selecting a runtime. This reference supplies
executor contracts, encoding, and runtime constraints for a step that needs a script.

Resolve the executor contract live before use (`autopi commands contract`). Field tables are
a 2026-10-14 snapshot, not proof of the target tenant's contract.

## Contents

- [When to use](#when-to-use)
- [Executor IDs and variants](#executor-ids--variants)
- [Shared and runtime-specific inputs](#key-inputs-shared)
- [Outputs](#outputs--how-to-read-them)
- [Minimal snippets](#minimal-snippets)
- [Gotchas](#gotchas)

## When to use

- Arbitrary shell logic → `scripts-sapcp:ExecuteScript:2` (bash, plain-text script).
- Python, with pip packages → `ExecutePythonScript:1`.
- Node.js, with npm packages / a private registry → `ExecuteNodeJsScript:1`.
- PowerShell → `ExecutePowerShellScript:1`.
- Script output can contain secrets → the matching **`Sensitive*`** variant.

## Executor IDs & variants

| Executor | Language | Adds over bash |
|----------|----------|----------------|
| `scripts-sapcp:ExecuteScript:2` | bash (plain text) | — base executor |
| `scripts-sapcp:ExecutePythonScript:1` | Python | `packages` (pip) |
| `scripts-sapcp:ExecuteNodeJsScript:1` | Node.js | `packages` (npm), `npmrc` (sensitive) |
| `scripts-sapcp:ExecutePowerShellScript:1` | PowerShell | — |
| `scripts-sapcp:SensitiveExecuteScript:2` and `Sensitive*` of each above | — | output marked sensitive |

**Avoid `ExecuteScript:1`.** It is the legacy Base64 variant. Use `:2` (plain-text `script`).

## Key inputs (shared)

Required: **`script`**.

| Field | Default | Notes |
|-------|---------|-------|
| `parameters` | `[]` | Positional args passed to the script, in order (array). For Python/Node the first parameter is always the script file. |
| `stdin` | — | **Sensitive**; passed to the script's standard input. Separate multiple values with `\n`. |
| `environment` | `{}` | Env vars available during execution (object). |
| `timeout` | `30` (Python `60`) | **15–600 seconds.** On timeout the script is killed with exit code `124`. |
| `successExitCodes` | `["0"]` | Exit codes treated as success. `["x"]` = every code is success. |

### Python-only
- `packages` (`[]`): pip packages installed before the run.

### Node.js-only
- `packages` (`[]`): npm packages.
- `npmrc`: **sensitive** `.npmrc` content (private registry URLs/tokens); never persisted or logged.

## Outputs & how to read them

| Executor | Output keys |
|----------|-------------|
| `ExecuteScript:2` | `output` (array, last 64 KB), `exitCode` (number) |
| `ExecutePythonScript:1` / `ExecuteNodeJsScript:1` | `result` (array), `exitCode` (number) |

Reach them as `$(.alias.output.exitCode)` and `$(.alias.output.output)` / `$(.alias.output.result)`.
`exitCode` is `124` on timeout and `-1` when there were not enough resources to run.

## Minimal snippets

Bash:

```json
{
  "execute": "scripts-sapcp:ExecuteScript:2",
  "alias": "run",
  "input": {
    "script": "$(.execution.input.script)",
    "parameters": "$(.execution.input.parameters)",
    "timeout": "60"
  },
  "description": null, "progressMessage": null,
  "initialDelay": null, "pause": null,
  "when": null, "validate": null, "autoRetry": null, "repeat": null,
  "errorMessages": []
}
```

Python with packages:

```json
{
  "execute": "scripts-sapcp:ExecutePythonScript:1",
  "alias": "run",
  "input": {
    "script": "$(.execution.input.script)",
    "packages": "[\"requests\",\"pyyaml\"]",
    "parameters": "$(.execution.input.parameters)",
    "timeout": "120"
  },
  "description": null, "progressMessage": null, "initialDelay": null, "pause": null,
  "when": null, "validate": null, "autoRetry": null, "repeat": null, "errorMessages": []
}
```

## Gotchas

- `timeout` outside 15–600 is rejected; a long package install plus run must fit the budget.
- Use the `Sensitive*` variant only when the **output** can carry secrets. Sensitive `stdin`,
  `npmrc`, or env values alone do not require it; those inputs are already protected and the
  ordinary executor keeps a non-sensitive output visible.
- Output is capped at the last 64 KB, so have the script print only what the caller needs.
