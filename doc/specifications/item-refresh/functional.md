# Functional specification: Refresh one item from the tracked-items list

Status: ready
Owner: functional specification agent
User decision/reference: GitHub issue #1 — "Currently, a user that is in the list of items can only refresh the price of all of them. He should be able to do it item by item by executing it on a new menu item on the '...' button available for each item. This action will refresh the price of the selected item."

## Purpose and scope

Actor: the operator (no accounts in v1). Goal: request an immediate price check for a single tracked item directly from the tracked-items list, without opening its details page.

Included: a new action in each item row's existing "..." (overflow) actions menu, on both the Amazon and LeBoncoin tabs, that refreshes only that item, with progress and error feedback on the list page.

Excluded: changes to "Refresh all prices" (UC-TI-03), the details-page **Refresh price** button (UC-ID-02), the collection schedule, collection semantics, or the existing menu actions (View details, open listing, Delete). No new server behavior is requested; the existing single-item refresh (`POST /api/v1/items/{id}/refresh`, API_SPECIFICATION.md "Refresh one item") already defines what a refresh does.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-ITEM-REFRESH-001 | Tracked-items list shows at least one item | Each item row's "..." menu contains a **Refresh price** action (same label as the details-page action, UC-ID-02), placed with the non-destructive actions, above **Delete**. | Given a listed item, when the operator opens its "..." menu, then **Refresh price** is present and **View details**, the listing action and **Delete** are still present. |
| FR-ITEM-REFRESH-002 | Operator chooses **Refresh price** for an item | An immediate price check is requested for that item only, with the same server semantics as the details-page refresh (UC-ID-02): the regular schedule stays active and an already-running check for that item is not duplicated. | Given items A and B, when the operator refreshes A from its menu, then only A is checked; B's status and data are unaffected. |
| FR-ITEM-REFRESH-003 | Refresh request accepted | The list shows the item as being checked (existing pending status for that row) and keeps reloading the list while that item is pending, as the list already does for pending items; when the check completes, the row shows the latest server state (price, timestamp, status). | Given an accepted refresh, then the row shows the pending status until completion, then the updated price/status without a manual page reload. |
| FR-ITEM-REFRESH-004 | Refresh request in progress or item pending | The **Refresh price** action for that item is unavailable (disabled) while its refresh request is being submitted or while the item is pending, mirroring the details page which disables its refresh while refreshing. Other items' actions remain available. | Given A is pending, when the operator opens A's menu, then **Refresh price** is disabled; B's **Refresh price** is enabled. |
| FR-ITEM-REFRESH-005 | Refresh request fails (network or server error, including item not found) | The list page shows an error notification "Could not refresh price" with the error detail (same title as the details page, UC-ID-02) identifying the item by its displayed name; the last successful price and other rows stay visible. | Given the request fails, then an error notification naming the item appears and existing prices remain displayed. |
| FR-ITEM-REFRESH-006 | A collection attempt completes with failure | Reported via the item's existing status, without replacing the last successful price (UC-ID-02, UC-PC-01). | Given a failed check, then the row shows the failure status and the previous price. |
| FR-ITEM-REFRESH-007 | "Refresh all prices" is in progress | Single-item refresh remains available for items that are not pending; refresh-all's own progress indicator and button behavior are unchanged. A single-item refresh does not enable/disable "Refresh all prices" nor display the bulk "Refreshing prices for N items…" progress. | Given refresh-all is running, then pending items' **Refresh price** is disabled (FR-004) and the bulk indicator behaves exactly as before. |
| FR-ITEM-REFRESH-008 | Accessibility | The action is a standard menu item reachable and operable by keyboard within the existing labelled "Actions for <item>" menu; error feedback uses the existing notification component. | Keyboard-only operator can open the menu, select **Refresh price**, and perceive the resulting status/error. |

## States and corner cases

- Pending/loading: row's existing pending status tag is the progress indicator; no blocking page-wide loader.
- Item already being checked (scheduled, initial, or refresh-all): action disabled (FR-004); if the request still races with a running check, the server does not duplicate it (FR-002).
- Item deleted elsewhere: request fails with not found → error per FR-005; the next list reload removes the row.
- Multiple items refreshed in succession: each is independent; every pending row shows its own status.
- Platform tabs: behaves identically on both tabs; refreshing an item does not change the selected tab.
- Error dismissal: notification follows the existing list notification behavior; a subsequent successful refresh clears a previous single-item refresh error.

## Open questions

None. Feedback, errors, deduplication and concurrency are derived from UC-ID-02, UC-TI-03 and the existing list behavior.

## Traceability

- doc/FUNCTIONAL_SPECIFICATIONS.md (manual refresh of all items and of an individual item).
- doc/use-cases/item-details/UC-ID-02-refresh-item-price.md, doc/use-cases/tracked-items/UC-TI-03-refresh-all-prices.md, doc/use-cases/tracked-items/UC-TI-01-review-tracked-items.md, doc/use-cases/price-collection/UC-PC-01-collect-and-record-prices.md.
- API_SPECIFICATION.md "Refresh one item" (existing contract, expected to be reused unchanged).
- Technical specification: [technical.md](technical.md) (to be written).
