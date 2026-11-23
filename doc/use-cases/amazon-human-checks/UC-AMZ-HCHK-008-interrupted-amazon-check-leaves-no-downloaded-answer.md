# UC-AMZ-HCHK-008 — Interrupted Amazon check leaves no downloaded answer on the server's disk

**Primary actor:** PriceFollower application, on behalf of the user who tracks Amazon items
**Secondary actor:** Person who runs the server (stops the server, observes the server's disk and the server log)
**Supporting systems:** Real web browser, Amazon, Amazon request gate, SQLite store
**Trigger:** An Amazon check (first check after add, scheduled check, **Refresh all prices** or **Refresh price**) is interrupted, because it reaches the Amazon check time limit or because the server stops, while the browser is still requesting the product page or while the check waits after the product page was offered as a file download.
**Origin:** Issue #233 (review comment R2-01 on #231): when the check was interrupted, the browser could close before the downloaded answer was removed from its profile.
**Covers:** FR-AMZ-HCHK-015 (with FR-AMZ-HCHK-014 for the download guarantee and FR-AMZ-HCHK-011 for the check result)

## Preconditions

- One or more Amazon items are tracked and not stopped (or the user chose **Refresh price** on a stopped item).
- The Amazon browser's persistent profile exists in the server's data directory.
- The Downloads folder of the account running the server may or may not exist.

## Main flow

1. The application opens the item's Amazon page through the real web browser (UC-AMZ-HCHK-001, UC-AMZ-HCHK-003).
2. Amazon answers the product page with a response offered as a file download, for example HTTP 200 `text/html` with `Content-Disposition: attachment` (UC-AMZ-HCHK-007).
3. While the check waits after the download was offered, the check reaches the Amazon check time limit.
4. The browser closes. Once it has closed, no file holding that response remains in the Amazon browser's persistent profile, and the Downloads folder of the account running the server has no new file (and is not created if it did not exist).
5. The check is a counted failure, as today for a check that reaches the time limit; the item's last successful price and second-hand offer are kept.
6. The check ends within a bounded time, even if the browser does not confirm the removal.
7. The server log has no line saying that the browser cache could not be cleared.

## Alternatives and errors

- **Interrupted while the page is still being requested** (step 3 happens before step 2 completes): the outcome is the same; once the browser has closed, no file holding the response remains in the profile or the Downloads folder.
- **Server stopped during the check** (instead of the time limit): the person who runs the server stops it while the check is at step 2 or 3. The guarantee of step 4 holds; the server stops as today and within a bounded time, even if the browser does not confirm the removal.
- **Removal really fails:** the server log reports that the browser cache could not be cleared, with the real cause of that failure, never the interruption of the check. The check result is unchanged.
- **Fifth consecutive failure:** with 4 earlier consecutive failures, a check that reaches the time limit stops every tracked Amazon item, as today (UC-AMZ-HCHK-005).
- **Next check answered with the normal page:** the check of the same or another item behaves as if the download had never been offered; the price is detected and recorded (UC-AMZ-HCHK-001).
- **Protected mode:** the same applies to the Amazon items of all users.

## Postconditions

- The Amazon browser's persistent profile holds no file with the response offered as a download, and the Downloads folder of the account running the server has no new file (and still does not exist if it did not before).
- The check result, failure count, stop rules, item states, messages and server stop are as today; nothing else changes.
