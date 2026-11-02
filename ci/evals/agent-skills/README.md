# Fresh-agent plugin evaluations

Each case runs in a fresh workspace/session. Claude's plugin evaluator runs three
independent generations per case and a no-plugin baseline, retains tool traces, and
uses the case's grader to score observable actions. Cases cover proposal scope,
untrusted report/tenant instructions, checksum recovery, identity continuity, and
local authoring through the installed verified CLI.

Run `sh ci/evals/agent-evals.sh` from the repository with an authorized
`ANTHROPIC_API_KEY` available to the evaluator. The runner installs a staged plugin
through the Git marketplace path into disposable Claude configuration, removes the
test origin, unsets tenant credentials, disables report publication, and writes evidence beneath
`ci/evals/agent-skills/results/`. The plugin manifest points the evaluator to this
directory only in the disposable evaluator fixture. The published plugin excludes
evaluation cases and experimental evaluation configuration. Go and Claude are
contributor tools, not plugin runtime dependencies.
The default cost ceiling is $10; override `AUTOPI_EVAL_BUDGET_USD` when appropriate.

Inspect attempted tool actions, artifacts and grader judgments, including baseline
scores and variance, before drawing conclusions. A grader result is not a security
guarantee. These cases establish local behavior; the separate live suite under
`ci/evals/skills-e2e/` establishes tenant behavior only for its recorded revision and scope.
