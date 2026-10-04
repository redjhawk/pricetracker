# Technical specification: LeBoncoin AI review

Status: ready
Functional specification: [functional.md](functional.md), FR-LBC-AIR-001–014 (ready, 2026-10-04)
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "Claude token settings and AI reviews (2026-10-04)"
Companions: [Claude token settings technical](../claude-token-settings/technical.md) (Claude client, token storage), [item refresh](../item-refresh/technical.md)

## Requirement mapping

`AIR-*` abbreviates `FR-LBC-AIR-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-LBC-AIR-001 | AIR-002 | Collector parses full listing details from `__NEXT_DATA__` into `CollectionResult.Listing` (in memory, not persisted) | `internal/leboncoin/collector.go`, `internal/model/model.go` | Collector tests with a fixture page: description, attributes, all image URLs, location, date, seller type |
| TS-LBC-AIR-002 | AIR-002, AIR-003, AIR-014 | `claude.Client.Review`: prompt (text + up to 10 image URL blocks + price history), structured JSON output | `internal/claude/review.go` | `httptest` test asserting request body (preamble first, images, history) |
| TS-LBC-AIR-003 | AIR-003, AIR-014 | Robust parser: extract the JSON object, normalize and validate enums and fields | `internal/claude/review.go` | Table tests: fenced JSON, prose around JSON, label variants, invalid enum, missing field, no photos |
| TS-LBC-AIR-004 | AIR-004, AIR-007, AIR-011, AIR-012 | SQLite table `ai_reviews` (all attempts kept, FK cascade); interrupted `pending` rows failed at startup | `internal/store/store.go` | Store tests: insert/complete, history order, cascade on item delete, restart cleanup |
| TS-LBC-AIR-005 | AIR-001, AIR-009, AIR-010 | Triggers in the collection worker: first collection of a newly added LeBoncoin item; successful collection whose price differs from the previous recorded price | `internal/service/service.go`, `internal/store/store.go` | Service tests with fake collector/Claude: add → review; same price → none; changed price → review; pre-existing item → none |
| TS-LBC-AIR-006 | AIR-001, AIR-005–008 | Asynchronous review worker, at most one per item (in-memory set), token read at start, safe error messages | `internal/service/service.go` | Service tests: duplicate requests → one run; no token → no row; token rejected → failed row + `lastRejectedAt` |
| TS-LBC-AIR-007 | AIR-004–011 | `aiReview` field on `GET /api/v1/items/{id}`; `POST /api/v1/items/{id}/ai-review` | `internal/httpapi/server.go`, `internal/model/model.go` | Handler tests: 202/404/409/422; `aiReview` null for Amazon, absent from list |
| TS-LBC-AIR-008 | AIR-003–011, AIR-013, AIR-014 | Carbon "AI review" section in `ItemDetail`, refresh button, polling while running, history accordion | `src/components/AiReview.tsx` (new), `src/components/ItemDetail.tsx`, `src/App.tsx`, `src/api/items.ts`, `src/types.ts`, `src/index.css` (layout only if needed) | Playwright mocked-API spec `tests/leboncoin-ai-review.spec.ts`; manual screen-reader/400 px QA |

## Frontend

**Types/API (`src/types.ts`, `src/api/items.ts`).** Add `AiReview`, `AiReviewContent`, `AiReviewState` mirroring the API (prices kept as cents in the review types and formatted with the existing euro formatter / `cents / 100`). `TrackedItem.aiReview?: AiReviewState | null` passed through `mapItem` unchanged. New `requestAiReview(id): Promise<{ requestedAt: string; alreadyRunning: boolean }>` (`POST /api/v1/items/{id}/ai-review`).

**Component `src/components/AiReview.tsx`** (keeps `ItemDetail` readable), rendered by `ItemDetail` only when `item.platform === "leboncoin"` (AIR-004), as `<section aria-labelledby="ai-review-heading">` with `<h2 id="ai-review-heading">AI review</h2>`. Props: `state: AiReviewState`, `currentPriceCents: number | null`, `onRefresh()`, `requesting: boolean`, `requestError: string | null`.

Header row: Carbon `Button kind="tertiary" size="sm" renderIcon={Renew}` "Refresh AI review", disabled when `!state.tokenConfigured || state.running || requesting` (AIR-005, AIR-006, AIR-008).

States, in rendering order:

| Condition | Rendering |
| --- | --- |
| `!tokenConfigured` | `InlineNotification kind="info" lowContrast hideCloseButton` "Configure a Claude token in Settings" (AIR-008). Any stored reviews are still shown below. |
| `running` or `requesting` | `InlineLoading description="AI review in progress…"` inside an `aria-live="polite"` region (AIR-006, AIR-013) |
| `requestError` (POST failed) | `InlineNotification kind="error"` "Could not start the AI review" + message |
| `lastAttempt.status === "failed"` and not running | `InlineNotification kind="error" lowContrast hideCloseButton` title "The last AI review failed on {completedAt}", subtitle `lastAttempt.errorMessage` (AIR-007) |
| `latest === null` and not running | Text "No AI review yet." plus, when a token exists, "Use Refresh AI review to request one." (AIR-010) |
| `latest` present | Review body (below) |

Review body for `latest`:

- Caption: "Reviewed on {createdAt} at {price}" (price "Free" when 0 cents, "unknown" when null). When `currentPriceCents !== null && latest.priceCents !== currentPriceCents`: `InlineNotification kind="warning" lowContrast hideCloseButton` "This review was made at an older price ({reviewed price}); the current price is {current price}." (AIR-009). When a failed attempt is shown above, the caption reads as the latest *successful* review with its own date (AIR-007).
- Three Carbon `Tag`s with **text labels** (`Price: Good deal`, `Condition: Good`, `Recommendation: Negotiate`; colour secondary: green/gray/red by meaning; condition "Not assessable" gray), each followed by its explanation paragraph (AIR-003, AIR-013).
- `StructuredList` or simple definition list (`<dl>`), labelled parts (AIR-014): "Fair price range" `{min} – {max}`; "Suggested offer"; "Risk signs" — bullet list or "No scam or risk signs were found."; "Missing accessories or information" list or "Nothing notable."; "Questions to ask the seller" list; "Description vs photos" — "Matches the photos" / "Does not match the photos" / "Cannot be compared", the explanation, mismatches list.
- History (AIR-011): when `history.length > 1`, a Carbon `Accordion` with one `AccordionItem` per *previous* review (history minus the latest), titled "{createdAt} — {price} — {recommendation label}", content = the same review body component (no stale warning). Keyboard operable by Carbon.

Dates use the existing `en-GB` UTC `Intl.DateTimeFormat` pattern of `ItemDetail`. Labels map from enums: `good_deal`→"Good deal", `fair`→"Fair", `overpriced`→"Overpriced", `excellent`/`good`/`fair`/`poor`, `null` condition → "Not assessable", `buy`/`negotiate`/`avoid`.

**Page orchestration (`src/App.tsx`).** New state `aiReviewRequesting`, `aiReviewError` (reset when `itemId` changes). `handleAiReviewRefresh`: guard on `requesting`, `await requestAiReview(id)`, then `getItem(id)` to update `detailItem`; on error set `aiReviewError`. Extend the existing detail polling condition: poll every 3 s while `detailItem.status === "pending" || detailRefreshing || detailItem.aiReview?.running`. Polling stops when none applies. After a price refresh completes, the poll keeps running if the server reports a review started (`running`).

## Backend

### Listing details (TS-LBC-AIR-001)

Decision: listing details are **not persisted**. Every review uses details from a LeBoncoin fetch made for that review: automatic reviews reuse the in-memory result of the collection that triggered them; a manual refresh performs a fresh fetch (AIR-005 "from current listing data"). This also covers pre-existing items (AIR-010) with no migration, and avoids storing large listing text. Cost: one extra LeBoncoin request per manual review.

`internal/model`:

```go
type ListingDetails struct {
	Title       string
	Description string
	PriceCents  *int64 // nil if not detected; 0 for free/donation listings
	ImageURLs   []string // all https://img.leboncoin.fr/ URLs, urls_large preferred, deduplicated, in listing order
	Attributes  []ListingAttribute
	Category    string
	City, Zipcode, Department, Region string
	PublishedAt string // first_publication_date as given
	SellerType  string // "private" | "pro" | "" (owner.type)
}
type ListingAttribute struct{ Label, Value string } // key_label (fallback key) / value_label (fallback value)
```

`CollectionResult` gains `Listing *model.ListingDetails`, set by the LeBoncoin collector whenever the ad was verified and active (success *and* `price_not_found`). `listingData` gains `Attributes []{Key, KeyLabel, Value, ValueLabel}`, `Location{City, Zipcode, DepartmentName, RegionName}`, `FirstPublicationDate`, `CategoryName`, `Owner{Type}`. Strings are trimmed; description capped at 20 000 characters, at most 100 attributes. Amazon collection is unchanged. No new logging of listing content.

### Claude review request (TS-LBC-AIR-002)

`internal/claude/review.go`:

```go
type ReviewInput struct {
	Listing      model.ListingDetails
	URL          string
	PriceHistory []model.Observation // oldest → newest, at most the latest 100
}
func (c *Client) Review(ctx context.Context, token string, input ReviewInput) (model.AIReviewContent, error)
```

Request body: `model: claude.Model`, `max_tokens: 4096`, `system`: block 1 exactly the preamble, block 2 the reviewer instructions; one `user` message whose content is:

1. One `text` block: the listing as indented JSON (title, URL, price in euros or "free", description, category, attributes, location, publication date, seller type, number of photos) and the price history (date, euros), followed by the required output schema.
2. Up to **10** `image` blocks `{"type":"image","source":{"type":"url","url":"<img.leboncoin.fr URL>"}}`, the first 10 in listing order. If more exist, the text states "Only the first 10 of N photos are attached." If none, the text states "The listing has no photos."

Reviewer instructions (constant): act as an expert second-hand buyer for the French market; write in English; base the price opinion on general market knowledge and the price history; assess condition only from the photos and compare with the seller-stated condition; if no photos, set condition rating to null and say it cannot be assessed; for a free listing say it is free; list scam/risk signs or an empty list; respond with **only** one JSON object matching the schema, no Markdown.

Output schema (also the stored/API `review` object, amounts in integer euro cents):

```json
{
  "price": { "rating": "good_deal | fair | overpriced", "explanation": "…" },
  "condition": { "rating": "excellent | good | fair | poor | null", "explanation": "…" },
  "recommendation": { "rating": "buy | negotiate | avoid", "explanation": "…" },
  "fairPrice": { "minCents": 0, "maxCents": 0, "suggestedOfferCents": 0 },
  "risks": ["…"],
  "missingInformation": ["…"],
  "sellerQuestions": ["…"],
  "descriptionVsPhotos": { "matches": true, "explanation": "…", "mismatches": ["…"] }
}
```

The image URLs are sent to Anthropic, which fetches them; a URL Claude cannot fetch makes the API return 400 → `ErrBadResponse` (review fails with a safe message, retry possible).

### Parsing (TS-LBC-AIR-003)

1. Response must be 2xx JSON with `content[]`; concatenate `text` blocks. HTTP errors map as in TS-CLT-SET-002 (401/403 → `ErrRejected`, 429 → `ErrUsageLimit`, network/5xx → `ErrUnreachable`, other 4xx → `ErrBadResponse`). `stop_reason == "max_tokens"` → `ErrBadResponse`.
2. Strip optional Markdown fences; take the substring from the first `{` to the last `}`; `json.Unmarshal` into a struct with string/pointer fields (unknown fields ignored).
3. Normalize enum strings: lowercase, trim, spaces/hyphens → `_` (so "Good deal" → `good_deal`). Validate `price.rating` ∈ {good_deal, fair, overpriced}, `condition.rating` ∈ {excellent, good, fair, poor} or null/empty/"not_assessable" → null, `recommendation.rating` ∈ {buy, negotiate, avoid}. All explanations non-empty after trim. `fairPrice` values ≥ 0 and `minCents ≤ maxCents` (swap not attempted: invalid). Nil lists → empty lists; empty strings removed; `descriptionVsPhotos.matches` may be null.
4. Any failure → `ErrBadResponse`. The raw Claude text is never stored or logged.

`model.AIReviewContent` mirrors the schema with JSON tags (`Condition.Rating *string`).

### Persistence (TS-LBC-AIR-004)

```sql
CREATE TABLE IF NOT EXISTS ai_reviews (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  status TEXT NOT NULL CHECK (status IN ('pending', 'succeeded', 'failed')),
  price_cents INTEGER CHECK (price_cents IS NULL OR price_cents >= 0),
  review_json TEXT,
  error_message TEXT,
  created_at TEXT NOT NULL,
  completed_at TEXT
);
CREATE INDEX IF NOT EXISTS ai_reviews_item_time ON ai_reviews(item_id, created_at DESC, id DESC);
```

All rows are kept until item deletion (cascade, AIR-012). Store methods: `StartAIReview(ctx, itemID, now) (int64, error)`; `FinishAIReview(ctx, id, status, priceCents *int64, review *model.AIReviewContent, errorMessage string, now) error` (marshals `review_json`; an update of 0 rows — item deleted — is not an error); `AIReviews(ctx, itemID) (latestAttempt *model.AIReview, succeeded []model.AIReview, error)` (succeeded newest first; at most 50 returned); `FailInterruptedAIReviews(ctx, now)` called once from `Store.Open` after the schema: `pending` → `failed`, "The review was interrupted by a server restart." (no goroutine survives a restart). `LatestPriceCents(ctx, itemID) (*int64, error)` and `PriceHistory(ctx, itemID, limit)` for triggers and the prompt (reuse existing observation queries where possible).

### Triggers (TS-LBC-AIR-005)

In `collectReserved` (collection worker), for `platform == "leboncoin"`:

- Before `RecordCollection`, read `previous := store.LatestPriceCents`.
- After a successful record: request a review with the collected `result.Listing` when (a) this collection is the first one of a newly added item, or (b) `result.Result == "success"`, `previous != nil` and `*previous != result.AmountCents` (AIR-009). A same-price check never triggers (corner case).
- (a) is signalled by a `newItem bool` parameter: `Add`'s goroutine calls `collect(ctx, id, true)`; scheduler/refresh pass `false`. If that first collection did not yield `result.Listing` (unavailable, request error), the review still starts and fails immediately with "The listing could not be retrieved from LeBoncoin." (AIR-001 + AIR-007), so the operator sees why.
- Items added before the feature never satisfy (a), and (b) only applies to later price changes, as specified; there is no backfill (AIR-010).

### Review worker (TS-LBC-AIR-006)

`func (s *Service) startReview(itemID string, details *model.ListingDetails, fetchFresh bool) (started bool)`:

1. Read the Claude token (`store.ClaudeToken`). If no value: return false, no row (AIR-008).
2. Under `s.mu`: if `s.reviewing[itemID]` exists return false (AIR-006); else add it.
3. `StartAIReview` (row `pending`, visible as `running`), then `s.workers.Add(1)`; goroutine with `s.workerContext`:
   - If `fetchFresh`: run the same LeBoncoin fetch as a price check (`collectLeboncoin`, which also records the session outcome) **without** recording a price observation or collection attempt, keeping refresh price and refresh review independent (D-5). Not `Listing` → failed "The listing could not be retrieved from LeBoncoin."
   - Load price history (latest 100); call `claude.Review` with the token captured in step 1 (a token changed during the run does not affect it — corner case) under a 150 s context.
   - Success: `FinishAIReview(succeeded, details.PriceCents, review)`, `ClearClaudeTokenRejected(revision)`.
   - Errors → `failed` with message: `ErrRejected` "Claude rejected the token. Replace it in Settings."; `ErrUsageLimit` "Claude usage limit reached. Try again later."; both also `MarkClaudeTokenRejected(revision, now)` (FR-CLT-SET-009). `ErrUnreachable` "Claude could not be reached. Try again later."; `ErrBadResponse` "Claude returned an unusable review. Try again."; context cancelled at shutdown → "The review was interrupted by a server restart."
   - Always remove `itemID` from `s.reviewing`.
   - Logs: item ID, outcome, HTTP status class; never the token, prompt or response.
4. Concurrency: reviews do not use `refreshSlots`; at most one per item, and reviews across items are naturally bounded by add/price-change frequency.

`RequestAIReview(ctx, id) (requestedAt time.Time, alreadyRunning bool, err error)`: item missing → `sql.ErrNoRows`; Amazon → `&Error{422, "AI_REVIEW_UNSUPPORTED", "AI reviews are available for LeBoncoin items only."}`; no token → `&Error{409, "CLAUDE_TOKEN_MISSING", "Configure a Claude token in Settings."}`; already running → `alreadyRunning = true` (no new run); else `startReview(id, nil, true)`.

`Get` adds `aiReview` for LeBoncoin items: `tokenConfigured` (token value non-null), `running` (`s.reviewing` contains the item), `lastAttempt`, `latest` (first succeeded), `history` (succeeded, newest first). `List` does not include it.

## API

Contract: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) "Claude token settings and AI reviews (2026-10-04)": new `aiReview` field in the item **details** response only (null for Amazon), new `POST /api/v1/items/{id}/ai-review` (202 accepted, asynchronous; poll `GET /api/v1/items/{id}` while `aiReview.running`). Existing fields, statuses and endpoints unchanged; `GET /api/v1/items` unchanged. Error codes: `ITEM_NOT_FOUND` 404, `CLAUDE_TOKEN_MISSING` 409, `AI_REVIEW_UNSUPPORTED` 422, `METHOD_NOT_ALLOWED` 405.

## Scope and refactoring

Change boundary: `internal/leboncoin/collector.go`, `internal/claude/review.go` (new), `internal/model/model.go`, `internal/store/store.go`, `internal/service/service.go`, `internal/httpapi/server.go`, `src/types.ts`, `src/api/items.ts`, `src/App.tsx`, `src/components/ItemDetail.tsx`, `src/components/AiReview.tsx` (new), `src/index.css` (only if layout needs it), tests. No config, dependency or Amazon change.

Refactoring R-AIR-1 (perform, PR 4): add the `newItem bool` parameter by renaming `collectReserved(ctx, id)` to `collectReserved(ctx, id, newItem)` and adding it to `Collect`'s internal path (`Add` uses an internal `collectNew`); `Collect(ctx, id)` keeps its signature for the scheduler. Reason: AIR-001 needs to know the first collection; alternative (database flag) is heavier. Risk: low, covered by existing service tests.

Refactoring R-AIR-2 (declined): extracting a "fetch without recording" helper is unnecessary because `collectLeboncoin` already fetches without recording; the review worker calls it unchanged.

Risks: Claude may ignore the JSON instruction (handled as failed review with retry); Anthropic must fetch LeBoncoin image URLs (public CDN); subscription usage limits (surfaced as usage-limit failure).

## Verification and unresolved questions

- `go build ./...`, `go test ./...`, `npm run build`, `npx playwright test tests/leboncoin-ai-review.spec.ts`.
- QA cases: add LeBoncoin item with token → pending then review (AIR-001); without token → message, button disabled, no row (AIR-008); refresh button → progress, new review; double click → one run (AIR-006); failed review with previous one → error + old review with date (AIR-007); price change (dev seed/mocked) → automatic new review, stale-price notice meanwhile (AIR-009); pre-existing item → "No AI review yet" then refresh (AIR-010); history accordion with ≥2 reviews (AIR-011); delete item → reviews gone (AIR-012); listing without photos → condition "Not assessable"; free listing; Amazon details page has no section (AIR-004); keyboard, screen reader live region, 400 px (AIR-013); token absent from responses and logs.
- Unresolved questions: none.
