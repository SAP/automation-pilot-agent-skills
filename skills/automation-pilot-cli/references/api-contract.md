# API contracts and service corrections

Read this for request validation, permission failures, unusual response outcomes, or
differences between the API and authored-content policy. Run the shipped CLI's help
for available flags.

## Contents

- [Provenance and maintenance](#provenance-and-maintenance)
- [Authentication and permissions](#authentication-and-permissions)
- [Limits and response outcomes](#limits-and-response-outcomes)
- [Wire values](#wire-values)
- [Response handling](#response-handling)
- [CLI transport and deployment](#cli-transport-and-deployment)
- [Reviewed corrections and additional service rules](#reviewed-corrections-and-additional-service-rules)
- [API validation versus authoring policy](#api-validation-versus-authoring-policy)
- [Conflicts and request quotas](#conflicts-and-request-quotas)

## Provenance and maintenance

The CLI embeds a reviewed snapshot of ContentAPI.json, ExecutionsAPI.json, and
ScheduledExecutionsAPI.json from the autopi-services repository, reviewed 2026-10-28.
It covers 51 operations and includes the upstream revision and SHA-256 digests of
the Swagger/schema files. External schema refs are resolved at build time.
`ci/cli/contracts/snapshot.py SWAGGER_DIR [--check]` updates/checks that snapshot;
changes require reviewing these service corrections.

`ci/cli/contracts/helpdocs.py [--check]` generates/checks the command reference
against every shipped resource/verb's help. CI exercises all 51 operations through
in-memory transports and checks command-reference and binary-source parity.

MCP, connection, and tenant APIs are outside these three definitions. Raw `api`
bypasses the pinned request validator while retaining authentication and Input response
protection. Use it for evolving contracts supported by current service evidence.

## Authentication and permissions

The CLI supports only Bearer authentication through the service's API-token chain.
Connection setup is defined in the [CLI entrypoint](../SKILL.md#run-the-bundled-cli).
The supplied Swagger authentication declaration does not describe the supported
Bearer path.

| Operation | Required scope |
|---|---|
| Read catalogs, commands, Inputs, webhooks, executions, schedules | Read |
| Content create/update/delete/lifecycle/bulk operations | Write |
| Trigger executions, generic/ANS events, or webhooks | Execute |
| Request single/bulk execution actions | Execute |
| Delete completed execution history | Write |
| Create/update/delete/enable/disable schedules | Execute |
| Verify webhook command/Input bindings | Read |

Source: ApiTokenSecurityConfigurator.java:114,
and the operation-specific Swagger permissions. Credential configuration is defined in
the [CLI entrypoint](../SKILL.md).

## Limits and response outcomes

| Request | Limits or behavior |
|---|---|
| Execution list | limit 1–1000; default server limit 1000; no continuation cursor specified |
| Identifier filters | commandId/executionId/originatorId: at most 100 each, IDs at most 256 characters |
| Status/tag filters | at most 8 statuses and 128 tags |
| Identifier matching | exact by default; contains is case-insensitive partial matching |
| Log pages | page 0–10000; maxPageSize 0–20; omitted size uses server default 10 |
| Explicit log size zero | Returns an empty page; it does not mean default/unlimited |
| Executor log messages | at most 20 message types, at most 20 characters each |
| Event Inputs/tags | at most 64 Input references and 128 tags; repeated query entries |
| Trigger wait | Prefer: wait=N; 0–60 seconds; explicit timeout must exceed N |
| Bulk execution actions | 1–50 IDs; reason required; PAUSE/RESUME/ABORT/RETRY only |
| Schedule deadline | 5–59 minutes, and no longer than the shortest cyclic minute window |

HTTP 200 from a waited trigger means the execution completed within the wait window.
Inspect `status` to determine the execution's outcome; it can include `outputValues`.
HTTP 202 from a trigger or action means acceptance; verify the resulting state.
Webhook trigger returns HTTP 201 with an execution, whose completion must be inspected.
HTTP 204 has no response body; `--response-envelope` can expose its status.

Bulk actions return HTTP 200 with independent ordered `results`. A duplicate ID gets
one result entry per occurrence, while the action is applied once per unique execution.
Exit 7 means at least one item failed; inspect successes and failures before requesting
recovery. There is no cross-batch transaction or automatic replay.

Log pages contain links/paths for executors, not the log messages. Fetch a selected
path with `logs --executor PATH [--messages TYPE]`; do not blindly follow absolute
response links to a different host.

## Wire values

Stored values, inputValues, and key defaults use JSON strings regardless of declared
type. This applies to command/Input definitions, execution requests, and schedules.

| Declared type | Wire value |
|---|---|
| string | `"my-app"` |
| number | `"42"` |
| boolean | `"true"` |
| array | `"[\"a\",\"b\"]"` |
| object | `"{\"url\":\"https://example.invalid\"}"` |

Convert within expressions using toNumber, toObject, or toArray when a logical type
is needed. Read-back permits omitted/null hidden values and legacy numeric outputs;
those read shapes do not change the string-valued write contract.

## Response handling

Resource list verbs return metadata summaries. Use `--full` when the task needs
configuration, key contracts, Input bindings, or schedules. Single-resource get
returns the full object except Input get, which defaults to its protected contract
view; Input get `--full` includes the API's values block. `commands ids` returns IDs
and has no --full. Stored sensitive values remain hidden by the service.

Pretty JSON is the default. `--output raw` returns the body for local processing;
it retains Input response protection. `--response-envelope` returns status, selected
headers, and the protected body, and is mutually exclusive with raw/status-only/ETag
selectors. Use it when the body and ETag must come from the same read.

Supported get verbs offer `--status-only`, which prints the HTTP status and exits 0
even for an HTTP error; inspect the printed status to establish existence. A 404
establishes absence, while authentication or transport failures do not.
`--show-etag` returns just the ETag and fails when the header is missing.

## CLI transport and deployment

Requests and responses are limited to 32 MiB. Redirects return errors and are never
followed. Standard HTTPS_PROXY, HTTP_PROXY, and NO_PROXY settings are respected.
Use `--timeout` for an explicit deadline; use --insecure only when the user has
deliberately requested bypassing TLS verification.

Deploy derives Command/Input IDs from catalog:name:version. It updates after a GET
2xx and creates only after a 404; other read failures stop before a write. Explicit
--if-match prevents creation and conditions the update. Command/Input deploy reuses
the fetched ETag when present.
Schedule enable/disable uses a read-modify-conditional PUT with the fetched ETag.
Export validates the response before atomically replacing the protected output file.

## Reviewed corrections and additional service rules

- Schedules do not require `entityVersion`. The Swagger's undefined required entry
  is stale; ScheduledExecutionsDataConverter.java:55 and scheduled_execution.proto:13
  contain no such field. Conditional writes use the returned ETag.
- Execution status listeners do not require the undefined `properties` entry in their
  Swagger required list. They require status and execute; input and when are optional.
  Supported statuses are PAUSED, INPUT_REQUIRED, FINISHED, ABORTED, SUSPENDED, FAILED.
  Sources: domain-model/utils/DataConverter.java:874–907 and status_listener.proto:10–25.
- Exactly one non-null schedule type is required, and minutes must be multiples of
  five, including once.minute. Recurring arrays are non-empty and unique. These rules
  come from ScheduledExecutionsValidator.java:284–447, beyond the loose Swagger.
- Webhook names are required alphanumeric strings on creation and immutable afterward.
  Its readOnly annotation does not remove the creation requirement. Webhook type is
  the string GENERIC or ALERT_NOTIFICATION; its Swagger object type is erroneous.
  Sources: WebhooksValidator.java:68, DataConverter.java:317, WebhooksRegistry.java:75.
- The public action set is PAUSE, RESUME, ABORT, RETRY, RESET, CHOOSE, FAIL, COMMENT,
  and SKIP. SUSPEND and EXECUTE_STATUS_LISTENER are internal and rejected by
  ApiControllersUtils.java:62. RESET_AND_RETRY is deprecated; use RESET.
- RESET/CHOOSE/SKIP require explicit values in the CLI. CHOOSE uses a current userChoice
  option or an authorized free choice. SKIP's string value contains a JSON object with
  executorPath and optional string-valued outputValues/target. State-specific eligibility
  is checked by the service. Sources: ActionsRegistry.java:369–606.
- Input PATCH is raw JSON merge patch, not base64 or a quoted JSON object. Allowed
  fields are description, keys, values, and tags; an empty patch is rejected. Values
  and individual values cannot be null. A null key descriptor can delete a key;
  non-null descriptors require corresponding values. Sources: InputPatcher.java:26,
  PatchValidator.java:48–139, and UpdatableInputFields.java:14.
- Catalog force deletion uses the authenticated originator; no extra originator option
  is required. Default deletion requires an empty catalog and returns 204; --force
  cascades and returns deleted counts. CatalogsApiController.java:106 and
  CatalogsRegistry.java:287–322 specify this behavior; catalog If-Match is unsupported.
- Some read-back examples show numeric output or null sensitive values contrary to
  their string schemas. Writes enforce string values; reads remain tolerant. The
  shared error enum omits 412, but conditional operations can return it.

## API validation versus authoring policy

Wrapped calls validate the known request schema before sending. `lint --mode api`
reports those constraints alone. Default authoring mode additionally checks content
naming, filename consistency, explicit sensitivity, feature tags, and expression heuristics.
PascalCase/32-character content names are repository preferences; the API's dotted
alphanumeric names can extend to 64 characters. Catalog display names may contain spaces;
technicalName remains 1–10 alphanumeric characters.

Use explicit kinds for partial/update/envelope bodies. Full Inputs require values; an
Input patch is a different contract. Catalog updates require name, not technicalName.
`--read-back` allows returned atomic definitions, omitted hidden values, and service tags.

## Conflicts and request quotas

Use `get --show-etag` for header-only checks. For a replacement, derive the current body
and ETag from one `get --response-envelope` response, preserve the header exactly, and
pass --if-match for supported writes. A 412 requires a fresh read and reconciliation. Command/Input
deploy uses its existing read's ETag when supplied; direct update does not add another GET.

Optional --read-retries N retries only GET/HEAD 429/503 within one total deadline and
honors Retry-After when present. The Swagger does not guarantee this header. Without it,
bounded exponential backoff applies. Trigger/action/content mutations are never retried.
After an ambiguous mutation failure, inspect existing state rather than infer failure.
