#!/usr/bin/env bash
# SAP Automation Pilot - Update Scheduled Execution
#
# Usage:
#   autopi-sched-update.sh <schedule-id> [options]
#
# Options:
#   --hourly --minutes <list>          Change to hourly schedule
#   --daily --hour <h> --minute <m>    Change to daily schedule
#   --weekly --days <list> --hour <h> --minute <m>   Change to weekly schedule
#   --monthly --days <list> --hour <h> --minute <m>  Change to monthly schedule
#   --timezone <tz>       Update time zone
#   --input <json>        Update input parameters
#   --description <desc>  Update description
#   --enabled <bool>      Enable/disable (true/false)
#
# Examples:
#   autopi-sched-update.sh abc-123 --daily --hour 10 --minute 0
#   autopi-sched-update.sh abc-123 --enabled false
#   autopi-sched-update.sh abc-123 --timezone "America/New_York"

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-sched-config.sh"

usage() {
  echo "Usage: $(basename "$0") <schedule-id> [options]"
  echo ""
  echo "Schedule Types (to change schedule):"
  echo "  --hourly --minutes <list>          Change to hourly schedule"
  echo "  --daily --hour <h> --minute <m>    Change to daily schedule"
  echo "  --weekly --days <list> --hour <h> --minute <m>   Change to weekly schedule"
  echo "  --monthly --days <list> --hour <h> --minute <m>  Change to monthly schedule"
  echo ""
  echo "Options:"
  echo "  --timezone <tz>       Update time zone"
  echo "  --input <json>        Update input parameters"
  echo "  --description <desc>  Update description"
  echo "  --enabled <bool>      Enable/disable (true/false)"
  echo ""
  echo "Examples:"
  echo "  $(basename "$0") abc-123 --daily --hour 10 --minute 0"
  echo "  $(basename "$0") abc-123 --enabled false"
  echo "  $(basename "$0") abc-123 --timezone \"America/New_York\""
  exit 1
}

# Parse arguments
SCHEDULE_ID=""
UPDATE_SCHEDULE_TYPE=""
UPDATE_TIMEZONE=""
UPDATE_INPUT=""
UPDATE_DESCRIPTION=""
UPDATE_ENABLED=""

# Schedule-specific parameters
MINUTES=""
HOUR=""
MINUTE=""
DAYS=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --hourly)
      UPDATE_SCHEDULE_TYPE="hourly"
      shift
      ;;
    --daily)
      UPDATE_SCHEDULE_TYPE="daily"
      shift
      ;;
    --weekly)
      UPDATE_SCHEDULE_TYPE="weekly"
      shift
      ;;
    --monthly)
      UPDATE_SCHEDULE_TYPE="monthly"
      shift
      ;;
    --minutes)
      MINUTES="$2"
      shift 2
      ;;
    --hour)
      HOUR="$2"
      shift 2
      ;;
    --minute)
      MINUTE="$2"
      shift 2
      ;;
    --days)
      DAYS="$2"
      shift 2
      ;;
    --timezone)
      UPDATE_TIMEZONE="$2"
      shift 2
      ;;
    --input)
      UPDATE_INPUT="$2"
      shift 2
      ;;
    --description)
      UPDATE_DESCRIPTION="$2"
      shift 2
      ;;
    --enabled)
      UPDATE_ENABLED="$2"
      shift 2
      ;;
    -h|--help)
      usage
      ;;
    -*)
      autopi_error "Unknown option: $1"
      exit 1
      ;;
    *)
      if [[ -z "$SCHEDULE_ID" ]]; then
        SCHEDULE_ID="$1"
      else
        autopi_error "Unexpected argument: $1"
        exit 1
      fi
      shift
      ;;
  esac
done

# Validate required arguments
if [[ -z "$SCHEDULE_ID" ]]; then
  autopi_error "Schedule ID is required"
  usage
fi

# Check that at least one update option is provided
if [[ -z "$UPDATE_SCHEDULE_TYPE" && -z "$UPDATE_TIMEZONE" && -z "$UPDATE_INPUT" && -z "$UPDATE_DESCRIPTION" && -z "$UPDATE_ENABLED" ]]; then
  autopi_error "At least one update option is required"
  usage
