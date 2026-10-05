# QA execution: multi-user-login-phase1

Status: partially executed (open mode passed; protected-mode cases blocked)
Tested code/specification revisions: branch `ai-dev/issue-15-p8-docs`, HEAD `5b90e94`; specs in `doc/specifications/multi-user-login/`, `API_SPECIFICATION.md` § "Users and login (phase 1)".
Environment: `npm run dev` (Vite `http://localhost:5173`, Go API `http://127.0.0.1:3001`, `PRICEFOLLOWER_ENV=development`, dev data dir `.data` (gitignored, disposable; server log "Seeded 6 sample items into .data/pricefollower.sqlite", list showed 8 items). Chromium headless via Playwright 1.63.0 from a Node script.
Executed at: 2026-10-05 ~19:04–19:10 UTC
Tester: QA tester agent (second run)

## Executed cases

| ID | Type | Interface URL | Action sequence | Expected | Actual | Status |
|---|---|---|---|---|---|---|
| OPEN-1 | Normal journey, open mode | `http://localhost:5173/` | Load page, wait for network idle | Items list works as before; "Log in" link shown | Tracked items list ("8 items tracked"), "Add item", one "Log in" link in header | Pass |
| OPEN-2 | Navigation | `http://localhost:5173/` → `/login` | Click "Log in" | Login page with username and password | URL `/login`, 2 inputs | Pass |
| OPEN-3 | Keyboard | `http://localhost:5173/login` | Press Tab 6 times from page start | Logical focus order | username → password → password visibility button → "Log in" submit → "Back to tracked items" link → body | Pass |
| OPEN-4 | Error state | `http://localhost:5173/login` | Type unknown username `nobody` and a wrong 16-character password, press Enter | One generic error, stays on login | "Could not log in — Incorrect username or password.", URL unchanged | Pass |
| OPEN-5 | Deep link / refresh | `http://localhost:5173/settings` | Open, reload | Page loads without redirect in open mode | URL `/settings` kept, tracked items page rendered (no login redirect) | Pass |
| OPEN-6 | Open-mode access (browser request, supporting only) | `http://localhost:5173/api/v1/items` | GET from the browser context | 200 | 200 | Pass (API check, not counted as interface test) |

Supporting API check: `curl http://localhost:5173/api/v1/auth/session` returned `{"mode":"open","user":null}`.

## Existing automated suite

`npx playwright test --reporter=line`: 48 tests, 48 passed, 0 failed (2.0 min). The suite does not cover login/users.

## Blocked cases

All remaining cases need an admin account, which could not be created:

- `go run ./cmd/pricefollower admin-password` without `PRICEFOLLOWER_ENV=development` targets `/var/lib/pricefollower` and failed with "permission denied".
- Any command with an environment prefix (`PRICEFOLLOWER_ENV=…` / `PRICEFOLLOWER_DATA_DIR=…`) requires approval in this sandbox and was refused; creating an isolated temp dir under `/tmp` was also refused, so the default gitignored `.data` dir was used.

Blocked (not run): admin login and admin sees only users page; admin 403 on items/settings; "Add user" validation (username 2/3/32/33 chars, invalid characters, case-insensitive duplicate incl. `Admin`, password 11/12 chars) and focus after failure; first user switches to protected mode (unauthenticated redirects to login) and inherits items/settings; lockout after 5 failures for 1 minute even with correct password, then success; logout; second user isolation and 404 on another user's item; same URL tracked by two users; deep link/refresh while logged in and logged out in protected mode; `admin-password` reset ends admin sessions; 360 px viewport.

## Findings

None raised. No failures observed in executed cases.

## Coverage and limitations

Only open mode was exercised. Protected-mode behavior (the core of this change) is untested at interface level and must be run in an environment that allows `PRICEFOLLOWER_ENV=development go run ./cmd/pricefollower admin-password` (or equivalent) against the dev server's data dir. Randomized exploration was not performed; the action order above is exact.

## Handoff and cleanup

Dev server stopped. The throwaway script `.data/qa/open.mjs` is in the gitignored `.data` directory. No admin or users were created; no credentials recorded.
