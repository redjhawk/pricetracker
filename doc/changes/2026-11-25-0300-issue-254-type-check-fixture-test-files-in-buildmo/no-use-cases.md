# No use cases for issue #254

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It makes the fixture build check of issue #251 (`buildModule` and `TestFixtureModulesBuild_Issue251_TC001` in `internal/amazon/amazontest/server_dependencies_test.go`) compile the fixtures' test files too (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test code, test comments, test results and test failure messages:

- **FR-TEST-HYG-065**: for every valid fixture, the fixture build check compiles its test files as well as its other Go files, offline, including the two test files of row d.
- **FR-TEST-HYG-066**: a control fixture whose only error is in a test file makes the fixture build check fail with a message naming the undeclared name.
- **FR-TEST-HYG-067**: the fixture build check keeps its other results, messages and subtest names.
- **FR-TEST-HYG-068**: the comments describing the fixture build check say that it compiles the fixtures' test files too.
- **FR-TEST-HYG-069**: the dependency check and `TestListServerDependencies_TC003_FixtureModules` keep their results and messages; outside `doc/`, the diff contains only `internal/amazon/amazontest/server_dependencies_test.go`; `gofmt`, `go vet ./...` and `go test ./...` pass, and the operator sees no change in the application.

No production code, API, database, user interface or runtime behaviour changes, and the operator has no new goal or flow. The test cases and the deterministic gates check these requirements directly.
