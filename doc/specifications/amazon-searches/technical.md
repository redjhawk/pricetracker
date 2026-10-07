# Technical specification: Amazon searches

Status: ready
Functional specification: [functional.md](functional.md), FR-AMZ-SEARCH-001–016 (ready, 2026-10-07)
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "Amazon searches (2026-10-07)"
Companions: [Amazon human browsing technical](../amazon-human-browsing/technical.md) (worker, gate, windows), [Amazon AI review technical](../amazon-ai-review/technical.md)

## Requirement mapping

`SEARCH-*` abbreviates `FR-AMZ-SEARCH-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-AMZ-SEARCH-001 | 009, 010, 011, 012, 013 | Search items are ordinary rows of the existing `items` table (`platform = 'amazon'`). New column `items.tracked` (1 = in the Amazon tab). `UNIQUE(owner_id, canonical_url)` already makes one item per user and product, so sharing needs no extra logic. | `internal/store/store.go` | Store tests: same ASIN in two searches → one item row; tracked item reused by a search |
| TS-AMZ-SEARCH-002 | 002, 003, 005, 012, 014 | New tables `amazon_searches` and `amazon_search_items` (ordered membership, cascade on both sides) | `internal/store/searches.go` (new) | Store tests: fresh DB, upgrade of an existing DB, reopen, cascade |
| TS-AMZ-SEARCH-003 | 003, 014 | `amazon.ParseSearchURL`: `http`/`https`, host `www.amazon.<de|fr|es|it|nl|be>` or bare host, any path, ≤ 2048 chars; URL stored trimmed and otherwise unchanged; duplicate = same trimmed URL for the same owner | `internal/amazon/search.go` (new) | Table test: issue #56 URL accepted; LeBoncoin, `ftp:`, `amazon.com`, text rejected |
| TS-AMZ-SEARCH-004 | 005 | `amazon.Collector.FetchSearch` parses the results page, first 30 distinct ASINs in page order with link, title, price; capture is written once in one transaction and never changed | `internal/amazon/search.go`, `internal/store/searches.go` | Fixture tests (60 results → 30, order kept, duplicates skipped); store test that a second capture is refused |
| TS-AMZ-SEARCH-005 | 001, 002, 004, 006, 015, 016 | Third tab "Amazon searches"; `AmazonSearchesPage` (list + add) and `AmazonSearchItems` (one search) | `src/components/TrackedItemsPage.tsx`, `src/components/AmazonSearchesPage.tsx` (new), `src/components/AmazonSearchItems.tsx` (new), `src/App.tsx`, `src/api/searches.ts` (new), `src/types.ts` | Playwright mocked-API spec `tests/amazon-searches.spec.ts`; QA keyboard / 400 px |
| TS-AMZ-SEARCH-006 | 007, 008 | Item checks reuse `amazon.Collector` and `store.RecordCollection`, so history, `unavailable` status and "failure keeps last price" are unchanged | `internal/service/search_worker.go` (see human browsing) | Service test: unavailable keeps last price |
| TS-AMZ-SEARCH-007 | 010, 011 | `POST …/items/{itemId}/track` sets `tracked = 1` and schedules `next_check_at` | store, service, `internal/httpapi/searches.go` (new) | Handler tests 200 / 404 / 409 |
| TS-AMZ-SEARCH-008 | 012 | Search delete in one transaction removes the search and orphan untracked items; tracked-item delete of an item still in a search only untracks it | `internal/store/searches.go`, `internal/store/store.go` (`Delete`) | Store tests for each FR-012 case |
| TS-AMZ-SEARCH-009 | 013 | Every search query filters on `owner_id = ownerFrom(ctx)`; another owner's search is `404 SEARCH_NOT_FOUND` | service, store | Ownership handler test (two users, same URL) |

## Backend

### Persistence (additive, idempotent, in `Store.Open`)

```sql
-- items: new column via ensureColumn; existing rows stay tracked
ALTER TABLE items ADD COLUMN tracked INTEGER NOT NULL DEFAULT 1 CHECK (tracked IN (0, 1));

