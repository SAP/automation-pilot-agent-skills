# Plan for 160 Automation Pilot skill evaluations

> Historical design document. Current commands and behavior are documented in the [run guide](../../readme.md).

This is the original research plan. See README.md for the implemented runner,
case bank and outstanding service calibration/review gates.

Status: researched design, with twelve pilot specifications. The catalog-derived
runner, fixtures, graders, and complete case bank are still to be implemented.
Nothing in this directory has been scored as an agent evaluation.

The target is **160 independent agent tasks**, derived from real Automation Pilot
operations and judged by observable outcomes. Each task has input partitions and
failure checks; those checks do not inflate the task count. After a skill change,
the suite should show which behavior regressed, the evidence, and whether a repair
was needed. A generated command need not resemble a reference command's JSON.

## Evidence and implications

The scan used `autopi-content` revision
`6e2ecef00a2bfac89dd949a69182966a4db91315` and skills revision
`189914f7f7ee45be9bf71cb4caea0da6cd367979`, on 2026-11-19. The content checkout
was clean. [inventory.json](../../cases/inventory.json) records file counts, a source digest,
and 28 source dossiers with hashes and literal references from public test commands.

| Observed content | Count | Implication |
|---|---:|---|
| Catalog definitions | 33 | Enough domain breadth; metadata has Inputs but no Commands. |
| Command definitions | 523 | Mine a candidate pool, then select deliberately. |
| Command families, grouped by catalog and name | 503 | Multiple versions must not become duplicate evals. |
| Commands with composite configurations | 479 | Rich data flow, conditions, and failure behavior to reconstruct. |
| Deprecated commands | 62 | Exclude from normal generation; use selected ones as discovery distractors. |
| Commands tagged hidden | 76 | Study implementation patterns; direct-use eligibility needs verification. |
| Visible, nondeprecated command definitions | 389 | Initial pool before quality and fixture checks. |
| Public test command definitions | 651 | Includes substantial test infrastructure and helpers. |
| Files named OQ or OQSuite | 309 | 276 are outside `tests/resources`; naming is not proof of coverage. |

253 catalog commands have a retry control, 50 have repetition, 166 have executor
conditions, and 144 have validation. Only four have status listeners, all in the
MCP client catalog. This is a strong authoring corpus but does not evenly cover
all twelve skills. Scheduling, tenant identity, catalog lifecycle, and MCP server
publication need cases from the skills, CLI contracts, and core-functionality OQs.

Several inspected examples reveal useful distinctions:

- `applm/RestartCfApp` guards a rolling restart of one instance when an additional
  instance is disallowed. Its hidden rolling implementation alone is a weaker
  reference because it documents that limitation without owning the public guard.
- `sm/CreateServiceInstance` waits for the operation and fetches parameters only
  when the offering declares them retrievable. Its write-only-parameters OQ checks
  readiness, operation state, matching IDs, and empty parameters.
- `hanacloud/UpgradeHanaCloudInstanceByScope` returns a no-op when no version fits.
  Snapshot rotation and the upgrade must both remain skipped in that branch.
- `hanacloud/RotateHanaCloudSnapshot` deletes the oldest snapshot before creating
  its replacement. A request to guarantee a restore point throughout rotation
  changes the requirement; copying this implementation would not satisfy it.
- `hanacloud/ObtainHanaCloudManagedAdminApiBinding` has 23 executors spanning reuse,
  creation, rotation, CF/Other Environment paths, and concurrent creation conflicts.
- `mcp/CallMcpTool` distinguishes transport status, JSON-RPC validity, and tool
  `isError`, with cleanup listeners. A successful HTTP response is insufficient.

The content repository's
OQ testing guidance in autopi-content/.claude/skills/oq-testing/SKILL.md
supplies useful branch coverage and independent-verification principles. Its
illustrative templates need reconciliation with current contracts: for example,
the service-manager example uses parameters different from the inspected Command,
and one HTTP example joins status and body assertions with OR. Do not import
templates or validators as unquestioned gold answers. These evals belong in this
skills repository, rather than becoming new production-content OQs.

## Coverage target

Assign exactly one primary skill per case for counting and reporting. Record
secondary skills separately. These are proposed quotas, subject to pilot evidence;
any reallocation should retain a total of 160 and explain the coverage tradeoff.

