#!/usr/bin/env bash
# SAP Automation Pilot - Execution Troubleshooting
#
# Usage:
#   autopi-exec-troubleshoot.sh <execution-id>
#
# Provides comprehensive troubleshooting information:
# - Execution status and details
# - Command that was executed
# - Input parameters
# - Output (if finished)
# - Error messages (if failed)
# - Recent logs
# - Suggested actions

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-exec-config.sh"

usage() {
  echo "Usage: $(basename "$0") <execution-id>"
  echo ""
  echo "Provides comprehensive troubleshooting information for an execution."
  exit 1
}

[[ $# -lt 1 ]] && usage

exec_id="$1"

print_section() {
  echo ""
  echo "═══════════════════════════════════════════════════════════════════════════════"
  echo "  $1"
  echo "═══════════════════════════════════════════════════════════════════════════════"
}

print_subsection() {
  echo ""
  echo "─── $1 ───"
}

# Get execution details
autopi_info "Fetching execution details for: $exec_id"
exec_response=$(autopi_api GET "/executions/$exec_id")

# Check if it's an API error (has 'message' field without 'status' field)
if echo "$exec_response" | jq -e 'select(.message != null and .status == null)' >/dev/null 2>&1; then
  autopi_error "Execution not found: $exec_id"
  echo "$exec_response" | jq .
  exit 1
fi

status=$(echo "$exec_response" | jq -r '.status')
command_id=$(echo "$exec_response" | jq -r '.commandId')
started_at=$(echo "$exec_response" | jq -r '.startedAt')
finished_at=$(echo "$exec_response" | jq -r '.finishedAt // "N/A"')
trigger=$(echo "$exec_response" | jq -r '.trigger // "MANUAL"')

print_section "EXECUTION OVERVIEW"
echo ""
echo "  Execution ID:  $exec_id"
echo "  Command:       $command_id"
echo "  Status:        $status"
echo "  Trigger:       $trigger"
echo "  Started:       $started_at"
echo "  Finished:      $finished_at"

# Status indicator
case "$status" in
  FINISHED)
    autopi_ok "Execution completed successfully"
    ;;
  FAILED)
    autopi_error "Execution failed"
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
  INPUT_REQUIRED)
    autopi_warn "Execution is waiting for input"
    ;;
esac

# Input parameters
print_section "INPUT PARAMETERS"
input_response=$(autopi_api GET "/executions/$exec_id/input")
echo "$input_response" | jq .

# Output (if finished)
if [[ "$status" == "FINISHED" ]]; then
  print_section "OUTPUT"
  output_response=$(autopi_api GET "/executions/$exec_id/output")
  echo "$output_response" | jq .
fi

# Error details (if failed)
if [[ "$status" == "FAILED" ]]; then
  print_section "ERROR DETAILS"
  echo "$exec_response" | jq '{
    error: .error,
    errorMessage: .errorMessage,
    failedStep: .failedStep
  }'
fi

# Recent logs
print_section "RECENT LOGS"
logs_response=$(autopi_api GET "/executions/$exec_id/logs?page=0&maxPageSize=10")
echo "$logs_response" | jq -r '.logs[]? | "\(.timestamp) [\(.level)] \(.message)"' 2>/dev/null || echo "$logs_response" | jq .

# Suggested actions
print_section "SUGGESTED ACTIONS"
echo ""

case "$status" in
  FINISHED)
    echo "  ✓ Execution completed successfully. No action required."
    echo ""
    echo "  To view full output:"
    echo "    ./autopi-executions.sh output $exec_id"
    ;;
  FAILED)
    echo "  ✗ Execution failed. Review the error details above."
    echo ""
    echo "  Recommended steps:"
    echo "    1. Check the error message for root cause"
    echo "    2. Review the logs for more context:"
    echo "       ./autopi-exec-logs.sh $exec_id"
    echo "    3. Verify input parameters are correct"
    echo "    4. Check if the command has required dependencies"
    echo ""
    echo "  To retry the execution:"
    echo "    ./autopi-exec-trigger.sh \"$command_id\" '<input-json>'"
    ;;
  RUNNING)
    echo "  ⋯ Execution is still in progress."
    echo ""
    echo "  Options:"
    echo "    - Wait for completion"
    echo "    - View live logs:"
    echo "      ./autopi-exec-logs.sh $exec_id"
    echo "    - Abort if stuck:"
    echo "      ./autopi-exec-actions.sh abort $exec_id"
    ;;
  PAUSED)
    echo "  ⏸ Execution is paused."
    echo ""
    echo "  To resume:"
    echo "    ./autopi-exec-actions.sh resume $exec_id"
    echo ""
    echo "  To abort:"
    echo "    ./autopi-exec-actions.sh abort $exec_id"
    ;;
  ABORTED)
    echo "  ✗ Execution was manually aborted."
    echo ""
    echo "  To retry:"
    echo "    ./autopi-exec-trigger.sh \"$command_id\" '<input-json>'"
    ;;
  INPUT_REQUIRED)
    echo "  ⌨ Execution is waiting for input."
    echo ""
    echo "  Check suspended step:"
    echo "    ./autopi-executions.sh get $exec_id | jq '.suspendedStep'"
    ;;
  *)
    echo "  Status: $status"
    echo "  Review execution details and logs for more information."
    ;;
esac

echo ""
echo "═══════════════════════════════════════════════════════════════════════════════"
