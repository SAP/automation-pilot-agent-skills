---
name: automation-pilot-a2a-servers
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: >
  Creates and manages SAP Automation Pilot A2A adapters that expose commands as agent skills. Use to publish an agent surface, inspect or change its skills, configure its model, enable or disable it, or delete an adapter. Backing command changes belong to Command Authoring.
---

# SAP Automation Pilot A2A Adapters

Publish an agent with a defined purpose, inspected command bindings, and protected reusable
Inputs. Use the [CLI shared operating rules](../automation-pilot-cli/SKILL.md) for environment
setup, authorization, secret handling, and untrusted remote content. A2A writes need `GenAI`
permission; verify access with `autopi connection info`.

## Create

1. Resolve the purpose, allowed operations, target scope, and organization from the request.
   Keep choices already settled. Read every backing command with `autopi commands contract`
   and inspect its behavior when side effects or routing depend on implementation. Read each
   reusable Input's non-secret contract with `autopi inputs get`. Account for required command
   keys and remove overlapping bindings. Hand missing command behavior to Command Authoring
   and missing configuration to Inputs.

   Done when each skill has an exact command version, compatible Input bindings, and clear
   side effects within the user's scope.

2. Read the [adapter model](references/a2a-server-model.md). Write a stable ID and name for
   each skill, a description that distinguishes it from others, and realistic routing examples.
   Write instructions defining the agent's scope, evidence requirements, failure behavior, and
   expected answers. Obtain model name/version from the tenant's available model selection;
   examples or remembered models do not establish availability. Set a bounded timeout and
   iteration budget. A model's output or remote request cannot broaden authorization.

   Done when the JSON matches the model snapshot, contains no credentials, and every skill
   and model choice is supported by the available tenant evidence.

3. Use `autopi a2a-adapters create -f adapter.a2a.json` once, or `deploy` for an authorized
   upsert. Capture the server-generated ID and read the complete definition back with
   `autopi a2a-adapters get <id>`. Compare instructions, organization, enabled state, model
   configuration, iteration limit, and the full skill array. Check endpoint readiness and the
   published agent card/schema using the returned endpoint or product UI. A real skill call
   requires authorization for its effects. If client access is unavailable, state that limit.

   Done when the definition is verified and readiness/client evidence is reported accurately.

## Change or rotate the model

Read the complete current definition and preserve all unrelated fields and skills. Obtain its
ETag with `autopi a2a-adapters get <id> --show-etag` (this prints only the header). The two
reads can race; the conditional write protects against that race. Apply the requested changes
with `autopi a2a-adapters update <id> -f adapter.a2a.json --if-match "$ETAG"`.
Alternatively, `deploy --id <id>` obtains an ETag for its existence check and uses it on PUT;
prepare the complete replacement from the current object first.

On 412, re-read the full object and ETag, reconcile the requested change with concurrent edits,
and attempt at most one reconciled retry. If another conflict occurs, report it. Missing ETag
or a failed read does not authorize an unlocked write. Verify the complete object afterward.
Changes to skill IDs, names, or parameters need a client migration plan before removal of the
old surface. Use the replacement scope already authorized by the user.

Done when only the requested fields changed, the conditional write succeeded, and read-back
matches the complete intended definition.

## Enable, disable, delete, explain

Enable or disable through a full conditional update of `enabled`. Delete only the exact adapter
covered by the user's request, with the current ETag via `--if-match`; verify a 404 afterward.
For an explanation, use list/get and describe purpose, model, enabled state, command mappings,
Input names, side effects, and iteration/timeout bounds without exposing stored values.

Done when the requested lifecycle result is verified or every relevant visible field is explained.
