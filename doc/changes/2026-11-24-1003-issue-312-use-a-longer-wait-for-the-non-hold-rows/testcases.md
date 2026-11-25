# Test cases: use a longer wait for the non-hold rows of the issue #195 fixture check test (issue #312)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-044 to FR-TEST-HYG-047). There are no new use cases: see [no-use-cases.md](no-use-cases.md). UC-AMZ-BPROF-002 describes the application behaviour that TC-002 of issue #180 and its fixture sanity check stand for, and this change does not alter it.

Conventions:

- **Fixture check test:** `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal` in `internal/amazon/browser_profile_test.go`. Each row starts a stub executable (`refusalStubBrowser`) through `runBrowserOnProfile` on a profile locked by `otherhost`, with the `amazontest` fixture server, and then checks `fixtureCheckFailure`, the captured output, the page request and that the stub's process group is gone.
- **Hold rows:** the rows whose stub has `Hold: true` ("TC-001 refusal, then still running at the end of the wait", "TC-002 slow start"). **Non-hold rows:** the eight other rows; their stubs exit by themselves.
- **Fixture sanity check:** `assertBrowserRefusesProfile`, with its 15 s wait, used by TC-002 of issue #180. This change does not touch it.
- **Run the test:** `go test ./internal/amazon -run '^TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal$' -count=1 -v`. It needs no real browser.
- The change only touches test code. No production code, API, database or user interface changes, and there is no user flow, so there are no integration or e2e cases of the application. The cases are runs of the fixture check test (some with temporary fault injections that are reverted afterwards and never committed), a code check and the gates. The IDs below are this change's own, not those of the rows' names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-044 | TC-001, TC-002 | — |
| FR-TEST-HYG-045 | TC-001, TC-003 | — |
| FR-TEST-HYG-046 | TC-004 | UC-AMZ-BPROF-002 |
| FR-TEST-HYG-047 | TC-005 | UC-AMZ-BPROF-002 |

## Cases

### TC-001: The wait is chosen per row

- Covers: FR-TEST-HYG-044, FR-TEST-HYG-045
- Level: code check of `internal/amazon/browser_profile_test.go`
- Given the change applied on top of `master`
- When the `t.Run` loop of the fixture check test is read
- Then the wait passed to `runBrowserOnProfile` is 15 s for a row whose stub has `Hold: false` and 2 s for a row whose stub has `Hold: true`
- Corner cases:
  - the choice depends only on the row's `Hold` field, not on the row's name or position, so a row added later gets the right wait;
  - the non-hold wait equals the 15 s wait of the fixture sanity check, which the test stands for;
  - the rows table (names, stubs, expected failures) is unchanged.

### TC-002: A slow non-hold stub still gives the row's expected result

- Covers: FR-TEST-HYG-044
- Level: unit (Go, `internal/amazon`, stub executables and fixture server; temporary fault injection reverted after the run)
- Given the change applied, and the stub temporarily made to sleep 5 s before its page request and output when it does not hold (for example through a temporary field or variable read by `TestBrowserProfile_Issue195_RefusalStubProcess`)
- When the fixture check test is run
- Then every non-hold row passes with its expected result: "TC-001 refusal, then exit" passes the check, "TC-003 page requested with the refusal" fails it naming that the page was opened, "TC-003 page requested without output" names both conditions, and the TC-002 rows name the missing refusal message; each captured output equals the stub's stderr, and the page request matches the stub's `Request`
- And, with the same delay and the non-hold wait temporarily set back to 2 s, at least "TC-001 refusal, then exit" and "TC-003 page requested with the refusal" fail (output empty or page not requested), which shows the delay is the cause the longer wait removes
- Corner cases:
  - a delay just over 2 s (for example 3 s) passes as well as 5 s;
  - a non-hold stub that temporarily never exits ends its row after about 15 s, its process group is killed and gone, and the row fails with a message instead of hanging the test;
  - the stub's own HTTP client timeout (10 s) stays below the 15 s wait, so a slow page request is not cut off by the wait;
  - no temporary change remains in the diff (checked by TC-005).

### TC-003: Hold rows keep their 2 s wait

- Covers: FR-TEST-HYG-045
- Level: unit (Go, `internal/amazon`, as TC-002, without fault injection)
- Given the change applied
- When the fixture check test is run with `-v`
- Then "TC-001 refusal, then still running at the end of the wait" passes and "TC-002 slow start" fails the check naming the missing refusal message, as before, and each of the two rows takes about 2 s (well under 15 s) in the `--- PASS` timings
- Corner cases:
  - each hold stub is killed when its 2 s end: its process group is gone after the row;
  - for the hold row that writes the refusal, the captured output still holds the whole message.

### TC-004: A normal run passes unchanged, fast, and leaves no process

- Covers: FR-TEST-HYG-046; UC-AMZ-BPROF-002
- Level: unit (Go, `internal/amazon`, as TC-002, without fault injection)
- Given the change applied, on an unloaded machine
- When the fixture check test is run with `-v`, then again with `-count=3`
- Then all ten rows pass every time with the same expected results and assertions as on `master`; each non-hold row takes less than 2 s, because it ends as soon as its stub exits; the whole test takes about as long as on `master` (about 4 s for the two hold rows, plus the non-hold rows)
- Corner cases:
  - after every row, `syscall.Kill(-pid, 0)` reports `ESRCH`: no process group of a stub is left running;
  - repeated runs give the same results, so the outcome does not depend on timing;
  - the test still passes when run together with the other `TestBrowserProfile_*` tests (`-run 'TestBrowserProfile'`) and with `-race`.

### TC-005: Only the test file changes outside `doc/`, the sanity check is unchanged, and the gates pass

- Covers: FR-TEST-HYG-047; UC-AMZ-BPROF-002
- Level: repository check and suite run (deterministic gates: `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` and `git diff master -- internal/amazon/browser_profile_test.go` are listed and the gates run
- Then:
  - the only file listed outside `doc/` is `internal/amazon/browser_profile_test.go`;
  - its changed hunks are in the fixture check test (and, only if the technical specification chooses the optional output signal for hold rows, the stub and its helpers), and they do not touch `assertBrowserRefusesProfile` or its 15 s wait, `fixtureCheckFailure`, or TC-002 of issue #180;
  - `gofmt -l internal/amazon` prints nothing, and `go vet ./...` and `go test ./...` pass, including TC-002 of issue #180 with the real browser when it is available.
- Corner cases:
  - no production Go code, frontend source, `go.mod`, `go.sum`, `package.json`, `tests/` or `API_SPECIFICATION.md` changes, so the operator sees no change in the application;
  - the issue #195 change folder and `doc/todo/issue-195-r1-01.md` are unchanged;
  - no temporary fault injection from TC-002 remains;
  - the gates give the same result when run twice.
