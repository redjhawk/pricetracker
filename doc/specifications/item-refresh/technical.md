# Technical specification: Refresh one item from the tracked-items list

Status: ready
Functional specification: [functional.md](functional.md) (status ready, FR-ITEM-REFRESH-001–008)

## Requirement mapping

| Technical ID | Functional ID | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-ITEM-REFRESH-001 | FR-ITEM-REFRESH-001, 008 | Add `<OverflowMenuItem itemText="Refresh price" …/>` in each row's existing `OverflowMenu`, after the "Open on …" item and before **Delete** (Delete keeps `hasDivider`). Same component on both tabs (single rendering path). | `src/components/TrackedItemsPage.tsx` | Playwright: menu lists View details, Open on…, Refresh price, Delete in that order on both tabs. |
| TS-ITEM-REFRESH-002 | FR-ITEM-REFRESH-002 | New App handler `handleRefreshListItem(item)` calls existing `refreshItem(item.id)` (`POST /api/v1/items/{id}/refresh`). No call to `refreshAllItems`. | `src/App.tsx` | Playwright: only `POST /api/v1/items/<A>/refresh` is sent; B unchanged. |
| TS-ITEM-REFRESH-003 | FR-ITEM-REFRESH-003, 006 | On `202`, call `refreshItems(true)` (quiet list reload). The backend marks the item `pending` synchronously (in-flight set before the 202), so the existing list polling effect (`items.some(status === "pending")`) keeps reloading until it completes; the row's `StatusTag` shows pending, then the server's final status/price. Failure status and retained last price come from the server item as today. | `src/App.tsx` | Playwright: row shows pending tag, then updated price/status without reload; failure fixture shows failed status with previous price. |
| TS-ITEM-REFRESH-004 | FR-ITEM-REFRESH-004 | App state `refreshingItemIds: Set<string>` holds IDs while the POST is in flight (added before, removed in `finally`). Menu item `disabled={item.status === "pending" \|\| refreshingItemIds.has(item.id)}`. Per-item, so other rows stay enabled. | `src/App.tsx`, `src/components/TrackedItemsPage.tsx` | Playwright: pending A → A's item disabled, B's enabled. |
| TS-ITEM-REFRESH-005 | FR-ITEM-REFRESH-005 | App state `itemRefreshError: string \| null`, set on failure to `` `${item.title ?? item.listingId}: ${errorMessage(error)}` `` and cleared at the start of each single-item refresh (so a later success clears it). Rendered as a separate `InlineNotification kind="error" title="Could not refresh price" lowContrast className="detail-notification"` next to the existing refresh-all notification. Items are not cleared on failure. | `src/App.tsx`, `src/components/TrackedItemsPage.tsx` | Playwright: mocked 404/500 → notification with title and item name; prices still shown; a following successful refresh removes it. |
| TS-ITEM-REFRESH-006 | FR-ITEM-REFRESH-007 | Single-item refresh never touches `refreshing`, `refreshCount` or `refreshError` (refresh-all state). The "Refresh all prices" button and bulk `InlineLoading` stay driven only by refresh-all state. | `src/App.tsx` | Playwright: after single refresh, no "Refreshing prices for" text and the refresh-all button stays enabled. |
| TS-ITEM-REFRESH-007 | FR-ITEM-REFRESH-008 | Carbon `OverflowMenuItem` provides menu keyboard semantics inside the existing labelled menu (`aria-label="Actions for …"`); disabled uses the `disabled` prop. Errors use Carbon `InlineNotification`. | `src/components/TrackedItemsPage.tsx` | Keyboard: Tab to menu button, Enter, arrow to Refresh price, Enter. |

## Frontend

- `TrackedItemsPage` gets new props: `onRefreshItem: (item: TrackedItem) => void`, `refreshingItemIds: ReadonlySet<string>`, `itemRefreshError: string | null`. Add the menu item and the notification described above. No new components, icons, or styles.
- `App.tsx`: add `refreshingItemIds` and `itemRefreshError` state and `handleRefreshListItem`:
  1. clear `itemRefreshError`; add id to `refreshingItemIds`;
  2. `await refreshItem(item.id)`; then `await refreshItems(true)`;
  3. on error set `itemRefreshError` as above; `finally` remove the id.
- Polling: reuse the existing list polling effect unchanged (it runs whenever any item is pending). Platform tab selection is not touched. A deleted item returns 404 → error; the next reload removes the row.
- Details page (`ItemDetail`, `detailRefreshing`, `handleRefreshItem`) unchanged.

## Backend

No change. `Service.RefreshItem` already validates existence (404 via handler), reschedules next check, deduplicates against the in-flight set, and the item reads as `pending` immediately after the 202. Collection failures preserve the last price (existing behavior).

## API

No contract change. The existing `POST /api/v1/items/{id}/refresh` (API_SPECIFICATION.md "Refresh one item": no body, `202 {requestedAt, itemsQueued}`, `404 ITEM_NOT_FOUND`, no duplicate checks) and `GET /api/v1/items` (status `pending`) fully cover the behavior. Only the sentence "The detail page polls `GET /api/v1/items/{id}`" describes one consumer; the list consumer polling `GET /api/v1/items` does not alter the contract, so the specification is left unchanged.

## Scope and refactoring

- Permitted files: `src/App.tsx`, `src/components/TrackedItemsPage.tsx`, new test `tests/item-refresh.spec.ts`, `package.json` (add a `test:item-refresh` script mirroring existing ones). No backend, API client, type, or CSS changes.
- Refactoring: considered sharing state with the details-page refresh (`detailRefreshing`) — declined; it is single-item detail state and merging would couple two views. Considered extracting the refresh-all/single notifications into a shared component — declined; two inline notifications are simpler. No refactoring performed.
- Risks: `TrackedItemsPage` prop list grows by three; acceptable.

## Verification and unresolved questions

- Build: `npm run build` (tsc + vite); `go test ./...` and `go vet ./...` unaffected but run as regression.
- Playwright (`tests/item-refresh.spec.ts`, mocked API as in `tests/platform-tabs.spec.ts`) covering TS-001–006: menu order on both tabs; only item A POSTed; pending then updated row; disabled for pending item only; 404/500 error notification naming the item with prices kept and cleared by later success; refresh-all indicator untouched. Run existing `test:platform-tabs` and `test:settings` for regression.
- QA: exploratory run in the app (`npm run dev`) on both tabs, keyboard operation, refresh during refresh-all, item deleted then refreshed.
- Unresolved questions: none.
