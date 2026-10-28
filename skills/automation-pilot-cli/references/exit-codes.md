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
| 7 | partial-bulk-failure | bulk action contains failed individual results | HTTP 200 with succeeded=false items |

On a non-2xx status (`4`/`5`), `autopi` still prints the response body the API returned,
except Input paths, whose error bodies are suppressed to prevent secret echoes. Redirects
are not followed and return exit 4. Output or export failures return exit 1.

## HTTP status reference

| Code | Meaning |
|------|---------|
| 200 | Successful HTTP request; waited trigger completion or independent bulk outcomes require status/result inspection |
| 201 | Created (POST) |
| 202 | Accepted — asynchronous processing accepted; CLI exits **0**. Verify action completion with `executions get`. |
| 204 | No Content (DELETE) |
| 400 | Bad request — invalid JSON or missing required field |
| 401 | Unauthorized — invalid credentials |
| 403 | Forbidden — check the operation-specific Read, Write, or Execute scope |
| 404 | Not found — resource doesn't exist |
| 409 | Conflict — resource already exists or invalid state |
| 412 | ETag precondition failed — re-read and reconcile |
| 429 | Too many requests — rate limited |
| 500 | Internal server error |
| 503 | Service unavailable |

## Troubleshooting

**Exit 2 (config).** A required env var is unset. Set `AUTOMATION_PILOT_HOSTNAME` and
`AUTOMATION_PILOT_TOKEN`. You can still preview any request offline with
`--print-request`, which needs no credentials.

**Exit 4, HTTP 401.** The token is invalid or expired for the target host. Verify the token
belongs to the tenant at `AUTOMATION_PILOT_HOSTNAME`.

**Exit 4, HTTP 403.** List/get need Read; Content writes and completed-execution deletion
need Write; schedules and execution actions/triggers need Execute. See
[API contracts](api-contract.md) for the operation-specific map.

**Exit 4, HTTP 412.** Re-read the resource/ETag and reconcile the intended change.
Do not replay an outdated conditional write.

**Exit 7.** Bulk results contain failed entries. Preserve successes and inspect failures
before requesting a targeted retry. The CLI sends each bulk request once.

**429/503 reads.** Optional --read-retries N retries GET/HEAD only within the total
request timeout. Mutations are never automatically retried.

**Exit 4, HTTP 409 on create.** Read the existing resource and compare it with the
intended definition. Choose update/patch or deploy only when replacement is within
the user's established scope; a create conflict does not itself authorize replacement.

**Exit 4, HTTP 404 on update.** The id in the URL doesn't match any resource. Command/Input IDs are
`catalog:name:version`; MCP, webhook, and schedule IDs are server-generated. Confirm with `autopi <resource> get <id> --status-only`.

**Exit 3 (network).** No response reached the client. Check the hostname and
connectivity. `--timeout 60s` extends the deadline; `--insecure` skips TLS verification
(deliberate escape hatch only).

**Checking existence.** `autopi <resource> get <id> --status-only` prints just the status
code (`200`/`404`) and exits 0 regardless — the replacement for
`curl -o /dev/null -w "%{http_code}"`.
