# No use cases for issue #340

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It limits the `go vet` run of the fixture build check (`buildModule` and `TestFixtureModulesBuild_Issue251_TC001` in `internal/amazon/amazontest/server_dependencies_test.go`) so that only compile errors fail it, and corrects the comments that describe it (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test results, test failure messages and code comments:

- **FR-TEST-HYG-074**: a valid fixture fails the build check only on a compile error, in its non-test or test files; an analyzer finding alone does not fail it.
- **FR-TEST-HYG-075**: the fixtures' test files are still compiled, the subtest "control: a test file uses a name the stub does not declare" keeps its name and passes, and the other subtests keep their names, results and messages.
- **FR-TEST-HYG-076**: the comments say that `go vet` type-checks the packages with their test files and that its analyzer findings do not fail the check.
- **FR-TEST-HYG-077**: the diff outside `doc/` is limited to the test file, the formatting, vet and test gates pass, and the operator observes no change in the application.

No production code, API, database, user interface or runtime behaviour changes, and no operator goal or flow is involved. The repository's own `go vet ./...` gate keeps all its analyzers. The test cases and the deterministic gates check these requirements directly.
