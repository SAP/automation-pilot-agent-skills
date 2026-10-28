# Automation Pilot skill evaluation

This is a repeatable behavioral evaluation, with a live acceptance suite. It evaluates an
agent's ability to use the repository skills to generate and operate complete automations.
The supplied definitions are one candidate, not the expected text of future generations.
Observable behavior is the oracle. An unavailable tenant is a blocked test, never a pass.

## Agent task

Give a fresh agent this request, the repository skills, and `suite.json`. Withhold the supplied
definitions and earlier reports until the first candidate is captured:

> Design and deploy two isolated Automation Pilot catalogs for this suite. Use JSONPlaceholder
> for read-only post/user enrichment and httpbingo.org for real HTTP status sequences. Create
> the Inputs, commands, MCP servers, and schedule needed to satisfy every scenario in suite.json.
> Bind configuration, workload limits, and a generated fake secret through Inputs. Expose only
> business parameters to MCP callers. Use the configured AUTOMATION_PILOT_HOSTNAME and
> AUTOMATION_PILOT_TOKEN; keep their values out of artifacts. Use the repository skills and
> autopi for API calls. Validate definitions, deploy prerequisites before parents, run the suite,
> preserve failed first attempts, diagnose from live states/logs, and report first-attempt and
> repaired scores separately. Generated scripts execute only inside Automation Pilot. You may
> create and operate only this run's two catalogs and related resources. Keep any schedule
> bounded, and disable it on timeout. Do not modify unrelated tenant content.

No further design approval is required: the task delegates design and authorizes its bounded
test effects. A fake secret is never sent to either public service. Failed-run counters are
removed by listeners; counters use the parent execution ID. Listener event records remain as
small evidence markers within this run's store namespace. Retained catalogs and MCP servers
allow inspection after the run; the once schedule cannot recur after successful completion.

## Scenario design

Two catalogs separate public content processing from deterministic control tests. Five Inputs
hold public endpoints, HTTP fixtures, collection policy, scheduled arguments, and a fake secret
with an independent label. Ten commands cover HTTP transformation, cross-catalog composition,
parallel and sequential iteration, conditional fallback, retries, repeat polling, listeners,
sensitive binding, and initial delays. Two MCP servers cover business and control tools.

`suite.json` specifies expected results independently of command wording. JSONPlaceholder
post/user IDs are stable fixtures; assertions avoid mutable prose. HTTP controls use actual
503→200, 401, 503 exhaustion, 202→200, 400, and 202 exhaustion responses. A per-parent counter
selects the URL; the HTTP executor supplies the observed status. This differs from a Void
command that merely prints an HTTP status number. The public services remain outside our
control; rate limiting, outages, or blocked network access are recorded separately.

## Running the supplied candidate

The harness uses Go's standard library and calls the bundled autopi binary. It adds no runtime
requirement to the skills. From this directory, using the repository's pinned Go toolchain:

```sh
go run . check
go run . deploy --tenant YOUR_TENANT_ID --run run01
go run . run --tenant YOUR_TENANT_ID --run run01
go run . report --run run01
```

`deploy` and `run` require an explicit `--tenant`; there is no real tenant default. Offline
`check` needs no tenant or credentials, and `report` uses the selected run's saved tenant.

Choose a new run identifier for every fresh generation. `deploy` verifies the configured tenant,
creates each resource once, reads it back, and rejects validation issues. Existing resources
are never overwritten. An ambiguous create/trigger stops the run; inspect evidence before
retrying. `run` resumes only cases without a recorded attempt, and polls an already captured
execution rather than triggering it again. Evidence, resource IDs, candidate hashes, API results,
terminal states, and scenario verdicts are stored in `runs/<run>/`. Secret-bearing payloads are
sent through stdin and never saved. No generated Python runs locally; `check` uses autopi lint.
The published `EVALUATION.md` and `results.json` use anonymized identifiers. Raw run evidence
contains tenant/resource metadata and stays excluded from Git by the local `.gitignore`.

If catalog quota prevents fresh catalogs, `deploy --catalog EXISTING_EVALUATION_CATALOG`
can reuse a dedicated catalog covered by the evaluation's scope. Both logical roles use that
one physical catalog. The harness verifies it, preserves its metadata and existing resources,
and marks catalog creation REUSED instead of assigning a pass. Ensure candidate resource
names do not collide with existing content. The current candidate uses Eval-prefixed Input
names to coexist with the earlier weather evaluation. Never delete unrelated catalogs to make
space. A resumed run remembers its chosen catalog without requiring the flag again.
For another generation in the same catalog, add a distinct `--namespace RoundTwo` when first
initializing the run. It prefixes all candidate Command/Input names and their references;
business tool contracts remain the same. Namespace and catalog become immutable after creation.

`run` creates a once schedule at the next five-minute boundary at least 90 seconds in the
future, in Europe/Sofia. It records the full local calendar prediction and UTC offset, verifies
the timezone against the service, and requires a FINISHED execution with schedule lineage.
It waits at most nine minutes for delivery, then disables an undelivered schedule. MCP lifecycle
tests save the original definition before disabling a tool or server and restore it before
finishing. A resumed `run` restores pending MCP content after tenant validation and before
schema checks. Failed restoration retains its saved recovery state and stops dependent tests.
Schedule failures attempt an ETag-protected disable; cleanup failures retain the schedule ID
and the original error for inspection. After a killed process, inspect pending schedules and
executions before resuming. Report mode works without API access.

## Scoring and evidence

Use identical scenario sets, model settings, tool budgets, and tenant/service conditions for
baseline and revised-skill agents. Run at least three independent generations for a useful
comparison, including a no-skill baseline. Do not give the baseline agent generated definitions.

The harness reports complete scenario verdicts. For a finer evaluation, score each observable
assertion 0 or 1 from captured evidence. Report deployment correctness, first-attempt runtime
correctness, final runtime correctness, MCP contract correctness, repair count, elapsed time,
and agent token/tool use separately. The harness measures acceptance and timing; a human or
evaluation orchestrator records generation tokens, unauthorized actions, and repair count.
Record model identity, skill Git revision, environment/service identity, public-API dependency
errors, and skipped cases. Report blocked cases separately from both passes and failures.

Hard failures: credential disclosure, writes outside the two run catalogs, duplicate execution
after an ambiguous trigger, invented runtime success, or leaving an unbounded live schedule.
These disqualify a generation regardless of its numeric score. An expected FAILED execution
passes only with the specified failed executor; HTTP 200 / MCP isError=false alone cannot pass.

Skill coverage: Planning (delegated design), Connection, Discovery (live contracts), CLI,
Inputs, Command Authoring, Dynamic Expressions, Executions, Tags, MCP Servers, and Scheduling.
Untested branches include OAuth/mTLS, SAP destination APIs, Confirm choices, destructive
compensation, retention timing, recurrence/DST delivery, and catalog deletion. Those branches
receive no score. This suite is an acceptance test, not proof of general model reliability.
