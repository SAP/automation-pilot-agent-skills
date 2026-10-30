# CLI recipes

Load this for concrete request examples; the skill entrypoint owns connection setup and
shared operating rules. Run the binary's help for exact flags.

## Contents

- [Common examples](#common-examples)
- [Request preview](#request-preview)
- [Conditional updates and lifecycle changes](#conditional-updates-and-lifecycle-changes)
- [Wait for a run and inspect its outcome](#wait-for-a-run-and-inspect-its-outcome)
- [Execution actions and partial bulk results](#execution-actions-and-partial-bulk-results)
- [History, summaries, and executor logs](#history-summaries-and-executor-logs)
- [Events and webhooks](#events-and-webhooks)
- [Input patches and bulk import](#input-patches-and-bulk-import)
- [Catalog deletion](#catalog-deletion)

## Common examples

```bash
# Verify connectivity and permissions (no secrets printed)
autopi connection info

# Discover
autopi catalogs list --provided
autopi commands list --catalog applm-sapcp --include-issues
autopi commands get applm-sapcp:RestartCfApp:1
autopi commands contract applm-sapcp:RestartCfApp:1   # contract only (no configuration)

# Deploy content (upsert = GET then PUT or POST; stays in DRAFT)
autopi commands deploy -f MyCommand.command.json
autopi commands export applm-sapcp:RestartCfApp:1 -o RestartCfApp.command.json

# Rotate from an authorized, protected payload file; do not put secrets in argv
autopi inputs patch my-cat:JiraCredentials:1 -f - < /secure/path/rotation.json

# Preview offline, then trigger only within the user's authorized scope
autopi executions trigger --command-id my-cat:MyCommand:1 --input region=cf-eu10 --print-request
autopi executions summary
autopi executions logs <execution-id> --page 0 --max-page-size 20

# Anything not wrapped — the escape hatch (auth + base URL injected)
autopi api GET 'commands?catalog=applm-sapcp'
```

## Request preview

`--print-request` renders an offline request preview **without sending**
and **without needing credentials**. Authentication,
known credential fields, sensitive defaults, and entire `values`/`inputValues`/`outputValues` blocks are redacted;
non-JSON bodies are omitted. The preview is deliberately not a verbatim payload. Inspect the
non-secret definition locally for wire validation. Deploy previews show conditional GET/PUT/POST
branches, without deciding whether the target exists.

```bash
autopi inputs patch my-cat:JiraCredentials:1 --data '{"values":{"password":"x"}}' --print-request
# PATCH https://emea.autopilot.cloud.sap/api/v1/inputs/my-cat:JiraCredentials:1
# Authorization: Bearer ***
# Content-Type: application/merge-patch+json
# { "values": "***" }
```

## Conditional updates and lifecycle changes

Get the body and ETag from the same response. Prepare the replacement from that body,
then preserve the quoted ETag exactly. A 412 requires another read and reconciliation.

```bash
autopi scheduled-executions get schedule-id --response-envelope > schedule-response.json
ETAG=$(jq -r '.headers.ETag // empty' schedule-response.json)
test -n "$ETAG" || exit 1
jq '.body | del(.id, .state) | .enabled = false' schedule-response.json > disabled.schedule.json
autopi lint --kind scheduled-execution --mode api disabled.schedule.json
autopi scheduled-executions update schedule-id -f disabled.schedule.json --if-match "$ETAG"

# Convenience toggles perform that read-modify-conditional PUT themselves
autopi scheduled-executions disable schedule-id
autopi scheduled-executions enable schedule-id

# Inspect current state before an authorized lifecycle change
autopi commands get my-cat:OldCommand:1 --show-etag
autopi commands deprecate my-cat:OldCommand:1 --replacement my-cat:NewCommand:1 --if-match '"returned-etag"'
autopi commands get my-cat:OldCommand:1 --include-issues
```

Command/Input deploy reuses the ETag from its existence read when available. Direct
update, patch, delete, and lifecycle verbs accept `--if-match` without adding a GET.
Input read-back omits values; construct a full Input replacement only when all values
are available from an authorized source.

## Wait for a run and inspect its outcome

```bash
autopi executions trigger --command-id my-cat:MyCommand:1 --wait-seconds 60 --correlation-id change-42 --response-envelope
autopi executions get execution-id
autopi executions output execution-id
```

The default timeout extends to 65 seconds for a 60-second wait. An explicitly supplied
timeout must exceed the wait. HTTP 200 means completion during the window; check the
execution status for success. HTTP 202 means acceptance. After an HTTP timeout, inspect
history using the correlation ID in returned metadata before deciding whether to submit
again. Mutations are never retried automatically.

## Execution actions and partial bulk results

```bash
autopi executions get execution-id --show-etag
autopi executions action execution-id --type CHOOSE --reason 'approved option' --value proceed --if-match '"returned-etag"'
autopi executions bulk-action --execution-id first-id --execution-id second-id --type ABORT --reason 'superseded change'
# A protected file/stdin body works for complex actions or bulk requests
autopi executions action execution-id -f action.json
autopi executions bulk-action -f bulk-action.json
```

RESET, CHOOSE, and SKIP need `value`. SKIP's value is a serialized JSON object with
`executorPath` and optional string-valued `outputValues` and string `target`. Use a
protected file when values could contain secrets. State-specific action eligibility
remains a service check. Internal actions are unavailable; use RESET in place of the
deprecated RESET_AND_RETRY.

Bulk requests contain `executionIds`, `type`, and `reason`; they allow 1–50 IDs and
PAUSE/RESUME/ABORT/RETRY. HTTP 200 contains independent ordered results. Exit 7 means
an item failed. Inspect those results and current state before acting on failed IDs.
Duplicate IDs retain one result per occurrence. Missing/reordered results cause an
error and remain visible for diagnosis; successful entries are never resubmitted by
the CLI.

## History, summaries, and executor logs

```bash
autopi executions list --originator-id operator --filter-mode contains --started-after 24h --limit 100
autopi executions summary --started-after 48h --started-before 24h
autopi executions logs execution-id --page 0 --max-page-size 20
autopi executions logs execution-id --executor step-path --messages ERROR --messages WARNING
autopi executions list --status FAILED --started-after 24h --read-retries 2
```

Execution listing has no continuation cursor and returns at most 1000 results; a list
is not proof of complete history. Log pages contain executor links, so fetch selected
paths with `--executor`. Do not follow arbitrary absolute response links. An explicit
`--max-page-size 0` returns an empty page. Read retries apply only to GET/HEAD 429/503,
honor Retry-After when supplied, and share one total timeout.

## Events and webhooks

```bash
autopi triggers generic-event --command-reference my-cat:MyCommand:1 --input-reference my-cat:Config:1 --tag change=42 --correlation-id event-42
autopi triggers generic-event --command-reference my-cat:MyCommand:1 --map-event-to-input-key event -f event.txt --content-type text/plain
autopi triggers ans-event --command-reference my-cat:MyCommand:1 --input-reference my-cat:Config:1 --sub-account affected-id --disable-resource-properties-mapping=false --disable-region-properties-mapping=true -f ans-event.json
autopi lint --kind webhook --mode api Event.webhook.json
autopi webhooks validate -f Event.webhook.json --metadata contracts.json
autopi webhooks create -f Event.webhook.json
autopi webhooks trigger webhook-id -f event.txt --content-type text/plain --response-envelope
```

Generic events and webhook triggers allow an absent body or arbitrary bytes. ANS
requires a structured event. Repeat `--input-reference` and `--tag`; use the current
singular tag parameter. ANS also accepts resource group/name/instance overrides.
Webhook names are required at creation and immutable afterward. Read the
[webhook reference](webhooks.md) for the definition and offline metadata shape.
Without `--metadata`, webhook validation reads command/Input contracts from the API.
HTTP 201 from triggering a webhook supplies an execution; inspect its state afterward.

## Input patches and bulk import

```bash
# A metadata patch is a partial Input update, not just value rotation
autopi inputs patch my-cat:Config:1 --data '{"description":"Production configuration"}' --if-match '"returned-etag"'
autopi lint --kind input-patch --mode api rotation.json
autopi inputs patch my-cat:Config:1 -f - < /secure/path/rotation.json

# Build an import envelope around complete definitions
jq --slurpfile commands MyCommand.command.json '{commands: $commands, overwriteExisting: true}' > commands-import.json
autopi lint --kind bulk-commands --mode api commands-import.json
autopi commands bulk-create -f commands-import.json
autopi inputs bulk-create -f - < /secure/path/inputs-import.json
```

Input PATCH accepts a nonempty object of `description`, `keys`, `values`, and `tags`.
Omitted fields are preserved. Null key descriptors delete keys; non-null descriptors
need corresponding string values. Values cannot be null. Check visible read-back
after a patch; a successful write is the evidence for hidden value changes.

Import envelopes use `commands` or `inputs` arrays of full definitions. Both support
`overwriteExisting` or `skipExisting`; choose one behavior deliberately. Input imports
also support `keepExistingSensitiveValues: true` to preserve existing secrets. The
contract does not specify precedence when both existing-resource controls are true,
transactional behavior, or all interactions with missing values. New full Inputs still
require `values`. Use protected files/stdin for secret-bearing imports. The deprecated
`released` property/query is not recommended; use current lifecycle/tag workflows.

## Catalog deletion

```bash
autopi catalogs delete empty-catalog-id
# Within an explicitly authorized cascade-deletion scope
autopi catalogs delete catalog-id --force
```

Default deletion requires an empty catalog and returns 204. Force deletion also
deletes its Commands and Inputs and returns their counts with HTTP 200. The
authenticated account supplies originator identity. Catalogs do not support If-Match.

