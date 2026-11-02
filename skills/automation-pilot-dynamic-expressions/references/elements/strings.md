# Strings

Case, trimming, slicing, interpolation, splitting and joining, and regular expressions.

## Case

| Element | Does |
|---|---|
| `toUpperCase` / `toLowerCase` | convert the whole string |
| `ascii_upcase` / `ascii_downcase` | convert ASCII letters only; non-ASCII characters are unchanged |

## Trimming

| Element | Argument | Removes | Alias |
|---|---|---|---|
| `strip` | none | whitespace at both ends | |
| `stripStart` / `stripEnd` | none | leading / trailing whitespace | |
| `removePrefix(s)` | one string, the prefix to strip | a given prefix | `ltrimstr` |
| `removeSuffix(s)` | one string, the suffix to strip | a given suffix | `rtrimstr` |

## Merging, interpolation, and slicing

- Merge with `+`: `"Hi, my name is " + .name`.
- Interpolate inside a string literal with `\( ... )`: `"Hello \(.name)"`.
- Slice with ranges: `.text[5:]` (from index 5 to the end), `.text[0:3]` (first three characters), `.text[:3]` (same, open start). A single index gives one character: `"caterpillar"[3]`.

## Split and join

| Element | Argument | Does |
|---|---|---|
| `split(sep)` | one string separator; input is a string | split a string into an array of substrings |
| `join(sep)` | one string separator; input is an array | concatenate array elements into a string |

`join` casts non-string elements to text; `null` elements become empty. The separator is a literal string, not a regular expression.

## Membership and prefix tests

| Element | Argument | Does |
|---|---|---|
| `startsWith(s)` | one string | whether the input string begins with `s` |
| `endsWith(s)` | one string | whether the input string ends with `s` |
| `contains(s)` | one string (or value, on arrays/objects) | whether `s` occurs in the input |

`contains` also works on arrays and objects, testing containment of the argument value.

## Regular expressions

| Element | Argument | Does |
|---|---|---|
| `sub(re; replacement)` | two expressions: regex string `re`, then replacement string | replace the **first** match |
| `gsub(re; replacement)` | same, separated by `;` | replace **all** non-overlapping matches |

Both arguments are strings separated by a semicolon. `re` is a regex pattern string; `replacement` is a string. Regular-expression evaluation is subject to a timeout; keep patterns simple.

## Gotchas

- Comparison operators (`<`, `>`, `==`) on strings are lexicographic and character by character; normalize numeric-looking strings before comparing them. For version strings specifically, use `compareVersions`/`compareUnixVersions` rather than `<`/`>`, which rank `"1.10.0"` below `"1.2.0"`.
- `removePrefix`/`removeSuffix` remove the affix only when it is present; otherwise the string is returned unchanged.

## Examples

Extract the major version from a semantic version:

```text
split(".")[0]           # "22.5.3" -> "22"
```

URL-encode a query value by interpolation:

```text
"https://www.website.com?query=\(.value | toUrlEncoded)"   # "Monica and Bob" -> ...query=Monica+and+Bob
```

Join array elements onto separate lines:

```text
.names | join("\n")
```

Replace all matches with `gsub` (two arguments separated by `;`: the regex pattern, then the replacement; use `sub` for only the first match):

```text
.path | gsub("/+"; "/")   # "a//b///c" -> "a/b/c"
```
