# Test cases: type-check fixture test files in buildModule (issue #254)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-065 to FR-TEST-HYG-069). There are no use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **The test file:** `internal/amazon/amazontest/server_dependencies_test.go`.
- **The fixture build check:** `buildModule` and `TestFixtureModulesBuild_Issue251_TC001` in the test file.
- **The dependency check:** `listServerDependencies` and `checkServerDependencies`, run on the fixtures by `TestListServerDependencies_TC003_FixtureModules`.
- **Valid fixtures:** the `fixtureCases` entries whose `fails` is `""` or `"dependency"` (rows a to e and the three corners), and the `plainMain` fixture of the subtest `TC-003 h fixture: plainMain`. **Invalid fixtures:** rows f and g.
- **Offline:** `GOFLAGS=-mod=mod GOWORK=off GOTOOLCHAIN=local GOPROXY=off` (`fixtureEnv`), with no network access needed.
- **Fault injection:** a temporary edit of a fixture in the test file, made only for the run of a case, reverted afterwards and never committed.
- The change only touches test code. No production code, API, database or user interface changes, and there is no user flow, so there are no integration or e2e cases of the application. The cases are Go test runs (some with fault injection), code checks and the gates. The IDs below are this change's own, not those in the test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-065 | TC-001, TC-002, TC-003 | — |
| FR-TEST-HYG-066 | TC-003, TC-004 | — |
| FR-TEST-HYG-067 | TC-005 | — |
| FR-TEST-HYG-068 | TC-006 | — |
| FR-TEST-HYG-069 | TC-007, TC-008 | — |

## Cases

### TC-001: Every valid fixture, test files included, compiles offline

- Covers: FR-TEST-HYG-065
- Level: unit (Go, `internal/amazon/amazontest`; runs the go command on fixture modules in temporary folders)
- Given the change applied
- When `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/` runs, and, by hand, each valid fixture is written to a temporary folder and `go build ./...` and `go vet ./...` run in it offline
- Then the test passes, every valid-fixture subtest passes, and both commands succeed on every valid fixture
- Corner cases:
  - row d, the only fixture with test files (`cmd/app/main_test.go` and `internal/service/service_test.go`), passes;
  - the run needs no network (`GOPROXY=off`), no workspace and no toolchain download, and works with an empty module cache;
  - repeated runs (`-count=3`) give the same result.

### TC-002: The build check compiles the test files, not only the other Go files

- Covers: FR-TEST-HYG-065
- Level: code check of `buildModule`, and unit run
- Given the change applied
- When `buildModule` is read and the row d subtest of the fixture build check runs with `-v`
- Then `buildModule` runs a go command that type-checks `_test.go` files (for example `go vet ./...`) as well as `go build ./...` on the non-test files, offline, and returns an error when either fails; the error holds the tool's output and names the command that failed
- Corner cases:
  - a failure of the first command is not hidden by a success of the second, and the reverse;
  - a fixture with no test file (for example `plainMain`) still passes: a command that finds no test file is not a failure;
  - the go command is run with `fixtureEnv`, like the dependency check.

### TC-003: An error in a test file of row d makes the build check fail

- Covers: FR-TEST-HYG-065, FR-TEST-HYG-066
- Level: unit (Go, `internal/amazon/amazontest`; fault injection)
- Given the change applied, and one of these temporary edits of row d at a time, each leaving its non-test files unchanged:
  1. `cmd/app/main_test.go` replaced by a test file that imports the helper and uses a name the stub does not declare (for example `var _ = amazontest.Nope`);
  2. the same edit in `internal/service/service_test.go` instead;
  3. `cmd/app/main_test.go` given a syntax error
- When `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -count=1 ./internal/amazon/amazontest/` runs
- Then it fails on the row d subtest with each edit, and the failure holds the tool's output (for edits 1 and 2, the undeclared name `Nope`)
- And, without the temporary edit, it passes
- And, with edit 1 applied to the `master` version of the test file, it passes, which shows the old check did not compile the test files
- Corner cases:
  - an error in the test file of a non-`main` package (edit 2) is found as well as one in a `main` package (edit 1);
  - no fault injection remains in the diff (checked by TC-008).

### TC-004: A control fixture with an error only in a test file fails the build check

