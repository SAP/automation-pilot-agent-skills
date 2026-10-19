# HTTP Requests

Call a REST/HTTP endpoint from a command step. This is the most common executor in
authored commands: it reaches any SAP or third-party API that the pattern cookbook's
patterns (create-and-poll, chaining, retry) build on.

Resolve the executor contract live before use (`autopi commands contract`). Field tables are
a 2026-10-14 snapshot, not proof of the target tenant's contract.

## Contents

- [When to use](#when-to-use)
- [Executor IDs and variants](#executor-ids--variants)
- [Direct-request inputs and authentication](#key-inputs-httprequest1)
- [Destination requests](#key-inputs-httprequestviadestination1)
- [Outputs](#outputs--how-to-read-them)
- [Response transformation](#response-body-transformation)
- [Minimal snippet](#minimal-snippet)
- [Gotchas](#gotchas)

## When to use

- Call an API directly with credentials you pass at execution time → `HttpRequest:1`.
- Call a target the operator already manages as an **SAP BTP destination** (URL + auth
  resolved from the destination, no credential fields) → `HttpRequestViaDestination:1`.
- The response body can contain secrets (tokens, keys) → use the **`Sensitive*`** variant so
  the body is masked in logs/UI.
- A fitting higher-level catalog command already wraps the call (e.g. a `cf-sapcp:*` command).
  Prefer it over rebuilding the request by hand; the discovery skill finds it.

## Executor IDs & variants

| Executor | Use for |
|----------|---------|
| `http-sapcp:HttpRequest:1` | Direct HTTP call; auth supplied via input fields |
| `http-sapcp:HttpRequestViaDestination:1` | Call via a named BTP destination (no credential fields) |
| `http-sapcp:SensitiveHttpRequest:1` | As above; response treated as sensitive (masked) |
| `http-sapcp:SensitiveHttpRequestViaDestination:1` | Destination variant with sensitive response |

## Key inputs: `HttpRequest:1`

Required: **`url`**, **`method`** (`GET`/`POST`/`PUT`/`DELETE`/`PATCH`).

| Field | Default | Notes |
|-------|---------|-------|
| `body` | — | Request payload (string). Build JSON as a literal envelope with per-field `$()`. |
| `headers` | — | Object. An `Authorization` header here overrides `user`/`password`. |
| `timeout` | `3` | **1–90 seconds.** Out-of-range is rejected. |
| `successResponseCodes` | `["2xx"]` | Non-matching status **fails the executor**. Widen to read a non-2xx status in a later condition. |
| `succeedOnTimeout` | `false` | `true` → a timeout yields status `-1`, body `null`, headers `{}` instead of failing. |
| `responseBodyTransformer` | — | Dynamic-expression syntax **without** an outer `$()`; trims the body before it crosses the step boundary. See "Response body transformation" below. |
| `csrfUrl` | — | GET this URL first to obtain an `X-CSRF-TOKEN`. |
| `omitBodyInErrorMessage` | `false` | `true` keeps a secret-bearing body out of the error message. |
| `sccEnabled` / `sccLocationId` / `sccTechnicalUserPropagation` | `false` | Route via SAP Cloud Connector to private endpoints. |
| `trustAnyCert` / `trustedCerts` / `clientCert` | — | TLS trust overrides. Never set `trustAnyCert:"true"` in production. |

### Authentication (pick one method)

| Method | Fields |
|--------|--------|
| None | — |
| Basic | `user` + `password` (no `tokenUrl`) |
| OAuth client credentials | `clientId` + `clientSecret` + `tokenUrl` |
| OAuth password grant | `user` + `password` + `tokenUrl` + `clientId` |
| OAuth refresh token | `refreshToken` + `clientId` + `clientSecret` + `tokenUrl` (takes precedence over `user`/`password`) |
| Pre-fetched bearer | `authorizationHeader: "Bearer …"` (overrides every other mechanism) |
| IAS cert / mTLS | `clientCert` (cert+key concatenated) + `tokenUrl` |

`tokenType` (`ACCESS` default / `ID`) selects the OAuth token type. Set only the fields the
chosen method needs; the executor fetches OAuth tokens itself.

For any OAuth flow, **Automation Pilot fetches, caches, and rotates the access token for you**: it
obtains a token from `tokenUrl` and refreshes it when it expires. You supply only the credentials
(`clientId`/`clientSecret`, `user`/`password`, or `refreshToken`). You never pre-fetch a token,
track its lifetime, or renew it by hand.

**Basic** (`user` + `password`):

```json
"input": {
    "method": "POST",
    "url": "$(.execution.input.jenkinsUrl)/job/.../build",
    "user": "$(.execution.input.user)",
    "password": "$(.execution.input.password)"
}
```

**OAuth client credentials** (`clientId` / `clientSecret` / `tokenUrl`, and the executor fetches
the token). `clientCert`, cert+key concatenated, adds the mTLS variant:

```json
"input": {
  "method": "GET", "url": "$(.execution.input.serviceKey.sm_url)/v1/service_instances",
  "clientId": "$(.execution.input.serviceKey.clientid)",
  "clientSecret": "$(.execution.input.serviceKey.clientsecret)",
  "tokenUrl": "$(.execution.input.serviceKey.url)/oauth/token",
  "clientCert": "$(.execution.input.serviceKey.certificate + .execution.input.serviceKey.key)"
}
```

To select the plain vs. mTLS token endpoint from a service key, the idiom is
`if serviceKey.certificate == null then serviceKey.url else serviceKey.certurl end`.

**OAuth refresh token.** `refreshToken` is a first-class field; when set it takes precedence
over `user`/`password`:

```json
"input": {
    "clientId": "$(.execution.input.serviceKey.uaa.clientid)",
    "clientSecret": "$(.execution.input.serviceKey.uaa.clientsecret)",
    "tokenUrl": "$(.execution.input.serviceKey.uaa.url)/oauth/token",
    "refreshToken": "$(.execution.input.refreshToken)"
}
```

**Pre-fetched bearer token.** Pass a token produced by a prior step raw via
`authorizationHeader` (overrides every other mechanism):

```json
"input": {
    "method": "GET",
    "url": "$(.execution.input.concourseUrl)/api/v1/...",
    "authorizationHeader": "Bearer $(if .auth.executed then .auth.output.accessToken else .execution.input.accessToken end)"
}
```

## Key inputs: `HttpRequestViaDestination:1`

Required: **`destinationName`**, **`method`**. No credential fields; the destination supplies
URL and auth.

| Field | Default | Notes |
|-------|---------|-------|
| `uri` | — | Path appended to the destination's base URL. |
| `destinationLevel` | `subaccount` | or `instance`. |
| `destinationServiceHeaders` | — | e.g. `x-user-token` for identity propagation. |
| `serviceKey` | — | Only when the destination lives in another subaccount (sensitive). |
| `headers`, `body`, `timeout`, `successResponseCodes`, `succeedOnTimeout`, `responseBodyTransformer`, `omitBodyInErrorMessage` | | Same semantics as `HttpRequest:1`. |

## Outputs & how to read them

Both executors output: `status` (number), `body` (**string**), `headers` (object), `url`,
`method`, `size`, `time`.

- The **body is a string.** Parse before reading a field: `$(.alias.output.body | toObject.key)`
  (arrays: `| toArray`). This boundary rule is the #1 source of null reads.
- Read a response header case-insensitively:
  `$(.alias.output.headers | getCaseInsensitive("location"))`.
- `status` is `-1` on timeout/no-response (with `succeedOnTimeout:"true"`), `0` on no response.

## Response body transformation

`responseBodyTransformer` runs dynamic-expression syntax on the response body **without** the
outer `$()` wrapper. The transformer field is a normal string, so it can still contain `$()`
substitutions resolved from execution data before the response-body expression runs. The body
arrives as a string, so start with `toObject` when the response is a JSON object. Use it to drop
large or irrelevant fields before the body crosses into the rest of the command.

```jq
// keep a count and a trimmed item list
toObject | {num_items, items: (.items | map({id, platform_id}))}

// object shorthand that keeps four fields
toObject | {name, guid, state, lifecycle}

// pull a nested value out of each resource and join to a CSV
toObject.resources | map(.relationships.service_instance.data.guid) | join(",")
```

## Minimal snippet

```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "alias": "getResource",
  "input": {
    "method": "GET",
    "url": "$(.regionData.cfApiUrl)/v3/apps",
    "headers": "{\"Accept\":\"application/json\"}",
    "user": "$(.execution.input.user)",
    "password": "$(.execution.input.password)",
    "tokenUrl": "$(.regionData.uaaTokenUrl)",
    "clientId": "cf",
    "timeout": "20"
  },
  "description": null, "progressMessage": null,
  "initialDelay": null, "pause": null,
  "when": null, "validate": null, "autoRetry": null, "repeat": null,
  "errorMessages": []
}
```

## Gotchas

- **A non-2xx status fails the step before any condition reads it.** To branch on a 4xx/5xx or
  a `-1` timeout in `validate`, `repeat.until`, `autoRetry.when`, or `errorMessages`, widen
  `successResponseCodes` to include those codes and set `succeedOnTimeout:"true"`. This is the
  single most common HTTP authoring mistake. See the create-and-poll / retry patterns in the cookbook.
- `timeout` outside 1–90 is rejected.
- A sensitive input alone (e.g. `password`) does **not** require the `Sensitive*` variant. Use it
  only when the **output** (response body) can contain secrets.
