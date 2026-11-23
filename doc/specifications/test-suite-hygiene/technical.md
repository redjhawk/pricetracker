# Test suite hygiene: technical specification

Status: TS-TEST-HYG-001–003 ready (issue #208); TS-TEST-HYG-004–009 ready (issue #232); TS-TEST-HYG-010–015 implemented (issue #244); TS-TEST-HYG-016–020 implemented (issue #251); TS-TEST-HYG-021–024 implemented (issue #241; TS-TEST-HYG-022 is the manual close of issue #238 after the merge); TS-TEST-HYG-025–030 ready (issue #185)
Requirements: [functional](functional.md). Sources: issue #208, issue #232, issue #244, issue #251, issue #241, issue #185.

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

### Fixture modules compile (issue #251)

Only `internal/amazon/amazontest/server_dependencies_test.go` changes. `server_dependencies.go`, `amazontest.go` and the dependency check (`listServerDependencies`, `checkServerDependencies`) are unchanged.

- **Stub declares the name (chosen fix).** In `writeModule`, the stub `internal/amazon/amazontest/amazontest.go` becomes `"package amazontest\n\nconst DownloadMarker = \"\"\n"`. Rows a and c (`helperMain`: `var _ = amazontest.DownloadMarker`) and the renamed-import corner (`var _ = helper.DownloadMarker`) then type-check. The other fixtures do not change. Changing the fixtures to blank imports, the other option in the issue, is rejected: the renamed-import corner must keep a renamed import that uses its name (FR-TEST-HYG-015), and a renamed import that is not used does not compile.
- `DownloadMarker` mirrors the real helper's exported constant. Its value does not matter: the fixtures only refer to it. Rows f and g (missing package, syntax error) are unchanged and stay invalid; the listing fails on them before any declaration of the stub is read.
- **Build check.** New test code proves that every valid fixture compiles, so the fixtures cannot drift again:
  - `buildModule(dir string) error` runs `exec.Command("go", "build", "./...")` with `Dir = dir` and `Env = append(os.Environ(), fixtureEnv...)` (offline: no proxy, no workspace, no toolchain download). On error it returns `fmt.Errorf("go build ./...: %w: %s", err, strings.TrimSpace(string(out)))` with `out` from `CombinedOutput`, so a failure shows the tool's output. The arguments are constant. `./...` always matches at least two packages (the stub and a program), so `go build` writes no binary into the fixture.
  - The fixture table of `TestListServerDependencies_TC003_FixtureModules` moves to a package-level `var fixtureCases = []fixtureCase{…}` (type `fixtureCase struct { name string; files map[string]string; fails string }`), with the same entries, subtest names and `fails` values. The TC-003 test ranges over `fixtureCases` exactly as today, and the new build test ranges over the same table, so it builds each fixture exactly as TC-003 lists it. A fixture is valid when `fails` is `""` or `"dependency"`, and invalid otherwise (rows f and g).
  - Row h (`plainMain`, `go` not reachable) stays in the TC-003 test. The build test builds its module separately, with the normal `PATH`.
- **Delivery.** The fix is test code in the file that the TDD red commit locks. If the tests stage only added the build check (red), the developer could not then change the stub (green gate). So the **tests stage** delivers the whole change: the stub, the table move, `buildModule` and the build test, `gofmt`-clean. Like issue #208, it writes `<change folder>/no-tests.md` explaining that the change is itself test code, so the red-then-green sequence cannot run in the factory. The red state is proved inside the new test instead: its control case (the stub as on `master`, without `DownloadMarker`, with `helperMain`) must fail `go build ./...`. The development stage changes no code. It only sets the statuses of FR-TEST-HYG-014–017 and TS-TEST-HYG-016–020 to implemented.

Refactoring: moving the TC-003 case table from a local variable to the package-level `fixtureCases`. Reason: the build check must use the same fixtures as the listing check, not a copy that could drift. Scope: `server_dependencies_test.go` only, with entries and subtest names unchanged. Risk: low. The maps are only read (`writeModule` copies them), and no test in the package runs in parallel.

### Finished TC-005 follow-up closed (issue #241)

No code changes. The work of the follow-up record is already on `master`: `internal/httpapi/amazon_browser_interrupted_test.go` lines 40–44 hold, after the wait on `api.fixture.flushed` and before `cancel()`, `select { case <-stopped: t.Fatal("the check ended before the server stop: the interruption point was missed"); default: }`.

- **Record deletion (technical specification stage).** Only the stages with the `docs` tool profile (`Edit(doc/**)`: specification, use cases, test cases, technical specification) may change `doc/todo/`. The tests and development stages (`code` profile) may change only `doc/specifications/` under `doc/`, and the factory's `apply_deletions` keeps a file that `delete.txt` lists outside the stage's Edit rules. So this stage lists `doc/todo/issue-233-r1-03.md` in the change folder's `delete.txt`, and the pipeline deletes it in the technical specification commit. `prepare_tests` resets only paths outside `doc/` and `tests/`, so the deletion survives the later stages.
- **Issue #238 (manual follow-up).** No stage holds GitHub credentials, and the pull request body that the factory writes closes only #241 (`Closes #241`). The owner closes #238 by hand once the pull request is merged: `gh issue close 238 --reason completed --comment "Done by PR #236; its follow-up record doc/todo/issue-233-r1-03.md is removed by the pull request of #241."`. Running it again on a closed issue leaves it closed as completed. The issue is neither deleted nor transferred.
- **Unchanged.** TC-005 and its file; every other `doc/todo/` record, including `issue-233-r3-01.md` (the trace of issue #241); the issue #233 technical specification and test cases. The remaining mentions of the deleted path (`functional.md`, the `doc/FUNCTIONAL_SPECIFICATIONS.md` index, the change folder) describe this change, not open work, and stay.
- **Tests stage.** No test code is written: the change deletes a document and closes an issue, and TC-005 already exists and must not change. The stage writes `<change folder>/no-tests.md` giving that reason and where each TC is proved (see the mapping below); the red gate is then skipped.
- **Development stage.** No code. It sets the status of FR-TEST-HYG-018, FR-TEST-HYG-020 and FR-TEST-HYG-021 and of TS-TEST-HYG-021–024 to implemented, and records FR-TEST-HYG-019 as waiting for the manual close of #238. It runs the gates (`go vet ./...`, `go test ./...`, `npm run build`, `npx playwright test`).

### Missing tests of the Amazon human-like checks (issue #185)

Two new Go test files. No production code changes: the three behaviours already exist on `master` and are not altered.

- Missed window: `Service.collectDue` → `skipScheduledCheck` (`internal/service/service.go`). For an Amazon item, when `inDueAmazonWindow(dueAt, now, location)` is false, it stores `nextCheckAt("amazon", now)` and logs `scheduled Amazon check of item <id> missed its window; next check <next in RFC 3339>`. `next` comes from `nextAmazonCheckAt`, which returns UTC, so the logged time ends in `Z`.
- Upgrade redraw: `NewWithOptions` → `scheduleUnscheduledAmazonItems`. `store.AmazonIDsToSchedule` returns every Amazon item while `amazon_request_state.human_checks_redraw = 1`, each one gets `nextCheckAt("amazon", now)`, then `store.ClearAmazonRedraw` sets the mark to 0. The store exposes no getter or setter for the mark, so the test reads and writes it with its own `database/sql` connection (driver `sqlite`, file `<dir>/pricefollower.sqlite`), as `dropRestartMarker` in `amazon_human_checks_test.go` already does. Adding a store accessor would be production code, which FR-TEST-HYG-026 excludes.
- Empty-body error status: `BrowserCollector.Collect` sets `HTTPStatus` from the loaded page before `classifyPage`. 404 and 410 give `unavailable` / `unavailableMessage`. Any other non-2xx status gives `requestFailure` with the reason `Amazon returned HTTP <n>`: `Result` `request_error`, `Message` `amazonCheckFailedMessage`, `SecondHandStatus` `check_error`, `FailureCause` the reason. `printRequestError` writes the reason to standard output (`Amazon collection failed for amazon.fr/<ASIN>: <reason> url=…`). The existing `captureOutput` helper captures the `log` output and standard output together.
- `gofmt`: on `master`, `internal/amazon` has no call written `(t,<name>` without a space (the `browserCheck(t,collector, …)` call of the issue is already formatted). `internal/amazon/browser_test.go` is therefore expected to stay unchanged. The tests stage runs `gofmt -l internal/amazon`. If it lists a `_test.go` file, that file is formatted with `gofmt -w` in the tests stage, as it is locked afterwards. If it lists a non-test file, the stage writes no production change (FR-TEST-HYG-026) and reports the file in `no-tests.md` for the owner.

**Delivery (same as issue #251).** The change consists only of tests of behaviour that already exists, so they pass before any development: the red gate's "the tests pass before implementing" check would always fail. Test files cannot change after the TDD red commit either. So the **tests stage** delivers all the test code, `gofmt`-clean. It writes `<change folder>/no-tests.md` saying that the change is test code for existing behaviour, so the red phase cannot run in the factory, and that the failing state is proved by the mutation checks below. The red gate is then skipped. The development stage writes no code. It sets the statuses of FR-TEST-HYG-022–026 and TS-TEST-HYG-025–030 to implemented, runs the gates and the repository checks of TC-004 and TC-005, and reverts any production change left behind (see the mutation checks).

**Mutation checks (tests stage, never committed).** After writing the tests, the tests stage edits production code once per mutation, runs the named test, and restores the original code with `Edit`. It records the result in `no-tests.md` (test name, failing assertion message):

1. In `skipScheduledCheck`, insert `if !active { return false }` right after `now := s.now()`, so an Amazon item whose window has passed is checked. The TC-001 test must fail: the missed item is checked.
2. In `store.AmazonIDsToSchedule`, drop the `OR (SELECT human_checks_redraw …) = 1` clause. The TC-002 test must fail: the items keep 08:00 on day D+1.
3. In `classifyPage`, return `fmt.Errorf("Amazon returned HTTP %d", 404)` instead of the `page.status` value for non-2xx statuses. The 503 and 429 subtests of the TC-003 test must fail on `FailureCause`.

Afterwards, `git diff --name-only master -- . ':!doc'` must list only the two new `_test.go` files. The development stage checks this again (TC-005) and restores any production file that differs from `master`.

Refactoring: none. The new tests reuse the existing test helpers of their packages (`openHumanStore`, `startHuman`, `insertAt`, `nextCheck`, `newClock`, `amazonRecorder`, `captureLog`, `scheduleDay`, `windowOf`, `localDate`, `paris` in `internal/service`; `errorPageServer`, `amazonUsedOffer`, `testBrowserOptions`, `browserCheck`, `captureOutput`, `fixtureItem`, `unavailableMessage`, `amazonCheckFailedMessage` in `internal/amazon`) without changing them. Existing test files are not edited. `TestBrowserErrorPage_TC001_EmptyUnavailablePageSendsNoUsedOfferRequest` is not extended: the four statuses go in one new table, so the issue #187 test stays as it is.

## Tiers

- Frontend: not affected. Only Go test and test-support code changes; no view, component, state or asset.
- Backend: not affected at runtime. `internal/amazon/amazontest` is imported only by `_test.go` files, so the server binary does not change; issue #244 makes this a checked guarantee. No service, repository, validation, persistence, migration or concurrency change. `AssertNoDownload` runs after the browser has closed, and the dependency check runs `go list` as a child process of the test, so no application concurrency changes.
- API: not affected. No endpoint, request, response or error changes; `API_SPECIFICATION.md` is unchanged.
- Issue #251 touches only a `_test.go` file and the temporary modules it writes under `t.TempDir()`. No frontend, backend, API, database or migration change, and the server binary is the same.
- Issue #241 deletes one file under `doc/todo/` and closes a GitHub issue. Frontend, backend and API: not affected, as no file outside `doc/` changes; `API_SPECIFICATION.md` is unchanged.
- Issue #185 adds two `_test.go` files. Frontend: not affected, as no view, component or Playwright spec changes. Backend: not affected at runtime. No service, store, validation, persistence, migration or concurrency change; the tests run the existing code on a temporary SQLite file and a local `amazontest` server. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.

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
| TS-TEST-HYG-016 | FR-TEST-HYG-014 | The stub written by `writeModule` declares `const DownloadMarker = ""`, so every name the valid fixtures use is declared. |
| TS-TEST-HYG-017 | FR-TEST-HYG-015 | `fixtureCases` keeps every entry, subtest name, file and `fails` value of the TC-003 table. The renamed-import corner keeps `import helper "…/amazontest"` and `helper.DownloadMarker`, and the dependency check and its messages are unchanged. |
| TS-TEST-HYG-018 | FR-TEST-HYG-016 | Rows f and g and row h are unchanged and still go through `assertListingFailure`. The stub change adds only a declaration, which `go list` does not need on those rows. |
| TS-TEST-HYG-019 | FR-TEST-HYG-017 | The diff outside `doc/` is `internal/amazon/amazontest/server_dependencies_test.go` only. `go vet ./...` and `go test ./...` run the new build test with no flag, skip or build tag. |
| TS-TEST-HYG-020 | FR-TEST-HYG-014–016 (testability) | `buildModule` runs `go build ./...` offline on each fixture of the shared `fixtureCases` and on the `plainMain` module. Valid fixtures must build and rows f and g must not. A control module (stub as on `master`, `helperMain`) must fail with `DownloadMarker` in the output. |
| TS-TEST-HYG-021 | FR-TEST-HYG-018 | The technical specification stage lists `doc/todo/issue-233-r1-03.md` in `delete.txt`; the pipeline deletes it (docs profile, `Edit(doc/**)`). No `Status: done` variant. |
| TS-TEST-HYG-022 | FR-TEST-HYG-019 | Manual follow-up: the owner runs `gh issue close 238 --reason completed` after the merge; the factory cannot, as no stage has GitHub credentials and its pull request closes only #241. |
| TS-TEST-HYG-023 | FR-TEST-HYG-020 | `internal/httpapi/amazon_browser_interrupted_test.go` is not edited: its `select` on `stopped` with a `default` branch and the exact message stays before `cancel()`. |
| TS-TEST-HYG-024 | FR-TEST-HYG-021 | The only change outside the change folder and the specifications is the deletion of the record; no file outside `doc/`, no other `doc/todo/` file and no test file changes. |
| TS-TEST-HYG-025 | FR-TEST-HYG-022 | `TestAmazonMissedWindow_Issue185_TC001_SkippedRedrawnAndLogged` runs `collectDue` at 20:00 on a missed item due at 07:30 and a control item due at 19:30. It asserts no collector call for the missed item, a next check in the morning window of the next day, and the exact `missed its window` log line naming that time. |
| TS-TEST-HYG-026 | FR-TEST-HYG-023 | `TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce` sets `human_checks_redraw = 1` with its own SQL connection on a store holding three Amazon items at 08:00 on day D+1. It starts the service at 10:00 on day D and asserts evening-window times and the mark at 0, then restarts with a different seed and asserts that the times are unchanged. |
| TS-TEST-HYG-027 | FR-TEST-HYG-024 | `TestBrowserEmptyErrorStatus_Issue185_TC003_ClassifiedByStatus` runs one subtest per status (404, 410, 503, 429), each against an empty `/dp/B0PRICE001` answer from a new `errorPageServer`. It asserts `Result`, `Message`, exact `HTTPStatus`, and for 503 and 429 also `SecondHandStatus`, the exact `FailureCause` and the captured output. |
| TS-TEST-HYG-028 | FR-TEST-HYG-025 | The tests stage runs `gofmt -l internal/amazon` and `gofmt -w` on any listed test file before the files are locked. `browser_test.go` is already formatted on `master` and stays unchanged unless `gofmt` lists it. |
| TS-TEST-HYG-029 | FR-TEST-HYG-026 | The tests are ordinary tests (no build tag, flag or `t.Parallel`). The browser test skips only through `amazontest.Browser` when no Chromium is found, which does not happen in the gate. The diff outside `doc/` is the two new `_test.go` files. |
| TS-TEST-HYG-030 | FR-TEST-HYG-022–024 (proof of failure) | The tests stage writes `no-tests.md` (the red phase cannot run on existing behaviour) and runs the three mutation checks, reverted before the stage ends, recording their failures there. |

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
| `internal/amazon/amazontest/server_dependencies_test.go` | tests (issue #251) | stub declares `DownloadMarker`; case table moved to `fixtureCases`; new `buildModule` and the build test; header comment lists the issue #251 cases proved outside the file |
| `doc/changes/2026-11-23-1559-issue-251-make-the-tc-003-fixture-modules-compile/no-tests.md` | tests (issue #251) | explain why the red phase is skipped (the change is the locked test code itself) and where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #251) | set the status of FR-TEST-HYG-014–017 and TS-TEST-HYG-016–020 to implemented |
| `doc/todo/issue-233-r1-03.md` | technical specification (issue #241) | delete (via `delete.txt`) |
| `doc/changes/2026-11-23-1635-issue-241-close-the-finished-tc-005-interruption-c/no-tests.md` | tests (issue #241) | explain why no test is written and where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #241) | set the status of FR-TEST-HYG-018, 020, 021 and TS-TEST-HYG-021–024 to implemented; FR-TEST-HYG-019 waits for the manual close of #238 |
| `internal/service/amazon_human_checks_issue185_test.go` | tests (issue #185) | new: TC-001 and TC-002 tests and their two mark helpers |
| `internal/amazon/browser_empty_status_test.go` | tests (issue #185) | new: TC-003 test |
| `internal/amazon/browser_test.go` | tests (issue #185) | only if `gofmt -l internal/amazon` lists it: `gofmt -w` (expected: unchanged) |
| `doc/changes/2026-11-23-1706-issue-185-tests-missing-from-168-amazon-human-like/no-tests.md` | tests (issue #185) | explain why the red phase is skipped (tests of existing behaviour); record the three mutation checks and their failure messages; where TC-004 and TC-005 are proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #185) | set the status of FR-TEST-HYG-022–026 and TS-TEST-HYG-025–030 to implemented |

No production code, frontend, `API_SPECIFICATION.md`, `go.mod`, `go.sum`, `package.json`, `playwright.config.ts`, CI or `scripts/` change. Issues #232 and #244 change no existing test file other than `internal/amazon/amazontest/amazontest_test.go`, and issue #251 changes no test file other than `internal/amazon/amazontest/server_dependencies_test.go`. Issue #241 changes no test file. Issue #185 adds two test files and changes no existing one (except a `gofmt` fix if one is needed). No test file is deleted.

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

## Test design (issue #251)

`internal/amazon/amazontest/server_dependencies_test.go`, `package amazontest`, standard library only (adds `fmt` and `os/exec`), `gofmt`-clean, no `t.Parallel` and no `t.Skip`. The test case IDs of issue #251 overlap those of issue #244 in the same file, so the new test name carries the issue number: `TestFixtureModulesBuild_Issue251_TC001`. The header comment gains an issue #251 block that names the cases proved outside the file (TC-002 by the existing TC-003 test, TC-003 by its rows f–h, TC-004 by the gates).

- **`TestFixtureModulesBuild_Issue251_TC001`:**
  - one subtest per entry of `fixtureCases`, under the entry's name: `buildModule(writeModule(t, tc.files))`. A valid entry fails with `t.Errorf("valid fixture does not build: %v", err)` on error. Rows f and g fail the test if they build (the check is not vacuous);
  - subtest `TC-003 h fixture: plainMain`: the `plainMain` module builds;
  - subtest `control: a program uses a name the stub does not declare`: `writeModule` with `cmd/app/main.go` = `helperMain`, overriding `internal/amazon/amazontest/amazontest.go` with `"package amazontest\n"` (the stub as on `master`; the `files` argument overrides the default files). `buildModule` must fail, and its error must hold `DownloadMarker`. This control is the red state of the change.
- **`TestListServerDependencies_TC003_FixtureModules`:** the same body, ranging over `fixtureCases` instead of the local `cases`. Its results do not change.
- Run: `go test -count=1 -run 'Fixture|TC003' ./internal/amazon/amazontest/...` passes. With the stub reverted to `package amazontest\n` in a scratch copy (never committed), rows a, c and the renamed corner of the build test fail with `undefined: amazontest.DownloadMarker` / `undefined: helper.DownloadMarker`.

## Test design (issue #185)

Both files use only the standard library and the module's own packages, are `gofmt`-clean, and use no `t.Parallel`, as they replace the global `log` output. The test case IDs repeat those of other issues in the same packages, so test names carry the issue number.

`internal/service/amazon_human_checks_issue185_test.go`, `package service`. Day D is `scheduleDay` (Tuesday 5 March 2030, Paris), and `humanConfig` gives the default windows.

- Helpers in this file:
  - `redrawMark(t, dir) int` opens `sql.Open("sqlite", filepath.Join(dir, "pricefollower.sqlite"))` and reads `SELECT human_checks_redraw FROM amazon_request_state WHERE id = 1`.
  - `setRedrawMark(t, dir)` runs `UPDATE amazon_request_state SET human_checks_redraw = 1 WHERE id = 1` and fails unless one row is affected.
  - Both close their connection before returning.
- **`TestAmazonMissedWindow_Issue185_TC001_SkippedRedrawnAndLogged`:**
  - Setup: `openHumanStore`; `insertAt` of `B000001851` at `scheduleDay(7, 30)` (missed) and `B000001852` at `scheduleDay(19, 30)` (control); `now := newClock(scheduleDay(20, 0))`; `recorder := &amazonRecorder{}`; `startHuman(…, recorder, 185)`; `logs := captureLog(t)` (after the start, so start-up lines are not counted); then `service.collectDue(ctx)`.
  - Assertions:
    - `recorder.calls("B000001851") == 0` and `recorder.calls("B000001852") == 1`;
    - `next := nextCheck(t, database, "B000001851")` has `windowOf(next, paris) == morning`, `localDate(next, paris) == "2030-03-06"` and `!next.Equal(scheduleDay(7, 30))`;
    - `logs` contains `fmt.Sprintf("scheduled Amazon check of item %s missed its window; next check %s", "B000001851", next.UTC().Format(time.RFC3339))`;
    - `strings.Count(logs.String(), "missed its window") == 1`, so no such line is logged for `B000001852`.
  - Repeated run: `service.collectDue(ctx)` again at the same clock time. Then `recorder.calls("B000001851")` is still 0, the next check still equals `next`, and the `missed its window` count is still 1.
- **`TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce`:**
  - Setup:
    - `dir := t.TempDir()`; `database := openHumanStore(t, dir)`; `insertAt` of `B000001853`, `B000001854` and `B000001855` at `former := localTime(paris, 2030, time.March, 6, 8, 0)`; `database.Close()`.
    - `setRedrawMark(t, dir)`, then `database = openHumanStore(t, dir)`. The store's one-time restart already ran at the first open, so reopening keeps the mark.
    - Preconditions: `redrawMark` is 1, and each item's `nextCheck` equals `former`.
  - First start: `startHuman(t, database, dir, newClock(scheduleDay(10, 0)), &amazonRecorder{}, 1851)`. For each item, `nextCheck` has `windowOf == evening`, `localDate == "2030-03-05"` and is not `former`. `redrawMark(t, dir) == 0`. The three times are kept in a map.
  - Second start: `service.Close()`, `database.Close()`, `database = openHumanStore(t, dir)`, then `startHuman(…, newClock(scheduleDay(10, 5)), &amazonRecorder{}, 1852)`. The seed differs on purpose: the same seed with the same next window would redraw the same times, and the check would pass even if the start redrew them. Each item's `nextCheck` equals the stored time, and `redrawMark` is still 0.
  - No collector call is expected; the test does not run `collectDue`.

`internal/amazon/browser_empty_status_test.go`, `package amazon`.

- **`TestBrowserEmptyErrorStatus_Issue185_TC003_ClassifiedByStatus`:** a table `{status int; unavailable bool}` with rows 404 and 410 (`true`) and 503 and 429 (`false`), one subtest `"HTTP <n>"` each, run in sequence.
  - Server: `errorPageServer(t, product, amazonUsedOffer(t))`, where `product` sets `Content-Type: text/html; charset=utf-8` and `Cache-Control: no-store`, calls `w.WriteHeader(status)` and writes no body. The collector is `NewBrowserCollector(testBrowserOptions(t, server.URL))`, new in each subtest.
  - Check: `output := captureOutput(t)`; `got := browserCheck(t, collector, fixtureItem("B0PRICE001"))`; `logged := output()`.
  - All rows: `server.Count("/dp/B0PRICE001") >= 1`, and `got.HTTPStatus == status`.
  - 404 and 410: `got.Result == "unavailable"` and `got.Message == unavailableMessage`.
  - 503 and 429:
    - `got.Result == "request_error"`, `got.Message == amazonCheckFailedMessage` and `got.SecondHandStatus == "check_error"`;
    - `got.FailureCause == fmt.Sprintf("Amazon returned HTTP %d", status)` (exact, so a navigation error or `unexpected content type` fails);
    - `logged` contains `fmt.Sprintf("Amazon collection failed for amazon.fr/B0PRICE001: Amazon returned HTTP %d", status)`.
  - Failure messages print `got` with `%+v` and the server's requests, as the existing browser tests do.
- `amazontest.Browser` (through `testBrowserOptions`) skips when no Chromium is found. The gate has one, so the subtests must report `PASS` there (TC-005).

Run: `go test -count=1 -v -run 'Issue185' ./internal/service/ ./internal/amazon/` lists the three tests (and the four subtests) as `PASS`.

## Test case mapping (issue #185)

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/service/amazon_human_checks_issue185_test.go`: `TestAmazonMissedWindow_Issue185_TC001_SkippedRedrawnAndLogged` (control item, repeated run, exact log line with the new time). Mutation check 1, recorded in `no-tests.md` |
| TC-002 | `internal/service/amazon_human_checks_issue185_test.go`: `TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce` (three items, shared time outside the next window, mark cleared, second start with another seed changes nothing). Mutation check 2, recorded in `no-tests.md` |
| TC-003 | `internal/amazon/browser_empty_status_test.go`: `TestBrowserEmptyErrorStatus_Issue185_TC003_ClassifiedByStatus` (subtests `HTTP 404`, `HTTP 410`, `HTTP 503`, `HTTP 429`). Mutation check 3, recorded in `no-tests.md` |
| TC-004 | Repository check: `gofmt -l internal/amazon` prints nothing; `gofmt -w internal/amazon` then leaves `git status` unchanged. Also the red gate's `gofmt -l` on the changed test files |
| TC-005 | Gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`. `go test -count=1 -v -run 'Issue185' ./internal/service/ ./internal/amazon/`, run twice, shows the three tests as `PASS`. Repository check: `git diff --name-only master -- . ':!doc'` lists only `internal/service/amazon_human_checks_issue185_test.go` and `internal/amazon/browser_empty_status_test.go` (plus `internal/amazon/browser_test.go` if `gofmt` needed it) |

## Test case mapping (issue #241)

No test file implements these cases: the tests stage writes `no-tests.md`, and the cases are repository checks and runs of existing suites.

| Test case | Proved by |
| --- | --- |
| TC-001 | Repository check: `test ! -e doc/todo/issue-233-r1-03.md`; `git diff --name-status master -- doc/todo` lists only `D doc/todo/issue-233-r1-03.md`; `issue-233-r1-02.md`, `-r1-04.md`, `-r2-02.md`, `-r3-01.md`, `-r3-02.md` exist unchanged; `git grep -n 'issue-233-r1-03' -- ':!doc/changes' ':!doc/specifications' ':!doc/FUNCTIONAL_SPECIFICATIONS.md' ':!doc/todo/issue-233-r3-01.md'` finds nothing |
| TC-002 | Repository check after the manual close: `gh issue view 238 --json state,stateReason` shows `CLOSED` and `COMPLETED` |
| TC-003 | Repository check: `git diff master -- internal/httpapi/amazon_browser_interrupted_test.go` is empty and the file holds the `select` with `default` and the exact message before `cancel()`. Integration run of the existing, unchanged test: `go test -count=1 -run TestAmazonBrowserInterrupted_TC005 ./internal/httpapi/` |
| TC-004 | Repository check: `git diff --name-status master -- . ':!doc'` is empty. Gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test` (same test list as `master`) |

## Test case mapping (issue #251)

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/amazon/amazontest/server_dependencies_test.go`: `TestFixtureModulesBuild_Issue251_TC001` (valid entries of `fixtureCases` and the `plainMain` module build offline with `fixtureEnv`; rows f and g and the control module do not). The corner cases "run on every `go test ./...`" and "same result twice" are checked by `go test ./...` and `go test -count=1 ./internal/amazon/amazontest/...` |
| TC-002 | `internal/amazon/amazontest/server_dependencies_test.go`: `TestListServerDependencies_TC003_FixtureModules`, unchanged rows a–e and the three corners over `fixtureCases` (same subtest names and expected results). A repository check on the diff confirms that each fixture's files and import forms are unchanged (renamed import still `import helper …` and used) and that only the stub line changed |
| TC-003 | `internal/amazon/amazontest/server_dependencies_test.go`: rows f, g and h of `TestListServerDependencies_TC003_FixtureModules` (`exit status` plus `pricefollower.local/internal/nope` or `main.go`; `executable file not found`; `assertListingFailure` rejects `server depends on`), run with the new stub |
| TC-004 | Repository check: `git diff --name-status master -- . ':!doc'` lists only `M internal/amazon/amazontest/server_dependencies_test.go`, and `internal/service/zz_scratch_docs_test.go` is absent. Gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test` (same test list as `master`), and `go test -count=1 ./internal/amazon/amazontest/...` |

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

- Issues #208 to #251: none. The checks run in the existing `go test ./...` gate, so no CI or `scripts/` change is needed.
- Issue #241: after the merge, the owner closes issue #238 as completed (`gh issue close 238 --reason completed`, see TS-TEST-HYG-022). No test depends on it; TC-002 checks it afterwards.
- Issue #185: none. The tests run in `go test ./...`, and the gate environment already has Chromium for the browser test.
