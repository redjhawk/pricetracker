# Test cases: login attempt limits by client address (issue #266)

Sources: functional specification [login-attempt-limits](../../specifications/login-attempt-limits/functional.md) (FR-LOGIN-LIM-001 to FR-LOGIN-LIM-011), the superseded [FR-AUTH-005](../../specifications/multi-user-login/functional.md), and use cases [UC-LOGIN-LIM-001](../../use-cases/login-attempt-limits/UC-LOGIN-LIM-001-locked-out-after-repeated-failures-from-one-client.md), [UC-LOGIN-LIM-002](../../use-cases/login-attempt-limits/UC-LOGIN-LIM-002-log-in-while-another-client-is-locked-out.md) and [UC-LOGIN-LIM-003](../../use-cases/login-attempt-limits/UC-LOGIN-LIM-003-log-in-when-too-many-password-checks-are-waiting.md).

Conventions:

- **Service tests** (`internal/service`, package-internal) use `newSessionService` (real SQLite database, normal migrations), the lowered test PBKDF2 cost, and an injected clock (`service.now`). Users `alice` and `bob` exist with password `correct horse battery`. Addresses: A = `198.51.100.1`, B = `198.51.100.2`. The service's login receives the client address chosen by the technical specification; these tests pass it directly.
- **HTTP tests** (`internal/httpapi`) send `POST /api/v1/auth/login` through `server.Handler()` on a real service and SQLite database (`newSettingsServer`), setting `req.RemoteAddr` (connection address, with a port) and, when stated, the `CF-Connecting-IP` header.
- **Held password checks:** the test keeps a password check running (for example by occupying the password-check slot, or a test hook defined by the technical specification) so that further checks must wait. "Max" is the fixed maximum number of waiting checks set by the technical specification (at least 1). A call that returns while checks are held has not waited for, nor run, a password check.
- **Locked answer:** HTTP 429, code `LOGIN_LOCKED`, message "Too many failed attempts. Wait one minute and try again." **Busy answer:** HTTP 429, the busy code defined in `API_SPECIFICATION.md`, message "Too many login attempts are being checked. Try again in a moment." **Wrong credentials:** HTTP 401 `INVALID_CREDENTIALS`.
- **E2E tests** (Playwright, new spec file in `tests/`, added to `testMatch` in `playwright.config.ts`) mock the API with `page.route`/`context.route`, as the existing specs do; client addresses cannot vary there, so address rules are proved by the HTTP tests.
- Existing tests `TestLoginLockoutAndSessionExpiry` and `TestFailureDuringLockKeepsItAndOldEntriesArePruned` are adapted to the per-pair rule; they keep proving session expiry, logout and pruning.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-AUTH-005 | TC-002, TC-003 |
| FR-LOGIN-LIM-001 | TC-001, TC-003, TC-008, TC-009 |
| FR-LOGIN-LIM-002 | TC-002, TC-013 |
| FR-LOGIN-LIM-003 | TC-003, TC-009 |
| FR-LOGIN-LIM-004 | TC-002, TC-004 |
| FR-LOGIN-LIM-005 | TC-005, TC-007 |
| FR-LOGIN-LIM-006 | TC-002, TC-006 |
| FR-LOGIN-LIM-007 | TC-009 |
| FR-LOGIN-LIM-008 | TC-010 |
| FR-LOGIN-LIM-009 | TC-004, TC-011, TC-013 |
| FR-LOGIN-LIM-010 | TC-012 |
| FR-LOGIN-LIM-011 | TC-013, TC-014, TC-015 |

| Use case | Test cases |
| --- | --- |
| UC-LOGIN-LIM-001 | TC-001, TC-002, TC-004, TC-005, TC-006, TC-007, TC-008, TC-013, TC-014 |
| UC-LOGIN-LIM-002 | TC-003, TC-009, TC-010 |
| UC-LOGIN-LIM-003 | TC-004, TC-011, TC-012, TC-013, TC-015 |

## Cases

