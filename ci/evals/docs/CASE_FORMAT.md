# Implemented case and evidence format

The evaluator-only [bank.json](../cases/bank.json) uses schema version 1, with a cases
array. [bank.schema.json](../cases/bank.schema.json) is generated from the Go data types;
the Go loader additionally checks identities, paths, budgets, required predicates,
mutant targets and family separation. Unknown fields are rejected.

Each case has id/version/family, primary_skill, domain/origin, mode/tier/difficulty,
split/suites, prompt, public_files, source provenance, assertions, semantic
criteria, reference evidence, and at least two mutants. Artifact submissions name
one relative JSON file. max_turns/max_seconds bound each evaluated session.
Suite tags use the names listed by CLI help and reject duplicates. The default
smoketest tags twelve existing cases. Minimal aliases pilot membership; full
selects the entire bank, plus a separately reported E2E workflow. Legacy suite names retain their existing behavior.
scripted_user_replies sends a fixed reply only after the matching question. Claude
continues in memory; Codex starts an ephemeral continuation with public dialogue
and the same workspace. max_turns counts observable assistant messages; host
stream formats differ, so it is not a cross-host token or reasoning-turn metric.

## Agent boundary

Only prompt, public_files and the installed runtime are staged in the workspace.
Claude registers the plugin; Codex discovers its complete copy at .agents/skills.
Native mode preserves saved host configuration/authentication on macOS and Linux.
It relies on host permissions and does not isolate the bank at the OS level.
With --isolation bubblewrap, Linux has a private filesystem namespace; both the
plugin and Codex skill copy are read-only.
The bank, graders, source solutions, hidden inputs, other candidates and reports
are not mounted in bubblewrap mode. The operator's tenant credentials are never passed to the agent:
fixture cases receive synthetic credentials, and engine cases author locally.
The external evaluator performs separately scoped real-engine acceptance.
Its endpoint/token come from AUTOMATION_PILOT_HOSTNAME/AUTOMATION_PILOT_TOKEN.
Identity is resolved before agent sessions. An optional --tenant only asserts
identity. Reports record the resolved non-secret tenant_id, and cleanup verifies
journal ownership against the configured connection.

A fresh runtime snapshot is copied per case. without-skills keeps the same CLI,
launcher and public contracts while removing guidance/examples. Changing skills
never changes the bank's expected behavior implicitly.

## Deterministic assertions

Assertions identify an evidence source, JSON Pointer, operator and expectation.
Sources are answer, artifact, requests, state and real engine output. Supported
operators are equals, subset, nonempty, request_count, read_after_write, no_writes,
no_changes, lint_passed, launcher, engine and paired_weekly_schedule.

JSON Pointer escaping follows RFC 6901. Missing, null, false, empty string and
empty collection remain distinct. Subset comparisons retain array order. Request
counts filter actual recorded requests; a claimed action is insufficient.
Read-after-write requires an observed later successful GET of the same target.
Lint requires both evaluator lint and a successful observed agent invocation.

Engine assertions compare actual persisted outputs with hidden partition oracles.
There is no jq substitute. Expression artifacts contain one expression property;
the evaluator runs it in a trusted Void probe. The native iterator candidate is
validated against the permitted Void/ForEach surface before deployment, with only
namespace identity substitution. Submitted/deployed hashes are journaled.

## Semantic criteria and calibration

Each criterion has a stable ID and a concrete behavioral requirement. The external
model receives the task, criteria and candidate evidence as untrusted data, with
execution/skill tools disabled. Codex grader tool attempts are rejected. Its response must contain one judgment per criterion and an
existing evidence quotation for each pass. Private preseeded fixture state is
excluded from semantic grading; actual API responses remain observable.

References and mutants are grader-calibration evidence, not scored agent runs.
A mutant declares the predicates it should violate. check verifies deterministic
challenges; calibrate exercises semantic judgments against correct and wrong
candidates. calibrate-engine verifies references on the real service. None of
these automatically supplies independent domain sign-off or proves unseen-model
reliability. Review status remains explicit.

## Fixture behavior

fixture contains a synthetic tenant, initial resources, optional response sequences
and exact allowed write paths/counts. An optional credentials=missing scenario
omits connection variables. The HTTPS service checks authentication, handles
conditional updates, merge patches, metadata reads and MCP client schemas, and
records denied attempts. Stored Input values stay out of public metadata.
It never executes generated Commands or evaluates expressions.

## Results

Evidence records answer/artifact, actual tool trace, request/response journal,
post-operation fixture state, evaluator lint, runtime outputs and scoped failures.
Reports distinguish PASS, FAIL, UNVERIFIED, EVALUATOR_ERROR and UNRUN. A required
unverified predicate cannot pass. Infrastructure failures and intentional
blocked-action cases are different outcomes.

Each first generation retains its own evidence. Optional repair rounds write to
a separate directory and never overwrite the first verdict. Reports include
model/usage/tool counters, reported and reserved costs, latency, configuration
and source hashes. Codex token usage has no USD price; cost_known remains false.
A USD reservation limit is not a Codex billing cap.
Compare matching bank/model/host/settings; keep repeated generations and hidden
input partitions separate from the count of independent tasks.
