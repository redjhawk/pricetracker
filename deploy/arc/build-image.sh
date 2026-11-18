#!/usr/bin/env bash
# Builds the arc-dev runner image here and imports it into the containerd of each k3s node (no registry).
#   deploy/arc/build-image.sh <tag> <ssh host> [<ssh host>...]
#   deploy/arc/build-image.sh 1 usuario@k3s
# Use the same tag in values-arc-dev.yaml; bump it whenever Dockerfile.dev changes.
set -euo pipefail
[ $# -ge 2 ] || { sed -n '3,4p' "$0"; exit 2; }
tag=$1; shift
dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
image="arc-runner-dev:$tag"

docker build -f "$dir/Dockerfile.dev" -t "$image" "$dir"
for host in "$@"; do
  arch=$(ssh "$host" uname -m)
  [ "$arch" = x86_64 ] || { echo "$host is $arch; Dockerfile.dev builds amd64 only" >&2; exit 1; }
  echo "Importing $image into $host"
  docker save "$image" | ssh "$host" sudo k3s ctr images import -
  ssh "$host" sudo k3s ctr images ls -q | grep -F "docker.io/library/$image"
done
