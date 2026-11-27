# No use cases for issue #359

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It removes the duplicate TC-003 test of issue #240 (`TestBrowserCacheLog_TC003_SuccessfulClearLogsNothing` in `internal/amazon/browser_cache_log_test.go`), whose cases the issue #233 success-clear tests already check (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only which tests exist, their results and the gates:

- **FR-TEST-HYG-099**: the duplicate TC-003 test, its `// TC-003` comment and every test of the file whose fake browser answers the cache clear successfully are gone.
- **FR-TEST-HYG-100**: TC-001 and TC-002 of the file keep their names, subtests, assertions and results, and the file keeps no import or declaration only the removed test used.
- **FR-TEST-HYG-101**: TC-003 of issue #240 stays proven by the unchanged issue #233 success-clear tests, and the issue #240 test documents are unchanged.
- **FR-TEST-HYG-102**: the diff outside `doc/` only removes lines from the test file, the formatting, vet and test gates pass, and the operator observes no change in the application.

No production code, API, database, user interface or runtime behaviour changes, and no operator goal or flow is involved. The application's behaviour on a successful cache clear (no cache log line, FR-AMZ-HCHK-015 and FR-AMZ-HCHK-016) is unchanged and already covered by UC-AMZ-HCHK-008 (`doc/use-cases/amazon-human-checks/UC-AMZ-HCHK-008-interrupted-amazon-check-leaves-no-downloaded-answer.md`) and UC-AMZ-HCHK-009 (`doc/use-cases/amazon-human-checks/UC-AMZ-HCHK-009-failed-cache-clearing-logged-without-its-reason.md`). The test cases and the deterministic gates check these requirements directly.
