# E2E acceptance coverage

Use [the run guide](../readme.md) for campaigns. Full runs all independent tasks
plus this complete workflow. Standalone `run --suite e2e` generates a fresh candidate;
`run --suite e2e --e2e-mode reference` deploys the supplied reference implementations.
Both use AUTOMATION_PILOT_HOSTNAME and AUTOMATION_PILOT_TOKEN, with tenant identity
verified before model sessions or writes. There is no separate live-suite launcher.

## Core workflow

Nineteen definitions describe two catalogs, five Inputs, ten commands and two MCP
servers. Twenty-nine scenarios assert actual execution states/outputs, including
expected failures at the intended executor. Additional checks cover deployment
read-backs, public MCP schemas, tool/server disable and restoration, and one actual
once-schedule delivery with Input and lineage. HTTP transport success alone does not
establish command success. Missing runtime evidence never passes.

JSONPlaceholder supplies read-only post/user enrichment. httpbingo supplies real
503-to-200, permanent 401/400, exhausted retries, and 202-to-200/exhausted polling
responses. Per-parent runtime-store counters choose status URLs. A generated fake
credential remains inside the tenant. Public-service outages and rate limits can
prevent verification and should be distinguished from candidate defects.

## Feature groups

The complete E2E pipeline runs all groups automatically, followed by measurements.
These are service acceptance checks, recorded separately from independent task scores.

| Group | Additional coverage |
|---|---|
| Content | Catalogs, Commands, Inputs and MCP lifecycle; metadata, validation, import/export/upsert, typed values, secret rotation and conditional updates |
| Expressions | Real-engine serialized boundaries, false/null/missing values, collections, arithmetic, strings/regex/encoding/hash, dates/clocks/GUIDs/version comparison |
| Actions | Execution reads/actions, direct/MCP runs, asynchronous recovery, choices and free text |
| Schedules | Six disabled schedule shapes, conditional updates and toggles; the core separately verifies actual once delivery |
| Events | Event/webhook triggering and lineage |
| Scripts | Supported runtime probes emitting synthetic values |
| Orchestration | Native composition, child executions, tags, listeners, delays, retries and polling |
| Measurement | Focused versus full response bytes and CLI timing; bytes/4 estimates are not inference billing |

The candidate may use alternative executor implementations within the declared
native/owned dependency graph. Required fixture configuration is preserved, and
unrelated targets/bindings are rejected before tenant resource creation. Public role names,
required input/output contracts and scenario-visible executor identities are stable
interfaces required by acceptance. Supplied executor configurations are withheld from
agent workspaces. Offline reference mutation tests check the evaluator itself and
are not included as agent-authored correctness results.

## Boundaries and remaining gaps

Each generation/repair owns its catalogs and resources. IDs and ambiguous submissions
are journaled; cleanup does not touch unrelated tenant content. Once schedules are
bounded; failed delivery attempts conditional disabling, and cleanup removes recorded
entry points. Failed restoration/cleanup fails the campaign and retains evidence.
No external infrastructure provisioning or messaging is part of this suite.

OAuth/mTLS, SAP destinations/Cloud Connector and HANA/JDBC need separately provisioned
endpoints. Recurrence across DST/month boundaries, retention/auto-abort windows, quota
exhaustion and long execution limits require longer or disruptive experiments and
remain outside this routine campaign. Definitions alone do not establish these effects.

Fresh E2E generation consumes model tokens and can take tens of minutes. Reference
acceptance consumes no model tokens and verifies fixtures/service behavior. Neither a
single successful run nor reference acceptance establishes general agent reliability.
