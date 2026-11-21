#!/usr/bin/env bash
# Installs or updates ARC for this repository. Run where kubectl and helm point at the k3s cluster.
# Idempotent: keeps the controller, namespaces and secrets that already exist.
#   deploy/arc/install.sh                  # controller (if missing), arc-dev and arc-build
#   deploy/arc/install.sh redjhawk@malaga  # same, run on that ssh host
# Secrets are created only if missing: github-app from GITHUB_APP_ID, GITHUB_APP_INSTALLATION_ID and
# GITHUB_APP_KEY_FILE; ghcr-pull (arc-dev image) from GHCR_USER and GHCR_TOKEN (read:packages).
# With a host, this script, the values files, those variables and the key file are copied to a temporary
# directory on the host (removed afterwards) and the script runs there in a login shell, so helm and kubectl
# must work there; the repository does not need to be checked out on the host.
set -euo pipefail
dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

[ $# -le 1 ] || { sed -n '4,5p' "$0"; exit 2; }
if [ $# -eq 1 ]; then
  bundle="$(mktemp -d)"
  trap 'rm -rf "$bundle"' EXIT
  cp "$dir/install.sh" "$dir"/values-*.yaml "$dir/claude-settings.json" "$bundle/"
  : >"$bundle/env"
  for var in GITHUB_APP_ID GITHUB_APP_INSTALLATION_ID GHCR_USER GHCR_TOKEN; do
    if [ -n "${!var:-}" ]; then printf 'export %s=%q\n' "$var" "${!var}" >>"$bundle/env"; fi
  done
  if [ -n "${GITHUB_APP_KEY_FILE:-}" ]; then
    cp "$GITHUB_APP_KEY_FILE" "$bundle/github-app.pem"
    echo 'export GITHUB_APP_KEY_FILE="$PWD/github-app.pem"' >>"$bundle/env"
  fi
  remote='set -e; d="$(mktemp -d)"; trap "rm -rf \"$d\"" EXIT; tar -xzf - -C "$d"; cd "$d"; . ./env; bash ./install.sh'
  tar -czf - -C "$bundle" . | ssh "$1" "bash -lc $(printf %q "$remote")"
  exit
fi
version=0.15.1
charts=oci://ghcr.io/actions/actions-runner-controller-charts
repo_url=https://github.com/redjhawk/pricetracker

if ! helm status arc --namespace arc-systems >/dev/null 2>&1; then
  helm install arc --namespace arc-systems --create-namespace --version "$version" \
    "$charts/gha-runner-scale-set-controller"
fi

for set in arc-dev arc-build; do
  kubectl get namespace "$set" >/dev/null 2>&1 || kubectl create namespace "$set"
  if ! kubectl get secret github-app --namespace "$set" >/dev/null 2>&1; then
    : "${GITHUB_APP_ID:?secret github-app missing in $set: set GITHUB_APP_ID}"
    : "${GITHUB_APP_INSTALLATION_ID:?set GITHUB_APP_INSTALLATION_ID}"
    : "${GITHUB_APP_KEY_FILE:?set GITHUB_APP_KEY_FILE}"
    kubectl create secret generic github-app --namespace "$set" \
      --from-literal=github_app_id="$GITHUB_APP_ID" \
      --from-literal=github_app_installation_id="$GITHUB_APP_INSTALLATION_ID" \
      --from-file=github_app_private_key="$GITHUB_APP_KEY_FILE"
  fi
  if [ "$set" = arc-dev ] && ! kubectl get secret ghcr-pull --namespace arc-dev >/dev/null 2>&1; then
    : "${GHCR_USER:?secret ghcr-pull missing in arc-dev: set GHCR_USER}"
    : "${GHCR_TOKEN:?set GHCR_TOKEN (a token with read:packages)}"
    kubectl create secret docker-registry ghcr-pull --namespace arc-dev \
      --docker-server=ghcr.io --docker-username="$GHCR_USER" --docker-password="$GHCR_TOKEN"
  fi
  if [ "$set" = arc-dev ]; then
    kubectl create configmap claude-code-settings --namespace arc-dev \
      --from-file=settings.json="$dir/claude-settings.json" --dry-run=client -o yaml | kubectl apply -f -
  fi
  helm upgrade --install "$set" --namespace "$set" --version "$version" \
    --set githubConfigUrl="$repo_url" --set githubConfigSecret=github-app \
    -f "$dir/values-$set.yaml" \
    "$charts/gha-runner-scale-set"
done

kubectl get pods -n arc-systems
