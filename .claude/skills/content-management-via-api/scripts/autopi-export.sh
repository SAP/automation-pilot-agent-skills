#!/usr/bin/env bash
# SAP Automation Pilot Content API - Export
# Export commands or inputs to local files
#
# Usage:
#   autopi-export.sh command <command-id> [output-file]
#   autopi-export.sh input <input-id> [output-file]
#   autopi-export.sh catalog <catalog-id> [output-dir]

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

Examples:
  autopi-export.sh command my-cat:MyCmd:1 MyCmd.command.json
  autopi-export.sh input my-cat:MyInput:1
  autopi-export.sh catalog my-catalog-xxx ./exported/
EOF
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

# Main
case "${1:-}" in
  command) shift; cmd_export_command "$@" ;;
  input)   shift; cmd_export_input "$@" ;;
  catalog) shift; cmd_export_catalog "$@" ;;
  -h|--help|help) show_help ;;
  *)       show_help; exit 1 ;;
esac
