# No use cases for issue #185

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It adds automated tests for three behaviours that issue #168 (PR #179) shipped without a test, and formats `internal/amazon/browser_test.go` with `gofmt` (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test code and test results:

- **FR-TEST-HYG-022**: a test proves that a scheduled Amazon check whose morning window has passed is not sent, is redrawn into the next window and is logged as missed.
- **FR-TEST-HYG-023**: a test proves that the upgrade redraw mark gives every Amazon item a window time at start and is then cleared.
- **FR-TEST-HYG-024**: a test proves that empty-body 404 and 410 answers are "listing unavailable" and that 503 and 429 answers are failures with the cause `Amazon returned HTTP <status>`, each with its HTTP status recorded.
- **FR-TEST-HYG-025**: `gofmt -l internal/amazon` lists no file.
- **FR-TEST-HYG-026**: the tests run in the existing unit-test gate, the diff outside `doc/` is limited to `*_test.go` files, and the operator sees no change in the application.

The application behaviour these tests check is unchanged and already described by existing use cases: UC-AMZ-HCHK-001 (scheduled checks in the windows, FR-AMZ-HCHK-004, -005, -009), UC-AMZ-HCHK-004 (first start after the upgrade, FR-AMZ-HCHK-010), and UC-AMZ-HCHK-005 and UC-AMZ-HCHK-006 (statuses not opened as a page, FR-AMZ-HCHK-012). No production code, API, database, user interface or runtime behaviour changes, and no new operator goal or flow is involved. The test cases and the deterministic gates check these requirements directly.
