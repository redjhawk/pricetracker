# Test cases: move the duplicated browser-profile test helpers into `internal/amazon/amazontest` (issue #196)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-115 to FR-TEST-HYG-122). There are no use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **The helper package:** `internal/amazon/amazontest`. **The profile handler:** its exported HTTP handler that serves the profile fixture pages (FR-TEST-HYG-118); the technical specification names it.
- **Lock marks**, **stale lock of a host**, **profile fixture pages**, **local copies** and **issue #180 profile tests:** as defined in the terms of the functional specification (issue #196).
- **Expected links of `<host>`:** `SingletonLock` → `<host>-12345`, `SingletonSocket` → `/tmp/org.chromium.Chromium.gone/SingletonSocket`, `SingletonCookie` → `12345`, each a symbolic link (`os.Lstat` reports `ModeSymlink`, `os.Readlink` gives the target).
- **Recording TB:** the `recordingTB` of `internal/amazon/amazontest/amazontest_test.go` (or an equivalent), run in its own goroutine, so a helper's failure is recorded with its message instead of failing the test; `Fatal`/`Fatalf` stop that goroutine as `testing.T` does.
- **Stub browser:** an executable shell script in `t.TempDir()`, used by setting `PRICEFOLLOWER_BROWSER_PATH` to it with `t.Setenv`; it reads the profile from its `--user-data-dir=` argument, so its command line contains the profile path. Every process a stub starts is stopped in `t.Cleanup`.
- **With Chromium:** `amazontest.Browser(t)` finds a browser; with `CI` set, the cases cannot skip.
- **Fault injection:** a temporary edit or environment setting made only for the run of a case, reverted afterwards and never committed. "On `master`" means the same injection applied to the `master` tree.
- Levels: the helper cases (TC-001 to TC-009) are `unit` Go tests in the helper package (TC-005 with real Chromium). TC-012 to TC-014 run the existing tests: those of `internal/amazon` with real Chromium, those of `internal/service` and `internal/httpapi` as `integration` tests (real SQLite, service, HTTP handlers, real Chromium). The other cases are code, repository and gate checks. The change only touches test code: there is no user flow and no user interface change, so there is no e2e case. The IDs below are this change's own, not those in existing test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-115 | TC-001, TC-002, TC-003, TC-015 | — |
| FR-TEST-HYG-116 | TC-004, TC-015 | — |
| FR-TEST-HYG-117 | TC-005, TC-006, TC-007, TC-008, TC-015 | — |
| FR-TEST-HYG-118 | TC-009, TC-015 | — |
| FR-TEST-HYG-119 | TC-010 | — |
| FR-TEST-HYG-120 | TC-011, TC-012 | — |
| FR-TEST-HYG-121 | TC-012, TC-013, TC-014 | — |
| FR-TEST-HYG-122 | TC-015 | — |

## Cases

### TC-001: `StaleLock` creates a missing profile folder holding the stale lock

- Covers: FR-TEST-HYG-115
- Level: unit (Go, helper package)
- Given a path `<tmp>/a/profile` whose folders do not exist
- When `StaleLock(t, path, "somehost")` is called
- Then the folder exists, the test has not failed, and the folder holds exactly the expected links of `somehost`
- Corner cases:
  - the host is used as given: `StaleLock(t, path2, "otherhost")` gives `SingletonLock` → `otherhost-12345`, the other two targets being the same as for `somehost`;
  - the link targets need not exist (`/tmp/org.chromium.Chromium.gone` is never created).

### TC-002: `StaleLock` replaces existing lock marks and keeps the rest of the profile

- Covers: FR-TEST-HYG-115
- Level: unit (Go, helper package)
- Given, one at a time, a profile folder holding:
  1. regular files named `SingletonLock`, `SingletonSocket` and `SingletonCookie`;
  2. a non-empty directory `SingletonLock` and the links of another host (`StaleLock(t, dir, "firsthost")` called before);
  3. no lock mark at all;
  and, in every case, an unrelated file `Default/Preferences` and a file `SingletonLockExtra`
