# Red phase skipped for issue #219

This change is test code only. In `internal/amazon/browser_navigation_test.go` it removes the three Amazon-answer checks before calls of `assertNavigationFailure` and rewords the helper's first failure message. The helper's first check already fails on any result other than `request_error`. The technical specification (`doc/specifications/test-suite-hygiene/technical.md`, "No redundant Amazon-answer check before `assertNavigationFailure` (issue #219)", TS-TEST-HYG-145–149) changes no production code. The tests pass on `master` before and after the edit, so no red phase can fail. Test files are locked after this stage, so this stage makes the edit itself. The red gate is skipped. The development stage writes no code.

## Test edited

`internal/amazon/browser_navigation_test.go`:

- Line 35 `t.Fatalf("not a failure: %+v", got)` becomes `t.Fatalf("gave %q instead of a failure: %+v", got.Result, got)`, in place, two tabs. `t.Helper()` stays the helper's first statement. The condition, the URL/status check and the log-line loop keep their messages.
- Lines 100–102 (`attachment then a normal page`), 145–147 (`TC-002 existing Downloads folder, …`) and 178–180 (`HTTP 204`/`HTTP 205`) are deleted: each `if` line, its `t.Fatalf` line and its closing brace. No blank line or comment replaces them. The `success` comparisons after the normal page and in `inline page` are kept.

No other line changes. Every import is still used, and `answer.name` and `status` are still used in the subtest names.

## Runs

`N` is `go test -count=1 -run '^(TestBrowserNavigation_|TestBrowserCacheLog_|TestBrowserInterrupted_)' -v ./internal/amazon/`. `F` is `go test -count=1 -run '^TestBrowserNavigation_TC002_NoContentIsAFailure$' -v ./internal/amazon/`. Chromium was available. The injections were temporary edits of the `TestBrowserNavigation_TC002_NoContentIsAFailure` handler, each reverted with `Edit`.

1. **Baseline, before the edit.** `N` passed (`ok … 96.967s`) with 44 `=== RUN` lines: `TestBrowserCacheLog_` TC001 (4 subtests) and TC002 (4); `TestBrowserInterrupted_` TC001 (2), TC002 (3), TC003 (3), TC004 (5) and TC006 (2); `TestBrowserNavigation_TC001_AttachmentIsAFailure` with `attachment_then_a_normal_page`, the three `TC-002_existing_Downloads_folder,_…` subtests and `inline_page`; `TestBrowserNavigation_TC002_NoContentIsAFailure` with `HTTP_204` and `HTTP_205`; `TestBrowserNavigation_TC003_EmptyNonSuccessKeepsItsStatus` (4).
2. **Baseline message, before the edit.** Injection A: the handler writes `Content-Type: text/html; charset=utf-8` and the `price-found.html` fixture with status 200. `F` **failed** in both subtests with the removed check: `HTTP 204 gave the Amazon answer "success": {Result:success AmountCents:12999 …}` and `HTTP 205 gave the Amazon answer "success": …`. Reverted.
3. **Edit and greps.** `grep -n 'gave the Amazon answer'` and `grep -n 'price_not_found'` on the file print nothing. `grep -rn 'gave %q instead of a failure: %+v' internal/amazon/` prints one line: `browser_navigation_test.go:35: t.Fatalf("gave %q instead of a failure: %+v", got.Result, got)`. Before each of the four helper calls (lines 100, 110, 142 and 172) is the `}` of a request-count check or `logged := output()`. No result comparison precedes any of them.
4. **Diff.** `git diff --unified=0 origin/master -- internal/amazon/browser_navigation_test.go` has four hunks: `@@ -35 +35 @@`, `@@ -100,3 +99,0 @@`, `@@ -145,3 +141,0 @@` and `@@ -178,3 +171,0 @@`. `git diff --stat` reports `1 file changed, 1 insertion(+), 10 deletions(-)`.
5. **Injections after the edit.** Each subtest fails once, through the helper, with the failure reported at the helper call line (`browser_navigation_test.go:174` with the two injected lines, `:172` for B):
   - A with `price-found.html`: `F` **failed** in `HTTP_204` and `HTTP_205` with `gave "success" instead of a failure: {Result:success AmountCents:12999 … HTTPStatus:200 …}`.
   - A with `price-not-found.html`: `F` **failed** in both with `gave "price_not_found" instead of a failure: {Result:price_not_found … Message:Amazon did not show a detectable euro price. …}`.
   - B, an empty `w.WriteHeader(http.StatusNotFound)`: `F` **failed** in both with `gave "unavailable" instead of a failure: {Result:unavailable … HTTPStatus:404 …}`.

   Reverted. After that, `git diff` showed only the change of run 4.
6. **After the edit.** `N` passed (`ok … 97.074s`) with the same 44 `=== RUN` lines as run 1, in the same order. A repeated run passed too (`ok … 96.983s`). No test was skipped.
7. **Gates.** `git diff --name-status origin/master -- . ':!doc'` lists only `M internal/amazon/browser_navigation_test.go`. `gofmt -l internal/amazon` prints nothing. `go vet ./...` passes with no printf finding. `go test ./...` passes (every package `ok`).

## Where each test case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | Run 3 |
| TC-002 | Run 3 and reading the helper (see the note below) |
| TC-003 | Run 4 |
| TC-004 | Runs 2 and 5 |
| TC-005 | Runs 1 and 6 |
| TC-006 | Run 7 |

## Note on the `"not a failure: "` grep (TC-002)

The functional specification's acceptance criteria, TC-002 and run 3 of the design all say that `grep -rn '"not a failure: ' internal/amazon/` prints nothing. After the edit, it prints six lines. None is in `browser_navigation_test.go`. They are in tests and helpers that this issue does not cover:

- `browser_interrupted_test.go:59`
- `browser_test.go:364` and `:383`
- `browser_retrieval_test.go:417` and `:568`
- `browser_profile_test.go:136`

The issue asks only for a change to the message in `assertNavigationFailure`. FR-TEST-HYG-114 and TC-003/TC-006 limit the diff outside `doc/` to `browser_navigation_test.go`, with 1 insertion and 10 deletions. So these lines stay as they are. The grep is met for the file this change edits: `grep -n '"not a failure: ' internal/amazon/browser_navigation_test.go` prints nothing. The functional intent of FR-TEST-HYG-112 is met: the helper no longer uses the old message. The specification's grep should be read as limited to this file.
