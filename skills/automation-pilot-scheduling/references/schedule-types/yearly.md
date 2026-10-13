# `yearly` schedules

Fires in the listed month(s), on the listed day(s), at the listed time. This is how you
express every multi-month cadence (every two months, quarterly, semi-annual, annual).

**Fields**: `months` (1-12), `days` (1-31), `hours` (0-23), `minutes`.

## Constraints and gotchas

- List the exact months for the cadence you want: quarterly = `[1, 4, 7, 10]`,
  every two months = `[1, 3, 5, 7, 9, 11]`, semi-annual = `[1, 7]`, annual = a single month.
- A `day` absent from a listed month is skipped for that month. For example, `days: [31]`
  triggers only in listed months that have a 31st. There is no "last day of the month"
  primitive; choose a day present in every selected month when every occurrence must run.
- Anchor quarters to the user's fiscal calendar if they have one; the examples assume
  calendar quarters.

## Quarterly disaster-recovery exercise at 03:00

```json
{
  "commandId": "welcome-<<<TENANT_ID>>>:DisasterRecoveryExercise:1",
  "description": "Exercises disaster recovery procedures",
  "enabled": true,
  "schedule": {
    "timeZone": "Europe/Sofia",
    "yearly": {
      "months": [1, 4, 7, 10],
      "days": [1],
      "hours": [3],
      "minutes": [0]
    }
  }
}
```

## Annual credential rotation on January 1 at 00:00

```json
{
  "commandId": "welcome-<<<TENANT_ID>>>:CredentialRotation:1",
  "description": "Rotates credentials",
  "enabled": true,
  "schedule": {
    "timeZone": "Europe/Sofia",
    "yearly": { "months": [1], "days": [1], "hours": [0], "minutes": [0] }
  }
}
```
