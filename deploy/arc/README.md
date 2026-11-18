# ARC runners (k3s)

Self-hosted GitHub Actions runners for this repository, on the single-node k3s cluster `malaga`, with
[Actions Runner Controller](https://github.com/actions/actions-runner-controller) 0.15.1 and Helm 4.

| Scale set | Used by | Image | Max runners |
|---|---|---|---|
| `arc-dev` | `fabrica.yml` (agent factory) | `arc-runner-dev:<tag>` from [`Dockerfile.dev`](Dockerfile.dev), imported into k3s | 3 |
| `arc-build` | `release.yml` (build, tag and GitHub release) | default ARC runner image | 1 |

The controller (`arc` in `arc-systems`) is shared and kept across updates. `arc-test` and `arc-review` belonged to
the previous pipeline and are no longer used.

## Requirements

- On the machine that builds the image: Docker, and SSH access to `malaga`.
- On `malaga`: the SSH user (`redjhawk`) can run `sudo k3s` without a password, because the image is piped into
  `sudo k3s ctr images import -`. Check from the build machine:
  ```sh
  ssh redjhawk@malaga sudo -n k3s ctr version >/dev/null && echo ok
  ```
  If it asks for a password, allow only that command (`sudo visudo -f /etc/sudoers.d/k3s-ctr`, path from `which k3s`):
  ```
  redjhawk ALL=(root) NOPASSWD: /usr/local/bin/k3s ctr *
  ```
- Where `helm` and `kubectl` reach the cluster: this repository checked out, and the `github-app` secret in each
  scale set namespace (or the `GITHUB_APP_*` variables below to create it).

## Setup, in order

1. **Build the runner image and import it into k3s** (from the build machine, at the repository root):
   ```sh
   deploy/arc/build-image.sh 1 redjhawk@malaga
   ```
   When building on `malaga` itself, skip SSH:
   `docker build -f deploy/arc/Dockerfile.dev -t arc-runner-dev:1 deploy/arc && docker save arc-runner-dev:1 | sudo k3s ctr images import -`.
2. **Install or update the scale sets** (where `helm` reaches the cluster):
   ```sh
   GITHUB_APP_ID=... GITHUB_APP_INSTALLATION_ID=... GITHUB_APP_KEY_FILE=clave.pem \
     deploy/arc/install.sh --remove-old
   ```
   It installs the controller only if missing, creates the namespaces and `github-app` secrets only if missing
   (the `GITHUB_APP_*` variables are needed only then), installs or upgrades `arc-dev` and `arc-build`, and with
   `--remove-old` uninstalls `arc-test` and `arc-review`.
3. **Check:**
   ```sh
   kubectl get secrets -A -l owner=helm -L name,status   # arc, arc-dev, arc-build: deployed
   kubectl get pods -n arc-systems                       # controller + one listener per scale set
   sudo k3s ctr images ls -q | grep arc-runner-dev       # on malaga
   ```
4. **Merge PR #143 by hand** before labelling issues `ai-dev`: until then, `master` still runs the previous
   pipeline (`desarrollo`, `revision-pr`, ...), which also uses `arc-dev`.

## Changing the runner image

Edit `Dockerfile.dev`, build with a new tag (`build-image.sh 2 redjhawk@malaga`), set that tag in
[`values-arc-dev.yaml`](values-arc-dev.yaml) and run `install.sh` again. Idle runners scale to zero, so the next
job starts with the new image. Do not reuse a tag: with `imagePullPolicy: Never` the node keeps the old image.

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
- **Pod in `ErrImageNeverPull`**: the image tag in `values-arc-dev.yaml` is not imported on the node; repeat step 1
  with that tag.
- **Pods in `Pending`**: each `arc-dev` runner requests 2 CPU and 4 GiB (limit 8 GiB); three runners need 6 CPU and
  12 GiB free (`kubectl describe node | grep -A6 Allocatable`). Lower `maxRunners` or the requests if needed.
- **Namespace stuck in `Terminating`**: remove the ARC finalizers:
  ```sh
  for r in $(kubectl get autoscalingrunnersets,ephemeralrunnersets,ephemeralrunners,autoscalinglisteners -n arc-dev -o name); do
    kubectl patch "$r" -n arc-dev --type merge -p '{"metadata":{"finalizers":null}}'
  done
  ```
