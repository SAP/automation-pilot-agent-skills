# Triggering Executions

Read this before creating an execution. It defines the request shape, input encoding, and safeguards against accidental duplicate runs.

## Request shape

Pass this object as `autopi executions trigger`'s `execution` body:

```json
{
  "commandId": "welcome-<<<TENANT_ID>>>:<Command>:<version>",
  "inputReferences": ["welcome-<<<TENANT_ID>>>:<Input>:<version>"],
  "inputValues": { "key": "value" },
  "tags": { "environment": "production" }
}
```

`commandId` is required. The other fields are optional. Use the exact command version intended by the user.
The optional correlationId is exposed as --correlation-id. For a short synchronous
wait, use --wait-seconds N (0–60). The default HTTP deadline extends when necessary;
an explicit --timeout must exceed the server wait. HTTP 200 means completion within
the wait window; inspect status and optional outputValues. HTTP 202 returns an accepted
execution ID for later inspection. A timeout after submission remains ambiguous.

For events, --command-reference is a query parameter; the body is the event itself.
Both trigger wrappers accept repeated --input-reference/--tag, --correlation-id, and
--map-event-to-input-key. Generic bodies are optional and may use --content-type for
arbitrary bytes. ANS requires an event and adds resource/subaccount overrides and
resource/region mapping switches. Use the singular tag query parameter; tags is deprecated.

## Input binding

- `inputValues` is a flat object of command input key to execution-specific value.
- `inputReferences` is a flat list of reusable Input IDs. Resolve each ID with `autopi inputs get` and confirm that its key contract supplies keys in the command contract.
- Ensure the direct values and referenced Inputs together supply every required key. Remove overlaps that make the effective value ambiguous instead of guessing precedence.
- Put credentials and other secret-bearing values in a sensitive reusable Input. Keep resolved sensitive values out of conversation output and logs.

Every `inputValues` value travels as a string regardless of its declared command type. Stringify numbers and booleans, and JSON-encode arrays and objects into strings:

```json
{
  "inputValues": {
    "appName": "my-app",
    "instances": "3",
    "enabled": "true",
    "regions": "[\"cf-eu10\",\"cf-us10\"]",
    "config": "{\"maxRetries\":5}"
  }
}
```

Use the shared [CLI wire rules](../../automation-pilot-cli/SKILL.md#values-are-always-strings-on-the-wire).
The builder accepts repeatable `--input-reference ID`; `--input-json` must contain strings.
Prebuilt files cannot be mixed with builder modifiers. Malformed values fail command validation.
A value read through `autopi executions input <id>` has the same string representation.

## Trigger safety

The request must authorize the resolved command, target, and scope. Clarify a mismatch or ambiguity before triggering the run.

If the trigger call times out or fails after sending the request, the outcome may be ambiguous. Search recent executions with the resolved `--command-id`, a narrow started-time window, and any unique regular tag that was attached. If one matching run exists, use its ID. If the evidence remains ambiguous, report that state instead of issuing a replacement run.
