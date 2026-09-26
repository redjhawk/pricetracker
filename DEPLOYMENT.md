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

## Local development

Install Node.js 22 or newer and Go 1.25 or newer, then run:

```bash
npm ci
go mod download
npm run dev
```

Vite serves the frontend at `http://localhost:5173` and proxies API calls to the Go backend on port `3001`. The development backend seeds six sample listings into `.data/pricefollower.sqlite` on its first run. Development seed data is separate from `/var/lib/pricefollower`.

`scripts/copy-dist.sh` uploads the release executable and installer script. The production executable serves both the embedded frontend and the API, while SQLite data remains in the persistent data directory.
