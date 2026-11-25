# No use cases for issue #349

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It makes the findings guard of the analyzer-findings subtest of the fixture build check (`TestFixtureModulesBuild_Issue251_TC001` in `internal/amazon/amazontest/server_dependencies_test.go`) require an analyzer finding in each of the fixture's two files, so that the subtest keeps proving that the fixture build check ignores findings in test files (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test results, test failure messages and code comments:

- **FR-TEST-HYG-086**: the subtest fails with a message naming `cmd/app/main.go` or `cmd/app/main_test.go` when the full vet run's output has no finding position in that file; each file is checked independently and analyzer message wording is not matched.
- **FR-TEST-HYG-087**: the existing `the fixture has no analyzer finding, so the subtest proves nothing` failure is kept, and the findings guard and the fixture build check stay reported independently.
- **FR-TEST-HYG-088**: the comment above the findings guard names both files and says what the subtest proves.
- **FR-TEST-HYG-089**: on a normal run the subtest passes, every subtest keeps its name, result and messages, the diff outside `doc/` is limited to the test file, the formatting, vet and test gates pass, and the operator observes no change in the application.

No production code, API, database, user interface or runtime behaviour changes, and no operator goal or flow is involved. The fixture module, `buildModule` and the repository's own `go vet ./...` gate are unchanged. The test cases and the deterministic gates check these requirements directly.
