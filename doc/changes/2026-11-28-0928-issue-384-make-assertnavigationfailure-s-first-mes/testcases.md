# Test cases: accurate first message of `assertNavigationFailure` (issue #384)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-112, updated by this change, and FR-TEST-HYG-132 to FR-TEST-HYG-135). There are no use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **Test file:** `internal/amazon/browser_navigation_test.go`.
- **Helper:** `assertNavigationFailure` in the test file.
- **Result check:** the helper's condition that tests only `got.Result` against `request_error` and fails with `gave %q instead of a failure: %+v`.
- **Failure-detail check:** the helper's condition, right after the result check, that tests `got.Message` against `amazonCheckFailed` (`Amazon could not be reached for a price check.`) and `got.SecondHandStatus` against `check_error`, and fails with the message of FR-TEST-HYG-133.
- **Callers:** the helper calls in `internal/amazon/browser_navigation_test.go`, `internal/amazon/browser_cache_log_test.go` and `internal/amazon/browser_interrupted_test.go`.
- No production path gives `request_error` with another message or second-hand status, so the failure-detail check is proven by temporary local changes of its expected values, never committed (TC-004, TC-005).
- The change only edits a Go test file. No production code, API, database or user interface changes and there is no user flow, so no e2e case is needed; the Go test runs with real Chromium and the `amazontest` HTTP server (TC-003 to TC-006) are the integration-level proof. The IDs below are this change's own, not those in the test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-112 | TC-001, TC-003, TC-004 | — |
| FR-TEST-HYG-132 | TC-001, TC-003 | — |
| FR-TEST-HYG-133 | TC-001, TC-004, TC-005 | — |
| FR-TEST-HYG-134 | TC-001, TC-002, TC-006 | — |
| FR-TEST-HYG-135 | TC-002, TC-007 | — |

## Cases

### TC-001: The first check is split into the result check and the failure-detail check

- Covers: FR-TEST-HYG-112, FR-TEST-HYG-132, FR-TEST-HYG-133, FR-TEST-HYG-134
- Level: code check of the test file (`grep`, reading)
- Given the change applied
- When `grep -rn 'gave %q instead of a failure: %+v' internal/amazon/` and `grep -n '"not a failure: ' internal/amazon/browser_navigation_test.go` are run, and the helper is read
- Then the first grep prints exactly one line, the `t.Fatalf` of the result check, called with `got.Result` then `got`, inside the helper
- And the second grep prints nothing
- And the result check's condition tests only `got.Result != "request_error"`; it is the first statement after `t.Helper()`
- And the failure-detail check follows it directly; its condition is `got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error"`, and its `t.Fatalf` format ends with `%+v`, filled with `got`, and writes `got.Message`, `got.SecondHandStatus` and `amazonCheckFailed` with `%q`, and the expected status as `"check_error"` (a `%q` verb or the literal text)
- And the failure-detail message does not hold `instead of a failure`
- And the URL and status check with its message `diagnostics of a page not opened: url=%q status=%d`, and the log line loop with its message `no log line with the request URL %s and the browser's error:\n%s` follow, unchanged
- Corner cases:
  - a result check that still tests `Message` or `SecondHandStatus` fails the check (the contradictory message of the issue would remain);
  - the failure-detail check placed before the result check fails the check (FR-TEST-HYG-132: a wrong result must give the result message whatever the other fields are);
  - `t.Errorf` instead of `t.Fatalf` in either check fails the check (each must stop the test, so the later checks do not run);
  - `%v` or `%s` instead of `%q`, or the expected values missing from the failure-detail message, fail the check;
  - `t.Helper()` stays the helper's first statement, so each failure points at the caller's line.

### TC-002: The diff of the test file changes only the first check

- Covers: FR-TEST-HYG-134, FR-TEST-HYG-135
- Level: code check (`git diff --unified=0 origin/master -- internal/amazon/browser_navigation_test.go`)
- Given the change applied on top of `master`
- When the diff is listed
- Then it only replaces the lines of the helper's first check (its `if` line, its `t.Fatalf` line and its closing brace) with the result check and the failure-detail check
- And it changes no other line: the helper's later checks, its doc comment (unless it is reworded to describe the two checks), the imports and every test, subtest and statement of the callers stay as on `master`
- Corner cases:
  - no whitespace-only change elsewhere in the file;
  - no caller adds or removes a check before or after its helper call.

### TC-003: A wrong result fails the result check with the result message

