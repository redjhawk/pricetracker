# Test cases: TC-003 asserts the 'no answer within' cause and rejects 'context deadline exceeded' (issue #239)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-078 to FR-TEST-HYG-081). There are no use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **The test file:** `internal/amazon/browser_interrupted_test.go`.
- **TC-003 (the test):** `TestBrowserInterrupted_TC003_UnconfirmedClearIsBoundedAndLogged`, with the subtests **no answer**, **error answer** and **no answer, page still requested**. The **never-answered subtests** are **no answer** and **no answer, page still requested**.
- **The cache log line:** the output line holding `Amazon browser cache could not be cleared` (`cacheLogLine`, `cacheLines`). **A line about the cache:** any output line holding `cache` in any case, as checked by `assertNoInterruptionGivenForCache`.
- **The run:** `go test -run '^TestBrowserInterrupted_TC003_UnconfirmedClearIsBoundedAndLogged$' -v -count=1 ./internal/amazon/`.
- **The test case document:** `doc/changes/2026-11-23-1234-issue-233-clear-the-amazon-browser-http-cache-even/testcases.md`; its **TC-003 entry** is the section "TC-003 — A clear that never answers or fails does not hold the check, and is logged with its real cause".
- **Fault injection:** a temporary edit of `clearCache` in `internal/amazon/browser.go` (or of the `master` test file), made only for the run of a case, reverted afterwards and never committed.
- The change only touches test code and test documentation. No production code, API, database or user interface changes, and there is no user flow, so there are no integration or e2e cases of the application. The cases are Go test runs (some with fault injection), code and document checks, and the gates. The IDs below are this change's own, not those in the test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-078 | TC-001, TC-002, TC-003, TC-005 | — |
| FR-TEST-HYG-079 | TC-002, TC-004, TC-005 | — |
| FR-TEST-HYG-080 | TC-006 | — |
| FR-TEST-HYG-081 | TC-001, TC-005, TC-007, TC-008 | — |

## Cases

### TC-001: On a normal run, TC-003 passes with its three subtests and the never-answered lines give the real cause

- Covers: FR-TEST-HYG-078, FR-TEST-HYG-081
- Level: unit (Go, `internal/amazon`, fake browser)
- Given the change applied and `clearCache` as on `master`
- When the run is made
- Then it passes, its `-v` output lists exactly the three subtests **no answer**, **error answer** and **no answer, page still requested**, each passing; in the never-answered subtests the cache log line holds `no answer within` (the 5 s bound, `cacheClearTimeout`), and in **error answer** it holds `cache clear refused by fake`
- Corner cases:
  - the subtest names are byte-for-byte those on `master`;
  - repeated runs (`-count=3`) pass each time, so the new checks add no flakiness;
  - each subtest still takes about the time it takes on `master` (the bound is unchanged).

### TC-002: Without the deadline mapping in `clearCache`, both never-answered subtests fail

- Covers: FR-TEST-HYG-078, FR-TEST-HYG-079
- Level: unit (Go, `internal/amazon`; fault injection)
- Given the change applied and, temporarily, the `errors.Is(err, context.DeadlineExceeded)` mapping removed from `clearCache`, so a never-answered clear is logged as `...: context deadline exceeded`
- When the run is made
- Then the **no answer** and **no answer, page still requested** subtests fail, each with a message holding the cache log line (the offending line); **error answer** passes
- And, with the same temporary edit and the `master` version of the test file, the run passes, which shows the old checks missed this regression
- Corner cases:
  - the failure comes from the new checks, not from the other assertions (bound, `Browser.close`, failure result, one cache log line), which still hold under this edit;
  - either new check alone (missing `no answer within`, or presence of `context deadline exceeded`) is enough to fail the subtest.

### TC-003: A reworded cause fails the never-answered subtests

- Covers: FR-TEST-HYG-078
- Level: unit (Go, `internal/amazon`; fault injection)
- Given the change applied and, temporarily, `no answer within` in the message of `clearCache` reworded (for example to `no reply within`), the deadline mapping kept
- When the run is made
- Then both never-answered subtests fail with a message holding the cache log line; **error answer** passes
- Corner cases:
  - the reworded line holds neither `context canceled` nor `context deadline exceeded`, so this edit proves the `no answer within` check on its own;
  - **error answer** does not require `no answer within`, so it is not affected (the requirement is limited to the never-answered subtests).

