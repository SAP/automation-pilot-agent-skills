---
name: automation-pilot-connection
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: >
  Validates the configured SAP Automation Pilot connection and reports its non-secret account summary. Use when a user wants to test connectivity, confirm which tenant is configured, or inspect effective access.
---

# SAP Automation Pilot Connection

Resolve the installed verification launcher through the shared
`automation-pilot-cli` skill. For missing/invalid setup, run `autopi connection preflight`
first; it reports local variable presence and syntax without making API calls. Local
lint and authoring remain available without credentials. To test the connection,
call `autopi connection info` and report its non-secret summary. The command composes
`GET /server-info`, `/user-information`, and `/tenant` into `{connected, url, tenantId,
username, permissions, plan}` and never prints the credential. Setup through
`AUTOMATION_PILOT_HOSTNAME` and `AUTOMATION_PILOT_TOKEN` lives in
[`../automation-pilot-cli/SKILL.md`](../automation-pilot-cli/SKILL.md). Bearer is the
only supported authentication method. A `config error` (exit 2) means a required
connection variable is missing or invalid.

## Verdict

Open the response with one verdict on its own line, each prefixed with its emoji, then the supporting summary:

- ✅ **Healthy:** the call succeeded and contains the permissions needed for the current task: `Read` for discovery/review, `Write` for content changes and MCP administration, and `Execute` for runs. Check endpoint evidence for capabilities not established by the summary. If testing general access, report the full non-secret permission set without requiring every capability.
- ⚠️ **Degraded:** the call succeeded but lacks one or more permissions needed for the requested task. Name each missing permission and the affected skills.
- ❌ **Failed:** the call failed. Distinguish an authentication or authorization response from a network/TLS failure. Recommend checking the configured hostname and credential by name without printing secrets or raw configuration files.
