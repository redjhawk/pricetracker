# No red phase for issue #384

## Why the red phase cannot fail

The change is the test code itself: it splits the first check of `assertNavigationFailure` in `internal/amazon/browser_navigation_test.go` (FR-TEST-HYG-112, FR-TEST-HYG-132 to FR-TEST-HYG-135, TS-TEST-HYG-146, TS-TEST-HYG-176 to TS-TEST-HYG-180). No production code changes, and on `master` the helper's callers already give `request_error` with `amazonCheckFailed` and `check_error`. The two new conditions together equal the old one, so the tests pass before and after the edit. Test files are locked after this stage, so this stage makes the edit; the development stage writes no code.

## Test written

In `assertNavigationFailure`, line 34's condition keeps only the result, and the failure-detail check follows lines 35–36:

```go
if got.Result != "request_error" {
	t.Fatalf("gave %q instead of a failure: %+v", got.Result, got)
}
if got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error" {
	t.Fatalf("gave a failure with message %q and second-hand status %q, not %q and %q: %+v", got.Message, got.SecondHandStatus, amazonCheckFailed, "check_error", got)
}
```

Nothing else in the file or in other files changes.

## Runs and fault injections

`N` is `go test -count=1 -run '^(TestBrowserNavigation_|TestBrowserCacheLog_|TestBrowserInterrupted_)' -v ./internal/amazon/`. `F` is `go test -count=1 -run '^TestBrowserNavigation_TC002_NoContentIsAFailure$' -v ./internal/amazon/`. Chromium was available. Every injection was a temporary edit of the test file, reverted, never committed.

1. **Baseline, before the edit.** `N` passed (`ok … 97.5s`) with 44 `=== RUN` lines (10 `TestBrowserCacheLog_`, 20 `TestBrowserInterrupted_`, 14 `TestBrowserNavigation_`, including subtests), no skip.
2. **Contradictory message, before the edit.** With the combined condition made to expect `amazonCheckFailed+"x"`, `F` failed in both subtests at the helper call line with `browser_navigation_test.go:172: gave "request_error" instead of a failure: {Result:request_error … Message:Amazon could not be reached for a price check. SecondHandStatus:check_error …}`. This is the message the issue describes. Reverted.
3. **Edit and greps.** After the edit, `grep -rn 'gave %q instead of a failure: %+v' internal/amazon/` printed one line, `internal/amazon/browser_navigation_test.go:35: t.Fatalf("gave %q instead of a failure: %+v", got.Result, got)`. `grep -n '"not a failure: ' internal/amazon/browser_navigation_test.go` printed nothing. The helper reads `t.Helper()`, the result check (`t.Fatalf`), the failure-detail check (`t.Fatalf`, five `%q`/`%+v` verbs, five arguments), then the URL and status check and the log line loop, unchanged.
4. **Diff.** `git diff --unified=0 origin/master -- internal/amazon/browser_navigation_test.go` shows two hunks: `@@ -34 +34 @@` (the condition line) and `@@ -36,0 +37,3 @@` (the three lines of the failure-detail check). `git diff --stat` reports `1 file changed, 4 insertions(+), 1 deletion(-)`.
5. **Injection A, a 200 page.** In `TestBrowserNavigation_TC002_NoContentIsAFailure`, the handler set `Content-Type: text/html; charset=utf-8` and wrote `amazontest.Fixture(t, "price-found.html")` with status 200. `F` failed in `HTTP_204` and `HTTP_205`, once each, at the helper call line (`browser_navigation_test.go:178` with the injection), with `gave "success" instead of a failure: {Result:success AmountCents:12999 … Message: SecondHandStatus:not_found …`. With `price-not-found.html`: `gave "price_not_found" instead of a failure: {Result:price_not_found … Message:Amazon did not show a detectable e…`, once per subtest. Neither output held `gave a failure with message`, although the message and second-hand status differ from the expected ones. Reverted.
6. **Injection C, wrong expected message.** In the failure-detail check, `amazonCheckFailed` became `amazonCheckFailed+"x"` in the condition and the third argument. `F` failed in both subtests, once each, at `browser_navigation_test.go:175` (the helper call line) with `gave a failure with message "Amazon could not be reached for a price check." and second-hand status "check_error", not "Amazon could not be reached for a price check.x" and "check_error": {Result:request_error AmountCents:0 … FailureCause:Amazon page could not be opened: net::ERR_ABORTED CustomerRating:<nil>}`. The output held neither `instead of a failure`, `gave "request_error"`, `diagnostics of a page not opened` nor `no log line with the request URL`. Reverted.
7. **Injection D, wrong expected status.** In the failure-detail check, `"check_error"` became `"check_errorx"` in the condition and the fourth argument. `F` failed in both subtests, once each, at line 175 with `gave a failure with message "Amazon could not be reached for a price check." and second-hand status "check_error", not "Amazon could not be reached for a price check." and "check_errorx": {Result:request_error …}`, without `instead of a failure` and without the later checks' messages. Reverted.
8. **Repeated runs after the edit.** `N` passed three times (`ok … 97.5s`, then two more passes). Each run had the same 44 `=== RUN` lines as run 1, no skip and no failure.
9. **Gates.** `git diff --name-status origin/master -- . ':!doc'` lists only `M internal/amazon/browser_navigation_test.go`. `gofmt -l internal/amazon` printed nothing. `go vet ./...` printed nothing and `go test ./...` passed (all packages `ok`).

## Where each test case is proved

| Test case | Proof |
| --- | --- |
| TC-001 | runs 3 (greps, reading the helper), 5, 6 and 7 (each check stops the test with `t.Fatalf` and is reported at the caller's line) |
| TC-002 | run 4 |
| TC-003 | run 5, with the baseline of run 2 |
| TC-004 | run 6 |
| TC-005 | run 7 |
| TC-006 | runs 1 and 8 |
| TC-007 | run 9 |