- When `StaleLock(t, dir, "somehost")` is called
- Then the test has not failed, the three lock marks are the expected links of `somehost` (no regular file or directory remains under those names), and `Default/Preferences` and `SingletonLockExtra` are unchanged
- Corner cases:
  - repeated call: calling `StaleLock(t, dir, "somehost")` twice in a row passes and leaves the same links (no "file exists" error);
  - a dangling `SingletonLock` link is replaced like any other mark.

### TC-003: A `StaleLock` error fails the test and names the error

- Covers: FR-TEST-HYG-115
- Level: unit (Go, helper package; recording TB)
- Given, one at a time: a profile path whose parent is a regular file (`<tmp>/file/profile`), so the folder cannot be created; and a profile folder made read-only (mode `0o500`) holding a regular `SingletonLock` file, so the marks cannot be removed or created (skipped when running as root, where the mode does not deny)
- When `StaleLock` is called with the recording TB
- Then it records a failure whose message holds the error's text (for example `not a directory` or `permission denied`, and the path concerned), and it stops at that error
- Corner cases:
  - the read-only folder is made writable again in `t.Cleanup`, so `t.TempDir()` can be removed;
  - nothing is left half-written in the parent of the first path.

### TC-004: `BrowserProcesses` lists exactly the processes whose command line holds the profile

- Covers: FR-TEST-HYG-116
- Level: unit (Go, helper package; real processes)
- Given a unique path `<tmp>/profile-<random>` that no running command line contains
- When `BrowserProcesses(path)` is called before, during and after a process started with `sleep 30 --marker=<path>` (or another command whose arguments include the path) runs
- Then:
  - before: no entry (`len == 0`);
  - during: exactly one entry, equal to `<pid>: <command line>`, where `<pid>` is the started process's number and the command line shows its arguments separated by spaces and holds the path (no NUL byte in the entry);
  - after the process is killed and waited for: no entry
- Corner cases:
  - a second process started at the same time with a different path in its arguments is not listed for the first path, and gets its own single entry for its path;
  - the test process itself, whose command line does not hold the path, is not listed;
  - an unreadable process list gives no entry: checked by reading the code (an error reading `/proc` returns an empty result, and a process whose command line cannot be read, for example one that ended during the scan, is skipped), as `/proc` cannot be made unreadable in a test.

### TC-005: `KillBrowserOnProfile` leaves the lock of a killed browser and no process

- Covers: FR-TEST-HYG-117
- Level: unit (Go, helper package; with Chromium)
- Given, one at a time: a new empty profile folder; a profile folder holding the stale lock of `otherhost` (`StaleLock` called before); a profile on which `KillBrowserOnProfile` already ran once
- When `KillBrowserOnProfile(t, profile)` is called
- Then it returns without failing the test, `os.Lstat(profile/SingletonLock)` succeeds, and `BrowserProcesses(profile)` gives no entry
- Corner cases:
  - the existing `SingletonLock` (of another host, or of the previously killed browser) is removed before the start, so the browser holds the profile instead of refusing it;
  - repeated action: the third case shows two successive kills on the same profile both succeed;
  - the browser is the one given by `Browser(t)`; no executable argument is passed.

### TC-006: A browser that exits before holding the profile fails at once

- Covers: FR-TEST-HYG-117
- Level: unit (Go, helper package; recording TB; `PRICEFOLLOWER_BROWSER_PATH=/bin/true`)
- Given `PRICEFOLLOWER_BROWSER_PATH` set to `/bin/true`, and a profile folder holding a stale `SingletonLock`
- When `KillBrowserOnProfile` is called with the recording TB
- Then it records a failure with the message `the browser exited before holding the profile`, within 5 s (well before the 30 s wait ends), and the stale `SingletonLock` is gone (it was removed before the start)
- Corner cases:
  - a stub browser that exits with a non-zero status before creating the lock gives the same message;
  - a profile with no `SingletonLock` at all is not an error for the removal step.

