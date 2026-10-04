# Functional specification: LeBoncoin AI review

Status: needs-clarification
Owner: functional specification agent
User decision/reference: GitHub issue #4 "IA review of LeBoncoin items" (2026-10-04), quoted in the [change index](../../changes/leboncoin-ai-review/index.md). No further decisions recorded yet; see open questions.

## Purpose and scope

Obtain an AI (Claude) review of each LeBoncoin item and show it on the item details page, with a refresh button. Actor: the operator. Reviews are produced server-side with the credential from [Claude token settings](../claude-token-settings/functional.md).

Included: automatic review when a LeBoncoin item is added; manual review refresh on the details page; sending all listing information (characteristics, price, photos, ...) to the AI; review of price, condition from photos and other aspects; display, pending, empty and error states.

Excluded: Amazon items (request limited to LeBoncoin); showing reviews in the items list; notifications; contacting sellers.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-LBC-AIR-001 | A new LeBoncoin item is accepted (FR-01, FR-14) and a Claude credential is saved. | Automatically request an AI review asynchronously; adding the item is never delayed or failed by the review. | After adding a LeBoncoin item, its details page shows a pending review, then the completed review, without operator action. |
| FR-LBC-AIR-002 | A review is requested. | Send all listing information available from LeBoncoin: title, description, price, all photos, characteristics/attributes (e.g. brand, model, seller-stated condition, category attributes), location, publication date and seller type when available. | A review can reference the description, characteristics and photos (e.g. a visible defect or a stated attribute). |
| FR-LBC-AIR-003 | A review completes. | Content includes at least: **Price** — whether it is interesting, fair or too expensive, with reasons; **Condition** — apparent condition from the photos, compared with the seller's stated condition when given, including visible defects or photo limitations; plus other aspects (Q-AIR-4). Structure per Q-AIR-3; basis of the price judgment per Q-AIR-8; language per Q-AIR-1. | The details page shows labelled Price and Condition parts, each with a judgment and its reasons. |
| FR-LBC-AIR-004 | LeBoncoin item details page. | Show an "AI review" section with the review, the date/time it was produced and the listing price it was based on. Not shown for Amazon items. | Given a reviewed item, the section shows review, timestamp and reviewed price. Amazon details pages have no such section. |
| FR-LBC-AIR-005 | LeBoncoin details page, no review running for this item. | Provide a "Refresh AI review" button requesting a new review from current listing data, independent of the existing "Refresh price" action. | Clicking shows progress, then the new review and timestamp. |
| FR-LBC-AIR-006 | A review for the item is running. | No duplicate review; the button is disabled and progress is shown. | Repeated clicks produce at most one review. |
| FR-LBC-AIR-007 | A review attempt fails (credential rejected, usage limit, network, AI error, listing unretrievable). | Show an understandable error with its date/time in the section; keep showing the previous successful review, labelled with its own date; the button stays available. No credential in errors (FR-CLT-SET-008). | Given a previous review and a failed refresh, the old review stays visible with an error notice. |
| FR-LBC-AIR-008 | No Claude credential is saved. | Unresolved (Q-AIR-5). | Pending. |
| FR-LBC-AIR-009 | A price check detects a price different from the reviewed one. | Rerun policy unresolved (Q-AIR-2). In all cases the section shows the reviewed price so the difference is visible. | When the latest price differs, the section indicates the review was made at the older price. |
| FR-LBC-AIR-010 | LeBoncoin items added before this feature. | Unresolved (Q-AIR-6). | Pending. |
| FR-LBC-AIR-011 | A new review succeeds while one exists. | Replace vs. history unresolved (Q-AIR-7). | Pending. |
| FR-LBC-AIR-012 | An item is deleted. | Its reviews are deleted with it (FR-09; §10 no retention after deletion). | No review of the item remains after deletion. |
| FR-LBC-AIR-013 | Keyboard, assistive technology, ~400 px viewport. | Section has a heading; pending/error states are announced; judgments conveyed in text, not colour alone; readable without horizontal scrolling. | Screen reader reaches the "AI review" heading and hears state changes. |

## States and corner cases

- No review yet; pending; completed; failed with or without previous review; no credential; review made at an older price.
- Listing without photos: Condition states it cannot be assessed from photos.
- Free ("Gratuit") listing: the price part states the item is free.
- Listing removed/unretrievable at review time: FR-LBC-AIR-007.
- Credential replaced during a running review: that review finishes or fails with the credential it started with.

## Open questions

- Q-AIR-1: review language — (a) French; (b) English (current interface language); (c) language of the listing.
- Q-AIR-2: rerun on price change — (a) no, manual refresh only; (b) yes, on every price change; (c) only above a threshold (to define).
- Q-AIR-3: structure — (a) free text per aspect; (b) per-aspect verdict label (e.g. Good deal/Fair/Overpriced; Excellent/Good/Fair/Poor) plus explanation; (c) (b) plus an overall score or recommendation (buy/negotiate/avoid).
- Q-AIR-4: other aspects (any of) — (a) estimated fair price range and suggested offer; (b) scam/risk red flags; (c) completeness/accessories, missing information and questions to ask the seller; (d) description/photo consistency; (e) none.
- Q-AIR-5: no credential saved — (a) nothing attempted, section says "Configure a Claude token in Settings", refresh disabled; (b) items without review are reviewed automatically once a credential is saved.
- Q-AIR-6: existing LeBoncoin items — (a) no automatic review, use refresh; (b) reviewed automatically once.
- Q-AIR-7: previous reviews — (a) replaced, latest only; (b) history kept and viewable.
- Q-AIR-8: price judgment basis — (a) AI general market knowledge only; (b) plus the item's tracked price history; (c) plus other tracked items of the same product (e.g. Amazon prices).

## Traceability

- Request: [change index](../../changes/leboncoin-ai-review/index.md).
- [Functional specifications](../../FUNCTIONAL_SPECIFICATIONS.md) §5.3, §6, §7, FR-01, FR-09, FR-14, FR-18, FR-19.
- [Item refresh](../item-refresh/functional.md), [platform tabs](../platform-tabs/functional.md), [tracked items identity](../tracked-items-identity/functional.md), [Claude token settings](../claude-token-settings/functional.md).
- The current collector keeps only title, price and first image; FR-LBC-AIR-002 implies collecting description, all photos and attributes (technical consequence).
- Technical handoff target: `doc/specifications/leboncoin-ai-review/technical.md`.
