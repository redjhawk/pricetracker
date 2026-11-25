# No red phase for issue #340

The change is test code: `buildModule`'s vet command, one control check, one new subtest and three comments of `internal/amazon/amazontest/server_dependencies_test.go`. That file is locked after this stage, and FR-TEST-HYG-077 allows no other file outside `doc/`, so this stage makes the change itself, as the technical specification says (TS-TEST-HYG-101). Run 4 shows the red state: with `master`'s `go vet ./...`, the new subtest fails. The development stage writes no code. It only sets the statuses.

## Delivered in `internal/amazon/amazontest/server_dependencies_test.go`

- New `var vetArgs = []string{"vet", "-atomic", "./..."}` with its comment. `buildModule` runs `{{"build", "./..."}, vetArgs}`. Its loop, `fixtureEnv` and its error `go <args>: <err>: <output>` are unchanged.
- The test-file control checks the prefix `"go " + strings.Join(vetArgs, " ") + ":"`, and its message names that command. Its name, fixture, `t.Fatal` and `amazontest.Nope` check are unchanged.
- New subtest `analyzer findings in non-test and test files do not fail the check`, after the test-file control and not in `fixtureCases`. Its `cmd/app/main.go` has a printf and a copylocks finding, and its `cmd/app/main_test.go` has a printf and a tests finding.
- The comments above `buildModule` and the test are updated. In the header comment, the issue #254 line now says `go vet -atomic ./..., see issue #340`, and an issue #340 block is added.
- `writeModule`, `fixtureEnv`, `fixtureCases`, the fixture constants, the stub, the `fixtureCases` loop and its messages, `TC-003 h fixture: plainMain`, the `DownloadMarker` control, the dependency check and its helpers and tests have no hunk in `git diff master`. No import changes.

## Runs

Each fault injection was a temporary in-place edit of the test file, reverted before the next step. `git diff master` shows none of them. "`master`'s command" means `vetArgs` set to `{"vet", "./..."}`. With that, `buildModule` runs exactly what it runs on `master` (`go build ./...`, then `go vet ./...`), and the fixture rows are untouched. Run 5 compared against the real `master` file through `go test -overlay`, using the scratch files listed in `delete.txt`. Unless stated otherwise, the command is `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/`.

1. **New subtest and valid fixtures (TC-001, TC-004).** With the change, all 15 subtests pass: the 14 of `master` plus the new one. `-count=3` passes too. With `master`'s command (run 4, edit 1), the output of `buildModule` is recorded. It shows that `go build ./...` succeeded on the new fixture, because the failure comes from the vet step, and that `go vet ./...` reports every planned finding:

   ```text
   go vet ./...: exit status 1: cmd/app/main.go:11:14: fmt.Printf format %d has arg "x" of wrong type string
   cmd/app/main_test.go:9:14: fmt.Printf format %d has arg "x" of wrong type string
   cmd/app/main_test.go:8:6: Testfoo has malformed name: first letter after 'Test' must not be lowercase
   cmd/app/main.go:8:13: lock passes lock by value: sync.Mutex
   ```

   `go vet -atomic ./...` succeeds on that fixture, on every valid row of `fixtureCases` and on `plainMain`, so every valid subtest passes. Network use is off (`GOPROXY=off`, `GOWORK=off`, `GOTOOLCHAIN=local`).
2. **Analyzer findings added to valid rows (TC-002).** Each edit was run with the build test and with `TestListServerDependencies_TC003_FixtureModules`:
   1. Row a's `cmd/app/main.go` was replaced by a copy of `helperMain` that also imports `fmt` and calls `fmt.Printf("%d\n", "x")` in `main`. With the change, both tests pass. With `master`'s command, row a fails with `valid fixture does not build: go vet ./...: exit status 1: cmd/app/main.go:11:27: fmt.Printf format %d has arg "x" of wrong type string`.
   2. Row d's `cmd/app/main_test.go` was given `import "fmt"` and `func f() { fmt.Printf("%d\n", "x") }`. With the change, both tests pass. With `master`'s command, row d fails with `valid fixture does not build: go vet ./...: exit status 1: cmd/app/main_test.go:7:24: fmt.Printf format %d has arg "x" of wrong type string`.
   3. Row d's `internal/service/service_test.go` was given `import "sync"` and `func f() { var m sync.Mutex; n := m; _ = n }`. With the change, both tests pass. With `master`'s command, row d fails with `assignment copies lock value to n: sync.Mutex`, which shows that the finding is real.
