# No red phase for issue #239

This change is itself a test change. On `master` the behaviour it checks already holds: `clearCache` (`internal/amazon/browser.go`) maps a never-answered clear to `Network.clearBrowserCache: no answer within 5s`. So the tightened TC-003 passes, and it cannot show a red phase on the code. The tests stage delivers the change in `internal/amazon/browser_interrupted_test.go`. The red state is shown with fault injections instead (run 2). The development stage writes no code.

## Delivered change

One hunk in the `t.Run` function of `TestBrowserInterrupted_TC003_UnconfirmedClearIsBoundedAndLogged`, after the existing `context canceled` check: a comment, a check guarded by `test.clear == "never"` that `lines[0]` contains `no answer within` (the message holds the line), and `assertNoInterruptionGivenForCache(t, logged)`. The subtest table, the other checks, the helpers and the other tests are unchanged. `gofmt -l internal/amazon` prints nothing.

## Runs and fault injections

`R` is `go test -run '^TestBrowserInterrupted_TC003_UnconfirmedClearIsBoundedAndLogged$' -v -count=1 ./internal/amazon/`. The sandbox refused writes outside the repository, so there were no overlay files. Each injection was a temporary in-place edit of `clearCache` or of the test file, reverted before the next step. At the end, `git diff --name-only master -- . ':!doc'` lists only the test file.

1. **Normal run (TC-001).** `R` passes with exactly three subtests: `no_answer` (7.01 s), `error_answer` (2.00 s) and `no_answer,_page_still_requested` (7.01 s). These are the same names and about the same times as on `master`. With `-count=3` it also passes (48 s, three times 16 s).
2. **Deadline mapping removed from `clearCache` (TC-002).** The cache line becomes `...: context deadline exceeded`.
   - (a) With the new test, `R` fails `no answer` and `no answer, page still requested` at line 236 with `cache log line without "no answer within", the cause of a clear that never answers: ... Amazon browser cache could not be cleared after a page not opened: context deadline exceeded`. `error answer` passes.
   - (b) With the test file as on `master` (`git diff master` on it was empty), `R` passes, so the old checks missed this regression.
   - (c) With only the helper call added (no `no answer within` check), `R` fails both never-answered subtests with `the interruption of the check is given as the cause of a cache clear: ...: context deadline exceeded`. `error answer` passes.
   - (d) With only the `no answer within` check added (no helper call), `R` fails both with the `no answer within` message. `error answer` passes.
3. **Cause reworded to `no reply within` (TC-003).** `R` fails both never-answered subtests at line 236 with the `no answer within` message, which holds `...: Network.clearBrowserCache: no reply within 5s`. `error answer` passes.
4. **Context cause appended to other failures (TC-004).** The edit was `} else if err != nil { err = fmt.Errorf("%w: <cause>", err) }` after the deadline mapping.
   - Edit 1 (`context deadline exceeded`): `R` fails only `error answer`, with the helper's message `the interruption of the check is given as the cause of a cache clear: ...: Network.clearBrowserCache: cache clear refused by fake: context deadline exceeded`.
   - Edit 2 (`context canceled`): `R` fails only `error answer` at line 231, with the existing message `the interruption of the check is given as the cause of the cache clear failure: ...: cache clear refused by fake: context canceled`.
5. **Other tests of the file (TC-007).** `go test -run '^TestBrowserInterrupted_' -v -count=1 ./internal/amazon/` was run with the change and with the `master` test file. Both runs give the same tests and subtests (TC001 ×2, TC002 ×3, TC003 ×3, TC004 ×5, TC006 ×2), all passing, with none skipped.
6. **Gates (TC-008), run twice.** `gofmt -l internal/amazon` prints nothing. `go vet ./...` and `go test -count=1 ./...` pass. `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/browser_interrupted_test.go`.

## Where each test case is proved

| Test case | Proof |
| --- | --- |
| TC-001 | Run 1 |
| TC-002 | Run 2 (a–d) |
| TC-003 | Run 3 |
| TC-004 | Run 4 (edits 1 and 2) |
| TC-005 | Code check: `git diff master -- internal/amazon/browser_interrupted_test.go` is the single hunk described above |
| TC-006 | Document check: `git diff --name-status master -- doc/changes/2026-11-23-1234-issue-233-clear-the-amazon-browser-http-cache-even/` lists only `M .../testcases.md`, and exactly one line changes: the third "Then" result of its TC-003 entry. The technical specification stage made that edit. |
| TC-007 | Run 5 |
| TC-008 | Run 6 |
