---
name: automation-pilot-scheduled-executions-api
description: Manage SAP Automation Pilot scheduled executions via API. Use when creating, listing, updating, or deleting schedules for recurring or one-time command execution.
---

# SAP Automation Pilot Scheduled Executions API Management

This skill manages scheduled command executions in SAP Automation Pilot - creating, listing, updating, and deleting scheduled executions with flexible scheduling options and time zone support.

## Prerequisites

1. Set the following **required** environment variables:

```bash
export AUTOPI_HOSTNAME="emea.autopilot.cloud.sap"
export AUTOPI_USERNAME="your-username"
export AUTOPI_PASSWORD="your-password"
export AUTOPI_DEFAULT_CATALOG="mycommands-<<<TENANT_ID>>>"
```

For the full list of supported hostnames (emea, aus, apac, amer, ksa), see `../automation-pilot-content-management-via-api/SKILL.md` → Prerequisites.

2. Ensure `curl` is available in your environment.

---

# Schedule Types

SAP Automation Pilot supports the following schedule types:

## Once (One-time)
Execute once at a specific date/time.
```json
"once": { "month": 12, "day": 31, "hour": 23, "minute": 30 }
```

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
```json
"weekly": {
  "days": ["MONDAY", "WEDNESDAY", "FRIDAY"],
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
curl -s -X POST \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "commandId": "my-catalog:MyCommand:1",
    "schedule": {
      "timeZone": "Europe/Berlin",
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
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions"
```

**All `inputValues` entries must be strings on the wire**, regardless of the declared parameter type — see `../automation-pilot-content-management-via-api/SKILL.md` → "Values Are Always Strings on the Wire".

---

# Managing Scheduled Executions

## List All Schedules

```bash
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions"
```

## Get Schedule Details

```bash
SCHEDULE_ID="your-schedule-id"

curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/$SCHEDULE_ID"
```

## Enable/Disable Schedule

```bash
# PUT the full schedule body with "enabled" set to true or false
curl -s -X PUT \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '<full schedule body with enabled changed>' \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/$SCHEDULE_ID"
```

## Delete Schedule

```bash
curl -s -X DELETE \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/$SCHEDULE_ID"
```

---

# Updating Scheduled Executions

```bash
curl -s -X PUT \
  -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "commandId": "my-catalog:MyCommand:1",
    "schedule": {
      "timeZone": "Europe/Berlin",
      "daily": {
        "hours": [10],
        "minutes": [0]
      }
    },
    "enabled": true,
    "inputValues": {}
  }' \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/$SCHEDULE_ID"
```

---

# Time Zones

## List Available Time Zones

```bash
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/scheduled-executions/time-zones"
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
  "tags": {},
  "inputValues": {
    "param1": "value1"
  },
  "inputReferences": [],
  "schedule": {
    "timeZone": "Europe/Berlin",
    "daily": {
      "hours": [9],
      "minutes": [0]
    }
  },
  "enabled": true
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
| `429` | Rate Limited |

## Permissions Required

| Operation | Permission |
|-----------|-----------|
| List/Get schedules | `Read` |
| Create/Update/Delete | `Execute` |

## Common Errors

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

**Cause:** Schedule is disabled or the time zone is misconfigured.
**Solution:** GET the schedule and verify `"enabled": true`. Confirm the `timeZone` matches the intended region — a schedule set for `Europe/Berlin` at 09:00 fires at a different UTC time than one set for `Etc/UTC`.

---

# Examples

## Example 1: Create a daily health check schedule

User: "Schedule my health check command to run every day at 9 AM Berlin time"

1. POST to `/api/v1/scheduled-executions` with `commandId`, `enabled: true`, and `"daily": {"hours": [9], "minutes": [0]}`
2. Set `"timeZone": "Europe/Berlin"`
3. Capture the schedule ID from the response
4. Confirm with a GET that the schedule is enabled and the time is correct

## Example 2: Update a schedule's timing

User: "Change my schedule to run at 10 AM instead of 9 AM"

1. GET `/api/v1/scheduled-executions/$SCHEDULE_ID` to fetch the current body
2. Modify the `hours` value in the response body to `[10]`
3. PUT the full updated body back
4. Confirm the response shows the new time

## Example 3: Create an every-15-minutes monitoring schedule

User: "Run my monitoring command every 15 minutes"

1. POST to `/api/v1/scheduled-executions` with `"hourly": {"minutes": [0, 15, 30, 45]}`
2. Set `"timeZone": "Etc/UTC"`
3. Verify the schedule is created and enabled
