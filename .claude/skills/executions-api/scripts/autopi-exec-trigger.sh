#!/usr/bin/env bash
# SAP Automation Pilot - Trigger Command Execution
#
# Usage:
#   autopi-exec-trigger.sh <command-id> [input-json]
#   autopi-exec-trigger.sh <command-id> --input-file <file>
#
# Options:
#   --input-file <file>   Read input from JSON file
#   --async               Don't wait for completion (default)
#   --wait                Wait for execution to complete
#   --timeout <seconds>   Max wait time (default 300)
#   --dry-run             Execute in dry run mode (no actual changes)
#
# Examples:
#   autopi-exec-trigger.sh "my-catalog:MyCommand:1"
#   autopi-exec-trigger.sh "my-catalog:MyCommand:1" '{"param": "value"}'
#   autopi-exec-trigger.sh "my-catalog:MyCommand:1" --input-file params.json
#   autopi-exec-trigger.sh "my-catalog:MyCommand:1" --dry-run

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-exec-config.sh"

usage() {
  echo "Usage: $(basename "$0") <command-id> [input-json] [options]"
  echo ""
  echo "Options:"
  echo "  --input-file <file>   Read input from JSON file"
  echo "  --wait                Wait for execution to complete"
  echo "  --timeout <seconds>   Max wait time (default 300)"
  echo "  --dry-run             Execute in dry run mode (no actual changes)"
  echo ""
  echo "Examples:"
  echo "  $(basename "$0") \"my-catalog:MyCommand:1\""
  echo "  $(basename "$0") \"my-catalog:MyCommand:1\" '{\"param\": \"value\"}'"
  echo "  $(basename "$0") \"my-catalog:MyCommand:1\" --input-file params.json"
  echo "  $(basename "$0") \"my-catalog:MyCommand:1\" --dry-run"
  exit 1
}

[[ $# -lt 1 ]] && usage

command_id="$1"
shift

input_json="{}"
input_file=""
wait_mode=false
timeout=300
dry_run=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --input-file)
      input_file="$2"
      shift 2
      ;;
    --wait)
      wait_mode=true
      shift
      ;;
    --timeout)
      timeout="$2"
      shift 2
      ;;
    --async)
      wait_mode=false
      shift
      ;;
    --dry-run)
      dry_run=true
      shift
      ;;
    -*)
      autopi_error "Unknown option: $1"
      exit 1
      ;;
    *)
      input_json="$1"
      shift
      ;;
  esac
done

# Load input from file if specified
if [[ -n "$input_file" ]]; then
  if [[ ! -f "$input_file" ]]; then
    autopi_error "Input file not found: $input_file"
    exit 1
  fi
  input_json=$(cat "$input_file")
fi

# Build tags object - always include feature:logs, optionally include feature:dryRun
if [[ "$dry_run" == "true" ]]; then
  tags='{"feature:logs":"","feature:dryRun":""}'
  autopi_warn "Dry Run mode enabled - no actual changes will be made"
else
  tags='{"feature:logs":""}'
fi

# Build request payload with tags
payload=$(jq -n \
  --arg cmd "$command_id" \
  --argjson input "$input_json" \
  --argjson tags "$tags" \
  '{commandId: $cmd, input: $input, tags: $tags}')

autopi_info "Triggering execution of: $command_id"

# Trigger execution
response=$(autopi_api POST "/executions" "$payload")

# Check for errors
if echo "$response" | jq -e '.error' >/dev/null 2>&1; then
  autopi_error "Failed to trigger execution:"
  echo "$response" | jq .
  exit 1
fi

exec_id=$(echo "$response" | jq -r '.id')
autopi_ok "Execution triggered: $exec_id"

if [[ "$wait_mode" == "true" ]]; then
  autopi_info "Waiting for completion (timeout: ${timeout}s)..."

  start_time=$(date +%s)
  while true; do
    status=$(autopi_api GET "/executions/$exec_id" | jq -r '.status')

    case "$status" in
      FINISHED)
        autopi_ok "Execution completed successfully"
        autopi_api GET "/executions/$exec_id/output" | jq .
        exit 0
        ;;
      FAILED)
        autopi_error "Execution failed"
        autopi_api GET "/executions/$exec_id" | jq .
        exit 1
        ;;
      ABORTED)
        autopi_warn "Execution was aborted"
        exit 1
        ;;
      *)
        current_time=$(date +%s)
        elapsed=$((current_time - start_time))
        if [[ $elapsed -ge $timeout ]]; then
          autopi_warn "Timeout reached. Execution still $status"
          autopi_info "Execution ID: $exec_id"
          exit 1
        fi
        sleep 5
        ;;
    esac
  done
else
  echo "$response" | jq .
fi
