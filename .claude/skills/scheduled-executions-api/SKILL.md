---
name: manage-sap-automation-pilot-scheduled-executions-via-api
description: This skill should be used when the user asks to "schedule execution", "schedule command", "create schedule", "list schedules", "scheduled command", "update schedule", "delete schedule", "time zones", "recurring execution", "hourly schedule", "daily schedule", "weekly schedule", or discusses SAP Automation Pilot scheduled execution management.
version: 1.0.0
---

# SAP Automation Pilot Scheduled Executions API Management

This skill manages scheduled command executions in SAP Automation Pilot - creating, listing, updating, and deleting scheduled executions with flexible scheduling options and time zone support.

## Quick Start — Most Common Commands

```bash
source .env

# List all schedules (enabled and disabled)
source .claude/skills/scheduled-executions-api/scripts/autopi-sched-config.sh && autopi_sched_summary

# Quick summary: how many enabled vs disabled
source .claude/skills/scheduled-executions-api/scripts/autopi-sched-config.sh && autopi_sched_status

# Raw API (endpoint is /scheduled-executions, NOT /schedules)
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions" | jq '.[] | {id, description, enabled, command: .commandId}'
```

**API Endpoint:** `GET /api/v1/scheduled-executions` (not `/schedules`)

## Prerequisites

1. Set the following **required** environment variables:

```bash
export AUTOPI_HOSTNAME="emea.autopilot.cloud.sap"
export AUTOPI_USERNAME="your-username"
export AUTOPI_PASSWORD="your-password"
export AUTOPI_DEFAULT_CATALOG="mycommands-<<<TENANT_ID>>>"
```

**Supported hostnames**:
- `emea.autopilot.cloud.sap` (default - Europe)
- `aus.autopilot.cloud.sap` (Australia)
- `apac.autopilot.cloud.sap` (Asia Pacific)
- `amer.autopilot.cloud.sap` (Americas)
- `ksa.autopilot.cloud.sap` (Saudi Arabia)

2. Ensure `curl` and `jq` are available in your environment.

---

# Executable Scripts

Reusable shell scripts are available in `scripts/` directory.

## Quick Reference

| Script | Description |
|--------|-------------|
| `autopi-schedules.sh` | List, get, delete, enable/disable schedules |
| `autopi-sched-create.sh` | Create new scheduled executions |
| `autopi-sched-update.sh` | Update existing schedules |
| `autopi-sched-timezones.sh` | List supported time zones |

## Usage Examples

```bash
# List all schedules
./scripts/autopi-schedules.sh list

# Get schedule details
./scripts/autopi-schedules.sh get <schedule-id>

# Create a daily schedule at 9:00 AM Berlin time
./scripts/autopi-sched-create.sh "my-catalog:MyCommand:1" \
  --daily --hour 9 --minute 0 \
  --timezone "Europe/Berlin" \
  --input '{"param": "value"}'

# Create an hourly schedule at minutes 0, 15, 30, 45
./scripts/autopi-sched-create.sh "my-catalog:MyCommand:1" \
  --hourly --minutes "0,15,30,45"

# Create a weekly schedule on Mon, Wed, Fri at 8:00
./scripts/autopi-sched-create.sh "my-catalog:MyCommand:1" \
  --weekly --days "1,3,5" --hour 8 --minute 0

# Enable/disable a schedule
./scripts/autopi-schedules.sh enable <schedule-id>
./scripts/autopi-schedules.sh disable <schedule-id>

# Delete a schedule
./scripts/autopi-schedules.sh delete <schedule-id>

# List time zones
./scripts/autopi-sched-timezones.sh
./scripts/autopi-sched-timezones.sh --filter "Europe"
```

---

# Schedule Types

SAP Automation Pilot supports the following schedule types:

## Once (One-time)
Execute once at a specific date/time.

## Hourly
Execute at specified minutes within each hour.
```json
"hourly": {
  "minutes": [0, 15, 30, 45]
}
```

## Daily
Execute at specified time each day.
```json
"daily": {
  "hour": 9,
  "minute": 0
}
```

