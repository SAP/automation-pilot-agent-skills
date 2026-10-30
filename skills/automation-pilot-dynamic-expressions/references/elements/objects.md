# Objects and safety

Access object keys, test membership, inspect type, and handle missing data.

## Key access

- Dot and bracket forms: `.object.foo`, `.object."foo"`, `.object["foo"]`, `.array[0]`.
- Dynamic key: `.object[.inner.key]`.
- Value iterators: `.array[]`, `.object[]` produce a stream of values, which the engine concatenates into one text value unless you collect it with `[ ... ]`.

| Element | Argument | Does |
|---|---|---|
| `keys` | none | array of an object's keys |
| `values` | none | array of an object's values |
| `getCaseInsensitive(key)` | one string key | value for a key, matched case-insensitively, on a plain or stringified object |

## Membership and type

Note the direction: `has`/`in` are about keys/indices, `valueIn` is about values. `has` takes the key; `in`/`valueIn` take the container and test the piped-in input against it.

| Element | Argument | Does |
|---|---|---|
| `has(key)` | one string key (object) or integer index (array) | whether an object has the key, or an array has the index |
| `in(container)` | one object or array | whether the input is a key of the object, or a valid index of the array |
| `valueIn(container)` | one array or object (often an array literal, e.g. `valueIn([408, 429, 500])`) | whether the input is a value in the object or array |
| `type` | none | the type as a string: `null`, `boolean`, `number`, `string`, `array`, or `object` |

## Missing-data handling

- Accessing a missing key returns `null` silently, with no error.
- The alternative operator `//` emits the left-hand values that are neither `false` nor `null`, and otherwise the right-hand side. Use it for a scalar fallback: `.optional // "default"`. It filters a stream rather than coalescing a single value, and it treats a `false` left side as missing, so it needs care after an iterator or a comparison.

## Gotchas

- An object or array that crossed an executor or command output boundary arrives as a JSON string; parse it with `toObject` (object strings) or `toArray` (array strings) before key access. Reading a key off the unparsed string returns `null`.
- `getCaseInsensitive` is the safe way to read a header or key whose casing is not guaranteed.
- Reach root data from inside `map`/`filter`/`select` with `$`, not `.`.

## Examples

Read a value whose source might be missing, with a fallback:

```text
.getResource.output.region // "unknown"
```

Case-insensitive header lookup on a parsed object:

```text
(.getResource.output.headers | toObject) | getCaseInsensitive("content-type")
```

In a `responseBodyTransformer`, append an owner to a JSON response whose
`configurationParameters` is an object and `owners` is an array:

```text
toObject | . + { "configurationParameters": (.configurationParameters + { "owners": (.configurationParameters.owners + ["Monica"]) }) }
```

Object `+` merges keys at one level, with the right-hand value winning on a collision.
The outer merge preserves other top-level fields; the inner merge preserves other
configuration fields. The [nested owner fixtures](../fixtures.json) predict both
preservation checks, including an empty owners array.

Mind the direction — `has` takes the key and tests the piped-in object; `valueIn` takes the container and tests the piped-in value:

```text
.getResource.output.headers | toObject | has("ETag")   # does the object have this key?
.getResource.output.status | toNumber | valueIn([408, 429, 500])  # is this value one of these?
```
