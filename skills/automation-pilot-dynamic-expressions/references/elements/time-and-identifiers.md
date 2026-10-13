# Time and identifiers

Timestamps and date strings, version comparison, GUID generation and checking.

## Time

| Element | Argument | Returns |
|---|---|---|
| `now` | none | current time as a Unix timestamp in **seconds** (a number) |
| `nowMillis` | none | current time in **milliseconds** (a number) |
| `toDate(format)` | one date-time pattern string; input is a millisecond timestamp | a date-time string from the input timestamp |
| `fromDate(format)` | one date-time pattern string; input is a date-time string | a timestamp in milliseconds from the input string |

`now` and `nowMillis` read the clock and take no argument. They differ by unit: `now` returns **seconds** (ten-digit) and `nowMillis` returns **milliseconds** (thirteen-digit), so the two are not interchangeable; `now * 1000` is the millisecond scale. Pick the one whose unit matches what you compare against.

`toDate(format)` and `fromDate(format)` transform the piped-in value using a Java `SimpleDateFormat` pattern string. `toDate` always renders in UTC regardless of the pattern (a `0` input yields `1970-01-01T00:00:00Z`). `.execution.metadata.startTime` is in milliseconds, so compare it with `nowMillis`, not `now`.

## Version comparison

| Element | Argument | Returns |
|---|---|---|
| `compareVersions` | input: two versions as a two-element array (e.g. `["1.2.0", "1.10.0"]`) or an object | a number: `-1` if the first is lower, `0` if equal, `1` if the first is higher |
| `compareUnixVersions` | same input | the same `-1`/`0`/`1`, using the Unix version-sort algorithm |

Use these instead of comparing version strings with `<`/`>`, which compare lexicographically and rank `"1.10.0"` below `"1.2.0"`. Both compare segment by segment numerically, so `["1.2.0", "1.10.0"] | compareVersions` is `-1`.

## Identifiers

| Element | Argument | Does |
|---|---|---|
| `guid` | none | generate a random GUID |
| `guidShort` | none | generate a random GUID without hyphens |
| `isGuid` | input: a string | whether a text value is a GUID |

## Gotchas

- `now` and `nowMillis` resolve at execution time; a traced value is only representative.
- `now` is in seconds and `nowMillis` is in milliseconds. Mixing them (for example `startTime + 3600 < nowMillis`, where `startTime` is milliseconds) compares across scales and misfires: add `3600000` for one hour, not `3600`.
- `.execution.metadata.startTime`, `currentExecutorPath`, and `tags` do not behave as expected when a command relying on them is reused inside another command.

## Examples

Branch on an elapsed-time threshold (one hour after start; `startTime` and `nowMillis` are both milliseconds, so the offset is 3,600,000 ms):

```text
if $.execution.metadata.startTime + 3600000 < nowMillis then "expired" else "fresh" end
```

Pick the higher of two version strings (lexicographic `>` would rank `"1.10.0"` below `"1.9.0"`):

```text
if (.versions | compareVersions) == 1 then .versions[0] else .versions[1] end
```

Branch on whether an input is a GUID:

```text
if .execution.input.resourceId | isGuid then "valid" else "invalid" end
```

Render a millisecond timestamp as a UTC date-time string with a `SimpleDateFormat` pattern (`toDate` always renders in UTC regardless of the pattern):

```text
nowMillis | toDate("yyyy-MM-dd'T'HH:mm:ss'Z'")   # e.g. "2026-10-13T09:30:00Z"
```
