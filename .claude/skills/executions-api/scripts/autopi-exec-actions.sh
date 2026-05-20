#!/usr/bin/env bash
# SAP Automation Pilot - Execution Actions
#
# Usage:
#   autopi-exec-actions.sh abort <execution-id>   Abort running execution
#   autopi-exec-actions.sh pause <execution-id>   Pause execution
#   autopi-exec-actions.sh resume <execution-id>  Resume paused execution
#   autopi-exec-actions.sh list <execution-id>    List all actions for execution

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-exec-config.sh"

usage() {
  echo "Usage: $(basename "$0") <action> <execution-id>"
  echo ""
  echo "Actions:"
  echo "  abort <id>    Abort running execution"
  echo "  pause <id>    Pause execution"
  echo "  resume <id>   Resume paused execution"
  echo "  list <id>     List all actions for execution"
  exit 1
}

[[ $# -lt 2 ]] && usage

action="$1"
exec_id="$2"

request_action() {
  local action_type="$1"
  local exec_id="$2"

  autopi_info "Requesting $action_type for execution: $exec_id"

  local payload=$(jq -n --arg action "$action_type" '{action: $action}')
  local response=$(autopi_api POST "/executions/$exec_id/actions" "$payload")

  if echo "$response" | jq -e '.error' >/dev/null 2>&1; then
    autopi_error "Failed to $action_type execution:"
    echo "$response" | jq .
    exit 1
  fi

  autopi_ok "Action '$action_type' requested successfully"
  echo "$response" | jq .
}

list_actions() {
  local exec_id="$1"

  autopi_info "Listing actions for execution: $exec_id"
  autopi_api GET "/executions/$exec_id/actions" | jq .
}

case "$action" in
  abort)
    request_action "abort" "$exec_id"
    ;;
  pause)
    request_action "pause" "$exec_id"
    ;;
  resume)
    request_action "resume" "$exec_id"
    ;;
  list)
    list_actions "$exec_id"
    ;;
  *)
    autopi_error "Unknown action: $action"
    usage
    ;;
esac
