#!/usr/bin/env bash
# SAP Automation Pilot Content API - Webhook Operations
#
# Usage:
#   autopi-webhooks.sh list
#   autopi-webhooks.sh get <webhook-id>
#   autopi-webhooks.sh create <name> <command-id>
#   autopi-webhooks.sh trigger <webhook-id> [event-json]
#   autopi-webhooks.sh delete <webhook-id>

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-config.sh"

show_help() {
  cat << EOF
Usage: autopi-webhooks.sh <command> [options]

Commands:
  list                           List all webhooks
  get <webhook-id>               Get webhook details
  create <name> <command-id>     Create a new webhook
  trigger <webhook-id> [json]    Trigger/execute a webhook
  delete <webhook-id>            Delete a webhook

Examples:
  autopi-webhooks.sh list
  autopi-webhooks.sh get my-webhook-id
  autopi-webhooks.sh create "My Webhook" "my-catalog:MyCommand:1"
  autopi-webhooks.sh trigger my-webhook-id '{"message":"hello"}'
EOF
}

cmd_list() {
  autopi_info "Listing webhooks"
  autopi_api GET "/webhooks" | jq '[.[] | {id, name, commandId}]'
}

cmd_get() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "Webhook ID required"; exit 1; }

  autopi_api GET "/webhooks/$id" | jq .
}

cmd_create() {
  local name="$1"
  local cmd_id="$2"
  [[ -z "$name" ]] && { autopi_error "Webhook name required"; exit 1; }
  [[ -z "$cmd_id" ]] && { autopi_error "Command ID required"; exit 1; }

  local data=$(jq -n --arg name "$name" --arg cmdId "$cmd_id" \
    '{name: $name, commandId: $cmdId}')

  autopi_info "Creating webhook: $name -> $cmd_id"
  autopi_api POST "/webhooks" "$data" | jq '{id, name, commandId}'
}

cmd_trigger() {
  local id="$1"
  local event="${2:-}"
  [[ -z "$id" ]] && { autopi_error "Webhook ID required"; exit 1; }

  autopi_info "Triggering webhook: $id"

  if [[ -n "$event" ]]; then
    curl -s -X POST \
      -u "$AUTOPI_USER:$AUTOPI_PASS" \
      -H "Content-Type: application/json" \
      -d "$event" \
      "$AUTOPI_BASE_URL/webhooks/$id/trigger" | jq '{executionId: .id, status: .status}'
  else
    curl -s -X POST \
      -u "$AUTOPI_USER:$AUTOPI_PASS" \
      "$AUTOPI_BASE_URL/webhooks/$id/trigger" | jq '{executionId: .id, status: .status}'
  fi
}

cmd_delete() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "Webhook ID required"; exit 1; }

  autopi_warn "Deleting webhook: $id"
  curl -s -X DELETE -u "$AUTOPI_USER:$AUTOPI_PASS" "$AUTOPI_BASE_URL/webhooks/$id"
  autopi_ok "Webhook deleted"
}

# Main
case "${1:-}" in
  list)    shift; cmd_list "$@" ;;
  get)     shift; cmd_get "$@" ;;
  create)  shift; cmd_create "$@" ;;
  trigger) shift; cmd_trigger "$@" ;;
  delete)  shift; cmd_delete "$@" ;;
  -h|--help|help) show_help ;;
  *)       show_help; exit 1 ;;
esac
