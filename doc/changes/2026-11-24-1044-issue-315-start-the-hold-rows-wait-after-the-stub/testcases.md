# Test cases: start the hold rows' wait after the stub signals it wrote its output (issue #315)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-048 to FR-TEST-HYG-052). There are no new use cases: see [no-use-cases.md](no-use-cases.md). UC-AMZ-BPROF-002 describes the application behaviour that TC-002 of issue #180 and its fixture sanity check stand for, and this change does not alter it.

Conventions:

- **Fixture check test:** `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal` in `internal/amazon/browser_profile_test.go`. Each row starts a stub executable (`refusalStubBrowser`, run by `TestBrowserProfile_Issue195_RefusalStubProcess`) on a profile locked by `otherhost`, with the `amazontest` fixture server, and then checks `fixtureCheckFailure`, the captured output, the page request and that the stub's process group is gone.
- **Hold rows:** the rows whose stub has `Hold: true` ("TC-001 refusal, then still running at the end of the wait", "TC-002 slow start"). **Non-hold rows:** the eight other rows; their stubs exit by themselves.
- **Output signal:** the sign given by a hold stub after it has written its output (possibly empty) and made its page request if it makes one. **Signal bound:** 15 s from the stub's start. How the stub signals (for example a marker file) is a technical choice; the cases check only its effect.
- **Fixture sanity check:** `assertBrowserRefusesProfile`, with its 15 s wait counted from the browser's start, used by TC-002 of issue #180. This change does not touch it.
- **Run the test:** `go test ./internal/amazon -run '^TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal$' -count=1 -v`. It needs no real browser.
- **Fault injection:** a temporary edit of the stub (for example a sleep before it writes its output when it holds), made only for the run of a case, reverted afterwards and never committed.
- The change only touches test code. No production code, API, database or user interface changes, and there is no user flow, so there are no integration or e2e cases of the application. The cases are a code check, runs of the fixture check test (some with fault injection) and the gates. The IDs below are this change's own, not those of the rows' names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-048 | TC-001, TC-002 | — |
| FR-TEST-HYG-049 | TC-002, TC-003 | UC-AMZ-BPROF-002 |
| FR-TEST-HYG-050 | TC-004 | — |
| FR-TEST-HYG-051 | TC-005 | UC-AMZ-BPROF-002 |
| FR-TEST-HYG-052 | TC-001, TC-006 | UC-AMZ-BPROF-002 |

## Cases

### TC-001: The hold stub signals after its output, and a hold row waits for the signal, then 2 s

- Covers: FR-TEST-HYG-048, FR-TEST-HYG-052
- Level: code check of `internal/amazon/browser_profile_test.go`
- Given the change applied on top of `master`
- When `TestBrowserProfile_Issue195_RefusalStubProcess`, `refusalStubBrowser` and the `t.Run` loop of the fixture check test are read
- Then:
  - a stub with `Hold: true` gives the output signal only after its page request (when `Request` is set) and after writing `Stderr` to standard error, and then keeps sleeping until it is killed;
  - a hold row waits for that signal for at most 15 s counted from the stub's start, then lets the stub run 2 s more before killing its process group;
  - a non-hold row keeps its 15 s wait counted from the stub's start, ending at once when its stub exits;
  - the choice between the two waits depends only on the row's `Hold` field, not on the row's name or position.
