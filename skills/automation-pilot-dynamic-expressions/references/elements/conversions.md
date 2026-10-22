# Conversions, encodings, and serialization

Convert a value between types, encode or hash it, and serialize JSON. Use the exact spellings below; they are the documented element names.

Most conversions take no argument; they transform the piped-in value (`.body | toObject`). The two that do take arguments are shown with their signatures.

## Type conversion

| Element | Argument | Converts |
|---|---|---|
| `toObject` | none | value to an object (parses an **object** string; returns `null` for an array string) |
| `toArray` | none | value to an array (parses an **array** string) |
| `toString` | none | value to a string |
| `toNumber` | none | value to a number |
| `toBoolean` | none | value to a boolean |
| `toMap(k; v)` | two expressions, separated by `;` | an array to an object, using `k` for each key and `v` for each value, both evaluated with `.` bound to the element (e.g. `toMap(.id; .name)`) |
| `toEntries` | none | an object to an array of `{key, value}` objects |
| `fromEntries` | none | an array of `{key, value}` objects to an object |

## Formats

| Element | Converts |
|---|---|
| `fromXml` | an XML string to a JSON object |
| `fromYaml` / `toYaml` | between a YAML string and JSON |
| `fromProperties` / `toProperties` | between a `.properties`-formatted string and JSON |

## Encodings and hashes

| Element | Does |
|---|---|
| `toBase64` / `fromBase64` | Base64 encode / decode |
| `toUrlEncoded` / `fromUrlEncoded` | form (`application/x-www-form-urlencoded`) encode / decode, where a space becomes `+` (e.g. `"a b&c"` → `a+b%26c`) |
| `toPercentEncoded` | percent-encode, where a space becomes `%20` (e.g. `"a b&c"` → `a%20b%26c`) |
| `toEscapedJson` | stringify and escape all JSON special characters |
| `toMd5` / `toSha256` | MD5 / SHA-256 hash |

## JSON serialization

| Element | Does |
|---|---|
| `toMinifiedJsonString` | compact single-line JSON string |
| `toPrettyJsonString` | formatted, human-readable JSON string |

## Gotchas

- `toObject` parses object strings only; `toArray` parses array strings. Picking the wrong one returns `null`: `"[1,2,3]" | toObject` is `null`, not the array. Apply them to a string, not to a value that already has the target type.
- `toMinifiedJsonString` serves a receiving field or operation that requires JSON text. An output key uses the [output-key type boundary](../expression-reference.md#output-key-type-boundary) instead.
- An impossible conversion returns `null` silently rather than erroring.

## Examples

Parse a JSON string from an output declared as `string`, then select inside it:

```text
(.getResource.output.body | toObject).items[0].name
```

Convert an object to an array of entries and back:

```text
{ "a": 1, "b": 2 } | toEntries        # [ {"key":"a","value":1}, {"key":"b","value":2} ]
[ {"key":"a","value":1} ] | fromEntries  # {"a":1}
```

Extract attribute values from parsed XML (the `@` prefix reads an attribute):

```text
fromXml.response.descriptors | map(.["@id"])
```
