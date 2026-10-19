# Collections

Transform, filter, sort, group, and reduce arrays and objects, and run the array-analysis functions.

Argument conventions: a function argument is itself an expression evaluated against each element, with `.` bound to that element. `f` below is such an expression (for example `.priority`, or a pipeline `.name | ascii_downcase`); `cond` is a boolean expression. Multiple arguments are separated by **semicolons**, not commas (`gsub(re; replacement)`), because a comma inside the parentheses would be the stream operator.

Inside any of these argument expressions `.` is the current element, not the root. To reach execution data from inside `map`/`filter`/`select`, use `$` (for example `filter(.name == $.execution.input.name)`); writing `.execution.input.name` there looks for that path on each element and silently finds nothing.

## Transform and filter

| Element | Argument | Does |
|---|---|---|
| `map(f)` | one expression, applied per element | apply `f` to every element. On an **array** returns an array; on an **object** applies `f` to each value and returns an **object** with the same keys |
| `filter(cond)` | one boolean expression | keep elements matching `cond`. On an **array** returns an array; on an **object** keeps the matching key/value pairs and returns an **object** |
| `select(cond)` | one boolean expression | keep values matching `cond` |
| `add` | none | sum, merge, or concatenate the elements of an array |
| `flatten` / `flatten(N)` | optional `N`, an integer depth | flatten all levels, or `N` levels |

To keep the matching elements of an array, use `filter(cond)` — it is the single-step form and returns an array. Reserve `select` for a value stream (inside `reduce`, or after `.[]`): `map(select(cond))` is the verbose spelling of `filter(cond)`, and `map(select(cond) | f)` is `filter(cond) | map(f)`.

## Order and dedup

| Element | Argument | Does |
|---|---|---|
| `sort` | none | ascending order |
| `sortBy(f)` / `sort_by(f)` | one expression (the sort key) | ascending by the value of `f` for each element |
| `reverse` | none | reverse order |
| `unique` | none | sort and keep the first of each duplicate |
| `uniqueBy(f)` | one expression (the dedup key) | sort and dedup by `f` |
| `groupBy(f)` / `group_by(f)` | one expression (the grouping key) | group elements with the same `f` into sub-arrays, sorted by the grouping value |

## Reshape

| Element | Argument | Does |
|---|---|---|
| `reduce` | `reduce EXPR as $var (INIT; UPDATE)` | combine a stream into one accumulated value (see below) |
| `transpose` | none | turn rows into columns across inner arrays |
| `zip(a; b)` | two array expressions, separated by `;` | pair two arrays by index into two-element arrays (e.g. `zip([1,2]; [3,4])` is `[[1,3],[2,4]]`); it does not take the input as a source |
| `unzip` | none | split an array of pairs back into two arrays (operates on the piped-in array) |

`reduce` is a syntactic form, not a parenthesized call: `reduce .items[] as $item (0; . + $item.count)`. `EXPR` is a stream generator (iterate an array with `.arr[]`, an object with `toEntries[]`); `$var` binds each streamed value; `INIT` seeds the accumulator; `UPDATE` runs once per streamed value with `.` bound to the accumulator so far.

## Analysis

| Element | Argument | Does |
|---|---|---|
| `length` | none | length of an array, object, or string |
| `any(cond)` / `all(cond)` | one boolean expression | whether at least one / every element matches |
| `contains(x)` | one value (same type as input) | whether `x` is contained in the input |
| `inside(x)` | one value (same type as input) | whether the input is contained in `x` (inverse of `contains`) |
| `minBy(f)` / `maxBy(f)` | one expression (the comparison key) | the element with the smallest / largest `f` |

## Gotchas

- `add` over an empty array produces no value. Use `(<array expression> | add) // 0` when an empty array should default to zero: the parenthesized left side produces one scalar or no value. This differs from applying `//` directly to a multi-value stream, where fallback evaluation happens per streamed value and can pass through multiple results.
- For the minimum or maximum of a plain array, use `sort | .[0]` and `sort | .[-1]`. Do not assume a bare `max`/`min` filter exists; `minBy`/`maxBy` take an expression argument and return an element.
- `reduce` needs an iterable input: wrap an array with `[]` or `toArray[]`, and iterate an object with `toEntries[]`.
- A value iterator (`.arr[]`), `map`, and `filter` applied to a stream produce a stream of results, which the engine concatenates into one text value unless you collect it. Wrap the stream in `[ ... ]` to get an array: `[.arr[] | . * 100]` yields `[100, 200, 300]`, whereas `.arr[] | . * 100` yields `100200300`.

## Examples

Filter objects by a field, then project a value:

```text
.getItems.output.items | filter(.state == "Stopped") | map(.id)
```

Combine two arrays and remove duplicates:

```text
.owners + .players | unique
```

Apply `filter` to an object, keeping key/value pairs by value (the result is an object):

```text
{ "a": 1, "b": 2, "c": 3 } | filter(. > 1)   # { "b": 2, "c": 3 }
```

Keep only elements whose value is in an allow-list:

```text
toArray | filter(.name | valueIn(["banana", "apple"]))
```

Drop blank or null elements:

```text
toArray | filter(. | length > 0)
```

Sum a field across an array with `reduce` (a syntactic form, not a call; the input must be iterated, here with `.items[]`):

```text
reduce .getItems.output.items[] as $item (0; . + $item.count)
```

Pick the element with the largest field — use `maxBy(f)`, not a bare `max`:

```text
.getItems.output.items | maxBy(.priority)        # the whole element, not just .priority
```
