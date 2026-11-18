# ARC runners (k3s)

Self-hosted GitHub Actions runners for this repository, on the single-node k3s cluster `malaga`, with
[Actions Runner Controller](https://github.com/actions/actions-runner-controller) 0.15.1 and Helm 4.

| Scale set | Used by | Image | Max runners |
|---|---|---|---|
| `arc-dev` | `fabrica.yml` (agent factory) | private package `ghcr.io/redjhawk/arc-runner-dev:<tag>`, from [`Dockerfile.dev`](Dockerfile.dev) | 3 |
| `arc-build` | `release.yml` (build, tag and GitHub release) | default ARC runner image | 1 |

The controller (`arc` in `arc-systems`) is shared and kept across updates. `arc-test` and `arc-review` belonged to
the previous pipeline and are no longer used.

## Requirements

No root access on `malaga` is needed: the image goes through a private GitHub Container Registry package and k3s
pulls it with a Kubernetes secret.

- On the machine that builds the image: Docker, logged in to ghcr.io with a classic token that has `write:packages`:
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
     deploy/arc/install.sh --remove-old
   ```
   It installs the controller only if missing, creates the namespaces and secrets only if missing, installs or
   upgrades `arc-dev` and `arc-build`, and with `--remove-old` uninstalls `arc-test` and `arc-review`.
3. **Check:**
   ```sh
   kubectl get secrets -A -l owner=helm -L name,status   # arc, arc-dev, arc-build: deployed
   kubectl get pods -n arc-systems                       # controller + one listener per scale set
   kubectl get secret ghcr-pull -n arc-dev
   ```
   Then start `fabrica` by hand from the Actions tab and watch `kubectl get pods -n arc-dev -w`: the runner pod
   must reach `Running`.
4. **Merge PR #143 by hand** before labelling issues `ai-dev`: until then, `master` still runs the previous
   pipeline (`desarrollo`, `revision-pr`, ...), which also uses `arc-dev`.

## Changing the runner image

Edit `Dockerfile.dev`, build with a new tag (`build-image.sh 2`), set that tag in
[`values-arc-dev.yaml`](values-arc-dev.yaml) and run `install.sh` again. Idle runners scale to zero, so the next
job starts with the new image. Never reuse a tag: with `imagePullPolicy: IfNotPresent` the node keeps the image it
already pulled under that tag.

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
