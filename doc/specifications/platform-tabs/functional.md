# Functional specification: platform tabs in the tracked-items list

Status: ready
Owner: distinct functional specification agent
User decision/reference: requested separate Amazon and LeBoncoin tabs and removal of the second-hand-offer column for LeBoncoin.

## Purpose and scope

The operator reviews tracked items by source platform. This subject covers the existing tracked-items page, its platform selection, and its table presentation. Item details, collection, API behavior, persistence, new marketplaces, and new refresh scopes are excluded.

Prerequisite: the existing shared collection supplies each item's platform and existing display data. This specification refines the previously shared list presentation; all tracked items remain accessible through their platform tab.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-PLATFORM-TABS-001 | Operator opens the tracked-items page | Provide Amazon and LeBoncoin tabs in that order, with Amazon selected initially. Each tab shows only items from that platform, retaining their existing order. | With mixed data, Amazon rows appear only under Amazon and LeBoncoin rows only under LeBoncoin. Both tabs remain present when either platform or the entire collection is empty. |
| FR-PLATFORM-TABS-002 | A platform table is displayed | Retain Item, Prices, Status, and Actions for both platforms. Show Amazon second-hand offer only in the Amazon table. | Amazon has five columns with its existing offer states and condition information. LeBoncoin has four columns and no second-hand-offer heading, cell, or placeholder. Empty search results span the applicable complete table width. |
| FR-PLATFORM-TABS-003 | Operator types a search and switches tabs | Apply the existing case-insensitive, whitespace-trimmed search to the selected platform only; retain the typed query across tab switches. Preserve matching by title, listing ID, ASIN, marketplace, and platform. | A query matching only the other platform produces no matching rows in the current tab. Switching to that other tab shows its matches without changing the query. Clearing the query restores all rows of the selected platform. |
| FR-PLATFORM-TABS-004 | Collection or selected platform is empty, or search has no matches | Distinguish no tracked items, no items for the selected platform, and no search matches. Preserve access to adding items and switching tabs. | An entirely empty collection retains the existing first-item guidance. An empty platform is named in the empty feedback even if the other platform has items. A populated platform with an unmatched query shows “No items match your search.” |
| FR-PLATFORM-TABS-005 | Operator refreshes prices or reads the page count | Preserve the global tracked-item count and refresh of all tracked items across both platforms. Make the refresh's all-items scope clear. | A mixed collection's count includes both platforms even when one is selected or searched. The refresh action is labeled “Refresh all prices”; progress reports the existing total queued count. An empty selected platform does not disable refresh if the collection has items. |
| FR-PLATFORM-TABS-006 | Operator opens details, returns to the list, adds, deletes, or receives updated server data | Retain the chosen platform for the current running application. Existing title, source-listing, detail and delete actions remain available. Add/delete and collection updates respect the current platform filter. | Returning from details preserves the selected tab. Adding an item from another platform does not switch tabs; it becomes visible when its platform is selected and its data matches the search. Deleting the last item in a platform shows that platform's empty state. Reloading the application starts at Amazon; no new persistent preference or tab URL is required. |
| FR-PLATFORM-TABS-007 | Data is loading, failed, pending, stale, unavailable, or refreshing | Preserve existing loading, retrieval error/retry, refresh error/progress, price history, last successful prices, status, missing-title and thumbnail behavior. | Switching tabs does not misrepresent loading or failed retrieval as a successful empty result. Existing “Gratuit”, awaiting-first-price, status tags and Amazon offer states remain correct in their applicable tab. |
| FR-PLATFORM-TABS-008 | Operator uses keyboard, assistive technology, or a narrow viewport | Expose labeled tabs, selected state and associated tab panels through the existing Carbon accessibility patterns. Keep focus visible and all existing row actions reachable. | Keyboard users can focus and switch tabs using the standard Carbon tab interactions, then reach search and row actions. At narrow widths the full marketplace and listing ID remain readable using the existing table scrolling behavior without overlapping other cells. |

## Decision provenance and states

The two platform tabs and removal of LeBoncoin's second-hand-offer column are the requested behavior. Amazon-first ordering/default and retention of the chosen tab within the running application are routine interaction defaults supplied by the coordinator; they are not recorded as separate user approvals. Keeping the query across tab switches avoids altering an explicit user input. Existing global count, refresh semantics, collection order, add/delete rules, details, and price/status rendering are preserved.

Tab selection is independent of the search query, successful additions, polling, refresh completion, and list/detail navigation. Search behavior across a detail-page round trip remains the existing behavior; only the selected platform must survive that round trip. Errors and loading take precedence over a platform-empty message. No new input validation is introduced.

## Open questions

None affecting this scoped handoff. No new domain rule, API, persistence choice, or refactoring is required. Status ready does not assert user approval beyond the request.

## Traceability

- [Existing functional specifications](../../FUNCTIONAL_SPECIFICATIONS.md), sections 5.2 and 7, FR-05, FR-08, FR-09, FR-16, FR-17 and FR-19.
- [Tracked-item identity requirements](../tracked-items-identity/functional.md): preserve marketplace-before-listing-ID presentation; FR-TI-IDENTITY-002's five-column rule now applies to Amazon while LeBoncoin uses four columns.
- [Tracked-items use case](../../use-cases/tracked-items/UC-TI-01-review-tracked-items.md).
- [Approved API](../../../API_SPECIFICATION.md): existing platforms, global collection, and global refresh remain sufficient and unchanged.
- Technical design is the next handoff; it must map its design and verification to these requirement IDs.
