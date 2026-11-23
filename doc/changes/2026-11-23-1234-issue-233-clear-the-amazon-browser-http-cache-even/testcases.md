# Test cases: interrupted Amazon check leaves no downloaded answer (issue #233)

Sources: [functional specification](../../specifications/amazon-human-checks/functional.md) (FR-AMZ-HCHK-015, with FR-AMZ-HCHK-014 for the download guarantee and FR-AMZ-HCHK-011 for the check result), use case [UC-AMZ-HCHK-008](../../use-cases/amazon-human-checks/UC-AMZ-HCHK-008-interrupted-amazon-check-leaves-no-downloaded-answer.md).

Conventions:

- **Interruption:** the check's context ends. A **time limit** is a context deadline (`context.WithTimeout`, the way `amazonCheckTimeout` bounds a check); a **server stop** is a cancellation (`cancel()` of the context, the way the service's context ends on shutdown). Cases that list both run once per kind.
- **Fake browser:** a test executable started by the collector in place of Chromium (`BrowserOptions.ExecPath`), speaking the DevTools protocol on the remote-debugging pipe. It answers the set-up calls like Chromium, follows a per-test script for `Page.navigate` and `Network.clearBrowserCache`, answers `Browser.close` and exits, and records each received method in order, with the time each answer was sent. It makes the order "clear answered, then `Browser.close`" observable, which a real browser does not. The technical specification chooses how it is built (for example the Go test binary re-executed as a helper process).
- **Browser tests** run the real Chromium (`amazontest.Browser`) against a local `amazontest.Server`, with a persistent profile under `t.TempDir()` and an isolated home (`amazontest.IsolatedHome`), as for issue #217. No test contacts amazon.*.
- **Download response:** HTTP 200, `Content-Type: text/html; charset=utf-8`, `Content-Disposition: attachment; filename="page.html"`, body = `price-found.html` + `amazontest.DownloadMarker`.
- **No file written:** after the check has returned (the browser is closed), `amazontest.AssertNoDownload(t, []string{"page.html"}, profile, home)` passes (no file named like the download, no partial download, no regular file holding the marker, the HTTP disk cache `<profile>/Default/Cache` included), and `amazontest.AssertDownloadsFolder(t, home, existed)` passes.
- **Failure** result: `Result: "request_error"`, `Message` = the Amazon check failed message, `SecondHandStatus: "check_error"`; a `request_error` is counted by the service as today (FR-AMZ-STOP tests, unchanged).
- **Cache log line:** the line starting `Amazon browser cache could not be cleared`. "No cache log line" means the captured output (`captureOutput`) has no such line.
- **Bounded:** `Collect` (or the service's `Collect`) returns within 20 s of the interruption, whatever the browser does; the test fails instead of hanging.
- The UI, the API and the stored data do not change: no Playwright case. The user-visible state of a failed check is already covered by `tests/item-refresh.spec.ts` and `tests/platform-tabs.spec.ts`, which must keep passing. Narrow screens and keyboard are not relevant.
- The tests of issues #186, #187 and #217 keep their assertions (`browser_navigation_test.go`, `browser_error_page_test.go`, `internal/httpapi/amazon_browser_navigation_test.go`, `internal/httpapi/amazon_browser_download_test.go`): an uninterrupted check is unchanged.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-AMZ-HCHK-015 | TC-001, TC-002, TC-003, TC-004, TC-005, TC-006 |

| Use case | Test cases |
| --- | --- |
| UC-AMZ-HCHK-008 | TC-001 (main flow 3–7), TC-002 (alternative "interrupted while the page is still being requested"), TC-003 (main flow 6, alternative "removal really fails"), TC-004 (main flow 2–7, alternatives "interrupted while the page is still being requested", "next check answered with the normal page"), TC-005 (alternatives "server stopped during the check", "next check answered with the normal page", "protected mode"), TC-006 (postcondition "nothing else changes") |

UC-AMZ-HCHK-008 alternative "fifth consecutive failure": the interrupted check returns the failure result (TC-001, TC-002, TC-004), which the existing stop tests already count and stop on (UC-AMZ-HCHK-005). The 90 s `amazonCheckTimeout` is a constant of the service, so no case waits for it through the service.

## Collector with a fake browser

### TC-001 — Interrupted while waiting after the download was offered: the cache is cleared and confirmed before the browser closes

- Covers: FR-AMZ-HCHK-015 (with FR-AMZ-HCHK-014); UC-AMZ-HCHK-008 (main flow steps 3–7)
- Level: unit (`internal/amazon`, fake browser)
- Given a fake browser that answers `Page.navigate` at once with a frame ID and `errorText: "net::ERR_ABORTED"`, sends no `Network.responseReceived` (so the collector waits for the response event), and answers `Network.clearBrowserCache` successfully 300 ms after receiving it
- When `Collect` runs with a context that ends 200 ms after the fake answered `Page.navigate` (time limit; then server stop)
- Then:
  - the fake received `Network.clearBrowserCache` after `Page.navigate`, and `Browser.close` only after it had sent the answer to `Network.clearBrowserCache` (the clear is not abandoned because the check's context ended);
  - the result is a failure; the check is bounded;
  - no cache log line, and no line reports `context canceled` or `context deadline exceeded` as the cause of a cache clear
- Corner cases: both interruption kinds; a clear slower than the interruption; repeated action: a second `Collect` with the same fake and the same interruption gives the same order and result (each check clears again).

### TC-002 — Interrupted while the page is still being requested: the cache is cleared before the browser closes

- Covers: FR-AMZ-HCHK-015 (with FR-AMZ-HCHK-014); UC-AMZ-HCHK-008 (alternative "interrupted while the page is still being requested")
- Level: unit (`internal/amazon`, fake browser)
- Given a fake browser that does not answer `Page.navigate` until it receives `Browser.close` (or answers it only after the interruption, with `errorText`), and answers `Network.clearBrowserCache` successfully 300 ms after receiving it
- When `Collect` runs with a context that ends 200 ms after the fake received `Page.navigate` (time limit; then server stop)
- Then the fake received `Network.clearBrowserCache` after the interruption and `Browser.close` only after it had answered the clear; the result is a failure; the check is bounded; no cache log line
- Corner cases: both interruption kinds; a late `Page.navigate` answer arriving after the interruption changes nothing (no second clear required, no panic, same result).

### TC-003 — A clear that never answers or fails does not hold the check, and is logged with its real cause

- Covers: FR-AMZ-HCHK-015; UC-AMZ-HCHK-008 (main flow step 6, alternative "removal really fails", alternative "server stopped during the check" for the bound)
- Level: unit (`internal/amazon`, fake browser)
- Given the interruption of TC-001 (cancellation, as a server stop), per subtest:
  - **no answer:** the fake never answers `Network.clearBrowserCache`;
  - **error answer:** the fake answers `Network.clearBrowserCache` with the protocol error message `cache clear refused by fake`;
  - **no answer, page still requested:** as the first subtest, with the interruption of TC-002
- When `Collect` runs
- Then:
  - `Collect` returns within the bound and the fake still received `Browser.close`;
  - the result is the same failure as in TC-001;
  - exactly one cache log line; with **error answer** it contains `cache clear refused by fake`; in every subtest it does not contain `context canceled` (the interruption of the check is never given as the cause)
- Corner cases: unresponsive browser; browser error; both interruption points.

## Real browser

### TC-004 — A real check interrupted around a download leaves no file in the profile or the Downloads folder

- Covers: FR-AMZ-HCHK-015 (with FR-AMZ-HCHK-014, FR-AMZ-HCHK-011); UC-AMZ-HCHK-008 (main flow steps 2–7, alternatives "interrupted while the page is still being requested", "next check answered with the normal page")
- Level: integration (browser test, `internal/amazon`, real Chromium, persistent profile, isolated home)
- Given a fixture server for `/dp/B0PRICE001` that reports each request's arrival and, per subtest:
  - **while requested:** holds the request until the test releases it, then writes the download response;
  - **after the download was offered:** writes the download response headers and body at once and flushes, then keeps the connection open until the request ends
- When a check runs with its context ended (time limit; then server stop) at that point: **while requested** — after the request arrived, the context ends, then the request is released; **after the download was offered** — 100 ms after the body was flushed; then the server answers the normal page (no `Content-Disposition`) and the item is checked again with a live context
- Then:
  - the interrupted check is a failure and is bounded;
  - no file written (Downloads folder absent before and after); in a second run of the **after the download was offered** subtest with `<home>/Downloads` created empty beforehand, it stays empty;
  - no cache log line;
  - the next check is `success` with 12999 cents, and still no file written
- Corner cases: both interruption points and both kinds; Downloads folder absent and present; the persistent profile is reused by the next check, so a cached body left by the interrupted check would be found; the exact order of Chromium's events is not controlled, so the assertions must hold whichever order the run takes.

## Through the application

### TC-005 — A server stop during Refresh price answered with a download writes nothing in the data directory

- Covers: FR-AMZ-HCHK-015 (with FR-AMZ-HCHK-014); UC-AMZ-HCHK-008 (alternatives "server stopped during the check", "next check answered with the normal page", "protected mode")
- Level: integration (`internal/httpapi`, real SQLite, HTTP handlers, real browser collector with its persistent profile in `<data>/amazon-browser`, isolated home; `openNavigationAPI`)
- Given two tracked Amazon items owned by two users (protected mode) with a recorded last price of 1999 cents, and the fixture server answering `/dp/<ASIN>` with the download response written and flushed while the request is held open, reporting each request's arrival
- When the service's `Collect` of the first item runs with a context that the test cancels once the product page request has arrived and the body was flushed (server stop); then the fixture answers the normal page (`answerPage`) and `POST /api/v1/items/{id}/refresh` is called for each item (owner's session)
- Then:
  - the stopped `Collect` returns within the bound;
  - no file written under `<data>/amazon-browser` or the isolated home, the Downloads folder is not created, and the data directory holds no new file outside `amazon-browser` other than the SQLite files (`dataFiles` snapshot, as TC-003 of issue #217);
  - no cache log line;
  - after the refreshes, `GET /api/v1/items/{id}` shows each item with its latest price 12999 cents, and still no file written
- Corner cases: server stop rather than time limit; items of several users; the next checks behave as if no download had been offered.

### TC-006 — An uninterrupted check is unchanged

- Covers: FR-AMZ-HCHK-015 (postcondition: results and stop behaviour as today); UC-AMZ-HCHK-008 (postcondition "nothing else changes")
- Level: unit (`internal/amazon`, fake browser)
- Given a live context (not interrupted), per subtest:
  - **download:** the fake answers `Page.navigate` with `errorText` and a 2xx `Document` response event;
  - **error page:** the fake answers `Page.navigate` with `errorText` and a 503 `Document` response event, and `Runtime.evaluate` with an empty error page HTML
- When `Collect` runs (reading pause 0)
- Then **download**: `Network.clearBrowserCache` is sent before `Browser.close`, the result is the failure, no cache log line; **error page**: no `Network.clearBrowserCache` is sent and the result is the counted failure logged with HTTP 503, as today (FR-AMZ-HCHK-012)
- Corner cases: the non-2xx error page is not affected by the change; the existing real-browser tests of issues #186, #187 and #217 must keep passing.
