# Issue #230 Remove redundant used-offer count check in TC-004 refresh test

**Problem:** In `internal/httpapi/amazon_browser_error_page_test.go:150`, `TestAmazonBrowserErrorPage_TC004_RefreshSendsNoUsedOfferRequest` checks `api.usedOfferCount() != 0` again right after the loop. The loop's last attempt (line 146) has just made the same check, and the count only grows, so the later check can never fail on its own. The Refresh all prices step (lines 133-139) has no count check of its own; only the cumulative check in the loop covers it.

**Suggested work:** Add a `usedOfferCount() == 0` check right after the Refresh all prices step so each step checks its own count, and remove the redundant check after the loop. Another option is to keep only one total check at the end and drop the per-attempt check inside the loop.

From review comment R1-01 on #229 (`doc/todo/issue-187-r1-01.md`).

## Comments by trusted authors

