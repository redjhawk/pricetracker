# Test suite hygiene: technical specification

Status: TS-TEST-HYG-001–003 ready (issue #208); TS-TEST-HYG-004–009 ready (issue #232); TS-TEST-HYG-010–015 implemented (issue #244); TS-TEST-HYG-016–020 implemented (issue #251); TS-TEST-HYG-021–024 implemented (issue #241; TS-TEST-HYG-022 is the manual close of issue #238 after the merge); TS-TEST-HYG-025–030 implemented (issue #185); TS-TEST-HYG-031–034 implemented (issue #238; TS-TEST-HYG-033 replaces the manual close of TS-TEST-HYG-022); TS-TEST-HYG-035–037 implemented (issue #181; the document edit is made by the technical specification stage); TS-TEST-HYG-038–043 implemented (issue #182); TS-TEST-HYG-044–047 implemented (issue #295); TS-TEST-HYG-048–055 implemented (issue #195); TS-TEST-HYG-056–060 implemented (issue #312); TS-TEST-HYG-061–068 implemented (issue #315); TS-TEST-HYG-069–073 implemented (issue #324); TS-TEST-HYG-074–079 implemented (issue #290); TS-TEST-HYG-080–084 implemented (issue #322); TS-TEST-HYG-085–090 implemented (issue #254); TS-TEST-HYG-091–095 implemented (issue #253); TS-TEST-HYG-096–101 implemented (issue #340); TS-TEST-HYG-102–106 implemented (issue #239); TS-TEST-HYG-107–112 implemented (issue #344); TS-TEST-HYG-113–117 implemented (issue #349); TS-TEST-HYG-118–122 implemented (issue #260; the document edit is made by the technical specification stage); TS-TEST-HYG-123–128 implemented (issue #199; the document edit is made by the technical specification stage); TS-TEST-HYG-129–134 implemented (issue #359); TS-TEST-HYG-135–140 implemented (issue #358; the document edits are made by the technical specification stage); TS-TEST-HYG-141–144 implemented (issue #222); TS-TEST-HYG-145–149 implemented (issue #219); TS-TEST-HYG-150–160 implemented (issue #196); TS-TEST-HYG-161–166 implemented (issue #387); TS-TEST-HYG-167–175 implemented (issue #218); TS-TEST-HYG-176–180 implemented, TS-TEST-HYG-146 updated (issue #384)
Requirements: [functional](functional.md). Sources: issue #208, issue #232, issue #244, issue #251, issue #241, issue #185, issue #238, issue #181, issue #182, issue #295, issue #195, issue #312, issue #315, issue #324, issue #290, issue #322, issue #254, issue #253, issue #340, issue #239, issue #344, issue #349, issue #260, issue #199, issue #359, issue #358, issue #222, issue #219, issue #196, issue #387, issue #218.

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

### TC-005 guard kept and issue #238 closed (issue #238)

No code changes. On `master`, `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile` (`internal/httpapi/amazon_browser_interrupted_test.go`) already holds the guard that the issue asks for, added by PR #236:

- lines 35–39: wait on `api.fixture.flushed` (60 s limit, message unchanged);
- lines 40–44: `select { case <-stopped: t.Fatal("the check ended before the server stop: the interruption point was missed"); default: }`, the exact message of FR-TEST-HYG-027, with a `default` branch so it never waits;
- line 45: the first `cancel()` (the server stop), followed by the unchanged 20 s wait, the cache-clear log check, the download and data-file checks and the two Refresh price checks.

`stopped` is closed only by the goroutine that runs `api.items.Collect`, after `Collect` returns, so the guard reads exactly "the check has ended". The draft wording of the issue ("interruption point missed") is not used. The file is not edited.

- **Closing issue #238.** The factory's pull request body holds `Closes #238` (`scripts/factory/factory.sh` writes `Closes #$ISSUE` for the running issue), so the merge into `master` closes #238 with the reason "completed". This settles FR-TEST-HYG-019 and replaces the manual close planned by issue #241 (TS-TEST-HYG-022). No stage needs GitHub credentials. If the issue is already closed as completed, the merge leaves it so.
- **Tests stage.** No test code is written: the guard exists and the change must not touch TC-005. The stage writes `<change folder>/no-tests.md` giving that reason and where each TC is proved (see the mapping below); the red gate is then skipped. It runs the missed-interruption mutation check (below) and records the command and the failure output in `no-tests.md`.
- **Missed-interruption mutation (tests stage, never committed).** In TC-005, right after the `flushed` wait and before the guard, insert `cancel(); <-stopped`, so the check has really ended (stopped with the context, which TC-005 already proves happens within 20 s) when the guard runs. Run `go test -count=1 -run TestAmazonBrowserInterrupted_TC005 ./internal/httpapi/`: it must fail with `the check ended before the server stop: the interruption point was missed` and with none of `the stopped check did not return within 20 s`, `cache clear failure logged` or a download or data-file message. Then restore the file with `Edit` and check `git diff --quiet master -- internal/httpapi/amazon_browser_interrupted_test.go`. A collector change that returns once the download is answered is rejected as the mutation: the guard runs right after `flushed`, so whether the check has ended by then depends on timing, and the check would be flaky. The test-side mutation is deterministic and leaves no `Collect` goroutine running after the failure.
- **Development stage.** No code. It sets FR-TEST-HYG-027–029 and TS-TEST-HYG-031–034 to implemented, and FR-TEST-HYG-019 to "closed by the merge of the issue #238 change". It runs the gates and the repository checks of TC-001 and TC-004, and restores any file outside `doc/` that differs from `master`.

Refactoring: none.

### TC-006 failure statuses match the specification (issue #181)

No code changes. Two lines of the issue #168 test case document (`doc/changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n/testcases.md`) change, so that it matches FR-AMZ-HCHK-012 and the tests already on `master`:

- **TC-006, Given line.** `HTTP 404 for the product page` becomes `HTTP 403 for the product page`. The line then reads: "Given a fixture server answering, in separate sub-tests, HTTP 503, HTTP 403 for the product page, and HTTP 200 with an Amazon robot-check (captcha) page". These are the three sub-tests of `TestBrowserCollector_TC006_NonSuccessAndRobotCheckAreFailures` (`internal/amazon/browser_test.go`: `HTTP 503`, `HTTP 403`, `robot check`). The rest of the entry is unchanged and holds no `404` or `410`.
- **TC-004, Corner cases line.** After the three existing corner cases, unreworded, the line gains `; product page answered with HTTP 404 or HTTP 410 gives "listing unavailable" (an Amazon answer, not a failure)`. This is what `classifyPage` does (404 and 410 → `unavailable` / `unavailableMessage`) and what the tests prove: the `product page answering 404` sub-test of `TestBrowserCollector_TC004_ResultsMatchTheParser`, and for 410 `TestBrowserErrorPage_TC001_EmptyUnavailablePageSendsNoUsedOfferRequest` and `TestBrowserEmptyErrorStatus_Issue185_TC003_ClassifiedByStatus`.

- **Who edits (technical specification stage).** Only the `docs` tool profile (`Edit(doc/**)`: specification, use cases, test cases, technical specification) may edit `doc/changes/`. The tests and development stages (`code` profile) may write only `doc/specifications/` under `doc/`. So this stage makes both edits, as issue #241 did for `doc/todo/`, and they land in the technical specification commit. `prepare_tests` resets only paths outside `doc/` and `tests/`, so the edit survives the later stages.
- **Unchanged.** The issue #168 `tests.lock` (it hashes test and gate files, not `testcases.md`, and the factory checks only the running issue's lock); every other file and test case of the issue #168 folder; `internal/amazon/browser_test.go` and all other code; the amazon-human-checks specifications; `doc/todo/issue-168-r1-04.md`, which stays as it is (FR-TEST-HYG-032).
- **Tests stage.** No test code is written: the change edits a document, and the existing tests already prove the behaviour it describes. The stage writes `<change folder>/no-tests.md` giving that reason and where each TC is proved (see the mapping below); the red gate is then skipped.
- **Development stage.** No code. It sets FR-TEST-HYG-030–032 and TS-TEST-HYG-035–037 to implemented, runs the gates and the repository checks of TC-001 to TC-004, and restores any file outside `doc/` that differs from `master`. If the issue #168 document does not hold the two edits, it cannot fix them (no `doc/changes/` access) and stops with `blocked.md`, `Category: owner`.

Refactoring: none.

### Scheduled checks around a Refresh price (issue #182)

Only Go test files of `internal/service/` change. The production code on `master` already behaves as FR-TEST-HYG-033 and FR-TEST-HYG-035 state:

- `RefreshItem` calls `rescheduleOverdueAmazonCheck` before it reserves the item. When the stored due time of an Amazon item is not after the request time, that function draws a time in the next window (`nextAmazonCheckAt`).
- `store.DueChecks` returns only items with `amazon_stopped_at IS NULL`, so a stopped item keeps its 07:00 due time through the 07:00 run.
- `skipScheduledCheck`, for an Amazon item that is in flight (`active`), stores `nextCheckAt("amazon", now)` and returns `true`. This is the in-flight branch that the old subtest never reached.
- `amazonGate.do` runs a manual check of a stopped item and, on any result other than `request_error`, calls `ClearAmazonStop`.
- Neither a manual check nor `collectReserved` writes `next_check_at`, so after the Refresh price the due time is whatever `RefreshItem` or `skipScheduledCheck` stored.

**TC-032 "Refresh price running"** (`internal/service/amazon_human_checks_test.go`). After `waitStarted(t, recorder.started)` and before `service.collectDue(ctx)`, insert the same step as the "first check after add running" subtest:

```go
if err := database.SetNextCheck(ctx, id, now.Now().Add(-time.Minute)); err != nil {
	t.Fatal(err)
}
```

The due time is then 07:04 while the Refresh price is still blocked, so `collectDue` at 07:05 finds the item due and `skipScheduledCheck` takes the in-flight branch. Nothing else changes: the item, the clock, seed 32, `close(recorder.block)`, `waitIdle`, the `calls(id) == 1` check and `assertScheduled` stay as they are. `assertScheduled` is read after `waitIdle`, and it fails on a past due time (`!next.After(now)`). The "first check after add running" subtest and the `assertScheduled` helper are not edited.

**Restart after stop** (new file `internal/service/amazon_human_checks_issue182_test.go`, `package service`). The test case IDs of issue #182 repeat those of issue #168 in this package, so the name carries the issue number: `TestAmazonRestartAfterStop_Issue182_TC004_NoScheduledCheckAfterRefreshPrice`. It uses only the existing helpers (`openHumanStore`, `insertAt`, `newClock`, `amazonRecorder`, `startHuman`, `stopped`, `nextCheck`, `waitIdle`, `windowOf`, `localDate`, `scheduleDay`, `paris`, `evening`) and none of them changes. Steps:

1. Setup: `dir := t.TempDir()`, `database := openHumanStore(t, dir)`, `ctx := context.Background()`, `const id = "B000001821"`, `insertAt(t, database, id, "amazon", scheduleDay(7, 0))`, then `database.StopAmazonItems(ctx, scheduleDay(7, 0))`. Then `now := newClock(scheduleDay(7, 0))`, `recorder := &amazonRecorder{}` (each check succeeds through `succeeded()`), and `service := startHuman(t, database, dir, now, recorder, 182)`.
2. Preconditions after the start: `stopped(t, database, id)` is true and `nextCheck` equals `scheduleDay(7, 0)`. This proves that the start did not restart or redraw the item.
3. The 07:00 run: `service.collectDue(ctx)`. Then `recorder.total() == 0`, the item is still stopped, and `nextCheck` still equals `scheduleDay(7, 0)`.
4. The 07:30 Refresh price: `now.Set(scheduleDay(7, 30))`, `service.RefreshItem(ctx, id)` (fail on error), `waitIdle(t, service)`. Then `stopped(...)` is false and `recorder.calls(id) == 1`.
5. The 07:31 run: `now.Set(scheduleDay(7, 31))`, then `service.collectDue(ctx)` twice. The second call is the repeated run.
6. Final checks: `recorder.calls(id) == 1` and `recorder.total() == 1`. Then `next := nextCheck(t, database, id)` has `windowOf(next, paris) == evening` and `localDate(next, paris) == "2030-03-05"`. Failure messages print the count and `next.In(paris)`.

No `t.Parallel`, build tag or skip is used. The test does not capture the log.

- **Tests stage.** It writes both test changes and runs `gofmt -l internal/service`, then `gofmt -w` on any listed file it changed. Both tests pass on `master` code because they test shipped behaviour, so the red phase cannot fail. The stage writes `<change folder>/no-tests.md`, which gives that reason, so the red gate is skipped. It then runs the mutation checks below and records each command and failure message in `no-tests.md`. Each mutation is reverted with `Edit`, and `git diff --quiet master -- internal/service/service.go internal/service/amazon_gate.go` must hold before the stage ends.
- **Mutation checks (tests stage, never committed).**
  1. TC-002: in `skipScheduledCheck`, after `now := s.now()`, insert `if active && item.Platform == "amazon" { return true }`. `go test -count=1 -run 'TestAmazonHumanChecks_TC032_NoDuplicateCheckWhileOneIsRunning/Refresh_price_running' ./internal/service/` must fail with `want a later window` (the due time stays 07:04). The stage first runs the same command with the mutation before it adds the `SetNextCheck` step. On the `master` subtest it passes, which shows that the old subtest did not reach the branch.
  2. TC-005 a: in `RefreshItem`, replace `err = s.rescheduleOverdueAmazonCheck(ctx, id, requestedAt)` with `err = nil`. `go test -count=1 -run Issue182 ./internal/service/` must fail on the call count after the 07:31 run (2 checks).
  3. TC-005 b: in `amazonGate.do`, remove the `if stopped { ClearAmazonStop … }` block. The same command must fail on the item still being stopped.
  4. After each revert, the named test passes again.
- **Development stage.** No code. It sets FR-TEST-HYG-033–036 and TS-TEST-HYG-038–043 to implemented, runs the gates and the repository checks of TC-003 and TC-006, and restores any file outside `doc/` that differs from `master` and is not one of the two test files.

Refactoring: none. The restart test is a new test in its own file, not a new subtest of TC-032: its scenario (a stopped item, no check in flight) does not fit TC-032's "no duplicate while running" case. A separate file also leaves the issue #168 file's numbered tests unchanged apart from the one step.

### TC-010 proves the accepted image is requested (issue #295)

Only `tests/platform-tabs.spec.ts` changes, inside the test `TC-010 Amazon list shows the accepted image or the placeholder`. On `master`, its own route handler (lines 324–334) pushes into `imageRequests` only when `request.url() === imageUrl`, records any other image in `api.unexpected` and aborts every other non-local request. So line 361, `expect(imageRequests.every((url) => url === imageUrl)).toBe(true)`, can only pass.

- **Change.** Line 361 is replaced, in place, by:

  ```ts
  await expect.poll(() => imageRequests.length, { message: "the accepted Amazon image was requested" }).toBeGreaterThan(0);
  ```

  Line 362, `expect(api.unexpected).toEqual([])`, stays right after it. Nothing else in the file changes: the route handler, `imageUrl`, the fixtures, `expectThumbnails`, the navigation, reload and 320 px steps, `mockApi` and the `afterEach` check are kept as they are.
- **Why `expect.poll` and not `expect(imageRequests.length)`.** The list renders the thumbnail as `<img … loading="lazy">` (`src/components/TrackedItemsPage.tsx`). `expectThumbnails` checks the `src` attribute, not that the request has reached the Node route handler. The poll retries until the expect timeout of the suite, so the assertion does not depend on that race. It still fails when no request is ever recorded, with the given message. On a normal run the first `page.goto("/")` loads the image, long before line 361, so the poll passes on its first try.
- **Why "greater than zero" and not an exact count.** After `page.reload()` and the second `page.goto("/")` the browser may take the image from its cache without a new routed request (FR-TEST-HYG-038, TC-002 corner case).
- **Other images.** The check that no other image is requested is still `api.unexpected` (line 332 pushes `image <url>`; line 362 and the `afterEach` check fail on it). It covers other URLs on accepted Amazon hosts, as it compares the full URL.

**Delivery (same as issues #251 and #182).** The change is the test code itself, and TC-010 passes on `master` application code both before and after it, so a red phase cannot fail. Test files are locked after the TDD red commit. So the **tests stage** makes the edit above and writes `<change folder>/no-tests.md`. That file says why the red phase is skipped, records the fault injections below and says where each TC is proved. The red gate is then skipped. The development stage writes no code.

**Fault injections (tests stage, never committed).** Each one is a temporary edit of TC-010, then `npx playwright test tests/platform-tabs.spec.ts -g "TC-010"`, then a revert with `Edit`. The command and the failing assertion are recorded in `no-tests.md`:

1. Remove `imageRequests.push(request.url());` from the route handler. The run must fail at the new poll with `the accepted Amazon image was requested`.
2. Give `withImage` `thumbnailUrl: null`. The run must fail, at the `src` check of `expectThumbnails` or at the poll.
3. After the last `expectThumbnails()`, insert a load of another image:

   ```ts
   await page.evaluate((url) => new Promise<void>((done) => { const image = new Image(); image.onload = image.onerror = () => done(); image.src = url; }), "https://images-eu.ssl-images-amazon.com/images/I/other.jpg");
   ```

   The route handler pushes `image https://images-eu.ssl-images-amazon.com/images/I/other.jpg` before it aborts the request, and `onerror` fires only after the abort, so the result does not depend on timing. The run must fail at `expect(api.unexpected).toEqual([])` with that value.

After the last revert, TC-010 passes again (`--repeat-each=3`), and `git diff master -- tests/platform-tabs.spec.ts` shows only the one replaced line.

- **Development stage.** No code. It sets FR-TEST-HYG-037–039 and TS-TEST-HYG-044–047 to implemented. It runs the gates and the repository checks of TC-001 and TC-004, and restores any file outside `doc/` that differs from `master` and is not `tests/platform-tabs.spec.ts`.

Refactoring: none.

### TC-002's fixture sanity check proves the lock is the cause (issue #195)

Only `internal/amazon/browser_profile_test.go` changes. On `master`, `assertBrowserRefusesProfile` (lines 343–366) starts the browser on the locked profile, waits up to 15 s, kills the process group, waits for `Wait`, and fails only when `server.Count(path)` changed. The browser output is captured but used only in the failure message.

**Lock refusal message.** On Linux, Chromium's `ProcessSingleton` finds a `SingletonLock` naming a host other than its own and logs, at `ERROR` level on standard error, its `IDS_PROFILE_IN_USE_POSIX` string: `The profile appears to be in use by another <product> process (<pid>) on another computer (<host>). …`. The lock written by TC-002 is `otherhost-12345`, so the line holds `on another computer (otherhost)`. The check matches exactly that substring, built from the host the test wrote:

```go
// lockRefusal is the part of Chromium's refusal of a profile locked by host on another computer
// that names that host.
func lockRefusal(host string) string { return "on another computer (" + host + ")" }
```

It is case-sensitive and holds the host name, so a crash, a missing library, an unknown option, an empty output, `ProcessSingleton` alone or `Failed to create a ProcessSingleton …` (the generic lock-error line) do not match. The product name and the process ID are left out because they differ between Chromium and Chrome builds and are not what proves the cause.

**Split of the check.** `assertBrowserRefusesProfile` is split into a run and a judgement, so the judgement can be tested without a real browser:

```go
// browserRun is what one run of the browser on the locked profile showed.
type browserRun struct {
	pid       int    // process ID and process group of the run, killed and waited for
	requested bool   // the page was requested while the browser ran
	output    string // standard output and standard error, read after Wait returned
}

// runBrowserOnProfile starts executable on profile with url server.URL+path in its own process group,
// waits until it exits or wait has passed, kills the group and waits for it.
func runBrowserOnProfile(t *testing.T, executable, profile string, server *amazontest.Server, path string, wait time.Duration) browserRun

// fixtureCheckFailure returns why run does not prove that the browser refused the profile because
// host locks it, or "" when it does.
func fixtureCheckFailure(run browserRun, host string) string

// assertBrowserRefusesProfile starts the browser directly on profile and checks that it does not
// open the page and says that host on another computer holds the profile.
func assertBrowserRefusesProfile(t *testing.T, executable, profile, host string, server *amazontest.Server, path string) {
	t.Helper()
	if failure := fixtureCheckFailure(runBrowserOnProfile(t, executable, profile, server, path, 15*time.Second), host); failure != "" {
		t.Fatal(failure)
	}
}
```

- `runBrowserOnProfile` is the body of today's function, unchanged in order: same arguments to the browser, `Setpgid`, one `bytes.Buffer` for both streams, `before := server.Count(path)` read before `Start`, a `Wait` goroutine closing `exited`, `select` on `exited` or the wait, `syscall.Kill(-pid, SIGKILL)`, `<-exited`. Only then are `requested` (`server.Count(path) != before`) and `output` (`output.String()`) read, so the output is complete and the read happens after every write to the buffer (closing `exited` happens after `Wait` returned). The result does not depend on whether the process exited by itself or was killed. A `Start` error stays `t.Fatal(err)`.
- `fixtureCheckFailure` collects the failed conditions in this order, then returns `""` when there is none, otherwise:

  ```go
  fmt.Sprintf("fixture check: %s; browser output (%d bytes):\n%s", strings.Join(failed, "; "), len(run.output), run.output)
  ```

  - `run.requested`: `the browser opened the page on a profile locked by another computer` (today's wording);
  - `!strings.Contains(run.output, lockRefusal(host))`: `fmt.Sprintf("the browser output does not hold the lock refusal message %q", lockRefusal(host))`.

  When both fail, both are named. The byte count makes an empty output visible.
- TC-002 changes only at its call: `assertBrowserRefusesProfile(t, options.ExecPath, options.ProfileDir, "otherhost", server, "/dp/B0PRICE001")`. The `writeStaleLock` lines around it and everything after it stay as they are. The wait stays 15 s for the real browser.

**Permanent unit test of the judgement on stub executables (TC-002, TC-003).** The cases run as an ordinary test with stubs, so the gate proves on every run that the new condition is not dead, and they need no Chromium.

- **Stub.** The package test binary is run again in place of the browser, as the fake browser of issue #233 does, but without changing `TestMain` (`fake_browser_test.go` is outside the allowed diff). `refusalStubBrowser(t, stub refusalStub) string` encodes `stub` as JSON into the variable `PRICEFOLLOWER_REFUSAL_STUB` with `t.Setenv` (inherited by the started process, since `cmd.Env` is nil), and writes under `t.TempDir()` a `/bin/sh` script, mode `0o755` with `// #nosec G306 -- test executable`:

  ```sh
  #!/bin/sh
  exec '<os.Executable()>' '-test.run=^TestBrowserProfile_Issue195_RefusalStubProcess$' -- "$@"
  ```

  The `--` stops the test flags, so the browser arguments do not reach the flag parser. `TestMain` sees no `PRICEFOLLOWER_FAKE_BROWSER` and runs `m.Run`, which runs only the stub test.
- `TestBrowserProfile_Issue195_RefusalStubProcess` returns at once when `PRICEFOLLOWER_REFUSAL_STUB` is empty (a normal run, where it passes). Otherwise it decodes

  ```go
  type refusalStub struct {
  	Request bool   `json:"request"` // GET the last argument (the page URL) and read the body first
  	Stderr  string `json:"stderr"`  // written to standard error, after the request
  	Hold    bool   `json:"hold"`    // then sleep one minute instead of exiting (killed by the check)
  	Exit    int    `json:"exit"`    // exit status otherwise
  }
  ```

  and does what it says, then `os.Exit(stub.Exit)`, so the testing package prints nothing of its own. The request uses an `http.Client` with a 10 s timeout (`// #nosec G107 -- fixture server of the test`). A decode or request error is written to standard error and exits 2, which the cases see as a missing refusal.
- `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal` uses the TC-002 fixture: `amazontest.NewServer(t, profileFixtureHandler(t))` and a profile under `t.TempDir()` prepared by `writeStaleLock(t, profile, "otherhost")`. Each subtest builds the stub, calls `runBrowserOnProfile(t, stub, profile, server, "/dp/B0PRICE001", 2*time.Second)` (since issue #312, 15 s for the rows without `Hold`) and `fixtureCheckFailure(run, "otherhost")`, and asserts the result. The refusal line used by the stubs is Chromium's: `[1:1:ERROR:process_singleton_posix.cc(358)] The profile appears to be in use by another Chromium process (12345) on another computer (otherhost). Chromium has locked the profile so that it doesn't get corrupted.`

  | Subtest | Stub | Expected |
  | --- | --- | --- |
  | `refusal, then exit` | refusal line, exit 21 | `""`; not requested |
  | `refusal, then still running at the end of the wait` | refusal line, hold | `""`; not requested |
  | `exit at once without output` | exit 1 | missing message; output empty (`(0 bytes)`) |
  | `missing library` | `error while loading shared libraries: libnss3.so: cannot open shared object file: No such file or directory`, exit 127 | missing message; output holds the error |
  | `unknown option` | `unknown option --headless=new`, exit 1 | missing message; output holds the error |
  | `slow start` | hold, no output | missing message; output empty |
  | `generic lock error only` | `Failed to create a ProcessSingleton for your profile directory.`, exit 21 | missing message |
  | `refusal naming another host` | the refusal line with `elsewhere` instead of `otherhost`, exit 21 | missing message |
  | `page requested with the refusal` | request, refusal line, exit 0 | opened-page message only; requested |
  | `page requested without output` | request, exit 0 | opened-page and missing messages |

  For every failing row the test asserts that the result starts with `fixture check: `, holds exactly the condition texts of the Expected column (the other one is absent), and ends with `browser output (<len> bytes):\n` followed by the captured output. Rows without `request` also assert `run.requested == false`, which is the case where the check of `master` passed. After every row, `syscall.Kill(-run.pid, 0)` must return `syscall.ESRCH`: no process of the stub's group is left, including the held ones.
- Run time: the two `hold` rows (`refusal, then still running …` and `slow start`) last the 2 s wait each; the others end when the stub exits. The test adds about 5 s to `go test ./internal/amazon`. It uses no `t.Parallel`, as `t.Setenv` requires.

**Delivery (same as issues #251, #182 and #295).** The change is test code in the file that the TDD red commit locks, and the helper must stay in that file (FR-TEST-HYG-043), so the development stage could not write it after the lock. The **tests stage** delivers the whole change above, `gofmt`-clean, and writes `<change folder>/no-tests.md`: why the red phase is skipped, the real-browser runs below with their results, and where each TC is proved. The red gate is then skipped. The development stage writes no code.

**Real-browser runs (tests stage, recorded in `no-tests.md`).**

1. `go test ./internal/amazon -run 'TestBrowserProfile_TC002' -count=1 -v`, then `-count=3`: TC-002 passes every time (TC-004). The stage also records the refusal line the real browser wrote: it adds, temporarily, `t.Log(run.output)` in `assertBrowserRefusesProfile`, runs once, copies the line into `no-tests.md`, and reverts with `Edit`.
2. Fault injection (TC-005): remove the first `writeStaleLock(t, options.ProfileDir, "otherhost")` of TC-002, run it once, record that it fails at the fixture check with `the browser opened the page on a profile locked by another computer` and the output, then revert with `Edit` and run 1 again.

If the real browser's output does not hold `on another computer (otherhost)`, the stage does not loosen the match to a text that a crash or a start error could print. When the output holds a refusal line that names `otherhost` and "another computer" in another wording, the stage sets `lockRefusal` to the part of that line holding both, keeps it built from `host`, updates the stub refusal line to the real one, and records the line in `no-tests.md`; the development stage then updates this section and TS-TEST-HYG-049 to the recorded text. When no such line is written (for example the browser opens the page, or prints only the generic lock error), the lock refusal message of FR-TEST-HYG-040 cannot be observed with this browser: the stage writes `blocked.md` with `Category: owner`, quoting the captured output.

- **Development stage.** No code. It sets FR-TEST-HYG-040–043 and TS-TEST-HYG-048–055 to implemented. It runs the gates and the repository checks of TC-001 and TC-006, and restores any file outside `doc/` that differs from `master` and is not `internal/amazon/browser_profile_test.go`.

Refactoring: splitting `assertBrowserRefusesProfile` into `runBrowserOnProfile` and `fixtureCheckFailure`. Reason: the judgement must be testable with stubs and a short wait (TC-002, TC-003), while TC-002 keeps the 15 s wait. Scope: that function and its one caller in TC-002 (new `host` argument). Risk: low; the run keeps its steps and their order, and the process group is still killed and waited for before the result is read (FR-TEST-HYG-042). `killBrowserOnProfile`, `writeStaleLock`, `stubBrowser`, `assertNothingLeft` and the other tests are not edited.

### Longer wait for the non-hold rows of the fixture check test (issue #312)

Only `internal/amazon/browser_profile_test.go` changes, and only inside the `t.Run` loop of `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal`. On `master` (line 500) every row calls `runBrowserOnProfile(t, executable, profile, server, path, 2*time.Second)`. `runBrowserOnProfile` already returns as soon as the stub exits (`select` on `exited` or `time.After(wait)`), so the wait only bounds a stub that does not exit.

**Wait per row.** The wait depends only on the row's `Hold` field, never on its name or position, so a row added later gets the right wait:

```go
executable := refusalStubBrowser(t, row.stub)
wait := 15 * time.Second // as the fixture sanity check; a stub that exits ends the run at once
if row.stub.Hold {
	wait = 2 * time.Second // a held stub runs until the end of the wait
}
run := runBrowserOnProfile(t, executable, profile, server, path, wait)
```

- Non-hold rows (eight) get 15 s, the wait of `assertBrowserRefusesProfile`, which the test stands for. A stub that makes its request, writes its output and exits within 15 s gives the same `browserRun` as one that does it at once (FR-TEST-HYG-044). The stub's HTTP client timeout (10 s) is below 15 s, so a slow page request ends by itself before the wait does.
- Hold rows (two) keep 2 s, so they take no longer than before (FR-TEST-HYG-045).
- Unchanged: the rows table, `refusalStub`, `refusalStubBrowser`, `TestBrowserProfile_Issue195_RefusalStubProcess`, `runBrowserOnProfile`, `fixtureCheckFailure`, `lockRefusal`, `assertBrowserRefusesProfile` and its 15 s wait, the assertions after the run (output, `requested`, `ESRCH`, failure text) and the other tests (FR-TEST-HYG-046, FR-TEST-HYG-047). On a normal run every non-hold stub exits in well under 2 s, so the test still takes about 5 s (two hold rows of 2 s each).
- No concurrency change: the row still kills the process group and waits for `<-exited` before reading the result, however the run ended.

**Output signal for hold rows: not chosen.** The issue offers it as optional. It would change what the hold stubs do, which the functional specification excludes for issue #312, and FR-TEST-HYG-045 keeps the hold rows' plain 2 s wait. Issue #315 adds it (see below).

**Delivery (same as issue #195).** The change is test code in the file that the TDD red commit locks, so the **tests stage** delivers it, `gofmt`-clean, and writes `<change folder>/no-tests.md`: why the red phase is skipped (the change is the locked test code itself, and a normal run passes before and after it), the runs and fault injections below with their results, and where each TC is proved. The red gate is then skipped. The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** Run command: `go test ./internal/amazon -run '^TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal$' -count=1 -v`. Each injection is a temporary `Edit` of `internal/amazon/browser_profile_test.go`, reverted with `Edit` before the next step; none is committed.

1. Normal run with `-v`, then `-count=3`, then `-race`, then `-run 'TestBrowserProfile' -count=1`: all ten rows pass every time; in the `--- PASS` timings the two hold rows take about 2 s each and every non-hold row less than 2 s (TC-003, TC-004).
2. Slow non-hold stub: in `TestBrowserProfile_Issue195_RefusalStubProcess`, right after the decode, add `if !stub.Hold { time.Sleep(5 * time.Second) }`. The test passes, every non-hold row taking about 5 s. Repeat with 3 s: it passes (TC-002).
3. As 2 (5 s), with the non-hold wait also set back to `2 * time.Second`: at least `TC-001 refusal, then exit` and `TC-003 page requested with the refusal` fail (empty output or page not requested), which shows that the delay is what the longer wait removes (TC-002).
4. Non-hold stub that does not end: replace the sleep of 2 by `if stub.Exit == 21 && strings.Contains(stub.Stderr, "(otherhost)") { time.Sleep(time.Minute) }`, which matches only `TC-001 refusal, then exit`. That row ends after about 15 s and fails with `captured output "", want …` and the missing-message failure; its process group is gone (no `ESRCH` error) and the test does not hang (TC-002 corner case).
5. After the reverts: `git diff master -- internal/amazon/browser_profile_test.go` shows only the wait lines of the `t.Run` loop, and run 1 passes again.

- **Development stage.** No code. It sets FR-TEST-HYG-044–047 and TS-TEST-HYG-056–060 to implemented. It runs the gates and the repository checks of TC-001 and TC-005, and restores any file outside `doc/` that differs from `master` and is not `internal/amazon/browser_profile_test.go`.

Refactoring: none.

### Hold rows' wait starts after the stub's output (issue #315)

Only `internal/amazon/browser_profile_test.go` changes. On `master`, a hold row calls `runBrowserOnProfile(…, 2*time.Second)`, so its 2 s count from `cmd.Start`, and the stub (`/bin/sh`, then the test binary, then the JSON decode) can be killed before `io.WriteString(os.Stderr, stub.Stderr)` runs.

**Output signal: a marker file per row.** A marker file is the simplest signal that crosses the `/bin/sh` `exec` and needs no change to how the process is started (no extra file descriptor, no output parsing).

- `refusalStub` gains ``Signal string `json:"signal"` `` (comment: `created, empty, once Stderr is written, when Hold`). The `Hold` comment becomes `then create Signal and sleep one minute instead of exiting (killed by the check)`.
- `refusalStubBrowser(t, stub refusalStub) (executable, signal string)`: `dir := t.TempDir()` (already used for the script) holds both the script and `signal := filepath.Join(dir, "output-written")`. It sets `stub.Signal = signal` on its copy before `json.Marshal`, so the rows table and `row.stub` (compared with the captured output) are unchanged. Each subtest, and each run under `-count`, gets a new folder, so the marker never exists before its stub starts and an earlier row cannot release a later one.
- `TestBrowserProfile_Issue195_RefusalStubProcess`: in the `if stub.Hold` branch, before `time.Sleep(time.Minute)`, `if err := os.WriteFile(stub.Signal, nil, 0o600); err != nil { fail(err) }`. It runs after the page request (when `Request` is set) and after the `Stderr` write. `os.Stderr` is unbuffered, so the output is in the pipe when the file appears; the parent reads it after the group kill and `<-exited` (`Wait` drains the pipe), so the kill loses none of it. The marker is a file, not output, so the captured output still holds only `Stderr`. An empty `PRICEFOLLOWER_REFUSAL_STUB` still returns at once, writing nothing.

**Run split (refactoring).** `runBrowserOnProfile` becomes a wrapper with the same signature and behaviour:

```go
// runBrowserOnProfileUntil starts executable on profile with url server.URL+path in its own process
// group, calls wait, which returns when the run must end (exited is closed when the process has
// exited), then kills the group and waits for it.
func runBrowserOnProfileUntil(t *testing.T, executable, profile string, server *amazontest.Server, path string, wait func(exited <-chan struct{})) browserRun

func runBrowserOnProfile(t *testing.T, executable, profile string, server *amazontest.Server, path string, wait time.Duration) browserRun {
	t.Helper()
	return runBrowserOnProfileUntil(t, executable, profile, server, path, func(exited <-chan struct{}) {
		select {
		case <-exited:
		case <-time.After(wait):
		}
	})
}
```

`runBrowserOnProfileUntil` is today's body with the `select` replaced by `wait(exited)`: same arguments, `Setpgid`, shared buffer, `before` read before `Start`, `Wait` goroutine, group `SIGKILL`, `<-exited`, then the result. `wait` runs in the caller's goroutine, so no new concurrency. `assertBrowserRefusesProfile` and its 15 s call are not edited.

**Hold wait.** New helper next to `refusalStubBrowser`:

```go
// waitForOutputSignal waits until the file signal exists, the process has exited or bound has passed,
// polling every 10 ms, and reports whether the file exists.
func waitForOutputSignal(signal string, exited <-chan struct{}, bound time.Duration) bool
```

It checks `os.Stat(signal)` (`err == nil` means signalled; any other result keeps polling), then `select`s on `exited` (one last `Stat`, then returns), the `time.After(bound)` deadline (returns `false`) or a 10 ms `time.Ticker` (stopped on return).

In the `t.Run` loop, the `wait` choice still reads only `row.stub.Hold`:

```go
executable, signal := refusalStubBrowser(t, row.stub)
var run browserRun
signalled, held := true, true // a non-hold row has no signal to wait for and no hold to keep
if row.stub.Hold {
	// a held stub runs 2 s after it signalled its output, which it does within 15 s of its start
	run = runBrowserOnProfileUntil(t, executable, profile, server, path, func(exited <-chan struct{}) {
		if signalled = waitForOutputSignal(signal, exited, 15*time.Second); signalled {
			select {
			case <-exited:
				held = false
			case <-time.After(2 * time.Second):
			}
		}
	})
} else {
	run = runBrowserOnProfile(t, executable, profile, server, path, 15*time.Second) // as the fixture sanity check; a stub that exits ends the run at once
}
got := fixtureCheckFailure(run, "otherhost")
if !signalled {
	t.Errorf("the stub did not signal its output within 15 s")
}
if !held {
	t.Errorf("the stub exited before the end of the wait")
}
```

- Still running at the end of the wait (FR-TEST-HYG-049, review R1-01 of #316): a stub that signalled and then exited before the 2 s ended clears `held`, and the row fails with `the stub exited before the end of the wait`. The `ESRCH` check alone cannot tell a killed stub from one that exited by itself.

- The 15 s bound starts when `wait` is called, right after `Start`, so it counts from the stub's start. The 2 s start at the signal, so a slow start does not shorten them (FR-TEST-HYG-048).
- No signal (FR-TEST-HYG-050): a stub still running at 15 s is killed at once, without the 2 s (the row takes about 15 s). A stub that exits without signalling ends the row at once. `t.Errorf` and not `t.Fatalf`, so the existing assertions still run, including the `ESRCH` check of the process group. Other messages can follow (for example `captured output ""`); the signal message is always the first one. A stub that wrote its output but never signalled fails on that message alone.
- Non-hold rows: the same 15 s `runBrowserOnProfile` call as on `master`, only moved into the `else` branch with its comment (FR-TEST-HYG-052).
- Unchanged: the rows table, `fixtureCheckFailure`, `lockRefusal`, `assertBrowserRefusesProfile`, TC-002 of issue #180, and every assertion after the run. On a normal run the signal comes within about 0.1 s, so each hold row takes about 2.1 s (FR-TEST-HYG-051). A row that never signals adds at most about 15 s.

**Delivery (same as issues #195 and #312).** The change is test code in the file that the TDD red commit locks, and a normal run passes before and after it. So the **tests stage** delivers the whole change above, `gofmt`-clean, and writes `<change folder>/no-tests.md`. That file says why the red phase is skipped, records the runs and fault injections below with their results and `--- PASS`/`--- FAIL` timings, and says where each TC is proved. The red gate is then skipped. The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** Run command: `go test ./internal/amazon -run '^TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal$' -count=1 -v`. Each injection is a temporary `Edit` of `internal/amazon/browser_profile_test.go`, reverted with `Edit` before the next step; none is committed.

1. Normal run with `-v`, then `-count=3`, then `-race`, then `-run 'TestBrowserProfile' -count=1`: all ten rows pass every time. Each hold row takes at least 2 s and less than 3 s, and each non-hold row less than 2 s (TC-003, TC-005).
2. Slow hold stub: in `TestBrowserProfile_Issue195_RefusalStubProcess`, right after the decode, add `if stub.Hold { time.Sleep(5 * time.Second) }`. Both hold rows pass with their expected results, taking about 7 s each, and the non-hold rows pass. Repeat with 3 s and 12 s (TC-002).
3. As 2 (5 s), with the hold branch's call temporarily replaced by `run = runBrowserOnProfile(t, executable, profile, server, path, 2*time.Second)` (the `master` wait): `TC-001 refusal, then still running at the end of the wait` fails with `captured output ""` and `the check failed on a refusal without page request` (TC-002).
4. Never signals: as 2 with 20 s. Each hold row fails with `the stub did not signal its output within 15 s`, takes about 15 s and less than 20 s, and reports no process group left. The non-hold rows pass and the test does not hang (TC-004).
5. Exits without signalling: replace the sleep of 2 with `if stub.Hold { os.Exit(0) }`. Each hold row fails with the signal message and ends at once (TC-004).
6. Output but no signal: remove the `os.WriteFile(stub.Signal, …)` lines. Each hold row fails with the signal message after about 15 s. In `TC-001 refusal, then still running …` it is the only error, so a missing signal is never taken for a pass (TC-004).
7. Exits after signalling: add `os.Exit(0)` right after the `os.WriteFile(stub.Signal, …)` lines. Each hold row fails at once with `the stub exited before the end of the wait`, and `TC-002 slow start` fails on that message alone (TC-003).
8. After the reverts: `git diff master -- internal/amazon/browser_profile_test.go` holds only the hunks listed in TS-TEST-HYG-067, and run 1 passes again (TC-006).

- **Development stage.** No code. It sets FR-TEST-HYG-048–052 and TS-TEST-HYG-061–068 to implemented. It runs the gates and the repository checks of TC-001 and TC-006, and restores any file outside `doc/` that differs from `master` and is not `internal/amazon/browser_profile_test.go`.

Refactoring: splitting `runBrowserOnProfile` into the wrapper and `runBrowserOnProfileUntil`. Reason: a hold row must end its run on an event (the signal plus 2 s), not on a fixed time, while the process start, kill and result read stay shared with the fixture sanity check. Scope: that function in `browser_profile_test.go`; its two callers keep their calls (`assertBrowserRefusesProfile`, and the non-hold rows). Risk: low. The wrapper's `wait` does the same `select` as today, so the fixture sanity check behaves the same: 15 s from the browser's start, ending at once when the browser exits. The order of start, wait, kill and `<-exited` before reading the result does not change.

### TC-003 removes its temporary profile however it ends (issue #324)

Only the subtest "TC-003 temporary profile" of `TestBrowserProfile_Issue197_TC003_PreparedProfileReturnsItsPath` in `internal/amazon/browser_profile_values_test.go` changes. On `master`, the `pricefollower-browser-*` directory that `profileOf("")` creates under `os.TempDir()` is removed only by the explicit `cleanup()` call, so a `t.Fatal`/`t.Fatalf` before it (error, nil cleanup, not a directory) leaves it behind.

**Added removal.** Right after `path, cleanup, err := profileOf("")`, before the first check, the subtest registers on its own `t`:

```go
path, cleanup, err := profileOf("")
t.Cleanup(func() { // the temporary profile is removed however the subtest ends, after its checks (issue #324)
	if filepath.Dir(path) != filepath.Clean(os.TempDir()) || !strings.HasPrefix(filepath.Base(path), "pricefollower-browser-") {
		return
	}
	if err := os.RemoveAll(path); err != nil {
		t.Errorf("remove the temporary profile: %v", err)
	}
})
```

- `t.Cleanup` functions run once the subtest function has returned or stopped through `t.Fatal` (`runtime.Goexit`), so the removal always runs and always after every check, including the check that follows `cleanup()`. It therefore cannot make "the cleanup did not remove the temporary profile" pass (FR-TEST-HYG-053, FR-TEST-HYG-054). It is registered on the subtest's `t`, not the parent's, so it runs when this subtest ends.
- It targets only `path`, the value returned by `profileOf("")`; `path` is not reassigned afterwards. It removes `path` only when it passes the subtest's own location check: its parent is `filepath.Clean(os.TempDir())` and its base name starts with `pricefollower-browser-`. Any other path is left alone, so a faulty `profile()` that returned an empty path, the system temporary folder, a persistent profile directory or any other path only fails the location check (`temporary profile %q, want a new directory under %s`) and removes nothing (FR-TEST-HYG-055; review comment R1-01). `filepath.Dir("")` is `.` and `filepath.Dir(os.TempDir())` is its parent, so the guard covers the empty path and the temporary folder itself. No parent, glob or other path is touched.
- On a normal run the collector's cleanup has already removed `path`, and `os.RemoveAll` of a missing path returns `nil`, so the removal adds no failure and no log (FR-TEST-HYG-056). The `t.Errorf` reports only a real removal failure (for example a permission error), which would otherwise be a silent leak; the inner `err` shadows the outer one only inside the closure.
- Unchanged: the explicit `cleanup()` call, the `os.Stat` check after it and its message, the other checks and messages (`cleanup is nil`, `temporary profile %q, want a new directory under %s`, `the temporary profile was not created: %v`), the other subtests, `profileOf` and every helper. No import changes (`os`, `path/filepath` and `strings` are already imported). `BrowserCollector.profile` in `internal/amazon/browser.go` is not edited, and neither is the source check of its body in the same test file.

**Delivery (same as issues #195, #312 and #315).** The change is test code in the file that the TDD red commit locks, and a normal run passes before and after it. So the **tests stage** delivers the change above, `gofmt`-clean, and writes `<change folder>/no-tests.md`: why the red phase is skipped, the runs and fault injections below with their results, and where each TC is proved. The red gate is then skipped. The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** Run command: `go test ./internal/amazon -run '^TestBrowserProfile_Issue197_TC003_PreparedProfileReturnsItsPath$' -count=1 -v`, with `TMPDIR` set to a fresh empty folder made under the stage's own `$TMPDIR` for each run (`mktemp -d`), so the leftover check is `ls -A` of that folder after the run and a deliberate leftover needs no deletion. Each injection is a temporary `Edit` of the test file or of `BrowserCollector.profile` in `internal/amazon/browser.go`, reverted with `Edit` before the next step; none is committed. While `browser.go` is edited, the source check of `profile()` in the same file would fail, which is why the run selects only the TC-003 test.

1. Normal run, then `-count=3`, then `-race`, then `-run 'TestBrowserProfile' -count=1`: every run passes, all subtests of the TC-003 test pass, and the `TMPDIR` folder is empty afterwards (TC-005).
2. In the subtest, right after the `os.Stat`/`IsDir` check, add `os.WriteFile(filepath.Join(path, "keep"), []byte("x"), 0o600)` and `t.Fatal("injected")`: the subtest fails with `injected` and the folder is empty. With the `t.Cleanup` block also commented out (the `master` behaviour), one `pricefollower-browser-*` directory is left (TC-002, edit 1 and the comparison with `master`).
3. In `profile()`, return the created directory with a `nil` cleanup: fails with `cleanup is nil`, folder empty (TC-002, edit 2).
4. In `profile()`, return the created directory with `errors.New("injected")`: fails with `injected`, folder empty (TC-002, edit 3).
5. In `profile()`, create a regular file with `os.CreateTemp("", "pricefollower-browser-")`, close it and return its name with its removal as cleanup: fails with `the temporary profile was not created: <nil>`, folder empty (TC-002, edit 4).
6. In `profile()`, make the temporary profile's cleanup `func() {}`: the subtest fails once, with `the cleanup did not remove the temporary profile: <nil>`, no other message, and the folder is empty afterwards (TC-003).
7. In `profile()`, return `"", func() {}, nil`, then `"", nil, errors.New("injected")`: the subtest fails on its existing checks (location, then `the temporary profile was not created`; then the error). `ls -A internal/amazon` and `ls -A` of the `TMPDIR` folder, taken before and after into files under the stage's `$TMPDIR`, are identical, and both folders still exist (TC-004).
8. After the reverts: `git diff master -- . ':!doc'` holds only the `t.Cleanup` block in the subtest, and run 1 passes again (TC-006).

- **Development stage.** No code. It sets FR-TEST-HYG-053–056 and TS-TEST-HYG-069–073 to implemented. It runs the gates and the repository checks of TC-001 and TC-006, and restores any file outside `doc/` that differs from `master` and is not `internal/amazon/browser_profile_values_test.go` (for example a fault injection left in `browser.go`).

Refactoring: none.

### The 1025-space token test asserts that no save request is sent (issue #290)

Only the test `long whitespace TC-005 a 1025-space token input sends no replacement, shows no message and closes` in `tests/claude-token-settings.spec.ts` changes. On `master`, after `await expect(dialog).toBeHidden()` (line 100):

- line 101, `await expect(page.getByText(/too long|Saving settings…/i)).toHaveCount(0)`: the `Saving settings…` alternative can never match, as `InlineLoading` with that text is rendered only inside the open modal (`src/components/SettingsModal.tsx:211`);
- lines 102–105 loop over `api.puts` and pass when the list is empty;
- line 106, `expect(JSON.stringify(api.puts)).not.toContain(" ".repeat(1025))`, also passes on an empty list.

**Change.** Lines 101–106 become:

```ts
  await expect(page.getByText(/too long/i)).toHaveCount(0);
  expect(api.puts).toEqual([]);
```

- The `Saving settings…` alternative, the loop and the `JSON.stringify` line are removed, not commented out (FR-TEST-HYG-057). The `JSON.stringify` check follows from the empty list, so it is dropped as redundant.
- `expect(api.puts).toEqual([])` is the same assertion as TC-016 of issue #265 (an untouched Save sends nothing). It fails on any recorded body, including `{}` and `{ claudeToken: { remove: true } }`, and reports it (FR-TEST-HYG-058).
- **Why right after `toBeHidden()` and a non-retrying `expect`.** `save()` either closes the modal at once without a request (empty change, `SettingsModal.tsx:128–131`) or closes it only after `await saveSettings(change)` has its answer (`:138–139`). The `mockApi` route pushes the body into `api.puts` before it fulfils the request. So once `toBeHidden()` passes, any request sent by the Save click is already recorded, and none can follow. The assertion runs before the second `openSettings(page)`, so a request of the reopened modal cannot be confused with it; the reopened modal sends only `GET` requests anyway.
- The "too long" check keeps the whole page (`page`, not `dialog`) and the `i` flag (FR-TEST-HYG-058).
- Unchanged: the test name and its comment, the `mockApi` call with `lastRejectedAt: "2026-10-03T20:00:00Z"`, the 1025-space fill, the Save click, `toBeHidden()`, the three reopening checks (FR-TEST-HYG-059), `mockApi`, `openSettings` and every other test of the file, including the 9000-space session test.

**Delivery (same as issue #295).** The change is the test code itself, and the test passes on `master` application code before and after it, so a red phase cannot fail. Test files are locked after the TDD red commit. So the **tests stage** makes the edit above and writes `<change folder>/no-tests.md`: why the red phase is skipped, the runs and fault injections below with their results, and where each TC is proved. The red gate is then skipped. The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** Run command: `npx playwright test tests/claude-token-settings.spec.ts -g "1025-space token input"`. The web server is the Vite dev server (`playwright.config.ts`), so an edit of `src/` applies without a build. Each injection is a temporary `Edit`, reverted with `Edit` before the next step; none is committed.

1. Normal run, then `--repeat-each=3`, then the whole file `tests/claude-token-settings.spec.ts`: every run passes (TC-002).
2. In `save()` of `src/components/SettingsModal.tsx`, one edit at a time (TC-003):
   1. `else if (claudeValue.trim()) change.claudeToken = { value: claudeValue };` → `else if (claudeValue) change.claudeToken = { value: claudeValue };`;
   2. append `else if (claudeValue) change.claudeToken = { remove: true };` after that line;
   3. remove the early return `if (!change.leboncoinSession && !change.claudeToken) { onClose(); return; }`, so `{}` is sent.

   Each run fails at `expect(api.puts).toEqual([])`, with the sent body in the received value, before the reopening.
3. In the test, right after `await expect(dialog).toBeHidden()`, add `await page.evaluate((text) => { const note = document.createElement("p"); note.textContent = text; document.body.append(note); }, "<text>")`, run once each with `Token too long`, `TOO LONG` and `Value is Too Long`: each run fails at `page.getByText(/too long/i)` (TC-004).
4. After the reverts: run 1 passes again, and `git diff master -- . ':!doc'` holds only the hunk of the change above (TC-005).

- **Development stage.** No code. It sets FR-TEST-HYG-057–060 and TS-TEST-HYG-074–079 to implemented. It runs the gates and the repository checks of TC-001 and TC-005, and restores any file outside `doc/` that differs from `master` and is not `tests/claude-token-settings.spec.ts` (for example a fault injection left in `SettingsModal.tsx`).

Refactoring: none.

### The source-text test for TC-005 of issue #197 is removed (issue #322)

Only `internal/amazon/browser_profile_values_test.go` changes, and only by deleting lines. On `master` the file ends with `TestBrowserProfile_Issue197_TC005_ProfileUsesTheExplicitForm` (its two-line comment starts at line 275, the function at line 277), which parses `browser.go` with `go/parser` and compares the `go/format` text of `BrowserCollector.profile` with fixed strings.

**Change.**

- Delete the blank line after the closing `}` of `TestBrowserProfile_Issue197_TC004_UndiscardableLockIsAStartFailure`, the comment `// TC-005 of issue #197: the stale lock branch of profile() has the explicit form and the other` / `// branches keep theirs. ...`, and the whole function up to its closing `}`. The file then ends with the closing `}` of the TC-004 test and one newline. The test is deleted, not skipped, renamed, commented out or moved (FR-TEST-HYG-061).
- Delete the imports `"go/ast"`, `"go/format"`, `"go/parser"` and `"go/token"`. They are used only by the deleted test. The remaining imports `errors`, `io/fs`, `os`, `path/filepath`, `strings` and `testing` are still used by the behaviour tests and their helpers (`strings.Contains` at lines 74, 105 and 256, `strings.HasPrefix` at lines 213 and 226), so they stay. The deleted test declares no package-level helper, variable or constant (its `source` closure is local), so nothing else is removed (FR-TEST-HYG-062).
- Unchanged: the header comment, `assertZeroValues` and the other helpers, the four behaviour tests TC-001 to TC-004 of issue #197 and their subtests, the other test files of `internal/amazon`, and `internal/amazon/browser.go` (FR-TEST-HYG-063, FR-TEST-HYG-064). TC-005 of issue #197 stays covered by the factory gates and review, as the issue #197 technical specification already says (`doc/specifications/amazon-browser-profile/technical.md`, test table, TC-005 row), so that document is not edited.

**Delivery (same as issues #324 and #290).** The change is the test code itself, and it only deletes a test that passes on `master`, so a red phase cannot fail. Test files are locked after the TDD red commit, so the **tests stage** makes the deletion with `Edit` (the file stays; only lines go, so `delete.txt` is not used) and writes `<change folder>/no-tests.md`: why the red phase is skipped, the runs and fault injections below with their results, and where each TC is proved. The red gate is then skipped. The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** Each injection is a temporary edit of `BrowserCollector.profile` in `internal/amazon/browser.go`, reverted before the next step; none is committed. As delivered, the sandbox refused edits of `browser.go`, so each edit was made in a scratch copy that replaced `browser.go` through `go test -overlay`; run 1 also used an overlay copy of the `master` test reading that copy, since the test reads `browser.go` from disk (details in `no-tests.md`).

1. Before the deletion, with injection 2.1 applied: `go test -run 'TestBrowserProfile_Issue197_TC005' -count=1 ./internal/amazon/` fails on the temporary-profile branch text. Revert. This shows the brittleness the change removes (TC-005).
2. After the deletion, one edit at a time, each keeping the returned path, cleanup effect and error; `go test -count=1 ./internal/amazon/` passes with each (TC-005):
   1. `log.Printf("remove temporary browser profile: %v", err)` → `log.Printf("could not remove the temporary browser profile: %v", err)`;
   2. the local variable `directory` renamed to `dir` (declaration, return and `os.RemoveAll`);
   3. the persistent-profile branch moved into an unexported helper `preparePersistentProfile(dir string) (string, func(), error)` holding the same `MkdirAll`, `removeStaleLocks` and return statements, which `profile()` returns directly.
3. After the deletion, a result-changing edit: the `removeStaleLocks` failure returns `c.options.ProfileDir, func() {}, err`. `go test -run 'TestBrowserProfile_Issue197' -count=1 ./internal/amazon/` fails in the TC-001 test (`assertZeroValues`), so behaviour coverage of FR-AMZ-BPROF-008 and FR-AMZ-BPROF-009 is kept (TC-005 corner case).
4. After the reverts: `go test -run 'TestBrowserProfile_Issue197' -v -count=1 ./internal/amazon/`, then with `-count=3`, pass (`-race` needs cgo and `gcc`, which the runner lacks, as for issues #315 and #324); the four behaviour tests and their subtests run, TC-004 is not skipped, and no other `Issue197` test runs (TC-004). `go test -run 'TC005_ProfileUsesTheExplicitForm' -v -count=1 ./internal/amazon/` passes with `testing: warning: no tests to run` (TC-001). `git diff master -- internal/amazon/browser.go` is empty.

- **Development stage.** No code. It sets FR-TEST-HYG-061–064 and TS-TEST-HYG-080–084 to implemented. It runs the gates and the repository checks of TC-001, TC-002, TC-003 and TC-006, and restores any file outside `doc/` that differs from `master` and is not `internal/amazon/browser_profile_values_test.go` (for example a fault injection left in `browser.go`).

Refactoring: none. Removing the test is the change itself.

### The fixture build check compiles the fixtures' test files (issue #254)

Only `internal/amazon/amazontest/server_dependencies_test.go` changes. `server_dependencies.go`, `amazontest.go`, `writeModule`, `fixtureEnv`, `fixtureCases`, the dependency check (`listServerDependencies`, `checkServerDependencies`), `assertDependsOnHelper`, `assertListingFailure`, `assertPassesCheck`, `TestCheckServerDependencies_TC002_JudgesListing`, `TestListServerDependencies_TC003_FixtureModules` and `TestServerDoesNotDependOnTestHelper` are not edited (FR-TEST-HYG-069).

**Command (chosen: `go vet ./...` after `go build ./...`, as the issue suggests).** `go vet ./...` type-checks every package of the fixture with its `_test.go` files (the package, its internal test variant and any external `_test` package) and writes no binary. `go build ./...` is kept first, so the non-test files are still compiled exactly as before and the existing results of rows f and g and of the `DownloadMarker` control do not depend on vet. Rejected: `go test -run '^$' ./...` (also compiles test files, but links one test binary per package, slower, and runs them); `go vet` alone (would work, but changes which command reports the existing failures, for no gain).

**`buildModule(dir string) error`.**

- Runs, in order, `go build ./...` and `go vet ./...`, each with `exec.Command("go", args...)` from a fixed table in the function (`[][]string{{"build", "./..."}, {"vet", "./..."}}`), `Dir = dir` and `Env = append(os.Environ(), fixtureEnv...)` (offline, as the dependency check). The arguments are constant; gosec does not scan `_test.go` files in the gate (no `-tests`), and no `#nosec` is added.
- On the first failing command it returns `fmt.Errorf("go %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))` with `out` from `CombinedOutput`, so the error names the command (`go build ./...` or `go vet ./...`) and holds the tool's output, which names the file, line and undeclared name. It returns `nil` only when both succeed, so neither command's failure is hidden by the other's success. A build failure stops before vet, which would only repeat it.
- A fixture with no test file (for example `plainMain`) passes: vet with no `_test.go` file is not a failure.

**Comment above `buildModule`:** `// buildModule compiles the fixture module dir offline: go build ./... compiles its non-test files, then` / `// go vet ./... type-checks its packages with their test files. A failure names the command and holds the` / `// tool's output.`

**`TestFixtureModulesBuild_Issue251_TC001`.**

- The loop over `fixtureCases`, the subtest `TC-003 h fixture: plainMain`, the control `control: a program uses a name the stub does not declare` and every failure message are unchanged (FR-TEST-HYG-067). Rows f and g still fail at `go build ./...`. Row d now also passes `go vet ./...`, so its two test files are compiled (FR-TEST-HYG-065).
- New control subtest, after the `DownloadMarker` control: `control: a test file uses a name the stub does not declare` (FR-TEST-HYG-066). It is not added to `fixtureCases`, so the dependency check does not run it. Its module is `writeModule(t, map[string]string{"cmd/app/main.go": plainMain, "cmd/app/main_test.go": "package main\n\nimport \"pricefollower.local/internal/amazon/amazontest\"\n\nvar _ = amazontest.Nope\n"})`, with the default stub (which declares only `DownloadMarker`). Checks:
  - `err == nil` → `t.Fatal("the module compiles, want a failure on the undeclared Nope in a test file")`;
  - the error must hold `amazontest.Nope`, else `t.Errorf("build failure %q does not name amazontest.Nope", err)`;
  - the error must start with `go vet ./...`, else `t.Errorf("build failure %q is not reported by go vet ./..., so the non-test files do not build on their own", err)`. As `buildModule` runs vet only after `go build ./...` succeeds, this proves the non-test files compile and the failure comes only from the test file.
- New comment above the test: `// Issue #251 TC-001 and issue #254: every valid fixture module, its test files included, compiles offline;` / `// the invalid ones, a program using a name the stub does not declare and a test file doing so do not,` / `// so the check is not vacuous.`

**Header comment.** After the issue #251 block, add an issue #254 block: `// Issue #254: the fixture build check also compiles the fixtures' test files (go vet ./...);` / `// TestFixtureModulesBuild_Issue251_TC001 and its test-file control prove it. Cases proved outside this file:` / `//   - TC-003: temporary edits of row d's test files, never committed, make that test fail.` / `//   - TC-005 and TC-007: runs on master and with the change give the same subtests and results.` / `//   - TC-008: the diff outside doc/ is limited to this file, and the gates pass.` The issue #251 block is unchanged; no comment says that the check only runs `go build ./...` or that fixture test files are out of scope (FR-TEST-HYG-068).

**Delivery (same as issue #251).** The change is the test code itself, in the file that the TDD red commit locks, so the **tests stage** delivers it with `Edit`, `gofmt`-clean, and writes `<change folder>/no-tests.md`: why the red phase is skipped, the runs and fault injections below with their results, and where each TC is proved. The red state is proved inside the test by the new control, which fails with `buildModule` as on `master` (run 3). The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** Each injection is a temporary edit, reverted before the next step and never committed. When the sandbox refuses an in-place edit, the edited copy is written under `$TMPDIR` and passed with `go test -overlay`, as for issue #322.

1. `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/` passes, then with `-count=3`; every subtest passes, including row d, `TC-003 h fixture: plainMain` and both controls (TC-001, TC-004, TC-005). By hand: each valid fixture (rows a–e, the three corners and `plainMain`) written to a temporary folder with an empty `GOMODCACHE`, then `go build ./...` and `go vet ./...` with `fixtureEnv` succeed (TC-001).
2. One edit of row d at a time: (1) `cmd/app/main_test.go` = `package main` + `import "pricefollower.local/internal/amazon/amazontest"` + `var _ = amazontest.Nope`; (2) the same in `internal/service/service_test.go` (`package service`); (3) `cmd/app/main_test.go` with a syntax error (`import "fmt` unclosed). The same command fails on the row d subtest with `valid fixture does not build: go vet ./...: …`, naming `Nope` for edits 1 and 2. Without the edit it passes (TC-003).
3. Edit 1 of run 2 applied to the `master` version of the file (overlay): the test passes, so the old check did not compile test files (TC-003). `buildModule` reverted to `go build ./...` only, in the new file: the new control fails with its `t.Fatal` message (TC-004).
4. `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1` and `go test -run '^(TestListServerDependencies_TC003_FixtureModules|TestCheckServerDependencies_TC002_JudgesListing|TestServerDoesNotDependOnTestHelper)$' -v -count=1` on `./internal/amazon/amazontest/`, each on `master` (scratch worktree or overlay) and with the change; the subtest names are compared: the build test only gains the new control, and the dependency tests are identical and pass (TC-005, TC-007).
5. `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and `git diff --name-only master -- . ':!doc'` (TC-008).

- **Development stage.** No code. It sets FR-TEST-HYG-065–069 and TS-TEST-HYG-085–090 to implemented. It runs the gates and the repository checks of TC-002, TC-006, TC-007 and TC-008, and restores any file outside `doc/` that differs from `master` and is not `internal/amazon/amazontest/server_dependencies_test.go`.

Refactoring: none. `buildModule` keeps its signature and callers; only its body and comment change.

### Rows f and g of the fixture build check check their failure text (issue #253)

Only `internal/amazon/amazontest/server_dependencies_test.go` changes, and in it only the `default` branch of the `fixtureCases` loop of `TestFixtureModulesBuild_Issue251_TC001`, the comment above that test and the header comment. `buildModule`, `writeModule`, `fixtureEnv`, `fixtureCases` (entries, names, files, `fails` values), the `case "", "dependency":` branch, `TC-003 h fixture: plainMain`, both control subtests, the dependency check and its helpers and tests are not edited (FR-TEST-HYG-072).

**Expected texts.** The branch uses the row's own `tc.fails`: `pricefollower.local/internal/nope` for row f and `main.go` for row g, the values `TestListServerDependencies_TC003_FixtureModules` already requires in the `go list` error. `go build ./...` loads packages with the same loader as `go list`, so it is expected to report the same errors: for row f the unresolvable import, naming `pricefollower.local/internal/nope`; for row g the unterminated string, at a position in `cmd/app/main.go`. `buildModule` keeps the tool's output in its error (`go build ./...: exit status <n>: <output>`), so `err.Error()` holds the text. The tests stage confirms both outputs by hand (run 1). If an output does not hold the row's text, the tests stage stops with `blocked.md`: the texts are fixed by FR-TEST-HYG-070 and are not changed here.

**`default` branch (rows whose `fails` is neither `""` nor `"dependency"`, that is f and g).**

```go
default:
	if err == nil {
		t.Error("invalid fixture builds, want a build failure")
	} else if !strings.Contains(err.Error(), tc.fails) {
		t.Errorf("build failure %q does not hold %q", err, tc.fails)
	}
```

- A fixture that builds still fails with the unchanged message `invalid fixture builds, want a build failure` (FR-TEST-HYG-071).
- A failure without the row's text fails with `build failure "<error>" does not hold "<tc.fails>"`, which names the expected text and holds the whole build failure, in the wording of the `DownloadMarker` control (FR-TEST-HYG-071).
- The check is a plain substring match on the whole error, as in the listing test and the controls. Not required: that the error starts with `go build ./...`. The functional specification asks only for the text, and both rows already stop at `go build ./...`.
- `strings` is already imported; no import changes.

**Comment above the test.** It becomes: `// Issue #251 TC-001, issue #254 and issue #253: every valid fixture module, its test files included, compiles` / `// offline; the invalid ones (rows f and g, each failing with its own fails text), a program using a name the` / `// stub does not declare and a test file doing so do not, so the check is not vacuous.`

**Header comment.** After the issue #254 block, add: `//` / `// Issue #253: rows f and g of TestFixtureModulesBuild_Issue251_TC001 pass only on a build failure holding` / `// their fails text. Cases proved outside this file:` / `//   - TC-001: by hand, go build ./... on the row f and g fixtures prints that text.` / `//   - TC-003: temporary edits, never committed, make the row f and g subtests fail.` / `//   - TC-004 and TC-005: runs on master and with the change give the same subtests and results.` / `//   - TC-006: the diff outside doc/ is limited to this file, and the gates pass.` The other blocks are unchanged.

**Delivery (same as issues #251 and #254).** The change is the locked test code itself, so the **tests stage** delivers it with `Edit`, `gofmt`-clean, and writes `<change folder>/no-tests.md`: why the red phase is skipped (on `master` the rows already pass, and the change only narrows when they pass), the runs and fault injections below with their results, and where each TC is proved. The red state is shown by run 3 (edit 1 passes on `master` and fails with the change). The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** Each injection is a temporary edit, reverted before the next step and never committed. When the sandbox refuses an in-place edit, the edited copy is written under `$TMPDIR` and passed with `go test -overlay`, as for issues #322 and #254.

1. By hand, the row f and g fixtures (the `writeModule` layout: `go.mod`, the default stub and the row's `cmd/app/main.go`) are written to a temporary folder, and `go build ./...` runs there with `fixtureEnv`: it fails, and its output is recorded and holds `pricefollower.local/internal/nope` (f) or `main.go` (g). Then `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/` passes, also with `-count=3` (TC-001).
2. One edit at a time, each with the same command: (1) the `buildModule` table becomes `{{"nosuchcmd", "./..."}, {"vet", "./..."}}`; (2) row f's `fails` becomes `pricefollower.local/internal/other`; (3) row g's `fails` becomes `other.go`; (4) row f's `cmd/app/main.go` becomes `plainMain`. Expected: edit 1 fails the row f and g subtests with `does not hold` and the row's text (the valid subtests and the controls fail too, as expected); edit 2 fails row f, edit 3 row g, each with `does not hold`; edit 4 fails row f with `invalid fixture builds, want a build failure`. Without the edit the command passes (TC-003).
3. Edit 1 on the `master` version of the file (overlay): the row f and g subtests pass, which shows the old check accepted any failure (TC-003).
4. `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1` and `go test -run '^(TestListServerDependencies_TC003_FixtureModules|TestCheckServerDependencies_TC002_JudgesListing|TestServerDoesNotDependOnTestHelper)$' -v -count=1` on `./internal/amazon/amazontest/`, each on `master` (scratch worktree or overlay) and with the change: the subtest names and results are identical, and all pass (TC-004, TC-005).
5. `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...`, `go test -count=1 ./internal/amazon/amazontest/...` and `git diff --name-only master -- . ':!doc'` (TC-006).

- **Development stage.** No code. It sets FR-TEST-HYG-070–073 and TS-TEST-HYG-091–095 to implemented. It runs the gates and the code and repository checks of TC-002, TC-004, TC-005 and TC-006, and restores any file outside `doc/` that differs from `master` and is not `internal/amazon/amazontest/server_dependencies_test.go`.

Refactoring: none.

### Only compile errors fail the fixture build check (issue #340)

Only `internal/amazon/amazontest/server_dependencies_test.go` changes, and in it only `buildModule` and its comment, a new package-level `vetArgs`, the last check of the control `control: a test file uses a name the stub does not declare`, one new subtest of `TestFixtureModulesBuild_Issue251_TC001`, the comment above that test and the header comment. `writeModule`, `fixtureEnv`, `fixtureCases` (entries, names, files, `fails` values), the fixture constants, the stub, the `fixtureCases` loop and its messages, `TC-003 h fixture: plainMain`, the `DownloadMarker` control, the dependency check and its helpers and tests are not edited (FR-TEST-HYG-075, FR-TEST-HYG-077).

**Command (chosen: `go vet -atomic ./...`, the issue's second option).** `go vet` always type-checks each package with its `_test.go` files and fails on a type or syntax error, whatever analyzers are on. Its documented rule is that when any analyzer flag is set to true, only those analyzers run, so `-atomic` turns off every other default analyzer (printf, copylocks, tests and the rest). `atomic` reports only `x = atomic.AddInt64(&x, 1)`-style assignments, which no fixture holds or will need, so in practice only compile errors fail the check (FR-TEST-HYG-074). `go build ./...` stays first and unchanged. Rejected: documenting that the analyzers must pass (the issue's first option; FR-TEST-HYG-074 asks that an analyzer finding not fail the check); `go test -run '^$' -vet=off ./...` (compiles test files with no analyzer, but links and runs one test binary per package, as rejected for issue #254); `go vet -vettool=<none>` (no such built-in no-op tool).

**`vetArgs` and `buildModule(dir string) error`.**

- New, right above `buildModule`: `// vetArgs is the go vet command of buildModule; see buildModule for why it names one analyzer.` / `var vetArgs = []string{"vet", "-atomic", "./..."}`.
- `buildModule`'s table becomes `[][]string{{"build", "./..."}, vetArgs}`; the loop, `fixtureEnv`, `CombinedOutput` and the error `go <args>: <err>: <output>` are unchanged. A vet failure therefore reads `go vet -atomic ./...: exit status 1: <output>`.
- New comment above `buildModule`: `// buildModule compiles the fixture module dir offline: go build ./... compiles its non-test files, then` / `// go vet -atomic ./... type-checks its packages with their test files. Naming one analyzer (atomic, which` / `// no fixture triggers) turns vet's other analyzers off, so an analyzer finding does not fail the check;` / `// only a compile error does. A failure names the command and holds the tool's output.` (FR-TEST-HYG-076).

**Test-file control (FR-TEST-HYG-075).** Name, fixture, `t.Fatal` and the `amazontest.Nope` check are unchanged. The prefix check is derived from `vetArgs`, so it matches the command `buildModule` runs and cannot fail only because that text changed:

```go
vetCmd := "go " + strings.Join(vetArgs, " ")
if !strings.HasPrefix(err.Error(), vetCmd+":") {
	t.Errorf("build failure %q is not reported by %s, so the non-test files do not build on their own", err, vetCmd)
}
```

The message keeps its meaning and now names `go vet -atomic ./...`. A failure from `go build ./...` still fails the check.

**New subtest (FR-TEST-HYG-074), after the test-file control:** `analyzer findings in non-test and test files do not fail the check`. It is not added to `fixtureCases`, so the listing test is unchanged. Its module is `writeModule(t, map[string]string{"cmd/app/main.go": ..., "cmd/app/main_test.go": ...})` with the default stub, whose two files are written as Go string literals in the subtest (gofmt-formatted fixture code):

- `cmd/app/main.go`: `package main` importing `fmt` and `sync`, with `func lock(m sync.Mutex) {}` (copylocks: lock passed by value) and `func main() { fmt.Printf("%d\n", "x"); lock(sync.Mutex{}) }` (printf: `%d` with a string), split over lines as gofmt would.
- `cmd/app/main_test.go`: `package main` importing `fmt` and `testing`, with `func Testfoo(t *testing.T) { fmt.Printf("%d\n", "x") }` (tests: malformed test name; printf).
- Check: `if err := buildModule(dir); err != nil { t.Errorf("valid fixture does not build: %v", err) }`, the message of the valid rows.
- Both files compile (no unused import or variable, `m` is an unused parameter, which Go accepts), and no `sync/atomic` is used, so `atomic` has nothing to report. The findings live in string literals of the test file, so the repository's own `go vet ./...` gate does not see them (FR-TEST-HYG-077).

**Comment above the test.** It becomes: `// Issue #251 TC-001, issue #254, issue #253 and issue #340: every valid fixture module, its test files included,` / `// compiles offline, and an analyzer finding alone does not fail it; the invalid ones (rows f and g, each failing` / `// with its own fails text), a program using a name the stub does not declare and a test file doing so do not,` / `// so the check is not vacuous.`

**Header comment.** In the issue #254 block, `(go vet ./...)` becomes `(go vet -atomic ./..., see issue #340)`. After the issue #253 block, add: `//` / `// Issue #340: only compile errors fail the fixture build check; go vet runs with the atomic analyzer alone, so` / `// its other analyzers' findings do not fail it. The subtest with analyzer findings proves it. Cases proved outside` / `// this file:` / `//   - TC-002 and TC-003: temporary edits of rows a and d, never committed, pass on analyzer findings and fail` / `//     on compile errors.` / `//   - TC-004: with buildModule reduced to go build ./..., the test-file control fails.` / `//   - TC-005: runs on master and with the change give the same subtests and results, plus the new subtest.` / `//   - TC-007: the diff outside doc/ is limited to this file, and the gates pass.` The other blocks are unchanged; no comment says that `go vet ./...` with its default analyzers is only a type check (FR-TEST-HYG-076).

**Delivery (same as issues #251, #254 and #253).** The change is the locked test code itself, so the **tests stage** delivers it with `Edit`, `gofmt`-clean, and writes `<change folder>/no-tests.md`: why the red phase is skipped (the change is the locked test code itself), the runs and fault injections below with their results, and where each TC is proved. The red state is shown by run 4 (the new subtest fails with `master`'s `go vet ./...`). The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** Each injection is a temporary edit, reverted before the next step and never committed. When the sandbox refuses an in-place edit, the edited copy is written under `$TMPDIR` and passed with `go test -overlay`, as for issues #322, #254 and #253. If run 1 or 3 shows that `go vet -atomic ./...` does not fail on a compile error in a test file, or fails on the new subtest's fixture, the tests stage stops with `blocked.md` so that this design is revised.

1. By hand, the new subtest's fixture (the `writeModule` layout) in a temporary folder with `fixtureEnv`: `go build ./...` and `go vet -atomic ./...` succeed; `go vet ./...` fails, and its output, recorded, names the printf, copylocks and tests findings. `go vet -atomic ./...` on each valid fixture of `fixtureCases` and `plainMain` succeeds. Then `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/` passes, also with `-count=3` (TC-001, TC-004).
2. One edit at a time, each with the same command and with `-run '^TestListServerDependencies_TC003_FixtureModules$'`: (1) row a's `cmd/app/main.go` replaced by a literal copy of `helperMain` that also imports `fmt` and calls `fmt.Printf("%d\n", "x")` in `main` (the `helperMain` constant itself is not edited); (2) row d's `cmd/app/main_test.go` given the same import and call in a function; (3) row d's `internal/service/service_test.go` given `import "sync"` and `func f() { var m sync.Mutex; n := m; _ = n }`. Both tests pass. Edits 1 and 2 on the `master` version of the file fail the row's build subtest with `valid fixture does not build` (TC-002).
3. One edit at a time: (1) row d's `cmd/app/main_test.go` = `package main` + `import "pricefollower.local/internal/amazon/amazontest"` + `var _ = amazontest.Nope`; (2) the same in `internal/service/service_test.go` (`package service`); (3) row d's `cmd/app/main_test.go` with a syntax error (`import "fmt` unclosed); (4) row a's `cmd/app/main.go` replaced by a literal copy of `helperMain` with `var _ int = "x"` added. The build test fails on the edited row with `valid fixture does not build`, for edits 1–3 from `go vet -atomic ./...` and naming `amazontest.Nope` for edits 1 and 2, for edit 4 from `go build ./...`. Without the edit it passes (TC-003).
4. In the new file: (1) `vetArgs` = `{"vet", "./..."}` (`master`'s command): the new subtest fails with `valid fixture does not build` (TC-001); (2) `buildModule`'s table reduced to `{{"build", "./..."}}`: the test-file control fails with its `t.Fatal` message (TC-004); (3) `vetArgs` = `{"vet", "-unusedresult", "./..."}`: the test-file control still passes, so its prefix follows the command (TC-004).
5. `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1` and `go test -run '^(TestListServerDependencies_TC003_FixtureModules|TestCheckServerDependencies_TC002_JudgesListing|TestServerDoesNotDependOnTestHelper)$' -v -count=1` on `./internal/amazon/amazontest/`, each on `master` (scratch worktree or overlay) and with the change: the build test only gains the new subtest, every other name and result is identical, and the dependency tests are identical and pass (TC-005).
6. `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...`, `go test -count=1 ./internal/amazon/amazontest/...` and `git diff --name-only master -- . ':!doc'`, twice (TC-007).

- **Development stage.** No code. It sets FR-TEST-HYG-074–077 and TS-TEST-HYG-096–101 to implemented. It runs the gates and the code and repository checks of TC-005, TC-006 and TC-007, and restores any file outside `doc/` that differs from `master` and is not `internal/amazon/amazontest/server_dependencies_test.go`.

Refactoring: none. `buildModule` keeps its signature and callers; its vet arguments move to `vetArgs` only so that the control can name the same command.

### TC-003 checks the real cause of a failed cache clear (issue #239)

Only `internal/amazon/browser_interrupted_test.go` changes, and in it only the `t.Run` function of `TestBrowserInterrupted_TC003_UnconfirmedClearIsBoundedAndLogged`: a comment and two checks are added after the existing `context canceled` check. The subtest table (three entries, names, `navigate`, `clear` and `cause` values), the interruption, the `Page.navigate` and `Browser.close` checks, `assertInterruptedFailure`, the one-line check, the `test.cause` check, the existing `context canceled` check on `lines[0]` and its message, the comment above the test, the header comment, the helpers (`cacheLines`, `assertNoCacheLogLine`, `assertNoInterruptionGivenForCache` and the others), the other tests, `fake_browser_test.go` and `internal/amazon/browser.go` are not edited (FR-TEST-HYG-081).

**Added lines (after the `context canceled` check, before the end of the subtest).**

```go
// Issue #239: a clear that never answers gives its bound as the cause, and no line about the cache gives the
// interruption of the check (context canceled or context deadline exceeded).
if test.clear == "never" && !strings.Contains(lines[0], "no answer within") {
	t.Fatalf("cache log line without \"no answer within\", the cause of a clear that never answers: %s", lines[0])
}
assertNoInterruptionGivenForCache(t, logged)
```

- **Never-answered subtests (FR-TEST-HYG-078).** They are selected by their table entry: `clear` is `"never"` for **no answer** and **no answer, page still requested** only, and that value is what makes the fake never answer. The `cause` field cannot carry the expected text, as it is sent to the fake as `ClearError`. The expected text is `no answer within`, the fixed part of `clearCache`'s `Network.clearBrowserCache: no answer within %s`; the bound (`5s`) is not matched, as FR-TEST-HYG-078 asks only for `no answer within`. The message holds the cache log line.
- **Rejection of both context causes (FR-TEST-HYG-079).** `assertNoInterruptionGivenForCache(t, logged)`, as the issue suggests, runs in every subtest on every output line holding `cache` in any case, so on the cache log line and on any other line about the cache. It fails with `the interruption of the check is given as the cause of a cache clear: <line>` on `context canceled` or `context deadline exceeded`. The helper is not edited, so TC-001 of issue #233, which already calls it, keeps its behaviour. The existing `context canceled` check on `lines[0]` is kept unchanged, so a `context canceled` line still fails with its current message first.
- **Order.** The `test.cause` and `context canceled` checks run first, as on `master`. On a never-answered line giving `context deadline exceeded`, the `no answer within` check fails first; each new check also fails alone (runs 2c and 2d below).
- `strings` is already imported; no import changes. The helper calls `t.Helper()` and `t.Fatalf` on the subtest's `t`.

**Issue #233 test case document (FR-TEST-HYG-080; edited in this stage).** As for issue #181, only the `docs` tool profile may edit `doc/changes/`, so the technical specification stage makes the edit and it lands in its commit. In the TC-003 entry of `doc/changes/2026-11-23-1234-issue-233-clear-the-amazon-browser-http-cache-even/testcases.md`, only the third "Then" result changes. It now reads: "exactly one cache log line; with **no answer** and **no answer, page still requested** it contains `no answer within` (the clear's 5 s bound is given as the cause); with **error answer** it contains `cache clear refused by fake`; in every subtest it contains neither `context canceled` nor `context deadline exceeded` (the interruption of the check is never given as the cause)". The title, Covers, Level, Given, When, the other results, the corner cases, the other entries and the other files of that folder are unchanged. The issue #233 `tests.lock` keeps its old hash of the test file: the factory checks only the running issue's lock, as for issues #324 and #322. `doc/todo/issue-233-r1-04.md` is not changed (functional scope).

**Delivery (same as issues #254, #253 and #340).** On `master` the behaviour already holds, so the tightened test passes and cannot show a red phase. The **tests stage** delivers the test change with `Edit`, `gofmt`-clean, and writes `<change folder>/no-tests.md`: why the red phase is skipped, the runs and fault injections below with their results, and where each TC is proved. The red state is shown by run 2 (without the deadline mapping, the new test fails and the `master` test passes). The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** `R` is `go test -run '^TestBrowserInterrupted_TC003_UnconfirmedClearIsBoundedAndLogged$' -v -count=1 ./internal/amazon/`. Each injection is a temporary edit, reverted before the next step and never committed. When the sandbox refuses an in-place edit, the edited copy (of `browser.go`, or `git show master:internal/amazon/browser_interrupted_test.go` for the `master` test file) is written under `$TMPDIR` and passed with `go test -overlay`, as for issues #322, #254 and #253.

1. `R` passes with exactly the three subtests, named as on `master`, also with `-count=3`; each subtest takes about as long as on `master` (TC-001).
2. In `clearCache`, the `if errors.Is(err, context.DeadlineExceeded) { ... }` block is removed (`errors` stays used by `navigationError`). (a) `R` fails **no answer** and **no answer, page still requested** with the `no answer within` message, the line holding `context deadline exceeded`; **error answer** passes. (b) With the `master` test file, `R` passes. (c) With the `no answer within` check also removed from the test, `R` fails both never-answered subtests with the helper's message. (d) With only the helper call removed from the test, `R` fails them with the `no answer within` message (TC-002).
3. In `clearCache`, `no answer within` becomes `no reply within`: `R` fails both never-answered subtests with the `no answer within` message holding the line; **error answer** passes (TC-003).
4. In `clearCache`, after the deadline mapping, `else if err != nil { err = fmt.Errorf("%w: context deadline exceeded", err) }` (edit 1), then the same with `context canceled` (edit 2): `R` fails **error answer** with the helper's message (edit 1) or the existing `context canceled` message (edit 2), each holding the line with `cache clear refused by fake` (TC-004).
5. `go test -run '^TestBrowserInterrupted_' -v -count=1 ./internal/amazon/` on `master` (overlay of the `master` test file) and with the change: same tests, subtests, names and results; tests skipped for lack of a browser are skipped in both (TC-007).
6. `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and `git diff --name-only master -- . ':!doc'`, twice (TC-008).

If run 1 fails because another line about the cache gives a context cause on a normal run, the tests stage stops with `blocked.md` so that this design is revised: FR-TEST-HYG-081 requires TC-003 to pass on a normal run, and production code is out of scope.

- **Development stage.** No code. It sets FR-TEST-HYG-078–081 and TS-TEST-HYG-102–106 to implemented, runs the gates and the code, document and repository checks of TC-005, TC-006 and TC-008, and restores any file outside `doc/` that differs from `master` and is not `internal/amazon/browser_interrupted_test.go`. If the issue #233 document does not hold the edit, it cannot fix it (no `doc/changes/` access) and stops with `blocked.md`, `Category: owner`.

Refactoring: none.

### The analyzer-findings subtest proves its fixture has findings (issue #344)

Only `internal/amazon/amazontest/server_dependencies_test.go` changes, and in it only the body of the subtest `analyzer findings in non-test and test files do not fail the check` (lines added after its `buildModule` check) and the header comment (a new issue #344 block). The subtest name, its fixture (both files, written with `writeModule` and the default stub), its `buildModule` check and message, `buildModule`, `vetArgs`, `writeModule`, `fixtureEnv`, `fixtureCases`, the fixture constants, the other subtests and their messages, the comment above the test, the other header blocks, the dependency check and its helpers and tests are not edited (FR-TEST-HYG-084). No import changes: `os` and `os/exec` are already imported.

**Added lines (after the existing `buildModule` check, at the end of the subtest).**

```go
// Issue #344: the fixture must have analyzer findings, or this subtest proves nothing, so go vet ./... with
// all its default analyzers (not vetArgs) must fail on the same module.
cmd := exec.Command("go", "vet", "./...")
cmd.Dir = dir
cmd.Env = append(os.Environ(), fixtureEnv...)
if cmd.Run() == nil {
	t.Error("the fixture has no analyzer finding, so the subtest proves nothing")
}
```

- **Full vet run (FR-TEST-HYG-082).** The command is written out, as the issue suggests, and does not reuse `vetArgs`: `vetArgs` names one analyzer, which turns the others off, so the guard would not see the fixture's findings. It runs on `dir`, the folder passed to `buildModule`, so there is no second fixture copy that could drift, and with `fixtureEnv`, so it runs offline like `buildModule`. `vet` only reads the module, so the order of the two checks does not matter. The output is not needed (the guard fails only when vet succeeds, which prints nothing useful), so `Run` is used and the output is discarded.
- **Independent reports (FR-TEST-HYG-083).** The `buildModule` check keeps `t.Errorf` and the guard uses `t.Error`; neither returns nor calls `t.Fatal`, so both run in every run and each failure is reported on its own or together.
- **Analyzer messages: not checked.** The issue makes it optional, and FR-TEST-HYG-082 asks only that a successful full vet run fail the subtest. Exact messages (`wrong type string`, `passes lock by value`, `malformed name`) can change between Go releases, and matching them would make the subtest fail on a toolchain update while the fixture still has findings. Any finding keeps the subtest meaningful: it proves that `buildModule` ignores at least that finding. Recorded alternative, not chosen: also require each of the three texts in `CombinedOutput`.
- **Failure causes other than findings.** A compile error or a go command that cannot start also makes the full vet run fail, so the guard stays silent; the `buildModule` check then fails with `valid fixture does not build`, so the run still fails (TC-003 edit 3).

**Header comment.** After the issue #340 block, add: `//` / `// Issue #344: the subtest with analyzer findings also runs go vet ./... with all its default analyzers on its` / `// fixture and fails if that succeeds, so it cannot pass without findings. Cases proved outside this file:` / `//   - TC-002: with every finding removed from the fixture, never committed, the subtest fails with the guard's message.` / `//   - TC-003: temporary edits of vetArgs and of the fixture, never committed, show that the guard and the build` / `//     check are reported independently.` / `//   - TC-005: runs on master and with the change give the same subtests and results.` / `//   - TC-006: the diff outside doc/ is limited to this file, and the gates pass.` Lines stay within the file's current width (120 columns); the developer may rewrap them.

**Delivery (same as issues #254, #253 and #340).** The change is the locked test code itself, so the **tests stage** delivers it with `Edit`, `gofmt`-clean, and writes `<change folder>/no-tests.md`: why the red phase is skipped (on a normal run the fixture has findings, so the new guard passes; its red state needs a finding-free fixture), the runs and fault injections below with their results, and where each TC is proved. The red state is shown by run 2 (the finding-free fixture fails the new subtest and passes the `master` one). The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** `R` is `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/`. Each injection is a temporary edit of the test file, reverted before the next step and never committed. When the sandbox refuses an in-place edit, the edited copy (or `git show master:internal/amazon/amazontest/server_dependencies_test.go` for the `master` file) is written under `$TMPDIR` and passed with `go test -overlay`, as for issues #322, #254, #253 and #340.

1. By hand, the subtest's two files in a temporary module with the `writeModule` layout (`go.mod` `module pricefollower.local` / `go 1.25` and the default stub), with `fixtureEnv`: `go vet ./...` fails and its output, recorded, names the printf, copylocks and tests findings; `go build ./...` and `go vet -atomic ./...` succeed. Then `R` passes with the subtest named as on `master`, also with `-count=3` (TC-001).
2. In the subtest's fixture: `fmt.Printf("%d\n", 1)` in both files, `func lock(m *sync.Mutex) {}` called with `lock(&sync.Mutex{})`, and `Testfoo` renamed `TestFoo`. `R` fails the subtest with the guard message and without `valid fixture does not build`; the other subtests pass. With the same edit on the `master` file, `R` passes. With only `Testfoo` renamed, `R` passes (TC-002).
3. One edit at a time: (1) `vetArgs` = `{"vet", "./..."}`: the subtest fails with `valid fixture does not build` and the vet output, without the guard message; the test-file control still passes; (2) `vetArgs` = `{"nosuchcmd"}` together with the finding-free fixture of run 2: the subtest reports both `valid fixture does not build` and the guard message; the other valid-fixture subtests fail too, as expected; (3) `var _ int = "x"` added at package level to the fixture's `cmd/app/main.go`, findings kept: the subtest fails with `valid fixture does not build` and without the guard message. Without the edits, `R` passes (TC-003).
4. `R` and `go test -run '^(TestListServerDependencies_TC003_FixtureModules|TestCheckServerDependencies_TC002_JudgesListing|TestServerDoesNotDependOnTestHelper)$' -v -count=1 ./internal/amazon/amazontest/`, each on `master` and with the change: identical test and subtest names and results, all passing (TC-005).
5. `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...`, `go test -count=1 ./internal/amazon/amazontest/...` and `git diff --name-only master -- . ':!doc'`, twice (TC-006).

If run 1 shows that the full vet run succeeds on the unchanged fixture, the tests stage stops with `blocked.md` so that this design is revised: FR-TEST-HYG-084 keeps the fixture unchanged.

- **Development stage.** No code. It sets FR-TEST-HYG-082–085 and TS-TEST-HYG-107–112 to implemented, runs the gates and the code and repository checks of TC-004, TC-005 and TC-006, and restores any file outside `doc/` that differs from `master` and is not `internal/amazon/amazontest/server_dependencies_test.go`.

Refactoring: none. `buildModule` and `vetArgs` are not shared with the guard; the three lines of command setup are not factored out, as `buildModule` loops over its own commands and returns the first error with output, which the guard does not need.

### The analyzer-findings guard requires a finding in each file (issue #349)

Only `internal/amazon/amazontest/server_dependencies_test.go` changes, and in it only the findings guard of the subtest `analyzer findings in non-test and test files do not fail the check` (its comment and the lines after `cmd.Env`) and the header comment (a new issue #349 block after the issue #344 block). The subtest name, its fixture, its `buildModule` check and message, the guard's command (`exec.Command("go", "vet", "./...")`, `cmd.Dir = dir`, `cmd.Env = append(os.Environ(), fixtureEnv...)`), `buildModule`, `vetArgs`, `writeModule`, `fixtureEnv`, `fixtureCases`, `plainMain`, `helperMain`, the stub, the other subtests and their messages, the comment above the test, the other header blocks (the issue #344 block included) and the dependency check and its tests are not edited (FR-TEST-HYG-089). No import changes: `os`, `os/exec` and `strings` are already imported.

**Findings guard after the change (replaces lines 422–430 on `master`).**

```go
// Issue #344 and issue #349: this subtest proves that the fixture build check ignores analyzer findings in
// non-test files and in test files only if the fixture has a finding in cmd/app/main.go and one in
// cmd/app/main_test.go. So go vet ./... with all its default analyzers (not vetArgs, which names one analyzer and
// so turns the others off) must fail on the same module and print a finding position in each of the two files.
// Positions are matched, not the analyzers' messages, which can change between Go releases.
cmd := exec.Command("go", "vet", "./...")
cmd.Dir = dir
cmd.Env = append(os.Environ(), fixtureEnv...)
out, err := cmd.CombinedOutput()
if err == nil {
	t.Error("the fixture has no analyzer finding, so the subtest proves nothing")
}
for _, file := range []string{"cmd/app/main.go", "cmd/app/main_test.go"} {
	if !strings.Contains(string(out), file+":") {
		t.Errorf("the fixture has no analyzer finding in %s, so the subtest proves nothing", file)
	}
}
```

- **File positions (FR-TEST-HYG-086).** vet prints each finding as `<file>:<line>:<column>: <message>`, with the file relative to `cmd.Dir`; the issue #344 record (`no-tests.md`) shows `cmd/app/main.go:8:13`, `cmd/app/main.go:11:14`, `cmd/app/main_test.go:9:14` and `cmd/app/main_test.go:8:6`. The check is `file + ":"`: neither `cmd/app/main.go:` nor `cmd/app/main_test.go:` is a substring of the other, so a finding in one file does not satisfy the other's check. `CombinedOutput` is used because vet writes findings to standard error. The loop runs both iterations whatever the first one's result, and each reports its own message with `t.Errorf`, so each file is checked independently. No analyzer message is matched (as in issue #344).
- **Module message kept (FR-TEST-HYG-087).** `err == nil` replaces `cmd.Run() == nil` with the same meaning: the full vet run succeeded, so the fixture has no finding at all. In that case vet prints nothing, so both file messages are reported too; TC-004 accepts that. `string(nil)` is `""`, so the empty output needs no special case. The file checks are not skipped when vet succeeds: the module message alone would say the same, but skipping would add a branch for no gain.
- **Independent reports (FR-TEST-HYG-087).** The `buildModule` check keeps `t.Errorf`; the module check uses `t.Error` and the file checks `t.Errorf`; there is no `t.Fatal` or `return` in or between them, so the build check, the module check and the two file checks are all reported in every run.
- **Other failure causes.** A compile error makes vet fail with the error's position, so the file checks may pass on that position; the `buildModule` check then fails with `valid fixture does not build`, so the run still fails (as in issue #344). A go command that cannot start returns an error with no output, so both file messages are reported; `buildModule` fails in the same way, so this only adds messages to a failing run.
- **Comment (FR-TEST-HYG-088).** The comment above the guard is replaced by the one above, which names both files and what the subtest proves. The issue #344 header block is kept unchanged: it records what issue #344 added (vet must fail), and the issue #349 block right after it records the per-file requirement, so no comment says that only the module as a whole is guarded.

**Header comment.** After the issue #344 block, add: `//` / `// Issue #349: the subtest with analyzer findings also requires the output of go vet ./... to hold a finding position` / `// in cmd/app/main.go and one in cmd/app/main_test.go, so it keeps proving that findings in each are ignored. Cases` / `// proved outside this file:` / `//   - TC-002 and TC-003: with the findings removed from one file, never committed, the subtest fails naming it.` / `//   - TC-004: with every finding removed, never committed, the subtest still fails with the module message.` / `//   - TC-005: temporary edits of vetArgs and of the fixture, never committed, show that the guard's checks and the` / `//     build check are reported independently.` / `//   - TC-007: runs on master and with the change give the same subtests and results.` / `//   - TC-008: the diff outside doc/ is limited to this file, and the gates pass.` Lines stay within 120 columns; the developer may rewrap them.

**Delivery (same as issues #254, #253, #340 and #344).** The change is the locked test code itself, so the **tests stage** delivers it with `Edit`, `gofmt`-clean, and writes `<change folder>/no-tests.md`: why the red phase is skipped (on a normal run the fixture has findings in both files, so the new checks pass; their red state needs a fixture edit), the runs and fault injections below with their results, and where each TC is proved. The red state is shown by run 2 (findings removed from `cmd/app/main_test.go` fail the new subtest and pass the `master` one). The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** `R` is `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/`. "Findings removed from main.go" and "from main_test.go" are as defined in `testcases.md`. Each injection is a temporary edit of the test file, reverted before the next step and never committed. When the sandbox refuses an in-place edit, the edited copy (or `git show master:internal/amazon/amazontest/server_dependencies_test.go` for the `master` file) is written under `$TMPDIR` and passed with `go test -overlay`, as for issues #322, #254, #253, #340 and #344.

1. By hand (or with a temporary `t.Log(string(out))`, reverted), the full vet run on the subtest's fixture, with `fixtureEnv`: non-zero exit, and the recorded output holds `cmd/app/main.go:` and `cmd/app/main_test.go:`. Then `R` passes with the subtest named as on `master` and none of the four guard and build messages, also with `-count=3` (TC-001).
2. Findings removed from main_test.go: `R` fails the subtest with the main_test.go message only (no main.go, module or build message); the other subtests pass. With the same edit on the `master` file, `R` passes. With only `Testfoo` renamed `TestFoo`, and with only its `Printf` call changed, `R` passes (TC-002).
3. Findings removed from main.go: `R` fails the subtest with the main.go message only. With only the `Printf` call, and with only the copylocks finding, removed from `cmd/app/main.go`, `R` passes (TC-003).
4. Findings removed from both files: `R` fails the subtest with the module message (and the two file messages), without the build message and without a panic (TC-004).
5. One edit at a time: (1) `vetArgs` = `{"vet", "./..."}` with findings removed from main_test.go: the subtest reports the build message and the main_test.go message, not the main.go message; (2) `vetArgs` = `{"nosuchcmd"}` with findings removed from both files: the subtest reports the build message and the module message; the other valid-fixture subtests fail too, as expected; (3) `vetArgs` = `{"vet", "./..."}`, fixture unchanged: the subtest fails with the build message only. In each, the test-file control behaves as recorded for issue #344. Without the edits, `R` passes (TC-005).
6. `go test -run '^(TestFixtureModulesBuild_Issue251_TC001|TestListServerDependencies_TC003_FixtureModules|TestCheckServerDependencies_TC002_JudgesListing|TestServerDoesNotDependOnTestHelper)$' -v -count=1 ./internal/amazon/amazontest/` on `master` and with the change: identical test and subtest names and results, all passing (TC-007).
7. `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...`, `go test -count=1 ./internal/amazon/amazontest/...` and `git diff --name-only master -- . ':!doc'`, twice (TC-008).

If run 1 shows that the full vet run's output lacks either position on the unchanged fixture, the tests stage stops with `blocked.md` so that this design is revised: FR-TEST-HYG-089 keeps the fixture unchanged.

- **Development stage.** No code. It sets FR-TEST-HYG-086–089 and TS-TEST-HYG-113–117 to implemented, runs the gates and the code and repository checks of TC-006, TC-007 and TC-008, and restores any file outside `doc/` that differs from `master` and is not `internal/amazon/amazontest/server_dependencies_test.go`.

Refactoring: none. The guard's command setup is kept; only its result handling changes.

### Each Amazon item gets its own redraw time (issue #260)

Only `TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce` in `internal/service/amazon_human_checks_issue185_test.go` changes, and in it only one block is added. `scheduleUnscheduledAmazonItems` (`internal/service/service.go`) and `nextAmazonCheckAt` (`internal/service/amazon_schedule.go`) are not edited: the redraw already calls `s.nextCheckAt` once per item (FR-AMZ-HCHK-005). The draw is uniform to the second over the evening window (14 401 values), so with seed 1851 three equal times are not expected; run 1 below confirms it.

**New check (FR-TEST-HYG-090).** Right after the first start's loop that fills `drawn` (after its closing brace, before the `redrawMark` check), add:

```go
	// FR-AMZ-HCHK-005: each item's time is drawn on its own, so no two items share one (issue #260).
	for i, id := range ids {
		for _, other := range ids[i+1:] {
			if drawn[id].Equal(drawn[other]) {
				t.Fatalf("items %s and %s both have the next check %s after the start, want a time drawn for each item", id, other, drawn[id].In(paris))
			}
		}
	}
```

- It compares every pair of `ids` (3 pairs, including the first and third item), in the order of `ids`, not of the store, so the message names the items in that order.
- `time.Time.Equal` compares instants, so a location or monotonic difference of the values read back by `nextCheck` cannot hide an equal time.
- `t.Fatalf` and the `.In(paris)` time format follow the test's other messages. No helper is added: the check is used once.

**Unchanged (FR-TEST-HYG-091).** The test name, the comment above it, the seeds 1851 and 1852, the clocks, the item IDs, `former`, the checks before the start, the window and mark checks after the first start, the second start's checks and every message. No other test or helper of the file (`redrawMark`, `setRedrawMark`, and `insertAt`, `nextCheck`, `startHuman`, `windowOf`, `localDate` from the other test files) changes, nor the file's header comment.

**Issue #185 test cases (FR-TEST-HYG-092).** Made by this stage, as for issue #181: only the `docs` profile may edit `doc/changes/`. In TC-002 of `doc/changes/2026-11-23-1706-issue-185-tests-missing-from-168-amazon-human-like/testcases.md`, the first start's Then list gains, after the window line, `the three items' stored next scheduled checks are pairwise different, as each item's time is drawn on its own;`, and the corner cases gain, before the mutation check, `one time drawn and given to every item (for example the draw moved out of the per-item loop) makes this test fail, although that time is within the window;`. The title, Covers, Level, Given, When, the other Then lines, the second start, the other corner cases and the rest of the document are unchanged. The issue #185 `tests.lock`, `no-tests.md` and `doc/todo/issue-185-r1-01.md` are unchanged (the factory checks only the running issue's lock).

**Delivery (same as issues #251, #254, #253 and #340).** The change is the locked test code itself, so the **tests stage** adds the block with `Edit`, `gofmt`-clean, and writes `<change folder>/no-tests.md`: why the red phase is skipped (the change is an added check on shipped behaviour, so it passes on `master`'s production code; its red state is shown by fault injection 2), the runs below with their results and failure messages, and where each TC is proved. The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** Each injection is a temporary edit of `scheduleUnscheduledAmazonItems`, reverted before the next step and never committed; when the sandbox refuses an in-place edit, the edited copy is written under `$TMPDIR` and passed with `go test -overlay`, as for issues #322, #254, #253 and #340. Every injected time is still drawn by `s.nextCheckAt("amazon", s.now())`, so it is in the evening window and the mark is still cleared. The run is `go test -count=1 -run '^TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce$' -v ./internal/service/`.

1. With the change, no injection: the run passes, also with `-count=3`, and `go test ./internal/service/...` passes (TC-001). If it fails on the new check, the seed gives two equal times and no fix is allowed (the seeds must stay, FR-TEST-HYG-091): the tests stage stops with `blocked.md` so that the specification is revised.
2. `shared := s.nextCheckAt("amazon", s.now())` before the loop and `SetNextCheck(ctx, id, shared)` in it: the run fails with `items B000001853 and B000001854 both have the next check <time> after the start, want a time drawn for each item`, `<time>` in the evening of 2030-03-05. With the same injection, the `master` version of the test file passes (TC-002).
3. One at a time: `shared` drawn before the loop, stored for the two named items only (selected by ID, so the store's order does not matter), and a time drawn in the loop for the third: (1) `B000001853` and `B000001854`; (2) `B000001854` and `B000001855`; (3) `B000001853` and `B000001855`. Each run fails with the message naming that pair (TC-003).
4. The `-v` output of the run on `master` (scratch worktree or overlay of the `master` test file) and with the change: the same test name and `PASS`; `git diff master -- internal/service/amazon_human_checks_issue185_test.go` has one hunk that only adds the block above (TC-004).
5. `gofmt -l internal/service`, `go vet ./...`, `go test ./...`, `git diff --name-only master -- . ':!doc'` and `git diff --name-only master -- doc/changes/2026-11-23-1706-issue-185-tests-missing-from-168-amazon-human-like/`, twice (TC-006).

- **Development stage.** No code. It sets FR-TEST-HYG-090–093 and TS-TEST-HYG-118–122 to implemented. It runs the gates and the checks of TC-004 to TC-006, and restores any file outside `doc/` that differs from `master` and is not `internal/service/amazon_human_checks_issue185_test.go`. If the issue #185 document does not hold the TC-002 edit, it cannot fix it (no `doc/changes/` access) and stops with `blocked.md`, `Category: owner`.

Refactoring: none.

### One real-browser test for the cache log through the application (issue #358)

Two test files of `internal/httpapi` change: TC-005 of issue #233 in `amazon_browser_interrupted_test.go` gains a check in its per-owner loop, and `amazon_browser_cache_log_test.go` is deleted. No helper is added or changed: `captureCheckOutput` and `linesWith` (`amazon_browser_navigation_test.go`) and `waitStoredCheck` (`amazon_browser_retrieval_test.go`) are reused as they are. Production code (`clearCache` in `internal/amazon/browser.go`, its log line of FR-AMZ-HCHK-016) is not edited.

**Added lines in the per-owner loop of `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile` (FR-TEST-HYG-103).** The loop becomes, with the added lines marked `+`:

```go
	for index, owner := range owners {
		id, at := ids[index], humanTime(14, 10+index)
		api.clock.Set(at)
+		refreshOutput := captureCheckOutput(t)
		principal := service.WithPrincipal(context.Background(), service.Principal{UserID: owner, Role: "user"})
		if _, err := api.items.RefreshItem(principal, id); err != nil {
			t.Fatalf("Refresh price of %s by owner %d: %v", id, owner, err)
		}
		waitStoredCheck(t, api.database, id, at)
+		// FR-AMZ-HCHK-016: the Refresh price logs no cache clear failure, in either wording (issues #240 and #358).
+		if lines := linesWith(refreshOutput(), "Amazon browser cache could not be cleared"); len(lines) > 0 {
+			t.Fatalf("cache clear failure logged during Refresh price of %s:\n%s", id, strings.Join(lines, "\n"))
+		}
		items, err := api.database.List(context.Background(), owner)
		…
```

- One capture per owner, started before `RefreshItem` and read after `waitStoredCheck`. The collector logs any cache line inside `Collect`, before the service records the attempt that `waitStoredCheck` waits for, so the read cannot miss a line of that check. Calling `refreshOutput()` ends the capture (restores `os.Stdout` and the `log` output, `sync.Once`); a failure before it is covered by the capture's `t.Cleanup`. The server-stop capture `output` has already ended at its own check, before the loop, so the captures do not nest and a server-stop line cannot be blamed on an item.
- The single prefix matches both wordings, as the old one (`… cleared after a page not opened: …`) begins with it; `assertNoCacheLine`'s separate `after a page not opened` search is not carried over.
- The message holds the item ID and the found lines, one per line, after the same `cache clear failure logged` words as the server-stop check. It comes before the item-list checks, as in the deleted copy, so a failure for the second item comes after all the first item's checks.
- `strings` is already imported; no import changes. The comment above the test and its `// TC-005` line are unchanged.

**Unchanged (FR-TEST-HYG-104).** The test name, comments, IDs `B0INTR0051`/`B0INTR0052`, owners, fixture answers, server-stop sequence and its three guards, the server-stop cache check and its message, `assertNoDownloadWritten`, `assertNoNewDataFile`, the item-list and price checks and every message. On a normal run the check passes: the normal page opens, so `load` does not call `clearCache` and no cache line is logged.

**Deletion (FR-TEST-HYG-105).** `internal/httpapi/amazon_browser_cache_log_test.go` is listed in the tests stage's `delete.txt`. It declares only `assertNoCacheLine` and `TestAmazonBrowserCacheLog_TC004_SuccessfulClearThroughTheApplicationLogsNothing`; no other Go file uses either, and every helper it calls is declared in another file and used by TC-005, so the package compiles without it. `doc/todo/issue-240-r1-01.md` and the issue #240 `tests.lock`, `review-r1.json` and `triage-r1.json` keep naming it as history (excluded by the functional specification).

**Issue #240 documents (FR-TEST-HYG-106).** Made by this stage, as for issues #239 and #260: only the `docs` profile may edit `doc/changes/`.

- TC-004 entry of `doc/changes/2026-11-25-1255-issue-240-neutral-wording-for-the-cache-clear-fail/testcases.md`: the heading's `(existing test, unchanged)` becomes `(existing tests; TC-005 of issue #233 extended by issue #358)`; the Level names TC-005 as extended by issue #358 with a check after each owner's Refresh price, and the `internal/amazon` test as unchanged; the Then line says that no line contains the prefix after the server stop or during either owner's Refresh price. Covers, Given, When, the corner cases and the rest of the document are unchanged.
- TC-004 row of the issue #240 section of `doc/specifications/amazon-human-checks/technical.md`: the file column marks `internal/httpapi/amazon_browser_interrupted_test.go` as extended by issue #358 and only `internal/amazon/browser_interrupted_test.go` as unchanged; the notes say that TC-005 checks for no cache line after the server stop and during each owner's Refresh price. No other row or section changes.

**Delivery (same as issues #239 and #260).** The change is the locked test code itself and it checks shipped behaviour, so the **tests stage** makes the edit with `Edit`, `gofmt`-clean, lists the duplicate in `delete.txt`, and writes `<change folder>/no-tests.md`: why the red phase is skipped (it passes on `master`'s production code; its red state is shown by injections 2 and 3), the runs below with their results and failure messages, and where each TC is proved. The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** Each injection is a temporary edit, reverted before the next step and never committed; when the sandbox refuses an in-place edit, the edited copy is written under `$TMPDIR` and passed with `go test -overlay`, as for issues #322, #254 and #260. An overlay entry mapping `internal/httpapi/amazon_browser_cache_log_test.go` to `""` stands for the deletion before the pipeline applies it. "The run" is `CI=1 go test -count=1 -run '^TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile$' -v ./internal/httpapi/`; "`master`'s test" is the run with `git show master:internal/httpapi/amazon_browser_interrupted_test.go` as overlay.

1. With the change, no injection: the run passes, also with `-count=2`, and `go test ./internal/httpapi/` passes (TC-001).
2. In `BrowserCollector.Collect` (`internal/amazon/browser.go`), right after `tab.load` returns without error: `log.Printf("Amazon browser cache could not be cleared: test")`. The run fails with `cache clear failure logged during Refresh price of B0INTR0051:` and the line, not at the server-stop check (that check is answered with a download, so `load` fails). Repeated with `fmt.Println` instead of `log.Printf`: same failure. With the `log.Printf` injection, `master`'s test passes (TC-002).
3. The same place, only `if strings.Contains(item.URL, "B0INTR0052")`, logging `Amazon browser cache could not be cleared after a page not opened: test`: the run fails with the message naming `B0INTR0052` and holding the old-wording line, and not naming `B0INTR0051` (TC-003).
4. One at a time, each also run with `master`'s test (test-file edits applied to both versions): (1) a `defer` at the top of `Collect` logging `Amazon browser cache could not be cleared: test` when `ctx.Err() != nil`: both fail with `cache clear failure logged:` and the line, and with no Refresh-check message; (2) in `newNavigationFixture` (`amazon_browser_navigation_test.go`) the `answerAttachmentHeld` branch returning right after its flush signal, and TC-005 waiting up to 20 s on `stopped` before its "ended before the server stop" guard: both fail with `the check ended before the server stop: the interruption point was missed` (TC-004).
5. `git diff master -- internal/httpapi/amazon_browser_interrupted_test.go`: one hunk in the per-owner loop, adding only the five lines above (TC-005).
6. With the deletion overlay: `go vet ./internal/httpapi/` passes; `go test -list . ./internal/httpapi/` equals the `master` list minus `TestAmazonBrowserCacheLog_TC004_SuccessfulClearThroughTheApplicationLogsNothing` (TC-006).
7. `gofmt -l internal/httpapi`, `go vet ./...`, `go test ./...` and `git diff --name-status master -- . ':!doc'`, twice (TC-008).

- **Development stage.** No code. It sets FR-TEST-HYG-103–107 and TS-TEST-HYG-135–140 to implemented. On the committed tree it runs the gates and the checks of TC-005 to TC-008 (`ls`, `grep` and `go test -list` of TC-006 without overlay), and restores any file outside `doc/` that differs from `master` and is neither of the two test files. If the duplicate file is still present, it cannot delete it (the green gate refuses a test file changed after the red commit), and if the issue #240 test case document lacks the TC-004 edit, it cannot fix it (no `doc/changes/` access); in either case it stops with `blocked.md`, `Category: owner`.

Refactoring: none. The deletion is the issue's point, not a refactoring; `captureCheckOutput`, `linesWith` and the other helpers stay as they are.

### TC-005 wrap check counts the item at its base width (issue #199)

Only `tests/responsive-layout.spec.ts` changes: the wrap check of the test `TC-005 card values side by side` and the `cells` function of `layoutProbe` with its type. `src/index.css` and every component are not edited (edited only temporarily for fault injections).

**Why the `master` check is blind.** A card is a wrapping flex row (`.responsive-table tr`, TS-RESPONSIVE-017). The browser breaks lines with the values' base sizes: the item cell `flex: 1 1 14rem`, every other cell `flex: 0 1 auto` (its max-content width, not grown); the auto start margin of the actions cell counts as 0. After the break, the item cell grows to the end of its line. `master` computes `free = card.right - previous.right`; when the previous value is the grown item, `free` is about 0, so any too-large item basis passes.

**New check (FR-TEST-HYG-094).** The free space of the previous line is computed as the browser does it:

- *Previous line.* For a value at `position` that does not follow on the line of `cells[position - 1]`, the line before is `cells[start..position - 1]`, where `start` walks back from `position - 1` while `followsOnLine(cells[start - 1], cells[start])`. All its values are counted, not only the last one; a third line has no item value.
- *Item value.* The item value is the first cell of a tracked-item card (DOM order of `TrackedItemsPage.tsx`; it has no `data-label`). It counts as `ITEM_BASIS_REM * rem`, with `const ITEM_BASIS_REM = 14;` declared next to `cardLabels` with a comment naming the `td:has(> .item-cell)` basis of FR-RESPONSIVE-024, and `rem` the root font size read in the page after each `setViewportSize`: `parseFloat(getComputedStyle(document.documentElement).fontSize)`. The basis is a constant, never read from the page's styles, so a changed basis in `src/index.css` is detected.
- *Other values.* Their rendered `width` (border box). On a line the browser filled, they are not shrunk, so this is their base size.
- *Content width.* `card.right - card.left`. `layoutProbe.cells` gains `left: box.left + parseFloat(style.paddingLeft) + parseFloat(style.borderLeftWidth)`, mirroring its `right`; the `Window.layoutProbe.cells` type becomes `{ left: number; right: number; cells: CardCell[] }`. The other callers (TC-006, the users list tests) read `cells` and `right` only, so they are unaffected.
- `free = card.right - card.left - used`, `used` the sum above; the check stays `expect(free, message).toBeLessThan(current.width + 1)`.

The wrap check becomes:

```ts
        // A value starts a new line only when it does not fit in the space left on the previous one.
        // Lines break on base sizes: the item value (first cell) counts at its 14rem basis, not at the
        // width it grows to; the other values do not grow (issue #199).
        card.cells.forEach((current, position) => {
          if (position === 0 || followsOnLine(card.cells[position - 1], current)) return;
          let start = position - 1;
          while (start > 0 && followsOnLine(card.cells[start - 1], card.cells[start])) start--;
          const used = card.cells.slice(start, position)
            .reduce((sum, cell, offset) => sum + (start + offset === 0 ? ITEM_BASIS_REM * rem : cell.width), 0);
          const free = card.right - card.left - used;
          expect(free, `${what}: free space ${free.toFixed(1)} before value ${current.label || position} (width ${current.width.toFixed(1)})`).toBeLessThan(current.width + 1);
        });
```

**Message (FR-TEST-HYG-095).** `what` stays `${width}px ${tab} card ${index}`, which names the width, the tab and the card (for example `775px Amazon card 0`). It is followed by the free space, the value's label (`Prices`) and its width; Playwright also prints the received free space. With Prices pushed below an item whose basis is too large, the line before holds the item alone, so `free` is the content width minus 14rem, clearly larger than the Prices width.

**Unchanged (FR-TEST-HYG-096).** The test name, widths 775 and 1055 px, tabs, `sampleItems`, the row count check, the same-line and label checks and their messages, and `sameLine`, `followsOnLine`, `cardLabels`, `mockApi` and the rest of `layoutProbe`. No other test changes. If run 1 below fails on the unchanged application, the card layout disagrees with FR-RESPONSIVE-024 and no CSS change is allowed (FR-TEST-HYG-098): the tests stage stops with `blocked.md` so that the specification is revised.

**Issue #165 test cases (FR-TEST-HYG-097).** Made by this stage, as for issues #260 and #239: in TC-005 of `doc/changes/2026-11-19-2053-issue-165-use-all-space-to-show-item-list-when-the/testcases.md`, the Then line about a value starting a later line now reads "the free space of the previous line, counting the item value at its base width of 14rem, not at its grown width, is smaller than that value's width, …". Nothing else in that folder changes, including `tests.lock` (the factory checks only the running issue's lock), and `doc/todo/issue-165-r1-01.md` is unchanged.

**Delivery (same as issues #295 and #290).** The change is the test code itself and TC-005 passes on the application before and after it, so a red phase cannot fail. The **tests stage** makes the edit above and writes `<change folder>/no-tests.md`: why the red phase is skipped, the runs below with their results and failure messages, and where each TC is proved. The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** `R` is `npx playwright test tests/responsive-layout.spec.ts -g "TC-005 card values side by side"`. Each injection is a temporary edit of `src/index.css`, reverted with `Edit` before the next step and never committed. "The `master` test" is the test file with its wrap check and `cells` temporarily restored to `git show master:tests/responsive-layout.spec.ts`, then the change re-applied.

1. No injection: `R` passes, also with `--repeat-each=3`, at 775 and 1055 px on both tabs for every card, both Amazon cards included (TC-001).
2. `.responsive-table tbody tr td:has(> .item-cell)` gets `flex: 1 1 40rem`: `R` fails at 775 px with the wrap-check message naming `775px Amazon card 0`, `Prices`, its width and a free space larger than that width; the same-line and label checks of that card passed before it. With the same injection, the `master` wrap check passes every 775 px card (TC-002); the `master` test still fails later at 1055 px, where Prices and Status no longer fit on one line beside a 40rem item, so run 3 gives the full `master` pass.
3. The same rule gets `flex: 1 1 100%`: `R` fails at 775 px on the value starting the line after the item, with the same message form; the `master` test passes with the same injection (TC-004).
4. `npx playwright test tests/responsive-layout.spec.ts --reporter=list` with the `master` test file and with the change: the same test names and results; `git diff master -- tests/responsive-layout.spec.ts` touches only `ITEM_BASIS_REM`, the `cells` type and its `left` field, the `rem` read and the wrap check of TC-005 (TC-003, TC-005).
5. Gates and repository checks of TC-007, twice: no fault injection remains in `src/index.css`.

- **Development stage.** No code. It sets FR-TEST-HYG-094–098 and TS-TEST-HYG-123–128 to implemented. It runs the gates and the checks of TC-003 and TC-005 to TC-007, and restores any file outside `doc/` that differs from `master` and is not `tests/responsive-layout.spec.ts` (for example a fault injection left in `src/index.css`). If the issue #165 document does not hold the TC-005 edit, it cannot fix it (no `doc/changes/` access) and stops with `blocked.md`, `Category: owner`.

Refactoring: none. The probe gains one field used by the new check; no shared helper changes behaviour.

### The duplicate TC-003 test of issue #240 is removed (issue #359)

Only `internal/amazon/browser_cache_log_test.go` changes, and only by deleting lines. On `master` the file ends with `TestBrowserCacheLog_TC003_SuccessfulClearLogsNothing`: the blank line 168 after the closing `}` of `TestBrowserCacheLog_TC002_PageStillRequested`, the comment lines 169–170 (`// A successful clear logs no cache line, whatever started it.` and `// TC-003`) and the function, lines 171–212. Its subtests are "live check, page not opened, repeated" and, for each interruption kind, "interrupted after the offer, …" and "page still requested, …", all with `Clear: "answer"`.

**Change.**

- Delete lines 168–212. The file then ends with the closing `}` of the TC-002 test and one newline. The test is deleted, not skipped, renamed, commented out or moved (FR-TEST-HYG-099).
- Imports and declarations: nothing else goes. After the deletion `net/http` is still used (`http.StatusOK` in TC-001), `regexp` by `logMessage`, `strings` by the helpers, `testing` everywhere and `time` by TC-001 and TC-002 (`time.Now`, `time.Second`). The constants, `logMessage`, `assertNeutralCacheLine`, `assertNoOldCacheWording` and `assertBrowserClosed` are all still called by TC-001 or TC-002. The deleted test declares no package-level name (FR-TEST-HYG-100).
- Unchanged: the header comment, the helpers, TC-001 and TC-002 with their subtests, assertions and messages; `internal/amazon/browser_interrupted_test.go`, including the helpers the deleted test used (`assertNoCacheLogLine`, `assertClearConfirmedBeforeClose`, `assertNavigatedBeforeInterruption`, `assertInterruptedFailure`, `interruptedContext`, `collectInterrupted`, `cacheLines`, `cacheLogLine`), which other tests still use; `internal/amazon/browser.go` (FR-TEST-HYG-100, FR-TEST-HYG-102).

**Coverage kept (FR-TEST-HYG-101).** Each deleted subtest has an unchanged counterpart that calls `assertNoCacheLogLine` (it fails on any line containing `cacheLogLine`, "Amazon browser cache could not be cleared", so it also rejects the old wording "… after a page not opened"):

| Deleted subtest | Counterpart |
| --- | --- |
| live check, page not opened, repeated | `TestBrowserInterrupted_TC006_UninterruptedCheckIsUnchanged/download` (same script `Navigate: "error"`, `Response: 200`, `Clear: "answer"`); repetition with one collector by `TestBrowserInterrupted_TC001_ClearConfirmedBeforeClose` (two checks per subtest) |
| interrupted after the offer, each kind | `TestBrowserInterrupted_TC001_ClearConfirmedBeforeClose/<kind>` (same script, `ClearDelay` 300 ms) |
| page still requested, each kind | `TestBrowserInterrupted_TC002_PageStillRequestedIsCleared/navigation held, <kind>` (and `late navigation answer, server stop`) |

The issue #240 test documents already say so (issue #240 `testcases.md` TC-003 "existing tests, unchanged"; `doc/specifications/amazon-human-checks/technical.md`, issue #240 section, TC-003 row and files table giving `browser_cache_log_test.go` TC-001 and TC-002 only), so none of them is edited. The "keep and list it" option of the issue is not taken.

**Delivery (same as issue #322).** The change only deletes a test that passes on `master`, so a red phase cannot fail. Test files are locked after the TDD red commit, so the **tests stage** makes the deletion with `Edit` (the file stays; `delete.txt` is not used) and writes `<change folder>/no-tests.md`: why the red phase is skipped, the runs and fault injections below with their results, and where each TC is proved. The red gate is then skipped. The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** `S` is `go test -count=1 -run '^(TestBrowserInterrupted_TC001_ClearConfirmedBeforeClose|TestBrowserInterrupted_TC002_PageStillRequestedIsCleared|TestBrowserInterrupted_TC006_UninterruptedCheckIsUnchanged)$' -v ./internal/amazon/`. Each injection is a temporary edit of `clearCache` in `internal/amazon/browser.go`, reverted before the next step and never committed; if the sandbox refuses edits of `browser.go`, the edit is made in a scratch copy passed with `go test -overlay`, as for issue #322.

1. Before the deletion: `go test -count=1 -run '^TestBrowserCacheLog_' -v ./internal/amazon/` passes; record the test and subtest names and the elapsed time (TC-002 baseline).
2. After the deletion: the same command passes, with `-count=1` and `-count=2`; only TC-001 and TC-002 run, with the same subtest names as in run 1, about 8 s faster. `go test -count=1 -run 'TestBrowserCacheLog_TC003' -v ./internal/amazon/` passes with `testing: warning: no tests to run` (TC-001, TC-002).
3. `S` passes; its `-v` output lists both kinds of TC-001, the three subtests of TC-002 and TC-006 `download` (TC-005).
4. Injection A: in `clearCache`, `log.Printf("Amazon browser cache could not be cleared: %v", err)` moved before `if err != nil` (logged also on success). `S` fails with `cache clear failure logged:` in the subtests of TC-001, TC-002 and TC-006 `download`. Revert (TC-006).
5. Injection B: `if err == nil { log.Print("Amazon browser cache could not be cleared after a page not opened") }` added after the existing `if`. `S` fails in the same subtests with the same message. Revert; `S` passes again (TC-006).
6. `git diff master -- internal/amazon/browser.go internal/amazon/browser_interrupted_test.go` is empty; the gates of TC-008 pass twice.

- **Development stage.** No code. It sets FR-TEST-HYG-099–102 and TS-TEST-HYG-129–134 to implemented. It runs the gates and the repository checks of TC-001, TC-003, TC-004, TC-007 and TC-008, and restores any file outside `doc/` that differs from `master` and is not `internal/amazon/browser_cache_log_test.go` (for example a fault injection left in `browser.go`).

Refactoring: none. Removing the test is the change itself.

### `await close()` on its own line in TC-007 (issue #222)

Only `tests/app-information.spec.ts` changes, inside `TC-007 (#215) closing after fast tabbing returns focus to the menu control`. On `master`, line 378 is 6 spaces, `for (let presses = 0; presses <= count; presses += 1) await page.keyboard.press("Tab");`, 6 spaces and `await close();`. A `for` without braces takes one statement as its body, so `await close();` already runs once, after the loop; only the layout is misleading.

**Change.** Line 378 is replaced, in place, by two lines:

```ts
      for (let presses = 0; presses <= count; presses += 1) await page.keyboard.press("Tab");
      await close();
```

Each line is exactly 6 spaces followed by the text shown, with no trailing space (FR-TEST-HYG-108). No braces are added, the loop is not rewritten and no blank line is inserted. Nothing else in the file changes: the test name, `test.setTimeout(90_000)`, the closers and their order, the rounds, the `count` check, the checks after the close, the other tests, the helpers and the imports stay byte-for-byte the same (FR-TEST-HYG-109). The statement sequence the test executes is identical, so its results and timing are unchanged.

**Delivery (same as issue #295).** The change is the test code itself, and TC-007 passes on `master` before and after it, so a red phase cannot fail. Test files are locked after the TDD red commit, so the **tests stage** makes the edit above with `Edit` and writes `<change folder>/no-tests.md`: why the red phase is skipped, the runs below with their results, and where each TC is proved. The red gate is then skipped. The development stage writes no code.

**Runs and fault injection (tests stage, recorded in `no-tests.md`).** `E` is `npx playwright test tests/app-information.spec.ts`.

1. Before the edit: `E` passes; record `E --list` (TC-003 baseline).
2. After the edit: `grep -n 'press("Tab");  *await close();' tests/app-information.spec.ts` prints nothing; `grep -n ' $' tests/app-information.spec.ts` prints neither line; `grep -n 'await close();' tests/app-information.spec.ts` shows the close line directly after the loop line (TC-001). `git diff --unified=0 origin/master -- tests/app-information.spec.ts` is one hunk `@@ -378 +378,2 @@` and `git diff --stat` reports 2 insertions and 1 deletion (TC-002).
3. `E` passes, `E -g "TC-007" --repeat-each=2` passes, and `E --list` matches run 1 (TC-003).
4. Fault injection, never committed: wrap the loop body with the close (`{ await page.keyboard.press("Tab"); await close(); }`) and run `E -g "TC-007"`. It must fail (the closer then runs on every press); record the failing assertion, then revert with `Edit` and rerun `E -g "TC-007"`, which passes. This shows the test detects a close inside the loop, so the placement after the loop matters (TC-003 corner case).
5. `git diff --name-status origin/master -- . ':!doc'` lists only `M tests/app-information.spec.ts` (TC-004).

- **Development stage.** No code. It sets FR-TEST-HYG-108–110 and TS-TEST-HYG-141–144 to implemented (and the test suite hygiene row of `doc/FUNCTIONAL_SPECIFICATIONS.md` if it names issue #222). It runs the gates and the checks of TC-001, TC-002 and TC-004, and restores any file outside `doc/` that differs from `master` and is not `tests/app-information.spec.ts`.

Gates: `tsconfig.json` includes only `src` and `vite.config.ts`, and the repository has no lint script, so `npm run build` (`tsc -b && vite build`) does not read `tests/`; the spec file is parsed by Playwright in run 3. TC-004's type check, lint and build are the pipeline's frontend gates as they exist; no new gate is added.

Refactoring: none. The line break is the change itself.

### No redundant Amazon-answer check before `assertNavigationFailure` (issue #219)

Only `internal/amazon/browser_navigation_test.go` changes. On `master`, `assertNavigationFailure` starts with `t.Helper()` (line 33) and the first check (line 34) `if got.Result != "request_error" || got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error"`, which fails with `t.Fatalf("not a failure: %+v", got)` (line 35). Any result other than `request_error`, including `price_not_found`, `success` and `unavailable`, already fails there, so the three Amazon-answer checks before its calls are dead weight:

| Lines on `master` | Test / subtest | Check |
| --- | --- | --- |
| 100–102 | `TestBrowserNavigation_TC001_AttachmentIsAFailure` / `attachment then a normal page` | `an attachment gave the Amazon answer %q: %+v` |
| 145–147 | same test / `TC-002 existing Downloads folder, …` (3 subtests) | `%s gave the Amazon answer %q: %+v` |
| 178–180 | `TestBrowserNavigation_TC002_NoContentIsAFailure` / `HTTP 204`, `HTTP 205` | `HTTP %d gave the Amazon answer %q: %+v` |

**Change.**

- Delete each of the three `if` statements whole: its `if` line, its `t.Fatalf` line and its closing brace, 9 lines in all. The statements before and after (`}` of the request-count check or `logged := output()`, then the helper call) become adjacent; no blank line or comment is left in their place (FR-TEST-HYG-111).
- Replace line 35, in place and at the same indentation (two tabs), with `t.Fatalf("gave %q instead of a failure: %+v", got.Result, got)`. `%q` quotes the `Result` string and `%+v` prints the whole `model.CollectionResult` value with field names, for example `gave "success" instead of a failure: {Result:success AmountCents:12999 …}`. `t.Fatalf` stays, so the helper still stops the subtest; `t.Helper()` stays first, so the failure is reported at the caller's line (FR-TEST-HYG-112).
- Unchanged: the helper's doc comment, signature, condition, the URL/status check and the log-line loop with their messages; the file's header comment, imports (`strconv`, `io`, `os`, `filepath`, `atomic` are still used), `navigationServer`, TC-003, and every other statement of TC-001 and TC-002, including the comparisons with `success` after the normal page and in `inline page`, which do not precede a helper call. `answer.name` and `status` stay used by the subtest names. `browser_cache_log_test.go` and `browser_interrupted_test.go` keep calling the helper unchanged (FR-TEST-HYG-113).

**Failure path after the change (FR-TEST-HYG-113).** In each affected subtest, a check that gives `price_not_found`, `success` or `unavailable` now reaches the helper's first check, which fails the subtest once with `gave "<result>" instead of a failure: {…}` at the helper call line. As before, `t.Fatalf` stops the subtest, so the no-file checks after the call do not run on that path. On a normal run every result is `request_error`, so the tests run the same statements minus three always-false comparisons and pass as before.

**Delivery (same as issue #222).** The change is the test code itself and the tests pass on `master` before and after it, so a red phase cannot fail. Test files are locked after the TDD red commit, so the **tests stage** makes the edit above with `Edit` and writes `<change folder>/no-tests.md`: why the red phase is skipped, the runs below with their results, and where each TC is proved. The red gate is then skipped. The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** `N` is `go test -count=1 -run '^(TestBrowserNavigation_|TestBrowserCacheLog_|TestBrowserInterrupted_)' -v ./internal/amazon/`; `F` is `go test -count=1 -run '^TestBrowserNavigation_TC002_NoContentIsAFailure$' -v ./internal/amazon/`. Chromium is needed, as for the existing browser tests.

1. Before the edit: `N` passes; record its `=== RUN` lines (TC-005 baseline).
2. Before the edit, injection A (below) with the `price-found.html` page: `F` fails with `HTTP 204 gave the Amazon answer "success"` (baseline of the message the change replaces). Revert.
3. Make the edit. `grep -n 'gave the Amazon answer' internal/amazon/browser_navigation_test.go` and `grep -n 'price_not_found' internal/amazon/browser_navigation_test.go` print nothing; reading the four helper calls of the file shows no result comparison before any of them (TC-001). `grep -rn 'gave %q instead of a failure: %+v' internal/amazon/` prints one line, the helper's `t.Fatalf` with `got.Result, got`; `grep -n '"not a failure: ' internal/amazon/browser_navigation_test.go` prints nothing (TC-002).
4. `git diff --unified=0 origin/master -- internal/amazon/browser_navigation_test.go` shows one hunk replacing line 35 and three hunks removing lines 100–102, 145–147 and 178–180; `git diff --stat` reports 1 insertion and 10 deletions (TC-003).
5. Injection A, never committed: in `TestBrowserNavigation_TC002_NoContentIsAFailure`, read `page := amazontest.Fixture(t, "price-found.html")` before `navigationServer`, and make the handler set `Content-Type: text/html; charset=utf-8` and write `page` with status 200 instead of `w.WriteHeader(status)`. `F` fails in both `HTTP 204` and `HTTP 205`, once each, with `gave "success" instead of a failure: {`, reported at the helper call line of the test. With `price-not-found.html` instead, `F` fails with `gave "price_not_found" instead of a failure: {`. Injection B: the handler answers an empty `w.WriteHeader(http.StatusNotFound)`; `F` fails with `gave "unavailable" instead of a failure: {` (FR-TEST-HYG-113, TC-004). Revert with `Edit`; `git diff` shows only the change of run 4.
6. `N` passes twice; its `=== RUN` lines match run 1, with the subtests named in TC-005 (TC-005).
7. `git diff --name-status origin/master -- . ':!doc'` lists only `M internal/amazon/browser_navigation_test.go`; `gofmt -l internal/amazon` prints nothing; `go vet ./...` (no printf finding) and `go test ./...` pass (TC-006).

- **Development stage.** No code. It sets FR-TEST-HYG-111–114 and TS-TEST-HYG-145–149 to implemented, and the issue #219 entry of the test suite hygiene row of `doc/FUNCTIONAL_SPECIFICATIONS.md` to implemented. It runs the gates and the checks of TC-001 to TC-003 and TC-006, and restores any file outside `doc/` that differs from `master` and is not `internal/amazon/browser_navigation_test.go` (none is expected, as the injections edit that file only).

Refactoring: none. Removing the checks and rewording the message are the change itself.

### Shared browser-profile test helpers (issue #196)

On `master` the issue #180 tests carry their own copies of four helpers (functional terms of issue #196). The copies agree on the lock marks, the browser arguments and the fixture pages, and differ only where the `internal/amazon` versions are stricter: `killBrowserOnProfile` fails at once when the browser exits before holding the profile (the service's `killBrowser` waits the full 30 s), and `browserProcesses` gives `<pid>: <command line>` (the service's `profileProcesses` gives only the pid). The shared helpers take the `internal/amazon` form.

**New file `internal/amazon/amazontest/browser_profile.go`** (non-test file of the helper package, like `amazontest.go`; package doc comment unchanged). Every helper takes `testing.TB`, as `Browser`, `Fixture` and `NewServer` do, so the unit tests can pass a recording TB; `*testing.T` callers are unchanged. Each has a doc comment.

- `StaleLock(t testing.TB, profile, host string)` (FR-TEST-HYG-115). `t.Helper()`; `os.MkdirAll(profile, 0o700)`; then, for each of `SingletonLock`, `SingletonSocket`, `SingletonCookie` in that order, `os.RemoveAll(filepath.Join(profile, name))`; then, in the same order, `os.Symlink` of `host + "-12345"`, `/tmp/org.chromium.Chromium.gone/SingletonSocket` and `12345`. The names and targets are one ordered unexported table (a slice of name/target pairs, not a map, returned by `lockMarks(host)`), so the order of operations and the first error are deterministic. Any error: `t.Fatal(err)`; the `*PathError`/`*LinkError` text names the operation, the path and the cause (`not a directory`, `permission denied`). Other entries of the profile, including `SingletonLockExtra`, are not touched, as only the three exact names are removed.
- `BrowserProcesses(profile string) []string` (FR-TEST-HYG-116). The body of `browserProcesses` of `internal/amazon/browser_test.go`, unchanged: `os.ReadDir("/proc")` (error: `nil`), entries whose name is a number (`strconv.Atoi`), `os.ReadFile("/proc/<pid>/cmdline")` (error, for example a process that ended during the scan: skipped), kept when `strings.Contains(cmdline, profile)`, entry `<pid> + ": " + strings.ReplaceAll(cmdline, "\x00", " ")`. Each argument is followed by a space, so an entry ends with a space, as on `master`. No entry gives a `nil` slice.
- `KillBrowserOnProfile(t testing.TB, profile string)` (FR-TEST-HYG-117). The body of `killBrowserOnProfile` of `internal/amazon/browser_profile_test.go`, with the executable taken from `Browser(t)` instead of a parameter: remove `<profile>/SingletonLock` (`fs.ErrNotExist` ignored, any other error `t.Fatal`); `exec.Command(Browser(t), "--headless=new", "--user-data-dir="+profile, "--no-sandbox", "--no-first-run", "about:blank")` with `#nosec G204 -- test browser` and `SysProcAttr{Setpgid: true}`; `Start` error `t.Fatal`; a goroutine `Wait`s and closes `exited`; `kill` sends `SIGKILL` to `-pid` and waits for `exited`. Every 20 ms for 30 s: `Lstat(lock)` succeeds → stop waiting; `exited` closed → `t.Fatal("the browser exited before holding the profile")`; deadline passed → `kill()`, then `t.Fatal("the browser did not create SingletonLock in the profile")`. Then `time.Sleep(500ms)`, `kill()`, poll `BrowserProcesses(profile)` every 20 ms for up to 5 s; still non-empty → `t.Fatalf("browser processes left after the kill: %v", left)`; then `Lstat(lock)` error → `t.Fatalf("precondition: the killed browser left no SingletonLock: %v", err)`. The `Wait` goroutine always ends: the browser either exits or is killed.
- `ProfileHandler(t testing.TB) http.Handler` (FR-TEST-HYG-118). Reads `Fixture(t, "price-found.html")` and `Fixture(t, "aod-none.html")` once, then returns an `http.ServeMux` with: `/dp/` → `http.SetCookie` of `session-id=fixture-session-180`, `Path: "/"`, `MaxAge: 3600`, `Content-Type: text/html; charset=utf-8`, the page; `/gp/aod/ajax` and `/gp/aod/ajax/` → same content type, the `aod-none.html` page, no cookie; `/static/` → `WriteHeader(204)`. The mux answers every other path (`/`, `/other`, `/gp/aod`) with 404. The handler keeps no state, so it can serve several servers and repeated requests alike. It is the merged body of `browserProfileServer` and of `profileServer` without its blocking.

The new file imports only the standard library (`errors`, `io`, `io/fs`, `net/http`, `os`, `os/exec`, `path/filepath`, `strconv`, `strings`, `syscall`, `testing`, `time`); no `internal/service`, `internal/httpapi` or `internal/store`, so no import cycle. The helper package stays imported only by `_test.go` files, so the dependency check (FR-TEST-HYG-010) is unaffected.

**Callers (FR-TEST-HYG-119, FR-TEST-HYG-120).** Each local copy is deleted with its doc comment, and each call is replaced in place, with the same arguments otherwise:

| File | Delete | Replace |
| --- | --- | --- |
| `internal/amazon/browser_profile_test.go` | `writeStaleLock`, `killBrowserOnProfile` | `writeStaleLock(t, X, "otherhost")` → `amazontest.StaleLock(t, X, "otherhost")` (lines 253, 518, 605, 607, 656, 683, 750, 756); `killBrowserOnProfile(t, options.ExecPath, X)` → `amazontest.KillBrowserOnProfile(t, X)` (lines 636, 680) |
| `internal/amazon/browser_profile_values_test.go` | — | `writeStaleLock(t, profile, "otherhost")` → `amazontest.StaleLock(t, profile, "otherhost")` (lines 167, 258); add the `amazontest` import |
| `internal/amazon/browser_test.go` | `browserProcesses` | `assertNothingLeft` calls `amazontest.BrowserProcesses(profile)` in its loop and its check; its message, 5 s deadline and goroutine check are unchanged |
| `internal/service/amazon_browser_profile_test.go` | `leaveStaleLock`, `profileProcesses`, `killBrowser` | `leaveStaleLock(t, X)` → `amazontest.StaleLock(t, X, "otherhost")` (line 370); `killBrowser(t, X)` → `amazontest.KillBrowserOnProfile(t, X)` (lines 395, 411, 437, 479); `profileServer` rewritten (below) |
| `internal/httpapi/amazon_browser_profile_test.go` | `browserProfileServer`, `leaveOtherHostLock` | `browserProfileServer(t)` → `amazontest.NewServer(t, amazontest.ProfileHandler(t))` (2 calls); `leaveOtherHostLock(t, X)` → `amazontest.StaleLock(t, X, "otherhost")` (2 calls) |

Line numbers are those on `master`. The `internal/amazon` kill calls passed `options.ExecPath`, which `testBrowserOptions` takes from `amazontest.Browser(t)`, so the same browser starts. `profileLockNames`, `leaveLockDirectory`, `profileCollector`, `browserProfileOptions`, `profileFixtureHandler` and `pageCookieSent` stay. Imports that become unused (for example `os/exec`, `syscall`, `strconv`, `io`, `io/fs`, `errors`) are removed only where the compiler reports them; nothing else is reordered.

`profileServer` keeps its doc comment's meaning (reworded to name `amazontest.ProfileHandler`), signature `(t *testing.T, blocked string) (*amazontest.Server, <-chan string, func())`, the `arrived` buffer of 16, `release`, `sync.Once` and `t.Cleanup(releaseAll)` registered after `NewServer` (so it runs before the server closes). Its handler becomes one `http.HandlerFunc` passed to `amazontest.NewServer`: when `strings.CutPrefix(r.URL.Path, "/dp/")` matches, it reports the ASIN on `arrived` (non-blocking send) and, for `blocked`, waits for `release` or returns on `r.Context().Done()`; then every request goes to `pages.ServeHTTP(w, r)`, `pages` being `amazontest.ProfileHandler(t)` created once. No `http.SetCookie`, `aod-none.html`, `/gp/aod/ajax` or `/static/` remains in the file, and the page of `blocked` is still written only after release, with the cookie.

**Behaviour after the change (FR-TEST-HYG-121).** Normal runs execute the same steps: same lock marks, same browser arguments, same pages, so the tests and subtests keep their names and pass. Two failure paths change, as the functional specification states: `internal/service` leftover processes are now listed as `<pid>: <command line>`, and its kill steps fail at once with `the browser exited before holding the profile` when the browser exits first. The stale lock written is the same in the three packages, as all copies removed the three marks before writing the same links.

**Unit tests of the helpers: new `internal/amazon/amazontest/browser_profile_test.go`** (package `amazontest`, header comment naming issue #196 and the TCs proved outside the file). It reuses `recordingTB` of `amazontest_test.go` through a small local runner `record(t, func(tb testing.TB)) (failed bool, message string, took time.Duration)` that runs the function in its own goroutine, like `runCheck`. Stub browsers are `#!/bin/sh` scripts written to `t.TempDir()` with mode `0o700` and selected with `t.Setenv("PRICEFOLLOWER_BROWSER_PATH", stub)`. They are run by `/bin/sh <stub> --headless=new --user-data-dir=<profile> …`, so their own command line holds the profile; they read the profile from `--user-data-dir=` and loop with `while :; do sleep 0.1; done` rather than `exec sleep` (an `exec` would drop the profile from the command line). Processes whose command line must hold a path are started as `sh -c 'sleep 30; :' <path>`: the `$0` argument puts the path in the command line, and the trailing `:` stops the shell from replacing itself by `sleep`. GNU `sleep` refuses extra arguments, so `sleep 30 <path>` cannot be used. Every started process is killed and waited for in `t.Cleanup`.

| Test | TC | Design |
| --- | --- | --- |
| `TestStaleLock_Issue196_TC001_CreatesTheProfile` | TC-001 | `StaleLock` on `<tmp>/a/profile` and on a second path with `otherhost`; `os.ReadDir` gives exactly the three names; `Lstat` mode `ModeSymlink` and `Readlink` targets per host; when `/tmp/org.chromium.Chromium.gone` is absent before the call, it is still absent after it |
| `TestStaleLock_Issue196_TC002_ReplacesTheMarks` | TC-002 | subtests for the three rows, each with `Default/Preferences` and `SingletonLockExtra` written first; after the call, the three links of `somehost` and both files byte-identical; repeated call and dangling `SingletonLock` as extra subtests |
| `TestStaleLock_Issue196_TC003_ErrorFailsTheTest` | TC-003 | recording TB; `<tmp>/file/profile` with `file` a regular file → failed, message holds `not a directory`; a `0o500` folder with a regular `SingletonLock` → message holds `permission denied` and the path; skipped when `os.Geteuid() == 0`; `t.Cleanup` restores `0o700` |
| `TestBrowserProcesses_Issue196_TC004_ListsTheProfileProcesses` | TC-004 | unique path under `t.TempDir()`; empty before; with two `sh -c 'sleep 30; :' <path>` processes on two paths, each path gives exactly `strconv.Itoa(pid) + ": sh -c sleep 30; : " + path + " "`; empty after kill and `Wait`; the unreadable `/proc` branch is a code check |
| `TestKillBrowserOnProfile_Issue196_TC005_LeavesTheLock` | TC-005 | real Chromium (`Browser(t)` skips without it, fails in CI); subtests new profile, `StaleLock(…, "otherhost")` first, and a second kill on the same profile; each: no failure, `Lstat(SingletonLock)` succeeds, `BrowserProcesses` empty |
| `TestKillBrowserOnProfile_Issue196_TC006_ExitBeforeHolding` | TC-006 | recording TB; `/bin/true`, a stub `exit 3`, and a profile without `SingletonLock`; message exactly `the browser exited before holding the profile`, `took < 5 s`, stale `SingletonLock` gone |
| `TestKillBrowserOnProfile_Issue196_TC007_NeverHolding` | TC-007 | recording TB; stub loops and starts `sh -c 'sleep 120; :' <profile> &` in its group; message exactly `the browser did not create SingletonLock in the profile`, `30 s ≤ took ≤ 40 s`; `BrowserProcesses(profile)` empty afterwards (stub and child killed with the group); not parallel (`t.Setenv`) |
| `TestKillBrowserOnProfile_Issue196_TC008_LeftoverAndMissingLock` | TC-008 | recording TB; row 1: stub creates `SingletonLock`, runs `setsid sh -c 'sleep 60; :' <profile> &`, writes `$!` to `<tmp>/escaped.pid`, loops; message starts with `browser processes left after the kill: ` and holds `<escaped pid>: `; `took < 10 s` (500 ms plus the 5 s wait, not the 30 s one); `t.Cleanup` kills the pid read from the file. Row 2: stub creates the lock, `sleep 0.1`, removes it, loops; message starts with `precondition: the killed browser left no SingletonLock: ` and holds `no such file or directory` |
| `TestProfileHandler_Issue196_TC009_ServesTheProfilePages` | TC-009 | `NewServer(t, ProfileHandler(t))`, and a second server on the same handler; the nine requests with the status, content type, body and `Set-Cookie` checks of TC-009 (`resp.Cookies()` for name, value, path and `MaxAge`; no `session-id` cookie on the aod answers); each request sent twice |

TC-006 to TC-008 use `PRICEFOLLOWER_BROWSER_PATH` and therefore do not call `t.Parallel()`. TC-007 lasts about 30 s and TC-008 row 1 about 5.5 s, which `go test ./...` (default 10 min timeout) allows.

**Delivery.** The consumer files and the new test file are all `_test.go` files, which are locked after the TDD red commit, so the **tests stage** writes the new test file and makes every edit of the callers table and of `profileServer`, then runs `gofmt -w` on the six test files. The red gate fails because `StaleLock`, `BrowserProcesses`, `KillBrowserOnProfile` and `ProfileHandler` are undefined: `internal/amazon/amazontest`, `internal/amazon`, `internal/service` and `internal/httpapi` do not compile. That is the red phase; no `no-tests.md` is written. The **development stage** adds `internal/amazon/amazontest/browser_profile.go` only, runs the gates and the greps of TC-010, and marks FR-TEST-HYG-115–122 and TS-TEST-HYG-150–160 implemented, with the issue #196 entry of the test suite hygiene row of `doc/FUNCTIONAL_SPECIFICATIONS.md`. If a locked caller edit turns out wrong, the developer writes `test-problem.md`; the developer leaves the test files as locked (the fault injections below are reverted before the gates).

**Fault injections (developer stage, after the helpers exist; never committed; commands and results recorded in `<change folder>/implementation.md`).** TC-013: just before `assertNothingLeft` in `TestBrowserProfile_TC003_BrowserKilledOnThisMachineDoesNotPreventTheCheck`, start `sh -c 'sleep 30; :' <options.ProfileDir>` → that test fails with `browser processes left after the check: [<pid>: sh -c sleep 30; : <profile> ]`. TC-014: `PRICEFOLLOWER_BROWSER_PATH=/bin/true go test -count=1 -run '^TestAmazonBrowserProfile_TC009_ScheduledCheckAndRefreshAfterKilledBrowsers$' -v ./internal/service/` → each subtest fails at its kill step with `the browser exited before holding the profile` within a few seconds. Both reverted; `git diff` shows only the change.

Refactoring: the move itself is the change. Scope: the helper package gains one file; the five caller files lose their copies. Risk: low; the shared bodies are those of `internal/amazon`, already used by the issue #180 tests, and the service and HTTP fixtures serve byte-identical pages. The only behaviour difference (service failure messages and fail-fast on early exit) is the one FR-TEST-HYG-121 asks for.

### The early exit of `KillBrowserOnProfile` kills the browser group (issue #387)

On `master`, the `case <-exited:` branch of the wait loop of `KillBrowserOnProfile` (`internal/amazon/amazontest/browser_profile.go`, line 95) calls `t.Fatal("the browser exited before holding the profile")` without signalling the group. The 30 s branch and the normal path call `kill()`, which sends `SIGKILL` to `-cmd.Process.Pid` and then waits for `exited`.

**Change (FR-TEST-HYG-117, FR-TEST-HYG-123, FR-TEST-HYG-124).** The branch becomes `kill()` then the unchanged `t.Fatal`, as in the 30 s branch:

```go
case <-exited:
	kill()
	t.Fatal("the browser exited before holding the profile")
```

- The leader has been reaped, but the group lives on while one member runs, so `syscall.Kill(-pid, SIGKILL)` reaches it. With an empty group the call gives `ESRCH`, which `kill` already ignores (`_ =`).
- `<-exited` returns at once, as the channel is closed, so the failure adds no wait and stays well under 5 s. Its message and the use of `t.Fatal` are unchanged.
- `cmd.Wait` returns as soon as the leader exits even when a child is left: `Stdout`, `Stderr` and `Stdin` are nil, so they are `/dev/null` files with no copying goroutine. `exited` therefore closes even though a background child still runs.
- PID reuse: the group id is the reaped leader's pid. It still exists when a child is left; with no child, a new process reusing that pid as a group leader in the few microseconds before the call is the same negligible risk that the former service `killBrowser` and the 30 s branch take.
- Unchanged: the doc comment (it already says the helper starts the browser in its own group), the lock removal, the browser arguments, the 20 ms polling, the 30 s, 500 ms and 5 s waits, the other three messages and their order. Processes that leave the group (`setsid`) are not reached, as the functional specification excludes.

**Unit test: new `TestKillBrowserOnProfile_Issue387_TC001_ExitBeforeHoldingKillsTheGroup` in `internal/amazon/amazontest/browser_profile_test.go`** (FR-TEST-HYG-125), placed after the issue #196 TC-008 test. It reuses `record`, `writeStub`, `killPidFile`, `readFile` and `waitNoProcesses`. Two subtests, `stale lock` and `no SingletonLock`, each:

1. `profile := filepath.Join(t.TempDir(), "profile")`, `os.MkdirAll(profile, 0o700)`; for `stale lock`, `os.Symlink("otherhost-12345", <profile>/SingletonLock)`.
2. `child := filepath.Join(t.TempDir(), "child.pid")`; `t.Cleanup(func() { killPidFile(child, profile, false) })`, registered before the stub so a failing run leaves no child.
3. `stub, _ := writeStub(t, fmt.Sprintf("sh -c 'sleep 60; :' \"$profile\" &\necho $! > '%s'\nexit 0", child))` and `t.Setenv("PRICEFOLLOWER_BROWSER_PATH", stub)`. The stub never creates `SingletonLock`. Its background child has no job control, so it stays in the stub's group, and its command line holds the profile both before and after it runs `sh -c` (the forked shell holds the stub's `--user-data-dir=<profile>`). The test cases write the child as `sleep 60 "$profile" &`; GNU `sleep` rejects the extra operand (`invalid time interval`) and would exit at once, so the `sh -c 'sleep N; :' <path>` form of the issue #196 tests is used. The `sleep 60` grandchild is in the same group, so `writeStub`'s cleanup (a kill of the stub's group) removes it too.
4. `failed, message, took := record(t, func(tb testing.TB) { KillBrowserOnProfile(tb, profile) })`.
5. Checks, with `t.Errorf` unless stated: `failed && message == "the browser exited before holding the profile"` (exact, so no suffix or listed process); `took < 5*time.Second`; `pid := strings.TrimSpace(readFile(t, child))` is a number above 1 (`strconv.Atoi`, `t.Fatalf` otherwise), which proves the child was started, as the stub wrote the file before it exited; `left := waitNoProcesses(profile)` is empty, else `t.Errorf("processes left after the early exit (child %s): %q", pid, left)`; `os.Lstat(<profile>/SingletonLock)` gives `fs.ErrNotExist`.

Not parallel (`t.Setenv`). Each subtest takes well under a second with the change; without it, the leftover check waits its full 5 s and fails, and the cleanups kill the child. The file's header comment gains one sentence: issue #387's TC-001 is proved by this test, TC-002 by the issue #196 TC-006 test and TC-004 by its TC-005, TC-007 and TC-008 tests; TC-003 and TC-005 to TC-007 are runs and checks. `TestKillBrowserOnProfile_Issue196_TC006_ExitBeforeHolding` and every other test of the file are unchanged.

**Delivery.** The **tests stage** adds the test and the header sentence, then `gofmt -w` the file. Red: on `master`'s helper the two subtests fail after the 5 s wait with `processes left after the early exit (child <pid>): ` followed by the child's entry (`<pid>: sh -c sleep 60; : <profile> `), while every issue #196 test still passes. The **development stage** adds the one `kill()` line, runs the gates and the runs below, records commands and results in `<change folder>/implementation.md`, and marks FR-TEST-HYG-123–126 (and FR-TEST-HYG-117 as updated) and TS-TEST-HYG-161–166 implemented, with the issue #387 entry of the test suite hygiene row of `doc/FUNCTIONAL_SPECIFICATIONS.md`.

**Runs (development stage, recorded in `implementation.md`).** `K` is `go test -count=1 -run '^TestKillBrowserOnProfile_' -v ./internal/amazon/amazontest/`, with Chromium.

1. `K` passes, and passes with `-count=2` (TC-001, TC-002, TC-004).
2. Fault injection, never committed: remove the new `kill()` line. `K` fails in both subtests of the issue #387 test with `processes left after the early exit`, while the three TC-006 rows pass; after the run, `pgrep -af 'sleep 60'` lists no process started by the test (the cleanups killed the child and its `sleep`). Restore the line; `git diff` shows only the change; `K` passes again (TC-003).
3. `go test -count=1 ./internal/amazon/... ./internal/service/... ./internal/httpapi/...` passes; `go test -list .` of those packages gives the same names as on `master` plus the new test (TC-005).
4. `git diff --name-only origin/master -- . ':!doc'` lists only the two helper files; `gofmt -l internal` prints nothing; `go vet ./...` and `go test ./...` pass, including `TestServerDoesNotDependOnTestHelper` (TC-006, TC-007).

Refactoring: none. The added call is the change itself; no helper is extracted, as the 30 s branch keeps its own `kill()` call.

### Shared log and standard output capture (issue #218)

On `master` the three local capture copies (functional terms of issue #218) share one body: redirect `log` to a mutex-guarded buffer, create a pipe with `os.Pipe`, set `os.Stdout` to its write end, copy the read end into a buffer in a goroutine, and on the end restore `os.Stdout`, close the write end, wait for the copy, close the read end and restore `log`. `captureOutput` and `captureCheckOutput` return a `sync.Once` ending read registered with `t.Cleanup`; `captureProfileOutput` copies both streams into one buffer, returns it for live reads (`String()`) and ends only in `t.Cleanup`. The shared helper keeps that body and offers both reads.

**New file `internal/amazon/amazontest/capture.go`** (non-test file of the helper package, like `browser_profile.go`; package doc comment unchanged; standard library only: `io`, `log`, `os`, `strings` or `bytes`, `sync`, `testing`).

```go
// Capture holds what the code under test logs (standard logger) and prints (os.Stdout) from
// StartCapture until End, which the test's cleanup also calls.
type Capture struct { /* logs, printed outputBuffer; previousLog io.Writer; previousStdout, writer, reader *os.File; copied chan struct{}; once sync.Once; text string */ }

func StartCapture(t testing.TB) *Capture
func (c *Capture) Live() string
func (c *Capture) End() string

// CaptureOutput captures the logged and printed text until the returned function is first called
// or the test ends; the function returns the logged text followed by the printed text.
func CaptureOutput(t *testing.T) func() string { t.Helper(); return StartCapture(t).End }
```

- `outputBuffer` (unexported): a `sync.Mutex` and a `bytes.Buffer`, with `Write` and `String` under the lock, as `master`'s `lockedBuffer`. It must not be named `lockedBuffer` or `profileOutput`, since TC-010's grep forbids those names in every Go file. The capture has two: `logs` and `printed`.
- `StartCapture(t testing.TB)` (FR-TEST-HYG-127). `t.Helper()`. It creates the pipe **first**: `reader, writer, err := os.Pipe()`, and on error calls `t.Fatalf("capture the standard output: %v", err)`. The `*os.SyscallError` text names the cause, for example `pipe2: too many open files`. Creating the pipe before `log.SetOutput` means a failed setup changes neither destination (on `master` the logger stayed redirected). Then: `previousLog := log.Writer()`, `log.SetOutput(&c.logs)`, `previousStdout := os.Stdout`, `os.Stdout = writer`, `go func() { io.Copy(&c.printed, reader); close(c.copied) }()`, `t.Cleanup(func() { c.End() })`, and it returns `c`. `testing.TB`, like the package's other helpers; `CaptureOutput` keeps the required `*testing.T`.
- `End()` (FR-TEST-HYG-128). Inside `c.once.Do`: `os.Stdout = previousStdout`, `writer.Close()`, `<-c.copied` (the copy reaches EOF only once every byte written before the close is read, so text printed before the call is whole, even past the pipe buffer), `reader.Close()`, `log.SetOutput(previousLog)`, then `c.text = c.logs.String() + c.printed.String()`. It returns `c.text`. `sync.Once` makes later calls, and the cleanup, return the same text at once. Restoring in this order is `master`'s. After `End`, `log` writes go to the previous writer and `os.Stdout` writes to the previous file. A goroutine that still holds the old `os.Stdout` value gets `file already closed`. So nothing more is captured.
- `Live()` (FR-TEST-HYG-129): `c.logs.String() + c.printed.String()`, under each buffer's lock, with no wait and no change of state. The copy goroutine may not yet have moved the last printed bytes into `printed`, so text printed just before may appear only in a later read. Each part only grows until `End`, so a live read's logged part is a prefix of the ending read's logged part, and likewise for the printed part. After `End` the buffers no longer change, so `Live()` equals the ending read.
- Concurrency (FR-TEST-HYG-127). `log.Logger` already serialises its writes. The output buffers lock every `Write` and `String`, so concurrent writers, the copy goroutine and live readers do not race. `os.File` writes are safe across goroutines. Each `fmt.Println` is one `write` of at most `PIPE_BUF` bytes, so it is not interleaved. As on `master`, the global `os.Stdout` is assigned only in `StartCapture` and `End`, so tests must not print from goroutines that run across those two calls. The `internal/service` tests already started the service before the capture on `master`, and this does not change.
- Order: logged text, then printed text, in every read, as `captureOutput`/`captureCheckOutput` returned. `captureProfileOutput` interleaved both streams in one buffer. Its readers (`waitStartFailures`, TC-009's `strings.Contains`) only count or search whole lines, and every log line ends with `\n`, so the split does not change their results.
- Nesting: each capture saves the destinations current at its start. Sequential captures, and nested captures ended in reverse order (directly or by LIFO cleanups), therefore restore the destinations that were in place before the first capture.

**Callers (FR-TEST-HYG-130).** Replacements are made in place, and nothing else in the files changes:

| File | Delete | Replace |
| --- | --- | --- |
| `internal/amazon/browser_test.go` | `captureOutput` with its doc comment, `lockedBuffer` and its two methods | `captureOutput(t)` → `amazontest.CaptureOutput(t)` (line 340) |
| `internal/amazon/browser_interrupted_test.go`, `browser_retrieval_test.go`, `browser_navigation_test.go`, `browser_profile_test.go`, `browser_error_page_test.go`, `browser_profile_values_test.go` | — | `captureOutput(t)` → `amazontest.CaptureOutput(t)` (6, 4, 4, 3, 2 and 1 calls) |
| `internal/amazon/browser_cache_log_test.go`, `browser_empty_status_test.go` | — | same (4 and 1 calls); add the `amazontest` import |
| `internal/httpapi/amazon_browser_navigation_test.go` | `captureCheckOutput` with its doc comment | `captureCheckOutput(t)` → `amazontest.CaptureOutput(t)` (2 calls) |
| `internal/httpapi/amazon_browser_interrupted_test.go` | — | same (2 calls, `output` and `refreshOutput`) |
| `internal/httpapi/amazon_browser_error_page_test.go` | — | same (2 calls); add the `amazontest` import |
| `internal/service/amazon_browser_profile_test.go` | `profileOutput` with its two methods, `captureProfileOutput` with its doc comment | `captureProfileOutput(t)` → `amazontest.StartCapture(t)` (3 calls: TC-007 both subtests, TC-009 first subtest); `waitStartFailures(t *testing.T, output *amazontest.Capture, want int)` with both `output.String()` → `output.Live()`; TC-009's `output.String()` → `output.Live()` |

Calls are counted on `master`. The variables keep their names (`output`, `logged`, `refreshOutput`) and are read as before (`output()`). Imports that become unused (`bytes`, `io`, `log`, `os`, `sync` in the three declaring files) are removed only where the compiler reports them. `syncBuffer` and the logger-only helpers (`captureRetrievalLog`, `captureSecretLogs`, `captureLogs`, `captureLog`) stay. The amazontest package does not import `internal/amazon`, so `package amazon` tests can use it without an import cycle, as they already do for `Browser`.

**Unit tests: new `internal/amazon/amazontest/capture_test.go`** (package `amazontest`). Its header comment names issue #218 and says that TC-010 to TC-013 are repository checks and runs. The comment must not spell the former helper or type names, because TC-010's word grep covers comments too. Shared local helper (no name already in the package's test files, such as `readFile`, `writeFile` or `record`): `previousDestinations(t) (*outputBuffer, *os.File)`. It sets `log.SetOutput` to a new buffer and `os.Stdout` to a file created in `t.TempDir()`, and `t.Cleanup` restores the originals. It returns the buffer and the file, whose content the checks read with `os.ReadFile`. The cases compare `log.Writer()` and `os.Stdout` with these values by identity. No test calls `t.Parallel()`, since they all change process-wide state.

| Test | TC | Design |
| --- | --- | --- |
| `TestCaptureOutput_Issue218_TC001_Signature` | TC-001 | `var f func(*testing.T) func() string = CaptureOutput` (compile-time check); `go doc` and the callers' compilation are runs |
| `TestCaptureOutput_Issue218_TC002_LoggedThenPrinted` | TC-002 | `fmt.Println("printed")` then `log.Print("logged")`; `strings.HasSuffix(read, "logged\nprinted\n")`; previous buffer and file empty; identity restored. Subtests for several lines, `fmt.Print("partial")` (`HasSuffix(read, "partial")`) and no write (`""`) |
| `TestCaptureOutput_Issue218_TC003_LaterCallsSameText` | TC-003 | after the first call, `after-log`/`after-print` written; 2nd and 3rd calls equal the first, each `< 1 s`; `after-log` in the buffer, `after-print` in the file; a `StartCapture`/`End`/`Live` variant checks the live read after the end |
| `TestCaptureOutput_Issue218_TC004_WaitsForPrintedText` | TC-004 | `fmt.Print(strings.Repeat("0123456789abcdef", 65536))` then `fmt.Println("marker")`; the read equals the expected bytes exactly (no logged text); subtest with a goroutine joined by `sync.WaitGroup` before the call. The `-count=20` corner is a run |
| `TestCaptureOutput_Issue218_TC005_EndsAtTestEnd` | TC-005 | parent sets the destinations; `t.Run` keeps `read := CaptureOutput(t)` in a parent variable; after it, identity restored, both destinations without `logged`/`printed`; `read()` ends with `logged\nprinted\n`; parent writes afterwards are absent from `read()` |
| `TestCaptureOutput_Issue218_TC006_SequentialCaptures` | TC-006 | in a subtest, capture 1 (`one`) ended, then capture 2 (`two`) ended; each holds only its word; after `t.Run`, identity is the parent's destinations. Nested subtest: capture 2 started before capture 1 is ended; `read2()` then `read1()` |
| `TestCaptureOutput_Issue218_TC007_LiveRead` | TC-007 | `c := StartCapture(t)`; `c.Live() == ""`; after `log.Print("logged")` the live read holds it (the log write is synchronous); after `fmt.Println("printed")` poll `Live()` every 10 ms for up to 5 s; `os.Stdout != previous file` and both destinations empty; `log.Print("later")`; `End()` holds all three. Each `Live()` timed `< 1 s`; two live reads with no write in between are equal. Every non-empty line of each live read is a line of `End()` |
| `TestCaptureOutput_Issue218_TC008_ConcurrentWritesNoRace` | TC-008 | `c := StartCapture(t)`; 4 goroutines `log.Printf("log-%d-%03d", g, i)` and 4 `fmt.Printf("print-%d-%03d\n", g, i)`, 100 each; a reader loops `c.Live()` until a `done` channel closes; `WaitGroup`, then `End()`; split into lines; each expected line appears exactly once (a log line matched with `strings.HasSuffix(line, " log-g-iii")`, a printed line with equality); the last log line's index is before the first printed line's. Race freedom is proved by the `-race` run |
| `TestCaptureOutput_Issue218_TC009_SetupFailureFailsTheTest` | TC-009 | When `AMAZONTEST_CAPTURE_SETUP_FAILURE=1` is set, the child branch runs. It lowers the soft `RLIMIT_NOFILE` to 64 with `syscall.Getrlimit`/`Setrlimit` and opens `os.DevNull` until `errors.Is(err, syscall.EMFILE)` (the files are kept open, at most 64 iterations). It then calls `CaptureOutput(t)` and, if that returns, `t.Log("capture set up")`. The parent branch runs `exec.Command(os.Args[0], "-test.run=^TestCaptureOutput_Issue218_TC009_SetupFailureFailsTheTest$", "-test.count=1", "-test.v")` with the variable added to `os.Environ()`, `#nosec G204 -- test binary`. It requires a non-nil `*exec.ExitError` and output holding `too many open files` and `--- FAIL`, without `capture set up`. The parent changes no limit or destination |

**Delivery.** All the files to change except `capture.go` are `_test.go` files, which are locked after the TDD red commit. The **tests stage** therefore writes `capture_test.go`, makes every edit of the callers table, and runs `gofmt -w` on the files it changed. Red: `StartCapture`, `Capture` and `CaptureOutput` are undefined, so `internal/amazon/amazontest`, `internal/amazon`, `internal/service` and `internal/httpapi` do not compile. No `no-tests.md` is written. The **development stage** adds `internal/amazon/amazontest/capture.go` only and runs the gates and the runs below. It records commands and results in `<change folder>/implementation.md`, and marks FR-TEST-HYG-127–131 and TS-TEST-HYG-167–175 implemented, along with the issue #218 entry of the test suite hygiene row of `doc/FUNCTIONAL_SPECIFICATIONS.md`. If a locked caller edit turns out wrong, the developer writes `test-problem.md`.

**Runs (development stage, recorded in `implementation.md`).** `C` is `go test -count=1 -run '^TestCaptureOutput_Issue218_' -v ./internal/amazon/amazontest/`.

1. `C` passes; `go test -race -count=1 -run '^TestCaptureOutput_Issue218_' ./internal/amazon/amazontest/` passes with no race report; `go test -count=20 -run '^TestCaptureOutput_Issue218_TC004' ./internal/amazon/amazontest/` passes (TC-002 to TC-009).
2. `go doc pricefollower.local/internal/amazon/amazontest CaptureOutput` shows `func CaptureOutput(t *testing.T) func() string` (TC-001).
3. The three greps and the word grep of TC-010 give the expected output (TC-010).
4. With Chromium, `go test -count=1 -v ./internal/amazon/ ./internal/service/ ./internal/httpapi/ | grep -E '^\s*=== RUN' | sort` is compared with the same command run on a copy of `master` (`git archive origin/master | tar -x -C "$TMPDIR/master"`). The lists are equal and both runs pass (TC-011).
5. Fault injection, never committed: `Live` returns `""`. `go test -count=1 -run '^TestAmazonBrowserProfile_TC007_FailureIsLoggedCountedAndStopsItems$' ./internal/service/` fails in `waitStartFailures` with `0 browser start failures logged, want 1`. Restore; it passes; `git diff` shows only the change (TC-011).
6. `git diff --name-only origin/master -- . ':!doc'` lists only `internal/amazon/amazontest/capture.go`, `capture_test.go` and the caller files of the table. `gofmt -l internal` prints nothing. `go vet ./...` and `go test ./...` pass, including `TestServerDoesNotDependOnTestHelper` (TC-012, TC-013).

Refactoring: the merge of the three copies is the change. Scope: one new non-test file and one new test file in the helper package, three declaring files lose their copies, and the thirteen caller files (the three declaring files included) change only their calls, types, reads and imports. Risk: low. The capture body is `master`'s, with two deliberate differences. A failed pipe creation no longer leaves the logger redirected. The `internal/service` live reads give logged text before printed text instead of the arrival order, which their line-based readers do not notice (run 4 proves it).

### Accurate first message of `assertNavigationFailure` (issue #384)

Only `internal/amazon/browser_navigation_test.go` changes. On `master`, `assertNavigationFailure` starts with `t.Helper()` (line 33), then the first check (lines 34–36):

```go
if got.Result != "request_error" || got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error" {
	t.Fatalf("gave %q instead of a failure: %+v", got.Result, got)
}
```

`amazonCheckFailed` is the constant of `internal/amazon/browser_test.go` (`Amazon could not be reached for a price check.`).

**Change (FR-TEST-HYG-112, FR-TEST-HYG-132, FR-TEST-HYG-133).** Only line 34 changes in place: its condition keeps only `got.Result != "request_error"`. Lines 35 and 36 stay. Right after them, at the same indentation (one tab), the failure-detail check is inserted:

```go
if got.Result != "request_error" {
	t.Fatalf("gave %q instead of a failure: %+v", got.Result, got)
}
if got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error" {
	t.Fatalf("gave a failure with message %q and second-hand status %q, not %q and %q: %+v", got.Message, got.SecondHandStatus, amazonCheckFailed, "check_error", got)
}
```

- The message is the example of FR-TEST-HYG-133. The expected status is passed as a `%q` argument rather than written in the format, so the format needs no escaped quotes and each expected value appears once in the condition and once in the arguments. Five verbs take five arguments, so `go vet` finds no printf problem. The message holds neither `instead of a failure` nor `gave "request_error"`. For example: `gave a failure with message "…x" and second-hand status "check_error", not "Amazon could not be reached for a price check." and "check_error": {Result:request_error …}`.
- Both checks use `t.Fatalf`, so a wrong result stops the subtest at the result check, and the failure-detail check, the URL/status check and the log-line loop do not run (FR-TEST-HYG-132). A wrong message or status stops at the failure-detail check, before the URL/status check (FR-TEST-HYG-133). `t.Helper()` stays first, so both failures are reported at the caller's line.
- Accepted results do not change (FR-TEST-HYG-134). The two conditions together equal the former condition, so a result passes both checks exactly when it passed the former one.
- Unchanged: the doc comment (it does not describe the first check), the signature, the URL/status check, the log-line loop and their messages, the imports, and every other line of the file. The callers in `browser_navigation_test.go`, `browser_cache_log_test.go` and `browser_interrupted_test.go` are not touched (FR-TEST-HYG-134, FR-TEST-HYG-135).

**Delivery (same as issues #219 and #222).** The change edits only test code for behaviour already on `master`, and the tests pass before and after it, so no red phase can fail and no manual follow-up is needed for the TDD red gate. Test files are locked after the TDD red commit, so the **tests stage** makes the edit above with `Edit`. It writes `<change folder>/no-tests.md`: why the red phase is skipped, the runs and fault injections below with their results, and where each TC is proved. The development stage writes no code.

**Runs and fault injections (tests stage, recorded in `no-tests.md`).** `N` is `go test -count=1 -run '^(TestBrowserNavigation_|TestBrowserCacheLog_|TestBrowserInterrupted_)' -v ./internal/amazon/`. `F` is `go test -count=1 -run '^TestBrowserNavigation_TC002_NoContentIsAFailure$' -v ./internal/amazon/`. Chromium is needed, as for the existing browser tests. Every injection is a temporary edit made with `Edit`, reverted with `Edit` and never committed. After each revert, `git diff` shows only the change.

1. Before the edit: `N` passes. Record its `=== RUN` lines (TC-006 baseline).
2. Before the edit, injection C (below, applied to the combined condition only): `F` fails with `gave "request_error" instead of a failure: {`. This is the baseline of the contradictory message the change removes. Revert.
3. Make the edit. `grep -rn 'gave %q instead of a failure: %+v' internal/amazon/` prints one line, the result check's `t.Fatalf` with `got.Result, got`. `grep -n '"not a failure: ' internal/amazon/browser_navigation_test.go` prints nothing. Reading the helper shows the order and conditions of TC-001 (TC-001).
4. `git diff --unified=0 origin/master -- internal/amazon/browser_navigation_test.go` touches only lines 34–36 of `master` (the first check) and adds only the result check's condition line and the three lines of the failure-detail check (Git may align the unchanged `t.Fatalf` line and brace either way). `git diff --stat` reports 4 insertions and 1 deletion (TC-002).
5. Injection A of the issue #219 design: in `TestBrowserNavigation_TC002_NoContentIsAFailure`, read `page := amazontest.Fixture(t, "price-found.html")` before `navigationServer`. Make the handler set `Content-Type: text/html; charset=utf-8` and write `page` with status 200 instead of `w.WriteHeader(status)`. `F` fails in `HTTP 204` and `HTTP 205`, once each, with `gave "success" instead of a failure: {`, reported at the helper call line. The output does not hold `gave a failure with message`. With `price-not-found.html`, `F` fails with `gave "price_not_found" instead of a failure: {` and no failure-detail message. Revert (TC-003).
6. Injection C: in the failure-detail check, replace `amazonCheckFailed` with `amazonCheckFailed + "x"` in the condition and in the third argument. `F` fails in both subtests, once each. The message holds `"Amazon could not be reached for a price check."`, `"Amazon could not be reached for a price check.x"` and `"check_error"`, and ends with `{Result:request_error …}`. It holds neither `instead of a failure` nor `gave "request_error"`. The output holds neither `diagnostics of a page not opened` nor `no log line with the request URL`. Revert (TC-004).
7. Injection D: in the failure-detail check, replace `"check_error"` with `"check_errorx"` in the condition and in the fourth argument. `F` fails with a message that holds `"check_error"`, `"check_errorx"` and the given and expected message `"Amazon could not be reached for a price check."`, ends with the whole result, and does not hold `instead of a failure`. Revert (TC-005).
8. `N` passes twice. Its `=== RUN` lines match run 1 (TC-006).
9. `git diff --name-status origin/master -- . ':!doc'` lists only `M internal/amazon/browser_navigation_test.go`. `gofmt -l internal/amazon` prints nothing. `go vet ./...` (no printf finding) and `go test ./...` pass (TC-007).

- **Development stage.** No code. It sets FR-TEST-HYG-132–135 and TS-TEST-HYG-176–180 to implemented, keeps FR-TEST-HYG-112 and TS-TEST-HYG-146 implemented as updated, and sets the issue #384 entry of the test suite hygiene row of `doc/FUNCTIONAL_SPECIFICATIONS.md` to implemented. It runs the gates and the checks of TC-001, TC-002 and TC-007. It restores any file outside `doc/` that differs from `master` and is not `internal/amazon/browser_navigation_test.go`. None is expected, as the injections edit that file only.

Refactoring: none. Splitting the check is the change itself.

## Tiers

- Frontend: not affected. Only Go test and test-support code changes; no view, component, state or asset.
- Backend: not affected at runtime. `internal/amazon/amazontest` is imported only by `_test.go` files, so the server binary does not change; issue #244 makes this a checked guarantee. No service, repository, validation, persistence, migration or concurrency change. `AssertNoDownload` runs after the browser has closed, and the dependency check runs `go list` as a child process of the test, so no application concurrency changes.
- API: not affected. No endpoint, request, response or error changes; `API_SPECIFICATION.md` is unchanged.
- Issue #251 touches only a `_test.go` file and the temporary modules it writes under `t.TempDir()`. No frontend, backend, API, database or migration change, and the server binary is the same.
- Issue #241 deletes one file under `doc/todo/` and closes a GitHub issue. Frontend, backend and API: not affected, as no file outside `doc/` changes; `API_SPECIFICATION.md` is unchanged.
- Issue #185 adds two `_test.go` files. Frontend: not affected, as no view, component or Playwright spec changes. Backend: not affected at runtime. No service, store, validation, persistence, migration or concurrency change; the tests run the existing code on a temporary SQLite file and a local `amazontest` server. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #238 changes no file outside `doc/`. Frontend: not affected, as no view, component or Playwright spec changes. Backend: not affected, as no Go file changes (TC-005 is kept as it is on `master`). API: not affected, `API_SPECIFICATION.md` is unchanged.
- Issue #181 changes no file outside `doc/`. Frontend: not affected, as no view, component or Playwright spec changes. Backend: not affected, as no Go file changes (the status classification of FR-AMZ-HCHK-012 and its tests stay as on `master`). API: not affected, `API_SPECIFICATION.md` is unchanged.
- Issue #182 changes one `_test.go` file and adds another in `internal/service/`. Frontend: not affected, as no view, component or Playwright spec changes. Backend: not affected at runtime. No service, store, validation, persistence, migration or concurrency change. The tests run the existing scheduler, gate and Refresh price code on a temporary SQLite file, with a fake clock and a fake Amazon collector. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #295 changes one line of one Playwright spec. Frontend: not affected, as no view, component, state, style or asset changes; the spec keeps mocking the API and the image as before. Backend: not affected, as no Go file changes. API: not affected, as the mocked requests and responses are unchanged and `API_SPECIFICATION.md` is unchanged.
- Issue #195 changes one `_test.go` file in `internal/amazon/`. Frontend: not affected, as no view, component or Playwright spec changes. Backend: not affected at runtime. No production Go file changes, including the removal of the lock marks in `browser.go`; the server binary is the same, as `_test.go` files are not part of it. No validation, persistence, migration or application concurrency change; the only concurrency is in the test (the `Wait` goroutine, whose channel close orders the read of the output after its writes). API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #312 changes the wait of one test in `internal/amazon/browser_profile_test.go`. Frontend: not affected, as no view, component or Playwright spec changes. Backend: not affected at runtime, as no production Go file changes and `_test.go` files are not part of the server binary; no validation, persistence, migration or application concurrency change. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #315 changes the hold stub and the hold rows' wait in `internal/amazon/browser_profile_test.go`. Frontend: not affected, as no view, component or Playwright spec changes. Backend: not affected at runtime, as no production Go file changes and `_test.go` files are not part of the server binary; no validation, persistence, migration or application concurrency change. The only new concurrency is in the test: the stub process creates a marker file that the test goroutine polls. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #324 adds a `t.Cleanup` to one subtest of `internal/amazon/browser_profile_values_test.go`. Frontend: not affected, as no view, component or Playwright spec changes. Backend: not affected at runtime, as `BrowserCollector.profile` and every production Go file are unchanged and `_test.go` files are not part of the server binary; no validation, persistence, migration or concurrency change (the cleanup runs in the test's own goroutine after the subtest). API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #290 changes the checks of one Playwright test in `tests/claude-token-settings.spec.ts`. Frontend: not affected, as `SettingsModal.tsx` and every view, component, state, style and asset are unchanged (edited only temporarily for fault injections); its behaviour stays FR-SECRET-014/TS-SECRET-019, and there is no accessibility or UI state change. Backend: not affected, as no Go file changes. API: not affected, as the mocked `PUT /api/v1/settings` contract and the other mocked requests are unchanged and `API_SPECIFICATION.md` is unchanged.
- Issue #322 deletes one test and four standard-library imports from `internal/amazon/browser_profile_values_test.go`. Frontend: not affected, as no view, component, state, style, asset or Playwright spec changes. Backend: not affected at runtime, as `BrowserCollector.profile`, `browser.go` and every production Go file are unchanged (edited only temporarily for fault injections) and `_test.go` files are not part of the server binary; no validation, persistence, migration or concurrency change, and `go.mod`/`go.sum` are unchanged (`go/*` are standard library). API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #254 changes `buildModule`, one test's comment and subtests, and the header comment of `internal/amazon/amazontest/server_dependencies_test.go`. Frontend: not affected, as no view, component, state, style, asset or Playwright spec changes. Backend: not affected at runtime, as every production Go file and the non-test files of `internal/amazon/amazontest/` are unchanged and `_test.go` files are not part of the server binary; no validation, persistence, migration or concurrency change (the go commands run one after the other as child processes of the test, on modules under `t.TempDir()`). `go.mod` and `go.sum` are unchanged. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #253 changes one `switch` branch and two comments of `internal/amazon/amazontest/server_dependencies_test.go`. Frontend: not affected, as no view, component, state, style, asset or Playwright spec changes. Backend: not affected at runtime, as every production Go file and the non-test files of `internal/amazon/amazontest/` are unchanged and `_test.go` files are not part of the server binary; no validation, persistence, migration or concurrency change (`buildModule` and its child processes are unchanged). `go.mod` and `go.sum` are unchanged. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #340 changes `buildModule`'s vet arguments, one control check, one new subtest and three comments of `internal/amazon/amazontest/server_dependencies_test.go`. Frontend: not affected, as no view, component, state, style, asset or Playwright spec changes. Backend: not affected at runtime, as every production Go file and the non-test files of `internal/amazon/amazontest/` are unchanged and `_test.go` files are not part of the server binary; no validation, persistence, migration or concurrency change (the go commands still run one after the other as child processes of the test, on modules under `t.TempDir()`). `go.mod`, `go.sum` and the repository's own `go vet ./...` gate are unchanged. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #239 adds two checks to one test of `internal/amazon/browser_interrupted_test.go` and edits one line of the issue #233 test case document. Frontend: not affected, as no view, component, state, style, asset or Playwright spec changes. Backend: not affected at runtime, as `clearCache`, its 5 s bound, its log line (FR-AMZ-HCHK-015) and every production Go file are unchanged (edited only temporarily for fault injections) and `_test.go` files are not part of the server binary; no validation, persistence, migration or concurrency change. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #344 adds a findings guard to one subtest and a header block to `internal/amazon/amazontest/server_dependencies_test.go`. Frontend: not affected, as no view, component, state, style, asset or Playwright spec changes. Backend: not affected at runtime, as every production Go file and the non-test files of `internal/amazon/amazontest/` are unchanged and `_test.go` files are not part of the server binary; no validation, persistence, migration or application concurrency change (the guard's `go vet` runs as a child process of the test after `buildModule`'s, on the same module under `t.TempDir()`). `go.mod`, `go.sum`, `buildModule` and the repository's own `go vet ./...` gate are unchanged. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #349 changes the findings guard of one subtest and adds a header block to `internal/amazon/amazontest/server_dependencies_test.go`. Frontend: not affected, as no view, component, state, style, asset or Playwright spec changes. Backend: not affected at runtime, as every production Go file and the non-test files of `internal/amazon/amazontest/` are unchanged and `_test.go` files are not part of the server binary; no validation, persistence, migration or application concurrency change (the guard's `go vet` still runs as a child process of the test after `buildModule`'s, on the same module under `t.TempDir()`; only its output is now read). `go.mod`, `go.sum`, `buildModule`, `vetArgs`, the fixture and the repository's own `go vet ./...` gate are unchanged. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #358 adds one check to one test of `internal/httpapi/amazon_browser_interrupted_test.go`, deletes `internal/httpapi/amazon_browser_cache_log_test.go` and edits the TC-004 entry and row of the issue #240 documents. Frontend: not affected, as no view, component, state, style, asset or Playwright spec changes. Backend: not affected at runtime, as `clearCache`, its log line (FR-AMZ-HCHK-016) and every production Go file are unchanged (edited only temporarily for fault injections) and `_test.go` files are not part of the server binary; no validation, persistence, migration or concurrency change (the capture is read after the check is stored, in the test's goroutine). `go.mod` and `go.sum` are unchanged. API: not affected, as `RefreshItem` is called as before and `API_SPECIFICATION.md` is unchanged.
- Issue #260 adds one check to one test of `internal/service/amazon_human_checks_issue185_test.go` and edits TC-002 of the issue #185 test case document. Frontend: not affected, as no view, component, state, style, asset or Playwright spec changes. Backend: not affected at runtime, as `scheduleUnscheduledAmazonItems`, `nextAmazonCheckAt` and every production Go file are unchanged (edited only temporarily for fault injections) and `_test.go` files are not part of the server binary; no validation, persistence, migration or concurrency change (the test reads the stored times after the start returns). `go.mod` and `go.sum` are unchanged. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #199 changes the wrap check of TC-005 and one field of the in-page `layoutProbe` in `tests/responsive-layout.spec.ts`, and edits TC-005 of the issue #165 test case document. Frontend: not affected, as `src/index.css`, `TrackedItemsPage.tsx` and every view, component, state, style and asset are unchanged (the CSS is edited only temporarily for fault injections); the card layout stays FR-RESPONSIVE-024/TS-RESPONSIVE-017, with no accessibility or UI state change. Backend: not affected, as no Go file changes. API: not affected, as the mocked requests and responses are unchanged and `API_SPECIFICATION.md` is unchanged.
- Issue #359 deletes one test from `internal/amazon/browser_cache_log_test.go`. Frontend: not affected, as no view, component, state, style, asset or Playwright spec changes. Backend: not affected at runtime, as `clearCache`, its log line (FR-AMZ-HCHK-015–016) and every production Go file are unchanged (edited only temporarily for fault injections) and `_test.go` files are not part of the server binary; no validation, persistence, migration or concurrency change, and `go.mod`/`go.sum` are unchanged. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #222 splits one line of one Playwright spec into two. Frontend: not affected, as `src/` (including the Information modal and `AppMenu`) and every view, component, state, style and asset are unchanged; FR-APP-INFO-014 and its accessibility behaviour stay as they are, with no UI state change. Backend: not affected, as no Go file changes. API: not affected, as the mocked requests and responses are unchanged and `API_SPECIFICATION.md` is unchanged.
- Issue #219 removes three checks from, and rewords one failure message of, `internal/amazon/browser_navigation_test.go`. Frontend: not affected, as no view, component, state, style, asset or Playwright spec changes. Backend: not affected at runtime, as `BrowserCollector`, the classification of Amazon answers (FR-AMZ-HCHK-011) and every production Go file are unchanged and `_test.go` files are not part of the server binary; no validation, persistence, migration or concurrency change, and `go.mod`/`go.sum` are unchanged. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #196 adds `internal/amazon/amazontest/browser_profile.go` and its test file, and replaces local helpers in five `_test.go` files. Frontend: not affected, as no view, component, state, style, asset or Playwright spec changes. Backend: not affected at runtime, as every production Go file is unchanged, the helper package is imported only by `_test.go` files (dependency check FR-TEST-HYG-010) and `_test.go` files are not part of the server binary; no validation, persistence, migration or application concurrency change. The only concurrency is in the tests: the browser `Wait` goroutine of `KillBrowserOnProfile`, the recording TB's goroutine, and `profileServer`'s blocked handler, all as on `master`. `go.mod` and `go.sum` are unchanged (standard library only). API: not affected, as the `internal/httpapi` test drives the same handlers and `API_SPECIFICATION.md` is unchanged.
- Issue #387 adds one line to `KillBrowserOnProfile` and one test to `internal/amazon/amazontest/browser_profile_test.go`. Frontend: not affected, as no view, component, state, style, asset or Playwright spec changes. Backend: not affected at runtime, as every production Go file is unchanged and the helper package is imported only by `_test.go` files (dependency check FR-TEST-HYG-010), so the server binary is the same; no validation, persistence, migration or application concurrency change. The only concurrency is in the test: the helper's `Wait` goroutine and the recording TB's goroutine, both as on `master`. `go.mod` and `go.sum` are unchanged. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.
- Issue #218 adds `internal/amazon/amazontest/capture.go` and its test file, and replaces the local capture copies in thirteen `_test.go` files of `internal/amazon`, `internal/service` and `internal/httpapi`. Frontend: not affected, as no view, component, state, style, asset or Playwright spec changes. Backend: not affected at runtime. Every production Go file is unchanged, including what the collector logs and prints. The helper package is imported only by `_test.go` files (dependency check FR-TEST-HYG-010), so the server binary is the same. There is no validation, persistence, migration or application concurrency change. The only concurrency is in the tests: the capture's copy goroutine and its locked buffers, as on `master`. `go.mod` and `go.sum` are unchanged (standard library only). API: not affected, as the `internal/httpapi` tests drive the same handlers and `API_SPECIFICATION.md` is unchanged.
- Issue #384 splits the first check of `assertNavigationFailure` in `internal/amazon/browser_navigation_test.go`. Frontend: not affected, as no view, component, state, style, asset or Playwright spec changes. Backend: not affected at runtime, as `BrowserCollector`, the classification of Amazon answers and every production Go file are unchanged and `_test.go` files are not part of the server binary; no validation, persistence, migration or concurrency change, and `go.mod`/`go.sum` are unchanged. API: not affected, as no handler is involved and `API_SPECIFICATION.md` is unchanged.

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
| TS-TEST-HYG-022 | FR-TEST-HYG-019 | Manual follow-up: the owner runs `gh issue close 238 --reason completed` after the merge; the factory cannot, as no stage has GitHub credentials and its pull request closes only #241. Replaced by TS-TEST-HYG-033 (issue #238). |
| TS-TEST-HYG-023 | FR-TEST-HYG-020 | `internal/httpapi/amazon_browser_interrupted_test.go` is not edited: its `select` on `stopped` with a `default` branch and the exact message stays before `cancel()`. |
| TS-TEST-HYG-024 | FR-TEST-HYG-021 | The only change outside the change folder and the specifications is the deletion of the record; no file outside `doc/`, no other `doc/todo/` file and no test file changes. |
| TS-TEST-HYG-025 | FR-TEST-HYG-022 | `TestAmazonMissedWindow_Issue185_TC001_SkippedRedrawnAndLogged` runs `collectDue` at 20:00 on a missed item due at 07:30 and a control item due at 19:30. It asserts no collector call for the missed item, a next check in the morning window of the next day, and the exact `missed its window` log line naming that time. |
| TS-TEST-HYG-026 | FR-TEST-HYG-023 | `TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce` sets `human_checks_redraw = 1` with its own SQL connection on a store holding three Amazon items at 08:00 on day D+1. It starts the service at 10:00 on day D and asserts evening-window times and the mark at 0, then restarts with a different seed and asserts that the times are unchanged. |
| TS-TEST-HYG-027 | FR-TEST-HYG-024 | `TestBrowserEmptyErrorStatus_Issue185_TC003_ClassifiedByStatus` runs one subtest per status (404, 410, 503, 429), each against an empty `/dp/B0PRICE001` answer from a new `errorPageServer`. It asserts `Result`, `Message`, exact `HTTPStatus`, and for 503 and 429 also `SecondHandStatus`, the exact `FailureCause` and the captured output. |
| TS-TEST-HYG-028 | FR-TEST-HYG-025 | The tests stage runs `gofmt -l internal/amazon` and `gofmt -w` on any listed test file before the files are locked. `browser_test.go` is already formatted on `master` and stays unchanged unless `gofmt` lists it. |
| TS-TEST-HYG-029 | FR-TEST-HYG-026 | The tests are ordinary tests (no build tag, flag or `t.Parallel`). The browser test skips only through `amazontest.Browser` when no Chromium is found, which does not happen in the gate. The diff outside `doc/` is the two new `_test.go` files. |
| TS-TEST-HYG-030 | FR-TEST-HYG-022–024 (proof of failure) | The tests stage writes `no-tests.md` (the red phase cannot run on existing behaviour) and runs the three mutation checks, reverted before the stage ends, recording their failures there. |
| TS-TEST-HYG-031 | FR-TEST-HYG-027 | TC-005 keeps, after the `flushed` wait and before the first `cancel()`, the non-blocking `select` on `stopped` (closed only after `Collect` returns) that calls `t.Fatal("the check ended before the server stop: the interruption point was missed")`. Proved by the reverted `cancel(); <-stopped` mutation. |
| TS-TEST-HYG-032 | FR-TEST-HYG-028 | The guard's `default` branch goes on to the unchanged `cancel()`, 20 s wait and assertions; TC-005 passes on repeated runs. |
| TS-TEST-HYG-033 | FR-TEST-HYG-019, FR-TEST-HYG-029 | The factory's pull request body `Closes #238` closes the issue as completed at the merge; no manual close. |
| TS-TEST-HYG-034 | FR-TEST-HYG-029 | No file outside `doc/` changes; `internal/httpapi/amazon_browser_interrupted_test.go` is not edited, and the mutation is never committed. |
| TS-TEST-HYG-035 | FR-TEST-HYG-030 | The technical specification stage replaces `HTTP 404` with `HTTP 403` in the Given line of TC-006 in the issue #168 `testcases.md`; no other line of the entry changes, so it holds no `404` or `410` and its three cases match the three sub-tests of `TestBrowserCollector_TC006_NonSuccessAndRobotCheckAreFailures`. |
| TS-TEST-HYG-036 | FR-TEST-HYG-031 | The same stage appends the corner case `product page answered with HTTP 404 or HTTP 410 gives "listing unavailable" (an Amazon answer, not a failure)` to the Corner cases line of TC-004, after the three existing ones, unreworded. |
| TS-TEST-HYG-037 | FR-TEST-HYG-032 | Only those two lines of the issue #168 folder change; no file outside `doc/` and nothing under `doc/todo/` changes; the tests stage writes `no-tests.md` and the development stage writes no code, so `go vet ./...` and `go test ./...` run the same code as on `master`. |
| TS-TEST-HYG-038 | FR-TEST-HYG-033 | In the TC-032 subtest "Refresh price running", `database.SetNextCheck(ctx, id, now.Now().Add(-time.Minute))` (07:04) runs after `waitStarted` and before `collectDue`. The check count of 1 and `assertScheduled` (a window, after 07:05), read after `waitIdle`, are unchanged. |
| TS-TEST-HYG-039 | FR-TEST-HYG-033 (proof) | Mutation 1 (`return true` for an in-flight Amazon item before `SetNextCheck` in `skipScheduledCheck`) makes the changed subtest fail with `want a later window`, and the `master` subtest pass. Recorded in `no-tests.md`, then reverted. |
| TS-TEST-HYG-040 | FR-TEST-HYG-034 | The "first check after add running" subtest, its name, seed 33 and `assertScheduled` are not edited. |
| TS-TEST-HYG-041 | FR-TEST-HYG-035 | `TestAmazonRestartAfterStop_Issue182_TC004_NoScheduledCheckAfterRefreshPrice` covers the scenario: the item is due at 07:00 and stopped at 07:00; the 07:00 run makes no check; the Refresh price at 07:30 restarts the item; the two 07:31 runs give one check in total and a next check in the evening window of 2030-03-05. |
| TS-TEST-HYG-042 | FR-TEST-HYG-035 (proof) | Mutation 2 (`rescheduleOverdueAmazonCheck` not called by `RefreshItem`) fails the test on the call count. Mutation 3 (no `ClearAmazonStop` in `amazonGate.do`) fails it on the stop state. Recorded in `no-tests.md`, then reverted. |
| TS-TEST-HYG-043 | FR-TEST-HYG-036 | Ordinary tests (no build tag, flag, skip or `t.Parallel`) run by `go test ./...`. `gofmt -l internal/service` is empty. The diff outside `doc/` is `M internal/service/amazon_human_checks_test.go` and `A internal/service/amazon_human_checks_issue182_test.go`, and no production file changes. |
| TS-TEST-HYG-044 | FR-TEST-HYG-037 | Line 361 of `tests/platform-tabs.spec.ts`, `expect(imageRequests.every((url) => url === imageUrl)).toBe(true)`, is removed, not commented out; no other assertion of TC-010 compares the recorded URLs with `imageUrl`. |
| TS-TEST-HYG-045 | FR-TEST-HYG-038 | In its place, `await expect.poll(() => imageRequests.length, { message: "the accepted Amazon image was requested" }).toBeGreaterThan(0)`. The route handler, which records only `imageUrl` and puts any other image into `api.unexpected`, and `expect(api.unexpected).toEqual([])` are kept. |
| TS-TEST-HYG-046 | FR-TEST-HYG-038 (proof) | The tests stage writes `no-tests.md` (the change is test code, so the red phase cannot fail) and runs the three reverted fault injections (no push, no thumbnail, another Amazon image), recording each failure there. |
| TS-TEST-HYG-047 | FR-TEST-HYG-039 | The diff outside `doc/` is `M tests/platform-tabs.spec.ts`, one line inside TC-010. No production, Go, frontend, dependency or `API_SPECIFICATION.md` change. The gates run the same test list as on `master`. |
| TS-TEST-HYG-048 | FR-TEST-HYG-040 | `assertBrowserRefusesProfile(t, executable, profile, host, server, path)` fails unless `fixtureCheckFailure` returns `""`, which needs both `!run.requested` and `strings.Contains(run.output, lockRefusal(host))`. `runBrowserOnProfile` reads both only after the group kill and `<-exited`, whether the process exited by itself or was killed. |
| TS-TEST-HYG-049 | FR-TEST-HYG-040, FR-TEST-HYG-041 | `lockRefusal(host)` is `on another computer (<host>)`, the host-naming part of Chromium's `IDS_PROFILE_IN_USE_POSIX` line; case-sensitive, so `ProcessSingleton` alone or the generic lock-error line do not match. Replaced only by the recorded real line under the rule of the issue #195 design. |
| TS-TEST-HYG-050 | FR-TEST-HYG-041 | The failure is `fixture check: <failed conditions joined by "; ">; browser output (<n> bytes):\n<output>`, naming `the browser opened the page on a profile locked by another computer` and/or `the browser output does not hold the lock refusal message "<lockRefusal(host)>"`. |
| TS-TEST-HYG-051 | FR-TEST-HYG-041 (proof) | `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal` runs `runBrowserOnProfile` (2 s wait) and `fixtureCheckFailure` on ten stub runs of the package test binary (`refusalStubBrowser`, `PRICEFOLLOWER_REFUSAL_STUB`, `TestBrowserProfile_Issue195_RefusalStubProcess`), asserting the result, `requested` and that the process group is gone (`ESRCH`). |
| TS-TEST-HYG-052 | FR-TEST-HYG-042 | TC-002 changes only at its call (new `"otherhost"` argument); the real browser keeps the 15 s wait, the arguments, `Setpgid`, the group kill and the wait for `Wait`, so no browser process of the profile is left. Proved by TC-002 runs with `-count=1` and `-count=3` (tests stage, recorded in `no-tests.md`) and by the gate. |
| TS-TEST-HYG-053 | FR-TEST-HYG-041 (proof) | Reverted fault injection: without the first `writeStaleLock` of TC-002, TC-002 fails at the fixture check with the opened-page condition and the output. Recorded in `no-tests.md`. |
| TS-TEST-HYG-054 | FR-TEST-HYG-040–042 (delivery) | The tests stage delivers all the code and writes `no-tests.md` (the change is locked test code, so the red phase cannot run); the development stage writes no code and sets the statuses. |
| TS-TEST-HYG-055 | FR-TEST-HYG-043 | The diff outside `doc/` is `M internal/amazon/browser_profile_test.go`; `fake_browser_test.go`, its `TestMain` and every production file are unchanged. Ordinary tests (no build tag, flag, skip or `t.Parallel`) run by `go test ./...`; `gofmt -l internal/amazon` is empty; `go vet ./...` passes. New `#nosec` comments follow the file's style. |
| TS-TEST-HYG-056 | FR-TEST-HYG-044 | In the `t.Run` loop of `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal`, `wait := 15 * time.Second`, passed to `runBrowserOnProfile`, for every row whose `row.stub.Hold` is false; the choice reads only `Hold`. |
| TS-TEST-HYG-057 | FR-TEST-HYG-045 | `wait = 2 * time.Second` when `row.stub.Hold` is true; no output signal is added to the hold stubs. Replaced by TS-TEST-HYG-062 (issue #315). |
| TS-TEST-HYG-058 | FR-TEST-HYG-046 | `runBrowserOnProfile`, the rows, the stubs and the assertions are unchanged, so a stub that exits still ends its row at once, and its group is killed and waited for (`ESRCH` asserted). Proved by runs 1 and 4 of the issue #312 design. |
| TS-TEST-HYG-059 | FR-TEST-HYG-044 (proof) | Reverted fault injections 2 and 3 (5 s and 3 s delay of the non-hold stubs, with a 15 s and with a 2 s wait), recorded in `no-tests.md`. |
| TS-TEST-HYG-060 | FR-TEST-HYG-047 | The diff outside `doc/` is `M internal/amazon/browser_profile_test.go`, limited to the wait lines of the `t.Run` loop; `assertBrowserRefusesProfile` and its 15 s wait are unchanged. The tests stage delivers the change and writes `no-tests.md`; the development stage writes no code. `gofmt -l internal/amazon` is empty; `go vet ./...` and `go test ./...` pass. |
| TS-TEST-HYG-061 | FR-TEST-HYG-048 | `refusalStub.Signal` is set by `refusalStubBrowser` to `output-written` in the row's own `t.TempDir()` and returned with the executable. A `Hold` stub creates it (`os.WriteFile`, empty, `0o600`) after its page request and `Stderr` write, then sleeps until killed. |
| TS-TEST-HYG-062 | FR-TEST-HYG-048 | A row with `row.stub.Hold` runs through `runBrowserOnProfileUntil` with a wait of `waitForOutputSignal(signal, exited, 15*time.Second)` (polled every 10 ms, counted from the call right after `Start`), then 2 s or the exit. The choice reads only `Hold`. |
| TS-TEST-HYG-063 | FR-TEST-HYG-048, FR-TEST-HYG-052 | `runBrowserOnProfileUntil` holds the body of `runBrowserOnProfile` with the `select` replaced by `wait(exited)`; `runBrowserOnProfile` keeps its signature and calls it with the same `select`, so `assertBrowserRefusesProfile` and the non-hold rows behave as before. |
| TS-TEST-HYG-064 | FR-TEST-HYG-049 | The rows table and `row.stub` are unchanged (the signal is set on `refusalStubBrowser`'s copy). The signal is a file, so the captured output is only `Stderr`. The hold stubs make no request, and the output is read after the group kill and `<-exited`. |
| TS-TEST-HYG-065 | FR-TEST-HYG-050 | Without the marker at 15 s, or when the stub exits without it, `waitForOutputSignal` returns `false`, the 2 s are skipped and the group is killed. The row reports `t.Errorf("the stub did not signal its output within 15 s")`, and the `ESRCH` check still runs. |
| TS-TEST-HYG-066 | FR-TEST-HYG-051 | Normal runs (run 1 of the issue #315 design): ten rows pass; hold rows take 2–3 s, non-hold rows under 2 s; `ESRCH` after every row; with `-count=3`, `-race` and `-run 'TestBrowserProfile'`. |
| TS-TEST-HYG-067 | FR-TEST-HYG-052 | The diff outside `doc/` is `M internal/amazon/browser_profile_test.go`. Its hunks: `runBrowserOnProfile` split into the wrapper and `runBrowserOnProfileUntil`, `refusalStub`, `refusalStubBrowser`, `waitForOutputSignal`, the `Hold` branch of `TestBrowserProfile_Issue195_RefusalStubProcess`, and the run lines of the `t.Run` loop. `assertBrowserRefusesProfile`, `fixtureCheckFailure`, the rows and TC-002 of issue #180 are not edited. The tests stage delivers the change and writes `no-tests.md`; the development stage writes no code. `gofmt -l internal/amazon` is empty; `go vet ./...` and `go test ./...` pass. |
| TS-TEST-HYG-068 | FR-TEST-HYG-048–050 (proof) | Reverted fault injections 2–6 of the issue #315 design (slow hold stub with the new wait and with the `master` wait; 20 s delay; exit without signal; output without signal), recorded in `no-tests.md`. |
| TS-TEST-HYG-069 | FR-TEST-HYG-053 | Right after `profileOf("")` and before any check, the subtest calls `t.Cleanup` on its own `t` with a function that runs `os.RemoveAll(path)`, so the temporary profile is removed when the subtest ends, whether it passed, failed or stopped on `t.Fatal`/`t.Fatalf`, whatever the profile holds. |
| TS-TEST-HYG-070 | FR-TEST-HYG-054 | The removal runs only after the subtest function has returned, so after the explicit `cleanup()` and its `os.Stat` check, which are kept with the message `the cleanup did not remove the temporary profile: %v`. The other checks and messages are unchanged. |
| TS-TEST-HYG-071 | FR-TEST-HYG-055 | The removal removes `path` only when the parent of `path` is `filepath.Clean(os.TempDir())` and its base name starts with `pricefollower-browser-` (the subtest's location check), and otherwise removes nothing; it never removes the parent of `path`, a glob or another path. A removal error is reported with `t.Errorf("remove the temporary profile: %v", err)`; a missing path is not an error. |
| TS-TEST-HYG-072 | FR-TEST-HYG-056 | The diff outside `doc/` is `M internal/amazon/browser_profile_values_test.go`, limited to the `t.Cleanup` block in the subtest; no import, helper, other subtest or production file changes. The tests stage delivers the change and writes `no-tests.md`; the development stage writes no code. `gofmt -l internal/amazon` is empty; `go vet ./...` and `go test ./...` pass. |
| TS-TEST-HYG-073 | FR-TEST-HYG-053–055 (proof) | Runs and reverted fault injections 1–8 of the issue #324 design (fatal stop with a non-empty profile, with and without the removal; nil cleanup; error; regular file; cleanup that does nothing; empty path), each run with a fresh `TMPDIR`, recorded in `no-tests.md`. |
| TS-TEST-HYG-074 | FR-TEST-HYG-057 | In the 1025-space token test, the `Saving settings…` alternative of the line 101 regular expression, the `for (const put of api.puts)` loop (lines 102–105) and the `JSON.stringify(api.puts)` check (line 106) are removed, not commented out; no `Saving settings…` text is left in the test. |
| TS-TEST-HYG-075 | FR-TEST-HYG-058 | `expect(api.puts).toEqual([])` runs after `await expect(dialog).toBeHidden()` and before the second `openSettings(page)`. A non-retrying `expect` is enough: the modal closes either without a request or after the answer to its request, and `mockApi` records the body before fulfilling it. |
| TS-TEST-HYG-076 | FR-TEST-HYG-058 | `await expect(page.getByText(/too long/i)).toHaveCount(0)` is kept right after `toBeHidden()`, on the whole page and case-insensitive. |
| TS-TEST-HYG-077 | FR-TEST-HYG-059 | The setup (saved token `wxyz` with `lastRejectedAt` `2026-10-03T20:00:00Z`), the 1025-space fill, the Save click, `toBeHidden()` and the three reopening checks are unchanged. Proved by run 1 of the issue #290 design. |
| TS-TEST-HYG-078 | FR-TEST-HYG-058 (proof) | Reverted fault injections 2 (three `save()` edits: no `trim()`, removal for a blank input, no early return) and 3 (a "too long" text in three casings added after the modal has closed) of the issue #290 design, each failing at its assertion, recorded in `no-tests.md`. |
| TS-TEST-HYG-079 | FR-TEST-HYG-060 | The diff outside `doc/` is `M tests/claude-token-settings.spec.ts`, limited to lines 101–106 of the 1025-space token test; `mockApi`, `openSettings`, the other tests, `src/`, Go code, `API_SPECIFICATION.md`, `package.json` and the lock file are unchanged. The tests stage delivers the change and writes `no-tests.md`; the development stage writes no code. The e2e suite, `go vet ./...`, `go test ./...` and `node --test scripts/*.test.mjs` pass. |
| TS-TEST-HYG-080 | FR-TEST-HYG-061 | `TestBrowserProfile_Issue197_TC005_ProfileUsesTheExplicitForm`, its two-line comment and the blank line before it are deleted from `internal/amazon/browser_profile_values_test.go`; no test is added, skipped, renamed or moved, and no test under `internal/amazon/` parses or reads `browser.go`. |
| TS-TEST-HYG-081 | FR-TEST-HYG-062 | The imports `go/ast`, `go/format`, `go/parser` and `go/token` are deleted; `errors`, `io/fs`, `os`, `path/filepath`, `strings` and `testing` stay, as the behaviour tests use them. No blank import or dummy use is added; the deleted test had no package-level declaration to remove. |
| TS-TEST-HYG-082 | FR-TEST-HYG-063 | The header comment, helpers and the TC-001 to TC-004 behaviour tests of issue #197 are byte-for-byte unchanged, and so are `browser.go` and the other test files. Proved by run 4 of the issue #322 design. |
| TS-TEST-HYG-083 | FR-TEST-HYG-063 (proof) | Reverted fault injections of the issue #322 design: the log rewording fails the `master` test (run 1); the log rewording, the `directory` rename and the extracted persistent-profile helper pass after the deletion (run 2); a result-changing edit still fails the behaviour tests (run 3). Recorded in `no-tests.md`. |
| TS-TEST-HYG-084 | FR-TEST-HYG-064 | The diff outside `doc/` is `M internal/amazon/browser_profile_values_test.go` with removed lines only. The tests stage delivers the change and writes `no-tests.md`; the development stage writes no code. `gofmt -l internal/amazon` is empty; `go vet ./...` and `go test ./...` pass. |
| TS-TEST-HYG-085 | FR-TEST-HYG-065 | `buildModule` runs `go build ./...`, then `go vet ./...`, offline with `fixtureEnv`, and returns `nil` only when both succeed; vet type-checks every fixture package with its `_test.go` files, so row d's `cmd/app/main_test.go` and `internal/service/service_test.go` are compiled. |
| TS-TEST-HYG-086 | FR-TEST-HYG-066 | The first failing command gives `go <args>: <err>: <output>`. The new subtest `control: a test file uses a name the stub does not declare` (`plainMain` plus a `cmd/app/main_test.go` using `amazontest.Nope`, not in `fixtureCases`) requires an error that holds `amazontest.Nope` and starts with `go vet ./...`. |
| TS-TEST-HYG-087 | FR-TEST-HYG-067 | `fixtureCases`, the loop, `TC-003 h fixture: plainMain`, the `DownloadMarker` control and every failure message of the test are unchanged; rows f and g still fail at `go build ./...`. |
| TS-TEST-HYG-088 | FR-TEST-HYG-068 | The comments above `buildModule` and the test, and a new issue #254 block in the header comment, say that the fixtures' test files are compiled (`go vet ./...`); none says the check only runs `go build ./...` or that test files are out of scope. |
| TS-TEST-HYG-089 | FR-TEST-HYG-069 | The diff outside `doc/` is `M internal/amazon/amazontest/server_dependencies_test.go`, limited to `buildModule`, the test's comment and new control, and the header comment; the dependency check, its helpers and tests are not edited. `gofmt -l internal/amazon` is empty; `go vet ./...` and `go test ./...` pass. |
| TS-TEST-HYG-090 | FR-TEST-HYG-065–066 (delivery and proof) | The tests stage delivers the change and writes `no-tests.md` with runs and fault injections 1–5 of the issue #254 design (row d test-file edits fail the new check and pass the `master` one; `buildModule` without vet fails the new control); the development stage writes no code. |
| TS-TEST-HYG-091 | FR-TEST-HYG-070 | In the `default` branch of the `fixtureCases` loop of `TestFixtureModulesBuild_Issue251_TC001` (rows f and g), a non-nil `buildModule` error passes only when `strings.Contains(err.Error(), tc.fails)`; the text is the row's own `fails` value, the one the listing test requires. |
| TS-TEST-HYG-092 | FR-TEST-HYG-071 | A mismatch reports `t.Errorf("build failure %q does not hold %q", err, tc.fails)`; a nil error keeps `t.Error("invalid fixture builds, want a build failure")`. |
| TS-TEST-HYG-093 | FR-TEST-HYG-072 | `buildModule`, `fixtureCases`, the valid-row branch, `TC-003 h fixture: plainMain`, both controls, their messages and the dependency check, its helpers and tests are not edited; no subtest is added or removed. Only the comment above the test and a new issue #253 block in the header comment change besides the branch. |
| TS-TEST-HYG-094 | FR-TEST-HYG-073 | The diff outside `doc/` is `M internal/amazon/amazontest/server_dependencies_test.go`, limited to the branch and the two comments. `gofmt -l internal/amazon` is empty; `go vet ./...`, `go test ./...` and `go test -count=1 ./internal/amazon/amazontest/...` pass. |
| TS-TEST-HYG-095 | FR-TEST-HYG-070–071 (delivery and proof) | The tests stage delivers the change and writes `no-tests.md` with runs and fault injections 1–5 of the issue #253 design (recorded `go build` outputs of rows f and g; `go nosuchcmd`, wrong `fails` texts and a building row f fail the new check; `go nosuchcmd` passes the `master` one); the development stage writes no code. |
| TS-TEST-HYG-096 | FR-TEST-HYG-074 | `buildModule` runs `go build ./...`, then `vetArgs` = `go vet -atomic ./...`, offline with `fixtureEnv`; naming the `atomic` analyzer turns vet's other analyzers off, while vet still type-checks every package with its `_test.go` files and fails on any compile error. Replaces the `go vet ./...` of TS-TEST-HYG-085. |
| TS-TEST-HYG-097 | FR-TEST-HYG-074 | New subtest `analyzer findings in non-test and test files do not fail the check` (not in `fixtureCases`): a `cmd/app/main.go` with printf and copylocks findings and a `cmd/app/main_test.go` with printf and tests findings, both compiling; `buildModule` must return `nil`, else `valid fixture does not build: <err>`. |
| TS-TEST-HYG-098 | FR-TEST-HYG-075 | The test-file control keeps its name, fixture, `t.Fatal` and `amazontest.Nope` check; its prefix check uses `"go " + strings.Join(vetArgs, " ") + ":"`, so the failure must come from the vet step. Replaces the `go vet ./...` prefix of TS-TEST-HYG-086. `fixtureCases`, the loop and its messages, `TC-003 h fixture: plainMain` and the `DownloadMarker` control are unchanged. |
| TS-TEST-HYG-099 | FR-TEST-HYG-076 | The comments above `buildModule` and the test, the issue #254 header line and a new issue #340 header block say that `go vet -atomic ./...` type-checks with test files and that its analyzer findings do not fail the check, and why one analyzer is named. |
| TS-TEST-HYG-100 | FR-TEST-HYG-077 | The diff outside `doc/` is `M internal/amazon/amazontest/server_dependencies_test.go`, limited to `vetArgs`, `buildModule`, the control's prefix check, the new subtest and the three comments. `gofmt -l internal/amazon` is empty; `go vet ./...`, `go test ./...` and `go test -count=1 ./internal/amazon/amazontest/...` pass. |
| TS-TEST-HYG-101 | FR-TEST-HYG-074–075 (delivery and proof) | The tests stage delivers the change and writes `no-tests.md` with runs and fault injections 1–6 of the issue #340 design (recorded vet outputs; analyzer findings in rows a and d pass and fail on `master`; compile errors in rows a and d fail; `master`'s vet fails the new subtest; `buildModule` without vet fails the control); the development stage writes no code. |
| TS-TEST-HYG-102 | FR-TEST-HYG-078 | In the `t.Run` function of `TestBrowserInterrupted_TC003_UnconfirmedClearIsBoundedAndLogged`, `if test.clear == "never" && !strings.Contains(lines[0], "no answer within")` fails with `t.Fatalf("cache log line without \"no answer within\", the cause of a clear that never answers: %s", lines[0])`; the subtests are selected by their `clear` value, not by a text shared by every subtest. |
| TS-TEST-HYG-103 | FR-TEST-HYG-079 | After it, every subtest calls the unchanged `assertNoInterruptionGivenForCache(t, logged)`, which fails with the offending line on any line holding `cache` (any case) and `context canceled` or `context deadline exceeded`. The existing `context canceled` check on `lines[0]` and its message are kept. |
| TS-TEST-HYG-104 | FR-TEST-HYG-080 | The technical specification stage edits only the third "Then" result of the TC-003 entry of the issue #233 `testcases.md`: `no answer within` for **no answer** and **no answer, page still requested**, neither `context canceled` nor `context deadline exceeded` in every subtest, `cache clear refused by fake` kept for **error answer**. No other line or file of that folder, including `tests.lock`, changes. |
| TS-TEST-HYG-105 | FR-TEST-HYG-081 | The diff outside `doc/` is `M internal/amazon/browser_interrupted_test.go`, limited to the comment and the two checks of TS-TEST-HYG-102 and TS-TEST-HYG-103; the subtest table, the other assertions, the helpers, the other tests and `browser.go` are not edited. `gofmt -l internal/amazon` is empty; `go vet ./...` and `go test ./...` pass. |
| TS-TEST-HYG-106 | FR-TEST-HYG-078–079 (delivery and proof) | The tests stage delivers the change and writes `no-tests.md` with runs and fault injections 1–6 of the issue #239 design (deadline mapping removed: the new test fails and the `master` test passes, each new check failing alone; reworded cause; context causes appended to the "error answer" line); the development stage writes no code. |
| TS-TEST-HYG-107 | FR-TEST-HYG-082 | After its unchanged `buildModule` check, the subtest `analyzer findings in non-test and test files do not fail the check` runs `exec.Command("go", "vet", "./...")` (default analyzers, not `vetArgs`) with `cmd.Dir = dir` (the folder passed to `buildModule`) and `cmd.Env = append(os.Environ(), fixtureEnv...)`; when `cmd.Run()` returns `nil` it reports `t.Error("the fixture has no analyzer finding, so the subtest proves nothing")`. Analyzer messages are not matched. Issue #349 reads the command's output with `CombinedOutput` instead of `Run` and adds a check per file (TS-TEST-HYG-113, TS-TEST-HYG-114). |
| TS-TEST-HYG-108 | FR-TEST-HYG-083 | The `buildModule` check keeps `t.Errorf` and the guard uses `t.Error`; there is no `t.Fatal` or `return` between or in them, so each is reported whatever the other's result. |
| TS-TEST-HYG-109 | FR-TEST-HYG-082 | A comment above the guard says the fixture must have analyzer findings and why the guard does not use `vetArgs`; a new issue #344 header block describes the guard and lists TC-002, TC-003, TC-005 and TC-006 as proved outside the file. |
| TS-TEST-HYG-110 | FR-TEST-HYG-084 | The subtest's name, fixture and `buildModule` check, `buildModule`, `vetArgs`, `writeModule`, `fixtureEnv`, `fixtureCases`, the other subtests and their messages, the comment above the test and the dependency check and its tests are not edited; no subtest is added or removed. |
| TS-TEST-HYG-111 | FR-TEST-HYG-085 | The diff outside `doc/` is `M internal/amazon/amazontest/server_dependencies_test.go`, limited to the guard, its comment and the header block. `gofmt -l internal/amazon` is empty; `go vet ./...`, `go test ./...` and `go test -count=1 ./internal/amazon/amazontest/...` pass. |
| TS-TEST-HYG-112 | FR-TEST-HYG-082–084 (delivery and proof) | The tests stage delivers the change and writes `no-tests.md` with runs and fault injections 1–5 of the issue #344 design (recorded full vet output on the fixture; a finding-free fixture fails the new subtest with the guard message and passes the `master` one; `vetArgs` and fixture edits show independent reports; `master` and change runs compared); the development stage writes no code. |
| TS-TEST-HYG-113 | FR-TEST-HYG-086 | In the findings guard of the subtest `analyzer findings in non-test and test files do not fail the check`, the unchanged full vet run command is executed with `out, err := cmd.CombinedOutput()`; then, for each `file` of `"cmd/app/main.go"` and `"cmd/app/main_test.go"`, `!strings.Contains(string(out), file+":")` reports `t.Errorf("the fixture has no analyzer finding in %s, so the subtest proves nothing", file)`. Only the two file positions are matched, never an analyzer message. |
| TS-TEST-HYG-114 | FR-TEST-HYG-087 | `err == nil` keeps reporting `t.Error("the fixture has no analyzer finding, so the subtest proves nothing")`. The `buildModule` check, the module check and both file checks run in every run: no `t.Fatal`, `return` or condition on another check's result between or in them; the file checks also run when vet succeeds (empty output, both file messages added). |
| TS-TEST-HYG-115 | FR-TEST-HYG-088 | The comment above the guard (issue #344 and issue #349) names `cmd/app/main.go` and `cmd/app/main_test.go`, says the subtest proves that the fixture build check ignores findings in non-test files and in test files only if each has a finding, and why `vetArgs` is not used and messages are not matched. A new issue #349 header block, after the unchanged issue #344 block, describes the per-file requirement and lists TC-002, TC-003, TC-004, TC-005, TC-007 and TC-008 as proved outside the file. |
| TS-TEST-HYG-116 | FR-TEST-HYG-089 | The subtest's name, fixture, `buildModule` check and guard command, `buildModule`, `vetArgs`, `writeModule`, `fixtureEnv`, `fixtureCases`, `plainMain`, `helperMain`, the stub, the other subtests and their messages, the comment above the test, the other header blocks and the dependency check and its tests are not edited; no subtest is added or removed. The diff outside `doc/` is `M internal/amazon/amazontest/server_dependencies_test.go`, limited to the guard, its comment and the header block. `gofmt -l internal/amazon` is empty; `go vet ./...`, `go test ./...` and `go test -count=1 ./internal/amazon/amazontest/...` pass. |
| TS-TEST-HYG-117 | FR-TEST-HYG-086–087 (delivery and proof) | The tests stage delivers the change and writes `no-tests.md` with runs and fault injections 1–7 of the issue #349 design (recorded full vet output with both positions; findings removed from one file fail the new subtest naming that file, and from main_test.go pass the `master` one; findings removed from both give the module message; `vetArgs` and fixture edits show independent reports; `master` and change runs compared); the development stage writes no code. |
| TS-TEST-HYG-118 | FR-TEST-HYG-090 | After the first start's loop of `TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce`, a loop over every pair of `ids` fails with `t.Fatalf("items %s and %s both have the next check %s after the start, want a time drawn for each item", …)` when `drawn[id].Equal(drawn[other])`. |
| TS-TEST-HYG-119 | FR-TEST-HYG-091 | The test's name, comment, seeds, clocks, item IDs, `former`, other checks and messages, the file's other test, helpers and header comment are unchanged; with seed 1851 the three times differ, so the test passes. |
| TS-TEST-HYG-120 | FR-TEST-HYG-092 | The technical specification stage adds one Then line (pairwise different times) and one corner case (one time for every item fails the test) to TC-002 of the issue #185 `testcases.md`; nothing else in that folder changes. |
| TS-TEST-HYG-121 | FR-TEST-HYG-093 | The diff outside `doc/` is `M internal/service/amazon_human_checks_issue185_test.go`, limited to the added block; in the issue #185 folder only `testcases.md` changes. `gofmt -l internal/service` is empty; `go vet ./...` and `go test ./...` pass. |
| TS-TEST-HYG-122 | FR-TEST-HYG-090–091 (delivery and proof) | The tests stage delivers the block and writes `no-tests.md` with runs and fault injections 1–5 of the issue #260 design (one shared time fails the new check and passes `master`'s test; each shared pair fails naming it); the development stage writes no code. |
| TS-TEST-HYG-123 | FR-TEST-HYG-094 | For each value of a card that does not follow on the line of the value before it, the wrap check of TC-005 sums the widths of every value of the previous line (found by walking back with `followsOnLine`), the first cell (the item value) at `ITEM_BASIS_REM * rem` (constant 14, root font size read in the page) and the others at their rendered `width`, and fails unless `card.right - card.left - used < current.width + 1`; `card.left` is the content-box left that `layoutProbe.cells` now returns. |
| TS-TEST-HYG-124 | FR-TEST-HYG-095 | The failure message is `${width}px ${tab} card ${index}: free space <free> before value <label or position> (width <width>)`, so it names the width, tab, card, value, its width and the free space. |
| TS-TEST-HYG-125 | FR-TEST-HYG-096 | TC-005's name, widths, tabs, data, row count, same-line and label checks and messages, the shared helpers and the other tests are unchanged; the probe's new `left` field is read only by TC-005. TC-005 passes on the unchanged application. |
| TS-TEST-HYG-126 | FR-TEST-HYG-097 | The technical specification stage rewrites only the Then line of TC-005 about a value starting a later line in the issue #165 `testcases.md`: the free space counts the item value at its base width of 14rem, not at its grown width; nothing else in that folder changes. |
| TS-TEST-HYG-127 | FR-TEST-HYG-098 | The diff outside `doc/` is `M tests/responsive-layout.spec.ts`, limited to `ITEM_BASIS_REM`, the `cells` type and `left` field, the `rem` read and the wrap check; in the issue #165 folder only `testcases.md` changes. The e2e suite, `go vet ./...`, `go test ./...` and `node --test scripts/*.test.mjs` pass. |
| TS-TEST-HYG-128 | FR-TEST-HYG-094–095 (delivery and proof) | The tests stage delivers the change and writes `no-tests.md` with runs and fault injections 1–5 of the issue #199 design (item basis 40rem and 100% fail the new check naming the wrapped value; with 40rem the `master` wrap check passes every 775 px card, with 100% the whole `master` test passes); the development stage writes no code. |
| TS-TEST-HYG-129 | FR-TEST-HYG-099 | Lines 168–212 of `internal/amazon/browser_cache_log_test.go` (blank line, the two comment lines and `TestBrowserCacheLog_TC003_SuccessfulClearLogsNothing`) are deleted; no `// TC-003` comment and no `Clear: "answer"` script remain in the file, and no test of the repository has that name. |
| TS-TEST-HYG-130 | FR-TEST-HYG-100 | The imports `net/http`, `regexp`, `strings`, `testing`, `time` and every declaration of the file stay, as TC-001 and TC-002 use them; no blank import or dummy use is added. TC-001, TC-002, the helpers and the header comment are byte-for-byte unchanged, and `go test -run '^TestBrowserCacheLog_'` runs the same subtests as on `master` and passes. |
| TS-TEST-HYG-131 | FR-TEST-HYG-101 | `TestBrowserInterrupted_TC001_ClearConfirmedBeforeClose`, `TestBrowserInterrupted_TC002_PageStillRequestedIsCleared` and `TestBrowserInterrupted_TC006_UninterruptedCheckIsUnchanged/download` are unchanged and cover each deleted subtest through `assertNoCacheLogLine` (match on `cacheLogLine`, which also rejects the old wording). |
| TS-TEST-HYG-132 | FR-TEST-HYG-101 | The issue #240 change folder and `doc/specifications/amazon-human-checks/` are not edited; they already assign TC-003 to the tests of TS-TEST-HYG-131 and `browser_cache_log_test.go` to TC-001 and TC-002 only. |
| TS-TEST-HYG-133 | FR-TEST-HYG-102 | The diff outside `doc/` is `M internal/amazon/browser_cache_log_test.go` with removed lines only. `gofmt -l internal/amazon` is empty; `go vet ./...` and `go test ./...` pass. |
| TS-TEST-HYG-134 | FR-TEST-HYG-099–101 (delivery and proof) | The tests stage makes the deletion and writes `no-tests.md` with runs and fault injections 1–6 of the issue #359 design (before/after subtest lists and times; a cache line logged on a successful clear, with the neutral and with the old wording, fails the success-clear tests); the development stage writes no code. |
| TS-TEST-HYG-135 | FR-TEST-HYG-103 | In the per-owner loop of `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile`, `refreshOutput := captureCheckOutput(t)` before `RefreshItem`; after `waitStoredCheck`, `linesWith(refreshOutput(), "Amazon browser cache could not be cleared")` non-empty fails with `t.Fatalf("cache clear failure logged during Refresh price of %s:\n%s", id, strings.Join(lines, "\n"))`. One prefix covers both wordings. |
| TS-TEST-HYG-136 | FR-TEST-HYG-104 | Only the capture line, a comment and the check are added to the loop; the test name, comments, IDs, owners, fixture answers, server-stop sequence and guards, server-stop cache check, no-file, list and price checks and their messages are unchanged. The server-stop capture ends before the loop, so captures do not nest. |
| TS-TEST-HYG-137 | FR-TEST-HYG-105 | The tests stage lists `internal/httpapi/amazon_browser_cache_log_test.go` in `delete.txt`; `assertNoCacheLine` goes with it, and the helpers it called stay in their own files. `go test -list . ./internal/httpapi/` loses only `TestAmazonBrowserCacheLog_TC004_SuccessfulClearThroughTheApplicationLogsNothing`. |
| TS-TEST-HYG-138 | FR-TEST-HYG-106 | The technical specification stage edits the TC-004 entry of the issue #240 `testcases.md` (heading, Level, Then) and the TC-004 row of the issue #240 section of `doc/specifications/amazon-human-checks/technical.md` to name TC-005 as extended by issue #358, with the check after the server stop and during each owner's Refresh price; nothing else in either file changes. |
| TS-TEST-HYG-139 | FR-TEST-HYG-107 | The diff outside `doc/` is `M internal/httpapi/amazon_browser_interrupted_test.go` (the added lines only) and `D internal/httpapi/amazon_browser_cache_log_test.go`. `gofmt -l internal/httpapi` is empty; `go vet ./...` and `go test ./...` pass. |
| TS-TEST-HYG-140 | FR-TEST-HYG-103–104 (delivery and proof) | The tests stage delivers the edit and the deletion and writes `no-tests.md` with runs and fault injections 1–7 of the issue #358 design (a neutral line in every opened-page check fails naming `B0INTR0051` and passes `master`'s test; an old-wording line for `B0INTR0052` fails naming it; the existing failures are unchanged); the development stage writes no code. |
| TS-TEST-HYG-141 | FR-TEST-HYG-108 | Line 378 of `tests/app-information.spec.ts` is replaced by the loop line then the close line, each exactly 6 spaces plus its statement, no trailing space, no braces, no blank line between them. |
| TS-TEST-HYG-142 | FR-TEST-HYG-109 | No other line of the file changes; the braceless `for` already had only the Tab press as its body, so TC-007 executes the same statements in the same order (`count + 1` Tab presses, one close per closer and round) and passes as before. |
| TS-TEST-HYG-143 | FR-TEST-HYG-110 | The diff outside `doc/` is `M tests/app-information.spec.ts`, one hunk `@@ -378 +378,2 @@` (2 insertions, 1 deletion). `src/`, `web/`, Go code, `package.json`, lock files, `playwright.config.ts` and `API_SPECIFICATION.md` are unchanged; `npm run build` and `npx playwright test tests/app-information.spec.ts` pass. |
| TS-TEST-HYG-144 | FR-TEST-HYG-108–110 (delivery and proof) | The tests stage makes the edit and writes `no-tests.md` with runs and the fault injection 1–5 of the issue #222 design (baseline and after-edit list, greps and diff, repeated TC-007 run, a close inside the loop fails TC-007); the development stage writes no code. |
| TS-TEST-HYG-145 | FR-TEST-HYG-111 | Lines 100–102, 145–147 and 178–180 of `internal/amazon/browser_navigation_test.go` (the three Amazon-answer `if` statements, each `if` line, `t.Fatalf` line and closing brace) are deleted; no blank line or comment replaces them, and no comparison of the result is added before any helper call. |
| TS-TEST-HYG-146 | FR-TEST-HYG-112 | Line 35, `t.Fatalf("not a failure: %+v", got)`, becomes `t.Fatalf("gave %q instead of a failure: %+v", got.Result, got)` at the same indentation. `t.Helper()`, the URL/status check, the log-line loop and their messages are unchanged. Updated by issue #384: this message belongs to the result check, whose condition tests only `got.Result != "request_error"`; a `request_error` result with another message or second-hand status fails the failure-detail check of TS-TEST-HYG-177 instead. |
| TS-TEST-HYG-147 | FR-TEST-HYG-113 | No other statement of TC-001, TC-002 or the helper's other callers changes; names, subtests and order stay. A wrong result fails once, at the helper's first check, reported at the caller's line through `t.Helper()`; `t.Fatalf` stops the subtest as the removed checks did. |
| TS-TEST-HYG-148 | FR-TEST-HYG-114 | The diff outside `doc/` is `M internal/amazon/browser_navigation_test.go`, 1 insertion and 10 deletions. Production code, other test files, `go.mod`, `go.sum` and `API_SPECIFICATION.md` are unchanged; `gofmt -l internal/amazon` is empty; `go vet ./...` (two verbs, two arguments) and `go test ./...` pass. |
| TS-TEST-HYG-149 | FR-TEST-HYG-111–114 (delivery and proof) | The tests stage makes the edit and writes `no-tests.md` with runs and fault injections 1–7 of the issue #219 design (baseline run and message, greps, diff, a 200 page with `price-found.html`, `price-not-found.html` and an empty 404 failing through the helper with `gave "<result>" instead of a failure`, repeated runs, gates); the development stage writes no code. |
| TS-TEST-HYG-150 | FR-TEST-HYG-115 | `amazontest.StaleLock(t testing.TB, profile, host string)` in `internal/amazon/amazontest/browser_profile.go`: `MkdirAll(profile, 0o700)`, `RemoveAll` of the three lock marks, `Symlink` of `<host>-12345`, `/tmp/org.chromium.Chromium.gone/SingletonSocket`, `12345`, from one ordered table; any error `t.Fatal(err)`. |
| TS-TEST-HYG-151 | FR-TEST-HYG-116 | `amazontest.BrowserProcesses(profile string) []string`: the `/proc` scan of `master`'s `browserProcesses`, entries `<pid>: <cmdline with each NUL as a space>`; `nil` when `/proc` cannot be read, unreadable command lines skipped. |
| TS-TEST-HYG-152 | FR-TEST-HYG-117 | `amazontest.KillBrowserOnProfile(t testing.TB, profile string)`: `master`'s `killBrowserOnProfile` of `internal/amazon` with `Browser(t)` as executable; same arguments, process group, 20 ms polling, 30 s, 500 ms and 5 s waits and the four messages, leftover entries from `BrowserProcesses`. |
| TS-TEST-HYG-153 | FR-TEST-HYG-118 | `amazontest.ProfileHandler(t testing.TB) http.Handler`: a stateless `ServeMux` with `/dp/` (cookie `session-id=fixture-session-180`, `Path=/`, `MaxAge: 3600`, `price-found.html`), `/gp/aod/ajax` and `/gp/aod/ajax/` (`aod-none.html`), `/static/` (204), both pages `text/html; charset=utf-8`; other paths 404. |
| TS-TEST-HYG-154 | FR-TEST-HYG-115–118 | New `internal/amazon/amazontest/browser_profile_test.go` with the nine tests of the issue #196 test table (TC-001 to TC-009), using `recordingTB`, `/bin/sh` stubs selected by `PRICEFOLLOWER_BROWSER_PATH`, `sh -c 'sleep N; :' <path>` processes and real Chromium for TC-005; every started process is killed in `t.Cleanup`. |
| TS-TEST-HYG-155 | FR-TEST-HYG-119 | `writeStaleLock`, `killBrowserOnProfile` (`internal/amazon/browser_profile_test.go`), `browserProcesses` (`browser_test.go`), `leaveStaleLock`, `profileProcesses`, `killBrowser` (`internal/service/amazon_browser_profile_test.go`), `browserProfileServer` and `leaveOtherHostLock` (`internal/httpapi/amazon_browser_profile_test.go`) are deleted with their doc comments; no `/proc`, `org.chromium.Chromium.gone`, `SetCookie`, `aod-none.html`, `/gp/aod/ajax` or `/static/` remains in the service and HTTP files. |
| TS-TEST-HYG-156 | FR-TEST-HYG-120 | Every former call is replaced in place per the issue #196 callers table: stale-lock writers by `amazontest.StaleLock(t, X, "otherhost")`, kill helpers by `amazontest.KillBrowserOnProfile(t, X)`, the scan in `assertNothingLeft` by `amazontest.BrowserProcesses`, `browserProfileServer(t)` by `amazontest.NewServer(t, amazontest.ProfileHandler(t))`; only imports the compiler requires change. |
| TS-TEST-HYG-157 | FR-TEST-HYG-120 | `profileServer` keeps its signature, results, `arrived` buffer, release and cleanup order; one `http.HandlerFunc` reports `/dp/<ASIN>` on `arrived`, holds `blocked` until release or the request's end, then hands every request to `amazontest.ProfileHandler(t)`. |
| TS-TEST-HYG-158 | FR-TEST-HYG-121 | No test or subtest name, step or assertion of the issue #180 profile tests or of the other `assertNothingLeft` callers changes; `assertNothingLeft` keeps its message, 5 s deadline and goroutine check. Service leftover messages now list `<pid>: <command line>` and its kill steps fail at once on an early exit. |
| TS-TEST-HYG-159 | FR-TEST-HYG-122 | The diff outside `doc/` is `A internal/amazon/amazontest/browser_profile.go`, `A internal/amazon/amazontest/browser_profile_test.go` and `M` of the five caller files; `browser_profile.go` imports only the standard library. `gofmt -l internal` is empty; `go vet ./...` and `go test ./...` (including the dependency check) pass. |
| TS-TEST-HYG-160 | FR-TEST-HYG-115–122 (delivery and proof) | The tests stage writes the new test file and all caller edits (red: four packages do not compile); the development stage writes only `browser_profile.go`, runs the TC-010 greps, the TC-012 to TC-014 runs (with the two reverted fault injections of the issue #196 design, recorded in `implementation.md`) and the gates. |
| TS-TEST-HYG-161 | FR-TEST-HYG-117, FR-TEST-HYG-123 | The `case <-exited:` branch of `KillBrowserOnProfile` calls `kill()` (`SIGKILL` to `-cmd.Process.Pid`, error ignored, then `<-exited`, already closed) before `t.Fatal`; a child left in the group is killed, an empty group gives an ignored `ESRCH`. |
| TS-TEST-HYG-162 | FR-TEST-HYG-124 | The message `the browser exited before holding the profile` and `t.Fatal` are unchanged and no wait is added, so the failure stays immediate with or without a stale `SingletonLock`. |
| TS-TEST-HYG-163 | FR-TEST-HYG-125 | New `TestKillBrowserOnProfile_Issue387_TC001_ExitBeforeHoldingKillsTheGroup`, subtests `stale lock` and `no SingletonLock`: a `writeStub` stub starts `sh -c 'sleep 60; :' "$profile" &`, writes `$!` to `child.pid` and exits; checks the exact message, `took < 5 s`, the pid file, `waitNoProcesses(profile)` empty and `SingletonLock` absent; `t.Cleanup` kills the recorded child. |
| TS-TEST-HYG-164 | FR-TEST-HYG-125 | `TestKillBrowserOnProfile_Issue196_TC006_ExitBeforeHolding` and the other tests of the file keep their names, rows and checks; only the file's header comment gains the issue #387 sentence. |
| TS-TEST-HYG-165 | FR-TEST-HYG-126 | The diff outside `doc/` is `M internal/amazon/amazontest/browser_profile.go` (one added line) and `M internal/amazon/amazontest/browser_profile_test.go`. The other branches of the helper, its callers, `go.mod` and `go.sum` are unchanged; `gofmt -l internal` is empty; `go vet ./...` and `go test ./...` (including `TestServerDoesNotDependOnTestHelper`) pass. |
| TS-TEST-HYG-166 | FR-TEST-HYG-123–126 (delivery and proof) | The tests stage writes the new test (red: both subtests fail with `processes left after the early exit`); the development stage adds the `kill()` line and records in `implementation.md` runs 1–4 of the issue #387 design, including the reverted fault injection that removes the line. |
| TS-TEST-HYG-167 | FR-TEST-HYG-127 | `amazontest.CaptureOutput(t *testing.T) func() string` in `internal/amazon/amazontest/capture.go` returns `StartCapture(t).End`. `StartCapture(t testing.TB) *Capture` creates the pipe before redirecting anything; a setup error gives `t.Fatalf("capture the standard output: %v", err)`. Then it sets `log` to the logs buffer and `os.Stdout` to the pipe's write end, starts the copy goroutine and registers `End` with `t.Cleanup`. |
| TS-TEST-HYG-168 | FR-TEST-HYG-127 | Two unexported `outputBuffer`s (mutex and `bytes.Buffer`, locked `Write`/`String`) hold the logged and the printed text. Writers, the copy goroutine and live readers share them without a data race; `os.Stdout` is assigned only in `StartCapture` and `End`. |
| TS-TEST-HYG-169 | FR-TEST-HYG-128 | `(*Capture).End` runs once (`sync.Once`). It restores `os.Stdout`, closes the write end, waits for the copy to reach EOF, closes the read end, restores `log`, stores `logs + printed` and returns it. Later calls and the cleanup return the stored text; nothing written afterwards is captured. |
| TS-TEST-HYG-170 | FR-TEST-HYG-129 | `(*Capture).Live` returns `logs.String() + printed.String()` without waiting or changing state; text still in the pipe appears in a later read. |
| TS-TEST-HYG-171 | FR-TEST-HYG-127–129 | New `internal/amazon/amazontest/capture_test.go` with `TestCaptureOutput_Issue218_TC001` to `TC009` of the issue #218 test table. It uses previous destinations set and restored by `previousDestinations`. TC-009 re-runs the test binary with `AMAZONTEST_CAPTURE_SETUP_FAILURE=1` and a lowered `RLIMIT_NOFILE`. |
| TS-TEST-HYG-172 | FR-TEST-HYG-130 | `captureOutput` and `lockedBuffer` (`internal/amazon/browser_test.go`), `captureCheckOutput` (`internal/httpapi/amazon_browser_navigation_test.go`), and `captureProfileOutput` and `profileOutput` (`internal/service/amazon_browser_profile_test.go`) are deleted with their doc comments. Every call of the first two becomes `amazontest.CaptureOutput(t)` per the issue #218 callers table. No `os.Stdout =` or `os.Pipe()` remains outside `internal/amazon/amazontest/` (and `cdp.go`). |
| TS-TEST-HYG-173 | FR-TEST-HYG-130 | `internal/service/amazon_browser_profile_test.go` calls `amazontest.StartCapture(t)` in its three captures. `waitStartFailures` takes `*amazontest.Capture` and reads `output.Live()`, and so does TC-009's final check. |
| TS-TEST-HYG-174 | FR-TEST-HYG-131 | No test or subtest name, step or assertion of the caller files changes; only the capture calls, the types and reads of `internal/service`, the removed copies and the imports the compiler requires. The diff outside `doc/` is `A capture.go`, `A capture_test.go` and `M` of the thirteen caller files; `capture.go` imports only the standard library. `gofmt -l internal` is empty; `go vet ./...` and `go test ./...` (including the dependency check) pass. |
| TS-TEST-HYG-175 | FR-TEST-HYG-127–131 (delivery and proof) | The tests stage writes `capture_test.go` and all caller edits; in the red phase, four packages do not compile. The development stage writes only `capture.go` and records in `implementation.md` runs 1–6 of the issue #218 design, including the `=== RUN` comparison with `master` and the reverted `Live` fault injection. |
| TS-TEST-HYG-176 | FR-TEST-HYG-112, FR-TEST-HYG-132 | The result check: the first statement after `t.Helper()` in `assertNavigationFailure` is `if got.Result != "request_error" { t.Fatalf("gave %q instead of a failure: %+v", got.Result, got) }`, line 34's condition reduced in place; lines 35–36 unchanged. |
| TS-TEST-HYG-177 | FR-TEST-HYG-133 | The failure-detail check directly follows: `if got.Message != amazonCheckFailed \|\| got.SecondHandStatus != "check_error" { t.Fatalf("gave a failure with message %q and second-hand status %q, not %q and %q: %+v", got.Message, got.SecondHandStatus, amazonCheckFailed, "check_error", got) }`. |
| TS-TEST-HYG-178 | FR-TEST-HYG-134 | The two conditions together equal `master`'s, so the same results pass. The URL/status check, the log-line loop, their messages, the doc comment and every caller in `browser_navigation_test.go`, `browser_cache_log_test.go` and `browser_interrupted_test.go` are unchanged; `N` runs the same tests and subtests and passes. |
| TS-TEST-HYG-179 | FR-TEST-HYG-135 | The diff outside `doc/` is `M internal/amazon/browser_navigation_test.go`, 4 insertions and 1 deletion, all within the helper's first check. Production code, other test files, `go.mod`, `go.sum` and `API_SPECIFICATION.md` are unchanged; `gofmt -l internal/amazon` is empty; `go vet ./...` (two and five verbs, matched by their arguments) and `go test ./...` pass. |
| TS-TEST-HYG-180 | FR-TEST-HYG-132–135 (delivery and proof) | The tests stage makes the edit and writes `no-tests.md` with runs and fault injections 1–9 of the issue #384 design (baseline run and contradictory message, greps, diff, a 200 page with `price-found.html` and `price-not-found.html` failing at the result check only, a wrong expected message and a wrong expected status failing at the failure-detail check, repeated runs, gates); the development stage writes no code. |

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
| `doc/changes/2026-11-23-2118-issue-238-tc-005-fail-if-the-check-ended-before-th/no-tests.md` | tests (issue #238) | explain why no test is written (the guard exists on `master`); record the mutation run and its failure output; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #238) | set the status of FR-TEST-HYG-027–029 and TS-TEST-HYG-031–034 to implemented; FR-TEST-HYG-019 closed by the merge |
| `doc/changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n/testcases.md` | technical specification (issue #181) | TC-006 Given line: `HTTP 404` → `HTTP 403`; TC-004 Corner cases line: add the 404/410 "listing unavailable" corner case (done in this stage) |
| `doc/changes/2026-11-24-0350-issue-181-correct-tc-006-to-use-a-non-404-410-http/no-tests.md` | tests (issue #181) | explain why no test is written (document correction, behaviour already tested) and where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #181) | set the status of FR-TEST-HYG-030–032 and TS-TEST-HYG-035–037 to implemented |
| `internal/service/amazon_human_checks_test.go` | tests (issue #182) | TC-032 "Refresh price running": add the `SetNextCheck` step after `waitStarted` |
| `internal/service/amazon_human_checks_issue182_test.go` | tests (issue #182) | new: `TestAmazonRestartAfterStop_Issue182_TC004_NoScheduledCheckAfterRefreshPrice` |
| `doc/changes/2026-11-24-0458-issue-182-make-the-tc-032-refresh-price-subtest-re/no-tests.md` | tests (issue #182) | explain why the red phase is skipped (tests of shipped behaviour); record the three mutation checks and their failure messages; where TC-003 and TC-006 are proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #182) | set the status of FR-TEST-HYG-033–036 and TS-TEST-HYG-038–043 to implemented |
| `tests/platform-tabs.spec.ts` | tests (issue #295) | TC-010: replace the `imageRequests.every` assertion with the `expect.poll` on `imageRequests.length` |
| `doc/changes/2026-11-24-0608-issue-295-replace-the-dead-image-request-assertion/no-tests.md` | tests (issue #295) | explain why the red phase is skipped (the change is the locked test code itself); record the three fault injections and their failures; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #295) | set the status of FR-TEST-HYG-037–039 and TS-TEST-HYG-044–047 to implemented |
| `internal/amazon/browser_profile_test.go` | tests (issue #195) | split `assertBrowserRefusesProfile` into `runBrowserOnProfile` and `fixtureCheckFailure` with `lockRefusal` and a `host` parameter; TC-002 passes `"otherhost"`; new `refusalStub`, `refusalStubBrowser`, `TestBrowserProfile_Issue195_RefusalStubProcess` and `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal`; imports as needed (`encoding/json`, `fmt`) |
| `doc/changes/2026-11-24-0917-issue-195-make-tc-002-s-fixture-sanity-check-prove/no-tests.md` | tests (issue #195) | explain why the red phase is skipped (the change is the locked test code itself); record the real refusal line, the TC-002 runs and the fault injection with its failure; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #195) | set the status of FR-TEST-HYG-040–043 and TS-TEST-HYG-048–055 to implemented (and, only if `no-tests.md` records another refusal text, update TS-TEST-HYG-049 and the issue #195 design to it) |
| `internal/amazon/browser_profile_test.go` | tests (issue #312) | `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal`: wait of 15 s for rows without `Hold`, 2 s for hold rows |
| `doc/changes/2026-11-24-1003-issue-312-use-a-longer-wait-for-the-non-hold-rows/no-tests.md` | tests (issue #312) | explain why the red phase is skipped (the change is the locked test code itself); record runs 1–5 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #312) | set the status of FR-TEST-HYG-044–047 and TS-TEST-HYG-056–060 to implemented |
| `internal/amazon/browser_profile_test.go` | tests (issue #315) | `runBrowserOnProfileUntil` with `runBrowserOnProfile` as its wrapper; `refusalStub.Signal`; `refusalStubBrowser` returns the signal path; the hold stub creates the marker; new `waitForOutputSignal`; hold rows wait for the signal (15 s bound), then 2 s, and report a missing signal |
| `doc/changes/2026-11-24-1044-issue-315-start-the-hold-rows-wait-after-the-stub/no-tests.md` | tests (issue #315) | explain why the red phase is skipped (the change is the locked test code itself); record runs and fault injections 1–7 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #315) | set the status of FR-TEST-HYG-048–052 and TS-TEST-HYG-061–068 to implemented |
| `internal/amazon/browser_profile_values_test.go` | tests (issue #324) | subtest "TC-003 temporary profile": register the guarded `t.Cleanup` removal of `path` right after `profileOf("")` |
| `doc/changes/2026-11-24-1503-issue-324-register-removal-of-the-temporary-profil/no-tests.md` | tests (issue #324) | explain why the red phase is skipped (the change is the locked test code itself); record runs and fault injections 1–8 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #324) | set the status of FR-TEST-HYG-053–056 and TS-TEST-HYG-069–073 to implemented |
| `tests/claude-token-settings.spec.ts` | tests (issue #290) | 1025-space token test: replace lines 101–106 with the `/too long/i` check and `expect(api.puts).toEqual([])` |
| `doc/changes/2026-11-25-0104-issue-290-replace-the-vacuous-progress-check-in-th/no-tests.md` | tests (issue #290) | explain why the red phase is skipped (the change is the locked test code itself); record runs and fault injections 1–4 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #290) | set the status of FR-TEST-HYG-057–060 and TS-TEST-HYG-074–079 to implemented |
| `internal/amazon/browser_profile_values_test.go` | tests (issue #322) | delete `TestBrowserProfile_Issue197_TC005_ProfileUsesTheExplicitForm`, its comment and the blank line before it, and the `go/ast`, `go/format`, `go/parser`, `go/token` imports |
| `doc/changes/2026-11-25-0155-issue-322-remove-the-source-text-test-for-tc-005-o/no-tests.md` | tests (issue #322) | explain why the red phase is skipped (the change deletes locked test code that passes on `master`); record runs and fault injections 1–4 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #322) | set the status of FR-TEST-HYG-061–064 and TS-TEST-HYG-080–084 to implemented |
| `internal/amazon/amazontest/server_dependencies_test.go` | tests (issue #254) | `buildModule` runs `go build ./...` then `go vet ./...` and names the failing command; new subtest `control: a test file uses a name the stub does not declare`; comments above `buildModule` and the test; issue #254 block in the header comment |
| `doc/changes/2026-11-25-0300-issue-254-type-check-fixture-test-files-in-buildmo/no-tests.md` | tests (issue #254) | explain why the red phase is skipped (the change is the locked test code itself); record runs and fault injections 1–5 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #254) | set the status of FR-TEST-HYG-065–069 and TS-TEST-HYG-085–090 to implemented |
| `internal/amazon/amazontest/server_dependencies_test.go` | tests (issue #253) | `TestFixtureModulesBuild_Issue251_TC001`: the `default` branch also requires `tc.fails` in the build failure; comment above the test; issue #253 block in the header comment |
| `doc/changes/2026-11-25-0342-issue-253-check-the-build-failure-message-of-fixtu/no-tests.md` | tests (issue #253) | explain why the red phase is skipped (the change is the locked test code itself); record runs and fault injections 1–5 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #253) | set the status of FR-TEST-HYG-070–073 and TS-TEST-HYG-091–095 to implemented |
| `internal/amazon/amazontest/server_dependencies_test.go` | tests (issue #340) | new `vetArgs` (`go vet -atomic ./...`) used by `buildModule`; the test-file control's prefix check derived from `vetArgs`; new subtest `analyzer findings in non-test and test files do not fail the check`; comments above `buildModule` and the test; issue #254 header line and new issue #340 block |
| `doc/changes/2026-11-25-0424-issue-340-clarify-or-narrow-the-go-vet-check-in-bu/no-tests.md` | tests (issue #340) | explain why the red phase is skipped (the change is the locked test code itself); record runs and fault injections 1–6 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #340) | set the status of FR-TEST-HYG-074–077 and TS-TEST-HYG-096–101 to implemented |
| `doc/changes/2026-11-23-1234-issue-233-clear-the-amazon-browser-http-cache-even/testcases.md` | technical specification (issue #239) | TC-003 entry, third "Then" result: `no answer within` for the never-answered subtests, neither context cause in every subtest (done in this stage) |
| `internal/amazon/browser_interrupted_test.go` | tests (issue #239) | `TestBrowserInterrupted_TC003_UnconfirmedClearIsBoundedAndLogged`: comment, `no answer within` check for `clear == "never"`, `assertNoInterruptionGivenForCache(t, logged)` |
| `doc/changes/2026-11-25-0601-issue-239-tc-003-assert-the-no-answer-within-cause/no-tests.md` | tests (issue #239) | explain why the red phase is skipped (the behaviour already holds on `master`); record runs and fault injections 1–6 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #239) | set the status of FR-TEST-HYG-078–081 and TS-TEST-HYG-102–106 to implemented |
| `internal/amazon/amazontest/server_dependencies_test.go` | tests (issue #344) | subtest `analyzer findings in non-test and test files do not fail the check`: comment and findings guard (`go vet ./...` on `dir` with `fixtureEnv`, `t.Error` when it succeeds); new issue #344 block in the header comment |
| `doc/changes/2026-11-25-0652-issue-344-guard-the-analyzer-findings-subtest-agai/no-tests.md` | tests (issue #344) | explain why the red phase is skipped (the change is the locked test code itself); record runs and fault injections 1–5 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #344) | set the status of FR-TEST-HYG-082–085 and TS-TEST-HYG-107–112 to implemented |
| `internal/amazon/amazontest/server_dependencies_test.go` | tests (issue #349) | subtest `analyzer findings in non-test and test files do not fail the check`: guard comment replaced; `CombinedOutput` instead of `Run`; a check per file (`cmd/app/main.go:`, `cmd/app/main_test.go:`) with its own `t.Errorf`; new issue #349 block in the header comment |
| `doc/changes/2026-11-25-0742-issue-349-require-analyzer-findings-in-both-the-no/no-tests.md` | tests (issue #349) | explain why the red phase is skipped (the change is the locked test code itself); record runs and fault injections 1–7 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #349) | set the status of FR-TEST-HYG-086–089 and TS-TEST-HYG-113–117 to implemented |
| `doc/changes/2026-11-23-1706-issue-185-tests-missing-from-168-amazon-human-like/testcases.md` | technical specification (issue #260) | TC-002: one Then line (pairwise different times) and one corner case (one time for every item fails the test) (done in this stage) |
| `internal/service/amazon_human_checks_issue185_test.go` | tests (issue #260) | `TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce`: add the pairwise check after the first start's loop |
| `doc/changes/2026-11-25-0515-issue-260-assert-that-the-upgrade-redraw-draws-a-d/no-tests.md` | tests (issue #260) | explain why the red phase is skipped (added check on shipped behaviour, red shown by fault injection); record runs and fault injections 1–5 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #260) | set the status of FR-TEST-HYG-090–093 and TS-TEST-HYG-118–122 to implemented |
| `doc/changes/2026-11-19-2053-issue-165-use-all-space-to-show-item-list-when-the/testcases.md` | technical specification (issue #199) | TC-005: the Then line about a value starting a later line counts the item value at its base width of 14rem (done in this stage) |
| `tests/responsive-layout.spec.ts` | tests (issue #199) | `ITEM_BASIS_REM`; `layoutProbe.cells` returns the content-box `left` (and its type); TC-005 reads `rem` and computes the wrap check's free space from the previous line's base sizes |
| `doc/changes/2026-11-25-1636-issue-199-strengthen-tc-005-wrap-check-after-the-g/no-tests.md` | tests (issue #199) | explain why the red phase is skipped (the change is the locked test code itself); record runs and fault injections 1–5 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #199) | set the status of FR-TEST-HYG-094–098 and TS-TEST-HYG-123–128 to implemented |
| `internal/amazon/browser_cache_log_test.go` | tests (issue #359) | delete `TestBrowserCacheLog_TC003_SuccessfulClearLogsNothing`, its two comment lines and the blank line before them (lines 168–212) |
| `doc/changes/2026-11-25-1822-issue-359-remove-the-duplicate-tc-003-test-of-issu/no-tests.md` | tests (issue #359) | explain why the red phase is skipped (the change deletes locked test code that passes on `master`); record runs and fault injections 1–6 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #359) | set the status of FR-TEST-HYG-099–102 and TS-TEST-HYG-129–134 to implemented |
| `doc/changes/2026-11-25-1255-issue-240-neutral-wording-for-the-cache-clear-fail/testcases.md` | technical specification (issue #358) | TC-004 entry: heading, Level and Then name TC-005 as extended with the check during each owner's Refresh price (done in this stage) |
| `doc/specifications/amazon-human-checks/technical.md` | technical specification (issue #358) | TC-004 row of the issue #240 section: TC-005 extended by issue #358, the `internal/amazon` test unchanged (done in this stage) |
| `internal/httpapi/amazon_browser_interrupted_test.go` | tests (issue #358) | `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile`: per-owner capture and no-cache-line check after `waitStoredCheck` |
| `internal/httpapi/amazon_browser_cache_log_test.go` | tests (issue #358) | delete (via `delete.txt`) |
| `doc/changes/2026-11-25-1454-issue-358-merge-the-issue-240-httpapi-cache-log-te/no-tests.md` | tests (issue #358) | explain why the red phase is skipped (added check on shipped behaviour, red shown by fault injection); record runs and fault injections 1–7 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #358) | set the status of FR-TEST-HYG-103–107 and TS-TEST-HYG-135–140 to implemented |
| `tests/app-information.spec.ts` | tests (issue #222) | TC-007: split line 378 into the loop line and the close line |
| `doc/changes/2026-11-27-2146-issue-222-put-await-close-on-its-own-line-in-the-t/no-tests.md` | tests (issue #222) | explain why the red phase is skipped (the change is the locked test code itself); record runs and the fault injection 1–5 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md` | development (issue #222) | set the status of FR-TEST-HYG-108–110 and TS-TEST-HYG-141–144 to implemented |
| `internal/amazon/browser_navigation_test.go` | tests (issue #219) | delete the three Amazon-answer checks (lines 100–102, 145–147, 178–180); line 35 becomes `t.Fatalf("gave %q instead of a failure: %+v", got.Result, got)` |
| `doc/changes/2026-11-28-0058-issue-219-fold-the-redundant-amazon-answer-check-i/no-tests.md` | tests (issue #219) | explain why the red phase is skipped (the change is the locked test code itself); record runs and fault injections 1–7 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md`, `doc/FUNCTIONAL_SPECIFICATIONS.md` | development (issue #219) | set the status of FR-TEST-HYG-111–114 and TS-TEST-HYG-145–149, and the issue #219 entry of the test suite hygiene row, to implemented |
| `internal/amazon/amazontest/browser_profile_test.go` | tests (issue #196) | new: TC-001 to TC-009 tests of the helpers, with the local `record` runner and stub writers |
| `internal/amazon/browser_profile_test.go` | tests (issue #196) | delete `writeStaleLock` and `killBrowserOnProfile`; calls to `amazontest.StaleLock` and `amazontest.KillBrowserOnProfile`; unused imports removed |
| `internal/amazon/browser_profile_values_test.go` | tests (issue #196) | two calls to `amazontest.StaleLock`; add the `amazontest` import |
| `internal/amazon/browser_test.go` | tests (issue #196) | delete `browserProcesses`; `assertNothingLeft` calls `amazontest.BrowserProcesses`; unused imports removed |
| `internal/service/amazon_browser_profile_test.go` | tests (issue #196) | delete `leaveStaleLock`, `profileProcesses`, `killBrowser`; `profileServer` on `amazontest.ProfileHandler`; calls to `amazontest.StaleLock` and `amazontest.KillBrowserOnProfile`; unused imports removed |
| `internal/httpapi/amazon_browser_profile_test.go` | tests (issue #196) | delete `browserProfileServer` and `leaveOtherHostLock`; `amazontest.NewServer(t, amazontest.ProfileHandler(t))` and `amazontest.StaleLock`; unused imports removed |
| `internal/amazon/amazontest/browser_profile.go` | development (issue #196) | new: `StaleLock`, `BrowserProcesses`, `KillBrowserOnProfile`, `ProfileHandler` with doc comments |
| `doc/changes/2026-11-28-0214-issue-196-move-duplicated-browser-profile-test-hel/implementation.md` | development (issue #196) | the TC-010 greps, the TC-012 to TC-014 runs and the two fault injections, with their results |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md`, `doc/FUNCTIONAL_SPECIFICATIONS.md` | development (issue #196) | set the status of FR-TEST-HYG-115–122 and TS-TEST-HYG-150–160, and the issue #196 entry of the test suite hygiene row, to implemented |
| `internal/amazon/amazontest/browser_profile_test.go` | tests (issue #387) | new `TestKillBrowserOnProfile_Issue387_TC001_ExitBeforeHoldingKillsTheGroup` after the issue #196 TC-008 test; one sentence added to the header comment |
| `internal/amazon/amazontest/browser_profile.go` | development (issue #387) | `kill()` before `t.Fatal` in the `case <-exited:` branch of `KillBrowserOnProfile` |
| `doc/changes/2026-11-28-0408-issue-387-kill-the-browser-s-process-group-when-it/implementation.md` | development (issue #387) | runs 1–4 of the issue #387 design, including the reverted fault injection, with their results |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md`, `doc/FUNCTIONAL_SPECIFICATIONS.md` | development (issue #387) | set the status of FR-TEST-HYG-123–126 (FR-TEST-HYG-117 updated) and TS-TEST-HYG-161–166, and the issue #387 entry of the test suite hygiene row, to implemented |
| `internal/amazon/amazontest/capture_test.go` | tests (issue #218) | new: `TestCaptureOutput_Issue218_TC001` to `TC009` and `previousDestinations` |
| `internal/amazon/browser_test.go` | tests (issue #218) | delete `captureOutput` and `lockedBuffer`; one call to `amazontest.CaptureOutput`; unused imports removed |
| `internal/amazon/browser_interrupted_test.go`, `browser_retrieval_test.go`, `browser_navigation_test.go`, `browser_profile_test.go`, `browser_error_page_test.go`, `browser_profile_values_test.go`, `browser_cache_log_test.go`, `browser_empty_status_test.go` | tests (issue #218) | calls to `amazontest.CaptureOutput`; the `amazontest` import added to the last two |
| `internal/httpapi/amazon_browser_navigation_test.go` | tests (issue #218) | delete `captureCheckOutput`; calls to `amazontest.CaptureOutput`; unused imports removed |
| `internal/httpapi/amazon_browser_interrupted_test.go`, `amazon_browser_error_page_test.go` | tests (issue #218) | calls to `amazontest.CaptureOutput`; the `amazontest` import added to the second |
| `internal/service/amazon_browser_profile_test.go` | tests (issue #218) | delete `profileOutput` and `captureProfileOutput`; `amazontest.StartCapture`, `*amazontest.Capture` in `waitStartFailures`, `Live()` reads; unused imports removed |
| `internal/amazon/amazontest/capture.go` | development (issue #218) | new: `Capture`, `StartCapture`, `Live`, `End`, `CaptureOutput`, `outputBuffer`, with doc comments |
| `doc/changes/2026-11-28-0457-issue-218-share-the-log-stdout-capture-test-helper/implementation.md` | development (issue #218) | runs 1–6 of the issue #218 design, including the reverted fault injection, with their results |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md`, `doc/FUNCTIONAL_SPECIFICATIONS.md` | development (issue #218) | set the status of FR-TEST-HYG-127–131 and TS-TEST-HYG-167–175, and the issue #218 entry of the test suite hygiene row, to implemented |
| `internal/amazon/browser_navigation_test.go` | tests (issue #384) | `assertNavigationFailure`: line 34's condition keeps only `got.Result != "request_error"`; the failure-detail check is inserted after line 36 |
| `doc/changes/2026-11-28-0928-issue-384-make-assertnavigationfailure-s-first-mes/no-tests.md` | tests (issue #384) | explain why the red phase is skipped (the change is the locked test code itself); record runs and fault injections 1–9 and their results; where each TC is proved |
| `doc/specifications/test-suite-hygiene/functional.md`, `technical.md`, `doc/FUNCTIONAL_SPECIFICATIONS.md` | development (issue #384) | set the status of FR-TEST-HYG-132–135 and TS-TEST-HYG-176–180 (FR-TEST-HYG-112 and TS-TEST-HYG-146 updated), and the issue #384 entry of the test suite hygiene row, to implemented |

No production code, frontend, `API_SPECIFICATION.md`, `go.mod`, `go.sum`, `package.json`, `playwright.config.ts`, CI or `scripts/` change. Issue #182 changes one existing test file and adds one. Issues #232 and #244 change no existing test file other than `internal/amazon/amazontest/amazontest_test.go`, and issue #251 changes no test file other than `internal/amazon/amazontest/server_dependencies_test.go`. Issue #241 changes no test file. Issue #185 adds two test files and changes no existing one (except a `gofmt` fix if one is needed). Issue #238 changes no file outside `doc/`. Issue #181 changes no file outside `doc/`. Issue #295 changes one line of `tests/platform-tabs.spec.ts` and no other test file. Issue #195 changes `internal/amazon/browser_profile_test.go` and no other test file. Issue #312 changes only the `t.Run` loop of one test in that file. Issue #315 changes only that file. Issue #324 changes only one subtest of `internal/amazon/browser_profile_values_test.go`; `internal/amazon/browser.go` is edited only temporarily for fault injections and never committed. Issue #290 changes only one test of `tests/claude-token-settings.spec.ts`; `src/components/SettingsModal.tsx` is edited only temporarily for fault injections and never committed. Issue #322 only removes lines from `internal/amazon/browser_profile_values_test.go`; `internal/amazon/browser.go` is edited only temporarily for fault injections and never committed. Issue #254 changes only `internal/amazon/amazontest/server_dependencies_test.go`; its fault injections are temporary edits of that file, never committed. Issue #253 changes only one branch and two comments of the same file; its fault injections are temporary edits of that file, never committed. Issue #340 changes only `buildModule`, `vetArgs`, one control check, one new subtest and three comments of the same file; its fault injections are temporary edits of that file, never committed. Issue #239 changes only the `t.Run` function of one test of `internal/amazon/browser_interrupted_test.go`; `internal/amazon/browser.go` is edited only temporarily for fault injections and never committed. Issue #344 changes only one subtest and the header comment of `internal/amazon/amazontest/server_dependencies_test.go`; its fault injections are temporary edits of that file, never committed. Issue #349 changes only the findings guard of one subtest and the header comment of the same file; its fault injections are temporary edits of that file, never committed. Issue #260 only adds one block to one test of `internal/service/amazon_human_checks_issue185_test.go`; `internal/service/service.go` is edited only temporarily for fault injections and never committed. Issue #199 changes only the `cells` function and type of `layoutProbe`, one constant and the wrap check of TC-005 in `tests/responsive-layout.spec.ts`; `src/index.css` is edited only temporarily for fault injections and never committed. Issue #359 only removes lines from `internal/amazon/browser_cache_log_test.go`; `internal/amazon/browser.go` is edited only temporarily for fault injections and never committed. Issue #358 only adds lines to the per-owner loop of one test of `internal/httpapi/amazon_browser_interrupted_test.go` and deletes `internal/httpapi/amazon_browser_cache_log_test.go`; `internal/amazon/browser.go`, `internal/httpapi/amazon_browser_navigation_test.go` and the TC-005 setup are edited only temporarily for fault injections and never committed. Issue #222 only splits line 378 of `tests/app-information.spec.ts` into two lines; its fault injection is a temporary edit of that file, never committed. Issue #219 only deletes three checks from `internal/amazon/browser_navigation_test.go` and replaces one line of it; its fault injections are temporary edits of that file, never committed. Issue #196 adds one non-test file and one test file to `internal/amazon/amazontest/` and changes only the helpers, their calls and imports in five caller test files; its fault injections are temporary edits or environment settings, never committed. Issue #387 adds one line to `browser_profile.go` and one test plus a header sentence to `internal/amazon/amazontest/browser_profile_test.go`; its fault injection is a temporary edit of `browser_profile.go`, never committed. Issue #218 adds one non-test file and one test file to `internal/amazon/amazontest/` and changes only the capture copies, their calls, types, reads and imports in thirteen caller test files; its fault injection is a temporary edit of `capture.go`, never committed. Issue #384 only splits the first check of `assertNavigationFailure` in `internal/amazon/browser_navigation_test.go`; its fault injections are temporary edits of that file, never committed. Apart from issue #208's scratch file and issue #358's duplicate, no test file is deleted.

## Test case mapping (issue #384)

No new test file: the tests stage changes `internal/amazon/browser_navigation_test.go` and writes `no-tests.md`. Runs below refer to the issue #384 design; `N` and `F` are the runs named there.

| Test case | Proved by |
| --- | --- |
| TC-001 | Run 3, code check: one line for the result check's message, inside the helper, with `got.Result, got`; no `"not a failure: `; the result check first after `t.Helper()` and testing only `Result`; the failure-detail check right after it, with its condition, `%q` verbs, `"check_error"` and the ending `%+v` of `got`; both `t.Fatalf`; the later checks unchanged (TS-TEST-HYG-176, TS-TEST-HYG-177) |
| TC-002 | Run 4, code check: `git diff --unified=0 origin/master -- internal/amazon/browser_navigation_test.go` touches only the helper's first check; 4 insertions, 1 deletion (TS-TEST-HYG-178, TS-TEST-HYG-179) |
| TC-003 | `internal/amazon/browser_navigation_test.go`: `TestBrowserNavigation_TC002_NoContentIsAFailure` under injection A of run 5 (`F` fails with `gave "success"` and `gave "price_not_found" instead of a failure: {`, once per subtest, without the failure-detail message); reverted afterwards (TS-TEST-HYG-176) |
| TC-004 | Same test under injection C of run 6 (expected message `amazonCheckFailed + "x"`): the failure-detail message with both messages and `"check_error"`, ending with the result, without `instead of a failure`, `gave "request_error"` or the later checks' messages; run 2 records the former contradictory message; reverted afterwards (TS-TEST-HYG-177) |
| TC-005 | Same test under injection D of run 7 (expected status `check_errorx`): the failure-detail message with `"check_error"` and `"check_errorx"`; reverted afterwards (TS-TEST-HYG-177) |
| TC-006 | `internal/amazon/browser_navigation_test.go`, `browser_cache_log_test.go`, `browser_interrupted_test.go`: runs 1 and 8, `N` passes before and twice after, same `=== RUN` lines (TS-TEST-HYG-178) |
| TC-007 | Run 9 and the gates: outside `doc/` only `M internal/amazon/browser_navigation_test.go`; `gofmt -l internal/amazon` empty; `go vet ./...` and `go test ./...` pass (TS-TEST-HYG-179) |

## Test case mapping (issue #218)

Runs refer to the issue #218 design.

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/amazon/amazontest/capture_test.go`: `TestCaptureOutput_Issue218_TC001_Signature`; run 2 (`go doc`); compilation of the caller packages in run 6 (TS-TEST-HYG-167) |
| TC-002 | same file: `TestCaptureOutput_Issue218_TC002_LoggedThenPrinted` (TS-TEST-HYG-167, TS-TEST-HYG-169) |
| TC-003 | same file: `TestCaptureOutput_Issue218_TC003_LaterCallsSameText` (TS-TEST-HYG-169) |
| TC-004 | same file: `TestCaptureOutput_Issue218_TC004_WaitsForPrintedText`; `-count=20` in run 1 (TS-TEST-HYG-169) |
| TC-005 | same file: `TestCaptureOutput_Issue218_TC005_EndsAtTestEnd` (TS-TEST-HYG-167, TS-TEST-HYG-169) |
| TC-006 | same file: `TestCaptureOutput_Issue218_TC006_SequentialCaptures` (TS-TEST-HYG-169) |
| TC-007 | same file: `TestCaptureOutput_Issue218_TC007_LiveRead` (TS-TEST-HYG-170) |
| TC-008 | same file: `TestCaptureOutput_Issue218_TC008_ConcurrentWritesNoRace`, with `-race` in run 1 (TS-TEST-HYG-168) |
| TC-009 | same file: `TestCaptureOutput_Issue218_TC009_SetupFailureFailsTheTest` (TS-TEST-HYG-167) |
| TC-010 | Run 3: the three greps and the word grep of TC-010 (TS-TEST-HYG-172) |
| TC-011 | Run 4 (same `=== RUN` lists as `master`, both pass) and run 5 (the reverted `Live` fault injection fails TC-007 of `internal/service` in `waitStartFailures`); code check of `git diff origin/master` on the caller files against the issue #218 callers table (TS-TEST-HYG-172–174) |
| TC-012 | `internal/amazon/amazontest/server_dependencies_test.go`: `TestServerDoesNotDependOnTestHelper`, in run 6 (TS-TEST-HYG-174) |
| TC-013 | Run 6: `git diff --name-only origin/master -- . ':!doc'`, `gofmt -l internal`, `go vet ./...`, `go test ./...` (TS-TEST-HYG-174) |

## Test case mapping (issue #387)

Runs refer to the issue #387 design; `K` is the run named there.

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/amazon/amazontest/browser_profile_test.go`: `TestKillBrowserOnProfile_Issue387_TC001_ExitBeforeHoldingKillsTheGroup`, subtests `stale lock` and `no SingletonLock` (TS-TEST-HYG-161–163) |
| TC-002 | same file: the three rows of `TestKillBrowserOnProfile_Issue196_TC006_ExitBeforeHolding`, unchanged; run 1 with `-count=1` and `-count=2` (TS-TEST-HYG-161, TS-TEST-HYG-162, TS-TEST-HYG-164) |
| TC-003 | Run 2: the new test fails with the `kill()` line temporarily removed, the TC-006 rows pass, no `sleep 60` process is left; passes again once restored (TS-TEST-HYG-163, TS-TEST-HYG-166) |
| TC-004 | same file: `TestKillBrowserOnProfile_Issue196_TC005_LeavesTheLock` (real Chromium), `TC007_NeverHolding` and `TC008_LeftoverAndMissingLock`, unchanged, in run 1 (TS-TEST-HYG-165) |
| TC-005 | Run 3: existing tests of `internal/amazon`, `internal/service` and `internal/httpapi` that call `KillBrowserOnProfile`, with Chromium (TS-TEST-HYG-165) |
| TC-006 | `internal/amazon/amazontest/server_dependencies_test.go`: `TestServerDoesNotDependOnTestHelper`, in run 4 (TS-TEST-HYG-165) |
| TC-007 | Run 4: `git diff --name-only origin/master -- . ':!doc'`, `gofmt -l internal`, `go vet ./...`, `go test ./...` (TS-TEST-HYG-165) |

## Test case mapping (issue #196)

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/amazon/amazontest/browser_profile_test.go`: `TestStaleLock_Issue196_TC001_CreatesTheProfile` (TS-TEST-HYG-150) |
| TC-002 | same file: `TestStaleLock_Issue196_TC002_ReplacesTheMarks` (TS-TEST-HYG-150) |
| TC-003 | same file: `TestStaleLock_Issue196_TC003_ErrorFailsTheTest` (TS-TEST-HYG-150) |
| TC-004 | same file: `TestBrowserProcesses_Issue196_TC004_ListsTheProfileProcesses`; the unreadable-`/proc` corner case by reading `BrowserProcesses` (TS-TEST-HYG-151) |
| TC-005 | same file: `TestKillBrowserOnProfile_Issue196_TC005_LeavesTheLock`, real Chromium (TS-TEST-HYG-152) |
| TC-006 | same file: `TestKillBrowserOnProfile_Issue196_TC006_ExitBeforeHolding` (TS-TEST-HYG-152) |
| TC-007 | same file: `TestKillBrowserOnProfile_Issue196_TC007_NeverHolding` (TS-TEST-HYG-152) |
| TC-008 | same file: `TestKillBrowserOnProfile_Issue196_TC008_LeftoverAndMissingLock` (TS-TEST-HYG-152) |
| TC-009 | same file: `TestProfileHandler_Issue196_TC009_ServesTheProfilePages` (TS-TEST-HYG-153) |
| TC-010 | Repository check by the development stage: the three greps and the word grep of TC-010, recorded in `implementation.md` (TS-TEST-HYG-155) |
| TC-011 | Code check of `git diff origin/master` on the five caller files (review), against the issue #196 callers table (TS-TEST-HYG-156, TS-TEST-HYG-157) |
| TC-012 | Existing tests: `TestBrowserProfile_*` in `internal/amazon/browser_profile_test.go` and `browser_profile_values_test.go`, `TestAmazonBrowserProfile_TC007_…` and `TC009_…` in `internal/service/amazon_browser_profile_test.go`, `TestAmazonBrowserProfile_TC010_…` in `internal/httpapi/amazon_browser_profile_test.go`; run with `-count=1` and `-count=2` and `go test -list .` compared with `master` (TS-TEST-HYG-156–158) |
| TC-013 | Existing `TestBrowserCollector_*` and `TestBrowserRetrieval_*` of `internal/amazon` through `assertNothingLeft`, plus the reverted injection of the issue #196 design (TS-TEST-HYG-158) |
| TC-014 | `TestAmazonBrowserProfile_TC009_ScheduledCheckAndRefreshAfterKilledBrowsers` with `PRICEFOLLOWER_BROWSER_PATH=/bin/true` (fault injection, development stage) (TS-TEST-HYG-158) |
| TC-015 | `go doc pricefollower.local/internal/amazon/amazontest`, `git diff --name-only origin/master -- . ':!doc'`, `gofmt -l internal`, `go vet ./...`, `go test ./...` including `server_dependencies_test.go` (TS-TEST-HYG-159) |

## Test case mapping (issue #219)

No new test file: the tests stage changes `internal/amazon/browser_navigation_test.go` and writes `no-tests.md`. Runs below refer to the issue #219 design; `N` and `F` are the runs named there.

| Test case | Proved by |
| --- | --- |
| TC-001 | Run 3, code check: both greps print nothing; no result comparison precedes any of the four helper calls of the file; the `success` comparisons after the normal page and in `inline page` are kept (TS-TEST-HYG-145) |
| TC-002 | Run 3, code check: the new message appears once, in the helper's `t.Fatalf` with `got.Result, got`; the old message is gone; `t.Helper()`, the condition and the other two messages are unchanged (TS-TEST-HYG-146) |
| TC-003 | Run 4, code check: `git diff --unified=0 origin/master -- internal/amazon/browser_navigation_test.go` replaces line 35 and removes lines 100–102, 145–147 and 178–180 only; `git diff --stat` 1 insertion, 10 deletions (TS-TEST-HYG-145, TS-TEST-HYG-146, TS-TEST-HYG-148) |
| TC-004 | `internal/amazon/browser_navigation_test.go`: `TestBrowserNavigation_TC002_NoContentIsAFailure` under the temporary injections of runs 2 and 5 (`F` fails with `gave "success"`, `gave "price_not_found"` and `gave "unavailable" instead of a failure: {` in both subtests, at the helper call line); reverted afterwards (TS-TEST-HYG-146, TS-TEST-HYG-147) |
| TC-005 | `internal/amazon/browser_navigation_test.go`, `browser_cache_log_test.go`, `browser_interrupted_test.go`: runs 1 and 6, `N` passes before and twice after, same `=== RUN` lines (TS-TEST-HYG-147) |
| TC-006 | Run 7 and the gates: outside `doc/` only `M internal/amazon/browser_navigation_test.go`; `gofmt -l internal/amazon` empty; `go vet ./...` and `go test ./...` pass (TS-TEST-HYG-148) |

## Test case mapping (issue #222)

No new test file: the tests stage changes `tests/app-information.spec.ts` and writes `no-tests.md`. Runs below refer to the issue #222 design; `E` is the run named there.

| Test case | Proved by |
| --- | --- |
| TC-001 | Run 2, code check: the tab-and-close grep prints nothing, the trailing-space grep prints neither line, `await close();` appears once in TC-007, directly after the loop line, both at exactly 6 spaces (TS-TEST-HYG-141) |
| TC-002 | Run 2, code check: `git diff --unified=0 origin/master -- tests/app-information.spec.ts` is one hunk removing the tab-and-close line and adding the loop line then the close line; `git diff --stat` 2 insertions, 1 deletion; name, timeout, rounds, closers and checks unchanged (TS-TEST-HYG-141, TS-TEST-HYG-142, TS-TEST-HYG-143) |
| TC-003 | `tests/app-information.spec.ts`: TC-007 and the file's other tests. Runs 1 and 3: `E` passes before and after, `E -g "TC-007" --repeat-each=2` passes, same `--list` output; run 4: a close inside the loop fails TC-007 (TS-TEST-HYG-142, TS-TEST-HYG-144) |
| TC-004 | Run 5 and the gates: outside `doc/` only `M tests/app-information.spec.ts`; no fault injection remains; `npm run build` and the pipeline's frontend gates pass, twice (TS-TEST-HYG-143) |

## Test case mapping (issue #358)

No new test file: the tests stage changes `internal/httpapi/amazon_browser_interrupted_test.go`, deletes the duplicate and writes `no-tests.md`. Runs below refer to the issue #358 design.

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/httpapi/amazon_browser_interrupted_test.go`: `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile` with the per-owner check. Run 1: `-v -count=1`, `-count=2` and `go test ./internal/httpapi/` pass (TS-TEST-HYG-135, TS-TEST-HYG-136) |
| TC-002 | Run 2: a neutral line logged (then printed) after every opened page fails TC-005 naming `B0INTR0051`, not at the server-stop check; `master`'s test passes with the same injection (TS-TEST-HYG-135, TS-TEST-HYG-140) |
| TC-003 | Run 3: an old-wording line for `B0INTR0052` only fails TC-005 naming `B0INTR0052`; code check that only the prefix is searched (TS-TEST-HYG-135, TS-TEST-HYG-140) |
| TC-004 | Run 4: a line logged by the interrupted check and a check ending before the server stop give the same messages as `master`'s test, with no Refresh-check message (TS-TEST-HYG-136, TS-TEST-HYG-140) |
| TC-005 | Run 5, code check: `git diff master` of the test file is one hunk in the per-owner loop adding the capture, comment and check; two captures, not the server-stop one (TS-TEST-HYG-135, TS-TEST-HYG-136) |
| TC-006 | Run 6 with the deletion overlay, then on the committed tree by the development stage: the file is absent, `grep` finds neither name in `*.go`, `go test -list` loses only the duplicate, `go vet ./internal/httpapi/` passes (TS-TEST-HYG-137) |
| TC-007 | Document check by this stage's edits: `git diff master` of the issue #240 `testcases.md` touches only the TC-004 entry, and of `amazon-human-checks/technical.md` only the issue #240 TC-004 row; neither names the deleted file or test (TS-TEST-HYG-138) |
| TC-008 | Run 7 and the gates: outside `doc/` only `M` of the interrupted test and `D` of the duplicate; no fault injection remains; `gofmt -l internal/httpapi`, `go vet ./...` and `go test ./...` pass, twice (TS-TEST-HYG-139) |

## Test case mapping (issue #359)

No test is added: the tests stage deletes the duplicate test and writes `no-tests.md`. Runs below refer to the issue #359 design; `S` is the run named there.

| Test case | Proved by |
| --- | --- |
| TC-001 | Repository checks: `grep -rn 'TestBrowserCacheLog_TC003_SuccessfulClearLogsNothing' --include='*.go' .` prints nothing; `internal/amazon/browser_cache_log_test.go` contains neither `// TC-003` nor `Clear: "answer"`, nor "A successful clear logs no cache line"; no other test file of `internal/amazon` gained its subtest names. Run 2: `go test -count=1 -run 'TestBrowserCacheLog_TC003' -v ./internal/amazon/` reports no tests to run and passes (TS-TEST-HYG-129) |
| TC-002 | Runs 1–2: `go test -run '^TestBrowserCacheLog_' -v ./internal/amazon/` on `master` and with the change (`-count=1`, `-count=2`) passes with the same TC-001 and TC-002 subtest names, no TC-003 test, about 8 s shorter (TS-TEST-HYG-130) |
| TC-003 | Code check of the import block and declarations of `internal/amazon/browser_cache_log_test.go` (each import and declaration still used, no blank import); `go vet ./internal/amazon/` passes; `browser_interrupted_test.go` keeps its helpers (TS-TEST-HYG-130) |
| TC-004 | `git diff master -- internal/amazon/browser_cache_log_test.go` has no added line; its removed lines are lines 168–212 of `master` only; `gofmt -l internal/amazon` is empty (TS-TEST-HYG-129, TS-TEST-HYG-130, TS-TEST-HYG-133) |
| TC-005 | Run 3: `S` passes and lists TC-001 (both kinds), TC-002 (three subtests) and TC-006 `download` (TS-TEST-HYG-131) |
| TC-006 | Runs 4–5: injections A (neutral line on success) and B (old wording on success) each fail `S` with `cache clear failure logged:` in TC-001, TC-002 and TC-006 `download`; after each revert `S` passes (TS-TEST-HYG-131, TS-TEST-HYG-134) |
| TC-007 | Repository checks: `git diff master -- internal/amazon/browser_interrupted_test.go` and `git diff master -- doc/changes/2026-11-25-1255-issue-240-neutral-wording-for-the-cache-clear-fail/ doc/specifications/amazon-human-checks/` are empty; the issue #240 `testcases.md` TC-003 and the amazon-human-checks technical TC-003 row still name the success-clear tests (TS-TEST-HYG-132) |
| TC-008 | Repository checks: `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/browser_cache_log_test.go`; `browser.go`, other production Go code, `src/`, `tests/`, `go.mod`, `go.sum`, `package.json` and `API_SPECIFICATION.md` are unchanged and no fault injection remains. Gates: `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` pass, twice (TS-TEST-HYG-133) |

## Test case mapping (issue #199)

No new test file: the tests stage changes `tests/responsive-layout.spec.ts` and writes `no-tests.md`. Runs below refer to the issue #199 design; `R` is the run named there.

| Test case | Proved by |
| --- | --- |
| TC-001 | `tests/responsive-layout.spec.ts`: `TC-005 card values side by side` with the new wrap check. Run 1: `R` and `R --repeat-each=3` pass at 775 and 1055 px, both tabs, every card (TS-TEST-HYG-123, TS-TEST-HYG-125) |
| TC-002 | Run 2: item basis `40rem` fails `R` at 775 px with the message naming `775px Amazon card 0`, `Prices`, its width and a free space larger than it; without the injection `R` passes; with it, the `master` wrap check passes every 775 px card (TS-TEST-HYG-123, TS-TEST-HYG-124, TS-TEST-HYG-128) |
| TC-003 | Code check of the wrap check: no `card.right - card.cells[position - 1].right`; free space from `card.right - card.left` minus every value of the previous line; the first cell at `ITEM_BASIS_REM * rem` with the constant 14, not a style read; the others at `width`; `toBeLessThan(current.width + 1)` kept (TS-TEST-HYG-123) |
| TC-004 | Run 3: item basis `100%` fails `R` at 775 px on the value starting the line after the item, with the same message form; with it, the `master` test passes (TS-TEST-HYG-123, TS-TEST-HYG-128) |
| TC-005 | Run 4: the list output of `tests/responsive-layout.spec.ts` is the same with the `master` test file and with the change; code check that the diff touches only `ITEM_BASIS_REM`, the `cells` type and `left` field, the `rem` read and the wrap check (TS-TEST-HYG-125) |
| TC-006 | Document check: `git diff master` of the issue #165 `testcases.md` changes only the TC-005 Then line, which contains "14rem" (TS-TEST-HYG-126) |
| TC-007 | Run 5 and the gates: outside `doc/` only `tests/responsive-layout.spec.ts`, in the issue #165 folder only `testcases.md`; no fault injection remains; the e2e suite, `go vet ./...`, `go test ./...` and `node --test scripts/*.test.mjs` pass, twice (TS-TEST-HYG-127) |

## Test case mapping (issue #260)

No new test file: the tests stage changes `internal/service/amazon_human_checks_issue185_test.go` and writes `no-tests.md`. Runs below refer to the issue #260 design.

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/service/amazon_human_checks_issue185_test.go`: `TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce` with the pairwise check. Run 1: `-v -count=1`, `-count=3` and `go test ./internal/service/...` pass (TS-TEST-HYG-118, TS-TEST-HYG-119) |
| TC-002 | Run 2: one shared time for every item fails the test with the message naming `B000001853`, `B000001854` and the time; without the injection it passes; with it, the `master` test file passes (TS-TEST-HYG-118, TS-TEST-HYG-122) |
| TC-003 | Run 3: each of the three shared pairs, selected by ID, fails the test with the message naming that pair, including `B000001853` and `B000001855`; code check that the loop covers every pair of `ids` (TS-TEST-HYG-118, TS-TEST-HYG-122) |
| TC-004 | Run 4: same test name and `PASS` on `master` and with the change; `git diff master` of the test file is one added block after the first start's loop, with no hunk in the other test, the helpers or the header comment (TS-TEST-HYG-119) |
| TC-005 | Document check: `git diff master` of the issue #185 `testcases.md` adds only the TC-002 Then line and corner case of the issue #260 design (TS-TEST-HYG-120) |
| TC-006 | Run 5 and the gates: outside `doc/` only `internal/service/amazon_human_checks_issue185_test.go`, in the issue #185 folder only `testcases.md`; no fault injection remains; `gofmt -l internal/service`, `go vet ./...` and `go test ./...` pass, twice (TS-TEST-HYG-121) |

## Test case mapping (issue #349)

No new test file: the tests stage changes `internal/amazon/amazontest/server_dependencies_test.go` and writes `no-tests.md`. Runs below refer to the issue #349 design; `R` is the run named there.

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/amazon/amazontest/server_dependencies_test.go`: subtest `analyzer findings in non-test and test files do not fail the check` of `TestFixtureModulesBuild_Issue251_TC001`. Run 1: the full vet run on its fixture fails and its recorded output holds both positions; `R` passes with `-count=1` and `-count=3` (TS-TEST-HYG-113, TS-TEST-HYG-116) |
| TC-002 | Run 2: findings removed from main_test.go fail the subtest with the main_test.go message only, and pass with the `master` file; renaming only `Testfoo`, or changing only its `Printf` call, leaves it passing (TS-TEST-HYG-113, TS-TEST-HYG-117) |
| TC-003 | Run 3: findings removed from main.go fail the subtest with the main.go message only; removing only one of its two findings leaves it passing (TS-TEST-HYG-113, TS-TEST-HYG-117) |
| TC-004 | Run 4: findings removed from both files fail the subtest with the module message, without the build message or a panic (TS-TEST-HYG-114, TS-TEST-HYG-117) |
| TC-005 | Run 5: the three `vetArgs` edits give build and main_test.go messages, build and module messages, and the build message only (TS-TEST-HYG-114, TS-TEST-HYG-117) |
| TC-006 | Code check of `git diff master -- internal/amazon/amazontest/server_dependencies_test.go`: one hunk in the subtest replacing the guard comment and `cmd.Run()` with `CombinedOutput`, the `err == nil` module check and the two file checks, with no `t.Fatal`, `return` or analyzer message; the command lines, `buildModule`, `vetArgs`, the subtest name and its fixture have no hunk; one hunk adding the issue #349 header block listing TC-002, TC-003, TC-004, TC-005, TC-007 and TC-008; no comment says that only the module as a whole is guarded (TS-TEST-HYG-113, TS-TEST-HYG-114, TS-TEST-HYG-115) |
| TC-007 | Run 6: the `-v` test and subtest names and results are identical on `master` and with the change; code check that `fixtureCases`, `plainMain`, `helperMain`, the stub, `writeModule` and `fixtureEnv` have no hunk (TS-TEST-HYG-116) |
| TC-008 | Run 7 and the gates: `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/amazontest/server_dependencies_test.go`; no fault injection remains; `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and `go test -count=1 ./internal/amazon/amazontest/...` pass, twice (TS-TEST-HYG-116) |

## Test case mapping (issue #344)

No new test file: the tests stage changes `internal/amazon/amazontest/server_dependencies_test.go` and writes `no-tests.md`. Runs below refer to the issue #344 design; `R` is the run named there.

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/amazon/amazontest/server_dependencies_test.go`: subtest `analyzer findings in non-test and test files do not fail the check` of `TestFixtureModulesBuild_Issue251_TC001`. Run 1: by-hand `go vet ./...` on its fixture fails (findings recorded) while `go build ./...` and `go vet -atomic ./...` succeed; `R` passes with `-count=1` and `-count=3` (TS-TEST-HYG-107, TS-TEST-HYG-110) |
| TC-002 | Run 2: the finding-free fixture fails the subtest with the guard message and without `valid fixture does not build`, and passes with the `master` file; renaming only `Testfoo` leaves it passing (TS-TEST-HYG-107, TS-TEST-HYG-112) |
| TC-003 | Run 3: `vetArgs` without `-atomic` gives only `valid fixture does not build`; `vetArgs` = `nosuchcmd` with the finding-free fixture gives both messages; a compile error with the findings kept gives only `valid fixture does not build` (TS-TEST-HYG-107, TS-TEST-HYG-108, TS-TEST-HYG-112) |
| TC-004 | Code check of `git diff master -- internal/amazon/amazontest/server_dependencies_test.go`: one hunk at the end of the subtest adding the comment and `exec.Command("go", "vet", "./...")` with `cmd.Dir = dir`, `cmd.Env = append(os.Environ(), fixtureEnv...)` and `t.Error` with the guard message only when it succeeds, after the `buildModule` check and with no `t.Fatal` or `return`; one hunk adding the issue #344 header block listing TC-002, TC-003, TC-005 and TC-006; `buildModule`, `vetArgs`, the subtest name and its fixture have no hunk (TS-TEST-HYG-107, TS-TEST-HYG-108, TS-TEST-HYG-109) |
| TC-005 | Run 4: the `-v` test and subtest names and results of the build test and of the three dependency tests are identical on `master` and with the change; code check that `fixtureCases`, `plainMain`, `helperMain`, the stub, `writeModule` and `fixtureEnv` have no hunk (TS-TEST-HYG-110) |
| TC-006 | Run 5 and the gates: `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/amazontest/server_dependencies_test.go`; no fault injection remains; `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and `go test -count=1 ./internal/amazon/amazontest/...` pass, twice (TS-TEST-HYG-111) |

## Test case mapping (issue #239)

No new test file: the tests stage changes `internal/amazon/browser_interrupted_test.go` and writes `no-tests.md`. Runs below refer to the issue #239 design; `R` is the run of TC-003 named there. `<233 doc>` is `doc/changes/2026-11-23-1234-issue-233-clear-the-amazon-browser-http-cache-even/testcases.md`.

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/amazon/browser_interrupted_test.go`: subtests `no answer`, `error answer` and `no answer, page still requested` of `TestBrowserInterrupted_TC003_UnconfirmedClearIsBoundedAndLogged`. Run 1: `R` with `-count=1` and `-count=3`, exactly three subtests named as on `master` (TS-TEST-HYG-102, TS-TEST-HYG-103) |
| TC-002 | Run 2: without the deadline mapping, `R` fails the two never-answered subtests and passes `error answer`; with the `master` test file it passes; with either new check removed, the other still fails them (TS-TEST-HYG-102, TS-TEST-HYG-103, TS-TEST-HYG-106) |
| TC-003 | Run 3: with `no reply within`, `R` fails the two never-answered subtests with the `no answer within` message and passes `error answer` (TS-TEST-HYG-102, TS-TEST-HYG-106) |
| TC-004 | Run 4: with `: context deadline exceeded` or `: context canceled` appended to the "error answer" line, `R` fails `error answer` with a message holding that line (TS-TEST-HYG-103, TS-TEST-HYG-106) |
| TC-005 | Code check of `git diff master -- internal/amazon/browser_interrupted_test.go`: one hunk inside the `t.Run` function of TC-003, adding the comment, the check guarded by `test.clear == "never"` with `lines[0]` in its message, and `assertNoInterruptionGivenForCache(t, logged)`; the subtest table, the other checks of the subtest and the helpers have no hunk (TS-TEST-HYG-102, TS-TEST-HYG-103, TS-TEST-HYG-105) |
| TC-006 | Document check: `git diff --name-status master -- doc/changes/2026-11-23-1234-issue-233-clear-the-amazon-browser-http-cache-even/` lists only `M <233 doc>`, and `git diff master -- <233 doc>` changes exactly one line, the third "Then" result of the TC-003 entry, which holds `no answer within`, `context canceled`, `context deadline exceeded` and `cache clear refused by fake` (TS-TEST-HYG-104) |
| TC-007 | Run 5: `go test -run '^TestBrowserInterrupted_' -v -count=1 ./internal/amazon/` gives the same tests, subtests, names and results on `master` and with the change (TS-TEST-HYG-105) |
| TC-008 | Run 6 and the gates: `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/browser_interrupted_test.go`; no fault injection remains; `gofmt -l internal/amazon` prints nothing; `go vet ./...` and `go test ./...` pass, twice (TS-TEST-HYG-105) |

## Test case mapping (issue #340)

No new test file: the tests stage changes `internal/amazon/amazontest/server_dependencies_test.go` and writes `no-tests.md`. Runs below refer to the issue #340 design.

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/amazon/amazontest/server_dependencies_test.go`: subtest `analyzer findings in non-test and test files do not fail the check` of `TestFixtureModulesBuild_Issue251_TC001`. Run 1: by-hand `go build ./...`, `go vet ./...` (fails, findings recorded) and `go vet -atomic ./...` on its fixture, then `-v -count=1` and `-count=3`; run 4 edit 1: with `master`'s `go vet ./...` it fails (TS-TEST-HYG-096, TS-TEST-HYG-097) |
| TC-002 | Run 2: analyzer findings added to row a's `cmd/app/main.go` and row d's two test files leave the build test and the listing test passing; edits 1 and 2 on the `master` version of the file fail with `valid fixture does not build` (TS-TEST-HYG-096, TS-TEST-HYG-101) |
| TC-003 | Run 3: the `amazontest.Nope` test files of row d, a syntax error in its `cmd/app/main_test.go` and a type error in row a's `cmd/app/main.go` fail the edited row with `valid fixture does not build`, and pass without the edit (TS-TEST-HYG-096, TS-TEST-HYG-101) |
| TC-004 | `internal/amazon/amazontest/server_dependencies_test.go`: subtest `control: a test file uses a name the stub does not declare` (error holds `amazontest.Nope` and starts with `go vet -atomic ./...:` from `vetArgs`), and `control: a program uses a name the stub does not declare`. Run 4 edits 2 and 3: without vet the control fails; with other vet arguments it still passes (TS-TEST-HYG-098, TS-TEST-HYG-101) |
| TC-005 | Run 5: the `-v` subtest names and results of the build test on `master` and with the change (only the new subtest added) and of the three dependency tests (identical); code check that `fixtureCases`, the fixtures, the stub, the loop's messages, `TC-003 h fixture: plainMain`, the `DownloadMarker` control, `writeModule` and `fixtureEnv` have no hunk in `git diff master` (TS-TEST-HYG-098) |
| TC-006 | Code check of the comments above `vetArgs`, `buildModule` and the test and of the header comment: they name `go vet -atomic ./...`, say why one analyzer is named and that analyzer findings do not fail the check; the issue #340 block lists TC-002 to TC-005 and TC-007; no comment calls plain `go vet ./...` only a type check (TS-TEST-HYG-099) |
| TC-007 | Run 6 and the gates: `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/amazontest/server_dependencies_test.go`; no fault injection remains; `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and `go test -count=1 ./internal/amazon/amazontest/...` pass, twice (TS-TEST-HYG-100) |

## Test case mapping (issue #253)

No new test file: the tests stage changes `internal/amazon/amazontest/server_dependencies_test.go` and writes `no-tests.md`. Runs below refer to the issue #253 design.

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/amazon/amazontest/server_dependencies_test.go`: subtests `TC-003 f: a program imports a package that does not exist` and `TC-003 g: a program has a syntax error` of `TestFixtureModulesBuild_Issue251_TC001`. Run 1: the recorded by-hand `go build ./...` outputs, then `-v -count=1` and `-count=3` (TS-TEST-HYG-091) |
| TC-002 | Code check of the `default` branch: `err == nil` fails, otherwise `strings.Contains(err.Error(), tc.fails)` is required, and the mismatch message names `tc.fails` and holds `err`; the valid-row branch checks no text (TS-TEST-HYG-091, TS-TEST-HYG-092) |
| TC-003 | Runs 2 and 3: edits 1–4 fail the row f and g subtests with the expected messages and pass without the edit; edit 1 on the `master` version of the file passes (TS-TEST-HYG-092, TS-TEST-HYG-095) |
| TC-004 | Run 4: the `-v` subtest names and results of the build test are identical on `master` and with the change; code check that `buildModule`, `fixtureCases`, the valid-row branch, `TC-003 h fixture: plainMain`, both controls and their messages have no hunk in `git diff master` (TS-TEST-HYG-093) |
| TC-005 | Run 4: `TestListServerDependencies_TC003_FixtureModules`, `TestCheckServerDependencies_TC002_JudgesListing` and `TestServerDoesNotDependOnTestHelper` give the same subtests and pass on `master` and with the change; `git diff master` shows no hunk in `listServerDependencies`, `checkServerDependencies`, `assertDependsOnHelper`, `assertListingFailure`, `assertPassesCheck`, `writeModule` or `fixtureEnv` (TS-TEST-HYG-093) |
| TC-006 | Run 5 and the gates: `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/amazontest/server_dependencies_test.go`; no fault injection remains; `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and `go test -count=1 ./internal/amazon/amazontest/...` pass, twice (TS-TEST-HYG-094) |

## Test case mapping (issue #254)

No new test file: the tests stage changes `internal/amazon/amazontest/server_dependencies_test.go` and writes `no-tests.md`. Runs below refer to the issue #254 design.

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/amazon/amazontest/server_dependencies_test.go`: `TestFixtureModulesBuild_Issue251_TC001`, valid subtests of `fixtureCases` (row d included) and `TC-003 h fixture: plainMain`. Run 1: `-v -count=1` and `-count=3`, and the by-hand `go build ./...` and `go vet ./...` on each valid fixture with `fixtureEnv` and an empty module cache (TS-TEST-HYG-085) |
| TC-002 | Code check of `buildModule` (both commands, `fixtureEnv`, first failure returned with the command name and output, `nil` only when both succeed); the `-v` row d subtest of run 1 passes, and `plainMain` passes with no test file (TS-TEST-HYG-085, TS-TEST-HYG-086) |
| TC-003 | Runs 2 and 3: the three temporary edits of row d's test files fail the row d subtest (`Nope` named for edits 1 and 2) and pass without the edit; edit 1 on the `master` version of the file passes (TS-TEST-HYG-090) |
| TC-004 | `internal/amazon/amazontest/server_dependencies_test.go`: subtest `control: a test file uses a name the stub does not declare` of `TestFixtureModulesBuild_Issue251_TC001` (error holds `amazontest.Nope` and starts with `go vet ./...`). Run 3: with `buildModule` reverted to `go build ./...` only, it fails (TS-TEST-HYG-086, TS-TEST-HYG-090) |
| TC-005 | Run 4: the `-v` subtest names of the build test on `master` and with the change (only the new control added); code check that `fixtureCases`, the `DownloadMarker` control and the failure messages are unchanged in the diff (TS-TEST-HYG-087) |
| TC-006 | Code check of the comments above `buildModule` and the test and of the header comment; `grep -n 'out of scope\|only runs go build' internal/amazon/amazontest/server_dependencies_test.go` prints nothing (TS-TEST-HYG-088) |
| TC-007 | Run 4: `TestListServerDependencies_TC003_FixtureModules`, `TestCheckServerDependencies_TC002_JudgesListing` and `TestServerDoesNotDependOnTestHelper` give the same subtests and pass on `master` and with the change; `git diff master` shows no hunk in `listServerDependencies`, `checkServerDependencies`, `assertDependsOnHelper`, `assertListingFailure`, `assertPassesCheck` or `fixtureEnv` (TS-TEST-HYG-089) |
| TC-008 | Repository checks: `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/amazontest/server_dependencies_test.go`; no fault injection remains. Gates: `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` pass, twice (TS-TEST-HYG-089) |

## Test case mapping (issue #322)

No test is added: the tests stage deletes the source-text test and writes `no-tests.md`. Runs below refer to the issue #322 design.

| Test case | Proved by |
| --- | --- |
| TC-001 | Repository checks: `grep -rn 'TestBrowserProfile_Issue197_TC005_ProfileUsesTheExplicitForm' --include='*.go' .` prints nothing; no `*_test.go` under `internal/amazon/` contains `parser.ParseFile`, `"browser.go"` as a file to read or `os.ReadFile` of a `.go` file; the deleted comment is gone. Run 4: `go test -run 'TC005_ProfileUsesTheExplicitForm' -v -count=1 ./internal/amazon/` reports no tests to run and passes (TS-TEST-HYG-080) |
| TC-002 | Code check of the import block and declarations of `internal/amazon/browser_profile_values_test.go` (no `go/*` import, `parser.ParseFile`, `token.NewFileSet`, `format.Node` or `ast.`; every remaining import used; no blank import); `go vet ./internal/amazon/` passes (TS-TEST-HYG-081) |
| TC-003 | `git diff master -- internal/amazon/browser_profile_values_test.go` has no added line; its removed lines are the four imports, the blank line, the comment and the test; the other test files of `internal/amazon` are unchanged; `gofmt -l internal/amazon` is empty (TS-TEST-HYG-080–082) |
| TC-004 | Run 4: the `TestBrowserProfile_Issue197` runs with `-count=1`, `-count=3` and `-race` pass, with the four behaviour tests and their subtests run and TC-004 not skipped (TS-TEST-HYG-082) |
| TC-005 | Runs 1–3: the log rewording fails the `master` test; the three result-keeping edits pass `go test ./internal/amazon/` after the deletion; the result-changing edit fails the behaviour tests; each edit reverted (TS-TEST-HYG-083) |
| TC-006 | Repository checks: `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/browser_profile_values_test.go`; `browser.go`, other production Go code, `src/`, `tests/`, `go.mod`, `go.sum`, `package.json` and `API_SPECIFICATION.md` are unchanged and no fault injection remains. Gates: `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` pass, twice (TS-TEST-HYG-084) |

## Test case mapping (issue #290)

No test is added: the tests stage edits the 1025-space token test and writes `no-tests.md`. Runs below refer to the issue #290 design.

| Test case | Proved by |
| --- | --- |
| TC-001 | Code check of the 1025-space token test in `tests/claude-token-settings.spec.ts`: no `Saving settings…` text and no loop over `api.puts`; `await expect(page.getByText(/too long/i)).toHaveCount(0)` and `expect(api.puts).toEqual([])` between `await expect(dialog).toBeHidden()` and the second `openSettings(page)`; setup, fill, Save click and reopening checks unchanged (TS-TEST-HYG-074–077) |
| TC-002 | Run 1 (tests stage, recorded in `no-tests.md`): the test alone, with `--repeat-each=3`, and the whole file pass. Also the `npx playwright test` gate |
| TC-003 | Fault injections 2.1–2.3: each run fails at `expect(api.puts).toEqual([])` with the sent body, before the reopening, and passes again after the revert |
| TC-004 | Fault injection 3: each of `Token too long`, `TOO LONG` and `Value is Too Long` makes the run fail at `page.getByText(/too long/i)`, and the test passes again after the revert |
| TC-005 | Repository checks: `git diff --name-only master -- . ':!doc'` lists only `tests/claude-token-settings.spec.ts`; `git diff master -- tests/claude-token-settings.spec.ts` holds only the replaced lines 101–106; `src/`, `API_SPECIFICATION.md`, `package.json`, the lock file, the issue #286 change folder and `doc/todo/issue-286-r1-03.md` are unchanged and no fault injection remains. Gates: `npx playwright test`, `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs` pass, twice |

## Test case mapping (issue #324)

No test is added: the tests stage edits the subtest and writes `no-tests.md`. Runs below refer to the issue #324 design.

| Test case | Proved by |
| --- | --- |
| TC-001 | Code check of the subtest "TC-003 temporary profile" in `internal/amazon/browser_profile_values_test.go`: `t.Cleanup` on the subtest's `t` right after `profileOf("")`, before the first `t.Fatal`; it removes `path` only when `path` passes the subtest's location check (parent is `filepath.Clean(os.TempDir())`, base name starts with `pricefollower-browser-`) and otherwise removes nothing; `cleanup()`, the `os.Stat` check after it and every other check and message are unchanged; the other subtests are unchanged (TS-TEST-HYG-069–071) |
| TC-002 | Fault injections 2–5 (tests stage, recorded in `no-tests.md`): each fails with its expected message and leaves the fresh `TMPDIR` empty, including with a non-empty profile; injection 2 without the `t.Cleanup` block leaves one `pricefollower-browser-*` directory |
| TC-003 | Fault injection 6: one failure, `the cleanup did not remove the temporary profile: <nil>`, and the `TMPDIR` is empty afterwards |
| TC-004 | Fault injection 7: the subtest fails on its existing checks, and the before/after listings of `internal/amazon` and of the `TMPDIR` are identical |
| TC-005 | Run 1: `-v`, `-count=3`, `-race` and `-run 'TestBrowserProfile'` pass with no leftover. Also the `go test ./...` gate |
| TC-006 | Repository checks: `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/browser_profile_values_test.go`; `git diff master -- internal/amazon/browser_profile_values_test.go` holds only the `t.Cleanup` block; `browser.go`, the issue #197 change folder and `doc/todo/issue-197-r1-03.md` are unchanged and no fault injection remains. Gates: `gofmt -l internal/amazon` prints nothing; `go vet ./...` and `go test ./...` pass, twice |

## Test case mapping (issue #315)

No test is added: the tests stage edits the fixture check test and writes `no-tests.md`.

| Test case | Proved by |
| --- | --- |
| TC-001 | Code check of `internal/amazon/browser_profile_test.go`. The `Hold` branch of `TestBrowserProfile_Issue195_RefusalStubProcess` creates `stub.Signal` after the request and the `Stderr` write, then sleeps. `refusalStubBrowser` puts the signal in the row's own `t.TempDir()`. The `t.Run` loop picks the hold wait (`waitForOutputSignal`, 15 s, then 2 s) or the 15 s `runBrowserOnProfile` call from `row.stub.Hold` only. An empty `PRICEFOLLOWER_REFUSAL_STUB` still returns at once. The rows table is unchanged (TS-TEST-HYG-061–064) |
| TC-002 | Fault injections 2 and 3 of the issue #315 design (tests stage, recorded in `no-tests.md`). With a 3 s, 5 s or 12 s delay of the hold stub, both hold rows pass with their expected results, taking about the delay plus 2 s, and the non-hold rows pass. With the delay and the `master` 2 s wait, `TC-001 refusal, then still running at the end of the wait` fails with `captured output ""` and `the check failed on a refusal without page request` |
| TC-003 | `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal`, rows `TC-001 refusal, then still running at the end of the wait` and `TC-002 slow start`, run 1 with `-v` (captured output equal to `Stderr`, not requested, check result and exact failure text as on `master`, `ESRCH`) |
| TC-004 | Fault injections 4–6 of the issue #315 design (tests stage, recorded in `no-tests.md`): each hold row fails with `the stub did not signal its output within 15 s`, ends within about 15 s of the stub's start (at once when the stub exits), leaves no process group, and the non-hold rows still pass |
| TC-005 | The same test, run 1 with `-v`, `-count=3`, `-race` and `-run 'TestBrowserProfile'`: ten rows pass every time, hold rows 2–3 s and non-hold rows under 2 s in the `--- PASS` timings, `ESRCH` after every row. Also the `go test ./...` gate |
| TC-006 | Repository checks: `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/browser_profile_test.go`. `git diff master -- internal/amazon/browser_profile_test.go` holds only the hunks of TS-TEST-HYG-067 and leaves `assertBrowserRefusesProfile` with its 15 s call, `fixtureCheckFailure`, the non-hold 15 s call, the rows and TC-002 of issue #180 untouched. The issue #195 and #312 change folders and `doc/todo/issue-312-r1-01.md` are unchanged, and no fault injection remains. Gates: `gofmt -l internal/amazon` prints nothing; `go vet ./...` and `go test ./...` pass, twice |

## Test case mapping (issue #312)

No test is added: the tests stage edits the fixture check test and writes `no-tests.md`.

| Test case | Proved by |
| --- | --- |
| TC-001 | Code check of the `t.Run` loop of `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal` in `internal/amazon/browser_profile_test.go`: `wait` is 15 s unless `row.stub.Hold`, then 2 s, and is passed to `runBrowserOnProfile`; the rows table is unchanged (TS-TEST-HYG-056, TS-TEST-HYG-057) |
| TC-002 | Fault injections 2–4 of the issue #312 design (tests stage, recorded in `no-tests.md`): with a 5 s or 3 s delay every non-hold row passes; with the delay and a 2 s wait at least the two named rows fail; a non-hold stub that does not end fails its row after about 15 s, with its group gone |
| TC-003 | `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal`, rows `TC-001 refusal, then still running at the end of the wait` and `TC-002 slow start`, run 1 with `-v` (expected results, about 2 s each, `ESRCH`) |
| TC-004 | The same test, run 1 with `-v`, `-count=3`, `-race` and `-run 'TestBrowserProfile'` (all ten rows pass, non-hold rows under 2 s, `ESRCH` after every row), and the `go test ./...` gate |
| TC-005 | Repository checks: `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/browser_profile_test.go`; `git diff master -- internal/amazon/browser_profile_test.go` touches only the wait lines of the `t.Run` loop, not `assertBrowserRefusesProfile`, `fixtureCheckFailure` or TC-002 of issue #180; the issue #195 change folder and `doc/todo/issue-195-r1-01.md` are unchanged. Gates: `gofmt -l internal/amazon` prints nothing; `go vet ./...` and `go test ./...` pass, twice |

## Test case mapping (issue #195)

| Test case | Proved by |
| --- | --- |
| TC-001 | Code check of `internal/amazon/browser_profile_test.go`: `assertBrowserRefusesProfile` fails unless `fixtureCheckFailure` returns `""`, which needs no page request and `lockRefusal(host)` in the output; `runBrowserOnProfile` reads both after the group kill and `<-exited` (TS-TEST-HYG-048). Also the rows `refusal, then exit` and `refusal, then still running at the end of the wait` of `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal` |
| TC-002 | `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal`, rows `exit at once without output`, `missing library`, `unknown option`, `slow start`, `generic lock error only` and `refusal naming another host` (missing-message failure, output in the message, page not requested, process group gone) |
| TC-003 | Same test, rows `page requested with the refusal` (opened-page failure only) and `page requested without output` (both conditions named) |
| TC-004 | `TestBrowserProfile_TC002_LockOfAnotherHostDoesNotPreventTheCheck`, run with `go test ./internal/amazon -run 'TestBrowserProfile_TC002' -count=1` and `-count=3` (tests stage, recorded in `no-tests.md` with the real refusal line) and in the `go test ./...` gate |
| TC-005 | Fault injection TS-TEST-HYG-053 (tests stage, recorded in `no-tests.md`): TC-002 fails at the fixture check with the opened-page condition and the output, and passes again after the revert |
| TC-006 | Repository checks: `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/browser_profile_test.go`; `git diff master -- internal/amazon/browser_profile_test.go` touches only the fixture sanity check, its new helpers and tests, the import block and the one call line of TC-002; `doc/todo/issue-180-r1-01.md` and the issue #180 change folder are unchanged. Gates: `gofmt -l internal/amazon` prints nothing; `go vet ./...` and `go test ./...` pass, twice |

## Test case mapping (issue #295)

No test file is added: the tests stage edits TC-010 and writes `no-tests.md`.

| Test case | Proved by |
| --- | --- |
| TC-001 | Repository check: `git grep -n 'imageRequests.every' -- tests/platform-tabs.spec.ts` finds nothing; the only remaining uses of `imageRequests` in TC-010 are its declaration, the `push` in the route handler and `imageRequests.length` in the poll |
| TC-002 | `tests/platform-tabs.spec.ts`: `TC-010 Amazon list shows the accepted image or the placeholder`, run with `npx playwright test tests/platform-tabs.spec.ts -g "TC-010"` and again with `--repeat-each=3` (poll on `imageRequests.length` greater than zero, `expect(api.unexpected).toEqual([])` kept, all other steps unchanged) |
| TC-003 | Fault injections 1–3 (tests stage, recorded in `no-tests.md`): each run of TC-010 fails at the stated assertion, and TC-010 passes again after each revert |
| TC-004 | Repository check: `git diff --name-only master -- . ':!doc'` lists only `tests/platform-tabs.spec.ts`; `git diff master -- tests/platform-tabs.spec.ts` has one hunk inside TC-010, replacing line 361 only; `doc/todo/issue-275-r1-01.md` and the issue #275 change folder are unchanged. Gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test` (same test list as `master`) |

## Test case mapping (issue #182)

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/service/amazon_human_checks_test.go`: subtest `Refresh price running` of `TestAmazonHumanChecks_TC032_NoDuplicateCheckWhileOneIsRunning` (due time set to 07:04 after `waitStarted`, `collectDue` while blocked, one check, next check after 07:05 in a window, read after `waitIdle`) |
| TC-002 | Mutation 1 (tests stage, recorded in `no-tests.md`): `go test -count=1 -run 'TestAmazonHumanChecks_TC032_NoDuplicateCheckWhileOneIsRunning/Refresh_price_running' ./internal/service/` fails on the changed subtest, passes on the `master` subtest, and passes again after the revert |
| TC-003 | Subtest `first check after add running` of the same test, unchanged. Repository check: `git diff master -- internal/service/amazon_human_checks_test.go` shows only the added `SetNextCheck` step inside `Refresh price running`. `go test -count=1 -run 'TestAmazonHumanChecks_TC032_NoDuplicateCheckWhileOneIsRunning/first_check_after_add_running' ./internal/service/` passes |
| TC-004 | `internal/service/amazon_human_checks_issue182_test.go`: `TestAmazonRestartAfterStop_Issue182_TC004_NoScheduledCheckAfterRefreshPrice` (no check at 07:00, restarted at 07:30, one check in total after two 07:31 runs, next check in the evening window of 2030-03-05) |
| TC-005 | Mutations 2 and 3 (tests stage, recorded in `no-tests.md`): `go test -count=1 -run Issue182 ./internal/service/` fails with each mutation and passes after each revert |
| TC-006 | Gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test` (same test list as `master`). `go test -count=1 -v -run 'TestAmazonHumanChecks_TC032\|Issue182' ./internal/service/` lists both TC-032 subtests and the TC-004 test as `PASS`. Repository checks: `gofmt -l internal/service` prints nothing; `git diff --name-status master -- . ':!doc'` lists only the two test files of TS-TEST-HYG-043; `go test -list '.*' ./internal/service/` keeps every test name of `master` |

## Test case mapping (issue #181)

No test file is added or changed: the tests stage writes `no-tests.md`, and the cases are repository checks and runs of the existing suites. `<168 doc>` is `doc/changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n/testcases.md`.

| Test case | Proved by |
| --- | --- |
| TC-001 | Repository check: the TC-006 entry of `<168 doc>` (from `### TC-006` to the next `###`) has no `404` or `410` and names HTTP 503, HTTP 403 for the product page and the robot-check page; `git diff master -- <168 doc>` shows, in that entry, only the Given line changed; the sub-test names in `TestBrowserCollector_TC006_NonSuccessAndRobotCheckAreFailures` are `HTTP 503`, `HTTP 403`, `robot check` |
| TC-002 | Repository check: the TC-004 Corner cases line of `<168 doc>` holds the three existing corner cases unchanged followed by the HTTP 404 or HTTP 410 "listing unavailable" case; `git diff master -- <168 doc>` shows, in that entry, only that line changed. Agreement with the tests: the `product page answering 404` sub-test of `TestBrowserCollector_TC004_ResultsMatchTheParser`, and the 410 tests named under Conventions in `testcases.md` |
| TC-003 | Repository check: `git diff --name-status master -- doc/changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n` lists only `M <168 doc>`, and `git diff master -- <168 doc>` has exactly two changed lines, inside the TC-004 and TC-006 entries |
| TC-004 | Repository check: `git diff --name-status master -- . ':!doc'` and `git diff --name-status master -- doc/todo` are empty. `go test -count=1 -run 'TestBrowserCollector_TC004\|TestBrowserCollector_TC006' ./internal/amazon/` passes. Gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test` (same test list as `master`) |

## Test case mapping (issue #238)

No test file is added or changed: the tests stage writes `no-tests.md`, and the cases run the existing TC-005 or check the repository.

| Test case | Proved by |
| --- | --- |
| TC-001 | Repository check of `internal/httpapi/amazon_browser_interrupted_test.go`: the `select` on `stopped` with a `default` branch and the exact message sits after the `api.fixture.flushed` wait and before the first `cancel()`; the 60 s message is unchanged. Integration run with the reverted `cancel(); <-stopped` mutation (tests stage, recorded in `no-tests.md`): `go test -count=1 -run TestAmazonBrowserInterrupted_TC005 ./internal/httpapi/` fails with only the guard's message |
| TC-002 | Integration run of the unchanged test: `go test -count=3 -run TestAmazonBrowserInterrupted_TC005 ./internal/httpapi/` passes; `git diff --quiet master -- internal/httpapi/amazon_browser_interrupted_test.go` shows no assertion removed or weakened |
| TC-003 | Repository check after the merge: `gh issue view 238 --json state,stateReason` shows `CLOSED` and `COMPLETED`; the pull request body holds `Closes #238` |
| TC-004 | Repository check: `git diff --name-only master -- . ':!doc'` is empty. Gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test` (same test list as `master`) |

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
  - First start: `startHuman(t, database, dir, newClock(scheduleDay(10, 0)), &amazonRecorder{}, 1851)`. For each item, `nextCheck` has `windowOf == evening`, `localDate == "2030-03-05"` and is not `former`. `redrawMark(t, dir) == 0`. The three times are kept in a map. Issue #260 adds a check that they are pairwise different (see its design).
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
- Issue #241: after the merge, the owner closes issue #238 as completed (`gh issue close 238 --reason completed`, see TS-TEST-HYG-022). No test depends on it; TC-002 checks it afterwards. Replaced by issue #238: its own merge closes it (TS-TEST-HYG-033).
- Issue #238: none. The pull request closes the issue at the merge; no CI or `scripts/` change is needed.
- Issue #185: none. The tests run in `go test ./...`, and the gate environment already has Chromium for the browser test.
- Issue #181: none. The document edit is made in the technical specification stage; no CI or `scripts/` change is needed.
- Issue #182: none. The tests run in `go test ./...` and need no CI or `scripts/` change.
- Issue #295: none. TC-010 runs in the existing `npx playwright test` gate; no CI, `scripts/` or `playwright.config.ts` change is needed.
- Issue #195: none. The tests run in `go test ./...`; the stub tests need only `/bin/sh`, and the gate environment already has Chromium for TC-002. No CI, `scripts/` or dependency change is needed.
- Issue #312: none. The test runs in `go test ./...`; no CI, `scripts/` or dependency change is needed.
- Issue #315: none. The test runs in `go test ./...` and needs only `/bin/sh` and a temporary folder; no CI, `scripts/` or dependency change is needed.
- Issue #324: none. The subtest runs in `go test ./...` and needs only the system temporary folder; no CI, `scripts/` or dependency change is needed.
- Issue #290: none. The test runs in the existing `npx playwright test` gate; no CI, `scripts/`, dependency or `playwright.config.ts` change is needed.
- Issue #322: none. The remaining tests run in `go test ./...`; no CI, `scripts/` or dependency change is needed.
- Issue #254: none. The check runs in `go test ./...` and needs only the local Go toolchain, offline; no CI, `scripts/` or dependency change is needed.
- Issue #253: none. The check runs in `go test ./...` and needs only the local Go toolchain, offline; no CI, `scripts/` or dependency change is needed.
- Issue #340: none. The check runs in `go test ./...` and needs only the local Go toolchain, offline; the repository's `go vet ./...` gate keeps all its analyzers, and no CI, `scripts/` or dependency change is needed.
- Issue #239: none. The test runs in `go test ./...` with the existing fake browser; the document edit is made in the technical specification stage, and no CI, `scripts/` or dependency change is needed.
- Issue #344: none. The guard runs in `go test ./...` and needs only the local Go toolchain, offline; the repository's `go vet ./...` gate is unchanged, and no CI, `scripts/` or dependency change is needed.
- Issue #349: none. The guard runs in `go test ./...` and needs only the local Go toolchain, offline; the repository's `go vet ./...` gate is unchanged, and no CI, `scripts/` or dependency change is needed.
- Issue #199: none. TC-005 runs in the existing `npx playwright test` gate; the document edit is made in the technical specification stage, and no CI, `scripts/`, dependency or `playwright.config.ts` change is needed.
- Issue #359: none. The remaining tests run in `go test ./...`; no CI, `scripts/` or dependency change is needed.
- Issue #358: none. TC-005 runs in `go test ./...` with the Chromium the gate environment already has; the document edits are made in the technical specification stage, and no CI, `scripts/` or dependency change is needed.
- Issue #222: none. TC-007 runs in the existing `npx playwright test` gate; no CI, `scripts/`, dependency or `playwright.config.ts` change is needed.
- Issue #219: none. The tests run in `go test ./...` with the Chromium the gate environment already has; no CI, `scripts/` or dependency change is needed.
- Issue #196: none. The helper tests run in `go test ./...` with `/bin/sh`, `/bin/true`, `setsid` and the Chromium the gate environment already has; no CI, `scripts/` or dependency change is needed.
- Issue #387: none. The new test runs in `go test ./...` with `/bin/sh` and `sleep`; no CI, `scripts/` or dependency change is needed.
- Issue #218: none. The helper tests run in `go test ./...` and need only the standard library and `/dev/null`; the `-race` run is a development-stage check, not a gate. No CI, `scripts/` or dependency change is needed.
- Issue #384: none. The change only edits a test for behaviour already on `master`, so the TDD red gate needs no follow-up: the tests stage proves the edit by fault injection and records it in `no-tests.md`. The tests run in `go test ./...` with the Chromium the gate environment already has; no CI, `scripts/` or dependency change is needed.
