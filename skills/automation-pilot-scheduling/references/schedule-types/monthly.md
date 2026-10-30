# `monthly` schedules

Fires on the listed day(s) of **every** month.

**Fields**: `days` (1-31), `hours` (0-23), `minutes`.

## Constraints and gotchas

- No month selector. It runs every month and cannot be restricted to selected months.
- `days: [31]` only fires in months that have a 31st; shorter months skip it without error.
  There is **no "last day of the month"** primitive. Safe choices present in every
  month are `1` (first) or `28`.
- "Twice a month" uses two days, e.g. `days: [1, 15]`. It does not mean "every two
  months."

## Patch-compliance scan on the first of every month at 02:00

```json
{
  "commandId": "welcome-<<<TENANT_ID>>>:PatchComplianceScan:1",
  "description": "Checks systems for patch compliance",
  "enabled": true,
  "schedule": {
    "timeZone": "Europe/Sofia",
    "monthly": { "days": [1], "hours": [2], "minutes": [0] }
  }
}
```

## Configuration-drift scan on the first and fifteenth at 02:00

```json
{
  "commandId": "welcome-<<<TENANT_ID>>>:ConfigurationDriftScan:1",
  "description": "Checks systems for configuration drift",
  "enabled": true,
  "schedule": {
    "timeZone": "Europe/Sofia",
    "monthly": { "days": [1, 15], "hours": [2], "minutes": [0] }
  }
}
```

