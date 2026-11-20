# Test cases: Amazon human-like checks (issue #168)

Sources: [functional specification](../../specifications/amazon-human-checks/functional.md), use cases [UC-AMZ-HCHK-001](../../use-cases/amazon-human-checks/UC-AMZ-HCHK-001-scheduled-amazon-checks-at-random-times.md), [UC-AMZ-HCHK-002](../../use-cases/amazon-human-checks/UC-AMZ-HCHK-002-add-amazon-item-and-start-random-schedule.md), [UC-AMZ-HCHK-003](../../use-cases/amazon-human-checks/UC-AMZ-HCHK-003-refresh-amazon-prices-immediately.md), [UC-AMZ-HCHK-004](../../use-cases/amazon-human-checks/UC-AMZ-HCHK-004-restart-stopped-amazon-items-once-after-upgrade.md).

Conventions:

- Times are local to the schedule timezone `Europe/Paris` unless stated. Morning window 06:00–09:00, evening window 19:00–23:00.
- Schedule tests use an injected clock and an injected (seeded) random source, so they are deterministic and never sleep in real time.
- **Browser tests** run the real browser engine against a local HTTP test server that serves Amazon-like fixture pages (the Amazon host is redirected to it in tests). No test contacts amazon.*.
- **Fake collector** means the existing `fakeAmazon` style stub in `internal/service`; it records every call.
- Integration tests use a real SQLite database in `t.TempDir()` and, where stated, the HTTP handlers (`GET /api/v1/items`, `POST /api/v1/items`, `POST /api/v1/items/{id}/refresh`, `POST /api/v1/items/refresh`).
- The UI does not change in this change; existing Playwright tests for the Amazon tab warning and refresh (`tests/platform-tabs.spec.ts`, `tests/item-refresh.spec.ts`, `tests/responsive-layout.spec.ts`) must keep passing. One e2e case below covers the user-visible result of the one-time restart.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-AMZ-HCHK-001 | TC-001, TC-002, TC-003, TC-020 |
| FR-AMZ-HCHK-002 | TC-004, TC-005, TC-020 |
| FR-AMZ-HCHK-003 | TC-006, TC-007, TC-008, TC-017, TC-021 |
| FR-AMZ-HCHK-004 | TC-009, TC-010, TC-011, TC-012, TC-013, TC-020 |
| FR-AMZ-HCHK-005 | TC-014, TC-015 |
| FR-AMZ-HCHK-006 | TC-016, TC-017 |
| FR-AMZ-HCHK-007 | TC-018, TC-019 |
| FR-AMZ-HCHK-008 | TC-021, TC-022 |
| FR-AMZ-HCHK-009 | TC-023, TC-024 |
| FR-AMZ-HCHK-010 | TC-025, TC-026, TC-027, TC-028, TC-029, TC-030 |
| FR-AMZ-STOP-001 | TC-006, TC-008 |
| FR-AMZ-STOP-004 | TC-019, TC-021, TC-022 |
| FR-AMZ-STOP-008 | TC-018, TC-025 |
| FR-AMZ-SRCH-DEL-006 | TC-009, TC-011 |

| Use case | Test cases |
| --- | --- |
| UC-AMZ-HCHK-001 | TC-001–TC-015, TC-020–TC-024, TC-031, TC-032 |
| UC-AMZ-HCHK-002 | TC-001, TC-004, TC-016, TC-017 |
| UC-AMZ-HCHK-003 | TC-001, TC-004, TC-008, TC-018, TC-019 |
| UC-AMZ-HCHK-004 | TC-025–TC-030 |

## Real browser requests

### TC-001 — Every Amazon check goes through the browser collector

- Covers: FR-AMZ-HCHK-001; UC-AMZ-HCHK-001, UC-AMZ-HCHK-002, UC-AMZ-HCHK-003
- Level: unit
- Given a service built with the default configuration
- When the collector used for the `amazon` platform is inspected, and an Amazon item is checked through each entry point (first check after add, scheduled check, Refresh price, Refresh all prices) with a recording browser collector stub
- Then the `amazon` collector is the browser-based collector, every entry point calls it exactly once per item, and no plain HTTP Amazon collector is reachable from the service
- Corner cases: Refresh all prices with Amazon and LeBoncoin items (LeBoncoin still uses its own collector, not the browser).

### TC-002 — The browser runs page scripts and loads page resources

