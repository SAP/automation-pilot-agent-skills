# Verified CLI distribution

Use the prebuilt CLI supplied with the installed plugin. The launchers select the
platform binary and verify its SHA-256 before every call, including help and lint.
Skill usage needs no Go runtime, compiler, package manager, source checkout or build.

## Verify the installed bundle

The [manifest](../scripts/bin/build-manifest.json) records the source revision,
source digest, version, platform paths and binary hashes. The corresponding
[SHA256SUMS](../scripts/bin/SHA256SUMS) file comes from the same distribution.
macOS/Linux use `shasum` or `sha256sum`; Windows uses built-in Windows PowerShell
and .NET SHA-256 APIs, with no extra modules.
The launcher performs the required verification automatically.

Checksums establish consistency with the trusted manifest. Install the complete
plugin from the trusted SAP repository or an authenticated release archive. A
hash list obtained with an untrusted binary does not authenticate its publisher.

For a published release archive, GitHub CLI can verify its attestation:

```bash
gh attestation verify automation-pilot-skills.zip \
  --repo SAP/automation-pilot-agent-skills \
  --signer-workflow SAP/automation-pilot-agent-skills/.github/workflows/release.yml \
  --source-ref refs/tags/v1.0.0  # use the intended release tag
```

GitHub CLI is optional for this archive verification path. Normal marketplace
installation and skill operations do not require it. Releases without attestations
cannot be authenticated using this command.

## Recover a missing or damaged installation

Restore or reinstall the complete plugin from the trusted distribution, including
its launcher, checksum list, manifest and all platform binaries. Run the installed
launcher again to verify recovery. Preserve the configured connection and settled
resource IDs while recovering; the failed invocation does not authorize a new run.

A mismatched hash, malformed list or missing binary stops execution. Keep that
failure visible until a trusted complete bundle is restored. Never replace an
expected hash with the observed hash, run an unverified executable, or install Go
and rebuild as a skill-recovery step.

Supported platforms are macOS ARM64/x64, Linux ARM64/x64 and Windows x64. If the
host platform is unsupported, report it and use a supported host with the installed
plugin. Contributor source changes and release builds belong to repository maintenance.
