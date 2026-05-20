#!/usr/bin/env bash
# SAP Automation Pilot Content API - Command Operations
#
# Usage:
#   autopi-commands.sh list [catalog]
#   autopi-commands.sh get <command-id>
#   autopi-commands.sh deploy <file.command.json>
#   autopi-commands.sh delete <command-id>
#   autopi-commands.sh release <command-id>
#   autopi-commands.sh deprecate <command-id>
#   autopi-commands.sh restore <command-id>

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-config.sh"

show_help() {
  cat << EOF
Usage: autopi-commands.sh <command> [options]

Commands:
  list [catalog]              List commands in catalog (default: \$AUTOPI_CATALOG)
  ids [catalog]               List only command IDs (lightweight)
  get <command-id>            Get command details
  deploy <file>               Deploy command (create or update)
  delete <command-id>         Delete a command
  release <command-id>        Release a draft command
  deprecate <command-id>      Deprecate a command
  restore <command-id>        Restore a deprecated command

Examples:
  autopi-commands.sh list
  autopi-commands.sh list my-catalog-xxx
  autopi-commands.sh get my-catalog-xxx:MyCommand:1
  autopi-commands.sh deploy MyCommand.command.json
  autopi-commands.sh release my-catalog-xxx:MyCommand:1
EOF
}

cmd_list() {
  local catalog="${1:-$AUTOPI_CATALOG}"
  [[ -z "$catalog" ]] && { autopi_error "Catalog required (set AUTOPI_CATALOG or pass as argument)"; exit 1; }

  autopi_info "Listing commands in: $catalog"
  autopi_api GET "/commands?catalog=$catalog" | jq '[.[] | {id, name, description: .description[0:50]}]'
}

cmd_ids() {
  local catalog="${1:-$AUTOPI_CATALOG}"
  [[ -z "$catalog" ]] && { autopi_error "Catalog required"; exit 1; }

  autopi_api GET "/commands/ids?catalog=$catalog" | jq .
}

cmd_get() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "Command ID required"; exit 1; }

  autopi_api GET "/commands/$id" | jq .
}

cmd_deploy() {
  local file="$1"
  [[ -z "$file" ]] && { autopi_error "Command file required"; exit 1; }
  [[ ! -f "$file" ]] && { autopi_error "File not found: $file"; exit 1; }

  # Validate JSON
  if ! jq empty "$file" 2>/dev/null; then
    autopi_error "Invalid JSON in: $file"
    exit 1
  fi

  local cmd_id=$(jq -r '.id' "$file")
  autopi_info "Deploying: $cmd_id"

  # Check if exists
  local status=$(autopi_check "/commands/$cmd_id")

  if [[ "$status" == "200" ]]; then
    autopi_info "Updating existing command..."
    autopi_api_file PUT "/commands/$cmd_id" "$file" | jq '{id, name, issues: [.issues[]? | {severity, name}]}'
  else
    autopi_info "Creating new command..."
    autopi_api_file POST "/commands" "$file" | jq '{id, name, issues: [.issues[]? | {severity, name}]}'
  fi
}

cmd_delete() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "Command ID required"; exit 1; }

  autopi_warn "Deleting command: $id"
  curl -s -X DELETE -u "$AUTOPI_USER:$AUTOPI_PASS" "$AUTOPI_BASE_URL/commands/$id"
  autopi_ok "Command deleted"
}

cmd_lifecycle() {
  local action="$1"
  local id="$2"
  [[ -z "$id" ]] && { autopi_error "Command ID required"; exit 1; }

  autopi_info "${action^}ing command: $id"
  autopi_api PUT "/commands/$id/$action" | jq '{id, name, status: "success"}'
}

# Main
case "${1:-}" in
  list)      shift; cmd_list "$@" ;;
  ids)       shift; cmd_ids "$@" ;;
  get)       shift; cmd_get "$@" ;;
  deploy)    shift; cmd_deploy "$@" ;;
  delete)    shift; cmd_delete "$@" ;;
  release)   shift; cmd_lifecycle "release" "$@" ;;
  deprecate) shift; cmd_lifecycle "deprecate" "$@" ;;
  restore)   shift; cmd_lifecycle "restore" "$@" ;;
  -h|--help|help) show_help ;;
  *)         show_help; exit 1 ;;
esac
