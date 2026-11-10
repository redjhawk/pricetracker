#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET="${1:-7}"

case "$TARGET" in
  6|7)   GO_ENV=(GOARCH=arm GOARM="$TARGET"); TARGET_NAME="ARMv$TARGET" ;;
  amd64) GO_ENV=(GOARCH=amd64); TARGET_NAME="x86-64 (amd64)" ;;
  *)
    echo "Usage: $0 [6|7|amd64]" >&2
    echo "Use 7 for ARMv7 Raspberry Pi systems, 6 for ARMv6 systems, or amd64 for x86 64-bit PCs." >&2
    exit 2
    ;;
esac

for command in go npm; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "Error: $command is required to build a release." >&2
    exit 1
  fi
done

cd "$PROJECT_DIR"
VERSION="${PRICEFOLLOWER_VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || echo dev)-dev}"
npm run build

EMBED_DIR="$PROJECT_DIR/web/static/dist"
RELEASE_DIR="$PROJECT_DIR/release"
rm -rf "$EMBED_DIR"
mkdir -p "$EMBED_DIR" "$RELEASE_DIR"
cp -R "$PROJECT_DIR/dist/." "$EMBED_DIR/"
cleanup() { rm -rf "$EMBED_DIR"; }
trap cleanup EXIT

env GOOS=linux "${GO_ENV[@]}" CGO_ENABLED=0 \
  go build -trimpath -ldflags="-s -w -X pricefollower.local/internal/version.Version=$VERSION" -o "$RELEASE_DIR/pricefollower" ./cmd/pricefollower

echo "Built $RELEASE_DIR/pricefollower for Linux $TARGET_NAME (version $VERSION) with the frontend embedded."
