# Evaluation preset command cookbook

The implemented presets are **smoketest (12 cases)**, **minimal (20)** and
**full (160 tasks + agent-generated E2E)**. All default to one generation, no repairs and one worker.
Legacy pilot/smoke/all/regression/release selections remain available.

No model runs were used to prepare this document. Model evaluation/calibration in
this collaboration requires the user's approval. Each recipe starts an individual
campaign, not the entire model matrix.

## Setup

Use the existing configured Claude/Codex login and the repository's pinned Go.
Every real-tenant call uses AUTOMATION_PILOT_HOSTNAME and AUTOMATION_PILOT_TOKEN.
The runner resolves the tenant ID from that connection before starting model
sessions; no manual tenant ID or extra credential destination is required.
Missing credentials or failed identity checks stop engine/E2E campaigns early. Full includes one E2E generation per task generation, using the chosen candidate model and no per-scenario model grader.

```sh
cd ~/Git/automation-pilot-agent-skills
export GOTOOLCHAIN="go$(cat cli/.go-version)"
# These presence checks print variable names only if configuration is missing.
: "${AUTOMATION_PILOT_HOSTNAME:?Set AUTOMATION_PILOT_HOSTNAME}"
: "${AUTOMATION_PILOT_TOKEN:?Set AUTOMATION_PILOT_TOKEN}"
```

The optional --tenant flag asserts that the resolved identity equals a specific
ID; it never changes the destination. Real resources stay in the run-owned
namespace, and cleanup verifies the recorded tenant against the same connection.

The fixture tier uses a local synthetic API. Smoketest/minimal filters can omit engine cases; full rejects filters and always includes E2E. Use bank-only legacy selections for deliberate partial campaigns.

## Both hosts with configured default models

These six commands retain each host's configured agent/grader model and effort.
Each explicitly uses one generation, no repairs and one worker. Overall budget
remains uncapped unless you supply a positive --budget-usd; per-session defaults
remain in effect. Run names include a timestamp so earlier reports are preserved.

### smoketest: claude, configured defaults

```sh
sh ci/evals/evals.sh run \
  --suite smoketest --agent claude --grader claude \
  --runs 1 --repair 0 --parallel 1 \
  --run "smoketest-claude-default-$(date -u +%Y%m%dT%H%M%SZ)"
```

### smoketest: codex, configured defaults

```sh
sh ci/evals/evals.sh run \
  --suite smoketest --agent codex --grader codex \
  --runs 1 --repair 0 --parallel 1 \
  --run "smoketest-codex-default-$(date -u +%Y%m%dT%H%M%SZ)"
```

### minimal: claude, configured defaults

```sh
sh ci/evals/evals.sh run \
  --suite minimal --agent claude --grader claude \
  --runs 1 --repair 0 --parallel 1 \
  --run "minimal-claude-default-$(date -u +%Y%m%dT%H%M%SZ)"
```

### minimal: codex, configured defaults

```sh
sh ci/evals/evals.sh run \
  --suite minimal --agent codex --grader codex \
  --runs 1 --repair 0 --parallel 1 \
  --run "minimal-codex-default-$(date -u +%Y%m%dT%H%M%SZ)"
```

### full: claude, configured defaults

```sh
sh ci/evals/evals.sh run \
  --suite full --agent claude --grader claude \
  --runs 1 --repair 0 --parallel 1 \
  --run "full-claude-default-$(date -u +%Y%m%dT%H%M%SZ)"
```

### full: codex, configured defaults

```sh
sh ci/evals/evals.sh run \
  --suite full --agent codex --grader codex \
  --runs 1 --repair 0 --parallel 1 \
  --run "full-codex-default-$(date -u +%Y%m%dT%H%M%SZ)"
```

## Agent and grader model choices

Claude examples cover haiku, sonnet and opus aliases where your provider supports
them. A cheaper agent and stronger grader is a useful tradeoff; a different grader
needs approved calibration before trusting its judgments.

