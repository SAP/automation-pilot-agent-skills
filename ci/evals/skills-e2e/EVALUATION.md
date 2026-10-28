# Live skill evaluation

The current skills completed the complex live suite with **34/34 live acceptance checks**
and **60/60 final counted checks** overall. All 29 direct/MCP workflow scenarios matched
status and semantic expectations on their first runtime attempt. One generated command
needed a deployment repair: HttpSequence exceeded the service's expression element limit.
The skills are useful for this scope with service validation and an evidence-based repair
loop; this run does not establish perfect first-draft generation or general model reliability.

The [suite and evaluation protocol](README.md), [behavioral scenarios](suite.json), and
[curated machine-readable results](results.json) are reusable. This was one Codex agent/session
using skill revision `547613053a4d4d497033d9ca897df5c5f8985d12` on tenant `<TENANT_ID>`.
There was no no-skill control group or independent model replicate. Raw numbered CLI evidence,
first candidate snapshots, resource IDs, and archived pre-repair states remain locally in the
Git-ignored `runs/` directory. The private run reports list every final verdict. Published tenant,
resource, execution, and run identifiers are anonymized labels and cannot address live resources.

## Scope and environment

The planned two-catalog creation was rejected because the tenant already had its maximum
50 catalogs. No existing catalog was deleted. The run reused the dedicated
`<EVALUATION_CATALOG>` catalog from the previous weather evaluation, preserving its metadata,
commands, Inputs, and MCP server. Catalog creation receives no live pass. The quota rejection
remains in the private run evidence; the successful adapted run is labeled `evaluation01`.

Five new Eval-prefixed Inputs, ten new commands, and two new MCP servers were created and
verified. The Inputs supply public API endpoints, workload limits, HTTP fixtures, schedule
arguments, and a generated fake secret with an independent label. JSONPlaceholder supplies
read-only post/user fixtures. httpbingo.org supplies real HTTP status responses. Authentication
used only the existing AUTOMATION_PILOT_HOSTNAME and AUTOMATION_PILOT_TOKEN environment
variables. No generated Python was executed locally; guard and aggregation scripts ran in
Automation Pilot.

The original connection failures were temporary: the configured host later returned a healthy
connection with Read/Write/Execute permissions. The hostname and credential were preserved.
These transport errors and the quota limit are environmental outcomes, separate from generation
correctness.

## Scores

| Measure | Result | Meaning |
|---|---:|---|
| Offline candidate, binding, and deliberate-invalid checks | 9/9 | Authored structure and both binding maps accepted; six specific defects rejected |
| New resource read-backs | 17/17 | Five Inputs, ten commands, two MCP definitions match authored fields |
| First-attempt command creates | 9/10 | One expression-limit rejection before runtime; the corrected command deployed |
| First-attempt direct/MCP runtime scenarios | 29/29 | Nineteen FINISHED and ten expected FAILED outcomes, with semantic/path assertions |
| Live client schemas | 2/2 | Injected keys absent; exact caller keys, requiredness, types, hints, and polling tool verified |
| MCP tool/server lifecycle | 2/2 | Disabled tool hidden, complete tool array restored; server disabled and restored |
| Actual once schedule | 1/1 | Delivered once, correct Input/output and lineage, schedule eventually COMPLETED |
| **Final live acceptance** | **34/34** | Runtime scenarios plus schema, lifecycle, and schedule checks |
| **Final counted checks** | **60/60** | Nine offline checks, seventeen read-backs, thirty-four live checks |

First-attempt lint is recorded as an observation, not counted twice. Expected failures pass only
when the execution fails at the specified guard or resilience executor. HTTP 200 and MCP
isError=false do not imply successful workflow completion. The harness followed the captured
execution IDs and checked outputs from the persisted execution output endpoint.

Seven Go harness regression tests passed with the race detector using pinned Go 1.27.2.
Go vet, offline checks with a separate resource namespace, git diff --check, and the existing
five CLI binary/source checksums passed. The CLI source and bundled binaries were unchanged.
CI now runs the harness tests and the candidate's offline checks without tenant credentials.

## What the live checks established

Parallel reporting returned the requested posts in order, with three FINISHED children in two
batches. Parent IDs, executor paths, role tags, and child execution metadata matched. Empty
lists skipped iteration and returned empty arrays. Oversized lists, blank strings, non-string
members, and invalid post ID members failed before iteration or public API calls.

The sequential comparison used two flat scalar IDs and correctly distinguished shared authors
from different authors. An absent post produced an explicit 404 fallback to post one; ordinary
requests remained on the primary path. HTTP bodies were transformed before crossing executor
boundaries, and JSON objects/arrays were decoded according to the string wire representation.

Six resilience cases used the actual HTTP executor: 503→200 with one incremental retry;
503 exhaustion; permanent 401 without transient retries; 202→200 polling; terminal 400; and
202 exhaustion. Relevant native logs captured validation, retry, and repetition behavior.
FINISHED/FAILED status listeners produced separate executions with the expected lineage,
recorded the parent outcome, and completed counter cleanup. Fractional division, zero fallbacks,
and real false values survived the expression and output boundaries.

