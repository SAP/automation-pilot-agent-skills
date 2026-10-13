# Reusable Input Discovery

Read this when finding, inspecting, or recommending Automation Pilot Inputs. An Input entity stores configuration or credentials that can supply one or more command input keys; it is distinct from a command's input-key definition.

## Evidence from the tools

`automation-pilot:list_inputs` filters by `catalog_id` or tags and returns only the general fields — `id`, `catalog`, `name`, `description`, and `tags` — so use it to find candidates by name, purpose, and tags. `automation-pilot:get_input` reads one exact `catalog:name:version` ID and returns the full non-secret metadata. Both deliberately omit stored values.

`list_inputs` surfaces for each candidate:

- `id`, `catalog`, `name`, and `description` for identity and purpose.
- `tags` for categorization evidence.

`get_input` adds, for one Input:

- `keys` for supplied key names, types, descriptions, and sensitivity.

A command input key's `defaultValueFromInput` is a discovery lead. Confirm the referenced Input with `automation-pilot:get_input` before recommending or binding it.

## Evaluating fit

Compare the Input's key contract with the target command contract. Confirm that required key names
and types match, sensitivity is appropriate, and the Input's described purpose and relevant tags fit
the target environment. Metadata can establish contract compatibility but cannot prove that the
hidden stored values are current or correct.

## Reporting

For each useful Input, report its exact ID, purpose, ownership, relevant tags, supplied keys with
types and sensitivity, target command keys it satisfies, unresolved gaps, and whether it is suitable
to bind. If none fits, state the catalogs, tags, and key names searched.
