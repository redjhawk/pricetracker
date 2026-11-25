# No red phase for issue #322

The change only deletes test code: `TestBrowserProfile_Issue197_TC005_ProfileUsesTheExplicitForm`, its comment and the `go/ast`, `go/format`, `go/parser` and `go/token` imports from `internal/amazon/browser_profile_values_test.go`. That file is locked after this stage, and FR-TEST-HYG-064 allows no other file outside `doc/`, so this stage makes the deletion itself, as the technical specification says (TS-TEST-HYG-084). The deleted test passes on `master`, so no test can fail first on a missing implementation. The red gate is skipped. The development stage writes no code: it only sets the statuses.

## Delivered in `internal/amazon/browser_profile_values_test.go`

Removed lines only (`git diff master` of the file has no added line):

- the four imports `"go/ast"`, `"go/format"`, `"go/parser"`, `"go/token"`;
- the blank line after the TC-004 test, the two-line comment `// TC-005 of issue #197: ...` and the whole function `TestBrowserProfile_Issue197_TC005_ProfileUsesTheExplicitForm`.

The file now ends with the closing `}` of `TestBrowserProfile_Issue197_TC004_UndiscardableLockIsAStartFailure` and one newline. The header comment, the helpers, the TC-001 to TC-004 tests of issue #197 and the imports `errors`, `io/fs`, `os`, `path/filepath`, `strings`, `testing` are unchanged. The test had no package-level declaration, so nothing else was removed. No other file outside `doc/` changes.

## Runs

The sandbox refused edits of `internal/amazon/browser.go` and writes outside the repository, so the fault injections did not edit `browser.go`. Each one edited a copy of it in `fi/browser_log.go.txt`, first checked identical to `browser.go` with `diff`. That copy replaced `browser.go` in the build through `go test -overlay fi/overlay.json`. `browser.go` was never touched (`git diff master -- internal/amazon/browser.go` is empty). The scratch files under `fi/` are listed in `delete.txt` so the pipeline removes them. Paths below are relative to this change folder.

1. Before the deletion, with the log reworded in the copy (`log.Printf("could not remove the temporary browser profile: %v", err)`): the `master` TC-005 test reads `browser.go` from disk through `parser.ParseFile`, so it does not follow `-overlay` and still passes. To show the failure, a temporary overlay test file (`fi/run1_test.go.txt`, added as `internal/amazon/zz_run1_test.go`) held the `master` body of the test unchanged, except for the parsed path, which pointed at the edited copy. It FAILS with `temporary-profile branch of profile(): ... log.Printf("could not remove the temporary browser profile: %v", err) ... want: ... log.Printf("remove temporary browser profile: %v", err)`. That is the brittleness the change removes. The test reading the source from disk, rather than the code that is built, is a further sign that it checks the source and not the behaviour.
2. After the deletion, `go test -overlay fi/overlay.json -count=1 ./internal/amazon/`, one edit of the copy at a time, each keeping the returned path, cleanup effect and error:
   1. log reworded as in run 1: `ok` (151.7 s);
   2. log restored, local variable `directory` renamed to `dir` (declaration, return and `os.RemoveAll`): `ok` (149.6 s);
   3. rename restored, the persistent-profile branch moved into `preparePersistentProfile(dir string) (string, func(), error)` with the same `MkdirAll`, `removeStaleLocks` and return statements, which `profile()` returns directly: `ok` (149.4 s).
3. After the deletion, helper restored, a result-changing edit: the `removeStaleLocks` failure returns `c.options.ProfileDir, func() {}, err`. `go test -overlay fi/overlay.json -run 'TestBrowserProfile_Issue197' -count=1 ./internal/amazon/` FAILS in `TestBrowserProfile_Issue197_TC001_...` (`path "...", want an empty path` and `cleanup is not nil` from `assertZeroValues`, for each lock name and both SingletonSocket subtests) and in `TestBrowserProfile_Issue197_TC002_...` (stale-lock subtest). Behaviour coverage of FR-AMZ-BPROF-008 and FR-AMZ-BPROF-009 is kept.
4. Without overlay, on the real tree:
   - `go test -run 'TestBrowserProfile_Issue197' -v -count=1 ./internal/amazon/` passes. TC-001 runs with 5 subtests, TC-002 with 2, TC-003 with 5 and TC-004 with 1. TC-004 runs, not skipped. No other `Issue197` test runs.
   - `-count=3` passes.
   - `-race` cannot run on this runner: cgo is off, and with `CGO_ENABLED=1` there is no `gcc` (as for issues #315 and #324). The change adds no code and no concurrency.
   - `go test -run 'TC005_ProfileUsesTheExplicitForm' -v -count=1 ./internal/amazon/` prints `testing: warning: no tests to run` and passes.
5. Repository checks and gates:
   - A search of every `*.go` file finds no `TestBrowserProfile_Issue197_TC005_ProfileUsesTheExplicitForm`, `parser.ParseFile`, `token.NewFileSet`, `format.Node`, `go/*` import or `os.ReadFile` of a `.go` file.
   - The `*_test.go` files of `internal/amazon` contain no `"browser.go"`, `ast.` or `TC-005 of issue #197`.
   - `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/browser_profile_values_test.go`, and its diff has no added line.
   - `gofmt -l internal/amazon` prints nothing. `go vet ./...` passes. `go test -count=1 ./...` passes, with every package `ok`.

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | Run 5 searches; run 4: the TC-005 name matches no test |
| TC-002 | Code check of the import block (six imports left, all used, no blank import); `go vet ./internal/amazon/` and `go vet ./...` pass |
| TC-003 | `git diff master` of the file has removed lines only: the four imports, the blank line, the comment and the test; no other test file changes; `gofmt -l` is empty |
| TC-004 | Run 4: `-v`, `-count=3` (`-race` is not available on this runner) |
| TC-005 | Runs 1 to 3 |
| TC-006 | Run 5. The development stage runs the gates again |
