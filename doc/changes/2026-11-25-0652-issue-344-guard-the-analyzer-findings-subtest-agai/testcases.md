# Test cases: the analyzer-findings subtest proves its fixture has findings (issue #344)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-082 to FR-TEST-HYG-085). There are no use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **The test file:** `internal/amazon/amazontest/server_dependencies_test.go`.
- **The fixture build check:** `buildModule` and `TestFixtureModulesBuild_Issue251_TC001` in the test file.
- **The analyzer-findings subtest:** the subtest `analyzer findings in non-test and test files do not fail the check` of `TestFixtureModulesBuild_Issue251_TC001`, with its fixture (`cmd/app/main.go` with `fmt.Printf("%d\n", "x")` and `lock(m sync.Mutex)`; `cmd/app/main_test.go` with `Testfoo` and the same `Printf` call).
- **Full vet run:** `go vet ./...` with the default analyzers (no analyzer flag), in the subtest's fixture folder, with `fixtureEnv` (`GOFLAGS=-mod=mod GOWORK=off GOTOOLCHAIN=local GOPROXY=off`).
- **Findings guard:** the check, in the analyzer-findings subtest, that the full vet run fails.
- **Guard message:** `the fixture has no analyzer finding, so the subtest proves nothing`.
- **The run:** `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/`.
- **Fault injection:** a temporary edit of the test file, made only for the run of a case, reverted afterwards and never committed.
- The change only touches test code. No production code, API, database or user interface changes and there is no user flow, so there are no integration or e2e cases of the application. The cases are Go test runs (some with fault injection), code checks and the gates. The IDs below are this change's own, not those in the test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-082 | TC-001, TC-002, TC-003, TC-004 | — |
| FR-TEST-HYG-083 | TC-003, TC-004 | — |
| FR-TEST-HYG-084 | TC-001, TC-005 | — |
| FR-TEST-HYG-085 | TC-006 | — |

## Cases

### TC-001: On a normal run the subtest passes and its fixture has analyzer findings

- Covers: FR-TEST-HYG-082, FR-TEST-HYG-084
- Level: unit (Go, `internal/amazon/amazontest`; runs the go command on the fixture module in a temporary folder), plus a manual run
- Given the change applied and the analyzer-findings subtest's fixture unchanged
- When the run executes, and, by hand, the fixture's two files are written to a temporary module (same `go.mod` as `writeModule`) and the full vet run is executed in it
- Then the analyzer-findings subtest passes, with its name unchanged
- And the manual full vet run fails (non-zero exit) and reports findings of the fixture (printf, copylocks, tests)
- And `go build ./...` and `go vet -atomic ./...` succeed on the same module, so the fixture build check passes it
- Corner cases:
  - the guard runs on the same folder as the fixture build check (no second fixture copy that could drift);
  - the full vet run works offline (no network, workspace or toolchain download);
  - repeated runs (`-count=3`) give the same result.

### TC-002: A fixture with no analyzer finding makes the subtest fail with the guard message

- Covers: FR-TEST-HYG-082
- Level: unit (Go, `internal/amazon/amazontest`; fault injection)
- Given the change applied, and the subtest's fixture temporarily stripped of every finding: `fmt.Printf("%d\n", 1)` in both files, `lock` taking a `*sync.Mutex` and called with `&sync.Mutex{}`, and `Testfoo` renamed `TestFoo`
- When the run executes
- Then the analyzer-findings subtest fails with the guard message, and without "valid fixture does not build"
- And, with the same fixture edit on the `master` version of the test file, the subtest passes, which shows the guard catches the vacuous case
- Corner cases:
  - removing only some findings (for example only renaming `Testfoo`) leaves the subtest passing, as the full vet run still fails;
  - the other subtests keep passing during the injection;
  - no fault injection remains in the diff (checked by TC-006).

### TC-003: The guard and the fixture build check are reported independently

- Covers: FR-TEST-HYG-082, FR-TEST-HYG-083
- Level: unit (Go, `internal/amazon/amazontest`; fault injection)
- Given the change applied, and one of these temporary edits at a time:
  1. `vetArgs` set to `vet ./...` (no `-atomic`), so `buildModule` fails on the fixture's findings;
  2. `vetArgs` set to an unknown subcommand (for example `nosuchcmd`), so `buildModule` fails on every fixture whatever its content, together with the finding-free fixture of TC-002;
  3. the fixture given a compile error (for example `var _ int = "x"` in `cmd/app/main.go`) while keeping its findings
