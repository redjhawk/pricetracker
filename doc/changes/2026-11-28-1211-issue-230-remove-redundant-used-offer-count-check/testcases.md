# Test cases: one used-offer count check per step of the TC-004 refresh test (issue #230)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-140 to FR-TEST-HYG-143). There are no use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **Test file:** `internal/httpapi/amazon_browser_error_page_test.go`.
- **TC-004 refresh test:** the subtest "Refresh price, Refresh all prices, Refresh price again" of `TestAmazonBrowserErrorPage_TC004_RefreshSendsNoUsedOfferRequest`.
- **Steps:** the first Refresh price (14:00), Refresh all prices (14:05, both items), the repeated Refresh price 1 (14:10) and the repeated Refresh price 2 (14:15).
- **Step's count check:** an `if got := api.usedOfferCount(); got != 0 { t.Fatalf(...) }` right after a step's `assertUnavailableOfferKept` calls.
- **Injected request:** a temporary local change, never committed, that sends one `GET` to `api.fixture.URL + "/gp/aod/ajax"` (a used-offer path counted by `usedOfferCount`) at a given place of the TC-004 refresh test. The fixture server records it before answering, so the used-offer count becomes 1. No production path sends a used-offer request after an empty-body 404, so the checks are proven this way (TC-003 to TC-005).
- The change only edits a Go test file. No production code, API, database or user interface changes and there is no user flow, so no e2e case is needed; the Go test runs with real Chromium, the HTTP handlers, a real SQLite database and the `amazontest` fixture server (TC-003 to TC-006) are the integration-level proof. The IDs below are this change's own, not those in the test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-140 | TC-001, TC-003, TC-004, TC-005 | — |
| FR-TEST-HYG-141 | TC-001, TC-003, TC-004, TC-005 | — |
| FR-TEST-HYG-142 | TC-002 | — |
| FR-TEST-HYG-143 | TC-006, TC-007 | — |

## Cases

### TC-001: Each of the four steps is followed by its own count check

- Covers: FR-TEST-HYG-140, FR-TEST-HYG-141
- Level: code check of the test file (`grep`, reading)
- Given the change applied
- When the TC-004 refresh test is read and `grep -n 'usedOfferCount()' internal/httpapi/amazon_browser_error_page_test.go` is run
- Then the TC-004 refresh test holds exactly three `usedOfferCount()` calls: one right after the first Refresh price's `assertUnavailableOfferKept` and before `api.clock.Set(humanTime(14, 5))`; one right after the loop that waits for both items of Refresh all prices and before the repeated Refresh price loop; one inside the repeated Refresh price loop, after its `assertUnavailableOfferKept`
- And each is followed by a `t.Fatalf` whose format starts with `%d`, filled with `got`, names its step and ends with `%+v`, filled with `api.fixture.Requests()`
- And the loop's message is still `%d used-offer requests after repeated Refresh price %d: %+v`, filled with `got`, `attempt+1` and `api.fixture.Requests()`
- And the first Refresh price message names the first Refresh price and the Refresh all prices message names Refresh all prices; the three formats differ from one another and from `used-offer requests in total`
- Corner cases:
  - a Refresh all prices check placed inside the loop over `ids` (after only the first item) fails the check: it must come after the checks of both items;
  - a check placed before `waitBrowserCheck` or `assertUnavailableOfferKept` of its step fails the check;
  - `t.Errorf` instead of `t.Fatalf` fails the check (the test must stop at once);
  - a first Refresh price message that could be read as a repeated Refresh price message (for example `after Refresh price`) fails the check.

### TC-002: No count check follows the repeated Refresh price loop

- Covers: FR-TEST-HYG-142
- Level: code check of the test file (`grep`, reading)
- Given the change applied
- When `grep -c 'used-offer requests in total' internal/httpapi/amazon_browser_error_page_test.go` is run and the TC-004 refresh test is read
- Then the grep gives 0
- And the statement right after the closing brace of the repeated Refresh price loop is the log check `if lines := linesWith(output(), usedOfferFailedLine); len(lines) != 0`, unchanged with its message `used-offer failure logged: %v`
- Corner cases:
  - no `usedOfferCount()` call, under any message, sits between the loop and the log check;
  - the log check is not removed with the redundant check.

### TC-003: A used-offer request during the first Refresh price fails the first step's check

