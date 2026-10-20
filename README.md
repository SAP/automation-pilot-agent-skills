[![REUSE status](https://api.reuse.software/badge/github.com/SAP/automation-pilot-agent-skills)](https://api.reuse.software/info/github.com/SAP/automation-pilot-agent-skills)

# <img src="sap-automation-pilot.svg" width="32" height="32" alt="SAP Automation Pilot" valign="middle"> SAP Automation Pilot Agent Skills

### Build and operate SAP BTP automations from your coding agent.

A plugin for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). It gives your agent **10 skills** that encode SAP [Automation Pilot](https://help.sap.com/docs/automation-pilot) patterns and uses MCP to connect to your tenant. You describe the automation you want; the agent builds, deploys, runs, and monitors it without leaving your editor.

Review and test generated content against the target tenant before you rely on it.

## Table of contents

- [What you get](#what-you-get)
- [Skills](#skills)
- [Installation](#installation)
- [Updating](#updating)
- [Verify the connection](#verify-the-connection)
- [How to use](#how-to-use)
- [About SAP Automation Pilot](#about-sap-automation-pilot)
- [Resources](#resources)
- [Support and contributing](#support-and-contributing)
- [License](#license)

## What you get

Full control over your Automation Pilot resources from your favorite coding agent — **build new automations, debug and extend existing ones, run and troubleshoot executions, and manage schedules**, all from plain-language requests.

- **Build:** Author Commands and multi-step automations — HTTP calls, scripts, polling loops, sequential and parallel batch operations — and deploy them straight to your tenant.
- **Debug and extend:** Pull an existing Command down, understand it, fix a broken dynamic expression, add a step, or harden its error handling, then push it back.
- **Manage Inputs:** Create reusable configuration and credential sets, rotate values safely, and update their definitions.
- **Operate:** Trigger Commands, follow Executions live, read logs and outputs, and troubleshoot failures without leaving your editor.
- **Schedule:** Create, inspect, change, enable, disable, and delete scheduled Executions.
- **Embedded expertise:** The skills carry Command structure, expression syntax, security requirements, and error-handling patterns. You describe *what* to automate; the agent handles the *how*.
- **Built-in checks:** The skills verify contracts, retries, sensitivity flags, and read-back before anything touches your tenant.

## Skills

| Skill | What it does |
|-------|--------------|
| [**Planning**](skills/automation-pilot-plan/SKILL.md) | Explicitly invoked interview that produces an implementation-ready automation spec |
| [**Command Authoring**](skills/automation-pilot-command-authoring/SKILL.md) | Author Commands and composite automations (HTTP, scripts, polling, batching) and deploy them |
| [**Executions**](skills/automation-pilot-executions/SKILL.md) | Trigger Commands, monitor and troubleshoot Executions, and browse execution history |
| [**Dynamic Expressions**](skills/automation-pilot-dynamic-expressions/SKILL.md) | Craft and debug Automation Pilot dynamic expressions against representative data |
| [**Discovery**](skills/automation-pilot-discovery/SKILL.md) | Find and inspect catalogs, Commands, and reusable Inputs; judge reuse fit before building |
| [**Inputs**](skills/automation-pilot-inputs/SKILL.md) | Create and maintain reusable configuration and credential Inputs without exposing stored values |
| [**Scheduling**](skills/automation-pilot-scheduling/SKILL.md) | Create, inspect, change, enable, disable, and delete scheduled Executions |
| [**MCP Server Management**](skills/automation-pilot-mcp-servers/SKILL.md) | Create and maintain Automation Pilot MCP server definitions that expose Commands as tools |
| [**Tags**](skills/automation-pilot-tags/SKILL.md) | Apply and interpret regular, feature, and service tags, including inheritance and execution lineage |
| [**Connection**](skills/automation-pilot-connection/SKILL.md) | Validate the configured connection and summarize tenant, identity, permissions, and service plan |

## Installation

**Prerequisite:** Install [`uv`](https://docs.astral.sh/uv/getting-started/installation/). The plugin needs it at runtime.

Install the plugin from inside Claude Code using slash commands.

**1. Start Claude Code.** Open your terminal and run:

```bash
claude
```

**2. Add the plugin marketplace.** Inside Claude Code, run:

```text
/plugin marketplace add SAP/automation-pilot-agent-skills#development
```

**3. Install the plugin.** Still inside Claude Code, run:

```text
/plugin install automation-pilot-skills@automation-pilot
```

**4. Configure the connection.** On install, Claude Code prompts for the `hostname` and `token` settings. Enter your tenant values, for example:

| Setting | Example value |
|---------|---------------|
| `hostname` | `emea.autopilot.cloud.sap` (tenant host, no `https://`) |
| `token` | `ap_T000000R4-claude-1_EHOA...` (service account bearer token) |

## Updating

To update the plugin manually, run this command in your terminal:

```bash
claude plugin update automation-pilot-skills@automation-pilot
```

Claude Code updates the plugin from the `development` branch configured when you added the
marketplace.

To enable automatic updates, run `/plugin` in a Claude Code session and open the
**Marketplaces** tab. Select the marketplace, then choose **Enable auto-update**. You can
turn automatic updates off from the same menu by choosing **Disable auto-update**.

## Verify the connection

After installing or updating, call the connection skill to verify that the plugin and configured connection work:

```text
/automation-pilot-skills:automation-pilot-connection
```

## How to use

Describe your Automation Pilot task in plain language and the matching skill engages. Some examples:

**Generate a Command**

```text
Create an Automation Pilot Command that restarts a Cloud Foundry app
and waits for it to become healthy, with proper error handling.
```

**Craft a dynamic expression**

```text
Write an Automation Pilot expression that filters this execution data to
enabled resources and returns their IDs.
```

**Deploy, trigger, and monitor**

```text
Upload MyCommand.command.json to my tenant, trigger it with Input
{"appName": "my-app"}, and show me the results when it finishes.
```

**Troubleshoot a failure**

```text
Execution abc123 failed — show me the logs and help me understand what went wrong.
```

**Manage schedules**

```text
List my enabled scheduled executions and disable the active ones.
```

**Plan an automation.** The planning skill never auto-invokes. Call it by name:

```text
/automation-pilot-skills:automation-pilot-plan
```

## About SAP Automation Pilot

[SAP Automation Pilot](https://help.sap.com/docs/automation-pilot) is SAP's native engine for automating cloud operations on Business Technology Platform. It turns the manual, repetitive, and error-prone tasks of running SAP landscapes — provisioning, scaling, patching, health checks, backups, incident response, housekeeping — into reusable automations you can trigger on demand, run on a schedule, or wire into your own tools and agents.

**The building blocks:**

- **Commands** define automations. A Command can be a single action or a composite that chains many steps together.
- **Inputs** store the credentials, endpoints, and configuration your Commands consume, with sensitive values kept out of logs.
- **Executions** are running instances of a Command — you trigger them, watch them progress, and inspect their logs, inputs, and outputs.
- **Schedules** run Commands automatically on a recurring cadence.

**Built for complex, agentic automation.** Automation Pilot is more than a script runner. Commands compose into real workflows: run a sub-command once per resource, **fan out across many resources in parallel**, poll an asynchronous operation until it completes, retry transient failures with backoff, branch on conditions, validate postconditions, and surface meaningful errors. A dynamic-expression language lets each step shape and route data to the next, so an automation can make decisions mid-flight instead of running a fixed path. The result is production-grade orchestration — the kind of durable, long-running, parallel work that a coding agent can design but shouldn't babysit by hand.

**A broad, ready-made catalog.** Automation Pilot ships with hundreds of built-in Commands across SAP HANA Cloud, BTP management, Cloud Foundry, Kubernetes/Kyma, Neo, Service Manager, SAP Cloud ALM, AI Core, monitoring (Dynatrace, ANS), DevOps (GitHub, Jenkins, JIRA), notifications, security, databases, transport (cTMS), generic HTTP/REST, and scripting (Bash, Python, Node.js, PowerShell, Terraform). You compose these into your own automations rather than building everything from scratch.

**Exposed as tools.** Automation Pilot can publish your Commands as MCP servers, turning any automation into a tool other agents can call — so the automations you build here become capabilities your wider agent fleet can use.

This plugin puts all of it in your coding agent: the catalog to reuse, the composition patterns to build on, and live access to run and operate it.

## Resources

- [SAP Automation Pilot documentation](https://help.sap.com/docs/automation-pilot): guides, API reference, expression language
- [SAP Automation Pilot examples](https://github.com/SAP-samples/automation-pilot-examples): ready-to-import examples for HANA Cloud, BTP, Cloud Foundry, Kubernetes, and more
- [Claude Code plugins](https://code.claude.com/docs/en/plugins)

## Support and contributing

Feature requests, bug reports, and contributions are welcome via [GitHub Issues](https://github.com/SAP/automation-pilot-agent-skills/issues). See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

**Security:** if you discover a vulnerability, follow our [security policy](https://github.com/SAP/automation-pilot-agent-skills/security/policy). Do not open GitHub issues for security concerns.

**Code of conduct:** by participating in this project you agree to abide by its [Code of Conduct](https://github.com/SAP/.github/blob/main/CODE_OF_CONDUCT.md).

## License

Copyright 2026 SAP SE or an SAP affiliate company and automation-pilot-agent-skills contributors. See [LICENSE](LICENSE) for details. Third-party component information is available via the [REUSE tool](https://api.reuse.software/info/github.com/SAP/automation-pilot-agent-skills).