The inspected Codex configuration names gpt-6.1-sol. No local model cache was
present, so this guide does not claim a complete provider-supported model list.
Set smaller/stronger IDs from your configured provider for those recipes:

```sh
export EVAL_CODEX_SMALL_MODEL="YOUR_SUPPORTED_SMALLER_MODEL_ID"
export EVAL_CODEX_STRONG_MODEL="YOUR_SUPPORTED_STRONGER_MODEL_ID"
```

Any supported ID can be selected independently with --model for solving and
--grader-model for judging. Unsupported IDs fail; no provider/model substitution
is performed. Examples are configuration recipes, not model-availability tests.

Define this function once. Defining it does not start model sessions. Its five
arguments are preset, agent host, agent model, grader host and grader model.
An empty model argument preserves that host's configured default.

```sh
eval_run_preset() (
  eval_preset=$1
  eval_agent_host=$2
  eval_agent_model=$3
  eval_grader_host=$4
  eval_grader_model=$5
  eval_run_id="$eval_preset-$eval_agent_host-$(date -u +%Y%m%dT%H%M%SZ)"
  set -- run --suite "$eval_preset" \
    --agent "$eval_agent_host" --grader "$eval_grader_host" \
    --runs 1 --repair 0 --parallel 1 \
    --run "$eval_run_id"
  if [ -n "$eval_agent_model" ]; then
    set -- "$@" --model "$eval_agent_model"
  fi
  if [ -n "$eval_grader_model" ]; then
    set -- "$@" --grader-model "$eval_grader_model"
  fi
  if [ -n "${EVAL_AGENT_EFFORT:-}" ]; then
    set -- "$@" --effort "$EVAL_AGENT_EFFORT"
  fi
  if [ -n "${EVAL_GRADER_EFFORT:-}" ]; then
    set -- "$@" --grader-effort "$EVAL_GRADER_EFFORT"
  fi
  if [ -n "${EVAL_BUDGET_USD:-}" ]; then
    set -- "$@" --budget-usd "$EVAL_BUDGET_USD"
  fi
  if [ -n "${EVAL_CASE_BUDGET_USD:-}" ]; then
    set -- "$@" --case-budget-usd "$EVAL_CASE_BUDGET_USD"
  fi
  if [ -n "${EVAL_GRADER_BUDGET_USD:-}" ]; then
    set -- "$@" --grader-budget-usd "$EVAL_GRADER_BUDGET_USD"
  fi
  printf 'Report directory: ci/evals/catalog-derived/runs/%s/\n' "$eval_run_id"
  sh ci/evals/evals.sh "$@"
)
```

Optional effort settings are independent:

```sh
export EVAL_AGENT_EFFORT=low
export EVAL_GRADER_EFFORT=low
```

Use supported effort values. The inspected Codex default is xhigh, so omitting
an override does not automatically make it a cheap run. Leave these variables
unset to preserve configured effort.

Optional budget settings, using approved numeric limits:

```sh
export EVAL_BUDGET_USD="YOUR_APPROVED_RUN_RESERVATION_LIMIT"
export EVAL_CASE_BUDGET_USD="YOUR_APPROVED_AGENT_SESSION_LIMIT"
export EVAL_GRADER_BUDGET_USD="YOUR_APPROVED_GRADING_SESSION_LIMIT"
```

Replace placeholders with numbers; leave variables unset for the existing defaults.
Overall budget zero disables the overall cap. Codex USD reservations do not enforce
provider billing. CLI-reported API costs may differ from subscription charges.

## Individually copyable model pairings

Invoke one recipe for an approved run. Agent model is the third argument; grader
model is the fifth. Each host needs its own working normal configuration.

## smoketest: Claude

Agent **haiku**, grader **haiku**:

```sh
eval_run_preset smoketest claude haiku claude haiku
```

Agent **haiku**, grader **sonnet**:

```sh
eval_run_preset smoketest claude haiku claude sonnet
```

Agent **haiku**, grader **opus**:

```sh
eval_run_preset smoketest claude haiku claude opus
```

Agent **sonnet**, grader **haiku**:

