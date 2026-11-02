# Command Discovery

Read this when finding, inspecting, comparing, or reusing commands. It defines the command evidence, search method, fit model, dependency limits, and result format.

## Evidence from the CLI

Discovery funnels through three tools, from cheap and broad to full and narrow:

1. `autopi commands list` returns only the general fields of each command — `id` in `catalog:name:version` form, `catalog`, `name`, `description`, and `tags` — and filters by `--catalog` or `--tag`. Its default summary omits `inputKeys`, `outputKeys`, and `executors`; `--include-issues` requests validation issues. Use it to find candidates by name, description, and tags (including lifecycle tags).
2. `autopi commands contract` returns one command's caller-facing contract — `inputKeys`, `outputKeys`, version, tags, and the identifying fields — without the `executors` implementation. Call it on each shortlisted candidate to judge the contract.
3. `autopi commands get` returns the full definition, including the `executors` tree and `issues`. Call it only when conditions, retries, repeats, validation, child commands, or validation issues can change fit.

Resolve a candidate just far enough down this funnel to decide its fit: stop at the contract when the inputs and outputs settle the question, and read the full definition only when internal behavior matters.

## Searching

Search across these surfaces:

- The intended action and synonyms used by the product or target service.
- The exact resource type; an application, application instance, service instance, binding, deployment, and pod are distinct.
- Command name and description.
- Tags and relevant catalog ownership.
- Input keys that identify what the command acts on.
- Output keys required by a downstream step.

Custom content may use tenant-specific vocabulary. Search relevant custom catalogs and tags alongside SAP terminology. A failed name search is only one piece of evidence.

Scope by catalog or tag when possible to keep listings focused and relevant. Broaden the scope when the intended functional area is uncertain or a narrow search produces no credible candidate. Because `autopi commands list` returns only names and descriptions, confirm a candidate's input and output keys with `autopi commands contract` before relying on the contract.

## Evaluating fit

Weigh every candidate on:

1. **Intent:** whether it performs the requested outcome or only a related operation.
2. **Resource:** whether it targets the exact resource type.
3. **Contract:** whether the caller can safely supply every required input, including sensitive keys through reusable Inputs, and whether the outputs support downstream needs.
4. **Behavior:** whether waits, retries, conditions, validation, and child commands match the requested semantics.
5. **Abstraction:** whether a complete operation or a lower-level primitive best fits the intended use.
6. **Lifecycle:** whether it is current, deprecated, or points to a replacement.
7. **Ownership:** whether it is SAP-provided or tenant content; apply the same fit criteria to both.

Assign exactly one primary role:

- `DIRECT_USE` — performs the requested operation with a usable contract and behavior.
- `BUILDING_BLOCK` — performs a required sub-operation suitable for composition.
- `REFERENCE_IMPLEMENTATION` — is useful as a design example but not as the reusable operation itself.
- `NOT_SUITABLE` — fails a material intent, resource, contract, behavior, or lifecycle requirement.

Record secondary usefulness in notes rather than assigning multiple primary roles. Confidence reflects inspected evidence: `HIGH` requires the relevant contract (from `autopi commands contract`) and behavior (from `autopi commands get`) to be confirmed; `MEDIUM` means the name, description, and contract suggest fit but material internals remain unchecked; `LOW` means relevance is plausible from the name and description alone but the contract and suitability are unresolved.

## Dependencies and limits

`autopi commands get` can return an expanded live `executors` tree with `commandId` and
`subExecutors`; authored files express dependencies through `configuration.executors[].execute`.
Inspect the shape actually returned and follow both forms when internal behavior affects fit.
Track visited command IDs to stop cycles, and bound traversal (for example 50 unique commands).
Report unvisited dependencies when the bound is reached. Resolve each dependency's contract live. A child command can be a better building block than its parent.

The tools expose forward dependencies only. They cannot find every command that uses a given command, so discovery evidence cannot establish that a command is safe to change.

## Lifecycle and ownership

Lifecycle is expressed through service tags: `autopi:released`, `autopi:deprecated`, and `autopi:replacement`. Surface deprecation and the named replacement. Other tags are search evidence, not proof of behavior.

SAP-provided content commonly uses a `-sapcp` catalog suffix or `sapcp` catalog owner; tenant content uses its tenant-owned catalog. Treat these as ownership evidence, not quality rankings.

## Reporting

For one strong result, report:

```text
Command:    <catalog:name:version>
Ownership:  <SAP-provided | tenant>
Role:       <primary role>   Confidence: <HIGH | MEDIUM | LOW>
Match:      <evidence tied to intent and resource>
Inputs:     <required keys, types, and sensitive key names without values>
Outputs:    <relevant results>
Behavior:   <waits, retries, validation, or dependencies that affect reuse>
Lifecycle:  <current | deprecated -> replacement | unresolved>
Next use:   <trigger, compose, inspect further, or reject>
```

For multiple candidates, start with a compact comparison table and expand only the constraints that distinguish them. For a no-result, state the search terms, catalogs or tags checked, evidence limits, and closest building blocks.

For a command-authoring handoff, include each candidate's exact ID, ownership, primary role, match reason, input and output contracts, lifecycle, inspected dependencies, and unresolved notes.
