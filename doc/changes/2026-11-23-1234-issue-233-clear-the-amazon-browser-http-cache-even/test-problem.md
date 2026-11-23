# Locked test problem: TC-004 "after the download was offered" can miss its interruption point silently

Blocking review comments R1-01 and R2-01 are both about the locked test file
`internal/amazon/browser_interrupted_test.go` (hash in `tests.lock`). Only that test can fix them; changing
production code cannot. Any change made here would be undone, because the factory restores locked test files
after the development stage. The round-1 development pass left the file untouched for the same reason, so
R2-01 repeats R1-01.

## Problem

`TestBrowserInterrupted_TC004_RealCheckInterruptedAroundADownloadWritesNoFile`, subtests
"after the download was offered, …" (around lines 325–343):

- The fixture flushes the attachment at `at − 100 ms`.
- The test then only checks that the request arrived and that the body was flushed before `at`.
- It never checks that the check was still running when its context ended.

Real Chromium may answer `Page.navigate` with `errorText` and report the 200 `Document` response
(TS-AMZ-HCHK-017) within those 100 ms. If it does, `awaitDocument` returns, `clearCache` runs with the live
context, and `Collect` returns before `at`. The deadline or cancel then interrupts nothing. The subtest still
passes, and it would also pass on the pre-change code. That goes against the technical specification's risk
note: "a test that misses its interruption point must fail with a message saying so, never pass silently".

## Required fix (test lock must be lifted for this file)

In the TC-004 subtest, note the time right after `collectInterrupted` returns, and fail if it is before the
interruption:

```go
got := collectInterrupted(t, collector, ctx, item, at)
returned := time.Now()
...
if returned.Before(at) {
	t.Fatalf("check returned at %s, before the interruption at %s: the interruption point was missed",
		returned.Format("15:04:05.000"), at.Format("15:04:05.000"))
}
```

If, with this check, real Chromium always reports the response together with the `Page.navigate` error, so the
check always ends before `at`, the point cannot be reached with a real browser. In that case:

- Say so in TC-004 of `testcases.md` and in the technical specification.
- Rely on TC-001 (fake browser) for this path.
- Drop the "after the download was offered" subtests, or relabel them so they no longer claim to prove
  FR-AMZ-HCHK-015.

A possible alternative that keeps the real-browser case: flush the body at or after `at` instead of at
`at − 100 ms`, so the interruption is sure to come while the check is still waiting. Whoever writes the test
should choose.

The implementation (`internal/amazon/browser.go`, round 0) does not need to change for this comment.
