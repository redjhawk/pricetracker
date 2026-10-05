# QA execution: multi-user-login-phase1

Status: executed (open mode and protected mode); no failures
Tested code/specification revisions: branch `ai-dev/issue-15-p8-docs`, HEAD `caff355`; specs in `doc/specifications/multi-user-login/`, `API_SPECIFICATION.md` § "Users and login (phase 1)".
Environment: `npm run dev` (Vite `http://localhost:5173`, Go API `http://127.0.0.1:3001`, `PRICEFOLLOWER_ENV=development`, dev data dir `.data` (gitignored, disposable; seeded sample items, list showed 8 items). Admin created with `PRICEFOLLOWER_ENV=development go run ./cmd/pricefollower admin-password`. Chromium headless via Playwright 1.63.0 from Node scripts in `.data/qa/`.
Executed at: open mode 2026-10-05 ~19:04–19:10 UTC; protected mode 2026-10-05 ~19:29–19:36 UTC
Tester: QA tester agent (second and third runs)

## Executed cases: open mode (second run)

| ID | Type | Interface URL | Action sequence | Expected | Actual | Status |
|---|---|---|---|---|---|---|
| OPEN-1 | Normal journey, open mode | `http://localhost:5173/` | Load page, wait for network idle | Items list works as before; "Log in" link shown | Tracked items list ("8 items tracked"), "Add item", one "Log in" link in header | Pass |
| OPEN-2 | Navigation | `http://localhost:5173/` → `/login` | Click "Log in" | Login page with username and password | URL `/login`, 2 inputs | Pass |
| OPEN-3 | Keyboard | `http://localhost:5173/login` | Press Tab 6 times from page start | Logical focus order | username → password → password visibility button → "Log in" submit → "Back to tracked items" link → body | Pass |
| OPEN-4 | Error state | `http://localhost:5173/login` | Type unknown username `nobody` and a wrong 16-character password, press Enter | One generic error, stays on login | "Could not log in — Incorrect username or password.", URL unchanged | Pass |
| OPEN-5 | Deep link / refresh | `http://localhost:5173/settings` | Open, reload | Page loads without redirect in open mode | URL `/settings` kept, tracked items page rendered (no login redirect) | Pass |
| OPEN-6 | Open-mode access (browser request, supporting only) | `http://localhost:5173/api/v1/items` | GET from the browser context | 200 | 200 | Pass (API check, not counted as interface test) |

## Executed cases: protected mode (third run)

Users created (disposable): `bob` (3 chars, first user), `cccccccccccccccccccccccccccccccc` (32 chars, "user C"). Passwords were disposable and are not recorded. API calls marked "browser fetch" were made with `fetch` from the logged-in page; they support but do not replace the interface checks.

| ID | Type | Req. | Interface URL | Action sequence | Expected | Actual | Status |
|---|---|---|---|---|---|---|---|
| ADM-1 | Normal journey | FR-ADMIN-002 | `http://localhost:5173/login` → `/admin` | Log in as `Admin` (capitalised) with the generated password | Admin sees only the users page | Redirect to `/admin`, only heading "Users", no items list; empty state "No users yet. Adding the first user turns on login and gives them the existing items and settings." | Pass |
| ADM-2 | Access refused | FR-ADMIN-002 | `http://localhost:5173/admin` (browser fetch) | GET `/api/v1/items`, GET `/api/v1/settings/claude-token`, PUT `/api/v1/settings` as admin | 403 FORBIDDEN | 403 `FORBIDDEN` "You do not have access to this page." for all three | Pass |
| ADM-3 | Deep link | FR-ADMIN-002 | `http://localhost:5173/` | Open `/` as admin | No items view | URL `/`, users page ("Users") rendered | Pass |
| ADD-1 | Validation boundary | FR-ADMIN-003 | `http://localhost:5173/admin`, "Add user" dialog | Username `ab` (2 chars), 12-char password, submit | Error on username, focus on username | "Use 3 to 32 letters, digits, dots, dashes or underscores."; dialog stays open; focus `add-user-username` | Pass |
| ADD-2 | Validation boundary | FR-ADMIN-003 | same | Username 33 × `a` | Same error, focus username | Same error; focus `add-user-username` | Pass |
| ADD-3 | Invalid characters | FR-ADMIN-003 | same | Username `bad name!`, then `bad/é` | Error, focus username | Same error both times; focus `add-user-username` | Pass |
| ADD-4 | Duplicate (case-insensitive, admin) | FR-ADMIN-003 | same | Username `Admin` | Duplicate error | "This username is already used."; focus `add-user-username` | Pass |
| ADD-5 | Validation boundary | API § Add a user | same | Username `bob`, 11-char password | Password error, focus password | "The password must have at least 12 characters."; focus `add-user-password` | Pass |
| ADD-6 | Normal journey, boundary | FR-ADMIN-003 | same | Username `bob` (3 chars), 15-char password | User created | Dialog closes, success "User added — bob can now log in.", table row `bob` / "Never" | Pass |
| ADD-7 | Duplicate (case-insensitive) | FR-ADMIN-003 | same | Username `BOB` | Duplicate error | "This username is already used."; focus `add-user-username` | Pass |
| ADD-8 | Validation boundary | FR-ADMIN-003 | same | Username 32 × `c`, 15-char password | User created | Created; table lists `bob`, user C | Pass |
| PROT-1 | Mode switch, logged out | FR-MODE-002, FR-AUTH-001 | `http://localhost:5173/` (new browser context) | Open `/` after first user created | Login screen | URL stays `/`, only "Log in to Price follower" form rendered; session `{"mode":"protected","user":null}`; items API 401 | Pass |
| PROT-2 | Deep link logged out | FR-AUTH-001 | `http://localhost:5173/settings` | Open | Login screen | URL `/settings`, login form only | Pass |
| PROT-3 | Admin API without/with wrong role | API § Modes | `http://localhost:5173/` (browser fetch) | GET `/api/v1/admin/users` logged out, then as user C | 401 / 403 | 401 / 403 | Pass |
| LOCK-1 | Lockout | FR-AUTH-005 | `http://localhost:5173/login` | 5 wrong passwords alternating `bob`/`BOB` | Generic error each time | "Incorrect username or password." × 5 | Pass |
| LOCK-2 | Lockout with correct password | FR-AUTH-005 | same | 6th attempt `bob` + correct password | Refused with wait message | "Could not log in — Too many failed attempts. Wait one minute and try again."; URL `/login`; browser fetch login → 429 `LOGIN_LOCKED` | Pass |
| LOCK-3 | Lockout scope | FR-AUTH-005 | same (browser fetch) | Log in as user C during bob's lockout | Unaffected | 200 | Pass |
| LOCK-4 | Lockout expiry | FR-AUTH-005 | same | Wait 62 s, log in as `bob` with correct password | Success | Redirect to `/`, "Tracked items" | Pass |
| INH-1 | First user inherits items | FR-MODE-002 | `http://localhost:5173/` as bob | View list, reload | Open-mode items shown | "8 items tracked" before and after reload | Pass |
| INH-2 | First user inherits settings | FR-MODE-002 | fresh `.data` DB; open mode then `http://localhost:5173/settings` as `dora` | In open mode PUT `/api/v1/settings/leboncoin-session` (value `datadome=QaInherit~…`); create admin; add `dora` as first user; log in as `dora`; GET LeBoncoin session; open `/settings` | Settings belong to dora | GET returned value `QaInherit~Abc123_test`, revision 1, status active; `/settings` loaded with 8 items | Pass |
| ISO-1 | Isolation | API § Modes | `http://localhost:5173/` as user C | Log in | Empty list | "0 items tracked", "No items tracked yet"; API `{"items":[]}` | Pass |
| ISO-2 | Other user's item | API § Modes | browser fetch as user C | GET and DELETE `/api/v1/items/sample-lbc-pool` (bob's) | 404 | 404 `ITEM_NOT_FOUND` for both; bob still has 8 items | Pass |
| ISO-3 | Other user's item, deep link | API § Modes | `http://localhost:5173/items/sample-lbc-pool` as user C | Open | Not found | "Could not load item — Tracked item was not found." with "Back to tracked items" | Pass |
| SAME-1 | Same URL, two users | API § Modes | browser fetch as user C, then `http://localhost:5173/` | POST `/api/v1/items` with bob's listing URL `https://www.leboncoin.fr/ad/jardin_plantes/3259094860` | Created for C | 201; C's page "1 item tracked"; bob still 8 | Pass |
| SAME-2 | Duplicate per user | API § Modes | browser fetch | C posts same URL again; bob posts it again | 409 for each | 409 `ITEM_ALREADY_TRACKED` for both | Pass |
| OUT-1 | Logout | FR-AUTH-003 | `http://localhost:5173/` as bob | Open header overflow menu, choose "Log out", then reload | Login screen; session ended | Login form shown; session `user:null`; items API 401; after reload still login form | Pass |
| RST-1 | Admin password reset ends sessions | FR-ADMIN-001, D-9 | `http://localhost:5173/` with saved admin cookie | Before: users page, session admin, `/api/v1/admin/users` 200. Run `admin-password` again ("Administrator password reset; administrator sessions ended."). After: reload with same cookie; log in with old then new password | Session ended; only new password works | After: login form, session `user:null`, admin users API 401; old password 401, new password 200 | Pass |
| NAR-1 | 360 px viewport | — | `http://localhost:5173/` as admin, 360×740 | Load | No horizontal overflow | `scrollWidth` 360; users table readable | Pass |

## Existing automated suite

`npx playwright test --reporter=line` (second run): 48 tests, 48 passed. The suite does not cover login/users.

## Findings

None. No failures observed.

## Coverage and limitations

- SAME-1 creates a real LeBoncoin item, which starts a background price collection against the live site; its result was not checked or needed.
- INH-2 ran on a second, fresh dev database (the first run's open-mode settings were empty, so inheritance of settings could not be shown there). Only the LeBoncoin session was used; the Claude token was not set because saving it requires live verification with Claude.
- "Log in" on the logged-out protected pages renders the login form at the requested URL (no URL change to `/login`); the specification only requires the login screen to be shown.
- 30-day session expiry (FR-AUTH-004) was not tested (time-dependent). Randomized exploration was not performed; the action order above is exact.
- Admin page fields from the full feature (temporary password flag, creation date, item count) are later phases and were not expected.

## Handoff and cleanup

Dev server stopped (processes terminated by PID). Throwaway scripts remain in the gitignored `.data/qa/`; password files and the saved admin cookie were deleted. The `.data` database now contains test users `admin`, `dora`; no credentials recorded here.
