# No use cases for issue #244

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It replaces the text search for the test helper's import path in the unit tests of `internal/amazon/amazontest` with a check of the server's actual dependencies, run in the existing unit-test gate (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test results and test failure messages:

- **FR-TEST-HYG-009**: the helper's unit tests no longer search the source of `cmd/` and `internal/`, so a non-test import from a package the server does not use fails no test.
- **FR-TEST-HYG-010**: the dependency check fails when any program under `cmd/` depends on the test helper package, directly or indirectly.
- **FR-TEST-HYG-011**: the failure message names `pricefollower.local/internal/amazon/amazontest` and says the server depends on it.
- **FR-TEST-HYG-012**: imports only from test files or non-server packages pass; failing to list the dependencies fails the check with the Go tool's error.
- **FR-TEST-HYG-013**: the check runs in the existing unit-test gate, passes on the changed repository, the issue #208 scratch-file check still runs, and the operator sees no change in the application.

No production code, API, database, user interface or runtime behaviour changes, and no operator goal or flow is involved. The test cases and the deterministic gates check these requirements directly.
