# Issue #340 Clarify or narrow the go vet check in buildModule

**Problem:** `buildModule` in `internal/amazon/amazontest/server_dependencies_test.go:307-319` runs `go vet ./...` on fixture modules. Its comment says vet "type-checks its packages with their test files", but vet also runs its full default analyzer set (printf, copylocks, tests, ...). A future valid fixture that compiles but has an analyzer finding would fail `TestFixtureModulesBuild_Issue251_TC001` with the misleading message "valid fixture does not build".

**Suggested work:** Do one of these:
- Say in the comment (and, if needed, in the failure message) that vet's analyzers must pass too.
- Limit vet to the type check plus one trivial analyzer, e.g. `{"vet", "-atomic", "./..."}` (naming one analyzer turns the others off), so that only compile errors fail the check.

Keep the test-file control subtest (`control: a test file uses a name the stub does not declare`) passing.

From review comment R1-01 on #339 (`doc/todo/issue-254-r1-01.md`).

## Comments by trusted authors

