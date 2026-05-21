#!/usr/bin/env bash
# SAP Automation Pilot Content API - Smart MCP Server Deploy
# Deploys an MCP server file with validation and upsert logic
#
# Usage: autopi-deploy-mcp-server.sh <file.json>

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-config.sh"

FILE="$1"

if [[ -z "$FILE" ]]; then
  echo "Usage: autopi-deploy-mcp-server.sh <file.json>"
  echo ""
  echo "Smart deploy an MCP server to SAP Automation Pilot."
  echo "Creates the MCP server if it doesn't exist, updates if it does."
  echo "Handles ETag automatically for updates."
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

# Extract MCP server details
MCP_SERVER_NAME=$(jq -r '.name' "$FILE")
MCP_ENABLED=$(jq -r '.enabled' "$FILE")
MCP_TOOL_COUNT=$(jq '.mcpTools | length' "$FILE")
MCP_ENABLED_TOOLS=$(jq '[.mcpTools[] | select(.enabled == true)] | length' "$FILE")

if [[ "$MCP_SERVER_NAME" == "null" || -z "$MCP_SERVER_NAME" ]]; then
  autopi_error "MCP server file must have a 'name' field"
  exit 1
fi

autopi_info "Deploying MCP server: $MCP_SERVER_NAME"
autopi_info "  Tools: $MCP_TOOL_COUNT ($MCP_ENABLED_TOOLS enabled)"
autopi_info "  Enabled: $MCP_ENABLED"

# URL-encode the name for API path
url_encode() {
  local string="$1"
  python3 -c "import urllib.parse; print(urllib.parse.quote('''$string''', safe=''))"
}

ENCODED_NAME=$(url_encode "$MCP_SERVER_NAME")

# Check if MCP server exists
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -u "$AUTOPI_USER:$AUTOPI_PASS" \
  "$AUTOPI_BASE_URL/mcp-servers/$ENCODED_NAME")

if [[ "$STATUS" == "200" ]]; then
  autopi_info "MCP server exists, updating..."

  # Get current ETag
  ETAG=$(curl -s -I -u "$AUTOPI_USER:$AUTOPI_PASS" \
    "$AUTOPI_BASE_URL/mcp-servers/$ENCODED_NAME" | \
    grep -i "^etag:" | awk '{print $2}' | tr -d '\r\n')

  if [[ -z "$ETAG" ]]; then
    autopi_error "Could not get ETag for update"
    exit 1
  fi

  RESPONSE=$(curl -s -X PUT \
    -u "$AUTOPI_USER:$AUTOPI_PASS" \
    -H "Content-Type: application/json" \
    -H "If-Match: $ETAG" \
    -d @"$FILE" \
    "$AUTOPI_BASE_URL/mcp-servers/$ENCODED_NAME")
else
  autopi_info "Creating new MCP server..."

  RESPONSE=$(curl -s -X POST \
    -u "$AUTOPI_USER:$AUTOPI_PASS" \
    -H "Content-Type: application/json" \
    -d @"$FILE" \
    "$AUTOPI_BASE_URL/mcp-servers")
fi

# Check for errors
if echo "$RESPONSE" | jq -e '.code' >/dev/null 2>&1; then
  autopi_error "API Error: $(echo "$RESPONSE" | jq -r '.message')"
  exit 1
fi

# Show result
RESULT_NAME=$(echo "$RESPONSE" | jq -r '.name')
RESULT_TOOLS=$(echo "$RESPONSE" | jq '.mcpTools | length')

autopi_ok "Deployed: $RESULT_NAME ($RESULT_TOOLS tools)"