## Weekly
Execute on specified days of the week at a specific time.
Days: 1=Monday, 2=Tuesday, ..., 7=Sunday
```json
"weekly": {
  "days": [1, 3, 5],
  "hour": 9,
  "minute": 0
}
```

## Monthly
Execute on specified days of each month.
```json
"monthly": {
  "days": [1, 15],
  "hour": 9,
  "minute": 0
}
```

## Yearly
Execute on specific date each year.
```json
"yearly": {
  "month": 1,
  "day": 1,
  "hour": 0,
  "minute": 0
}
```

---

# Creating Scheduled Executions

## Using the Script

```bash
# Daily schedule at 9:00 AM
./scripts/autopi-sched-create.sh "my-catalog:MyCommand:1" \
  --daily --hour 9 --minute 0 \
  --timezone "Europe/Berlin"

# With input parameters
./scripts/autopi-sched-create.sh "my-catalog:MyCommand:1" \
  --daily --hour 9 --minute 0 \
  --timezone "America/New_York" \
  --input '{"region": "us-east", "verbose": true}'

# Create disabled (won't run until enabled)
./scripts/autopi-sched-create.sh "my-catalog:MyCommand:1" \
  --daily --hour 0 --minute 0 \
  --disabled

# Hourly at specific minutes
./scripts/autopi-sched-create.sh "my-catalog:MyCommand:1" \
  --hourly --minutes "0,30" \
  --description "Run every 30 minutes"
```

## Using curl

```bash
# Daily schedule
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "commandId": "my-catalog:MyCommand:1",
    "schedule": {
      "timeZone": "Europe/Berlin",
      "deadlineInMinutes": 5,
      "daily": {
        "hour": 9,
        "minute": 0
      }
    },
    "enabled": true,
    "inputValues": {
      "param1": "value1"
    },
    "description": "Daily morning check"
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions" | jq .

# Hourly schedule
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "commandId": "my-catalog:MyCommand:1",
    "schedule": {
      "timeZone": "Etc/UTC",
      "deadlineInMinutes": 5,
      "hourly": {
        "minutes": [0, 15, 30, 45]
      }
    },
    "enabled": true,
    "inputValues": {}
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions" | jq .
```

---

# Managing Scheduled Executions

## List All Schedules

```bash
# Using script
./scripts/autopi-schedules.sh list

# Using curl
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions" | jq .
```

## Get Schedule Details

```bash
SCHEDULE_ID="your-schedule-id"

# Using script
./scripts/autopi-schedules.sh get $SCHEDULE_ID

# Using curl (captures ETag for updates)
curl -s -i -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/$SCHEDULE_ID"
```

## Enable/Disable Schedule

```bash
# Using script
./scripts/autopi-schedules.sh enable $SCHEDULE_ID
./scripts/autopi-schedules.sh disable $SCHEDULE_ID

# Using update script
./scripts/autopi-sched-update.sh $SCHEDULE_ID --enabled true
./scripts/autopi-sched-update.sh $SCHEDULE_ID --enabled false
```

## Delete Schedule

```bash
# Using script
./scripts/autopi-schedules.sh delete $SCHEDULE_ID

# Using curl (requires ETag)
ETAG=$(curl -s -I -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/$SCHEDULE_ID" | \
  grep -i "etag:" | awk '{print $2}' | tr -d '\r')

curl -s -X DELETE \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "If-Match: $ETAG" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/$SCHEDULE_ID"
```

---

# Updating Scheduled Executions

Updates require an ETag header for optimistic concurrency control. The scripts handle this automatically.

## Using the Script

```bash
# Update to daily schedule
./scripts/autopi-sched-update.sh $SCHEDULE_ID --daily --hour 10 --minute 0

# Update time zone
./scripts/autopi-sched-update.sh $SCHEDULE_ID --timezone "America/Los_Angeles"

# Update input
./scripts/autopi-sched-update.sh $SCHEDULE_ID --input '{"newParam": "newValue"}'

# Enable/disable
./scripts/autopi-sched-update.sh $SCHEDULE_ID --enabled true
```

## Using curl

