# Template acceptance scenarios

These templates are repository design examples reviewed on 2026-10-22, not recordings of
executions on the current service. Lint verifies structure and heuristics; only a controlled
run establishes SAP retry/validation/repeat ordering. Use a disposable command and an
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

Test the guard, iterator, expression conversions, and skip/output behavior through an
authorized Automation Pilot execution using the scenarios above. The original Open-Meteo evaluation recorded equivalent
guarded fan-out behavior, not an execution of this new template.

After an ordering mismatch, correct the template from observed behavior and add that evidence
here. Avoid labeling a hand trace or stock jq evaluation as a service execution test.
