# Issue #185 Tests missing from #168 (Amazon human-like checks)

Follow-up of #168 / PR #179. The review asked for tests that the developer stage could not add (no test files after the TDD red commit):

1. A scheduled Amazon check due in the morning window while `now` is in the evening window: no request is sent, the item is redrawn for the next window and "missed its window" is logged (R1-01).
2. The one-time redraw after the upgrade: `amazon_request_state.human_checks_redraw = 1` makes `service.New` draw a window time for every Amazon item and then clears the mark (R1-03).
3. Browser check with an empty-body HTTP response: 404 and 410 → `unavailable` with `HTTPStatus`; 503 and 429 → failure "Amazon returned HTTP n" with `HTTPStatus` (R4-01).

Also: `internal/amazon/browser_test.go` is not gofmt-ed (`browserCheck(t,collector, …)`).

Add the `ai-dev` label once #168 is merged.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

## Comments by trusted authors

