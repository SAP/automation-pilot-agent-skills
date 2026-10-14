---
name: automation-pilot-a2a-server-generation
description: Create, configure, and deploy SAP Automation Pilot A2A adapter (agent) definitions and their skills. Use when building A2A adapters/agents, exposing Automation Pilot commands as agent skills, setting up A2A agent configs, or deploying A2A adapter definitions via API.
---

# SAP Automation Pilot A2A Adapter Definition Generator

This skill guides the creation of A2A adapter definitions for SAP Automation Pilot. An A2A adapter (agent-to-agent) is an AI agent definition in JSON that exposes Automation Pilot commands as the agent's *skills*, so external AI agents can delegate work to it. The adapter carries an LLM configuration, a natural-language instruction prompt, and a set of skills that each map to an Automation Pilot command.

---

## JSON Schema

### Top-Level Structure

Every A2A adapter definition follows this structure:

```json
{
  "name": "Human Readable Agent Name",
  "description": "<Short summary of what the agent does>",
  "enabled": true,
  "instructions": "<Agent job prompt — what it does, how it reasons, how it answers>",
  "modelConfiguration": {
    "name": "anthropic--claude-4.5-sonnet",
    "version": "latest",
    "timeout": 60
  },
  "maxIterations": 15,
  "skills": [ ...skill objects... ],
  "organization": "SAP SE",
  "documentationUrl": null
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Human-readable adapter name. Pattern `^[a-zA-Z0-9-_ ]+$`, 3–64 chars. **Spaces are allowed** (unlike MCP server kebab-case names). |
| `description` | string | – | Short summary of the adapter. Max 256 chars. |
| `enabled` | boolean | – | Activates/deactivates the entire adapter. |
| `instructions` | string | ✅ | The agent's job prompt. Max 8192 chars. Describes what the agent does, how it analyzes, and how it should structure responses. |
| `modelConfiguration` | object | – | LLM configuration (see below). |
| `maxIterations` | integer | ✅ | Max reasoning/tool-call iterations per request. 1–15, default 10. |
| `skills` | array | – | Array of skill objects (see below). |
| `organization` | string | ✅ | Owning organization. Max 256 chars (e.g. `"SAP SE"`). |
| `documentationUrl` | string | – | Optional link to docs. Max 512 chars. May be `null`. |

### Model Configuration Object

```json
{
  "name": "anthropic--claude-4.5-sonnet",
  "version": "latest",
  "timeout": 60
}
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Language model used for agent evaluation (e.g. `anthropic--claude-4.5-sonnet`). Max 256 chars. |
| `version` | string | Model version. Defaults to `"latest"`. Max 32 chars. |
| `timeout` | integer | Max seconds allowed for a single model request. 10–300, default 20. |

### Skill Object Structure

Each entry in `skills` maps one Automation Pilot command to an agent skill:

