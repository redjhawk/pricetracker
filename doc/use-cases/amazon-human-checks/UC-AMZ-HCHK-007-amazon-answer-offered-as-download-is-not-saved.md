# UC-AMZ-HCHK-007 — Amazon answer offered as a file download is not saved on the server's disk

**Primary actor:** PriceFollower application, on behalf of the user who tracks Amazon items
**Secondary actor:** Person who runs the server (observes the server's disk)
**Supporting systems:** Real web browser, Amazon, Amazon request gate, SQLite store
**Trigger:** During an Amazon check (first check after add, scheduled check, **Refresh all prices** or **Refresh price**), the browser receives a response offered as a file download, for example a product page answered with HTTP 200 `text/html` and `Content-Disposition: attachment`.
**Origin:** Issue #217 (review comment R1-01 on #216): the browser set no download behaviour, so such a response could be written to the browser profile or to the Downloads folder on every such check.
**Covers:** FR-AMZ-HCHK-014 (with FR-AMZ-HCHK-011 for the check result)

## Preconditions

- One or more Amazon items are tracked and not stopped (or the user chose **Refresh price** on a stopped item).
- The Amazon browser's persistent profile exists in the server's data directory.
- The Downloads folder of the account running the server may or may not exist.

## Main flow

1. The application opens the item's Amazon page through the real web browser (UC-AMZ-HCHK-001, UC-AMZ-HCHK-003).
2. Amazon answers the product page with a response offered as a file download.
3. The browser refuses the download: no file from that response, complete or partial, is written in the Amazon browser's persistent profile or in the Downloads folder of the account running the server.
4. The Downloads folder is not created if it did not exist.
5. The check is a counted failure as in UC-AMZ-HCHK-005 (FR-AMZ-HCHK-011): logged with the request URL and the browser's error, never recorded as "price not found", and the item's last successful price and second-hand offer are kept.

## Alternatives and errors

- **Same check repeated** (scheduled, **Refresh all prices** or **Refresh price**) and answered again as a download: each repetition is a counted failure and adds no file on the server's disk.
- **Fifth consecutive failure:** with 4 earlier consecutive failures, this check stops every tracked Amazon item, as in UC-AMZ-HCHK-005.
- **Next check answered with the normal page:** the check of the same or another item behaves as if the download had never been offered; the price is detected and recorded (UC-AMZ-HCHK-001).
- **Protected mode:** the same applies to the Amazon items of all users.

## Postconditions

- The Amazon browser's persistent profile holds no file with the response offered as a download, and the Downloads folder of the account running the server has no new file (and still does not exist if it did not before).
- The check result, failure count, stop rules, item states and messages are those of UC-AMZ-HCHK-005; nothing else changes.
