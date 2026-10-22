# autopi CLI and agent skills review

Reviewed on 22 October 2026 at `774eeba`, branch `development-cli`.

**Historical audit:** the findings below describe the pre-fix revision. The user subsequently
authorized fixes; see [Fix resolution](#fix-resolution) for the implementation and validation
status. Original evidence line numbers refer to that reviewed revision.

## Assessment

The repository has a useful foundation: a small standard-library-only CLI, explicit resource verbs, offline request previews, live contract discovery, focused skill ownership, and consistent read-back verification in most mutation workflows. The existing Go tests pass.

The highest-priority problems are in behavior the tests do not currently exercise: redirects can disclose authentication and request bodies, Input mutations do not implement the documented secret protection, prebuilt trigger bodies silently discard operational modifiers, and deploy converts failed reads into writes. Validation and skill examples also contain contradictions that weaken the promised production-readiness checks.

This is a review, not an implementation change. No tenant requests, credential reads, public issues, or production mutations were made. Source files and bundled binaries were left unchanged; this report is the only repository addition.

## Scope, evidence, and limitations

- Reviewed all 29 Go source/test files, shared client/configuration, resource handlers, linter, build script, launchers, and five bundled platform binaries.
- Reviewed all 11 existing skills, 45 skill/reference Markdown files, five complete JSON templates, README, contribution/security guidance, and both hidden plugin manifests.
- Used current repository documentation as the behavior specification. No originating issue or fixed diff baseline was supplied; this is a whole-repository audit.
- `go test -race -cover ./...` and `go vet ./...` passed. Local TLS test servers required execution outside the filesystem/network sandbox; they used synthetic credentials and only loopback connections.
- Statement coverage: client **35.4%**, command handlers **59.1%**, linter **82.2%**. Passing coverage does not establish the uncovered safety contracts.
- Fresh native build succeeded. All 60 resource-verb help outputs matched the bundled native binary. The Linux/Windows binaries were inspected, not executed.
- All five example JSON files parse and pass the current linter. One missing local Markdown target was found: the advertised A2A skill.
- Added review probes through Go overlays stored in `/private/tmp`, without altering tracked implementation or tests. Their intentional failures demonstrate the findings below.
- SAP-side execution ordering, accepted feature tags, schedule edge behavior, and plugin configuration injection were not tested against a live tenant/client. Findings dependent on those integrations are explicitly labeled.

Priority means: **P1** = address before relying on unattended mutations or secret-bearing workflows; **P2** = correctness, workflow, or integration problem; **P3** = lower-impact interface/documentation defect. “Reproduced” means a local/offline reproduction, not a production incident.

## CLI behavior findings

### C1 — P1: redirects can send the bearer token and body over HTTP

**Evidence:** [client.go:69](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/client/client.go:69), particularly the default `http.Client` redirect policy at line 76.

The client uses Go's default redirect handling. A trusted HTTPS response redirecting to HTTP on the same hostname can retain `Authorization`; a 307/308 can also resend the request body. Host matching does not enforce the original scheme or port.

**Reproduced:** a local TLS server with a trusted certificate redirected a POST to a plaintext HTTP listener on the same hostname. The listener received both the synthetic bearer token and body. TLS verification was enabled (`Insecure=false`), so this does not require `--insecure`.

**Impact:** an unexpected or compromised redirect can disclose authentication and submitted Input values. Redirects may also move a mutation to an unintended endpoint or change its method.

**Proposal:** reject redirects by default, or explicitly allow only redirects preserving the required HTTPS origin and safe method semantics. Treat redirect failures distinctly. Add HTTPS→HTTP, different-origin/port, and 301/302/307/308 tests.

### C2 — P1: Input write responses expose secrets despite an explicit protection promise

**Evidence:** [inputs.go:74](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/cmd/inputs.go:74), [common.go:216](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/cmd/common.go:216), [client.go:49](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/client/client.go:49). The promised behavior is in [input-model.md:53](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-inputs/references/input-model.md:53).

Input create/update/patch/bulk-create/deploy use generic response rendering. Successful responses are printed unchanged. Error bodies are printed to stdout, then included again in `HTTPError.Error()` on stderr. Only the ordinary successful `inputs get` path strips top-level `values`.

**Reproduced:** a successful Input PATCH response containing a synthetic `values.password` appeared on stdout. A 400 response echoing the same synthetic secret appeared on both stdout and stderr.

**Impact:** submitted credentials can enter logs and agent context even when the skill follows its documented safe workflow. Whether the real API echoes a particular secret is response-dependent; the missing protection is established.

**Proposal:** give Input mutations a safe output policy, preferably returning only identity/status and non-secret metadata. Suppress untrusted Input error bodies while retaining status and safe diagnostics. Ensure every specialized handler uses the same policy. Test nested `values`, bulk responses, error echoes, and non-JSON errors.

### C3 — P1: prebuilt execution requests silently discard modifiers

**Evidence:** [executions.go:209](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/cmd/executions.go:209).

Supplying `-f` or `--data` bypasses the entire request builder. `--dry-run`, `--logs`, `--tag`, `--input`, `--input-json`, and `--command-id` are accepted but ignored.

**Reproduced:** `autopi executions trigger --data '{"commandId":"cat:C:1"}' --dry-run --print-request` returns success and renders a body without the dry-run tag. A caller-specified command override or logging modifier is discarded in the same branch.

**Impact:** the submitted command, inputs, tags, and requested execution behavior can differ from the visible invocation. This matters especially because reusable Input bindings currently require a prebuilt body.

**Proposal:** reject conflicting prebuilt/builder arguments before any request, or define and implement explicit merge semantics for supported modifiers. Do not silently discard requested execution options.

### C4 — P1: deploy treats every failed existence check as absence

**Evidence:** [deploy.go:94](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/cmd/deploy.go:94).

Only a successful GET selects PUT. Every other outcome selects POST, including 401, 403, 429, 5xx, TLS/DNS failures, and timeouts. The original error is discarded and the CLI says “not found.”

**Reproduced:** test servers returning GET 401/403/429/500/503 received a subsequent POST. When that POST succeeded, deploy returned exit 0.

**Impact:** a failed read can initiate an unintended create and hide the actual failure. This violates the ordinary meaning of an existence-based upsert.

**Proposal:** create only after HTTP 404. Propagate every other error without a write. Test GET→PUT, 404→POST, failure→no-write, and a concurrent create conflict. Avoid automatic replay of non-idempotent POSTs.

### C5 — P2: `--input-json` sends non-string wire values

**Evidence:** [executions.go:229](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/cmd/executions.go:229); string serialization is specified in [CLI SKILL.md:124](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-cli/SKILL.md:124) and [triggering.md:27](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-executions/references/triggering.md:27).

The implementation copies `map[string]any` values directly into `inputValues`. Numbers, booleans, arrays, objects, and null remain native JSON values.

**Reproduced:** inputs such as `{"count":42,"enabled":true,"items":["a"],"config":{"x":"y"}}` remain typed in the rendered payload.

**Proposal:** accept only string-valued input maps with clear errors, or provide a deliberate conversion mode. Preserve strings without double encoding and preserve numeric precision when converting.

### C6 — P2: extra positional arguments can produce successful mutations

**Evidence:** [verbs.go:6](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/cmd/verbs.go:6), [common.go:109](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/cmd/common.go:109).

`requireID` uses only the first positional argument. List/create/deploy handlers often discard all positionals. The parser also resumes flag parsing after `--` instead of preserving the normal end-of-options contract.

**Reproduced:** `autopi commands delete first second --print-request` exits 0 and renders DELETE for `first`.

**Proposal:** enforce exact positional arity centrally. Reject unexpected arguments and irrelevant flags. Honor `--` consistently, with tests for mutation verbs.

### C7 — P2: documented Input definitions do not upsert without an explicit ID

**Evidence:** [deploy.go:13](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/cmd/deploy.go:13), [deploy.go:58](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/cmd/deploy.go:58), and the [Input model](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-inputs/references/input-model.md:7).

Deploy derives identity only from top-level `id`. The documented Input body contains `catalog`, `name`, and `version`, but no `id`. Deploying that valid shape always selects POST, even though command/Input identity is deterministic.

**Reproduced:** request preview for the documented shape renders “no id → create” without an existence check. Repeated deployment therefore creates/conflicts instead of updating unless the caller manually passes `--id`.

**Proposal:** derive command/Input IDs from `catalog:name:version`; validate disagreement between explicit ID and body identity. Preserve the separate create-without-ID behavior for MCP/A2A resources with server-assigned IDs.

### C8 — P2: linter accepts incomplete definitions and misses wire/sensitivity rules

**Evidence:** [model.go:45](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/lint/model.go:45), [rules.go:44](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/lint/rules.go:44).

The linter parses a loose partial model but does not enforce required identity/structure, string-valued `values`/`defaultValue`, or output-key sensitivity. Unknown/null documents can pass with no findings.

**Reproduced:** `{}`, `null`, a tenant command with `configuration:null`, numeric stored/default values, and an `outputKeys.accessToken` with `sensitive:false` each passed relevant local probes.

**Proposal:** distinguish schema validation from heuristic advice. Validate required tenant-authored structures and documented wire types using contextual traversal or maintained schemas. Inspect output keys as well as input keys. Support explicit atomic/read-back versus authored-composite modes rather than applying one incomplete schema to both.

### C9 — P2: credential-name heuristics have harmful false positives and gaps

**Evidence:** [rules.go:15](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/lint/rules.go:15), [rules.go:125](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/lint/rules.go:125).

The unbounded substring pattern treats `tokenUrl` as a credential while missing secret-bearing `authorizationHeader`, `serviceKey`, and `clientCert` names documented elsewhere in the repository.

**Reproduced:** a non-sensitive OAuth endpoint named `tokenUrl` produced a CRITICAL finding. Non-sensitive `authorizationHeader` and `serviceKey` definitions passed with no findings.

**Impact:** the validation gate can block legitimate metadata while accepting credential-bearing contracts.

**Proposal:** use a documented, field-aware name policy with explicit non-secret exceptions. Keep inferred names as advice when evidence is ambiguous; require sensitivity for known secret-bearing contract fields. Include both positive and negative fixtures.

### C10 — P2: whole-document regex checks misidentify context and miss partial URL safety

**Evidence:** [rules.go:143](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/lint/rules.go:143), [rules.go:183](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/lint/rules.go:183).

Duration checks match every `delay`/`interval` string anywhere in command JSON, rather than only retry/repeat/initialDelay controls. URL checking suppresses the entire finding when any occurrence of `toUrlEncoded` exists. The bad-function scan also inspects prose and script strings, rather than expression fields.

**Reproduced:** ordinary executor input values `interval:"1h"` and `delay:"3s"` were flagged as invalid control intervals. A URL with one encoded substitution and a second unencoded substitution passed.

**Proposal:** walk parsed JSON with full paths, restrict checks to owning fields, and inspect each dynamic-expression substitution. State explicitly when checks are heuristic. Use the documented converter suitable for the URL location, rather than assuming form encoding and path encoding are interchangeable.

### C11 — P2: enterprise proxy configuration is ignored

**Evidence:** [client.go:70](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/client/client.go:70).

Constructing an empty `http.Transport` leaves `Proxy=nil`. The CLI therefore does not use Go's normal `HTTPS_PROXY`/`HTTP_PROXY`/`NO_PROXY` handling.

**Verified locally:** the configured transport has no proxy callback. This can prevent use on networks requiring a corporate proxy; no production network failure was asserted.

**Proposal:** clone `http.DefaultTransport`, then apply explicit TLS options. Add a proxy-routing integration test and retain safe default timeouts.

### C12 — P2: bundled binary provenance does not match the release contract

**Evidence:** [build.sh:24](/Users/I061125/Git/automation-pilot-agent-skills/cli/build.sh:24) and [build.md:55](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-cli/references/build.md:55).

The documentation promises `CGO_ENABLED=0`, stripped/trimmed builds, and an embedded version. All five bundled binaries record a dirty predecessor revision `c1f672e`; the native binary reports `autopi dev`, records `CGO_ENABLED=1`, and lacks the script's recorded linker flags. Files are approximately 9.3–10 MB rather than the documented 6–7 MB.

This proves the published build contract was not followed. Dirty predecessor metadata alone does **not** prove source staleness; native help comparisons matched current source.

**Proposal:** produce binaries through one automated, pinned build. Record source revision, toolchain, version, checksums, and target metadata. Verify those properties and execute platform smoke tests before release.

### C13 — P3: root-directory build advice fails

**Evidence:** [build.sh:25](/Users/I061125/Git/automation-pilot-agent-skills/cli/build.sh:25), [CLI SKILL.md:39](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-cli/SKILL.md:39), and selector fallback text.

The script builds an absolute module directory without changing to it. The build command fails when invoked from repository root, where no `go.mod` exists. The build reference includes `cd cli`, but the main skill/selector recommend `cli/build.sh` directly.

**Proposal:** change directory to the script's module before building; make invocation independent of the caller's working directory.

### C14 — P3: lint has inconsistent help, flag, and JSON-output behavior

**Evidence:** [lint.go:29](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/cmd/lint.go:29).

`lint --help` prints help then returns exit 1; flags after filenames are treated as filenames; unsupported output formats silently select text. Empty machine-readable findings serialize as `null` rather than an array.

**Proposal:** reuse shared argument/help handling, validate the output enum, and consistently return `[]` for zero findings.

## Skill and documented-behavior findings

### S1 — P2: the tag registry omits a supported execution feature

**Evidence:** [executions.go:192](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/cmd/executions.go:192), [executions.go:248](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/cmd/executions.go:248), [Tags SKILL.md:33](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-tags/SKILL.md:33), and [rules.go:35](/Users/I061125/Git/automation-pilot-agent-skills/cli/internal/lint/rules.go:35).

CLI `--dry-run` adds `"feature:dryRun":""` to an execution POST. The documented feature allow-list omits that tag, and the original linter rejects it in authored content.

**Correction following user clarification:** dry run is a supported SAP Automation Pilot execution mode activated by the tag. Converting `--dry-run` into an offline alias was an incorrect fix and has been reverted. The CLI again adds the original empty-valued execution tag; trigger validation and lint accept it. The user requested removal of dry-run guidance from all skill documents for now. `--print-request` remains the offline request preview.

**Evidence limit:** tag serialization and CLI request behavior are verified with synthetic servers. SAP execution behavior follows the user's clarification and was not tested against a tenant.

### S2 — P2: clean-clone skill activation is missing from onboarding

**Evidence:** [README.md:156](/Users/I061125/Git/automation-pilot-agent-skills/README.md:156), [.gitignore:26](/Users/I061125/Git/automation-pilot-agent-skills/.gitignore:26), and `.claude-plugin` manifests.

README says opening the clone makes skills automatically available. Plugin manifests exist, but no plugin install/load step is documented; project skill registration is an ignored local symlink rather than a tracked clean-clone setup.

**Proposal:** document and verify an actual supported installation path, including skill discovery and environment availability in the agent process. Distinguish shell `source .env` from launching a desktop/client process that inherits that environment. Provide Windows-equivalent setup if Windows is a supported user path.

### S3 — P2: the advertised A2A skill is absent

**Evidence:** [README.md:86](/Users/I061125/Git/automation-pilot-agent-skills/README.md:86).

The A2A Servers link targets a nonexistent `skills/automation-pilot-a2a-servers/SKILL.md`. The CLI has adapter verbs, but no corresponding skill/schema/workflow.

**Proposal:** restore that skill and its current model/ETag guidance, or remove the advertised skill until available.

### S4 — P2: authoring does not run its promised validation gate

**Evidence:** [Command Authoring SKILL.md:62](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-command-authoring/SKILL.md:62), versus [CLI SKILL.md:184](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-cli/SKILL.md:184).

The authoring workflow goes directly from descriptions/sensitivity to tenant creation/update. It never invokes `autopi lint`, despite the CLI skill promising it is used before deployment.

**Proposal:** add a checkable local validation step before any tenant mutation. Define blocking versus advisory findings after fixing the linter's false positives. Use relevant Input validation in the Input workflow too.

### S5 — P2: empty-issues completion criteria reject legitimate informational issues

**Evidence:** [Command Authoring SKILL.md:73](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-command-authoring/SKILL.md:73), versus [command-model.md:76](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-command-authoring/references/command-model.md:76).

The skill requires an empty `issues` array. Its own reference permits informational validation issues when optional authentication comes through reusable Inputs and cautions against inventing default secrets.

**Proposal:** require no blocking issues, evaluate severity, and record accepted informational findings. Do not make legitimate optional credentials impossible to complete.

### S6 — P2: batch/count examples conflict with the documented string boundary

**Evidence:** [ProcessAppsBatch:106](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-command-authoring/examples/ProcessAppsBatch.command.json:106), [ProcessAppsBatch:150](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-command-authoring/examples/ProcessAppsBatch.command.json:150), [ProcessItemsInParallel:171](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-command-authoring/examples/ProcessItemsInParallel.command.json:171), and [collections.md:65](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-dynamic-expressions/references/elements/collections.md:65).

The repository says array outputs cross boundaries as JSON strings, yet examples call `length`, `filter`, `reduce`, and `maxBy` directly on those outputs. With the documented wire representation, `"[]"` has text length 2, so the empty-list guard passes; `"[\"a\",\"b\"]"` has text length 9, producing an incorrect processed count.

**Proposal:** parse with `toArray` before operations that need arrays, retain valid string passthrough where the receiving field itself accepts a wire string, and publish fixtures showing the actual boundary representation. Reconcile the iteration reference's guard with the same rule.

**Evidence limit:** consequences were checked with local string fixtures and the documented semantics; tenant executor representations were not independently measured.

### S7 — P2: retry examples appear to disable the intended recovery path

**Evidence:** [GetResourceWithRetry:144](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-command-authoring/examples/GetResourceWithRetry.command.json:144), [ResolveAndGetResource:207](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-command-authoring/examples/ResolveAndGetResource.command.json:207).

The examples widen HTTP accepted codes to transient failures, then enforce `validate status == 200`, but set `applyOnValidation:false`. That makes a transient accepted response fail validation while apparently excluding validation failures from retry.

**Proposal:** verify engine ordering, then enable validation retries for this design or express the transient failure through an appropriate alternative. Add 503→200, timeout→200, permanent 401, and exhausted-budget acceptance cases.

**Evidence limit:** high-confidence configuration/logic finding; the exact SAP engine ordering and fix require tenant validation.

### S8 — P2: the polling template lacks an explicit terminal-failure postcondition

**Evidence:** [WaitForOperation:72](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-command-authoring/examples/WaitForOperation.command.json:72), [WaitForOperation:91](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-command-authoring/examples/WaitForOperation.command.json:91), versus [patterns.md:34](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-command-authoring/references/patterns.md:34).

The repeat stops for `FAILED` as well as successful states, with `validate:null`. A custom error message describes failure but does not visibly establish a failing condition. The pattern cookbook separately recommends explicit validation to fail on terminal errors.

**Proposal:** assert successful terminal states or fail explicitly on terminal failure, with representative success/failure/budget tests. Keep error-message customization separate from the condition causing failure.

**Evidence limit:** template logic review; no live run established the resulting parent status.

### S9 — P2: expression operator explanations contradict each other

**Evidence:** [collections.md:55](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-dynamic-expressions/references/elements/collections.md:55), [math-and-control-flow.md:32](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-dynamic-expressions/references/elements/math-and-control-flow.md:32), [math-and-control-flow.md:63](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-dynamic-expressions/references/elements/math-and-control-flow.md:63), and [engine-gotchas.md:21](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-dynamic-expressions/references/engine-gotchas.md:21).

Collections describes `//` fallback per streamed value; other references describe fallback only when the complete left stream has no survivor. Math states the comparison precedence one way, then gives a parse example that implies the opposite. Its retry example also uses `.getResource.output.retriesCount`, while the fixture puts `retriesCount` directly on the executor.

**Proposal:** maintain one authoritative operator definition, correct the retry path, and version small engine-backed input/output fixtures. Hand traces should remain labeled predictions until checked against the actual engine.

### S10 — P2: execution actions are missing from the owning skill

**Evidence:** [Executions SKILL.md:32](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-executions/SKILL.md:32), [monitoring.md:10](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-executions/references/monitoring.md:10).

The CLI supports pause/resume/retry/abort/reset/comment/delete, but the skill covers only trigger/monitor/troubleshoot/history. Monitoring says resume requires product action even though the CLI supports RESUME.

**Proposal:** add exact-target/state-checked actions and deletion branches, explain repeat-side-effect risks for retry/reset, and verify resulting state. Preserve human choices for Confirm/input-required steps; do not infer a choice from authorization to monitor.

### S11 — P3: review and local-only authoring modes are not defined

**Evidence:** [README.md:168](/Users/I061125/Git/automation-pilot-agent-skills/README.md:168), [Command Authoring SKILL.md:8](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-command-authoring/SKILL.md:8), [Command Authoring SKILL.md:136](/Users/I061125/Git/automation-pilot-agent-skills/skills/automation-pilot-command-authoring/SKILL.md:136).

README advertises production-readiness review, but there is no review branch or invocation trigger. The only authoring completion path includes deployment and a representative live run. Existing-definition preservation is also less explicit than the full-object preservation rules for schedules and MCP servers.

**Proposal:** define review, local artifact creation, update, deployment, and verification branches. Keep findings-only review read-only, preserve existing unrelated fields during updates, and require live testing only within the user's authorized operational scope. Reuse previously supplied exact targets and approvals instead of asking again for settled choices.

## Per-skill assessment and targeted improvements

| Skill | What works | Focused improvement |
|---|---|---|
| CLI | Central API surface, stdlib-only implementation, live help, offline preview. | Address C1–C14; document secure input/output boundaries. |
| Plan | Structured decision frontier, acceptance scenarios, scope/exclusion discussion. | Preserve settled user decisions; check capability feasibility while designing rather than only afterward. Clarify explicit handoff from a user-invoked skill. |
| Connection | Small non-secret summary and clear failure classes. | Assess permissions against the requested task; include optional GenAI readiness without implying every read-only account is unhealthy. |
| Discovery | Evidence funnel, exact IDs, role/confidence classification, sensible broadening. | Reconcile documented `commandId` dependency traversal with authored `execute` shapes; handle both verified live representations. Add visited/cycle bounds for recursive inspection. |
| Command Authoring | Contract-first design, resilience, sensitivity, live contract resolution, read-back. | Add lint/review/local modes, preserve existing definitions, and use severity-aware validation criteria; repair templates. |
| Inputs | Narrow merge patches, hidden-value awareness, dependency-risk limits. | Implement promised redaction; support authorized existing local secret sources and protected payload files/stdin, with cleanup and no secret reporting. |
| Executions | Single-trigger ambiguity handling, scoped history, evidence-based diagnosis and polling limits. | Add actions; make reusable Input binding available without manual full-body assembly; accurately distinguish polling predictions from proof. |
| Scheduling | Strong time-zone, DST, cartesian-product, lifecycle and complete-update guidance. | Preview next occurrences, validate against supported time zones, and explain when paired times need multiple schedules. Verify date/DST behavior in integration tests. |
| MCP Servers | Stable tool identities, injected Inputs, advisory hints, full-array preservation. | Add permission/readiness preflight and replacement migration; verify the published client schema. Check hint constraints against the service contract. |
| Tags | Clear ownership, inheritance, behavior, retention and logging rules. | Share registry with CLI/linter; validate values/ranges as well as names and maintain rule provenance. |
| Dynamic Expressions | Scope awareness, string-boundary explanation, explicit hand-trace limits, progressive disclosure. | Correct contradictions and paths; publish actual wire-shaped fixtures and engine/version provenance. |

The advertised A2A skill is missing, so it cannot be assessed beyond CLI adapter support and the broken README entry.

## Further proposals, separate from proven defects

1. **Define a capability contract.** Maintain a compact mapping of user operation → skill → CLI verb → required permission → body schema → verification method. Generate help/reference fragments where practical; avoid duplicating mutable product rules across many prose files.
2. **Make secret ingress usable.** Add stdin/file payload support (`-f -`, or equivalent), prefer protected files over inline password `--data` examples, and use existing authorized secret sources. Exact request previews currently print bodies and custom headers; provide redacted preview by default for secret-bearing operations and document any explicit full-preview escape hatch.
3. **Centralize response policy.** Shared execution should own formatting, projection, status-only behavior, sanitization, and output-write failures. Contract/Input/ETag/connection/export handlers partially reimplement it. Prefer allow-listed non-secret projections where the output is a security boundary; avoid passing unexpected secret-bearing shapes through unchanged.
4. **Strengthen transport/input bounds.** Validate hostname syntax without unnecessarily restricting legitimate test/private hosts. Reject nonpositive timeout values, malformed limits/pages/actions, and conflicting selectors before networking. Consider bounded response reads and reliable output error propagation.
5. **Improve concurrency behavior.** Use a retrieved ETag for conditional A2A updates when supported; fail explicit `--show-etag` if none is returned and reject conflicting selectors. An empty `--if-match` should not silently turn an intended locked update into an unlocked write. Automatic locking is currently a proposal, since documentation does not promise it.
6. **Make exports resilient.** Write local definitions atomically and validate the response before replacing a prior artifact. Distinguish a filesystem/output failure from invalid CLI usage.
7. **Verify plugin integration.** `plugin.json` describes hostname/token configuration as automatically exposed environment variables, but repository files contain no explicit mapping. Verify this in each supported client and document the actual configuration path. This is an integration risk, not a reproduced runtime failure.
8. **Add an agent trust boundary.** Treat catalog descriptions, MCP instructions, scripts, logs, and server error text as evidence, not additional authorization or instructions. No malicious artifact was observed; this is robustness guidance for agents consuming untrusted remote content.

## Recommended delivery order

1. **Protect mutations and secrets:** C1–C4; resolve S1; add targeted regression tests with these exact local reproductions.
2. **Make validation trustworthy:** C5–C10 and S4–S9. Repair examples with actual wire fixtures and verify retry/polling semantics on a disposable tenant scenario before claiming production readiness.
3. **Repair workflow and packaging:** S2/S3/S10/S11, C11–C14, supported-client setup, and deterministic release provenance.
4. **Add maintainability gates:** CI for race/tests/vet, all supported builds, binary metadata, skill frontmatter/manifests, local links, examples, and documented CLI invocations. Add narrowly scoped tenant integration tests for product-specific contracts; avoid broad blind mutation tests.

## Reproduction artifacts

The temporary probes deliberately assert desired safety/correctness behavior and fail against the reviewed implementation:

```sh
cd cli
go test -overlay /private/tmp/autopi-client-review/overlay.json \
  ./internal/client ./internal/cmd ./internal/lint -run '^TestReview' -v
go test -overlay /private/tmp/autopi-cli-review/overlay.json \
  ./internal/cmd -run '^TestReview' -v
```

The first overlay covers redirect downgrade, proxy defaults, host shape, Input response disclosure, schema/wire/sensitivity gaps, duration context, and partial URL encoding. The second covers non-404 deploy fallback, prebuilt trigger modifiers, typed input JSON, extra mutation arguments, lint help, and deterministic Input identity. They are temporary review aids rather than committed tests.

Standards axis: no documented style-rule breach was identified; the main maintainability concerns are duplicated response paths and duplicated product rules. Behavior/spec axis: 25 grouped findings are recorded above (14 CLI and 11 skill/documentation findings), with the strongest established risks in redirects, Input output handling, trigger modifiers, and deploy fallback. Recommendations without demonstrated failures are kept separate.


## Fix resolution

Implemented on 22 October 2026 in the working tree. All 25 findings have repository changes;
product-dependent behavior is explicitly separated from the offline evidence below. No tenant
credentials were inspected, and no tenant requests, executions, or external messages were sent.

| Finding | Implemented resolution | Verification |
|---|---|---|
| C1 | HTTP redirects never followed, preserving request origin and preventing token/body forwarding. | Trusted TLS → HTTP/other TLS origin/same-origin tests across 301/302/303/307/308; destinations receive no request. |
| C2 | Input writes return identity metadata only; Input errors suppress both Result and HTTPError bodies. Strict default Input projection also redacts sensitive defaults. | POST/PUT/PATCH/DELETE, normal/full paths, bulk/raw path forms, nested echoes, 400/500; preview redaction tests. |
| C3 | Prebuilt trigger bodies reject explicitly supplied builder command/input/reference/tag/log/dry-run flags; previews work for both forms. | Conflict tests prove zero requests; builder wire/reference body checked at a synthetic server. |
| C4 | Deploy creates only after 404; other HTTP/transport failures propagate without a write. No automatic POST replay on create conflict. | GET 200→PUT, 404→POST, 401/403/429/500/503/307→no-write, dropped connection, create 409. |
| C5 | `--input-json` and prebuilt `inputValues` require JSON strings, without numeric conversion/precision loss. | Number/null rejection and exact large-number/array-string preservation. |
| C6 | Exact positional arity, interspersed flags, and end-of-options honored. Catalog filters bound only where applicable. | Mutation/list/raw argument rejection; dash-prefixed ID after `--`; no flags parsed after `--`. |
| C7 | Command/Input IDs derive from catalog:name:version; explicit/body/derived mismatch rejected. | Input existence request uses derived ID; conflicting identity fails before networking. |
| C8 | Authored structure, identity, contract maps/types, executor aliases/IDs, wire strings, output sensitivity, Input key bound validated. `--read-back` permits atomic definitions/service tags. | Valid/invalid/null/atomic fixtures, output-secret tests, malformed schema types; all five templates pass lint. |
| C9 | Shared credential-name policy excludes endpoint/identity metadata and detects authorizationHeader/serviceKey/clientCert/API-key forms. | Positive/negative name fixtures and Input/output schema tests. |
| C10 | Parsed contextual control paths and per-substitution URL checks replace whole-document duration scans; scripts/prose excluded from expression advice. | Ordinary interval/delay fields pass, actual invalid controls fail, partially encoded URL fails. Documented `type` filter is permitted. |
| C11 | Clone default transport, retaining standard proxy behavior and timeouts. | Default callback parity and actual CONNECT proxy-routing test. |
| C12 | Pinned Go 1.27.2, CGO=0, trimpath/stripped builds, embedded source SHA-256, revision/version/toolchain/checksum manifest. | All five targets rebuilt; metadata and source/checksum verification; repeat build comparison. |
| C13 | Build changes to its own module directory, independent of invocation directory. | Invoked from repository root successfully. |
| C14 | Shared argument/help handling; explicit text/json enum; empty findings `[]`; output errors propagated. | Help exit 0, flags after filenames, unsupported format rejection, empty-array output. |
| S1 | Original `--dry-run` execution-tag behavior restored after user clarification. Shared validation accepts the empty-valued tag. Dry-run guidance removed from skills; their feature table is explicitly partial. `--print-request` remains offline. | Synthetic server receives one execution POST with `"feature:dryRun":""`; preview sends none and retains the tag. Prebuilt/modifier conflicts fail before networking. |
| S2 | Document actual `claude --plugin-dir .` activation and launch-process environment setup, including Windows. Remove unmapped userConfig injection promises. | Installed Claude help confirms flag; strict plugin/marketplace manifest validators pass. End-to-end model-session activation not run. |
| S3 | Restore A2A skill/model with current env names, live model selection, protected bindings, ETag reconciliation, readiness/migration guidance. | Frontmatter/links/JSON checks; missing-ETag/conflicting-selector/conditional-write tests. Schema provenance is historical revision 825056a. |
| S4 | Authoring and Input workflows explicitly run lint before deployment. | Workflow checks and template lint. |
| S5 | Completion distinguishes blocking issues from explained informational warnings; no fabricated secret default to silence warnings. | Documentation review and mode-specific completion criteria. |
| S6 | Collection guards/count/filter/reduce/maxBy parse serialized arrays with toArray; skipped batch output defaults after parsing. Numeric status membership uses toNumber. | Wire-shaped fixture scenarios and repaired templates parse/lint. Expected expression outcomes are predictions pending SAP engine execution. |
| S7 | Retry templates enable applyOnValidation for accepted transient codes that fail validation, keeping transient conditions bounded. | Configuration reviewed/linted; 503→200, timeout→200, permanent 401, exhausted-budget tenant scenarios documented. Live ordering not tested. |
| S8 | Poll template explicitly rejects terminal FAILED while allowing intermediate states; failOnMaxCount remains enabled. | Configuration reviewed/linted; success/failure/budget tenant scenarios documented. Live repeat/validation ordering not tested. |
| S9 | Stream fallback and comparison precedence explanations agree; retry count is on executor, not output. Wire fixture provenance recorded. | Documentation review and JSON fixture validation; hand traces labeled predictions. |
| S10 | Execution action/delete workflows added with exact target/state/history checks, repeat-side-effect analysis, and read-back verification. CLI RESUME documented accurately. | Existing action builder plus enum/arity tests; action eligibility remains service-enforced. |
| S11 | Explicit review/local/tenant branches; preserve existing definition and settled choices; deployment/run only within authorized mode. | All skill frontmatter and workflow references reviewed; local modes have their own completion criteria. |

Further proposals were implemented as shared CLI operating rules, stdin (`-f -` / `--data @-`),
redacted previews including encoded headers, centralized Input/response projections, 32 MiB
bounds, host/timeout/page/status/action/conflict validation, ETag locking, atomic protected
exports, explicit plugin setup, an agent-content trust boundary, a capability/verification map,
calendar-preview and MCP/A2A readiness/migration guidance, fixture provenance, and CI gates.

### Validation performed

- `go test -race -cover ./...` passed using synthetic loopback HTTP/TLS servers. Current
  statement coverage: client **67.9%**, command handlers **68.8%**, domain policy **100.0%**,
  linter **81.8%**. New structural/parser branches expand the denominator; coverage alone
  does not prove the remaining product contracts.
- `go vet ./...`, Go formatting, and `git diff --check` passed.
- Repository content validator passed all **12 skills**, local Markdown file links, JSON,
  manifests, five complete templates, shared policy, and every registered resource/verb help.
- Skill-creator validator passed 11 automatically invoked skills. Planning's intentional
  `disable-model-invocation` frontmatter was checked by the repository validator; the generic
  skill-creator validator does not support that client extension.
- Installed Claude Code's **plugin and marketplace manifests** passed strict validation.
  Its directory validator did not enumerate skill contents, so skill coverage comes from
  the repository/frontmatter validators, not that command.
- All five platform binaries rebuilt from root with CGO=0 and recorded build provenance.
  Source hash, target/toolchain/trimpath metadata, and binary checksums verified. A repeat
  local build matched all five checksums. Native macOS help/preview/lint smoke checks passed.
  Linux/Windows artifacts were cross-built and inspected locally; CI is configured for native
  macOS/Linux/Windows smoke tests but has not run in this working tree.

### Remaining integration evidence

The repository fixes are complete. A tenant/client validation run is still needed to prove
SAP retry/validation/repeat ordering, expression fixture outcomes, current A2A schema/model
availability, MCP hint acceptance and published client schema, and DST/month-end trigger
behavior. These are documented acceptance scenarios/evidence limits, not claims of completed
live validation. The installed client's local plugin-load flag and manifest acceptance are
verified; a full authenticated agent session was not launched. No production-readiness claim
should be made for those integrations until their actual results are recorded.
