# No use cases for issue #251

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It makes the fixture modules used by the unit tests of the dependency check (issue #244, in `internal/amazon/amazontest`) compile, except those meant to make the listing fail (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test code, test results and test failure messages:

- **FR-TEST-HYG-014**: every valid fixture module builds with `go build ./...`, offline.
- **FR-TEST-HYG-015**: each valid fixture keeps the case it is named after and gives the same dependency check result as before.
- **FR-TEST-HYG-016**: the missing-package, syntax-error and go-command-not-found cases keep failing with the same messages.
- **FR-TEST-HYG-017**: `go vet ./...` and `go test ./...` pass, the diff outside `doc/` is limited to the test code of `internal/amazon/amazontest/`, and the operator sees no change in the application.

No production code, API, database, user interface or runtime behaviour changes, and no operator goal or flow is involved. The test cases and the deterministic gates check these requirements directly.
