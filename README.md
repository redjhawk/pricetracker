# PriceFollower

PriceFollower is a self-hosted price tracker for European Amazon and French LeBoncoin listings. Add a listing URL and the server checks its price twice a day, stores dated observations, and shows the latest price and recent history in a web interface. It can run on a Raspberry Pi with a single Go executable and a SQLite database.

## Features

- Track products from Amazon Germany, France, Spain, Italy, the Netherlands, and Belgium.
- Track listings from LeBoncoin France, including explicit donations displayed as “Gratuit”.
- Browse separate Amazon and LeBoncoin tabs, with search scoped to the selected platform and second-hand offers shown only for Amazon.
- Collect the first price as soon as a product is added, then check twice daily.
- View the latest detected price, its timestamp, collection status, and the three latest observations.
- Track the lowest-priced second-hand offer explicitly sold by Amazon, with its condition and recent history; third-party offers are ignored.
- Manually refresh all tracked item prices and Amazon-sold second-hand offers from the main page.
- Manually refresh an item from its details page.
- Keep checking after stale prices or collection failures.
- Delete tracked products and their stored price history.
- Use one shared collection without accounts or login.
- Run the backend and React frontend in development mode with sample data.
- Build a self-contained Linux ARM executable with the frontend embedded for Raspberry Pi deployment.

Prices are shown in euros and represent the item price. Shipping, taxes, alerts, trend charts, and marketplaces other than European Amazon and LeBoncoin France are outside the current v1 scope.

## Technology

- **Frontend:** React 19, TypeScript, Vite, IBM Carbon Design System
- **Backend:** Go HTTP server
- **Storage:** SQLite
- **Production deployment:** One Go executable serving the embedded frontend and API

## Development

Install Node.js 22 or newer and Go 1.25 or newer. From the project directory:

```bash
npm ci
go mod download
npm run dev
```

Open <http://localhost:5173>. Vite serves the frontend and proxies API requests to the Go server on port `3001`. The development backend seeds sample products into `.data/pricefollower.sqlite` on its first run. This development database is separate from the production data directory.

The API contract is documented in [API_SPECIFICATION.md](API_SPECIFICATION.md). Product scope and behavior are in [doc/FUNCTIONAL_SPECIFICATIONS.md](doc/FUNCTIONAL_SPECIFICATIONS.md), with user and system flows listed in [doc/use-cases/README.md](doc/use-cases/README.md).

### Platform-tab browser tests

With Node.js 22+ and dependencies installed, install Playwright's Chromium browser once and run the focused suite:

```bash
npx playwright install chromium
npm run test:platform-tabs
```

The suite starts its own Vite frontend at `http://127.0.0.1:4173`; that port must be free. It intercepts API requests with in-memory fixtures and blocks external requests, so it needs no Go server or database. To use an existing Chromium installation, set `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH` to its executable path. Failed tests retain traces in `test-results/`.

## Raspberry Pi deployment

Build on a machine with Node.js 22+, Go 1.25+, and SSH/rsync access to the Pi. Choose ARMv7 or ARMv6 for the target processor:

```bash
npm ci
go mod download
./scripts/build-release.sh 7
```

Use `./scripts/build-release.sh 6` for ARMv6. Upload the executable and installer to the Pi:

```bash
./scripts/copy-dist.sh pi@raspberry-pi:/tmp/
```

On the Pi, install and start the systemd service:

```bash
sudo /tmp/install-pricefollower.sh /tmp/pricefollower
```

The installer creates the `pricefollower` service user and group, installs the app in `/opt/pricefollower`, and stores the SQLite database in `/var/lib/pricefollower`. The web interface and API are served from port `3001`. See [DEPLOYMENT.md](DEPLOYMENT.md) for details and updates.

## Project status

This is an early, single-instance application intended for a small shared collection of around 100 products. It does not include user accounts. Collection depends on product pages remaining accessible and parseable; a failed check is retained as a collection outcome and does not replace the last successful price.

## Similar projects and search terms

Self-hosted Amazon and LeBoncoin price tracker · Raspberry Pi price monitoring · European price history · Go and SQLite price tracking · React Carbon price tracker
