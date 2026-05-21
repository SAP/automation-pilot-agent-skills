#!/usr/bin/env bash
# SAP Automation Pilot Content API - Export
# Export commands, inputs, or MCP servers to local files
#
# Usage:
#   autopi-export.sh command <command-id> [output-file]
#   autopi-export.sh input <input-id> [output-file]
#   autopi-export.sh catalog <catalog-id> [output-dir]
#   autopi-export.sh mcp-server <name> [output-file]
#   autopi-export.sh mcp-servers [output-dir]

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-config.sh"

show_help() {
  cat << EOF
Usage: autopi-export.sh <type> <id> [output]

Types:
  command <command-id> [file]    Export a command to file
  input <input-id> [file]        Export an input to file
  catalog <catalog-id> [dir]     Export all commands in catalog to directory
  mcp-server <name> [file]       Export an MCP server to file
  mcp-servers [dir]              Export all MCP servers to directory

Examples:
  autopi-export.sh command my-cat:MyCmd:1 MyCmd.command.json
  autopi-export.sh input my-cat:MyInput:1
  autopi-export.sh catalog my-catalog-xxx ./exported/
  autopi-export.sh mcp-server "BTP Resource Discovery" btp-discovery.json
  autopi-export.sh mcp-servers ./exported-servers/
EOF
}

# URL-encode a string for use in API paths
url_encode() {
  local string="$1"
  python3 -c "import urllib.parse; print(urllib.parse.quote('''$string''', safe=''))"
}

cmd_export_command() {
  local id="$1"
  local output="$2"
  [[ -z "$id" ]] && { autopi_error "Command ID required"; exit 1; }

  # Default filename from command name
  if [[ -z "$output" ]]; then
    local name=$(echo "$id" | cut -d: -f2)
    output="${name}.command.json"
  fi

  autopi_info "Exporting command: $id"
  autopi_api GET "/commands/$id" | jq . > "$output"
  autopi_ok "Exported to: $output"
}

cmd_export_input() {
  local id="$1"
  local output="$2"
  [[ -z "$id" ]] && { autopi_error "Input ID required"; exit 1; }

  if [[ -z "$output" ]]; then
    local name=$(echo "$id" | cut -d: -f2)
    output="${name}.input.json"
  fi

  autopi_info "Exporting input: $id"
  autopi_api GET "/inputs/$id" | jq . > "$output"
  autopi_ok "Exported to: $output"
}

cmd_export_catalog() {
  local catalog="$1"
  local dir="${2:-.}"
  [[ -z "$catalog" ]] && { autopi_error "Catalog ID required"; exit 1; }

  mkdir -p "$dir"
  autopi_info "Exporting all commands from catalog: $catalog"
  autopi_info "Output directory: $dir"

  # Get all command IDs
  local ids=$(autopi_api GET "/commands/ids?catalog=$catalog" | jq -r '.[]')

  local count=0
  for id in $ids; do
    local name=$(echo "$id" | cut -d: -f2)
    local output="$dir/${name}.command.json"
    autopi_api GET "/commands/$id" | jq . > "$output"
    echo "  - $name"
    ((count++))
  done

  autopi_ok "Exported $count commands to: $dir"
}

cmd_export_mcp_server() {
  local name="$1"
  local output="$2"
  [[ -z "$name" ]] && { autopi_error "MCP server name required"; exit 1; }

  if [[ -z "$output" ]]; then
    # Sanitize name for filename (replace spaces with hyphens)
    local safe_name=$(echo "$name" | tr ' ' '-')
    output="${safe_name}.json"
  fi

  local encoded_name=$(url_encode "$name")
  autopi_info "Exporting MCP server: $name"
  autopi_api GET "/mcp-servers/$encoded_name" | jq . > "$output"
  autopi_ok "Exported to: $output"
}

cmd_export_mcp_servers() {
  local dir="${1:-.}"
  mkdir -p "$dir"
  autopi_info "Exporting all MCP servers"
  autopi_info "Output directory: $dir"

  # Get all MCP server names
  local names=$(autopi_api GET "/mcp-servers" | jq -r '.[].name')

  local count=0
  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    local encoded_name=$(url_encode "$name")
    # Sanitize name for filename
    local safe_name=$(echo "$name" | tr ' ' '-')
    local output="$dir/${safe_name}.json"
    autopi_api GET "/mcp-servers/$encoded_name" | jq . > "$output"
    echo "  - $name"
    ((count++))
  done <<< "$names"

  autopi_ok "Exported $count MCP servers to: $dir"
}

# Main
case "${1:-}" in
  command)     shift; cmd_export_command "$@" ;;
  input)       shift; cmd_export_input "$@" ;;
  catalog)     shift; cmd_export_catalog "$@" ;;
  mcp-server)  shift; cmd_export_mcp_server "$@" ;;
  mcp-servers) shift; cmd_export_mcp_servers "$@" ;;
  -h|--help|help) show_help ;;
  *)           show_help; exit 1 ;;
esac
