# No use cases for issue #181

This change is test documentation maintenance with no user-visible behaviour, so it has no use cases.

It corrects two entries of the issue #168 test case document (`doc/changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n/testcases.md`) so that they match FR-AMZ-HCHK-012 and the existing tests (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only that document and the change's diff:

- **FR-TEST-HYG-030**: TC-006 lists HTTP 503, HTTP 403 for the product page and the robot-check page as failures, and no longer lists HTTP 404 or 410.
- **FR-TEST-HYG-031**: TC-004 gains a corner case: a product page answered with HTTP 404 or 410 is "listing unavailable", an Amazon answer, not a failure.
- **FR-TEST-HYG-032**: no other test case or file of the issue #168 change folder changes, the diff outside `doc/` is empty, nothing under `doc/todo/` changes, `go vet ./...` and `go test ./...` pass, and the operator sees no change in the application.

The application behaviour these test cases describe is unchanged and already covered by existing use cases: a 404 or 410 product page is recorded as listing unavailable (UC-AMZ-HCHK-005, UC-AMZ-HCHK-006, FR-AMZ-HCHK-012). No production code, test code, API, database, user interface or runtime behaviour changes, and no new user goal or flow is involved. The test cases and the deterministic gates check these requirements directly.
