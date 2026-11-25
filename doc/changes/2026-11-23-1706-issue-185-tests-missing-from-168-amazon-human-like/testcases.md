# Test cases: tests missing from #168, Amazon human-like checks (issue #185)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-022 to FR-TEST-HYG-026). No use cases: see [no-use-cases.md](no-use-cases.md). The behaviours under test are those of [Amazon human-like checks](../../specifications/amazon-human-checks/functional.md) FR-AMZ-HCHK-004, -005, -009, -010 and -012 (use cases UC-AMZ-HCHK-001, -004, -005, -006), which this change does not alter.

Conventions:

- The IDs below are this change's own; they are not the issue #168 cases of the same number.
- **Windows:** the default Amazon windows, morning 06:00–09:00 and evening 19:00–23:00, in the configured location. "Next window" means the next window that begins after the given time.
- **Service tests** run `service.New` and the scheduled-check run on a real SQLite database (temporary file, the store's normal migrations), with an injected clock (`Options.Now`), an injected random source, no pause between scheduled requests (`Options.SchedulePause`), and a recording fake Amazon collector (`Options.Amazon`) that counts calls per item. The server log is captured from the standard `log` output. No HTTP handler is involved: these cases are integration-level for the database and service only.
- **Browser tests** run the real `BrowserCollector` with headless Chromium against a local `amazontest` server, as the existing browser tests do (`testBrowserOptions`, `browserCheck`, `captureOutput`). Like those tests they skip when no Chromium is found; the gate environment has one.
- A **mutation check** is a manual, uncommitted change of production code made once to confirm that the new test fails; it is reverted before the change is committed.
- The change adds Go tests only: no API, database schema, user interface or Playwright test changes, so narrow screens, keyboard and screen states do not apply.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-TEST-HYG-022 | TC-001 |
| FR-TEST-HYG-023 | TC-002 |
| FR-TEST-HYG-024 | TC-003 |
| FR-TEST-HYG-025 | TC-004 |
| FR-TEST-HYG-026 | TC-001, TC-002, TC-003, TC-005 |

## Cases

### TC-001 — A scheduled Amazon check whose morning window has passed is skipped, redrawn and logged

- Covers: FR-TEST-HYG-022, FR-TEST-HYG-026 (behaviour: FR-AMZ-HCHK-004, FR-AMZ-HCHK-009; UC-AMZ-HCHK-001)
- Level: integration (Go, `internal/service`, real SQLite database, fake Amazon collector, injected clock)
- Given a tracked, not stopped Amazon item whose next scheduled check is stored at 07:30 on day D (morning window), and a service whose clock reads 20:00 on day D (evening window), the application having not run in between
- When the scheduled checks run once at 20:00
- Then:
  - the fake Amazon collector received no call for the item;
  - the item's stored next scheduled check is within the morning window of day D+1 (06:00–09:00), the next window that begins after 20:00, and is no longer 07:30 on day D;
  - the captured log has the line `scheduled Amazon check of item <id> missed its window; next check <time>`, where `<id>` is the item's ID and `<time>` is the stored next check (RFC 3339).
- Corner cases:
  - control item: a second tracked Amazon item due at 19:30 on day D (inside the current evening window) is checked in the same run, so the test does not pass merely because the run checked nothing; no "missed its window" line is logged for it;
  - repeated action: a second run at the same clock time sends no request for the missed item, leaves its next check unchanged and logs no second "missed its window" line for it;
  - the log line names the item's new time, not its old due time;
  - mutation check: making the scheduled run check an Amazon item whose window has passed (for example making `skipScheduledCheck` return `false` for it) makes this test fail.

### TC-002 — The upgrade redraw mark gives every Amazon item its own window time at start, then is cleared

- Covers: FR-TEST-HYG-023, FR-TEST-HYG-026 (behaviour: FR-AMZ-HCHK-004, FR-AMZ-HCHK-005, FR-AMZ-HCHK-010; UC-AMZ-HCHK-004)
- Level: integration (Go, `internal/service`, real SQLite database, injected clock)
- Given a database with `amazon_request_state.human_checks_redraw = 1` and three tracked Amazon items that all hold the same next scheduled check, 08:00 on day D+1 (the former configured time), and a start time of 10:00 on day D, so the next window is the evening window of day D (19:00–23:00) and the shared time is outside it
- When `service.New` is called on this database
- Then:
  - each of the three Amazon items has a stored next scheduled check within 19:00–23:00 on day D; none keeps 08:00 on day D+1;
  - the three items' stored next scheduled checks are pairwise different, as each item's time is drawn on its own;
  - `amazon_request_state.human_checks_redraw` reads 0.
- And when the service is closed and `service.New` is called again on the same database (mark now 0), with the clock at 10:05 on day D
- Then the three items' next scheduled checks are exactly those stored by the first start.
- Corner cases:
  - several items, not one, so a redraw of only the first item fails the test;
  - the shared time is chosen outside the next window, so an item that kept it is detected (a shared time inside the window could pass by chance);
  - repeated start: the second start, mark cleared, redraws nothing;
  - one time drawn and given to every item (for example the draw moved out of the per-item loop) makes this test fail, although that time is within the window;
  - mutation check: making the start ignore the redraw mark (scheduling only items without a next check) makes this test fail.

### TC-003 — An empty-body error answer is classified by its HTTP status

- Covers: FR-TEST-HYG-024, FR-TEST-HYG-026 (behaviour: FR-AMZ-HCHK-012; UC-AMZ-HCHK-005, UC-AMZ-HCHK-006)
- Level: unit (Go, `internal/amazon`, real headless Chromium against a local `amazontest` server)
- Given, for each status of 404, 410, 503 and 429 (one subtest each, a new collector and server each), a fixture item whose product page `/dp/<ASIN>` is answered with that status, `Content-Type: text/html`, `Cache-Control: no-store` and an empty body
- When the item is checked through the browser collector, with the log captured
- Then:
  - 404 and 410: `Result` is `unavailable` with the unavailable-listing message (not `request_error`), and `HTTPStatus` equals the status;
  - 503 and 429: `Result` is `request_error` with the Amazon-check-failed message, `SecondHandStatus` is `check_error`, `HTTPStatus` equals the status, `FailureCause` is exactly `Amazon returned HTTP 503` or `Amazon returned HTTP 429`, and the captured log contains that cause.
- Corner cases:
  - the product page is really requested (the server counts at least one request on `/dp/<ASIN>`), so a check that fails before loading does not pass;
  - the body is empty, so the browser shows its own error page: the result comes from the status, not from a navigation error or an "unexpected content type" cause;
  - 429 is covered as well as 503: both are non-2xx statuses other than 404 and 410;
  - each status gives its own HTTP status, not another one (for example 410 recorded as 404);
  - 404 and 410 may be asserted by extending the existing `TestBrowserErrorPage_TC001_EmptyUnavailablePageSendsNoUsedOfferRequest`, which already checks result and status for them; 503 and 429 need the new assertions on the failure cause.

### TC-004 — `internal/amazon` is formatted with gofmt

- Covers: FR-TEST-HYG-025
- Level: repository check (`gofmt -l internal/amazon`)
- Given the change applied on top of `master`
- When `gofmt -l internal/amazon` runs
- Then it prints nothing
- Corner cases:
  - `internal/amazon/browser_test.go` is included (the `browserCheck(t,collector, …)` call of the issue is written `browserCheck(t, collector, …)`);
  - every test file added or changed by this change under `internal/amazon` is included, as is `internal/amazon/amazontest`;
  - running `gofmt -w` on the package afterwards changes no file.

### TC-005 — The new tests run in the existing gate and only test files change

- Covers: FR-TEST-HYG-026
- Level: repository check and suite run (deterministic gates: `go vet ./...`, `go test ./...`, and the change's diff)
- Given the change applied on top of `master`
- When `go vet ./...` and `go test ./...` run, `go test -v -count=1 -run '<TC-001, TC-002 and TC-003 test names>' ./internal/service/ ./internal/amazon/` lists the new tests, and `git diff --name-only master -- . ':!doc'` is listed
- Then `go vet ./...` succeeds; `go test ./...` passes and runs the tests of TC-001, TC-002 and TC-003 (reported `PASS`, not `SKIP`, in the gate environment) with no other command or flag; the diff outside `doc/` lists only `*_test.go` files
- Corner cases:
  - no production Go code, frontend source, `go.mod`, `go.sum`, `package.json`, `API_SPECIFICATION.md` or database migration changes, so the operator observes no change in the application;
  - the tests give the same result without cached results (`-count=1`) and when run twice;
  - the other gates (`node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`) pass unchanged.
