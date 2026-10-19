# Command-authoring troubleshooting

Read this after deployment validation or a runtime failure identifies an authoring symptom. Each
next action names the owning material. Load it directly from the skill's reference table rather
than treating this table as another copy of its contract.

| Symptom | Likely cause | Next action |
|---|---|---|
| Validation rejects a key or executor, or returns an issue | Command schema or live child contract mismatch | Read `automation-pilot:get_command` issues and compare the affected shape with the command model, composition controls, and the live child contract |
| Expression is too large or resolves incorrectly | Expression-engine scope, limit, or serialization issue | Reproduce it with the Dynamic Expressions skill against an execution-shaped fixture |
| A nested field is null after crossing a command or HTTP boundary | Encoded object or array was read without parsing | Apply the boundary rule owned by the command model and the relevant executor reference |
| A conditionally skipped step has no output | Final mapping assumes every executor ran | Add a fallback of the declared output type using the command model |
| A generated command uses an SAP catalog suffix | Tenant catalog naming was confused with an SAP-provided catalog | Move it to the intended tenant catalog using the command model |
| A metadata region ID returns 404 from `get_command` | The ID belongs to a reusable Input | Resolve it with `automation-pilot:get_input` |
| HTTP retry or validation cannot observe a status | The HTTP executor fails before the control block runs | Follow the HTTP reference's non-success and timeout observability rules |
| Sequential or parallel iteration rejects its payload | Executor version, list type, element alias, or mapping shape is wrong | Resolve the live iterator contract and apply the iteration reference |
| Deployment succeeds but behavior is wrong | Static validation cannot prove ordering, expressions, or integration | Inspect the representative execution, then return the focused defect to the owning reference or skill |