CREATE TABLE IF NOT EXISTS amazon_searches (
  id TEXT PRIMARY KEY,                     -- 32 hex chars, like item ids
  owner_id INTEGER NOT NULL,               -- 0 = open mode
  url TEXT NOT NULL,                       -- as entered, trimmed
  added_at TEXT NOT NULL,
  captured_at TEXT,                        -- NULL until the first results page succeeds
  next_run_at TEXT NOT NULL,               -- when the next pass is due (UTC)
  pass_position INTEGER NOT NULL DEFAULT 0,-- 0 = pass starts with the results page; n = next item position
  last_error_at TEXT,                      -- latest failed results-page request, cleared on success
  last_error_message TEXT,
  UNIQUE (owner_id, url)
);
CREATE TABLE IF NOT EXISTS amazon_search_items (
  search_id TEXT NOT NULL REFERENCES amazon_searches(id) ON DELETE CASCADE,
  item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  position INTEGER NOT NULL,               -- 1..30, Amazon order
  PRIMARY KEY (search_id, item_id),
  UNIQUE (search_id, position)
);
CREATE INDEX IF NOT EXISTS amazon_search_items_item ON amazon_search_items(item_id);
```

- Search items created by a capture are inserted with `tracked = 0` and `next_check_at = NULL`, so the existing scheduler never checks them (FR-AMZ-HUMAN-001/002).
- `store.List`, `store.IDs`, `store.DueIDs`, `store.IsCanonicalTracked` add `AND tracked = 1`. `store.Listing` and `store.get` are unchanged (details of a search item work through `GET /items/{id}`), and `get` also reads `tracked`.
- `store.Delete(owner, id)` (tracked list delete): in one transaction, if the item belongs to a search of the owner, `UPDATE items SET tracked = 0, next_check_at = NULL`; otherwise delete as today. Only `tracked = 1` rows are deletable (an untracked item returns not found). Satisfies "deleted from the tracked list but still in a search stays in the search".
- `service.Add` of an Amazon URL whose canonical URL exists with `tracked = 0` for the owner: set `tracked = 1` and `next_check_at = NextCheckAt(now)`, return `201` with the existing item and start an immediate collection as for a new item (it goes through the Amazon gate). The item keeps its history (FR-SEARCH-009).

New store functions (`internal/store/searches.go`): `InsertSearch`, `Searches(owner)`, `Search(owner, id)`, `SearchItemIDs(searchID)` (ordered), `CaptureSearch(searchID, owner, []CapturedItem, now)`, `DeleteSearch(owner, id)`, `TrackItem(owner, itemID, nextCheck)`, `SetSearchProgress(id, position, nextRunAt)`, `SetSearchError(id, at, message)`, `DueSearches(now)`.

`CaptureSearch` (one transaction, never holding a network call): refuse when `captured_at` is not NULL (FR-005 frozen set); for each captured product, in order, `INSERT … ON CONFLICT(owner_id, canonical_url) DO NOTHING` an item (`tracked = 0`, `asin`, `marketplace`, `canonical_url` = `amazon.ParseURL(link).Canonical` (same key as the Amazon tab, so a tracked item is found), `url` = the product link from the results page reduced to `https://www.amazon.<tld>/dp/<ASIN>`, `title` from the card), read back its id, insert membership with its position; when the card shows a price insert one `price_observations` row (no `collection_attempts` row, so status stays `pending` until the item is read); set `captured_at`. An existing item keeps its title and history.

`DeleteSearch` (one transaction): read the search's item ids, delete the search (cascade removes memberships), then `DELETE FROM items WHERE id IN (…) AND tracked = 0 AND id NOT IN (SELECT item_id FROM amazon_search_items)`; observations and reviews cascade. A running pass notices the missing search before its next action (human-browsing TS-AMZ-HUMAN-006).

### Service (`internal/service/searches.go`, new)

- `AddSearch(ctx, rawURL)`: trim, `amazon.ParseSearchURL` (`400 INVALID_URL` for empty/malformed or non-http(s); `422 UNSUPPORTED_SEARCH` for a non-Amazon or unsupported marketplace host); duplicate → `409 SEARCH_ALREADY_ADDED` whose message names the existing search label; insert with `next_run_at = now` (the worker waits for the window). No network work happens in the request.
- `Searches(ctx)`, `SearchDetails(ctx, id)`, `DeleteSearch(ctx, id)`, `TrackSearchItem(ctx, searchID, itemID)` (item must be in that search of that owner; already tracked → `409 ITEM_ALREADY_TRACKED`).
- Search response state (computed, not stored): `running` when the worker is processing it; `stopped` when Amazon requests are blocked (gate) and the search has unfinished work (`captured_at IS NULL`, `pass_position > 0` or `next_run_at <= now`); `waiting` when it has no capture yet or its pass is due; otherwise `done`. `waitingUntil` = next window start when `waiting` and outside a window, else `null`.
- `label` = host without `www.` + path, e.g. `amazon.fr/joursprime/`.
- Item responses gain `tracked` (boolean). `RefreshItem` on an untracked item returns `409 ITEM_NOT_TRACKED` (manual refresh would bypass the request windows).

