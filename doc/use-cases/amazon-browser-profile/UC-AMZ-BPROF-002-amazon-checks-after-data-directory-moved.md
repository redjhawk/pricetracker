# UC-AMZ-BPROF-002 — Amazon checks after the data directory is moved to another machine

**Primary actor:** PriceFollower application, on behalf of the user who tracks Amazon items
**Supporting systems:** Real web browser with its persistent profile, Amazon, SQLite store
**Trigger:** An Amazon check starts after the application was started with a data directory used before on another server or container.
**Origin:** Issue #180 (review comment R1-02 on #179): a profile whose lock names another host made every Amazon check fail with "browser could not start" and, after 5 failures, stopped all Amazon items.
**Covers:** FR-AMZ-BPROF-002, FR-AMZ-BPROF-004, FR-AMZ-BPROF-006

## Preconditions

- One or more Amazon items are tracked.
- The data directory was moved to another server or container (or a new container uses it); its persistent profile holds a stale lock naming the previous machine (for example `otherhost-12345`) and cookies stored by Amazon in earlier checks.
- No other application instance uses the same data directory at the same time.

## Main flow

1. An Amazon check of a non-stopped item starts (scheduled check, first check after add, **Refresh all prices** or **Refresh price**).
2. The application starts the browser with the persistent profile; the stale lock naming the other machine is discarded and the rest of the profile is kept.
3. The browser starts and opens the item's Amazon page, sending the cookies stored in earlier checks.
4. The check proceeds as in UC-AMZ-HCHK-001: the result is recorded, the consecutive failure count is reset and no failure is counted.
5. The user sees the updated price and check time; nothing about the recovery is shown.

## Alternatives and errors

- **Items stopped before the installation** (for example stopped by stale-lock failures): they stay stopped and are not restarted automatically. When the user chooses **Refresh price** on such an item, it is restarted (FR-AMZ-STOP-006) and its check now returns the price through steps 2–5.
- **Browser still cannot start for another reason:** the failure is logged, counted and stops items as in FR-AMZ-HCHK-003 and FR-AMZ-STOP-001–010 (FR-AMZ-BPROF-006).
- **Protected mode:** the same applies to the Amazon items of all users.

## Postconditions

- Amazon checks work on the new machine without any manual action on the data directory.
- The persistent profile keeps its cookies and other browsing data.
