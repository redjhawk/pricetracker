# UC-AMZ-HCHK-002 — Add an Amazon item and start its random schedule

**Primary actor:** Signed-in user
**Supporting systems:** Real web browser, Amazon, PriceFollower scheduler, SQLite store
**Trigger:** The user adds an Amazon listing (UC-IM-01).
**Covers:** FR-AMZ-HCHK-001, FR-AMZ-HCHK-002, FR-AMZ-HCHK-003, FR-AMZ-HCHK-006

## Preconditions

- The listing is a supported Amazon product page not yet tracked by the user.

## Main flow

1. The user adds the Amazon listing.
2. The item is stored and its first check starts immediately, without waiting for a window.
3. The page is opened and read through a real web browser, and the result is recorded as in UC-PC-01.
4. The user sees the item's price (or its retrieval outcome) in the tracked-items list.
5. The item's scheduled checks start with the next window that begins after the add, at a random time in that window (UC-AMZ-HCHK-001).

## Alternatives and errors

- **Added inside a window:** an item added at 07:00 is checked at once and then in the evening window of the same day, not again in that morning window.
- **First check fails:** the failure is logged and counted (FR-AMZ-STOP-001, FR-AMZ-STOP-002); the item shows its retrieval issue and still gets its scheduled checks.
- **Amazon items stopped:** the add and first check follow [Amazon request stop](../../specifications/amazon-request-stop/functional.md) unchanged.

## Postconditions

- The item is tracked, has had one immediate check, and has a random scheduled check in the next window.
