# Dynamic Expressions Reference

SAP Automation Pilot uses jq 1.6 for data manipulation between command steps. Expressions are wrapped in `$()`.

## Scope and Access

### Identity and Scope

| Expression | Description |
|------------|-------------|
| `.` | Current data (scoped to pipe context) |
| `$` | Global execution data (access anywhere) |
| `.execution.input.key` | Access execution inputs |
| `.stepAlias.output.key` | Access previous step outputs |
| `.execution.metadata.key` | Access execution metadata |

### Examples

```
$(.execution.input.region)                    # Input value
$(.getApp.output.body | toObject.guid)        # Step output parsed
$(.regionData.cfApiUrl)                       # Value from input reference
$($.execution.input.name)                     # Global access inside pipe
```

## Mathematical Operations

### Arithmetic

| Operation | Example |
|-----------|---------|
| Addition | `$(1 + 2)` → `3` |
| Subtraction | `$(5 - 3)` → `2` |
| Multiplication | `$(3 * 4)` → `12` |
| Division | `$(10 / 3)` → `3.333...` |
| Modulo | `$(10 % 3)` → `1` |

### Math Functions

| Function | Example | Result |
|----------|---------|--------|
| `floor` | `$(3.7 \| floor)` | `3` |
| `ceil` | `$(3.2 \| ceil)` | `4` |
| `round` | `$(3.5 \| round)` | `4` |

## String Operations

### Basic Operations

| Operation | Example | Result |
|-----------|---------|--------|
| Concatenation | `$("Hello" + " " + "World")` | `"Hello World"` |
| Interpolation | `$("Hello \(.name)")` | `"Hello John"` |
| Slicing | `$("caterpillar"[0:3])` | `"cat"` |
| From index | `$("caterpillar"[5:])` | `"pillar"` |
| Length | `$("hello" \| length)` | `5` |

### Case Conversion

| Function | Example | Result |
|----------|---------|--------|
| `toUpperCase` | `$("hello" \| toUpperCase)` | `"HELLO"` |
| `toLowerCase` | `$("HELLO" \| toLowerCase)` | `"hello"` |

### Trimming

| Function | Example | Result |
|----------|---------|--------|
| `strip` | `$("  hello  " \| strip)` | `"hello"` |
| `stripStart` | `$("  hello" \| stripStart)` | `"hello"` |
| `stripEnd` | `$("hello  " \| stripEnd)` | `"hello"` |
| `removePrefix` | `$("prefix-value" \| removePrefix("prefix-"))` | `"value"` |
| `removeSuffix` | `$("value.json" \| removeSuffix(".json"))` | `"value"` |
| `ltrimstr` | `$("hello" \| ltrimstr("hel"))` | `"lo"` |
| `rtrimstr` | `$("hello" \| rtrimstr("lo"))` | `"hel"` |

### Search and Replace

| Function | Example | Result |
|----------|---------|--------|
| `sub` | `$("hello world" \| sub("world"; "jq"))` | `"hello jq"` |
| `gsub` | `$("a-b-c" \| gsub("-"; "_"))` | `"a_b_c"` |
| `split` | `$("a,b,c" \| split(","))` | `["a","b","c"]` |
| `join` | `$(["a","b","c"] \| join("-"))` | `"a-b-c"` |

### String Tests

| Function | Example | Result |
|----------|---------|--------|
| `startsWith` | `$("hello" \| startsWith("he"))` | `true` |
| `endsWith` | `$("hello" \| endsWith("lo"))` | `true` |
| `contains` | `$("hello" \| contains("ell"))` | `true` |

## Array Operations

### Selection

| Operation | Example | Result |
|-----------|---------|--------|
| Index | `$(["a","b","c"][1])` | `"b"` |
| Slice | `$(["a","b","c","d"][1:3])` | `["b","c"]` |
| Iterator | `$(["a","b","c"][])` | `"a"`, `"b"`, `"c"` |
| First | `$(["a","b","c"] \| first)` | `"a"` |
| Last | `$(["a","b","c"] \| last)` | `"c"` |

### Transformation

