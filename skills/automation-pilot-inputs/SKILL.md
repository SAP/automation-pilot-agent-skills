---
name: automation-pilot-inputs
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: >
  Creates and manages reusable SAP Automation Pilot Inputs that store configuration and credentials. Use when a user wants to create, change, rotate, explain, or delete an Input. Do not use merely to find an existing Input for a command.
---

# SAP Automation Pilot Inputs

Before API work, read the [shared CLI operating rules](../automation-pilot-cli/SKILL.md)
for setup, authorization, secret handling, and response safety.

## Goal

Maintain reusable values without exposing secrets or breaking commands, schedules, executions, or MCP tools that bind them.

## Required inputs

- The Input's purpose and target catalog.
- Its key contract: names, types, descriptions, sensitivity, and the values the user has already authorized the agent to transmit.

For an existing Input, require its exact `catalog:name:version` ID. Ask about omitted choices only
when they change the key contract, sensitivity, or stored values.

## Workflow

### Creating an Input

#### Step 1: Design the contract

Choose one cohesive reusable purpose and define no more than 32 keys. Use PascalCase for the Input name and camelCase for keys. Mark every credential, token, private key, certificate, or secret-bearing structure sensitive. Read the [Input model](references/input-model.md) before assembling the definition.

Done when the catalog, unique name, purpose, complete key set, types, descriptions, sensitivity flags, and intended values are each accounted for.

#### Step 2: Handle values safely

Serialize every stored value as a JSON string regardless of its declared type. Keep secrets out of descriptions, tags, reports, and examples. Use an existing user-authorized secret source (protected file, environment variable, or secret
manager) without printing it or reading it into conversation. Assemble JSON within a local
process and pipe it to `autopi inputs create -f -` or `inputs patch -f -`; never interpolate a
secret into argv or enable shell tracing. Follow the shared CLI operating rules for protected
temporary payloads and cleanup. If no authorized source exists, finish the non-secret contract
and let the user populate values through the product UI. Do not request secrets in chat.
Run `autopi lint <Name>.input.json` on the non-secret definition before creating it.

Done when each value matches its declared type after parsing, sensitive keys are protected, and no secret is repeated in user-facing output.

#### Step 3: Create and verify

Call `autopi inputs create` once, capture the returned ID, then read it with `autopi inputs get`. Compare the catalog, name, version, description, key contract, sensitivity, and tags. Stored values cannot be read back; report value presence only from the successful write, never from the metadata read.

Done when the exact ID is captured and the non-secret read-back matches the intended definition without claiming that hidden values were retrieved.

### Changing or rotating an Input

Read the current Input with `autopi inputs get`, compare its visible contract and metadata
with the requested change, then choose the narrowest mutation.

- Use `autopi inputs patch` for value rotation, descriptions, tags, and additive changes. Its JSON Merge Patch semantics preserve omitted members.
- Use `autopi inputs update` only for an intentional full replacement whose complete values are available; metadata returned by `autopi inputs get` cannot reconstruct hidden values.

Read the Input back and verify every visible changed field. Treat a successful write response as the only evidence for hidden value changes and report only the affected key names.

Done when omitted values were preserved, visible metadata matches the request, and secret values were
neither retrieved nor disclosed.

### Deleting an Input

Deletion with `autopi inputs delete` can break every dependent command, schedule, execution request, or MCP tool, and the CLI cannot prove the absence of reverse dependencies. Require an exact ID and disclose that evidence limit. Proceed only when the user's request already acknowledges that dependency risk or after the user explicitly confirms it. Verify deletion with a `404` from `autopi inputs get`; authentication, authorization, network, and other failures do not prove absence.

Done when the target and dependency risk are explicit and absence is verified after deletion.

### Explaining an Input

Find an indirect reference with `autopi inputs list` when needed, then read it with
`autopi inputs get`. Report its purpose, ownership, relevant tags, and key names with types,
descriptions, and sensitivity. State that stored values are intentionally unavailable; never infer
or request them merely to explain the Input.

Done when every visible key and relevant metadata field is covered without revealing or inventing
stored values.

## Handoffs

Discovery finds compatible existing Inputs. Command Authoring defines command input keys and embedded
Input references. Executions, Scheduling, and MCP Server Management bind existing Input IDs. This
skill owns creation, mutation, credential rotation, explanation, and deletion.

## Definition of done

The Input has one exact ID and a verified non-secret contract, every secret-bearing key is sensitive,
and values use the correct wire representation. Hidden values remain undisclosed, focused updates
preserve omitted values, and every write is verified.
