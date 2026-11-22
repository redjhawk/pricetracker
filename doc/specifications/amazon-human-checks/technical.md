# Technical specification: Amazon human-like checks

Status: ready (issue #168); TS-AMZ-HCHK-013–014 ready (issue #186)
Functional specification: [functional.md](functional.md), FR-AMZ-HCHK-001–010, FR-AMZ-HCHK-011–012 (issue #186); modified lines FR-AMZ-STOP-001/004/008 ([Amazon request stop](../amazon-request-stop/functional.md)) and FR-AMZ-SRCH-DEL-006 ([Amazon searches removal](../amazon-searches-removal/functional.md))
Test cases: [testcases.md](../../changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n/testcases.md), TC-001–032; issue #186: [testcases.md](../../changes/2026-11-22-1943-issue-186-treat-a-2xx-navigation-with-errortext-as/testcases.md), TC-001–006 (section [Success response not opened as a page](#success-response-not-opened-as-a-page-issue-186))
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "Amazon human-like checks (issue #168)" (no shape change)
Companion: [Amazon request stop technical](../amazon-request-stop/technical.md) (gate, failure count and stops unchanged)

## Requirement mapping

`HCHK-*` abbreviates `FR-AMZ-HCHK-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change |
| --- | --- | --- | --- |
| TS-AMZ-HCHK-001 | HCHK-001 | New `amazon.BrowserCollector`: launches a headless Chromium per check and drives it through the Chrome DevTools Protocol over `--remote-debugging-pipe` (standard library only). It is the only `amazon` collector; the plain HTTP fetching code is deleted | `internal/amazon/browser.go` (new), `internal/amazon/cdp.go` (new), `internal/amazon/collector.go`, `internal/service/service.go` |
| TS-AMZ-HCHK-002 | HCHK-002 | Parsing extracted into HTML-only functions, applied to the rendered DOM; the used-offer requests are made from inside the page with `fetch` | `internal/amazon/collector.go`, `internal/amazon/browser.go` |
| TS-AMZ-HCHK-003 | HCHK-003, STOP-001 | Every browser failure (start, network, timeout, non-2xx other than 404/410, wrong content type, robot check, unsupported redirect) is a `request_error` with request URL, HTTP status and excerpt; the gate counts, logs and stops as today. Amazon check timeout 90 s | `internal/amazon/browser.go`, `internal/service/service.go` |
| TS-AMZ-HCHK-004 | HCHK-004, SRCH-DEL-006 | `nextAmazonCheckAt(after)`: uniform random instant (1 s resolution) in the first window (06:00–09:00, 19:00–23:00 in `config.Location`) that starts strictly after `after`. `PRICEFOLLOWER_CHECK_TIMES` applies to LeBoncoin only | `internal/service/amazon_schedule.go` (new), `internal/service/service.go` |
| TS-AMZ-HCHK-005 | HCHK-005 | One draw per item, each time its next check is set; the scheduler checks only due items, in `next_check_at` order, with the existing 5–60 s pause between checks of one pass | `internal/service/service.go`, `internal/store/store.go` |
| TS-AMZ-HCHK-006 | HCHK-006 | `Add` stores `nextAmazonCheckAt(now)` for an Amazon item; the immediate first check is unchanged | `internal/service/service.go` |
| TS-AMZ-HCHK-007 | HCHK-007, HCHK-008, STOP-008 | `RefreshItem` and `RefreshAll` no longer write a future `next_check_at` of Amazon items (LeBoncoin unchanged); `RefreshItem` redraws an already due one for the next window; stopped items still skipped by Refresh all | `internal/service/service.go` |
| TS-AMZ-HCHK-008 | HCHK-008, STOP-004 | Unchanged mechanism kept: `DueIDs` excludes stopped items and the gate re-reads the stop under its lock before each request, so a stop in a window prevents the remaining checks | `internal/store/store.go` |
| TS-AMZ-HCHK-009 | HCHK-009 | `next_check_at` stays durable and is advanced before the check (existing). A due Amazon item found outside the window it was due in (+15 min grace), or already being checked, is redrawn for the next window without a request | `internal/service/service.go`, `internal/service/amazon_schedule.go`, `internal/store/store.go` |
| TS-AMZ-HCHK-010 | HCHK-010 | One-time store migration guarded by `amazon_request_state.human_checks_restarted`: clears every Amazon stop, resets the count when it restarted items, sets their `next_check_at = NULL` and marks the other Amazon items' pre-change check times for one redraw (`human_checks_redraw`); `service.New` draws a window time for Amazon items with a `NULL` next check and, once, for every marked item | `internal/store/amazon_requests.go`, `internal/store/store.go`, `internal/service/service.go` |
| TS-AMZ-HCHK-011 | all (testability) | `service.NewWithOptions` (collector, clock, random source, scheduler pause) and `s.now()` in the scheduling/recording paths of `service.go` | `internal/service/service.go` |
| TS-AMZ-HCHK-013 | HCHK-011, HCHK-012 | `browserTab.load`: when `Page.navigate` returns an `errorText`, the recorded Document response of the navigated frame is the loaded page only when its status is outside 200–299; a 2xx response, or none, gives the failure `Amazon page could not be opened: <errorText>` (issue #186) | `internal/amazon/browser.go` |
| TS-AMZ-HCHK-014 | HCHK-011 | The collector's failure log line also prints the request URL (`item.URL`, the result's `RequestURL`), so a check that fails in the browser is logged with the request URL and the browser's error on one line (issue #186) | `internal/amazon/collector.go` |
| TS-AMZ-HCHK-012 | HCHK-001, HCHK-003 | Environment `PRICEFOLLOWER_BROWSER_PATH` (read by `amazon.FindBrowser`), `PRICEFOLLOWER_BROWSER_NO_SANDBOX` (read where the service builds the default collector); browser profile in the data directory; Chromium documented as a server requirement (README text deferred to a separate PR) | `internal/amazon/browser.go`, `internal/service/service.go` |

## Backend

### Browser collector (TS-AMZ-HCHK-001, -002, -003)

Library choice: no new Go dependency. Chromium is driven over `--remote-debugging-pipe` (CDP JSON messages separated by NUL on file descriptors 3 and 4), which needs no WebSocket. Reason: the skills ask for standard-library code, and `go.mod` changes are not merged automatically by the factory. Rejected: chromedp/rod/playwright-go (dependency, and rod/playwright download browsers).

```go
// internal/amazon/browser.go
type BrowserOptions struct {
    ExecPath          string                // Chromium/Chrome executable; "" = FindBrowser()
    ProfileDir        string                // persistent profile (cookies); "" = temporary directory removed after the check
    NoSandbox         bool                  // adds --no-sandbox (containers, root)
    NavigationTimeout time.Duration         // 0 = 45 s; page load deadline
    Pause             func() time.Duration  // nil = random 2–5 s after load ("reading"); tests return 0
    Origin            string                // tests only: replaces scheme+host of Amazon URLs and is accepted as a supported host
}
type BrowserCollector struct {
    options BrowserOptions
    mu      sync.Mutex                      // one browser at a time (one profile)
}
func NewBrowserCollector(options BrowserOptions) *BrowserCollector
func (c *BrowserCollector) Collect(ctx context.Context, item model.Listing) model.CollectionResult
func FindBrowser() (string, error)          // PRICEFOLLOWER_BROWSER_PATH; else PATH: chromium, chromium-browser, google-chrome-stable, google-chrome
```

Start failures are tested with a missing executable and one that exits at once, so no launcher hook is needed. With `ExecPath` empty, `FindBrowser` runs at each check: a browser installed later is used without a restart.

`cdp.go` (unexported): `launchBrowser` starts the process with `exec.Command` (cancellation is handled by `close`), `ExtraFiles` = the two pipes (the child ends are closed in the parent, so the reader sees the browser's exit), `SysProcAttr{Setpgid: true}`; one reader goroutine dispatches responses by `id` and events to a per-session handler (`on(sessionID, handler)`, run on the reader goroutine, must not block); `call(ctx, sessionID, method, params, result)`; `close()` sends `Browser.close`, waits up to 5 s for the exit, then kills the process group (also ending children still running after the browser) and always waits for exit and for the reader. gosec G204 on the configured executable is annotated (`#nosec G204`, path from configuration, arguments built by the code).

Flags: `--headless=new`, `--remote-debugging-pipe`, `--user-data-dir=<profile>`, `--no-first-run`, `--no-default-browser-check`, `--disable-blink-features=AutomationControlled`, `--lang=<marketplace language>`, `--window-size=1366,768`, `--no-sandbox` only with `NoSandbox`, `about:blank`. No `--enable-automation`. The browser runs headless: no display is needed and nothing is shown.

`Collect` (holds `mu`; every step honours `ctx`):

1. Launch. Error → failure "browser could not start: <reason>" (status 0).
2. `Browser.getVersion`; user agent = the browser's own, `HeadlessChrome/` replaced by `Chrome/`. The client hints are overridden to match (`userAgentMetadata`: brands Chromium / Google Chrome / Not_A Brand with the engine's version, platform `Linux`, architecture of the server, not mobile), since headless Chromium otherwise announces a `HeadlessChrome` brand. `AMAZON_USER_AGENT` no longer applies to Amazon (still used by LeBoncoin).
3. `Target.createTarget(about:blank)`, `Target.attachToTarget(flatten)`; on the session: `Network.enable`, `Page.enable`, `Emulation.setUserAgentOverride{userAgent, acceptLanguage: locale(marketplace), platform: "Linux x86_64" (or "Linux aarch64"), userAgentMetadata}` (`navigator.platform` keeps the real Linux value).
4. `Page.navigate(item.URL)` (with `Origin` applied in tests). Record the last `Network.responseReceived` of type `Document` per frame and use the navigated frame's (status, MIME type, final URL after redirects). `errorText` with a recorded response of the navigated frame (waited for up to 1 s) whose status is outside 200–299 (Chromium sets `net::ERR_HTTP_RESPONSE_CODE_FAILURE` for a non-2xx answer with an empty body) → that response is the loaded page, without waiting for the load event, so step 7 classifies its status (FR-AMZ-HCHK-012). `errorText` with a 2xx response (a download such as `Content-Disposition: attachment`, or 204/205 without content: the frame stays on `about:blank` or an error page) or without any response (DNS, refused, reset) → failure "Amazon page could not be opened: <errorText>" (FR-AMZ-HCHK-011, issue #186). Wait for `Page.loadEventFired` within `NavigationTimeout`; timeout → failure "Amazon page did not finish loading". Scripts and page resources (stylesheets, images, scripts) load as in a normal browser; cookies are kept in the profile.
5. Pause (`Pause`).
6. Final URL host not a supported Amazon host (or `Origin`) → failure "redirected to an unsupported host". Read `document.documentElement.outerHTML` (`Runtime.evaluate`, `returnByValue`, first 5 MB).
7. Classify with the existing rules, now in `classifyPage(page)` used only here: 404/410 → `unavailable`; other non-2xx → failure "Amazon returned HTTP <n>"; MIME not `text/html`/`application/xhtml+xml` → failure; robot check/captcha pattern → failure "Amazon blocked the product request". Failures keep `RequestURL` (item URL), `HTTPStatus`, `ResponseExcerpt` = first 2 KB of the rendered HTML via the existing `excerpt`; the user message stays "Amazon could not be reached for a price check." and `SecondHandStatus = "check_error"`.
8. Otherwise `parseProductPage(html)` (extracted from today's `productPage`: unavailable text, price, title, thumbnail, `price_not_found`), then `secondHandOfferFromProductPage(html)` as today. If no Amazon offer is on the page, the two used-offer URLs built as today (`usedOfferURLs(item)`) are requested from inside the page: `Runtime.evaluate` with `awaitPromise` of `fetch(url, {credentials: "include", headers: {"X-Requested-With": "XMLHttpRequest"}})` returning `{status, body}` (body ≤ 5 MB; network error → failed response). The browser sets cookies, `Referer` and fetch metadata. Results go to `parseUsedOffers(pages, failedResponses)` (extracted from today's `secondHandOffer`, same logs and results).
9. `close()` in a `defer`: no browser process survives the check, also on error, timeout or cancellation.

`collector.go` keeps `ParseURL`, `hosts`, `locale`, `excerpt`, all parsing helpers and `printRequestError`; it loses `Collector`, `NewCollector`, `fetchPage`, `newRequest`, `withSecondHand` and the HTTP part of `secondHandOffer`. Logs never include cookies or headers (FR-AMZ-HUMAN-012); no Amazon login is used.

### Collection in the service (TS-AMZ-HCHK-001, -003, -011)

- `New(cfg, db)` = `NewWithOptions(cfg, db, Options{})` (the default collector is built only when `Options.Amazon` is nil):

  ```go
  type AmazonCollector interface { Collect(context.Context, model.Listing) model.CollectionResult }
  type Options struct {
      Amazon        AmazonCollector                  // nil = amazon.NewBrowserCollector from config
      Now           func() time.Time                 // nil = time.Now
      Random        func(n int64) int64              // nil = math/rand/v2 Int64N (goroutine-safe); value in [0, n)
      SchedulePause func(context.Context) bool       // nil = waitScheduledRequest (5–60 s); false = cancelled
  }
  ```

  The default collector uses `ExecPath = ""` (so `FindBrowser`: `PRICEFOLLOWER_BROWSER_PATH`, then `PATH`), `ProfileDir = <DataDirectory>/amazon-browser` (created 0700 by the collector), `NoSandbox = PRICEFOLLOWER_BROWSER_NO_SANDBOX == "true"`. When no browser is found, `New` logs once "Amazon browser not found; Amazon checks will fail until Chromium is installed or PRICEFOLLOWER_BROWSER_PATH is set"; each check then fails (counted, may stop items) — the application still starts and LeBoncoin keeps working. The gate gets the same `now`.
- `collectReserved`: Amazon timeout 31 s → `amazonCheckTimeout = 90 * time.Second` (browser start, page with resources, pause, used-offer fetches). The gate lock is held for the whole check, as today; Amazon checks stay strictly sequential.
- `time.Now()` in `Add`, `RefreshAll`, `RefreshItem`, `collect`, `collectReserved` (recording timestamp) and `collectDue` becomes `s.now()`. LeBoncoin session code is untouched.

### Schedule (TS-AMZ-HCHK-004, -005, -006, -007, -009)

```go
// internal/service/amazon_schedule.go
var amazonWindows = []struct{ start, end int }{{6 * 60, 9 * 60}, {19 * 60, 23 * 60}} // local minutes
const amazonWindowGrace = 15 * time.Minute
func nextAmazonCheckAt(after time.Time, location *time.Location, random func(int64) int64) time.Time
func inDueAmazonWindow(dueAt, now time.Time, location *time.Location) bool // now in [start, end + grace] of a window with start <= dueAt
func (s *Service) nextCheckAt(platform string, after time.Time) time.Time // amazon → nextAmazonCheckAt, else NextCheckAt
```

- `nextAmazonCheckAt`: for today, tomorrow and the day after (local dates), for each window, `start := time.Date(y, m, d, start/60, start%60, 0, 0, location)`; the first `start.After(after)` wins; `end` built the same way; result `start + random(int64(end.Sub(start)/time.Second) + 1)` seconds, returned in UTC. Bounds are inclusive ([06:00:00, 09:00:00]). `time.Date` handles daylight saving; the windows contain no skipped or repeated hour in `Europe/Paris`. With another timezone, the windows are that timezone's.
- Examples (TC-012): after 05:00 → today morning; 07:30 → today evening; 14:00 → today evening; 20:30 → tomorrow morning; 23:30 → tomorrow morning. After a check inside a window the next one is always in the following window, so one check per window and two per day.
- `NextCheckAt` (fixed `CheckTimes`) is kept for LeBoncoin and the development seed.
- `Add`: `Insert(..., s.nextCheckAt(platform, s.now()))`.
- `RefreshItem`: `SetNextCheck` only for non-Amazon items (the platform comes from the `ownedListing` it already loads). For an Amazon item, `rescheduleOverdueAmazonCheck` reads `store.NextCheck(ctx, id)` and, only when it is already due (`<= now`, in practice a check skipped while the item was stopped), stores `nextAmazonCheckAt(now)`; a future time is kept. Otherwise the restart by a successful Refresh price would let the next tick run the skipped check right after the manual one (FR-AMZ-HCHK-008). `RefreshAll`: `SetNextChecks` with the non-Amazon IDs only; the platform and stop come from `store.ItemRefs(ctx, owner) ([]store.ItemRef{ID, Platform, Stopped}, error)`, which replaces `IDs` and the per-item `AmazonStopped` queries (same order and filter). Response and queueing unchanged.
- `collectDue` uses `store.DueChecks(ctx, now) ([]store.DueCheck{ID, DueAt}, error)` (the `DueIDs` filter and order: tracked, not stopped, `next_check_at <= now`, plus the due time; `DueIDs` now wraps it) and reads each due item's platform with `store.Listing` (a deleted item is skipped). The in-flight and window rules below are in `skipScheduledCheck(ctx, item, dueAt)`. Per item, in order:
  - LeBoncoin: unchanged (skip if in flight; pause between checks; `SetNextCheck(NextCheckAt(now))`; `Collect`).
  - Amazon in flight (refresh or first check running): `SetNextCheck(nextAmazonCheckAt(now))`, no request — the running check covers this window (TC-032).
  - Amazon with `!inDueAmazonWindow(dueAt, now)`: `now` is outside every window (+ 15 min grace), or inside a window that started after `dueAt` (the check was due in an earlier window, missed while the application was down, e.g. due 08:30 and started at 19:05, or due days ago). `SetNextCheck(nextAmazonCheckAt(now))`, log `scheduled Amazon check of item <id> missed its window; next check <time>`, no request. So a start-up pass never sends the overdue checks of earlier windows back to back. Grace reason: 30 s ticker, 5–60 s pauses and gate queueing must not drop a check due at 08:59.
  - Otherwise: pause if a check already ran in this pass (`SchedulePause`), `SetNextCheck(nextAmazonCheckAt(now))` before the request (restart-safe, FR-HCHK-009), then `Collect` (non-manual; the gate skips an item stopped meanwhile).
- Restart (FR-HCHK-009): nothing is redrawn at start for items that have a time; a stored future time is kept, a check already made has already moved the time to the next window.

### One-time restart (TS-AMZ-HCHK-010)

Store, at the end of `createAmazonRequestSchema` (after `convertGlobalAmazonStop`, so a pre-#111 global stop is converted then restarted):

1. `ensureColumn(amazon_request_state, human_checks_restarted, "INTEGER NOT NULL DEFAULT 0")` and `ensureColumn(amazon_request_state, human_checks_redraw, "INTEGER NOT NULL DEFAULT 0")`.
2. `restartAmazonItemsOnce`: one transaction; if `human_checks_restarted = 1` return; `UPDATE items SET amazon_stopped_at = NULL, next_check_at = NULL WHERE platform = 'amazon' AND amazon_stopped_at IS NOT NULL` (all owners); if rows were affected, `consecutive_failures = 0`; set `human_checks_redraw = 1` if an Amazon item still has a `next_check_at` (the not-stopped items keep the configured check time written before the change, the same instant for all of them); set `human_checks_restarted = 1`; log `restarted <n> stopped Amazon items once (issue #168)` when n > 0. Price observations, attempts and second-hand data are not touched, and opening the store changes no `next_check_at` of a not-stopped item (TC-027).

A fresh database gets the marker at its first open (with no item, so no redraw mark), so later stops are never cleared by a restart. Service: `New` calls `store.AmazonIDsToSchedule(ctx)` (Amazon items with `next_check_at IS NULL`, or every Amazon item while `human_checks_redraw = 1`), sets its own `nextAmazonCheckAt(now)` for each, then `store.ClearAmazonRedraw(ctx)`; on a save error it logs and keeps the mark, so the next start redraws again. Every Amazon item therefore gets its own draw at the first start after the upgrade, and the restarted items are checked at their next window time, not at start-up.

Tests simulate a pre-change database by opening the store, inserting and stopping items, closing it, running `ALTER TABLE amazon_request_state DROP COLUMN human_checks_restarted` on the file, and reopening.

### Concurrency and persistence summary

- No new table; two new columns (`amazon_request_state.human_checks_restarted`, `amazon_request_state.human_checks_redraw`). No transaction is held during a browser check.
- Amazon traffic stays serialized by the gate; `BrowserCollector.mu` additionally guarantees one Chromium per profile directory.
- `Options.Random` must be goroutine-safe (Add, Refresh and the scheduler can draw concurrently); tests wrap a seeded `rand.Rand` in a mutex.
- Shutdown cancels `workerContext`/scheduler context, which ends the CDP calls and kills the browser process group.

## Configuration and documentation (TS-AMZ-HCHK-012)

- `PRICEFOLLOWER_BROWSER_PATH` (optional) is read by `amazon.FindBrowser`, the same lookup the test helper `amazontest.Browser` starts with; `PRICEFOLLOWER_BROWSER_NO_SANDBOX=true` (default false) is read where the service builds the default collector. `config.Config` is not changed. The path is not checked at start-up (a missing browser must not stop the application; only a log line is written).
- `README.md`: Amazon checks need Chromium or Google Chrome on the server (for example `apt install chromium`), the two variables, the profile directory `<data dir>/amazon-browser`, the Amazon windows (twice a day at random times, 06:00–09:00 and 19:00–23:00) while LeBoncoin keeps `PRICEFOLLOWER_CHECK_TIMES`, and that browser tests need a Chromium (`npx playwright install chromium` is found automatically). Deferred to a separate PR (decision by redjhawk on 2026-11-21, issue #168); not part of this change. `deploy/README.md` is not changed (sensitive path).

## Frontend

Not affected: the item fields and their meaning for the UI are unchanged (`nextCheckAt`, `amazonStoppedAt`, statuses); the Amazon tab warning already disappears when no item has `amazonStoppedAt`. Only an e2e case is added (TC-030) to `tests/platform-tabs.spec.ts` (new spec files are not run until `playwright.config.ts` is edited by hand).

## API

No new endpoint, field or error. Behaviour changes documented in API_SPECIFICATION.md "Amazon human-like checks (issue #168)": Amazon `nextCheckAt` is a random time in a window; Refresh price / Refresh all prices no longer change a future Amazon `nextCheckAt`, and Refresh price moves an already passed one (a check skipped while the item was stopped) to a random time in the next window; after the upgrade, items stopped before it return `amazonStoppedAt: null`. Clients are compatible without change.

## Test files

Browser tests use a Chromium found by `internal/amazon/amazontest.Browser(t)` (`PRICEFOLLOWER_BROWSER_PATH`, `PATH`, then Playwright's `$PLAYWRIGHT_BROWSERS_PATH` or `~/.cache/ms-playwright` `chromium-*/chrome-linux*/chrome`); none found → `t.Skip`, except when `CI` is set → `t.Fatal`. They run with `NoSandbox: true`, `Pause` returning 0, a temporary profile, and `Origin` pointing at an `httptest` fixture server (`amazontest.NewServer`, which logs requests with headers). Fixture pages live in `internal/amazon/testdata/`. Schedule tests use `Options.Now`, a seeded `Options.Random` and a `SchedulePause` returning true at once.

| Test case | Test file |
| --- | --- |
| TC-001 | `internal/service/amazon_human_checks_test.go` |
| TC-002, TC-003, TC-004, TC-006, TC-007 | `internal/amazon/browser_test.go` (+ `internal/amazon/testdata/*.html`, `internal/amazon/amazontest/amazontest.go`) |
| TC-005, TC-016, TC-017, TC-018, TC-019, TC-029 | `internal/httpapi/amazon_human_checks_test.go` (`service.NewWithOptions` with fake collector and clock) |
| TC-008 | `internal/httpapi/amazon_human_checks_test.go` (Refresh price through HTTP; scheduled checks through the exported `Service.Collect`) |
| TC-009, TC-010, TC-011, TC-012, TC-013, TC-014 | `internal/service/amazon_schedule_test.go` |
| TC-015, TC-020, TC-021, TC-022, TC-023, TC-024, TC-031, TC-032 | `internal/service/amazon_human_checks_test.go` (TC-020 with the real browser collector and fixture server) |
| TC-025 | `internal/store/amazon_human_checks_test.go` (data) and `internal/service/amazon_human_checks_test.go` (`nextCheckAt`, gate not stopped) |
| TC-026, TC-027, TC-028 | `internal/store/amazon_human_checks_test.go` (TC-028 window part in `internal/service/amazon_human_checks_test.go`) |
| TC-030 | `tests/platform-tabs.spec.ts` |

Existing tests keep compiling and passing: `internal/amazon/collector_test.go` tests `productPrice`, which stays; no existing test uses `store.IDs`, `amazon.NewCollector` or Refresh moving Amazon `next_check_at`; `store.DueIDs` (used by `internal/store/amazon_requests_test.go`) keeps its signature and result.

## Scope and refactoring

| Ref | Refactoring | Reason | Scope | Risk |
| --- | --- | --- | --- | --- |
| R-1 | Split `collector.go` parsing from HTTP fetching; delete the HTTP Amazon client | FR-HCHK-001 (no plain download remains) and FR-HCHK-002 (same parser on the browser HTML) | `internal/amazon/collector.go` | Parser regression: covered by TC-004 on fixtures and the existing `productPrice` test; Chrome's HTML serialization uses double-quoted attributes, which the regexes already expect |
| R-2 | `NewWithOptions`, `s.now()` in service paths | Deterministic tests with clock, random source and fake collector through HTTP | `internal/service/service.go` | Low: defaults equal today's values |
| R-3 | `IDs` → `ItemRefs` (id, platform, stopped) | Refresh all needs the platform; one query instead of one stop query per item | `internal/store/store.go`, `RefreshAll` | Low: same SQL filter and order |
| R-4 | `store.get` reads an item in one read transaction | Its queries could straddle a recorded collection (new attempt with the old price history), which made TC-005 flaky | `internal/store/store.go` | Low: read-only, same queries; no caller holds a transaction |

Out of scope: LeBoncoin requests and schedule, the failure count and stop rules, price parsing rules, UI, `deploy/` and `go.mod` (no change needed).

## Success response not opened as a page (issue #186)

Origin: review comment R5-01 on #179 (`doc/todo/issue-168-r5-01.md`). Before this change, every `errorText` navigation with a recorded Document response returned that response as the loaded page, a 2xx one included. A 2xx answer that Chromium does not open (download, 204/205) leaves the frame on `about:blank` or an error page. `parseProductPage` then found no price, so the check gave `price_not_found`, an Amazon answer that resets the gate's failure count.

### Frontend

Not affected: no new state, field or message. The failure surfaces through the existing retrieval issue, second-hand `check_error`, stop warning and "refresh failed" states (`tests/item-refresh.spec.ts`, `tests/platform-tabs.spec.ts` unchanged and must keep passing).

### Backend (TS-AMZ-HCHK-013, -014)

`internal/amazon/browser.go`, `browserTab.load`, `errorText` branch only:

```go
if navigation.ErrorText != "" {
    // A non-2xx answer with an empty body also ends with an error text (Chromium then shows its own
    // error page): its status still classifies the page. A 2xx answer the browser did not open as a
    // page (a download, no content) fails like a navigation without an answer.
    if page, found := awaitDocument(loadContext, document, responded); found && (page.status < 200 || page.status >= 300) {
        return page, nil
    }
    return loadedPage{}, fmt.Errorf("Amazon page could not be opened: %s", navigation.ErrorText)
}
```

- The failure goes through the existing path. `Collect` returns `requestFailure` with `Result: "request_error"`, the message "Amazon could not be reached for a price check.", `SecondHandStatus: "check_error"`, `RequestURL` = item URL and `HTTPStatus` 0, because no page was opened. The gate logs the failure (`Amazon request failed url=… status=0 …`), counts it and stops every Amazon item at the 5th consecutive failure. `RecordCollection` keeps the last price and second-hand offer as for any failure. No used-offer fetch is made.
- FR-AMZ-HCHK-012 is unchanged. A recorded status outside 200–299 (404/410/403/503 with an empty body, or a status below 200) is still returned without waiting for the load event, so `classifyPage` gives `unavailable` for 404/410 and "Amazon returned HTTP <n>" otherwise, with `HTTPStatus` set.
- The 1 s wait in `awaitDocument` is kept: when Chromium reports the error before the response event, a non-2xx status must still be found. A 2xx response that arrives after the error is also a failure.
- A navigation without `errorText` is unchanged. The `Content-Disposition: inline` case opens the page normally (TC-001 corner case). If some Chromium version ended a download or 204 navigation without `errorText`, the load wait would end in the existing failure "Amazon page did not finish loading", which is still a failure and never `price_not_found`.
- `internal/amazon/collector.go`, `printRequestError(item, reason)`: append the request URL, giving `Amazon collection failed for <marketplace>/<ASIN>: <reason> url=<item.URL>`. `item.URL` is the canonical product URL from `ParseURL` and holds no cookie, header or secret. This is the only caller (`requestFailure`), so the format gains a field for every collector failure.
- Validation, persistence, migrations: none. Concurrency: unchanged. The gate lock and `BrowserCollector.mu` still serialize checks, each check starts its own browser, and a failed download leaves nothing for the next check.

### API

No new endpoint, field, error code or shape change. Behaviour: an Amazon check answered with a success status that the browser does not open as a page now gives `lastAttempt.result: "request_error"` (item `status: "retrieval_error"`, second-hand `check_error`) instead of `price_not_found`. `POST /api/v1/items/{id}/refresh` and `POST /api/v1/items/refresh` still answer `202`. `API_SPECIFICATION.md`, section "Amazon human-like checks (issue #168)", gets one bullet. Clients need no change.

### Files to change

| File | Change |
| --- | --- |
| `internal/amazon/browser.go` | TS-AMZ-HCHK-013 (`load`, `errorText` branch and its comment) |
| `internal/amazon/collector.go` | TS-AMZ-HCHK-014 (`printRequestError` prints the URL) |
| `internal/amazon/browser_navigation_test.go` (new) | TC-001, TC-002, TC-003 |
| `internal/httpapi/amazon_browser_navigation_test.go` (new) | TC-004, TC-005, TC-006 |
| `API_SPECIFICATION.md` | Behaviour bullet (done in this stage) |

### Test files

Test names: `TestBrowserNavigation_TCnnn_…` in `internal/amazon` and `TestAmazonBrowserNavigation_TCnnn_…` in `internal/httpapi`, because the `TestBrowserCollector_TCnnn_` and `TestAmazonHumanChecks_TCnnn_` names of issue #168 are already taken. Each test file carries a `// TC-nnn` comment per test. All tests use the real Chromium (`amazontest.Browser`), `amazontest.NewServer` and `Origin`. They need no new fixture file: the body is `amazontest.Fixture(t, "price-found.html")`, whose price is 129.99.

| Test case | File | Notes |
| --- | --- | --- |
| TC-001 | `internal/amazon/browser_navigation_test.go` | A mux whose `/dp/B0PRICE001` handler serves a mode switched with an `atomic.Int32` (attachment / normal / inline), with `/` delegating to the existing `fixtureHandler(t)` for resources and used offers. The collector is built from `testBrowserOptions`, the check runs through `browserCheck`, and the log is read with `captureOutput`. Assert the failure fields (`amazonCheckFailed`, `check_error`, `RequestURL` suffix), that the log contains `/dp/B0PRICE001` and `Amazon page could not be opened: net::`, then a second check on the same collector in normal mode → `success`, 12999 cents. The inline subtest → `success`, 12999. |
| TC-002 | `internal/amazon/browser_navigation_test.go` | Subtests 204 and 205: `w.WriteHeader(status)` and no body. Assert the failure (never `price_not_found`/`unavailable`) and the log content as in TC-001. |
| TC-003 | `internal/amazon/browser_navigation_test.go` | Subtests 404, 410, 503, 403 with `Content-Type: text/html` and an empty body. 404/410 → `unavailable` with `unavailableMessage`. 503/403 → failure with `HTTPStatus` = the status. The excerpt (Chromium's own error page) is not asserted. |
| TC-004 | `internal/httpapi/amazon_browser_navigation_test.go` | `openBrowserProfileAPI` with `amazon.NewBrowserCollector(browserProfileOptions(t, filepath.Join(dir, "amazon-browser"), server.URL))`. A switchable fixture server answers 503 page / attachment / empty-body statuses / normal page. Items come from `insertHumanItem` + `RecordCollection` (success 1999 cents, second-hand `available` with an amount and condition). Scheduled checks run with `clock.Set` + `items.Collect`. A local `captureCheckOutput` helper captures `log` and `os.Stdout` (as `captureOutput` of `internal/amazon`), because the collector line is printed to stdout. Stops are read from `listItems`. The protected-mode subtest inserts the items with two `OwnerID`s (as `insertOwned` in `internal/service`) and asserts both stopped through the store (`AmazonStopped`). |
| TC-005 | `internal/httpapi/amazon_browser_navigation_test.go` | `refresh` + `waitBrowserCheck`, then `POST /api/v1/items/refresh` (expects `202`) and a wait for the next attempt. Assert `itemDetails`/`listItems`: `retrieval_error`, `request_error`, second-hand `check_error`, price and second-hand offer unchanged. Stopped item: `database.StopAmazonItems` beforehand, `amazonStoppedAt` still set afterwards. Repeated refresh: `storedFailures` grows by one per attempt. |
| TC-006 | `internal/httpapi/amazon_browser_navigation_test.go` | The same harness. 4 real checks answered 503 with a page, then 404 with an empty body (subtest: 410), then 4 checks of 503 with an empty body (no item stopped), then one more (every item stopped). Each 503 gate line contains `status=503`. |

The existing tests must keep passing, in particular `TestBrowserCollector_TC004`, `TC006` and `TC007` in `internal/amazon/browser_test.go`: an error text without a response still fails, and a non-2xx with a page is unchanged. No existing test asserts the `printRequestError` format.

### Refactoring

None. Out of scope: download behaviour of the browser (for example `Browser.setDownloadBehavior` to deny downloads), which concerns where Chromium may save an attachment, not the check result; the failure count and stop rules; the UI.

## Verification and open questions

- `gofmt -l .`, `go vet ./...`, `go test ./...` (browser tests need Chromium; present in the CI image under `/ms-playwright`), `npm run build`, `npx playwright test`; gosec and govulncheck.
- QA on a machine with Chromium: add an Amazon item and watch the logs for a browser check; `nextCheckAt` of Amazon items in a window; Refresh price keeps it; a missing browser gives counted failures and the start-up log line.
- Open functional questions: none. Technical interpretations recorded: overdue Amazon checks outside their window (beyond 15 min) are redrawn instead of sent (FR-HCHK-004); a scheduled check due while the item is already being checked is redrawn (no duplicate).
- Issue #186: no open question. The browser tests (TC-001–006) need Chromium; with `CI` set they cannot skip.
