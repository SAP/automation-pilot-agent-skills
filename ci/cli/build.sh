#!/bin/sh
# Usage from any working directory: ci/cli/build.sh [version]
# Requires the Go toolchain pinned in cli/.go-version. Runtime binaries need no Go.
set -eu
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MODULE_DIR="$(cd "$SCRIPT_DIR/../../cli" && pwd)"
cd "$MODULE_DIR"
PINNED="$(cat .go-version)"
ACTUAL="$(go env GOVERSION)"
[ "$ACTUAL" = "go$PINNED" ] || { echo "build requires Go $PINNED (found $ACTUAL)" >&2; exit 1; }
REVISION="$(git rev-parse HEAD)"
VERSION="${1:-$(git describe --tags --always --dirty)}"
case "$VERSION" in
  *[!a-zA-Z0-9._+-]*|'') echo "invalid version" >&2; exit 1 ;;
esac
SOURCE_SHA="$(go run "$SCRIPT_DIR/buildmeta/main.go" source "$MODULE_DIR")"
VERSION="$VERSION+source.$SOURCE_SHA"
BIN_DIR="$MODULE_DIR/../skills/automation-pilot-cli/scripts/bin"
LOCK="$BIN_DIR.lock"
mkdir "$LOCK" 2>/dev/null || { echo "another bundle build is active (or a stale bin.lock needs review)" >&2; exit 1; }
# Stage beside the destination so replacement uses same-filesystem renames.
# Failed builds and verification leave the existing complete bundle untouched.
STAGE=""
BACKUP=""
cleanup() {
  if [ -n "$BACKUP" ] && [ -d "$BACKUP" ] && [ ! -d "$BIN_DIR" ]; then mv "$BACKUP" "$BIN_DIR"; fi
  if [ -n "$STAGE" ]; then rm -rf "$STAGE"; fi
  rmdir "$LOCK"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
STAGE="$(mktemp -d "$BIN_DIR.stage.XXXXXX")"
BACKUP="$STAGE.previous"
TARGETS="darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64"
echo "building autopi $VERSION"
for target in $TARGETS; do
  target_os="${target%/*}"
  target_arch="${target#*/}"
  out="$STAGE/${target_os}-${target_arch}/autopi"
  [ "$target_os" = "windows" ] && out="${out}.exe"
  mkdir -p "$(dirname "$out")"
  CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" \
    go build -trimpath -buildvcs=false -ldflags "-s -w -X main.Version=$VERSION" -o "$out" .
done
go run "$SCRIPT_DIR/buildmeta/main.go" manifest "$MODULE_DIR" "$STAGE" "$VERSION" "$REVISION"
go run "$SCRIPT_DIR/buildmeta/main.go" verify "$MODULE_DIR" "$STAGE"
if [ -d "$BIN_DIR" ]; then mv "$BIN_DIR" "$BACKUP"; fi
mv "$STAGE" "$BIN_DIR"
rm -rf "$BACKUP"
echo "built five targets; source digest and checksums recorded"
