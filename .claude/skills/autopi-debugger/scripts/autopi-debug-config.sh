#!/usr/bin/env bash
# SAP Automation Pilot Debug - Configuration & Helper Functions
#
# Source this file to load debug functions
# Usage: source .claude/skills/autopi-debug/scripts/autopi-debug-config.sh
#
# Required environment variables (from .env):
#   AUTOPI_HOSTNAME, AUTOPI_USERNAME, AUTOPI_PASSWORD

# Check required environment variables
_debug_check_env() {
  local missing=()
  [[ -z "$AUTOPI_HOSTNAME" ]] && missing+=("AUTOPI_HOSTNAME")
  [[ -z "$AUTOPI_USERNAME" ]] && missing+=("AUTOPI_USERNAME")
  [[ -z "$AUTOPI_PASSWORD" ]] && missing+=("AUTOPI_PASSWORD")

  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "Error: Missing required environment variables: ${missing[*]}" >&2
    return 1
  fi
  return 0
}

if ! _debug_check_env; then
  return 1 2>/dev/null || exit 1
fi

export DEBUG_HOST="$AUTOPI_HOSTNAME"
export DEBUG_USER="$AUTOPI_USERNAME"
export DEBUG_PASS="$AUTOPI_PASSWORD"
export DEBUG_BASE_URL="https://$DEBUG_HOST/api/v1"

# Color output
_debug_info()  { echo -e "\033[0;34m[INFO]\033[0m $*"; }
_debug_ok()    { echo -e "\033[0;32m[OK]\033[0m $*"; }
_debug_warn()  { echo -e "\033[0;33m[WARN]\033[0m $*"; }
_debug_error() { echo -e "\033[0;31m[ERROR]\033[0m $*" >&2; }
_debug_fail()  { echo -e "\033[0;31m[FAIL]\033[0m $*"; }
_debug_pass()  { echo -e "\033[0;32m[PASS]\033[0m $*"; }

# Base API call
_debug_api() {
  local method="$1"
  local endpoint="$2"
  local data="$3"

  if [[ -n "$data" ]]; then
    curl -s -X "$method" \
      -u "$DEBUG_USER:$DEBUG_PASS" \
      -H "Content-Type: application/json" \
      -d "$data" \
      "$DEBUG_BASE_URL$endpoint"
  else
    curl -s -X "$method" \
      -u "$DEBUG_USER:$DEBUG_PASS" \
      "$DEBUG_BASE_URL$endpoint"
  fi
}

# ============================================
# SUMMARY FUNCTIONS
# ============================================

# Show summary of last N executions
# Usage: autopi_debug_summary [limit]
autopi_debug_summary() {
  local limit="${1:-10}"

  _debug_info "Last $limit executions summary:"
  echo ""

  local response=$(_debug_api GET "/executions?limit=$limit")

  local total=$(echo "$response" | jq 'length')
  local finished=$(echo "$response" | jq '[.[] | select(.status == "FINISHED")] | length')
  local failed=$(echo "$response" | jq '[.[] | select(.status == "FAILED")] | length')
  local running=$(echo "$response" | jq '[.[] | select(.status == "RUNNING")] | length')
  local other=$((total - finished - failed - running))

  echo "  Total:    $total"
  echo -e "  \033[0;32mPassed:   $finished\033[0m"
  echo -e "  \033[0;31mFailed:   $failed\033[0m"
  echo -e "  \033[0;33mRunning:  $running\033[0m"
  [[ $other -gt 0 ]] && echo "  Other:    $other"
  echo ""

  echo "Details:"
  echo "$response" | jq -r '.[] | "\(.status | if . == "FINISHED" then "[32m[PASS][0m" elif . == "FAILED" then "[31m[FAIL][0m" elif . == "RUNNING" then "[33m[RUN][0m" else "[\(.)]" end) \(.commandId | split(":")[1]) - \(.id)"'
}

