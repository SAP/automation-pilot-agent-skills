---
name: automation-pilot-discovery
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: >
  Finds and evaluates existing SAP Automation Pilot catalogs, commands, child dependencies, and reusable Inputs. Use when a user wants to locate something to run, check for an existing capability before building, identify reusable building blocks, or understand a command's contract and behavior. Do not use to create, change, or execute content.
---

# SAP Automation Pilot Discovery

Before API work, read the [shared CLI operating rules](../automation-pilot-cli/SKILL.md)
for setup, authorization, secret handling, and response safety.

## Goal

Find the best-fitting existing content for the intended automation, and support every recommendation with inspected evidence.

## Required inputs

- The intended outcome or content to inspect.

Optional:
- Technology, resource type, catalog, tags, or ownership constraints.
- Whether the result will be triggered directly, composed into a command, or used as a reference.
- Required inputs, outputs, waiting behavior, validation, or other execution semantics.

Ask about omitted constraints only when they change what counts as a fit. Otherwise search broadly enough to test the likely interpretations.

## Workflow

### Finding commands or building blocks

#### Step 1: Frame the search

Translate the request into search facets: intended action, exact resource type, required contract, important behavior, and lifecycle or ownership constraints. Search for the outcome rather than one guessed command name.

Done when the intended outcome, resource type, caller-facing contract, behavior, and material lifecycle or ownership constraints are either recorded or explicitly marked unconstrained.

#### Step 2: Choose the scope

Use `autopi catalogs list` to identify likely functional areas, then call `autopi commands list` with the narrowest justified catalog or tag scope. Search across the tenant when no area is clear or scoped results do not cover the intent.

Done when the initial catalog/tag scope has an evidence-based reason and the condition for broadening beyond it is explicit.

#### Step 3: Inspect and evaluate candidates

Read [command discovery](references/commands.md). Shortlist candidates from the names and descriptions `autopi commands list` returns, then call `autopi commands contract` to judge each shortlisted command's input/output contract and tags. Use `autopi commands get` only when behavior, executor wiring, or child-command dependencies affect fit. Continue until the plausible search terms and relevant scopes have either produced evaluated candidates or established a useful no-result.

Done when every plausible shortlisted candidate has a live contract assessment or a recorded exclusion reason, and no unevaluated result could reasonably change the recommendation.

#### Step 4: Return the evidence

Assign each candidate one primary role from the command reference, state confidence from the evidence actually inspected, and surface the constraints that matter to reuse. Report the search scope and closest building blocks when no direct match exists.

Done when each recommendation has an exact ID, primary role, inspected evidence, confidence, reuse constraints, and next action; a no-result names the searched scope and closest alternatives.

### Inspecting a known command

Read [command discovery](references/commands.md), then call `autopi commands get` with the exact ID. Explain its purpose, contract, behavior, dependencies, lifecycle, ownership, and suitability for the user's intended use. Keep sensitive values hidden.

Done when the explanation accounts for the live contract, implementation-relevant behavior, dependencies, lifecycle, ownership, and fit for the stated use.

### Finding or inspecting reusable Inputs

Read [reusable Input discovery](references/inputs.md). Use `autopi inputs list` to find candidates and `autopi inputs get` to confirm an exact ID and key contract. Recommend an Input only when its non-secret metadata shows that it supplies the required command keys with compatible types and sensitivity.

Done when every recommended Input has an exact ID and inspected key metadata proving type and sensitivity compatibility, or the no-result records the scopes and key names searched.

### Browsing catalogs

Call `autopi catalogs list` and report the catalogs relevant to the user's purpose, including ownership and documentation links when present. Use a catalog's exact `id` for subsequent command or Input filters.

Done when every reported catalog is tied to the stated purpose and carries the exact ID and available ownership or documentation evidence.

## Handoffs

Send a direct-use command to executions for triggering. Send command building blocks or a useful command no-result to command authoring. Send an Input no-result to Inputs when a reusable value set must be created. Send a request for custom catalog creation or metadata changes to Catalogs with the inspected inventory and placement constraints. Discovery supplies evidence and IDs; it does not execute, deploy, release, or modify content.

## Error handling

A failed or narrow search does not prove that content is absent. Report tool failures and the scopes, filters, and terms actually checked. Broaden the search when justified; otherwise label the conclusion with its evidence limits.

## Definition of done

Every recommended result has an exact ID, primary role, evidence-based confidence, relevant reuse
constraints, and a clear next use. A no-result records the searched scope and the closest reusable
content. Sensitive values remain hidden.
