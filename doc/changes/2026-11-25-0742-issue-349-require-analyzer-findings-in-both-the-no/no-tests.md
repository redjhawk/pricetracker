# No red phase for issue #349

The change is test code: the findings guard of the subtest `analyzer findings in non-test and test files do not fail the check` of `TestFixtureModulesBuild_Issue251_TC001`, and a header comment block, in `internal/amazon/amazontest/server_dependencies_test.go`. That file is locked after this stage, and FR-TEST-HYG-089 allows no other file outside `doc/`, so this stage makes the change itself (TS-TEST-HYG-117). On a normal run the fixture has findings in both files, so the new checks pass. Run 2 shows the red state: findings removed from `cmd/app/main_test.go` fail the new subtest and pass the `master` one. The development stage writes no code. It only sets the statuses.

## Delivered in `internal/amazon/amazontest/server_dependencies_test.go`

- In the subtest, after its unchanged `buildModule` check and the unchanged command lines (`exec.Command("go", "vet", "./...")`, `cmd.Dir = dir`, `cmd.Env = append(os.Environ(), fixtureEnv...)`):
  - the comment above the guard is replaced by the issue #344 and issue #349 comment of the technical specification: it names `cmd/app/main.go` and `cmd/app/main_test.go`, says what the subtest proves, why `vetArgs` is not used and why only positions are matched;
  - `cmd.Run() == nil` becomes `out, err := cmd.CombinedOutput()` and `err == nil`, which keeps the module message;
  - a loop over `"cmd/app/main.go"` and `"cmd/app/main_test.go"` reports `t.Errorf("the fixture has no analyzer finding in %s, so the subtest proves nothing", file)` when `string(out)` does not hold `file+":"`.

  There is no `t.Fatal`, `return` or condition on another check between or in the build check, the module check and the two file checks. No analyzer message is matched.
- An issue #349 block after the unchanged issue #344 block of the header comment, listing TC-002, TC-003, TC-004, TC-005, TC-007 and TC-008 as proved outside the file.
- `git diff master` has two hunks (22 added lines, 4 removed). The subtest name, its fixture, `buildModule`, `vetArgs`, `writeModule`, `fixtureEnv`, `fixtureCases`, `plainMain`, `helperMain`, the stub, the other subtests and their messages, the comment above the test, the other header blocks and the dependency check have no hunk. No import changes. No new line is longer than 120 columns.

## Runs

Each fault injection was a temporary in-place edit of the test file, reverted before the next step. The final `git diff master` shows none of them. "The `master` subtest" means the changed file with the guard lines after `cmd.Env` set back to `if cmd.Run() == nil { t.Error(...) }` (the header block does not affect results). "Findings removed from main.go" and "from main_test.go" are as defined in `testcases.md`. Unless stated otherwise, the command is `R` = `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/`. Toolchain: the local Go, offline (`fixtureEnv`). Line numbers are those of the changed file: 430 is the `buildModule` check, 442 the module message, 446 the file messages.

1. **Normal run and the fixture's findings (TC-001).** The sandbox of this stage allowed only `go test`, so the full vet output was recorded through a temporary `t.Logf("... err=%v output:\n%s", err, out)` after `CombinedOutput` (reverted). On the subtest's own `dir`, `go vet ./...` returned `exit status 1` with:

   ```text
   cmd/app/main.go:8:13: lock passes lock by value: sync.Mutex
   cmd/app/main.go:11:14: fmt.Printf format %d has arg "x" of wrong type string
   cmd/app/main_test.go:9:14: fmt.Printf format %d has arg "x" of wrong type string
   cmd/app/main_test.go:8:6: Testfoo has malformed name: first letter after 'Test' must not be lowercase
   ```

   The output holds both `cmd/app/main.go:` and `cmd/app/main_test.go:`. In the same run the subtest passed, so `buildModule` succeeded on that `dir`; the guard and `buildModule` share the one folder. Without the log, `R` passes: 14 subtests, all PASS, the analyzer-findings subtest named as on `master`, with none of the module, file or build messages. `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -count=3 ./internal/amazon/amazontest/` passes.
2. **Findings removed from main_test.go (TC-002).**
   - With the change, `R` FAILS. The subtest reports only `server_dependencies_test.go:446: the fixture has no analyzer finding in cmd/app/main_test.go, so the subtest proves nothing`: no main.go message, no module message (vet still fails on `cmd/app/main.go`), no build message. The other 13 subtests pass.
   - With the same edit and the `master` subtest, `R` passes. That is the vacuous pass this change closes.
   - With the original fixture and only `Testfoo` renamed `TestFoo`, `R` passes (the `Printf` finding of `cmd/app/main_test.go` remains).
   - With the original fixture and only the `Printf` call of `cmd/app/main_test.go` changed to `fmt.Printf("%d\n", 1)`, `R` passes (the `Testfoo` finding remains).
