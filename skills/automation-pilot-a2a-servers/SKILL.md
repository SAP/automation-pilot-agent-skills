---
name: automation-pilot-a2a-servers
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: >
  Creates and manages SAP Automation Pilot A2A adapter (agent) definitions that expose commands as agent skills. Use when a user wants to publish commands as an agent-to-agent surface, inspect or change an adapter's skill set, enable or disable an adapter, or delete one. Do not use to run an exposed skill or author or fix its backing command.
---

# SAP Automation Pilot A2A Adapters

## Goal

Publish a stable, safe agent-to-agent surface backed by existing Automation Pilot commands —
an AI agent whose *skills* each map to a command — and keep that surface correct over its
lifetime. An A2A adapter carries an LLM configuration, a natural-language instruction prompt,
and a set of skills, so external AI agents can delegate work to it.

## Required inputs

- The agent's purpose and the scope it reasons over.
- The operations to expose and a resolvable backing command for each operation.

Optional:
- Adapter and skill names.
- Reusable Inputs to inject into each command as credentials.
- Model configuration, `maxIterations`, and enabled state.

Ask about omitted choices that change the published behavior or credential binding;
otherwise derive clear names, descriptions, examples, and the instruction prompt from the
resolved commands.

All backend calls use the `autopi` CLI — see
[`../automation-pilot-cli/SKILL.md`](../automation-pilot-cli/SKILL.md) for setup (the
`AUTOPI_*` environment variables, regional hosts, and command surface). Write operations
require the `GenAI` permission.

## Workflow

### Creating an adapter

#### Step 1: Resolve the skill set

Resolve every backing command with `autopi commands list` and `autopi commands contract`,
reaching for `autopi commands get` only when a command's behavior must be confirmed. Confirm
that each command performs the promised operation and that its description makes a useful
`description` and `examples` for the agent to route on. If no command fits, hand the
unresolved requirement to command authoring.

Done when every proposed skill has one exact backing command ID, an inspected live contract,
and evidence that its behavior matches the promised operation.

#### Step 2: Bind injected Inputs

For credentials Automation Pilot should inject at call time, find reusable Inputs with
`autopi inputs list` and confirm their metadata with `autopi inputs get`. Add their IDs to
the skill's `inputReferences`. Confirm the referenced keys match the command's input
contract. Keep stored values hidden and never inline credentials in the adapter definition.
Reuse one credential reference across all skills of the same service. If no compatible Input
exists, hand the required key contract to Inputs for creation before publishing the skill.

Done when every injected Input exists, supplies compatible keys, and no stored credential is
exposed in the definition.

#### Step 3: Design the agent surface

For each skill, derive its `id` (kebab-case, no spaces), `name` (snake_case, verb-prefixed),
`description`, `examples`, and `tags` from the command's real behavior. Write the adapter's
`instructions` job prompt — role, the reasoning the agent must perform, and the required
response shape — and set `modelConfiguration` and `maxIterations` to fit the agent's job.

Apply these rules:

- A skill `id` is immutable in effect: it identifies the skill, so keep it stable and unique.
- A2A skills carry **no hint booleans**. The agent reasons from `description`, `name`, and
  `examples`, so author those three carefully. A dangerous operation needs its confirmation
  step in the backing command.
- Raise `maxIterations` toward 15 and `timeout` toward 60 only for agents that chain many
  skills to correlate data and reason to a root cause; keep both low for simple adapters.
- `tags` is an array of strings, not an object.

Before assembling or editing a definition, read the exact schema and field rules in the
[A2A adapter model](references/a2a-server-model.md) and adapt its payload. Listing,
deleting, or giving a high-level explanation does not require loading that reference.

Done when skill ids and names follow the conventions, the instruction prompt and skill
schemas describe actual behavior, and model/iteration settings fit the agent's job.

#### Step 4: Write and verify

Call `autopi a2a-adapters create -f <file>` once with the assembled body; the response
carries an `ETag`. Read the created adapter back by its returned `id`
(`autopi a2a-adapters get <id> --show-etag`) and compare its name, instructions, enabled
state, model configuration, complete skill set, command mappings, and injected Inputs with
the request. `deploy -f <file>` upserts (GET-then-PUT/POST, no `If-Match`) when you want
create-or-update in one step.

Done when the adapter ID is captured and the complete read-back matches every intended
adapter and skill field.

### Changing an adapter

Resolve an indirect reference with `autopi a2a-adapters list` when needed. Read the current
definition with `autopi a2a-adapters get <id> --show-etag`, change only the requested
properties, and send the full object with `autopi a2a-adapters update <id> -f <file>`.
Preserve every unchanged skill and field because the update replaces the complete
definition and `skills` array. Pass the current ETag via `--if-match` for optimistic
locking — the update is rejected with `412` if the resource changed since you read it; on a
`412`, re-fetch for the fresh ETag and retry.

Resolve the backing command and injected Inputs before adding or remapping a skill. Warn
before removing or renaming a skill because external agents may depend on its `id` and
schema. Read the result back and verify it.

Done when requested fields changed, all untouched skills and fields are preserved, and the
read-back confirms the full definition.

### Enabling, disabling, deleting

Set the adapter's `enabled` to `false` to take it offline as a draft without deleting it,
and back to `true` to reactivate it (write the full definition with
`autopi a2a-adapters update`). `autopi a2a-adapters delete <id>` permanently removes an
adapter and requires an unambiguous target. If "turn off" could mean disabling or deleting,
ask which outcome the user wants.

Done when the exact target and lifecycle action are unambiguous and the resulting state or
confirmed deletion is verified.

### Explaining an adapter

Resolve an indirect reference with `autopi a2a-adapters list` when needed, then read it with
`autopi a2a-adapters get <id>`. Describe its purpose, enabled state, skill-to-command
mappings, injected Input names, the agent's instruction prompt in user terms, and its model
and iteration settings. Keep sensitive Input values hidden and avoid dumping raw JSON unless
the user requests it.

Done when every exposed skill is mapped to its command and non-secret injected Input names,
the agent's job and response behavior are clear, and the adapter's enabled state is reported.

## Definition of done

The published surface matches inspected command and Input contracts, uses stable skill ids
and a clear instruction prompt, and keeps credentials out of the definition. A create or
update is read back and matches the complete intended adapter. Lifecycle actions are
verified, and explanations account for every exposed skill without revealing stored values.
