# UC-PC-02 — Restart a stopped Amazon item while Refresh all is pending

**Primary actor:** Signed-in user
**Supporting systems:** Amazon collector, Amazon request gate, SQLite store
**Trigger:** The user chooses Refresh price on a stopped Amazon item while a Refresh all prices request is still queued.
**Origin:** Review finding REV-001 (issue #111): the Refresh price was accepted but silently ignored, because Refresh all had already reserved the stopped item for a check that the gate skips.

## Preconditions

- Amazon checks of several Amazon items are stopped after repeated failures (FR-AMZ-STOP-003).
- The user has chosen Refresh all prices and its checks are still waiting for a free collection slot.

## Main flow

1. Refresh all answers with the number of all tracked items, stopped ones included, but does not reserve the stopped Amazon items; they are not shown as refreshing.
2. The user opens a stopped Amazon item and chooses Refresh price.
3. The service reserves the item and queues a manual check, which the gate runs although the item is stopped.
4. Amazon answers: the check is recorded, the stop of this item is cleared, and the consecutive failure count is reset.

## Alternatives and errors

- If the request fails, the item stays stopped, the attempt is recorded and the failure counts (FR-AMZ-STOP-006).
- If a manual check of the item is already running, the second Refresh price starts nothing (one check per item at a time).

## Postconditions

- Only the refreshed item is restarted; the other Amazon items stay stopped and receive no request.

Regression test: `TestRefreshItemRestartsStoppedItemWhileRefreshAllQueued` in `internal/service/amazon_gate_test.go`.
