# Issue #359 Remove the duplicate TC-003 test of issue #240 in internal/amazon

**Problem:** `internal/amazon/browser_cache_log_test.go`, `TestBrowserCacheLog_TC003_SuccessfulClearLogsNothing` (around lines 170-212), repeats `TestBrowserInterrupted_TC001_ClearConfirmedBeforeClose`, `TestBrowserInterrupted_TC002_PageStillRequestedIsCleared` and `TestBrowserInterrupted_TC006_UninterruptedCheckIsUnchanged` (subtest `download`) in `browser_interrupted_test.go`. It adds about 4 × 2 s of interrupted fake-browser runs. Its `assertNoOldCacheWording` calls are redundant after `assertNoCacheLogLine`, which matches the kept prefix.

**Suggested work:** Delete the test and keep TC-003 on the existing tests, as the technical specification (issue #240 section, TC-003 row) and `testcases.md` state. If the test is kept, list it in the specification's test tables instead.

From review comment R1-02 on #357 (`doc/todo/issue-240-r1-02.md`).

## Comments by trusted authors

