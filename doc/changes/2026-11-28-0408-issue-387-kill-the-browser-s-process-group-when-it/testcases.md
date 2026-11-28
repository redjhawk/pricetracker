# Test cases: kill the browser's process group when it exits before holding the profile (issue #387)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-123 to FR-TEST-HYG-126, FR-TEST-HYG-117 updated). There are no use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **The helper:** `KillBrowserOnProfile(t, profile)` of `internal/amazon/amazontest`. **Early-exit failure:** its failure with exactly `the browser exited before holding the profile`. **Browser group:** as defined in the terms of the functional specification (issue #387).
- **Recording TB**, **stub browser** (`writeStub`, set through `PRICEFOLLOWER_BROWSER_PATH` with `t.Setenv`, reading the profile from `--user-data-dir=`) and **with Chromium:** as in the issue #196 test cases, reused from `internal/amazon/amazontest/browser_profile_test.go`.
- **Background-child stub:** a stub browser that starts `sleep 60 "$profile" &` (a background child in its own group, not `setsid`, whose command line holds the profile path), records the child's PID in a file of `t.TempDir()`, and then exits without creating `SingletonLock`. A `t.Cleanup` kills the recorded PID if it still runs, so a failing run leaves no process behind.
- **Fault injection:** a temporary edit made only for the run of a case, reverted afterwards and never committed.
- Levels: TC-001 to TC-004 are `unit` Go tests in the helper package (TC-004 with real Chromium). TC-005 and TC-006 run the existing tests that use the helper, those of `internal/service` and `internal/httpapi` as `integration` tests (real SQLite, service, HTTP handlers, real Chromium). TC-007 is a repository and gate check. The change only touches a test helper: there is no user flow and no user interface change, so there is no e2e case. The IDs below are this change's own, not those in existing test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-117 | TC-001, TC-002, TC-003, TC-004, TC-005 | — |
| FR-TEST-HYG-123 | TC-001, TC-002, TC-003 | — |
| FR-TEST-HYG-124 | TC-001, TC-002 | — |
| FR-TEST-HYG-125 | TC-001, TC-002, TC-003 | — |
| FR-TEST-HYG-126 | TC-004, TC-005, TC-006, TC-007 | — |

## Cases

### TC-001: the early exit kills a background child left in the browser group

- Covers: FR-TEST-HYG-117, FR-TEST-HYG-123, FR-TEST-HYG-124, FR-TEST-HYG-125
- Level: unit (Go, helper package; a new row or subtest of `TestKillBrowserOnProfile_Issue196_TC006_ExitBeforeHolding`, or a new test of the same package)
- Given a new profile folder and the background-child stub as browser
- When the helper is called through the recording TB
- Then it gives the early-exit failure, exactly `the browser exited before holding the profile`, in less than 5 s; and within 5 s after the failure `BrowserProcesses(profile)` returns no entry, in particular none for the recorded child PID
- Corner cases:
  - the child's command line contains the profile path, so it is listed by `BrowserProcesses(profile)` before the helper returns (checked through the PID file, so the test does not pass because the child was never started);
  - both with a stale `SingletonLock` (`otherhost-12345`) in the profile before the call and with none: same message, same timing, no process left, and `SingletonLock` absent afterwards;
  - the failure message is the whole message: no suffix, no listed processes.

### TC-002: the early exit without a leftover process is unchanged

- Covers: FR-TEST-HYG-117, FR-TEST-HYG-123, FR-TEST-HYG-124, FR-TEST-HYG-125
- Level: unit (Go, helper package; the existing rows of `TestKillBrowserOnProfile_Issue196_TC006_ExitBeforeHolding`)
- Given each existing row: `/bin/true` with a stale lock, a stub exiting with status 3 with a stale lock, `/bin/true` on a profile without `SingletonLock`
- When the helper is called through the recording TB
- Then each row still gives exactly the early-exit failure in less than 5 s, `SingletonLock` is absent afterwards, and the rows keep their names and checks
- Corner cases:
  - the browser group is already empty when the kill is sent (error ESRCH): the kill is harmless, the helper neither hangs nor fails with another message;
  - running the test twice in a row (`-count=2`) gives the same result each time.

### TC-003: the new case fails without the group kill on the early exit

- Covers: FR-TEST-HYG-123, FR-TEST-HYG-125
- Level: unit (Go, helper package, fault injection)
- Given the helper with the group kill removed from its early-exit branch (fault injection)
- When `go test -count=1 -run '^TestKillBrowserOnProfile_' ./internal/amazon/amazontest/` runs
- Then the TC-001 case fails, reporting that `BrowserProcesses(profile)` still lists the background child after 5 s; the TC-002 rows still pass; after the run and the cleanup, no `sleep 60` process on the test's profile is left
- Corner cases:
  - with the fault reverted, the same command passes (TC-001 and TC-002).

### TC-004: the other branches of the helper are unchanged

- Covers: FR-TEST-HYG-117, FR-TEST-HYG-126
- Level: unit (Go, helper package; existing tests, TC-005 of issue #196 with real Chromium)
- Given the existing tests `TestKillBrowserOnProfile_Issue196_TC005_LeavesTheLock`, `TestKillBrowserOnProfile_Issue196_TC007_NeverHolding` and `TestKillBrowserOnProfile_Issue196_TC008_LeftoverAndMissingLock`, with Chromium
- When `go test -count=1 -run '^TestKillBrowserOnProfile_' -v ./internal/amazon/amazontest/` runs
- Then they pass unchanged: with Chromium the helper returns with `SingletonLock` present and no entry in `BrowserProcesses(profile)`; a browser that never creates `SingletonLock` fails with `the browser did not create SingletonLock in the profile` after 30 s to 40 s and leaves no process; a process escaping the group fails with `browser processes left after the kill: ` and its entry; a lock removed before the kill fails with `precondition: the killed browser left no SingletonLock: ` and the error

### TC-005: the tests that use the helper pass as before

- Covers: FR-TEST-HYG-117, FR-TEST-HYG-126
- Level: unit and integration (Go; `internal/amazon` with real Chromium; `internal/service` and `internal/httpapi` with real SQLite, service, HTTP handlers and real Chromium)
- Given the tests of `internal/amazon`, `internal/service` and `internal/httpapi` that call `KillBrowserOnProfile` (the issue #180 profile tests)
- When `go test -count=1 ./internal/amazon/... ./internal/service/... ./internal/httpapi/...` runs with Chromium
- Then they pass, with the same names, subtests and assertions as on `master`

### TC-006: the application is unchanged

- Covers: FR-TEST-HYG-126
- Level: integration (Go, repository check)
- Given the change's diff
- When the dependency check (FR-TEST-HYG-010) runs as part of `go test ./...`
- Then it passes: the server's dependencies still exclude `internal/amazon/amazontest`, so the operator observes no change in the application

### TC-007: the diff stays within the helper files and the gates pass

- Covers: FR-TEST-HYG-126
- Level: repository and gate check
- Given the change branch
- When `git diff --name-only origin/master`, `gofmt -l internal`, `go vet ./...` and `go test ./...` run
- Then the diff outside `doc/` lists only files among `internal/amazon/amazontest/browser_profile.go` and `internal/amazon/amazontest/browser_profile_test.go`; `gofmt -l internal` prints nothing; `go vet ./...` and `go test ./...` succeed
