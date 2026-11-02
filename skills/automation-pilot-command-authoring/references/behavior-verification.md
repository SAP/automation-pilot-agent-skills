# Path evidence and behavioral verification

Read this when designing data flow between executors or defining acceptance checks for an
authored command. Contracts prove available keys; representative output proves nested paths
and runtime representation. Keep the evidence with the design or review artifact.

## Path evidence

For each prior-executor field used by an input, condition, validation, or final output mapping,
record its source. A live child contract establishes the top-level output key and declared
type. A sanitized representative execution establishes nested keys, sibling relationships,
and whether a structured value is encoded as a string. Field names alone establish none of
those relationships.

Use a compact table when several paths need tracking:

| Consumer | Path and conversion | Evidence | Remaining assumption |
|---|---|---|---|
| Application-list URL | `.resolveSpace.output.body` → `toObject.resources[0].guid` | Child contract declares body; captured response shows a string-encoded object with resources | Target space still resolves uniquely |
| Final results | `.processItems.output.outputs` → parse with an empty-array fallback | Iterator contract plus captured non-empty and skipped runs | None for those cases |

These rows illustrate the record; substitute the actual contract and execution identifiers.

Preserve the fixture's nesting and encoded values when handing an expression to Dynamic
Expressions. If representative output is unavailable, name the unverified path and include
a check for it in the acceptance scenarios. Local authoring can finish with that assumption
recorded; deployment/read-back alone does not resolve it.

## Branches and input partitions

Derive scenarios from the command's caller-visible behavior. Cover each conditional outcome
and a representative value from each input category that the command maps into one result.
Include absent optional data and empty collections when supported, guard rejection before
side effects, transient/permanent failure distinctions, and skipped-output fallbacks where
they change the result. Scale the matrix to the command; a straight-line command needs only
its applicable cases.

| Scenario | Expected outcome | Evidence obtained | Remaining check |
|---|---|---|---|
| Valid item list | Each item processed; output in requested order | Live execution ID and persisted output | None for this input category |
| Empty item list | Iterator skipped; empty results and count zero | Trace against sanitized captured output | Run empty-list case if authorized |
| Oversized item list | Guard fails before any item is processed | Unverified | Authorized guard-rejection run |

The table rows illustrate evidence labels, not recorded results for a particular command.
For a routing command, exercise each destination branch. For a classifier, sample each
category even when several categories produce the same label. Check status, output, and the
executors that ran; transport success alone does not establish workflow success.

## Match verification to the requested mode

- **Review:** identify missing path evidence or coverage and propose the focused checks.
- **Local authoring:** lint the file and include acceptance scenarios and unresolved contract
  or runtime assumptions.
- **Tenant authoring:** compare full read-back with the design. Perform runtime cases within
  the user's execution scope, capturing execution IDs and persisted output.

Label each case **live run**, **trace against captured output**, or **unverified**. When a
branch has no safe representative input within scope, trace it against sanitized evidence
if available and report the limit. A trace, offline lint, and a live run establish different
facts; report only the behavior supported by the evidence obtained.
