#!/usr/bin/env bash
# SAP Automation Pilot - List Time Zones
#
# Usage:
#   autopi-sched-timezones.sh [options]
#
# Options:
#   --filter <pattern>   Filter time zones by pattern
#   --json               Output raw JSON
#
# Examples:
#   autopi-sched-timezones.sh
#   autopi-sched-timezones.sh --filter "Europe"
#   autopi-sched-timezones.sh --filter "America/New"

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-sched-config.sh"

usage() {
  echo "Usage: $(basename "$0") [options]"
  echo ""
  echo "Options:"
  echo "  --filter <pattern>   Filter time zones by pattern (case-insensitive)"
  echo "  --json               Output raw JSON"
  echo ""
  echo "Examples:"
  echo "  $(basename "$0")"
  echo "  $(basename "$0") --filter \"Europe\""
  echo "  $(basename "$0") --filter \"America/New\""
  exit 1
}

FILTER=""
JSON_OUTPUT=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --filter)
      FILTER="$2"
      shift 2
      ;;
    --json)
      JSON_OUTPUT=true
      shift
      ;;
    -h|--help)
      usage
      ;;
    *)
      autopi_error "Unknown option: $1"
      exit 1
      ;;
  esac
done

autopi_info "Fetching time zones..."

RESPONSE=$(autopi_api GET "/scheduled-executions/time-zones")

if [[ "$JSON_OUTPUT" == "true" ]]; then
  if [[ -n "$FILTER" ]]; then
    echo "$RESPONSE" | jq --arg filter "$FILTER" '[.[] | select(. | ascii_downcase | contains($filter | ascii_downcase))]'
  else
    echo "$RESPONSE" | jq .
  fi
else
  if [[ -n "$FILTER" ]]; then
    autopi_info "Filtering by: $FILTER"
    echo "$RESPONSE" | jq -r --arg filter "$FILTER" '.[] | select(. | ascii_downcase | contains($filter | ascii_downcase))' | sort
  else
    echo "$RESPONSE" | jq -r '.[]' | sort
  fi

  # Show count
  if [[ -n "$FILTER" ]]; then
    COUNT=$(echo "$RESPONSE" | jq --arg filter "$FILTER" '[.[] | select(. | ascii_downcase | contains($filter | ascii_downcase))] | length')
  else
    COUNT=$(echo "$RESPONSE" | jq 'length')
  fi
  echo ""
  autopi_info "Total: $COUNT time zones"
fi