| Primary skill | Cases | Required distinctions |
|---|---:|---|
| Command Authoring | 46 | Native composition, guards, async completion, retry versus polling, updates, repair |
| Dynamic Expressions | 24 | Serialized values, missing data, false, streams, encoding, collection and version logic |
| Discovery | 16 | Fit, waiting semantics, version/replacement, dependencies, scoped no-result |
| Executions | 14 | Accepted versus finished, failure diagnosis, resume, actions, partial bulk results |
| Inputs | 12 | Type/sensitivity compatibility, binding, rotation, preservation, lifecycle |
| Planning | 10 | Consequential questions, settled choices, feasibility, exclusions, success/failure criteria |
| MCP Servers | 10 | Injected keys, schema, hints, sync fallback, full replacement, immutable names |
| Scheduling | 8 | Time zones, paired times, calendar edges, limited recurrence, safe updates |
| Catalogs | 6 | Placement/reuse, ownership, capacity, preservation, exact lifecycle target |
| Tags | 6 | Registry validity, feature semantics, inheritance, lineage, logging |
| Connection | 4 | Missing configuration, identity, permissions, mismatch before mutation |
| CLI | 4 | Verified launcher, redaction, conditional writes, checksum recovery |
| **Total** | **160** | |

Use **100 catalog-derived cases** and **60 skill-workflow cases**. The latter
exercise capabilities whose evidence is sparse in the public catalogs; they
still need sources, fixtures, and concrete assertions. Proposed domain allocation
within the 100 catalog-derived tasks:

| Domain | Cases | Starting sources in `content/public/catalogs/` |
|---|---:|---|
| Cloud Foundry and application lifecycle | 18 | `applm/RestartCfApp`, `cf/ResolveCfIds`, `cf/CreateCfServiceInstance` |
| HANA Cloud | 20 | Upgrade scopes, snapshots, guarded deletion, managed admin bindings |
| Kubernetes | 10 | Restart/poll, resource scope, deletion, metrics and logs |
| BTP accounts, Service Manager, identity | 16 | `sm/CreateServiceInstance`, entitlement versions, role collections |
| HTTP, authentication, destinations | 12 | `HttpRequest`, sensitive variants, destination requests |
| Integration Suite and transport | 8 | `BulkPromoteIFlowToEnvironment`, transport import/status |
| DevOps | 8 | GitHub file updates, Jira cloning, Jenkins/Concourse completion |
| Monitoring and logging | 4 | Cloud Logging, critical metric transformations |
| MCP client operations | 2 | `CallMcpTool`, session cleanup and protocol errors |
| Utilities and scripting | 2 | Iterator mappings and justified runtime selection |
| **Total** | **100** | |

Use three execution tiers, provisionally **80 local, 56 fixture, 24 engine**.
Finalize individual assignments after the pilot; the tiers have different oracles:

| Tier | What runs | What a pass establishes |
|---|---|---|
| Local | Fresh agent, installed skills, offline contracts/fixtures, real bundled lint | Requested artifact, review, or design satisfies its local requirements |
| Fixture | Fresh agent and real bundled CLI against a scripted HTTPS test tenant | Correct API/tool actions, preserved state, diagnosis and evidence handling |
| Engine | Fresh agent plus the real Automation Pilot engine and controlled dependencies | Generated workflow actually satisfies outputs, branches, timing bounds and effects |

A fixture tenant does **not** evaluate generated expressions or execute Commands.
Predetermined successful execution responses must never validate a generated
workflow. Cases whose required oracle is the real engine are unverified/blocked
when that engine is unavailable. Local lint, static predicates, and captured-output
reasoning are useful evidence but cannot receive runtime credit.

Target 32 straightforward, 80 intermediate, and 48 complex tasks. Complexity
comes from decisions, state transitions, and failure paths, rather than prompt
length. Include creation, targeted repair, review, discovery, diagnosis, and
multi-turn planning. Paraphrases and renamed resources are variants of the same
case; they do not contribute to the 160 count.

## Reverse-engineering a prompt

1. **Extract a source dossier.** Record original ID/version/tags, input/output
   contracts, executor dependencies, value bindings, conditions, validations,
   retries, repetitions, error messages, and output mappings. Keep source hashes.
   Normalize tenant placeholders for analysis, never by blindly changing literals
   in a deployment payload.
2. **Recover the caller's need.** State the business outcome, intended callers,
   scope, required inputs/outputs, waiting semantics, failure guarantees, and
   meaningful optional behavior. Separate a documented requirement from an
   implementation assumption. Missing nested-path evidence remains explicit.
3. **Build the behavioral matrix first.** Select relevant equivalence classes:
   supported inputs, empty/absent data, invalid members, boundaries, each branch,
   transient versus permanent failure, deadline exhaustion, and recovery. Avoid
   mechanically enumerating every Cartesian product.
