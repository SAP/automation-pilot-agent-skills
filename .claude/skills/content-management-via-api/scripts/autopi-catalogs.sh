#!/usr/bin/env bash
# SAP Automation Pilot Content API - Catalog Operations
#
# Usage:
#   autopi-catalogs.sh list [--own|--provided|--all]
#   autopi-catalogs.sh get <catalog-id>
#   autopi-catalogs.sh create <name> <description>
#   autopi-catalogs.sh delete <catalog-id>

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-config.sh"

show_help() {
  cat << EOF
Usage: autopi-catalogs.sh <command> [options]

Commands:
  list [--own|--provided|--all]  List catalogs (default: --all)
  get <catalog-id>               Get catalog details
  create <name> <description>    Create a new catalog
  delete <catalog-id>            Delete an empty catalog

Examples:
  autopi-catalogs.sh list --own
  autopi-catalogs.sh get my-catalog-xxx
  autopi-catalogs.sh create "My Catalog" "Description here"
  autopi-catalogs.sh delete my-catalog-xxx
EOF
}

cmd_list() {
  local filter=""
  case "${1:-}" in
    --own)      filter="?own=true" ;;
    --provided) filter="?provided=true" ;;
    --all|"")   filter="" ;;
    *)          autopi_error "Unknown option: $1"; exit 1 ;;
  esac

  autopi_api GET "/catalogs$filter" | jq '[.[] | {id, name, description: .description[0:50]}]'
}

cmd_get() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "Catalog ID required"; exit 1; }

  autopi_api GET "/catalogs/$id" | jq .
}

cmd_create() {
  local name="$1"
  local desc="${2:-}"
  [[ -z "$name" ]] && { autopi_error "Catalog name required"; exit 1; }

  local data=$(jq -n --arg name "$name" --arg desc "$desc" \
    '{name: $name, description: $desc}')

  autopi_info "Creating catalog: $name"
  autopi_api POST "/catalogs" "$data" | jq .
}

cmd_delete() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "Catalog ID required"; exit 1; }

  autopi_warn "Deleting catalog: $id"
  local status=$(autopi_check "/catalogs/$id")

  if [[ "$status" == "404" ]]; then
    autopi_error "Catalog not found: $id"
    exit 1
  fi

  curl -s -X DELETE -u "$AUTOPI_USER:$AUTOPI_PASS" "$AUTOPI_BASE_URL/catalogs/$id"
  autopi_ok "Catalog deleted: $id"
}

# Main
case "${1:-}" in
  list)   shift; cmd_list "$@" ;;
  get)    shift; cmd_get "$@" ;;
  create) shift; cmd_create "$@" ;;
  delete) shift; cmd_delete "$@" ;;
  -h|--help|help) show_help ;;
  *)      show_help; exit 1 ;;
esac