A narrow fake-secret rotation preserved the separate label. Direct and MCP probes exposed
only bindingPresent and the label, while the client schema exposed zero probe parameters.
No real third-party credential or OAuth flow was tested.

Both MCP servers expose seven business/control tools plus their generated polling tools.
Prefilled Inputs reduce the aggregate backing-command surface from **17 parameters to 8 caller
parameters (53%)**. All seven tools ran through the MCP endpoint. The one-second synchronous
binding probe returned an active execution and completed through polling. A control workflow
returned FAILED while MCP transport succeeded; the harness correctly treated it as the expected
workflow failure. Two MEDIUM array-item advisories remain: the platform emits unconstrained
items, and the commands validate member shape and count explicitly before processing.

The once schedule triggered approximately **27.3 seconds after its predicted time**, within
the trigger window. Exact calendar timestamps are omitted from this published report. Its execution
`EXECUTION_SCHEDULED` finished with post ID one and the matching schedule lineage.
Schedule `SCHEDULE_001` eventually reported COMPLETED and cannot recur.
Both new MCP servers remain enabled for inspection; the original weather resources are intact.

## Usefulness by skill

| Skill | Observed usefulness and limits |
|---|---|
| Plan | Delegated design produced a concrete multi-feature suite, scope, failure guarantees, and acceptance oracles; the interview branch was not tested. |
| Connection | Correctly distinguished transport failures from a healthy permitted tenant once service access recovered. |
| Discovery | Live built-in contracts and scoped existing-resource inspection supported composition and safe reuse; catalog capacity was discovered too late. |
| CLI | Named APIs, string-wire values, masked authentication, stdin Input writes, compact reads, and native binding validation worked. Lint did not catch the expression budget rejection. |
| Inputs | Five coherent prefills, sensitivity metadata, cross-role references, and narrow rotation worked; hidden-value read-back and real authentication remain unproved. |
| Command Authoring | Ten valid final commands and all runtime cases passed; one command needed expression decomposition before deployment. |
| Dynamic Expressions | Typed boundaries, transformations, filtering, arithmetic, conditions, and skipped-output fallbacks worked. The documented element limit was missed in the first large routing expression. |
| Executions | Captured IDs, terminal states, outputs, paths, native logs, and expected failures supported accurate verdicts without duplicate triggers. |
| Tags | Logging and priority overrides, parent/child tags, status-listener lineage, MCP lineage, and schedule origin were verified. Retention timing and priority eviction were not tested. |
| MCP Servers | Simplified contracts, seven tool calls, sync fallback, generated polling, advisory hints, and availability changes passed. |
| Scheduling | One real schedule delivered with correct Inputs and lineage. State lag needs tolerant observation; recurring/DST delivery remains untested. |

## Findings and improvements to pursue

1. **Respect expression budgets before deployment.** The expression reference already documents
   this limit. The agent failed to apply it to a long URL/status routing field. Splitting retry
   and polling decisions into small Void executors fixed the rejection. Preserve this fixture
   as an authoring regression. Any additional mechanical complexity heuristic belongs in the
   autopi Go CLI and must be checked against service behavior; do not invent an exact element
   count or add a Python preflight.
2. **Check catalog capacity early.** A multi-catalog plan should inspect capacity before its first
   create. When capacity is unavailable, explicitly reuse a dedicated evaluation catalog and
   give catalog creation no score. The harness supports --catalog and --namespace so future
   generations can coexist without overwriting the prior candidate.
3. **Treat schedule state as eventually observed.** The scheduled execution was already FINISHED
   while the schedule's state briefly read null. An immediate completion assertion produced a
   false failure. The harness now waits within its deadline for COMPLETED while retaining the
   accepted execution; it does not create a replacement schedule.
4. **Keep instrumentation failures separate from skill failures.** A typed Go schedule payload
   initially compared unequal to equivalent decoded JSON. JSON normalization fixed it and a
   regression test now covers it. Negative fixtures now require the intended diagnostic, not
   merely exit 6. Archived failed instrumentation states remain evidence and do not count as
   generated-command failures.

For comparative skill evaluation, run at least three independent generations per condition,
including a no-skill baseline, with the same task, model settings, budgets, and side-effect
scope. Withhold these candidate definitions and previous findings from the generating agent.
Measure first-attempt correctness, final correctness, repair count, generation token/tool use,
and elapsed time separately. The harness measures acceptance; it does not measure a causal
skill benefit or supply a statistically meaningful model benchmark by itself.

Untested branches include OAuth/mTLS, destinations, Confirm choices, destructive compensation,
recurring and DST execution, retention timing, and catalog deletion. Namespace-based reuse was
validated offline, not deployed in this live run. Those branches receive no live pass.
