# Build and deploy on Raspberry Pi (32-bit ARM)

The production backend is Go. The release build embeds the React `dist/` files in the Go executable, so the Pi needs only one application file. The SQLite database remains separate in a persistent data directory.

## Build a release

On a Linux or macOS build machine, install Node.js 22 or newer for the React toolchain and Go 1.25 or newer for the backend. From the project root, run:

```bash
npm ci
go mod download
./scripts/build-release.sh 7
```

The script builds the React app, embeds its files, and writes `release/pricefollower` for Linux ARMv7. For an ARMv6 target, use:

```bash
./scripts/build-release.sh 6
```

Confirm the Pi's processor and OS architecture before choosing the target. The Go SQLite driver is pure Go and supports Linux ARM, so the release does not need a C compiler or a native module installed on the Pi.

## Deploy an ARMv6 release in one command

After installing dependencies above, use a configured SSH target:

```bash
./scripts/deploy-armv6.sh pi@raspberry-pi
./scripts/deploy-armv6.sh pi@raspberry-pi /tmp/pricefollower-release
```

This rebuilds using `build-release.sh 6`, creates the remote staging directory, transfers the executable and installer using `copy-dist.sh`, then runs the installer with `sudo bash` over an SSH terminal. Success requires `systemctl is-active` to confirm the service is active. Failures stop subsequent stages; no rollback or continuous health monitoring is provided.

To also create the administrator account, or reset its password, after the installation, add `--admin-password` before the target:

```bash
./scripts/deploy-armv6.sh --admin-password pi@raspberry-pi
```

The new password is printed in the SSH terminal; see [Administrator account](#administrator-account).

The optional staging directory defaults to `pricefollower` relative to the SSH user's home; the binary is its child `pricefollower/pricefollower`. Absolute and home-relative paths, including `./pricefollower`, are accepted. Paths allow only ASCII letters, digits, underscores, dots, hyphens and slashes; spaces, shell metacharacters, literal `~`, root, dot and `..` segments are rejected. Staging under `/opt/pricefollower` or `/var/lib/pricefollower` is rejected. Choose an ordinary staging directory without symlink aliases into those locations. Installed files and persistent SQLite data retain the existing installer locations; no database is uploaded or deleted by the wrapper.

Targets accept ASCII hostnames or SSH aliases containing letters, digits, dots, underscores and hyphens, starting with a letter or digit, optionally preceded by a username and `@`. Usernames start with a letter, digit or underscore and use the same characters. Use an SSH config alias for IPv6 addresses or custom ports. Local Go, npm, SSH and rsync are required. The ARMv6-compatible remote Linux device needs SSH, rsync, Bash, sudo, systemd and the existing installer prerequisites. Sudo is required even when connecting as root. Interactive sudo authentication uses the SSH terminal; unattended deployment needs preconfigured SSH authentication and sudo privileges. The wrapper stores no credentials.

## Install and run manually

Upload the executable and installer to the Pi's `/tmp` directory with the deployment script:

```bash
./scripts/copy-dist.sh pi@raspberry-pi:/tmp/
```

The script copies the release binary produced by `scripts/build-release.sh` and `install-pricefollower.sh`. It does not copy the source tree or a separate `dist/` directory.

On the Pi, run the installer as root:

```bash
sudo /tmp/install-pricefollower.sh /tmp/pricefollower
```

The installer creates the system user and group, installs the executable under `/opt/pricefollower`, creates `/var/lib/pricefollower`, assigns both directories to the service account, and enables and starts the systemd service. Open `http://<raspberry-pi-address>:3001`. The database file is `/var/lib/pricefollower/pricefollower.sqlite`.

## Start at boot with systemd

View logs with `sudo journalctl -u pricefollower -f`. To update, build and upload the new executable from the development machine:

```bash
./scripts/build-release.sh 7
./scripts/copy-dist.sh pi@raspberry-pi:/tmp/
```

Then rerun the installer on the Pi; it replaces the executable and restarts the service:

```bash
sudo /tmp/install-pricefollower.sh /tmp/pricefollower
```

The database stays in `/var/lib/pricefollower` across updates.

## Administrator account

Without regular users, PriceFollower works without login (open mode). To add users, create the `admin` account on the Pi:

```bash
sudo -u pricefollower PRICEFOLLOWER_DATA_DIR=/var/lib/pricefollower /opt/pricefollower/pricefollower admin-password
```

The command prints `Username: admin` and a new random password; keep it safe. Run the same command again to reset a forgotten administrator password: the old password stops working and the administrator's open sessions end. It cannot be run from the browser and it does not change other users' passwords. The installer prints this command at the end, and `deploy-armv6.sh --admin-password` runs it after deploying.

Then open the application, choose "Log in", and log in as `admin` to add users. Creating the first user turns on login for everyone and gives that user the existing items and settings.

## Local development

Install Node.js 22 or newer and Go 1.25 or newer, then run:

```bash
npm ci
go mod download
npm run dev
```

Vite serves the frontend at `http://localhost:5173` and proxies API calls to the Go backend on port `3001`. The development backend seeds six sample listings into `.data/pricefollower.sqlite` on its first run. Development seed data is separate from `/var/lib/pricefollower`.

`scripts/copy-dist.sh` uploads the release executable and installer script. The production executable serves both the embedded frontend and the API, while SQLite data remains in the persistent data directory.