- Covers: FR-TEST-HYG-140, FR-TEST-HYG-141
- Level: integration (Go test with real Chromium, the HTTP handlers, a real SQLite database and the fixture server; temporary local change, never committed)
- Given the change applied, Chromium available, and the injected request placed right after `refresh(t, api.server, ids[0])` of the first Refresh price
- When `go test -count=1 -run '^TestAmazonBrowserErrorPage_TC004_RefreshSendsNoUsedOfferRequest$' ./internal/httpapi/` runs
- Then the TC-004 refresh test fails once, with the first Refresh price message: it holds `1`, names the first Refresh price and ends with the recorded requests, which include the `/gp/aod/ajax` request
- And the output holds neither the Refresh all prices message nor `after repeated Refresh price`, and no Refresh all prices request is made (the test stopped at once)
- Corner cases:
  - the subtest "Refresh price of a stopped item" still passes;
  - after the temporary change is reverted, `git diff` shows only the change of TC-007.

### TC-004: A used-offer request during Refresh all prices fails that step's check and no earlier one

- Covers: FR-TEST-HYG-140, FR-TEST-HYG-141
- Level: integration (Go test with real Chromium, the HTTP handlers, a real SQLite database and the fixture server; temporary local change, never committed)
- Given the change applied and Chromium available
- When the injected request is placed right after the `POST /api/v1/items/refresh` call and the test command of TC-003 runs; then the same with the injected request placed instead between the `assertUnavailableOfferKept` of the first item and that of the second item of Refresh all prices
- Then each run fails once, with the Refresh all prices message: it holds `1`, names Refresh all prices and ends with the recorded requests
- And neither run's output holds the first Refresh price message or `after repeated Refresh price`
- Corner cases:
  - a request sent between the two items' checks is still caught by the Refresh all prices check, which proves the check comes after both items;
  - the first Refresh price check passed, so a request after it does not fail an earlier step.

### TC-005: A used-offer request during a repeated Refresh price fails that attempt's check

- Covers: FR-TEST-HYG-140, FR-TEST-HYG-141
- Level: integration (Go test with real Chromium, the HTTP handlers, a real SQLite database and the fixture server; temporary local change, never committed)
- Given the change applied and Chromium available
- When the injected request is placed right after `refresh(t, api.server, ids[0])` inside the repeated Refresh price loop, only when `attempt == 0`, and the test command of TC-003 runs; then the same only when `attempt == 1`
- Then the first run fails once with `1 used-offer requests after repeated Refresh price 1: [` followed by the recorded requests, and no request is made at 14:15
- And the second run fails once with `1 used-offer requests after repeated Refresh price 2: [`
- And neither run's output holds the first Refresh price message or the Refresh all prices message
- Corner cases:
  - together with TC-003 and TC-004, the four failure messages seen are pairwise different;
  - no run's output holds `used-offer requests in total` or `used-offer failure logged`.

### TC-006: The test passes as before on a normal run

- Covers: FR-TEST-HYG-143
- Level: integration (Go tests with real Chromium, the HTTP handlers, a real SQLite database and the fixture server)
- Given the change applied, without any temporary change, and Chromium available
- When `go test -count=1 -run '^TestAmazonBrowserErrorPage_' -v ./internal/httpapi/` runs on the branch and on `master`, and the branch run is repeated
- Then every test and subtest passes on the branch
- And the `=== RUN` lines (test and subtest names) are the same on the branch as on `master`, including `TestAmazonBrowserErrorPage_TC004_RefreshSendsNoUsedOfferRequest` and both its subtests
- Corner cases:
  - the repeated run passes too (no added flakiness);
  - no test is skipped on the branch that runs on `master`.

### TC-007: Only the test file changes outside `doc/`, the diff is limited to the checks, and the Go gates pass

- Covers: FR-TEST-HYG-143
- Level: repository check and gates (`git diff`, `gofmt`, `go vet`, `go test`)
- Given the change applied on top of `master`
- When `git diff --name-status origin/master -- . ':!doc'`, `git diff --unified=0 origin/master -- internal/httpapi/amazon_browser_error_page_test.go`, `gofmt -l internal/httpapi`, `go vet ./...` and `go test ./...` run
- Then the only entry is `M internal/httpapi/amazon_browser_error_page_test.go`
- And the file's diff only adds the two count checks of TC-001 (first Refresh price and Refresh all prices) and removes the three lines of the check after the loop; the test's name, its doc comment, its subtests, its steps and their other assertions stay as on `master`, in the same order
- And `gofmt` prints nothing, and `go vet` and `go test` succeed
- Corner cases:
  - no whitespace-only change elsewhere in the file; `usedOfferCount`, `assertUnavailableOfferKept` and the other helpers, TC-003 of the file and the subtest "Refresh price of a stopped item" are unchanged;
  - no production Go file, `web/`, `src/`, `tests/`, `go.mod`, `go.sum` or `API_SPECIFICATION.md` changes, so the operator sees no change in the application;
  - `go vet` reports no printf finding for the new `t.Fatalf` calls (as many arguments as verbs).
