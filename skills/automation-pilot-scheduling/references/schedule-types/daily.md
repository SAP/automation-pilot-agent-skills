# `daily` schedules

Fires every day at each hour x minute combination listed.

**Fields**: `hours` (0-23), `minutes`.

## Constraints and gotchas

- The cartesian product bites here: `hours: [9, 17], minutes: [0, 30]` is four runs
  (09:00, 09:30, 17:00, 17:30), not two. For distinct times that are not a clean product,
  confirm with the user.

## Platform health snapshot every day at 06:00

```json
{
  "commandId": "welcome-<<<TENANT_ID>>>:PlatformHealthSnapshot:1",
  "description": "Captures a platform health snapshot",
  "enabled": true,
  "schedule": {
    "timeZone": "Europe/Sofia",
    "daily": { "hours": [6], "minutes": [0] }
  }
}
```

## Capacity health check twice daily at 09:00 and 17:00

One minute value combined with two hour values produces exactly two runs.

```json
{
  "commandId": "welcome-<<<TENANT_ID>>>:CapacityHealthCheck:1",
  "description": "Checks platform capacity health",
  "enabled": true,
  "schedule": {
    "timeZone": "Europe/Sofia",
    "daily": { "hours": [9, 17], "minutes": [0] }
  }
}
```

## Service health checks every 30 minutes from 09:00 through 10:30

The cartesian product of two hours and two minute boundaries produces four runs: 09:00,
09:30, 10:00, and 10:30.

```json
{
  "commandId": "welcome-<<<TENANT_ID>>>:ServiceHealthCheck:1",
  "description": "Checks service health",
  "enabled": true,
  "schedule": {
    "timeZone": "Europe/London",
    "daily": { "hours": [9, 10], "minutes": [0, 30] }
  }
}
```
