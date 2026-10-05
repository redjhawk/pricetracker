#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  echo "Usage: $0 [--admin-password] user@host [remote-directory]" >&2
  echo "The remote staging directory defaults to pricefollower in the SSH user's home." >&2
  echo "--admin-password creates the admin account, or resets its password, after installing and prints the new password." >&2
}

ADMIN_PASSWORD=false
if [[ "${1-}" == "--admin-password" ]]; then
  ADMIN_PASSWORD=true
  shift
fi

if [[ $# -lt 1 || $# -gt 2 ]]; then
  usage
  exit 2
fi

TARGET="$1"
REMOTE_DIR="${2-pricefollower}"
if [[ ! "$TARGET" =~ ^([a-zA-Z0-9_][a-zA-Z0-9_.-]*@)?[a-zA-Z0-9][a-zA-Z0-9_.-]*$ ]]; then
  echo "Error: use a host or SSH alias, optionally prefixed by username@; ports and IPv6 require an SSH alias." >&2
  usage
  exit 2
fi
if [[ ! "$REMOTE_DIR" =~ ^[a-zA-Z0-9_./][a-zA-Z0-9_./-]*$ ]]; then
  echo "Error: staging paths accept only ASCII letters, digits, underscores, dots, hyphens and slashes." >&2
  usage
  exit 2
fi
while [[ "$REMOTE_DIR" == */ ]]; do
  REMOTE_DIR="${REMOTE_DIR%/}"
done
if [[ -z "$REMOTE_DIR" || "$REMOTE_DIR" == . || "/$REMOTE_DIR/" == */../* ]]; then
  echo "Error: choose a staging directory other than root, dot or a path containing .. segments." >&2
  exit 2
fi

# Normalize harmless separators before checking protected absolute paths.
CHECK_DIR="$REMOTE_DIR"
while [[ "$CHECK_DIR" == *//* || "$CHECK_DIR" == */./* ]]; do
  CHECK_DIR="${CHECK_DIR//\/\//\/}"
  CHECK_DIR="${CHECK_DIR//\/\.\//\/}"
done
CHECK_DIR="${CHECK_DIR%/.}"
case "$CHECK_DIR" in
  ''|/|.)
    echo "Error: choose a staging directory other than root or dot." >&2
    exit 2
    ;;
  /opt/pricefollower|/opt/pricefollower/*|/var/lib/pricefollower|/var/lib/pricefollower/*)
    echo "Error: staging must be separate from /opt/pricefollower and /var/lib/pricefollower." >&2
    exit 2
    ;;
esac

for command in go npm ssh rsync; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "Error: $command is required for deployment." >&2
    exit 1
  fi
done

cd "$PROJECT_DIR"
echo "Building the ARMv6 release…"
"$PROJECT_DIR/scripts/build-release.sh" 6
echo "Creating remote staging directory $REMOTE_DIR…"
ssh "$TARGET" "mkdir -p -- '$REMOTE_DIR'"
echo "Uploading the release and installer…"
"$PROJECT_DIR/scripts/copy-dist.sh" "$TARGET:$REMOTE_DIR/"
echo "Installing and checking the remote service…"
ssh -t "$TARGET" "cd -- '$REMOTE_DIR' && sudo bash ./install-pricefollower.sh ./pricefollower && systemctl is-active --quiet pricefollower"
if [[ "$ADMIN_PASSWORD" == true ]]; then
  echo "Setting the administrator password…"
  ssh -t "$TARGET" "sudo -u pricefollower PRICEFOLLOWER_DATA_DIR=/var/lib/pricefollower /opt/pricefollower/pricefollower admin-password"
fi
echo "Deployment complete: pricefollower is active on $TARGET."
