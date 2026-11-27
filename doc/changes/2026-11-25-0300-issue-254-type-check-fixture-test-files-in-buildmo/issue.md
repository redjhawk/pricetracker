# Issue #254 Type-check fixture test files in buildModule or document the scope

**Problem:** `buildModule` in `internal/amazon/amazontest/server_dependencies_test.go:302-310` runs `go build ./...`, which skips `_test.go` files. The test files of row d (`cmd/app/main_test.go`, `internal/service/service_test.go`) are never compiled.

**Suggested work:** Also run `go vet ./...` (which type-checks test files) in the fixture module inside `buildModule`. Otherwise, state in the comment above `buildModule` / `TestFixtureModulesBuild_Issue251_TC001` (line 312) that fixture test files are out of scope.

From review comment R1-02 on #252 (`doc/todo/issue-251-r1-02.md`).

## Comments by trusted authors

