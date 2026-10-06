# Functional specification: LeBoncoin AI review

Status: ready
Owner: functional specification agent
User decision/reference: GitHub issue #4 "IA review of LeBoncoin items" (2026-10-04) and user decisions D-4–D-12 recorded in the [change index](../../changes/2026-10-04-0710-issue-4-leboncoin-ai-review/index.md).

## Purpose and scope

Obtain an AI (Claude) review of each LeBoncoin item and show it on the item details page, with a refresh button. Actor: the operator. Reviews are produced server-side with the token from [Claude token settings](../claude-token-settings/functional.md).

Included: automatic review when a LeBoncoin item is added and when its recorded price changes; manual refresh; sending all listing information to the AI; structured review; review history; pending, empty and error states.

Excluded: Amazon items; showing reviews in the items list; notifications; contacting sellers; automatic review of items added before this feature.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-LBC-AIR-001 | A new LeBoncoin item is accepted (FR-01, FR-14) and a Claude token is saved. | Automatically request a review asynchronously; adding the item is never delayed or failed by the review. | After adding a LeBoncoin item, its details page shows a pending review, then the completed review, without operator action. |
| FR-LBC-AIR-002 | A review is requested. | Send all listing information available from LeBoncoin: title, description, price, all photos, characteristics/attributes (e.g. brand, model, seller-stated condition, category attributes), location, publication date and seller type when available, plus the item's recorded price history (D-12). | A review can reference the description, characteristics, photos and price history. |
| FR-LBC-AIR-003 | A review completes. | Written in English (D-4). Contains: **Price** rating — Good deal / Fair / Overpriced — with explanation based on Claude's general market knowledge and the recorded price history (D-12); **Condition** rating — Excellent / Good / Fair / Poor — with explanation from the photos, compared with the seller's stated condition when given; **Overall recommendation** — Buy / Negotiate / Avoid — with explanation (D-6/7). | The details page shows the Price and Condition ratings, the overall recommendation, and their explanations. |
| FR-LBC-AIR-014 | A review completes. | Also contains (D-8): estimated fair price range and suggested offer; scam/risk signs (or a statement that none were found); missing accessories/information and questions to ask the seller; whether the description matches the photos, with mismatches listed. | Each of the four parts is shown, labelled, on the details page. |
| FR-LBC-AIR-004 | LeBoncoin item details page. | Show an "AI review" section with the latest review, the date/time it was produced and the listing price it was based on. Not shown for Amazon items. | Given a reviewed item, the section shows review, timestamp and reviewed price. Amazon details pages have no such section. |
| FR-LBC-AIR-005 | LeBoncoin details page, token saved, no review running for this item. | Always provide a "Refresh AI review" button requesting a new review from current listing data, independent of the existing "Refresh price" action (D-5). | Clicking shows progress, then the new review and timestamp. |
| FR-LBC-AIR-006 | A review for the item is running. | No duplicate review; the button is disabled and progress is shown. | Repeated clicks or a price change during a run produce at most one running review. |
| FR-LBC-AIR-007 | A review attempt fails (token rejected, usage limit, network, AI error, listing unretrievable). | Show an understandable error with its date/time in the section; keep showing the latest successful review, labelled with its own date; the button stays available to retry. No token in errors (FR-CLT-SET-008). | Given a previous review and a failed refresh, the old review stays visible with an error notice. |
| FR-LBC-AIR-008 | No Claude token is saved. | No review is attempted (automatic or manual). The section says "Configure a Claude token in Settings" and the refresh button is disabled (D-9). Saving a token later does not review items automatically. | Without a token, adding an item creates no review; the message is shown and refresh is disabled. |
| FR-LBC-AIR-009 | A price check (automatic or manual) records a price different from the previous recorded price, and a token is saved. | Automatically request a new review (D-5). Until it completes, the section indicates when the shown review was made at an older price. | After a price change, a new review appears based on the new price. |
| FR-LBC-AIR-010 | LeBoncoin items added before this feature. | No automatic review; the section shows that no review exists yet and the refresh button produces one (D-10). | An existing item shows "no review yet"; clicking refresh produces a review. |
| FR-LBC-AIR-011 | A new review succeeds while others exist. | Keep all successful reviews (D-11). The details page lets the operator view previous reviews, newest first, each with its date/time and reviewed price. | After two reviews, both are viewable on the details page. |
| FR-LBC-AIR-012 | An item is deleted. | Its reviews are deleted with it (FR-09; §10 no retention after deletion). | No review of the item remains after deletion. |
| FR-LBC-AIR-013 | Keyboard, assistive technology, ~400 px viewport. | Section has a heading; pending/error states are announced; ratings conveyed in text, not colour alone; history reachable by keyboard; readable without horizontal scrolling. | Screen reader reaches the "AI review" heading and hears state changes. |

## States and corner cases

- No review yet; pending; completed; failed with or without previous review; no token; review made at an older price; several reviews in history.
- Listing without photos: Condition states it cannot be assessed from photos.
- Free ("Gratuit") listing: the price part states the item is free.
- Listing removed/unretrievable at review time: FR-LBC-AIR-007.
- Token replaced or removed during a running review: that review finishes or fails with the token it started with.
- A recorded price unchanged by a price check does not trigger a review.

## Open questions

None.

## Traceability

- Request and decisions: [change index](../../changes/2026-10-04-0710-issue-4-leboncoin-ai-review/index.md) D-4–D-12.
- [Functional specifications](../../FUNCTIONAL_SPECIFICATIONS.md) §5.3, §6, §7, FR-01, FR-09, FR-14, FR-18, FR-19.
- [Item refresh](../item-refresh/functional.md), [platform tabs](../platform-tabs/functional.md), [tracked items identity](../tracked-items-identity/functional.md), [Claude token settings](../claude-token-settings/functional.md).
- The current collector (`internal/leboncoin/collector.go`) keeps only title, price and first image; FR-LBC-AIR-002 requires collecting description, all photos, attributes, location and other listing fields (technical consequence).
- Technical handoff target: `doc/specifications/leboncoin-ai-review/technical.md`.
