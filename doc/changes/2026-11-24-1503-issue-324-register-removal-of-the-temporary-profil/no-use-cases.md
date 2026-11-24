# No use cases for issue #324

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It changes only the subtest "TC-003 temporary profile" of `TestBrowserProfile_Issue197_TC003_PreparedProfileReturnsItsPath` in `internal/amazon/browser_profile_values_test.go` (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test code and test results:

- **FR-TEST-HYG-053**: a non-empty temporary profile path given by `profileOf("")` no longer exists once the subtest has ended, including when it stops early on a fatal failure.
- **FR-TEST-HYG-054**: the subtest still calls the collector's cleanup and checks that this call alone removed the temporary profile; the added removal runs only after the checks, and the failure message "the cleanup did not remove the temporary profile" and the other checks are unchanged.
- **FR-TEST-HYG-055**: on an empty path the subtest removes nothing, and it never removes anything other than the temporary profile it was given.
- **FR-TEST-HYG-056**: on a normal run the subtest passes; outside `doc/`, the diff contains only `internal/amazon/browser_profile_values_test.go`; `gofmt`, `go vet ./...` and `go test ./...` pass, and the operator sees no change in the application.

How the collector creates and removes the temporary profile is excluded from the change and stays as specified in `doc/specifications/amazon-browser-profile/functional.md`. No production code, API, database, user interface or runtime behaviour changes, and the operator has no new goal or flow. The test cases and the deterministic gates check these requirements directly.