### TC-001 — Failures are counted per pair only

- Covers: FR-LOGIN-LIM-001; UC-LOGIN-LIM-001
- Level: integration (Go service, real SQLite)
- Given no failures recorded
- When A sends 4 wrong passwords for `alice`, then B sends 1 wrong password for `alice`, then A sends the right password for `alice`
- Then every wrong attempt answers wrong credentials and A's last attempt logs in (token returned).
- Corner cases:
  - letter case: A sends 3 wrong passwords for `Alice` and 2 for `alice`; the next attempt from A for `ALICE` with the right password gets the locked answer (same pair);
  - unknown username: A sends 5 wrong passwords for `nobody`; the 6th attempt from A for `nobody` gets the locked answer, and A then logs in as `alice` with the right password;
  - B's single failure does not combine with A's 4: B then sends 4 more wrong passwords (5 in total for B) and is locked, while A stays unlocked.

### TC-002 — Five failures lock the pair for exactly one minute; refusals do not extend it

- Covers: FR-LOGIN-LIM-002, FR-LOGIN-LIM-004, FR-LOGIN-LIM-006, FR-AUTH-005; UC-LOGIN-LIM-001
- Level: integration (Go service, real SQLite, injected clock)
- Given A has sent 5 wrong passwords for `alice`, the 5th at time T
- When A sends the right password for `alice` at T, at T+30s and at T+59s
- Then each attempt gets the locked answer and creates no session.
- And when A sends the right password at T+1min, then A logs in.
- Corner cases:
  - repeated attempts during the lock (T+30s, T+59s) are not counted and do not extend it: the T+1min attempt succeeds;
  - a wrong password sent during the lock also gets the locked answer, not wrong credentials, and does not extend the lock;
  - after the lock ends: a second scenario where, at T+1min, A sends 4 wrong passwords then the right one logs in (5 new failures are needed); a third where 5 new wrong passwords after T+1min lock the pair again;
  - the threshold is 5: after 4 failures the right password logs in (no lock at 4).

### TC-003 — A locked pair does not affect other clients or other usernames

- Covers: FR-LOGIN-LIM-003, FR-LOGIN-LIM-001, FR-AUTH-005; UC-LOGIN-LIM-002
- Level: integration (Go service, real SQLite)
- Given A/`alice` is locked (5 wrong passwords from A within the minute)
- When B sends the right password for `alice`, and A sends the right password for `bob`
- Then both log in; A's next attempt for `alice` still gets the locked answer.
- Corner cases:
  - B's earlier count is unchanged: B, with 4 prior failures for `alice`, logs in on its 5th attempt with the right password;
  - A/`bob` is not counted by A/`alice` failures: A sends 4 wrong passwords for `bob`, then the right one logs in;
  - the administrator: with A/`admin` locked, `admin` logs in from B.

### TC-004 — A locked attempt is refused without a password check and without waiting

- Covers: FR-LOGIN-LIM-004, FR-LOGIN-LIM-009 (locked before busy); UC-LOGIN-LIM-001, UC-LOGIN-LIM-003
- Level: integration (Go service, real SQLite, held password checks)
- Given A/`alice` is locked, and a password check of another login is held running
- When A sends the right password for `alice`
- Then the call returns the locked answer while the other check is still held; after release, the held login completes normally (its result is that of its own credentials).
- Corner cases:
  - with the running check held and Max checks waiting, the locked attempt still gets the locked answer, not the busy answer (the lock is checked first);
  - the locked attempt does not take a waiting place: with Max − 1 checks waiting, after the locked refusal one more unlocked login is still accepted to wait (not busy);
  - unknown username locked (`nobody` from A): refused the same way, without the dummy password check.

### TC-005 — Success restarts the pair's count only

- Covers: FR-LOGIN-LIM-005; UC-LOGIN-LIM-001
- Level: integration (Go service, real SQLite)
- Given A has 4 failures for `alice` and B has 4 failures for `alice`
- When A logs in with the right password, then A sends 4 more wrong passwords, then A sends the right password
- Then both of A's right-password attempts log in.
- Corner cases:
  - B's count is unchanged by A's success: B's next wrong password locks B/`alice` (5th failure);
  - repeated success: two successes in a row leave the count at zero (4 wrong then right still logs in).

