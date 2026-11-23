# Technical specification: Amazon human-like checks

Status: ready (issue #168); TS-AMZ-HCHK-013–014 ready (issue #186); TS-AMZ-HCHK-015 ready (issue #187); TS-AMZ-HCHK-016–017 ready (issue #217); TS-AMZ-HCHK-018–019 ready (issue #233)
Functional specification: [functional.md](functional.md), FR-AMZ-HCHK-001–010, FR-AMZ-HCHK-011–012 (issue #186), FR-AMZ-HCHK-013 (issue #187), FR-AMZ-HCHK-014 (issue #217), FR-AMZ-HCHK-015 (issue #233); modified lines FR-AMZ-STOP-001/004/008 ([Amazon request stop](../amazon-request-stop/functional.md)) and FR-AMZ-SRCH-DEL-006 ([Amazon searches removal](../amazon-searches-removal/functional.md))
Test cases: [testcases.md](../../changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n/testcases.md), TC-001–032; issue #186: [testcases.md](../../changes/2026-11-22-1943-issue-186-treat-a-2xx-navigation-with-errortext-as/testcases.md), TC-001–006 (section [Success response not opened as a page](#success-response-not-opened-as-a-page-issue-186)); issue #187: [testcases.md](../../changes/2026-11-23-1043-issue-187-skip-used-offer-requests-when-the-produc/testcases.md), TC-001–004 (section [Unavailable listing on the browser's error page](#unavailable-listing-on-the-browsers-error-page-issue-187)); issue #217: [testcases.md](../../changes/2026-11-23-1124-issue-217-deny-downloads-in-the-amazon-browser-and/testcases.md), TC-001–004 (section [Downloads refused](#downloads-refused-issue-217)); issue #233: [testcases.md](../../changes/2026-11-23-1234-issue-233-clear-the-amazon-browser-http-cache-even/testcases.md), TC-001–006 (section [Interrupted check](#interrupted-check-issue-233))
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
| TS-AMZ-HCHK-015 | HCHK-013 (with HCHK-012) | `loadedPage.errorPage` is set when `load` returns the non-2xx response of an `errorText` navigation (the tab shows Chromium's error page); `Collect` then returns the unavailable result with `SecondHandStatus = "check_error"` without `fetchAll`, so no used-offer request and no used-offer log line (issue #187) | `internal/amazon/browser.go` |
| TS-AMZ-HCHK-016 | HCHK-014 | `browserTab.setUp` calls `Browser.setDownloadBehavior{behavior: "deny"}` on the browser (default browser context) before the page is created; an error fails the check as a browser start failure, before any navigation (issue #217) | `internal/amazon/browser.go` |
| TS-AMZ-HCHK-017 | HCHK-014 (with HCHK-011) | `browserTab.load`: when an `errorText` navigation did not record a non-2xx Document response (a download, no content, or no response seen within the wait), `Network.clearBrowserCache` is called before the failure is returned, so the response body is not kept in the profile's HTTP disk cache (issue #217) | `internal/amazon/browser.go` |
| TS-AMZ-HCHK-018 | HCHK-015 (with HCHK-014) | `browserTab.clearCache(ctx)` sends `Network.clearBrowserCache` in its own context, `context.WithTimeout(context.WithoutCancel(ctx), 5 s)`. The end of the check's context never abandons the clear, and a browser that does not answer holds the check for at most 5 s. The failure log line gives the real cause: a protocol error, the browser exited, or no answer within 5 s (issue #233) | `internal/amazon/browser.go` |
| TS-AMZ-HCHK-019 | HCHK-015 (with HCHK-014) | `browserTab.load`: when `Page.navigate` fails because `loadContext` ended (check interrupted, or navigation timeout), `clearCache` runs before the error is returned. The result is unchanged (issue #233) | `internal/amazon/browser.go` |
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
3. `Browser.setDownloadBehavior{behavior: "deny"}` (issue #217, TS-AMZ-HCHK-016), then `Target.createTarget(about:blank)`, `Target.attachToTarget(flatten)`; on the session: `Network.enable`, `Page.enable`, `Emulation.setUserAgentOverride{userAgent, acceptLanguage: locale(marketplace), platform: "Linux x86_64" (or "Linux aarch64"), userAgentMetadata}` (`navigator.platform` keeps the real Linux value).
4. `Page.navigate(item.URL)` (with `Origin` applied in tests). Record the last `Network.responseReceived` of type `Document` per frame and use the navigated frame's (status, MIME type, final URL after redirects). `errorText` with a recorded response of the navigated frame (waited for up to 1 s) whose status is outside 200–299 (Chromium sets `net::ERR_HTTP_RESPONSE_CODE_FAILURE` for a non-2xx answer with an empty body) → that response is the loaded page, marked `errorPage` (the tab shows `chrome-error://chromewebdata/`), without waiting for the load event, so step 7 classifies its status (FR-AMZ-HCHK-012). `errorText` with a 2xx response (a download such as `Content-Disposition: attachment`, or 204/205 without content: the frame stays on `about:blank` or an error page) or without any response (DNS, refused, reset) → failure "Amazon page could not be opened: <errorText>" (FR-AMZ-HCHK-011, issue #186); with a 2xx response the HTTP cache is cleared first (issue #217, TS-AMZ-HCHK-017), also when the check is interrupted. An interrupted `Page.navigate` clears it too (issue #233, TS-AMZ-HCHK-018–019). Wait for `Page.loadEventFired` within `NavigationTimeout`; timeout → failure "Amazon page did not finish loading". Scripts and page resources (stylesheets, images, scripts) load as in a normal browser; cookies are kept in the profile.
5. Pause (`Pause`).
6. Final URL host not a supported Amazon host (or `Origin`) → failure "redirected to an unsupported host". Read `document.documentElement.outerHTML` (`Runtime.evaluate`, `returnByValue`, first 5 MB).
7. Classify with the existing rules, now in `classifyPage(page)` used only here: 404/410 → `unavailable`; other non-2xx → failure "Amazon returned HTTP <n>"; MIME not `text/html`/`application/xhtml+xml` → failure; robot check/captcha pattern → failure "Amazon blocked the product request". Failures keep `RequestURL` (item URL), `HTTPStatus`, `ResponseExcerpt` = first 2 KB of the rendered HTML via the existing `excerpt`; the user message stays "Amazon could not be reached for a price check." and `SecondHandStatus = "check_error"`.
8. Otherwise `parseProductPage(html)` (extracted from today's `productPage`: unavailable text, price, title, thumbnail, `price_not_found`), then `secondHandOfferFromProductPage(html)` as today. A page marked `errorPage` (only a 404/410 reaches this step, as `unavailable`) gets `SecondHandStatus = "check_error"` and no used-offer request: `fetch` from Chromium's opaque-origin error page cannot succeed (issue #187, FR-AMZ-HCHK-013). Otherwise, if no Amazon offer is on the page, the two used-offer URLs built as today (`usedOfferURLs(item)`) are requested from inside the page: `Runtime.evaluate` with `awaitPromise` of `fetch(url, {credentials: "include", headers: {"X-Requested-With": "XMLHttpRequest"}})` returning `{status, body}` (body ≤ 5 MB; network error → failed response). The browser sets cookies, `Referer` and fetch metadata. Results go to `parseUsedOffers(pages, failedResponses)` (extracted from today's `secondHandOffer`, same logs and results).
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
- Shutdown cancels `workerContext`/scheduler context, which ends the CDP calls and kills the browser process group. A cache clear that is due still runs first, for at most 5 s (issue #233, TS-AMZ-HCHK-018).

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
- Validation, persistence, migrations: none. Concurrency: unchanged. The gate lock and `BrowserCollector.mu` still serialize checks, and each check starts its own browser, so the failure state of one check (the navigation error) never reaches the next one. This change alone did not stop Chromium from saving the attachment; the disk guarantee comes from issue #217 (section [Downloads refused](#downloads-refused-issue-217)).

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

None. Out of scope: the failure count and stop rules; the UI. The browser's download behaviour is specified by issue #217 (section [Downloads refused](#downloads-refused-issue-217)).

## Unavailable listing on the browser's error page (issue #187)

Origin: review comment R5-02 on #179 (`doc/todo/issue-168-r5-02.md`). A 404/410 product page with an empty body reaches `Collect` through the `errorText` branch of `load` (TS-AMZ-HCHK-013). The tab then shows `chrome-error://chromewebdata/`, an opaque origin. `Collect` still called `tab.fetchAll`. The two `fetch` calls (`credentials: "include"`, header `X-Requested-With`) cannot succeed from that origin and may send a CORS preflight with `Origin: null`. `parseUsedOffers` then logged a misleading `Amazon used-offer check failed for …` line.

### Frontend

Not affected: no new state, field or message. **Listing unavailable** and the second-hand check error are existing states (`tests/item-refresh.spec.ts`, `tests/platform-tabs.spec.ts` unchanged and must keep passing).

### Backend (TS-AMZ-HCHK-015)

`internal/amazon/browser.go` only:

```go
type loadedPage struct {
    status    int
    mimeType  string
    url       string // after redirects
    html      string // rendered DOM
    errorPage bool   // the browser did not open the answer and shows its own error page
}
```

- `browserTab.load`, `errorText` branch: the non-2xx response it returns gets `errorPage = true` (`page.errorPage = true; return page, nil`). No other path sets it. The 1 s `awaitDocument` wait and the 2xx failure are unchanged.
- `Collect`: `classifyPage` is unchanged, so an error page with a status other than 404/410 is still a failure (step 7). Before the used-offer step, a page with `errorPage` returns the result built so far (`Result: "unavailable"`, `unavailableMessage`, `HTTPStatus` = page status) with `SecondHandStatus = "check_error"`, amount 0 and no condition. `fetchAll` and `parseUsedOffers` are not called, so no used-offer request and no used-offer log line. No new log line is added: the check is an Amazon answer, not a failure. The rendered HTML of the error page is still read (step 6) as today. Sketch:

  ```go
  if page.errorPage {
      // Chromium's error page has an opaque origin: used-offer requests from it cannot succeed (issue #187).
      result.SecondHandStatus = "check_error"
      return result
  }
  urls := usedOfferURLs(item)
  ```

- The flag lives in the `loadedPage` value of one check. The next check on the same collector (same profile) loads a new page, so the skip never carries over.
- A 404/410 page that Chromium opens (with a body) has no `errorText` and no `errorPage`: its used-offer requests are sent as before.
- Service, gate and store unchanged. `unavailable` is an Amazon answer: the gate resets the consecutive failure count and Refresh price restarts a stopped item (FR-AMZ-STOP-006). `RecordCollection` stores the `check_error` second-hand check and adds no offer observation, so the last detected offer is kept.
- Validation, persistence, migrations: none. Concurrency: unchanged (gate lock, `BrowserCollector.mu`).

### API

No new endpoint, field, error code or shape change. Behaviour: an Amazon product page answered with 404/410 that the browser does not open gives `lastAttempt.result: "unavailable"` (item `status: "unavailable"`), second-hand status `check_error` with the last detected offer unchanged, and Amazon gets no used-offer request for that check. Before the change this check usually also ended in `check_error`, after failed used-offer requests. `API_SPECIFICATION.md`, section "Amazon human-like checks (issue #168)", gets one bullet. Clients need no change.

### Files to change

| File | Change |
| --- | --- |
| `internal/amazon/browser.go` | TS-AMZ-HCHK-015 (`loadedPage.errorPage`, `load` `errorText` branch, `Collect` skip before `usedOfferURLs`) |
| `internal/amazon/browser_error_page_test.go` (new) | TC-001, TC-002 |
| `internal/httpapi/amazon_browser_error_page_test.go` (new) | TC-003, TC-004 |
| `API_SPECIFICATION.md` | Behaviour bullet (done in this stage) |

### Test files

Test names: `TestBrowserErrorPage_TCnnn_…` in `internal/amazon` and `TestAmazonBrowserErrorPage_TCnnn_…` in `internal/httpapi`, because the TC numbers of issues #168 and #186 are already used by other names. Each test file carries a `// TC-nnn` comment per test. All tests use the real Chromium (`amazontest.Browser`), `amazontest.NewServer` and `Origin`. No new fixture file.

Used-offer requests are counted on both used-offer paths: `/gp/aod/ajax/ref=tmm_pap_used_aod_0` (used offers) and `/gp/aod/ajax` (all offers, built by `usedOfferURLs`). A local helper returns `server.Count` of the first path plus `server.Count` of the second. `Count` matches the path regardless of method, so an `OPTIONS` preflight is counted too. A normal check sends 2.

| Test case | File | Notes |
| --- | --- | --- |
| TC-001 | `internal/amazon/browser_error_page_test.go` | A mux with `/` delegating to `fixtureHandler(t)` (resources); `/gp/aod/ajax` and `/gp/aod/ajax/` serving `aod-amazon-used.html` (Amazon Seconde main, 95.50, "Comme neuf"); `/dp/B0PRICE001` switched with an `atomic.Int32` between empty 404, empty 410 (`Content-Type: text/html`, `WriteHeader`, no body) and `price-found.html`. Subtests 404 and 410 build the collector from `testBrowserOptions`, run `browserCheck` under `captureOutput` and assert `Result: "unavailable"`, `Message: unavailableMessage`, `HTTPStatus` = status, `SecondHandStatus: "check_error"`, amount 0, empty condition, used-offer count 0, no log line containing `Amazon used-offer check failed for`. Then on the same collector, after switching to `price-found.html`: `success`, 12999, used-offer count 2, `SecondHandStatus: "available"`, 9550. |
| TC-002 | `internal/amazon/browser_error_page_test.go` | Same mux shape, `/dp/B0PRICE001` answering 404 or 410 (subtests) with `<html><body><h1>Page introuvable</h1></body></html>`. Assert `unavailable`, used-offer count > 0, `SecondHandStatus: "available"`, 9550. Subtest with the used-offer paths answering 503 (with a body): `check_error` and a log line containing `Amazon used-offer check failed for amazon.fr/B0PRICE001`. |
| TC-003 | `internal/httpapi/amazon_browser_error_page_test.go` | Reuses `openNavigationAPI`, `answerEmpty404`/`answerEmpty410`/`answerEmpty503`, `api.check`, `captureCheckOutput`, `linesWith`, `storedFailures`, `itemDetails`, `listItems` of `amazon_browser_navigation_test.go`. Set 4 failures with `api.database.SaveAmazonFailures(ctx, 4)`, `api.fixture.Reset()`, capture output, run the check (subtests 404 and 410). Assert from `itemDetails` and `listItems`: `status: "unavailable"`, `lastAttempt.result: "unavailable"`, second-hand `check_error` with `LatestDetection` 8950 kept; `storedFailures` 0; `Count("/dp/<ASIN>")` 1 and used-offer count 0; no `Amazon used-offer check failed for` and no `Amazon request failed` line. Then one `answerEmpty503` check: no item stopped. Corner case "no earlier offer": an item added with `database.Insert` only, checked the same way, shows `check_error` without `LatestDetection`. Protected mode: `openNavigationAPI(t, ids, 1, 2)`, each item checked with `answerEmpty404`: both unavailable, used-offer count 0. |
| TC-004 | `internal/httpapi/amazon_browser_error_page_test.go` | `openNavigationAPI` with two ids, `answerEmpty404`. `refresh` + `waitBrowserCheck`, then `POST /api/v1/items/refresh` (expects `202`) and `waitBrowserCheck` at the next clock time, then a repeated `refresh`: after each, `itemDetails` shows `unavailable` with second-hand `check_error` and the offer at 89.50 kept; used-offer count 0 in total; no used-offer failure line. Stopped item: `api.database.StopAmazonItems` beforehand, `refresh` of the first id → its `amazonStoppedAt` is `null`, the second item stays stopped, used-offer count 0. |

The existing tests must keep passing, in particular `TestBrowserCollector_TC004`, `TC006`, `TC007` (`internal/amazon/browser_test.go`), `TestBrowserNavigation_TC003` (`internal/amazon/browser_navigation_test.go`) and `TestAmazonBrowserNavigation_TC003`–`TC006` (`internal/httpapi/amazon_browser_navigation_test.go`): FR-AMZ-HCHK-012 is unchanged. None of them asserts used-offer requests after an empty-body 404/410.

### Refactoring

None. Out of scope: reading the error page's HTML (kept, harmless), the empty-body follow-up of other statuses (503/429), the failure count and stop rules, the UI.

## Downloads refused (issue #217)

Origin: review comment R1-01 on #216 (`doc/todo/issue-186-r1-01.md`). `browser.go` set no download behaviour. With `--headless=new`, Chromium handles an attachment like a desktop browser does and may save it to the account's Downloads folder (`$HOME/Downloads`, or the `XDG_DOWNLOAD_DIR` of `user-dirs.dirs`). The issue #186 sentence "a failed download leaves nothing for the next check" was only true for the check result.

### Frontend

Not affected: no new state, field or message. A download is still the failed check of issue #186 (`tests/item-refresh.spec.ts`, `tests/platform-tabs.spec.ts` unchanged and must keep passing).

### Backend (TS-AMZ-HCHK-016, -017)

`internal/amazon/browser.go` only.

- **TS-AMZ-HCHK-016**, `browserTab.setUp`: right after `Browser.getVersion` and before `Target.createTarget`:

  ```go
  // Amazon answers offered as a file must never be saved on the server's disk (issue #217).
  if err := t.browser.call(ctx, "", "Browser.setDownloadBehavior", map[string]any{"behavior": "deny"}, nil); err != nil {
      return err
  }
  ```

  - The call goes to the browser itself (session `""`). Without `browserContextId` it applies to the default browser context, which holds the page created next and every frame in it (iframes included). With `deny`, Chromium cancels the download before choosing a target file. It deletes its temporary file and never creates the Downloads folder.
  - The setting lives in the browser process and is not stored in the profile. Each check starts its own browser (`open`), so every check sets it again.
  - An error from this call fails `setUp`, so `open` returns `browser could not start: Browser.setDownloadBehavior: <message>`. The check is the existing counted browser start failure, and no page is requested. This is fail-closed: a check never runs with downloads allowed.
  - `fetch` used-offer requests and subresources never start downloads; only navigations do, and the setting covers them all.
- **TS-AMZ-HCHK-017**, `browserTab.load`, `errorText` branch. The network stack stores a navigation response in the profile's HTTP disk cache (`<profile>/Default/Cache`, since the profile is outside `~/.config`) as it reads the body. It does this even when the response then goes to the download system. A small attachment can therefore be cached whole before the download is cancelled. Unless the branch found a non-2xx response (an error page), the HTTP cache is cleared before the failure is returned. The clear does not depend on `awaitDocument` finding the response, since Chromium may report the error before the response event and that event may come after the 1 s wait:

  ```go
  if navigation.ErrorText != "" {
      page, found := awaitDocument(loadContext, document, responded)
      if found && (page.status < 200 || page.status >= 300) {
          page.errorPage = true
          return page, nil
      }
      // A 2xx answer the browser did not open (a download, no content) may already be in the profile's
      // HTTP cache, also when its response event comes late: the answer must not stay on disk (issue #217).
      t.clearCache(ctx) // own bounded context since issue #233 (TS-AMZ-HCHK-018)
      return loadedPage{}, fmt.Errorf("Amazon page could not be opened: %s", navigation.ErrorText)
  }
  ```

  - The cache entry is created when the request starts, so it exists at this point. Clearing dooms the active entries too: a cache write still running is deleted when it ends.
  - The call runs in its own context: the check's `ctx` without its cancellation and deadline, bounded by 5 s (issue #233, TS-AMZ-HCHK-018). Neither the end of `loadContext` nor the end of the check abandons it. An error is logged with its real cause and the result stays the same failure.
  - Cookies and other site data are kept: `Network.clearBrowserCache` clears only the HTTP cache. After such a failure, the next check reloads page resources over the network, as a returning visitor with an empty cache would. This happens only after a failed navigation that is not a non-2xx error page.
  - A navigation without a response (DNS, refused) also clears the HTTP cache, since a late response cannot be told apart from none; its result stays the same failure. A non-2xx answer (TS-AMZ-HCHK-013/015) is unchanged.
- Check results, messages, logs, the gate, the failure count and stops are unchanged (FR-AMZ-HCHK-011). `Browser.close` ends each check, and the browser shuts down cleanly before the process group is killed, so no partial download is left behind. Service, store, validation, persistence and migrations: none. Concurrency: unchanged (gate lock, `BrowserCollector.mu`).
- Compatibility: `Browser.setDownloadBehavior` and `Network.clearBrowserCache` exist in every Chromium/Chrome that supports `--headless=new`, so no version is newly excluded.

### API

Not affected: no endpoint, field, error code or result changes. The check result of a download stays as described for issue #186 in `API_SPECIFICATION.md`, so that file is not changed.

### Files to change

| File | Change |
| --- | --- |
| `internal/amazon/browser.go` | TS-AMZ-HCHK-016 (`setUp`), TS-AMZ-HCHK-017 (`load`, `errorText` branch) |
| `internal/amazon/amazontest/amazontest.go` | Test helpers `IsolatedHome`, `DownloadMarker`, `AssertNoDownload`, `AssertDownloadsFolder` |
| `internal/amazon/browser_navigation_test.go` | TC-001, TC-002 |
| `internal/httpapi/amazon_browser_navigation_test.go` | Marker in the attachment body, new `answerPage` answer, TC-004 |
| `internal/httpapi/amazon_browser_download_test.go` (new) | TC-003 |

### Test files

Shared helpers in `internal/amazon/amazontest` (a non-test package already imported by both test packages):

```go
// DownloadMarker is added to the bodies offered as downloads; normal pages never contain it.
const DownloadMarker = "<!-- download-marker-217 -->"

// IsolatedHome sets HOME and XDG_CONFIG_HOME (t.Setenv) to new directories of the test and returns the home,
// so the browser's default Downloads folder is <home>/Downloads. Call it after Browser(t), which may look
// for Playwright's Chromium under the real home.
func IsolatedHome(t testing.TB) string

// AssertNoDownload fails t when a root holds a file left by a download, after the browser has closed:
// a regular file whose base name is one of names, or that name uniquified by Chromium ("page (1).html"),
// or that starts with "download"; a partial download (suffix ".crdownload", prefix "Unconfirmed ",
// ".com.google.Chrome." or ".org.chromium.Chromium."); or a regular file containing DownloadMarker.
// Missing roots are ignored; symbolic links are not followed.
func AssertNoDownload(t testing.TB, names []string, roots ...string)

// AssertDownloadsFolder fails t unless <home>/Downloads is absent (existed false) or an empty directory (existed true).
func AssertDownloadsFolder(t testing.TB, home string, existed bool)
```

Every check returns only after `close`, so the assertions run on a closed browser. No test uses `t.Parallel`, so `t.Setenv` is allowed. Chromium may create unrelated files in the isolated home (for example `.pki/nssdb`, `.cache/fontconfig`); they are neither named like a download nor hold the marker. Each test file carries a `// TC-nnn` comment per test. Existing names are reused where the test cases extend a test. The new test is named `TestAmazonBrowserDownload_TC003_…`, because `TC003` of issues #186 and #187 is taken in the other packages and the prefix keeps issue #217 apart.

| Test case | File | Notes |
| --- | --- | --- |
| TC-001 | `internal/amazon/browser_navigation_test.go` | Extends the subtest "attachment then a normal page" of `TestBrowserNavigation_TC001_AttachmentIsAFailure` (comment `// TC-001, TC-002 (issue #217)` added on the function). The attachment body becomes `page + amazontest.DownloadMarker`, and the normal and inline bodies stay `page`. After `testBrowserOptions` (which resolves the browser), `home := amazontest.IsolatedHome(t)`. Check 1 (attachment): existing assertions, then `AssertNoDownload(t, []string{"page.html"}, options.ProfileDir, home)` and `AssertDownloadsFolder(t, home, false)`. Check 2 (attachment again, same collector): `assertNavigationFailure`, `server.Count("/dp/B0PRICE001") == 2`, the same two assertions. Check 3 (normal): `success`, 12999, the same two assertions. |
| TC-002 | `internal/amazon/browser_navigation_test.go` | New subtests in the same function, one per answer: `attachment; filename="page.html"` with `text/html`; `attachment` without a file name with `text/html`; `application/octet-stream` with `attachment; filename="B0PRICE001.bin"`. Every body is `page + DownloadMarker`. Each subtest: `testBrowserOptions`, `IsolatedHome`, `os.Mkdir(home/Downloads, 0o700)`, one check under `captureOutput`, `assertNavigationFailure`, `AssertDownloadsFolder(t, home, true)`, `AssertNoDownload(t, []string{"page.html", "B0PRICE001", "B0PRICE001.bin"}, options.ProfileDir, home)`. Base names starting with `B0PRICE001` count as the name derived from the URL, with or without an added extension. |
| TC-003 | `internal/httpapi/amazon_browser_download_test.go` (new) | `TestAmazonBrowserDownload_TC003_RefreshAnsweredWithADownloadWritesNoFile`. `openNavigationAPI(t, []string{id})`, then `home := amazontest.IsolatedHome(t)`. Profile = `filepath.Join(api.dir, "amazon-browser")`. A local helper lists the regular files of `api.dir` outside `amazon-browser` (relative paths). It is snapshotted before the first refresh and compared after each refresh: no new path except base names starting with `pricefollower.sqlite` (database, `-wal`, `-shm`, `-journal`). Steps as `TestAmazonBrowserNavigation_TC005`: `answerAttachment`, Refresh price at 14:00, Refresh all prices at 14:05 (`202`). After each: `assertRetrievalIssue` on `itemDetails` and `listItems`, `AssertNoDownload(t, []string{"page.html"}, profile, home)`, `AssertDownloadsFolder(t, home, false)`. Then `answerPage` and Refresh price at 14:10: `waitBrowserCheck` gives `success`, latest price 12999, and the same no-file assertions. |
| TC-004 | `internal/httpapi/amazon_browser_navigation_test.go` | Extends `TestAmazonBrowserNavigation_TC004_DownloadIsACountedFailure`. Subtest "download as the 5th failure": `IsolatedHome` after `openNavigationAPI`; after the 5th check, the existing assertions, then `AssertNoDownload(t, []string{"page.html"}, filepath.Join(api.dir, "amazon-browser"), home)` and `AssertDownloadsFolder(t, home, false)`. New subtest "download as the 5th failure, two owners": `openNavigationAPI(t, ids, 1, 2)`, `IsolatedHome`, 4 × `answerPage503`, then `answerAttachment` at 14:05. For owners 1 and 2, `api.database.List(ctx, owner)` (what `GET /api/v1/items` returns to each user) holds that owner's item with `AmazonStoppedAt` = 14:05 and latest price 1999. Then the same no-file assertions. The existing subtests keep their assertions. |

Fixture changes in `internal/httpapi/amazon_browser_navigation_test.go`: `answerAttachment` writes `page + amazontest.DownloadMarker`. A new constant `answerPage` (HTTP 200 `price-found.html`, no `Content-Disposition`) is appended after `answerEmpty410`, so existing values do not change. The tests of issues #186 and #187 using these answers keep their assertions: the marker is an HTML comment after the page and changes no result.

The existing tests must keep passing, in particular all of `browser_navigation_test.go`, `browser_error_page_test.go`, `browser_test.go`, `browser_profile_test.go`, `browser_retrieval_test.go` and the `internal/httpapi` browser tests. The new set-up call and the cache clear change no check result.

Risks, for the developer to check when the tests first run:

- If Chromium kept the attachment body in a place other than the HTTP cache (seen as a marker hit outside `Default/Cache` with downloads denied), report it in the PR rather than weakening `AssertNoDownload`. The functional guarantee covers the whole profile.
- If the isolated `<home>/Downloads` appeared with downloads denied, the browser created it at start-up. That contradicts FR-AMZ-HCHK-014 and must be reported the same way, not hidden in the test.

### Refactoring

None. Out of scope: other site data in the profile (cookies stay, FR-AMZ-BPROF), the failure count and stop rules, the UI, `README.md` (no new configuration or requirement).

## Interrupted check (issue #233)

Origin: review comment R2-01 on #231 (`doc/todo/issue-217-r2-01.md`). `load` sent `Network.clearBrowserCache` with the check's `ctx`. When that context ended (server stop, `amazonCheckTimeout`) while `awaitDocument` waited, `cdpBrowser.call` wrote the request and then returned `ctx.Err()` without waiting for the answer. `close` then sent `Browser.close` at once, so the browser could exit before the cache was cleared, and the log said `context canceled` although the request had been sent. When `ctx` ended during `Page.navigate`, `load` returned through `loadError` without clearing at all.

### Frontend

Not affected: no new state, field or message. An interrupted check gives the failure it gives today (`tests/item-refresh.spec.ts`, `tests/platform-tabs.spec.ts` unchanged and must keep passing). No Playwright case.

### Backend (TS-AMZ-HCHK-018, -019)

`internal/amazon/browser.go` only. `cdp.go`, the service, the gate and the store are unchanged.

- **TS-AMZ-HCHK-018**, new method used by both clear sites:

  ```go
  // cacheClearTimeout bounds the wait for the browser to confirm that its HTTP cache is cleared.
  const cacheClearTimeout = 5 * time.Second

  // clearCache removes the profile's HTTP cache, which may hold the body of an answer offered as a
  // download (issue #217). It ignores the end of ctx, so an interrupted check still clears the cache
  // before the browser closes, and waits at most cacheClearTimeout for the answer (issue #233).
  func (t *browserTab) clearCache(ctx context.Context) {
      clearContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), cacheClearTimeout)
      defer cancel()
      err := t.browser.call(clearContext, t.session, "Network.clearBrowserCache", nil, nil)
      if errors.Is(err, context.DeadlineExceeded) {
          err = fmt.Errorf("Network.clearBrowserCache: no answer within %s", cacheClearTimeout)
      }
      if err != nil {
          log.Printf("Amazon browser cache could not be cleared after a page not opened: %v", err)
      }
  }
  ```

  - `context.WithoutCancel` drops both the cancellation and the deadline of the check, so the only context error `call` can return is the 5 s bound. That error is logged as "no answer within 5s", never as `context canceled` or `context deadline exceeded`. A protocol error is logged as `Network.clearBrowserCache: <message>`. If the browser exits, `call` returns `the browser exited (…)`. The log prefix stays `Amazon browser cache could not be cleared` (the test cases' "cache log line").
  - `clearCache` returns before `Collect`'s deferred `tab.close()` runs. So `Browser.close` is sent only after the browser answered the clear, or after the 5 s bound.
  - The `errorText` branch of `load` calls `t.clearCache(ctx)` in place of the inline call. The conditions of TS-AMZ-HCHK-017 are unchanged. When the check's context ends during `awaitDocument`, `awaitDocument` returns at once (through `loadContext`), and the clear runs.
- **TS-AMZ-HCHK-019**, `load`, the `Page.navigate` error:

  ```go
  if err := t.browser.call(loadContext, t.session, "Page.navigate", map[string]any{"url": target}, &navigation); err != nil {
      if loadContext.Err() != nil {
          // The answer may be arriving and already be in the HTTP cache (issue #233).
          t.clearCache(ctx)
      }
      return loadedPage{}, loadError(ctx, err)
  }
  ```

  - `loadContext` ends when the check is interrupted or when the navigation timeout (45 s) passes. In both cases the request may still be running, so the cache is cleared. A protocol error or an exited browser (`loadContext` still live) does not clear: the browser is gone or never started the request.
  - Clearing dooms the active entries too (TS-AMZ-HCHK-017), so an answer that arrives after the clear is deleted when its cache write ends.
  - A late `Page.navigate` answer is dropped by the reader, since `call` removed its pending entry. Nothing else reads it.
- Results, messages and logs other than the cache line are unchanged. An interruption during `Page.navigate` still returns `loadError(ctx, err)`: cause `context canceled` / `context deadline exceeded`. An interruption during `awaitDocument` still returns `Amazon page could not be opened: <errorText>`. The service records them, counts them and stops items as today (FR-AMZ-STOP).
- Bound: after the interruption, `load` returns at once. The clear takes at most 5 s, then `close` takes at most 2 s for `Browser.close` and 5 s for the exit, then kills the process group. An interrupted check therefore ends within about 12 s whatever the browser does. Before this change the bound was 7 s. `Service.Close` waits for its workers, so a server stop waits at most 5 s longer than before.
- An uninterrupted check also uses the 5 s bound. Before, the clear used the remaining 90 s of the check. A clear slower than 5 s is now logged and the check goes on to the same failure.
- Validation, persistence, migrations: none. Concurrency: the clear runs on the check's goroutine with `BrowserCollector.mu` held, as before. No goroutine is added.

### API

Not affected: no endpoint, field, error code or result changes. `API_SPECIFICATION.md` is not changed.

### Files to change

| File | Change |
| --- | --- |
| `internal/amazon/browser.go` | TS-AMZ-HCHK-018 (`cacheClearTimeout`, `clearCache`, `errorText` branch), TS-AMZ-HCHK-019 (`Page.navigate` error) |
| `internal/amazon/fake_browser_test.go` (new) | `TestMain` and the fake browser, with its helpers |
| `internal/amazon/browser_interrupted_test.go` (new) | TC-001, TC-002, TC-003, TC-004, TC-006 |
| `internal/httpapi/amazon_browser_navigation_test.go` | New fixture answer `answerAttachmentHeld` and the `flushed` signal |
| `internal/httpapi/amazon_browser_interrupted_test.go` (new) | TC-005 |

### Test files

Test names: `TestBrowserInterrupted_TCnnn_…` in `internal/amazon` and `TestAmazonBrowserInterrupted_TC005_…` in `internal/httpapi`, because the earlier issues already use these TC numbers. Each test file carries a `// TC-nnn` comment per test. No test uses `t.Parallel`, so `t.Setenv` is allowed.

**Fake browser** (`internal/amazon/fake_browser_test.go`). It is the Go test binary run again as a helper process:

- `TestMain(m)`: when `PRICEFOLLOWER_FAKE_BROWSER` is set, run the fake and `os.Exit` before `m.Run`. This happens before flag parsing, so Chromium's flags are ignored. Otherwise `os.Exit(m.Run())`. The package has no `TestMain` yet.
- Options: `testBrowserOptions`-like, with `ExecPath` = `os.Executable()`, a profile under `t.TempDir()`, `Pause` 0, and `Origin` = `http://127.0.0.1:9`. The fake does not use the profile. The collector does not set `cmd.Env`, so a script set with `t.Setenv` before `Collect` reaches the fake, and each `Collect` starts a new fake with the current script.
- Script, JSON in the variable (field names are the developer's choice):
  - `record`: a file under `t.TempDir()`.
  - `setUpUntil`: the time at which the last set-up call is answered.
  - `navigate`: one of `error` (answered at once with `frameId` and `errorText: "net::ERR_ABORTED"`), `hold` (answered with that `errorText` only when `Browser.close` arrives) or `late` (answered with that `errorText` at a given time after the interruption).
  - `response`: the status of a `Network.responseReceived` event (`type: "Document"`, the navigated `frameId`, the target URL, `text/html`) sent after the `Page.navigate` answer. 0 means no event.
  - `clear`: answer after a delay, never answer, or answer with a protocol error message.
- Protocol: messages are NUL-separated JSON on fd 3 (read) and fd 4 (write), from `os.NewFile`. Writes are serialized by a mutex, and delayed answers run in goroutines so the fake keeps reading. The fake answers:
  - `Browser.getVersion`: `product: "HeadlessChrome/120.0.0.0"` and a matching `userAgent`.
  - `Browser.setDownloadBehavior`, `Network.enable`, `Page.enable`: `{}`.
  - `Target.createTarget`: a target ID. `Target.attachToTarget`: a session ID.
  - `Emulation.setUserAgentOverride`: `{}`, at `setUpUntil`.
  - `Runtime.evaluate`: `{"result":{"value":""}}`.
  - `Page.navigate` and `Network.clearBrowserCache`: as scripted.
  - `Browser.close`: first sends a held `Page.navigate` answer, then answers, then exits 0. EOF on fd 3 also makes it exit.
- Record: one JSON line per received call and per answer sent, with the method, the kind (`received`/`answered`) and the time in Unix nanoseconds. It is written synchronously. A helper reads it after `Collect` returns, since `close` waits for the exit.
- Timing: the test sets `interruptAt := time.Now().Add(2 * time.Second)` and `setUpUntil = interruptAt − 200 ms`. So `Page.navigate` is received, and with `error` answered, about 200 ms before the interruption. A helper `interruptedContext(kind, interruptAt)` returns:
  - time limit: `context.WithDeadline(context.Background(), interruptAt)`;
  - server stop: `context.WithCancel` with `time.AfterFunc(time.Until(interruptAt), cancel)`.

  The test fails with a clear message if the record shows `Page.navigate` received after `interruptAt`, because the interruption point was then missed. It asserts that `Collect` returned before `interruptAt + 20 s`.

| Test case | File | Notes |
| --- | --- | --- |
| TC-001 | `internal/amazon/browser_interrupted_test.go` | `navigate: error`, `response: 0`, clear answered 300 ms after it is received. Subtests `time limit` and `server stop`. Each runs `Collect` twice on the same collector, with a new `interruptAt` and record file each time. Under `captureOutput`, assert from the record: `Network.clearBrowserCache` was received after `Page.navigate`, and the clear's `answered` time is before the `Browser.close` `received` time. Assert the failure (`request_error`, `amazonCheckFailed`, `check_error`) and the bound. Assert that no output line contains `Amazon browser cache could not be cleared`, and that no line containing `cache` contains `context canceled` or `context deadline exceeded`. Fails before the change: `Browser.close` arrives before the clear's answer, and the log line says `context canceled`. |
| TC-002 | `internal/amazon/browser_interrupted_test.go` | `navigate: hold` (subtests `time limit`, `server stop`) and `navigate: late` (answer 300 ms after `interruptAt`, server stop), clear answered after 300 ms. Assert that the clear was received after `interruptAt`, that `Browser.close` was received after the clear's answer, exactly one clear, the failure, the bound and no cache log line. Fails before the change: no clear is sent. |
| TC-003 | `internal/amazon/browser_interrupted_test.go` | Server stop. Subtests: `no answer` (`navigate: error`, clear never answered); `error answer` (`navigate: error`, clear answered with the error `cache clear refused by fake`); `no answer, page still requested` (`navigate: hold`, clear never answered). Assert the bound (the 5 s clear wait fits in it), that `Browser.close` was received, the failure, and exactly one cache log line. With `error answer` the line contains `cache clear refused by fake`. No cache log line contains `context canceled`. |
| TC-004 | `internal/amazon/browser_interrupted_test.go` | Real Chromium (`testBrowserOptions`, then `amazontest.IsolatedHome`). A mux whose `/` delegates to `fixtureHandler(t)` and whose `/dp/B0PRICE001` follows a mode switched with an `atomic.Int32`. Each handler records its arrival time and also returns when `r.Context()` ends. Modes: **while requested**: wait until `interruptAt + 100 ms`, then write the download response. **After the download was offered**: wait until `interruptAt − 100 ms`, write the download headers and body, flush, record the flush, then wait for `r.Context()`. **Normal**: write `price-found.html`. Here `interruptAt = time.Now().Add(5 * time.Second)`, which leaves room for Chromium to start, and the check runs with `interruptedContext`. Assert that the request arrived before `interruptAt`, and for the second mode that the flush was recorded. Subtests: both modes × both kinds, plus **after the download was offered** with `<home>/Downloads` created empty (`AssertDownloadsFolder(t, home, true)`). After each interrupted check: the failure, the bound, `AssertNoDownload(t, []string{"page.html"}, options.ProfileDir, home)`, `AssertDownloadsFolder`, and no cache log line. Then normal mode with `browserCheck` (live context) on the same collector: `success`, 12999, and the same no-file assertions. |
| TC-005 | `internal/httpapi/amazon_browser_interrupted_test.go` | See below. |
| TC-006 | `internal/amazon/browser_interrupted_test.go` | Fake browser, `browserCheck` (live context), `setUpUntil` zero. Subtest `download`: `navigate: error`, `response: 200`, clear answered at once. The clear is recorded before `Browser.close`, the result is the failure, and there is no cache log line. Subtest `error page`: `navigate: error`, `response: 503`. No `Network.clearBrowserCache` is recorded, the result is `request_error` with `HTTPStatus` 503, and the output has a line containing `Amazon returned HTTP 503`. |

TC-005, `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile`:

- Fixture: `navigationFixture` gets a `flushed chan struct{}` (buffered 1, made in `newNavigationFixture`). A new constant `answerAttachmentHeld` is appended after `answerPage`, so existing values do not change. That answer writes the `answerAttachment` headers and body, flushes, sends on `flushed` without blocking, then waits for `r.Context()`. The tests that use the other answers are unchanged.
- Set-up: `api := openNavigationAPI(t, ids, 1, 2)`, then `home := amazontest.IsolatedHome(t)`, then `before := dataFiles(t, api.dir)`.
- Interruption: `answerAttachmentHeld`. `api.items.Collect(ctx, ids[0])` runs in a goroutine with a `context.WithCancel` context. The test waits up to 60 s on `api.fixture.flushed`, calls `cancel()`, and asserts that the goroutine ends within 20 s. Under `captureCheckOutput`, assert no cache log line. Then `assertNoDownloadWritten(t, api, home)` and `assertNoNewDataFile(t, api, before, …)`.
- Protected mode: the API harness runs in open mode, so Refresh price for each owner calls the handler's service method with that owner's principal: `api.items.RefreshItem(service.WithPrincipal(ctx, service.Principal{UserID: owner, Role: "user"}), id)`, with `answerPage` and `api.clock.Set` at 14:10 and 14:11. The test waits until `api.database.Get(ctx, id)` shows the attempt at that time, not pending. Then `api.database.List(ctx, owner)`, which is what `GET /api/v1/items` returns to that user (as in TC-004 of issue #217), holds the owner's item with latest price 12999. The no-file and no-new-data-file assertions hold again.

The existing tests must keep passing, in particular `browser_navigation_test.go`, `browser_error_page_test.go`, `browser_test.go` and the `internal/httpapi` browser tests. An uninterrupted check behaves as before, apart from the 5 s bound on the clear.

Risks, for the developer to check when the tests first run:

- If TC-004 **while requested** finds the marker in `<profile>/Default/Cache`, then an answer that arrives after the clear was cached anew. Report it in the PR, as for issue #217, rather than weakening `AssertNoDownload`.
- The fake's timings use 200–300 ms margins, and the real-browser tests leave Chromium 5 s to request the page. A test that misses its interruption point must fail with a message saying so, never pass silently.

### Refactoring

| Ref | Refactoring | Reason | Scope | Risk |
| --- | --- | --- | --- | --- |
| R-5 | The inline cache clear of `load` becomes `browserTab.clearCache` | Two call sites need the same bounded clear and log line (TS-AMZ-HCHK-018–019) | `internal/amazon/browser.go` | Low: same CDP call and log prefix; covered by TC-001–003 and TC-006 |

Out of scope: `cdp.go` (`call` keeps returning `ctx.Err()` at once for every other call), the failure causes of interrupted checks, the service's time limit, the stop rules, the UI.

## Verification and open questions

- `gofmt -l .`, `go vet ./...`, `go test ./...` (browser tests need Chromium; present in the CI image under `/ms-playwright`), `npm run build`, `npx playwright test`; gosec and govulncheck.
- QA on a machine with Chromium: add an Amazon item and watch the logs for a browser check; `nextCheckAt` of Amazon items in a window; Refresh price keeps it; a missing browser gives counted failures and the start-up log line.
- Open functional questions: none. Technical interpretations recorded: overdue Amazon checks outside their window (beyond 15 min) are redrawn instead of sent (FR-HCHK-004); a scheduled check due while the item is already being checked is redrawn (no duplicate).
- Issue #186: no open question. The browser tests (TC-001–006) need Chromium; with `CI` set they cannot skip.
- Issue #187: no open question. The browser tests (TC-001–004) need Chromium; with `CI` set they cannot skip. Technical interpretation: the test cases' "used-offer path" covers both used-offer URLs, `/gp/aod/ajax/ref=tmm_pap_used_aod_0` and `/gp/aod/ajax`.
- Issue #217: no open question. The browser tests (TC-001–004) need Chromium; with `CI` set they cannot skip. Technical interpretations: "no file holding that response in the profile" includes the profile's HTTP disk cache (hence TS-AMZ-HCHK-017); a failure to set the download behaviour fails the check rather than running it with downloads allowed.
- Issue #233: no open question. TC-004 and TC-005 need Chromium; with `CI` set they cannot skip. TC-001–003 and TC-006 use the fake browser and need no Chromium. Technical interpretations:
  - The clear waits at most 5 s, so a server stop waits at most about 12 s for an Amazon check.
  - A navigation timeout during `Page.navigate` also clears the cache, since the same path is affected.
  - The HTTP harness runs in open mode, so TC-005 runs each owner's Refresh price through `RefreshItem` with that owner's principal and reads the results with `store.List(owner)`.
