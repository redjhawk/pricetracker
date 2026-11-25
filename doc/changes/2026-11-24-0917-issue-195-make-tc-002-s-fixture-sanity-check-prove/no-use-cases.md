# No use cases for issue #195

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It tightens the fixture sanity check of TC-002 in `internal/amazon/browser_profile_test.go` (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test code and test results:

- **FR-TEST-HYG-040**: the fixture sanity check passes only when the browser did not request the product page and the browser output holds the lock refusal message.
- **FR-TEST-HYG-041**: the fixture sanity check fails when the lock refusal message is missing (crash, missing library, unknown option, slow start) or when the page is requested; each failure message says which condition failed and holds the browser output.
- **FR-TEST-HYG-042**: on a normal run TC-002 passes, its later steps and assertions are unchanged, and no browser process of the profile is left running.
- **FR-TEST-HYG-043**: outside `doc/`, the diff contains only `internal/amazon/browser_profile_test.go`; `gofmt`, `go vet ./...` and `go test ./...` pass, and the operator sees no change in the application.

The application behaviour TC-002 checks does not change, and an existing use case already describes it:

- UC-AMZ-BPROF-002 (Amazon checks keep working after the data directory is moved to another machine, whose profile holds a lock naming the old host; FR-AMZ-BPROF-002, FR-AMZ-BPROF-004, FR-AMZ-BPROF-006)

No production code, API, database, user interface or runtime behaviour changes, and the operator has no new goal or flow. The test cases and the deterministic gates check these requirements directly.
