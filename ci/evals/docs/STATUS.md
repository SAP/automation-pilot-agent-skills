# Saved campaign status

Final code review (2026-11-26) added E2E evidence/definition redaction, rejection
of known environment-secret disclosures, dependency/fixture scope checks, and
cleanup recovery without candidate files. These changes are checked offline.
The grader fingerprint also includes shared redaction code; earlier semantic
calibration cannot certify the changed implementation. No model or real-tenant
validation was run for this review. Go was updated to 1.27.4 to resolve
GO-2026-6859; all five CLI bundles were rebuilt and verified. Both modules
pass the current vulnerability scan.


The unified entry point is [evals.sh](../readme.md). The current full preset includes 160 independent tasks plus agent-generated E2E; E2E is also available separately with an explicit reference mode. The implementation is verified with offline tests; no new model campaign or tenant execution has validated this integration. Historical reports below describe the earlier task-only campaign and do not establish a full E2E score.

Snapshot: 2026-11-25. The bank contains 160 tasks; it remains provisional.
The most recent historical 160-task campaign used Claude Sonnet for both the agent and grader.

A later Codex smoketest, `smoketest-codex-20261125T131259Z`, recorded 12
EVALUATOR_ERROR results before any answer or tool trace. Offline reproduction
found malformed quoted MCP override paths: Codex treated the quote characters as
part of a new server name and rejected its transport. The runner now encodes all
disabled server names inside one TOML table, preserving the saved provider and
login. An installed-Codex offline regression check validates both candidate and
grader configuration and confirms configured MCP commands stay disabled.
Process exits now retain bounded, redacted stderr details in per-case reasons.
No model rerun was performed; the saved failed smoketest report is unchanged.

The smoketest (12, default), minimal (20) and full (160 + E2E) presets are implemented
and verified offline. Engine tenant identity is now resolved automatically from
AUTOMATION_PILOT_HOSTNAME and AUTOMATION_PILOT_TOKEN before model sessions.
No new model campaign or calibration was run for this implementation; the
historical verdicts below are unchanged.

| Latest reviewed campaign | Cases |
|---|---:|
| Pass | 67 |
| Fail | 31 |
| Evaluator error | 61 |
| Unverified | 1 |
| Total | 160 |

The reviewed report is stored locally at
`runs/finalization-reviewed-campaign/report.json`. Its reported cost is $153.10,
including reused original candidate sessions and subsequent grading. Do not add
the original campaign's $131.27 again. Separate calibration attempts are additional;
these figures are CLI accounting, not a verified billing total.

51 evaluator errors are unsuccessful Claude grader process exits. The remaining
10 are candidate-process errors, including budget errors. These errors prevent a
reliable overall skill-quality score. Real task failures include expression edge
cases and generated artifact contracts. Inspect individual predicates and evidence
before attributing failures to a skill.

Historical calibration records:

| Record relative to ci/evals/ | Checks | Meaning |
|---|---:|---|
| `runs/finalization-calibration.json` | 411/411 pass | 137 semantic cases, each with a reference and two incorrect candidates |
| `runs/finalization-engine-calibration/calibration.json` | 24/24 pass | Correct references execute on the actual engine |
| `runs/finalization-engine-mutant-calibration-v2/calibration.json` | 46/46 pass | Engine assertions reject intentionally broken solutions |

All 160 cases still require independent review sign-off. The historical calibration
used the previous grader implementation. Subsequent changes to its evidence payload
and error handling have been checked offline, and have not been run against a model.
New calibration records include a grader implementation digest; historical records
without that digest cannot be resumed as calibration of the changed implementation.

Offline verification after polishing passes: runner race tests, `go vet`, all
160 bank definitions, reference-artifact lint, 54 deterministic mutant challenges
and public documentation links. These checks make no model or real tenant calls.

Generate a summary of the saved campaign without invoking Claude, Codex or a tenant:

```sh
# From the repository root; the output path must be fresh.
sh ci/evals/evals.sh report \
  --run runs/finalization-reviewed-campaign \
  --out runs/finalization-reviewed-campaign/summary.md
```

Reports and evidence are Git-ignored local artifacts. Another checkout will not
contain these files. The summary command preserves the original report and verdicts.

Remaining work is to validate the changed grader, resolve infrastructure errors,
review the 31 failed cases, and complete independent case review. Further model
evaluations or calibration in this collaboration require the user's approval.