- Covers: FR-AMZ-HCHK-001
- Level: integration (browser test)
- Given a local fixture server whose product page has no price in its HTML; an inline script inserts the price element, and the page references a stylesheet and an image on the same server
- When the Amazon browser collector checks the page
- Then the price inserted by the script is detected, and the server log shows requests for the page, the stylesheet and the image
- Corner cases: a plain HTTP download of the same page would find no price (asserted by fetching the page with Go's `http.Get` in the test and parsing it with the existing price parser: no price).

### TC-003 — The browser keeps cookies and is never shown

- Covers: FR-AMZ-HCHK-001
- Level: integration (browser test)
- Given a fixture server that sets a cookie on the product page and records the request headers of the page's resource requests
- When the collector checks the page twice in a row
- Then the resource request of the first check carries the cookie set by the page, and the browser runs headless (no display is required: the test runs with `DISPLAY` unset)
- Corner cases: no Amazon login or credential is sent (no `Authorization` header, no cookie other than those set by the fixture).

### TC-004 — Browser results match the current parsing results

- Covers: FR-AMZ-HCHK-002; UC-AMZ-HCHK-001, UC-AMZ-HCHK-002, UC-AMZ-HCHK-003
- Level: integration (browser test)
- Given the existing Amazon fixture pages used by the current parser tests (price found, second-hand offer sold by Amazon, second-hand offer by another seller, price not found, listing unavailable), served by the fixture server
- When the browser collector checks each page
- Then each `CollectionResult` (result, amount, currency, second-hand status, amount and condition) equals the result the current parser gives for the same HTML
- Corner cases: page with a price but no second-hand offer; page with a second-hand offer and no new price; title and thumbnail kept as today.

### TC-005 — Browser success is stored like today

- Covers: FR-AMZ-HCHK-002
- Level: integration (SQLite + HTTP)
- Given an Amazon item with 2 consecutive failures and a browser collector returning a success for the price-found fixture
- When Refresh price is requested through `POST /api/v1/items/{id}/refresh` and the check finishes
- Then `GET /api/v1/items` shows the new price, the check time and the second-hand offer, the price history gains one entry, and the consecutive failure count is 0
- Corner cases: unavailable listing result stores status `unavailable` and resets the failure count (an Amazon answer, not a failure).

### TC-006 — Non-success response and challenge page are failures

- Covers: FR-AMZ-HCHK-003, FR-AMZ-STOP-001
- Level: integration (browser test)
- Given a fixture server answering, in separate sub-tests, HTTP 503, HTTP 404 for the product page, and HTTP 200 with an Amazon robot-check (captcha) page
- When the browser collector checks the page
- Then the result is a failure (not success, not price not found, not unavailable), with the request URL, HTTP status and a response excerpt for the log
- Corner cases: the log line contains no cookie values or session secrets (FR-AMZ-HUMAN-012).

### TC-007 — Network error, timeout and browser start failure are failures

- Covers: FR-AMZ-HCHK-003
- Level: integration (browser test)
- Given a browser collector whose page load fails with: connection refused (closed port), a fixture page that never finishes loading (short test timeout), and a browser engine that cannot start (stub launcher returns an error)
- When an Amazon item is checked
- Then each case returns a failure result without panicking, the check ends (no goroutine or browser process left running after the test), and the in-flight marker of the item is released
- Corner cases: a context cancelled during the page load ends the check promptly.

### TC-008 — Browser failures keep the last price and stop after 5

- Covers: FR-AMZ-HCHK-003, FR-AMZ-STOP-001; UC-AMZ-HCHK-001, UC-AMZ-HCHK-003
- Level: integration (SQLite + HTTP)
- Given two Amazon items with a stored price and a browser collector that always fails
- When 5 checks fail in a row (mix of scheduled checks and Refresh price)
- Then after failures 1–4 the items are not stopped and keep their last successful price, and a failed Refresh price is reported as failed by the item's refresh status; after the 5th failure both items have a non-null `amazonStoppedAt` in `GET /api/v1/items` and each failure was logged
- Corner cases: a success after 4 failures resets the count, so 4 more failures do not stop the items.

## Random schedule

### TC-009 — Two checks per day, one in each window

- Covers: FR-AMZ-HCHK-004, FR-AMZ-SRCH-DEL-006; UC-AMZ-HCHK-001
- Level: unit
- Given the schedule function for Amazon items with a seeded random source, starting at 00:00 on a given day
- When the next check time is computed repeatedly from each previous check time over 7 days
- Then there are exactly 14 checks: on each day one in [06:00, 09:00] and one in [19:00, 23:00], none outside the windows
- Corner cases: random values at the extremes of the source (minimum and maximum) still give times inside the windows; times are stored in UTC and converted from `Europe/Paris`.

### TC-010 — Times change from day to day

- Covers: FR-AMZ-HCHK-004
- Level: unit
- Given one Amazon item and a seeded random source
- When the morning and evening times are drawn for 7 consecutive days
- Then the 7 morning times are not all equal and the 7 evening times are not all equal; with a stub source returning a different value each day, the drawn time moves with the returned value (the time comes from the random source, not from a fixed slot)
- Corner cases: times have minute (or finer) resolution, not only whole hours.

### TC-011 — Configured check times no longer apply to Amazon

- Covers: FR-AMZ-HCHK-004, FR-AMZ-SRCH-DEL-006
- Level: unit
- Given `PRICEFOLLOWER_CHECK_TIMES` set to `12:00,15:00` and a random source that would otherwise choose 12:00 if allowed
- When the next check of an Amazon item and of a LeBoncoin item is computed at 10:00
- Then the Amazon item's next check is in 19:00–23:00, and the LeBoncoin item's next check is 12:00
- Corner cases: default configuration (08:00, 20:00): LeBoncoin items still get exactly 08:00 and 20:00.

### TC-012 — Next window from any time of day

- Covers: FR-AMZ-HCHK-004
- Level: unit
- Given a seeded random source
- When the next Amazon check is computed after a check made at 05:00, 07:30 (inside the morning window, after the morning check), 14:00, 20:30 (after the evening check) and 23:30
- Then the result is respectively in today's morning window, today's evening window, today's evening window, tomorrow's morning window, tomorrow's morning window
- Corner cases: computing across a month end and a year end (31 Dec 23:30 → 1 Jan morning).

### TC-013 — Daylight saving days

- Covers: FR-AMZ-HCHK-004
- Level: unit
- Given the timezone `Europe/Paris` and the days of the spring (28 Mar 2027) and autumn (25 Oct 2026) clock changes
- When the checks of those days are computed
- Then each day still has one check in each local window, expressed in local time, and no check falls outside the windows or is skipped
- Corner cases: another configured timezone (`UTC`) uses that timezone's windows.

### TC-014 — Each item draws its own time

- Covers: FR-AMZ-HCHK-005; UC-AMZ-HCHK-001
- Level: unit
- Given 10 Amazon items and a seeded random source
- When their next morning check times are computed from the same instant
- Then the times are drawn independently: at least 2 distinct times exist and the times span more than 30 minutes of the window (not one shared time with small offsets)
- Corner cases: with a single Amazon item the schedule works the same.

### TC-015 — Only due items are checked at a scheduled run

- Covers: FR-AMZ-HCHK-005; UC-AMZ-HCHK-001
- Level: integration (SQLite)
- Given 3 Amazon items with stored next checks at 06:20, 07:10 and 08:40, a fake clock and a fake collector
- When the scheduler runs at 06:21, then at 07:11, then at 08:41
- Then each run checks only the one item whose time has passed, the other items are not checked, and each checked item gets a new next check in the evening window
- Corner cases: no Amazon item due → no collector call; an item whose check is already in progress is not checked again by the scheduler.

## Adding and refreshing

### TC-016 — Add: immediate check, then the next window

- Covers: FR-AMZ-HCHK-006; UC-AMZ-HCHK-002
- Level: integration (SQLite + HTTP)
- Given a fake clock at 07:00 and a fake collector
- When a supported Amazon URL is added through `POST /api/v1/items`
- Then the first check starts immediately (collector called without waiting for a window), and once it finishes `GET /api/v1/items` shows a `nextCheckAt` in the evening window of the same day, not in the current morning window
- Corner cases: added at 05:59 → next check in today's morning window; added at 23:30 → next check in tomorrow's morning window; adding the same listing twice still returns the existing error and makes no extra check.

### TC-017 — Failed first check keeps the schedule

- Covers: FR-AMZ-HCHK-003, FR-AMZ-HCHK-006; UC-AMZ-HCHK-002
- Level: integration (SQLite + HTTP)
- Given a fake clock at 14:00 and a collector that fails
- When an Amazon item is added
- Then the item shows a retrieval issue, the failure is counted, and its `nextCheckAt` is in the evening window of the same day
- Corner cases: added while all Amazon items are stopped: the add and first check follow the existing Amazon request stop rules (`amazonStoppedAt: null` for the new item).

### TC-018 — Refresh price at any time does not move the schedule

- Covers: FR-AMZ-HCHK-007, FR-AMZ-STOP-008; UC-AMZ-HCHK-003
- Level: integration (SQLite + HTTP)
- Given an Amazon item with `nextCheckAt` at 20:47, a fake clock at 14:00 and a fake collector
- When `POST /api/v1/items/{id}/refresh` is called
- Then the response is `202`, the collector is called at once, the item shows the refresh in progress while it runs, and after it finishes `nextCheckAt` is still 20:47
- Corner cases: a second refresh while the first is running starts nothing and makes no second collector call; Refresh price on a stopped Amazon item still checks it, restarts it on an Amazon answer (existing FR-AMZ-STOP-006 behaviour) and keeps its `nextCheckAt`.

### TC-019 — Refresh all prices: immediate, skips stopped items, schedule unchanged

- Covers: FR-AMZ-HCHK-007, FR-AMZ-STOP-004; UC-AMZ-HCHK-003
- Level: integration (SQLite + HTTP)
- Given at 14:00 two Amazon items (one stopped) and one LeBoncoin item, each with a stored `nextCheckAt`
- When Refresh all prices is requested
- Then the non-stopped Amazon item and the LeBoncoin item are checked at once, the stopped Amazon item is not checked, and every item keeps its `nextCheckAt`
- Corner cases: no tracked item → nothing is checked and no error.

## Scheduled flow end to end

### TC-020 — Scheduled check through the browser over several days

- Covers: FR-AMZ-HCHK-001, FR-AMZ-HCHK-002, FR-AMZ-HCHK-004; UC-AMZ-HCHK-001
- Level: integration (SQLite + browser test)
- Given one Amazon item, a fake clock, a seeded random source and the browser collector pointed at the price-found fixture page
- When the clock is advanced over 3 days and the scheduler runs after each due time
- Then the fixture server receives exactly 6 browser page loads (one per window per day), each at the item's drawn time, and the stored price history has one entry per check
- Corner cases: the scheduled times on day 1, 2 and 3 are not identical.

### TC-021 — A stop during a window prevents later scheduled checks

- Covers: FR-AMZ-HCHK-003, FR-AMZ-HCHK-008, FR-AMZ-STOP-004; UC-AMZ-HCHK-001
- Level: integration (SQLite)
- Given 7 Amazon items with next checks spread through the morning window, a collector that always fails and a fake clock
- When the scheduler runs as each time passes
- Then the first 5 items are checked and fail, the 5th failure stops all 7 items, and items 6 and 7 are not sent to the collector at their scheduled times
- Corner cases: the following evening window also sends no request for the stopped items.

### TC-022 — Stopped item skipped at its scheduled time

- Covers: FR-AMZ-HCHK-008, FR-AMZ-STOP-004; UC-AMZ-HCHK-001
- Level: integration (SQLite)
- Given one stopped and one active Amazon item, both due at 07:00
- When the scheduler runs at 07:01
- Then only the active item is checked; the stopped item gets no collector call and stays stopped
- Corner cases: a deleted item that was due gets no collector call and leaves no scheduled check (it no longer appears in due items).

### TC-023 — Restart before the drawn time keeps the check

- Covers: FR-AMZ-HCHK-009; UC-AMZ-HCHK-001
- Level: integration (SQLite)
- Given an Amazon item whose stored next check is 07:40 and a service started at 06:30 on the same database file
- When the service is closed and a new service is opened on that file at 06:35, and the scheduler runs at 07:41
- Then the item is checked exactly once in the morning window, at or after 07:40, and its stored time was not redrawn by the restart
- Corner cases: two restarts within the window still give a single morning check.

### TC-024 — Restart after the check does not repeat it

- Covers: FR-AMZ-HCHK-009; UC-AMZ-HCHK-001
- Level: integration (SQLite)
- Given an Amazon item checked by the scheduler at 06:50, so its next check is in the evening window
- When a new service is opened on the same database at 07:00 and the scheduler runs at 07:00 and 08:59
- Then no further check of that item happens in the morning window; its next check stays in the evening window
- Corner cases: same behaviour for an evening check followed by a restart at 22:00.

## One-time restart after the upgrade

### TC-025 — Stopped items are restarted on first start

- Covers: FR-AMZ-HCHK-010, FR-AMZ-STOP-008; UC-AMZ-HCHK-004
- Level: integration (SQLite)
- Given a database in the state before this change with 3 Amazon items stopped on 20 Nov 2026 07:26 UTC, a consecutive failure count of 5 and price histories, plus one LeBoncoin item
- When the store is opened (migrations run) and the service starts
- Then no Amazon item is stopped, the failure count is 0, every price history and past check outcome is unchanged, the LeBoncoin item is unchanged, and each Amazon item has a `nextCheckAt` in the next window
- Corner cases: the Amazon gate is not stopped after start (a scheduled check is sent to the collector).

### TC-026 — The restart runs only once

- Covers: FR-AMZ-HCHK-010; UC-AMZ-HCHK-004
- Level: integration (SQLite)
- Given a database on which the one-time restart has already run
- When an Amazon item is stopped again by 5 failures, then the store and service are closed and opened again
- Then the item is still stopped after the second start and its failure count is kept
- Corner cases: opening the database a third time changes nothing either.

### TC-027 — No stopped item: restart still recorded

- Covers: FR-AMZ-HCHK-010; UC-AMZ-HCHK-004
- Level: integration (SQLite)
- Given a database with no stopped Amazon item (and one with no item at all)
- When the store is opened for the first time after the change
- Then nothing is changed, and after a later stop and a reopen the stopped item stays stopped (the restart is recorded as done)
- Corner cases: a fresh empty database created after the change never restarts later stops.

### TC-028 — Protected mode restarts every owner's items

- Covers: FR-AMZ-HCHK-010; UC-AMZ-HCHK-004
- Level: integration (SQLite)
- Given a database in the state before this change with stopped Amazon items of two different owners
- When the store is opened for the first time after the change
- Then the stopped Amazon items of both owners are restarted
- Corner cases: the scheduled windows also apply to both owners' Amazon items (each has a `nextCheckAt` in a window).

### TC-029 — API shows no stopped item after the upgrade

- Covers: FR-AMZ-HCHK-010; UC-AMZ-HCHK-004
- Level: integration (SQLite + HTTP)
- Given the database of TC-025 opened by the application
- When `GET /api/v1/items` is called
- Then every Amazon item has `amazonStoppedAt: null`, so the Amazon tab warning has no stopped item to show
- Corner cases: the response still includes the item's price history and latest price.

### TC-030 — Amazon tab shows no stop warning after the upgrade

- Covers: FR-AMZ-HCHK-010; UC-AMZ-HCHK-004
- Level: e2e (Playwright, `tests/`)
- Given the items API returns the restarted Amazon items with `amazonStoppedAt: null` (as produced by TC-029)
- When the user opens the Amazon tab, on a desktop viewport and on a narrow (mobile) viewport
- Then no Amazon stop warning is shown and no item shows as stopped; the items list and prices are visible
- Corner cases: reaching the Amazon tab with the keyboard (arrow keys between tabs) shows the same result.

## Unchanged behaviour

### TC-031 — LeBoncoin keeps its fixed schedule and requests

- Covers: UC-AMZ-HCHK-001 (LeBoncoin alternative); scope exclusion of the functional specification
- Level: integration (SQLite)
- Given one LeBoncoin item and one Amazon item, a fake clock and the default check times
- When the scheduler runs over one day
- Then the LeBoncoin item is checked at 08:00 and 20:00 with its existing collector, and the Amazon item at its own random times in the windows
- Corner cases: a LeBoncoin failure does not count as an Amazon failure.

### TC-032 — No duplicate check while one is in progress

- Covers: UC-AMZ-HCHK-001 (check already in progress)
- Level: integration (SQLite)
- Given an Amazon item whose Refresh price is still running (blocking fake collector) when its scheduled time passes
- When the scheduler runs
- Then the collector is called only once for that item, and the item still has a scheduled check in a window afterwards
- Corner cases: a scheduled check that is due while a first-check-after-add is still running is not duplicated either.