fi

# Validate input JSON if provided
if [[ -n "$UPDATE_INPUT" ]]; then
  if ! echo "$UPDATE_INPUT" | jq . >/dev/null 2>&1; then
    autopi_error "Invalid JSON input: $UPDATE_INPUT"
    exit 1
  fi
fi

# Validate enabled value if provided
if [[ -n "$UPDATE_ENABLED" ]]; then
  if [[ "$UPDATE_ENABLED" != "true" && "$UPDATE_ENABLED" != "false" ]]; then
    autopi_error "Invalid enabled value: must be 'true' or 'false'"
    exit 1
  fi
fi

autopi_info "Updating schedule: $SCHEDULE_ID"

# Get current schedule and ETag
RESPONSE=$(autopi_get_with_headers "/scheduled-executions/$SCHEDULE_ID")

ETAG=$(echo "$RESPONSE" | grep -i "^etag:" | awk '{print $2}' | tr -d '\r\n')
BODY=$(echo "$RESPONSE" | sed -n '/^$/,$p' | tail -n +2)

if [[ -z "$ETAG" ]]; then
  # Check if schedule exists
  HTTP_CODE=$(autopi_check "/scheduled-executions/$SCHEDULE_ID")
  if [[ "$HTTP_CODE" == "404" ]]; then
    autopi_error "Schedule not found: $SCHEDULE_ID"
  else
    autopi_error "Could not get ETag for schedule"
  fi
  exit 1
fi

autopi_info "Using ETag: $ETAG"

# Apply updates to current schedule
UPDATED="$BODY"

# Update schedule type if specified
if [[ -n "$UPDATE_SCHEDULE_TYPE" ]]; then
  autopi_info "  Updating schedule type: $UPDATE_SCHEDULE_TYPE"

  # Get current timezone and deadline
  CURRENT_TZ=$(echo "$BODY" | jq -r '.schedule.timeZone // "Etc/UTC"')
  CURRENT_DEADLINE=$(echo "$BODY" | jq -r '.schedule.deadlineInMinutes // 5')

  # Use update timezone if provided, otherwise keep current
  TZ_TO_USE="${UPDATE_TIMEZONE:-$CURRENT_TZ}"

  case "$UPDATE_SCHEDULE_TYPE" in
    hourly)
      if [[ -z "$MINUTES" ]]; then
        autopi_error "Hourly schedule requires --minutes"
        exit 1
      fi
      local minutes_array
      minutes_array=$(echo "$MINUTES" | tr ',' '\n' | jq -R 'tonumber' | jq -s '.')
      NEW_SCHEDULE=$(jq -n \
        --arg tz "$TZ_TO_USE" \
        --argjson deadline "$CURRENT_DEADLINE" \
        --argjson minutes "$minutes_array" \
        '{
          timeZone: $tz,
          deadlineInMinutes: $deadline,
          hourly: {minutes: $minutes}
        }')
      ;;
    daily)
      if [[ -z "$HOUR" || -z "$MINUTE" ]]; then
        autopi_error "Daily schedule requires --hour and --minute"
        exit 1
      fi
      # API expects hours and minutes as arrays
      NEW_SCHEDULE=$(jq -n \
        --arg tz "$TZ_TO_USE" \
        --argjson deadline "$CURRENT_DEADLINE" \
        --argjson hour "$HOUR" \
        --argjson minute "$MINUTE" \
        '{
          timeZone: $tz,
          deadlineInMinutes: $deadline,
          daily: {hours: [$hour], minutes: [$minute]}
        }')
      ;;
    weekly)
      if [[ -z "$DAYS" || -z "$HOUR" || -z "$MINUTE" ]]; then
        autopi_error "Weekly schedule requires --days, --hour, and --minute"
        exit 1
      fi
      local days_array
      days_array=$(echo "$DAYS" | tr ',' '\n' | jq -R 'tonumber' | jq -s '.')
      # API expects hours and minutes as arrays
      NEW_SCHEDULE=$(jq -n \
        --arg tz "$TZ_TO_USE" \
        --argjson deadline "$CURRENT_DEADLINE" \
        --argjson days "$days_array" \
        --argjson hour "$HOUR" \
        --argjson minute "$MINUTE" \
        '{
          timeZone: $tz,
          deadlineInMinutes: $deadline,
          weekly: {days: $days, hours: [$hour], minutes: [$minute]}
        }')
      ;;
    monthly)
      if [[ -z "$DAYS" || -z "$HOUR" || -z "$MINUTE" ]]; then
        autopi_error "Monthly schedule requires --days, --hour, and --minute"
        exit 1
      fi
      local days_array
      days_array=$(echo "$DAYS" | tr ',' '\n' | jq -R 'tonumber' | jq -s '.')
      # API expects hours and minutes as arrays
      NEW_SCHEDULE=$(jq -n \
        --arg tz "$TZ_TO_USE" \
        --argjson deadline "$CURRENT_DEADLINE" \
        --argjson days "$days_array" \
        --argjson hour "$HOUR" \
        --argjson minute "$MINUTE" \
        '{
          timeZone: $tz,
          deadlineInMinutes: $deadline,
          monthly: {days: $days, hours: [$hour], minutes: [$minute]}
        }')
      ;;
  esac

  UPDATED=$(echo "$UPDATED" | jq --argjson schedule "$NEW_SCHEDULE" '.schedule = $schedule')