```sh
eval_run_preset smoketest claude sonnet claude haiku
```

Agent **sonnet**, grader **sonnet**:

```sh
eval_run_preset smoketest claude sonnet claude sonnet
```

Agent **sonnet**, grader **opus**:

```sh
eval_run_preset smoketest claude sonnet claude opus
```

Agent **opus**, grader **haiku**:

```sh
eval_run_preset smoketest claude opus claude haiku
```

Agent **opus**, grader **sonnet**:

```sh
eval_run_preset smoketest claude opus claude sonnet
```

Agent **opus**, grader **opus**:

```sh
eval_run_preset smoketest claude opus claude opus
```

## smoketest: Codex

Agent **smaller**, grader **smaller**:

```sh
eval_run_preset smoketest codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}" codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}"
```

Agent **smaller**, grader **current configured model**:

```sh
eval_run_preset smoketest codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}" codex gpt-6.1-sol
```

Agent **smaller**, grader **stronger**:

```sh
eval_run_preset smoketest codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}" codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}"
```

Agent **current configured model**, grader **smaller**:

```sh
eval_run_preset smoketest codex gpt-6.1-sol codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}"
```

Agent **current configured model**, grader **current configured model**:

```sh
eval_run_preset smoketest codex gpt-6.1-sol codex gpt-6.1-sol
```

Agent **current configured model**, grader **stronger**:

```sh
eval_run_preset smoketest codex gpt-6.1-sol codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}"
```

Agent **stronger**, grader **smaller**:

```sh
eval_run_preset smoketest codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}" codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}"
```

Agent **stronger**, grader **current configured model**:

```sh
eval_run_preset smoketest codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}" codex gpt-6.1-sol
```

Agent **stronger**, grader **stronger**:

```sh
eval_run_preset smoketest codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}" codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}"
```

## smoketest: mixed hosts

Codex agent, Claude Sonnet grader:

```sh
eval_run_preset smoketest codex gpt-6.1-sol claude sonnet
```

Claude Haiku agent, Codex grader:

```sh
eval_run_preset smoketest claude haiku codex gpt-6.1-sol
```

## minimal: Claude

Agent **haiku**, grader **haiku**:

```sh
eval_run_preset minimal claude haiku claude haiku
```

Agent **haiku**, grader **sonnet**:

```sh
eval_run_preset minimal claude haiku claude sonnet
```

Agent **haiku**, grader **opus**:

```sh
eval_run_preset minimal claude haiku claude opus
```

Agent **sonnet**, grader **haiku**:

```sh
eval_run_preset minimal claude sonnet claude haiku
```

Agent **sonnet**, grader **sonnet**:

```sh
eval_run_preset minimal claude sonnet claude sonnet
```

Agent **sonnet**, grader **opus**:

```sh
eval_run_preset minimal claude sonnet claude opus
```

Agent **opus**, grader **haiku**:

```sh
eval_run_preset minimal claude opus claude haiku
```

Agent **opus**, grader **sonnet**:

```sh
eval_run_preset minimal claude opus claude sonnet
```

Agent **opus**, grader **opus**:

```sh
eval_run_preset minimal claude opus claude opus
```

## minimal: Codex

Agent **smaller**, grader **smaller**:

```sh
eval_run_preset minimal codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}" codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}"
```

Agent **smaller**, grader **current configured model**:

```sh
eval_run_preset minimal codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}" codex gpt-6.1-sol
```

Agent **smaller**, grader **stronger**:

```sh
eval_run_preset minimal codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}" codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}"
```

Agent **current configured model**, grader **smaller**:

```sh
eval_run_preset minimal codex gpt-6.1-sol codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}"
```

Agent **current configured model**, grader **current configured model**:

```sh
eval_run_preset minimal codex gpt-6.1-sol codex gpt-6.1-sol
```

Agent **current configured model**, grader **stronger**:

```sh
eval_run_preset minimal codex gpt-6.1-sol codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}"
```

Agent **stronger**, grader **smaller**:

```sh
eval_run_preset minimal codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}" codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}"
```

