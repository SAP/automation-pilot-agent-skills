# A2A Adapter Model

The exact JSON schema and field rules for an Automation Pilot A2A adapter (agent)
definition. An A2A adapter exposes Automation Pilot commands as an agent's *skills*, so
external AI agents can delegate work to it. It carries an LLM configuration, a
natural-language instruction prompt, and a set of skills that each map to a command.

## Shape

```json
{
  "name": "HANA Cloud Observability Agent",
  "description": "Agent for HANA Cloud observability",
  "enabled": true,
  "instructions": "Your job is to ... \n\nAnalyze and correlate ...",
  "modelConfiguration": {
    "name": "anthropic--claude-4.5-sonnet",
    "version": "latest",
    "timeout": 60
  },
  "maxIterations": 15,
  "skills": [
    {
      "id": "list-hana-cloud-instances",
      "commandId": "hanacloud-sapcp:ListHanaCloudInstances:1",
      "name": "list_hana_cloud_instances",
      "description": "List the available HANA Cloud instances and their basic details",
      "inputReferences": ["hanaops-<<<TENANT_ID>>>:HanaCloudCredentials:1"],
      "tags": ["hana-cloud", "discovery"],
      "examples": ["Please list all HANA Cloud instances associated with my account"]
    }
  ],
  "organization": "SAP SE",
  "documentationUrl": null
}
```

## Adapter fields

