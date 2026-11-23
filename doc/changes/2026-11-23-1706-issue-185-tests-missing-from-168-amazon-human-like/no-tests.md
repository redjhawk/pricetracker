# No red phase for issue #185

This change **does** add tests, but the red phase cannot run on it. The tests cover behaviour that is already on `master` (issue #168, PR #179). They pass before any development, so the red gate's "the tests fail before implementing" check would always fail. The tests are delivered complete and `gofmt`-clean in this stage, as the technical specification states (`doc/specifications/test-suite-hygiene/technical.md`, "Delivery", TS-TEST-HYG-030).

## Tests added

| Test case | Test |
| --- | --- |
| TC-001 | `internal/service/amazon_human_checks_issue185_test.go`: `TestAmazonMissedWindow_Issue185_TC001_SkippedRedrawnAndLogged` |
| TC-002 | `internal/service/amazon_human_checks_issue185_test.go`: `TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce` |
| TC-003 | `internal/amazon/browser_empty_status_test.go`: `TestBrowserEmptyErrorStatus_Issue185_TC003_ClassifiedByStatus` (subtests `HTTP 404`, `HTTP 410`, `HTTP 503`, `HTTP 429`) |
| TC-004 | Repository check: `gofmt -l internal/amazon` prints nothing. `internal/amazon/browser_test.go` was already formatted on `master` and is unchanged. |
| TC-005 | Gates: `go vet ./...` and `go test ./...` pass; `go test -count=1 -v -run 'Issue185' ./internal/service/ ./internal/amazon/` reports the three tests and four subtests as `PASS`. `git diff --name-only master -- . ':!doc'` lists only the two new `_test.go` files. |

## Proof of failure (mutation checks)

Each mutation was applied to production code, the new tests were run, and the code was restored. `git diff master -- internal/service/service.go internal/store/amazon_requests.go internal/amazon/browser.go` is empty afterwards.

1. `skipScheduledCheck` (`internal/service/service.go`): `if !active { return false }` inserted after `now := s.now()`. The TC-001 test fails with `the item whose morning window has passed was checked at 20:00: [B000001851 B000001852]`.
2. `AmazonIDsToSchedule` (`internal/store/amazon_requests.go`): the `OR (SELECT human_checks_redraw …) = 1` clause dropped. The TC-002 test fails with `next check of B000001853 at 2030-03-06 08:00:00 +0100 CET after the start, want the evening window of 2030-03-05`.
3. `classifyPage` (`internal/amazon/browser.go`): `fmt.Errorf("Amazon returned HTTP %d", 404)` instead of `page.status`. Subtests `HTTP 503` and `HTTP 429` of the TC-003 test fail with `failure cause "Amazon returned HTTP 404", want "Amazon returned HTTP 503"` (and `… 429`). Subtests `HTTP 404` and `HTTP 410` still pass, as expected.

## Note for the owner

`gofmt -l internal/` on `master` lists `internal/service/amazon_browser_retrieval_test.go` (a one-line function literal at line 221). It is outside `internal/amazon` and outside this issue's scope, so it is left unchanged.
