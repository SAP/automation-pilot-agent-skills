# MCP client verification

Use this after server read-back, or when validating a client integration. Authentication
and the host come from the shared CLI environment variables. All Automation Pilot requests
use `autopi`, including JSON-RPC.

## Connect and inspect

The endpoint is `/api/v1/mcp/<server-id>` on the configured host. The published definition
may contain no endpoint field; use its returned ID. SAP documents the endpoint in its
[Joule Studio integration procedure](https://help.sap.com/docs/automation-pilot/automation-pilot/integrating-service-with-joule-studio).

Save `initialize.json`:

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"autopi-verifier","version":"1.0"}}}
```

This protocol version is the one exercised in the repository evaluation; verify the
server's returned version. The evaluated endpoint accepted independent requests without a
session header. If a tenant negotiates stateful transport, use a client that preserves the
session header and sends `notifications/initialized`; independent CLI requests do not
establish compatibility with that transport.

```sh
autopi api POST mcp/SERVER_ID -f initialize.json --header 'Accept: application/json, text/event-stream'
```

Save `tools-list-request.json`:

```json
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
```

```sh
autopi api POST mcp/SERVER_ID -f tools-list-request.json --header 'Accept: application/json, text/event-stream' > tools-list.json
```

Inspect `result.tools[].inputSchema`: compare properties with backing-command keys minus
injected keys, confirm requiredness and types, and check disabled tools are absent. Check
the full list, including Automation Pilot's generated `get_execution` tool. For collections,
inspect `items`; an unconstrained item schema requires a documented command-side guard or
a wrapper with flat scalar parameters. Hints describe actual side effects and external data
access; verify `openWorldHint` for tools consulting an external API.

## Call within the authorized scope

Save `tool-call.json`, using the actual tool name and its remaining caller parameters:

```json
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_city_weather","arguments":{"city":"Sofia"}}}
```

```sh
autopi api POST mcp/SERVER_ID -f tool-call.json --timeout 75s --header 'Accept: application/json, text/event-stream'
```

The 75-second client deadline exceeds the maximum 60-second synchronous server wait.
Prefer async mode for consistently longer commands. A returned execution ID means the
run was accepted, not that it succeeded. A timeout after submission is ambiguous: locate
the accepted run before considering another call.

Read `result.structuredContent`, or parse JSON in a `result.content` text block. Check
JSON-RPC `error`, tool `isError`, and the contained execution `status` separately.
`isError: false` can wrap `status: FAILED`.

For an active execution, call the generated tool with its returned ID:

```json
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"get_execution","arguments":{"executionId":"RETURNED_EXECUTION_ID"}}}
```

Poll at bounded intervals with an overall deadline. Pause for user-interaction states and
report terminal failure or abort explicitly. Claim success only for `FINISHED` with outputs
matching the acceptance criteria. Declared arrays/objects can arrive as JSON strings in
`outputValues`; parse according to the backing contract before semantic assertions.

Record definition read-back, visible schema, execution ID, terminal state, and relevant
output checks separately. Definition verification alone does not establish client access
or runtime behavior.
