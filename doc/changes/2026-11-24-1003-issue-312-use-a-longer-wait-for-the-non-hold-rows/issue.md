# Issue #312 Use a longer wait for the non-hold rows of the issue #195 fixture-check test

**Problem:** `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal` in `internal/amazon/browser_profile_test.go` (around line 500) gives every row a 2 s wait in `runBrowserOnProfile`. Rows without `Hold` end as soon as the stub exits, so they only need the short wait to stop a stub that never ends. On a heavily loaded runner, a stub that is killed before it writes its stderr or finishes its GET would make the row fail for a timing reason.

**Suggested work:** Choose the wait for each row, for example `wait := 15 * time.Second; if row.stub.Hold { wait = 2 * time.Second }`, and pass `wait` to `runBrowserOnProfile`. This costs nothing on a normal run. For the `Hold` rows, you could also make the stub signal that it has written its output (for example by closing a file or writing a marker) before the wait starts counting.

**Where:** `internal/amazon/browser_profile_test.go`, the `t.Run` loop of `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal` (the `runBrowserOnProfile(..., 2*time.Second)` call).

From review comment R1-01 on #311 (`doc/todo/issue-195-r1-01.md`).

## Comments by trusted authors