| Field | Notes |
|-------|-------|
| `id` | System-generated, returned on create/list. Address get/update/delete by `id`, not `name`. Omit when creating. |
| `name` | Human-readable. Pattern `^[a-zA-Z0-9-_ ]+$`, 3–64 chars. **Spaces are allowed** (unlike kebab-case MCP server names). |
| `description` | Short summary. Max 256 chars. Optional. |
| `enabled` | Activates/deactivates the whole adapter. `false` keeps it as a draft without serving requests. |
| `instructions` | **Required.** The agent's job prompt — role, reasoning, and required response shape. Max 8192 chars. Richer than an MCP server's "what this does" blurb. See [Instructions field](#instructions-field). |
| `modelConfiguration` | LLM config (see below). Optional. |
| `maxIterations` | **Required.** Max reasoning/tool-call iterations per request. 1–15, default 10. |
| `skills` | Array of skill objects (see below). |
| `organization` | **Required.** Owning organization. Max 256 chars (e.g. `"SAP SE"`). |
| `documentationUrl` | Optional docs link. Max 512 chars. May be `null`. |

### Model configuration object

| Field | Notes |
|-------|-------|
| `name` | Language model for agent evaluation (e.g. `anthropic--claude-4.5-sonnet`). Max 256 chars. Default to a current Claude model. |
| `version` | Model version. Max 32 chars. Use `"latest"` unless pinning is required. |
| `timeout` | Max seconds for a single model request. 10–300, default 20. Raise toward 60 for heavier reasoning (correlation/observability agents). |

`maxIterations` defaults to 10; raise toward 15 for agents that chain many skills to
correlate data and reason to a root cause. Keep it low for simple single-skill adapters.

## Skill fields

Each entry in `skills` maps one Automation Pilot command to an agent skill. Unlike MCP
tools, A2A skills carry **no hint booleans** (`destructiveHint`, `idempotentHint`,
`openWorldHint`, `readOnlyHint`) — the agent reasons about each skill from its
`description`, `name`, and `examples`, so author those three carefully.

| Field | Notes |
|-------|-------|
| `id` | **Required.** Machine identifier. Pattern `^[a-zA-Z0-9-_]+$`, 3–64 chars — **kebab-case, no spaces** (e.g. `get-hana-cloud-metric-values`). Unique within the adapter. Keep it stable. |
| `commandId` | **Required.** `<catalog>-<suffix>:<CommandName>:<version>`. Max 256 chars. See [Command ID format](#command-id-format). |
| `name` | **Required.** Human-readable. Pattern `^[a-zA-Z0-9-_ ]+$`, 3–64 chars. Convention: **snake_case, verb-prefixed** (e.g. `list_hana_cloud_instances`). Spaces are technically allowed but the established convention uses snake_case. |
| `description` | What the skill returns or does — one clear sentence written so the agent can decide *when* to pick it over another. Max 1024 chars. |
| `inputReferences` | Credential/input references injected at runtime (unique items). See [Input references](#input-references). Use `[]` if credentials are embedded in the command. |
| `tags` | **Array of strings** for grouping/discovery (e.g. `["hana-cloud", "metrics"]`). Note: an array, not the `{}` object used by command/MCP-tool tags. Use `[]` when none apply. |
| `examples` | 1–3 realistic natural-language prompts that should route to this skill, phrased the way a real user would ask. Each max 256 chars. |

### Verb prefixes for skill `name`

| Prefix | Usage |
|--------|-------|
| `list_` | List/enumerate collections |
| `get_` | Retrieve a single resource or value |
| `fetch_` | Retrieve data from external sources |
| `create_` | Create a new resource |
| `update_` | Modify an existing resource |
| `delete_` | Remove a resource |
| `start_` / `stop_` / `restart_` | Lifecycle transitions |
| `enable_` | Enable a feature/capability |
| `send_` / `trigger_` | Send a message/event or trigger an action |

## Command ID format

Command IDs follow `<catalog>-<suffix>:<CommandName>:<version>`.

| Suffix | Usage | Example |
|--------|-------|---------|
| `-sapcp` | SAP-provided (shared/public) commands | `hanacloud-sapcp:ListHanaCloudInstances:1` |
| `-<<<TENANT_ID>>>` | Tenant-specific (custom) commands | `capops-<<<TENANT_ID>>>:GetIncident:1` |

`<<<TENANT_ID>>>` is a placeholder substituted at deployment time with the actual tenant
identifier.

### Known catalog prefixes

| Prefix | Domain |
|--------|--------|
| `ans` | Alert Notification Service |
| `applm` | Application Lifecycle Management |
| `calmhm` | Cloud ALM Health Monitoring |
| `capops` | Custom operations (tenant-specific) |
| `cf` | Cloud Foundry |
| `cld` | Cloud Landscape Directory |
| `ctms` | Cloud Transport Management |
| `hanacloud` | HANA Cloud (observability / instance operations) |
| `hanalm` | HANA Cloud Lifecycle Management |
| `jira` | Jira Integration |
| `monitoring` | Monitoring |
| `welcome` | Welcome/onboarding (credential inputs) |

## Input references

Input references inject pre-configured credentials into the command at runtime. Format:
`<catalog>-<<<TENANT_ID>>>:<InputName>:<version>`.

| Input Reference | Domain | Used By |
|----------------|--------|---------|
| `hanaops-<<<TENANT_ID>>>:HanaCloudCredentials:1` | HANA Cloud | HANA Cloud Observability |
| `capops-<<<TENANT_ID>>>:BtpCredentials:1` | BTP / Cloud Foundry | BTP Resource Discovery, App Logs & Metrics |
| `capops-<<<TENANT_ID>>>:AnsCredentials:1` | Alert Notification Service | ANS Event Producer, App Logs & Metrics |
| `capops-<<<TENANT_ID>>>:JiraDestination:1` | Jira | Incident Management |
| `welcome-<<<TENANT_ID>>>:CldDefaultValues:1` | Cloud Landscape Directory | Cloud Landscape Directory |
| `welcome-<<<TENANT_ID>>>:CtmsCredentials:1` | Cloud Transport Management | Cloud Transport Management |
| `welcome-<<<TENANT_ID>>>:ServiceManager:1` | SAP Service Manager | HANA Cloud Lifecycle Management |

- Most skills have exactly **one** input reference for their credential type.
- Use an **empty array `[]`** when credentials are embedded directly in the command.
- A single adapter can mix credential types if its skills span different services.
- The same credential reference can — and usually should — be reused across all skills in
  an adapter.

## Instructions field

The `instructions` field is the agent's **job prompt** — it defines the agent's role, how
it reasons, and how it presents answers. Structure it as:

1. **Role / job** — one sentence stating what the agent does and over what scope ("Your job is to ...").
2. **Analysis** — what the agent should correlate, detect, or determine.
3. **Response format** — how answers must be structured (concise; highlight the issue, the evidence, the root cause, next steps).

Name the underlying service/domain, spell out the required reasoning, and define the
response shape so answers are consistent. Use `\n` for newlines within the JSON string.

## Validation checklist

- [ ] `name` present, 3–64 chars, matches `^[a-zA-Z0-9-_ ]+$`
- [ ] `instructions` present and clearly describes the agent's job and response format
- [ ] `maxIterations` present and within 1–15
- [ ] `organization` present (e.g. `"SAP SE"`)
- [ ] `enabled` set appropriately (`true` active, `false` draft)
- [ ] `modelConfiguration` (if present) has a valid `name`, `version`, and `timeout` (10–300)
- [ ] Every skill has `id`, `commandId`, and `name`
- [ ] Each skill `id` is kebab-case with **no spaces**, unique within the adapter
- [ ] Each skill `name` follows the snake_case verb convention
- [ ] `commandId` follows `<catalog>-<suffix>:<Name>:<version>`
- [ ] `inputReferences` uses the correct credential type for the domain (or `[]` if embedded)
- [ ] `tags` is an **array of strings** (not an object)
- [ ] `examples` contains realistic trigger prompts
- [ ] The output is valid JSON

## Reference examples

### Minimal single-skill adapter

```json
{
  "name": "ANS Event Agent",
  "description": "Agent that sends Alert Notification Service events",
  "enabled": true,
  "instructions": "Your job is to send Alert Notification Service events on behalf of the user. When asked to notify about an operational incident, call the send_ans_event skill with the provided details and confirm the event was published.",
  "modelConfiguration": {
    "name": "anthropic--claude-4.5-sonnet",
    "version": "latest",
    "timeout": 20
  },
  "maxIterations": 5,
  "skills": [
    {
      "id": "send-ans-event",
      "commandId": "ans-sapcp:SendEvent:1",
      "name": "send_ans_event",
      "description": "Send an event through the Alert Notification Service",
      "inputReferences": ["capops-<<<TENANT_ID>>>:AnsCredentials:1"],
      "tags": ["ans", "notifications"],
      "examples": ["Send an ANS alert that the payment service is down"]
    }
  ],
  "organization": "SAP SE",
  "documentationUrl": null
}
```

### Full observability adapter (HANA Cloud Observability Agent)

```json
{
  "name": "HANA Cloud Observability Agent",
  "description": "Agent for HANA Cloud observability",
  "enabled": true,
  "instructions": "Your job is to discover SAP HANA Cloud instances across the user’s SAP BTP landscape and monitor their health, metrics, and alerts.\n\nAnalyze and correlate multiple metrics and alerts to:\n- Identify the likely root cause of performance, availability, or resource-related issues.\n- Provide a clear health overview of a selected HANA Cloud instance.\n- Troubleshoot detected issues and recommend actionable remediation steps.\n\nYour responses must be concise, structured, and easy to understand. Clearly highlight:\n- The detected issue or health status.\n- The key evidence and correlated metrics supporting the assessment.\n- The likely root cause, when it can be determined.\n- The recommended next steps to resolve or further investigate the issue",
  "modelConfiguration": {
    "name": "anthropic--claude-4.5-sonnet",
    "version": "latest",
    "timeout": 60
  },
  "maxIterations": 15,
  "skills": [
    {
      "id": "list-hana-cloud-instances",
      "commandId": "hanacloud-sapcp:ListHanaCloudInstances:1",
      "name": "list_hana_cloud_instances",
      "description": "List the available HANA Cloud instances and their basic details",
      "inputReferences": ["hanaops-<<<TENANT_ID>>>:HanaCloudCredentials:1"],
      "tags": ["hana-cloud", "discovery"],
      "examples": ["Please list all HANA Cloud instances associated with my account"]
    },
    {
      "id": "get-hana-cloud-metric-definitions",
      "commandId": "hanacloud-sapcp:GetHanaCloudMetricDefinitions:1",
      "name": "get_hana_cloud_metric_definitions",
      "description": "List all available metric definitions for HANA Cloud databases and details including its type and supported aggregations.",
      "inputReferences": ["hanaops-<<<TENANT_ID>>>:HanaCloudCredentials:1"],
      "tags": ["hana-cloud"],
      "examples": [
        "What metrics are currently available on my HANA Cloud database?",
        "Which metrics are related to CPU usage?"
      ]
    },
    {
      "id": "get-hana-cloud-metric-values",
      "commandId": "hanacloud-sapcp:GetHanaCloudMetricValues:1",
      "name": "get_hana_cloud_metric_values",
      "description": "Get the exact values for the specified HANA Cloud metrics",
      "inputReferences": ["hanaops-<<<TENANT_ID>>>:HanaCloudCredentials:1"],
      "tags": ["hana-cloud", "metrics"],
      "examples": ["What is the CPU usage per the HDBCPU metric?"]
    },
    {
      "id": "get-hana-cloud-instance-details",
      "commandId": "hanacloud-sapcp:GetHanaCloudInstance:1",
      "name": "get_hana_cloud_instance_details",
      "description": "Retrieve details about the configurations of the HANA Cloud database",
      "inputReferences": ["hanaops-<<<TENANT_ID>>>:HanaCloudCredentials:1"],
      "tags": [],
      "examples": [
        "What is the version of my HANA Cloud database?",
        "How is my HANA Cloud database sized?",
        "Is HA/DR configured on my HANA Cloud database?"
      ]
    }
  ],
  "organization": "SAP SE",
  "documentationUrl": null
}
```

## Troubleshooting

| Symptom | Cause | Resolution |
|---|---|---|
| Duplicate skill IDs | Two skills share the same `id`. | Each skill `id` must be unique within the adapter. Use descriptive kebab-case ids. |
| Invalid skill `id` (pattern rejected) | The `id` contains spaces or disallowed characters. | `id` must match `^[a-zA-Z0-9-_]+$`, 3–64 chars. Put the human-readable label in `name`. |
| Invalid `commandId` format | Missing separator, wrong suffix, or missing version. | Verify catalog prefix, suffix (`-sapcp` SAP-provided, `-<<<TENANT_ID>>>` custom), PascalCase name, integer version. |
| Missing required adapter fields | One of `name`, `instructions`, `maxIterations`, `organization` is absent. | All four are required. `maxIterations` within 1–15; `organization` set. |
| Adapter not responding | `enabled` is `false`. | Set `enabled: true`. Use `false` only for drafts. |
| `412 Precondition Failed` on update/delete | The `--if-match` ETag no longer matches the current version. | Re-fetch with `autopi a2a-adapters get <id> --show-etag`, then retry with the fresh ETag. |