### TC-007: A browser that never holds the profile is killed after 30 s and fails

- Covers: FR-TEST-HYG-117
- Level: unit (Go, helper package; recording TB; stub browser)
- Given a stub browser that keeps running (for example `exec sleep 120`) and never creates `SingletonLock`
- When `KillBrowserOnProfile` is called with the recording TB
- Then, after about 30 s (at least 30 s, at most 40 s), it records a failure with the message `the browser did not create SingletonLock in the profile`, and afterwards `BrowserProcesses(profile)` gives no entry, so the stub's process group was killed
- Corner cases:
  - the case runs for about 30 s, as the wait it proves is 30 s (it sets the environment with `t.Setenv`, so it cannot run in parallel);
  - the stub started in its own process group: a child it starts in the background is killed too.

### TC-008: Leftover processes and a missing lock after the kill fail with their messages

- Covers: FR-TEST-HYG-117
- Level: unit (Go, helper package; recording TB; stub browsers)
- Given, one at a time:
  1. a stub browser that creates `SingletonLock` in its profile, then starts with `setsid` a process outside its group whose arguments hold the profile (for example `sleep 60 <profile>`), and keeps running;
  2. a stub browser that creates `SingletonLock`, removes it about 100 ms later (within the 500 ms wait), and keeps running
- When `KillBrowserOnProfile` is called with the recording TB
- Then:
  1. after the kill and the 5 s wait, it records a failure whose message starts with `browser processes left after the kill: ` and holds the escaped process's entry (`<pid>: ` followed by its command line);
  2. it records a failure whose message starts with `precondition: the killed browser left no SingletonLock: ` followed by the `Lstat` error (`no such file or directory`)
- Corner cases:
  - in row 1 the escaped process is stopped in `t.Cleanup`, so no process survives the test;
  - in row 1 the wait lasts at most about 5 s after the kill, not longer;
  - in row 2 the 500 ms wait after the lock appears is observed (the removal inside it is seen after the kill).

### TC-009: The profile handler serves the profile fixture pages and 404 elsewhere

- Covers: FR-TEST-HYG-118
- Level: unit (Go, helper package; `httptest`)
- Given the profile handler served by `amazontest.NewServer` (or `httptest.NewServer`)
- When `GET` requests are sent to `/dp/B0ANY00001`, `/dp/B0OTHER002`, `/gp/aod/ajax?asin=B0ANY00001`, `/gp/aod/ajax/B0ANY00001`, `/static/x.css`, `/static/js/a.js`, `/other`, `/` and `/gp/aod`
- Then:
  - both `/dp/` requests give HTTP 200, the content type `text/html; charset=utf-8`, the body equal to `amazontest.Fixture(t, "price-found.html")`, and a `Set-Cookie` header with `session-id=fixture-session-180`, `Path=/` and `Max-Age=3600`;
  - both `/gp/aod/ajax` requests give HTTP 200, the same content type and the body equal to `amazontest.Fixture(t, "aod-none.html")`, without a `session-id` cookie;
  - both `/static/` requests give HTTP 204 with an empty body;
  - `/other`, `/` and `/gp/aod` give HTTP 404
- Corner cases:
  - repeated requests to the same path give the same answer (the handler keeps no state);
  - the handler can be served by two servers at the same time (TC-012 uses it in `internal/service` and `internal/httpapi`).

### TC-010: The local copies no longer exist

