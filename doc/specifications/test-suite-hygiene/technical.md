# Test suite hygiene: technical specification

Status: TS-TEST-HYG-001–003 ready (issue #208); TS-TEST-HYG-004–009 ready (issue #232); TS-TEST-HYG-010–015 implemented (issue #244)
Requirements: [functional](functional.md). Sources: issue #208, issue #232, issue #244.

## Design

### Leftover scratch file (issue #208)

Delete `internal/service/zz_scratch_docs_test.go`. On `master` it holds only `package service` and a comment, so it declares no test, helper, import or symbol. Every other file in `internal/service/` declares `package service`, so the package keeps its package clause and test set. Nothing else refers to the file outside `doc/`.

The file is a test file (`_test.go`). The green gate fails on any test file changed after the TDD red commit, so the deletion must land in the **tests stage**: it writes `no-tests.md` (no new test is written, see `testcases.md`) and lists the path in its `delete.txt`. The development stage then has no code to change.

The issue #204 records (`doc/changes/2026-11-22-2139-issue-204-retrieval-error/`, including `tests.lock`) stay unchanged: the factory checks only the running issue's lock, and the functional specification excludes them.

### No-download check fails on unreadable files (issue #232)

Only `internal/amazon/amazontest/amazontest.go` changes (a non-`_test.go` file of a test-support package; the server binary never imports it).

- `downloadFile(path, name string, patterns []*regexp.Regexp) (string, error)`. The name rules (attachment name or a Chromium-derived variant, `download` prefix, partial-download prefixes and suffix) stay first and in the same order, and return `(reason, nil)` without reading the file. Only then does the content rule run `os.ReadFile(path)`:
  - `err == nil`: `("holds the downloaded body", nil)` when the data contains `DownloadMarker`, otherwise `("", nil)`;
  - `errors.Is(err, fs.ErrNotExist)`: `("", nil)`. The file disappeared after it was listed, and a removed parent folder gives the same error;
  - any other error: `("", err)`. The error is returned unchanged; it is the `*fs.PathError` of `os.ReadFile` (for example `open <path>: permission denied`), so it already names the full path and the cause.
- In `AssertNoDownload`, the `filepath.WalkDir` callback returns the error of `downloadFile`. The walk stops, and the existing `t.Fatalf("list the files of %s: %v", root, err)` reports it. The message holds the root, the file's path and the read error, and it is not the `files left by a download` message. Downloads found before the read error are not reported. The test fails either way, and the developer fixes the read error first.
- Unchanged: a missing root and a `fs.ErrNotExist` passed to the callback are ignored; entries that are not regular files (folders, symbolic links, including dangling ones) are skipped and never read; `WalkDir` does not follow symbolic links; the `files left by a download` message and its reasons do not change. The helper keeps no state between calls.
- Update the doc comments of `AssertNoDownload` and `downloadFile` to say that an unreadable file fails the check and that a file that has disappeared is ignored.

Refactoring: changing `downloadFile`'s signature is the change the feature needs. The function is unexported, and only `AssertNoDownload` calls it. The risk is low: the four callers of `AssertNoDownload` (`internal/amazon/browser_navigation_test.go` (2), `internal/amazon/browser_interrupted_test.go`, `internal/httpapi/amazon_browser_navigation_test.go`) keep the same API and are not edited.

### Server dependency check (issue #244)

The text search in `TestAssertNoDownload_TC006_ChangeLeavesApplicationUntouched` is replaced by a check of what `go list -deps ./cmd/...` reports. The check lives in the test helper package itself, `internal/amazon/amazontest`: it is a test-only package (imported only by `_test.go` files), and the check is about that package. Its logic goes into a new non-test file so that the tests stage can write failing tests first (TDD red: the tests call functions that do not exist yet) and the development stage implements them.

New file `internal/amazon/amazontest/server_dependencies.go` (`package amazontest`, standard library only, unexported names):

- `var helperImportPath = reflect.TypeOf(Request{}).PkgPath()`: the import path of the compiled helper package (`pricefollower.local/internal/amazon/amazontest`), so it cannot go stale when the package moves or the module is renamed. `TestHelperImportPath` compares it with the expected literal and fails when the real path changes, instead of the check silently never matching (FR-TEST-HYG-012).
- `listServerDependencies(dir string, env []string) (string, error)` runs `exec.Command("go", "list", "-deps", "./cmd/...")` with `Dir = dir` and `Env = append(os.Environ(), env...)`, and returns its standard output (`cmd.Output()`) unchanged. On error it returns the output together with an error that holds the Go tool's message:
  - `*exec.ExitError`: `fmt.Errorf("go list -deps ./cmd/...: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))`, so the error holds the exit status and the tool's standard error (for example `exit status 1: cmd/app/main.go:3:8: package nope/x is not in std`);
  - any other error (the `go` command cannot be found or started): `fmt.Errorf("go list -deps ./cmd/...: %w", err)`, for example `exec: "go": executable file not found in $PATH`.
  - No `-e` and no `-test` flag: a load error makes the tool exit non-zero, and `_test.go` imports are not listed. Blank (`_`) and renamed imports are listed like any other import.
  - `go` is looked up in the test process's `PATH` (`go test` puts its own `GOROOT/bin` first). The command arguments are constant.
- `checkServerDependencies(listing string, listErr error) error` judges a listing and keeps no state:
  - `listErr != nil`: returns `fmt.Errorf("list the server dependencies: %w", listErr)`, whatever `listing` holds. A partial listing never passes.
  - otherwise it splits `listing` on `"\n"`, trims each line with `strings.TrimSpace`, skips empty lines and compares each line with `helperImportPath` for equality (no prefix or substring match). On the first match it returns `fmt.Errorf("the server depends on the test helper %s: a program under cmd/ imports it, directly or through another package (go list -deps ./cmd/...); only _test.go files may import it", helperImportPath)`; one error however often the path appears.
  - no match: `nil`.

Changes to `internal/amazon/amazontest/amazontest_test.go` (tests stage):

- `TestAssertNoDownload_TC006_ChangeLeavesApplicationUntouched` keeps only its issue #208 part (the `os.Stat` of `../../service/zz_scratch_docs_test.go`, failing unless the error is `fs.ErrNotExist`). The `importPath` constant, the `filepath.WalkDir` over `../../../cmd` and `../../../internal` and the `os.ReadFile`/`strings.Contains` search are removed. Imports that become unused are removed; the file stays `gofmt`-clean.
- The issue #232 tests (TC-001 to TC-005) are unchanged.

New file `internal/amazon/amazontest/server_dependencies_test.go` (`package amazontest`) holds the tests of the check, including the real check on the repository: `TestServerDoesNotDependOnTestHelper` runs `listServerDependencies(filepath.Join("..", "..", ".."), nil)` (the module root, from the package folder where `go test` runs), fails with the error of `checkServerDependencies` when it is not `nil`, and also fails when the listing does not hold `pricefollower.local/cmd/pricefollower` (proof that the server was really listed). It does not call `t.Skip` or `t.Parallel`. Because it is an ordinary test of the package, `go test ./...` runs it on every gate run (FR-TEST-HYG-013) with no extra command or flag.

Refactoring: removing the import scan from the issue #232 test file is the change the feature asks for. Risk: low. The scan's guarantee moves to `TestServerDoesNotDependOnTestHelper`, which is stricter on what reaches the binary (indirect imports) and no longer fails on non-server packages.

## Tiers

- Frontend: not affected. Only Go test and test-support code changes; no view, component, state or asset.
- Backend: not affected at runtime. `internal/amazon/amazontest` is imported only by `_test.go` files, so the server binary does not change; issue #244 makes this a checked guarantee. No service, repository, validation, persistence, migration or concurrency change. `AssertNoDownload` runs after the browser has closed, and the dependency check runs `go list` as a child process of the test, so no application concurrency changes.
- API: not affected. No endpoint, request, response or error changes; `API_SPECIFICATION.md` is unchanged.

## Technical IDs

| ID | Requirement | Design |
| --- | --- | --- |
| TS-TEST-HYG-001 | FR-TEST-HYG-001 | The tests stage lists `internal/service/zz_scratch_docs_test.go` in `delete.txt`; the pipeline deletes it. |
| TS-TEST-HYG-002 | FR-TEST-HYG-002 | No replacement code: the deleted file defined nothing, so `go build`, `go vet` and `go test` of `./...` are unaffected. |
| TS-TEST-HYG-003 | FR-TEST-HYG-003 | No other path outside `doc/` changes; the issue #204 change folder is untouched. Issue #232 does not touch `internal/service/`. |
| TS-TEST-HYG-004 | FR-TEST-HYG-004 | `downloadFile` returns any `os.ReadFile` error other than `fs.ErrNotExist`. The `WalkDir` callback returns it, and `AssertNoDownload` fails with `t.Fatalf`. |
| TS-TEST-HYG-005 | FR-TEST-HYG-005 | The read error is passed on unchanged (`*fs.PathError`: operation, full path, cause) into `list the files of <root>: <err>`. |
| TS-TEST-HYG-006 | FR-TEST-HYG-006 | `downloadFile` maps `fs.ErrNotExist` from `os.ReadFile` to `("", nil)`. |
| TS-TEST-HYG-007 | FR-TEST-HYG-007 | The name rules run before any read and keep their order and reasons. The marker rule, missing roots, non-regular entries and not following links are unchanged. |
| TS-TEST-HYG-008 | FR-TEST-HYG-008 | `AssertNoDownload`'s signature does not change, and its callers are untouched. The diff outside `doc/` is limited to `internal/amazon/amazontest/`. |
| TS-TEST-HYG-009 | FR-TEST-HYG-004–007 (testability) | New internal unit tests in `package amazontest` observe failures with a recording `testing.TB` (see below). |
| TS-TEST-HYG-010 | FR-TEST-HYG-009 | `TestAssertNoDownload_TC006_ChangeLeavesApplicationUntouched` keeps only the issue #208 `os.Stat`; the walk of `cmd/` and `internal/` and the import-path text search are removed. |
| TS-TEST-HYG-011 | FR-TEST-HYG-010 | `listServerDependencies` runs `go list -deps ./cmd/...` (every program under `cmd/`, direct and indirect non-test imports); `checkServerDependencies` fails when a line equals `helperImportPath`. |
| TS-TEST-HYG-012 | FR-TEST-HYG-011 | The failure error is `the server depends on the test helper pricefollower.local/internal/amazon/amazontest: …`. |
| TS-TEST-HYG-013 | FR-TEST-HYG-012 | Exact line match only, and `go list` without `-test`, so test-only and non-server imports pass. A listing error (exit status with the tool's standard error, or the start error) always fails with `list the server dependencies: <error>`, even with partial output. |
| TS-TEST-HYG-014 | FR-TEST-HYG-013 | `TestServerDoesNotDependOnTestHelper` is an ordinary test of `internal/amazon/amazontest`, run by `go test ./...` with no skip, flag or build tag; it lists the module root and also requires `pricefollower.local/cmd/pricefollower` in the listing. The issue #208 check stays in its test. |
| TS-TEST-HYG-015 | FR-TEST-HYG-009–013 (testability) | The listing and the judgement are separate functions: unit tests feed `checkServerDependencies` listings directly, and integration tests run `listServerDependencies` on fixture modules under `t.TempDir()` (see below). |

## Files to change

| File | Stage | Change |
| --- | --- | --- |
| `internal/service/zz_scratch_docs_test.go` | tests (issue #208) | delete (via `delete.txt`) |
| `doc/changes/2026-11-23-0952-issue-208-remove-leftover-empty-test-file-zz-scrat/no-tests.md` | tests (issue #208) | explain why no test is added |
| `internal/amazon/amazontest/amazontest_test.go` | tests (issue #232) | new: unit tests of TC-001 to TC-005 |
| `internal/amazon/amazontest/amazontest.go` | development (issue #232) | `downloadFile` returns `(string, error)`; `AssertNoDownload` callback returns it; doc comments |
| `doc/specifications/test-suite-hygiene/functional.md` | development | set the status of FR-TEST-HYG-004–008 to implemented |
| `internal/amazon/amazontest/amazontest_test.go` | tests (issue #244) | remove the import scan from `TestAssertNoDownload_TC006_ChangeLeavesApplicationUntouched` and the imports it leaves unused |
| `internal/amazon/amazontest/server_dependencies_test.go` | tests (issue #244) | new: tests of TC-002 to TC-004 |
| `internal/amazon/amazontest/server_dependencies.go` | development (issue #244) | new: `helperImportPath`, `listServerDependencies`, `checkServerDependencies` |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #244) | set the status of FR-TEST-HYG-009–013 and TS-TEST-HYG-010–015 to implemented |

No production code, frontend, `API_SPECIFICATION.md`, `go.mod`, `go.sum`, `package.json`, `playwright.config.ts`, CI or `scripts/` change. No existing test file other than `internal/amazon/amazontest/amazontest_test.go` changes, and no test file is deleted.

## Test design (issue #232)

`internal/amazon/amazontest/amazontest_test.go`, `package amazontest` (internal, so TC-003 can call `downloadFile`). Standard library only; it must be `gofmt`-clean.

- **Recording TB:** `type recordingTB struct { testing.TB; failed bool; message string }` embedding the real `*testing.T` (so `Helper`, `Name`, `TempDir` etc. work). It overrides `Fatalf`/`Fatal` (record, then `runtime.Goexit()`) and `Errorf`/`Error` (record). A helper `runCheck(t, names, roots...) (failed bool, message string)` runs `AssertNoDownload` in a new goroutine and waits on a channel closed by a `defer`.
- **Unreadable file:** `os.WriteFile`, then `os.Chmod(path, 0)`, with `t.Cleanup` restoring `0o600` (and `0o700` for folders) so `t.TempDir` can be removed. If `os.ReadFile` still succeeds (running as root), `t.Skip("file still readable after chmod 000 (running as root?)")`.
- Expected messages are checked with `strings.Contains`. Read failure: the message contains the full path and `permission denied`, and does not contain `files left by a download`. Downloads: the message contains `files left by a download`, the path and the reason text.
- The red state comes from TC-001: the helper passes today, so the assertions fail. TC-003 calls `downloadFile` with its new two-value result, so the package does not compile until development; this is the expected TDD red.

## Test design (issue #244)

`internal/amazon/amazontest/server_dependencies_test.go`, `package amazontest`, standard library only, `gofmt`-clean. No `t.Parallel` in this file (row h of TC-003 uses `t.Setenv`), and no `t.Skip`: when the `go` tool is missing the tests fail.

- **TC-002 (unit):** a table over `checkServerDependencies(strings.Join(paths, "\n"), listErr)`. Failures are checked with `strings.Contains` on `err.Error()`: `pricefollower.local/internal/amazon/amazontest` and `server depends on` for rows a and the duplicate corner case; the Go tool's text (`exit status 1: cmd/pricefollower/main.go:3:2: package x is not in std`, given as `errors.New`) for rows e and f. Corner cases: the helper path repeated, the listing with surrounding spaces and a trailing `"\n"`, and the same input judged twice with equal results.
- **Fixture module (TC-003):** `writeModule(t, files map[string]string) string` writes, under `t.TempDir()`, `go.mod` = `module pricefollower.local\n\ngo 1.25\n` (no `toolchain` line, no `require`), the stub `internal/amazon/amazontest/amazontest.go` = `package amazontest`, and the given files. The listing runs as `listServerDependencies(dir, fixtureEnv)` with `fixtureEnv = []string{"GOFLAGS=-mod=mod", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off"}`, so no network or toolchain download is needed. Each row passes the result to `checkServerDependencies` and checks pass or failure as in the table.
  - Row b: `internal/service/service.go` imports the helper; `cmd/app/main.go` imports `pricefollower.local/internal/service`.
  - Row e: `internal/httpapi/httpapitest/helper.go` imports the helper; no program imports it.
  - Row g: `go list` reads Go files only up to their imports, so a syntax error inside a function body is not reported. The syntax error must be in the package clause or the import block (for example an unterminated import string `import "fmt`).
  - Row h: `t.Setenv("PATH", "")` in the subtest, then `listServerDependencies`; the error holds `executable file not found`, and `checkServerDependencies` fails.
  - Rows f, g, h check that the failure message holds the tool's or the start error's text, and not `server depends on`.
- **TC-004 (integration on the repository):** `TestServerDoesNotDependOnTestHelper`, as described in the design.
- **Red state:** the new test file calls `listServerDependencies`, `checkServerDependencies` and uses `helperImportPath`, which do not exist before development, so `go vet ./...` and `go test ./...` fail to compile `internal/amazon/amazontest`. This is the expected TDD red.

## Test case mapping (issue #244)

| Test case | Proved by |
| --- | --- |
| TC-001 | Repository check: `internal/amazon/amazontest/amazontest_test.go` has no `filepath.WalkDir` over `../../../cmd` or `../../../internal` and no test searches file contents for the helper's import path (`git grep -n 'pricefollower.local/internal/amazon/amazontest' -- internal/amazon/amazontest/amazontest_test.go` finds nothing; the path appears only in `server_dependencies_test.go` as fixture and listing data, never compared with source files). Suite: `go test -count=1 ./internal/amazon/amazontest/...`, including `TestAssertNoDownload_TC006_ChangeLeavesApplicationUntouched` (issue #208 part) and the issue #232 tests TC-001 to TC-005 |
| TC-002 | `internal/amazon/amazontest/server_dependencies_test.go`: `TestCheckServerDependencies_TC002_JudgesListing` (rows a–f and the corner cases) |
| TC-003 | `internal/amazon/amazontest/server_dependencies_test.go`: `TestListServerDependencies_TC003_FixtureModules` (rows a–h; corner cases: no helper import passes, blank and renamed imports fail) |
| TC-004 | `internal/amazon/amazontest/server_dependencies_test.go`: `TestServerDoesNotDependOnTestHelper`, run by the unit-test gate (`go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`); corner cases by `go test -count=1 ./...` and `go test -v ./internal/amazon/amazontest/...` showing it and the issue #208 test as run |
| TC-005 | Repository check in a scratch worktree, never committed: the three mutations of `testcases.md`, `go test ./...` after each and after each revert. Mutation 1 passes; mutations 2 and 3 fail in `TestServerDoesNotDependOnTestHelper` with the `server depends on the test helper pricefollower.local/internal/amazon/amazontest` message |
| TC-006 | Repository check on the diff (`git diff --name-status master -- . ':!doc'` lists only `internal/amazon/amazontest/amazontest_test.go`, `internal/amazon/amazontest/server_dependencies_test.go` and `internal/amazon/amazontest/server_dependencies.go`; `internal/service/zz_scratch_docs_test.go` absent) and the gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test` |

## Test case mapping (issue #232, unchanged record)

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/amazon/amazontest/amazontest_test.go`: `TestAssertNoDownloadFailsOnUnreadableFile` (subtests: marker, no marker, nested `Default/Cache/data_1`, second root, clean neighbours, repeated call) |
| TC-002 | `internal/amazon/amazontest/amazontest_test.go`: `TestAssertNoDownloadReportsUnreadableDownloadByName` (`page.html`, `x.crdownload`, `download.html`) |
| TC-003 | `internal/amazon/amazontest/amazontest_test.go`: `TestDownloadFileIgnoresRemovedFile` (`downloadFile` on a removed `Preferences` with or without marker, and under a removed parent, returns `("", nil)`) |
| TC-004 | `internal/amazon/amazontest/amazontest_test.go`: `TestAssertNoDownloadJudgesReadableFiles` (table rows a–j and the corner cases: no roots, several downloads in one message, `page (1).htm`, `page.html.bak`) |
| TC-005 | Unit: `internal/amazon/amazontest/amazontest_test.go`: `TestAssertNoDownloadPassesOnBrowserProfile`. Integration: the existing `internal/amazon/browser_navigation_test.go`, `internal/amazon/browser_interrupted_test.go` and `internal/httpapi/amazon_browser_navigation_test.go`, unchanged, run by `go test ./...` |
| TC-006 | Repository check on the diff (`git diff --name-status master -- . ':!doc'` lists only `internal/amazon/amazontest/`) and the gates: `go vet ./...`, `go test ./...`, `npm run build`, `npx playwright test` |

Issue #208 cases (unchanged record):

| Test case | Proved by |
| --- | --- |
| TC-001 | `git diff --name-status master -- internal/service/zz_scratch_docs_test.go` shows `D`; `git ls-files 'internal/service/zz_scratch*'` is empty |
| TC-002 | `go vet ./...`, `go test ./...` (including `./internal/service/...`) |
| TC-003 | `git diff --name-status master -- . ':!doc'` lists only the deletion |
| TC-004 | `npm run build`, `node --test scripts/*.test.mjs`, `npx playwright test` |

## Manual follow-ups

None. The check runs in the existing `go test ./...` gate, so no CI or `scripts/` change is needed.
