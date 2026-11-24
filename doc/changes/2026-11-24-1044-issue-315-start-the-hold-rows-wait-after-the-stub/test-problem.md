# Locked test problem: blocking comment R1-01 needs a change to the locked test

Blocking comment R1-01 is about the locked test `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal` in `internal/amazon/browser_profile_test.go`. The whole change of issue #315 is in that file (no-tests.md, TS-TEST-HYG-067), and FR-TEST-HYG-052 allows no other file outside `doc/`. The development stage may not edit test files, and the factory restores them to the TDD red commit after this run. So no development change can fix R1-01; the locked test must be corrected by the test stage.

## Problem

In the hold branch of the `t.Run` loop (lines 544-549), the 2 s wait after the output signal is:

```go
select {
case <-exited:
case <-time.After(2 * time.Second):
}
```

`case <-exited:` ends the wait early without any trace. The later `ESRCH` check passes whether the row killed the stub or the stub exited by itself. So the test does not prove the part of FR-TEST-HYG-049 and TC-003 that says both hold stubs are still running when the 2 s end. A "TC-002 slow start" stub that signals and then exits gives exactly the expected result (empty output, no request, the `missing` failure) and passes.

## Required correction (test stage)

Record which way the hold wait ended and fail when a signalled stub exited before the 2 s, for example:

```go
held := true
if row.stub.Hold {
	run = runBrowserOnProfileUntil(t, executable, profile, server, path, func(exited <-chan struct{}) {
		if signalled = waitForOutputSignal(signal, exited, 15*time.Second); signalled {
			select {
			case <-exited:
				held = false
			case <-time.After(2 * time.Second):
			}
		}
	})
}
...
if !held {
	t.Errorf("the stub exited before the end of the wait")
}
```

Non-hold rows keep `held` true. The no-tests.md row for TC-003 should then cite this check, and a fault-injection run should show it: with `os.Exit(0)` right after the `os.WriteFile(stub.Signal, …)` of a hold stub, both hold rows fail with `the stub exited before the end of the wait` (and "TC-002 slow start" fails on that message alone).
