#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
ASSET_NAME="pricefollower-linux-amd64"
export GIT_TERMINAL_PROMPT=0

fail() {
  echo "Error: $*" >&2
  exit 1
}

for command in go npm gh git; do
  if ! command -v "$command" >/dev/null 2>&1; then
    fail "$command is required to create a GitHub release."
  fi
done

cd "$PROJECT_DIR"

if [ -n "$(git status --porcelain)" ]; then
  fail "the working tree has uncommitted or untracked changes; commit or remove them first."
fi

if ! gh auth status >/dev/null 2>&1; then
  fail "the GitHub CLI is not authenticated; run 'gh auth login' first."
fi

if ! REPO="$(gh repo view "$(git remote get-url origin)" --json nameWithOwner --jq .nameWithOwner)"; then
  fail "cannot resolve the GitHub repository of the origin remote."
fi

if ! REMOTE_TAGS="$(git ls-remote --tags --refs origin | cut -f2)"; then
  fail "cannot reach the origin remote."
fi

if ! "$PROJECT_DIR/scripts/build-release.sh" amd64; then
  fail "building the amd64 release failed; no tag or release was created."
fi

DATE="$(date -u +%Y.%m.%d)"
N=1
while git rev-parse -q --verify "refs/tags/v$DATE-$N" >/dev/null ||
  grep -qxF "refs/tags/v$DATE-$N" <<<"$REMOTE_TAGS"; do
  N=$((N + 1))
done
TAG="v$DATE-$N"

ASSET_DIR="$(mktemp -d)"
trap 'rm -rf "$ASSET_DIR"' EXIT
cp "$PROJECT_DIR/release/pricefollower" "$ASSET_DIR/$ASSET_NAME"

git tag -a "$TAG" -m "$TAG" HEAD
if ! git push origin "$TAG"; then
  fail "pushing tag $TAG to origin failed; the tag exists locally (remove it with 'git tag -d $TAG')."
fi

if ! RELEASE_URL="$(gh release create "$TAG" --repo "$REPO" --verify-tag --title "$TAG" --generate-notes "$ASSET_DIR/$ASSET_NAME")"; then
  fail "creating the GitHub release for tag $TAG failed; the tag is on origin. Do not rerun this script (it would make a new tag). Create the release by hand with 'cp $PROJECT_DIR/release/pricefollower /tmp/$ASSET_NAME && gh release create $TAG --repo $REPO --verify-tag --title $TAG --generate-notes /tmp/$ASSET_NAME', or remove the tag with 'git push origin --delete $TAG && git tag -d $TAG'."
fi

echo "Released $TAG: $RELEASE_URL"
