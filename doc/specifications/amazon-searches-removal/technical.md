# Technical specification: Amazon searches removal

Status: ready
Functional specification: [functional.md](functional.md), FR-AMZ-SRCH-DEL-001–007 (ready, 2026-11-10)
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "Amazon searches (2026-10-07) — removed (issue #111)"
Companion: [Amazon request stop technical](../amazon-request-stop/technical.md) (per-item stop, TS-AMZ-STOP-*). Supersedes the technical specifications of [Amazon searches](../amazon-searches/technical.md), [Amazon human browsing](../amazon-human-browsing/technical.md) and [Amazon AI review](../amazon-ai-review/technical.md).

## Requirement mapping

`DEL-*` abbreviates `FR-AMZ-SRCH-DEL-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-AMZ-SRCH-DEL-001 | DEL-001 | Remove the **Amazon searches** tab, the `/searches/{id}` route and the search pages; an old `/searches/…` URL falls through to the main page (Amazon tab) | delete `src/components/AmazonSearchesPage.tsx`, `src/components/AmazonSearchItems.tsx`; edit `src/App.tsx`, `src/components/TrackedItemsPage.tsx`, `src/types.ts` | `npm run build`; QA: two tabs only; `/searches/x` shows the Amazon tab |
| TS-AMZ-SRCH-DEL-002 | DEL-002, DEL-006 | Remove the search HTTP routes, service, worker, request windows, random pauses and results-page parser | delete `internal/httpapi/searches.go`(+test), `internal/service/searches.go`(+test), `internal/service/search_worker.go`(+test), `internal/service/search_window.go`, `internal/amazon/search.go`(+test), `internal/model/searches.go`; edit `internal/httpapi/server.go`, `internal/service/service.go`, `cmd/pricefollower/main.go` | `go vet ./...`, `go test ./...`; `GET /api/v1/amazon-searches` → `404 ROUTE_NOT_FOUND` (existing unknown-route behavior) |
| TS-AMZ-SRCH-DEL-003 | DEL-002, DEL-003 | Remove the search store; startup migration deletes search-only items (`tracked = 0`, cascading history, attempts, second-hand data, reviews) and drops `amazon_search_items`, `amazon_searches` | delete `internal/store/searches.go`(+test); edit `internal/store/store.go`; new `internal/store/search_removal_test.go` | Store test on a DB built with the issue #56 schema (below) |
| TS-AMZ-SRCH-DEL-004 | DEL-004 | Tracked items untouched by the migration; `Delete` deletes again (no "untrack"); `Add` no longer converts a search-only item; `ITEM_NOT_TRACKED` removed | `internal/store/store.go`, `internal/service/service.go` | Migration test keeps a tracked item that was also in a search, with history; delete test |
| TS-AMZ-SRCH-DEL-005 | DEL-005 | Remove Amazon AI reviews: service runner, Claude prompt/parser, Amazon review shape, product-details extraction, Amazon details `aiReview` (always `null` again); migration deletes `ai_reviews` rows of Amazon items. LeBoncoin reviews unchanged | delete `internal/service/amazon_review.go`(+test), `internal/claude/amazon_review.go`(+test), `src/components/AmazonAiReview.tsx`; edit `internal/service/review.go`, `internal/service/service.go`, `internal/service/settings.go`, `internal/store/ai_reviews.go`(+test), `internal/model/model.go`, `internal/amazon/collector.go`(+test), `src/components/ItemDetail.tsx` | Go tests: Amazon details `aiReview: null`; LeBoncoin review tests unchanged and passing |
| TS-AMZ-SRCH-DEL-006 | DEL-007 | Global stop conversion | see TS-AMZ-STOP-007 | see there |

## Frontend

- `TrackedItemsPage`: `Tab` type back to `TrackedItem["platform"]`, `platforms = ["amazon", "leboncoin"]`, no third `<Tab>`, no `onOpenSearch` prop. `formatSearchDate` (imported from the deleted page) is replaced by a local date formatter in `TrackedItemsPage` (same `Intl.DateTimeFormat` output) until the warning is rewritten (TS-AMZ-STOP-005).
- `App.tsx`: remove `getSearchId`, the `AmazonSearchItems` branch and `onOpenSearch`. A selected-tab value `amazon-searches` (URL/query or stored state, if any) maps to `amazon`.
- `ItemDetail.tsx`: remove the `AmazonAiReview` branch; the LeBoncoin `AiReview` branch is unchanged.
- `src/types.ts`: remove `AmazonSearch`, `SearchItem`, `AIReviewSummary`-style search types and `tracked`. `AmazonRequests` and `src/api/searches.ts` (`getAmazonRequests` only) stay until the per-item stop frontend part, then are deleted.
- No new Carbon components. No CSS change required (`.tracked-search` classes belong to the item filter and stay).

## Backend

- HTTP: drop the `searchesPath` routing and handlers. `handleAmazonRequests` stays (moved into `server.go`) until TS-AMZ-STOP-006 removes it.
- Service: drop `runningSearch`, `RunSearchWorker`, `trackSearchOnlyItem`, the `!listing.Tracked` refresh check, `AmazonAIReview` launch after Amazon collections and the `ReviewAmazon` method from the Claude interface (test fakes updated). `amazonGate` stays (rewritten by TS-AMZ-STOP-*); its comment no longer mentions searches.
- Amazon collector: remove `productDetails` and `CollectionResult.Product` (only used by Amazon reviews); price/second-hand parsing unchanged.
- Store: remove `createSearchSchema`, `UntrackedItemID`, `IsSearchItem`, the Amazon branch of review decoding; `Delete` becomes `DELETE FROM items WHERE id = ? AND owner_id = ?`.
- The `items.tracked` column is kept (no table rebuild; after the migration every row is `1` and new rows default to `1`), and existing `tracked = 1` filters stay. Decision: rebuilding `items` again for a dead column adds risk (FK rebuild already done once in `migrateItemsOwner`) for no user-visible gain. `model.Item.Tracked`/JSON `tracked` are removed; `model.Listing.Tracked` is removed.

### Migration (`removeAmazonSearches`, in `createSchema` after the items migrations, one transaction, idempotent)

```sql
-- only when table amazon_search_items exists (sqlite_master check):
DELETE FROM items WHERE tracked = 0;                                   -- search-only items; FKs cascade
DELETE FROM ai_reviews WHERE item_id IN (SELECT id FROM items WHERE platform = 'amazon');
DROP TABLE IF EXISTS amazon_search_items;
DROP TABLE IF EXISTS amazon_searches;
```

`PRAGMA foreign_keys=ON` is already set per connection, so `price_observations`, `collection_attempts`, `second_hand_offer_*` and `ai_reviews` rows of deleted items cascade. Running `DELETE … WHERE tracked = 0` and the Amazon `ai_reviews` delete unconditionally is also safe (no rows after the first run); the existence check only avoids work. A failure aborts startup with a wrapped error (existing pattern), leaving the database unchanged.

## API

Removed: `GET|POST /api/v1/amazon-searches`, `GET|DELETE /api/v1/amazon-searches/{id}`, `POST /api/v1/amazon-searches/{id}/refresh`, `POST /api/v1/amazon-searches/{id}/items/{itemId}/track` (now `404 ROUTE_NOT_FOUND`), the `tracked` item field, `409 ITEM_NOT_TRACKED`, Amazon `aiReview` objects (Amazon details return `aiReview: null`, as before issue #56). `DELETE /api/v1/items/{id}` deletes (no untrack). `POST /api/v1/items` no longer reuses search items. Rationale in the [change index](../../changes/2026-11-10-0141-issue-111-delete-amazon-searches/index.md).

## Scope and refactoring

Only the files listed here and in TS-AMZ-STOP-*. No refactoring beyond moving `handleAmazonRequests` into `server.go` and replacing `formatSearchDate` (both needed so deletions compile; temporary). LeBoncoin, scheduling of tracked items and item display are unchanged. Delivery order: see the [implementation plan](../amazon-request-stop/technical.md#implementation-plan).

## Verification and unresolved questions

- Each part: `go vet ./...`, `go test ./...`, `npm run build`.
- Migration test (DEL-002–005): create a DB with the issue #56 tables (`amazon_searches`, `amazon_search_items`, `items.tracked`), a search-only item with observations, attempts and an Amazon review, a tracked item also in a search with history, a LeBoncoin item with a review; open the store; assert search tables gone, search-only item and its child rows gone, tracked item and history kept, Amazon reviews gone, LeBoncoin review kept; reopen succeeds (idempotent). Fresh DB opens without search tables.
- QA: two tabs; old `/searches/…` URL; Amazon item details without AI review; LeBoncoin details review intact; Claude helper text (TS-CLT-SET-009).
- Unresolved functional questions: none.