### TC-006 — Old failures and ended locks are forgotten; memory stays bounded

- Covers: FR-LOGIN-LIM-006; UC-LOGIN-LIM-001
- Level: integration (Go service, real SQLite, injected clock); bound check unit (package-internal state)
- Given A sent 4 wrong passwords for `alice` at time T
- When A sends 4 more wrong passwords at T+1h+1s, then the right password
- Then A logs in.
- Corner cases:
  - a failure within the hour keeps the earlier ones: 4 failures at T, a 5th at T+59min locks A/`alice`;
  - bound: 1,000 different addresses each fail once for `alice` at T, and one pair is locked at T; after one more failure of a new pair at T+1h+1s, the failure memory holds only that new pair (expired lock and old failures removed);
  - an active lock is kept by pruning: a failure of another pair at T+30s does not remove A/`alice`'s lock (still the locked answer).

### TC-007 — An attempt already waiting when its pair becomes locked is still checked

- Covers: FR-LOGIN-LIM-005; UC-LOGIN-LIM-001
- Level: unit (Go service, package-internal, held password checks)
- Given an attempt of A/`alice` that passed the lock check and waits for its password check, while A/`alice` reaches its 5th failure and becomes locked until L
- When the held check is released
- Then with a wrong password the attempt answers wrong credentials and the lock still ends at L (not extended); with the right password (second scenario) the attempt logs in and A/`alice`'s failures are cleared, so A's next right-password attempt logs in at once.
- Corner cases: both outcomes, wrong and right password, are covered.

### TC-008 — A cancelled waiting login is not counted

- Covers: FR-LOGIN-LIM-001 (only failed checks are counted; specification "States and corner cases"); UC-LOGIN-LIM-001
- Level: integration (Go service, real SQLite, held password checks)
- Given A has 4 failures for `alice` and a password check is held running
- When A sends a wrong password for `alice` with a context that is cancelled while it waits, then the held check is released, then A sends the right password
- Then the cancelled call returns the cancellation error (not wrong credentials) and A's last attempt logs in.

### TC-009 — `CF-Connecting-IP` is the client address

- Covers: FR-LOGIN-LIM-007, FR-LOGIN-LIM-001, FR-LOGIN-LIM-003; UC-LOGIN-LIM-002
- Level: integration (Go HTTP handler, real service and SQLite)
- Given every request has `RemoteAddr` `192.0.2.10:5000` (one proxy)
- When 5 requests with `CF-Connecting-IP: 203.0.113.1` send wrong passwords for `alice`, then a request with `203.0.113.1` sends the right password, then a request with `203.0.113.2` sends the right password
- Then the 5 answers are 401, the 6th is the locked answer without a session cookie, and the last is 200 with a session cookie.
- Corner cases:
  - the header wins over the connection: a request with `CF-Connecting-IP: 203.0.113.1` from another `RemoteAddr` (`192.0.2.11:6000`) is also locked;
  - a request from `192.0.2.10:5000` without the header (connection address) logs in as `alice`;
  - IPv6: 5 failures with `CF-Connecting-IP: 2001:db8::1` lock that address only; `2001:db8::2` logs in.

### TC-010 — Without a valid `CF-Connecting-IP`, the connection address is the client address

- Covers: FR-LOGIN-LIM-008; UC-LOGIN-LIM-002
- Level: integration (Go HTTP handler, real service and SQLite)
- Given 5 requests without the header from `RemoteAddr` `198.51.100.7:40000` sent wrong passwords for `alice`
- When requests from `198.51.100.7` send the right password for `alice` with: no header; `CF-Connecting-IP: not-an-ip`; an empty `CF-Connecting-IP`; `CF-Connecting-IP: 203.0.113.1:443` (not a bare IP)
- Then each gets the locked answer.
- Corner cases:
  - the port is not part of the address: a request from `198.51.100.7:40001` without the header is locked too;
  - another connection address, `198.51.100.8:40000`, without the header, logs in as `alice`;
  - invalid headers from different connections are different clients: `not-an-ip` from `198.51.100.8` logs in.

