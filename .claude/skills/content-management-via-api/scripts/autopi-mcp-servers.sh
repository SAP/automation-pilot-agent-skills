#!/usr/bin/env bash
# SAP Automation Pilot Content API - MCP Server Operations
#
# Usage:
#   autopi-mcp-servers.sh list
#   autopi-mcp-servers.sh get <mcp-server-id>
#   autopi-mcp-servers.sh create <file.json>
#   autopi-mcp-servers.sh update <mcp-server-id> <file.json>
#   autopi-mcp-servers.sh delete <mcp-server-id>

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-config.sh"

show_help() {
  cat << EOF
Usage: autopi-mcp-servers.sh <command> [options]

Commands:
  list                            List all MCP servers
  get <mcp-server-id>             Get MCP server details
  create <file.json>              Create a new MCP server from file
  update <mcp-server-id> <file>   Update MCP server (auto-fetches ETag)
  delete <mcp-server-id>          Delete MCP server (auto-fetches ETag)

Note: The MCP server ID is its name (e.g., "BTP Resource Discovery").

Examples:
  autopi-mcp-servers.sh list
  autopi-mcp-servers.sh get "BTP Resource Discovery"
  autopi-mcp-servers.sh create my-server.json
  autopi-mcp-servers.sh update "BTP Resource Discovery" updated-server.json
  autopi-mcp-servers.sh delete "My Old Server"
EOF
}

# URL-encode a string for use in API paths
url_encode() {
  local string="$1"
  python3 -c "import urllib.parse; print(urllib.parse.quote('''$string''', safe=''))"
}

# Get ETag for an MCP server
get_etag() {
  local id="$1"
  local encoded_id=$(url_encode "$id")
  curl -s -I -u "$AUTOPI_USER:$AUTOPI_PASS" \
    "$AUTOPI_BASE_URL/mcp-servers/$encoded_id" | \
    grep -i "^etag:" | awk '{print $2}' | tr -d '\r\n'
}

cmd_list() {
  autopi_info "Listing MCP servers"
  autopi_api GET "/mcp-servers" | jq '[.[] | {name, enabled, toolCount: (.mcpTools | length)}]'
}

cmd_get() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "MCP server ID (name) required"; exit 1; }

  local encoded_id=$(url_encode "$id")
  autopi_api GET "/mcp-servers/$encoded_id" | jq .
}

cmd_create() {
  local file="$1"
  [[ -z "$file" ]] && { autopi_error "MCP server file required"; exit 1; }
  [[ ! -f "$file" ]] && { autopi_error "File not found: $file"; exit 1; }

  # Validate JSON
  if ! jq empty "$file" 2>/dev/null; then
    autopi_error "Invalid JSON in: $file"
    exit 1
  fi

  local name=$(jq -r '.name' "$file")
  local tool_count=$(jq '.mcpTools | length' "$file")

  autopi_info "Creating MCP server: $name"
  autopi_info "  Tools: $tool_count"

  autopi_api_file POST "/mcp-servers" "$file" | jq '{name, enabled, toolCount: (.mcpTools | length)}'
  autopi_ok "MCP server created: $name"
}

cmd_update() {
  local id="$1"
  local file="$2"
  [[ -z "$id" ]] && { autopi_error "MCP server ID (name) required"; exit 1; }
  [[ -z "$file" ]] && { autopi_error "MCP server file required"; exit 1; }
  [[ ! -f "$file" ]] && { autopi_error "File not found: $file"; exit 1; }

  # Validate JSON
  if ! jq empty "$file" 2>/dev/null; then
    autopi_error "Invalid JSON in: $file"
    exit 1
  fi

  autopi_info "Updating MCP server: $id"

  # Get current ETag
  local etag=$(get_etag "$id")
  if [[ -z "$etag" ]]; then
    autopi_error "Could not get ETag for: $id (does it exist?)"
    exit 1
  fi

  local encoded_id=$(url_encode "$id")
  curl -s -X PUT \
    -u "$AUTOPI_USER:$AUTOPI_PASS" \
    -H "Content-Type: application/json" \
    -H "If-Match: $etag" \
    -d @"$file" \
    "$AUTOPI_BASE_URL/mcp-servers/$encoded_id" | jq '{name, enabled, toolCount: (.mcpTools | length)}'

  autopi_ok "MCP server updated: $id"
}

cmd_delete() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "MCP server ID (name) required"; exit 1; }

  autopi_warn "Deleting MCP server: $id"

  # Get current ETag
  local etag=$(get_etag "$id")
  if [[ -z "$etag" ]]; then
    autopi_error "Could not get ETag for: $id (does it exist?)"
    exit 1
  fi

  local encoded_id=$(url_encode "$id")
  curl -s -X DELETE \
    -u "$AUTOPI_USER:$AUTOPI_PASS" \
    -H "If-Match: $etag" \
    "$AUTOPI_BASE_URL/mcp-servers/$encoded_id"

  autopi_ok "MCP server deleted: $id"
}

# Main
case "${1:-}" in
  list)   shift; cmd_list "$@" ;;
  get)    shift; cmd_get "$@" ;;
  create) shift; cmd_create "$@" ;;
  update) shift; cmd_update "$@" ;;
  delete) shift; cmd_delete "$@" ;;
  -h|--help|help) show_help ;;
  *)      show_help; exit 1 ;;
esac
