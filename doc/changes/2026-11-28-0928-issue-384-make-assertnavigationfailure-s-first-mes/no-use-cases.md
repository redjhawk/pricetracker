# No use cases for issue #384

This change only edits the test file `internal/amazon/browser_navigation_test.go`. It splits the first check of `assertNavigationFailure` into a result check, which keeps the message `gave %q instead of a failure: %+v`, and a check of the message and second-hand status with its own message (FR-TEST-HYG-112 updated, FR-TEST-HYG-132 to FR-TEST-HYG-135). The only observable outputs are test results and test failure messages. Application behaviour and the user interface do not change, so no use cases are needed.
