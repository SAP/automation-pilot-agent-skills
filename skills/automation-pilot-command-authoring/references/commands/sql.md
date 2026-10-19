# SQL Against HANA Cloud

Run a SQL statement against an SAP HANA Cloud instance from a command step: a query that
feeds later steps, a DML/DDL change, or a stored-procedure call.

## Contents

- [When to use](#when-to-use)
- [Executor IDs and variants](#executor-ids--variants)
- [Inputs and authentication](#key-inputs)
- [Outputs](#outputs--how-to-read-them)
- [Result transformation](#result-transformation)
- [Minimal snippet](#minimal-snippet)
- [Gotchas](#gotchas)

## When to use

- To read or change data in HANA Cloud directly over JDBC, use `ExecuteHanaCloudSqlStatement:1`.
- When the result set can contain sensitive data, use `SensitiveExecuteHanaCloudSqlStatement:1`,
  which treats `statement`, `parameters`, `result`, and `metadata` as sensitive.
- To reach an on-premise HANA via SAP Cloud Connector, set `sccEnabled:"true"` (plus `sccLocationId`).

## Executor IDs & variants

| Executor | Use for |
|----------|---------|
| `sql-sapcp:ExecuteHanaCloudSqlStatement:1` | SQL against HANA Cloud; result is non-sensitive |
| `sql-sapcp:SensitiveExecuteHanaCloudSqlStatement:1` | Same; `statement`/`parameters`/`result`/`metadata` treated as sensitive |

## Key inputs

Required: `statement` (the SQL) and `connectionUrl` (JDBC).

| Field | Default | Notes |
|-------|---------|-------|
| `connectionUrl` | — | JDBC URL. Allowed JDBC properties: `currentSchema`, `readOnly`, `siteType`, `splitBatchCommands`. |
| `statement` | — | The SQL statement. Use `parameters` for values rather than string-concatenating them in. |
| `parameters` | — | Array, bound positionally to `?` placeholders in `statement`. |
| `user` + `password` | — | Basic (database-user) auth. `password` is sensitive. |
| `clientCert` | — | Client-certificate auth: PKCS#8 key + X.509 cert(s), PEM (optionally Base64). Sensitive. |
| `resultRowFormat` | `ARRAY` | `ARRAY` → each row `[v1,v2,…]`; `OBJECT` → `{"COL":v,…}` keyed by column name. |
| `resultTransformer` | — | Dynamic-expression syntax without an outer `$()`; replaces `result`. Start with `toObject`/`toArray` (the result is a string). See "Result transformation" below. |
| `succeedOnErrorCodes` | `[]` | Error codes treated as success. Example: `["2xx","3xxx","708","-5"]`. |
| `succeedOnTimeout` | `false` | `true` → a timeout does not fail the step. |
| `timeout` | `3` | **1–90 seconds.** |
| `sccEnabled` / `sccLocationId` | `false` | Route via Cloud Connector to on-premise HANA. |
| `trustAnyCert` / `trustedCerts` | `false` / — | TLS trust overrides. Never set `trustAnyCert:"true"` in production. |

### Authentication (pick one)
- **Basic:** `user` + `password`.
- **Client certificate:** `clientCert` (and `trustedCerts` for the server chain if needed).
## Outputs & how to read them

| Output | Type | Meaning |
|--------|------|---------|
| `result` | string | Result sets: an array of result sets, each an array of rows formatted per `resultRowFormat`. The last set holds `IN`/`INOUT` params (columns `P1`,`P2`,…). |
| `metadata` | array | Per-column `[name, type]` for each result set. |
| `updateCount` | number | Rows affected by a DML statement. |
| `errorMessage` | string | Set when the statement fails. |
| `errorCode` | number | Set when the statement fails. |

- `result` is a string, so parse before indexing: `$(.runSql.output.result | toArray)`.
  With `resultRowFormat:"OBJECT"`, a cell is `$(.runSql.output.result | toArray[0][0].ID)`;
  with `ARRAY` it is positional `… | toArray[0][0][0]`.
- Note the double nesting: `result[<resultSetIndex>][<rowIndex>]`.

## Result transformation

`resultTransformer` runs dynamic-expression syntax on the `result` string without the outer
`$()` wrapper, and the transformed value replaces `result`. The transformer field is a normal
string, so it can still contain `$()` substitutions resolved from execution data before the
result expression runs. The result arrives as a string, so start with `toObject`/`toArray`. Use
it to reshape or trim rows before the result crosses into the rest of the command.

```jq
// with resultRowFormat OBJECT: keep two columns from the first result set's rows
toArray[0] | map({ID, DATA})

// flatten the first result set to a list of a single column's values
toArray[0] | map(.ID)
```

## Minimal snippet

```json
{
  "execute": "sql-sapcp:ExecuteHanaCloudSqlStatement:1",
  "alias": "runSql",
  "input": {
    "connectionUrl": "$(.execution.input.connectionUrl)",
    "user": "$(.execution.input.user)",
    "password": "$(.execution.input.password)",
    "statement": "SELECT ID, DATA FROM MY_SCHEMA.MY_TABLE WHERE STATUS = ?",
    "parameters": "[\"ACTIVE\"]",
    "resultRowFormat": "OBJECT",
    "timeout": "20"
  },
  "description": null, "progressMessage": null,
  "initialDelay": null, "pause": null,
  "when": null, "validate": null, "autoRetry": null, "repeat": null,
  "errorMessages": []
}
```

## Gotchas

- **Bind values via `parameters`, not string interpolation.** This avoids SQL injection and quoting bugs.
- `result` is nested as result-set, then row, then column; a single-query single-row read is
  `result[0][0]`, not `result[0]`.
- `timeout` outside 1–90 is rejected.
- Use the `Sensitive*` variant only when the result can carry secrets.
- The connection string must be JDBC and may only carry the allowed JDBC properties listed above.
