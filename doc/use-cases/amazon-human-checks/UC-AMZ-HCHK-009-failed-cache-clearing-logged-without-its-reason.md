# UC-AMZ-HCHK-009 — Failed browser cache clearing is logged without the reason it was started

**Primary actor:** PriceFollower application, on behalf of the user who tracks Amazon items
**Secondary actor:** Person who runs the server (reads the server log)
**Supporting systems:** Real web browser, Amazon, Amazon request gate
**Trigger:** During an Amazon check (first check after add, scheduled check, **Refresh all prices** or **Refresh price**), the application clears the Amazon browser cache because the product page was not opened (for example offered as a file download, UC-AMZ-HCHK-007) or because the check was interrupted while the browser was still requesting the product page (UC-AMZ-HCHK-008), and the clearing fails.
**Origin:** Issue #240 (review comment R2-02 on #236): the log line said `after a page not opened` even when the clearing was started because the check was interrupted, which could mislead the reader of the log.
**Covers:** FR-AMZ-HCHK-016 (with FR-AMZ-HCHK-015 for when the line appears and its cause)

## Preconditions

- One or more Amazon items are tracked and not stopped (or the user chose **Refresh price** on a stopped item).
- The browser cannot clear its cache, for example it returns an error or gives no answer within the time limit of the clearing.

## Main flow

1. The application opens the item's Amazon page through the real web browser (UC-AMZ-HCHK-001, UC-AMZ-HCHK-003).
2. Amazon answers the product page with a response the browser does not open as a page, for example offered as a file download (UC-AMZ-HCHK-007).
3. The application asks the browser to clear its cache, and the clearing fails.
4. The server log has exactly one line for this failed clearing: `Amazon browser cache could not be cleared: <cause>`, where `<cause>` is the real cause of the failure (the browser's error, or no answer within the time limit of the clearing).
5. The line does not say why the clearing was started; no log line contains `after a page not opened`.
6. The check result is unchanged: a counted failure as in UC-AMZ-HCHK-005.

## Alternatives and errors

- **Check interrupted while the page is still being requested** (time limit reached or server stopped, UC-AMZ-HCHK-008) and the clearing fails: the log line is the same as in step 4, with the real cause of the clearing failure, never the interruption of the check; no log line contains `after a page not opened`. The check result and the server stop are as today.
- **Clearing succeeds:** the server log has no line saying the browser cache could not be cleared.
- **Protected mode:** the same applies to the Amazon items of all users.

## Postconditions

- Each failed clearing has exactly one log line starting with `Amazon browser cache could not be cleared: ` followed by its real cause.
- The check result, failure count, stop rules, item states, messages, server stop and the download guarantees of UC-AMZ-HCHK-007 and UC-AMZ-HCHK-008 are as today; nothing else changes.
