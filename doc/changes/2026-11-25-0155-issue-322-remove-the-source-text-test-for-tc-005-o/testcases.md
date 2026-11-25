# Test cases: remove the source-text test for TC-005 of issue #197 (issue #322)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-061 to FR-TEST-HYG-064). There are no use cases: see [no-use-cases.md](no-use-cases.md). `profile()` and FR-AMZ-BPROF-001 to FR-AMZ-BPROF-010 stay as specified in [amazon-browser-profile](../../specifications/amazon-browser-profile/functional.md); this change does not alter them.

Conventions:

- **The test file:** `internal/amazon/browser_profile_values_test.go`.
- **The source-text test:** `TestBrowserProfile_Issue197_TC005_ProfileUsesTheExplicitForm` and its comment ("TC-005 of issue #197: the stale lock branch of profile() has the explicit form ...").
- **The behaviour tests:** `TestBrowserProfile_Issue197_TC001_UndiscardableLockGivesZeroValuesAndTheError`, `..._TC002_BothFailureBranchesReturnTheSameForm`, `..._TC003_PreparedProfileReturnsItsPath` and `..._TC004_UndiscardableLockIsAStartFailure`, with the helpers they use (`assertZeroValues`, `writeUndiscardableLock`, `assertFileContent`, `assertLocksAbsent`, `profileOf`).
- **Fault injection:** a temporary edit of `BrowserCollector.profile` in `internal/amazon/browser.go`, made only for the run of a case, reverted afterwards and never committed.
- The change only deletes test code. No production code, API, database or user interface changes, and there is no user flow, so there are no integration or e2e cases of the application. The cases are code checks, test runs (one with fault injection) and the gates. The IDs below are this change's own, not those in the test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-061 | TC-001 | — |
| FR-TEST-HYG-062 | TC-002, TC-003 | — |
| FR-TEST-HYG-063 | TC-003, TC-004, TC-005 | — |
| FR-TEST-HYG-064 | TC-006 | — |

## Cases

### TC-001: The source-text test is gone and no test reads browser.go

- Covers: FR-TEST-HYG-061
- Level: code check and unit run (Go, `internal/amazon`)
- Given the change applied on top of `master`
- When `grep -rn 'TestBrowserProfile_Issue197_TC005_ProfileUsesTheExplicitForm' --include='*.go' .` runs, the `*_test.go` files under `internal/amazon/` are searched for `parser.ParseFile`, `"browser.go"` used as a file to read, and `os.ReadFile` of a `.go` file, and `go test -run 'TC005_ProfileUsesTheExplicitForm' -v -count=1 ./internal/amazon/` runs
- Then the grep prints nothing, the search finds no test that reads the source of `browser.go` or compares the text of `profile()`, and the `go test` run reports no test run (`testing: warning: no tests to run`) and passes
- Corner cases:
  - the test is not merely renamed, skipped (`t.Skip`) or moved to another file or package: no test anywhere parses `browser.go` or holds the expected `profile()` text (for example the string `pricefollower-browser-` together with `os.RemoveAll(directory)` as source text);
  - the comment of the source-text test is removed with it;
  - mentions of `browser.go` in failure messages of other tests (for example "the browser got the profile %q") are not source reading and stay.

### TC-002: The test file keeps no import or declaration that only the source-text test used

- Covers: FR-TEST-HYG-062
- Level: code check of the test file, and `go vet` / build of `internal/amazon`
- Given the change applied on top of `master`
- When the import block and the declarations of the test file are read, and `go vet ./internal/amazon/` runs
- Then:
  - the file contains none of `"go/ast"`, `"go/format"`, `"go/parser"`, `"go/token"`, `parser.ParseFile`, `token.NewFileSet`, `format.Node` or `ast.`;
  - every remaining import (`errors`, `io/fs`, `os`, `path/filepath`, `strings`, `testing`) is still used by the behaviour tests or their helpers, and `strings` is kept because the behaviour tests use `strings.Contains` and `strings.HasPrefix`;
  - no helper, variable or constant remains that only the source-text test used;
  - `go vet ./internal/amazon/` passes (no unused import, no compile error).
- Corner cases:
  - an import still used by the behaviour tests is not removed by mistake (the package would not compile, or a helper would be missing);
  - no blank import (`_ "go/ast"`) or dummy use is left to keep an import alive.

### TC-003: The diff of the test file only removes lines

- Covers: FR-TEST-HYG-062, FR-TEST-HYG-063
- Level: repository check (`git diff master -- internal/amazon/browser_profile_values_test.go`)
- Given the change applied on top of `master`
- When the diff of the test file is listed
- Then it contains no added line, and its removed lines are only the source-text test, its comment, the blank line separating it, and the four imports `go/ast`, `go/format`, `go/parser`, `go/token`
- Corner cases:
  - the behaviour tests, their subtests and helpers, and the file's header comment are byte-for-byte unchanged;
  - `internal/amazon/browser_profile_test.go` and the other test files of `internal/amazon` are unchanged;
  - the file still ends with a single newline after the last remaining test (checked by `gofmt` in TC-006).

### TC-004: The issue #197 behaviour tests still pass

- Covers: FR-TEST-HYG-063
- Level: unit (Go, `internal/amazon`, without fault injection)
- Given the change applied
- When `go test -run 'TestBrowserProfile_Issue197' -v -count=1 ./internal/amazon/` runs, then again with `-count=3` and with `-race`
- Then every run passes, the four behaviour tests and all their subtests run and pass (TC-001 to TC-004 of issue #197), and no other `Issue197` test runs
- Corner cases:
  - repeated runs give the same result;
  - the TC-004 test, which needs a browser stub only, still runs rather than being skipped.

### TC-005: An edit of profile() that keeps its results makes no test fail

- Covers: FR-TEST-HYG-063
- Level: unit (Go, `internal/amazon`; fault injection)
- Given the change applied, and one of these temporary edits of `BrowserCollector.profile` at a time, each keeping the returned path, cleanup effect and error:
  1. the log message `remove temporary browser profile: %v` reworded (for example to `could not remove the temporary browser profile: %v`);
  2. the local variable `directory` renamed (for example to `dir`);
  3. the persistent-profile branch rewritten with a single `if err := ...; err != nil` chain or an extracted helper that returns the same values
- When `go test -count=1 ./internal/amazon/` runs
- Then it passes with each edit
- And, with edit 1 applied to the `master` version of the test file, `TestBrowserProfile_Issue197_TC005_ProfileUsesTheExplicitForm` fails, which shows the change removes the brittleness
- Corner cases:
  - an edit that changes a result (for example returning a non-nil cleanup with the error of the stale-lock branch) still makes the behaviour tests fail, so the deletion loses no behaviour coverage of FR-AMZ-BPROF-008 and FR-AMZ-BPROF-009;
  - no fault injection remains in the diff (checked by TC-006).

### TC-006: Only the test file changes outside `doc/`, and the gates pass

- Covers: FR-TEST-HYG-064
- Level: repository check and suite run (deterministic gates: `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` is listed and the gates run
- Then the only file listed outside `doc/` is `internal/amazon/browser_profile_values_test.go`, `gofmt -l internal/amazon` prints nothing, and `go vet ./...` and `go test ./...` pass
- Corner cases:
  - `internal/amazon/browser.go` and the other production Go code, frontend source, `go.mod`, `go.sum`, `package.json`, `tests/` and `API_SPECIFICATION.md` are unchanged, so the operator sees no change in the application;
  - `go.mod` keeps no dependency added or dropped by the change (the `go/*` packages are standard library);
  - no fault injection from TC-005 remains;
  - the gates give the same result when run twice.
