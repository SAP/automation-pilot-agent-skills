#!/usr/bin/env bash
# SAP Automation Pilot Catalog Explorer - Configuration Loader
# Source this file to load config and common functions
#
# Usage: source autopi-catalog-config.sh
#
# Required environment variables:
#   AUTOPI_HOSTNAME        - API hostname (e.g., emea.autopilot.cloud.sap)
#   AUTOPI_USERNAME        - Username
#   AUTOPI_PASSWORD        - Password
#
# Exports:
#   AUTOPI_HOST     - API hostname (from AUTOPI_HOSTNAME)
#   AUTOPI_USER     - Username (from AUTOPI_USERNAME)
#   AUTOPI_PASS     - Password (from AUTOPI_PASSWORD)
#   AUTOPI_BASE_URL - Full API base URL
#
# Functions:
#   autopi_api <method> <endpoint> [data] - Make API call
#   autopi_check <endpoint>               - Check HTTP status code

# Check required environment variables
_autopi_check_env() {
  local missing=()
  [[ -z "$AUTOPI_HOSTNAME" ]] && missing+=("AUTOPI_HOSTNAME")
  [[ -z "$AUTOPI_USERNAME" ]] && missing+=("AUTOPI_USERNAME")
  [[ -z "$AUTOPI_PASSWORD" ]] && missing+=("AUTOPI_PASSWORD")

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
export AUTOPI_BASE_URL="https://$AUTOPI_HOST/api/v1"

# Make an API call with inline data
# Usage: autopi_api GET "/catalogs"
#        autopi_api GET "/commands?catalog=http-sapcp"
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

# Check HTTP status code for an endpoint
# Usage: autopi_check "/catalogs"
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
