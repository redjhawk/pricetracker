# No use cases for issue #253

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It makes rows f and g of the fixture build check (`TestFixtureModulesBuild_Issue251_TC001` in `internal/amazon/amazontest/server_dependencies_test.go`) pass only on a build failure holding their expected text (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test results and test failure messages:

- **FR-TEST-HYG-070**: row f passes only when the build check fails with a message holding `pricefollower.local/internal/nope`; row g only when it holds `main.go`.
- **FR-TEST-HYG-071**: a build failure without the row's expected text fails the subtest with a message naming that text and holding the failure; a fixture that builds still fails the subtest.
- **FR-TEST-HYG-072**: on a normal run rows f and g pass, and the other subtests, the listing test and the dependency check keep their results and messages.
- **FR-TEST-HYG-073**: the diff outside `doc/` is limited to the test file, the formatting, vet and test gates pass, and the operator observes no change in the application.

No production code, API, database, user interface or runtime behaviour changes, and no operator goal or flow is involved. The test cases and the deterministic gates check these requirements directly.