4. **Write an ordinary user request.** Specify observable needs, not executor
   aliases, expression syntax, an exact step count, or the original command name.
   State local/review/deploy/run scope and target clearly. Ask for a custom wrapper
   only when the task requires one. Direct reuse is a valid solution when it fits.
5. **Provide realistic context.** Supply only needed leaf contracts, sanitized
   execution-shaped output, initial artifacts, and environment facts. Native
   contract information remains available to both baseline and skill agents.
   Discovery cases can expose full candidates, including near matches.
6. **Write independent graders.** Derive assertions from requirements and
   verified fixtures/OQs, not from the generated answer or a similarity metric.
   Accept alternative valid implementation shapes and meaningful error wording.
7. **Challenge the case.** Run it against a correct candidate and plausible
   broken candidates before any agent score is trusted. Review whether a correct
   agent could satisfy the prompt while failing the grader, or the reverse.

An authoring fixture may withhold the source composite while providing its leaf
contracts. Label this as authoring under supplied contracts. In a separate
discovery case, expose the source composite and reward direct reuse. Do not hide
a matching Command and then claim the agent failed normal tenant discovery.

Each source family should produce a small number of different tasks only when
the decisions or observable failure modes differ. For example, managed bindings
can support discovery of a managed operation, preserving a valid binding, repair
of concurrent-create recovery, and safe sensitive output handling. Four renamed
copies of the same creation prompt would add little evidence.

## Case format and quality gate

[PILOT.md](PILOT.md) contains twelve concrete proposed cases.
[CASE_FORMAT.md](../CASE_FORMAT.md) defines the public/evaluator split and an example.
The final case record must contain:

- Stable ID, family, primary/secondary skills, task mode, tier, difficulty, and split.
- Source paths/revisions/hashes and an evidence-strength label.
- Exact prompt, public fixture files, initial state, scoped authorization, budgets,
  and scripted user replies where the agent should ask consequential questions.
- Hidden input partitions, required predicates, accepted alternatives, specific
  hard failures, reference candidate(s), and intentionally broken candidates.
- Evidence required for each predicate, dependencies, cleanup, and failure triage.

Promotion from draft to scored case requires:

1. A domain engineer confirms the reconstructed behavior, fixture realism, and
   expected result. Source quality is reviewed, not inferred from repository location.
2. An eval reviewer checks prompt clarity, useful difficulty, oracle independence,
   sensitive-data handling, and overlap with existing cases.
3. A correct candidate passes every required predicate at the assigned tier.
4. At least two plausible, materially different broken candidates are rejected.
   Higher-risk/complex cases target three or more. No single mutation rule fits all cases.
5. A fresh-agent calibration demonstrates that the case can complete within its
   budget and that failures are actionable. Difficulty is recorded from results.
6. Oracle isolation and reset/cleanup checks pass. Only then freeze prompt,
   fixtures, and graders together as a versioned case.

Use targeted mutants: omit a wait; always upgrade; retry a permanent error;
use the wrong executor's current item; default false to true; select the first
snapshot; drop an untouched field on PUT; expose an injected secret; replay an
accepted execution. Test conjunctions with candidates that satisfy only half of
a requirement. Also provide a valid alternate implementation to detect graders
that accidentally enforce one reference's shape.

## Runner and repository integration

Reuse the existing infrastructure, with explicit limits:

- [Fresh-agent cases](../../readme.md) already supply isolated sessions,
  three generations, an ablation, and trace-based LLM criteria. Preserve the four
  existing cases and include their IDs in reporting without renaming them. They
  count within the final 160 if promoted under the new metadata/quality rules.
- [agent-evals.sh](../../evals.sh) stages the installed plugin and sets a cost
  ceiling. Extend or wrap it for case selection, deterministic grading, fixed
  model configuration, and comparison reports. Do not run a 160-case campaign
  under the present $10 default without a measured budget.
- [Live acceptance harness](../../e2e/README.md) supplies useful launcher,
  evidence, provenance, mutation-journal, read-back, polling, and cleanup logic.
  Its scenario names/definitions are specialized; extract reusable components
  rather than feeding every task through its supplied candidate. Keep that
  supplied-candidate acceptance test as a separate CLI/service health signal.

Two engineering spikes precede full generation:

**Oracle isolation.** Current `stage-evals` copies `ci/evals/agent-skills`, including
graders, into the evaluator plugin fixture. A fresh workspace alone does not prove
that the agent cannot read them. The evaluated process must see only the plugin
runtime, public prompts/fixtures, and its writable workspace. Keep graders, hidden
partitions, source composites used as references, historical reports, and other
candidates outside its filesystem/container. Inspect traces for attempted reads.
Evaluate whether the native plugin evaluator can enforce this boundary. If it
cannot, launch fresh agent sessions through an orchestrator and grade externally.
Use separate agent and grader filesystem/process boundaries, not a prompt asking
the agent to ignore accessible answers.

