# Test cases: only compile errors fail the fixture build check (issue #340)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-074 to FR-TEST-HYG-077). There are no use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **The test file:** `internal/amazon/amazontest/server_dependencies_test.go`.
- **The fixture build check:** `buildModule` and `TestFixtureModulesBuild_Issue251_TC001` in the test file.
- **The run:** `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/`.
- **Valid fixtures:** the `fixtureCases` entries whose `fails` is `""` or `"dependency"` (rows a to e and the three corners), and `plainMain` (subtest `TC-003 h fixture: plainMain`). Row d is the only one with test files (`cmd/app/main_test.go`, `internal/service/service_test.go`).
- **The test-file control:** the subtest `control: a test file uses a name the stub does not declare`.
- **Analyzer finding:** code that compiles but that a default `go vet` analyzer reports, for example `fmt.Printf("%d\n", "x")` (printf), a function taking a `sync.Mutex` by value (copylocks) or `func Testfoo(t *testing.T)` in a test file (tests).
- **Offline:** `GOFLAGS=-mod=mod GOWORK=off GOTOOLCHAIN=local GOPROXY=off` (`fixtureEnv`).
- **Fault injection:** a temporary edit of the test file, made only for the run of a case, reverted afterwards and never committed.
- The change only touches test code. No production code, API, database or user interface changes and there is no user flow, so there are no integration or e2e cases of the application. The cases are Go test runs (some with fault injection), code checks and the gates. The IDs below are this change's own, not those in the test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-074 | TC-001, TC-002, TC-003 | — |
| FR-TEST-HYG-075 | TC-003, TC-004, TC-005 | — |
| FR-TEST-HYG-076 | TC-006 | — |
| FR-TEST-HYG-077 | TC-007 | — |

## Cases

### TC-001: A valid fixture with an analyzer finding passes the build check

- Covers: FR-TEST-HYG-074
- Level: unit (Go, `internal/amazon/amazontest`; runs the go command on fixture modules in temporary folders)
- Given the change applied, with a new subtest of the fixture build check whose fixture compiles, test files included, but holds analyzer findings: a non-test file calling `fmt.Printf("%d\n", "x")` and passing a `sync.Mutex` by value, and a test file calling `fmt.Printf("%d\n", "x")` and declaring `func Testfoo(t *testing.T)`
- When the run executes
- Then the new subtest passes: `buildModule` returns no error for that fixture
- And, with the same subtest on the `master` version of `buildModule` (plain `go vet ./...`), the subtest fails, which shows it catches the old behaviour
- Corner cases:
  - findings in a non-test file and in a test file are both accepted;
  - the fixture's non-test files and test files compile (`go build ./...` succeeds on it, and so does a type check with no analyzer), so the subtest proves that only an analyzer finding is ignored;
  - the subtest's name is distinct from every existing subtest name, and its fixture is not added to `fixtureCases`, so the listing test is unchanged;
  - if `buildModule` keeps one analyzer turned on to switch off the others (for example `-atomic`), the fixture holds no finding of that analyzer;
  - repeated runs (`-count=3`) give the same result.

### TC-002: Analyzer findings added to existing valid fixtures do not fail their subtests

- Covers: FR-TEST-HYG-074
- Level: unit (Go, `internal/amazon/amazontest`; fault injection)
- Given the change applied, and one of these temporary edits at a time:
  1. row a's `cmd/app/main.go` given `fmt.Printf("%d\n", "x")`, with `fmt` imported;
  2. row d's `cmd/app/main_test.go` given the same call, with `fmt` imported;
  3. row d's `internal/service/service_test.go` given a copied `sync.Mutex` value (for example `var m sync.Mutex; n := m; _ = n`);
- When the run executes
- Then it passes, including the edited row's subtest
- And, with edit 1 or 2 applied to the `master` version of the test file, that row's subtest fails with "valid fixture does not build"
- Corner cases:
  - the edited rows also keep passing `TestListServerDependencies_TC003_FixtureModules` (an analyzer finding is not a dependency);
  - no fault injection remains in the diff (checked by TC-007).

### TC-003: A compile error in a valid fixture, in a non-test or a test file, still fails the build check

- Covers: FR-TEST-HYG-074, FR-TEST-HYG-075
- Level: unit (Go, `internal/amazon/amazontest`; fault injection)
- Given the change applied, and one of these temporary edits at a time:
  1. row d's `cmd/app/main_test.go` replaced by a test file that imports the helper and uses `var _ = amazontest.Nope`;
  2. the same edit in row d's `internal/service/service_test.go`;
  3. row d's `cmd/app/main_test.go` given a syntax error;
  4. row a's `cmd/app/main.go` given a type error (for example `var _ int = "x"`)
