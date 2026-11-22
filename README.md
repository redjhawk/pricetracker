# PriceFollower

PriceFollower is a self-hosted price tracker for European Amazon and French LeBoncoin listings. Add a listing URL and the server checks its price twice a day, stores dated observations, and shows the latest price and recent history in a web interface. It runs as a single Go executable with a SQLite database.

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
- Use one shared collection without login, or add local users: each user logs in and has their own items, Claude token and LeBoncoin session; an `admin` account only manages users.
- Run the backend and React frontend in development mode with sample data.
- Build a self-contained Linux amd64 executable with the frontend embedded, published as a GitHub release on every merge.

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

## Build and release

Every merge to `master` builds a static Linux amd64 executable with the frontend embedded and publishes it as a GitHub release. Building by hand, releases and the administrator account are described in [deploy/README.md](deploy/README.md), together with the self-hosted CI runners.

For LeBoncoin verification challenges, follow [LeBoncoin verification: capture a session and paste it in Settings](doc/leboncoin-session.md).

## Automated development (agent factory)

Claude agents develop the issues of this repository. An issue goes through specification, tests, implementation, review and triage to a merged pull request, and the merge publishes a release. The full process (stages, gates, review and security rules) is in [doc/workflow/WORKFLOW.md](doc/workflow/WORKFLOW.md).

| Workflow | Starts | Does | Runner |
|---|---|---|---|
| [`fabrica.yml`](.github/workflows/fabrica.yml) | an issue gets the `ai-dev` label, or the owner comments on an `ai-dev` issue | runs the factory on that issue until it stops or merges | `arc-dev` |
| [`supervisor.yml`](.github/workflows/supervisor.yml) | every 15 minutes | resumes stopped runs and launches the next issue | `arc-dev` |
| [`release.yml`](.github/workflows/release.yml) | every push to `master` | builds and publishes the GitHub release | `arc-build` |

### Setup

Nothing runs on the PriceFollower server. All three workflows run on the self-hosted runners, and the supervisor needs nothing beyond what the factory needs:

1. **Runners.** Deploy the `arc-dev` and `arc-build` scale sets as described in [deploy/arc/README.md](deploy/arc/README.md). The `arc-dev` image ([`Dockerfile.dev`](deploy/arc/Dockerfile.dev)) contains the whole toolchain: Go, Node, `gh`, `jq`, the Claude CLI, Playwright Chromium, `govulncheck` and `gosec`. After changing the image, publish it under a new tag (see "Changing the runner image" in that README).
2. **Secrets** (GitHub → Settings → Secrets and variables → Actions):
   - `DEV_TOKEN`, in the `dev` environment: a token of the repository owner that can write contents, pull requests and issues. The factory pushes, opens and merges pull requests with it, and the supervisor comments and labels with it. Because the events it creates count as the owner's, they start the other workflows: a comment or an `ai-dev` label starts `fabrica.yml`, and a merge starts `release.yml`.
   - `CLAUDE_CODE_OAUTH_TOKEN`, a repository secret: the Claude subscription token from `claude setup-token`. Every agent of the factory and of the supervisor uses it, so they all share that account's usage limit.
3. **Optional repository variables** (same page, *Variables* tab):
   - `FACTORY_SUPERVISOR=off` stops the supervisor completely. After switching it back on, run `supervisor` once by hand (Actions → `supervisor` → *Run workflow*) to restart its loop.
   - `FACTORY_MAX_ACTIVE` sets how many issues are in development at a time. The default is `1`, and `0` stops the launch of new issues while the supervisor still resumes the stopped ones.

GitHub's own schedule is best effort and has never fired in this repository. Each `supervisor` run therefore starts the next one itself, about 15 minutes later, from a short job on a GitHub-hosted runner, which is free for a public repository. The schedule stays as a backup.

### Daily use

