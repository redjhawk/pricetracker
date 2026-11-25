# UC-AMZ-ITEM-AIR-006 — See the AI review of a stopped Amazon item

**Primary actor:** Operator (in protected mode, the item's owner)
**Supporting systems:** PriceFollower application, Amazon, SQLite store
**Trigger:** The operator opens the details page of an Amazon item stopped by consecutive Amazon failures (FR-AMZ-STOP-003).
**Origin:** Issue #305.
**Covers:** FR-AMZ-ITEM-AIR-032, FR-AMZ-ITEM-AIR-034, FR-AMZ-ITEM-AIR-022

## Preconditions

- The Amazon item is stopped; it may have earlier reviews. The owner may or may not have a Claude token.

## Main flow

1. The operator opens the item details page.
2. The **AI review** section shows the latest review and its history, and informs the operator: "Amazon checks are stopped: restart the item with Refresh price to request an AI review."
3. No **Refresh AI review** button is shown.
4. The operator chooses **Refresh price**; the check gets an Amazon answer with a price and restarts the item (FR-AMZ-STOP-006, UC-AMZ-BRET-004).
5. The stopped message disappears and the **Refresh AI review** button is shown again (enabled when a Claude token is saved).
6. No review starts; the shown review is unchanged.

## Alternatives and errors

- **Refresh price fails:** the item stays stopped; the message stays and no button is shown.
- **Restarted but the check finds no price:** the item has no current price; UC-AMZ-ITEM-AIR-007 applies.
- **Stopped and without a current price:** only the stopped message is shown.
- **Review requested without the button** (for example a request sent directly to the application): it is refused with the stopped message; no review is started or stored and no review failure is recorded.

## Postconditions

- While the item is stopped, no review can be requested and the review history is unchanged.
