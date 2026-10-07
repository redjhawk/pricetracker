# Functional specification: Amazon AI review

Status: ready
Owner: functional specification agent
User decision/reference: GitHub issue #56 (redjhawk) and clarifications D-1–D-21 recorded in the [change index](../../changes/2026-10-07-1117-issue-56-amazon-searches/index.md).

## Purpose and scope

Have Claude review each Amazon search item, judging the price with regard to the product characteristics (D-9), using the user's Claude token saved in Settings ([Claude token settings](../claude-token-settings/functional.md)).

Included: automatic review of search items; display; review after a price change; failures; no token.

Excluded: purchase goals (D-9 "no goal for now"); reviews of tracked Amazon items that never belonged to a search; photo condition analysis.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-AMZ-AIR-001 | An item's data has been read during search work ([FR-AMZ-HUMAN-005](../amazon-human-browsing/functional.md)) and the item has no review yet, and the owner has a Claude token. | Request a review as soon as the information is obtained (D-11), asynchronously; search work is not delayed by the review. | After an item is read, its review becomes pending, then available. |
| FR-AMZ-AIR-002 | A review is requested. | Send the item information obtained from Amazon: title, price, characteristics and description, and the recorded price history. | The review references the item's price and characteristics. |
| FR-AMZ-AIR-003 | A review completes. | Written in English (D-13). Judges the price with regard to the characteristics: a price rating (Good deal / Fair / Overpriced) with explanation (D-9, issue #56 "must consider price"). | Each review shows a price rating and an explanation in English. |
| FR-AMZ-AIR-004 | Item display. | The item's details page shows an **AI review** section with the latest review, its date/time and the price it was based on. The search item list shows the price rating, or pending / failed / none. | A reviewed item shows "Good deal" in the list and the full text on its details page. |
| FR-AMZ-AIR-005 | A later check records a price different from the price of the latest review. | No new review is requested automatically. The AI review section and the list indicate that the price has changed and that the last AI review was done with the older price, showing that price (D-14). | Given a review at 100 € and a new price 80 €, the item shows "Price changed: last AI review was based on 100,00 €". |
| FR-AMZ-AIR-006 | The user restarts after a stop ([FR-AMZ-HUMAN-010](../amazon-human-browsing/functional.md)). | Items are checked again, review included (D-20): an item without a successful review is reviewed; earlier reviews are kept. | After restart, an item whose review failed gets a review. |
| FR-AMZ-AIR-007 | A review attempt fails (token rejected, usage limit, network, AI error). | Show the failure with its date/time; keep the latest successful review. A rejected token updates the Settings warning (FR-CLT-SET-009). Review failures are not Amazon failures and do not count towards FR-AMZ-HUMAN-007. | A Claude 401 shows a failed review and the Settings warning; Amazon requests continue. |
| FR-AMZ-AIR-008 | The owner has no Claude token. | No review is attempted; the section says "Configure a Claude token in Settings". Prices are still tracked. | Without a token, items have prices and no review. |
| FR-AMZ-AIR-009 | Item shared by several searches or tracked list (FR-AMZ-SEARCH-009). | One review history for the item, shown everywhere it appears; it is reviewed once, not once per search. | A product in two searches gets one review. |
| FR-AMZ-AIR-010 | Accessibility. | Ratings and the price-changed notice are conveyed in text, not colour alone; pending and error states are announced (as FR-LBC-AIR-013). | Screen reader reads the rating and notice. |

## States and corner cases

- No review yet, pending, available, failed, price changed since review, no token.
- Item moved to tracked Amazon items keeps its reviews; the section stays on its details page.
- The token used is the search owner's (FR-SETTINGS-001).

## Open questions

None.

## Traceability

- Request and decisions: [change index](../../changes/2026-10-07-1117-issue-56-amazon-searches/index.md).
- Related: [LeBoncoin AI review](../leboncoin-ai-review/functional.md), [Claude token settings](../claude-token-settings/functional.md), [Amazon searches](../amazon-searches/functional.md), [Amazon human browsing](../amazon-human-browsing/functional.md).
- Technical handoff target: `doc/specifications/amazon-ai-review/technical.md`.