### TC-004: An interruption given as the cause fails the "error answer" subtest

- Covers: FR-TEST-HYG-079
- Level: unit (Go, `internal/amazon`; fault injection)
- Given the change applied and one of these temporary edits of `clearCache` at a time, each keeping the browser's error in the line:
  1. a failure other than a deadline is logged with `: context deadline exceeded` appended;
  2. a failure other than a deadline is logged with `: context canceled` appended
- When the run is made
- Then the **error answer** subtest fails with a message holding the offending line, with either edit
- Corner cases:
  - edit 2 checks that the existing rejection of `context canceled` is kept;
  - the line still holds `cache clear refused by fake`, so the failure comes from the rejection, not from the cause check;
  - the never-answered subtests are not judged here (their failures are the deadline path, covered by TC-002).

### TC-005: The checks apply to the right subtests and the rest of TC-003 is unchanged

- Covers: FR-TEST-HYG-078, FR-TEST-HYG-079, FR-TEST-HYG-081
- Level: code check of TC-003 in the test file
- Given the diff of the test file against `master`
- When TC-003 is read
- Then:
  - the `no answer within` check runs only in the never-answered subtests (selected by the subtest's table entry, for example its `clear` value `never` or a new field, not by a text written once for every subtest) and fails with a message holding the cache log line;
  - the rejection of `context canceled` and `context deadline exceeded` runs in every subtest, on every line about the cache (for example through `assertNoInterruptionGivenForCache(t, logged)`), and fails with a message holding the offending line;
  - the subtest table keeps its three entries, names, `navigate`, `clear` and `cause` values; the checks of the interruption point, `Browser.close`, the failure result, exactly one cache log line and the `cache clear refused by fake` cause are unchanged
- Corner cases:
  - when the check is not on the cache log line found by `cacheLines`, but on any line about the cache, a second line holding the interruption is still rejected;
  - the helper `assertNoInterruptionGivenForCache`, if used, keeps its behaviour for TC-002 of issue #233, which already calls it.

### TC-006: The TC-003 entry of the issue #233 test case document states the new expected results

- Covers: FR-TEST-HYG-080
- Level: document check
- Given the change applied
- When `git diff master -- doc/changes/2026-11-23-1234-issue-233-clear-the-amazon-browser-http-cache-even/` is read
- Then only `testcases.md` changes, and only inside the TC-003 entry's "Then" results: they state that with **no answer** and **no answer, page still requested** the cache log line contains `no answer within`, and that in every subtest it contains neither `context canceled` nor `context deadline exceeded`
- Corner cases:
  - the entry's title, "Covers", "Level", "Given", "When", other results and corner cases are unchanged, as are the other entries, the coverage tables and the other files of that folder (including `tests.lock`);
  - the entry still states the `cache clear refused by fake` cause for **error answer**.

### TC-007: The other tests of the test file keep their results

- Covers: FR-TEST-HYG-081
- Level: unit (Go, `internal/amazon`)
- Given the `-v` output of `go test -run '^TestBrowserInterrupted_' -v -count=1 ./internal/amazon/` on `master` and with the change
- When both runs are compared
- Then they run the same tests and subtests with the same names and the same results (TC-001, TC-002, TC-004 and TC-006 of issue #233, plus TC-003)
- Corner cases:
  - tests skipped on `master` for lack of a real browser are skipped the same way with the change;
  - the shared helpers (`cacheLines`, `assertNoCacheLogLine`, `assertNoInterruptionGivenForCache`, the fake browser) keep their behaviour.

### TC-008: Only the test file changes outside `doc/`, and the gates pass

- Covers: FR-TEST-HYG-081
- Level: repository check and suite run (deterministic gates: `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` is listed and the gates run
- Then the only file listed outside `doc/` is `internal/amazon/browser_interrupted_test.go`, `gofmt -l internal/amazon` prints nothing, and `go vet ./...` and `go test ./...` pass
- Corner cases:
  - production Go code (including `internal/amazon/browser.go`), frontend source, `go.mod`, `go.sum`, `package.json`, `tests/` and `API_SPECIFICATION.md` are unchanged, so the operator sees no change in the application;
  - no fault injection from TC-002 to TC-004 remains;
  - the gates give the same result when run twice.
