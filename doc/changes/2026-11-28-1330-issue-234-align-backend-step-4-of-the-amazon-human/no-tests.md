# No new automated tests for issue #234

This change is documentation only. It inserts `or none (a response may come after the 1 s wait),` into step 4 of the Backend `Collect` steps in `doc/specifications/amazon-human-checks/technical.md` (TS-AMZ-HCHK-022). The technical specification states that no Go file, frontend file, API, database or user interface changes, and `browserTab.load` (`internal/amazon/browser.go`) already behaves as the new wording says.

No Go or Playwright test can fail before the edit and pass after it: no test reads the specification documents, and the application behaviour is unchanged. A test that read a Markdown file would test documentation, not the application. The test cases are therefore checks run by the gates and the reviewer, as `testcases.md` defines them:

| Test case | How it is checked |
| --- | --- |
| TC-001 | `grep -nF 'with a 2xx response or none (a response may come after the 1 s wait), the HTTP cache is cleared first (issue #217, TS-AMZ-HCHK-017)' doc/specifications/amazon-human-checks/technical.md` matches exactly one line, step 4. On the current branch it matches none (step 4 is at line 76 and still holds the old clause), so the check fails before the implementation. |
| TC-002 | `grep -cF 'with a 2xx response the HTTP cache is cleared first'` on `technical.md` gives 0; step 4 still contains `also when the check is interrupted` and `clears it too (issue #233, TS-AMZ-HCHK-018–019)`; `git diff --word-diff master -- doc/specifications/amazon-human-checks/technical.md` shows only the insertion in step 4. On the current branch the count is 1, so the check fails before the implementation. |
| TC-003 | Review: step 4, the TS-AMZ-HCHK-017 row, its Backend bullet and the `errorText` branch of `browserTab.load` give the same rule. |
| TC-004 | `git diff --name-only master -- doc/` lists only `doc/FUNCTIONAL_SPECIFICATIONS.md`, the two amazon-human-checks specification files and this change folder. |
| TC-005 | `git diff --name-only master -- . ':!doc'` prints nothing. |
| TC-006 | The existing suites pass unchanged. Run in this stage: `go vet ./...` passed and `go test ./internal/amazon/` passed (`ok pricefollower.local/internal/amazon 171.579s`). The existing tests that prove the behaviour step 4 describes exist and are unchanged: `TestBrowserInterrupted_TC001_ClearConfirmedBeforeClose` and `TestBrowserInterrupted_TC006_UninterruptedCheckIsUnchanged` (`internal/amazon/browser_interrupted_test.go`), `TestBrowserCacheLog_TC001_PageNotOpened` (`internal/amazon/browser_cache_log_test.go`) and `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile` (`internal/httpapi/amazon_browser_interrupted_test.go`). The full gates (`go test ./...`, frontend build, `npx playwright test`) run after this stage. |

No test file and no production file was changed in this stage, so no fault injection was needed: nothing outside this change folder differs from the base branch.
