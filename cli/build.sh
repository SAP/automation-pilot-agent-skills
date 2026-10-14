#!/bin/sh
# Cross-compiles the autopi CLI for all supported platforms and writes the static
# binaries into the automation-pilot-cli skill. Requires a Go toolchain (build-time
# only — the committed binaries have no runtime dependencies).
#
# Usage: ./build.sh [version]
set -eu

VERSION="${1:-$(git describe --tags --always 2>/dev/null || echo dev)}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="$SCRIPT_DIR/../skills/automation-pilot-cli/scripts/bin"

# GOOS/GOARCH pairs → output subdirectory name.
TARGETS="darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64"

echo "building autopi $VERSION"
for t in $TARGETS; do
  os="${t%/*}"
  arch="${t#*/}"
  out="$BIN_DIR/${os}-${arch}/autopi"
  [ "$os" = "windows" ] && out="${out}.exe"
  mkdir -p "$(dirname "$out")"
  echo "  $os/$arch -> $out"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -trimpath -ldflags "-s -w -X main.Version=$VERSION" -o "$out" "$SCRIPT_DIR"
done
echo "done"
