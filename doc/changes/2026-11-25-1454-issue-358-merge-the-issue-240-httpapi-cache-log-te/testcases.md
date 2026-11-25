# Test cases: merge the issue #240 httpapi cache-log test into TC-005 of issue #233 (issue #358)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-103 to FR-TEST-HYG-107). There are no use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **TC-005:** `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile` in `internal/httpapi/amazon_browser_interrupted_test.go` (items `B0INTR0051` of owner 1 and `B0INTR0052` of owner 2, real Chromium collector with its persistent profile, real SQLite).
- **The duplicate:** `internal/httpapi/amazon_browser_cache_log_test.go`, with `TestAmazonBrowserCacheLog_TC004_SuccessfulClearThroughTheApplicationLogsNothing` and `assertNoCacheLine`.
- **Cache line:** a logged or printed line holding `Amazon browser cache could not be cleared` (neutral wording `Amazon browser cache could not be cleared: <cause>`, old wording `Amazon browser cache could not be cleared after a page not opened: <cause>`).
- **Refresh check:** the check added by this change after each owner's Refresh price: the output captured during that Refresh price (from before `RefreshItem` until after `waitStoredCheck`) must hold no cache line.
- **The run:** `go test -count=1 -run '^TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile$' -v ./internal/httpapi/`, with Chromium available (with `CI` set, the test cannot skip).
- **The 240 document:** the TC-004 entry of [the issue #240 test cases](../2026-11-25-1255-issue-240-neutral-wording-for-the-cache-clear-fail/testcases.md). **The 240 row:** the TC-004 row of the "Neutral cache-clear log line (issue #240)" section of `doc/specifications/amazon-human-checks/technical.md`.
- **Fault injection:** a temporary edit of production or test code, made only for the run of a case, reverted afterwards and never committed. "On `master`" means the same edit applied to the `master` tree.
- The change only touches test code and documents. No production code, API, database or user interface changes and there is no user flow, so there is no e2e case. TC-001 to TC-004 run the existing integration test (real SQLite, service, real Chromium); the others are code, document and repository checks and the gates. The IDs below are this change's own, not those in the test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-103 | TC-001, TC-002, TC-003, TC-005 | — |
| FR-TEST-HYG-104 | TC-001, TC-004, TC-005 | — |
| FR-TEST-HYG-105 | TC-006 | — |
| FR-TEST-HYG-106 | TC-007 | — |
| FR-TEST-HYG-107 | TC-001, TC-008 | — |

## Cases

### TC-001: On a normal run TC-005 passes with the Refresh checks

- Covers: FR-TEST-HYG-103, FR-TEST-HYG-104, FR-TEST-HYG-107
- Level: integration (Go, `internal/httpapi`; real SQLite, service, real Chromium collector)
- Given the change applied, no fault injection, Chromium available
- When the run executes, then the same command with `-count=2`
- Then TC-005 passes under its unchanged name, and its output reports no failure message (no cache line after the server stop, after the Refresh price of `B0INTR0051` or after the Refresh price of `B0INTR0052`)
- And each item records 12999 cents for its owner, no file is written, as before the change
- Corner cases:
  - empty captured output for a Refresh price (nothing logged) passes;
  - lines that are not cache lines (for example the browser start-up or check log lines) captured during a Refresh price do not fail the test;
  - repeated runs (`-count=2`) pass each time: each Refresh price has its own capture, so nothing from an earlier run or step leaks into it;
  - the standard output and the `log` output are restored after each capture (later tests of the package still print normally).

### TC-002: A neutral cache line during the Refresh prices fails TC-005, naming the first item

- Covers: FR-TEST-HYG-103
- Level: integration (Go, `internal/httpapi`; real Chromium; fault injection in `internal/amazon`)
- Given the change applied, and the Amazon browser collector temporarily logging `Amazon browser cache could not be cleared: test` at the end of every check that opened the product page (the server-stop check, answered with a download, does not open the page)
- When the run executes
- Then TC-005 fails at the Refresh check of the first owner, with a message naming `B0INTR0051` and holding the line `… Amazon browser cache could not be cleared: test`
- And it does not fail at the server-stop step (the server-stop check logs no cache line)
- And, with the same injection on `master`, TC-005 passes, which shows the gap this change closes
- Corner cases:
  - the line is caught whether it is written with `log` or printed to the standard output (`captureCheckOutput` captures both);
  - the line is logged before the stored check is read, so reading the capture after `waitStoredCheck` sees it (no race with the check's end);
  - no fault injection remains in the diff (checked by TC-008).

### TC-003: An old-wording cache line for the second item only fails TC-005, naming that item

- Covers: FR-TEST-HYG-103
- Level: integration (Go, `internal/httpapi`; real Chromium; fault injection in `internal/amazon`)
- Given the change applied, and the collector temporarily logging `Amazon browser cache could not be cleared after a page not opened: test` only at the end of the check of `B0INTR0052`
- When the run executes
- Then the Refresh price of `B0INTR0051` passes its Refresh check, and TC-005 fails at the Refresh check of the second owner, with a message naming `B0INTR0052` (not `B0INTR0051`) and holding the old-wording line
- Corner cases:
  - the old wording is caught by the single prefix check (`Amazon browser cache could not be cleared`); no separate check for `after a page not opened` exists or is needed;
  - the first item's checks (price, data files, no download) were all run before the failure, so the per-item order of steps is kept;
  - no fault injection remains in the diff (checked by TC-008).

### TC-004: The existing checks of TC-005 still fail as before

- Covers: FR-TEST-HYG-104
- Level: integration (Go, `internal/httpapi`; real Chromium; fault injection)
- Given the change applied, and one of these temporary edits at a time:
  1. the collector logging `Amazon browser cache could not be cleared: test` only at the end of a check whose context was cancelled (the server-stop check);
  2. the check allowed to end before the server stop (for example the held download answer returning right after its flush, and the test waiting for the check to end before its "ended before the server stop" guard)
- When the run executes
- Then:
  1. TC-005 fails at the server-stop step with `cache clear failure logged:` and the line, as on `master`, and reports no Refresh-check failure;
  2. TC-005 fails with `the check ended before the server stop: the interruption point was missed`, as on `master`
- Corner cases:
  - the messages are the same as on `master` for the same injection;
  - no fault injection remains in the diff (checked by TC-008).

### TC-005: The Refresh check is added to TC-005 and nothing else in it changes

- Covers: FR-TEST-HYG-103, FR-TEST-HYG-104
- Level: code check of `internal/httpapi/amazon_browser_interrupted_test.go` (`git diff origin/master -- internal/httpapi/amazon_browser_interrupted_test.go`)
- Given the change applied
- When the diff and TC-005 are read
- Then, inside the per-owner loop, a new `captureCheckOutput` is started before `RefreshItem`, and after `waitStoredCheck` the captured output is searched with `linesWith(…, "Amazon browser cache could not be cleared")`; when lines are found, the test fails with a message holding the item ID and the found lines (joined one per line)
- And the test name, its `// TC-005` comment, the IDs, owners, fixture answers, the server-stop sequence and its guards, the server-stop cache check, `assertNoDownloadWritten`, `assertNoNewDataFile`, the stored-price and item-list checks and all their messages are unchanged
- And the diff only adds lines to the per-owner loop (plus, if needed, a comment line mentioning the cache-line check and issue #240/#358); no import is added or removed beyond what `gofmt` and the compiler require
- Corner cases:
  - the capture of one owner is not reused for the other (two captures, two checks);
  - the server-stop capture is not used for the Refresh checks (a line from the server-stop step cannot be blamed on an item);
  - the check matches only the prefix, not a separate `after a page not opened` string.

### TC-006: The duplicate test no longer exists, and one real-Chromium scenario fewer runs

- Covers: FR-TEST-HYG-105
- Level: repository check
- Given the change applied
- When `ls internal/httpapi/amazon_browser_cache_log_test.go`, `grep -rn 'TestAmazonBrowserCacheLog_TC004\|assertNoCacheLine' --include='*.go' .` and `go test -list . ./internal/httpapi/` are run, and the `go test -list` output is compared with the same command on `master`
- Then the file does not exist, the grep prints nothing, and the list holds `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile` and no `TestAmazonBrowserCacheLog_` test
- And the list differs from `master` only by the missing `TestAmazonBrowserCacheLog_TC004_SuccessfulClearThroughTheApplicationLogsNothing`, so the `internal/httpapi` tests start one real-Chromium scenario fewer
- Corner cases:
  - the helpers the duplicate used (`openNavigationAPI`, `captureCheckOutput`, `linesWith`, `dataFiles`, `waitStoredCheck`, `humanTime`, `assertNoDownloadWritten`, `assertNoNewDataFile`) still exist and the package compiles (`go vet ./internal/httpapi/`);
  - the deletion is requested through `delete.txt`, and the file is absent from the final tree.

### TC-007: The issue #240 test case and technical row name TC-005 as covering TC-004

- Covers: FR-TEST-HYG-106
- Level: document check (`git diff origin/master -- doc/changes/2026-11-25-1255-issue-240-neutral-wording-for-the-cache-clear-fail/testcases.md doc/specifications/amazon-human-checks/technical.md`, and `grep`)
- Given the change applied
- When the 240 document and the 240 row are read and the diff is listed
- Then both name `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile` and the real-browser `TestBrowserInterrupted_TC004_RealCheckInterruptedAroundADownloadWritesNoFile` of `internal/amazon` as covering TC-004, and say that TC-005 checks for no cache line after the server stop and after each owner's Refresh price
- And neither describes `internal/httpapi/amazon_browser_interrupted_test.go` or TC-005 as unchanged (the `internal/amazon` test may still be described as unchanged)
- And `grep -n 'amazon_browser_cache_log_test.go\|TestAmazonBrowserCacheLog_TC004'` on both files prints nothing
- And the diff of the 240 document touches only its TC-004 entry, and the diff of `technical.md` touches only the 240 row
- Corner cases:
  - the coverage tables of the 240 document (FR-AMZ-HCHK-016, UC-AMZ-HCHK-009) are unchanged, as TC-004 keeps its ID and coverage;
  - the other issue sections of `technical.md` (including the issue #233 rows that name TC-005) are unchanged.

### TC-008: Only the two test files change outside `doc/`, and the gates pass

- Covers: FR-TEST-HYG-107
- Level: repository check and suite run (deterministic gates: `gofmt -l internal/httpapi`, `go vet ./...`, `go test ./...` and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-status origin/master -- . ':!doc'` is listed and the gates run
- Then the only entries are `M internal/httpapi/amazon_browser_interrupted_test.go` and `D internal/httpapi/amazon_browser_cache_log_test.go`; `gofmt -l internal/httpapi` prints nothing; `go vet ./...` and `go test ./...` pass
- Corner cases:
  - production Go code, frontend source, `go.mod`, `go.sum`, `package.json`, `tests/` and `API_SPECIFICATION.md` are unchanged, so the operator sees no change in the application;
  - no fault injection from TC-002 to TC-004 remains;
  - the gates give the same result when run twice.
