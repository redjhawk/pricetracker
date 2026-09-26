#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
RELEASE_BINARY="$PROJECT_DIR/release/pricefollower"
INSTALL_SCRIPT="$SCRIPT_DIR/install-pricefollower.sh"

usage() {
  echo "Usage: $0 user@host:/existing/remote/directory/" >&2
  echo "Example: $0 pi@raspberry-pi:/tmp/" >&2
}

if [[ $# -ne 1 ]]; then
  usage
  exit 2
fi

DESTINATION="$1"

if [[ ! -f "$RELEASE_BINARY" ]]; then
  echo "Error: $RELEASE_BINARY does not exist. Build a release first with './scripts/build-release.sh [6|7]'." >&2
  exit 1
fi

if [[ ! -f "$INSTALL_SCRIPT" ]]; then
  echo "Error: installer script not found: $INSTALL_SCRIPT" >&2
  exit 1
fi

if ! command -v rsync >/dev/null 2>&1; then
  echo "Error: rsync is required locally. Install rsync and try again." >&2
  exit 1
fi

if [[ "$DESTINATION" != *:* || "$DESTINATION" == *::* ]]; then
  echo "Error: destination must use SSH rsync format, for example pi@raspberry-pi:/tmp/" >&2
  usage
  exit 2
fi

echo "Copying the PriceFollower release binary and installer to $DESTINATION"
rsync --archive --compress --human-readable --rsh=ssh \
  "$RELEASE_BINARY" "$INSTALL_SCRIPT" "$DESTINATION"
echo "Copy complete."
