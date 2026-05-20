#!/usr/bin/env bash
# SAP Automation Pilot - Execution Status
#
# Usage:
#   autopi-exec-status.sh                      Summary of all executions
#   autopi-exec-status.sh <execution-id>       Quick status of one execution
#   autopi-exec-status.sh --running            List running executions
#   autopi-exec-status.sh --failed             List failed executions
#   autopi-exec-status.sh --finished           List finished executions

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-exec-config.sh"

usage() {
  echo "Usage: $(basename "$0") [options]"
  echo ""
  echo "Options:"
  echo "  <execution-id>    Quick status of one execution"
  echo "  --running         List running executions"
  echo "  --failed          List failed executions"
  echo "  --finished        List finished executions"
  echo "  --paused          List paused executions"
  echo "  --summary         Show execution counts by status"
  echo "  --limit <n>       Max results (default 20)"
  exit 1
}

limit=20

show_summary() {
  autopi_info "Execution summary:"
  autopi_api GET "/executions/summary" | jq .
}

list_by_status() {
  local status="$1"
  autopi_info "Listing $status executions (limit: $limit)..."

  autopi_api GET "/executions?status=$status&limit=$limit" | jq -r '
    .[]? |
    "\(.id) | \(.commandId) | \(.startTime) | \(.status)"
  ' | column -t -s '|'
}

quick_status() {
  local exec_id="$1"

  local response=$(autopi_api GET "/executions/$exec_id")

  if echo "$response" | jq -e '.error' >/dev/null 2>&1; then
    autopi_error "Execution not found: $exec_id"
    exit 1
  fi

  local status=$(echo "$response" | jq -r '.status')
  local command=$(echo "$response" | jq -r '.commandId')
  local started=$(echo "$response" | jq -r '.startedAt')
  local finished=$(echo "$response" | jq -r '.finishedAt // "N/A"')

  echo "Execution: $exec_id"
  echo "Command:   $command"
  echo "Status:    $status"
  echo "Started:   $started"
  echo "Finished:  $finished"

  case "$status" in
    FINISHED)
      autopi_ok "Execution completed successfully"
      ;;
    FAILED)
      autopi_error "Execution failed"
      echo ""
      echo "Error details:"
      echo "$response" | jq -r '.error // "No error details available"'
      ;;
    RUNNING)
      autopi_info "Execution is still running"
      ;;
    PAUSED)
      autopi_warn "Execution is paused"
      ;;
    ABORTED)
      autopi_warn "Execution was aborted"
      ;;
  esac
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    --running)
      list_by_status "RUNNING"
      exit 0
      ;;
    --failed)
      list_by_status "FAILED"
      exit 0
      ;;
    --finished)
      list_by_status "FINISHED"
      exit 0
      ;;
    --paused)
      list_by_status "PAUSED"
      exit 0
      ;;
    --summary)
      show_summary
      exit 0
      ;;
    --limit)
      limit="$2"
      shift 2
      ;;
    --help|-h)
      usage
      ;;
    -*)
      autopi_error "Unknown option: $1"
      exit 1
      ;;
    *)
      # Assume it's an execution ID
      quick_status "$1"
      exit 0
      ;;
  esac
done

# Default: show summary
show_summary
