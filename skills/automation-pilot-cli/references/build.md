# Building the autopi binaries

End users never build anything — the binaries are committed under
`scripts/bin/<os>-<arch>/` and the `scripts/autopi` selector picks the right one at
runtime. This page is for maintainers who change the CLI source and need to regenerate
the committed binaries.

## Contents

- [Source layout](#source-layout)
- [Prerequisites](#prerequisites)
- [Build all platforms](#build-all-platforms)
- [Other make targets](#other-make-targets)
- [Refresh reviewed API contracts and help](#refresh-reviewed-api-contracts-and-help)
- [Adding a platform](#adding-a-platform)

## Source layout

The Go source lives in `cli/` at the repo root (kept out of the skill folder so the
skill stays documentation + binaries only).

```
cli/
  go.mod                 module github.com/SAP/automation-pilot-agent-skills/cli (stdlib only, zero requires)
  main.go  version.go    entrypoint; Version injected via -ldflags
  internal/client/       HTTP client, env-only connection + operational flags, --print-request renderer
  internal/cmd/          one file per resource + api.go, lint.go, help.go, root.go
  internal/contract/     embedded Swagger snapshot, request validation, service corrections
  internal/lint/         lint model + rules
  Makefile               contributor targets delegating builds to ci/
ci/
  cli/build.sh           cross-compilation
  cli/buildmeta/         source and binary provenance
  cli/contracts/         offline snapshot generation and CLI-help documentation parity
  check_links.py         guidance links and anchors
  evals/skills-e2e/       skill behavior evaluation and local run evidence
```

Dependencies: **none beyond the Go standard library** (`net/http`, `encoding/json`,
`flag`, `os`, `runtime`). No cobra, no third-party modules. `go.mod` has no `require`.

## Prerequisites

The Go toolchain pinned in `cli/.go-version` (currently 1.27.2) — build-time only. The produced binaries are statically linked
with `CGO_ENABLED=0` and have no runtime dependencies.

```bash
# macOS
brew install go
```

## Build all platforms

```bash
ci/cli/build.sh              # from repository root or any working directory
ci/cli/build.sh v0.2.0        # explicit version
# or from cli/: make build
```

`ci/cli/build.sh` cross-compiles five targets into `skills/automation-pilot-cli/scripts/bin/`
at the repository root:

| Target dir | GOOS/GOARCH |
|------------|-------------|
| `darwin-arm64`  | darwin/arm64 |
| `darwin-amd64`  | darwin/amd64 |
| `linux-amd64`   | linux/amd64 |
| `linux-arm64`   | linux/arm64 |
| `windows-amd64` | windows/amd64 (`autopi.exe`) |

Each is built with `CGO_ENABLED=0`, `-trimpath`, `-buildvcs=false`, and stripped
`-s -w` linker flags. `autopi version` embeds the selected version plus the full CLI
source SHA-256. The default version includes a dirty marker when appropriate.
The source digest includes Go, JSON, Python, and build files under `cli/` and
`ci/cli/`, including the embedded contract snapshot and its generators. Evaluation
evidence and unrelated maintenance checks do not change binary provenance.
`bin/build-manifest.json` records revision, toolchain, flags, source digest, and every
binary checksum; `bin/SHA256SUMS` is a conventional checksum list. Disabling implicit
VCS stamps avoids attributing an uncommitted build solely to its predecessor revision.

From repository root, run
`go run ci/cli/buildmeta/main.go verify cli skills/automation-pilot-cli/scripts/bin`
to check the bundled binaries against their source and checksum manifest. CI also rebuilds
and compares checksums, tests race/vet, and runs native platform smoke tests. Size depends
on the Go toolchain; no fixed byte-size range is promised.

## Other make targets

```bash
make test    # go test ./...
make vet     # go vet ./...
make fmt     # gofmt -w .
make clean   # remove local scratch (dist/)
make contract-docs # verify the command reference against the bundled CLI help
```

## Refresh reviewed API contracts and help

From the repository root, regenerate only after reviewing changes in the service's
Swagger definitions and implementation. The generator resolves external references
locally and records source hashes and the service revision; it makes no API requests.

```bash
python3 ci/cli/contracts/snapshot.py /path/to/autopi-services/public-api-swagger/src/main/resources/swagger
python3 ci/cli/contracts/snapshot.py /path/to/autopi-services/public-api-swagger/src/main/resources/swagger --check
ci/cli/build.sh
python3 ci/cli/contracts/helpdocs.py
python3 ci/cli/contracts/helpdocs.py --check
```

Review service corrections in [API contracts](api-contract.md) alongside snapshot
changes. The CLI's tests check schema constraints, every pinned method/path through
its working wrapper, response outcomes, and credential protection. CI uses the
embedded snapshot and shipped binaries; it does not need the sibling service repository.
Help generation covers every registered resource/verb, including APIs outside these
three Swagger documents.

## Adding a platform

Append a `GOOS/GOARCH` pair to `TARGETS` in `ci/cli/build.sh` and add the matching case to the
`scripts/autopi` selector (and `scripts/autopi.cmd` for a Windows arch). Run `go tool
dist list` for valid pairs.
