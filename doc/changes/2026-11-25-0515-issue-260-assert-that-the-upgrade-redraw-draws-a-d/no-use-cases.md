# No use cases for issue #260

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It makes the upgrade redraw test (`TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce` in `internal/service/amazon_human_checks_issue185_test.go`) fail when two of its three Amazon items get the same drawn time, and updates the TC-002 entry of the issue #185 test case document to match (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test results, test failure messages and test documentation:

- **FR-TEST-HYG-078**: after the first start, the test fails if any two of the three items have the same stored next scheduled check, with a message naming the two items and that time.
- **FR-TEST-HYG-079**: the test keeps its name and its other checks and messages, and passes on the repository as changed.
- **FR-TEST-HYG-080**: TC-002 in the issue #185 test case document states that the three items' times are pairwise different, and that one time given to every item makes the test fail.
- **FR-TEST-HYG-081**: the diff outside `doc/` is limited to the test file, the formatting, vet and test gates pass, and the operator observes no change in the application.

No production code, API, database, user interface or runtime behaviour changes, and no operator goal or flow is involved. The upgrade redraw behaviour itself (FR-AMZ-HCHK-005) is unchanged and already has its use cases. The test cases and the deterministic gates check these requirements directly.