**Real CLI against a fixture tenant.** `client.Config.Validate` accepts valid
hostnames/IPs with ports; CLI tests already use `httptest.NewTLSServer`. Build a
small Go HTTPS fixture service with expected request validation and scripted
responses. Set the two connection variables to its endpoint and a synthetic token.
Trust the fixture CA in the disposable test environment before the agent starts;
preserve normal TLS and launcher verification. Validate this on the canonical
Linux runner before claiming support on other hosts. Unexpected calls become
recorded failures rather than generic 200 responses. Model ETags, pagination,
permission errors, delayed reads, partial outcomes, and committed-but-timed-out
mutations where the case requires them. Do not write a substitute jq engine.

For engine cases, use a dedicated evaluation tenant and per-run namespaces.
Expose controlled HTTP dependencies reachable from that tenant; localhost
fixtures are not reachable from Automation Pilot. Synthetic child Commands can
record requested operations and return realistic outputs where actual HANA/CF
provisioning is unnecessary. This establishes engine orchestration against
controlled children, not real service lifecycle correctness. A smaller opt-in
service-integration group needs separately provisioned systems and credentials.

The runtime oracle must execute the **agent's captured artifact**, with only
documented run-ID/tenant substitution. Save hashes before adaptation and after
deployment. The evaluator must not repair JSON, inject a missing wait, substitute
the reference candidate, or silently choose better input bindings. Runtime input
partitions are hidden until the first candidate is frozen.

Use unique state per case/partition and a reset endpoint or fresh tenant-store
namespace. Capture attempted API calls, persisted outputs, terminal states, and
relevant executor/listener evidence. Maintain a resource journal and an external
cleanup path so a crashed agent does not strand schedules or executions.
Scheduling tests verify disabled definitions by default; actual delivery is a
separate engine case with bounded waiting and captured lineage.

Implement maintainer tooling in Go to fit the repository's existing harness and
CI. A runner adapter should isolate host-specific session launch and transcript
parsing. Start with Claude's installed plugin path already tested here; add Codex
as a separate adapter when it can provide equivalent skill registration and
tool evidence. Comparisons stay within the same model/host/settings.

Keep the bank, fixtures, and graders under `ci/evals/`; exclude them from normal
plugin release assets. Store raw reports/transcripts in an ignored, private run
directory, and publish a sanitized summary only through an explicit evaluation
workflow. The existing root ignore rules exclude `docs` and `evaluations`, which
is why this design is stored under `ci/evals/catalog-derived/`.

Proposed interface, **not implemented commands**:

```text
catalog-evals check                         # case validity, fixtures, grader calibration
catalog-evals run --suite smoke --compare BASELINE
catalog-evals run --suite regression --runs 1 --compare BASELINE
catalog-evals run --suite release --runs 3 --compare BASELINE
catalog-evals report --run RUN_ID
```

`run` must support tier/skill/family filters, budget ceilings, fresh sessions,
timeouts, cancellation cleanup, and explicit result states. Filtering to
local/fixture tiers is useful without a tenant; omitted engine cases are listed
and cannot silently disappear from the headline coverage.

## Scoring and comparison

Grade observable requirements with deterministic predicates first: JSON structure
and types, contract compatibility, real lint result, requested field preservation,
request/effect counts, scoped actions, engine outputs, failure location, client
schema, and cleanup. Use a small evidence-bound LLM rubric for intent recovery,
planning questions, explanation quality, or diagnoses where wording is flexible.
Require evidence references; a claimed tool action receives no credit by itself.

Record each predicate as pass, fail, or unverified, plus the evidence locator.
A case passes only when all required predicates pass and no hard failure occurs.
Optional observations are reported separately. Do not average away an unsafe
action, missing output, or a required-but-unverified runtime branch.

Hard failures are task-specific and include secret disclosure, actions outside
authorized scope, replay of an already accepted mutation, fabricated success,
and unbounded schedule effects. Observe attempts even if the sandbox rejects
them. Distinguish an expected diagnosed failure from an unauthorized action.

Freeze the first complete submission before hidden feedback. Its initial runtime
result is the first-attempt score. Optional repairs receive bounded diagnostic
feedback and a separate repair round; preserve initial artifacts and failures.
Distinguish ordinary lint edits before submission from post-submission repairs.

