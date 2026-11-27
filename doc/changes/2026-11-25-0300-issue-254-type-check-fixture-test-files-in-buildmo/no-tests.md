# No red phase for issue #254

The change is test code: `buildModule`, one subtest and the comments of `internal/amazon/amazontest/server_dependencies_test.go`. That file is locked after this stage, and FR-TEST-HYG-069 allows no other file outside `doc/`, so this stage makes the change itself, as the technical specification says (TS-TEST-HYG-090). The red state is shown inside the test: the new control fails when `buildModule` runs only `go build ./...`, as on `master` (run 3). The development stage writes no code: it only sets the statuses.

## Delivered in `internal/amazon/amazontest/server_dependencies_test.go`

- `buildModule` runs `go build ./...`, then `go vet ./...`, from the table `[][]string{{"build", "./..."}, {"vet", "./..."}}`, each with `Dir = dir` and `Env = append(os.Environ(), fixtureEnv...)`. It returns the first failure as `go <args>: <err>: <output>`, and `nil` only when both succeed. Its comment names both commands.
- New subtest `control: a test file uses a name the stub does not declare`, after the `DownloadMarker` control. It is not added to `fixtureCases`. Its fixture is `plainMain` plus a `cmd/app/main_test.go` that uses `amazontest.Nope`, with the default stub. It requires an error that holds `amazontest.Nope` and starts with `go vet ./...`.
- The comment above `TestFixtureModulesBuild_Issue251_TC001` is updated, and an issue #254 block is added to the header comment. The issue #251 block is unchanged.
- `fixtureCases`, the loop, `TC-003 h fixture: plainMain`, the `DownloadMarker` control, every failure message, `writeModule`, `fixtureEnv`, the dependency check and its tests and helpers have no hunk in `git diff master`.

## Runs

Fault injections 2 and 3 were temporary in-place edits of the test file, each reverted before the next step. Edit 1 on `master` (run 3) edited a copy of the `master` file (`fi_master_test.go.txt`), which `go test -overlay fi_overlay.json` used in place of the test file. The scratch files `fi_*` in this folder are listed in `delete.txt`. The command is `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -count=1 ./internal/amazon/amazontest/` unless stated otherwise.

1. With the change, `-v -count=1` passes. All 14 subtests pass: rows a to g, the three corners, `TC-003 h fixture: plainMain` and both controls. Row d (2 test files) passes, and so does `plainMain` (no test file). `-count=3` passes. Every valid fixture goes through `go build ./...` and `go vet ./...` with `fixtureEnv` (`GOPROXY=off`, `GOWORK=off`, `GOTOOLCHAIN=local`) in a `t.TempDir()` module. Two things were not done. The sandbox needs approval for an environment-prefixed command, so the run with an empty `GOMODCACHE` could not be made. A separate by-hand run was not made either, because the subtests run exactly those two commands on each fixture. The fixtures have no `require`, so they need nothing from the module cache.
2. One edit of row d at a time:
   1. `cmd/app/main_test.go` = `package main` + import of the helper + `var _ = amazontest.Nope`: FAILS on row d with `valid fixture does not build: go vet ./...: exit status 1: ... vet: cmd/app/main_test.go:5:20: undefined: amazontest.Nope`;
   2. the same in `internal/service/service_test.go` (`package service`): FAILS on row d with `... go vet ./...: ... vet: internal/service/service_test.go:5:20: undefined: amazontest.Nope`;
   3. `cmd/app/main_test.go` with an unclosed `import "fmt`: FAILS on row d with `valid fixture does not build: go build ./...: exit status 1: cmd/app/main_test.go:3:8: string literal not terminated`. `go build` already reports this one, because the go command parses `_test.go` files when it loads packages. Only type errors in test files needed vet.

   No other subtest failed. Without the edits the test passes (run 1).
3. Edit 1 on the `master` version (through the overlay): `-v` passes with the 12 `master` subtests, row d included. So the old check did not compile test files. With the change and `buildModule` reduced to `{{"build", "./..."}}`, the new control FAILS with `the module compiles, want a failure on the undeclared Nope in a test file`, and no other subtest fails.
4. Subtest names. The `-v` names of `TestFixtureModulesBuild_Issue251_TC001`, `TestListServerDependencies_TC003_FixtureModules`, `TestCheckServerDependencies_TC002_JudgesListing` and `TestServerDoesNotDependOnTestHelper` were saved on `master` and then compared with the change. The only difference is the added line `--- PASS: TestFixtureModulesBuild_Issue251_TC001/control:_a_test_file_uses_a_name_the_stub_does_not_declare`. On both, every test passes, row d passes the dependency check and row h stays a listing failure.
5. Repository checks and gates:
   - `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/amazontest/server_dependencies_test.go`, and no fault injection is left in it.
   - A search of the file for `out of scope`, `only runs go build` and `not compiled` finds nothing.
   - `gofmt -l internal/amazon` prints nothing. `go vet ./...` passes. `go test -count=1 ./...` passes, with every package `ok`.

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | `TestFixtureModulesBuild_Issue251_TC001`, valid subtests and `TC-003 h fixture: plainMain`; run 1 |
| TC-002 | Code check of `buildModule` (above); run 1 (row d, `plainMain`); run 2 (the failing command is named: `go vet ./...` for edits 1 and 2, `go build ./...` for edit 3) |
| TC-003 | Runs 2 and 3 |
| TC-004 | Subtest `control: a test file uses a name the stub does not declare`; run 3 (fails without vet) |
| TC-005 | Run 4; code check: `fixtureCases`, the `DownloadMarker` control and the failure messages have no hunk in the diff |
| TC-006 | Code check of the three comments; run 5 search |
| TC-007 | Run 4; code check: no hunk in the dependency check, its helpers or `fixtureEnv` |
| TC-008 | Run 5. The development stage runs the gates again |
