# Descriptions and sensitivity

Read this when describing a command or its keys, or when deciding which values and executor outputs
must be marked sensitive. The command contract should explain caller-visible behavior rather than
the implementation behind it.

## Sensitivity

- Mark every secret-bearing input or output key `sensitive: true`.
- Use a `Sensitive*` executor variant when that executor's output can contain sensitive data.
- A sensitive input alone does not require a sensitive-output variant when the ordinary executor
  protects the input and its output is non-sensitive.
- Describe a secret's purpose and expected source without including an example secret value.

### Sensitive outputs read back redacted

A sensitive output is redacted on API readback: `get_execution` shows it as `null` or `****`. The
engine propagates sensitivity through derived values, so `get_execution` also redacts a value
computed from a sensitive output. Readback alone cannot distinguish this redaction from a broken
expression.

- Assert a sensitive value's expected shape in a `validate` on a composing step. The engine
  evaluates the real value and returns only the validation result.
- Keep the `Sensitive*` executor; a plain variant exposes the secret in logs and readback.

## Standard input descriptions

Use one matching sentence exactly, or begin with it and append task-specific constraints.

**`password`** (string, sensitive):

- `The password for the specified technical user or the client secret for the specified OAuth 2.0 client ID to be used for authentication. Related input keys: 'user' or 'tokenUrl'`
- `The password of a service account with [permissions]. Related input keys: 'user'`
- `The password of the provided user`

**`user`** (string):

- `The name of a technical user or an OAuth 2.0 client ID to be used for authentication. Related input keys: 'password' or 'tokenUrl'`
- `The ID (username) of a service account with [permissions]. Related input keys: 'password'`
- `The user ID or the email of a Cloud Foundry user to be used for authentication`
- `The username of the [service/system]`

**`refreshToken`** (string, sensitive):

- `An OAuth 2.0 refresh token to be used for authentication. If 'refreshToken' is passed, 'user' and 'password' will be ignored. Related input keys: 'clientId', 'clientSecret', 'tokenUrl'`
- `An OAuth 2.0 refresh token which will be used to get a new access token via the Refresh Token grant type. Related input keys: 'clientId', 'clientSecret', 'tokenUrl'`

**`region`** (string, constrained by the appropriate reusable Input):

- `The technical name of the Cloud Foundry region. Example: cf-eu10, cf-eu10-002`
- `The technical name of the Neo region. Example: neo-eu1, neo-us1`

**`subAccount`** (string):

- `The name or the ID of the Cloud Foundry organization. Examples: my-org-name-1, 0ffeb410-5f78-0000-af5c-5b26baf46623`

Other standard inputs:

- `resourceName`: `The technical name of the [resource type]. Example: [examples]`
- `serviceKey`: `A service key for [service]`
- `name`: `The name of the [object]`

## Standard output descriptions

**`status`** (number, string, or object):

- `The status of the [object]`
- `The status code of the response. Examples: 200, 301, 404, 503`
- `The status code of the response. Examples: 200, 301, 404, 503. In case of no response or a timeout, the status codes may be 0 and -1.`

Other standard outputs:

- `responseCode`: `The HTTP response code`
- `state`: `The state of the [object/entity]. Examples: [examples]`
- `resourceInstancesStates`: `An array of all application instances after command execution`
- `output`: `The original response from [service/API]`
- `result`: `The result of the [operation]`
- `exitCode`: `The exit code returned by the script execution`
- `instanceId`: `The ID of the [instance/object]`

Replace bracketed placeholders with the concrete resource, system, permissions, or examples. For
keys not listed here, state the caller-visible meaning, accepted shape, and constraints directly.

