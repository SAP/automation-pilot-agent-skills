# Contributing

## Code of Conduct

All members of the project community must abide by the [SAP Open Source Code of Conduct](https://github.com/SAP/.github/blob/main/CODE_OF_CONDUCT.md).
Only by respecting each other we can develop a productive, collaborative community.
Instances of abusive, harassing, or otherwise unacceptable behavior may be reported by contacting [a project maintainer](REUSE.toml).

## Engaging in Our Project

We use GitHub to manage reviews of pull requests.

* If you are a new contributor, see: [Steps to Contribute](#steps-to-contribute)

* Before implementing your change, create an issue that describes the problem you would like to solve or the code that should be enhanced. Please note that you are willing to work on that issue.

* The team will review the issue and decide whether it should be implemented as a pull request. In that case, they will assign the issue to you. If the team decides against picking up the issue, the team will post a comment with an explanation.

## Steps to Contribute

Should you wish to work on an issue, please claim it first by commenting on the GitHub issue that you want to work on. This is to prevent duplicated efforts from other contributors on the same issue.

If you have questions about one of the issues, please comment on them, and one of the maintainers will clarify.

## Contributing Code or Documentation

You are welcome to contribute code in order to fix a bug or to implement a new feature that is logged as an issue.

The following rule governs code contributions:

* Contributions must be licensed under the [Apache 2.0 License](./LICENSE).
* Due to legal reasons, contributors will be asked to accept a Developer Certificate of Origin (DCO) when they create the first pull request to this project. This happens in an automated fashion during the submission process. SAP uses [the standard DCO text of the Linux Foundation](https://developercertificate.org/).
* Contributions must follow our [guidelines on AI-generated code](https://github.com/SAP/.github/blob/main/CONTRIBUTING_USING_GENAI.md) in case you are using such tools.

## Issues and Planning

* We use GitHub issues to track bugs and enhancement requests.

* Please provide as much context as possible when you open an issue. The information you provide must be comprehensive enough to reproduce that issue for the assignee.

## Local validation

Use the toolchain pinned in `cli/.go-version`. From `cli/`, run `go test -race ./...`
and `go vet ./...`. From repository root, run `ci/cli/build.sh` after source changes and
`skills/automation-pilot-cli/scripts/autopi lint skills/automation-pilot-command-authoring/examples/*.command.json`.
Run `go run ci/links/main.go` for local guidance links and anchors, and
`go run ci/cli/contracts/helpdocs/main.go --check` for generated CLI help parity.
For each directory under `ci/links` and `ci/cli/contracts/{helpdocs,snapshot}`, run
`go test main.go main_test.go` to verify the maintenance utilities.
The Go suite includes native launcher checksum enforcement, covering missing/modified
binaries, invalid checksum lists, and argument forwarding. To run these checks alone,
run `go test -count=1 -v -run '^TestLauncher' .` from `cli/`.
From `ci/evals/skills-e2e/`, run `go test -race ./...` and `go vet ./...` for the harness.
CI runs these checks alongside formatting, source/binary checksums, cross-build
reproducibility, command-example lint, and native platform smoke tests.
Product-specific template acceptance scenarios live in
`skills/automation-pilot-command-authoring/references/template-validation.md`; record
actual service evidence separately from offline validation.

For plugin changes, run `sh ci/plugin/install-test.sh` with Claude Code 2.1.308 available.
It installs into disposable configuration and checks the cached bundle without source
or build tools. Run `go test ci/plugin/main.go ci/plugin/main_test.go` for packaging,
`go test ci/contracts/main.go ci/contracts/main_test.go` and `go run ci/contracts/main.go`
for review freshness. See [agent evaluations](https://github.com/SAP/automation-pilot-agent-skills/tree/main/ci/evals/agent-skills/README.md) for fresh sessions and a
baseline; these require evaluator credentials but no Automation Pilot credentials.

Keep tenant-specific reports, session transcripts and development notes out of
commits. Local review artifacts may be kept in `ci/.local/`; live run state stays in
`ci/evals/skills-e2e/runs/`. Both directories are ignored and excluded from release
packages and validation fingerprints. Commit reusable fixtures and tests instead.