elif [[ -n "$UPDATE_TIMEZONE" ]]; then
  # Just update timezone, keep existing schedule type
  autopi_info "  Updating timezone: $UPDATE_TIMEZONE"
  UPDATED=$(echo "$UPDATED" | jq --arg tz "$UPDATE_TIMEZONE" '.schedule.timeZone = $tz')
fi

if [[ -n "$UPDATE_INPUT" ]]; then
  autopi_info "  Updating input"
  UPDATED=$(echo "$UPDATED" | jq --argjson input "$UPDATE_INPUT" '.inputValues = $input')
fi

if [[ -n "$UPDATE_DESCRIPTION" ]]; then
  autopi_info "  Updating description: $UPDATE_DESCRIPTION"
  UPDATED=$(echo "$UPDATED" | jq --arg desc "$UPDATE_DESCRIPTION" '.description = $desc')
fi

if [[ -n "$UPDATE_ENABLED" ]]; then
  autopi_info "  Updating enabled: $UPDATE_ENABLED"
  UPDATED=$(echo "$UPDATED" | jq --argjson enabled "$UPDATE_ENABLED" '.enabled = $enabled')
fi

# Remove read-only fields that shouldn't be sent in update
UPDATED=$(echo "$UPDATED" | jq 'del(.id, .state, .owner)')

# PUT with ETag
RESULT=$(curl -s -w "\n%{http_code}" -X PUT \
  -u "$AUTOPI_USER:$AUTOPI_PASS" \
  -H "Content-Type: application/json" \
  -H "If-Match: $ETAG" \
  -d "$UPDATED" \
  "$AUTOPI_BASE_URL/scheduled-executions/$SCHEDULE_ID")

HTTP_CODE=$(echo "$RESULT" | tail -n 1)
RESULT_BODY=$(echo "$RESULT" | sed '$d')

if [[ "$HTTP_CODE" == "200" ]]; then
  autopi_ok "Updated schedule: $SCHEDULE_ID"
  echo "$RESULT_BODY" | jq .
elif [[ "$HTTP_CODE" == "412" ]]; then
  autopi_error "ETag mismatch - schedule was modified by another user. Please retry."
  exit 1
elif [[ "$HTTP_CODE" == "400" ]]; then
  autopi_error "Bad request - invalid schedule configuration or input"
  echo "$RESULT_BODY" | jq . 2>/dev/null || echo "$RESULT_BODY"
  exit 1
else
  autopi_error "Failed to update schedule (HTTP $HTTP_CODE)"
  echo "$RESULT_BODY" | jq . 2>/dev/null || echo "$RESULT_BODY"
  exit 1
fi