Agent **stronger**, grader **current configured model**:

```sh
eval_run_preset minimal codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}" codex gpt-6.1-sol
```

Agent **stronger**, grader **stronger**:

```sh
eval_run_preset minimal codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}" codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}"
```

## minimal: mixed hosts

Codex agent, Claude Sonnet grader:

```sh
eval_run_preset minimal codex gpt-6.1-sol claude sonnet
```

Claude Haiku agent, Codex grader:

```sh
eval_run_preset minimal claude haiku codex gpt-6.1-sol
```

## full: Claude

Agent **haiku**, grader **haiku**:

```sh
eval_run_preset full claude haiku claude haiku
```

Agent **haiku**, grader **sonnet**:

```sh
eval_run_preset full claude haiku claude sonnet
```

Agent **haiku**, grader **opus**:

```sh
eval_run_preset full claude haiku claude opus
```

Agent **sonnet**, grader **haiku**:

```sh
eval_run_preset full claude sonnet claude haiku
```

Agent **sonnet**, grader **sonnet**:

```sh
eval_run_preset full claude sonnet claude sonnet
```

Agent **sonnet**, grader **opus**:

```sh
eval_run_preset full claude sonnet claude opus
```

Agent **opus**, grader **haiku**:

```sh
eval_run_preset full claude opus claude haiku
```

Agent **opus**, grader **sonnet**:

```sh
eval_run_preset full claude opus claude sonnet
```

Agent **opus**, grader **opus**:

```sh
eval_run_preset full claude opus claude opus
```

## full: Codex

Agent **smaller**, grader **smaller**:

```sh
eval_run_preset full codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}" codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}"
```

Agent **smaller**, grader **current configured model**:

```sh
eval_run_preset full codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}" codex gpt-6.1-sol
```

Agent **smaller**, grader **stronger**:

```sh
eval_run_preset full codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}" codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}"
```

Agent **current configured model**, grader **smaller**:

```sh
eval_run_preset full codex gpt-6.1-sol codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}"
```

Agent **current configured model**, grader **current configured model**:

```sh
eval_run_preset full codex gpt-6.1-sol codex gpt-6.1-sol
```

Agent **current configured model**, grader **stronger**:

```sh
eval_run_preset full codex gpt-6.1-sol codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}"
```

Agent **stronger**, grader **smaller**:

```sh
eval_run_preset full codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}" codex "${EVAL_CODEX_SMALL_MODEL:?Set a supported smaller model ID}"
```

Agent **stronger**, grader **current configured model**:

```sh
eval_run_preset full codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}" codex gpt-6.1-sol
```

Agent **stronger**, grader **stronger**:

```sh
eval_run_preset full codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}" codex "${EVAL_CODEX_STRONG_MODEL:?Set a supported stronger model ID}"
```

## full: mixed hosts

Codex agent, Claude Sonnet grader:

```sh
eval_run_preset full codex gpt-6.1-sol claude sonnet
```

Claude Haiku agent, Codex grader:

```sh
eval_run_preset full claude haiku codex gpt-6.1-sol
```

## Preview and read results without model calls

Listing any preset needs no model or tenant call:

```sh
sh ci/evals/evals.sh list --suite smoketest
sh ci/evals/evals.sh list --suite minimal
sh ci/evals/evals.sh list --suite full
```

These existing commands already work on saved reports and the current bank:

```sh
# Replace RUN_ID with the report directory printed by your run.
sh ci/evals/evals.sh report --run runs/RUN_ID
sh ci/evals/evals.sh report --run runs/RUN_ID \
  --out runs/RUN_ID/summary.md

# Offline bank, reference and deterministic challenge validation.
sh ci/evals/evals.sh check
```

Markdown output must be a fresh file. Reports preserve first-attempt verdicts,
separate repairs, actual selection counts, usage and cost. Evaluator errors and
unverified cases prevent a reliable overall skill-quality score. See the
[current results guide](../readme.md) for interpretation, comparisons and cleanup.
