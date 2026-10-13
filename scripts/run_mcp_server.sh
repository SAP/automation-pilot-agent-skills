#!/usr/bin/env bash
#
# Start the SAP Automation Pilot MCP server locally over HTTP.
#
# Required environment variables:
#   AUTOMATION_PILOT_HOSTNAME  Automation Pilot host (e.g. my-tenant.automation.cloud.sap)
#   AUTOMATION_PILOT_TOKEN     Bearer token used to authenticate API calls
#
# Optional environment variables:
#   MCP_HOST  Address to bind the HTTP server to (default: 127.0.0.1)
#   MCP_PORT  Port to listen on                   (default: 8000)
#
# Usage:
#   export AUTOMATION_PILOT_HOSTNAME="my-tenant.automation.cloud.sap"
#   export AUTOMATION_PILOT_TOKEN="..."
#   ./scripts/run_mcp_server.sh

set -euo pipefail

if [[ -z "${AUTOMATION_PILOT_HOSTNAME:-}" ]]; then
  echo "Error: AUTOMATION_PILOT_HOSTNAME is not set." >&2
  exit 1
fi

if [[ -z "${AUTOMATION_PILOT_TOKEN:-}" ]]; then
  echo "Error: AUTOMATION_PILOT_TOKEN is not set." >&2
  exit 1
fi

MCP_HOST="${MCP_HOST:-127.0.0.1}"
MCP_PORT="${MCP_PORT:-8000}"

# Resolve the mcp/ directory relative to this script so it works from anywhere.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MCP_DIR="$(cd "${SCRIPT_DIR}/../mcp" && pwd)"

echo "Starting Automation Pilot MCP server on http://${MCP_HOST}:${MCP_PORT} ..."

exec uv run --locked --project "${MCP_DIR}" fastmcp run "${MCP_DIR}/server.py" \
  --transport http \
  --host "${MCP_HOST}" \
  --port "${MCP_PORT}"
