# No use cases for issue #238

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It keeps the guard in TC-005 (`internal/httpapi/amazon_browser_interrupted_test.go`) that fails the test when the Amazon check has already ended at the moment the server is stopped. PR #236 already added this guard. Merging the change closes issue #238 as completed (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test code, test results and the GitHub issue state:

- **FR-TEST-HYG-027**: before the server stop, TC-005 fails at once with the message "the check ended before the server stop: the interruption point was missed" if the check has already ended.
- **FR-TEST-HYG-028**: when the check is still running, TC-005 stops the server and checks the same results as before, and it passes on a normal run.
- **FR-TEST-HYG-029**: merging the change closes issue #238 as completed (which also settles FR-TEST-HYG-019). The diff outside `doc/` is empty or limited to `internal/httpapi/amazon_browser_interrupted_test.go`, `go vet ./...` and `go test ./...` pass, and the operator sees no change in the application.

The application behaviour TC-005 checks is unchanged and already described by UC-AMZ-HCHK-008 (an interrupted Amazon check leaves no downloaded answer, FR-AMZ-HCHK-015). No production code, API, database, user interface or runtime behaviour changes, and no new operator goal or flow is involved. The test cases and the deterministic gates check these requirements directly.
