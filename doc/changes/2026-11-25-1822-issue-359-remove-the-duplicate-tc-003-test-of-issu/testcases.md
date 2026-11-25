# Test cases: remove the duplicate TC-003 test of issue #240 (issue #359)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-099 to FR-TEST-HYG-102). There are no use cases: see [no-use-cases.md](no-use-cases.md). The behaviour the kept tests prove (no cache log line on a successful clear, FR-AMZ-HCHK-015 and FR-AMZ-HCHK-016) stays as specified in [amazon-human-checks](../../specifications/amazon-human-checks/functional.md); this change does not alter it.

Conventions:

- **The test file:** `internal/amazon/browser_cache_log_test.go`.
- **The duplicate test:** `TestBrowserCacheLog_TC003_SuccessfulClearLogsNothing`, with its comments ("A successful clear logs no cache line, whatever started it." and `// TC-003`).
- **The kept tests of the file:** `TestBrowserCacheLog_TC001_PageNotOpened` and `TestBrowserCacheLog_TC002_PageStillRequested`, with the declarations they use (`neutralCachePrefix`, `oldCacheWording`, `clearRefusedCause`, `clearUnboundCause`, `logMessage`, `assertNeutralCacheLine`, `assertNoOldCacheWording`, `assertBrowserClosed`) and the imports `net/http`, `regexp`, `strings`, `testing`, `time`.
- **The success-clear tests:** the issue #233 tests in `internal/amazon/browser_interrupted_test.go` that the issue #240 TC-003 entry names: `TestBrowserInterrupted_TC001_ClearConfirmedBeforeClose`, `TestBrowserInterrupted_TC002_PageStillRequestedIsCleared` and `TestBrowserInterrupted_TC006_UninterruptedCheckIsUnchanged` (subtest `download`).
- **The issue #240 test documents:** `doc/changes/2026-11-25-1255-issue-240-neutral-wording-for-the-cache-clear-fail/` (its `testcases.md` and technical specification) and the issue #240 section of `doc/specifications/amazon-human-checks/`.
- **Fault injection:** a temporary edit of the cache clear in `internal/amazon/browser.go`, made only for the run of a case, reverted afterwards and never committed.
- The change only deletes test code. No production code, API, database or user interface changes, and there is no user flow, so there are no integration or e2e cases of the application. The cases are code checks, unit runs (one with fault injection) and the gates. The IDs below are this change's own, not those in the test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-099 | TC-001 | — |
| FR-TEST-HYG-100 | TC-002, TC-003, TC-004 | — |
| FR-TEST-HYG-101 | TC-005, TC-006, TC-007 | — |
| FR-TEST-HYG-102 | TC-004, TC-008 | — |

## Cases

### TC-001: The duplicate test is gone, and the file has no successful-clear test

- Covers: FR-TEST-HYG-099
- Level: code check and unit run (Go, `internal/amazon`)
- Given the change applied on top of `master`
- When `grep -rn 'TestBrowserCacheLog_TC003_SuccessfulClearLogsNothing' --include='*.go' .` runs, the test file is searched for `// TC-003` and `Clear: "answer"`, and `go test -count=1 -run 'TestBrowserCacheLog_TC003' -v ./internal/amazon/` runs
- Then the grep prints nothing, the test file contains neither `// TC-003` nor `Clear: "answer"`, and the `go test` run reports no test run (`testing: warning: no tests to run`) and passes
- Corner cases:
  - the test is not merely renamed, skipped (`t.Skip`), commented out or moved to another file of `internal/amazon`: no test of the repository has its subtests ("live check, page not opened, repeated", "interrupted after the offer, …", "page still requested, …") with a fake browser answering the clear;
  - its comment "A successful clear logs no cache line, whatever started it." is removed with it;
  - the success-clear tests in `browser_interrupted_test.go`, which do use `Clear: "answer"`, are not affected by this check (they are in another file).

### TC-002: The kept tests keep their names, subtests and results

- Covers: FR-TEST-HYG-100
- Level: unit (Go, `internal/amazon`, fake browser)
- Given the change applied
- When `go test -count=1 -run '^TestBrowserCacheLog_' -v ./internal/amazon/` runs on `master` and with the change, then again with the change with `-count=2`
- Then every run passes; with the change it runs exactly `TestBrowserCacheLog_TC001_PageNotOpened` (subtests for each clear failure, "interrupted after the offer, error answer", "repeated") and `TestBrowserCacheLog_TC002_PageStillRequested` (each interruption kind × each clear failure), with the same subtest names as on `master`, and no `TestBrowserCacheLog_TC003_…` test or subtest
- Corner cases:
  - repeated runs give the same result;
  - the run with the change is about 8 s shorter than on `master` (four interrupted fake-browser subtests of about 2 s fewer), and no kept subtest is lost.

### TC-003: The file keeps no import or declaration only the duplicate test used

- Covers: FR-TEST-HYG-100
- Level: code check of the test file, and `go vet` / build of `internal/amazon`
- Given the change applied on top of `master`
- When the import block and the declarations of the test file are read, and `go vet ./internal/amazon/` runs
- Then:
  - every remaining import (`net/http`, `regexp`, `strings`, `testing`, `time`) is still used by the kept tests or their helpers;
  - every remaining declaration (constants, `logMessage`, `assertNeutralCacheLine`, `assertNoOldCacheWording`, `assertBrowserClosed`) is still used by the kept tests;
  - `assertNoOldCacheWording` stays, as the kept tests call it;
  - `go vet ./internal/amazon/` passes (no unused import, no compile error).
