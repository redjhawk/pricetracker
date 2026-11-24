# No use cases for issue #182

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It corrects one automated test and adds another for scheduling behaviour that issue #168 (PR #179) shipped (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test code and test results:

- **FR-TEST-HYG-033**: the TC-032 subtest "Refresh price running" sets the item's due time to a past time of the morning window while the Refresh price is still in progress, so the scheduled checks find the item due. The subtest proves that only one check is made and that the next scheduled check is in a later window.
- **FR-TEST-HYG-034**: the TC-032 subtest "first check after add running" keeps its case and its pass condition.
- **FR-TEST-HYG-035**: a new test proves that an Amazon item stopped at its 07:00 scheduled time and restarted by a Refresh price at 07:30 is not checked again by the 07:31 scheduled checks, and gets its next scheduled check in the evening window of the same day.
- **FR-TEST-HYG-036**: the tests run in the existing unit-test gate and `gofmt -l internal/service` lists no file. Outside `doc/`, the diff contains only `internal/service/*_test.go` files, and the operator sees no change in the application.

The application behaviour these tests check does not change, and existing use cases already describe it:

- UC-AMZ-HCHK-001 (scheduled checks in the windows; stopped items are skipped at their scheduled time; FR-AMZ-HCHK-008, FR-AMZ-STOP-004)
- UC-AMZ-HCHK-003 (Refresh price starts at once, does not change the scheduled checks, and restarts a stopped item on an Amazon answer; FR-AMZ-HCHK-007, FR-AMZ-STOP-006)
- UC-ID-02 (Refresh price on the details page)

No production code, API, database, user interface or runtime behaviour changes, and the operator has no new goal or flow. The test cases and the deterministic gates check these requirements directly.