- **Start an issue:** add the `ai-dev` label, or let the supervisor pick it from the queue (see below). A label added by anyone other than the owner starts nothing.
- **Answer questions:** when a functional point is unclear, the factory asks in a comment and stops with `factory:needs-input`. Answer in a comment on the issue, and the factory resumes from where it stopped.
- **Follow progress:** one label shows where each issue is.
  - Stage labels: `factory:spec`, `factory:usecases`, `factory:testcases`, `factory:techspec`, `factory:tests`, `factory:implement`, `factory:pr`, `factory:review`, `factory:triage`, `factory:merge` and `factory:done`.
  - Labels of a stopped run:
    - `factory:needs-input`: questions;
    - `factory:needs-human`: a failed gate, a merge conflict, or the supervisor gave up;
    - `factory:blocked`: the environment;
    - `factory:waiting`: a bug on `master`;
    - `factory:waiting-limit`: the Claude usage limit;
    - `factory:failed`: the run failed with an error.
  - Every stop comment says why the run stopped and what happens next.
- **Merge by hand:** the factory never merges a pull request that changes `.github/` or `scripts/factory/`. It stops with `factory:needs-human` and leaves the merge to you.
- **Issues the automation manages:** "Tests fail on master" issues go through the factory like any other. The supervisor opens the "Agent factory: the runner cannot run agents" issue and closes it itself.

### What the supervisor does on its own

| Situation | Action |
|---|---|
| The Claude usage limit ran out | resumes the run 5 minutes after the limit resets, and relaunches nothing else until then |
| A run failed, crashed, met a broken environment or could not merge | retries after 15, 30 and 45 minutes, at most 3 times at the same stage and round. After that it stops retrying and labels the issue `factory:needs-human` |
| Tests fail on `master` and not because of the change | the factory opens a "Tests fail on master" issue with `ai-dev`, so the factory fixes it. The blocked issue resumes once that issue is closed |
| The runner cannot run agents | opens one `factory-infra` issue, relaunches nothing, and closes the issue when the runner works again |
| An owner comment or an `ai-dev` label started no run | starts the run itself |
| Functional questions, a failed gate, a merge conflict, sensitive files | waits for you |

### The queue

When the factory has a free place, the supervisor launches the next issue by itself:

- **Which issues:** open issues you opened (or the factory opened with your token). An issue is skipped if it was already in the factory (`ai-dev` or a `factory:*` label) or carries `factory-hold`, `factory-infra`, `question`, `wontfix`, `duplicate` or `invalid`. Issues opened by anyone else are never launched.
- **In which order:** issues labelled `priority:high` go first and those labelled `priority:low` go last. Within each group, a Claude agent orders them: first bugs that block, then by dependencies, by size and by age. If the agent fails, the oldest issue goes first.
- **How many:** one at a time by default (`FACTORY_MAX_ACTIVE`). An issue holds its place while the factory goes on with it without you. One stopped for you (questions, a failed gate, a conflict) frees its place, so the next issue starts.
- The supervisor comments on each issue it launches with the reason and the rest of the queue, then adds `ai-dev`.

### Controls

- **`factory-hold` label:** the supervisor neither launches nor resumes that issue.
- **Remove `ai-dev`:** takes the issue out of the factory. The queue does not launch it again, because its `factory:*` label shows it has been in the factory.
- **Cancel a run** in the Actions tab: the supervisor does not relaunch cancelled runs. A comment by you resumes the issue.
- **Preview** what the supervisor would do, without acting: `node scripts/factory/supervisor.mjs --dry-run`. It needs an authenticated `gh` and, to order the queue, the Claude CLI.
- **Run the factory on your machine:** `scripts/factory/factory.sh <issue>`. The options are described in [WORKFLOW.md](doc/workflow/WORKFLOW.md#running-it).

### Troubleshooting

- **The supervisor does not run:** its loop stops if a run is cancelled or GitHub has an outage. Check that `FACTORY_SUPERVISOR` is not `off`, then start it once by hand from Actions → `supervisor` → *Run workflow*. That run restarts the loop.
- **An issue does not move:** read its last factory or supervisor comment and its `factory:*` label. The summary of each `supervisor` run in Actions lists its decision for every issue and the current queue.
- **Runner pods do not start:** see "Troubleshooting" in [deploy/arc/README.md](deploy/arc/README.md).

## Project status

This is an early, single-instance application intended for a small shared collection of around 100 products. It does not include user accounts. Collection depends on product pages remaining accessible and parseable; a failed check is retained as a collection outcome and does not replace the last successful price.

## Similar projects and search terms

Self-hosted Amazon and LeBoncoin price tracker · European price history · Go and SQLite price tracking · React Carbon price tracker
