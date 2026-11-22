# Test cases: Amazon browser profile recovery (issue #180)

Sources: [functional specification](../../specifications/amazon-browser-profile/functional.md), use cases [UC-AMZ-BPROF-001](../../use-cases/amazon-browser-profile/UC-AMZ-BPROF-001-amazon-check-after-browser-ended-abruptly.md), [UC-AMZ-BPROF-002](../../use-cases/amazon-browser-profile/UC-AMZ-BPROF-002-amazon-checks-after-data-directory-moved.md).

Conventions:

- **Stale lock** in tests means the browser's lock marks in the profile directory: `SingletonLock` (symlink to `<host>-<pid>`), `SingletonSocket` (symlink to a socket path) and `SingletonCookie` (symlink to a number).
- **Killed browser** means: the test starts the real browser executable (`amazontest.Browser(t)`) itself on the profile (`--headless=new --user-data-dir=<profile> --no-sandbox about:blank`) in its own process group, waits until `SingletonLock` exists, sends SIGKILL to the group and waits for the exit. The test asserts that `SingletonLock` is still present before the next check, so the precondition is real.
- **Browser tests** run the real browser against the local fixture server of `internal/amazon` (`amazontest.NewServer`, Amazon host sent to it through `Origin`), with a profile in `t.TempDir()`. No test contacts amazon.*.
- **Stub executable** means a shell script used as `ExecPath` that writes the listing of its `--user-data-dir` (names, kinds and symlink targets) to a file and exits 3, so a unit test sees the profile exactly as the browser would find it at launch, without a real browser.
- **Cookie fixture**: the product page sets `session-id=fixture-session-180`; a cookie "still sent" means the page request (or a resource request) of a later check carries it.
- Integration tests use a real SQLite database in `t.TempDir()` and the service harness of `internal/service/amazon_human_checks_test.go`; TC-010 also uses the HTTP handlers.
- New Go test names use the prefix `TestBrowserProfile_TCnnn_` (browser package) or `TestAmazonBrowserProfile_TCnnn_` (service) so they do not clash with the `TCnnn` names of issue #168.
- No UI change: there is no e2e case. The existing browser tests of issue #168 (`internal/amazon/browser_test.go`, `internal/service/amazon_human_checks_test.go`) and the Playwright tests in `tests/` must keep passing.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-AMZ-BPROF-001 | TC-001, TC-003, TC-004, TC-009 |
| FR-AMZ-BPROF-002 | TC-001, TC-002, TC-010 |
| FR-AMZ-BPROF-003 | TC-001, TC-005 |
| FR-AMZ-BPROF-004 | TC-001, TC-002, TC-003, TC-010 |
| FR-AMZ-BPROF-005 | TC-004, TC-009 |
| FR-AMZ-BPROF-006 | TC-006, TC-007 |
| FR-AMZ-BPROF-007 | TC-008, TC-009 |

| Use case | Test cases |
| --- | --- |
| UC-AMZ-BPROF-001 | TC-001, TC-003, TC-004, TC-005, TC-006, TC-007, TC-008, TC-009 |
| UC-AMZ-BPROF-002 | TC-001, TC-002, TC-007, TC-010 |

## Profile preparation

### TC-001 — Only the lock marks are discarded before the browser starts

- Covers: FR-AMZ-BPROF-001, FR-AMZ-BPROF-002, FR-AMZ-BPROF-003, FR-AMZ-BPROF-004; UC-AMZ-BPROF-001, UC-AMZ-BPROF-002
- Level: unit
- Given a persistent profile directory containing `SingletonLock` → `otherhost-12345`, `SingletonSocket` → `<outside dir>/socket-target` (an existing file outside the profile), `SingletonCookie` → `4242`, plus `Default/Cookies`, `Local State` and a file named `SingletonLock.bak`, and a collector whose `ExecPath` is the stub executable
- When an Amazon item is checked
- Then the stub's listing shows none of `SingletonLock`, `SingletonSocket`, `SingletonCookie`, while `Default/Cookies`, `Local State` and `SingletonLock.bak` are listed; after the check these files are byte-identical to before and `<outside dir>/socket-target` still exists (only the links are removed, never their targets); the check result is the usual browser-start failure of the stub
- Corner cases:
  - dangling symlinks (targets that do not exist, like a real stale lock) are removed too;
  - only `SingletonLock` present (no socket, no cookie mark): removed, no error;
  - a mark that is a regular file instead of a symlink: removed;
  - no mark at all and the profile directory not yet created: the directory is created, the stub is launched, no error;
  - lock naming this host (`<os.Hostname()>-<pid of the test process>`, a live process): removed as well;
  - temporary profile (`ProfileDir` empty): the stub receives a new empty temporary directory, removed after the check; nothing else on disk is touched.

## Browser start with a stale lock

### TC-002 — Lock naming another host does not prevent the check

- Covers: FR-AMZ-BPROF-002, FR-AMZ-BPROF-004; UC-AMZ-BPROF-002
- Level: integration (browser test)
- Given a profile used by a first successful check of the price-found fixture (cookie fixture set), then given the stale lock `SingletonLock` → `otherhost-12345`, `SingletonSocket` → `/tmp/org.chromium.Chromium.gone/SingletonSocket` (missing) and `SingletonCookie` → `12345`
- When a new collector on the same profile checks the price-found fixture
- Then the result is `success` with the fixture price, the page request of that check carries `session-id=fixture-session-180`, no `browser could not start` line is logged, and no browser process of the profile is left
- Corner cases: fixture sanity check: starting the browser executable directly on the prepared profile (without the collector) does not open the page because the profile is in use on another computer, proving the lock is the cause the change removes; a second check right after also succeeds.

### TC-003 — Browser killed on this machine does not prevent the next check

