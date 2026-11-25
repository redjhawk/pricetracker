# No red phase for issue #358

The change is test code. It adds a check to the per-owner loop of `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile` (`internal/httpapi/amazon_browser_interrupted_test.go`) and deletes the duplicate `internal/httpapi/amazon_browser_cache_log_test.go`. The check covers behaviour that has already shipped (FR-AMZ-HCHK-016: a check that opens the page logs no cache clear failure). So it passes on `master`'s production code, and there is nothing for the development stage to implement. The test file is locked after this stage, and FR-TEST-HYG-107 allows no other file outside `doc/`. So this stage makes the change itself (TS-TEST-HYG-140). Runs 2 and 3 show the red state. The development stage writes no code and only sets the statuses.

## Delivered

- `internal/httpapi/amazon_browser_interrupted_test.go`: the lines of the technical specification are added verbatim. `refreshOutput := captureCheckOutput(t)` goes after `api.clock.Set(at)` and before `RefreshItem`. After `waitStoredCheck` come the comment and the check:

  ```go
  		// FR-AMZ-HCHK-016: the Refresh price logs no cache clear failure, in either wording (issues #240 and #358).
  		if lines := linesWith(refreshOutput(), "Amazon browser cache could not be cleared"); len(lines) > 0 {
  			t.Fatalf("cache clear failure logged during Refresh price of %s:\n%s", id, strings.Join(lines, "\n"))
  		}
  ```

  `git diff master` of the file has one hunk (`@@ -58,11 +58,16 @@`) that adds only these 5 lines. The test name, the comments, the IDs, the owners, the fixture answers, the server-stop sequence and its guards, the server-stop cache check, the no-file, list and price checks and every message are unchanged. No import changes.
- `internal/httpapi/amazon_browser_cache_log_test.go` is listed in `delete.txt`. No other Go file uses `TestAmazonBrowserCacheLog_TC004_SuccessfulClearThroughTheApplicationLogsNothing` or `assertNoCacheLine`.

## Runs

Production code and the other test files were never edited. Writes outside the repository were refused, so each injection is an edited copy kept in this change folder's `inj/` directory and passed with `go test -overlay`. Every overlay also maps the duplicate to `""`, which stands for its deletion. After the runs, the `.go` copies were reduced to `package inj`, so `./...` still builds. All `inj/` files are listed in `delete.txt`. "The run" is `CI=1 go test -count=1 -run '^TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile$' -v ./internal/httpapi/`. "`master`'s test" means the same run with `git show master:internal/httpapi/amazon_browser_interrupted_test.go` as an overlay.

1. **No injection (TC-001).** The run passes: `--- PASS: TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile (2.97s)`. With `-count=2`, both runs pass. `CI=1 go test -count=1 ./internal/httpapi/` passes (`ok`, 127 s).
2. **Neutral line after every opened page (TC-002).** In `BrowserCollector.Collect`, right after `tab.load` returns without an error: `log.Printf("Amazon browser cache could not be cleared: test")`. The run fails at the first Refresh check, not at the server-stop check:

   ```text
   amazon_browser_interrupted_test.go:69: cache clear failure logged during Refresh price of B0INTR0051:
       2026/11/25 15:08:37 Amazon browser cache could not be cleared: test
   ```

   With `fmt.Println` instead of `log.Printf`, the run fails in the same place: `cache clear failure logged during Refresh price of B0INTR0051:` / `Amazon browser cache could not be cleared: test`. With the `log.Printf` injection, `master`'s test passes (`--- PASS`). It prints the line twice, once for each item, and fails on neither. That shows the gap this change closes.
3. **Old wording for the second item only (TC-003).** At the same place, only when `strings.Contains(item.URL, "B0INTR0052")`, the injection logs `Amazon browser cache could not be cleared after a page not opened: test`. The run fails at the second owner's check and does not name `B0INTR0051`:

   ```text
   amazon_browser_interrupted_test.go:69: cache clear failure logged during Refresh price of B0INTR0052:
       2026/11/25 15:08:53 Amazon browser cache could not be cleared after a page not opened: test
   ```

   The single prefix catches the old wording. No `after a page not opened` search exists.
4. **Existing failures unchanged (TC-004).** Each injection was run with the changed test and with `master`'s test:
   1. A `defer` at the top of `Collect` logs `Amazon browser cache could not be cleared: test` when `ctx.Err() != nil`. Both versions fail at line 52, the server-stop check, with `cache clear failure logged:` / `… Amazon browser cache could not be cleared: test`. Neither reports a Refresh-check message.
   2. In `newNavigationFixture` (`amazon_browser_navigation_test.go`), the `answerAttachmentHeld` branch returns right after its flush signal. In both versions of TC-005, the test waits up to 20 s on `stopped` before its "ended before the server stop" guard. Both fail at line 46 with `the check ended before the server stop: the interruption point was missed`.
5. **Code check (TC-005).** `git diff master -- internal/httpapi/amazon_browser_interrupted_test.go` is the single hunk described above. There are two captures, one per owner, each started before `RefreshItem` and read after `waitStoredCheck`. The server-stop capture `output` is read before the loop and is not reused.
6. **Deletion (TC-006).** Run with the deletion overlay. `go vet ./internal/httpapi/` passes. The `go test -list . ./internal/httpapi/` output differs from the list without the overlay only by the missing `TestAmazonBrowserCacheLog_TC004_SuccessfulClearThroughTheApplicationLogsNothing`, and it still holds `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile`. `grep` for `TestAmazonBrowserCacheLog_TC004\|assertNoCacheLine` in `*.go` finds only the file to delete.
7. **Gates (TC-008), run twice with the same result.** `gofmt -l internal/httpapi` prints nothing. `go vet ./...` passes. `CI=1 go test -count=1 ./...` passes, with every package `ok` and no `FAIL`. Before the deletion, `git diff --name-status master -- . ':!doc'` lists only `M internal/httpapi/amazon_browser_interrupted_test.go`. After the pipeline applies `delete.txt`, it also lists `D internal/httpapi/amazon_browser_cache_log_test.go`.

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile` with the per-owner check; run 1 |
| TC-002 | Run 2 |
| TC-003 | Run 3. Code check: only the prefix is searched |
| TC-004 | Run 4 |
| TC-005 | Run 5 (code check of the one-hunk diff) |
| TC-006 | Run 6, then on the committed tree by the development stage (`ls`, `grep`, `go test -list` without an overlay) |
| TC-007 | Document check of the technical specification stage's edits to the issue #240 `testcases.md` and the issue #240 TC-004 row of `doc/specifications/amazon-human-checks/technical.md` |
| TC-008 | Run 7. The development stage runs the gates again on the committed tree |
