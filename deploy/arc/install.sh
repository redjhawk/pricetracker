#!/usr/bin/env bash
# Installs or updates ARC for this repository. Run where kubectl and helm point at the k3s cluster.
# Idempotent: keeps the controller, namespaces and secrets that already exist.
#   deploy/arc/install.sh                  # controller (if missing), arc-dev and arc-build
# Secrets are created only if missing: github-app from GITHUB_APP_ID, GITHUB_APP_INSTALLATION_ID and
# GITHUB_APP_KEY_FILE; ghcr-pull (arc-dev image) from GHCR_USER and GHCR_TOKEN (read:packages).
set -euo pipefail
dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
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
  helm upgrade --install "$set" --namespace "$set" --version "$version" \
    --set githubConfigUrl="$repo_url" --set githubConfigSecret=github-app \
    -f "$dir/values-$set.yaml" \
    "$charts/gha-runner-scale-set"
done

kubectl get pods -n arc-systems
