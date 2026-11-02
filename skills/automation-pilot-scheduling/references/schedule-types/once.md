# `once` schedules

Fires one time at the next occurrence of the given month/day/time, then becomes
`COMPLETED` and never runs again.

**Fields** (all **singular integers**, all required): `month` (1-12), `day` (1-31),
`hour` (0-23), `minute`.

## Constraints and gotchas

- No year field. It resolves to the **next** future occurrence of that month/day/time.
- A `day` that does not exist in the chosen `month` (e.g. `month: 2, day: 30`) will not
  fire; pick a valid date.

## One-time platform maintenance on December 31 at 23:30

```json
{
  "commandId": "welcome-<<<TENANT_ID>>>:PlatformMaintenance:1",
  "description": "Performs planned platform maintenance",
  "enabled": true,
  "schedule": {
    "timeZone": "Europe/Sofia",
    "once": { "month": 12, "day": 31, "hour": 23, "minute": 30 }
  }
}
```

## Disabled one-time cutover draft

Use `enabled: false` to save the schedule without allowing it to trigger until it is
reviewed and enabled.

```json
{
  "commandId": "welcome-<<<TENANT_ID>>>:ProductionCutover:1",
  "description": "Draft production cutover",
  "enabled": false,
  "schedule": {
    "timeZone": "America/New_York",
    "once": { "month": 4, "day": 10, "hour": 1, "minute": 15 }
  }
}
```
