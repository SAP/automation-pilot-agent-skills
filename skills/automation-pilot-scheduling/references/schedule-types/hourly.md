# `hourly` schedules

Fires at the listed minute(s) of **every** hour. Use it both for "every hour" and for
sub-hourly cadences.

**Fields**: `minutes`.

## Constraints and gotchas

- For a regular sub-hourly interval, the interval must **divide 60**. "Every 7 minutes" or
  "every 25 minutes" cannot be expressed; offer 5, 10, 15, 20, or 30 instead.
- For "every hour," list a **single** minute (e.g. `[0]`), not all twelve boundaries.

## Every 15 minutes

```json
{
  "commandId": "welcome-<<<TENANT_ID>>>:HealthCheck:1",
  "description": "Performs a scheduled health check",
  "enabled": true,
  "schedule": {
    "timeZone": "Etc/UTC",
    "hourly": { "minutes": [0, 15, 30, 45] }
  }
}
```

## Every 30 minutes, offset by five minutes

The two boundaries produce runs at `:05` and `:35` every hour.

```json
{
  "commandId": "welcome-<<<TENANT_ID>>>:RefreshCache:1",
  "description": "Refreshes the application cache",
  "enabled": true,
  "schedule": {
    "timeZone": "Etc/UTC",
    "hourly": { "minutes": [5, 35] }
  }
}
```

## Metrics rollup every hour on the hour

```json
{
  "commandId": "welcome-<<<TENANT_ID>>>:MetricsRollup:1",
  "description": "Aggregates operational metrics",
  "enabled": true,
  "schedule": {
    "timeZone": "Etc/UTC",
    "hourly": { "minutes": [0] }
  }
}
```
