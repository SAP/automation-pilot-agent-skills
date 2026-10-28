# Build and validation tools

Repository maintenance lives in this directory. CLI source remains in [cli/](../cli/),
and bundled runtime binaries remain in the CLI skill. Builds write directly to the
bundled binary directory.

| Task | Tool | Run from repository root |
|---|---|---|
| Rebuild all five bundled targets | [CLI build](cli/build.sh) | `ci/cli/build.sh` |
| Verify source and binary provenance | [Build metadata](cli/buildmeta/main.go) | `go run ci/cli/buildmeta/main.go verify cli skills/automation-pilot-cli/scripts/bin` |
| Refresh pinned API schemas | [Swagger snapshot](cli/contracts/snapshot.py) | `python3 ci/cli/contracts/snapshot.py SWAGGER_DIR` |
| Check generated command help | [Help reference](cli/contracts/helpdocs.py) | `python3 ci/cli/contracts/helpdocs.py --check` |
| Check guidance links and anchors | [Link checker](check_links.py) | `python3 ci/check_links.py` |
| Evaluate skill behavior | [Evaluation suite](evals/skills-e2e/README.md) | Follow the suite's offline and live instructions |

Use the Go toolchain pinned in `cli/.go-version`. Go helpers use the standard library;
Python helpers use Python 3's standard library. These are contributor requirements.
The shipped CLI runs without Go or Python.

The [GitHub workflow](../.github/workflows/validate.yml) runs validation automatically.
See [local validation](../CONTRIBUTING.md#local-validation) for CLI and harness tests.
Evaluation runs and raw evidence stay in `ci/evals/skills-e2e/runs/`, covered by the
suite's own `.gitignore`. Keep this directory when resuming an evaluation.
