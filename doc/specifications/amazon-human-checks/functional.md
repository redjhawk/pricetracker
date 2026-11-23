# Functional specification: Amazon human-like checks

Status: ready (issue #168); FR-AMZ-HCHK-011–012 ready (issue #186)
Owner: functional specification agent
User decision/reference: GitHub issue #168 ("Amazon checks were stopped on 20 Nov 2026, 07:26 UTC after repeated failures") and the answers by redjhawk on 2026-11-20 to the questions of the change: (1) windows 06:00–09:00 and 19:00–23:00; (2) Amazon only; (3) each item its own random time; (4) stopped items restarted once automatically. GitHub issue #186 ("Treat a 2xx navigation with errorText as a failed Amazon check", from review comment R5-01 on #179) for FR-AMZ-HCHK-011–012.

## Purpose and scope

Amazon stopped answering the tracked-item checks, probably because it detects them as a bot. The user asks that Amazon pages be requested by a real web browser and that checks no longer happen at the same time every day, so that requests look like a person's.

Included: how tracked Amazon item pages are requested; when scheduled Amazon checks happen; a one-time restart of the Amazon items stopped before this change.

Excluded: LeBoncoin items, which keep their current requests and schedule (08:00 and 20:00 by default); the failure count, stop, warning and restart rules ([Amazon request stop](../amazon-request-stop/functional.md)), which stay unchanged; price and second-hand offer detection; AI reviews.

Actor: the user who tracks Amazon items. Amazon is the external system.

Terms:

- **Amazon check:** retrieval of a tracked Amazon item's page: first check after add, scheduled check, **Refresh all prices**, **Refresh price**.
- **Real web browser:** a full web browser engine that loads and renders the page as it would for a person (page scripts, page resources, cookies), as opposed to a plain download of the page.
- **Morning window:** 06:00 to 09:00; **evening window:** 19:00 to 23:00. Times are in the schedule timezone (`Europe/Paris` by default).
- **Failure:** as defined in [Amazon request stop](../amazon-request-stop/functional.md): no usable Amazon answer.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-AMZ-HCHK-001 | Any Amazon check. | The Amazon page is requested and read through a real web browser, not a plain page download (issue #168). | The request reaching Amazon comes from a real browser engine that runs the page's scripts and loads its resources; no Amazon check is made with a plain HTTP download. |
| FR-AMZ-HCHK-002 | An Amazon check through the browser gets a usable Amazon answer. | The results are the same as today: price, second-hand offer sold by Amazon, price not found, listing unavailable ([FR-AMAZON-PRICE-*](../amazon-price-detection/functional.md), FR-15). | For the same listing page, the stored price, offer and condition match those of the current behaviour. |
| FR-AMZ-HCHK-003 | The browser gets no usable Amazon answer (non-success response, challenge, network error) or cannot open the page. | It is a failure: logged, counted and stopping items exactly as in FR-AMZ-STOP-001–010. The last successful price is kept (FR-07). | 5 consecutive browser failures stop every tracked Amazon item and show the Amazon tab warning. |
| FR-AMZ-HCHK-004 | Scheduled Amazon checks. | Each tracked Amazon item gets two scheduled checks per day: one at a random time within the morning window and one at a random time within the evening window. The times are drawn anew every day (issue #168, answer 1). The configured check times (08:00 and 20:00 by default) no longer apply to Amazon items. | Over 7 days, each item has exactly one scheduled check per day in 06:00–09:00 and one in 19:00–23:00, and its check times differ from one day to the next. |
| FR-AMZ-HCHK-005 | Several tracked Amazon items. | Each item's scheduled time is drawn independently of the other items; items are not checked together in one run (answer 3). | With 10 Amazon items, their scheduled check times within the same window are spread over the window instead of following each other within one run. |
| FR-AMZ-HCHK-006 | An Amazon item is added. | Its first check is immediate (FR-14). Its scheduled checks start with the next window that begins after the add. | An item added at 07:00 is checked at once, then in the evening window of the same day, not again in that morning window. |
| FR-AMZ-HCHK-007 | **Refresh price**, **Refresh all prices**, first check after add. | Not delayed by the windows: they start immediately, as today (FR-14, FR-17, FR-18), and do not change the item's scheduled checks. Stopped items are still skipped by Refresh all prices (FR-AMZ-STOP-004). | Refresh price on an item at 14:00 starts its check at once and shows progress; the item still gets its evening scheduled check. |
| FR-AMZ-HCHK-008 | Stopped Amazon items. | Stopped items are skipped at their scheduled time (FR-AMZ-STOP-004); a stop during a window prevents the remaining scheduled checks of all stopped items. | After the 5th failure, no Amazon request is sent for stopped items at their later scheduled times. |
| FR-AMZ-HCHK-009 | Application restart. | A scheduled check whose time has not yet passed still happens within its window; a scheduled check already made in a window is not repeated in that window because of the restart. | Restarting at 07:00 after the item's morning check gives no second morning check; restarting at 06:30 before it still gives one morning check. |
| FR-AMZ-HCHK-010 | First start after this change is installed, with Amazon items stopped (such as on 20 Nov 2026). | Every stopped Amazon item is restarted once automatically: stopped state cleared, failure count reset to 0, price history kept. Each is then checked at its next scheduled time (answer 4). This happens only once; later stops follow [Amazon request stop](../amazon-request-stop/functional.md). | After installation, the Amazon tab warning is gone, no item shows as stopped, and each item gets its next scheduled check in the next window. A stop occurring after installation is not cleared by a later restart of the application. |
| FR-AMZ-HCHK-011 | An Amazon check gets a success response (HTTP 200–299) that the browser does not open as a page, such as a response offered as a file download or a response without content (HTTP 204, 205) (issue #186). | It is a failure, as in FR-AMZ-HCHK-003: logged with the request URL and the browser's error, counted toward the 5 consecutive failures, never recorded as "price not found" or as another Amazon answer, so it does not reset the failure count. The item's last successful price and second-hand offer are kept. | A listing answered with HTTP 200 `text/html` and `Content-Disposition: attachment` gives a logged, counted failure (retrieval issue, second-hand check error), not "price not found"; 4 earlier failures followed by this check stop every tracked Amazon item. |
| FR-AMZ-HCHK-012 | An Amazon check gets a non-success response (outside HTTP 200–299) that the browser does not open as a page (for example an error status with an empty body). | Unchanged: the response status decides the result: 404 or 410 → listing unavailable (an Amazon answer); any other status → failure logged with that HTTP status and counted. | A 404 with an empty body shows the listing as unavailable and resets the count; a 503 with an empty body is a counted failure logged with status 503. |

## States and corner cases

- Item states and messages are unchanged (pending, active, retrieval issue, unavailable, stopped).
- A check that starts while a previous check of the same item is in progress is not duplicated (as today).
- The browser is never shown to the user and never needs user action; no Amazon login is used.
- Logs never contain tokens or session secrets (FR-AMZ-HUMAN-012).
- Deleting an item cancels its scheduled checks.
- In protected mode the windows and the one-time restart apply to the Amazon items of all users.

## Open questions

None. Answers recorded in FR-AMZ-HCHK-004, -005, -010 and in the scope (LeBoncoin excluded). Issue #186 needed no question: it states the expected result (FR-AMZ-HCHK-011); FR-AMZ-HCHK-012 keeps the current behaviour.

## Traceability

- Related: [Amazon request stop](../amazon-request-stop/functional.md), [item refresh](../item-refresh/functional.md), [Amazon price detection](../amazon-price-detection/functional.md).
- Technical handoff target: `doc/specifications/amazon-human-checks/technical.md`.