- When the run executes
- Then it fails on the edited row's subtest with "valid fixture does not build" and the build failure; for edits 1 and 2 the message names `amazontest.Nope`
- And, without the temporary edit, it passes
- Corner cases:
  - an error in the test file of a non-`main` package (edit 2) is found as well as one in a `main` package (edit 1);
  - a type error with no analyzer finding (edit 4) is still a failure, so narrowing vet did not turn off the type check;
  - no fault injection remains in the diff (checked by TC-007).

### TC-004: The test-file control keeps its name and passes, and the failure comes from the test-file step

- Covers: FR-TEST-HYG-075
- Level: unit (Go, `internal/amazon/amazontest`), code check, and fault injection
- Given the change applied
- When the run executes and the test-file control is read
- Then the subtest `control: a test file uses a name the stub does not declare` is present with that exact name and passes
- And it fails if `buildModule` succeeds on its fixture, if the failure does not name `amazontest.Nope`, or if the failure is not reported by the step that compiles the test files (its prefix is the command of that step, for example `go vet -atomic ./...`, not `go build ./...`)
- And, with `buildModule` temporarily reduced to `go build ./...` only, the control fails
- Corner cases:
  - the prefix the control checks matches the command `buildModule` actually runs for the test files, so the control cannot fail only because the command text changed, nor pass because the failure came from `go build ./...`;
  - the control `control: a program uses a name the stub does not declare` still fails on `DownloadMarker` and passes.

### TC-005: The other subtests keep their names, results and messages

- Covers: FR-TEST-HYG-075
- Level: unit (Go, `internal/amazon/amazontest`) and code check
- Given the `-v` output of the run on `master` and with the change, and the diff of the test file
- When both runs are compared and the diff is read
- Then:
  - every subtest name of the `master` run is present, unchanged and passing in the new run; the only new subtest is the one of TC-001;
  - rows f and g still pass only on a build failure holding `pricefollower.local/internal/nope` and `main.go` (FR-TEST-HYG-070–072);
  - the messages `valid fixture does not build: ...`, `invalid fixture builds, want a build failure`, `build failure %q does not hold %q` and those of the two controls keep their meaning;
  - `fixtureCases`, the fixtures, the stub and their expected texts are unchanged;
  - `go test -run '^(TestListServerDependencies_TC003_FixtureModules|TestCheckServerDependencies_TC002_JudgesListing|TestServerDoesNotDependOnTestHelper)$' -v -count=1 ./internal/amazon/amazontest/` runs the same subtests with the same results on both.
- Corner cases:
  - `buildModule` still runs `go build ./...` first and still returns an error naming the failed command and holding the tool's output;
  - a fixture with no test file (`plainMain`) still passes.

### TC-006: The comments say vet type-checks with test files and its analyzer findings do not fail the check

- Covers: FR-TEST-HYG-076
- Level: code check of the test file
- Given the change applied
- When the comment above `buildModule`, the comment above `TestFixtureModulesBuild_Issue251_TC001` and the file's header comment are read
- Then they say that `go vet` is used to type-check the packages with their test files and that its analyzer findings do not fail the check
- And no comment says or implies that `go vet ./...` with its default analyzers is only a type check
- Corner cases:
  - the comment of `buildModule` names the commands it actually runs, including the analyzer flag if one is used, and why it is used (naming one analyzer turns the others off);
  - the issue #254 line of the header (`go vet ./...`) is corrected or completed so that it does not contradict the new behaviour, and an issue #340 entry lists the cases proved outside the file (TC-002, TC-003, TC-005, TC-007).

### TC-007: Only the test file changes outside `doc/`, and the gates pass

- Covers: FR-TEST-HYG-077
- Level: repository check and suite run (deterministic gates: `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` is listed and the gates run, as well as `go test -count=1 ./internal/amazon/amazontest/...`
- Then the only file listed outside `doc/` is `internal/amazon/amazontest/server_dependencies_test.go`, `gofmt -l internal/amazon` prints nothing, and `go vet ./...`, `go test ./...` and `go test -count=1 ./internal/amazon/amazontest/...` pass
- Corner cases:
  - the repository's own `go vet ./...` gate keeps all its analyzers (no change to CI, scripts or factory gates);
  - production Go code, frontend source, `go.mod`, `go.sum`, `package.json`, `tests/` and `API_SPECIFICATION.md` are unchanged, so the operator sees no change in the application;
  - the analyzer findings of TC-001 live only in fixture strings, so the repository's `go vet ./...` does not report them;
  - no fault injection from TC-002, TC-003 or TC-004 remains;
  - the gates give the same result when run twice.
