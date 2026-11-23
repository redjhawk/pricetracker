# Blocking review comments (round 1)

## R1-01 `internal/amazon/amazontest/server_dependencies_test.go:32`

TestHelperImportPath compares the constant helperImportPath with the same string literal, so it proves nothing about the real package path. If the package is moved or the module renamed (for example to internal/amazon/testsupport), helperImportPath keeps the old value, no line of `go list -deps` ever equals it, and TestServerDoesNotDependOnTestHelper passes even when the server imports the helper. FR-TEST-HYG-012 requires the check never to pass without a real judgement. Fix: derive the expected value from the compiled package, e.g. `if got := reflect.TypeOf(Request{}).PkgPath(); got != helperImportPath { t.Errorf(...) }` (Request is an exported type of amazontest.go). The check then fails when the constant goes stale instead of silently failing open.

Triage: Confirmed: TestHelperImportPath (server_dependencies_test.go:31-35) compares the constant with the same literal, and the fixture modules also hard-code that path, so if the package or module path changes, checkServerDependencies never matches a listing line and TestServerDoesNotDependOnTestHelper passes even when the server imports the helper. The check fails open (FR-TEST-HYG-010/012), and the test meant to guard the constant proves nothing.

