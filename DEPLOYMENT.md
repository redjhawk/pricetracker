# Deployment Recommendation — Raspberry Pi

## Recommended format

Package the application as an architecture-specific Debian package (`.deb`). This provides one installable file while using the Debian package manager for installation and upgrades.

Initial target: ARM64, assuming a Raspberry Pi 3 or newer running a 64-bit Debian-based OS. Build a separate ARM32 package only if older 32-bit Raspberry Pi systems need support. Raspberry Pi OS is available in both 32-bit and 64-bit editions; select a package that matches the installed OS architecture. See the [Raspberry Pi OS architecture documentation](https://www.raspberrypi.com/documentation/computers/os.html).

Example installation:

```bash
sudo apt install ./pricefollower_arm64.deb
```

## Package contents and service behavior

The package should contain:

- The Node.js backend and production dependencies.
- The built React frontend, served by the backend.
- A pinned Node.js 24 LTS runtime, avoiding dependence on whichever Node.js version happens to be installed on the Pi. Node.js 24 is listed as an LTS release in the [Node.js release schedule](https://nodejs.org/en/about/previous-releases).
- A `systemd` service definition so the application starts at boot and can be managed as a service.

Keep the SQLite database outside the package-managed application directory, for example under `/var/lib/pricefollower/`. This allows package upgrades to replace application files without replacing tracked items or price history.

The backend should serve both the frontend and API from one service. Developers and operators can then open the Pi's hostname or address in a browser; the browser's frontend requests data from that same backend.

## Quick manual install on a Raspberry Pi

For a quick trial without building a Debian package, install Node.js 22 or newer on the Pi and copy the project directory to it. Then run:

```bash
cd /path/to/pricefollower
npm ci
npm run build
mkdir -p ./data
PRICEFOLLOWER_DATA_DIR=./data npm start
```

The Node.js server serves the built files from `dist/` and the API from the same origin. Open `http://<raspberry-pi-address>:3001` from a browser on the network. The SQLite database will be created under `./data/pricefollower.sqlite`. Ensure that the account running Node.js can write to the selected data directory. Keep the process running while the application is in use; use the `.deb` and `systemd` service approach above for startup at boot and ongoing use.

## Local development

From the project directory, install dependencies and start both tiers with one command:

```bash
npm install
npm run dev
```

Open `http://localhost:5173`. Vite serves the React frontend and proxies `/api` requests to the Node.js backend on port `3001`. On the first development run, the backend seeds six representative items into `.data/pricefollower.sqlite`; this development database is separate from the production data directory. Remove that file manually only when a clean sample database is needed.

Use Node.js 22 or newer for development, matching the declared project engine and Carbon dependency requirements.

For production, build the frontend with `npm run build` and start the Node.js service with `npm start`. The same Node process serves the built frontend and API. Set `PRICEFOLLOWER_DATA_DIR` to the service's persistent data directory when deploying.

## Why not a standalone executable?

Node.js Single Executable Applications (SEA) can bundle a Node.js application into one executable, but Node.js currently marks the feature as active development. Bundling application dependencies and frontend assets also adds complexity. For this service, a `.deb` is the simpler production choice while still providing a single installable artifact. See the [Node.js SEA documentation](https://nodejs.org/download/release/v26.8.1/docs/api/single-executable-applications.html).

## Release artifact

The initial release should produce one package for the selected target, for example `pricefollower_arm64.deb`. If both ARM64 and ARM32 systems must be supported, publish one package per architecture. Confirm the Pi model and OS architecture before producing the first release.
