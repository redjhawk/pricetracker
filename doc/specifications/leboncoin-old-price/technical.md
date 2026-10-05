# Technical specification: LeBoncoin old price

Status: ready
Functional specification: [functional.md](functional.md) (status ready, FR-LBC-OLD-PRICE-001–009)

## Source shape of the old price

No fixture, test or parser in the repository references `old_price`; its exact shape in the LeBoncoin `__NEXT_DATA__` ad JSON (`props.pageProps.ad`) is unverified. The design therefore parses defensively, mirroring the existing `price` handling (`price` is a one-element euro array, `price_cents` an integer):

- `ad.old_price`, decoded as `json.RawMessage`, accepted as a JSON number or a one-element array of numbers, in euros (same parsing as `parsePrice`). Any other shape, a negative value or absence means "no old price" (FR-LBC-OLD-PRICE-006). A malformed `old_price` never affects current-price parsing.
- No date field for the old price is known. The collector returns an optional date (`OldPriceAt *time.Time`), left `nil` unless a known date field is found; the whole chain (storage, API, UI) supports dated and undated entries, so a date field can be wired later in the collector alone. Today entries are recorded undated and shown as "Old price" (FR-LBC-OLD-PRICE-004).

QA must inspect a real listing with a crossed-out price and record the observed shape; if it differs (e.g. an object or cents field), the developer adjusts only `oldPriceCents` in the collector and its tests.

## Requirement mapping

| ID | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- |
| TS-LBC-OLD-PRICE-001 (FR-001, 006) | Collector parses `old_price` into `CollectionResult.OldPriceCents *int64` / `OldPriceAt *time.Time` only on a successful result. | `internal/leboncoin/collector.go`, `internal/model/model.go` | Go unit tests on parser: number, array, absent, invalid, negative. |
| TS-LBC-OLD-PRICE-002 (FR-001, 002, 007, 009) | `collectReserved` clears the old-price fields unless `newItem` and `result.Result == "success"`; only LeBoncoin collector sets them. | `internal/service/service.go` | Service test: non-new collection with old price records none. |
| TS-LBC-OLD-PRICE-003 (FR-001, 003, 004, 005, 008) | `RecordCollection` inserts the old-price row in the same transaction as the current price, with `is_old_price = 1`; `observed_at` is the date or `''` when undated. Cascade delete unchanged. | `internal/store/store.go` | Store tests: dated/undated insert, ordering, cascade. |
| TS-LBC-OLD-PRICE-004 (FR-003, 004, 005) | API observation gains `oldPrice` boolean; `timestamp` is `null` for an undated old price. | `internal/model/model.go`, `internal/store/store.go`, `API_SPECIFICATION.md` | Store/handler test on JSON. |
| TS-LBC-OLD-PRICE-005 (FR-004, 005) | UI shows "Old price" instead of the time when `timestamp` is null. | `src/types.ts`, `src/api/items.ts`, `src/components/ItemDetail.tsx`, `src/components/TrackedItemsPage.tsx` | Typecheck/build; QA on details and list. |

## Frontend

- `src/types.ts`: `PriceObservation.timestamp: string | null`, add `oldPrice: boolean`. `src/api/items.ts`: `ApiPriceObservation` mirrors the API; `mapObservation` copies both.
- `ItemDetail.tsx` price history and `TrackedItemsPage.tsx` list periods: when `timestamp` is null render the plain text `Old price` in place of `<time>` (no `<time>` element, so screen readers read the label). Dated old prices render like any observation (FR-LBC-OLD-PRICE-003). Row keys must not rely on `timestamp` alone (use `index` fallback, as details already does).
- No new column, component, state, navigation or error case. Second-hand (Amazon) history unchanged.

## Backend

