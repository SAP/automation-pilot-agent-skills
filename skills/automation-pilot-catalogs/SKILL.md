---
name: automation-pilot-catalogs
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: >
  Creates and manages custom SAP Automation Pilot catalogs. Use when a user wants a new catalog, changes its metadata, or requests catalog deletion. Discovery owns finding commands and Inputs within catalogs.
---

# SAP Automation Pilot Catalogs

Before API work, read the [shared CLI operating rules](../automation-pilot-cli/SKILL.md)
for setup, authorization, and response handling.

## Decide whether a catalog is needed

Prefer an existing custom catalog when it already has the same purpose and lifecycle.
Create a new one for durable separation such as a distinct team, product, environment,
or content lifecycle. Creating a command or Input alone does not call for a new catalog.
Preserve a target already settled by the user.

Inspect owned catalogs with `autopi catalogs list --own`. Before a plan creates catalogs,
check the applicable tenant capacity when available and compare it with the current
inventory. A quota rejection calls for an explicit reuse or revised placement decision;
keep existing content intact and report unknown capacity as a feasibility limit.

## Create a catalog

Read the [catalog model](references/catalog-model.md) for naming, payload fields, and a
create example. Resolve the technical name, display name, and purpose from the request.
Run `autopi lint --kind catalog catalog.json`, then create with
`autopi catalogs create -f catalog.json` within the established scope.

Capture the returned ID; Automation Pilot generates it from the technical name and tenant.
Read `autopi catalogs get <id>` and compare the technical name, display name, description,
and any other authored metadata. For a conflict or ambiguous response, inspect the existing
catalog before another create; use the returned ID rather than guessing a suffix.

Done when the intended catalog exists and the read-back matches, or local-only work has
a validated payload and explicit live feasibility assumptions.

## Change catalog metadata

Read the current full definition with `autopi catalogs get <id>`. Use the
[catalog model](references/catalog-model.md) for editable fields and update validation.
Preserve untouched metadata, validate the replacement, and send
`autopi catalogs update <id> -f catalog-update.json`. Read back the complete result.
Changing the technical name requires moving content to a new catalog; plan that migration
only when requested. Display-name changes keep the existing catalog ID.

Done when the requested metadata changed and untouched fields are preserved in read-back.

## Delete a catalog

Read the exact catalog and inspect its commands and Inputs. Before deleting nonempty
content, use **Discovery** to inspect its dependencies and document the removal impact.
The CLI cannot prove the absence of reverse dependencies; report any unresolved impact
instead of treating the inventory as proof that nothing else depends on it. Use the
[deletion rules](references/catalog-model.md#deletion) to distinguish an empty-catalog
delete from removal of its contained content. Apply the deletion already authorized by
the user; resolve any missing cascade scope after describing the concrete contents.
Verify the catalog is absent after the delete.

Done when dependency impact and any evidence limits are explicit, and the exact requested
catalog and authorized contents are removed, supported by deletion results and a
subsequent 404.
