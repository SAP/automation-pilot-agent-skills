# Evaluation preset design

> Historical design document. Current commands and behavior are documented in the [run guide](../../readme.md).

Status: implemented. This design does not authorize model runs or calibration.
See the [command cookbook](../models.md) and [runner guide](../../readme.md).

## Presets

| Preset | Cases | Local | Fixture | Engine | Default generations | Base agent + grader sessions |
|---|---:|---:|---:|---:|---:|---:|
| `smoketest` | 12 | 4 | 7 | 1 | 1 | 12 + 11 = 23 |
| `minimal` | 20 | 11 | 7 | 2 | 1 | 20 + 19 = 39 |
| `full` | 160 | 80 | 56 | 24 | 1 | 160 + 137 = 297 |

Presets are nested. Smoketest takes one existing case per primary skill from
minimal; minimal retains exactly the current pilot's 20 cases; full includes all
160. Session counts exclude scripted continuations, citation retries and repairs.
These counts are not token or dollar estimates: skill reads/tool output affect cost.

## Smoketest membership

| Primary skill | Existing case | Tier |
|---|---|---|
| CLI | `local-authoring` | local |
| Connection | `connection-tenant-mismatch` | fixture |
| Planning | `proposal-scope` | local |
| Discovery | `discover-cf-start-completion` | fixture |
| Catalogs | `catalog-update-preserve-metadata` | fixture |
| Tags | `tag-one-run-priority-logging` | fixture |
| Inputs | `input-secret-value-rotation` | fixture |
| Scheduling | `schedule-paired-weekday-times` | local |
| MCP servers | `publish-mcp-bound-tool` | fixture |
| Executions | `recover-cf-accepted-run` | fixture |
| Dynamic expressions | `preserve-policy-false` | engine |
| Command authoring | `http-timeout-success-policy` | local |

Initially reuse prompts, public files, references and assertions unchanged. Later
prompt reductions or grading changes require independent review and approved
calibration. Membership changes alter bank/case digests; old reports remain readable.

All real-tenant calls use AUTOMATION_PILOT_HOSTNAME and AUTOMATION_PILOT_TOKEN.
Resolve the tenant ID through the verified launcher's connection info; no separate
tenant selection or credential destination is required. Isolate created resources
within the run-owned namespace on that configured tenant.

All complete presets include real engine checks. Fixture cases retain synthetic
local APIs. Filtering out engine changes counts to 11/18/136 and must not be
reported as a complete 12/20/160 run.

## Implemented behavior

1. Tag the twelve records with `smoketest`; map `minimal` to existing pilot
   membership and `full` to the full bank. Share selection between listing,
   execution and calibration.
2. Make smoketest the default instead of the current 40-case smoke. Keep new
   presets at one generation, no repairs and one worker by default.
3. Preserve legacy names: pilot=20, smoke=40, all/regression=160; release still
   defaults to three generations. Full must not inherit release's repetition.
4. Validate preset names before host resolution/run-directory creation. Preserve
   skill/tier/case filter intersections and explicit repetition overrides.
   Resolve engine tenant identity from the two environment variables before model
   sessions. Missing credentials or failed identity checks stop the run. An explicit
   --tenant is only an assertion against that resolved ID, never a destination
   override. Cleanup verifies journal identity using the same configured connection.
   Never print credentials or put tokens in command arguments.
5. Add offline tests for counts, nesting, one core case per skill, unchanged pilot
   membership, complete full membership, legacy/default behavior and invalid or
   empty selections. Preserve separate run errors, repairs and first-attempt scores.
6. Update help, coverage, README and command cookbook. Display the chosen preset
   and actual count after filtering. The cookbook documents the implemented
   interface after offline verification.
7. Run only offline tests, reference lint, bank/schema validation, link validation
   and shell syntax checks. Do not launch model sessions to check recipe syntax.

## Models and documentation

Preserve each CLI's normal login, provider, configured model and effort. Agent and
grader overrides remain independent; omitted overrides retain that host's defaults.

The cookbook supplies six configured-default examples; every Claude Haiku/Sonnet/
Opus agent/grader pairing for all three presets; Codex smaller/current/stronger
profiles with independent provider IDs; and mixed-host grading in both directions.
Every recipe is individually copyable rather than a batch campaign.

The inspected Codex configuration names `gpt-6.1-sol` with `xhigh` effort; no local
model cache is present. This is configuration evidence, not a live model availability
or cost check. Smaller/stronger Codex IDs must come from the provider. Claude alias
availability also depends on provider support. Any supported ID can be selected for
either role. Unsupported IDs fail rather than being silently replaced.

Budget documentation must distinguish reported API cost from subscription charges
and Codex reservations from enforceable provider billing. Overall budget zero
continues to disable the overall cap; existing per-session budgets remain.

## Approval and completion

Preset selection, tenant preflight and documentation are verified offline.
Adding presets does not certify skill quality or finish independent case review.
Live model validation/calibration remains subject to approval; scores are provisional.

Before any model campaign or calibration, ask the user to approve preset,
agent/grader hosts and models, effort, generations, repairs, concurrency and budget.
Start with an approved smoketest and report usage before proposing more runs.