# List only failed executions
# Usage: autopi_debug_failures [limit]
autopi_debug_failures() {
  local limit="${1:-10}"

  _debug_info "Last $limit failed executions:"
  echo ""

  _debug_api GET "/executions?status=FAILED&limit=$limit" | \
    jq -r '.[] | "\(.id) | \(.commandId | split(":")[1]) | \(.error // "No error message" | .[0:80])"'
}

# List only successful executions
# Usage: autopi_debug_successes [limit]
autopi_debug_successes() {
  local limit="${1:-10}"

  _debug_info "Last $limit successful executions:"
  echo ""

  _debug_api GET "/executions?status=FINISHED&limit=$limit" | \
    jq -r '.[] | "\(.id) | \(.commandId | split(":")[1])"'
}

# ============================================
# DIAGNOSIS FUNCTIONS
# ============================================

# Full diagnosis of a specific execution
# Usage: autopi_debug_diagnose <execution-id>
autopi_debug_diagnose() {
  local exec_id="$1"

  if [[ -z "$exec_id" ]]; then
    _debug_error "Usage: autopi_debug_diagnose <execution-id>"
    return 1
  fi

  _debug_info "Diagnosing execution: $exec_id"
  echo ""

  # Fetch and save to temp file to handle special characters
  local tmp_file=$(mktemp)
  _debug_api GET "/executions/$exec_id" > "$tmp_file"

  # Basic info - use raw output to handle special chars
  local cmd_id=$(jq -r '.commandId // "Unknown"' "$tmp_file" 2>/dev/null || echo "Unknown")
  local exec_status=$(jq -r '.status // "Unknown"' "$tmp_file" 2>/dev/null || echo "Unknown")
  local progress_msg=$(jq -r '.progressMessage // "N/A"' "$tmp_file" 2>/dev/null || echo "N/A")
  local error_msg=$(jq -r '.error // "None"' "$tmp_file" 2>/dev/null || echo "Check raw output")
  local start_time=$(jq -r '.startTime // 0' "$tmp_file" 2>/dev/null || echo "0")
  local mod_time=$(jq -r '.modificationTime // 0' "$tmp_file" 2>/dev/null || echo "0")

  rm -f "$tmp_file"

  echo "=== EXECUTION INFO ==="
  echo "  Command:  $cmd_id"
  echo "  Status:   $exec_status"
  echo "  Progress: $progress_msg"
  echo "  Started:  $(date -r $((start_time / 1000)) 2>/dev/null || echo $start_time)"
  echo ""

  # Status-specific output
  if [[ "$exec_status" == "FAILED" ]]; then
    echo "=== ERROR ==="
    _debug_fail "$error_msg"
    echo ""

    # Try to match error pattern
    echo "=== SUGGESTED FIX ==="
    if [[ "$error_msg" == *"is required but not provided"* ]]; then
      _debug_warn "Known issue: Required parameters may not be passed correctly via API."
      echo "  Workaround: Hardcode values in command definition or make inputs optional."
    elif [[ "$error_msg" == *"Missing valid combination of input values for authentication"* ]]; then
      _debug_warn "Authentication required for this service."
      echo "  Fix: Provide 'user' + 'password' or 'clientCert' for authentication."
    elif [[ "$error_msg" == *"sensitive"* && "$error_msg" == *"defaultValue"* ]]; then
      _debug_warn "Sensitive fields cannot have defaultValue."
      echo "  Fix: Remove defaultValue from sensitive input keys."
    else
      echo "  No known pattern matched. Check logs for more details."
    fi
    echo ""
  elif [[ "$exec_status" == "RUNNING" ]]; then
    _debug_warn "Execution is still running."
    echo "  Current step: $progress_msg"
    echo "  To abort: autopi_debug_abort $exec_id"
    echo ""
  elif [[ "$exec_status" == "FINISHED" ]]; then
    _debug_pass "Execution completed successfully."
    echo ""

    # Show output
    echo "=== OUTPUT ==="
    _debug_api GET "/executions/$exec_id/output" | jq '.values'
  fi

  # Show input
  echo ""
  echo "=== INPUT ==="
  _debug_api GET "/executions/$exec_id/input" | jq '.values'
}

