#!/usr/bin/env bash
# SAP Automation Pilot - Execution Logs
#
# Usage:
#   autopi-exec-logs.sh <execution-id>                    Get all logs
#   autopi-exec-logs.sh <execution-id> --page <n>         Paginated logs
#   autopi-exec-logs.sh <execution-id> <executor-path>    Specific executor logs
#
# Options:
#   --page <n>           Page number (0-indexed)
#   --size <n>           Page size (default 20)
#   --raw                Output raw JSON without formatting

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-exec-config.sh"

usage() {
  echo "Usage: $(basename "$0") <execution-id> [options]"
  echo ""
  echo "Options:"
  echo "  <executor-path>   Get logs for specific executor"
  echo "  --page <n>        Page number (0-indexed)"
  echo "  --size <n>        Page size (default 20)"
  echo "  --raw             Output raw JSON without formatting"
  echo ""
  echo "Examples:"
  echo "  $(basename "$0") abc-123                    # All logs"
  echo "  $(basename "$0") abc-123 --page 0          # First page"
  echo "  $(basename "$0") abc-123 step1             # Logs for step1"
  exit 1
}

[[ $# -lt 1 ]] && usage

exec_id="$1"
shift

executor_path=""
page=""
page_size=20
raw_output=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --page)
      page="$2"
      shift 2
      ;;
    --size)
      page_size="$2"
      shift 2
      ;;
    --raw)
      raw_output=true
      shift
      ;;
    -*)
      autopi_error "Unknown option: $1"
      exit 1
      ;;
    *)
      executor_path="$1"
      shift
      ;;
  esac
done

format_logs() {
  if [[ "$raw_output" == "true" ]]; then
    cat
  else
    jq -r '.logs[]? | "\(.timestamp) [\(.level)] \(.message)"' 2>/dev/null || jq .
  fi
}

if [[ -n "$executor_path" ]]; then
  # Get logs for specific executor
  autopi_info "Getting logs for executor '$executor_path' in execution: $exec_id"
  autopi_api GET "/executions/$exec_id/logs/$executor_path" | format_logs
elif [[ -n "$page" ]]; then
  # Get paginated logs
  autopi_info "Getting logs page $page for execution: $exec_id"
  autopi_api GET "/executions/$exec_id/logs?page=$page&maxPageSize=$page_size" | format_logs
else
  # Get all logs (first page)
  autopi_info "Getting logs for execution: $exec_id"
  autopi_api GET "/executions/$exec_id/logs?page=0&maxPageSize=$page_size" | format_logs
fi