### TC-011 — Busy answer when the maximum number of checks are waiting

- Covers: FR-LOGIN-LIM-009; UC-LOGIN-LIM-003
- Level: integration (Go service, real SQLite, held password checks)
- Given A has 4 failures for `alice`, one password check is held running and Max logins are waiting
- When A sends the right password for `alice` twice
- Then both calls return the busy answer while the checks are still held, and no session is created.
- And when the held and waiting checks are released and finished, A sends the right password for `alice`
- Then A logs in (the busy refusals were not counted; 4 + 2 would have locked the pair).
- Corner cases:
  - boundary: with Max − 1 waiting, one more login is accepted to wait (not busy); the next is busy;
  - an unknown username is refused busy the same way;
  - excluded flows keep waiting: with the queue full, creating a user (`CreateUser`) is not refused busy and completes after release.

### TC-012 — A login waits its turn while a waiting place is free

- Covers: FR-LOGIN-LIM-010; UC-LOGIN-LIM-003
- Level: integration (Go service, real SQLite, held password checks)
- Given one password check is held running and none waiting
- When B sends the right password for `alice`
- Then the call has not returned while the check is held; after release it logs in.
- Corner cases: with a wrong password instead, the waiting login answers wrong credentials after release and the failure counts for its pair.

### TC-013 — HTTP answers of the login flow: locked and busy

- Covers: FR-LOGIN-LIM-002, FR-LOGIN-LIM-009, FR-LOGIN-LIM-011 (server side); UC-LOGIN-LIM-001, UC-LOGIN-LIM-003
- Level: integration (Go HTTP handler, real service and SQLite)
- Given protected mode (user `alice` exists)
- When from one `RemoteAddr`, 5 wrong passwords for `alice` are posted, then the right one; and, in a second scenario, a login is posted while a check is held running and Max are waiting
- Then the locked attempt is HTTP 429 with code `LOGIN_LOCKED` and the locked message; the busy attempt is HTTP 429 with the busy code and message; neither sets a session cookie, and `GET /api/v1/auth/session` without a cookie still reports `user: null`.
- Corner cases: the two 429 answers have different codes and messages; invalid JSON is still 400 `INVALID_JSON` and not counted as a failure (the right password then logs in after 4 failures and an invalid body).

### TC-014 — The login screen shows the locked message

- Covers: FR-LOGIN-LIM-011; UC-LOGIN-LIM-001
- Level: e2e (Playwright, mocked API)
- Given protected mode with no user, and `POST /api/v1/auth/login` mocked to answer 429 `LOGIN_LOCKED` with the locked message
- When the user types a username and password and submits with the Enter key
- Then the "Could not log in" error shows "Too many failed attempts. Wait one minute and try again.", focus moves to the error, the login page stays (URL unchanged), the fields and the **Log in** button are enabled again.
- Corner cases:
  - repeated submit: submitting again shows the same single error (no duplicate notification);
  - narrow screen: at 320 px the message is fully visible inside the page, without horizontal scroll.

### TC-015 — The login screen shows the busy message, then a retry logs in

- Covers: FR-LOGIN-LIM-011; UC-LOGIN-LIM-003
- Level: e2e (Playwright, mocked API)
- Given protected mode with no user, and the login route mocked to answer the busy 429 once, then 200 with user `alice`
- When the user submits the form, then clicks **Log in** again
- Then after the first submit the "Could not log in" error shows "Too many login attempts are being checked. Try again in a moment." and the page stays; after the second, the app navigates to `/`.
- Corner cases: the busy message is distinct from the locked message (TC-014); while the request is pending the button reads "Logging in…" and is disabled, as today.