3. **Findings removed from main.go (TC-003).**
   - With the change, `R` FAILS. The subtest reports only `server_dependencies_test.go:446: the fixture has no analyzer finding in cmd/app/main.go, so the subtest proves nothing`: no main_test.go, module or build message. The other 13 subtests pass.
   - With only the `Printf` call of `cmd/app/main.go` changed (copylocks finding kept), `R` passes.
   - With only the copylocks finding removed (`lock(m *sync.Mutex)` called with `&sync.Mutex{}`, `Printf` finding kept), `R` passes.
4. **Findings removed from both files (TC-004).** `R` FAILS. The subtest reports `server_dependencies_test.go:442: the fixture has no analyzer finding, so the subtest proves nothing` and both file messages (line 446, main.go then main_test.go), without the build message and without a panic. The other 13 subtests pass.
5. **Independent reports (TC-005).** One edit at a time:
   1. `vetArgs` = `{"vet", "./..."}` with findings removed from main_test.go: the subtest FAILS and reports, in the same run, `valid fixture does not build: go vet ./...: exit status 1: cmd/app/main.go:8:13: lock passes lock by value: sync.Mutex` / `cmd/app/main.go:11:14: fmt.Printf format %d has arg "x" of wrong type string` (line 430) and the main_test.go message (line 446); no main.go or module message. The other 13 subtests pass, the test-file control (prefix from `vetArgs`) included, as recorded for issue #344.
   2. `vetArgs` = `{"nosuchcmd"}` with findings removed from both files: the subtest FAILS and reports `valid fixture does not build: go nosuchcmd: exit status 2: go nosuchcmd: unknown command` (line 430) and the module message (line 442), plus both file messages, in the same run. The guard does not use `vetArgs`: its `go vet ./...` ran and succeeded on the finding-free fixture. As expected for this edit, and as recorded for issue #344, rows a to e, the three corners and `TC-003 h fixture: plainMain` fail with the build message, and the test-file control fails because the failure does not name `amazontest.Nope`; rows f and g and the `DownloadMarker` control pass.
   3. `vetArgs` = `{"vet", "./..."}`, fixture unchanged: the subtest FAILS with the build message only (line 430, holding the four findings of run 1), no guard message. The other 13 subtests pass.

   Without the edits, `R` passes (run 1).
6. **Comparison with `master` (TC-007).** `go test -run '^(TestFixtureModulesBuild_Issue251_TC001|TestListServerDependencies_TC003_FixtureModules|TestCheckServerDependencies_TC002_JudgesListing|TestServerDoesNotDependOnTestHelper)$' -v -count=1 ./internal/amazon/amazontest/` was run with the change and with the `master` subtest. Both give the same 41 `--- ` result lines, all PASS, with the same names: 4 tests, 12 subtests of `TestCheckServerDependencies_TC002_JudgesListing`, 11 of `TestListServerDependencies_TC003_FixtureModules` and 14 of `TestFixtureModulesBuild_Issue251_TC001`. There is no new or removed subtest. `TC-003 h fixture: plainMain` (no test file) passes.
7. **Repository checks and gates (TC-008), run twice:**
   - `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/amazontest/server_dependencies_test.go`, and no fault injection is left in it.
   - `gofmt -l internal/amazon` prints nothing.
   - `go vet ./...` passes. The repository gate keeps all its analyzers; the fixture's findings are in string literals, so vet does not report them.
   - `go test -count=1 ./...` passes, with every package `ok`.
   - `go test -count=1 ./internal/amazon/amazontest/...` passes.

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | The subtest `analyzer findings in non-test and test files do not fail the check`; run 1 (recorded vet output with both positions, `-count=3`) |
| TC-002 | Run 2 |
| TC-003 | Run 3 |
| TC-004 | Run 4 |
| TC-005 | Run 5 |
| TC-006 | Code check of `git diff master`: the guard command lines unchanged (`go vet ./...`, not `vetArgs`; `cmd.Dir = dir`; `os.Environ()` plus `fixtureEnv`); `CombinedOutput`; `t.Error` for the module message when `err == nil`; one `t.Errorf` per file position in a loop, with no `t.Fatal`, `return` or analyzer message; the new comment naming both files; the issue #349 header block listing TC-002, TC-003, TC-004, TC-005, TC-007 and TC-008; `buildModule`, `vetArgs`, the subtest name and its fixture have no hunk |
| TC-007 | Run 6. Code check: `fixtureCases`, `plainMain`, `helperMain`, the stub, `writeModule` and `fixtureEnv` have no hunk |
| TC-008 | Run 7. The development stage runs the gates again |