3. **Compile errors still fail (TC-003).** One edit at a time:
   1. Row d's `cmd/app/main_test.go` was replaced by an `amazontest.Nope` test file. Row d fails with `valid fixture does not build: go vet -atomic ./...: exit status 1: # pricefollower.local/cmd/app ... vet: cmd/app/main_test.go:5:20: undefined: amazontest.Nope`.
   2. The same was done in `internal/service/service_test.go` (`package service`). Row d fails with `valid fixture does not build: go vet -atomic ./...: exit status 1: # pricefollower.local/internal/service ... vet: internal/service/service_test.go:5:20: undefined: amazontest.Nope`.
   3. Row d's `cmd/app/main_test.go` was given an unclosed `import "fmt`. Row d fails with `valid fixture does not build: go build ./...: exit status 1: cmd/app/main_test.go:3:8: string literal not terminated`. Unlike the design's expectation, `go build ./...` reports this one. To load packages, the go command parses the import clause of every file, test files included. A syntax error outside the imports (`func f() { x := }`) only reaches vet. Row d then fails with `valid fixture does not build: go vet -atomic ./...: exit status 1: cmd/app/main_test.go:5:17: expected operand, found '}'`.
   4. Row a's `cmd/app/main.go` was replaced by a copy of `helperMain` with `var _ int = "x"` added. Row a fails with `valid fixture does not build: go build ./...: exit status 1: # pricefollower.local/cmd/app cmd/app/main.go:7:13: cannot use "x" (untyped string constant) as int value in variable declaration`.

   Every other subtest passes, and without the edits the test passes (run 1).
4. **Edits of `buildModule` (TC-001, TC-004):**
   1. With `vetArgs` = `{"vet", "./..."}` (`master`'s command), the new subtest FAILS with the output shown in run 1. Every other subtest passes.
   2. With `buildModule`'s table reduced to `{{"build", "./..."}}`, the test-file control FAILS with `the module compiles, want a failure on the undeclared Nope in a test file`.
   3. With `vetArgs` = `{"vet", "-unusedresult", "./..."}`, all 15 subtests pass, the test-file control included. Its prefix follows the command.
5. **Comparison with `master` (TC-005).** This run used `go test -overlay` with the `master` file in place of the changed one. It ran `TestFixtureModulesBuild_Issue251_TC001`, `TestListServerDependencies_TC003_FixtureModules`, `TestCheckServerDependencies_TC002_JudgesListing` and `TestServerDoesNotDependOnTestHelper` with `-v -count=1`. The `--- ` result lines, without durations, were compared with `diff`:
   - `master`: 40 tests and subtests, all PASS.
   - With the change: the same 40, all PASS, plus `analyzer_findings_in_non-test_and_test_files_do_not_fail_the_check` (PASS). That is the only line of the diff.
6. **Repository checks and gates (TC-007), run twice:**
   - `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/amazontest/server_dependencies_test.go`, and no fault injection is left in it.
   - `gofmt -l internal/amazon` prints nothing.
   - `go vet ./...` passes. The repository gate keeps all its analyzers. The fixture findings are in string literals, so vet does not report them.
   - `go test -count=1 ./...` passes, with every package `ok`.
   - `go test -count=1 ./internal/amazon/amazontest/...` passes.

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | The subtest `analyzer findings in non-test and test files do not fail the check`; run 1 (recorded vet output, `-count=3`); run 4, edit 1 |
| TC-002 | Run 2 |
| TC-003 | Run 3 |
| TC-004 | The subtests `control: a test file uses a name the stub does not declare` (prefix from `vetArgs`) and `control: a program uses a name the stub does not declare`; run 4, edits 2 and 3 |
| TC-005 | Run 5. Code check: `fixtureCases`, the fixtures, the stub, the loop and its messages, `TC-003 h fixture: plainMain`, the `DownloadMarker` control, `writeModule` and `fixtureEnv` have no hunk in the diff |
| TC-006 | Code check of the comments above `vetArgs`, `buildModule` and the test and of the header comment. They name `go vet -atomic ./...`, say why one analyzer is named and that analyzer findings do not fail the check, and the issue #340 block lists TC-002 to TC-005 and TC-007 |
| TC-007 | Run 6. The development stage runs the gates again |
