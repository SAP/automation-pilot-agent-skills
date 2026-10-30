# Execution History

Read this for execution lookup, health sweeps, reliability questions, and history analytics.

## Filters and returned fields

`autopi executions list` supports repeated --status, --command-id, --execution-id,
--originator-id, and --tag filters, plus --started-after, --started-before, and --limit.
--filter-mode contains provides case-insensitive partial identifier matching; exact is
the default. Each repeated flag takes one value. Use multiple status values for active
sets such as RUNNING, PAUSED, INPUT_REQUIRED, and FAILED.

Each returned summary includes the execution ID, command ID, originator ID when present, tenant ID, tags, start and modification times, status, progress, progress message, size, and used quota. It does not carry the detailed executor array or a reliable failure explanation; call `autopi executions get` for a failed run whose cause the user wants explained.

## Time windows

`--started-after`, `--started-before`, `startTime`, and `modificationTime` use epoch milliseconds (the flags also accept a relative form such as `24h`). Translate relative wording into exact bounds before filtering. Distinguish a rolling duration such as "last 24 hours" from a calendar period such as "since Monday", and interpret calendar boundaries in the user's time zone.

Use a narrow enough window and limit to answer the question without fetching unrelated history. If a result could be truncated by `--limit`, say so rather than presenting it as a complete population.
The API returns at most 1000 list results and documents no continuation cursor.
For status totals over a time window, use `executions summary --started-after ...
--started-before ...`; disclose the scope and avoid inferring complete history from a limited list.

## Analysis

- For a health sweep, group the returned executions by status and report totals, the time window, and the standout non-success states.
- For command reliability, filter by exact `--command-id`; versions are distinct command IDs. State the denominator and window with any success rate.
- For workflow or source analysis, filter with a stable regular, feature, or service tag. Apply the
  rules from the `automation-pilot-tags` skill before interpreting tag inheritance or
  platform-stamped origin tags.
- Fetch individual details for the failed runs needed to explain the verdict. Do not imply that a list summary supplied an error message it does not contain.

Keep sensitive input and tag values out of the report. Prefer a short verdict, supporting counts, and the few executions that need attention over a raw response dump.
