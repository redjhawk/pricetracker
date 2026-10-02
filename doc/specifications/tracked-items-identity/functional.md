# Functional specification: tracked item identity in the list

Status: ready
Owner: distinct functional specification agent
User decision/reference: requested removal of the separate marketplace column and display of marketplace followed by article ID in the item-name column.

## Functional subjects

1. Tracked-items list identity presentation. Item details is a separate subject and is outside this change.

## Purpose and scope

The operator reviews the shared tracked-items list. Combine marketplace and article identity in the existing Item column to reduce the space occupied by a separate Marketplace column. Preserve the existing title, details access, and thumbnail presentation.

Prerequisite: the application is available; the existing server response provides marketplace and listing ID for each tracked item. This is a presentation change only. Adding marketplaces, modifying tracking or collection, changing item details, or introducing new identity data is excluded.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-TI-IDENTITY-001 | Operator views tracked items | Show each item's marketplace followed by its article/listing ID in the Item column. Retain the title as the primary text and place this identity information below it, where the listing ID is currently shown. | For an Amazon item and a LeBoncoin item, the Item cell contains the existing marketplace value before the complete listing ID. The order is title, then marketplace and ID. The ID is the external listing identifier, not the application's internal item identifier. |
| FR-TI-IDENTITY-002 | List contains items | Remove the standalone Marketplace column while retaining Item, Prices, Amazon second-hand offer, Status, and Actions. | The table has five columns. Marketplace is available in each Item cell and has no separate header or cell. |
| FR-TI-IDENTITY-003 | Operator activates an item title or uses existing row actions | Preserve existing details navigation, title fallback, thumbnail or placeholder, listing access, and row actions. | Clicking or keyboard-activating a title opens that item's existing details view. A missing title displays “Title unavailable”. Existing thumbnails, placeholders, source-listing access, and deletion actions continue to work. |
| FR-TI-IDENTITY-004 | Operator searches the list | Preserve local filtering by title, listing ID, ASIN, marketplace, and platform. | Searching an existing marketplace value or listing ID still returns the matching row. A query without matches displays the existing no-match message across the five-column table. |
| FR-TI-IDENTITY-005 | List is loading, empty, failed, refreshing, or has pending/stale/error collection data | Preserve existing feedback, retry and refresh controls, prices, second-hand offer information, and status behavior. | Loading, initial empty state, retrieval failure with retry, and refresh progress remain available. Identity presentation does not change amounts, dates, collection statuses, or offer states. |
| FR-TI-IDENTITY-006 | Operator uses a narrow viewport or an item has a long title | Keep marketplace and full listing ID readable with the existing table's narrow-screen behavior, without overlapping the title or adjacent content. Preserve keyboard access to interactive controls. | At a narrow viewport, the operator can reach and read the complete marketplace/ID text using the existing scroll behavior where needed. Long titles do not hide or overlap identity text. Existing focus visibility and title/action accessible names remain usable. |

## States and corner cases

- Both supported platforms use the same marketplace-before-listing-ID presentation; marketplace is the existing server value rather than a newly invented platform label.
- Missing titles retain the current fallback; the identity line remains present.
- Missing thumbnails retain the current placeholder.
- Long titles, narrow viewports, and complete listing identifiers must be exercised in QA.
- Search without matches spans the remaining five columns. The initially empty collection continues to use its existing dedicated empty state.
- Existing loading, failure/retry, refresh, prices, and offer states are preserved rather than redesigned.
- No new validation or data mutation is introduced. Marketplace and listing ID remain supplied under the existing API contract.

## Open questions

None affecting the handoff. Exact separator and typography are technical presentation details; the required observable order is marketplace then listing ID. Retaining the existing title and moving marketplace into the existing identity line limits the change to the requested column consolidation.

## Traceability

- [Existing functional specifications](../../FUNCTIONAL_SPECIFICATIONS.md), sections 5.2 and 7: tracked-item metadata and local search.
- [UC-TI-01](../../use-cases/tracked-items/UC-TI-01-review-tracked-items.md): tracked-items review, metadata, and existing states.
- Existing behavior inspected in `src/components/TrackedItemsPage.tsx`.
- [Technical specification](technical.md): next-stage design covering frontend, backend, and API.
