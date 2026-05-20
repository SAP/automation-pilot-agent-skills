#!/usr/bin/env bash
# SAP Automation Pilot - Execution Management
#
# Usage:
#   autopi-executions.sh list [options]     List executions
#   autopi-executions.sh get <id>           Get execution details
#   autopi-executions.sh delete <id>        Delete completed execution
#   autopi-executions.sh input <id>         Get execution input
#   autopi-executions.sh output <id>        Get execution output
#
# List options:
#   --status <status>      Filter by status (RUNNING, FINISHED, FAILED, etc.)
#   --command <id>         Filter by command ID
#   --after <timestamp>    Started after (epoch ms)
#   --before <timestamp>   Started before (epoch ms)
#   --limit <n>            Max results (default 100)

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-exec-config.sh"

usage() {
  echo "Usage: $(basename "$0") <command> [options]"
  echo ""
  echo "Commands:"
  echo "  list [options]    List executions"
  echo "  get <id>          Get execution details"
  echo "  delete <id>       Delete completed execution"
  echo "  input <id>        Get execution input"
  echo "  output <id>       Get execution output"
  echo ""
  echo "List options:"
  echo "  --status <status>    Filter by status"
  echo "  --command <id>       Filter by command ID"
  echo "  --after <timestamp>  Started after (epoch ms)"
  echo "  --before <timestamp> Started before (epoch ms)"
  echo "  --limit <n>          Max results (default 100)"
  exit 1
}

list_executions() {
  local query=""
  local limit=100
  local status=""
  local command_id=""
  local after=""
  local before=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --status)
        status="$2"
        shift 2
        ;;
      --command)
        command_id="$2"
        shift 2
        ;;
      --after)
        after="$2"
        shift 2
        ;;
      --before)
        before="$2"
        shift 2
        ;;
      --limit)
        limit="$2"
        shift 2
        ;;
      *)
        autopi_error "Unknown option: $1"
        exit 1
        ;;
    esac
  done

  query="limit=$limit"
  [[ -n "$status" ]] && query="$query&status=$status"
  [[ -n "$command_id" ]] && query="$query&commandId=$command_id"
  [[ -n "$after" ]] && query="$query&startedAfter=$after"
  [[ -n "$before" ]] && query="$query&startedBefore=$before"

  autopi_info "Listing executions..."
  autopi_api GET "/executions?$query" | jq .
}

get_execution() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "Execution ID required"; exit 1; }

  autopi_info "Getting execution: $id"
  autopi_api GET "/executions/$id" | jq .
}

delete_execution() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "Execution ID required"; exit 1; }

  autopi_info "Deleting execution: $id"
  local status=$(autopi_check "/executions/$id")

  if [[ "$status" == "404" ]]; then
    autopi_error "Execution not found: $id"
    exit 1
  fi

  curl -s -X DELETE \
    -u "$AUTOPI_USER:$AUTOPI_PASS" \
    "$AUTOPI_BASE_URL/executions/$id"

  autopi_ok "Deleted execution: $id"
}

get_input() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "Execution ID required"; exit 1; }

  autopi_info "Getting input for execution: $id"
  autopi_api GET "/executions/$id/input" | jq .
}

get_output() {
  local id="$1"
  [[ -z "$id" ]] && { autopi_error "Execution ID required"; exit 1; }

  autopi_info "Getting output for execution: $id"
  autopi_api GET "/executions/$id/output" | jq .
}

# Main
case "${1:-}" in
  list)
    shift
    list_executions "$@"
    ;;
  get)
    get_execution "$2"
    ;;
  delete)
    delete_execution "$2"
    ;;
  input)
    get_input "$2"
    ;;
  output)
    get_output "$2"
    ;;
  *)
    usage
    ;;
esac
