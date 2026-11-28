# Test cases: fold the redundant Amazon-answer check into `assertNavigationFailure` (issue #219)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-111 to FR-TEST-HYG-114). There are no use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **Test file:** `internal/amazon/browser_navigation_test.go`.
- **Helper:** `assertNavigationFailure` in the test file. **First check:** its `if got.Result != "request_error" || got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error"` statement.
- **Amazon-answer checks:** the three `if` statements on `master` that compare `got.Result` with `price_not_found`, `success` and `unavailable` and fail with a message holding `gave the Amazon answer`, just before a call of the helper: in the subtest `attachment then a normal page` and in the `TC-002 existing Downloads folder, …` subtests of `TestBrowserNavigation_TC001_AttachmentIsAFailure`, and in the `HTTP 204`/`HTTP 205` subtests of `TestBrowserNavigation_TC002_NoContentIsAFailure`.
- **Other callers:** the helper calls in `internal/amazon/browser_cache_log_test.go` and `internal/amazon/browser_interrupted_test.go`.
- The change only edits a Go test file. No production code, API, database or user interface changes and there is no user flow, so no e2e case is needed; the Go test runs with real Chromium (TC-004, TC-005) are the integration-level proof. The IDs below are this change's own, not those in the test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-111 | TC-001, TC-003 | — |
| FR-TEST-HYG-112 | TC-002, TC-003, TC-004 | — |
| FR-TEST-HYG-113 | TC-003, TC-004, TC-005 | — |
| FR-TEST-HYG-114 | TC-003, TC-006 | — |

## Cases

### TC-001: The Amazon-answer checks are gone

- Covers: FR-TEST-HYG-111
- Level: code check of the test file (`grep`, reading)
- Given the change applied
- When `grep -n 'gave the Amazon answer' internal/amazon/browser_navigation_test.go` and `grep -n 'price_not_found' internal/amazon/browser_navigation_test.go` are run, and the statement before each helper call in the test file is read
- Then both greps print nothing
- And no statement directly before a helper call compares the result with `price_not_found`, `success` or `unavailable`
- Corner cases:
  - all three checks are removed, including the one in the issue #217 `TC-002 existing Downloads folder, …` subtests, not only the two the issue names;
  - a check moved elsewhere (for example after the helper call) or reworded without `gave the Amazon answer` fails the reading check;
  - the comparisons with `success` that do not precede a helper call (`next.Result != "success"` after the normal page, the `inline page` subtest) are kept.

### TC-002: The helper's first failure message names the result given

- Covers: FR-TEST-HYG-112
- Level: code check of the test file (`grep`, reading)
- Given the change applied
- When `grep -rn 'gave %q instead of a failure: %+v' internal/amazon/` and `grep -rn '"not a failure: ' internal/amazon/` are run, and the helper is read
- Then the first grep prints exactly one line, the `t.Fatalf` of the helper's first check, called with `got.Result` then `got`
- And the second grep prints nothing
- And the first check's condition, the URL and status check with its message `diagnostics of a page not opened: url=%q status=%d`, and the log line loop with its message `no log line with the request URL %s and the browser's error:\n%s` are unchanged
- Corner cases:
  - arguments in the wrong order (`got` then `got.Result`) or `%v` instead of `%q` fail the check;
  - `t.Errorf` instead of `t.Fatalf` fails the check (the helper must still stop the test);
  - `t.Helper()` stays the helper's first statement, so the failure points at the caller's line.

### TC-003: The diff of the test file holds only the removals and the new message

- Covers: FR-TEST-HYG-111, FR-TEST-HYG-112, FR-TEST-HYG-113, FR-TEST-HYG-114
- Level: code check (`git diff --unified=0 origin/master -- internal/amazon/browser_navigation_test.go`)
- Given the change applied on top of `master`
- When the diff is listed
- Then it removes, for each of the three Amazon-answer checks, its `if` line, its `t.Fatalf` line and its closing brace (9 lines)
- And it replaces the line `t.Fatalf("not a failure: %+v", got)` with `t.Fatalf("gave %q instead of a failure: %+v", got.Result, got)`
- And it adds no other line (`git diff --stat` reports 1 insertion and 10 deletions)
- And both tests keep their names, subtests and other statements in the same order
- Corner cases:
  - no blank line is left in place of a removed check;
  - no comment, import, other test or the helper's doc comment changes;
  - no whitespace-only change elsewhere in the file.

### TC-004: A wrong result fails through the helper with a message naming it

- Covers: FR-TEST-HYG-112, FR-TEST-HYG-113
- Level: integration (Go test with real Chromium and the `amazontest` HTTP server; temporary local change, never committed)
- Given the change applied and Chromium available
- When the server of `TestBrowserNavigation_TC002_NoContentIsAFailure` is temporarily made to answer HTTP 200 with the `price-found.html` fixture, and `go test -count=1 -run '^TestBrowserNavigation_TC002_NoContentIsAFailure$' ./internal/amazon/` runs; then the same with the `price-not-found.html` fixture
- Then the first run fails with a message holding `gave "success" instead of a failure: {` followed by the whole result, reported at the helper call line of the test
- And the second run fails with a message holding `gave "price_not_found" instead of a failure: {`
- Corner cases:
  - each failing subtest fails once, through the helper, not through a removed check;
  - both subtests (`HTTP 204`, `HTTP 205`) fail with the message;
  - after the temporary change is reverted, `git diff` shows only the change of TC-003.

### TC-005: The navigation tests and the helper's other callers pass as before

- Covers: FR-TEST-HYG-113
- Level: integration (Go tests with real Chromium and the `amazontest` HTTP server)
- Given the change applied and Chromium available
- When `go test -count=1 -run '^(TestBrowserNavigation_|TestBrowserCacheLog_|TestBrowserInterrupted_)' -v ./internal/amazon/` runs on the branch and on `master`, and the branch run is repeated
- Then every test and subtest passes, and the list of tests and subtests run (`=== RUN` lines) is the same on the branch as on `master`
- And `TestBrowserNavigation_TC001_AttachmentIsAFailure` still runs `attachment then a normal page`, the three `TC-002 existing Downloads folder, …` subtests and `inline page`; `TestBrowserNavigation_TC002_NoContentIsAFailure` still runs `HTTP 204` and `HTTP 205`
- Corner cases:
  - the repeated run passes too (no added flakiness);
  - no test is skipped on the branch that runs on `master`.

### TC-006: Only the test file changes outside `doc/`, and the Go gates pass

- Covers: FR-TEST-HYG-114
- Level: repository check and gates (`gofmt`, `go vet`, `go test`)
- Given the change applied on top of `master`
- When `git diff --name-status origin/master -- . ':!doc'`, `gofmt -l internal/amazon`, `go vet ./...` and `go test ./...` run
- Then the only entry is `M internal/amazon/browser_navigation_test.go`, `gofmt` prints nothing, and `go vet` and `go test` succeed
- Corner cases:
  - no production Go file, `web/`, `src/`, `tests/`, `go.mod`, `go.sum` or `API_SPECIFICATION.md` changes, so the operator sees no change in the application;
  - `go vet` reports no printf finding for the new `t.Fatalf` (two verbs, two arguments).
