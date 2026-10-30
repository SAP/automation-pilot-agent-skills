# Template acceptance scenarios

The HTTP/collection templates were reviewed on 2026-10-22; the human-choice and explicit
child-execution templates were added on 2026-10-28. These are repository design examples,
not recordings of executions on the current service. Lint verifies structure and heuristics;
only a controlled run establishes runtime ordering, choice behavior, and output representation.
Use a disposable command and an
endpoint you control, with fake values and no production side effects. Record tenant/service
version, substituted command contracts, execution ID, ordered responses, retry/repeat counts,
parent status, and output. The existing user's scope must authorize the test run.

| Template | Scenario | Expected behavior to verify |
|---|---|---|
| GetResourceWithRetry / ResolveAndGetResource | 503, then 200 | The wire status is parsed with `toNumber` for membership. Accepted 503 fails `validate`; `applyOnValidation: true` enables the transient retry; final output is successful. |
| GetResourceWithRetry / ResolveAndGetResource | Timeout, then 200 | Timeout observability produces the configured transient outcome; retry reaches 200. Confirm the live HTTP executor exposes it as documented. |
| GetResourceWithRetry / ResolveAndGetResource | Permanent 401 | Fail immediately; do not consume the transient retry budget. |
| GetResourceWithRetry / ResolveAndGetResource | Repeated 503 | Stop after maxCount; parent fails and reports the safe failure message. |
| WaitForOperation | RUNNING, SUCCEEDED (or COMPLETE) | Intermediate states pass validation; stop at successful terminal state and return it. |
| WaitForOperation | RUNNING, FAILED | Stop and fail explicit NOT_EQUALS FAILED validation; custom errors only explain failure. |
| WaitForOperation | Always RUNNING | Reach maxCount and fail because failOnMaxCount is true. |
| ProcessAppsBatch / ProcessItemsInParallel | Array wire value `["a","b"]` | `toArray` parses it; guard/count sees 2, not the string length. |
| ProcessAppsBatch / ProcessItemsInParallel | Empty array or skipped step | Guard skips empty collections; missing processed outputs default to an empty parsed array and count 0. |
| ProcessValidatedItems | Three names whose serialized JSON exceeds five characters | Count elements against maxItems, independently of serialization length; process each valid name in order. |
| ProcessValidatedItems | Empty list | Validation accepts it; iteration is skipped; results is [] and processedCount is 0. |
| ProcessValidatedItems | Too many items, blank/non-string member, or invalid maxItems | The validation executor fails before processing any item. |
| ChooseEnvironment | Each offered key: dev, staging, prod | Run reaches INPUT_REQUIRED; an authorized CHOOSE returns the key rather than its label; parent finishes with that environment and performs no environment mutation. |
| ChooseEnvironment | No answer | Run stays in attention until answered or an applicable limit/Auto-Abort intervenes; monitoring does not choose a default answer. |
| GetUserInput-based adaptation | Allowed and rejected free-text categories | Returned text is validated before a downstream operation; rejected text causes no downstream side effects. This adaptation is not part of ChooseEnvironment. |
| TriggerAndAwaitCommand | Child finishes | Exactly one child is created; its ID is reused by await and output read; parent returns FINISHED and the captured, parsed child output. |
| TriggerAndAwaitCommand | Child FAILED, ABORTED, or SUSPENDED | Await returns the configured state; FINISHED validation fails; childOutput does not run and the parent does not claim child success. |
| TriggerAndAwaitCommand | Child never reaches an accepted status | Await exhausts its deadline and fails without triggering another child. |
| Await/GetExecutionStatus-based adaptation | Missing target or child needs attention | Verify succeedOnNotFound and the accepted status set from live contracts; route found:false or PAUSED/INPUT_REQUIRED explicitly without an unconditional output read. |

Test the guard, iterator, expression conversions, and skip/output behavior through an
authorized Automation Pilot execution using the scenarios above. The original Open-Meteo evaluation recorded equivalent
guarded fan-out behavior, not an execution of this new template.

For the explicit child template, capture the GetExecutionOutput contract and representative
output to verify the `childOutput` conversion. Its child uses native `HttpRequest` to read
`https://example.com`; substitute an endpoint you control for deterministic HTTP outcomes.
For child failure/cancellation cases, use a
controlled child command or an authorized action on the disposable child; record the exact
configuration exercised. Keep each branch's live-run, captured-trace, or unverified evidence
label in the acceptance matrix.

After an ordering mismatch, correct the template from observed behavior and add that evidence
here. Avoid labeling a hand trace or stock jq evaluation as a service execution test.
