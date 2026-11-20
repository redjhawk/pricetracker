# Functional specification: Amazon human-like checks

Status: draft, waiting for answers (issue #168)
Owner: functional specification agent
User decision/reference: GitHub issue #168 ("Amazon checks were stopped on 20 Nov 2026, 07:26 UTC after repeated failures"). Open questions: [questions](../../changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n/questions.md).

## Purpose and scope

Amazon stopped answering the tracked-item checks, probably because it detects them as a bot. The user asks that Amazon pages be requested by a real web browser and that checks no longer happen at the same time every day, so that requests look like a person's.

Included: how tracked Amazon item pages are requested; when scheduled Amazon checks happen.

Excluded: the failure count, stop, warning and restart rules ([Amazon request stop](../amazon-request-stop/functional.md)), which stay unchanged; price and second-hand offer detection; AI reviews. LeBoncoin items, the check windows and the items stopped before this change are pending the open questions.

Actor: the user who tracks Amazon items. Amazon is the external system.

Terms:

- **Amazon check:** retrieval of a tracked Amazon item's page: first check after add, scheduled check, **Refresh all prices**, **Refresh price**.
- **Real web browser:** a full web browser engine that loads and renders the page as it would for a person (page scripts, page resources, cookies), as opposed to a plain download of the page.
- **Failure:** as defined in [Amazon request stop](../amazon-request-stop/functional.md): no usable Amazon answer.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-AMZ-HCHK-001 | Any Amazon check. | The Amazon page is requested and read through a real web browser, not a plain page download (issue #168). | The request reaching Amazon comes from a real browser engine that runs the page's scripts and loads its resources; no Amazon check is made with a plain HTTP download. |
| FR-AMZ-HCHK-002 | An Amazon check through the browser gets a usable Amazon answer. | The results are the same as today: price, second-hand offer sold by Amazon, price not found, listing unavailable ([FR-AMAZON-PRICE-*](../amazon-price-detection/functional.md), FR-15). | For the same listing page, the stored price, offer and condition match those of the current behaviour. |
| FR-AMZ-HCHK-003 | The browser gets no usable Amazon answer (non-success response, challenge, network error) or cannot open the page. | It is a failure: logged, counted and stopping items exactly as in FR-AMZ-STOP-001–010. The last successful price is kept (FR-07). | 5 consecutive browser failures stop every tracked Amazon item and show the Amazon tab warning. |
| FR-AMZ-HCHK-004 | Scheduled Amazon checks. | Each tracked Amazon item is still checked twice per day, but not at the same time every day: each scheduled check happens at a randomly chosen time that changes from day to day (issue #168). The time range is open (question 1). | Over 7 days, the logged times of an item's scheduled checks are not identical from one day to the next, and there are two scheduled checks per day. |
| FR-AMZ-HCHK-005 | **Refresh price**, **Refresh all prices**, first check after add. | Not delayed by the random schedule: they start immediately, as today (FR-14, FR-17, FR-18). Stopped items are still skipped by Refresh all prices (FR-AMZ-STOP-004). | Refresh price on an item starts its check at once and shows progress. |
| FR-AMZ-HCHK-006 | Application restart. | A scheduled check not yet run keeps two checks per day for the item; no day is skipped and no check is duplicated because of the restart. | After a restart, the item still gets two scheduled checks that day. |

## States and corner cases

- Item states and messages are unchanged (pending, active, retrieval issue, unavailable, stopped).
- A check that starts while a previous check of the same item is in progress is not duplicated (as today).
- The browser is never shown to the user and never needs user action; no Amazon login is used.
- Logs never contain tokens or session secrets (FR-AMZ-HUMAN-012).

## Open questions

See [questions](../../changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n/questions.md): check time range, LeBoncoin scope, spreading of items within a check run, items stopped before the change.

## Traceability

- Related: [Amazon request stop](../amazon-request-stop/functional.md), [item refresh](../item-refresh/functional.md), [Amazon price detection](../amazon-price-detection/functional.md).
- Technical handoff target: `doc/specifications/amazon-human-checks/technical.md`.
