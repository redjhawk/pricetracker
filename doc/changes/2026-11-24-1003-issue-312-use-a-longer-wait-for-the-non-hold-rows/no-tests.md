# No red phase for issue #312

The change is the test code itself: the wait of `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal` in `internal/amazon/browser_profile_test.go`. That file is locked after this stage, and FR-TEST-HYG-047 allows no other file outside `doc/`. So the development stage cannot write any of it. This stage delivers the whole change, as the technical specification says (TS-TEST-HYG-060). A normal run passes before and after the change, so no test can fail first on a missing implementation. The red gate is skipped. The development stage writes no code: it only sets the statuses.

## Delivered in `internal/amazon/browser_profile_test.go`

In the `t.Run` loop only:

```go
executable := refusalStubBrowser(t, row.stub)
wait := 15 * time.Second // as the fixture sanity check; a stub that exits ends the run at once
if row.stub.Hold {
	wait = 2 * time.Second // a held stub runs until the end of the wait
}
run := runBrowserOnProfile(t, executable, profile, server, path, wait)
```

The choice reads only `row.stub.Hold`. The rows table, the stub, `runBrowserOnProfile`, `fixtureCheckFailure`, `assertBrowserRefusesProfile` (and its 15 s wait) and the assertions are unchanged. No output signal is added for the hold rows.

## Runs

Run command: `go test ./internal/amazon -run '^TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal$' -count=1 -v`. Each injection was a temporary `Edit` of the test file, reverted with `Edit` before the next step.

1. Normal run: all ten rows PASS (4.09 s). Both hold rows take 2.00 s each. Every non-hold row takes 0.00–0.01 s. With `-count=3`: PASS three times (4.11 s, 4.08 s, 4.10 s). With `-run 'TestBrowserProfile' -count=1`: every `TestBrowserProfile_*` test passes, including `TestBrowserProfile_TC002_LockOfAnotherHostDoesNotPreventTheCheck` with the real browser (2.90 s). `-race` could not run here: it needs cgo, and this runner has no C compiler (`cgo: C compiler "gcc" not found`). The change adds no concurrency: the row still kills the group and waits for `<-exited` before it reads the result.
2. Slow non-hold stub: I added `if !stub.Hold { time.Sleep(5 * time.Second) }` right after the decode in `TestBrowserProfile_Issue195_RefusalStubProcess`. All ten rows PASS (44.13 s total). Each non-hold row takes 5.01–5.02 s, and the hold rows take 2.00 s. With a 3 s delay instead: all ten PASS (28.11 s), non-hold rows 3.01–3.02 s.
3. The same 5 s delay, with the non-hold wait set back to `2 * time.Second`: FAIL. Seven non-hold rows fail, among them `TC-001 refusal, then exit` (`captured output "", want "[1:1:ERROR:…(otherhost)…"` and `the check failed on a refusal without page request`) and `TC-003 page requested with the refusal` (`page requested: false, want true`, and the wrong conditions named). The other failing rows are `missing library`, `unknown option`, `generic lock error only`, `refusal naming another host` and `TC-003 page requested without output`. `TC-002 exit at once without output` still passes, because it expects empty output and no request. So the delay is what the longer wait removes.
4. A non-hold stub that does not end: the wait was back at 15 s, and the sleep of step 2 was replaced by `if stub.Exit == 21 && strings.Contains(stub.Stderr, "(otherhost)") { time.Sleep(time.Minute) }`, which matches only `TC-001 refusal, then exit`. That row FAILS after 15.02 s with `captured output "", want …` and `the check failed on a refusal without page request: fixture check: the browser output does not hold the lock refusal message "on another computer (otherhost)"; browser output (0 bytes):`. There is no process-group error (so `ESRCH`: the group is gone). The other nine rows pass, and the test ends (19.10 s) without hanging.
5. After the reverts: `git diff master -- internal/amazon/browser_profile_test.go` shows only the wait lines of the `t.Run` loop (one line replaced by the five above). Run 1 passes again (4.10 s). `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/browser_profile_test.go`. `gofmt -l internal/amazon` prints nothing, `go vet ./...` passes, and `go test ./...` passes (all packages `ok`).

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | Code check of the `t.Run` loop (above): 15 s unless `row.stub.Hold`, then 2 s; rows table unchanged |
| TC-002 | Runs 2 (5 s and 3 s delays pass), 3 (2 s wait fails on the named rows) and 4 (a non-ending non-hold stub fails after about 15 s, group gone) |
| TC-003 | Run 1: the hold rows `TC-001 refusal, then still running at the end of the wait` (passes) and `TC-002 slow start` (fails the check naming the missing message) take 2.00 s each, with the `ESRCH` assertion |
| TC-004 | Run 1 with `-v`, `-count=3` and `-run 'TestBrowserProfile'` (`-race` not available on this runner), and `go test ./...` in run 5 |
| TC-005 | Repository checks and gates, run 5. The development stage runs them again |
