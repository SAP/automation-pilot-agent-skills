#!/usr/bin/env bash
# SAP Automation Pilot - Create Scheduled Execution
#
# Usage:
#   autopi-sched-create.sh <command-id> [options]
#
# Schedule Types (choose one):
#   --hourly --minutes <list>          Hourly at specified minutes (e.g., "0,15,30,45")
#   --daily --hour <h> --minute <m>    Daily at specified time
#   --weekly --days <list> --hour <h> --minute <m>   Weekly on specified days (1=Mon, 7=Sun)
#   --monthly --days <list> --hour <h> --minute <m>  Monthly on specified days of month
#
# Options:
#   --timezone <tz>     Time zone (default: Etc/UTC)
#   --input <json>      Input parameters as JSON
#   --description <desc> Description
#   --deadline <min>    Deadline in minutes (default: 5)
#   --enabled           Create enabled (default)
#   --disabled          Create disabled
#   --tag <key>=<value> Add tag (can be repeated)
#
# Examples:
#   autopi-sched-create.sh "my-catalog:MyCommand:1" --daily --hour 9 --minute 0
#   autopi-sched-create.sh "my-catalog:MyCommand:1" --hourly --minutes "0,30" --timezone "Europe/Berlin"
#   autopi-sched-create.sh "my-catalog:MyCommand:1" --weekly --days "1,3,5" --hour 8 --minute 0

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-sched-config.sh"

usage() {
  echo "Usage: $(basename "$0") <command-id> [options]"
  echo ""
  echo "Schedule Types (choose one):"
  echo "  --hourly --minutes <list>          Hourly at specified minutes (e.g., '0,15,30,45')"
  echo "  --daily --hour <h> --minute <m>    Daily at specified time"
  echo "  --weekly --days <list> --hour <h> --minute <m>   Weekly on days (1=Mon, 7=Sun)"
  echo "  --monthly --days <list> --hour <h> --minute <m>  Monthly on days of month"
  echo ""
  echo "Options:"
  echo "  --timezone <tz>       Time zone (default: Etc/UTC)"
  echo "  --input <json>        Input parameters as JSON"
  echo "  --description <desc>  Description"
  echo "  --deadline <min>      Deadline in minutes (default: 5)"
  echo "  --enabled             Create enabled (default)"
  echo "  --disabled            Create disabled"
  echo "  --tag <key>=<value>   Add tag (can be repeated)"
  echo ""
  echo "Examples:"
  echo "  $(basename "$0") \"my-catalog:MyCommand:1\" --daily --hour 9 --minute 0"
  echo "  $(basename "$0") \"my-catalog:MyCommand:1\" --hourly --minutes \"0,30\""
  echo "  $(basename "$0") \"my-catalog:MyCommand:1\" --weekly --days \"1,3,5\" --hour 8 --minute 0"
  exit 1
}

# Parse arguments
COMMAND_ID=""
SCHEDULE_TYPE=""
TIMEZONE="Etc/UTC"
INPUT="{}"
DESCRIPTION=""
DEADLINE=5
ENABLED="true"
TAGS="{}"

# Schedule-specific parameters
MINUTES=""
HOUR=""
MINUTE=""
DAYS=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --hourly)
      SCHEDULE_TYPE="hourly"
      shift
      ;;
    --daily)
      SCHEDULE_TYPE="daily"
      shift
      ;;
    --weekly)
      SCHEDULE_TYPE="weekly"
      shift
      ;;
    --monthly)
      SCHEDULE_TYPE="monthly"
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
      TIMEZONE="$2"
      shift 2
      ;;
    --input)
      INPUT="$2"
      shift 2
      ;;
    --description)
      DESCRIPTION="$2"
      shift 2
      ;;
    --deadline)
      DEADLINE="$2"
      shift 2
      ;;
    --enabled)
      ENABLED="true"
      shift
      ;;
    --disabled)
      ENABLED="false"
      shift
      ;;
    --tag)
      TAG_KV="$2"
      TAG_KEY="${TAG_KV%%=*}"
      TAG_VAL="${TAG_KV#*=}"
      TAGS=$(echo "$TAGS" | jq --arg k "$TAG_KEY" --arg v "$TAG_VAL" '. + {($k): $v}')
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
      if [[ -z "$COMMAND_ID" ]]; then
        COMMAND_ID="$1"
      else
        autopi_error "Unexpected argument: $1"
        exit 1
      fi
      shift
      ;;
  esac
