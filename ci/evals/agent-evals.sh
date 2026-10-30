#!/bin/sh
set -eu
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
: "${ANTHROPIC_API_KEY:?Set an authorized evaluator API key; tenant credentials are not used}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
trap 'exit 1' HUP INT TERM
unset AUTOMATION_PILOT_HOSTNAME AUTOMATION_PILOT_TOKEN
. "$ROOT/ci/plugin/fixture.sh"
install_plugin_fixture "$ROOT" "$WORK" stage-evals
cd "$WORK/workspace"
OUTPUT="$ROOT/ci/evals/agent-skills/results/$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$OUTPUT"
git -C "$ROOT" rev-parse HEAD > "$OUTPUT/revision.txt"
git -C "$ROOT" diff --stat > "$OUTPUT/working-tree.txt"
go run "$ROOT/ci/cli/buildmeta/main.go" source "$ROOT/cli" > "$OUTPUT/source.sha256"
go run "$ROOT/ci/cli/buildmeta/main.go" validation "$ROOT/cli" > "$OUTPUT/validation.sha256"
claude --version > "$OUTPUT/evaluator.txt"
claude plugin eval automation-pilot-skills@automation-pilot --runs 3 --ablation with-without \
  --allow-tools Bash Write Edit --disallow-tools Agent WebFetch --no-publish --trust-plugin \
  --max-cost-usd "${AUTOPI_EVAL_BUDGET_USD:-10}" --output-dir "$OUTPUT" --json "$OUTPUT/results.json"
