# Technical specification: login attempt limits by client address

Status: ready (issue #266)
Functional specification: [functional.md](functional.md) (FR-LOGIN-LIM-001..011; supersedes FR-AUTH-005 of [multi-user-login](../multi-user-login/functional.md)).
Test cases: [testcases.md](../../changes/2026-11-23-2151-issue-266-login-limit-failed-attempts-by-client-ad/testcases.md) (TC-001..TC-015).
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "Users and login", "Log in".

## Requirement mapping

| Technical ID | Functional IDs | Technical solution | Files |
| --- | --- | --- | --- |
| TS-LOGIN-LIM-001 | FR-LOGIN-LIM-001, -003, -005, FR-AUTH-005 | Failure map keyed by `loginKey{client, username}` (username lowercased); success deletes only its own key. | `internal/service/auth.go`, `internal/service/service.go` |
| TS-LOGIN-LIM-002 | FR-LOGIN-LIM-002, -004 | Lock check of the pair first in `Login`, before the store lookup and the password check; refused attempts are not recorded. | `internal/service/auth.go` |
| TS-LOGIN-LIM-003 | FR-LOGIN-LIM-006 | Pruning on each recorded failure, as today, on the new key; memory bounded by the serial password check rate. | `internal/service/auth.go` |
| TS-LOGIN-LIM-004 | FR-LOGIN-LIM-005 (corner cases), FR-LOGIN-LIM-001 | A failure during an active lock keeps `lockedUntil`; a cancelled wait returns the context error without recording. | `internal/service/auth.go` |
| TS-LOGIN-LIM-005 | FR-LOGIN-LIM-007, -008 | `clientAddress(r)` in the HTTP layer: valid `CF-Connecting-IP`, else the host of `RemoteAddr`; passed to `Service.Login`. | `internal/httpapi/auth.go` |
| TS-LOGIN-LIM-006 | FR-LOGIN-LIM-009, -010 | Bounded wait for the password-check slot for logins only: at most `maxWaitingLoginChecks` (4) waiting; beyond, `429 LOGIN_BUSY` at once, not counted. | `internal/service/auth.go` |
| TS-LOGIN-LIM-007 | FR-LOGIN-LIM-009 (unknown usernames), FR-LOGIN-LIM-004 | Dummy hash for unknown usernames built without a PBKDF2 computation, so an unknown-user login never waits outside the bounded wait. | `internal/service/auth.go` |
| TS-LOGIN-LIM-008 | FR-LOGIN-LIM-011 | No frontend change: `LoginPage` already shows the API `message` of any error in "Could not log in". | none (tests only) |

## Backend

### Service (`internal/service/auth.go`, `service.go`)

- Signature change: `func (s *Service) Login(ctx context.Context, client, username, password string) (string, store.User, error)`. `client` is the client address chosen by the HTTP layer (TS-LOGIN-LIM-005); the service treats it as an opaque string.
- Key type: `type loginKey struct{ client, username string }`, `username = strings.ToLower(username)`. `Service.loginFailures` becomes `map[loginKey]loginFailures` (comment: by client address and lowercased username); `loginFailures{count, lockedUntil, lastFailure}` is unchanged. `recordLoginFailure(key loginKey, now time.Time)` keeps its body (prune, then count; set `lockedUntil = now + loginLockTime` only when it is zero and `count >= maxFailedLogins`).
- `Login` order:
  1. `key`; under `s.mu`, `locked := now.Before(s.loginFailures[key].lockedUntil)`; locked → `429 LOGIN_LOCKED` (existing error) with no store access and no password check (TS-LOGIN-LIM-002).
  2. `store.UserByUsername`; unknown username uses `dummyPasswordHash()` (TS-LOGIN-LIM-007).
  3. `matches, err := s.checkPassword(ctx, stored, password)`; `err` is the context error or the busy `*Error` and is returned as is, without recording a failure (cancelled or busy logins are not counted).
  4. Unknown user or no match → `s.recordLoginFailure(key, s.now())` (time read after the check, so the lock lasts 1 minute after the 5th failure even when the check waited), `401 INVALID_CREDENTIALS`.
  5. Success → `delete(s.loginFailures, key)` under `s.mu`, then sessions as today.
- Pair lock rules (FR-LOGIN-LIM-002..006) are the existing per-username rules applied to the pair: refused attempts are not counted and do not extend the lock; a failure recorded while the pair is still locked (an attempt that passed the check before the lock) keeps `lockedUntil`; pruning deletes the pair's entry once its lock has ended, so the next failure starts a new count at 1; unlocked entries older than `failureMemory` (1 hour) are deleted.
- Bounded memory (TS-LOGIN-LIM-003): an entry exists only for a pair with a failure within the hour or an active lock. Entries are only created by completed password checks, which run one at a time (about 1 s each on the device), so the map holds at most a few thousand entries; the O(n) prune per failure is cheap at that size. Locked and busy refusals never create entries.

### Bounded wait for password checks (TS-LOGIN-LIM-006)

- `hashSlot` (size 1, process-wide) stays the only PBKDF2 limit. New package state: `const maxWaitingLoginChecks = 4` and `var waitingLoginChecks atomic.Int32` (logins currently waiting for `hashSlot`). 4 waiting checks is about 5 s of wait on the device at worst; a household never has more simultaneous logins.
- `acquireLoginCheck(ctx) error`:
  ```go
  select { case hashSlot <- struct{}{}: return nil; default: }
  if waitingLoginChecks.Add(1) > maxWaitingLoginChecks {
      waitingLoginChecks.Add(-1)
      return loginBusyError() // &Error{Status: 429, Code: "LOGIN_BUSY", Message: "Too many login attempts are being checked. Try again in a moment."}
  }
  defer waitingLoginChecks.Add(-1)
  select { case hashSlot <- struct{}{}: return nil; case <-ctx.Done(): return ctx.Err() }
  ```
  A login is refused busy only when `maxWaitingLoginChecks` other logins are counted waiting (FR-LOGIN-LIM-010). The counter can lag by one for the instant between a waiter taking the slot and its decrement; this only matters under a flood, where busy is the intended answer.
- `checkPassword(ctx, stored, password) (bool, error)` replaces `passwordMatches`: parses the stored hash (malformed → `false, nil`, as today), calls `acquireLoginCheck`, releases with `defer func() { <-hashSlot }()`, calls `s.beforePasswordCheck(ctx)` when set (test hook), computes `pbkdf2.Key`, returns `ctx.Err()` if the context ended, else the constant-time comparison.
- `HashPassword` (user creation, admin password command, no more dummy hash) keeps `pbkdf2Key` with the unbounded, cancellable wait and does not touch `waitingLoginChecks`: the excluded flows keep waiting as today (TC-011 corner case).
- Test hooks, for tests only: `Options.BeforePasswordCheck func(context.Context)` stored in `Service.beforePasswordCheck` (called while a login holds the slot; a test blocks in it to hold a running check) and exported `func WaitingLoginChecks() int` (returns `waitingLoginChecks`; tests poll it to know that N logins are waiting). Service tests may instead occupy `hashSlot` directly (`hashSlot <- struct{}{}` / `<-hashSlot`).

### Dummy hash (TS-LOGIN-LIM-007)

- Replace `dummyHashOnce`/`dummyHash` (today computed on first use through `HashPassword(context.Background())`, an unbounded wait outside the busy rule) with `dummyPasswordHash() string` returning `pbkdf2-sha256$<passwordIterations>$<16 zero bytes b64>$<32 zero bytes b64>`. Verifying it runs PBKDF2 with the current cost, so unknown usernames still cost the same check (no timing difference); no password yields an all-zero key in practice.

### HTTP (`internal/httpapi/auth.go`, TS-LOGIN-LIM-005)

- `clientAddress(r *http.Request) string`:
  - `value := strings.TrimSpace(r.Header.Get("CF-Connecting-IP"))`; if `netip.ParseAddr(value)` succeeds, return `addr.WithZone("").Unmap().String()` (bare IPv4 or IPv6 only; `203.0.113.1:443`, `not-an-ip` and empty values fail).
  - Else `host, _, err := net.SplitHostPort(r.RemoteAddr)` (falls back to the whole `RemoteAddr` on error); if `host` parses as an address, return its canonical form as above, else `host`. The port is never part of the address.
- Login handler: `s.service.Login(r.Context(), clientAddress(r), body.Username, body.Password)`. Errors go through the existing `serviceError` (status, code, message of `*service.Error`); no session cookie on any error. `400 INVALID_JSON` is answered before `Login`, so it is not counted.
- The header is trusted because the application is reachable only through Cloudflare (functional spec); no proxy allow-list. Documented in `deploy/README.md`.

### Persistence, migrations, concurrency

- No schema change or migration; failures stay in memory and are lost on restart, as today.
- `s.mu` guards `loginFailures`; `waitingLoginChecks` is atomic; `hashSlot` serialises computations. No lock is held while waiting for or computing a hash.

## API

- `POST /api/v1/auth/login`: request and success response unchanged. Changed rule for `429 LOGIN_LOCKED` (per client address and username). New `429 LOGIN_BUSY`, "Too many login attempts are being checked. Try again in a moment.", answered at once, not counted. Updated in `API_SPECIFICATION.md`.
- Compatibility: additive error code with the same `{ "error": { "code", "message" } }` envelope; the frontend shows `message` for any code.

## Frontend

Not affected (TS-LOGIN-LIM-008): `LoginPage` sets its error from `ApiError.message` for any failed login, shows it in the existing `InlineNotification` "Could not log in", focuses it, re-enables the form and stays on the page; the busy state of the button is unchanged. Only e2e tests are added.

## Files to change

- `internal/service/auth.go`: key type, `Login` signature and order, `acquireLoginCheck`, `checkPassword`, `dummyPasswordHash`, `WaitingLoginChecks`, busy error.
- `internal/service/service.go`: `loginFailures map[loginKey]loginFailures`, `beforePasswordCheck` field, `Options.BeforePasswordCheck`.
- `internal/httpapi/auth.go`: `clientAddress`, login call.
- `internal/service/auth_test.go`: adapt `TestLoginLockoutAndSessionExpiry`, `TestFailureDuringLockKeepsItAndOldEntriesArePruned` and the other `Login` calls to the new signature and `loginKey`.
- `deploy/README.md`: one sentence under "Server setup": the server takes the client address of login limits from `CF-Connecting-IP` and must only be reachable through Cloudflare.
- Already done in this stage: `API_SPECIFICATION.md` (login contract) and the lockout line of `doc/specifications/multi-user-login/technical.md`.

## Tests

| Test case | Test file | Notes |
| --- | --- | --- |
| TC-001, TC-002, TC-003, TC-005, TC-006 | `internal/service/login_attempt_limits_test.go` | `newSessionService`, `service.now` clock, `Login(ctx, "198.51.100.1", ...)`; TC-006 bound reads `len(service.loginFailures)` and `service.loginFailures[loginKey{...}]` |
| TC-004, TC-007, TC-008, TC-011, TC-012 | `internal/service/login_attempt_limits_test.go` | hold a check with `hashSlot <- struct{}{}` (or `Options.BeforePasswordCheck`), start logins in goroutines, wait with `WaitingLoginChecks()`; Max is `maxWaitingLoginChecks`; TC-007 may seed the pair with `recordLoginFailure` |
| TC-009, TC-010, TC-013 | `internal/httpapi/login_attempt_limits_test.go` | `newSettingsServer` (TC-009, TC-010, TC-013 locked) and a server on `service.NewWithOptions(cfg, database, service.Options{BeforePasswordCheck: ...})` for TC-013 busy; `req.RemoteAddr` and `CF-Connecting-IP` set per request; reuse `assertError` |
| TC-014, TC-015 | `tests/responsive-layout.spec.ts` | new tests named `TC-014 (#266) ...`, `TC-015 (#266) ...`; the file's `mockApi` with a protected session, and a `page.route("**/api/v1/auth/login", ...)` that takes precedence over the file's context route; reuse `expectNoHorizontalScroll` for 320 px |

The e2e tests go into an existing spec because a new spec file needs `playwright.config.ts` (`testMatch`), which this change cannot edit; `responsive-layout.spec.ts` already covers the login page and holds the narrow-screen helpers. Moving them to a dedicated `tests/login-attempt-limits.spec.ts` is an optional manual follow-up (add it to `testMatch` in its own PR).

## Refactoring

- R-1 `passwordMatches` → `Service.checkPassword` with `acquireLoginCheck`. Reason: logins need a bounded, refusable wait and a per-service test hook, while user creation keeps the unbounded wait. Scope: `auth.go` only; `pbkdf2Key` stays for `HashPassword`. Risk: low; a missed release would block all hashing, covered by every login test.
- R-2 dummy hash without computation. Reason: the lazy `HashPassword(context.Background())` waits for the slot without bound or busy refusal (fails TC-011 unknown username, TC-004). Scope: `auth.go`. Risk: low; same PBKDF2 cost per unknown-user check.

## Manual follow-ups

- None required. Optional: dedicated e2e spec file (see Tests).