```bash
# First get current schedule and ETag
RESPONSE=$(curl -s -i -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/$SCHEDULE_ID")

ETAG=$(echo "$RESPONSE" | grep -i "etag:" | awk '{print $2}' | tr -d '\r')
BODY=$(echo "$RESPONSE" | tail -1)

# Update with If-Match header
curl -s -X PUT \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -H "If-Match: $ETAG" \
  -d '{
    "commandId": "my-catalog:MyCommand:1",
    "schedule": {
      "timeZone": "Europe/Berlin",
      "deadlineInMinutes": 5,
      "daily": {
        "hour": 10,
        "minute": 0
      }
    },
    "enabled": true,
    "inputValues": {}
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/$SCHEDULE_ID" | jq .
```

---

# Time Zones

## List Available Time Zones

```bash
# All time zones
./scripts/autopi-sched-timezones.sh

# Filter by region
./scripts/autopi-sched-timezones.sh --filter "Europe"
./scripts/autopi-sched-timezones.sh --filter "America"
./scripts/autopi-sched-timezones.sh --filter "Asia"

# Using curl
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/time-zones" | jq .
```

## Common Time Zones

| Time Zone | Description |
|-----------|-------------|
| `Etc/UTC` | Coordinated Universal Time (default) |
| `Europe/Berlin` | Central European Time (CET/CEST) |
| `Europe/London` | British Time (GMT/BST) |
| `America/New_York` | Eastern Time (EST/EDT) |
| `America/Los_Angeles` | Pacific Time (PST/PDT) |
| `Asia/Tokyo` | Japan Standard Time |
| `Asia/Singapore` | Singapore Time |
| `Australia/Sydney` | Australian Eastern Time |

---

# Scheduled Execution Schema

## Request Body (Create/Update)

```json
{
  "commandId": "catalog:CommandName:version",
  "description": "Optional description",
  "correlationId": "optional-correlation-id",
  "tags": {
    "feature:logs": "true"
  },
  "inputValues": {
    "param1": "value1"
  },
  "inputReferences": [],
  "schedule": {
    "timeZone": "Europe/Berlin",
    "deadlineInMinutes": 5,
    "daily": {
      "hour": 9,
      "minute": 0
    }
  },
  "enabled": true
}
```

## Response

```json
{
  "id": "T000153R2-0000001234567890-0-1",
  "commandId": "catalog:CommandName:version",
  "description": "Daily health check",
  "correlationId": null,
  "owner": null,
  "tags": {},
  "inputValues": {},
  "inputReferences": [],
  "schedule": {
    "timeZone": "Europe/Berlin",
    "deadlineInMinutes": 5,
    "once": null,
    "hourly": null,
    "daily": {
      "hour": 9,
      "minute": 0
    },
    "weekly": null,
    "monthly": null,
    "yearly": null
  },
  "enabled": true,
  "state": {
    "executionId": "...",
    "triggeredAt": 1708250400000,
    "status": "SUCCESS",
    "message": ""
  }
}
```

---

# Error Handling

## HTTP Status Codes

| Code | Description |
|------|-------------|
| `200` | Success (GET, PUT) |
| `201` | Created (POST) |
| `204` | No Content (DELETE) |
| `400` | Bad Request - invalid schedule or input |
| `401` | Unauthorized - invalid credentials |
| `403` | Forbidden - insufficient permissions |
| `404` | Not Found - schedule doesn't exist |
| `409` | Conflict - invalid state |
| `412` | Precondition Failed - ETag mismatch |
| `429` | Rate Limited |

## Permissions Required

| Operation | Permission |
|-----------|-----------|
| List/Get schedules | `Read` |
| Create/Update/Delete | `Execute` |

## Common Errors

### ETag Mismatch (412)

```
Error: Precondition Failed - ETag mismatch
```
The schedule was modified by another user. Fetch the latest version and retry.

### Command Not Found (409)

```
Error: Command "catalog:Command:1" not found
```
Verify the command ID exists and you have access to it.

### Invalid Schedule (400)

```
Error: Invalid schedule configuration
```
Ensure only one schedule type is set (hourly, daily, weekly, monthly, or yearly).
