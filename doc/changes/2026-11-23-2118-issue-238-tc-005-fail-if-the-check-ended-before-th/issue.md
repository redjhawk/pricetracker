# Issue #238 TC-005: fail if the check ended before the server stop

**Problem:** `internal/httpapi/amazon_browser_interrupted_test.go` (around line 40) calls `cancel()` after `api.fixture.flushed` without checking that the `Collect` goroutine is still running. A future change that made the check finish faster would make the 'server stop' a no-op without failing.

**Suggested work:** Before `cancel()`, add `select { case <-stopped: t.Fatal("the check ended before the server stop: interruption point missed"); default: }`, as the tech spec risk note requires (a missed interruption point must fail with a message).

From review comment R1-03 on #236 (`doc/todo/issue-233-r1-03.md`).

## Comments by trusted authors

