# Issue #373 Fix stale doc comment on Store.PurchaseGoals

**Problem:** The doc comment of `PurchaseGoals` (`internal/store/store.go:833`) says it returns goals of the owner's "LeBoncoin items", but since #305 it returns goals of LeBoncoin and Amazon items (no platform filter in the query). A maintainer could trust the comment and add a platform filter.

**Suggested work:** Change the comment to "PurchaseGoals returns the distinct non-empty purchase goals of the owner's tracked items (LeBoncoin and Amazon), unordered."

From review comment R3-01 on #327 (`doc/todo/issue-304-r3-01.md`).

## Comments by trusted authors

