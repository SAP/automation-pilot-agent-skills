#!/usr/bin/env bash
# SAP Automation Pilot Content API - Batch Sync
# Sync all .command.json files from a directory
#
# Usage: autopi-sync.sh [directory]

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/autopi-config.sh"

DIR="${1:-.}"

if [[ ! -d "$DIR" ]]; then
  autopi_error "Directory not found: $DIR"
  exit 1
fi

autopi_info "Syncing commands from: $DIR"
echo ""

# Count files
FILES=("$DIR"/*.command.json)
if [[ ! -f "${FILES[0]}" ]]; then
  autopi_warn "No .command.json files found in: $DIR"
  exit 0
fi

TOTAL=${#FILES[@]}
SUCCESS=0
FAILED=0

for file in "${FILES[@]}"; do
  [[ -f "$file" ]] || continue

  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

  if "$SCRIPT_DIR/autopi-deploy.sh" "$file"; then
    ((SUCCESS++))
  else
    ((FAILED++))
  fi

  echo ""
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
autopi_info "Sync complete"
echo "  Total:   $TOTAL"
echo "  Success: $SUCCESS"
echo "  Failed:  $FAILED"

if [[ "$FAILED" -gt 0 ]]; then
  exit 1
fi