### Validation and failure

Request bodies follow existing handlers: one JSON value, 1 MiB limit, `INVALID_JSON`. Errors never contain upstream bodies (they go to logs only, see human browsing).

## API

New routes in `internal/httpapi/searches.go`, dispatched from `handleData` by the prefix `/api/v1/amazon-searches`; all are protected/owner-scoped like `/api/v1/items`. Full contract: API_SPECIFICATION.md "Amazon searches (2026-10-07)":

- `GET /api/v1/amazon-searches` → `{ searches, amazonRequests }`
- `POST /api/v1/amazon-searches` `{ url }` → `201` search
- `GET /api/v1/amazon-searches/{id}` → `{ search, amazonRequests, items }`
- `POST /api/v1/amazon-searches/{id}/refresh` and `GET /api/v1/amazon/requests`: see [human browsing](../amazon-human-browsing/technical.md)
- `DELETE /api/v1/amazon-searches/{id}` → `204`
- `POST /api/v1/amazon-searches/{id}/items/{itemId}/track` → `200` item
- Item responses add `tracked`; `GET /api/v1/items` returns tracked items only (unchanged meaning for existing data).

## Frontend

- `TrackedItemsPage`: add `<Tab>Amazon searches</Tab>`; its panel renders `AmazonSearchesPage`. The header "Add item" button and the Amazon/LeBoncoin tables are unchanged (FR-001, D-21). The selected tab type becomes `"amazon" | "leboncoin" | "amazon-searches"` in `App.tsx`.
- `AmazonSearchesPage` (owns its own loading/error state, polls `GET /amazon-searches` every 30 s while visible): an inline add form (Carbon `TextInput` labelled "Amazon search URL", `Button` "Add search", inline `invalid`/`invalidText` from the API message); a Carbon `DataTable` with columns Search (link button with `label`, full URL in a `title`), Added, Items, State (`Tag` + text: "Waiting for next window (22:00)", "Running", "Done", "Stopped"), Actions (Delete; Refresh, see human browsing). States: `InlineLoading`; error `InlineNotification` with Retry; empty "No Amazon search yet. Add a results URL from Amazon.". Delete uses a Carbon danger `Modal` ("Delete search? Items that are not tracked or in another search are deleted with their price history and reviews.").
- `AmazonSearchItems` at path `/searches/{id}` (same `pathname` navigation as `/items/{id}`): heading with the label, back button, state line; a table like the Amazon tab rows (title + ASIN/marketplace, latest + recent prices, `StatusTag` — `unavailable` shows "Unreachable" as today, AI review summary from [Amazon AI review](../amazon-ai-review/technical.md)), and per row: "Details" (navigates to `/items/{itemId}`), "Move to tracked Amazon items" (`Button` kind ghost) or a `Tag` "Tracked" when `tracked` is true (FR-011). States: waiting with no items ("Waiting for the first retrieval at 22:00"), stopped, loading, error with Retry.
- `ItemDetail` for an item with `tracked: false`: hide Refresh and Delete (not offered for search items), keep history and AI review.
- Accessibility: tab labelled by text; actions are buttons with visible text; state conveyed by text; at ~400 px the tables scroll inside `TableContainer` as the existing ones.

## Scope and refactoring

- Refactoring R-1 (performed, part 3): add `tracked` filtering to the existing list queries and change `store.Delete` to untrack. Reason: reuse the items table instead of duplicating it (a parallel table would duplicate history, status and details code). Risk: existing queries returning search items; mitigated by store tests on `List`, `IDs`, `DueIDs`, `IsCanonicalTracked`.
- No change to LeBoncoin, purchase goals, or the existing Amazon tab rendering.

## Verification and unresolved questions

- `go vet ./...`, `go test ./...`, `npm run build`.
- Store: migration from a DB without the new tables/column; capture of 30; capture refused twice; share across two searches; delete-search cases (orphan removed, tracked kept, other-search kept); tracked delete untracks.
- HTTP: every status code of the contract; ownership (user B gets 404 on A's search).
- Playwright (mocked API): tab present, add/duplicate/invalid, list states, open a search, move to tracked, delete confirmation.
- QA (FR-AMZ-SEARCH acceptance criteria) on the running app with the issue #56 URL (results depend on Amazon; record actual outcome).
- Risk: Amazon deals pages may render results with JavaScript; if the fixture-based parser finds no ASIN on the real page, the first retrieval fails visibly (FR-015) and is logged with the response for diagnosis. No functional question is open.