Reports should show:

- First-attempt case pass rate and required-predicate completion, by primary skill,
  family, tier, task mode, and difficulty.
- Baseline-to-candidate transitions: pass→fail, fail→pass, stable, and unverified.
- Runtime behavior, repaired results, and hard failures as separate columns.
- Agent token/cost/tool use, latency, repairs, cleanup, and evaluator overhead.
- Agent host/model/settings, CLI binary/contract digest, skill/reference hashes
  including dirty files, bank/fixture/grader version, and engine/service identity.

The comparison baseline is the last accepted skills revision. Run a no-skill
baseline during calibration and periodically on a fixed representative subset.
Keep the CLI and public contract interfaces available in both conditions so
removing the plugin does not accidentally remove the tool being evaluated. A
no-skill ablation that also changes tool availability is a different experiment.

Run the same cases, inputs, budgets, and model settings for both revisions.
Interleave runs where possible. For release decisions, use three independent
generations per revision; summarize paired case outcomes and uncertainty grouped
by source family. Do not treat input partitions or repeated generations as
independent new tasks. Three runs reveal some variance but do not establish
production failure probabilities.

Triage infrastructure before attributing failures to skills: fixture/engine health
probe, supplied-candidate acceptance, CLI validation, then agent traces. Model
timeouts, evaluator errors, fixture mismatches, and service outages retain
separate reason codes. A skipped or blocked case never becomes a pass.

Proposed gates: any confirmed new hard failure fails the run; a new deterministic
regression fails its case; a stochastic regression is rerun against both revisions
and requires evidence review. Set aggregate tolerances from pilot variance and
human-reviewed error rates rather than choosing an unsupported percentage now.

## Delivery sequence

| Milestone | Deliverables | Exit criterion |
|---|---|---|
| 1. Mining and selection | Go inventory extractor, dependency/OQ links, reviewed shortlist of roughly 200–240 candidates | Sources pinned; obsolete/hidden eligibility and unsupported behavior marked |
| 2. Runner and 20-case pilot | Twelve cases below plus eight focused failure/engine cases; isolation, HTTPS fixtures, external graders, one complete artifact-to-engine path | Correct/alternate candidates pass, mutants fail, cleanup and isolation verified |
| 3. Calibration | Current-skills, prior-skills where available, and tool-equivalent no-skill runs; three generations of the pilot | Real cost/latency/variance measured; misleading or impossible cases repaired |
| 4. Development bank of 60 | Broad domains, all twelve skills, realistic creation/repair/diagnosis cases | Every case passes promotion rules; 40-case smoke subset chosen |
| 5. Development bank of 120 | Remaining domain/skill quotas, complex and failure-heavy workflows | Coverage matrix complete; deduplication and reviewer sign-off |
| 6. Frozen bank of 160 | 40 additional family-separated holdout cases, CI selection, comparison report, operator instructions | Full rerun works; held-out quality reviewed; release comparison completed |

Keep 120 development cases and 40 holdout cases. Split by source/dependency
family before prompt generation: related wrappers, versions, helpers, and close
paraphrases belong together. Shared generic primitives such as HttpRequest may
span splits. Holdout case payloads stay in evaluator-controlled storage, and
feedback exposes aggregate results until cases are deliberately retired.

The milestone counts are cumulative, not additional cases. Reserve holdout
families during mining and have their review happen without using them to edit
the skills. Quotas guide selection but cannot justify admitting weak cases.

After each skill change, run the **40-case smoke subset** first, then the
**160-case regression suite** for one fresh generation. A release or broad skill
rewrite gets **three generations** and a matched baseline. An impacted-family
run is useful during editing but does not replace periodic full coverage.

Budget from measurements: one complete regression is 160 agent sessions; three
generations are 480. A freshly rerun three-generation baseline plus candidate is
960 sessions. Estimate cost from pilot median/p95 input/output usage and grader
overhead, plus fixture/engine latency; cache only matching provenance. Stop
cleanly at the configured ceiling and report unrun cases. Do not extrapolate a
fixed dollar amount without actual evaluator measurements.

## Definition of the final deliverable

Completion means 160 reviewed cases, not 160 generated prompt files. Each has a
clear requirement, representative context, appropriate oracle, calibrated
graders, failure partitions, provenance, and an isolation/cleanup strategy. A
maintainer can run a documented command after a skill edit and receive an
evidence-backed comparison without manually judging every answer. Remaining
limits, such as actual HANA provisioning, long-duration retention, or DST delivery,
are explicit rather than counted as covered by a local mock.
