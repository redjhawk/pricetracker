# Test cases: the analyzer-findings guard requires a finding in each file (issue #349)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-086 to FR-TEST-HYG-089). There are no use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **The test file:** `internal/amazon/amazontest/server_dependencies_test.go`.
- **The fixture build check:** `buildModule` (`go build ./...` and `go vet -atomic ./...`) and `TestFixtureModulesBuild_Issue251_TC001` in the test file.
- **The analyzer-findings subtest:** the subtest `analyzer findings in non-test and test files do not fail the check` of `TestFixtureModulesBuild_Issue251_TC001`, with its fixture: `cmd/app/main.go` with `fmt.Printf("%d\n", "x")` and `lock(m sync.Mutex)` called with `sync.Mutex{}`; `cmd/app/main_test.go` with `Testfoo` and the same `Printf` call.
- **Full vet run:** `go vet ./...` with the default analyzers (no analyzer flag), in the subtest's fixture folder, with `fixtureEnv` (`GOFLAGS=-mod=mod GOWORK=off GOTOOLCHAIN=local GOPROXY=off`).
- **Findings guard:** the checks, in the analyzer-findings subtest, on the full vet run (its result and its output).
- **Module message:** `the fixture has no analyzer finding, so the subtest proves nothing`.
- **main.go message:** `the fixture has no analyzer finding in cmd/app/main.go, so the subtest proves nothing`.
- **main_test.go message:** `the fixture has no analyzer finding in cmd/app/main_test.go, so the subtest proves nothing`.
- **Build message:** `valid fixture does not build: ...`.
- **The run:** `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/`.
- **Fault injection:** a temporary edit of the test file, made only for the run of a case, reverted afterwards and never committed. "The `master` subtest" means the test file as on `master` (the issue #344 guard only).
- **Findings removed from main.go:** `fmt.Printf("%d\n", 1)` in `cmd/app/main.go`, and `lock` taking a `*sync.Mutex` and called with `&sync.Mutex{}`.
- **Findings removed from main_test.go:** `Testfoo` renamed `TestFoo` and its call changed to `fmt.Printf("%d\n", 1)`.
- The change only touches test code. No production code, API, database or user interface changes and there is no user flow, so there are no integration or e2e cases of the application. The cases are Go test runs (some with fault injection), code checks and the gates. The IDs below are this change's own, not those in the test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-086 | TC-001, TC-002, TC-003, TC-004, TC-006 | — |
| FR-TEST-HYG-087 | TC-004, TC-005, TC-006 | — |
| FR-TEST-HYG-088 | TC-006 | — |
| FR-TEST-HYG-089 | TC-001, TC-007, TC-008 | — |

## Cases

### TC-001: On a normal run the subtest passes and vet reports findings in both files

- Covers: FR-TEST-HYG-086, FR-TEST-HYG-089
- Level: unit (Go, `internal/amazon/amazontest`; runs the go command on the fixture module in a temporary folder), plus a manual run
- Given the change applied and the analyzer-findings subtest's fixture unchanged
- When the run executes, and, by hand (or through a temporary log of the guard's output, reverted afterwards), the full vet run is executed on the subtest's fixture module
- Then the analyzer-findings subtest passes, with its name unchanged, and reports none of the module, main.go, main_test.go or build messages
- And the full vet run fails (non-zero exit) and its combined output holds both `cmd/app/main.go:` and `cmd/app/main_test.go:`
- Corner cases:
  - the guard reads the output of the full vet run on the same folder as the fixture build check (no second fixture copy that could drift);
  - the full vet run works offline (no network, workspace or toolchain download);
  - repeated runs (`-count=3`) give the same result.

### TC-002: No finding in the test file makes the subtest fail naming main_test.go

- Covers: FR-TEST-HYG-086
- Level: unit (Go, `internal/amazon/amazontest`; fault injection)
- Given the change applied, and the fixture with the findings removed from main_test.go, `cmd/app/main.go` unchanged
- When the run executes
- Then the analyzer-findings subtest fails with the main_test.go message
- And it reports neither the main.go message, nor the module message (the full vet run still fails on `cmd/app/main.go`), nor the build message
- And, with the same fixture edit on the `master` subtest, the subtest passes, which shows the gap this change closes
- Corner cases:
  - renaming only `Testfoo` to `TestFoo` leaves the subtest passing, as `cmd/app/main_test.go` keeps its `Printf` finding;
  - changing only the `Printf` call of `cmd/app/main_test.go` leaves the subtest passing, as `Testfoo` keeps its tests finding;
  - the position `cmd/app/main.go:` reported for the non-test file does not satisfy the check of `cmd/app/main_test.go:` (and the reverse), as neither is a substring of the other;
  - the other subtests keep passing during the injection;
  - no fault injection remains in the diff (checked by TC-008).

### TC-003: No finding in the non-test file makes the subtest fail naming main.go

- Covers: FR-TEST-HYG-086
- Level: unit (Go, `internal/amazon/amazontest`; fault injection)
- Given the change applied, and the fixture with the findings removed from main.go, `cmd/app/main_test.go` unchanged
- When the run executes
- Then the analyzer-findings subtest fails with the main.go message
- And it reports neither the main_test.go message, nor the module message, nor the build message
- Corner cases:
  - removing only the `Printf` finding or only the copylocks finding of `cmd/app/main.go` leaves the subtest passing, as the file keeps one finding;
  - the other subtests keep passing during the injection;
  - no fault injection remains in the diff (checked by TC-008).

### TC-004: A fixture with no finding at all still fails with the module message

- Covers: FR-TEST-HYG-086, FR-TEST-HYG-087
- Level: unit (Go, `internal/amazon/amazontest`; fault injection)
- Given the change applied, and the fixture with the findings removed from both files
- When the run executes
- Then the analyzer-findings subtest fails with the module message, and without the build message
- And the subtest does not panic or stop before reporting (the empty vet output of a successful run is handled)
- Corner cases:
  - whether the main.go and main_test.go messages also appear in this case is not specified; either outcome is accepted, as long as the module message is reported;
  - the analyzer messages (`wrong type string`, `passes lock by value`, `malformed name`) are not matched, so a toolchain that rewords them gives the same results in TC-001 to TC-004.

### TC-005: The findings guard and the fixture build check stay reported independently

- Covers: FR-TEST-HYG-087
- Level: unit (Go, `internal/amazon/amazontest`; fault injection)
- Given the change applied, and one of these temporary edits at a time:
  1. `vetArgs` set to `vet ./...` (no `-atomic`), so `buildModule` fails on the fixture's findings, together with the findings removed from main_test.go;
  2. `vetArgs` set to an unknown subcommand (for example `nosuchcmd`), so `buildModule` fails on every fixture, together with the findings removed from both files;
  3. `vetArgs` set to `vet ./...` alone, fixture unchanged
- When the run executes
- Then:
  1. the analyzer-findings subtest fails and reports both the build message and the main_test.go message in the same run, and not the main.go message;
  2. the analyzer-findings subtest fails and reports both the build message and the module message in the same run;
  3. the analyzer-findings subtest fails with the build message only, and no guard message
- And, without the temporary edits, the run passes
- Corner cases:
  - no check stops the subtest (no `t.Fatal` or early return between the build check, the module check and the two file checks), so one failure does not hide another;
  - the guard does not reuse `vetArgs`: with edit 2, the full vet run still runs and succeeds on the finding-free fixture;
  - with edits 1 to 3, the test-file control still behaves as recorded for issue #344 (its prefix check follows `vetArgs`); with edit 2 the other valid-fixture subtests fail too, which is expected for that injection;
  - no fault injection remains in the diff (checked by TC-008).

### TC-006: The guard checks the file positions, and its comment names both files

- Covers: FR-TEST-HYG-086, FR-TEST-HYG-087, FR-TEST-HYG-088
- Level: code check of the test file
- Given the change applied
- When the analyzer-findings subtest and the comments of the test file are read
- Then the guard still runs `go vet ./...` with no analyzer flag (not `vetArgs`), with `cmd.Dir` set to the folder passed to `buildModule` and `cmd.Env` set to `os.Environ()` plus `fixtureEnv`, and captures its combined output (standard output and standard error)
- And it reports the module message with `t.Error` when that command succeeds, and checks `cmd/app/main.go:` and `cmd/app/main_test.go:` each in its own check, reporting the main.go or main_test.go message with `t.Error` (not `t.Fatal`), one check not depending on the other's result
- And no analyzer message text is matched, only the two file positions
- And the comment above the guard says that it requires an analyzer finding in `cmd/app/main.go` and in `cmd/app/main_test.go`, so that the subtest proves the fixture build check ignores findings in non-test files and in test files; no comment in the test file (subtest, test function, header) says or implies that only the module as a whole is guarded
- And the file header has an issue #349 entry listing the cases proved outside the file (TC-002, TC-003, TC-004, TC-005, TC-007, TC-008)
- Corner cases:
  - the subtest name, its fixture, `buildModule` and `vetArgs` are unchanged;
  - the guard does not change the folder's content, so the order of the build check and the guard does not matter.

### TC-007: The other subtests and tests keep their names, results and messages

- Covers: FR-TEST-HYG-089
- Level: unit (Go, `internal/amazon/amazontest`) and code check
- Given the `-v` output on `master` and with the change of `go test -run '^(TestFixtureModulesBuild_Issue251_TC001|TestListServerDependencies_TC003_FixtureModules|TestCheckServerDependencies_TC002_JudgesListing|TestServerDoesNotDependOnTestHelper)$' -v -count=1 ./internal/amazon/amazontest/`, and the diff of the test file
- When both runs are compared and the diff is read
- Then every test and subtest name of the `master` run is present, unchanged and passing in the new run, and there is no new or removed subtest
- And rows f and g still pass only on a build failure holding `pricefollower.local/internal/nope` and `main.go`, both controls still pass with their messages, and the build message, `invalid fixture builds, want a build failure`, `build failure %q does not hold %q` and the module message are unchanged
- Corner cases:
  - `fixtureCases`, `plainMain`, `helperMain`, the stub, `writeModule` and `fixtureEnv` are unchanged;
  - a fixture with no test file (`plainMain`) still passes, as the file checks apply only to the analyzer-findings subtest.

### TC-008: Only the test file changes outside `doc/`, and the gates pass

- Covers: FR-TEST-HYG-089
- Level: repository check and suite run (deterministic gates: `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` is listed and the gates run, as well as `go test -count=1 ./internal/amazon/amazontest/...`
- Then the only file listed outside `doc/` is `internal/amazon/amazontest/server_dependencies_test.go`, `gofmt -l internal/amazon` prints nothing, and `go vet ./...`, `go test ./...` and `go test -count=1 ./internal/amazon/amazontest/...` pass
- Corner cases:
  - the repository's own `go vet ./...` gate keeps all its analyzers and does not report the fixture's findings, which live only in strings;
  - production Go code, frontend source, `go.mod`, `go.sum`, `package.json`, `tests/` and `API_SPECIFICATION.md` are unchanged, so the operator sees no change in the application;
  - no fault injection from TC-002 to TC-005 remains;
  - the gates give the same result when run twice.
