# Amazon searches: strengthen two regression tests

Status: todo (non-blocking PR review comments, triaged on 2026-10-07; not scheduled)

## Problem

- `TestAmazonGateIgnoresCancelledRequests` (`internal/service/amazon_gate_test.go`) starts from 0 consecutive failures, so an implementation that reset the count on a cancelled request would still pass.
- `TestSearchPassRetriesFailedCapture` (`internal/service/search_worker_test.go`) no longer checks that a later pass captures the items after a failed capture.

## Suggested work

- Record at least one failure before the cancelled request and assert that the count is unchanged, both in memory and persisted.
- Run another pass after the failed capture and assert that `CapturedAt` is set and the items are recorded.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/88#pullrequestreview-5446335132
- PR comment: https://github.com/redjhawk/pricetracker/pull/90#pullrequestreview-5446335740
- Issue: not yet created
