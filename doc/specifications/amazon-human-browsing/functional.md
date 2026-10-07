# Functional specification: Amazon human browsing

Status: ready
Owner: functional specification agent
User decision/reference: GitHub issue #56 (redjhawk) and clarifications D-1–D-21 recorded in the [change index](../../changes/2026-10-07-1117-issue-56-amazon-searches/index.md).

## Purpose and scope

Retrieve Amazon searches and their items the way a person would: only at allowed hours, slowly, in a fixed order, and stop all Amazon requests when Amazon keeps failing. "There's no hurry"; the order of actions "IS REALLY IMPORTANT" (issue #56).

Included: request windows and pauses for search work; order of actions; failure counting, stop of all Amazon requests, logging, user information and restart.

Excluded: changes to the schedule of tracked Amazon items and LeBoncoin items, other than the stop on failures (D-16, D-17). LeBoncoin requests are never stopped by Amazon failures.

Terms:

- **Search work:** retrieving a search results page and checking its items ([Amazon searches](../amazon-searches/functional.md)), including their first check and later refreshes.
- **Amazon request:** any request from the application to Amazon, for search work or for tracked Amazon items.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-AMZ-HUMAN-001 | Search work is due. | Search work runs only between 22:00 and 01:00 and between 06:00 and 08:00, `Europe/Paris` time (D-16). Work not finished at the end of a window resumes at the next window. | No search request is sent at 01:05 or 12:00 Paris time; work pending at 01:00 continues at 06:00. |
| FR-AMZ-HUMAN-002 | Tracked Amazon items and LeBoncoin items. | The request windows do not apply to them; they keep their existing schedule and manual refresh (D-16). | A tracked Amazon item is still checked at 08:00 and 20:00 and on manual refresh. |
| FR-AMZ-HUMAN-003 | Between two consecutive search-work actions. | Wait a random pause between 30 seconds and 2 minutes (D-17). | Logged action times show gaps of 30–120 s, not constant. |
| FR-AMZ-HUMAN-004 | A search is processed. | Follow this order, one item at a time: (1) open the search results; (2) pause; (3) open one item's link; (4) pause while reading all of its data; (5) close the item; (6) pause; (7) go to the next item, repeating 3–6 (D-11, issue #56). Never open several items at once and never process two searches at the same time. | Logs show, for each item, open → read → close before the next item is opened. |
| FR-AMZ-HUMAN-005 | An item's data is read (step 4). | As soon as the information is obtained, its price is recorded and its AI review is requested ([FR-AMZ-AIR-001](../amazon-ai-review/functional.md)) (D-11). | After an item's data is read, its price appears and its review is pending. |
| FR-AMZ-HUMAN-006 | An Amazon request fails (non-success response, challenge, network error, unparsable page). | Log the failure with the request URL, the time, the HTTP status and the error response returned by the server (issue #56). The failed request is put aside and tried again once all other pending requests of the run have been done (D-18). A success continues the work normally. | A 503 is logged with its status and response body; the failed item is retried after the remaining items. |
| FR-AMZ-HUMAN-007 | Failures are counted. | Count consecutive failed Amazon requests (search work and tracked Amazon items). A success resets the count (D-18: "if there's a success, continue"). | 4 failures, 1 success, 4 failures do not stop requests. |
| FR-AMZ-HUMAN-008 | The count reaches 5 failures. | Stop all Amazon requests, including tracked Amazon item checks, scheduled or manual (D-17). Log the stop. No Amazon request is sent until the user restarts ([FR-AMZ-HUMAN-010](#requirements)). | After 5 failures, no Amazon request is sent, even at 08:00 for tracked items. |
| FR-AMZ-HUMAN-009 | Amazon requests are stopped. | In the list of searches, each search with unfinished work shows a **Refresh** button and, below it, an information line saying that Amazon requests were stopped after repeated failures, with the stop date/time (D-19). Tracked Amazon item checks show a clear stopped state instead of being silently skipped. | After a stop, the search row shows the button and the information line. |
| FR-AMZ-HUMAN-010 | The user chooses **Refresh** on a stopped search. | Restart Amazon requests and reset the failure count: check the search items again, AI reviews included, following FR-AMZ-HUMAN-001–008. Price history and existing reviews are never lost (D-20). The information line disappears once a refresh succeeds; it stays if requests fail again (D-19). | After a successful restart, the line disappears and new prices are added to the existing history. |
| FR-AMZ-HUMAN-011 | Refresh is chosen outside a request window. | The restart is accepted and the work starts at the next window (FR-AMZ-HUMAN-001); the search shows it is waiting. | Refresh at 14:00 shows "waiting" until 22:00. |
| FR-AMZ-HUMAN-012 | Logs. | Logs never contain the Claude token or session secrets ([FR-CLT-SET-008](../claude-token-settings/functional.md)). | A failure log shows the server response but no token. |

## States and corner cases

- Running, waiting for window, paused between actions, stopped after failures, restarted.
- Application restart during a window: pending work continues in the same order; a stop state persists across restart.
- A search deleted while being processed: no further requests for it.
- Several users' searches: processed one after another; the stop applies to all Amazon requests of the application, since Amazon refuses the device, not a user.
- A run that lasts longer than the windows continues over several days.

## Open questions

None. Interpretation recorded: "5 failures" counts consecutive failed Amazon requests; a success resets it (D-18 "If there's a success, continue").

## Traceability

- Request and decisions: [change index](../../changes/2026-10-07-1117-issue-56-amazon-searches/index.md).
- Related: [Amazon searches](../amazon-searches/functional.md), [Amazon AI review](../amazon-ai-review/functional.md), [item refresh](../item-refresh/functional.md).
- Technical handoff target: `doc/specifications/amazon-human-browsing/technical.md`.
