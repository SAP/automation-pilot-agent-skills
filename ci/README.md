# Build and validation tools

Repository maintenance lives in this directory. CLI source remains in [cli/](../cli/),
and bundled runtime binaries remain in the CLI skill. Builds stage and verify all five
targets before replacing the complete binary bundle. Plugin users need no build tools.
Read the [maintainer build guide](cli/README.md) when changing CLI source or generators.

| Task | Tool | Run from repository root |
|---|---|---|
| Rebuild all five bundled targets | [CLI build](cli/build.sh) | `ci/cli/build.sh` |
| Verify source and binary provenance | [Build metadata](cli/buildmeta/main.go) | `go run ci/cli/buildmeta/main.go verify cli skills/automation-pilot-cli/scripts/bin` |
| Test native launcher checksum enforcement | [Launcher tests](../cli/launcher_test.go) | `cd cli && go test -count=1 -v -run '^TestLauncher' .` |
| Refresh pinned API schemas | [Swagger snapshot](cli/contracts/snapshot/main.go) | `go run ci/cli/contracts/snapshot/main.go SWAGGER_DIR` |
| Check generated command help | [Help reference](cli/contracts/helpdocs/main.go) | `go run ci/cli/contracts/helpdocs/main.go --check` |
| Check guidance links and anchors | [Link checker](links/main.go) | `go run ci/links/main.go` |
| Test real isolated marketplace installation | [Plugin install test](plugin/install-test.sh) | `sh ci/plugin/install-test.sh` (Claude Code 2.1.308) |
| Validate packaged assets | [Plugin packager](plugin/main.go) | `go run ci/plugin/main.go check .` |
| Check contract review freshness | [Contract review](contracts/main.go) | `go run ci/contracts/main.go` |
| Evaluate skill behavior | [Unified evals guide](evals/readme.md) | `sh ci/evals/evals.sh check` (offline); choose smoketest/minimal/full/e2e for approved live runs |

Use the Go toolchain pinned in `cli/.go-version`. Most Go helpers use the standard
library; the unified evaluator also uses a TOML parser for host configuration.
Shell handles build and installation orchestration; Python is not required
by CI. The shipped CLI needs neither Go nor Python.

The [live suite coverage](evals/docs/E2E_COVERAGE.md) documents reusable
checks, cleanup, measurements and how to run release acceptance. Per-run
reports, tenant evidence and development reviews belong in ignored local storage.
The machine-readable `contracts/review.json` remains part of validation: it binds
the pinned API snapshot to a reviewed service revision and expiration date.

The [GitHub workflow](../.github/workflows/validate.yml) runs validation automatically.
See [local validation](../CONTRIBUTING.md#local-validation) for CLI and harness tests.
Evaluation runs and raw evidence stay in `ci/evals/runs/`, covered by the
evaluation module's `.gitignore`. Keep this directory when resuming an evaluation.
Per-call evidence records a source digest; `provenance/` captures the matching skill,
CLI bundle and harness file hashes. Resumed runs retain accepted execution IDs and label
mixed or legacy evidence rather than implying that old passes validate new skill text.

CI runs native launcher and harness checks for each shipped architecture, bounded Go
fuzzing, pinned govulncheck against the current vulnerability database, and weekly review
freshness checks. Tagged releases reuse these gates and attest the complete archive.
The [unified evaluator](evals/readme.md) uses existing Claude/Codex configuration
for local model sessions. Its CI checks are offline and explicitly disable live
host probes and tenant credentials. Full includes 160 tasks plus agent-generated
E2E; standalone E2E reference acceptance requires no model sessions. Reports remain
local and are not published.

The runtime plugin contains skills and public documentation. Eval cases, harness
source and local reports are maintainer assets excluded from release archives.
