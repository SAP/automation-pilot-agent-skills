# Mandatory Description Patterns

⚠️ **All descriptions of the following parameters must be one of these exact sentences or start with them.** No variations, alternatives, or creative rewording allowed for these Input and Output keys!

## Input Parameters

**Authentication:**
- **password**
  - description
    - The password for the specified technical user or the client secret for the specified OAuth 2.0 client ID to be used for authentication. Related input keys: 'user' or 'tokenUrl'
    - The password of a service account with [permissions]. Related input keys: 'user'
    - The password of the provided user
  - type - string | sensitive 🔒
- **user**
  - description
    - The name of a technical user or an OAuth 2.0 client ID to be used for authentication. Related input keys: 'password' or 'tokenUrl'
    - The ID (username) of a service account with [permissions]. Related input keys: 'password'
    - The user ID or the email of a Cloud Foundry user to be used for authentication
    - The username of the [service/system]
  - type - string
- **refreshToken**
  - description
    - An OAuth 2.0 refresh token to be used for authentication. If 'refreshToken' is passed, 'user' and 'password' will be ignored. Related input keys: 'clientId', 'clientSecret', 'tokenUrl'
    - An OAuth 2.0 refresh token which will be used to get a new access token via the Refresh Token grant type. Related input keys: 'clientId', 'clientSecret', 'tokenUrl'
  - type - string | sensitive 🔒

**Region & Organization:**
- **region**
  - description
    - The technical name of the Cloud Foundry region. Example: cf-eu10, cf-eu10-002
    - The technical name of the Neo region. Example: neo-eu1, neo-us1
  - type - string
  - Allowed values should be used
- **subAccount**
  - description
    - The name or the ID of the Cloud Foundry organization. Examples: my-org-name-1, 0ffeb410-5f78-0000-af5c-5b26baf46623
  - type - string

**Resources & Services:**
- **resourceName**
  - description
    - The technical name of the [resource type]. Example: [examples]
  - type - string
- **serviceKey**
  - description
    - A service key for [service]
  - type - object | sensitive 🔒
- **name**
  - description
    - The name of the [object]
  - type - string

## Output Parameters

**Status & Response:**
- **status**
  - description
    - The status of the [object]
    - The status code of the response. Examples: 200, 301, 404, 503
    - The status code of the response. Examples: 200, 301, 404, 503. In case of no response or a timeout, the status codes may be 0 and -1.
  - types - number, string, object
- **responseCode**
  - description
    - The HTTP response code
  - type - number
- **state**
  - description
    - The state of the [object/entity]. Examples: [examples]
  - type - string

**Data Collections:**
- **resourceInstancesStates**
  - description
    - An array of all application instances after command execution
  - type - array
- **output**
  - description
    - The original response from [service/API]
  - types - array, string, object
- **result**
  - description
    - The result of the [operation]
  - types - array, string, object

**System Operations:**
- **exitCode**
  - description
    - The exit code returned by the script execution
  - type - number
- **instanceId**
  - description
    - The ID of the [instance/object]
  - type - string
