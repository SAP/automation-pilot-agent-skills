#!/usr/bin/env bash
# SAP Automation Pilot - Scheduled Execution Management
#
# Usage:
#   autopi-schedules.sh list                List all schedules
#   autopi-schedules.sh get <id>            Get schedule details
#   autopi-schedules.sh delete <id>         Delete a schedule
#   autopi-schedules.sh enable <id>         Enable a schedule
#   autopi-schedules.sh disable <id>        Disable a schedule

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-sched-config.sh"

usage() {
  echo "Usage: $(basename "$0") <command> [options]"
  echo ""
  echo "Commands:"
  echo "  list              List all scheduled executions"
  echo "  get <id>          Get schedule details"
  echo "  delete <id>       Delete a schedule"
  echo "  enable <id>       Enable a schedule"
  echo "  disable <id>      Disable a schedule"
  echo ""
  echo "Examples:"
  echo "  $(basename "$0") list"
  echo "  $(basename "$0") get abc-123-def"
  echo "  $(basename "$0") enable abc-123-def"
  exit 1
}

list_schedules() {
  autopi_info "Listing scheduled executions..."
  autopi_api GET "/scheduled-executions" | jq .
}

get_schedule() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "Schedule ID required"; exit 1; }

  autopi_info "Getting schedule: $id"

  # Get with headers to show ETag
  local response
  response=$(autopi_get_with_headers "/scheduled-executions/$id")

  local etag
  etag=$(echo "$response" | grep -i "^etag:" | awk '{print $2}' | tr -d '\r\n')

  # Extract body (JSON line - last line or after blank line)
  local body
  body=$(echo "$response" | grep -E '^\{' | head -1)

  if [[ -n "$etag" ]]; then
    autopi_info "ETag: $etag"
  fi

  echo "$body" | jq .
}

delete_schedule() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "Schedule ID required"; exit 1; }

  autopi_info "Deleting schedule: $id"

  # First check if it exists
  local status
  status=$(autopi_check "/scheduled-executions/$id")

  if [[ "$status" == "404" ]]; then
    autopi_error "Schedule not found: $id"
    exit 1
  fi

  # Get ETag
  local etag
  etag=$(autopi_get_etag "/scheduled-executions/$id")

  if [[ -z "$etag" ]]; then
    autopi_error "Could not get ETag for schedule"
    exit 1
  fi

  autopi_info "Using ETag: $etag"

  # Delete with ETag
  local response
  response=$(curl -s -w "\n%{http_code}" -X DELETE \
    -u "$AUTOPI_USER:$AUTOPI_PASS" \
    -H "If-Match: $etag" \
    "$AUTOPI_BASE_URL/scheduled-executions/$id")

  local http_code
  http_code=$(echo "$response" | tail -n 1)
  local body
  body=$(echo "$response" | sed '$d')

  if [[ "$http_code" == "204" ]]; then
    autopi_ok "Deleted schedule: $id"
  elif [[ "$http_code" == "412" ]]; then
    autopi_error "ETag mismatch - schedule was modified. Please retry."
    exit 1
  else
    autopi_error "Failed to delete schedule (HTTP $http_code)"
    echo "$body" | jq . 2>/dev/null || echo "$body"
    exit 1
  fi
}

toggle_schedule() {
  local id="$1"
  local enabled="$2"

  [[ -z "$id" ]] && { autopi_error "Schedule ID required"; exit 1; }

  local action_word
  if [[ "$enabled" == "true" ]]; then
    action_word="Enabling"
  else
    action_word="Disabling"
  fi

  autopi_info "$action_word schedule: $id"

  # Get current schedule and ETag
  local response
  response=$(autopi_get_with_headers "/scheduled-executions/$id")

  local etag
  etag=$(echo "$response" | grep -i "^etag:" | awk '{print $2}' | tr -d '\r\n')

  local body
  body=$(echo "$response" | grep -E '^\{' | head -1)

  if [[ -z "$etag" ]]; then
    autopi_error "Could not get ETag for schedule"
    exit 1
  fi

  # Update the enabled field
  local updated
  updated=$(echo "$body" | jq --argjson enabled "$enabled" '.enabled = $enabled')

  # PUT with ETag
  local result
  result=$(curl -s -w "\n%{http_code}" -X PUT \
    -u "$AUTOPI_USER:$AUTOPI_PASS" \
    -H "Content-Type: application/json" \
    -H "If-Match: $etag" \
    -d "$updated" \
    "$AUTOPI_BASE_URL/scheduled-executions/$id")

  local http_code
  http_code=$(echo "$result" | tail -1)
  local result_body
  result_body=$(echo "$result" | sed '$d')

  if [[ "$http_code" == "200" ]]; then
    if [[ "$enabled" == "true" ]]; then
      autopi_ok "Enabled schedule: $id"
    else
      autopi_ok "Disabled schedule: $id"
    fi
    echo "$result_body" | jq .
  elif [[ "$http_code" == "412" ]]; then
    autopi_error "ETag mismatch - schedule was modified. Please retry."
    exit 1
  else
    autopi_error "Failed to update schedule (HTTP $http_code)"
    echo "$result_body" | jq . 2>/dev/null || echo "$result_body"
    exit 1
  fi
}

# Main
case "${1:-}" in
  list)
    list_schedules
    ;;
  get)
    get_schedule "$2"
    ;;
  delete)
    delete_schedule "$2"
    ;;
  enable)
    toggle_schedule "$2" "true"
    ;;
  disable)
    toggle_schedule "$2" "false"
    ;;
  *)
    usage
    ;;
esac