# Get just the error message
# Usage: autopi_debug_error <execution-id>
autopi_debug_error() {
  local exec_id="$1"

  if [[ -z "$exec_id" ]]; then
    _debug_error "Usage: autopi_debug_error <execution-id>"
    return 1
  fi

  _debug_api GET "/executions/$exec_id" | jq -r '.error // "No error"'
}

# Get execution logs
# Usage: autopi_debug_logs <execution-id> [page]
autopi_debug_logs() {
  local exec_id="$1"
  local page="${2:-0}"

  if [[ -z "$exec_id" ]]; then
    _debug_error "Usage: autopi_debug_logs <execution-id> [page]"
    return 1
  fi

  _debug_info "Logs for $exec_id (page $page):"
  _debug_api GET "/executions/$exec_id/logs?page=$page&maxPageSize=20" | jq '.logs[]?'
}

# ============================================
# ACTION FUNCTIONS
# ============================================

# Abort a running execution
# Usage: autopi_debug_abort <execution-id>
autopi_debug_abort() {
  local exec_id="$1"

  if [[ -z "$exec_id" ]]; then
    _debug_error "Usage: autopi_debug_abort <execution-id>"
    return 1
  fi

  _debug_warn "Aborting execution: $exec_id"
  _debug_api POST "/executions/$exec_id/actions" '{"action":"abort"}' | jq .
}

# ============================================
# COMMAND VERIFICATION
# ============================================

# Check if a command is released
# Usage: autopi_debug_check_command <command-id>
autopi_debug_check_command() {
  local cmd_id="$1"

  if [[ -z "$cmd_id" ]]; then
    _debug_error "Usage: autopi_debug_check_command <command-id>"
    return 1
  fi

  _debug_info "Checking command: $cmd_id"

  local cmd_data=$(_debug_api GET "/commands/$cmd_id")

  if echo "$cmd_data" | jq -e '.code == 404' >/dev/null 2>&1; then
    _debug_fail "Command not found"
    return 1
  fi

  local released=$(echo "$cmd_data" | jq -r '.tags["autopi:released"] // "NOT RELEASED"')
  local issues=$(echo "$cmd_data" | jq '.issues | length')

  echo "  Released: $([ "$released" != "NOT RELEASED" ] && echo "Yes" || echo "No")"
  echo "  Issues:   $issues"

  if [[ "$released" == "NOT RELEASED" ]]; then
    _debug_warn "Command is not released. Run: curl -X PUT .../commands/$cmd_id/release"
  fi

  if [[ "$issues" -gt 0 ]]; then
    _debug_warn "Command has $issues design-time issues:"
    echo "$cmd_data" | jq -r '.issues[] | "    - [\(.severity)] \(.name)"'
  fi
}

# Show help unless AUTOPI_DEBUG_QUIET is set
if [[ -z "$AUTOPI_DEBUG_QUIET" ]]; then
  _debug_info "Debug functions loaded. Available commands:"
  echo "  autopi_debug_summary [N]        - Summary of last N executions"
  echo "  autopi_debug_failures [N]       - List last N failures"
  echo "  autopi_debug_successes [N]      - List last N successes"
  echo "  autopi_debug_diagnose <id>      - Full diagnosis of execution"
  echo "  autopi_debug_error <id>         - Get error message only"
  echo "  autopi_debug_logs <id> [page]   - View execution logs"
  echo "  autopi_debug_abort <id>         - Abort running execution"
  echo "  autopi_debug_check_command <id> - Verify command status"
  echo ""
  echo "  Tip: export AUTOPI_DEBUG_QUIET=1 to suppress this message"
fi
