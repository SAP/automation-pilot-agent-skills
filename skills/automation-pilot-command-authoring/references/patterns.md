# Pattern Cookbook

Patterns for **composing** Automation Pilot commands: how steps chain, branch,
loop, and recover. For a single executor's full contract (IDs, variants, every input/output,
gotchas), read the matching executor reference in the skill's reference table. This cookbook
shows composition rather than restating each contract.

Snippets below show the load-bearing executor fields. Copy the shape, choose the service for
the task, and resolve each executor contract live before use. The skill workflow loads the command
model and composition-control shapes directly when the current flow needs them. Complete templates
are selected from the skill's example index; substitute the tenant ID and verify each referenced
executor contract before deployment.

Accessors used throughout: an earlier step's output is `$(.alias.output.key)`; an HTTP
body is a **string**, so cross a step boundary with `| toObject` / `| toArray`
before reading a field. Condition blocks (`when`, `validate`, `repeat.until`,
`autoRetry.when`, `errorMessages[].when`) use the composition-control shape loaded by the skill
workflow.

## Contents

- [1. Create and poll](#1-create-and-poll)
- [2. Hide configuration behind an enum](#2-hide-configuration-behind-an-enum)
- [3. Chain executor outputs](#3-chain-executor-outputs)
- [4. Conditional execution](#4-conditional-execution)
- [5. Postcondition validation](#5-postcondition-validation)
- [6. Custom error messages](#6-custom-error-messages)
- [7. Status listeners](#7-status-listeners)

---

## 1. Create and poll

POST to create, then poll a status URL with `repeat.until` until it reaches a terminal
state. Two refinements make it robust: a `validate` that **fails fast** on a terminal
error state (so the loop stops instead of spinning to `maxCount`), and `errorMessages`
that surface the API's own failure text. (To branch an HTTP step on a non-2xx or timeout
status, widen `successResponseCodes` / set `succeedOnTimeout` as the HTTP reference describes.)

In `cis-sapcp:CreateSubscription:2`, the poll step reuses the create step's `location`
response header as its URL:

```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "alias": "getSubscriptionStatus",
  "input": {
    "method": "GET",
    "url": "$(.execution.input.serviceKey.endpoints.saas_registry_service_url)$(.subscribeToSubscription.output.headers | getCaseInsensitive(\"location\"))"
  },
  "validate": { "semantic": "OR", "conditions": [ { "semantic": "OR", "cases": [
    { "expression": "$(.getSubscriptionStatus.output.body | toObject.status)",
      "operator": "NOT_EQUALS", "semantic": "AND", "values": ["FAILED", "RETRY"] } ] } ] },
  "repeat": {
    "maxCount": 120, "delay": "15s", "failOnMaxCount": true,
    "until": { "semantic": "OR", "conditions": [ { "semantic": "OR", "cases": [
      { "expression": "$(.getSubscriptionStatus.output.body | toObject.status)",
        "operator": "EQUALS", "semantic": "OR", "values": ["COMPLETED"] } ] } ] }
  },
  "errorMessages": [
    { "message": "$(.getSubscriptionStatus.output.body)",
      "when": { "semantic": "OR", "conditions": [ { "semantic": "OR", "cases": [
        { "expression": "$(.getSubscriptionStatus.output.body | toObject.status)",
          "operator": "NOT_EQUALS", "semantic": "OR", "values": ["COMPLETED"] } ] } ] } }
  ],
  "when": null, "autoRetry": null, "description": null, "progressMessage": null,
  "initialDelay": null, "pause": null
}
```

**Deadline-driven variant.** For a wall-clock budget rather than a fixed count, capture
start with a `utils-sapcp:Void:1` step emitting `$(now)`, then add a second `until` case
`$((.execution.input.deadline * 60) - (now - start)) LESS_THAN "0"`. Use the polling template
selected directly from the skill's example index for the complete definition.

## 2. Hide configuration behind an enum

A `values` entry loads a reusable Input by alias. Make `inputKey` a **dynamic
expression** keyed off a simple enum input. The config object for that choice
loads under the alias, so the caller picks one `region` enum instead of supplying a URL,
token endpoint, etc. This is the standard CF idiom (used by ~100 catalog commands):

```json
"values": [
  { "alias": "regionData",
    "valueFrom": { "inputReference": "metadata-sapcp:CfRegionData:1",
                   "inputKey": "$(.execution.input.region)" } }
]
```

The `inputReference` is a reusable **Input**, not a command. Select and verify the current region
Input through `autopi inputs get`, then read its live keys downstream by alias:

```json
"url": "$(.regionData.cfApiUrl)/v3/apps", "tokenUrl": "$(.regionData.uaaTokenUrl)"
```

Use the same verified Input ID in `allowedValuesFromInputKeys` on the region input so the caller
gets the available keys as an enum.

## 3. Chain executor outputs

A later step reads an earlier step's output. The source determines the access form:

- A **sub-command's declared output** (from its `outputKeys`) is a scalar. Use `$(.getIds.output.spaceId)`.
- An **HTTP body** is a string. Re-parse it to read a field: `$(.getOrg.output.body | toObject.resources[0].guid)`.

In `cf-sapcp:GetCfOrgAndSpaceIds:1`, step 2 reads the org GUID from step 1:

```json
{ "execute": "http-sapcp:HttpRequest:1", "alias": "getOrg",
  "input": { "method": "GET", "url": "$(.regionData.cfApiUrl)/v3/organizations?names=$(.execution.input.subAccount | toUrlEncoded)" } },
{ "execute": "http-sapcp:HttpRequest:1", "alias": "getSpace",
  "input": { "method": "GET",
    "url": "$(.regionData.cfApiUrl)/v3/spaces?organization_guids=$(.getOrg.output.body | toObject.resources[0].guid | toUrlEncoded)" } }
```

Use the identifier-chaining template selected directly from the skill's example index for the
complete two-step definition.

## 4. Conditional execution

Skip a resolve step when the input is already a GUID (from `cf-sapcp:GetCfOrgAndSpaceIds:1`):

```json
"when": { "semantic": "OR", "conditions": [ { "semantic": "OR", "cases": [
  { "expression": "$(.execution.input.subAccount | isGuid)", "operator": "EQUALS", "semantic": "OR", "values": ["false"] } ] } ] }
```

Combine conditions with `AND` and reference an earlier output in the test. In
`sm-sapcp:GetPlanId:1`, step 3 runs when the new plan value is a name **and** differs
from the current plan's name. A `values` entry can itself be an expression:

```json
"when": { "semantic": "AND", "conditions": [
  { "semantic": "OR", "cases": [
    { "expression": "$(.execution.input.newPlan | isGuid)", "operator": "EQUALS", "semantic": "OR", "values": ["false"] } ] },
  { "semantic": "OR", "cases": [
    { "expression": "$(.getCurrentPlan.output.body | toObject.name)", "operator": "NOT_EQUALS", "semantic": "OR", "values": ["$(.execution.input.newPlan)"] } ] } ] }
```

## 5. Postcondition validation

A `validate` block must hold after the step, or the step fails. Assert a single
match on a name lookup (from `cf-sapcp:GetCfOrgAndSpaceIds:1`):

```json
"validate": { "semantic": "OR", "conditions": [ { "semantic": "OR", "cases": [
  { "expression": "$(.getOrg.output.body | toObject.pagination.total_results)", "operator": "EQUALS", "semantic": "OR", "values": ["1"] } ] } ] }
```

"Contains none of these failure strings" uses `DOES_NOT_CONTAIN` with the **inner**
`semantic: "AND"` (must avoid *every* value) (from `cf-sapcp:AssignCfSpaceRole:1`):

```json
"validate": { "semantic": "OR", "conditions": [ { "semantic": "OR", "cases": [
  { "expression": "$(.assignRole.output.body | toObject.errors[].detail)", "operator": "DOES_NOT_CONTAIN", "semantic": "AND", "values": ["Users cannot be assigned", "Invalid space"] } ] } ] }
```

The inner `case.semantic` controls how multiple `values` combine for list operators:
`CONTAINS` + `OR` = contains any; `DOES_NOT_CONTAIN` + `AND` = contains none.

## 6. Custom error messages

Surface the API's own error text, gated on a condition (from
`cf-sapcp:AssignCfSpaceRole:1`, the mirror of the `validate` above):

```json
"errorMessages": [
  { "message": "$(.assignRole.output.body | toObject.errors[].detail)",
    "when": { "semantic": "OR", "conditions": [ { "semantic": "OR", "cases": [
      { "expression": "$(.assignRole.output.body | toObject.errors[].detail)", "operator": "CONTAINS", "semantic": "OR", "values": ["Users cannot be assigned", "Invalid space"] },
      { "expression": "$(.assignRole.output.status)", "operator": "EQUALS", "semantic": "OR", "values": ["401", "403"] } ] } ] } }
]
```

## 7. Status listeners

A listener runs a command when the **whole execution** reaches a status, outside the
normal step sequence. Define it in `configuration.listeners`; use `[]` when the command
needs no listeners. A listener entry mirrors an executor with `execute` (command ID), `input` (value
map), optional `when`, and a `status`:

```json
"listeners": [
  { "status": "FAILED",
    "execute": "ans-sapcp:SendAnsEvent:1",
    "input": { "subject": "Workflow failed", "body": "$(.execution.error.originalMessage // \"unknown\")" },
    "when": null }
]
```

`status` is one of `PAUSED`, `INPUT_REQUIRED`, `FINISHED`, `ABORTED`, `SUSPENDED`,
`FAILED` (no `RUNNING`). Key facts:

- A listener **triggers a separate child execution**. The child runs outside the parent sequence and carries the service tag `autopi:statusListenerOf = <firing execution ID>`.
- A listener can read anywhere in the execution context because it sits outside the sequence. Some data may not exist when it fires; provide a default with `//`.
- A **FAILED** listener can read the original error via `$(.execution.error.originalMessage)` (before any custom error message overrides it).
- **Top-level placement:** listeners on a command used as a sub-command inside a composite do **not** fire. Define them on the command the user triggers.