- Covers: FR-TEST-HYG-112, FR-TEST-HYG-132
- Level: integration (Go test with real Chromium and the `amazontest` HTTP server; temporary local change, never committed)
- Given the change applied and Chromium available
- When the server of `TestBrowserNavigation_TC002_NoContentIsAFailure` is temporarily made to answer HTTP 200 with the `price-found.html` fixture, and `go test -count=1 -run '^TestBrowserNavigation_TC002_NoContentIsAFailure$' ./internal/amazon/` runs; then the same with the `price-not-found.html` fixture
- Then the first run fails with a message holding `gave "success" instead of a failure: {` followed by the whole result
- And the second run fails with a message holding `gave "price_not_found" instead of a failure: {`
- And neither run's output holds the failure-detail message (the message of FR-TEST-HYG-133), although the message and second-hand status of a `success` or `price_not_found` result are not `amazonCheckFailed` and `check_error`
- Corner cases:
  - each failing subtest (`HTTP 204`, `HTTP 205`) fails once, through the result check, reported at the helper call line;
  - after the temporary change is reverted, `git diff` shows only the change of TC-002.

### TC-004: A wrong message fails the failure-detail check with its own message

- Covers: FR-TEST-HYG-112, FR-TEST-HYG-133
- Level: integration (Go test with real Chromium and the `amazontest` HTTP server; temporary local change, never committed)
- Given the change applied, Chromium available, and the failure-detail check temporarily made to expect the message `amazonCheckFailed + "x"`
- When `go test -count=1 -run '^TestBrowserNavigation_TC002_NoContentIsAFailure$' ./internal/amazon/` runs
- Then it fails with a message that holds `"Amazon could not be reached for a price check."` (the message given), `"Amazon could not be reached for a price check.x"` (the message expected) and `"check_error"`, and ends with the whole result (`{Result:request_error ...}`)
- And the message holds neither `instead of a failure` nor `gave "request_error"`
- And the output holds no URL and status message and no log line message (the later checks did not run)
- Corner cases:
  - the given and expected values appear quoted as `%q` writes them, so the trailing `x` is visible;
  - both subtests (`HTTP 204`, `HTTP 205`) fail with this message, each once.

### TC-005: A wrong second-hand status fails the failure-detail check with its own message

- Covers: FR-TEST-HYG-133
- Level: integration (Go test with real Chromium and the `amazontest` HTTP server; temporary local change, never committed)
- Given the change applied, Chromium available, the message expectation as committed, and the failure-detail check temporarily made to expect the second-hand status `check_errorx`
- When `go test -count=1 -run '^TestBrowserNavigation_TC002_NoContentIsAFailure$' ./internal/amazon/` runs
- Then it fails with a message that holds `"check_error"` (the status given) and `"check_errorx"` (the status expected), and the message given and expected, both `"Amazon could not be reached for a price check."`, and ends with the whole result
- And the message does not hold `instead of a failure`
- Corner cases:
  - only the second-hand status differs, so the check fails on the second operand of its condition alone;
  - after the temporary change is reverted, `git diff` shows only the change of TC-002.

### TC-006: The helper's callers pass as before

- Covers: FR-TEST-HYG-134
- Level: integration (Go tests with real Chromium and the `amazontest` HTTP server)
- Given the change applied and Chromium available
- When `go test -count=1 -run '^(TestBrowserNavigation_|TestBrowserCacheLog_|TestBrowserInterrupted_)' -v ./internal/amazon/` runs on the branch and on `master`, and the branch run is repeated
- Then every test and subtest passes, so a `request_error` result with `amazonCheckFailed` and `check_error` passes both checks and reaches the URL, status and log line checks
- And the list of tests and subtests run (`=== RUN` lines) is the same on the branch as on `master`
- Corner cases:
  - the repeated run passes too (no added flakiness);
  - no test is skipped on the branch that runs on `master`.

### TC-007: Only the test file changes outside `doc/`, and the Go gates pass

- Covers: FR-TEST-HYG-135
- Level: repository check and gates (`gofmt`, `go vet`, `go test`)
- Given the change applied on top of `master`
- When `git diff --name-status origin/master -- . ':!doc'`, `gofmt -l internal/amazon`, `go vet ./...` and `go test ./...` run
- Then the only entry is `M internal/amazon/browser_navigation_test.go`, `gofmt` prints nothing, and `go vet` and `go test` succeed
- Corner cases:
  - no production Go file, `web/`, `src/`, `tests/`, `go.mod`, `go.sum` or `API_SPECIFICATION.md` changes, so the operator sees no change in the application;
  - `go vet` reports no printf finding for either `t.Fatalf` (as many arguments as verbs).