```json
{
  "id": "list-hana-cloud-instances",
  "commandId": "<catalog>-<suffix>:<CommandName>:<version>",
  "name": "list_hana_cloud_instances",
  "description": "List the available HANA Cloud instances and their basic details",
  "inputReferences": [
    "<catalog>-<<<TENANT_ID>>>:<InputName>:<version>"
  ],
  "tags": ["hana-cloud", "discovery"],
  "examples": [
    "Please list all HANA Cloud instances associated with my account"
  ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | ✅ | Unique skill identifier. Pattern `^[a-zA-Z0-9-_]+$`, 3–64 chars. **No spaces** — use kebab-case (e.g. `list-hana-cloud-instances`). |
| `commandId` | string | ✅ | Reference to an Automation Pilot command: `<catalog>-<suffix>:<CommandName>:<version>`. Max 256 chars. |
| `name` | string | ✅ | Human-readable skill name. Pattern `^[a-zA-Z0-9-_ ]+$`, 3–64 chars. Convention: snake_case (e.g. `list_hana_cloud_instances`). |
| `description` | string | – | What the skill does. Max 1024 chars. Helps the agent decide when to invoke it. |
| `inputReferences` | string[] | – | Credential/input references injected at runtime (unique items). Use `[]` if credentials are embedded in the command. |
| `tags` | string[] | – | **Array of strings** for grouping/discovery (e.g. `["hana-cloud", "metrics"]`). Note: this is an array, not the `{}` object used by MCP tools. |
| `examples` | string[] | – | Natural-language prompts that should trigger this skill. Each max 256 chars. |

---

## Command ID Format

Command IDs follow the pattern: `<catalog>-<suffix>:<CommandName>:<version>`

### Suffix Rules

| Suffix | Usage | Example |
|--------|-------|---------|
| `-sapcp` | SAP-provided (shared/public) commands | `hanacloud-sapcp:ListHanaCloudInstances:1` |
| `-<<<TENANT_ID>>>` | Tenant-specific (custom) commands | `capops-<<<TENANT_ID>>>:GetIncident:1` |

The `<<<TENANT_ID>>>` is a placeholder substituted at deployment time with the actual tenant identifier.

### Known Catalog Prefixes

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

---

## Input References (Credential Injection)

Input references inject pre-configured credentials into the command at runtime. Format: `<catalog>-<<<TENANT_ID>>>:<InputName>:<version>`

### Known Credential Types

| Input Reference | Domain | Used By |
|----------------|--------|---------|
| `hanaops-<<<TENANT_ID>>>:HanaCloudCredentials:1` | HANA Cloud | HANA Cloud Observability |
| `capops-<<<TENANT_ID>>>:BtpCredentials:1` | BTP / Cloud Foundry | BTP Resource Discovery, App Logs & Metrics |
| `capops-<<<TENANT_ID>>>:AnsCredentials:1` | Alert Notification Service | ANS Event Producer, App Logs & Metrics |
| `capops-<<<TENANT_ID>>>:JiraDestination:1` | Jira | Incident Management |
| `welcome-<<<TENANT_ID>>>:CldDefaultValues:1` | Cloud Landscape Directory | Cloud Landscape Directory |
| `welcome-<<<TENANT_ID>>>:CtmsCredentials:1` | Cloud Transport Management | Cloud Transport Management |
| `welcome-<<<TENANT_ID>>>:ServiceManager:1` | SAP Service Manager | HANA Cloud Lifecycle Management |

### Rules

- Most skills have exactly **one** input reference for their credential type
- Use **empty array `[]`** when credentials are embedded directly in the command definition
- A single adapter can mix credential types if skills span different services
- The same credential reference can — and usually should — be reused across all skills in an adapter

---

## Skill Authoring Conventions

Unlike MCP tools, A2A skills carry **no hint booleans** (`destructiveHint`, `idempotentHint`, `openWorldHint`, `readOnlyHint`). Instead, the agent reasons about each skill from its `description`, `name`, and `examples`. Author those three carefully.

### `id` (machine identifier)

- **kebab-case**, no spaces (e.g. `get-hana-cloud-metric-values`)
- Unique within the adapter
- Keep it stable — it identifies the skill

### `name` (human-readable)

- Convention: **snake_case**, verb-prefixed (e.g. `list_hana_cloud_instances`)
- Spaces are technically allowed by the pattern, but the established convention (and the sample agent) use snake_case

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

### `description`

- One clear sentence describing what the skill returns or does
- Written so the agent can decide **when** to pick this skill over another

### `examples`

- 1–3 realistic natural-language user prompts that should route to this skill
- Phrase them the way a real user would ask (e.g. `"What is the CPU usage per the HDBCPU metric?"`)

### `tags`

- An **array of strings** used for grouping and discovery (e.g. `["hana-cloud", "metrics"]`)
- Use `[]` when no tags apply

---

## Model Configuration & Iterations Guidance

- **`modelConfiguration.name`**: default to a current Claude model, e.g. `anthropic--claude-4.5-sonnet`.
- **`version`**: use `"latest"` unless pinning a specific version is required.
- **`timeout`**: default 20s; raise toward 60s for agents that make heavier reasoning calls (e.g. correlation/observability agents).
- **`maxIterations`**: default 10. Raise toward the max of 15 for agents that must chain many skills to correlate data and reason to a root cause (e.g. an observability or troubleshooting agent). Keep it low for simple single-skill adapters.

---

## Instructions Field Guidelines

The `instructions` field is the agent's **job prompt** — it defines the agent's role, how it reasons, and how it must present answers. This is richer than an MCP server's "what this does" blurb.

### Structure a good instruction prompt

1. **Role / job** — one sentence stating what the agent does and over what scope.
2. **Analysis** — what the agent should correlate, detect, or determine.
3. **Response format** — how answers must be structured (concise, highlight the issue, evidence, root cause, next steps).

### Example (observability agent)

```
Your job is to discover SAP HANA Cloud instances across the user's SAP BTP landscape and monitor their health, metrics, and alerts.\n\nAnalyze and correlate multiple metrics and alerts to:\n- Identify the likely root cause of performance, availability, or resource-related issues.\n- Provide a clear health overview of a selected HANA Cloud instance.\n- Troubleshoot detected issues and recommend actionable remediation steps.\n\nYour responses must be concise, structured, and easy to understand.
```

### Guidelines

- Start with the agent's job ("Your job is to...")
- Name the underlying service/domain
- Spell out the reasoning the agent must perform
- Define the required response shape so answers are consistent
- Use `\n` for newlines within the JSON string value

---

## Validation Checklist

Before finalizing a generated A2A adapter definition, verify:

- [ ] `name` is present, 3–64 chars, matches `^[a-zA-Z0-9-_ ]+$`
- [ ] `instructions` is present and clearly describes the agent's job and response format
- [ ] `maxIterations` is present and within 1–15
- [ ] `organization` is present (e.g. `"SAP SE"`)
- [ ] `enabled` is set appropriately (`true` for active, `false` for draft)
- [ ] `modelConfiguration` (if present) has a valid `name`, `version`, and `timeout` (10–300)
- [ ] Every skill has `id`, `commandId`, and `name`
- [ ] Each skill `id` is kebab-case with **no spaces**, and unique within the adapter
- [ ] Each skill `name` follows the snake_case verb convention
- [ ] `commandId` follows the `<catalog>-<suffix>:<Name>:<version>` format
- [ ] `inputReferences` uses the correct credential type for the domain (or `[]` if embedded)
- [ ] `tags` is an **array of strings** (not an object)
- [ ] `examples` contains realistic trigger prompts
- [ ] The output is valid JSON

---

## Reference Examples

### Minimal single-skill adapter

```json
{
  "name": "ANS Event Agent",
  "description": "Agent that sends Alert Notification Service events",
  "enabled": true,
  "instructions": "Your job is to send Alert Notification Service events on behalf of the user. When asked to notify about an operational incident, call the send_event skill with the provided details and confirm the event was published.",
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
      "examples": [
        "Send an ANS alert that the payment service is down"
      ]
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
      "inputReferences": [
        "hanaops-<<<TENANT_ID>>>:HanaCloudCredentials:1"
      ],
      "tags": [
        "hana-cloud",
        "discovery"
      ],
      "examples": [
        "Please list all HANA Cloud instances associated with my account"
      ]
    },
    {
      "id": "get-hana-cloud-metric-definitions",
      "commandId": "hanacloud-sapcp:GetHanaCloudMetricDefinitions:1",
      "name": "get_hana_cloud_metric_definitions",
      "description": "List all available metric definitions for HANA Cloud databases and details including its type and supported aggregations.",
      "inputReferences": [
        "hanaops-<<<TENANT_ID>>>:HanaCloudCredentials:1"
      ],
      "tags": [
        "hana-cloud"
      ],
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
      "inputReferences": [
        "hanaops-<<<TENANT_ID>>>:HanaCloudCredentials:1"
      ],
      "tags": [
        "hana-cloud",
        "metrics"
      ],
      "examples": [
        "What is the CPU usage per the HDBCPU metric?"
      ]
    },
    {
      "id": "get-hana-cloud-instance-details",
      "commandId": "hanacloud-sapcp:GetHanaCloudInstance:1",
      "name": "get_hana_cloud_instance_details",
      "description": "Retrieve details about the configurations of the HANA Cloud database",
      "inputReferences": [
        "hanaops-<<<TENANT_ID>>>:HanaCloudCredentials:1"
      ],
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

---

## Examples

### Example 1: Create a simple single-skill adapter

User: "Create an A2A agent that can send ANS alerts"

1. Define the adapter with `name`, `description`, `enabled: true`, and a short agent `instructions` prompt
2. Add a `modelConfiguration` (default Claude model, `latest`, timeout 20) and a low `maxIterations` (e.g. 5)
3. Add one skill: `id: send-ans-event`, `name: send_ans_event`, mapping `ans-sapcp:SendEvent:1`
4. Attach `AnsCredentials` as an input reference, add `tags` and an `examples` prompt
5. Set `organization` and validate against the checklist

### Example 2: Create a multi-skill observability agent

User: "Build an A2A agent for monitoring HANA Cloud health"

1. Create discovery/read skills: `list_hana_cloud_instances`, `get_hana_cloud_metric_definitions`, `get_hana_cloud_metric_values`, `get_hana_cloud_instance_details`
2. Use the `hanacloud-sapcp` catalog prefix and `hanaops-<<<TENANT_ID>>>:HanaCloudCredentials:1` input reference on every skill
3. Write a rich `instructions` prompt: discover instances, correlate metrics/alerts, determine root cause, and define the response structure
4. Raise `maxIterations` to 15 and `timeout` to 60 so the agent can chain skills and reason
5. Give each skill a clear `description`, relevant `tags`, and realistic `examples`
6. Use the HANA Cloud Observability Agent reference above as a structural guide

---

# Deploying A2A Adapters via API

## API access

All calls use the `autopi` CLI — see [`../automation-pilot-cli/SKILL.md`](../automation-pilot-cli/SKILL.md) for setup (the `AUTOPI_*` environment variables, supported regional hosts, install, and the full command surface). Add the CLI's `scripts/` to your `PATH` so `autopi` runs directly.

The API requires the `GenAI` permission for all write operations.

## List A2A Adapters

Lists return metadata only (name, id, enabled, tags, …); add `--full` for complete
definitions including `skills`.

```bash
autopi a2a-adapters list
```

## Get A2A Adapter by ID

A2A adapters have a system-generated `id` field returned in the create/list responses. This is distinct from the human-readable `name` field. Always use the `id` for GET, update, and delete operations.

```bash
# First, find the ID from the list response
autopi a2a-adapters list --output raw | jq '[.[] | {name, id}]'

# Then fetch by ID (--show-etag prints the ETag response header, needed for optimistic-locked updates)
A2A_ADAPTER_ID="<id-from-list-response>"
autopi a2a-adapters get $A2A_ADAPTER_ID --show-etag
```

## Create A2A Adapter

```bash
A2A_ADAPTER_FILE="path/to/my-agent.json"
autopi a2a-adapters create -f "$A2A_ADAPTER_FILE"
```

The response includes an `ETag` header representing the created version.

## Update A2A Adapter

Use the system-generated `id` (not the `name`) in the URL. Optionally pass the current ETag via `--if-match` for optimistic locking — the update is rejected with `412` if the resource changed since you read it.

```bash
A2A_ADAPTER_ID="<id-from-list-response>"
A2A_ADAPTER_FILE="path/to/updated-agent.json"
ETAG="<etag-from-get-response>"   # optional

autopi a2a-adapters update $A2A_ADAPTER_ID -f "$A2A_ADAPTER_FILE" --if-match "$ETAG"
```

## Delete A2A Adapter

```bash
A2A_ADAPTER_ID="<id-from-list-response>"

autopi a2a-adapters delete $A2A_ADAPTER_ID
```

## Deploy A2A Adapter (Upsert Pattern)

Create if new, update if already deployed:

```bash
A2A_ADAPTER_FILE="my-agent.json"

autopi a2a-adapters deploy -f "$A2A_ADAPTER_FILE"
```

`deploy` upserts — it does a GET-then-PUT/POST (no `If-Match`): if the adapter already exists it updates it, otherwise it creates it.

---

## Troubleshooting

**Error:** Duplicate skill IDs in adapter
**Cause:** Two skills share the same `id` value.
**Solution:** Each skill `id` must be unique within the adapter. Use descriptive kebab-case ids (e.g. `list-instances`, `get-instance-details`).

**Error:** Invalid skill `id` (rejected by pattern)
**Cause:** The `id` contains spaces or disallowed characters.
**Solution:** `id` must match `^[a-zA-Z0-9-_]+$` (kebab-case, no spaces), 3–64 chars. Put the human-readable label in `name` instead.

**Error:** Invalid `commandId` format
**Cause:** Command ID doesn't follow the `<catalog>-<suffix>:<Name>:<version>` pattern — missing separator, wrong suffix, or missing version.
**Solution:** Verify catalog prefix, suffix (`-sapcp` for SAP-provided, `-<<<TENANT_ID>>>` for custom), PascalCase command name, and integer version number.

**Error:** Missing required adapter fields
**Cause:** One of `name`, `instructions`, `maxIterations`, or `organization` is absent.
**Solution:** These four are required. Ensure `maxIterations` is within 1–15 and `organization` is set (e.g. `"SAP SE"`).

**Error:** A2A adapter not responding to requests
**Cause:** The adapter `enabled` field is set to `false`.
**Solution:** Set `enabled: true` to activate the adapter. Use `false` only for drafts not yet ready for use.

**Error:** `412 Precondition Failed` on update or delete
**Cause:** The `If-Match` ETag you sent no longer matches the current version (the resource changed since you read it).
**Solution:** Re-fetch the adapter with GET to obtain the current ETag, then retry the update/delete with the fresh value.