- Covers: FR-TEST-HYG-066
- Level: unit (Go, `internal/amazon/amazontest`)
- Given the change applied
- When `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/` runs
- Then it runs a new control subtest whose fixture has non-test files that all compile (for example `plainMain` or row d's programs) and one test file that imports the helper and uses a name the stub does not declare; the subtest checks that `buildModule` returns an error and that the error names that undeclared name, and it passes
- Corner cases:
  - the control fixture's non-test files compile on their own: `go build ./...` alone succeeds on it, so the failure can only come from the test file;
  - the subtest fails (not passes) if `buildModule` succeeds on the control fixture, or if the failure does not name the undeclared name (checked by temporarily reverting `buildModule` to `go build ./...` only: the control subtest then fails);
  - the control subtest has a name distinct from every existing subtest name;
  - the control fixture is not added to `fixtureCases`, so the dependency check and its subtests are unchanged (TC-007).

### TC-005: The build check keeps its other results, messages and subtest names

- Covers: FR-TEST-HYG-067
- Level: unit (Go, `internal/amazon/amazontest`) and code check
- Given the change applied, and the `-v` output of the fixture build check on `master` and with the change
- When both runs are compared and the test is read
- Then:
  - every subtest name of the `master` run is present in the new run, unchanged, and the only new subtest is the control of TC-004;
  - rows f and g still expect a failure and fail the build check;
  - `TC-003 h fixture: plainMain` still passes;
  - the control `control: a program uses a name the stub does not declare` still expects a failure and still checks that the message names `DownloadMarker`, and passes;
  - the failure messages of the test (`valid fixture does not build: ...`, `invalid fixture builds, want a build failure`, `the module builds, want a failure on the undeclared DownloadMarker`, `build failure %q does not name DownloadMarker`) keep their meaning.
- Corner cases:
  - an invalid fixture still fails when its error is found by either command;
  - `fixtureCases` keeps its entries, names, files and `fails` values.

### TC-006: The comments say the build check compiles test files

- Covers: FR-TEST-HYG-068
- Level: code check of the test file
- Given the change applied
- When the comment above `buildModule`, the comment above `TestFixtureModulesBuild_Issue251_TC001` and the issue #251 summary in the file's header comment are read
- Then the comments of `buildModule` and of the test say that the fixtures' test files are compiled too
- And no comment in the file says that the check only runs `go build ./...`, or that fixture test files are out of scope or not compiled
- Corner cases:
  - the comment of `buildModule` names the command or commands it actually runs;
  - the comment of the test mentions the new control of TC-004 alongside the `DownloadMarker` control, as the reason the check is not vacuous.

### TC-007: The dependency check keeps its results and messages

- Covers: FR-TEST-HYG-069
- Level: unit (Go, `internal/amazon/amazontest`)
- Given the change applied, and the `-v` output of `go test -run '^(TestListServerDependencies_TC003_FixtureModules|TestCheckServerDependencies_TC002_JudgesListing|TestServerDoesNotDependOnTestHelper)$' -v -count=1 ./internal/amazon/amazontest/` on `master` and with the change
- When both runs are compared
- Then they run the same tests and subtests with the same names, and all pass
- And row d still passes the dependency check: test files are not server dependencies
- Corner cases:
  - `listServerDependencies`, `checkServerDependencies`, `assertDependsOnHelper`, `assertListingFailure`, `assertPassesCheck` and `fixtureEnv` are unchanged in the diff;
  - row h (the go command cannot be started) still fails as a listing failure.

### TC-008: Only the test file changes outside `doc/`, and the gates pass

- Covers: FR-TEST-HYG-069
- Level: repository check and suite run (deterministic gates: `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` is listed and the gates run
- Then the only file listed outside `doc/` is `internal/amazon/amazontest/server_dependencies_test.go`, `gofmt -l internal/amazon` prints nothing, and `go vet ./...` and `go test ./...` pass
- Corner cases:
  - production Go code (including the non-test files of `internal/amazon/amazontest/`), frontend source, `go.mod`, `go.sum`, `package.json`, `tests/` and `API_SPECIFICATION.md` are unchanged, so the operator sees no change in the application;
  - no fault injection from TC-003 or TC-004 remains;
  - the gates give the same result when run twice.
