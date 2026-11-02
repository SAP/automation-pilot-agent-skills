# `weekly` schedules

Fires on the listed weekday(s) at each hour x minute combination.

**Fields**: `days` (weekday enum
`MONDAY TUESDAY WEDNESDAY THURSDAY FRIDAY SATURDAY SUNDAY`), `hours` (0-23), `minutes`.

## Constraints and gotchas

- `days` are the uppercase enum values, **not** numbers.

## Every Monday at 08:00

```json
{
  "commandId": "welcome-<<<TENANT_ID>>>:RestartApplication:1",
  "description": "Restarts the application",
  "enabled": true,
  "schedule": {
    "timeZone": "Europe/Sofia",
    "weekly": { "days": ["MONDAY"], "hours": [8], "minutes": [0] }
  }
}
```

## Certificate-expiry check every weekday at 07:45

```json
{
  "commandId": "welcome-<<<TENANT_ID>>>:CertificateExpiryCheck:1",
  "description": "Checks certificates for upcoming expiry",
  "enabled": true,
  "schedule": {
    "timeZone": "Europe/Sofia",
    "weekly": {
      "days": ["MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY"],
      "hours": [7],
      "minutes": [45]
    }
  }
}
```