- When the run executes
- Then:
  1. the analyzer-findings subtest fails with "valid fixture does not build" and the build failure, and without the guard message;
  2. the analyzer-findings subtest fails and reports both "valid fixture does not build" and the guard message in the same run;
  3. the analyzer-findings subtest fails with "valid fixture does not build"; the full vet run fails too, so no guard message appears
- And, without the temporary edits, the run passes
- Corner cases:
  - neither check stops the subtest (no `t.Fatal` or early return between them), so a failure of the build check does not skip the guard and the reverse;
  - with edit 1, the subtests that hold no analyzer finding keep their results, and the test-file control still fails `go vet ./...` on `amazontest.Nope` (its prefix check follows `vetArgs`); with edit 2, the other valid-fixture subtests fail too, which is expected for that injection;
  - edit 2 also shows the guard does not reuse `vetArgs`: the full vet run still runs and succeeds on the finding-free fixture;
  - no fault injection remains in the diff (checked by TC-006).

### TC-004: The guard runs the full vet run on the same module, and the comments describe it

- Covers: FR-TEST-HYG-082, FR-TEST-HYG-083
- Level: code check of the test file
- Given the change applied
- When the analyzer-findings subtest and the comments of the test file are read
- Then the guard runs `go vet ./...` with no analyzer flag (not `vetArgs`), with `cmd.Dir` set to the folder passed to `buildModule` and `cmd.Env` set to `os.Environ()` plus `fixtureEnv`
- And it reports the guard message with `t.Error` (not `t.Fatal`) only when that command succeeds, after the fixture build check
- And the subtest name, its fixture and `buildModule` (commands `go build ./...` and `go vet -atomic ./...`, results and messages) are unchanged
- And the comment of the subtest or the test function says the guard proves the fixture has analyzer findings; the file header has an issue #344 entry listing the cases proved outside the file (TC-002, TC-003, TC-005, TC-006)
- Corner cases:
  - if the guard also checks vet's output for analyzer messages (optional in the issue), a missing message fails the subtest with a message naming it, and the run of TC-001 still passes on the current toolchain;
  - the guard does not change the folder's content, so the order of the two checks does not matter.

### TC-005: The other subtests and tests keep their names, results and messages

- Covers: FR-TEST-HYG-084
- Level: unit (Go, `internal/amazon/amazontest`) and code check
- Given the `-v` output of the run on `master` and with the change, and the diff of the test file
- When both runs are compared and the diff is read
- Then every subtest name of the `master` run is present, unchanged and passing in the new run, and there is no new subtest
- And rows f and g still pass only on a build failure holding `pricefollower.local/internal/nope` and `main.go`, both controls still pass with their messages, and the messages `valid fixture does not build: ...`, `invalid fixture builds, want a build failure` and `build failure %q does not hold %q` are unchanged
- And `go test -run '^(TestListServerDependencies_TC003_FixtureModules|TestCheckServerDependencies_TC002_JudgesListing|TestServerDoesNotDependOnTestHelper)$' -v -count=1 ./internal/amazon/amazontest/` runs the same subtests with the same results on both
- Corner cases:
  - `fixtureCases`, `plainMain`, `helperMain`, the stub and `fixtureEnv` are unchanged;
  - a fixture with no test file (`plainMain`) still passes.

### TC-006: Only the test file changes outside `doc/`, and the gates pass

- Covers: FR-TEST-HYG-085
- Level: repository check and suite run (deterministic gates: `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` is listed and the gates run, as well as `go test -count=1 ./internal/amazon/amazontest/...`
- Then the only file listed outside `doc/` is `internal/amazon/amazontest/server_dependencies_test.go`, `gofmt -l internal/amazon` prints nothing, and `go vet ./...`, `go test ./...` and `go test -count=1 ./internal/amazon/amazontest/...` pass
- Corner cases:
  - the repository's own `go vet ./...` gate keeps all its analyzers and does not report the fixture's findings, which live only in strings;
  - production Go code, frontend source, `go.mod`, `go.sum`, `package.json`, `tests/`, `API_SPECIFICATION.md`, the issue #340 change folder and `doc/todo/issue-340-r1-01.md` are unchanged, so the operator sees no change in the application;
  - no fault injection from TC-002 or TC-003 remains;
  - the gates give the same result when run twice.
