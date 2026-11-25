# No red phase for issue #344

The change is test code: a findings guard in the subtest `analyzer findings in non-test and test files do not fail the check` of `TestFixtureModulesBuild_Issue251_TC001`, and a header comment block, in `internal/amazon/amazontest/server_dependencies_test.go`. That file is locked after this stage, and FR-TEST-HYG-085 allows no other file outside `doc/`, so this stage makes the change itself (TS-TEST-HYG-112). On a normal run the fixture has findings, so the new guard passes. Run 2 shows the red state: a finding-free fixture fails the new subtest and passes the `master` one. The development stage writes no code. It only sets the statuses.

## Delivered in `internal/amazon/amazontest/server_dependencies_test.go`

- At the end of the subtest, after its unchanged `buildModule` check: a comment, then `exec.Command("go", "vet", "./...")` with `cmd.Dir = dir` and `cmd.Env = append(os.Environ(), fixtureEnv...)`. When `cmd.Run()` returns `nil`, it reports `t.Error("the fixture has no analyzer finding, so the subtest proves nothing")`. There is no `t.Fatal` or `return` in either check. Analyzer messages are not matched.
- An issue #344 block after the issue #340 block of the header comment. It lists TC-002, TC-003, TC-005 and TC-006 as proved outside the file.
- `git diff master` has two hunks, 18 added lines and no removed line. The subtest name, its fixture, `buildModule`, `vetArgs`, `writeModule`, `fixtureEnv`, `fixtureCases`, the other subtests and their messages, the comment above the test and the dependency check have no hunk. No import changes.

## Runs

Each fault injection was a temporary in-place edit of the test file, reverted before the next step. `git diff master` shows none of them. The guard and the header block are the only differences from `master`, so "the `master` subtest" means the changed file with the guard lines removed. Unless stated otherwise, the command is `R` = `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/`. Toolchain: the local Go, offline (`fixtureEnv`).

1. **Normal run and the fixture's findings (TC-001).** With the change, `R` passes: 14 subtests, all PASS, the analyzer-findings subtest included and named as on `master`. `-count=3` passes too. The sandbox of this stage allowed only `go test`, so the full vet output was recorded through a temporary edit of the guard that logged `cmd.CombinedOutput()` (reverted). On the subtest's own `dir`, `go vet ./...` exits with status 1 and reports:

   ```text
   cmd/app/main.go:8:13: lock passes lock by value: sync.Mutex
   cmd/app/main.go:11:14: fmt.Printf format %d has arg "x" of wrong type string
   cmd/app/main_test.go:9:14: fmt.Printf format %d has arg "x" of wrong type string
   cmd/app/main_test.go:8:6: Testfoo has malformed name: first letter after 'Test' must not be lowercase
   ```

   That is the copylocks, printf and tests findings. In the same run the `buildModule` check passed on that `dir`, so `go build ./...` and `go vet -atomic ./...` succeed on it. The guard and `buildModule` share the one folder, so there is no second fixture copy.
2. **Finding-free fixture (TC-002).** The fixture was edited to `fmt.Printf("%d\n", 1)` in both files, `func lock(m *sync.Mutex) {}` called with `lock(&sync.Mutex{})`, and `Testfoo` renamed `TestFoo`.
   - With the change, `R` FAILS. The subtest reports only `server_dependencies_test.go:429: the fixture has no analyzer finding, so the subtest proves nothing`, without `valid fixture does not build`. The other 13 subtests pass.
   - With the same edit and the `master` subtest, `R` passes. That is the vacuous pass the guard catches.
   - With the original fixture and only `Testfoo` renamed `TestFoo`, `R` passes with the change: the remaining printf and copylocks findings still fail the full vet run.
3. **Independent reports (TC-003).** One edit at a time:
   1. `vetArgs` = `{"vet", "./..."}`: the subtest FAILS with `valid fixture does not build: go vet ./...: exit status 1:` followed by the four findings of run 1, and no guard message. The other 13 subtests pass, the test-file control (`amazontest.Nope`, prefix from `vetArgs`) included.
   2. `vetArgs` = `{"nosuchcmd"}` with the finding-free fixture of run 2: the subtest FAILS and reports both `valid fixture does not build: go nosuchcmd: exit status 2: go nosuchcmd: unknown command` (line 420) and `the fixture has no analyzer finding, so the subtest proves nothing` (line 429) in the same run. The guard does not use `vetArgs`: its `go vet ./...` ran and succeeded. As expected for this edit, the other valid-fixture subtests fail too (rows a to e, the three corners, `TC-003 h fixture: plainMain`), and the test-file control fails because the failure does not name `amazontest.Nope`. Rows f and g and the `DownloadMarker` control pass, as `go build ./...` fails first on them.
   3. `var _ int = "x"` added at package level to the fixture's `cmd/app/main.go`, findings kept: the subtest FAILS with `valid fixture does not build: go build ./...: exit status 1: # pricefollower.local/cmd/app cmd/app/main.go:10:13: cannot use "x" (untyped string constant) as int value in variable declaration`, and no guard message. The other subtests pass.

   Without the edits, `R` passes (run 1).
4. **Comparison with `master` (TC-005).** `go test -run '^(TestFixtureModulesBuild_Issue251_TC001|TestListServerDependencies_TC003_FixtureModules|TestCheckServerDependencies_TC002_JudgesListing|TestServerDoesNotDependOnTestHelper)$' -v -count=1 ./internal/amazon/amazontest/` was run with the change and with the `master` subtest. Both give the same 41 `--- ` result lines, all PASS, with the same names: 4 tests, 12 subtests of `TestCheckServerDependencies_TC002_JudgesListing`, 11 of `TestListServerDependencies_TC003_FixtureModules` and 14 of `TestFixtureModulesBuild_Issue251_TC001`. There is no new or removed subtest. That run includes `R`.
5. **Repository checks and gates (TC-006), run twice:**
   - `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/amazontest/server_dependencies_test.go`, and no fault injection is left in it.
   - `gofmt -l internal/amazon` prints nothing.
   - `go vet ./...` passes. The repository gate keeps all its analyzers. The fixture's findings are in string literals, so vet does not report them.
   - `go test -count=1 ./...` passes, with every package `ok`.
   - `go test -count=1 ./internal/amazon/amazontest/...` passes.

Scratch files written in this change folder for run 1 (a fixture copy and a script that the sandbox did not let run) are listed in `delete.txt`.

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | The subtest `analyzer findings in non-test and test files do not fail the check`; run 1 (recorded vet output, `-count=3`) |
| TC-002 | Run 2 |
| TC-003 | Run 3 |
| TC-004 | Code check of `git diff master`: the guard (`go vet ./...`, not `vetArgs`; `cmd.Dir = dir`; `os.Environ()` plus `fixtureEnv`; `t.Error` only on success, after the `buildModule` check, no `t.Fatal` or `return`), its comment, and the issue #344 header block listing TC-002, TC-003, TC-005 and TC-006; `buildModule`, `vetArgs`, the subtest name and its fixture have no hunk |
| TC-005 | Run 4. Code check: `fixtureCases`, `plainMain`, `helperMain`, the stub, `writeModule` and `fixtureEnv` have no hunk |
| TC-006 | Run 5. The development stage runs the gates again |
