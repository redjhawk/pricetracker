#!/usr/bin/env bash
# Builds the arc-dev runner image and pushes it to the private GitHub Container Registry package
# ghcr.io/redjhawk/arc-runner-dev. k3s pulls it with the ghcr-pull secret created by install.sh.
#   deploy/arc/build-image.sh <tag>        e.g. deploy/arc/build-image.sh 1
# Needs `docker login ghcr.io` with a token that has write:packages. Use the same tag in values-arc-dev.yaml.
set -euo pipefail
[ $# -eq 1 ] || { sed -n '4p' "$0"; exit 2; }
dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
image="ghcr.io/redjhawk/arc-runner-dev:$1"

if docker manifest inspect "$image" >/dev/null 2>&1; then
  echo "$image already exists; use a new tag" >&2; exit 1
fi
docker build --platform linux/amd64 -f "$dir/Dockerfile.dev" -t "$image" "$dir"
docker push "$image"
echo "Pushed $image. Set it in deploy/arc/values-arc-dev.yaml and run deploy/arc/install.sh."