| Function | Example | Result |
|----------|---------|--------|
| `map` | `$([1,2,3] \| map(. * 2))` | `[2,4,6]` |
| `filter` | `$([1,2,3,4] \| filter(. > 2))` | `[3,4]` |
| `select` | `$([1,2,3,4] \| select(. > 2))` | `3`, `4` |
| `sort` | `$([3,1,2] \| sort)` | `[1,2,3]` |
| `sortBy` | `$([{a:2},{a:1}] \| sortBy(.a))` | `[{a:1},{a:2}]` |
| `reverse` | `$([1,2,3] \| reverse)` | `[3,2,1]` |
| `unique` | `$([1,2,2,3] \| unique)` | `[1,2,3]` |
| `uniqueBy` | `$([{a:1,b:1},{a:1,b:2}] \| uniqueBy(.a))` | `[{a:1,b:1}]` |
| `flatten` | `$([[1,2],[3,4]] \| flatten)` | `[1,2,3,4]` |
| `group_by` | `$([{a:1},{a:2},{a:1}] \| group_by(.a))` | `[[{a:1},{a:1}],[{a:2}]]` |

### Aggregation

| Function | Example | Result |
|----------|---------|--------|
| `length` | `$([1,2,3] \| length)` | `3` |
| `add` | `$([1,2,3] \| add)` | `6` |
| `min` | `$([1,2,3] \| min)` | `1` |
| `max` | `$([1,2,3] \| max)` | `3` |
| `reduce` | `$([1,2,3] \| reduce .[] as $x (0; . + $x))` | `6` |

### Tests

| Function | Example | Result |
|----------|---------|--------|
| `any` | `$([1,2,3] \| any(. > 2))` | `true` |
| `all` | `$([1,2,3] \| all(. > 0))` | `true` |
| `contains` | `$([1,2,3] \| contains([2]))` | `true` |
| `inside` | `$([2] \| inside([1,2,3]))` | `true` |
| `valueIn` | `$(2 \| valueIn([1,2,3]))` | `true` |

## Object Operations

### Access

| Operation | Example | Result |
|-----------|---------|--------|
| Key access | `$({a:1,b:2}.a)` | `1` |
| Bracket notation | `$({a:1}["a"])` | `1` |
| `getCaseInsensitive` | `$({Name:"John"} \| getCaseInsensitive("name"))` | `"John"` |
| Optional access | `$({a:1}.b?)` | `null` |

### Extraction

| Function | Example | Result |
|----------|---------|--------|
| `keys` | `$({a:1,b:2} \| keys)` | `["a","b"]` |
| `values` | `$({a:1,b:2} \| values)` | `[1,2]` |
| `toEntries` | `$({a:1} \| toEntries)` | `[{key:"a",value:1}]` |
| `fromEntries` | `$([{key:"a",value:1}] \| fromEntries)` | `{a:1}` |
| `has` | `$({a:1} \| has("a"))` | `true` |
| `in` | `$("a" \| in({a:1}))` | `true` |

### Modification

| Operation | Example | Result |
|-----------|---------|--------|
| Merge | `$({a:1} + {b:2})` | `{a:1,b:2}` |
| Override | `$({a:1} + {a:2})` | `{a:2}` |
| Delete | `$({a:1,b:2} \| del(.a))` | `{b:2}` |

## Type Conversion

### JSON Parsing

| Function | Example | Description |
|----------|---------|-------------|
| `toObject` | `$(.body \| toObject)` | Parse JSON string to object |
| `toArray` | `$(.body \| toArray)` | Parse JSON string to array |
| `toString` | `$(123 \| toString)` | Convert to string |
| `toNumber` | `$("123" \| toNumber)` | Convert to number |
| `toBoolean` | `$("true" \| toBoolean)` | Convert to boolean |

### Encoding

| Function | Example | Description |
|----------|---------|-------------|
| `toBase64` | `$("hello" \| toBase64)` | Base64 encode |
| `fromBase64` | `$("aGVsbG8=" \| fromBase64)` | Base64 decode |
| `toUrlEncoded` | `$("hello world" \| toUrlEncoded)` | URL encode |
| `fromUrlEncoded` | `$("hello%20world" \| fromUrlEncoded)` | URL decode |
| `toPercentEncoded` | `$("hello/world" \| toPercentEncoded)` | Percent encode |
| `toEscapedJson` | `$({a:1} \| toEscapedJson)` | Escape JSON for embedding |

### Hash Functions

| Function | Example | Description |
|----------|---------|-------------|
| `toMd5` | `$("hello" \| toMd5)` | MD5 hash |
| `toSha256` | `$("hello" \| toSha256)` | SHA256 hash |

