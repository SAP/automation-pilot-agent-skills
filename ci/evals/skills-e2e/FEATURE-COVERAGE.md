# Release acceptance coverage

The live suite calls the checksum-verifying bundled launcher. Go is required only
for the maintainer's harness; plugin users need no build tools. Private tenant
evidence stays in the ignored `runs/` directory with owner-only permissions.

| Skill | Acceptance evidence |
|---|---|
| Planning | Delegated synthetic design, explicit exclusions, dependencies, success/failure scenarios. This checks the resulting design, not independent interview quality. |
| Connection | Preflight, live identity and access, tenant mismatch protection. |
| Catalogs | Dedicated creation/read-back, metadata replacement, owned empty-catalog deletion. |
| Discovery | Scoped lists, tag filters, IDs, contract projections, behavior reads, live dependency resolution. |
| CLI | Verified launcher, lint, wire types, envelopes, ETags, upsert/export/import, mutation journal and error handling. Platform/tamper checks also run in installation CI. |
| Inputs | Metadata, typed values, secret binding/rotation, merge patches, key addition/removal, replacement from known values, lifecycle, imports/upserts, deletion. |
| Authoring | Native composition, HTTP transforms, collections, guards, branches, retries, polling, delays, listeners, script runtimes, updates and child orchestration. |
| Expressions | Real engine checks for serialized boundaries, missing values, streams, false preservation, collections, arithmetic, strings/regex/encoding/hash, clocks/dates/GUIDs/version comparison. |
| Executions | Direct/MCP/event/webhook runs, waited/async results, state/input/output/history/log reads, actions, partial bulk outcomes, choices/free text, cleanup. |
| MCP | Live bindings, initialization/schema/tool calls, injected-key exclusion, sync/async fallback, server/tool disable/restore, deletion. |
| Scheduling | One actual once delivery, Inputs and lineage; six schedule shapes, conditional updates and toggles. Recurring definitions remain disabled. |
| Tags | Registry validation, inheritance, filters, logs, execution origin and child/listener/schedule/MCP lineage. |

## Scope and design

The target is the explicit `--tenant`, verified before writes. Use a fresh run ID.
The supplied candidate creates two catalogs; `--catalog` instead uses one dedicated
test catalog for both roles. Never use production content. Public HTTP dependencies
are JSONPlaceholder (read-only resources) and httpbingo (status fixtures). Generated
fake credentials remain inside the tenant. Script probes emit a synthetic constant;
scripts are necessary specifically to exercise those runtimes. No messaging,
infrastructure provisioning, production operation or external credentials are used.

Inputs hold endpoints, bounded workload policy, scheduled arguments and a fake secret.
Outputs contain IDs, counts, transformed public data and synthetic results. Success
requires exact read-back or the expected execution state plus semantic output/path
assertions. Expected failures must stop at their intended executor. HTTP acceptance,
lint and disabled-schedule read-back never establish runtime delivery.

Defaults are explicit and changeable by a test operator: Europe/Sofia, the fixture
workload bounds, five-minute schedule granularity, native composition and a dedicated
namespace. The exclusion case is unrelated tenant content: cleanup requires recorded
IDs and rejects executions belonging to an unrecorded command. An ambiguous mutation
retains its evidence and cannot be automatically replayed.

## Remaining requirements

- OAuth, mTLS, SAP destinations/Cloud Connector and HANA/JDBC need independently
  provisioned test endpoints and authorized credentials. Existing tenant secrets
  are not inspected or repurposed to increase coverage.
- Recurrence across DST/month boundaries, Auto-Abort/retention windows, quota
  exhaustion and the 21-day execution limit need longer or disruptive tests.
  Contract acceptance does not establish those runtime effects.
- Native Windows/Linux behavior and publication attestations belong to the CI
  platform matrix; one macOS live run does not validate every platform.
- This is one supplied candidate, not a no-skill baseline or proof that every fresh
  agent authors equivalent workflows. Fresh-agent comparisons use the evaluator
  runner and cases in `ci/evals/agent-skills/`.

## Reuse after a release

From this directory, with the repository's pinned Go toolchain:

```sh
go run . check --run release01
go run . deploy --tenant YOUR_TENANT --run release01
go run . run --tenant YOUR_TENANT --run release01
go run . features --tenant YOUR_TENANT --run release01
go run . measure --tenant YOUR_TENANT --run release01
go run . cleanup --tenant YOUR_TENANT --run release01
go run . report --run release01
```

Run modes sequentially: they share a state file. `features --only GROUP` selects
content, expressions, actions, schedules, events, scripts or orchestration. A fresh
run is required for a fresh score; prior verdicts are never silently overwritten.
Resume an interrupted mode with the same run ID after reconciling unresolved writes.
The mutation journal reuses captured responses without blindly repeating requests.
For a deliberate repair evaluation, use a new PascalCase `--round Repair` and select
groups with `--only content,actions,schedules,events`. The round namespaces new
resources and verdicts while retaining the original failures and cleanup inventory.
This is a repaired score, not a replacement for the first-attempt score.

Add `measure --session-log PATH --since RFC3339` for host model-usage deltas across a
known task interval. Only usage counters are extracted. Response-size comparisons
use identical resources/formatting; bytes/4 token estimates are labeled estimates.
Actual model usage includes harness development and diagnosis and is not a controlled
per-skill efficiency gain. Separate first-time development from steady-state reruns.

Cleanup removes recorded entry points, execution history and content, and exact
runtime-store counters/listener markers. Reused catalogs remain. An externally
created test catalog additionally needs its saved HTTP 201 `catalog-created.json`
receipt and matching run tag before empty-catalog deletion. Save reports before
cleanup if the tenant UI must remain available for inspection.

e comparisons and source
   digests. Restrict raw tenant evidence and state to a short-lived private artifact;
   exclude credentials and session transcripts. A cleanup failure fails the job.
6. Establish a steady-state timing/token baseline from at least three runs. Keep
   model generation costs separate from deterministic acceptance runs. Add external
   authentication/database fixtures and long-duration schedule tests as separate
   opt-in jobs once those fixtures exist.
