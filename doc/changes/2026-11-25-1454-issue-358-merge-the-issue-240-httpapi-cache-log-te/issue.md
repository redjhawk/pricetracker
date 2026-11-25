# Issue #358 Merge the issue #240 httpapi cache-log test into the existing TC-005 of #233

**Problem:** `internal/httpapi/amazon_browser_cache_log_test.go` (`TestAmazonBrowserCacheLog_TC004_SuccessfulClearThroughTheApplicationLogsNothing`) copies `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile` in `internal/httpapi/amazon_browser_interrupted_test.go` (lines 19-79): same fixture, server-stop sequence, per-owner Refresh price and assertions. Each CI run therefore starts one more real-Chromium scenario. The technical specification (`doc/specifications/amazon-human-checks/technical.md`, issue #240 section, TC-004 row) says the existing test covers TC-004. In `assertNoCacheLine`, the second string `after a page not opened` cannot match on its own, because any old-wording line also contains the prefix `Amazon browser cache could not be cleared`.

**Suggested work:** Add the no-cache-line check after each Refresh price (`linesWith(output(), "Amazon browser cache could not be cleared")` with a `captureCheckOutput` per owner) to the existing TC-005 test. Then delete `internal/httpapi/amazon_browser_cache_log_test.go` and update the TC-004 row of the technical specification and `testcases.md` if needed.

From review comment R1-01 on #357 (`doc/todo/issue-240-r1-01.md`).

## Comments by trusted authors

