# Capability and verification map

Use this when choosing an owning skill, permission check, payload, or verification step.
The API and schema guidance are dated repository snapshots. Read live command/Input
contracts before binding them; unknown permissions or state rules require service evidence.

| User operation | Owning skill | CLI verbs | Access | Payload | Verification |
|---|---|---|---|---|---|
| Inspect tenant | Connection | `connection info` | Tenant read access | None | Non-secret identity and task-specific permissions |
| Find command/Input | Discovery | `catalogs list`, `commands list/contract/get`, `inputs list/get` | Read | None | Exact ID, live contract, dependency evidence |
| Manage custom catalog | Catalogs | `catalogs create/get/update/delete` | Read for inspection; Write for management | catalog or catalog-update model | Returned ID and full metadata read-back; contents/dependency impact and 404 for deletion |
| Review/local command | Command Authoring | `lint` | None locally; Read for live contracts | `Name.command.json` | Prioritized findings or lint plus assumptions |
| Deploy command | Command Authoring | `commands create/update/deploy` | Write (Read for deploy check) | command model | Full get, blocking issue resolution; authorized run for runtime proof |
| Create/rotate Input | Inputs | `inputs create/patch/update` | Write | input model, string values | Non-secret get; hidden value changes confirmed only by successful write |
| Trigger command | Executions | `executions trigger` | Execute | commandId, string inputValues, inputReferences, tags | Capture ID; get/output; investigate ambiguous failures before repeats |
| Control run | Executions | `executions action/bulk-action/actions/delete` | Execute for actions; Write for deletion | type, required reason, action-specific value | State and recorded actions; bulk per-item outcomes; 404 after delete |
| Schedule run | Scheduling | `scheduled-executions create/update/enable/disable/get/time-zones` | Read for inspection; Execute for writes | schedule type and input bindings | Complete get, predicted calendar times, actual trigger evidence when available |
| Manage webhook | CLI webhook reference | `webhooks list/get/create/update/delete/validate` | Read for inspection/validation; Write for administration | webhook model with command/Input/event bindings | Schema and binding validation; complete get and ETag reconciliation |
| Trigger webhook | Executions with CLI webhook reference | `webhooks trigger` | Execute | optional generic event or required ANS event | Capture execution ID; inspect state/output when requested |
| Publish MCP tools | MCP Servers | `mcp-servers create/update/deploy/get` | Read/Write; Execute for backing runs | MCP server model | Complete get plus binding checker and endpoint/client-visible tool schema |
| Apply feature tags | Tags | builder `--tag`, `--logs`; `policy`, `lint` | Permission of owning write | Shared string tag map | Registry snapshot and live read-back |
| Craft expression | Dynamic Expressions | `lint` heuristics; authorized execution via Executions | None for hand traces | Expression and wire-shaped fixture | Predicted output; record actual engine version/result for runtime proof |

`api` is an authenticated escape hatch, not proof that an undocumented endpoint or action
exists. It inherits the shared output policy and offline preview rules. Planning resolves
consequential choices and feasibility; it has no mutation endpoint of its own.
