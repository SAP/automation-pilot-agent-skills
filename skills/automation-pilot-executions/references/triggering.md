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

Malformed values fail command validation. A value read through `autopi executions input <id>` has the same string representation.

## Trigger safety

The request must authorize the resolved command, target, and scope. Clarify a mismatch or ambiguity before triggering the run.

If the trigger call times out or fails after sending the request, the outcome may be ambiguous. Search recent executions with the resolved `--command-id`, a narrow started-time window, and any unique regular tag that was attached. If one matching run exists, use its ID. If the evidence remains ambiguous, report that state instead of issuing a replacement run.
