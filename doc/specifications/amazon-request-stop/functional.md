# Functional specification: Amazon request stop

Status: ready
Owner: functional specification agent
User decision/reference: GitHub issue #111 (redjhawk), decision D-4 in the [change index](../../changes/2026-11-10-0141-issue-111-delete-amazon-searches/index.md); keeps logging rules FR-AMZ-HUMAN-006/012 of [Amazon human browsing](../amazon-human-browsing/functional.md).

## Purpose and scope

Stop checking tracked Amazon items when Amazon keeps failing, and let the user restart them one by one with **Refresh price**.

Current behavior (issue #56): 5 consecutive failed Amazon requests set one application-wide stop; every Amazon request is refused until **Refresh** on a search; the Amazon tab shows a warning pointing to the Amazon searches tab.

Excluded: LeBoncoin items (never stopped by Amazon failures); AI reviews.

**Failure:** an Amazon request with no usable Amazon answer (non-success response, challenge, network error). A price not found or an unavailable listing is an Amazon answer, not a failure (as today).

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-AMZ-STOP-001 | A tracked Amazon item request fails. | Log request URL, time, HTTP status and the server's error response (as FR-AMZ-HUMAN-006). Logs never contain tokens or session secrets (FR-AMZ-HUMAN-012). | A 503 is logged with status and response body, no secret. |
| FR-AMZ-STOP-002 | Amazon requests complete. | Count consecutive failures across all tracked Amazon items; any Amazon answer resets the count to 0. | 4 failures, 1 success, 4 failures stop nothing. |
| FR-AMZ-STOP-003 | The count reaches 5. | Every tracked Amazon item becomes **stopped**, each with its own stopped state and stop date/time (D-4). Log the stop. Remaining Amazon checks of the running batch are not sent. | After the 5th failure all tracked Amazon items are stopped. |
| FR-AMZ-STOP-004 | Scheduled 08:00/20:00 check or **Refresh all prices**. | Stopped Amazon items are skipped; other Amazon items and LeBoncoin items are checked normally (D-4). | With all Amazon items stopped, Refresh all prices sends no Amazon request and still checks LeBoncoin. |
| FR-AMZ-STOP-005 | At least one Amazon item is stopped. | The Amazon tab shows a warning: Amazon checks were stopped after repeated failures (stop date/time); restart each item with **Refresh price** on its details page (D-4). It no longer mentions a searches tab. | Warning visible in the Amazon tab; gone when no item is stopped. |
| FR-AMZ-STOP-006 | The user chooses **Refresh price** on a stopped item. | The item is checked although stopped. On an Amazon answer only that item restarts: stopped state cleared, price recorded in its history, count reset; other stopped items stay stopped (D-4). | Refresh one of 3 stopped items succeeds: 1 restarted, 2 still stopped. |
| FR-AMZ-STOP-007 | That manual check fails. | The item stays stopped; the failure is logged and counted; the details page shows the refresh failed. | A failed Refresh price leaves the item stopped and the warning shown. |
| FR-AMZ-STOP-008 | A restarted item. | It is checked again by schedule and Refresh all prices; it is stopped again only on a new count of 5. | A restarted item is checked at the next 08:00. |
| FR-AMZ-STOP-009 | Upgrade while the former global stop is active. | All tracked Amazon items become stopped with the former stop date/time, then follow this specification. | After upgrade the warning shows and each item restarts with its own Refresh price. |
| FR-AMZ-STOP-010 | Application restart. | Stopped states, stop dates and the count persist. | A stopped item is still stopped after restart. |

## States and corner cases

- Item states: not stopped, stopped, refresh in progress, refresh failed.
- Deleting a stopped item removes it; the warning disappears if it was the last stopped one.
- A stop or restart never loses price history.
- As today, the stop is application-wide in effect (Amazon refuses the device, not a user): in protected mode the 5th failure stops the tracked Amazon items of all users.

## Open questions

None blocking. Interpretation recorded (conservative, from "each item has its own stopped state"): an Amazon item added while other items are stopped is not stopped; its immediate first check is attempted and counts toward FR-AMZ-STOP-002.

## Traceability

- Replaces stop rules FR-AMZ-HUMAN-007–011 of [Amazon human browsing](../amazon-human-browsing/functional.md).
- Related: [item refresh](../item-refresh/functional.md), [platform tabs](../platform-tabs/functional.md).
- Technical handoff target: `doc/specifications/amazon-request-stop/technical.md`.
