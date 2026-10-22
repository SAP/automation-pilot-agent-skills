# Math and control flow

Arithmetic, conditionals, variables, and the stream operators.

## Arithmetic

- Operators: `+`, `-`, `*`, `/` (floating-point division), `%` (modulo). Example: `.number + 1`, `.number * 10`, `.number % 10`.
- `/` returns a real number: `7 / 2` is `3.5`, `1 / 3` is `0.3333333333333333`, `6 / 2` is `3`. It does not truncate.
- Rounding: `floor` (down), `ceil` (up), `round` (nearest). Apply one of these when you need an integer from a division.

## Comparison and boolean

- Number and value comparison: `>`, `>=`, `<`, `<=`, `==`, `!=`.
- Boolean: `and`, `or`.

## Control flow

| Construct | Form | Arguments |
|---|---|---|
| Conditional | `if cond then A else B end` | `cond` boolean; `A`/`B` expressions. `elif cond then …` chains. `else` is **required** — the engine rejects `if cond then A end` with "Expected 'else'" (unlike stock jq, where `else` is optional) |
| Variable binding | `expr as $x \| ...` | `expr` is any expression; `$x` is the binding name. For each value of `expr`, run the rest with `$x` bound to it and the original input as `.` |
| Alternative | `A // B` | `A`, `B` expressions. Emits every value of stream `A` that is neither `false` nor `null`; falls back to `B` only when `A` produces nothing but `false`/`null` (or no value at all) |
| Pipe | `A \| B` | feed `A`'s output into `B` |
| Comma | `A, B` | produce both as a stream |
| `range(N)` | one integer | integers from 0 up to `N` (exclusive) |

## Gotchas

- `/` is floating-point division (`7 / 2` is `3.5`), not integer division. Apply `floor`/`ceil`/`round` when you need a whole number.
- `else` is mandatory in `if`; `if cond then A end` without it is rejected at authoring.
- `//` filters a **stream**, it is not a scalar null-coalesce. `(null, 1, null, 2) // "fb"` yields `1, 2` (the nulls are dropped and the survivors pass, so the fallback never fires); `//` reaches the right side only when the whole left side is `false`/`null` or empty. So `iterator // "default"` is not "default when the stream was empty-ish"; a single surviving element suppresses the fallback.
- `//` binds **looser** than comparisons (`==`, `>`, …) and **tighter** than `|`. `null // 5 == 5` parses as `null // (5 == 5)`, and `.a // .b | .c` parses as `(.a // .b) | .c`. Because `false` triggers the fallback, `(1 > 2) // "fb"` is `"fb"` and `.flag // true` returns `true` even when `.flag` is a legitimate `false`. Parenthesize a comparison on the left, and do not use `//` to default a boolean.
- The comma produces a stream; multiple results concatenate into one text value unless collected with `[ ... ]`. `$(1, 2, 3)` is `123`, not `[1,2,3]`.
- A nested `$(...)` inside an outer `$(...)` is literal, not evaluated. To build a string inside an expression, use concatenation (`"prefix-" + (.x | toString)`) or interpolation (`"prefix-\(.x)"`).
- There is no `empty` element in the engine; `$(empty // x)` fails to parse. Produce an empty stream with a non-matching `select`/`filter` instead.

## Examples

Round a ratio up:

```text
((.used / .total) * 100) | ceil
```

Select even numbers from a merged array, as an array:

```text
[ (.a + .b)[] | select(. % 2 == 0) ]
```

Choose a value by condition:

```text
if (.getResource.output.status | toNumber) == 200 then "ok" else "retry" end
```

Default a possibly-missing scalar with `//` — safe here because the left side is a single value, not a stream or a boolean:

```text
.getResource.output.region // "unknown"
```

`//` binds looser than `==`: `.n == 5 // false` parses as `(.n == 5) // false`.
For a missing retry count, default the numeric value before comparing, and use the executor-level path:

```text
(.getResource.retriesCount // 0) == 0
```
