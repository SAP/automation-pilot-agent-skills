#!/bin/sh
# Real marketplace installation in disposable Claude state; no tenant calls.
set -eu
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
trap 'exit 1' HUP INT TERM
. "$ROOT/ci/plugin/fixture.sh"
install_plugin_fixture "$ROOT" "$WORK"
cd "$WORK/workspace"
# Read only the disposable install registry, never the caller's settings.
INSTALLED="$(find "$CLAUDE_CONFIG_DIR/plugins/cache" -name plugin.json -path '*/.claude-plugin/plugin.json' -print)"
[ -n "$INSTALLED" ] || { echo "plugin installation produced no cached manifest" >&2; exit 1; }
PLUGIN="$(dirname "$(dirname "$INSTALLED")")"
go run "$ROOT/ci/plugin/main.go" check "$PLUGIN"
# The fixture removed the marketplace source. Only OS hash facilities
# remain on PATH; Go/Python, source, and an unrelated PATH autopi are unnecessary.
AUTOPI="$PLUGIN/skills/automation-pilot-cli/scripts/autopi"
env -i PATH=/usr/bin:/bin "$AUTOPI" version
env -i PATH=/usr/bin:/bin "$AUTOPI" help > "$WORK/help.txt"
env -i PATH=/usr/bin:/bin "$AUTOPI" lint "$PLUGIN/skills/automation-pilot-command-authoring/examples/"*.command.json
env -i PATH=/usr/bin:/bin "$AUTOPI" executions trigger --command-id c:C:1 --print-request
code=0
env -i PATH=/usr/bin:/bin "$AUTOPI" connection preflight > "$WORK/preflight.json" || code=$?
[ "$code" -eq 2 ] || { echo "preflight did not diagnose missing configuration" >&2; exit 1; }
target_os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$(uname -m)" in
  arm64|aarch64) target_arch=arm64 ;;
  *) target_arch=amd64 ;;
esac
printf '\ntampered\n' >> "$PLUGIN/skills/automation-pilot-cli/scripts/bin/$target_os-$target_arch/autopi"
if env -i PATH=/usr/bin:/bin "$AUTOPI" version > "$WORK/tampered.txt" 2>&1; then
  echo "installed plugin accepted a tampered binary" >&2; exit 1
fi
grep -q 'verification failed' "$WORK/tampered.txt"
echo "Installed plugin works without build tools and rejects tampering"
