# Test suite hygiene: functional specification

Status: FR-TEST-HYG-001–003 implemented (issue #208); FR-TEST-HYG-004–008 implemented (issue #232); FR-TEST-HYG-009–013 implemented (issue #244); FR-TEST-HYG-014–017 implemented (issue #251); FR-TEST-HYG-018, 020 and 021 implemented (issue #241); FR-TEST-HYG-019 closed by the merge of the issue #238 change (issue #241); FR-TEST-HYG-022–026 implemented (issue #185); FR-TEST-HYG-027–029 implemented (issue #238, whose merge also settles FR-TEST-HYG-019); FR-TEST-HYG-030–032 implemented (issue #181); FR-TEST-HYG-033–036 specified (issue #182)
Source: issue #208 (from review comment R1-03 on PR #206, `doc/todo/issue-204-r1-03.md`); issue #232 ("Make AssertNoDownload fail on unreadable files", from review comment R1-02 on PR #231, `doc/todo/issue-217-r1-02.md`); issue #244 ("Replace the TC-006 import scan in amazontest_test.go with a dependency check", from review comment R1-01 on PR #243, `doc/todo/issue-232-r1-01.md`); issue #251 ("Make the TC-003 fixture modules compile", from review comment R2-01 on PR #247, `doc/todo/issue-244-r2-01.md`); issue #241 ("Close the finished TC-005 interruption-check todo (#238)", from review comment R3-01 on PR #236, `doc/todo/issue-233-r3-01.md`); issue #185 ("Tests missing from #168 (Amazon human-like checks)", from review comments R1-01, R1-03 and R4-01 on PR #179); issue #238 ("TC-005: fail if the check ended before the server stop", from review comment R1-03 on PR #236, `doc/todo/issue-233-r1-03.md`); issue #181 ("Correct TC-006 to use a non-404/410 HTTP status", from review comment R1-04 on PR #179, `doc/todo/issue-168-r1-04.md`); issue #182 ("Make the TC-032 Refresh price subtest reach the in-flight branch and test restart after stop", from review comment R3-02 on PR #179, `doc/todo/issue-168-r3-02.md`).
Scope: repository maintenance. Application behaviour is not affected.

## Purpose

- Issue #208: remove the empty leftover test file `internal/service/zz_scratch_docs_test.go`, left by the issue #204 test stage. It holds only a package clause and a comment saying it is safe to delete, tests nothing, and adds noise to the `service` package.
- Issue #232: make the test check "no file left by a download" (`AssertNoDownload` in `internal/amazon/amazontest`) fail closed. Today a file it cannot read is taken as "not a download", even if it holds the downloaded body, so the tests that prove FR-AMZ-HCHK-014 and FR-AMZ-HCHK-015 ([Amazon human-like checks](../amazon-human-checks/functional.md)) can pass when they should not.
- Issue #244: the guarantee "the server never uses the test helper package" is checked today by a text search in the helper's own unit tests (`TestAssertNoDownload_TC006_ChangeLeavesApplicationUntouched`). The search fails on any non-test Go file under `cmd/` or `internal/` that names the helper, even a future test-support package the server never uses, and it does not show what the server binary really depends on. Replace it with a check of the server's actual dependencies.
- Issue #251: the unit tests of the dependency check build small fixture modules (TC-003 of issue #244). Some fixtures that stand for valid code use a name (`DownloadMarker`) that the fixture's stub of the test helper does not declare, so those modules would not compile. The tests pass only because listing dependencies does not compile the code; they would break, or prove less than they claim, if the listing ever compiled it. Make the fixtures that stand for valid code compile.
- Issue #241: the follow-up record `doc/todo/issue-233-r1-03.md` and its GitHub issue #238 still say "todo" for work that PR #236 already did: TC-005 (`internal/httpapi/amazon_browser_interrupted_test.go`) fails with a message when the check ended before the server stop. Close the record and the issue so they no longer list finished work.
- Issue #185: three behaviours of [Amazon human-like checks](../amazon-human-checks/functional.md) (issue #168) shipped without a test that proves them: a scheduled Amazon check whose window has passed is not sent (FR-AMZ-HCHK-004, FR-AMZ-HCHK-009), every Amazon item gets its own window time at the first start after the upgrade (FR-AMZ-HCHK-004, FR-AMZ-HCHK-005, FR-AMZ-HCHK-010), and an empty-body error response is classified by its HTTP status (FR-AMZ-HCHK-012). Add these tests, and keep `internal/amazon/browser_test.go` formatted with `gofmt`.
- Issue #238: TC-005 (`internal/httpapi/amazon_browser_interrupted_test.go`) must fail with a message when the check has already ended at the moment the server is stopped, so that a faster check cannot turn the server stop into a no-op that still passes. PR #236 already added this guard (see issue #241), so issue #238 keeps it and is closed as completed by its own change.
- Issue #181: in the issue #168 test case document, TC-006 lists "HTTP 404 for the product page" as a failure, while [Amazon human-like checks](../amazon-human-checks/functional.md) (FR-AMZ-HCHK-012) and the shipped behaviour treat 404 and 410 as "listing unavailable", and the TC-006 test in `internal/amazon/browser_test.go` uses 403. Correct the document so that it matches the specification and the tests.
- Issue #182: two scheduling behaviours of [Amazon human-like checks](../amazon-human-checks/functional.md) are not proven by a test. (1) In TC-032 (`internal/service/amazon_human_checks_test.go`, subtest "Refresh price running"), the Refresh price has already moved the item's overdue due time to the evening window when the scheduled checks run, so the item is never due during that run and the subtest does not test a scheduled check that falls due while a Refresh price is running. (2) No test covers an item stopped at its morning scheduled time and restarted by a Refresh price later in the same window: the scheduled checks must not check it again in that window (FR-AMZ-HCHK-008, FR-AMZ-STOP-004, FR-AMZ-STOP-008). Make the subtest reach the due case and add the missing test.

## Actors

- Developer or agent working on the repository, who runs the test suite and reads its results.

## Terms

- **No-download check:** the test helper `AssertNoDownload`, which looks through given folders (the Amazon browser profile, the home folder of the account running the browser) after the browser has closed and fails the test when a file left by a download remains.
- **Content rule:** the part of the no-download check that reads a file to see whether it holds the downloaded body (`DownloadMarker`). It applies to a regular file whose name does not already identify it as a download.
- **Test helper package:** the Go package `pricefollower.local/internal/amazon/amazontest`, which holds the no-download check.
- **Server dependencies:** the Go packages that a program under `cmd/` is built from, directly or indirectly, as the Go tool reports them for a normal (non-test) build. Test files (`_test.go`) are not part of them.
- **Dependency check:** the repository check of issue #244 that fails when the test helper package is one of the server dependencies.
- **Fixture module:** a small temporary Go module, with its own stub of the test helper package, that the unit tests of the dependency check build to run the check on a known set of imports.
- **Follow-up record:** a file under `doc/todo/` that records a non-blocking review comment and the GitHub issue opened for it.
- **Morning window, evening window, scheduled check, failure:** as defined in [Amazon human-like checks](../amazon-human-checks/functional.md) (06:00–09:00 and 19:00–23:00 in the schedule timezone).
- **Due while running:** a scheduled check of an Amazon item whose due time is a past time of the current window when the scheduled checks run, while a check of the same item (Refresh price or first check after add) is still in progress.
- **Upgrade redraw mark:** the mark the database records at the first start after the issue #168 change when Amazon items still hold the check time written before that change (`amazon_request_state.human_checks_redraw = 1`); it asks the application to draw a window time for every Amazon item at its start.
- **Issue #168 test case document:** `doc/changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n/testcases.md`, the test cases written for issue #168, including TC-004 (browser results match the current parsing results) and TC-006 (non-success response and challenge page are failures).
- **Valid fixture:** a fixture module that stands for code the Go tool accepts: every fixture of the dependency check's unit tests except those whose purpose is to make the listing fail (a program importing a package that does not exist, a program with a syntax error).

## Scope and exclusions

- In scope (issue #208): deleting `internal/service/zz_scratch_docs_test.go`.
- In scope (issue #232): what the no-download check does when a file cannot be read.
- In scope (issue #244): removing the text search for the test helper's import path from the helper's unit tests, and adding the dependency check in its place.
- Excluded: production code, API, database and user interface changes.
- Excluded (issue #244): the issue #208 check in the same unit test that the leftover scratch file stays absent, which stays as it is, and the no-download check itself.
- In scope (issue #251): the fixture modules used by the unit tests of the dependency check, and their stub of the test helper package.
- Excluded (issue #251): the dependency check itself, its failure messages and when it passes or fails (FR-TEST-HYG-009–013 are unchanged); the real test helper package; the fixtures whose purpose is to make the listing fail, which stay invalid.
- In scope (issue #241): the follow-up record `doc/todo/issue-233-r1-03.md` and GitHub issue #238.
- Excluded (issue #241): any change to TC-005 or to other test or production code, to the issue #233 technical specification and test cases, and to other follow-up records, including `doc/todo/issue-233-r3-01.md` (the record of issue #241 itself, which stays as the trace of this issue, as the follow-up records of earlier done issues do).
- In scope (issue #238): the "check still running" guard of TC-005 and the closing of GitHub issue #238.
- Excluded (issue #238): production code, the rest of TC-005 and other tests, the issue #233 technical specification and test cases, and the follow-up records under `doc/todo/` (`doc/todo/issue-233-r1-03.md` was already deleted by issue #241).
- In scope (issue #185): new or extended automated tests for the three behaviours above, and the `gofmt` formatting of `internal/amazon/browser_test.go`.
- Excluded (issue #185): any change to the application's behaviour, including the window rules, the upgrade restart and redraw, and the classification of HTTP statuses. If a new test shows that the application does not behave as FR-AMZ-HCHK-004, -009, -010 or -012 and the requirements below state, that is a defect to report, not something this issue changes.
- In scope (issue #181): the TC-006 and TC-004 entries of the issue #168 test case document.
- Excluded (issue #181): test and production code, including `internal/amazon/browser_test.go`; the other test cases and the other files of the issue #168 change folder, including its `tests.lock`; the [Amazon human-like checks](../amazon-human-checks/functional.md) specifications; and the follow-up record `doc/todo/issue-168-r1-04.md`, which the issue does not ask to change. The classification of HTTP statuses stays as FR-AMZ-HCHK-012 states.
- In scope (issue #182): the "Refresh price running" subtest of TC-032 and a new automated test of the restart after stop, both in Go test files of `internal/service/`.
- Excluded (issue #182): any change to the application's behaviour, including the scheduling of Amazon checks, the stop and restart rules and Refresh price; the "first check after add running" subtest of TC-032, which already covers its case; the issue #168 test case document and change folder; and the follow-up record `doc/todo/issue-168-r3-02.md`, which the issue does not ask to change. If a test shows that the application does not behave as the requirements below state, that is a defect to report, not something this issue changes.
- Excluded: the other rules of the no-download check (which names and contents count as a download, missing folders, symbolic links), which stay as they are.
- Excluded: the issue #204 change records under `doc/changes/2026-11-22-2139-issue-204-retrieval-error/`. The factory checks a `tests.lock` only for the issue it is running, and issue #204 is merged, so its lock is no longer checked and stays unchanged as the historical record of that issue.

## Requirements

### Leftover scratch file (issue #208)

- **FR-TEST-HYG-001** The file `internal/service/zz_scratch_docs_test.go` no longer exists in the repository.
- **FR-TEST-HYG-002** After the removal, `go vet ./...` and `go test ./...` pass, including `go test ./internal/service/...`.
- **FR-TEST-HYG-003** No other test file is deleted or changed by issue #208, and the operator observes no change in the application.

### No-download check fails on unreadable files (issue #232)

- **FR-TEST-HYG-004** When the content rule meets a file under a checked folder that cannot be read (for example permission denied or an I/O error), the no-download check fails the test. The file is never treated as free of a downloaded body.
- **FR-TEST-HYG-005** The failure message of FR-TEST-HYG-004 names the path of the file that could not be read and the read error.
- **FR-TEST-HYG-006** A file that is listed but no longer exists when the check reads it (it disappeared in between) is ignored: it does not fail the test and is not reported as a download.
- **FR-TEST-HYG-007** Readable files are judged as before: a file whose name identifies it as a download (attachment name, uniquified attachment name, `download` prefix, partial-download name) is reported as a download whatever its content, a readable file holding `DownloadMarker` is reported as a download, and any other readable file passes. Missing folders are still ignored and symbolic links are still not followed.
- **FR-TEST-HYG-008** The tests that use the no-download check still pass on a normal run, `go vet ./...` and `go test ./...` pass, and the operator observes no change in the application.

### Server dependency check (issue #244)

- **FR-TEST-HYG-009** The unit tests of the test helper package no longer search the source text of `cmd/` and `internal/` for the test helper's import path. A non-test Go file that imports the test helper package and is not a server dependency (for example a future test-support package such as `internal/httpapi/httpapitest`) makes no test fail.
- **FR-TEST-HYG-010** The dependency check fails when the test helper package is among the server dependencies of any program under `cmd/`, whether that program imports it directly or through another package.
- **FR-TEST-HYG-011** The failure message of FR-TEST-HYG-010 names the import path `pricefollower.local/internal/amazon/amazontest` and says that the server depends on it.
- **FR-TEST-HYG-012** The dependency check passes when the test helper package is imported only by test files (`_test.go`) or by packages that are not server dependencies. When the server dependencies cannot be listed, the check fails with a message holding the Go tool's error; it never passes without having listed them.
- **FR-TEST-HYG-013** The dependency check runs automatically in the existing unit-test gate (`go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`) on every run, with no separate command to remember. On the repository as changed it passes, `go vet ./...` and `go test ./...` pass, the issue #208 scratch-file check still runs, and the operator observes no change in the application.

### Fixture modules compile (issue #251)

- **FR-TEST-HYG-014** Every valid fixture compiles: building all its packages with the Go tool (`go build ./...` run in the fixture module, offline) succeeds.
- **FR-TEST-HYG-015** Each valid fixture keeps the case it is named after: a program importing the test helper directly, a program importing it through another package, a second program being the only one to depend on it, only test files importing it, a test-support package no program imports using it, no import of it at all, a blank import of it in a program, and a renamed import of it in a server dependency. Each case gives the same dependency check result as before (fails as a server dependency, or passes).
- **FR-TEST-HYG-016** The fixtures whose purpose is to make the listing fail (a missing package, a syntax error) and the case where the Go tool cannot be started keep failing the dependency check with the same messages as before.
- **FR-TEST-HYG-017** On the repository as changed, `go vet ./...` and `go test ./...` pass, the diff outside `doc/` is limited to the test code of `internal/amazon/amazontest/`, and the operator observes no change in the application.

### Finished TC-005 follow-up closed (issue #241)

- **FR-TEST-HYG-018** The follow-up record `doc/todo/issue-233-r1-03.md` no longer exists in the repository.
- **FR-TEST-HYG-019** GitHub issue #238 is closed with the reason "completed" (not "not planned"), as the work it asks for was done by PR #236. The merge of the issue #238 change closes it (FR-TEST-HYG-029).
- **FR-TEST-HYG-020** TC-005 still fails the test, with the message "the check ended before the server stop: the interruption point was missed", when the check has already ended at the moment the server is stopped; `internal/httpapi/amazon_browser_interrupted_test.go` is unchanged.
- **FR-TEST-HYG-021** No other file under `doc/todo/` is deleted or changed, the change's diff outside `doc/` is empty, `go vet ./...` and `go test ./...` pass, and the operator observes no change in the application.

### Missing tests of the Amazon human-like checks (issue #185)

- **FR-TEST-HYG-022** (R1-01) An automated test covers a tracked, not stopped Amazon item whose scheduled check was due at a time in the morning window, when the scheduled checks run at a time in the evening window of the same day (the application was not running in between). The test passes only when, in that run: no Amazon request is sent for the item; the item's next scheduled check is a time within the next window that begins after the run time; and the server log has a line saying that the scheduled Amazon check of that item missed its window (`scheduled Amazon check of item <id> missed its window; next check <time>`), naming that next time. It fails if the item is checked in that run, keeps its past due time, or no such log line is written.
- **FR-TEST-HYG-023** (R1-03) An automated test covers the first start of the application on a database holding the upgrade redraw mark and several tracked Amazon items that all hold the same check time from before the issue #168 change, chosen outside the next window that begins after the start time. The test passes only when, after the start: every one of these Amazon items has a next scheduled check within the next window that begins after the start time, so none keeps the shared earlier time; the upgrade redraw mark is cleared; and a second start with the mark cleared leaves the Amazon items' next scheduled checks unchanged. It fails if any Amazon item keeps the earlier time or if the mark is still set after the start.
- **FR-TEST-HYG-024** (R4-01) An automated test checks an Amazon item through the real browser while the product page is answered with an HTTP error status and an empty body, once for each of 404, 410, 503 and 429:
  - 404 and 410: the check result is "listing unavailable" (an Amazon answer, not a failure), and the check records that HTTP status.
  - 503 and 429: the check is a failure (request error, second-hand check error), it records that HTTP status, and its failure cause, written to the server log, is `Amazon returned HTTP <status>` (for example `Amazon returned HTTP 429`).
  The test fails if any of the four statuses gives another result, records another HTTP status, or (503, 429) gives another failure cause.
- **FR-TEST-HYG-025** `gofmt -l internal/amazon` lists no file: `internal/amazon/browser_test.go` and every test file added or changed by issue #185 are formatted with `gofmt`.
- **FR-TEST-HYG-026** The tests of FR-TEST-HYG-022–024 run in the existing unit-test gate (`go vet ./...`, `go test ./...`) with no separate command, and pass on the repository as changed. The change's diff outside `doc/` is limited to Go test files (`*_test.go`), and the operator observes no change in the application.

### TC-005 fails when the interruption point is missed (issue #238)

- **FR-TEST-HYG-027** After the product page has been answered with the download and before the server is stopped, TC-005 checks whether the check has already ended. If it has, the test fails at once, without stopping the server, with the message "the check ended before the server stop: the interruption point was missed".
- **FR-TEST-HYG-028** When the check is still running at that moment, TC-005 stops the server and goes on as before: the stopped check returns within 20 s, no download file and no new data file are written, and the next Refresh price of each owner records the normal page. On a normal run it passes.
- **FR-TEST-HYG-029** Merging the issue #238 change closes GitHub issue #238 as completed. The change touches no production code: its diff outside `doc/` is either empty or limited to `internal/httpapi/amazon_browser_interrupted_test.go`. `go vet ./...` and `go test ./...` pass, and the operator observes no change in the application.

### TC-006 failure statuses match the specification (issue #181)

- **FR-TEST-HYG-030** In the issue #168 test case document, TC-006 lists as its failure cases HTTP 503, HTTP 403 for the product page, and HTTP 200 with an Amazon robot-check (captcha) page. TC-006 no longer lists HTTP 404 or HTTP 410 as a failure. Its other lines (ID, title, covered requirements, level, expected result, corner cases) are unchanged.
- **FR-TEST-HYG-031** In the same document, the corner cases of TC-004 include a product page answered with HTTP 404 or HTTP 410, whose result is "listing unavailable" (an Amazon answer, not a failure). The other lines of TC-004 and its existing corner cases are unchanged.
- **FR-TEST-HYG-032** The change touches no other test case of that document and no other file of the issue #168 change folder. Its diff outside `doc/` is empty and nothing under `doc/todo/` changes. `go vet ./...` and `go test ./...` pass, and the operator observes no change in the application.

### Scheduled checks around a Refresh price (issue #182)

- **FR-TEST-HYG-033** The TC-032 subtest "Refresh price running" covers a check that is due while running. A tracked, not stopped Amazon item gets a Refresh price in the morning window. While that check is still in progress (its Amazon request has started and has not returned), the item's due time is set to a past time of the morning window. Then the scheduled checks run, and then the Refresh price is allowed to finish. The subtest passes only when, at the end, exactly one Amazon check was made for the item (the Refresh price) and the item's next scheduled check is a time within a window that begins after the scheduled checks' run time. It fails if the scheduled checks make a second check of the item, or if the item keeps a past due time.
- **FR-TEST-HYG-034** The other subtest of TC-032 ("first check after add running") keeps its case and its pass condition: exactly one check of the added item and a next scheduled check in a later window.
- **FR-TEST-HYG-035** (R2-01 scenario) An automated test covers a tracked Amazon item whose scheduled check was due at 07:00 (morning window) and that is stopped at that time. At 07:30 the same day, a Refresh price of the item gets an Amazon answer. At 07:31 the scheduled checks run. The test passes only when all of these hold: the Refresh price restarts the item (it is no longer stopped); exactly one Amazon check of the item is made in total, the Refresh price, so the 07:31 run makes none; and after the 07:31 run, the item's next scheduled check is a time within the evening window (19:00–23:00) of the same day. It fails if the 07:31 run checks the item, if the item stays stopped, or if its next scheduled check is not in that evening window.
- **FR-TEST-HYG-036** The tests of FR-TEST-HYG-033–035 run in the existing unit-test gate (`go vet ./...`, `go test ./...`) with no separate command and pass on the repository as changed. `gofmt -l internal/service` lists no file. The change's diff outside `doc/` is limited to Go test files (`*_test.go`) of `internal/service/`, and the operator observes no change in the application.

## States

Not applicable: there is no user interface, loading, empty or error state. The only observable outputs are test results and test failure messages.

## Acceptance criteria

Issue #208:

- `ls internal/service/zz_scratch_docs_test.go` reports that the file does not exist.
- `go vet ./...` and `go test ./...` succeed.
- The change's diff outside `doc/` is limited to the deletion of `internal/service/zz_scratch_docs_test.go`.

Issue #232:

- With a checked folder holding a regular file that does not match any download name and cannot be read, the no-download check fails the test with a message holding that file's path and the read error (FR-TEST-HYG-004, FR-TEST-HYG-005).
- With a checked folder holding a file that is removed after the folder is listed and before it is read, the no-download check does not fail the test (FR-TEST-HYG-006).
- With readable files only, the check gives the same result as before for each rule of FR-TEST-HYG-007: a file named after the attachment fails, a file holding `DownloadMarker` fails, a missing folder and an unrelated readable file pass.
- `go vet ./...` and `go test ./...` succeed, and the change's diff outside `doc/` is limited to test code (FR-TEST-HYG-008).

Issue #244:

- No unit test of the test helper package reads the Go source files of `cmd/` or `internal/` looking for the import path `pricefollower.local/internal/amazon/amazontest` (FR-TEST-HYG-009).
- With a temporary non-test file importing the test helper package in a package the server does not depend on, the unit-test gate passes (FR-TEST-HYG-009, FR-TEST-HYG-012).
- With a temporary non-test import of the test helper package in a server dependency (for example a file of `cmd/pricefollower`), the unit-test gate fails, and the message names `pricefollower.local/internal/amazon/amazontest` (FR-TEST-HYG-010, FR-TEST-HYG-011, FR-TEST-HYG-013).
- When the Go tool cannot list the server dependencies, the dependency check fails with the tool's error (FR-TEST-HYG-012).
- On the repository as changed, `go vet ./...` and `go test ./...` succeed, and the diff outside `doc/` changes no production code (FR-TEST-HYG-013).

Issue #251:

- For each valid fixture of the dependency check's unit tests, writing the fixture module to a temporary folder and running `go build ./...` in it offline (`GOFLAGS=-mod=mod GOWORK=off GOTOOLCHAIN=local GOPROXY=off`) succeeds (FR-TEST-HYG-014).
- The unit tests still hold one case per situation listed in FR-TEST-HYG-015, including a renamed import of the test helper in a server dependency, and each gives the same pass or fail result as before (FR-TEST-HYG-015).
- The missing-package, syntax-error and go-command-not-found cases still fail with a message holding, respectively, the missing import path, the file name `main.go`, and the start error (FR-TEST-HYG-016).
- `go vet ./...` and `go test ./...` succeed, and the diff outside `doc/` touches only `internal/amazon/amazontest/*_test.go` (FR-TEST-HYG-017).

Issue #241:

- `ls doc/todo/issue-233-r1-03.md` reports that the file does not exist (FR-TEST-HYG-018).
- `gh issue view 238 --json state,stateReason` shows `CLOSED` and `COMPLETED` (FR-TEST-HYG-019).
- `git diff origin/master -- internal/httpapi/amazon_browser_interrupted_test.go` is empty, and the file still holds the check on `stopped` before `cancel()` with the message of FR-TEST-HYG-020.
- The change's diff touches only `doc/` and, under `doc/todo/`, only deletes `doc/todo/issue-233-r1-03.md`; `go vet ./...` and `go test ./...` succeed (FR-TEST-HYG-021).

Issue #185:

- `go test ./...` runs a test for each of FR-TEST-HYG-022, -023 and -024, and they pass (FR-TEST-HYG-026).
- Temporarily making the scheduled checks send a missed Amazon check instead of skipping it makes the FR-TEST-HYG-022 test fail (FR-TEST-HYG-022).
- Temporarily making the application ignore the upgrade redraw mark at start makes the FR-TEST-HYG-023 test fail (FR-TEST-HYG-023).
- The FR-TEST-HYG-024 test has a case for each of 404, 410, 503 and 429 with an empty body, and each case asserts the result and the recorded HTTP status; the 503 and 429 cases also assert the failure cause `Amazon returned HTTP <status>` (FR-TEST-HYG-024).
- `gofmt -l internal/amazon` prints nothing (FR-TEST-HYG-025).
- `go vet ./...` succeeds, and `git diff --name-only origin/master` outside `doc/` lists only `*_test.go` files (FR-TEST-HYG-026).

Issue #238:

- Between the wait for the download answer and `cancel()`, `internal/httpapi/amazon_browser_interrupted_test.go` checks whether the check goroutine has ended and fails with the message of FR-TEST-HYG-027 if it has (FR-TEST-HYG-027).
- If the check is temporarily made to end before the server stop (for example, the collector returns as soon as the download answer is sent), TC-005 fails with that message. Without the temporary change, TC-005 passes (FR-TEST-HYG-027, FR-TEST-HYG-028).
- After the merge, `gh issue view 238 --json state,stateReason` shows `CLOSED` and `COMPLETED` (FR-TEST-HYG-019, FR-TEST-HYG-029).
- Outside `doc/`, `git diff --name-only origin/master` lists nothing or only `internal/httpapi/amazon_browser_interrupted_test.go`. `go vet ./...` and `go test ./...` succeed (FR-TEST-HYG-029).

Issue #181:

- TC-006 in the issue #168 test case document names HTTP 503, HTTP 403 and the robot-check page as its failure cases and contains neither "404" nor "410" (FR-TEST-HYG-030).
- TC-004 in that document has a corner case stating that a product page answered with HTTP 404 or 410 is "listing unavailable", not a failure (FR-TEST-HYG-031).
- In `git diff origin/master`, the only change to the issue #168 change folder is in the TC-004 and TC-006 entries of `testcases.md`; nothing outside `doc/` and nothing under `doc/todo/` is changed; `go vet ./...` and `go test ./...` succeed (FR-TEST-HYG-032).

Issue #182:

- In the TC-032 subtest "Refresh price running", the scheduled checks run while the Refresh price is still blocked, after the item's due time has been set to a past time of the morning window. The subtest asserts one check and a next scheduled check in a later window (FR-TEST-HYG-033).
- The TC-032 subtest "first check after add running" keeps its case and assertions (FR-TEST-HYG-034).
- `go test ./internal/service/...` runs a test for the scenario of FR-TEST-HYG-035, and it passes. Temporarily making Refresh price leave an overdue due time of an Amazon item unchanged makes that test fail (FR-TEST-HYG-035).
- `go vet ./...` and `go test ./...` succeed, `gofmt -l internal/service` prints nothing, and `git diff --name-only origin/master` outside `doc/` lists only `internal/service/*_test.go` files (FR-TEST-HYG-036).

## Open questions

None. Issue #232 states the expected behaviour: fail on a read error, keep ignoring files that disappear between listing and reading. Issue #244's title asks to replace the text search with a dependency check, so the guarantee is kept. The check runs in the existing unit-test gate so that a regression is caught without a manual step. Issue #251 states the expected outcome (the fixture modules compile); whether the stub declares the missing name or the fixtures stop using it is a technical choice. Issue #241 accepts either deleting the record or marking it done; deletion is chosen, as no follow-up record uses a "done" status and the Git history keeps the file. Issue #185 needed no question: it states each scenario and its expected result, which match FR-AMZ-HCHK-004, -009, -010 and -012 and the behaviour shipped by PR #179; whether a case extends an existing test or is a new test is a technical choice. Issue #238 states the guard and its expected failure. PR #236 already added the guard, so the change keeps it and closes the issue as completed, which also settles FR-TEST-HYG-019. Issue #181 needed no question: it states both corrections (403 instead of 404 in TC-006, a 404/410 corner case in TC-004), and both match FR-AMZ-HCHK-012 and the existing tests. Issue #182 needed no question: it states both scenarios and their expected results, which match FR-AMZ-HCHK-007, FR-AMZ-HCHK-008, FR-AMZ-STOP-004–008 and the behaviour shipped by PR #179. Whether the restart scenario is a new test or a new subtest is a technical choice.
