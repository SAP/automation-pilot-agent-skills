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

Call `automation-pilot:get_connection_info` and report its non-secret summary.

## Verdict

Open the response with one verdict on its own line, each prefixed with its emoji, then the supporting summary:

- ✅ **Healthy:** the call succeeded and `permissions` contains `Read`, `Write`, and `Execute`.
- ⚠️ **Degraded:** the call succeeded but lacks one or more required permissions. Name each missing permission and the affected skills.
- ❌ **Failed:** the call failed. Distinguish an authentication or authorization response from a network/TLS failure. Recommend checking the configured hostname and credential by name without printing secrets or raw configuration files.
