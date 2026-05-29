---
name: automation-pilot-scheduled-executions-api
description: Manage SAP Automation Pilot scheduled executions via API. Use when creating, listing, updating, or deleting schedules for recurring or one-time command execution.
version: 1.0.0
---

# SAP Automation Pilot Scheduled Executions API Management

This skill manages scheduled command executions in SAP Automation Pilot - creating, listing, updating, and deleting scheduled executions with flexible scheduling options and time zone support.

## Quick Start — Most Common Commands

```bash
# List all schedules
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions" | \
  jq '.[] | {id, description, enabled, command: .commandId}'
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
  "hours": [9],
  "minutes": [0]
}
```

## Weekly
Execute on specified days of the week at a specific time.
Days: 1=Monday, 2=Tuesday, ..., 7=Sunday
```json
"weekly": {
  "days": [1, 3, 5],
  "hours": [9],
  "minutes": [0]
}
```

## Monthly
Execute on specified days of each month.
```json
"monthly": {
  "days": [1, 15],
  "hours": [9],
  "minutes": [0]
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

```bash
# Daily schedule at 9:00 AM Berlin time
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "commandId": "my-catalog:MyCommand:1",
    "schedule": {
      "timeZone": "Europe/Berlin",
      "deadlineInMinutes": 5,
      "daily": {
        "hours": [9],
        "minutes": [0]
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

# Weekly schedule (Mon, Wed, Fri at 9:00 AM)
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "commandId": "my-catalog:MyCommand:1",
    "schedule": {
      "timeZone": "Europe/Berlin",
      "deadlineInMinutes": 5,
      "weekly": {
        "days": [1, 3, 5],
        "hours": [9],
        "minutes": [0]
      }
    },
    "enabled": true,
    "inputValues": {}
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions" | jq .

# Monthly schedule (1st and 15th at 9:00 AM)
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "commandId": "my-catalog:MyCommand:1",
    "schedule": {
      "timeZone": "Europe/Berlin",
      "deadlineInMinutes": 5,
      "monthly": {
        "days": [1, 15],
        "hours": [9],
        "minutes": [0]
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
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions" | jq .
```

## Get Schedule Details

```bash
SCHEDULE_ID="your-schedule-id"

# Fetch details and capture ETag for updates
curl -s -i -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/$SCHEDULE_ID"
```

## Enable/Disable Schedule

```bash
# Get current schedule body and ETag, then PUT with enabled flag changed
RESPONSE=$(curl -s -i -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/$SCHEDULE_ID")
ETAG=$(echo "$RESPONSE" | grep -i "etag:" | awk '{print $2}' | tr -d '\r')
BODY=$(echo "$RESPONSE" | tail -1)

curl -s -X PUT \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -H "If-Match: $ETAG" \
  -d "$(echo "$BODY" | jq '.enabled = true')" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/$SCHEDULE_ID" | jq .
```

## Delete Schedule

```bash
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

Updates require an ETag header for optimistic concurrency control.

```bash
# Get current schedule and ETag
RESPONSE=$(curl -s -i -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/$SCHEDULE_ID")

ETAG=$(echo "$RESPONSE" | grep -i "etag:" | awk '{print $2}' | tr -d '\r')

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
        "hours": [10],
        "minutes": [0]
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
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/time-zones" | jq .

# Filter by region
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/time-zones" | \
  jq '.[] | select(. | test("Europe"))'
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
      "hours": [9],
      "minutes": [0]
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
      "hours": [9],
      "minutes": [0]
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

### Schedule Not Firing

**Cause:** Schedule is disabled, the time zone is misconfigured, or the `deadlineInMinutes` window is too short for the command to complete.
**Solution:** GET the schedule and verify `"enabled": true`. Confirm the `timeZone` matches the intended region — a schedule set for `Europe/Berlin` at 09:00 fires at a different UTC time than one set for `Etc/UTC`. If the command regularly exceeds `deadlineInMinutes`, increase it.

---

# Examples

## Example 1: Create a daily health check schedule

User: "Schedule my health check command to run every day at 9 AM Berlin time"

1. POST to `/api/v1/scheduled-executions` with `commandId`, `enabled: true`, and `"daily": {"hours": [9], "minutes": [0]}`
2. Set `"timeZone": "Europe/Berlin"` and a `deadlineInMinutes` appropriate for the command's expected duration
3. Capture the schedule ID from the response
4. Confirm with a GET that the schedule is enabled and the time is correct

## Example 2: Update a schedule's timing

User: "Change my schedule to run at 10 AM instead of 9 AM"

1. GET `/api/v1/scheduled-executions/$SCHEDULE_ID` with `-i` to capture the ETag header
2. Modify the `hours` value in the response body to `[10]`
3. PUT the full updated body back with `If-Match: $ETAG`
4. Confirm the response shows the new time

## Example 3: Create an every-15-minutes monitoring schedule

User: "Run my monitoring command every 15 minutes"

1. POST to `/api/v1/scheduled-executions` with `"hourly": {"minutes": [0, 15, 30, 45]}`
2. Set `"timeZone": "Etc/UTC"` and a short `deadlineInMinutes` (e.g., 5)
3. Verify the schedule is created and enabled