- Covers: FR-AMZ-BPROF-001, FR-AMZ-BPROF-004; UC-AMZ-BPROF-001
- Level: integration (browser test)
- Given a profile used by a first successful check (cookie fixture set), then a killed browser on that profile, leaving `SingletonLock` naming this host
- When the collector checks the price-found fixture
- Then the result is `success` with the fixture price, the cookie fixture is still sent, no `browser could not start` line is logged, and no browser process of the profile is left
- Corner cases: a lock naming this host and the PID of a live unrelated process (the test process, as after a PID reuse) gives the same success.

### TC-004 — Successive abrupt ends recover every time

- Covers: FR-AMZ-BPROF-005, FR-AMZ-BPROF-001; UC-AMZ-BPROF-001
- Level: integration (browser test)
- Given one collector and one profile
- When two cycles run, each made of a killed browser on the profile followed by a check of the price-found fixture
- Then both checks return `success` with the fixture price, with no file deleted or process restarted by the test between the kill and the check
- Corner cases: a third cycle alternating with a lock naming another host (TC-002 setup) also succeeds.

### TC-005 — Profile without stale lock: unchanged behaviour

- Covers: FR-AMZ-BPROF-003; UC-AMZ-BPROF-001
- Level: integration (browser test)
- Given a profile directory that does not exist yet
- When the collector checks the price-found fixture twice in a row (the first browser ends normally)
- Then both checks return `success` with the fixture price and the second carries the cookie set during the first
- Corner cases: a check of an unavailable listing and of a price-not-found page on the cleanly closed profile give `unavailable` and `price_not_found` as before.

## Start still impossible

### TC-006 — A lock that cannot be discarded is a failure

- Covers: FR-AMZ-BPROF-006; UC-AMZ-BPROF-001
- Level: integration (browser test)
- Given a profile whose `SingletonLock` is a non-empty directory (so it cannot be removed, whatever the user running the test)
- When the collector checks the price-found fixture
- Then the result is a failure (`request_error`, message `Amazon could not be reached for a price check.`, second-hand status `check_error`), the check ends within 20 s without panicking, no browser process of the profile or extra goroutine is left, and the `SingletonLock` directory content is unchanged
- Corner cases: a missing browser executable with a stale lock in the profile is still a failure (existing TC-007 of issue #168 keeps passing); a later check after the test replaces the directory by a normal stale lock succeeds.

### TC-007 — The failure is logged, counted and stops items as before

- Covers: FR-AMZ-BPROF-006; UC-AMZ-BPROF-001, UC-AMZ-BPROF-002
- Level: integration (SQLite + browser test)
- Given two Amazon items and the service started with a browser collector whose profile has the non-removable `SingletonLock` of TC-006
- When five Amazon checks fail in a row (first check after add, then Refresh price and scheduled checks)
- Then each failure is logged with `browser could not start`, the item shows a retrieval issue, the consecutive failure count goes 1 to 5, the items are not stopped after failures 1–4, and after the 5th failure both items are stopped
- Corner cases: after 4 failures, replacing the directory by a normal stale lock (other host) makes the next check succeed and reset the count to 0.

## Overlapping checks

### TC-008 — A check in progress keeps its profile

- Covers: FR-AMZ-BPROF-007; UC-AMZ-BPROF-001
- Level: integration (browser test)
- Given one collector, one profile, and a fixture server whose page for item A blocks until the test releases it, while item B's page answers at once
- When the check of A has reached the server (its browser is running and holds `SingletonLock`) and the check of B is started concurrently
- Then while A is blocked, `SingletonLock` still exists with the same target and the server receives no request for B; after the release, A and B both return `success` with their prices
- Corner cases: three concurrent checks on the same collector all succeed; a check cancelled while waiting for the profile ends without starting a browser.

## User flows

### TC-009 — Scheduled check and Refresh price after killed browsers

- Covers: FR-AMZ-BPROF-001, FR-AMZ-BPROF-005, FR-AMZ-BPROF-007; UC-AMZ-BPROF-001
- Level: integration (SQLite + browser test)
- Given an Amazon item with a stored price and 2 consecutive failures, a fake clock, the service started with a browser collector on a profile left by a killed browser
- When the scheduler runs after the item's due time, then the browser is killed again on the profile and Refresh price is requested for the item
- Then both checks record the fixture price and check time, the consecutive failure count is 0 after each, the item is active and not stopped, the next scheduled time is in the next window, and no `browser could not start` line is logged
- Corner cases: Refresh price of a second Amazon item requested while the scheduled check of the first is blocked on its page: both checks complete with their prices; protected mode with items of two owners: both owners' items are checked successfully after the recovery.

### TC-010 — Data directory moved to another machine

- Covers: FR-AMZ-BPROF-002, FR-AMZ-BPROF-004; UC-AMZ-BPROF-002
- Level: integration (SQLite + HTTP + browser test)
- Given a data directory whose `amazon-browser` profile holds a stale lock naming `otherhost-12345` and the cookie fixture from an earlier check, an active Amazon item with 4 consecutive failures, and the service and HTTP handlers started on that data directory with the browser collector using `<data dir>/amazon-browser`
- When `POST /api/v1/items/{id}/refresh` is called and the check finishes
- Then the response is `202`, `GET /api/v1/items` shows the fixture price, a new check time and `amazonStoppedAt: null`, the consecutive failure count is 0, and the page request carried the cookie fixture
- Corner cases: all Amazon items stopped before the start (failure count 5): after start they stay stopped and the scheduler sends them no request; `POST /api/v1/items/{id}/refresh` on one of them checks it through the browser, records the price and restarts only that item as in FR-AMZ-STOP-006; the other stopped items stay stopped.
