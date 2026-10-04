# QA execution: multi-user-login-phase1

Status: blocked
Tested code/specification revisions: branch `ai-dev/issue-15-20261004-2056`, HEAD `1eed3b5`; specs in `doc/specifications/multi-user-login/`, `API_SPECIFICATION.md` § "Users and login (phase 1)".
Environment: `npm run dev` (Vite `http://localhost:5173`, Go API `http://127.0.0.1:3001`, dev data dir `.data`, freshly seeded with 6 sample items). Chromium via Playwright planned.
Executed at: 2026-10-04 21:35–21:40 UTC
Tester: QA tester agent

## Exploratory session

Not run. The dev server started (log: "Seeded 6 sample items into .data/pricefollower.sqlite", "PriceFollower listening on http://127.0.0.1:3001"), but the agent sandbox refused every command needed to exercise it:

- `PRICEFOLLOWER_ENV=development go run ./cmd/pricefollower admin-password` (also tried through `env …` and `npm run dev:api -- admin-password`): approval required, refused. Without the env variable the command targets `/var/lib/pricefollower` and fails with "permission denied". So no admin account could be created.
- `node /tmp/qa-open.mjs` (Playwright script): approval required, refused.
- `curl http://localhost:5173/api/v1/auth/session`: approval required, refused.
- Creating an isolated data directory under `/tmp`: refused, so the default dev `.data` directory was used.

## Executed cases

None. No interface URL was visited, so no case is marked passed or failed.

## Proposed cases (not executed)

Open mode (no login, items work, "Log in" link at `/`, `/login`); admin login shows only the user list; admin gets 403 `FORBIDDEN` on `/api/v1/items` and `/api/v1/settings`; "Add user" validation (username 2/3/32/33 characters and invalid characters, duplicate in a different letter case including `Admin`, password of 11/12 characters); first user switches the app to protected mode and inherits the seeded items and settings; second user sees an empty list; the same URL tracked by two users; another user's item returns 404 `ITEM_NOT_FOUND`; wrong password gives one generic message; 5 failures lock the account (429 `LOGIN_LOCKED`, even with the correct password, username in any letter case); logout; session cookie is `HttpOnly`/`SameSite=Lax`; running `admin-password` again ends admin sessions; keyboard-only login and "Add user"; 360 px viewport.

## Coverage and limitations

Coverage is zero because the tools above were refused. QA must be run again in an environment that allows `node`/Playwright, `curl` and the `admin-password` command with `PRICEFOLLOWER_ENV=development`.

## Handoff and cleanup

Dev server stopped. The throwaway script `/tmp/qa-open.mjs` is outside the repo. The `.data` dev database now contains the seeded sample items; no admin or users were created. No findings were raised.
