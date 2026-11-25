# Issue #240 Neutral wording for the cache-clear failure log line

**Problem:** `clearCache` in `internal/amazon/browser.go` (around line 500) logs `Amazon browser cache could not be cleared after a page not opened: %v`. Since #233 it also runs when `Page.navigate` itself was interrupted (around line 429), where no answer to the navigation was received, so the suffix can mislead the reader of the log.

**Suggested work:** Drop the suffix (`Amazon browser cache could not be cleared: %v`) or let the caller pass the reason (navigation interrupted / page not opened). Keep the prefix required by TS-AMZ-HCHK-018 and check the tests matching `cacheLogLine` still pass.

From review comment R2-02 on #236 (`doc/todo/issue-233-r2-02.md`).

## Comments by trusted authors

