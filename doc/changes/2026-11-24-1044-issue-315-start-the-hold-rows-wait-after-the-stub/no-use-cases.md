# No use cases for issue #315

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It changes only the hold stub and the hold rows' wait in the fixture check test of issue #195 (`refusalStubBrowser` and `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal` in `internal/amazon/browser_profile_test.go`; see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test code and test results:

- **FR-TEST-HYG-048**: the hold stub gives an output signal after writing its output; a hold row waits for that signal (at most 15 s from the stub's start), then 2 s more before stopping the stub.
- **FR-TEST-HYG-049**: the hold rows keep their meaning: at the end of the wait, the TC-001 refusal stub has written the lock refusal, the TC-002 slow start stub has written nothing, and both are still running without requesting the page.
- **FR-TEST-HYG-050**: a hold row whose stub gives no signal within 15 s fails with a message saying so, stops the stub and leaves no stub process group running.
- **FR-TEST-HYG-051**: on a normal run every row passes with the same expected result and assertions, each hold row still takes about 2 s, a non-hold row still ends as soon as its stub exits, and no stub process group is left running.
- **FR-TEST-HYG-052**: outside `doc/`, the diff contains only `internal/amazon/browser_profile_test.go`; the fixture sanity check of TC-002 and the non-hold rows' wait are unchanged; `gofmt`, `go vet ./...` and `go test ./...` pass, and the operator sees no change in the application.

The application behaviour that TC-002 and its fixture sanity check stand for does not change, and an existing use case already describes it:

- UC-AMZ-BPROF-002 (Amazon checks keep working after the data directory is moved to another machine, whose profile holds a lock naming the old host; FR-AMZ-BPROF-002, FR-AMZ-BPROF-004, FR-AMZ-BPROF-006)

No production code, API, database, user interface or runtime behaviour changes, and the operator has no new goal or flow. The test cases and the deterministic gates check these requirements directly.
