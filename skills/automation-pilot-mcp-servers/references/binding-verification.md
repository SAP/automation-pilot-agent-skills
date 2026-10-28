# MCP binding verification

Run `autopi mcp-servers validate` before publishing or remapping a tool, and after
obtaining the client schema. Validation lives in the Go CLI and has no runtime dependencies.
It performs reads, never publishes a definition or triggers a command.

## Live metadata

```sh
autopi mcp-servers validate -f server.mcp.json
autopi mcp-servers validate -f server.mcp.json --client-schema tools-list.json
```

The CLI inherits `AUTOMATION_PILOT_HOSTNAME` and `AUTOMATION_PILOT_TOKEN`. Live validation
reads each command/Input contract once, discards stored values, and prints only findings
and the expected caller keys. Missing contracts stop validation; they never trigger a write.

## Offline metadata

Use `--metadata metadata.json` to perform no API calls. Save command contracts and Input
metadata, keyed by exact ID:

```json
{
  "commands": {
    "ops-TENANT:FindResource:1": {
      "inputKeys": {
        "resourceName": {"type": "string", "required": true, "sensitive": false},
        "apiConfig": {"type": "object", "required": true, "sensitive": false},
        "apiToken": {"type": "string", "required": true, "sensitive": true}
      }
    }
  },
  "inputs": {
    "ops-TENANT:ApiConfiguration:1": {
      "keys": {
        "apiConfig": {"type": "object", "sensitive": false},
        "apiToken": {"type": "string", "sensitive": true}
      }
    }
  }
}
```

Include metadata only; omit `values`, defaults containing secrets, and raw execution input.
CLI validation accepts a plain `tools/list` result, its JSON-RPC envelope, or the evaluation
runner's evidence envelope with `body` and `exitCode`.

## Interpretation

Exit 0 means there are no findings; exit 6 means lint findings were reported, including
MEDIUM heuristics. File/argument failures use exit 1; live reads retain the CLI's config,
network, and HTTP exit codes. Diagnostics name fields and IDs without printing supplied values. MEDIUM findings
identify unused Input keys and underspecified collection items.

The expected caller schema includes every backing-command key not supplied by an Input,
including optional parameters. A required key with a command default can be omitted by the
caller. Inspect the emitted schema rather than assuming a default makes a field disappear.
Extra Input keys are ignored for injection; they do not create caller parameters. Reject
overlapping matched keys instead of guessing precedence.

This checks metadata compatibility and the visible surface, not credential validity,
stored-value presence, OAuth behavior, or runtime success. An Input binding is proved at
runtime only by an authorized representative call and its execution result.
