# Test suite hygiene: technical specification

Status: TS-TEST-HYG-001–003 ready (issue #208); TS-TEST-HYG-004–009 ready (issue #232); TS-TEST-HYG-010–015 implemented (issue #244); TS-TEST-HYG-016–020 implemented (issue #251); TS-TEST-HYG-021–024 implemented (issue #241; TS-TEST-HYG-022 is the manual close of issue #238 after the merge); TS-TEST-HYG-025–030 implemented (issue #185); TS-TEST-HYG-031–034 implemented (issue #238; TS-TEST-HYG-033 replaces the manual close of TS-TEST-HYG-022); TS-TEST-HYG-035–037 implemented (issue #181; the document edit is made by the technical specification stage); TS-TEST-HYG-038–043 implemented (issue #182); TS-TEST-HYG-044–047 implemented (issue #295); TS-TEST-HYG-048–055 implemented (issue #195); TS-TEST-HYG-056–060 implemented (issue #312); TS-TEST-HYG-061–068 implemented (issue #315); TS-TEST-HYG-069–073 implemented (issue #324)
Requirements: [functional](functional.md). Sources: issue #208, issue #232, issue #244, issue #251, issue #241, issue #185, issue #238, issue #181, issue #182, issue #295, issue #195, issue #312, issue #315, issue #324.

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
	if path == "" || filepath.Clean(path) == filepath.Clean(os.TempDir()) {
		return
	}
	if err := os.RemoveAll(path); err != nil {
		t.Errorf("remove the temporary profile: %v", err)
	}
})
```

- `t.Cleanup` functions run once the subtest function has returned or stopped through `t.Fatal` (`runtime.Goexit`), so the removal always runs and always after every check, including the check that follows `cleanup()`. It therefore cannot make "the cleanup did not remove the temporary profile" pass (FR-TEST-HYG-053, FR-TEST-HYG-054). It is registered on the subtest's `t`, not the parent's, so it runs when this subtest ends.
- It targets only `path`, the value returned by `profileOf("")`; `path` is not reassigned afterwards. An empty path removes nothing (FR-TEST-HYG-055). The system temporary folder itself is never removed even if a faulty `profile()` returned it; a `profile()` that does so already fails the location check (`temporary profile %q, want a new directory under %s`). No parent, glob or other path is touched.
- On a normal run the collector's cleanup has already removed `path`, and `os.RemoveAll` of a missing path returns `nil`, so the removal adds no failure and no log (FR-TEST-HYG-056). The `t.Errorf` reports only a real removal failure (for example a permission error), which would otherwise be a silent leak; the inner `err` shadows the outer one only inside the closure.
- Unchanged: the explicit `cleanup()` call, the `os.Stat` check after it and its message, the other checks and messages (`cleanup is nil`, `temporary profile %q, want a new directory under %s`, `the temporary profile was not created: %v`), the other subtests, `profileOf` and every helper. No import changes (`os`, `path/filepath` are already imported). `BrowserCollector.profile` in `internal/amazon/browser.go` is not edited, and neither is the source check of its body in the same test file.

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
| TS-TEST-HYG-071 | FR-TEST-HYG-055 | The removal returns at once when `path` is `""` or equals `filepath.Clean(os.TempDir())`; otherwise it removes only `path`, never its parent, a glob or another path. A removal error is reported with `t.Errorf("remove the temporary profile: %v", err)`; a missing path is not an error. |
| TS-TEST-HYG-072 | FR-TEST-HYG-056 | The diff outside `doc/` is `M internal/amazon/browser_profile_values_test.go`, limited to the `t.Cleanup` block in the subtest; no import, helper, other subtest or production file changes. The tests stage delivers the change and writes `no-tests.md`; the development stage writes no code. `gofmt -l internal/amazon` is empty; `go vet ./...` and `go test ./...` pass. |
| TS-TEST-HYG-073 | FR-TEST-HYG-053–055 (proof) | Runs and reverted fault injections 1–8 of the issue #324 design (fatal stop with a non-empty profile, with and without the removal; nil cleanup; error; regular file; cleanup that does nothing; empty path), each run with a fresh `TMPDIR`, recorded in `no-tests.md`. |

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

No production code, frontend, `API_SPECIFICATION.md`, `go.mod`, `go.sum`, `package.json`, `playwright.config.ts`, CI or `scripts/` change. Issue #182 changes one existing test file and adds one. Issues #232 and #244 change no existing test file other than `internal/amazon/amazontest/amazontest_test.go`, and issue #251 changes no test file other than `internal/amazon/amazontest/server_dependencies_test.go`. Issue #241 changes no test file. Issue #185 adds two test files and changes no existing one (except a `gofmt` fix if one is needed). Issue #238 changes no file outside `doc/`. Issue #181 changes no file outside `doc/`. Issue #295 changes one line of `tests/platform-tabs.spec.ts` and no other test file. Issue #195 changes `internal/amazon/browser_profile_test.go` and no other test file. Issue #312 changes only the `t.Run` loop of one test in that file. Issue #315 changes only that file. Issue #324 changes only one subtest of `internal/amazon/browser_profile_values_test.go`; `internal/amazon/browser.go` is edited only temporarily for fault injections and never committed. No test file is deleted.

## Test case mapping (issue #324)

No test is added: the tests stage edits the subtest and writes `no-tests.md`. Runs below refer to the issue #324 design.

| Test case | Proved by |
| --- | --- |
| TC-001 | Code check of the subtest "TC-003 temporary profile" in `internal/amazon/browser_profile_values_test.go`: `t.Cleanup` on the subtest's `t` right after `profileOf("")`, before the first `t.Fatal`; it returns on `""` or the system temporary folder and otherwise removes only `path`; `cleanup()`, the `os.Stat` check after it and every other check and message are unchanged; the other subtests are unchanged (TS-TEST-HYG-069–071) |
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
