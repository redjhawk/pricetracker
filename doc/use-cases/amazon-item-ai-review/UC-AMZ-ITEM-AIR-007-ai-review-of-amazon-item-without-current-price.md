# UC-AMZ-ITEM-AIR-007 — See the AI review of an Amazon item without a current price

**Primary actor:** Operator (in protected mode, the item's owner)
**Supporting systems:** PriceFollower application, Amazon, SQLite store
**Trigger:** The operator opens the details page of an Amazon item that has no current price: no price has been recorded yet, or its latest Amazon answer is "price not found" or "listing unavailable".
**Origin:** Issue #305.
**Covers:** FR-AMZ-ITEM-AIR-033, FR-AMZ-ITEM-AIR-034, FR-AMZ-ITEM-AIR-022

## Preconditions

- The Amazon item is not stopped and has no current price; it may have earlier reviews. The owner may or may not have a Claude token.

## Main flow

1. The operator opens the item details page.
2. The **AI review** section shows the latest review and its history, and informs the operator: "No current price: an AI review needs a current Amazon price."
3. No **Refresh AI review** button is shown.
4. A later check (scheduled, **Refresh price** or **Refresh all prices**) records a price.
5. The message disappears and the **Refresh AI review** button is shown again (enabled when a Claude token is saved).
6. No review starts; the shown review is unchanged.

## Alternatives and errors

- **Latest check failed with a retrieval error after an earlier price:** the item keeps its current price; this use case does not apply (UC-AMZ-ITEM-AIR-003).
- **Item also stopped:** only the stopped message is shown (UC-AMZ-ITEM-AIR-006).
- **Review requested without the button:** it is refused with the no-current-price message; no review is started or stored and no review failure is recorded.

## Postconditions

- While the item has no current price, no review can be requested and the review history is unchanged.
