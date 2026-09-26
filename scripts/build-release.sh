#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
GOARM_VERSION="${1:-7}"

if [[ "$GOARM_VERSION" != "6" && "$GOARM_VERSION" != "7" ]]; then
  echo "Usage: $0 [6|7]" >&2
  echo "Use 7 for ARMv7 Raspberry Pi systems, or 6 for ARMv6 systems." >&2
  exit 2
fi

for command in go npm; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "Error: $command is required to build a release." >&2
    exit 1
  fi
done

cd "$PROJECT_DIR"
npm run build

EMBED_DIR="$PROJECT_DIR/web/static/dist"
RELEASE_DIR="$PROJECT_DIR/release"
rm -rf "$EMBED_DIR"
mkdir -p "$EMBED_DIR" "$RELEASE_DIR"
cp -R "$PROJECT_DIR/dist/." "$EMBED_DIR/"
cleanup() { rm -rf "$EMBED_DIR"; }
trap cleanup EXIT

GOOS=linux GOARCH=arm GOARM="$GOARM_VERSION" CGO_ENABLED=0 \
  go build -trimpath -ldflags="-s -w" -o "$RELEASE_DIR/pricefollower" ./cmd/pricefollower

echo "Built $RELEASE_DIR/pricefollower for Linux ARMv$GOARM_VERSION with the frontend embedded."
