# Technical specification: multiple local users and login (phase 1)

Status: ready
Functional specification: [functional.md](functional.md) and [delivery-plan.md](delivery-plan.md) (D-19 to D-26 take precedence), phase 1 scope in [functional-step.md](../../changes/multi-user-login-phase1/functional-step.md).
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "Users and login".

## Requirement mapping

| Functional ID | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- |
| FR-ADMIN-001, D-14, D-22 | `pricefollower admin-password` subcommand: create or reset `admin`, print generated password, delete admin sessions | `cmd/pricefollower`, `internal/service/auth.go`, `internal/store/users.go`, scripts, `DEPLOYMENT.md` | Service test; run twice, old password and session refused |
| FR-MODE-001, D-25 | Mode derived per request: protected if a `role='user'` row exists | `internal/httpapi/auth.go` | Without users all existing endpoints answer without cookie |
| FR-MODE-002, D-26 | Same; first user creation switches mode immediately | same | After creating a user, `GET /items` without cookie is 401 |
| FR-AUTH-001..003 | Login/logout/session endpoints, session cookie, frontend login page | `internal/httpapi/auth.go`, `src/components/LoginPage.tsx`, `src/App.tsx` | HTTP tests; QA login, wrong password, logout+reload |
| FR-AUTH-004, D-8 | Session row expires 30 days after login | `internal/store/users.go` | Store test with clock 31 days later |
| FR-AUTH-005, D-11 | In-memory limiter: 5 consecutive failures per lowercased username, 1 minute lock, 429 | `internal/service/auth.go` | Service test with injected clock |
| FR-ADMIN-002, -003, -007, D-10, D-17, D-20, D-24 | Admin endpoints and admin page (DataTable + add-user Modal) | `internal/httpapi/admin.go`, `src/components/AdminPage.tsx` | HTTP tests; QA add user, duplicate, short password |
| FR-SHARE-001, -005, D-19 | `items.owner_id`, unique `(owner_id, canonical_url)`, all item queries scoped | `internal/store/*.go`, `internal/service/*.go`, `internal/httpapi/*.go` | Two users add same URL; other user's id gives 404 |
| FR-SETTINGS-001, D-4, D-23 | Settings tables keyed by `owner_id`; scheduler and AI reviews use item owner | `internal/store/store.go`, `claude_token.go`, `internal/service` | Store/service tests per owner |
| FR-SHARE-006, D-5, D-26 | First user creation reassigns owner `0` items and settings in the same transaction | `internal/store/users.go` | Store test |
| Deployment (user instruction) | Installer prints the command; deploy wrapper option `--admin-password` | `scripts/install-pricefollower.sh`, `scripts/deploy-armv6.sh`, `DEPLOYMENT.md` | `bash -n`; manual run documented |

## Backend

### Schema (`internal/store`)

```sql
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE COLLATE NOCASE,
  role TEXT NOT NULL CHECK (role IN ('admin', 'user')),
  password_hash TEXT NOT NULL,
  created_at TEXT NOT NULL,
  last_login_at TEXT
);
CREATE TABLE IF NOT EXISTS sessions (
  token_hash TEXT PRIMARY KEY,           -- hex SHA-256 of the cookie token
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL
);
```

- Owner key: `owner_id INTEGER NOT NULL DEFAULT 0`, where `0` means "open mode owner" (no user). A non-null sentinel avoids SQLite NULL-distinct uniqueness pitfalls and keeps queries `WHERE owner_id = ?`. No foreign key, since `0` has no user row; users are never deleted in phase 1.
- `items`: today `canonical_url` is `UNIQUE`, which forbids FR-SHARE-005. Migration (once, when `owner_id` is missing): in one transaction with `PRAGMA foreign_keys=OFF` set before it, create `items_new` with the current columns plus `owner_id` and `UNIQUE (owner_id, canonical_url)`, copy rows with owner `0`, drop `items`, rename, recreate `items_next_check`, run `PRAGMA foreign_key_check`, re-enable foreign keys. Child tables keep referencing `items(id)` by name.
- `leboncoin_session` and `claude_token`: today a single row `CHECK (id = 1)`. Migration (once, when `owner_id` is missing): rebuild each with `owner_id INTEGER PRIMARY KEY` replacing `id`, copying row `1` as owner `0`. Store reads use `INSERT OR IGNORE ... (owner_id, revision) VALUES (?, 0)` before reading, so every owner implicitly has an empty row. Revisions stay per row.
- All migrations are idempotent, run in `createSchema`, and preserve data.

