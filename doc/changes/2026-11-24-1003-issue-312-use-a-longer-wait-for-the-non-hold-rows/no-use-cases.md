# No use cases for issue #312

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It changes only the row's wait in the fixture check test of issue #195 (`TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal` in `internal/amazon/browser_profile_test.go`; see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test code and test results:

- **FR-TEST-HYG-044**: every non-hold row waits 15 s instead of 2 s, so a non-hold stub that exits within 15 s, even after more than 2 s, gives the row's expected result.
- **FR-TEST-HYG-045**: every hold row keeps its 2 s wait.
- **FR-TEST-HYG-046**: on a normal run every row passes with the same expected result and assertions, a non-hold row still ends as soon as its stub exits, and no stub process group is left running.
- **FR-TEST-HYG-047**: outside `doc/`, the diff contains only `internal/amazon/browser_profile_test.go`, and the fixture sanity check of TC-002 is unchanged; `gofmt`, `go vet ./...` and `go test ./...` pass, and the operator sees no change in the application.

The application behaviour that TC-002 and its fixture sanity check stand for does not change, and an existing use case already describes it:

- UC-AMZ-BPROF-002 (Amazon checks keep working after the data directory is moved to another machine, whose profile holds a lock naming the old host; FR-AMZ-BPROF-002, FR-AMZ-BPROF-004, FR-AMZ-BPROF-006)

No production code, API, database, user interface or runtime behaviour changes, and the operator has no new goal or flow. The test cases and the deterministic gates check these requirements directly.
