# Webhook operations

Read this when authoring, changing, inspecting, deleting, or triggering a webhook.
Administration needs Write; reading contracts needs Read; triggering needs Execute.
Use the shared CLI setup and operating rules.

## Definition and bindings

```json
{
  "name": "HandleEvent",
  "type": "GENERIC",
  "commandId": "my-cat:HandleEvent:1",
  "inputReferences": ["my-cat:Configuration:1"],
  "eventKey": "event",
  "inputValues": {"region": "cf-eu10"},
  "enabled": true
}
```

Name is required, alphanumeric, at most 64 characters, and immutable after creation.
Type is GENERIC or ALERT_NOTIFICATION. Optional fields include description, tags,
correlationId, Input references, direct string values, eventKey, and enabled. The server
assigns the ID.

Read the exact command contract and referenced Input key contracts. Bind sensitive keys
through sensitive reusable Inputs; inline values cannot target sensitive command keys.
Account for every required command key and avoid overlapping Input/direct/event bindings.
Direct wire strings must decode to the command key's declared type; validation checks
numbers, true/false booleans, JSON arrays, and JSON objects without printing their values.
The eventKey must name a compatible command input; its value comes from the trigger event.

Validate the definition and then its bindings:

```bash
autopi lint --kind webhook --mode api webhook.json
autopi webhooks validate -f webhook.json
# Offline contract-map alternative; contains metadata only, never stored values
autopi webhooks validate -f webhook.json --metadata contracts.json
```

The metadata file contains commands and inputs maps keyed by exact IDs, with command
inputKeys and Input keys descriptors. Validation emits field findings and never values.

```json
{
  "commands": {
    "my-cat:HandleEvent:1": {
      "inputKeys": {
        "event": {"type": "string", "required": true, "sensitive": false},
        "region": {"type": "string", "required": true, "sensitive": false},
        "password": {"type": "string", "required": true, "sensitive": true}
      }
    }
  },
  "inputs": {
    "my-cat:Configuration:1": {
      "keys": {"password": {"type": "string", "sensitive": true}}
    }
  }
}
```

Completion: schema checks pass and every required key has one compatible binding.

## Create and update

Call create once within the user's scope, capture the ID, and get it back to verify the
command, type, name, bindings, enabled state, and tags. For an update, read the complete
definition and ETag from one `get --response-envelope` response. Preserve the name and
untouched fields, then send the full writable object with that ETag. On 412, re-read
and reconcile the requested change.

```bash
autopi webhooks create -f webhook.json
autopi webhooks get webhook-id
autopi webhooks get webhook-id --response-envelope > webhook-response.json
ETAG=$(jq -r '.headers.ETag // empty' webhook-response.json)
test -n "$ETAG" || exit 1
# Example requested update: disable while preserving the current definition
jq '.body | del(.id) | .enabled = false' webhook-response.json > disabled.webhook.json
autopi lint --kind webhook --mode api disabled.webhook.json
autopi webhooks update webhook-id -f disabled.webhook.json --if-match "$ETAG"
```

## Trigger and delete

GENERIC events may be absent or arbitrary bytes; choose the corresponding media type.
ALERT_NOTIFICATION webhooks expect an appropriate ANS event. Triggering runs the command
for real; capture the returned execution ID and inspect its state/output when requested.
A timeout after submission is ambiguous and does not authorize a replacement run.

```bash
autopi webhooks trigger webhook-id -f event.txt --content-type text/plain
autopi webhooks trigger webhook-id
autopi executions get execution-id
autopi webhooks delete webhook-id --if-match ETAG
```

Delete only the settled target within the requested deletion scope; verify 404 afterward.
Disabling uses update with enabled=false and preserves the definition.