### Store functions

`CreateAdminOrResetPassword(hash, now) (created bool)` (also deletes admin sessions), `CreateUser(username, hash, now)` (role `user`; in the same transaction, if no other `role='user'` exists, `UPDATE items/leboncoin_session/claude_token SET owner_id = new id WHERE owner_id = 0`; settings rows for the new id are deleted first so the inherited row wins), `UserByUsername`, `ListUsers` (role `user`, ordered by username), `HasRegularUser`, `CreateSession`, `SessionUser(tokenHash, now)` (ignores expired rows), `DeleteSession`, `DeleteExpiredSessions(now)` (called on login), `RecordLogin(userID, now)`. Duplicate username maps to `ErrUsernameTaken`.

Existing item, history, review, goal, settings functions gain an `ownerID int64` argument and filter by it; mismatched id behaves as not found. `DueIDs` and the AI review worker return/keep the item's `owner_id` so collection uses `LeboncoinSession(owner)` and reviews use `ClaudeToken(owner)` (FR-SETTINGS-001). `IsCanonicalTracked(owner, url)`.

### Service (`internal/service/auth.go`)

- Passwords: stdlib `crypto/pbkdf2` (Go 1.25 in `go.mod`), SHA-256, 600 000 iterations, 16-byte random salt, stored as `pbkdf2-sha256$600000$<salt b64>$<key b64>`; compare with `crypto/subtle`. No new dependency. On ARMv6 a login costs roughly a second, acceptable for rare logins.
- Username rules (FR-ADMIN-003): `^[A-Za-z0-9._-]{3,32}$`; `admin` collides through NOCASE uniqueness (409). Password: at least 12 characters (Unicode code points), at most 256.
- Generated admin password: 18 random bytes, base64url (24 characters).
- Session token: 32 random bytes, base64url in cookie; only the SHA-256 hex is stored. Expiry `now + 30 days`, not extended.
- Lockout: `map[lowercase username]{failures int; lockedUntil time.Time}` behind a mutex. Locked usernames are refused (429) without checking the password; the 5th consecutive failure sets `lockedUntil = now + 1 minute`; success or expiry of the lock resets. Unknown usernames are counted too (no enumeration). Unknown usernames still run one hash comparison against a fixed dummy hash.
- Context for request scope: `service.Principal{UserID int64; Role string}` stored in the request context by the middleware; open mode uses owner `0`.

### HTTP (`internal/httpapi/auth.go`, `admin.go`)

- Cookie `pricefollower_session`: `HttpOnly`, `SameSite=Lax`, `Path=/`, `Max-Age` 30 days; `Secure` only when the request arrived over TLS (the device serves plain HTTP on the LAN). Logout clears it with `Max-Age=-1`.
- Middleware on `/api/` in this order: auth endpoints `/api/v1/auth/*` always reachable. Resolve the cookie to a user (if any). If the user is `admin`: `/api/v1/admin/*` allowed, everything else 403 `FORBIDDEN`. `/api/v1/admin/*` without an admin user: 401 `AUTH_REQUIRED` if no session, 403 if a regular user. Otherwise, if a regular user exists (protected mode): no valid session gives 401 `AUTH_REQUIRED`; a regular user proceeds with owner = own id. Open mode with no session: owner `0`.
- Frontend routes (`/`, `/items/...`, `/login`, `/admin`) keep serving `index.html` without auth; the SPA decides what to show from `GET /api/v1/auth/session`. No data is in the HTML.
- CSRF: SameSite=Lax blocks cross-site cookie POSTs; mutating endpoints already require JSON bodies.

### Device command (`cmd/pricefollower`)

