#!/usr/bin/env bash
# SAP Automation Pilot Scheduled Executions API - Configuration Loader
# Source this file to load config and common functions
#
# Usage: source autopi-sched-config.sh
#
# Required environment variables:
#   AUTOPI_HOSTNAME        - API hostname (e.g., emea.autopilot.cloud.sap)
#   AUTOPI_USERNAME        - Username
#   AUTOPI_PASSWORD        - Password
#   AUTOPI_DEFAULT_CATALOG - Default catalog (e.g., mycommands-<<<TENANT_ID>>>)
#
# Exports:
#   AUTOPI_HOST     - API hostname (from AUTOPI_HOSTNAME)
#   AUTOPI_USER     - Username (from AUTOPI_USERNAME)
#   AUTOPI_PASS     - Password (from AUTOPI_PASSWORD)
#   AUTOPI_CATALOG  - Default catalog (from AUTOPI_DEFAULT_CATALOG)
#   AUTOPI_BASE_URL - Full API base URL
#
# Functions:
#   autopi_api <method> <endpoint> [data]     - Make API call
#   autopi_api_with_etag <method> <endpoint> <etag> [data] - API call with If-Match
#   autopi_get_etag <endpoint>                - Get ETag for resource
#   autopi_check <endpoint>                   - Check HTTP status code

# Check required environment variables
_autopi_check_env() {
  local missing=()
  [[ -z "$AUTOPI_HOSTNAME" ]] && missing+=("AUTOPI_HOSTNAME")
  [[ -z "$AUTOPI_USERNAME" ]] && missing+=("AUTOPI_USERNAME")
  [[ -z "$AUTOPI_PASSWORD" ]] && missing+=("AUTOPI_PASSWORD")
  [[ -z "$AUTOPI_DEFAULT_CATALOG" ]] && missing+=("AUTOPI_DEFAULT_CATALOG")

  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "Error: Missing required environment variables:" >&2
    for var in "${missing[@]}"; do
      echo "  - $var" >&2
    done
    echo "" >&2
    echo "Please set the following environment variables:" >&2
    echo "  export AUTOPI_HOSTNAME=\"emea.autopilot.cloud.sap\"" >&2
    echo "  export AUTOPI_USERNAME=\"your-username\"" >&2
    echo "  export AUTOPI_PASSWORD=\"your-password\"" >&2
    echo "  export AUTOPI_DEFAULT_CATALOG=\"mycommands-<<<TENANT_ID>>>\"" >&2
    return 1
  fi
  return 0
}

if ! _autopi_check_env; then
  return 1 2>/dev/null || exit 1
fi

export AUTOPI_HOST="$AUTOPI_HOSTNAME"
export AUTOPI_USER="$AUTOPI_USERNAME"
export AUTOPI_PASS="$AUTOPI_PASSWORD"
export AUTOPI_CATALOG="$AUTOPI_DEFAULT_CATALOG"
export AUTOPI_BASE_URL="https://$AUTOPI_HOST/api/v1"

# Make an API call with inline data
# Usage: autopi_api GET "/scheduled-executions"
#        autopi_api POST "/scheduled-executions" '{"commandId":"..."}'
autopi_api() {
  local method="$1"
  local endpoint="$2"
  local data="$3"

  if [[ -n "$data" ]]; then
    curl -s -X "$method" \
      -u "$AUTOPI_USER:$AUTOPI_PASS" \
      -H "Content-Type: application/json" \
      -d "$data" \
      "$AUTOPI_BASE_URL$endpoint"
  else
    curl -s -X "$method" \
      -u "$AUTOPI_USER:$AUTOPI_PASS" \
      "$AUTOPI_BASE_URL$endpoint"
  fi
}

# Make an API call with ETag header
# Usage: autopi_api_with_etag PUT "/scheduled-executions/id" "etag-value" '{"data":"..."}'
#        autopi_api_with_etag DELETE "/scheduled-executions/id" "etag-value"
autopi_api_with_etag() {
  local method="$1"
  local endpoint="$2"
  local etag="$3"
  local data="$4"

  if [[ -n "$data" ]]; then
    curl -s -X "$method" \
      -u "$AUTOPI_USER:$AUTOPI_PASS" \
      -H "Content-Type: application/json" \
      -H "If-Match: $etag" \
      -d "$data" \
      "$AUTOPI_BASE_URL$endpoint"
  else
    curl -s -X "$method" \
      -u "$AUTOPI_USER:$AUTOPI_PASS" \
      -H "If-Match: $etag" \
      "$AUTOPI_BASE_URL$endpoint"
  fi
}

# Get ETag for a resource
# Usage: etag=$(autopi_get_etag "/scheduled-executions/id")
autopi_get_etag() {
  local endpoint="$1"
  curl -s -I -u "$AUTOPI_USER:$AUTOPI_PASS" \
    "$AUTOPI_BASE_URL$endpoint" | \
    grep -i "^etag:" | awk '{print $2}' | tr -d '\r\n'
}

# Get response with headers (for extracting ETag and body)
# Usage: response=$(autopi_get_with_headers "/scheduled-executions/id")
autopi_get_with_headers() {
  local endpoint="$1"
  curl -s -i -u "$AUTOPI_USER:$AUTOPI_PASS" \
    "$AUTOPI_BASE_URL$endpoint"
}

# Check HTTP status code for an endpoint
# Usage: autopi_check "/scheduled-executions/uuid"
# Returns: HTTP status code (e.g., 200, 404)
autopi_check() {
  local endpoint="$1"
  curl -s -o /dev/null -w "%{http_code}" \
    -u "$AUTOPI_USER:$AUTOPI_PASS" \
    "$AUTOPI_BASE_URL$endpoint"
}

# Print colored output
autopi_info()  { echo -e "\033[0;34m[INFO]\033[0m $*"; }
autopi_ok()    { echo -e "\033[0;32m[OK]\033[0m $*"; }
autopi_warn()  { echo -e "\033[0;33m[WARN]\033[0m $*"; }
autopi_error() { echo -e "\033[0;31m[ERROR]\033[0m $*" >&2; }

# ============================================
# QUICK STATUS FUNCTIONS
# ============================================

# Quick status: counts of enabled/disabled schedules
# Usage: autopi_sched_status
autopi_sched_status() {
  autopi_info "Scheduled Executions Status:"
  echo ""

  local response=$(autopi_api GET "/scheduled-executions")

  local total=$(echo "$response" | jq 'length')
  local enabled=$(echo "$response" | jq '[.[] | select(.enabled == true)] | length')
  local disabled=$(echo "$response" | jq '[.[] | select(.enabled == false)] | length')

  echo "  Total:    $total"
  echo -e "  \033[0;32mEnabled:  $enabled\033[0m"
  echo -e "  \033[0;33mDisabled: $disabled\033[0m"
}

# Summary of all schedules with key details
# Usage: autopi_sched_summary
autopi_sched_summary() {
  autopi_info "All Scheduled Executions:"
  echo ""

  autopi_api GET "/scheduled-executions" | jq -r '.[] | "\(if .enabled then "[32m[ON][0m " else "[33m[OFF][0m" end) \(.description // .id | .[0:30]) - \(.commandId | split(":")[1])"'
}

# List only enabled schedules with details
# Usage: autopi_sched_enabled
autopi_sched_enabled() {
  autopi_info "Enabled Schedules:"
  echo ""

  autopi_api GET "/scheduled-executions" | jq -r '.[] | select(.enabled == true) | "  \(.description // "(no description)") - \(.commandId | split(":")[1])"'
}
