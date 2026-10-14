# Input patterns

Inputs are **reusable parameter sets** stored in Automation Pilot. They let you save
credentials, configurations, or commonly-used values that can be referenced when
triggering command executions.

## Input Structure

```json
{
  "name": "MyCfCredentials",
  "catalog": "mycatalog-xxx",
  "description": "CF credentials for eu10 region",
  "version": 1,
  "keys": {
    "region": { "type": "string", "sensitive": false },
    "user": { "type": "string", "sensitive": false },
    "password": { "type": "string", "sensitive": true }
  },
  "values": {
    "region": "cf-eu10",
    "user": "technical-user",
    "password": "<your-password>"
  },
  "tags": {}
}
```

- **keys**: Defines the schema (type and sensitivity) for each value
- **values**: The actual stored values (always JSON strings on the wire — see the
  "Values Are Always Strings on the Wire" table in the skill)
- **sensitive: true**: Value is masked in logs and the UI

## Common Input Patterns

### CF Credentials (for applm-sapcp, cf-sapcp commands)

```json
{
  "name": "CfCredentials-EU10",
  "catalog": "mycatalog-xxx",
  "description": "Cloud Foundry credentials for EU10",
  "version": 1,
  "keys": {
    "region": { "type": "string", "sensitive": false },
    "user": { "type": "string", "sensitive": false },
    "password": { "type": "string", "sensitive": true }
  },
  "values": {
    "region": "cf-eu10",
    "user": "cf-technical-user@example.com",
    "password": "your-password"
  }
}
```

### Service Key (for sm-sapcp, aicore-sapcp commands)

```json
{
  "name": "HanaCloudServiceKey",
  "catalog": "mycatalog-xxx",
  "description": "HANA Cloud service binding credentials",
  "version": 1,
  "keys": {
    "serviceKey": { "type": "object", "sensitive": true }
  },
  "values": {
    "serviceKey": "{\"url\":\"https://hana-instance.hana.cloud.sap\",\"user\":\"DBADMIN\",\"password\":\"<your-password>\",\"certificate\":\"...\"}"
  }
}
```

### API Token (for jira-sapcp, github-sapcp commands)

```json
{
  "name": "JiraCredentials",
  "catalog": "mycatalog-xxx",
  "description": "JIRA API credentials",
  "version": 1,
  "keys": {
    "host": { "type": "string", "sensitive": false },
    "user": { "type": "string", "sensitive": false },
    "password": { "type": "string", "sensitive": true }
  },
  "values": {
    "host": "https://jira.example.com",
    "user": "automation@example.com",
    "password": "api-token-here"
  }
}
```

### Kubernetes Config (for kubernetes-sapcp commands)

```json
{
  "name": "K8sClusterConfig",
  "catalog": "mycatalog-xxx",
  "description": "Kubernetes cluster kubeconfig",
  "version": 1,
  "keys": {
    "kubeconfig": { "type": "object", "sensitive": true }
  },
  "values": {
    "kubeconfig": "{\"apiVersion\":\"v1\",\"kind\":\"Config\",\"clusters\":[...],\"users\":[...],\"contexts\":[...]}"
  }
}
```

## Best Practices

1. **Separate inputs by environment** — distinct inputs for dev/staging/prod.
2. **Mark sensitive values** — always set `"sensitive": true` for passwords, tokens, keys.
3. **Use descriptive names** — include region/environment (e.g. `CfCredentials-EU10-Prod`).
4. **Keep inputs minimal** — store only reusable values; command-specific values go in
   `inputValues` at trigger time.
