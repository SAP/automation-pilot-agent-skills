# Building the autopi binaries

End users never build anything — the binaries are committed under
`scripts/bin/<os>-<arch>/` and the `scripts/autopi` selector picks the right one at
runtime. This page is for maintainers who change the CLI source and need to regenerate
the committed binaries.

## Source layout

The Go source lives in `cli/` at the repo root (kept out of the skill folder so the
skill stays documentation + binaries only).

```
cli/
  go.mod                 module github.com/SAP/automation-pilot-agent-skills/cli (stdlib only, zero requires)
  main.go  version.go    entrypoint; Version injected via -ldflags
  internal/client/       HTTP client, env-only connection + operational flags, --print-request renderer
  internal/cmd/          one file per resource + api.go, lint.go, help.go, root.go
  internal/lint/         lint model + rules
  build.sh  Makefile      cross-compilation
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
cli/build.sh              # from repository root or any working directory
cli/build.sh v0.2.0        # explicit version
# or from cli/: make build
```

`build.sh` cross-compiles five targets into `../skills/automation-pilot-cli/scripts/bin/`:

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
`bin/build-manifest.json` records revision, toolchain, flags, source digest, and every
binary checksum; `bin/SHA256SUMS` is a conventional checksum list. Disabling implicit
VCS stamps avoids attributing an uncommitted build solely to its predecessor revision.

From `cli/`, run `go run ./tools/buildmeta verify . ../skills/automation-pilot-cli/scripts/bin`
to check the bundled binaries against their source and checksum manifest. CI also rebuilds
and compares checksums, tests race/vet, and runs native platform smoke tests. Size depends
on the Go toolchain; no fixed byte-size range is promised.

## Other make targets

```bash
make test    # go test ./...
make vet     # go vet ./...
make fmt     # gofmt -w .
make clean   # remove local scratch (dist/)
```

## Adding a platform

Append a `GOOS/GOARCH` pair to `TARGETS` in `build.sh` and add the matching case to the
`scripts/autopi` selector (and `scripts/autopi.cmd` for a Windows arch). Run `go tool
dist list` for valid pairs.
