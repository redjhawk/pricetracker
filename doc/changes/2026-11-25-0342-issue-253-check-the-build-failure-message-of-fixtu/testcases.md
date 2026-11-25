# Test cases: check the build failure message of fixture rows f and g (issue #253)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-070 to FR-TEST-HYG-073). There are no use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **The test file:** `internal/amazon/amazontest/server_dependencies_test.go`.
- **The fixture build check:** `buildModule` and `TestFixtureModulesBuild_Issue251_TC001` in the test file.
- **The listing test:** `TestListServerDependencies_TC003_FixtureModules`. **The dependency check:** `listServerDependencies` and `checkServerDependencies`.
- **Row f:** `TC-003 f: a program imports a package that does not exist`, expected text `pricefollower.local/internal/nope`. **Row g:** `TC-003 g: a program has a syntax error`, expected text `main.go`. Both texts are the `fails` values of the rows in `fixtureCases`.
- **Offline:** `GOFLAGS=-mod=mod GOWORK=off GOTOOLCHAIN=local GOPROXY=off` (`fixtureEnv`), with no network access needed.
- **Fault injection:** a temporary edit of the test file, made only for the run of a case, reverted afterwards and never committed.
- The change only touches test code. No production code, API, database or user interface changes, and there is no user flow, so there are no integration or e2e cases of the application. The cases are Go test runs (some with fault injection), code checks and the gates. The IDs below are this change's own, not those in the test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-070 | TC-001, TC-002 | — |
| FR-TEST-HYG-071 | TC-002, TC-003 | — |
| FR-TEST-HYG-072 | TC-001, TC-004, TC-005 | — |
| FR-TEST-HYG-073 | TC-006 | — |

## Cases

### TC-001: Rows f and g fail to build with their expected text, and their subtests pass

- Covers: FR-TEST-HYG-070, FR-TEST-HYG-072
- Level: unit (Go, `internal/amazon/amazontest`; runs the go command on fixture modules in temporary folders)
- Given the change applied
- When, by hand, the fixtures of rows f and g are each written to a temporary folder and `go build ./...` runs in it offline, and `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/` runs
- Then `go build ./...` fails on both, its output holds `pricefollower.local/internal/nope` for row f and `main.go` for row g, and the subtests of rows f and g pass
- Corner cases:
  - the error returned by `buildModule` (not only the raw tool output) holds the text, since it is what the subtest checks;
  - the run needs no network (`GOPROXY=off`), no workspace and no toolchain download;
  - repeated runs (`-count=3`) give the same result.

### TC-002: The subtest checks the build failure against the row's own expected text

- Covers: FR-TEST-HYG-070, FR-TEST-HYG-071
- Level: code check of `TestFixtureModulesBuild_Issue251_TC001`
- Given the change applied
- When the branch of the test that handles rows f and g is read
- Then it fails the subtest when `buildModule` returns no error, and, when it returns an error, it checks that the error holds the row's `fails` value (`tc.fails`); on a mismatch it reports a message naming the expected text and holding the build failure
- Corner cases:
  - the expected text comes from the row, not from a text written once for both rows, so row f cannot pass on `main.go` alone;
  - the branch applies only to rows whose `fails` is neither `""` nor `"dependency"`; the valid rows are not checked for a text.

### TC-003: A failure without the expected text, or a fixture that builds, fails the subtest

- Covers: FR-TEST-HYG-071
- Level: unit (Go, `internal/amazon/amazontest`; fault injection)
- Given the change applied, and one of these temporary edits at a time:
  1. `buildModule` runs an unknown go subcommand (for example `go nosuchcmd ./...`) instead of `go build ./...`, so every fixture fails for a reason unrelated to the fixture;
  2. row f's expected text in `fixtureCases` changed to a text its build failure does not hold (for example `pricefollower.local/internal/other`);
  3. the same for row g (for example `other.go`);
  4. row f's fixture replaced by a program that builds (for example `plainMain`)
- When `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/` runs
- Then it fails: with edit 1 on the subtests of rows f and g, with edit 2 on row f, with edit 3 on row g, with edit 4 on row f; for edits 1 to 3 the message names the expected text and holds the build failure, for edit 4 it says the invalid fixture builds
- And, without the temporary edit, it passes
- And, with edit 1 applied to the `master` version of the test file, rows f and g pass, which shows the old check accepted any failure
- Corner cases:
  - edit 1 also makes the valid-fixture subtests fail, and edits 2 to 4 also make the listing test fail on that row; that is expected, and only the row f and g subtests of the fixture build check are judged here;
  - no fault injection remains in the diff (checked by TC-006).

### TC-004: The fixture build check keeps its other subtests, results and messages

- Covers: FR-TEST-HYG-072
- Level: unit (Go, `internal/amazon/amazontest`) and code check
- Given the `-v` output of the fixture build check on `master` and with the change
- When both runs are compared and the diff of the test is read
- Then:
  - both runs hold the same subtest names, with no subtest added or removed, and every subtest passes in both;
  - the valid fixtures (rows a to e, the three corners) and `TC-003 h fixture: plainMain` still pass with the message `valid fixture does not build: ...` on failure;
  - the two control subtests (the `DownloadMarker` program and the test-file control of issue #254) keep their fixtures, checks and messages;
  - `fixtureCases` keeps its entries, names, files and `fails` values, and `buildModule` is unchanged.
- Corner cases:
  - the message for an invalid fixture that builds (`invalid fixture builds, want a build failure`) keeps its meaning;
  - the requirements FR-TEST-HYG-065–067 still hold (row d's test files are compiled, the controls are not vacuous).

### TC-005: The listing test and the dependency check keep their results and messages

- Covers: FR-TEST-HYG-072
- Level: unit (Go, `internal/amazon/amazontest`)
- Given the `-v` output of `go test -run '^(TestListServerDependencies_TC003_FixtureModules|TestCheckServerDependencies_TC002_JudgesListing|TestServerDoesNotDependOnTestHelper)$' -v -count=1 ./internal/amazon/amazontest/` on `master` and with the change
- When both runs are compared
- Then they run the same tests and subtests with the same names, and all pass
- Corner cases:
  - `listServerDependencies`, `checkServerDependencies`, `assertDependsOnHelper`, `assertListingFailure`, `assertPassesCheck`, `writeModule` and `fixtureEnv` are unchanged in the diff;
  - rows f and g still fail the listing with their expected texts, and row h (the go command cannot be started) still fails as a listing failure.

### TC-006: Only the test file changes outside `doc/`, and the gates pass

- Covers: FR-TEST-HYG-073
- Level: repository check and suite run (deterministic gates: `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` is listed and the gates run, as well as `go test -count=1 ./internal/amazon/amazontest/...`
- Then the only file listed outside `doc/` is `internal/amazon/amazontest/server_dependencies_test.go`, `gofmt -l internal/amazon` prints nothing, and `go vet ./...`, `go test ./...` and `go test -count=1 ./internal/amazon/amazontest/...` pass
- Corner cases:
  - production Go code (including the non-test files of `internal/amazon/amazontest/`), frontend source, `go.mod`, `go.sum`, `package.json`, `tests/` and `API_SPECIFICATION.md` are unchanged, so the operator sees no change in the application;
  - no fault injection from TC-003 remains;
  - the gates give the same result when run twice.
