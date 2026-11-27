# No red phase for issue #359

The change only deletes test code: `TestBrowserCacheLog_TC003_SuccessfulClearLogsNothing` and its two comment lines from `internal/amazon/browser_cache_log_test.go`. That file is locked after this stage, and FR-TEST-HYG-102 allows no other file outside `doc/`, so this stage makes the deletion itself, as the technical specification says (TS-TEST-HYG-134). The deleted test passes on `master`, so no test can fail first on a missing implementation. The red gate is skipped. The development stage writes no code: it only sets the statuses.

## Delivered in `internal/amazon/browser_cache_log_test.go`

Removed lines only (`git diff master` of the file: 45 deletions, no added line): lines 168–212 of `master`, that is the blank line after `TestBrowserCacheLog_TC002_PageStillRequested`, the comments `// A successful clear logs no cache line, whatever started it.` and `// TC-003`, and the whole function.

The file now ends with the closing `}` of the TC-002 test and one newline. The header comment, the constants, `logMessage`, `assertNeutralCacheLine`, `assertNoOldCacheWording`, `assertBrowserClosed`, TC-001 and TC-002 and the imports `net/http` (`http.StatusOK` in TC-001), `regexp`, `strings`, `testing`, `time` are unchanged and all still used. The deleted test declared no package-level name, so nothing else was removed and no blank import or dummy use was added. `browser_interrupted_test.go` and its helpers are unchanged.

## Runs

`S` is `go test -count=1 -run '^(TestBrowserInterrupted_TC001_ClearConfirmedBeforeClose|TestBrowserInterrupted_TC002_PageStillRequestedIsCleared|TestBrowserInterrupted_TC006_UninterruptedCheckIsUnchanged)$' -v ./internal/amazon/`. The fault injections were temporary edits of `clearCache` in `internal/amazon/browser.go`, each reverted before the next step; `git diff master -- internal/amazon/browser.go` is empty after the runs.

1. Before the deletion, `go test -count=1 -run '^TestBrowserCacheLog_' -v ./internal/amazon/` passes in 34.3 s:
   - `TC001_PageNotOpened`: `live check, error answer`, `live check, no answer`, `interrupted after the offer, error answer`, `repeated`;
   - `TC002_PageStillRequested`: `time limit, error answer`, `time limit, no answer`, `server stop, error answer`, `server stop, no answer`;
   - `TC003_SuccessfulClearLogsNothing` (9.26 s): `live check, page not opened, repeated` and `interrupted after the offer` / `page still requested` for `time limit` and `server stop` (about 2.3 s each).
2. After the deletion, the same command passes in 25.1 s, about 9 s shorter. Only TC-001 and TC-002 run, with the same eight subtest names as in run 1; no TC-003 test or subtest. `-count=2` passes (50.1 s). `go test -count=1 -run 'TestBrowserCacheLog_TC003' -v ./internal/amazon/` prints `testing: warning: no tests to run` and passes.
3. `S` passes (16.2 s). Its `-v` output lists `TC001_ClearConfirmedBeforeClose/time limit` and `/server stop`, `TC002_PageStillRequestedIsCleared/navigation held, time limit`, `/navigation held, server stop` and `/late navigation answer, server stop`, and `TC006_UninterruptedCheckIsUnchanged/download` (and `/error page`).
4. Injection A: in `clearCache`, `log.Printf("Amazon browser cache could not be cleared: %v", err)` moved before the `if err != nil` (so logged also on success, as `...: <nil>`). `S` FAILS: both TC-001 subtests, the three TC-002 subtests and TC-006 `download` fail with `cache clear failure logged:` (`browser_interrupted_test.go:156`, `:192`, `:382`); TC-006 `error page` passes. Reverted.
5. Injection B: `if err == nil { log.Print("Amazon browser cache could not be cleared after a page not opened") }` added after the existing `if`. `S` FAILS in the same six subtests with the same message, the logged line being `... Amazon browser cache could not be cleared after a page not opened`. So `assertNoCacheLogLine` also rejects the old wording, and the deleted `assertNoOldCacheWording` calls were redundant. Reverted; `S` passes again (16.2 s).
6. Repository checks and gates:
   - A search of every `*.go` file finds no `TestBrowserCacheLog_TC003` and none of the deleted subtest names (`page not opened, repeated`, `"interrupted after the offer"`, `"page still requested"`). The test file contains no `// TC-003`, no `Clear: "answer"` and no "A successful clear logs no cache line".
   - `git diff master -- internal/amazon/browser.go internal/amazon/browser_interrupted_test.go doc/changes/2026-11-25-1255-issue-240-neutral-wording-for-the-cache-clear-fail/ doc/specifications/amazon-human-checks/` is empty.
   - `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/browser_cache_log_test.go`.
   - `gofmt -l internal/amazon` prints nothing. `go vet ./...` passes. `go test -count=1 ./...` passes with every package `ok`, twice.

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | Run 6 searches; run 2: the TC-003 name matches no test |
| TC-002 | Runs 1 and 2 (same subtest names, `-count=1` and `-count=2`, about 9 s shorter) |
| TC-003 | Code check of the imports and declarations (all still used, no blank import); `go vet ./internal/amazon/` and `go vet ./...` pass; `browser_interrupted_test.go` unchanged |
| TC-004 | `git diff master` of the file: removed lines 168–212 only, no added line; `gofmt -l` is empty |
| TC-005 | Run 3 |
| TC-006 | Runs 4 and 5 |
| TC-007 | Run 6: the diffs of `browser_interrupted_test.go`, the issue #240 change folder and `doc/specifications/amazon-human-checks/` are empty |
| TC-008 | Run 6. The development stage runs the gates again |
