# Exit codes & troubleshooting

`autopi` returns a distinct exit code per failure class so scripts can branch on the
*kind* of failure — mirroring curl's split between transport errors and HTTP status.

| Code | Name | Meaning | Typical cause |
|------|------|---------|---------------|
| 0 | ok | request succeeded (2xx) | — |
| 1 | usage | bad flags or arguments | missing positional id, both `-f` and `--data`, unknown verb |
| 2 | config | required configuration missing | `AUTOMATION_PILOT_HOSTNAME`/`AUTOMATION_PILOT_TOKEN` unset |
| 3 | network | no HTTP response | DNS failure, TLS error, connection refused, timeout |
| 4 | http-4xx | server returned 3xx/4xx (redirects are never followed) | 400/401/403/404/409/429 |
| 5 | http-5xx | server returned 5xx | 500/503 |
| 6 | lint | `autopi lint` reported findings | a command/input JSON failed a check |

On a non-2xx status (`4`/`5`), `autopi` still prints the response body the API returned,
except Input paths, whose error bodies are suppressed to prevent secret echoes. Redirects
are not followed and return exit 4. Output or export failures return exit 1.

## HTTP status reference

| Code | Meaning |
|------|---------|
| 200 | Success (GET, PUT) |
| 201 | Created (POST) |
| 204 | No Content (DELETE) |
| 400 | Bad request — invalid JSON or missing required field |
| 401 | Unauthorized — invalid credentials |
| 403 | Forbidden — insufficient permissions (`Write` needed for create/update/delete) |
| 404 | Not found — resource doesn't exist |
| 409 | Conflict — resource already exists or invalid state |
| 429 | Too many requests — rate limited |
| 500 | Internal server error |
| 503 | Service unavailable |

## Troubleshooting

**Exit 2 (config).** A required env var is unset. Set `AUTOMATION_PILOT_HOSTNAME` and
`AUTOMATION_PILOT_TOKEN`. You can still preview any request offline with
`--print-request`, which needs no credentials.

**Exit 4, HTTP 401.** The token is invalid or expired for the target host. Verify the token
belongs to the tenant at `AUTOMATION_PILOT_HOSTNAME`.

**Exit 4, HTTP 403.** The user lacks the required permission. List/get need `Read`;
create/update/delete need `Write`.

**Exit 4, HTTP 409 on create.** The resource already exists. Use `deploy` (GET-then-
PUT/POST upsert) instead of `create`, or `update`/`patch` the existing one.

**Exit 4, HTTP 404 on update.** The id in the URL doesn't match any resource. Command/Input IDs are
`catalog:name:version`; MCP/A2A and schedule IDs are server-generated. Confirm with `autopi <resource> get <id> --status-only`.

**Exit 3 (network).** No response reached the client. Check the hostname and
connectivity. `--timeout 60s` extends the deadline; `--insecure` skips TLS verification
(deliberate escape hatch only).

**Checking existence.** `autopi <resource> get <id> --status-only` prints just the status
code (`200`/`404`) and exits 0 regardless — the replacement for
`curl -o /dev/null -w "%{http_code}"`.
