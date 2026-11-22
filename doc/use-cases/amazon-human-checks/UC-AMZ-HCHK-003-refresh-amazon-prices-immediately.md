# UC-AMZ-HCHK-003 — Refresh Amazon prices immediately

**Primary actor:** Signed-in user
**Supporting systems:** Real web browser, Amazon, Amazon request gate, SQLite store
**Trigger:** The user chooses **Refresh price** on an Amazon item's details page (UC-ID-02) or **Refresh all prices** in the tracked-items view (UC-TI-03), at any time of day.
**Covers:** FR-AMZ-HCHK-001, FR-AMZ-HCHK-002, FR-AMZ-HCHK-003, FR-AMZ-HCHK-007

## Preconditions

- At least one Amazon item is tracked.

## Main flow

1. The user chooses Refresh price or Refresh all prices, for example at 14:00, outside both windows.
2. The checks start immediately, without waiting for a window, and the user sees the refresh progress as today.
3. Each Amazon page is opened and read through a real web browser, and the result is recorded as in UC-PC-01.
4. The user sees the updated price and check time.
5. The item's scheduled checks are unchanged: it still gets its scheduled check in the next window at the time drawn for it.

## Alternatives and errors

- **Stopped items with Refresh all prices:** stopped Amazon items are skipped; other Amazon items and LeBoncoin items are checked (FR-AMZ-STOP-004).
- **Refresh price on a stopped item:** the item is checked although stopped and restarts on an Amazon answer (FR-AMZ-STOP-006, UC-PC-02).
- **Failure:** the browser gets no usable answer or cannot open the page; the failure is logged and counted, the last successful price stays shown and the details page shows the refresh failed (FR-AMZ-STOP-001, FR-AMZ-STOP-002, FR-AMZ-STOP-007). The fifth consecutive failure stops every Amazon item.
- **Check already in progress:** a second refresh of the same item starts nothing.

## Postconditions

- Manual refreshes are never delayed by the windows and never move or remove the item's scheduled checks.
