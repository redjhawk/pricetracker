# UC-AMZ-BPROF-001 — Amazon check after the browser ended abruptly

**Primary actor:** PriceFollower application, on behalf of the user who tracks Amazon items
**Supporting systems:** Real web browser with its persistent profile, Amazon, SQLite store
**Trigger:** An Amazon check starts (first check after add, scheduled check, **Refresh all prices** or **Refresh price**).
**Origin:** Issue #180 (review comment R1-02 on #179): a browser killed or crashed during a check leaves lock marks in the persistent profile.
**Covers:** FR-AMZ-BPROF-001, FR-AMZ-BPROF-003, FR-AMZ-BPROF-004, FR-AMZ-BPROF-005, FR-AMZ-BPROF-006, FR-AMZ-BPROF-007

## Preconditions

- One or more Amazon items are tracked and not stopped.
- The Amazon browser uses the persistent profile in the data directory, which holds cookies stored by Amazon in earlier checks.
- An earlier browser of this application on the same machine did not end normally (killed when a check was closed or stopped, crash, server stop during a check) and left a stale lock in the profile.

## Main flow

1. The application starts the browser for the Amazon check with the persistent profile.
2. The stale lock is discarded; the rest of the profile (cookies and other browsing data) is kept.
3. The browser starts and opens the item's Amazon page, sending the cookies stored in earlier checks.
4. The check proceeds as in UC-AMZ-HCHK-001: the price (or price not found, listing unavailable) is recorded and the consecutive failure count is reset.
5. The user sees the updated price and check time in the tracked-items list and on the item details page; nothing about the recovery is shown.

## Alternatives and errors

- **No stale lock** (first use or after a normal end): the check proceeds as before (FR-AMZ-BPROF-003).
- **Repeated abrupt ends:** each later check whose browser start finds a new stale lock recovers the same way; no manual action is ever needed (FR-AMZ-BPROF-005).
- **Overlapping requests** (for example **Refresh price** during a scheduled check): the browser of the check in progress keeps its profile; both checks complete normally, as today (FR-AMZ-BPROF-007).
- **Browser still cannot start** (stale lock cannot be discarded, browser missing, other start error): the check fails with "browser could not start"; the failure is logged, counted and, at the fifth consecutive failure, every Amazon item is stopped (FR-AMZ-BPROF-006, FR-AMZ-HCHK-003, FR-AMZ-STOP-001–010).
- **No persistent profile** (temporary profile): nothing changes.
- **Protected mode:** the same applies to the Amazon items of all users.

## Postconditions

- The check result is recorded and no failure is counted because of the stale lock.
- The persistent profile keeps its cookies and other browsing data.
- Item states, messages, schedule and stop rules are unchanged.
