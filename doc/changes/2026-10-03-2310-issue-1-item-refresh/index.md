# Item refresh from the list

Stage: implementation complete; independent review next.
User request: GitHub issue #1 — add a per-item **Refresh price** action to the "..." menu of each item in the tracked-items list.

## Stage results

1. [Functional specification](../../specifications/item-refresh/functional.md) — ready, FR-ITEM-REFRESH-001–008, no open questions.
2. [Technical specification](../../specifications/item-refresh/technical.md) — ready, TS-ITEM-REFRESH-001–007 mapped to FR-001–008. Frontend only (`src/App.tsx`, `src/components/TrackedItemsPage.tsx`, new `tests/item-refresh.spec.ts`, `package.json` test script). No backend change. Refactoring was considered and declined; the rationale is recorded in the specification.
3. API assessment — no contract change. API_SPECIFICATION.md "Refresh one item" (`POST /api/v1/items/{id}/refresh`, 202/404, no duplicate checks) and the `pending` status in `GET /api/v1/items` already cover the behavior. The backend marks the item pending before returning 202, so the list's existing polling applies. API_SPECIFICATION.md is unchanged.
4. Implementation — done (developer agent). See "Implementation" below.
5. Independent review — pending.
6. Independent decisions — pending.
7. QA — pending.
8. Commit — pending.

## Scope

Frontend list-page action reusing the existing single-item refresh contract and existing refresh feedback patterns. No change to refresh-all, details page, or schedule.

## Implementation

Changed files: `src/App.tsx`, `src/components/TrackedItemsPage.tsx`, `package.json` (`test:item-refresh` script), `playwright.config.ts`, new `tests/item-refresh.spec.ts`.

Scope deviation: `playwright.config.ts` was not in the technical specification's permitted files. Its `testMatch` lists each spec file explicitly, so the new spec could not run unless it was added. The only change is appending `"item-refresh.spec.ts"` to that list. No other refactoring.

| Requirement | Technical ID | Change | Test (`tests/item-refresh.spec.ts`) |
| --- | --- | --- | --- |
| FR-001, FR-008 | TS-001, TS-007 | `OverflowMenuItem "Refresh price"` between "Open on …" and Delete | "menu lists Refresh price before Delete on both tabs"; keyboard in "action is disabled while its request is in flight and works by keyboard" |
| FR-002 | TS-002 | `handleRefreshListItem` calls `refreshItem(item.id)` | "refreshes only the selected item…" (only `POST /api/v1/items/amazon-one/refresh`) |
| FR-003, FR-006 | TS-003 | quiet `refreshItems(true)` after 202; existing pending polling | "refreshes only the selected item…" (pending, then 9,99/Active); "failed collection shows failure status and keeps previous price" |
| FR-004 | TS-004 | `refreshingItemIds` state; `disabled` when pending or in flight | "refreshes only the selected item…" (A disabled, B enabled); "action is disabled while its request is in flight…" |
| FR-005 | TS-005 | `itemRefreshError` + "Could not refresh price" notification, cleared on next refresh | "request errors name the item, keep prices, and clear on later success" (404 and 500) |
| FR-007 | TS-006 | refresh-all state untouched | "refreshes only the selected item…" (no bulk text, button enabled); "refresh-all in progress keeps its indicator and pending items disabled" |

## Verification (developer, actually executed)

- `npm run build` — passed (tsc + vite build).
- `npx playwright test tests/item-refresh.spec.ts` — 6 passed.
- `npx playwright test tests/platform-tabs.spec.ts tests/leboncoin-session-settings.spec.ts` (the `test:platform-tabs` and `test:settings` targets) — 30 passed.
- `go vet ./...` — no findings; `go test ./...` — all packages ok (results cached; no Go code changed).
- Note: the npm script wrappers (`npm run test:…`) needed extra approval in this environment, so the identical `npx playwright test <file>` commands were run directly.
