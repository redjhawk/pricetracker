#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
# Linux x86 64-bit is the only target; the optional "amd64" argument is kept for existing callers.
case "${1:-amd64}" in
  amd64) ;;
  *) echo "Usage: $0 [amd64]" >&2; exit 2 ;;
esac

for command in go npm; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "Error: $command is required to build a release." >&2
    exit 1
  fi
done

cd "$PROJECT_DIR"
TAG="$(git describe --tags --abbrev=0 2>/dev/null || true)"
VERSION="${PRICEFOLLOWER_VERSION:-${TAG:+$TAG-dev}}"
VERSION="${VERSION:-dev}"
npm run build

EMBED_DIR="$PROJECT_DIR/web/static/dist"
RELEASE_DIR="$PROJECT_DIR/release"
rm -rf "$EMBED_DIR"
mkdir -p "$EMBED_DIR" "$RELEASE_DIR"
cp -R "$PROJECT_DIR/dist/." "$EMBED_DIR/"
cleanup() { rm -rf "$EMBED_DIR"; }
trap cleanup EXIT

env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags="-s -w -X pricefollower.local/internal/version.Version=$VERSION" -o "$RELEASE_DIR/pricefollower" ./cmd/pricefollower

echo "Built $RELEASE_DIR/pricefollower for Linux amd64 (version $VERSION) with the frontend embedded."
