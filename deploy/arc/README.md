# ARC runners (k3s)

Self-hosted GitHub Actions runners for this repository (application deployment: [../README.md](../README.md)), on the single-node k3s cluster `malaga`, with
[Actions Runner Controller](https://github.com/actions/actions-runner-controller) 0.15.1 and Helm 4.

| Scale set | Used by | Image | Max runners |
|---|---|---|---|
| `arc-dev` | `fabrica.yml` (agent factory) | private package `ghcr.io/redjhawk/arc-runner-dev:<tag>`, from [`Dockerfile.dev`](Dockerfile.dev) | 3 |
| `arc-build` | `release.yml` (build, tag and GitHub release) | default ARC runner image | 1 |

The controller (`arc` in `arc-systems`) is shared and kept across updates.

## Requirements

No root access on `malaga` is needed: the image goes through a private GitHub Container Registry package and k3s
pulls it with a Kubernetes secret.

- Two classic GitHub tokens (Settings → Developer settings → Personal access tokens → Tokens (classic)): one
  with `write:packages` to push the image, and one with only `read:packages` that the cluster keeps to pull it.
- On the machine that builds the image: Docker, logged in to ghcr.io with the `write:packages` token:
  `echo "$TOKEN" | docker login ghcr.io -u redjhawk --password-stdin`.
- Where `helm` and `kubectl` reach the cluster: this repository checked out. The first run of `install.sh` also needs
  the variables for the secrets that do not exist yet:
  - `ghcr-pull` (arc-dev): `GHCR_USER=redjhawk` and `GHCR_TOKEN`, a classic token with only `read:packages`.
  - `github-app` (each scale set): `GITHUB_APP_ID`, `GITHUB_APP_INSTALLATION_ID`, `GITHUB_APP_KEY_FILE`.

## Setup, in order

1. **Build and push the runner image** (from the build machine, at the repository root):
   ```sh
   deploy/arc/build-image.sh 1
   ```
   The package stays private. The script refuses a tag that already exists.
2. **Install or update the scale sets** (where `helm` reaches the cluster):
   ```sh
   GHCR_USER=redjhawk GHCR_TOKEN=... \
   GITHUB_APP_ID=... GITHUB_APP_INSTALLATION_ID=... GITHUB_APP_KEY_FILE=clave.pem \
     deploy/arc/install.sh
   ```
   It installs the controller only if missing, creates the namespaces and secrets only if missing, installs or
   upgrades `arc-dev` and `arc-build`.
3. **Check:**
   ```sh
   kubectl get secrets -A -l owner=helm -L name,status   # arc, arc-dev, arc-build: deployed
   kubectl get pods -n arc-systems                       # controller + one listener per scale set
   kubectl get secret ghcr-pull -n arc-dev
   ```
   Then start `fabrica` by hand from the Actions tab (GitHub offers a manual run only for workflows on `master`)
   with a small test issue, and watch `kubectl get pods -n arc-dev -w`: the runner pod must reach `Running`, and the
   job's first step checks that the image has every tool. `arc-build` is tested by the next merge to `master`:
   `release.yml` must run on it and publish the release.

## Changing the runner image

Edit `Dockerfile.dev`, build with a new tag (`build-image.sh 2`), set that tag in
[`values-arc-dev.yaml`](values-arc-dev.yaml) and run `install.sh` again. Idle runners scale to zero, so the next
job starts with the new image. Never reuse a tag: with `imagePullPolicy: IfNotPresent` the node keeps the image it
already pulled under that tag.

## Alternative: import the image without a registry (restricted sudo)

The default setup above needs no root access on `malaga`. If the image must not leave the local network, it can
instead be piped over SSH into the node's containerd, which requires root. Do not grant `sudo k3s ctr *`: `ctr` can
start privileged containers, which equals full root. Grant one fixed script that only imports an image.

**Once, as root on `malaga`** (`su -`, or ask the administrator; without root nobody can grant sudo):

1. Create the import script (check the k3s path with `which k3s` and adjust it if it is not `/usr/local/bin/k3s`):
   ```sh
   cat > /usr/local/sbin/k3s-import-image <<'EOF'
   #!/bin/sh
   # Imports one image tarball from stdin into k3s containerd, then lists arc-runner images.
   # Takes no arguments: the only thing sudo allows the deploy user to do.
   set -eu
   [ $# -eq 0 ] || { echo "usage: sudo k3s-import-image < image.tar" >&2; exit 2; }
   /usr/local/bin/k3s ctr images import -
   /usr/local/bin/k3s ctr images ls -q | grep arc-runner || true
   EOF
   chown root:root /usr/local/sbin/k3s-import-image
   chmod 755 /usr/local/sbin/k3s-import-image
   ```
2. Allow `redjhawk` to run only that script, without a password and without arguments (the trailing `""`):
   ```sh
   echo 'redjhawk ALL=(root) NOPASSWD: /usr/local/sbin/k3s-import-image ""' > /etc/sudoers.d/k3s-import-image
   chmod 440 /etc/sudoers.d/k3s-import-image
   visudo -cf /etc/sudoers.d/k3s-import-image
   ```
   If `visudo -c` reports an error, delete the file: a broken sudoers file can lock everyone out of `sudo`.

**Check, as `redjhawk`:**
```sh
sudo -n /usr/local/sbin/k3s-import-image < /dev/null   # runs without a password prompt
sudo -n k3s ctr version                                 # must be refused
```

**Use it** instead of steps 1–2 of the setup:
```sh
docker build --platform linux/amd64 -f deploy/arc/Dockerfile.dev -t arc-runner-dev:1 deploy/arc
docker save arc-runner-dev:1 | ssh redjhawk@malaga sudo -n /usr/local/sbin/k3s-import-image
```
and in [`values-arc-dev.yaml`](values-arc-dev.yaml) remove `imagePullSecrets` and set:
```yaml
image: docker.io/library/arc-runner-dev:1
imagePullPolicy: Never
```
A pod on a node without that tag then fails with `ErrImageNeverPull`: import it on every node. `install.sh` still
creates the `ghcr-pull` secret, which is harmless; skip it by creating an empty one
(`kubectl create secret generic ghcr-pull -n arc-dev`).

## Troubleshooting

- **`UPGRADE FAILED: "arc-dev" has no deployed releases`**: a previous install failed or was left half-removed.
  Helm keeps its history in secrets; delete them and install again (the namespace and `github-app` stay):
  ```sh
  kubectl get secrets -A -l owner=helm -L name,status
  kubectl delete secrets -n arc-dev -l owner=helm,name=arc-dev
  deploy/arc/install.sh
  ```
  If the install then complains that ARC resources already exist:
  `kubectl delete autoscalingrunnersets,autoscalinglisteners -n arc-dev --all`.
- **Helm 4 flags**: `helm list -a` / `--all` no longer exist; use the `kubectl get secrets -l owner=helm` command above
  to see every release and its status.
- **Pod in `ImagePullBackOff`**: `kubectl describe pod -n arc-dev <pod>` shows why. `401`/`403`: the `ghcr-pull`
  token lacks `read:packages` or expired; recreate it with `kubectl delete secret ghcr-pull -n arc-dev` and
  `install.sh`. `not found`: the tag in `values-arc-dev.yaml` was never pushed.
- **Pods in `Pending`**: each `arc-dev` runner requests 2 CPU and 4 GiB (limit 8 GiB); three runners need 6 CPU and
  12 GiB free (`kubectl describe node | grep -A6 Allocatable`). Lower `maxRunners` or the requests if needed.
- **Namespace stuck in `Terminating`**: remove the ARC finalizers:
  ```sh
  for r in $(kubectl get autoscalingrunnersets,ephemeralrunnersets,ephemeralrunners,autoscalinglisteners -n arc-dev -o name); do
    kubectl patch "$r" -n arc-dev --type merge -p '{"metadata":{"finalizers":null}}'
  done
  ```
