# Functional specification: Amazon searches

Status: ready
Owner: functional specification agent
User decision/reference: GitHub issue #56 (redjhawk) and clarifications D-1–D-21 recorded in the [change index](../../changes/2026-10-07-1117-issue-56-amazon-searches/index.md).

## Purpose and scope

Let a user track an Amazon search (a filtered results page such as an Amazon "joursprime" deals URL) and follow the price of its first 30 items, so that Black Friday offers can be reviewed by the application instead of manually.

Actors: the logged-in user in protected mode, or the operator in open mode ([multi-user login](../multi-user-login/functional.md)).

Included: a new **Amazon searches** tab; adding a search; the list of searches; the items of a search (first 30, frozen); item display, price and price history like tracked Amazon items; unreachable items; sharing an item between searches and the tracked Amazon list; moving an item to the tracked Amazon list; deleting a search.

Excluded (D-4, D-7, D-8): renaming or editing a search; changing the item set after the first capture; parameters or extra filters; moving an item from the tracked list to a search; purchase goals for search items. Request timing, pauses, order of actions and failure handling are in [Amazon human browsing](../amazon-human-browsing/functional.md); AI reviews are in [Amazon AI review](../amazon-ai-review/functional.md).

Terms:

- **Search:** an Amazon results URL added by a user.
- **Search item:** one of the first 30 products captured from a search.
- **Tracked Amazon item:** an item of the existing Amazon tab ([platform tabs](../platform-tabs/functional.md)).

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-AMZ-SEARCH-001 | The user opens the tracked-items page. | An **Amazon searches** tab is available next to the existing Amazon and LeBoncoin tabs (D-1). The existing Amazon tab is kept unchanged as the list of tracked Amazon items (D-21). | Given the tracked-items page, then three tabs are shown: Amazon, LeBoncoin, Amazon searches. |
| FR-AMZ-SEARCH-002 | The Amazon searches tab is selected. | Show the list of the user's searches, each with its URL (or a readable label derived from it), its date added, its number of captured items and its state (waiting to run, running, done, stopped). | Given two added searches, the tab lists both with these fields. |
| FR-AMZ-SEARCH-003 | The user adds a search with an Amazon URL. | Accept an `http(s)` Amazon results URL on a supported Amazon marketplace; reject other URLs with a validation message. The search is saved with its URL as entered, without parameter changes or additional filters (D-4). | Given the issue #56 URL, the search is added. Given a LeBoncoin URL or text, an error is shown and nothing is saved. |
| FR-AMZ-SEARCH-004 | A search is added. | Its first retrieval happens within the allowed request windows ([FR-AMZ-HUMAN-001](../amazon-human-browsing/functional.md)); until then the search shows that it is waiting for the next window, with the window start time. | A search added at 15:00 Paris time shows "waiting" until 22:00. |
| FR-AMZ-SEARCH-005 | The first retrieval of a search succeeds. | Capture the first 30 items in the order Amazon shows them (fewer if the results have fewer). For each item keep its link, title and price (D-2). This set never changes afterwards: later refreshes never add, replace or reorder items (D-3). | Given results with 60 items, exactly the first 30 are kept; on a later refresh where Amazon shows other items, the same 30 remain. |
| FR-AMZ-SEARCH-006 | The user selects a search. | Show its items in captured order. Each item is shown like a tracked Amazon item: title, marketplace and ASIN, latest price, recent prices, status, link to the listing, and access to its details page with full price history (D-2, D-10). Each item also shows its AI review summary state ([Amazon AI review](../amazon-ai-review/functional.md)). | Selecting a search lists its 30 items with prices; opening an item shows its price history. |
| FR-AMZ-SEARCH-007 | A search item is checked again. | Its price is checked like a tracked Amazon item ("refresh is done like for the other Amazon items", D-3), within the human-browsing rules for search items. Each successful check is stored in its price history; a failed check never replaces the last successful price (D-10). | After two successful checks the item shows two observations. |
| FR-AMZ-SEARCH-008 | A search item's listing is no longer available on Amazon. | Show the item as **unreachable**, as done for tracked Amazon and LeBoncoin items (D-5); keep it in the search with its last successful price and history. | A product removed from Amazon shows "unreachable" and its previous prices. |
| FR-AMZ-SEARCH-009 | The same Amazon product (same ASIN and marketplace) appears in several searches of the same user, or is also a tracked Amazon item of that user. | The item is shared: one item, one price history, one AI review history, shown in each place (D-6). | Given a product in searches A and B, a price check updates the item in both. Given it already tracked in the Amazon tab, the search shows the same history. |
| FR-AMZ-SEARCH-010 | The user chooses **Move to tracked Amazon items** on a search item not already tracked. | The item becomes a tracked Amazon item of the user, keeping its price history and reviews, and stays visible in the search (D-7). It then follows the tracked Amazon item rules (schedule, refresh, deletion). | After moving, the item appears in the Amazon tab and still in the search; its history is unchanged. |
| FR-AMZ-SEARCH-011 | A search item is already a tracked Amazon item. | The move action is not offered; the item is marked as tracked. There is no action to move a tracked item into a search (D-7). | A tracked item in a search shows "Tracked" and no move action. |
| FR-AMZ-SEARCH-012 | The user deletes a search, after confirming. | The search is removed with all its items, their price histories and reviews, except items that are tracked Amazon items or still belong to another search of the user (D-8, D-6). No rename or edit exists (D-8). | Deleting a search removes its untracked items; a moved item stays in the Amazon tab with its history; an item also in another search stays there. |
| FR-AMZ-SEARCH-013 | Ownership (protected mode) or open mode. | Searches belong to the user who added them; other users never see them. In open mode searches work without login, like other items (D-15). Sharing between users applies only to the same user's searches and tracked items (D-6). | User B does not see user A's searches; two users adding the same search get independent searches and items. |
| FR-AMZ-SEARCH-014 | The same user adds a URL already added as a search. | The add is refused as a duplicate, with a message naming the existing search. | Adding the same URL twice shows a duplicate message; one search exists. |
| FR-AMZ-SEARCH-015 | Search page states. | Distinguish: no search yet; search waiting for its first retrieval (no items yet); search with items; search stopped by failures ([FR-AMZ-HUMAN-008](../amazon-human-browsing/functional.md)); loading and loading error with retry, as other lists. | Each state has a distinct message. |
| FR-AMZ-SEARCH-016 | Keyboard, assistive technology, ~400 px viewport. | Same accessibility rules as the existing tabs and lists (FR-PLATFORM-TABS-008): labelled tab, keyboard-operable actions, confirmation dialog for deletion, states conveyed in text. | Every action is reachable by keyboard; at ~400 px no content overlaps. |

## States and corner cases

- First retrieval fails: the search keeps no items and shows the failure; it is tried again per the human-browsing retry rules.
- Results page returns fewer than 30 items: keep those; never complete the set later.
- An item without detectable price at capture: kept with its link and title; its price status is "price not found" until a check succeeds.
- An item deleted from the tracked Amazon list that still belongs to a search: it stays in the search (it is no longer tracked) and can be moved again.
- Search URL is the only filter; no additional parameter is applied (D-4).
- Concurrent delete during a running retrieval: the deleted search produces no further requests.

## Open questions

None.

## Traceability

- Request and decisions: [change index](../../changes/2026-10-07-1117-issue-56-amazon-searches/index.md).
- Related: [platform tabs](../platform-tabs/functional.md), [item refresh](../item-refresh/functional.md), [Amazon price detection](../amazon-price-detection/functional.md), [multi-user login](../multi-user-login/functional.md), [Amazon human browsing](../amazon-human-browsing/functional.md), [Amazon AI review](../amazon-ai-review/functional.md).
- Technical handoff target: `doc/specifications/amazon-searches/technical.md`.
