# UC-AMZ-HCHK-001 — Scheduled Amazon checks at random times

**Primary actor:** PriceFollower scheduler, on behalf of the user who tracks Amazon items
**Supporting systems:** Real web browser, Amazon, Amazon request gate, SQLite store
**Trigger:** A tracked Amazon item's random scheduled time in the morning (06:00–09:00) or evening (19:00–23:00) window is reached.
**Origin:** Issue #168: Amazon stopped answering checks made at the same time every day with plain page downloads.
**Covers:** FR-AMZ-HCHK-001, FR-AMZ-HCHK-002, FR-AMZ-HCHK-003, FR-AMZ-HCHK-004, FR-AMZ-HCHK-005, FR-AMZ-HCHK-008, FR-AMZ-HCHK-009

## Preconditions

- One or more Amazon items are tracked and not stopped.
- Times are in the schedule timezone (`Europe/Paris` by default).

## Main flow

1. Each day, the scheduler draws for every tracked Amazon item one random time in the morning window and one in the evening window, independently of the other items and of the previous days. The configured check times (08:00 and 20:00 by default) are not used for Amazon items.
2. When an item's drawn time is reached, the scheduler starts a check of that item alone; other Amazon items wait for their own times.
3. The item's Amazon page is opened and read through a real web browser that runs the page's scripts and loads its resources; the browser is never shown to the user.
4. Amazon gives a usable answer: the price, any second-hand offer sold by Amazon, price not found or listing unavailable are recorded exactly as before (UC-PC-01), and the consecutive failure count is reset.
5. The user sees the updated price and check time in the tracked-items list and on the item details page.
6. The item gets no further scheduled check in that window; its next one is in the next window, at the time drawn for it.

## Alternatives and errors

- **Failure:** the browser gets no usable answer (non-success response, challenge, network error) or cannot open the page. The failure is logged without secrets and counted; the last successful price stays shown (FR-AMZ-STOP-001, FR-AMZ-STOP-002).
- **Fifth consecutive failure:** every tracked Amazon item becomes stopped and the Amazon tab shows the stop warning (FR-AMZ-STOP-003, FR-AMZ-STOP-005). At their later scheduled times stopped items are skipped and no Amazon request is sent for them.
- **Stopped item:** an item already stopped is skipped at its scheduled time (FR-AMZ-STOP-004).
- **Check already in progress:** if a check of the same item is running, the scheduled check is not duplicated.
- **Application restart before the drawn time:** the scheduled check still happens within the same window.
- **Application restart after the drawn time:** a check already made in the window is not repeated in that window.
- **Item deleted:** its scheduled checks are cancelled.
- **LeBoncoin items:** unaffected; they keep their configured check times (08:00 and 20:00 by default).

## Postconditions

- Each tracked, non-stopped Amazon item has exactly one scheduled check per day in each window, at times that differ from day to day and from item to item.
- No Amazon check is made with a plain page download.
- In protected mode, this applies to the Amazon items of all users.