- Collector (`internal/leboncoin/collector.go`): add `OldPrice json.RawMessage \`json:"old_price"\`` to `listingData`; helper `oldPriceCents(raw) (int64, bool)` returning a non-negative euro-cent value. Set on the result only in the two success branches.
- Model: `CollectionResult.OldPriceCents *int64`, `OldPriceAt *time.Time`; `Observation.Timestamp *time.Time` (`json:"timestamp"`), `Observation.OldPrice bool` (`json:"oldPrice"`). Existing uses of `Timestamp` (status computation on `latestPrice`) dereference; `latestPrice` is never an undated old price because the old price is only inserted with a newer collected observation.
- Service (`collectReserved`): before `RecordCollection`, `if !newItem || result.Result != "success" { result.OldPriceCents, result.OldPriceAt = nil, nil }`. A dated old price whose date is not strictly before the collection timestamp is recorded undated (keeps the collected price as the latest; FR "does not change the latest price"). No extra AI review (review reservation is unchanged).
- Store:
  - Migration: `price_observations` gains `is_old_price INTEGER NOT NULL DEFAULT 0 CHECK(is_old_price IN (0, 1))` in `createSchema`, plus an idempotent `ALTER TABLE ... ADD COLUMN` guarded by `hasColumn` for existing databases (generalise `ensureItemColumn` to take the table name, or add a sibling; record as minimal refactor). Existing rows keep 0; no back-fill (functional exclusion).
  - `RecordCollection`: on success with `OldPriceCents != nil`, insert `(item_id, amount_cents, 'EUR', observed_at, 1)` where `observed_at` is the UTC date in `timestampLayout` or `''`. `''` sorts before every ISO timestamp, so existing `ORDER BY observed_at` queries order undated old prices as oldest (FR-LBC-OLD-PRICE-005) without query changes beyond selecting `is_old_price`.
  - History and last-three queries select `is_old_price`; scan `observed_at` `''` to `nil` timestamp.
- Failure: any insert error rolls back the whole collection transaction (existing behaviour).
- AI review prompt (`internal/claude/review.go`): the price history sent to Claude includes the old price (part of the history, FR-LBC-OLD-PRICE-005). An undated old price (nil `Timestamp`) is sent with date `"old price"`; a dated old price is sent with its date, indistinguishable from collected prices (`PriceHistory` does not select `is_old_price`). No additional review is triggered.

## API

Contract change recorded in [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) ("LeBoncoin old price"): every item-price observation object (`latestPrice`, `lastThreeDetections[]`, `priceHistory[]`) gains `oldPrice: boolean`; `timestamp` becomes nullable, `null` only when `oldPrice` is true and the listing gave no date. No new endpoint, status code or error. Additive except for nullability, which only occurs for new LeBoncoin items; Amazon second-hand objects are unchanged.

## Scope and refactoring

Change boundary: files listed above plus their tests. Refactor: generalise `ensureItemColumn` to `ensureColumn(table, ...)` — needed to add the column to `price_observations`, three call-site lines, no behaviour change; performed with the feature. No other refactoring.

Risks: unknown `old_price` shape (mitigated by tolerant parsing and QA note); `timestamp` nullability for any client that assumed non-null (only the bundled frontend, updated here).

## Verification and unresolved questions

- Go (`go test ./...`): collector `oldPriceCents` cases (number, `[n]`, absent, string, negative, two-element array) and a fixture page with `old_price` producing `OldPriceCents`; service: new-item success records one old-price row, scheduled/refresh collection records none, failed first collection records none (FR-002/007); store: undated row returned last in `priceHistory` with `timestamp: null, oldPrice: true`, dated row ordered by date, cascade delete (FR-008), migration idempotent on an existing database.
- Frontend: `npm run build` (typecheck); QA adds a LeBoncoin listing with a crossed-out price and checks details history and list periods show "Old price"; an Amazon add and a listing without old price show no extra entry (FR-006/009).
- Unresolved: none functional. Technical limitation: old-price date field unknown; entries are undated until a field is identified.
