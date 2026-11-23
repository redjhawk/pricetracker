# UC-AMZ-HCHK-006 — Unavailable listing page the browser does not open sends no used-offer request

**Primary actor:** PriceFollower application, on behalf of the user who tracks Amazon items
**Supporting systems:** Real web browser, Amazon, Amazon request gate, SQLite store
**Trigger:** An Amazon check (first check after add, scheduled check, **Refresh all prices** or **Refresh price**) gets HTTP 404 or 410 for the product page, and the browser does not open it as a page (for example an empty body).
**Origin:** Issue #187 (review comment R5-02 on #179): the used-offer requests were still sent from the browser's error page, where they cannot succeed, and logged a misleading used-offer failure.
**Covers:** FR-AMZ-HCHK-012, FR-AMZ-HCHK-013

## Preconditions

- One or more Amazon items are tracked and not stopped (or the user chose **Refresh price** on a stopped item).
- The item may have a last detected second-hand offer from an earlier check.

## Main flow

1. The application opens the item's Amazon page through the real web browser (UC-AMZ-HCHK-001, UC-AMZ-HCHK-003).
2. Amazon answers the product page with HTTP 404 or 410 that the browser does not open as a page, for example with an empty body.
3. The listing is recorded as unavailable; this is an Amazon answer and resets the consecutive failure count (FR-AMZ-HCHK-012).
4. No second-hand (used-offer) request is sent to Amazon for this check.
5. The check's second-hand result is recorded as a check error; no used-offer failure is written to the server log for this check.
6. The item's last detected second-hand offer is kept.
7. The user sees **Listing unavailable** and the second-hand check error in the tracked-items list and on the item details page.

## Alternatives and errors

- **404 or 410 page that the browser opens** (with content): unchanged, the listing is unavailable and the used-offer requests are sent as before (UC-AMZ-HCHK-001).
- **Other status not opened as a page** (for example 503 with an empty body): a failure logged with that HTTP status and counted (UC-AMZ-HCHK-005, FR-AMZ-HCHK-012).
- **Success status not opened as a page:** a counted failure (UC-AMZ-HCHK-005, FR-AMZ-HCHK-011).
- **Refresh price on a stopped item:** the listing unavailable answer is an Amazon answer, so only that item is restarted as for any other Amazon answer (FR-AMZ-STOP-006).
- **Protected mode:** the same applies to the Amazon items of all users.

## Postconditions

- The item shows **Listing unavailable** and the consecutive failure count is 0.
- Amazon received only the product page request for this check, no used-offer request.
- The item's last detected second-hand offer is unchanged.
- Item states, messages, schedule and stop rules are otherwise unchanged.