`pricefollower admin-password`: loads the same config (`PRICEFOLLOWER_DATA_DIR`), opens and migrates the store, calls the service, prints `Administrator account created.` or `Administrator password reset; administrator sessions ended.` and `Username: admin` / `Password: <generated>` to stdout, exits 0; any other argument prints usage and exits 2. It does not start the server; the running service sees changes at the next request because sessions are read from SQLite. Not reachable over HTTP.

## Frontend

- `src/api/auth.ts`: `getSession`, `login`, `logout`, `listUsers`, `createUser` via the existing client. A 401 from any other call reloads the session state (shows login).
- `App.tsx` loads the session first (Carbon `Loading` while pending). State: `open` (today's app plus a header `HeaderGlobalAction`/link "Log in" to `/login`), `protected` without user (only `LoginPage`), regular user (today's app; AppMenu gains "Log out"), admin (only `AdminPage`, header with "Log out" in the menu, no Settings).
- `LoginPage`: Carbon `Form`, `TextInput` (username, `autoComplete="username"`), `PasswordInput` (`autoComplete="current-password"`), primary `Button` with busy state, `InlineNotification` kind `error` showing the API message (401 generic, 429 wait message). On success navigate to `/` (or `/admin` for admin). In open mode a "Back" link returns to `/`.
- `AdminPage`: heading "Users", Carbon `DataTable` columns Username, Last login (localized date or "Never"), toolbar `Button` "Add user" opening a `Modal` with `TextInput` username, `PasswordInput` password (helper "At least 12 characters"), field-level `invalid`/`invalidText` from API errors (`INVALID_USERNAME`, `PASSWORD_TOO_SHORT`, `USERNAME_TAKEN`); on 201 close, refresh list, success `InlineNotification`. Empty state text "No users yet. Adding the first user turns on login and gives them the existing items and settings."
- Accessibility: labelled inputs, focus moves to the first invalid field or notification, menu items reachable by keyboard.

## API

Defined in [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) "Users and login": `GET /auth/session`, `POST /auth/login`, `POST /auth/logout`, `GET /admin/users`, `POST /admin/users`, plus 401/403/404/429 semantics and owner scoping of every existing endpoint. Existing request/response shapes are unchanged.

## Scope and refactoring

- Refactor R-1 (required, done inside PR 4/5, no separate behavior): thread an owner id through store and service signatures. Needed because every query must be scoped; no alternative keeps the code simple. Risk: missed query; mitigated by tests per endpoint for cross-user 404.
- Declined: router library, session library, x/crypto bcrypt (dependency churn; stdlib PBKDF2 suffices).
- Out of scope: password change, user removal/rename, sharing, other list columns.

## Delivery split

Stacked PRs to `master`, each building and passing `go test ./...` and `npm run build` alone, at most ~450 changed lines (500 hard):

1. Specifications: functional cherry-pick, this file, API changes, `doc/changes` records.
2. Store: users/sessions tables, user/session store functions, owner migration of `items` and settings tables (functions still called with owner `0`), tests.
3. Auth service (hashing, lockout, admin reset, create user with inheritance) + `admin-password` subcommand + scripts + `DEPLOYMENT.md`.
4. HTTP: auth endpoints, middleware (mode, roles), admin endpoints, tests.
5. Ownership: owner id through item/settings/review/goal handlers, scheduler and AI worker use item owner, cross-user tests (R-1).
6. Frontend: session loading, login page, Log in/Log out, admin page.
7. Workflow evidence docs (review, decisions, commit step, QA).

If PR 5 exceeds the limit, split store/service scoping from HTTP scoping.

## Verification and unresolved questions

- Go tests: migration from an existing database (items, settings preserved, owner 0), same URL for two owners, inheritance on first user only, session expiry at 30 days, lockout 5/1 min, admin reset ends sessions, middleware matrix (open/protected x none/user/admin), cross-user 404.
- Frontend: `npm run build`, lint.
- QA: open mode unchanged; Log in as admin; add user (short password, duplicate `Alice`/`alice`); protected mode login screen; wrong password generic error; 6th attempt wait message; logout+reload; two users same URL independent; settings isolated.
- Unresolved functional questions: none.
