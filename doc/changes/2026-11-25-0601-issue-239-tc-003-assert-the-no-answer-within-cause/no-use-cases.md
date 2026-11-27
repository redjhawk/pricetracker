# No use cases for issue #239

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It tightens the cache log line checks of TC-003 of issue #233 (`TestBrowserInterrupted_TC003_UnconfirmedClearIsBoundedAndLogged` in `internal/amazon/browser_interrupted_test.go`) and updates that test case's entry in the issue #233 test case document (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test results, test failure messages and test documentation:

- **FR-TEST-HYG-078**: in the never-answered subtests, TC-003 fails when the cache log line does not contain `no answer within`.
- **FR-TEST-HYG-079**: in every subtest, TC-003 fails when a line about the cache contains `context canceled` or `context deadline exceeded`.
- **FR-TEST-HYG-080**: the TC-003 entry of the issue #233 test case document states these expected results; its other content is unchanged.
- **FR-TEST-HYG-081**: TC-003 and the other tests of the file keep their subtests, names and results on a normal run, the diff outside `doc/` is limited to the test file, the formatting, vet and test gates pass, and the operator observes no change in the application.

No production code, API, database, user interface or runtime behaviour changes, and no operator goal or flow is involved. The application's cache clear, its 5 s bound and its log line (FR-AMZ-HCHK-015) are unchanged and already covered by UC-AMZ-HCHK-008 (`doc/use-cases/amazon-human-checks/UC-AMZ-HCHK-008-interrupted-amazon-check-leaves-no-downloaded-answer.md`). The test cases and the deterministic gates check these requirements directly.
