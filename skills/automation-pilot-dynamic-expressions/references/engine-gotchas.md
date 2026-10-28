# Engine gotchas

Non-obvious behavior of the Automation Pilot jq engine that is not covered by the expression reference. Each entry here is a surprise that lives only in this file. Reason about these before relying on a subtle construct, and check here first when an expression returns an unexpected or empty result.

## Void-envelope string fields need escaped quotes

When passing a string field through a `utils-sapcp:Void:1` JSON envelope, wrap the hole in escaped quotes so an empty value still produces valid JSON:

```json
{ "name": "{\"delName\":\"$(EXPR)\"}" }
```

Without the quotes, `{"delName":$(EXPR)}` where `EXPR` can be empty produces malformed `{"delName":}`, which `toObject` parses as `null`. Number and boolean fields do not need the quotes; only string fields do.

## Division is floating-point

`/` performs floating-point division, not integer division: `7 / 2` is `3.5`, `1 / 3` is `0.3333333333333333`, `-7 / 2` is `-3.5`, and `6 / 2` is `3`. The result keeps its fractional part and its type is always `number`. When you need a whole number, apply `floor`/`ceil`/`round` explicitly; the operator will not truncate for you.

## The alternative operator `//` filters a stream

`A // B` emits every value of stream `A` that is neither `false` nor `null`, and uses `B` only when `A` yields nothing but `false`/`null` (or no value at all). It is not a scalar null-coalesce:

- `(null, 1, null, 2) // "fb"` is `1, 2`: the nulls drop out and the survivors pass, so the fallback never fires. A single surviving element suppresses `B`.
- An empty left side triggers the fallback: `([1,2,3][] | select(. > 10)) // "fb"` is `"fb"`.
- Precedence: `//` binds looser than comparisons and tighter than `|`. `null // 5 == 5` parses as `null // (5 == 5)`; `.a // .b | .c` parses as `(.a // .b) | .c`. Because `false` triggers the fallback, `(1 > 2) // "fb"` is `"fb"` and `.flag // true` returns `true` even when `.flag` is a real `false`. Parenthesize a comparison on the left of `//`, and never use `//` to default a boolean.
- There is no `empty` keyword in the engine; `$(empty // x)` fails to parse. Use a non-matching `select`/`filter` to make an empty stream.

## Nested `$()` is not interpolated

A `$(...)` written inside another `$(...)` is treated literally, not evaluated. To build a string inside an expression, use jq string concatenation or interpolation: `("prefix-" + (.x | toString))` or `"prefix-\(.x)"`. The interpolation-hole form `$(...)` is for literal text or JSON outside an expression.

## Prefer the array-aggregate forms, not bare `max`/`min`

The documented analysis functions are `minBy(f)` and `maxBy(f)`, which take an expression argument and return an element; for a plain array use `sort | .[0]` for the minimum and `sort | .[-1]` for the maximum. Do not assume a bare `max`/`min` filter exists.

## Current-executor counters do not generate changing inputs

In the 2026-10-26 tenant test, an executor input using its own
`.<alias>.retriesCount` or `.<alias>.repetitionsCount` stayed on the initial branch throughout
the retries or repetitions. A later output mapping observed the retry counter correctly;
the repeat-counter expression returned its zero fallback despite a changed second response.
Use the execution logs to verify repetition counts on the target tenant.
Read counters in a later step or output mapping. For controlled response sequences, have the
repeated command read external test state: a tenant-scoped store counter produced `503, 200`
and `RUNNING, SUCCEEDED`; the retry counter was one and the polling log confirmed a repeat. Initialize a unique counter per
test execution and clean it up after both success and failure. This restriction does not apply
to the iterator's documented `.<alias>.current.item` binding.
