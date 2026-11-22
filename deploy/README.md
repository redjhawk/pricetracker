# Build and release

Everything about building, releasing and running PriceFollower in production. The production backend is Go. The release build embeds the React `dist/` files in the Go executable, so a server needs only one application file; the SQLite database stays in a separate data directory (`PRICEFOLLOWER_DATA_DIR`).

## Build a release

On a Linux or macOS build machine with Node.js 22 or newer and Go 1.25 or newer, from the project root:

```bash
npm ci
go mod download
./scripts/build-release.sh
```

The script builds the React app, embeds its files, and writes a static Linux amd64 binary (`CGO_ENABLED=0`) to `release/pricefollower`. The Go SQLite driver is pure Go, so no C compiler or native module is needed on the server.

## Publish a GitHub release (amd64)

Every push to `master` (a merged PR) runs the `release` workflow, which builds the Linux amd64 binary with `./scripts/build-release.sh amd64` and publishes a GitHub release `v<UTC timestamp>` with `pricefollower-linux-amd64` and its `.sha256`. Use the script below only for a manual release.

From a clean, committed clone with Go, Node.js, `git` and an authenticated GitHub CLI (`gh auth login`), run:

```bash
./scripts/create-github-release.sh
```

The script refuses a dirty working tree, builds the Linux amd64 binary with `./scripts/build-release.sh amd64`, tags HEAD as `vYYYY.MM.DD-N` (UTC date; N is the first number not already used locally or on origin), pushes the tag, and creates a GitHub release titled after the tag with GitHub-generated notes and a single asset, `pricefollower-linux-amd64`. It prints the tag and the release URL. If the release step fails after the tag was pushed, do not rerun the script (it would create a new tag): the error prints the exact `gh release create` command to finish the release by hand, and the command to delete the tag instead.

Installing the release on the server is outside this repository: the server checks for new GitHub releases and downloads them itself.

## Server setup

The service user must be able to write the data directory (`PRICEFOLLOWER_DATA_DIR`). It holds the SQLite database and the browser profile used for Amazon checks, `amazon-browser/`.

### Browser for Amazon checks

Amazon prices are read through a headless web browser, so the server needs Chromium or Google Chrome. On Debian:

```bash
sudo apt-get install -y chromium
```

On Ubuntu and other amd64 systems: `wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb && sudo apt-get install -y ./google-chrome-stable_current_amd64.deb`.

The application uses `PRICEFOLLOWER_BROWSER_PATH` (full path to the browser executable) when it is set, otherwise the first of `chromium`, `chromium-browser`, `google-chrome-stable`, `google-chrome` found on the service's `PATH`. It looks for the browser again at each check, so a browser installed later is used without a restart.

In a container or for a service running as root, the browser cannot use its sandbox: set `PRICEFOLLOWER_BROWSER_NO_SANDBOX=true`. In Docker, also give the container enough shared memory (`--shm-size=1g`).

What the server log shows:

- At start, when no browser is found: `Amazon browser not found; Amazon checks will fail until Chromium is installed or …`.
- For each failed Amazon check: `Amazon request failed url=… at=… status=… cause=…`. The cause says whether the browser could not start (with the browser's own reason), the page could not be opened or did not finish loading, Amazon returned a non-success HTTP status, or Amazon showed a robot check.
- For a page without a detectable price: `Amazon check found no price url=… at=… status=… cause=…`. It is not counted as a failed request.

## Administrator account

Without regular users, PriceFollower works without login (open mode). To add users, create the `admin` account on the server, as the service user and with the service's data directory:

```bash
PRICEFOLLOWER_DATA_DIR=<data directory> ./pricefollower admin-password
```

The command prints `Username: admin` and a new random password; keep it safe. The password stays in the terminal scrollback, so do not run the command where the output is logged, such as CI jobs or shared terminals. Run the same command again to reset a forgotten administrator password: the old password stops working and the administrator's open sessions end. It cannot be run from the browser and it does not change other users' passwords.

Then open the application, choose "Log in", and log in as `admin` to add users. Creating the first user turns on login for everyone and gives that user the existing items and settings.

## CI runners

The GitHub Actions workflows run on self-hosted runners in a k3s cluster: `arc-dev` for the agent factory
(`fabrica.yml` and `supervisor.yml`) and `arc-build` for `release.yml`. Their image, Helm values, install scripts and troubleshooting are
in [arc/README.md](arc/README.md).
