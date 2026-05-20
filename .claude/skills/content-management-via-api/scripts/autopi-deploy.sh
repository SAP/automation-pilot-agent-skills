#!/usr/bin/env bash
# SAP Automation Pilot Content API - Smart Deploy
# Deploys a command file with validation and upsert logic
#
# Usage: autopi-deploy.sh <file.command.json>

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-config.sh"

FILE="$1"

if [[ -z "$FILE" ]]; then
  echo "Usage: autopi-deploy.sh <file.command.json>"
  echo ""
  echo "Smart deploy a command to SAP Automation Pilot."
  echo "Creates the command if it doesn't exist, updates if it does."
  exit 1
fi

if [[ ! -f "$FILE" ]]; then
  autopi_error "File not found: $FILE"
  exit 1
fi

# Validate JSON
if ! jq empty "$FILE" 2>/dev/null; then
  autopi_error "Invalid JSON in: $FILE"
  exit 1
fi

# Extract tenant ID from default catalog (format: name-TENANTID)
TENANT_ID="${AUTOPI_CATALOG##*-}"

# Replace <<<TENANT_ID>>> placeholder if present and get command details
COMMAND_JSON=$(sed "s/<<<TENANT_ID>>>/$TENANT_ID/g" "$FILE")
COMMAND_ID=$(echo "$COMMAND_JSON" | jq -r '.id')
COMMAND_NAME=$(echo "$COMMAND_JSON" | jq -r '.name')
CATALOG=$(echo "$COMMAND_JSON" | jq -r '.catalog')

autopi_info "Deploying: $COMMAND_NAME"
autopi_info "  ID: $COMMAND_ID"
autopi_info "  Catalog: $CATALOG"

# Check if command exists
STATUS=$(autopi_check "/commands/$COMMAND_ID")

if [[ "$STATUS" == "200" ]]; then
  autopi_info "Command exists, updating..."
  RESPONSE=$(echo "$COMMAND_JSON" | curl -s -X PUT \
    -u "$AUTOPI_USER:$AUTOPI_PASS" \
    -H "Content-Type: application/json" \
    -d @- \
    "$AUTOPI_BASE_URL/commands/$COMMAND_ID")
else
  autopi_info "Creating new command..."
  RESPONSE=$(echo "$COMMAND_JSON" | curl -s -X POST \
    -u "$AUTOPI_USER:$AUTOPI_PASS" \
    -H "Content-Type: application/json" \
    -d @- \
    "$AUTOPI_BASE_URL/commands")
fi

# Check for errors
if echo "$RESPONSE" | jq -e '.code' >/dev/null 2>&1; then
  autopi_error "API Error: $(echo "$RESPONSE" | jq -r '.message')"
  exit 1
fi

# Show result
RESULT_ID=$(echo "$RESPONSE" | jq -r '.id')
ISSUES=$(echo "$RESPONSE" | jq '[.issues[]? | {severity, name}]')
ISSUE_COUNT=$(echo "$ISSUES" | jq 'length')
ERROR_COUNT=$(echo "$ISSUES" | jq '[.[] | select(.severity == "ERROR")] | length')

autopi_ok "Deployed: $RESULT_ID"

if [[ "$ISSUE_COUNT" -gt 0 ]]; then
  if [[ "$ERROR_COUNT" -gt 0 ]]; then
    autopi_error "Issues found ($ERROR_COUNT errors):"
  else
    autopi_warn "Issues found:"
  fi
  echo "$ISSUES" | jq -r '.[] | "  - [\(.severity)] \(.name)"'
fi
