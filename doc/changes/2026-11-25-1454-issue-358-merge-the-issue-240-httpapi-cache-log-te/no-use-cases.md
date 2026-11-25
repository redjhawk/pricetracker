# No use cases for issue #358

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It adds the no-cache-clear-failure-line check after each owner's Refresh price to the existing real-browser test `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile` (`internal/httpapi/amazon_browser_interrupted_test.go`), deletes the duplicate test file `internal/httpapi/amazon_browser_cache_log_test.go`, and updates the TC-004 entry of the issue #240 test case document and technical section to match (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test results, test failure messages and test documentation:

- **FR-TEST-HYG-094**: after each owner's Refresh price, TC-005 fails if a cache-clear failure line (neutral or old wording) was logged, with a message naming the item and holding the lines.
- **FR-TEST-HYG-095**: TC-005 keeps its name and its other steps, checks and messages, and passes on a normal run.
- **FR-TEST-HYG-096**: the duplicate cache-log test and its helper `assertNoCacheLine` no longer exist, so the `internal/httpapi` tests start one real-Chromium scenario fewer.
- **FR-TEST-HYG-097**: the TC-004 entry of the issue #240 test case document and the TC-004 row of its technical section name TC-005 as covering TC-004 and no longer name the deleted test or file.
- **FR-TEST-HYG-098**: the diff outside `doc/` is limited to the two test files, the formatting, vet and test gates pass, and the operator observes no change in the application.

No production code, API, database, user interface or runtime behaviour changes, and no operator goal or flow is involved. The cache-clear logging behaviour itself (FR-AMZ-HCHK-016) is unchanged and already has its use cases. The test cases and the deterministic gates check these requirements directly.
