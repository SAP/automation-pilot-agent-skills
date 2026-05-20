#!/usr/bin/env bash
# SAP Automation Pilot Content API - Input Operations
#
# Usage:
#   autopi-inputs.sh list [catalog]
#   autopi-inputs.sh get <input-id>
#   autopi-inputs.sh create <file.input.json>
#   autopi-inputs.sh update <input-id> <file.input.json>
#   autopi-inputs.sh patch <input-id> '{"values": {...}}'
#   autopi-inputs.sh delete <input-id>
#   autopi-inputs.sh release <input-id>
#   autopi-inputs.sh deprecate <input-id>
#   autopi-inputs.sh restore <input-id>

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-config.sh"

show_help() {
  cat << EOF
Usage: autopi-inputs.sh <command> [options]

Commands:
  list [catalog]                  List inputs in catalog
  get <input-id>                  Get input details
  create <file.input.json>        Create a new input from file
  update <input-id> <file>        Full update of an input
  patch <input-id> '<json>'       Partial update (e.g., just values)
  delete <input-id>               Delete an input
  release <input-id>              Release a draft input
  deprecate <input-id>            Deprecate an input
  restore <input-id>              Restore a deprecated input

Examples:
  autopi-inputs.sh list my-catalog-xxx
  autopi-inputs.sh get my-catalog-xxx:MyInput:1
  autopi-inputs.sh patch my-catalog-xxx:MyInput:1 '{"values":{"key":"newvalue"}}'
EOF
}

cmd_list() {
  local catalog="${1:-$AUTOPI_CATALOG}"
  [[ -z "$catalog" ]] && { autopi_error "Catalog required"; exit 1; }

  autopi_info "Listing inputs in: $catalog"
  autopi_api GET "/inputs?catalog=$catalog" | jq '[.[] | {id, name, description: .description[0:50]}]'
}

cmd_get() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "Input ID required"; exit 1; }

  autopi_api GET "/inputs/$id" | jq .
}

cmd_create() {
  local file="$1"
  [[ -z "$file" ]] && { autopi_error "Input file required"; exit 1; }
  [[ ! -f "$file" ]] && { autopi_error "File not found: $file"; exit 1; }

  autopi_info "Creating input from: $file"
  autopi_api_file POST "/inputs" "$file" | jq '{id, name}'
}

cmd_update() {
  local id="$1"
  local file="$2"
  [[ -z "$id" ]] && { autopi_error "Input ID required"; exit 1; }
  [[ -z "$file" ]] && { autopi_error "Input file required"; exit 1; }
  [[ ! -f "$file" ]] && { autopi_error "File not found: $file"; exit 1; }

  autopi_info "Updating input: $id"
  autopi_api_file PUT "/inputs/$id" "$file" | jq '{id, name}'
}

cmd_patch() {
  local id="$1"
  local data="$2"
  [[ -z "$id" ]] && { autopi_error "Input ID required"; exit 1; }
  [[ -z "$data" ]] && { autopi_error "Patch data required (JSON)"; exit 1; }

  autopi_info "Patching input: $id"
  curl -s -X PATCH \
    -u "$AUTOPI_USER:$AUTOPI_PASS" \
    -H "Content-Type: application/merge-patch+json" \
    -d "$data" \
    "$AUTOPI_BASE_URL/inputs/$id" | jq '{id, name}'
}

cmd_delete() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "Input ID required"; exit 1; }

  autopi_warn "Deleting input: $id"
  curl -s -X DELETE -u "$AUTOPI_USER:$AUTOPI_PASS" "$AUTOPI_BASE_URL/inputs/$id"
  autopi_ok "Input deleted"
}

cmd_lifecycle() {
  local action="$1"
  local id="$2"
  [[ -z "$id" ]] && { autopi_error "Input ID required"; exit 1; }

  autopi_info "${action^}ing input: $id"
  autopi_api PUT "/inputs/$id/$action" | jq '{id, name}'
}

# Main
case "${1:-}" in
  list)      shift; cmd_list "$@" ;;
  get)       shift; cmd_get "$@" ;;
  create)    shift; cmd_create "$@" ;;
  update)    shift; cmd_update "$@" ;;
  patch)     shift; cmd_patch "$@" ;;
  delete)    shift; cmd_delete "$@" ;;
  release)   shift; cmd_lifecycle "release" "$@" ;;
  deprecate) shift; cmd_lifecycle "deprecate" "$@" ;;
  restore)   shift; cmd_lifecycle "restore" "$@" ;;
  -h|--help|help) show_help ;;
  *)         show_help; exit 1 ;;
esac
