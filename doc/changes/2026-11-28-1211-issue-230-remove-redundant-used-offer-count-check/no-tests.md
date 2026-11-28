# No red phase: issue #230

## Why the red phase cannot fail

The change is the test code itself: it edits the TC-004 refresh test of `internal/httpapi/amazon_browser_error_page_test.go` (subtest "Refresh price, Refresh all prices, Refresh price again" of `TestAmazonBrowserErrorPage_TC004_RefreshSendsNoUsedOfferRequest`) for behaviour already on `master`. No production code changes. The test passes before and after the edit, so no test can fail before an implementation. As the technical specification requires (issue #230 design, TS-TEST-HYG-191), this stage made the edit, and fault injection proves each check. The development stage writes no code.

## Test edit

- After the first Refresh price's `assertUnavailableOfferKept`, added `if got := api.usedOfferCount(); got != 0 { t.Fatalf("%d used-offer requests after the first Refresh price: %+v", got, api.fixture.Requests()) }`.
- After the loop over `ids` of Refresh all prices, outside it, added the same check with `%d used-offer requests after Refresh all prices: %+v`.
- Removed the redundant `%d used-offer requests in total: %+v` check after the repeated Refresh price loop. The loop's closing brace is now directly followed by the unchanged log check `used-offer failure logged: %v`.

## Runs and fault injections

`N` = `go test -count=1 -run '^TestAmazonBrowserErrorPage_' -v ./internal/httpapi/`, `F` = `go test -count=1 -run '^TestAmazonBrowserErrorPage_TC004_RefreshSendsNoUsedOfferRequest$' -v ./internal/httpapi/`, with Chromium. The injected request was the statement of the design (`http.Get(api.fixture.URL + "/gp/aod/ajax")`, body closed), added and removed with `Edit` and never committed. After each removal, `git diff` showed only the edit above.

1. **Baseline, before the edit:** `N` passed. `=== RUN` lines: `TestAmazonBrowserErrorPage_TC003_ScheduledCheckSendsNoUsedOfferRequest` with `HTTP_404`, `HTTP_410`, `item_without_an_earlier_second-hand_offer` and `protected_mode_with_two_owners`; `TestAmazonBrowserErrorPage_TC004_RefreshSendsNoUsedOfferRequest` with `Refresh_price,_Refresh_all_prices,_Refresh_price_again` and `Refresh_price_of_a_stopped_item`.
2. **Code check:** `grep -n 'usedOfferCount()'` lists lines 133, 144 and 153 inside the TC-004 refresh test: after the first Refresh price, after the loop over `ids`, and inside the repeated Refresh price loop. Each check calls `t.Fatalf` with a format that starts with `%d` (`got`) and ends with `%+v` (`api.fixture.Requests()`). The loop's message is unchanged. `grep -c 'used-offer requests in total'` gives 0, and the log check directly follows the loop.
3. **Diff:** `git diff --unified=0 origin/master` shows two hunks that each add three lines (`@@ -132,0 +133,3 @@`, `@@ -140,0 +144,3 @@`) and one hunk that removes three lines (`@@ -151,3 +156,0 @@`); `--stat` reports 6 insertions and 3 deletions.
4. **Injection A** (after the first `refresh(t, api.server, ids[0])`): `F` failed once with `1 used-offer requests after the first Refresh price: [{Method:GET Path:/gp/aod/ajax ...} {Method:GET Path:/dp/B0ERRP0041 ...}]`. The output did not contain `after Refresh all prices`, `after repeated Refresh price`, `in total` or `Path:/dp/B0ERRP0042`, so Refresh all prices never ran. "Refresh price of a stopped item" passed. Reverted.
5. **Injection B** (after the `POST /api/v1/items/refresh` check): `F` failed once with `1 used-offer requests after Refresh all prices: [`, and no other step message appeared. Reverted. **Injection C** (inside the loop over `ids`, after `assertUnavailableOfferKept`, only when `id == ids[0]`): `F` failed once with the same message. The recorded requests include `Path:/dp/B0ERRP0042`, so the check ran after both items. Neither output contained `after the first Refresh price` or `after repeated Refresh price`. "Refresh price of a stopped item" passed both times. Reverted.
6. **Injection D** (in the repeated Refresh price loop, after `refresh`, only when `attempt == 0`): `F` failed once with `1 used-offer requests after repeated Refresh price 1: [`. There were three `/dp/B0ERRP0041` requests, so no request was made at 14:15. Reverted. **Injection E** (the same, only when `attempt == 1`): `F` failed once with `1 used-offer requests after repeated Refresh price 2: [` (four `/dp/B0ERRP0041` requests). Neither output contained `after the first Refresh price`, `after Refresh all prices`, `used-offer requests in total` or `used-offer failure logged`. Reverted. Across A to E, the four failure messages were pairwise different.
7. **After the edit:** `N` passed twice, with the same `=== RUN` lines as run 1, eight `--- PASS` lines and no `--- SKIP`.
8. **Gates:** `git diff --name-status origin/master -- . ':!doc'` lists only `M internal/httpapi/amazon_browser_error_page_test.go`. `gofmt -l internal/httpapi` prints nothing, and `go vet ./...` reports nothing. `go test ./...` passed in every package except `internal/amazon/amazontest`. There, `TestBrowserProcesses_Issue196_TC004_ListsTheProfileProcesses` failed once under the full-suite load: it saw two `sh -c sleep 30` entries for one process. This change does not touch that package. The test passed alone with `-count=3`, and the whole package passed when run again (`go test -count=1 ./internal/amazon/amazontest/`).

## Where each test case is proved

| Test case | Proof |
| --- | --- |
| TC-001 | Run 2 (three checks at the required places, `t.Fatalf`, `%d` first and `%+v` last, distinct messages) |
| TC-002 | Run 2 (`grep -c 'used-offer requests in total'` is 0; the log check follows the loop) |
| TC-003 | Run 4, injection A |
| TC-004 | Run 5, injections B and C |
| TC-005 | Run 6, injections D and E |
| TC-006 | Runs 1 and 7 |
| TC-007 | Runs 3 and 8 |