- Corner cases:
  - a helper declared in `browser_interrupted_test.go` and used by the duplicate test (`assertNoCacheLogLine`, `assertClearConfirmedBeforeClose`, `assertNavigationFailure`, `assertNavigatedBeforeInterruption`, `assertInterruptedFailure`, `interruptedContext`, `collectInterrupted`) is not removed, since other tests use it and that file is unchanged;
  - no blank import or dummy use is left to keep an import alive.

### TC-004: The diff of the test file only removes the duplicate test

- Covers: FR-TEST-HYG-100, FR-TEST-HYG-102
- Level: repository check (`git diff origin/master -- internal/amazon/browser_cache_log_test.go`)
- Given the change applied on top of `master`
- When the diff of the test file is listed
- Then it contains no added line, and its removed lines are only the duplicate test, its two comment lines and the blank line separating it from the kept tests
- Corner cases:
  - the kept tests, their subtests, assertions and messages, the helpers and the file's header comment are byte-for-byte unchanged;
  - the file still ends with a single newline after the last kept test (checked by `gofmt` in TC-008).

### TC-005: The success-clear tests still prove TC-003 of issue #240 and pass

- Covers: FR-TEST-HYG-101
- Level: unit (Go, `internal/amazon`, fake browser)
- Given the change applied
- When `go test -count=1 -run '^(TestBrowserInterrupted_TC001_ClearConfirmedBeforeClose|TestBrowserInterrupted_TC002_PageStillRequestedIsCleared|TestBrowserInterrupted_TC006_UninterruptedCheckIsUnchanged)$' -v ./internal/amazon/` runs
- Then it passes, and its output shows the subtests that cover what the duplicate test covered: both interruption kinds of TC-001 (interrupted after the offer, each run twice by one collector), the held and late navigation subtests of TC-002 (page still requested), and the `download` subtest of TC-006 (live check); each calls `assertNoCacheLogLine`
- Corner cases:
  - `assertNoCacheLogLine` matches the kept prefix `Amazon browser cache could not be cleared`, so it also rejects a line with the old wording `… after a page not opened`, which makes the duplicate test's `assertNoOldCacheWording` calls redundant;
  - repeated checks stay covered by TC-001 of issue #233 (two `Collect` calls per subtest).

### TC-006: A cache line logged after a successful clear makes the success-clear tests fail

- Covers: FR-TEST-HYG-101
- Level: unit (Go, `internal/amazon`; fault injection)
- Given the change applied, and the cache clear in `internal/amazon/browser.go` temporarily edited to log `Amazon browser cache could not be cleared: <anything>` also when `Network.clearBrowserCache` succeeds (for example logging before the `if err != nil` check), and then, as a second edit, to log `Amazon browser cache could not be cleared after a page not opened` on success
- When the command of TC-005 runs with each edit
- Then it fails with each edit, with `cache clear failure logged:` in the subtests of TC-001, TC-002 and TC-006 `download`, which shows the deletion loses no coverage of FR-AMZ-HCHK-016
- And, without the edit, the command passes (TC-005)
- Corner cases:
  - the kept `TestBrowserCacheLog_` tests are not required to fail under these edits (they cover failed clears);
  - no fault injection remains in the diff (checked by TC-008).

### TC-007: The success-clear tests and the issue #240 test documents are unchanged

- Covers: FR-TEST-HYG-101
- Level: repository check (`git diff origin/master`)
- Given the change applied on top of `master`
- When `git diff origin/master -- internal/amazon/browser_interrupted_test.go` and `git diff origin/master -- doc/changes/2026-11-25-1255-issue-240-neutral-wording-for-the-cache-clear-fail/ doc/specifications/amazon-human-checks/` are listed
- Then both diffs are empty
- And the issue #240 `testcases.md` still assigns TC-003 to the success-clear tests ("existing tests, unchanged"), and the issue #240 test documents list `browser_cache_log_test.go` for TC-001 and TC-002 only
- Corner cases:
  - no test table of the issue #240 technical specification gains an entry for the duplicate test (the "keep it and list it" option of the issue is not taken);
  - the follow-up record `doc/todo/issue-240-r1-02.md` is not required to be kept or changed by this case.

### TC-008: Only the test file changes outside `doc/`, and the gates pass

- Covers: FR-TEST-HYG-102
- Level: repository check and suite run (deterministic gates: `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only origin/master -- . ':!doc'` is listed and the gates run
- Then the only file listed outside `doc/` is `internal/amazon/browser_cache_log_test.go` (TC-004 checks it only loses lines), `gofmt -l internal/amazon` prints nothing, and `go vet ./...` and `go test ./...` pass
- Corner cases:
  - `internal/amazon/browser.go` and the other production Go code, frontend source, `go.mod`, `go.sum`, `package.json`, `tests/` and `API_SPECIFICATION.md` are unchanged, so the operator observes no change in the application;
  - no fault injection from TC-006 remains;
  - the gates give the same result when run twice.
