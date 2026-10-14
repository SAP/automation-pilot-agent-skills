# Examples & troubleshooting

## Example 1: Deploy a command from a JSON file

User: "Deploy this command to Automation Pilot"

```bash
autopi commands deploy -f MyCommand.command.json
```

`deploy` reads the command's ID from the file, checks existence (GET), then PUT-updates
if it exists or POST-creates if not — all in one step. Report success with the command ID
from the response. The command stays in DRAFT; do not release unless asked.

## Example 2: Export all commands from a catalog

User: "Export all commands from my catalog to files"

```bash
for id in $(autopi commands ids --catalog "$AUTOPI_DEFAULT_CATALOG" --output raw | jq -r '.[]'); do
  name="${id##*:}"                          # or parse catalog:name:version as needed
  autopi commands export "$id" -o "${name}.command.json"
done
```

Report how many files were written and where.

## Example 3: Rotate credentials in an input

User: "Update the password in my JiraCredentials input"

```bash
autopi inputs patch mycatalog-xxx:JiraCredentials:1 --data '{"values": {"password": "<new-value>"}}'
```

Uses merge-patch — only the `password` value changes; the rest of the input is untouched.
Confirm the update succeeded (exit 0 / HTTP 200).

---

## Troubleshooting

**Exit 4, HTTP 401 Unauthorized** — Invalid credentials. Verify `AUTOPI_USERNAME` and
`AUTOPI_PASSWORD` are correct for the target `AUTOPI_HOSTNAME`.

**Exit 4, HTTP 403 Forbidden** — The user lacks the required permission. Create/update/
delete all need `Write`.

**Exit 4, HTTP 409 Conflict on create** — A resource with the same ID already exists. Use
`autopi commands deploy` / `autopi inputs deploy` (upsert) instead of `create`.

**Exit 4, HTTP 400 Bad Request on create** — Invalid JSON or a missing required field.
`autopi` prints the response body naming the field that failed. Run `autopi lint
<file>` first to catch most of these offline.

**Exit 4, HTTP 404 on update** — The command/input ID in the URL doesn't match anything.
IDs are `catalog:name:version`. Confirm with `autopi commands get <id> --status-only`.

See [`../../automation-pilot-cli/references/exit-codes.md`](../../automation-pilot-cli/references/exit-codes.md)
for the full exit-code and HTTP-status reference.
