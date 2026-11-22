# UC-AMZ-BRET-004 — Restart an Amazon item stopped by the retrieval errors

**Primary actor:** User who tracks Amazon items
**Supporting systems:** PriceFollower application, real web browser, Amazon, SQLite store
**Trigger:** After the fix of issue #204 is installed, the user chooses **Refresh price** on a stopped Amazon item.
**Origin:** Issue #204: the repeated retrieval errors stopped the tracked Amazon items.
**Covers:** FR-AMZ-BRET-007, FR-AMZ-BRET-001

## Preconditions

- The fix is installed on a documented installation.
- An Amazon item was stopped by the failures of issue #204; its listing is an available listing.

## Main flow

1. The user opens the item details page of the stopped item.
2. The user chooses **Refresh price**.
3. The check runs as in UC-AMZ-BRET-001 and records the listing's price.
4. Only this item is restarted: its stopped state is cleared, the price is recorded in its history and the failure count is reset (FR-AMZ-STOP-006). Other stopped items stay stopped.
5. The user sees the price and check time; the item is no longer stopped.

## Alternatives and errors

- **Item not stopped:** it gets its next scheduled check without user action (UC-AMZ-BRET-001).
- **Stopped item without user action:** it stays stopped; no automatic restart happens.
- **Refresh price fails** (robot check, non-success response, no answer): no price is recorded and the item stays stopped; the failure is logged with its cause (UC-AMZ-BRET-002) and counted (FR-AMZ-BRET-004).
- **Protected mode:** the same applies to the Amazon items of all users.

## Postconditions

- The item has a recorded price and is active again, with its schedule and stop rules unchanged.
