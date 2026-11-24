# Test cases: make TC-002's fixture sanity check prove the refusal is caused by the profile lock (issue #195)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-040 to FR-TEST-HYG-043). There are no new use cases: see [no-use-cases.md](no-use-cases.md). UC-AMZ-BPROF-002 describes the application behaviour TC-002 checks, and this change does not alter it.

Conventions:

- **TC-002 (Go):** the existing test `TestBrowserProfile_TC002_LockOfAnotherHostDoesNotPreventTheCheck` in `internal/amazon/browser_profile_test.go` (issue #180). It is not a case of this catalogue; the IDs below are this change's own.
- **Fixture sanity check:** `assertBrowserRefusesProfile` in the same file. It starts the test browser directly on the profile that holds a `SingletonLock` naming `otherhost`, waits up to 15 s, kills the browser's process group, and then decides whether the check passed.
- **Lock refusal message:** the text Chromium writes when it refuses a profile that is locked by another computer (for example "The profile appears to be in use by another … process … on another computer"). The exact text matched is a technical choice. It must be specific to that refusal: a generic word that a crash or start error could also print is not enough.
- **Run TC-002:** `go test ./internal/amazon -run 'TestBrowserProfile_TC002' -count=1`. It needs the test browser that the other browser profile tests use.
- The change only touches test code. No production code, API, database or user interface changes. The cases are Go test runs (with temporary fault injections that are reverted afterwards and never committed), a code check and the gates. Whether the stub-executable cases (TC-002, TC-003) also become permanent unit tests in the same file is a technical choice. The cases must hold either way.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-040 | TC-001, TC-004 | UC-AMZ-BPROF-002 |
| FR-TEST-HYG-041 | TC-002, TC-003, TC-005 | — |
| FR-TEST-HYG-042 | TC-004, TC-006 | UC-AMZ-BPROF-002 |
| FR-TEST-HYG-043 | TC-006 | — |

## Cases

### TC-001: The sanity check asserts both conditions

- Covers: FR-TEST-HYG-040
- Level: code check of `internal/amazon/browser_profile_test.go`
- Given the change applied on top of `master`
- When the fixture sanity check is read
- Then it fails the test unless both hold: the product page request count did not change while the browser ran, and the captured browser output (stdout and stderr) contains the lock refusal message
- Corner cases:
  - the message is checked only after the browser's process group is killed and `Wait` has returned, so the output is complete and no write to the buffer races with the read;
  - the message check does not depend on whether the browser exited by itself or was killed when the 15 s wait ended.

### TC-002: Missing refusal message fails the check even if the page was not requested

- Covers: FR-TEST-HYG-041
- Level: unit (Go, `internal/amazon`, stub executables written by the test in a temporary folder, with the same fixture server and locked profile as TC-002). Run temporarily as fault injection, or permanently if the technical specification chooses to.
- Given a profile holding a `SingletonLock` that names `otherhost`
- When the fixture sanity check starts each of these executables instead of the browser:
  1. one that exits at once with a non-zero status and writes nothing (for example `/bin/false`), as a crash or missing library would;
  2. one that writes an unrelated error to stderr (for example `error while loading shared libraries: libnss3.so` or `unknown option --headless=new`) and exits non-zero;
  3. one that writes nothing and sleeps longer than the 15 s wait, as a slow start would;
- Then the check fails in every run. The failure message says that the lock refusal message is missing and contains the captured output: empty in runs 1 and 3, and the unrelated error text in run 2.
- Corner cases:
  - the product page is never requested in these runs, so the old check would have passed; this proves the new condition is not dead;
  - in run 3 the stub is killed after the wait, and no process of it is left running (its process group is gone);
  - an output that holds only a generic word that is not the refusal (for example only `ProcessSingleton` in an unrelated line, if the chosen match is longer) does not pass.

### TC-003: A requested page still fails the check, with its own message

- Covers: FR-TEST-HYG-041
- Level: unit (Go, `internal/amazon`, as TC-002)
- Given the fixture server and a profile
- When the fixture sanity check starts:
  1. a stub executable that requests the product page from the fixture server, writes the lock refusal message, and exits;
  2. a stub executable that requests the product page and writes nothing;
- Then the check fails in both runs. The failure message says that the browser opened the page on a profile locked by another computer, and it contains the captured output.
- Corner cases:
  - run 1 proves that the refusal message alone is not enough to pass (both conditions are required, FR-TEST-HYG-040);
  - when both conditions fail (run 2), the message still names a failed condition and shows the output. Whether it names one or both conditions is a technical choice.

### TC-004: TC-002 passes on a normal run with the real browser

- Covers: FR-TEST-HYG-040, FR-TEST-HYG-042; UC-AMZ-BPROF-002
- Level: integration (Go, `internal/amazon`, real headless browser, `amazontest` fixture server and a real profile on disk)
- Given the change applied and the test browser available
- When TC-002 is run, then run again with `-count=3`
- Then TC-002 passes every time. The fixture sanity check finds no product page request and finds the lock refusal message in the browser output. Then the steps after it run unchanged: the stale lock is written again, the check after the lock of another host and the second check both find the price, the page request carries the cookie of the earlier check, nothing is left (`assertNothingLeft`), and no "browser could not start" is logged.
- Corner cases:
  - after the fixture sanity check no browser process using the profile is running: its process group has been killed and waited for before TC-002 goes on, and `assertNothingLeft` passes at the end;
  - repeated runs give the same result, so the match does not depend on timing or on output buffering;
  - the browser may print the refusal and exit, or print it and keep running until the kill. Both pass.

### TC-005: TC-002 fails when the profile is not locked

- Covers: FR-TEST-HYG-041
- Level: integration (Go, real browser, temporary fault injection in TC-002 that is reverted after the run)
- Given the change applied
- When TC-002 is run once with the first `writeStaleLock(t, options.ProfileDir, "otherhost")` call (the one before the fixture sanity check) removed, so the real browser opens the page on an unlocked profile
- Then TC-002 fails at the fixture sanity check. The message says that the page was opened and contains the browser output. With the temporary change reverted, TC-002 passes again (TC-004).
- Corner cases:
  - this real-browser run also has no refusal message. The failure must still be reported, and it must name at least one failed condition;
  - no temporary change remains in the diff (checked by TC-006).

### TC-006: Only the TC-002 test file changes outside `doc/`, and the gates pass

- Covers: FR-TEST-HYG-042, FR-TEST-HYG-043
- Level: repository check and suite run (deterministic gates: `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` and `git diff master -- internal/amazon/browser_profile_test.go` are listed and the gates run
- Then:
  - the only file listed outside `doc/` is `internal/amazon/browser_profile_test.go`;
  - its changed hunks touch only the fixture sanity check, and any helpers or tests the check needs;
  - the body of TC-002 after the call to the fixture sanity check is unchanged;
  - `gofmt -l internal/amazon` prints nothing, and `go vet ./...` and `go test ./...` pass.
- Corner cases:
  - the other `TestBrowserProfile_*` tests of issue #180 and their helpers (`writeStaleLock`, `stubBrowser`, `killBrowserOnProfile`, `assertNothingLeft`) keep their behaviour;
  - no production Go code, frontend source, `go.mod`, `go.sum`, `package.json`, `tests/` or `API_SPECIFICATION.md` changes, so the operator sees no change in the application;
  - `doc/todo/issue-180-r1-01.md` and the issue #180 change folder are unchanged by the code change;
  - the gates give the same result when run twice.
