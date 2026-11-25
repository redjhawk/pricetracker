# No use cases for issue #322

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It only deletes the source-text test `TestBrowserProfile_Issue197_TC005_ProfileUsesTheExplicitForm` and the imports only it uses from `internal/amazon/browser_profile_values_test.go` (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test code and test results:

- **FR-TEST-HYG-061**: no test is named `TestBrowserProfile_Issue197_TC005_ProfileUsesTheExplicitForm`, and no test under `internal/amazon/` reads the source of `browser.go` to compare the text of `profile()`.
- **FR-TEST-HYG-062**: `internal/amazon/browser_profile_values_test.go` no longer imports `go/ast`, `go/format`, `go/parser` or `go/token`, and keeps nothing that only the source-text test used.
- **FR-TEST-HYG-063**: the issue #197 behaviour tests are unchanged and pass, and an edit of `profile()` that keeps its results makes no test fail.
- **FR-TEST-HYG-064**: outside `doc/`, the diff contains only `internal/amazon/browser_profile_values_test.go`; `gofmt`, `go vet ./...` and `go test ./...` pass, and the operator sees no change in the application.

`profile()` and FR-AMZ-BPROF-001–010 are excluded from the change and stay as specified in `doc/specifications/amazon-browser-profile/functional.md`. No production code, API, database, user interface or runtime behaviour changes, and the operator has no new goal or flow. The test cases and the deterministic gates check these requirements directly.