### Format Conversion

| Function | Example | Description |
|----------|---------|-------------|
| `fromYaml` | `$(yamlString \| fromYaml)` | Parse YAML to object |
| `toYaml` | `$({a:1} \| toYaml)` | Convert to YAML string |
| `fromProperties` | `$(propsString \| fromProperties)` | Parse properties file |
| `toProperties` | `$({a:1} \| toProperties)` | Convert to properties format |
| `fromXml` | `$(xmlString \| fromXml)` | Parse XML to object |

### JSON Formatting

| Function | Example | Description |
|----------|---------|-------------|
| `toPrettyJsonString` | `$({a:1} \| toPrettyJsonString)` | Pretty print JSON |
| `toMinifiedJsonString` | `$({a:1} \| toMinifiedJsonString)` | Minify JSON |

## Control Flow

### Conditionals

```
$(if .condition then "yes" else "no" end)

$(if .x > 10 then "big"
  elif .x > 5 then "medium"
  else "small"
  end)
```

### Alternative Operator

The `//` operator provides fallback values:

```
$(.value // "default")                    # Use "default" if null/false
$(.obj.key // .obj.fallbackKey // "N/A")  # Chain fallbacks
```

### Variables

```
$(.input as $x | .items | map(. + $x))
$(.data | . as $d | .items | map({item: ., total: $d.count}))
```

### Pipe Operator

Chain operations with `|`:

```
$(.body | toObject | .items | map(.name) | join(", "))
```

### Multiple Outputs

Use `,` for multiple outputs:

```
$(.a, .b, .c)         # Outputs all three values
$([.a, .b, .c])       # Collects into array
```

## Date and Time

| Function | Example | Description |
|----------|---------|-------------|
| `now` | `$(now)` | Current Unix timestamp (seconds) |
| `nowMillis` | `$(nowMillis)` | Current Unix timestamp (milliseconds) |
| `toDate` | `$(1234567890 \| toDate)` | Unix timestamp to ISO date |
| `fromDate` | `$("2009-02-13T23:31:30Z" \| fromDate)` | ISO date to Unix timestamp |

### Date Formatting

```
$(now | strftime("%Y-%m-%d"))             # "2024-01-15"
$(now | strftime("%Y-%m-%dT%H:%M:%SZ"))   # ISO format
```

## UUID Generation

| Function | Example | Description |
|----------|---------|-------------|
| `guid` | `$(guid)` | Generate UUID v4 |
| `guidShort` | `$(guidShort)` | Generate short UUID |
| `isGuid` | `$(.id \| isGuid)` | Validate GUID format |

## Version Comparison

| Function | Example | Result |
|----------|---------|--------|
| `compareVersions` | `$("1.2.3" \| compareVersions("1.2.4"))` | `-1` |
| `compareUnixVersions` | `$("1.2" \| compareUnixVersions("1.10"))` | `-1` |

Returns: `-1` (less than), `0` (equal), `1` (greater than)

## Utility

### Range Generation

```
$(range(5))                   # 0, 1, 2, 3, 4
$(range(2; 5))               # 2, 3, 4
$([range(5)])                # [0, 1, 2, 3, 4]
```

### Type Checking

The `| type` function does not exist in Automation Pilot's expression engine. To check whether a value is present or non-empty, use `| length` with an `EQUALS "0"` condition instead.

```
$(.value | length)    # 0 if null/empty, >0 if present
```

## Common Patterns

### Parse and Extract

```
$(.response.output.body | toObject.data.items[0].id)
```

### Conditional Value

```
$(if .execution.input.enabled == "true" then "active" else "inactive" end)
```

### Transform Array

```
$(.items | map({name: .title, id: .guid}))
```

### Filter and Select

```
$(.apps | filter(.state == "STARTED") | map(.name))
```

### Safe Navigation

```
$(.response.output.body | toObject.result // {})
$(.data.items[0]?.name // "unknown")
```

### URL Building

```
$(.baseUrl + "/api/v1/apps/" + (.appId | toUrlEncoded))
```

### Retry Status Check

```
$([408, 429, 500, 502, 503, 504, -1] | filter(. == $.step.output.status) | length)
```

### Join with Transformation

```
$(.items | map(.name) | join(", "))
```

### Object from Pairs

```
$(.entries | map({key: .name, value: .data}) | fromEntries)
```
