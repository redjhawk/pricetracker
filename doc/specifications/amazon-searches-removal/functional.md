# Functional specification: Amazon searches removal

Status: ready
Owner: functional specification agent
User decision/reference: GitHub issue #111 (redjhawk), decisions D-1–D-6 in the [change index](../../changes/2026-11-10-0141-issue-111-delete-amazon-searches/index.md).

## Purpose and scope

Remove the Amazon searches feature ([superseded spec](../amazon-searches/functional.md)) because it does not work. Actor: any user, open or protected mode.

Included: tab, searches, search-only items and their data, search pacing, Amazon AI reviews. Excluded: tracked Amazon items (unchanged, D-3), LeBoncoin, the failure stop (see [Amazon request stop](../amazon-request-stop/functional.md)).

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-AMZ-SRCH-DEL-001 | Main page is shown. | No **Amazon searches** tab; only Amazon and LeBoncoin tabs remain (D-1). | The tab bar shows Amazon and LeBoncoin only. |
| FR-AMZ-SRCH-DEL-002 | Upgrade of an existing installation. | All saved searches and their search-to-item links are deleted for every user (D-1). | After upgrade no search exists and no screen or API adds, lists, refreshes or deletes one. |
| FR-AMZ-SRCH-DEL-003 | An item belongs only to searches and was never moved to the tracked Amazon list. | The item is deleted with its price history, collection outcomes and AI reviews (D-2). | After upgrade the item no longer exists anywhere. |
| FR-AMZ-SRCH-DEL-004 | A tracked Amazon item, including one moved from a search or also in a search. | Kept with its whole price history; works as before (D-3). | Still listed in the Amazon tab with the same history and refreshed as before. |
| FR-AMZ-SRCH-DEL-005 | Amazon AI reviews. | No Amazon item is reviewed or shows an AI review; stored Amazon reviews are removed (D-2, D-5). LeBoncoin reviews unchanged. | Amazon item details show no AI review; LeBoncoin reviews still appear. |
| FR-AMZ-SRCH-DEL-006 | Search request windows and random pauses. | Removed (D-6). Tracked Amazon items keep the 08:00/20:00 schedule and manual refresh. | No Amazon request depends on the 22:00–01:00 / 06:00–08:00 windows. |
| FR-AMZ-SRCH-DEL-007 | The former global Amazon stop is active at upgrade. | Handled by [FR-AMZ-STOP-009](../amazon-request-stop/functional.md). | See FR-AMZ-STOP-009. |

## States and corner cases

- Upgrade with no searches: nothing deleted; tracked items unchanged.
- An item in several searches and also tracked: kept (FR-AMZ-SRCH-DEL-004).
- An old link to the searches tab opens the normal main page (Amazon tab).

## Open questions

None.

## Traceability

- Supersedes [Amazon searches](../amazon-searches/functional.md), [Amazon human browsing](../amazon-human-browsing/functional.md) (pacing), [Amazon AI review](../amazon-ai-review/functional.md).
- Technical handoff target: `doc/specifications/amazon-searches-removal/technical.md`.
