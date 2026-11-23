# UC-AMZ-HCHK-005 — Amazon check answered with a page the browser does not open

**Primary actor:** PriceFollower application, on behalf of the user who tracks Amazon items
**Supporting systems:** Real web browser, Amazon, Amazon request gate, SQLite store
**Trigger:** An Amazon check (first check after add, scheduled check, **Refresh all prices** or **Refresh price**) gets a response that the browser does not open as a page.
**Origin:** Issue #186 (review comment R5-01 on #179): a success response offered as a file download, or without content, was recorded as "price not found" and reset the failure count.
**Covers:** FR-AMZ-HCHK-011, FR-AMZ-HCHK-012

## Preconditions

- One or more Amazon items are tracked and not stopped (or the user chose **Refresh price** on a stopped item).
- The item has a last successful price, and possibly a second-hand offer, from an earlier check.

## Main flow

1. The application opens the item's Amazon page through the real web browser (UC-AMZ-HCHK-001, UC-AMZ-HCHK-003).
2. Amazon answers with a success status (HTTP 200–299) that the browser does not open as a page, for example HTTP 200 `text/html` with `Content-Disposition: attachment` (a file download) or HTTP 204 or 205 (no content).
3. The check is a failure: it is logged with the request URL and the browser's error, without secrets, and counted toward the 5 consecutive failures.
4. Nothing is recorded as "price not found" or as another Amazon answer, so the consecutive failure count is not reset.
5. The item's last successful price and second-hand offer are kept.
6. The user sees the item's retrieval issue and the second-hand check error in the tracked-items list and on the item details page, not "price not found".

## Alternatives and errors

- **Fifth consecutive failure:** with 4 earlier consecutive failures, this check stops every tracked Amazon item and the Amazon tab shows the stop warning (FR-AMZ-HCHK-003, FR-AMZ-STOP-003, FR-AMZ-STOP-005).
- **Refresh price:** the details page shows that the refresh failed (FR-AMZ-STOP-007); a stopped item stays stopped because no Amazon answer was received.
- **Non-success status not opened as a page** (for example an error status with an empty body): unchanged, the status decides the result (FR-AMZ-HCHK-012):
  - 404 or 410: the listing is shown as unavailable; this is an Amazon answer and resets the consecutive failure count.
  - Any other status (for example 503): a failure logged with that HTTP status and counted.
- **Success response opened as a page:** the check proceeds as in UC-AMZ-HCHK-001 (price, second-hand offer, price not found or listing unavailable).
- **Protected mode:** the same applies to the Amazon items of all users.

## Postconditions

- A success response that the browser does not open as a page never resets the consecutive failure count and never changes the item's last successful price or second-hand offer.
- Item states, messages, schedule and stop rules are otherwise unchanged.