done

# Validate required arguments
if [[ -z "$COMMAND_ID" ]]; then
  autopi_error "Command ID is required"
  usage
fi

if [[ -z "$SCHEDULE_TYPE" ]]; then
  autopi_error "Schedule type is required (--hourly, --daily, --weekly, or --monthly)"
  usage
fi

# Validate input is valid JSON
if ! echo "$INPUT" | jq . >/dev/null 2>&1; then
  autopi_error "Invalid JSON input: $INPUT"
  exit 1
fi

# Build schedule object based on type
build_schedule() {
  local schedule_obj

  case "$SCHEDULE_TYPE" in
    hourly)
      if [[ -z "$MINUTES" ]]; then
        autopi_error "Hourly schedule requires --minutes"
        exit 1
      fi
      # Convert comma-separated minutes to JSON array
      local minutes_array
      minutes_array=$(echo "$MINUTES" | tr ',' '\n' | jq -R 'tonumber' | jq -s '.')
      schedule_obj=$(jq -n \
        --arg tz "$TIMEZONE" \
        --argjson deadline "$DEADLINE" \
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
      schedule_obj=$(jq -n \
        --arg tz "$TIMEZONE" \
        --argjson deadline "$DEADLINE" \
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
      schedule_obj=$(jq -n \
        --arg tz "$TIMEZONE" \
        --argjson deadline "$DEADLINE" \
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
      schedule_obj=$(jq -n \
        --arg tz "$TIMEZONE" \
        --argjson deadline "$DEADLINE" \
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

  echo "$schedule_obj"
}

SCHEDULE=$(build_schedule)

# Build request body
REQUEST=$(jq -n \
  --arg commandId "$COMMAND_ID" \
  --argjson schedule "$SCHEDULE" \
  --argjson enabled "$ENABLED" \
  --argjson inputValues "$INPUT" \
  --argjson tags "$TAGS" \
  --arg description "$DESCRIPTION" \
  '{
    commandId: $commandId,
    schedule: $schedule,
    enabled: $enabled,
    inputValues: $inputValues,
    tags: $tags
  } + (if $description != "" then {description: $description} else {} end)'
)

autopi_info "Creating scheduled execution..."
autopi_info "  Command: $COMMAND_ID"
autopi_info "  Schedule Type: $SCHEDULE_TYPE"
autopi_info "  Time Zone: $TIMEZONE"
autopi_info "  Enabled: $ENABLED"

# Make API call
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -u "$AUTOPI_USER:$AUTOPI_PASS" \
  -H "Content-Type: application/json" \
  -d "$REQUEST" \
  "$AUTOPI_BASE_URL/scheduled-executions")

HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [[ "$HTTP_CODE" == "201" ]]; then
  SCHEDULE_ID=$(echo "$BODY" | jq -r '.id')
  autopi_ok "Created schedule: $SCHEDULE_ID"
  echo "$BODY" | jq .
elif [[ "$HTTP_CODE" == "400" ]]; then
  autopi_error "Bad request - invalid schedule configuration or input"
  echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
  exit 1
elif [[ "$HTTP_CODE" == "409" ]]; then
  autopi_error "Conflict - command not found or invalid state"
  echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
  exit 1
else
  autopi_error "Failed to create schedule (HTTP $HTTP_CODE)"
  echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
  exit 1
fi
