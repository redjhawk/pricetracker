# UC-AMZ-ITEM-AIR-010 — Read an Amazon review after the price changed

**Primary actor:** Operator (in protected mode, the item's owner)
**Supporting systems:** PriceFollower application, Amazon, SQLite store
**Trigger:** A price check records a different price or second-hand offer for a reviewed Amazon item.
**Origin:** Issue #305.
**Covers:** FR-AMZ-ITEM-AIR-022, FR-AMZ-ITEM-AIR-023, FR-AMZ-ITEM-AIR-011

## Preconditions

- The Amazon item has a review made at 100,00 €.

## Main flow

1. A check (scheduled, **Refresh price** or **Refresh all prices**) records the new price 80,00 €.
2. No review is started.
3. The operator opens the item details page: the current price is 80,00 €, and the **AI review** section shows the same review as before, labelled as based on 100,00 €.
4. The operator chooses **Refresh AI review** (UC-AMZ-ITEM-AIR-003); the new review, labelled as based on 80,00 €, is shown first and the review based on 100,00 € is kept in the history.

## Alternatives and errors

- **Other events** (goal saved, token saved, item restarted after a stop): no review is started and the shown review is unchanged.

## Postconditions

- Each review keeps the price it was based on; reviews change only through **Refresh AI review**.
