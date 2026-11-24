# Issue #293 Restore the clientAddress doc comment in auth.go

**Problem:** In `internal/httpapi/auth.go` (lines 186-198) the new `secureRequest` function and its comment were inserted between the doc comment of `clientAddress` and `func clientAddress`. Now `secureRequest`'s doc comment starts with the `clientAddress` text, and `clientAddress` has no doc comment.

**Suggested work:** Move the two `clientAddress` comment lines (186-187) back directly above `func clientAddress` (line 198), leaving `secureRequest` with only its own comment and a blank line between the two functions. Do not change any behaviour.

From review comment R1-01 on #292 (`doc/todo/issue-268-r1-01.md`).

## Comments by trusted authors