- Covers: FR-TEST-HYG-119
- Level: repository check
- Given the change applied
- When `grep -rnE 'func (writeStaleLock|leaveStaleLock|leaveOtherHostLock|killBrowserOnProfile|killBrowser|browserProcesses|profileProcesses|browserProfileServer)\(' --include='*.go' .`, `grep -rln '"/proc"\|org.chromium.Chromium.gone' --include='*.go' internal cmd` and `grep -nE 'SetCookie|aod-none.html|"/gp/aod/ajax|"/static/' internal/service/amazon_browser_profile_test.go internal/httpapi/amazon_browser_profile_test.go` are run
- Then the first and third print nothing, and the second lists only files of `internal/amazon/amazontest/`
- Corner cases:
  - no other identifier is declared under one of those names (for example as a variable or method): `grep -rnwE 'writeStaleLock|leaveStaleLock|leaveOtherHostLock|killBrowserOnProfile|killBrowser|browserProcesses|profileProcesses|browserProfileServer' --include='*.go' .` prints nothing;
  - `profileFixtureHandler` and `pageCookieSent` of `internal/amazon`, and `profileLockNames`, `leaveLockDirectory`, `profileCollector` and `browserProfileOptions`, may stay (excluded by the specification) and are not reported.

### TC-011: The tests call the shared helpers where they used the local copies

- Covers: FR-TEST-HYG-120
- Level: code check (`git diff origin/master -- internal/amazon/browser_profile_test.go internal/amazon/browser_profile_values_test.go internal/amazon/browser_test.go internal/service/amazon_browser_profile_test.go internal/httpapi/amazon_browser_profile_test.go`)
- Given the change applied
- When the diff is read
- Then:
  - each removed call of `writeStaleLock(t, X, "otherhost")`, `leaveStaleLock(t, X)` or `leaveOtherHostLock(t, X)` is replaced, at the same place, by `amazontest.StaleLock(t, X, "otherhost")` with the same `X`;
  - each removed call of `killBrowserOnProfile(t, …, X)` or `killBrowser(t, X)` is replaced, at the same place, by `amazontest.KillBrowserOnProfile(t, X)` with the same `X`;
  - `assertNothingLeft` calls `amazontest.BrowserProcesses` and keeps its message, its 5 s wait and its goroutine check;
  - the server of `TestAmazonBrowserProfile_TC010_DataDirectoryMovedToAnotherMachine` serves the profile handler (through `amazontest.NewServer`);
  - `profileServer` keeps its parameters `(t, blocked)` and results `(server, arrived, release)`, still reports each `/dp/` request on `arrived` and holds the page of `blocked` until release or the request's end, then hands the request to the profile handler
- Corner cases:
  - in `internal/amazon` the former kill calls passed `options.ExecPath`; the options of those tests take it from `amazontest.Browser(t)`, so the shared helper starts the same browser;
  - no call site is added or removed beyond these replacements, and no assertion or step of the tests changes (the diff touches only helper declarations, these calls and the imports `gofmt` and the compiler require).

### TC-012: The issue #180 profile tests pass with the same tests and subtests

- Covers: FR-TEST-HYG-120, FR-TEST-HYG-121
- Level: integration (Go; `internal/amazon` with real Chromium; `internal/service` and `internal/httpapi` with real SQLite, service, HTTP handlers and real Chromium)
- Given the change applied, no fault injection, Chromium available
- When `go test -count=1 -run 'TestBrowserProfile_|TestAmazonBrowserProfile_' -v ./internal/amazon/ ./internal/service/ ./internal/httpapi/` runs, then the same command with `-count=2`
- Then every test and subtest passes, and the list of `=== RUN` names is the same as on `master` (including `TestAmazonBrowserProfile_TC007_FailureIsLoggedCountedAndStopsItems` with "five failures stop the items" and "a stale lock after four failures is recovered", `TestAmazonBrowserProfile_TC009_ScheduledCheckAndRefreshAfterKilledBrowsers` with its three subtests, `TestAmazonBrowserProfile_TC010_DataDirectoryMovedToAnotherMachine` with its two subtests, and the `TestBrowserProfile_*` tests of `browser_profile_test.go` and `browser_profile_values_test.go`)
- Corner cases:
  - "Refresh price during a blocked scheduled check" still passes, which shows that `profileServer` holds the blocked page until release and reports its arrival;
  - `TestBrowserProfile_TC004_SuccessiveAbruptEndsRecoverEveryTime` (repeated kills on one profile) passes;
  - repeated runs (`-count=2`) pass each time: no browser process or lock from an earlier run disturbs the next;
  - `go test -list . ./internal/amazon/ ./internal/service/ ./internal/httpapi/` gives the same test names as on `master`.

