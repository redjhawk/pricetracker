# Technical specification: Amazon request stop

Status: ready
Functional specification: [functional.md](functional.md), FR-AMZ-STOP-001–010 (ready, 2026-11-10)
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "Amazon per-item stop (issue #111)"
Companion: [Amazon searches removal technical](../amazon-searches-removal/technical.md). Replaces the stop design of [Amazon human browsing technical](../amazon-human-browsing/technical.md).

## Requirement mapping

`STOP-*` abbreviates `FR-AMZ-STOP-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-AMZ-STOP-001 | STOP-001 | Keep the existing failure log line in `amazonGate.do` (URL, time, status, message, response excerpt; no secrets) | `internal/service/amazon_gate.go` | Existing log test kept |
| TS-AMZ-STOP-002 | STOP-002, STOP-010 | Keep the global consecutive-failure counter in `amazon_request_state.consecutive_failures`; reset by any Amazon answer | `internal/service/amazon_gate.go`, `internal/store/amazon_requests.go` | Gate test: 4 fail, 1 ok, 4 fail → nothing stopped |
| TS-AMZ-STOP-003 | STOP-003, STOP-010 | New nullable column `items.amazon_stopped_at`; at the 5th failure, in one statement, every Amazon item of every owner with `amazon_stopped_at IS NULL` gets `now`; counter reset to 0; stop logged | `internal/store/store.go`, `internal/store/amazon_requests.go`, gate | Gate test: 5th failure stops all Amazon items, LeBoncoin untouched; reopen keeps it |
| TS-AMZ-STOP-004 | STOP-004, STOP-008 | Scheduler: `DueIDs` excludes `amazon_stopped_at IS NOT NULL`. Refresh all: queue unchanged, the gate skips a stopped item without request or attempt record. Restarted item has `NULL` and is due again | `internal/store/store.go`, `internal/service/service.go`, gate | Service tests with fake collector: no Amazon call for stopped items, LeBoncoin called; restarted item collected |
| TS-AMZ-STOP-005 | STOP-005 | Amazon tab warning derived from the items list (`amazonStoppedAt`); item details show the item's stopped state | `src/components/TrackedItemsPage.tsx`, `src/components/ItemDetail.tsx`, `src/types.ts`; delete `src/api/searches.ts` | `npm run build`; QA with stopped items |
| TS-AMZ-STOP-006 | STOP-006, STOP-007 | `RefreshItem` no longer refuses stopped items; the collection is marked manual and the gate lets it through. Answer → clear only that item's stop, counter 0. Failure → stays stopped, counted, attempt recorded (existing retrieval-error display) | `internal/service/service.go`, gate, store | Gate/service tests: 3 stopped, manual ok on 1 → 1 restarted, 2 stopped; manual failure keeps stop and increments count |
| TS-AMZ-STOP-007 | STOP-009, DEL-007 | Startup conversion: if `amazon_request_state.blocked = 1`, set `amazon_stopped_at = COALESCE(stopped_at, now)` on Amazon items not stopped, then `blocked = 0, consecutive_failures = 0` — one transaction, idempotent | `internal/store/amazon_requests.go` | Store test on a DB with `blocked = 1` |
| TS-AMZ-STOP-008 | STOP-005 | Remove `GET /api/v1/amazon/requests`, `model.AmazonRequests`, `Service.AmazonRequests` and `409 AMAZON_REQUESTS_STOPPED`; add `amazonStoppedAt` to the item response | `internal/httpapi/server.go`(+tests), `internal/model/*.go`, service | Handler tests: route → `404 ROUTE_NOT_FOUND`; stopped item refresh → `202` |

## Frontend

- `TrackedItem.amazonStoppedAt: string | null` in `src/types.ts`; `AmazonRequests` type and `src/api/searches.ts` deleted; `TrackedItemsPage` no longer fetches anything for the warning.
- Amazon tab warning (existing Carbon `InlineNotification`, `kind="warning"`, `lowContrast`, `hideCloseButton`, position unchanged above the table): shown while at least one Amazon item has `amazonStoppedAt`; date = the most recent `amazonStoppedAt`, formatted as today. Title: "Amazon checks were stopped on {date} after repeated failures." Subtitle: "Restart each item with Refresh price on its details page." It disappears when the list poll returns no stopped item (after deletion or restart).
- Item details, Amazon item with `amazonStoppedAt`: `InlineNotification kind="warning"` "Amazon checks of this item are stopped since {date}. Refresh price restarts it." **Refresh price** stays enabled. A failed manual check is shown by the existing retrieval-error notification; the stop notice stays. The existing pending polling refreshes the state after the check.
- Accessibility: Carbon notifications keep their `role` semantics; text-only, no color-only state; works at ~400 px like the existing notices.

## Backend

- Store (`amazon_requests.go`): `ensureColumn(items, amazon_stopped_at, TEXT)`; `StopAmazonItems(ctx, now)` (`UPDATE items SET amazon_stopped_at = ? WHERE platform = 'amazon' AND amazon_stopped_at IS NULL`); `ClearAmazonStop(ctx, id)`; `AmazonStopped(ctx, id) (bool, error)`; `SaveAmazonFailures(ctx, n)`. `AmazonRequestState` keeps only `ConsecutiveFailures`; the `blocked`/`stopped_at` columns stay in the table (no rebuild) and are only read by the conversion. Item select queries read `amazon_stopped_at` into `model.Item.AmazonStoppedAt *time.Time` (`json:"amazonStoppedAt"`, `null` for LeBoncoin).
- Gate: `do(ctx, id string, manual bool, request)`. Under the request mutex: if not manual and the item is stopped → `errAmazonStopped` (logged skip, no attempt recorded, as today). Run request; cancellation unchanged. `request_error` → log, counter++, at `>= 5` call `StopAmazonItems(now)`, counter = 0, log the stop. Otherwise → counter = 0 and, if the item was stopped, `ClearAmazonStop(id)`. Persistence errors are logged (existing pattern). The gate is application-wide (all owners), as before.
- Service: `RefreshItem` passes `manual = true` through `runQueuedCollection`/`collectReserved`; `RefreshAll`, scheduler and new-item first check pass `false`. A newly added item is not stopped, so its first check runs (recorded interpretation in the functional spec). Remaining checks of a running batch hit the gate as stopped and are skipped (STOP-003).
- `POST /api/v1/items/refresh` `itemsQueued` keeps counting every tracked item (contract unchanged); stopped ones end without a new attempt, so their status returns to the previous value.

## API

- Item response: new `amazonStoppedAt` (string or `null`), list, add and details responses.
- `POST /api/v1/items/{id}/refresh`: `202` for stopped Amazon items (manual restart); `409 AMAZON_REQUESTS_STOPPED` removed.
- `GET /api/v1/amazon/requests` removed (`404 ROUTE_NOT_FOUND`); the per-item field replaces it and is owner-scoped.

## Scope and refactoring

No refactoring beyond threading `manual` through the collection functions (required). Out of scope: LeBoncoin, AI reviews.

## Implementation plan

Stacked parts on a feature branch from `master`; each part must pass `go vet ./...`, `go test ./...`, `npm run build` on its own. Sizes are added+deleted lines (from `wc -l` of deleted files plus estimated edits); target ≤ ~450, hard 500. If a slice does not compile alone, move the smallest dependent symbol into the part (record it) rather than exceeding 500.

| Part | Content | Estimate |
| --- | --- | --- |
| 1 | Docs: functional + technical specs, superseded status lines, API_SPECIFICATION.md, change index | ~330 (measured at stage 3) |
| 2 | Frontend: delete `AmazonSearchItems.tsx` (196), `AmazonAiReview.tsx` (59); `App.tsx` route, `ItemDetail.tsx` branch | ~290 |
| 3 | Frontend: delete `AmazonSearchesPage.tsx` (225); tab in `TrackedItemsPage.tsx`, local date formatter, search types and `tracked` in `types.ts`; Claude helper text (TS-CLT-SET-009) | ~300 |
| 4 | Backend HTTP: delete search handlers + `httpapi/searches_test.go` (120+140), move `handleAmazonRequests` into `server.go` | ~270 |
| 5 | Backend service API: delete `service/searches.go` (210) + test (61), keeping `setRunningSearch`/`AmazonRequests` in `search_worker.go`/`service.go` | ~280 |
| 6 | Backend worker: delete `search_worker.go` (223) + test (188), `search_window.go` (35); `main.go`, `runningSearch` field | ~465 |
| 7 | Amazon AI review: delete `service/amazon_review.go` (64) + test (117), `claude/amazon_review.go` (87) + test (63); `review.go`, `settings.go` interface and fakes, `ai_reviews.go` Amazon decode, `model.go` Amazon review/product types, collector `productDetails` | ~420 |
| 8 | Delete `amazon/search.go` (133) + test (127), `model/searches.go` search types; remove `trackSearchOnlyItem`, `ITEM_NOT_TRACKED`, `Tracked` model/JSON fields | ~360 |
| 9 | Store: delete `store/searches.go` (251) + test (182), `createSearchSchema` call, `UntrackedItemID`, `IsSearchItem`; simplify `Delete` | ~460 |
| 10 | Migration TS-AMZ-SRCH-DEL-003/005 + migration test | ~130 |
| 11 | Per-item stop backend TS-AMZ-STOP-001–004, 006–008: column, store functions, conversion, gate rewrite + `amazon_gate_test.go` (187) rewrite, `amazonStoppedAt`, remove `/amazon/requests` and `model.AmazonRequests` | ~450 |
| 12 | Per-item stop frontend TS-AMZ-STOP-005; delete `src/api/searches.ts` (41) | ~110 |

Between parts 11 and 12 the frontend's `getAmazonRequests` call fails and is already caught (warning hidden); nothing deploys before the final feature→`master` merge.

## Verification and unresolved questions

- Go tests listed in the mapping; `amazon_gate_test.go` covers STOP-002/003/006/007/008 with a fake clock and store on a temp DB; store test for STOP-009 conversion and STOP-010 reopen.
- QA: force 5 failures (unreachable Amazon host / test collector), see warning with date, Refresh all prices sends no Amazon request and checks LeBoncoin, Refresh price on one stopped item restarts only it, failing manual refresh keeps the stop, delete the last stopped item hides the warning, restart the app keeps stops.
- Unresolved functional questions: none.
