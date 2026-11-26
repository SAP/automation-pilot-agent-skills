# Current calibration

Calibration on 2026-11-26 covers all **160 cases**. Codex passes all **414**
semantic reference/mutant challenges across 137 cases. The remaining expression
cases are checked on the real Automation Pilot engine. The native iterator case
has both semantic and engine checks; it is counted once in the 160-case bank.

These results validate the graders and reference expectations. They do not measure
fresh candidate agents using the skills. Independent case review and a new full
candidate campaign remain outstanding. The bank retains its provisional review
status until that review is complete.

## Qualified grader selections

| Selection | Configured Codex grader | Claude Sonnet grader |
|---|---:|---:|
| smoketest: 12 cases | 33/33 challenges pass | 33/33 pass |
| minimal: 20 cases | 57/57 pass | 57/57 pass |
| full: 160 cases | 414/414 pass | 412/414 pass; not qualified |

The Codex configuration used `gpt-6.1-sol`, configured effort `xhigh`, and
codex-cli 0.172.0. The Sonnet alias resolved to `anthropic--claude-4.6-sonnet`
through Claude Code 2.1.326, with its configured effort. Both used normal saved
authentication, native isolation, and `--grader-budget-usd 1`. Qualification is
specific to these profiles and the recorded bank/protocol/configuration digests.
Other grader models, effort settings, or smaller process budgets need their own
checks. Candidate models can differ independently.

Sonnet misses two added challenges: an incorrect explanation of the rolling versus
simple restart branch, and a polling repair that can accept an empty or unhealthy
instance list. The first challenge contains a valid guard repair but a false
branch explanation; the second contains an unsafe readiness predicate. A separate
Sonnet high-effort probe also accepted the unsafe polling repair. These failures
remain recorded. Use Codex grading for the full bank.

## Reference and runtime checks

| Check | Result |
|---|---|
| Source snapshots against autopi-content | Match |
| Offline case/reference checks | 160 cases pass |
| Deterministic mutant challenges | 55 rejected as intended |
| Engine reference execution | 24/24 pass; cleanup verified |
| Engine expression mutants | 46/46 rejected as intended; cleanup verified |
| Reference E2E | 118/118 checks pass, including 29 core scenarios; cleanup verified |

All real-tenant checks use `AUTOMATION_PILOT_HOSTNAME` and
`AUTOMATION_PILOT_TOKEN`; no alternate destination was selected. The command
engine mutant checks exclude the native iterator's unsafe-executor mutants, which
are covered by the adapter's offline guards and semantic calibration.

The third reference E2E attempt passes all 118 checks and verifies cleanup. Its
acceptance implementation and definition hashes match the current files. This
validates reference deployment/acceptance; agent-generated E2E remains unrun.

The first E2E attempt encountered tenant API timeouts; cleanup recovery verified
absence. The second exposed a webhook name containing hyphens and an interrupted
POST. Its failure is preserved, and cleanup verified absence. The harness now
uses alphanumeric webhook names and retries transport-failed GET operations once,
with separate redacted receipts. Creates, triggers and actions cannot replay
through that retry.

## Corrections made during calibration

47 cases changed after checking catalog content and observable evidence. Changes
include exact discovery limitations and dependency names, actual restart branch
selection, nonempty running-state polling, secret-output boundaries, preservation
of the upgrade failure path, and missing pre-edit reads. Conditional references
now include 12 response envelopes captured through the actual CLI against local
HTTPS fixtures. Local preview and preflight references also contain actual CLI
output. Three source-derived challenges were added without changing preset sizes.

Calibration resume now requires a recorded matching grader effort. Early records
from this session were copied with their observed default-effort invocation
metadata; original reports and judgments were retained unchanged. Historical
records with unknown effort cannot be silently resumed. Frozen E2E source copies
now use an underscore-prefixed directory so Go package discovery excludes reports.

## Reproduce or run with these graders

Run each command separately from the repository root, with a fresh output name.
Model calibration consumes tokens; engine/reference E2E commands deploy and clean
up resources. Explicit approval is required when delegating further model runs
in this collaboration.

```sh
# Small candidate model with the calibrated smoketest Sonnet grader.
sh ci/evals/evals.sh run --suite smoketest --agent claude --model haiku \
  --grader claude --grader-model sonnet --grader-budget-usd 1 \
  --runs 1 --repair 0 --parallel 1 --run smoke-calibrated-01

# Full skill evaluation; retain the calibrated Codex grader defaults.
sh ci/evals/evals.sh run --suite full --agent codex --grader codex \
  --grader-budget-usd 1 --runs 1 --repair 0 --parallel 1 --run full-calibrated-01

# Recalibrate every semantic case; this does not start candidate agents.
sh ci/evals/evals.sh calibrate --suite full --agent codex --grader codex \
  --grader-budget-usd 1 --parallel 4 --out runs/calibration-new.json

sh ci/evals/evals.sh calibrate-engine --suite full --out runs/engine-new
sh ci/evals/evals.sh calibrate-engine-mutants --suite full --out runs/mutants-new
sh ci/evals/evals.sh run --suite e2e --e2e-mode reference --run e2e-reference-new
```

Private evidence is under `runs/calibration-20261126/` and the three
`runs/calibration-20261126-e2e-reference*` directories. Reports, metadata copies,
source snapshots and cleanup journals remain Git-ignored. A fresh checkout does
not include them. The [machine-readable record](calibration-results.json) contains fingerprints,
qualified selections and failure counts without private evidence. See
[validation status](STATUS.md) for historical campaigns.
