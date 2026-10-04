# Implementation: multi-user login phase 1

Status: implemented, awaiting independent review. Developer agent, 2026-10-04.

## Commits (stacked, one per planned PR)

Reordered after review (REV-003) so that item and settings scoping land before any HTTP route that can create users or sessions. History before the reorder: `3f90121`. Sizes are `scripts/pr-size.sh <previous part's last commit>`.

| Part | Commits | Scope | Changed lines |
| --- | --- | --- | --- |
| 1 store | `064d106` | users/sessions tables and store functions, items rebuilt with `owner_id`, first user inherits items | 438 |
| 2 auth service | `411ed69`, `be988dd` (fix) | PBKDF2 hashing, lockout, login/session/logout service, user creation, `pricefollower admin-password`, installer, deploy wrapper, `DEPLOYMENT.md`; review fixes | 471 |
| 3 settings | `109fa63` | settings tables keyed by owner, request owner from context (`Principal`, moved here from the HTTP part), item owner's settings for collections and reviews | 372 |
| 4 items | `efceed5` | item operations scoped to the owner, per-user duplicates; service-level cross-user test | 106 |
| 5 HTTP | `e4bccef` | `/auth/*` and `/admin/users` endpoints, cookie, mode and role middleware; HTTP cross-user test | 365 |
| 6 web | `9ea4e2c`, `9f693ca` (fix) | session loading, login page, Log in / Log out, administrator page, Playwright mocks; review fix | 432 |
| 7 docs | this commit | README and this record | small |

Accepted intermediate state (REV-003 decision): after part 5 and before part 6, a user created through the raw admin API makes the web UI answer 401 with no login page; no data is exposed. Before part 5 no route creates users or sessions; only the CLI can create `admin`, which has no HTTP access until part 5 (and then only to auth and admin routes).

## Requirement mapping

| Requirement | Change |
| --- | --- |
| FR-ADMIN-001, D-14, D-22 | `cmd/pricefollower/main.go` `admin-password`; `Service.ResetAdminPassword`; `Store.CreateAdminOrResetPassword` deletes admin sessions on reset |
| FR-MODE-001/002, D-25, D-26 | `httpapi/auth.go` `handleAPI`: mode from `HasRegularUser` per request; `GET /auth/session` |
| FR-AUTH-001..003 | login/logout endpoints, `pricefollower_session` cookie, `LoginPage.tsx`, Log out in `AppMenu` |
| FR-AUTH-004, D-8 | sessions expire 30 days after login (`SessionDuration`), not extended |
| FR-AUTH-005, D-11 | in-memory lockout per lowercased username, 429 `LOGIN_LOCKED` |
| FR-ADMIN-002/003, D-10, D-17, D-20, D-24 | `/admin/users`, `AdminPage.tsx` (username, last login; add-user modal with field errors) |
| FR-SHARE-001/005, D-19 | `items.owner_id`, `UNIQUE(owner_id, canonical_url)`, `Service.ownedListing` returns not found for other owners |
| FR-SETTINGS-001, D-4, D-23 | `leboncoin_session`/`claude_token` keyed by `owner_id`; collections and reviews use the item owner |
| FR-SHARE-006, D-5, D-26 | `Store.CreateUser` moves owner 0 items and settings to the first user in the same transaction |
| Deployment instruction | installer prints the command; `deploy-armv6.sh --admin-password`; `DEPLOYMENT.md`, `README.md` |

## Recorded refactoring

- R-1 (from the technical specification): owner id threaded through the settings store functions and the item list/ids/delete/duplicate functions. Item reads by id stay unscoped in the store; the service checks the owner once with `ownedListing` (narrower than "every store function gains `ownerID`", same behavior, smaller diff). Background collection and review workers work by item id and read the owner from the item.
- `ensureItemColumn` now uses a new `hasColumn` helper shared with the owner migrations; `rebuildTable` runs the table rebuilds with foreign keys off and a `foreign_key_check`. Reason: three migrations need the same column check and rebuild steps. Impact: store package only.
- The former `handleAPI` is renamed `handleData`; the new `handleAPI` applies the access rules first.

## Deviations from the specification

- Administrator users table uses Carbon `Table` components inside `TableContainer`/`TableToolbar`, as `TrackedItemsPage` does, rather than the render-prop `DataTable`.
- `model.ClaudeToken` gains a non-serialized `OwnerID` so a review can record a rejection on its owner's token.
- After login and logout the frontend reloads the page (`window.location.assign`) instead of client navigation, to reset all page state.

## Review fixes

- REV-001 (`be988dd`): `recordLoginFailure` no longer resets an active lock; a failure during the lock keeps `lockedUntil` unchanged. Test `TestFailureDuringLockKeepsItAndOldEntriesArePruned`.
- REV-002 (`be988dd`): `passwordIterations = 100_000`; a size-1 channel semaphore (`hashSlot`) allows one PBKDF2 computation at a time and the wait honours `ctx`. `technical.md` updated. The ~1 s figure on ARMv6 is an estimate; no device measurement was possible here.
- REV-003: commits reordered as above; per-boundary checks below.
- REV-004 (`be988dd`): each failure prunes expired locks and unlocked entries whose last failure is older than 1 hour (`failureMemory`). Same test as REV-001.
- REV-005 (`9f693ca`): the add-user modal focuses the invalid field in an effect after the error renders and the input is enabled.
- REV-006: rejected (code); `DEPLOYMENT.md` now says not to run the command where output is logged (`be988dd`).

## Checks run

Per part boundary after the reorder (each checked out on the branch with `git reset --hard <hash>`, since detached checkout and worktrees were blocked): `064d106`, `be988dd`, `109fa63`, `efceed5`, `e4bccef`: `go vet ./...` clean and `go test ./...` all 5 test packages `ok`. `9f693ca` and `4959a6a`: same, plus `npm run build` success. At the tip, `npx playwright test`: 48 passed.

Before the reorder:

- After every commit: `go build ./... && go vet ./... && go test ./...`: all packages `ok`; `gofmt -l .`: no files.
- Frontend commit: `npm run build` (`tsc -b && vite build`): success. No lint script exists in `package.json`.
- `npx playwright test` (all 6 suites, mocked API): 48 passed.
- New Go tests: legacy items and settings migration, same URL for two owners, first-user inheritance only once, case-insensitive duplicate usernames including `admin`, 30-day session expiry, admin reset ends sessions and replaces the password, lockout after 5 failures for 1 minute, user validation codes, open/protected/admin/user access matrix, cross-user 404 on item endpoints, settings isolation.

## Limitations

- `bash -n` on the changed scripts and a manual run of `pricefollower admin-password` could not be executed in this environment (commands blocked by the sandbox); the command path is covered only by the service tests.
- Running `admin-password` opens the store like the server does, which marks AI reviews pending at that moment as interrupted; the running server still overwrites them with the final result when they finish.
- Lockout state is in memory and resets on restart (as specified).
- No Playwright test covers the login and administrator pages yet; QA covers them.
