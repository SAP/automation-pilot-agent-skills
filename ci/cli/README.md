# CLI maintenance

This guide is for contributors changing CLI source, generators or release artifacts.
Agent skills use the prebuilt CLI and require no Go runtime or build tools. Build
instructions stay in this directory and are excluded from the plugin distribution.

## Build and verify

Use the Go version pinned in `cli/.go-version` and a macOS or Linux maintenance host
(or Linux under WSL). From repository root:

```sh
cd cli
go test -race ./...
go vet ./...
cd ..
ci/cli/build.sh v1.0.0
go run ci/cli/buildmeta/main.go verify cli skills/automation-pilot-cli/scripts/bin
```

Choose the intended release version; it must agree with the plugin manifest. The
build stages darwin-arm64, darwin-amd64, linux-amd64, linux-arm64 and windows-amd64,
then replaces the complete bundle only after verification. A failed build preserves
the previous bundle. A lock prevents concurrent builds. After a forced interruption,
inspect any retained staging/backup directory before recovery.

Builds use `CGO_ENABLED=0`, `-trimpath` and `-buildvcs=false`, with stripped symbols
and an embedded version/source digest. The binaries have no external runtime
libraries. Both launchers verify the selected binary's checksum on every invocation.
The [distribution guide](../../skills/automation-pilot-cli/references/distribution.md)
covers installation and recovery for skill users.

Source provenance includes non-test module files, build/generator source under
`ci/cli/` and the verification launchers. Test files do not force a binary rebuild.
`go run ci/cli/buildmeta/main.go validation cli` separately fingerprints tests,
skills, plugin metadata and CI tooling. Local reports, run evidence and staging
output are excluded. The committed manifest records the toolchain, flags, source
digest, revision and all five binary hashes. CI rebuilds and compares checksums,
then runs native smoke tests on each supported platform.

## Refresh API contracts and help

Review changes in upstream Swagger and implementation before regenerating the
snapshot. The generator resolves local references within the supplied Swagger tree,
records source hashes and revision, and makes no network requests:

```sh
go run ci/cli/contracts/snapshot/main.go /path/to/autopi-services/public-api-swagger/src/main/resources/swagger
go run ci/cli/contracts/snapshot/main.go --check /path/to/autopi-services/public-api-swagger/src/main/resources/swagger
ci/cli/build.sh v1.0.0
go run ci/cli/contracts/helpdocs/main.go
go run ci/cli/contracts/helpdocs/main.go --check
go run ci/contracts/main.go
```

Review the [service corrections](../../skills/automation-pilot-cli/references/api-contract.md)
alongside snapshot changes. Update `ci/contracts/review.json` only after the service
review; its snapshot hash, revision and 90-day expiry are a validation gate. CI uses
the embedded snapshot and needs no sibling service checkout or tenant credentials.
Help generation calls only the checksum-verifying launcher and covers the complete
registered surface, including APIs outside the three Swagger documents.

Run `go test main.go main_test.go` in each generator directory for regression tests.
The generators and provenance tools use the Go standard library. Shell orchestrates
builds; Python is not a maintenance dependency.

## Platform and distribution changes

To add a platform, update `TARGETS` in `build.sh`, the launcher selectors, manifest
verification and the native CI matrix together. Rebuild and test the complete bundle.
Use `go tool dist list` for supported toolchain target names.

The release workflow checks the tag against the plugin version, packages the
runtime assets with `ci/plugin/main.go stage` and `zip`, and produces attestations
before creating a draft release. `stage-evals` is reserved for disposable evaluation
fixtures and is never used by the release job. Native Windows skill usage remains
independent of these contributor tools.
