# ARC runners (k3s)

Self-hosted GitHub Actions runners for this repository, on a k3s cluster with
[Actions Runner Controller](https://github.com/actions/actions-runner-controller) 0.15.1.

| Scale set | Used by | Image | Max runners |
|---|---|---|---|
| `arc-dev` | `fabrica.yml` (agent factory) | `arc-runner-dev:<tag>` from [`Dockerfile.dev`](Dockerfile.dev), imported into k3s | 3 |
| `arc-build` | `release.yml` | default ARC runner image | 1 |

## Setup or update

1. Build the image on a machine with Docker and import it into every k3s node over SSH (no registry):
   ```sh
   deploy/arc/build-image.sh 1 usuario@k3s
   ```
2. Where `kubectl` and `helm` reach the cluster, install or update the controller and both scale sets:
   ```sh
   GITHUB_APP_ID=... GITHUB_APP_INSTALLATION_ID=... GITHUB_APP_KEY_FILE=clave.pem \
     deploy/arc/install.sh --remove-old
   ```
   The `GITHUB_APP_*` variables are needed only where the `github-app` secret does not exist yet.
   `--remove-old` uninstalls the `arc-test` and `arc-review` scale sets, which the pipeline no longer uses.

## Changing the runner image

Edit `Dockerfile.dev`, build with a new tag (`build-image.sh 2 …`), set that tag in
[`values-arc-dev.yaml`](values-arc-dev.yaml) and run `install.sh` again. Idle runners scale to zero, so the next
job starts with the new image.

## Notes

- `values-arc-dev.yaml` uses `imagePullPolicy: Never`: a pod on a node without the image fails with
  `ErrImageNeverPull`. Import the image into every node, or pin `arc-dev` to one node with a `nodeSelector`.
- Each `arc-dev` runner requests 2 CPU and 4 GiB (limit 8 GiB); runners beyond the node's capacity wait in `Pending`.
- The controller and CRDs are kept across updates. Reinstall them only when changing the chart version.
