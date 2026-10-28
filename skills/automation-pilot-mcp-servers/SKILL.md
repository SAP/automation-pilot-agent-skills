---
name: automation-pilot-mcp-servers
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: >
  Creates and manages SAP Automation Pilot MCP servers with simplified tool contracts and reusable Input bindings. Use to publish commands as tools, verify injected configuration and client schemas, or change server and tool availability. Command Authoring owns backing-command changes; Executions owns run diagnosis.
---

# SAP Automation Pilot MCP Servers

SAP's [administration guide](https://help.sap.com/docs/automation-pilot/automation-pilot/mcp-server)
requires `Read` to inspect MCP servers and `Write` to create or edit them; backing runs require
`Execute`. Inspect the configured tenant with `autopi connection info` and use endpoint evidence
to resolve access failures. Apply the user's existing publication scope.

Before API work, read the [shared CLI operating rules](../automation-pilot-cli/SKILL.md)
for setup, authorization, secret handling, and response safety.

## Goal

Publish a stable, safe client-facing tool surface backed by existing Automation Pilot commands, and keep that surface correct over its lifetime.

## Required inputs

- The server's purpose.
- The operations to expose and a resolvable backing command for each operation.

Optional:
- Server and tool names.
- Reusable Inputs to inject into each command.
- Tool hints, synchronous wait behavior, and enabled state.

Ask about omitted choices that change the published behavior or credential binding; otherwise derive clear names, titles, hints, and instructions from the resolved commands.

## Workflow

### Creating a server

#### Step 1: Resolve the tool set

Resolve every backing command with `autopi commands list` and `autopi commands contract`, reaching for `autopi commands get` only when a command's behavior must be confirmed. Confirm that each command performs the promised operation and that its description and input-key descriptions make a useful client-facing tool schema. If no command fits, hand the unresolved requirement to command authoring.

Done when every proposed tool has one exact backing command ID, an inspected live contract, and evidence that its behavior matches the promised operation.

#### Step 2: Bind injected Inputs

For values Automation Pilot should inject at call time, find reusable Inputs with `autopi inputs list` and confirm their metadata with `autopi inputs get`. Add their IDs to the tool's `inputReferences`. Confirm that the referenced keys match the command's input contract; referenced keys disappear from the client-facing schema. Keep stored values hidden and never inline credentials in the server definition. If no compatible Input exists, hand the required key contract to Inputs for creation before publishing the tool.

Run `autopi mcp-servers validate -f server.mcp.json` against the proposed definition.
The Go CLI performs metadata-only reads, checks type and sensitivity, rejects overlapping
bindings and unbound sensitive keys, and prints contract summaries. Use `--metadata metadata.json`
for offline review; see [binding verification](references/binding-verification.md).

Done when every injected Input exists, supplies compatible keys, and leaves only the intended
caller parameters. Hidden stored values still require an authorized runtime check.

#### Step 3: Design the client-facing surface

For each tool, derive its name, title, behavioral hints, enabled state, and wait behavior from the command's real behavior. Write server `instructions` that explain what the server fronts, when to use each tool, how related tools fit together, and any safety or workflow constraints.

Apply these rules:

- Use a unique, verb-prefixed `snake_case` tool name and a `kebab-case` server name. Server and tool names are immutable after creation. Changing either name requires creating a replacement server or tool and removing the old one, which breaks clients bound to the old name.
- Treat `readOnlyHint`, `destructiveHint`, and `idempotentHint` as client guidance, not enforcement. A dangerous operation needs its confirmation step in the backing command.
- Use `sync: true` with `waitTimeSeconds` only for commands expected to finish within the bound. Use asynchronous mode for commands with delays, polling, or user-interaction steps. A synchronous timeout falls back to an execution ID for polling.
- Reserve `get_execution` for the execution-polling tool Automation Pilot adds to every server.
- Keep within 50 tools per server.

Before assembling or editing a definition, read the exact schema and field rules in the [MCP server model](references/mcp-server-model.md) and adapt its payload. Listing, deleting, or giving a high-level explanation does not require loading that reference.

Done when the server and tool names are unique and immutable, instructions and schemas describe actual behavior, hints match side effects, and synchronous wait settings fit the command's expected duration.

#### Step 4: Write and verify

Call `autopi mcp-servers create` once with the assembled body. Read the created server back by its returned `id` and compare its name, instructions, enabled state, complete tool set, command mappings, injected Inputs, hints, and wait behavior with the request.

After read-back, use the [JSON-RPC client recipe](references/client-verification.md) to check
the endpoint and capture `tools/list`. Run CLI validation again with
`--client-schema tools-list.json`; it checks injected keys are absent, remaining parameters
match, disabled tools are hidden, and `get_execution` is available.
A real tool call requires user authorization for its effects. If client access is unavailable,
report definition verification separately from endpoint/client validation.

Done when the server ID is captured, read-back matches the definition, and readiness/client
validation is completed or its specific evidence limit is reported.

### Changing a server

Resolve an indirect reference with `autopi mcp-servers list` when needed. Read the current definition with `autopi mcp-servers get`, change only the requested editable properties, and send the full object with `autopi mcp-servers update`. Preserve every unchanged tool and field because the update replaces the complete definition and `mcpTools` array.

Resolve the backing command and injected Inputs before adding or remapping a tool. Neither a server name nor an existing tool name can be edited. A new name requires creating a replacement server or tool and removing the old one. Prepare a migration that lists affected clients and parameter changes before replacing or
removing a tool. Keep the old surface until clients migrate when that is compatible with the
request; use existing authorization for the exact replacement scope. Read the result back and verify it.

Done when requested editable fields changed, all untouched tools and fields are preserved, immutable-name replacements are explicit, and the read-back confirms the full definition.

### Enabling, disabling, deleting

Set the server's `enabled` to `false` to take its endpoint offline without deleting it. Set a tool's `enabled` to `false` to keep it in the definition while hiding it from clients. Re-enable by setting the same field to `true` and writing the full server definition with `autopi mcp-servers update`.

`autopi mcp-servers delete` permanently removes a server and requires an unambiguous target. If "turn off" could mean disabling a server, disabling one tool, or deleting the server, ask which outcome the user wants.

Done when the exact target and lifecycle action are unambiguous and the resulting server/tool state or confirmed deletion is verified.

### Explaining a server

Resolve an indirect reference with `autopi mcp-servers list` when needed, then read it with `autopi mcp-servers get`. Describe its purpose, enabled state, tool-to-command mappings, injected Input names, state-changing tools, and synchronous or asynchronous behavior in user terms. Keep sensitive Input values hidden and avoid dumping raw JSON unless the user requests it.

Done when every exposed tool is mapped to its command and non-secret injected Input names, side effects and wait behavior are clear, and server/tool enabled states are reported.

## Definition of done

The published surface matches inspected command and Input contracts, uses stable client-facing
names and accurate hints, and keeps credentials out of the definition. A create or update is read
back and matches the complete intended server. Lifecycle actions are verified, and explanations
account for every exposed tool without revealing stored values.