### TC-013: The other tests that use `assertNothingLeft` still pass

- Covers: FR-TEST-HYG-121
- Level: unit (Go, `internal/amazon`; with Chromium)
- Given the change applied, Chromium available
- When `go test -count=1 -run 'TestBrowserCollector_|TestBrowserRetrieval_' -v ./internal/amazon/` runs
- Then every test and subtest passes, with the same `=== RUN` names as on `master`
- Corner cases:
  - `TestBrowserRetrieval_TC006_BrowserStartFailureLogsItsReason` and `TestBrowserCollector_TC007_ErrorsEndTheCheck`, which call `assertNothingLeft` with an executable path instead of a profile, pass;
  - fault injection: making `assertNothingLeft` run while a process holding the profile path runs (for example by starting `sleep 30 <profile>` just before the call in one test) makes that test fail with `browser processes left after the check: ` followed by `<pid>: ` and the command line, as on `master`.

### TC-014: The service tests that kill a browser fail at once when it exits first

- Covers: FR-TEST-HYG-121
- Level: integration (Go, `internal/service`; real SQLite; fault injection `PRICEFOLLOWER_BROWSER_PATH=/bin/true`)
- Given the change applied and `PRICEFOLLOWER_BROWSER_PATH=/bin/true`
- When `go test -count=1 -run '^TestAmazonBrowserProfile_TC009_ScheduledCheckAndRefreshAfterKilledBrowsers$' -v ./internal/service/` runs
- Then the subtests fail at their kill step with `the browser exited before holding the profile`, each within a few seconds; on `master` the same injection makes each wait 30 s and fail with `the browser did not create SingletonLock in the profile`
- Corner cases:
  - the failure message of leftover processes in `internal/service` now comes from `KillBrowserOnProfile` and lists `<pid>: <command line>` entries (proved by TC-008 row 1, as the service calls that helper: TC-011), where `master` listed only numbers;
  - no fault injection remains in the diff (checked by TC-015).

### TC-015: The helpers are documented, the diff is limited and the gates pass

- Covers: FR-TEST-HYG-115, FR-TEST-HYG-116, FR-TEST-HYG-117, FR-TEST-HYG-118, FR-TEST-HYG-122
- Level: repository check and suite run (deterministic gates)
- Given the change applied on top of `master`
- When `go doc pricefollower.local/internal/amazon/amazontest` and `git diff --name-only origin/master -- . ':!doc'` are run, and `gofmt -l internal`, `go vet ./...` and `go test ./...` run
- Then:
  - `go doc` lists `StaleLock`, `BrowserProcesses`, `KillBrowserOnProfile` and the profile handler, each with a doc comment;
  - the diff lists only files of `internal/amazon/amazontest/`, `internal/amazon/browser_profile_test.go`, `internal/amazon/browser_profile_values_test.go`, `internal/amazon/browser_test.go`, `internal/service/amazon_browser_profile_test.go` and `internal/httpapi/amazon_browser_profile_test.go`;
  - `gofmt -l internal` prints nothing; `go vet ./...` and `go test ./...` pass, including the dependency check of FR-TEST-HYG-010 (the server binary's dependencies do not include the helper package)
- Corner cases:
  - production Go code, `cmd/`, frontend source, `tests/`, `go.mod`, `go.sum`, `package.json` and `API_SPECIFICATION.md` are unchanged, so the operator observes no change in the application;
  - the new helpers add no import of `internal/service`, `internal/httpapi` or `internal/store` to the helper package (no import cycle, `go vet ./...` passes);
  - the existing helpers of the helper package (`Browser`, `Fixture`, `NewServer`, `AssertNoDownload` and the others) keep their tests passing;
  - no fault injection of TC-003 to TC-014 remains, and the gates give the same result when run twice.