- Corner cases:
  - each row has its own signal (for example a marker file in the row's own temporary directory), so the signal of an earlier row, or of an earlier run with `-count`, cannot release a later hold row before its own stub wrote its output;
  - the 2 s hold period is counted from the signal, not from the stub's start, so it is not shortened by a slow start;
  - when the environment variable of the stub is unset, `TestBrowserProfile_Issue195_RefusalStubProcess` still does nothing (no signal, no file written) in a normal `go test` run;
  - the rows table (names, stubs, expected failures) is unchanged.

### TC-002: A slow hold stub still gives each hold row's expected result

- Covers: FR-TEST-HYG-048, FR-TEST-HYG-049
- Level: unit (Go, `internal/amazon`, stub executables and fixture server; fault injection)
- Given the change applied, and the hold stub temporarily made to sleep 5 s after its start before writing its output
- When the fixture check test is run with `-v`
- Then both hold rows pass with their expected result: "TC-001 refusal, then still running at the end of the wait" passes the check with captured output equal to the refusal message and no page request, and "TC-002 slow start" fails the check naming only the missing refusal message, with empty captured output and no page request; each hold row takes about 7 s (5 s before the signal plus 2 s)
- And, with the same fault injection applied to the `master` version of the test (2 s counted from the stub's start), "TC-001 refusal, then still running at the end of the wait" fails (`captured output ""` and `the check failed on a refusal without page request`), which shows the delay is the cause the change removes
- Corner cases:
  - a delay just over 2 s (for example 3 s) and one close to the bound (for example 12 s) pass as well as 5 s;
  - the non-hold rows are unaffected by the fault injection and still pass;
  - after each hold row, its stub's process group is gone (`syscall.Kill(-pid, 0)` reports `ESRCH`);
  - no fault injection remains in the diff (checked by TC-006).

### TC-003: The hold rows keep their meaning on a normal run

- Covers: FR-TEST-HYG-049; UC-AMZ-BPROF-002
- Level: unit (Go, `internal/amazon`, as TC-002, without fault injection)
- Given the change applied
- When the fixture check test is run with `-v`
- Then, when the 2 s after the output signal end:
  - the stub of "TC-001 refusal, then still running at the end of the wait" has written the whole lock refusal message, has not requested the page, and is still running (the row kills it; it has not exited by itself), and the row passes the check;
  - the stub of "TC-002 slow start" has written nothing, has not requested the page, and is still running, and the check fails naming only the missing refusal message;
  - the failure text of "TC-002 slow start" is exactly `fixture check: <missing>; browser output (0 bytes):\n`, as before.
- Corner cases:
  - the signal itself adds nothing to the captured output (standard output and standard error hold only the stub's `Stderr`);
  - the hold stubs make no page request, so the fixture server's count for the page path is unchanged by the hold rows.

### TC-004: A hold stub that never signals makes its row fail, bounded, without leftover process

- Covers: FR-TEST-HYG-050
- Level: unit (Go, `internal/amazon`, as TC-002; fault injection)
- Given the change applied, and the hold stub temporarily made to sleep 20 s after its start before writing its output and signalling
- When the fixture check test is run with `-v`
- Then each hold row fails with a message saying that the stub did not signal its output within 15 s; each such row takes about 15 s and less than 20 s; its stub's process group is killed and gone after the row (`syscall.Kill(-pid, 0)` reports `ESRCH`); the test does not hang
- Corner cases:
  - a hold stub temporarily made to exit at once without writing or signalling also makes its row fail (it gives no signal within 15 s), ends no later than about 15 s after the stub's start and leaves no process group;
  - a hold stub temporarily made to write its output but never signal fails the same way, so a missing signal is never mistaken for a pass;
  - the other rows still run and pass: the failure of one hold row does not stop or slow the non-hold rows;
  - no fault injection remains in the diff (checked by TC-006).

### TC-005: A normal run passes unchanged, with hold rows of about 2 s, and leaves no process

- Covers: FR-TEST-HYG-051; UC-AMZ-BPROF-002
- Level: unit (Go, `internal/amazon`, as TC-002, without fault injection)
- Given the change applied, on an unloaded machine
- When the fixture check test is run with `-v`, then again with `-count=3`
- Then all ten rows pass every time with the same expected results and assertions as on `master`; in the `--- PASS` timings each hold row takes at least 2 s and less than 3 s, and each non-hold row takes less than 2 s because it ends as soon as its stub exits
- Corner cases:
  - after every row, `syscall.Kill(-pid, 0)` reports `ESRCH`: no process group of a stub is left running;
  - repeated runs give the same results, so the outcome does not depend on timing or on a signal left by an earlier run;
  - the test still passes when run together with the other `TestBrowserProfile_*` tests (`-run 'TestBrowserProfile'`) and with `-race`.

### TC-006: Only the test file changes outside `doc/`, the sanity check and non-hold wait are unchanged, and the gates pass

- Covers: FR-TEST-HYG-052; UC-AMZ-BPROF-002
- Level: repository check and suite run (deterministic gates: `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` and `git diff master -- internal/amazon/browser_profile_test.go` are listed and the gates run
- Then:
  - the only file listed outside `doc/` is `internal/amazon/browser_profile_test.go`;
  - its changed hunks are in the hold stub (`refusalStub`, `refusalStubBrowser`, `TestBrowserProfile_Issue195_RefusalStubProcess`) and the hold rows' wait of the fixture check test, plus any helper they need; they do not change `assertBrowserRefusesProfile`, its 15 s wait counted from the browser's start, `fixtureCheckFailure`, the non-hold rows' 15 s wait counted from the stub's start, the rows table, or TC-002 of issue #180;
  - `gofmt -l internal/amazon` prints nothing, and `go vet ./...` and `go test ./...` pass, including TC-002 of issue #180 with the real browser when it is available.
- Corner cases:
  - if `runBrowserOnProfile` is changed, the fixture sanity check still calls it with the same arguments and gets the same behaviour (15 s from the browser's start, ending at once when the browser exits);
  - no production Go code, frontend source, `go.mod`, `go.sum`, `package.json`, `tests/` or `API_SPECIFICATION.md` changes, so the operator sees no change in the application;
  - the issue #195 and #312 change folders and `doc/todo/issue-312-r1-01.md` are unchanged;
  - no fault injection from TC-002 or TC-004 remains;
  - the gates give the same result when run twice.
