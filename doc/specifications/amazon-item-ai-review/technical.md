# Technical specification: Amazon item AI review and purchase goal

Status: ready (issue #305)
Functional specification: [functional.md](functional.md), FR-AMZ-ITEM-AIR-001–034 (specified, decisions D-1–D-8); amended rows FR-PURCHASE-GOAL-001, -008, FR-LBC-AIR-004, FR-AMZ-SRCH-DEL-005, FR-CLT-SET-014, FR-AMZ-HCHK-002
Test cases: [testcases.md](../../changes/2026-11-24-0836-issue-305-purchase-goal-and-ai-review-is-also-avai/testcases.md), TC-001–029
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "Amazon item AI review and purchase goal (issue #305)"
Companions: [LeBoncoin AI review technical](../leboncoin-ai-review/technical.md) (reused review machinery), [LeBoncoin purchase goal technical](../leboncoin-purchase-goal/technical.md), [Amazon request stop technical](../amazon-request-stop/technical.md) (gate and stop unchanged), [Claude token settings technical](../claude-token-settings/technical.md)

## Requirement mapping

`AIR-*` abbreviates `FR-AMZ-ITEM-AIR-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change |
| --- | --- | --- | --- |
| TS-AMZ-ITEM-AIR-001 | AIR-030, FR-AMZ-HCHK-002 | `parseProductPage` reads the customer rating and number of ratings from the product's own rating block into `CollectionResult.CustomerRating`; price, second-hand and failure results unchanged | `internal/amazon/collector.go`, `internal/model/model.go`, `internal/amazon/testdata/rating-*.html` (new) |
| TS-AMZ-ITEM-AIR-002 | AIR-030, AIR-031, AIR-016, AIR-007 | Two nullable `items` columns written by `RecordCollection` only when a product page was read; `AmazonCustomerRating` read for reviews; never in item responses | `internal/store/store.go`, `internal/store/amazon_rating.go` (new) |
| TS-AMZ-ITEM-AIR-003 | AIR-001–004, AIR-007, AIR-022, FR-PURCHASE-GOAL-001 | `Add` keeps the trimmed goal for Amazon too; `SetPurchaseGoal` saves an Amazon goal without starting a review (`reviewStarted: false`) | `internal/service/service.go`, `internal/service/purchase_goal.go` |
| TS-AMZ-ITEM-AIR-004 | AIR-005, AIR-018, AIR-020, AIR-021 | `claude.Client.ReviewAmazon`: English prompt with the recorded data, the goal block shared with LeBoncoin, small JSON schema | `internal/claude/amazon_review.go` (new), `internal/claude/review.go` |
| TS-AMZ-ITEM-AIR-005 | AIR-014, AIR-020 | Amazon response parser: price rating, fair range, recommendation `buy`/`wait`/`avoid`; anything else is `ErrBadResponse`; LeBoncoin-only fields dropped | `internal/claude/amazon_review.go` |
| TS-AMZ-ITEM-AIR-006 | AIR-032, AIR-033, AIR-034 | Pure availability decision `stopped` > `no_current_price` > available, from the stored stop, latest price and latest attempt | `internal/service/amazon_review.go` (new) |
| TS-AMZ-ITEM-AIR-007 | AIR-010–014, AIR-021, AIR-023, AIR-025, AIR-026, AIR-034 | `RequestAIReview` for Amazon: refusal codes before the token check, snapshot of recorded data, existing reservation (one review per item), worker calls only Claude; outcome stored by the existing `finishReview` | `internal/service/review.go`, `internal/service/amazon_review.go`, `internal/service/settings.go` |
| TS-AMZ-ITEM-AIR-008 | AIR-010, AIR-011, AIR-016, AIR-023 | Amazon reviews stored in `ai_reviews` unchanged (`price_cents` = reviewed price); decoding by the item's platform | `internal/store/ai_reviews.go`, `internal/model/model.go` |
| TS-AMZ-ITEM-AIR-009 | AIR-010, AIR-015, AIR-032, AIR-033, FR-LBC-AIR-004 | `aiReview` object for Amazon details with new `unavailableReason` (always `null` for LeBoncoin) | `internal/service/review.go`, `internal/model/model.go` |
| TS-AMZ-ITEM-AIR-010 | AIR-022, AIR-024, AIR-019 | No automatic Amazon review: the collection worker's review triggers stay LeBoncoin-only; LeBoncoin paths unchanged | `internal/service/service.go` (comments only) |
| TS-AMZ-ITEM-AIR-011 | AIR-007, AIR-024, AIR-031, FR-AMZ-SRCH-DEL-005 | Idempotent additive migration; existing Amazon items get an empty goal, no review and an unknown rating | `internal/store/amazon_rating.go`, `internal/store/store.go` |
| TS-AMZ-ITEM-AIR-012 | AIR-010–017, AIR-020, AIR-023, AIR-032, AIR-033 | New Carbon section `AmazonAiReview` on Amazon details | `src/components/AmazonAiReview.tsx` (new), `src/components/ItemDetail.tsx`, `src/types.ts` |
| TS-AMZ-ITEM-AIR-013 | AIR-001, AIR-002, AIR-006, FR-PURCHASE-GOAL-001, -008 | Goal field rendered for Amazon details; add-form helper no longer says "LeBoncoin only" | `src/components/ItemDetail.tsx`, `src/components/AddItemModal.tsx` |
| TS-AMZ-ITEM-AIR-014 | FR-CLT-SET-014 | Claude token help text mentions Amazon and LeBoncoin reviews | `src/components/SettingsModal.tsx` |

## Frontend

### Types (`src/types.ts`)

```ts
export interface AmazonAiReviewContent {
  price: { rating: "good_deal" | "fair" | "overpriced"; explanation: string };
  fairPrice: { minCents: number; maxCents: number };
  recommendation: { rating: "buy" | "wait" | "avoid"; explanation: string };
}
export interface AiReview<C = AiReviewContent> { /* unchanged fields */ review: C | null }
export interface AiReviewState<C = AiReviewContent> {
  /* unchanged fields */
  unavailableReason: "stopped" | "no_current_price" | null; // always null for LeBoncoin
}
// TrackedItem
aiReview?: AiReviewState<AiReviewContent> | AiReviewState<AmazonAiReviewContent> | null;
```

`src/api/items.ts` needs no change: `mapItem` passes `aiReview` through, `addItem` already sends `purchaseGoal`, `requestAiReview` and `savePurchaseGoal` are platform-neutral.

### `AmazonAiReview.tsx` (new, TS-AMZ-ITEM-AIR-012)

Separate component so the LeBoncoin `AiReview.tsx` stays unchanged (AIR-019). Props as `AiReview` minus `currentPriceCents`: `state: AiReviewState<AmazonAiReviewContent>`, `onRefresh()`, `requesting`, `requestError`. Dates use `dateTime` from `src/dateTime.ts`; prices the `de-DE` euro formatter of `ItemDetail` (`129,99 €`). `<section className="ai-review" aria-labelledby="ai-review-heading">` with `<h2 id="ai-review-heading">AI review</h2>` (AIR-017).

- Header button `Button kind="tertiary" size="sm" renderIcon={Renew}` "Refresh AI review", **rendered only when `unavailableReason === null`** (AIR-032, AIR-033); disabled when `!tokenConfigured || running || requesting` (AIR-013, AIR-015).
- Notices, in order (`InlineNotification kind="info" lowContrast hideCloseButton`, title = exact text):
  - `stopped`: "Amazon checks are stopped: restart the item with Refresh price to request an AI review."
  - `no_current_price`: "No current price: an AI review needs a current Amazon price."
  - otherwise, `!tokenConfigured`: "Configure a Claude token in Settings". Only one of the three is shown.
- `aria-live="polite"` region with `InlineLoading description="AI review in progress…"` while `running || requesting`.
- `requestError` (POST refused, e.g. the item was stopped meanwhile): `InlineNotification kind="error"` "Could not start the AI review" + API message.
- Last attempt failed and not running: error notification "The last AI review failed on {date}" + `errorMessage` (AIR-014).
- No `latest` and not running: "No AI review yet." plus " Use Refresh AI review to request one." when the button is shown and enabled.
- Latest review: footnote "Reviewed on {date} · Based on price {price}" (`priceCents`; AIR-023, no comparison with the current price), then the body:
  - `Tag` "Price: Good deal | Fair | Overpriced" (green/gray/red, text carries the meaning) + explanation;
  - `<dl className="ai-review-details">` "Fair price range" `{min} – {max}`;
  - `Tag` "Overall recommendation: Buy | Wait for a lower price | Avoid" (green/gray/red) + explanation.
  No condition, suggested offer, risks, accessories, seller questions or photo parts (AIR-020).
- History (AIR-011): Carbon `Accordion` of the previous succeeded reviews (history minus latest), item title "{date} — Based on price {price} — {recommendation label}", content = the same body. No goal is shown.
- Layout reuses the existing `ai-review*` classes (no CSS change expected; phone width AIR-017).

### `ItemDetail.tsx`, `AddItemModal.tsx`, `SettingsModal.tsx`

- `ItemDetail`: render `<PurchaseGoal>` for both platforms (TS-AMZ-ITEM-AIR-013); render `<AmazonAiReview>` when `item.platform === "amazon" && item.aiReview` (an absent or `null` `aiReview` renders nothing, as existing mocks send). LeBoncoin branch unchanged.
- `PurchaseGoal.tsx` unchanged: the server answers `reviewStarted: false` for Amazon, so the success text is "Purchase goal saved." and no pending review appears (AIR-003, AIR-006).
- `AddItemModal`: helper text becomes "Sent to the AI review, e.g. what you will use the item for." (drops "LeBoncoin only.").
- `SettingsModal`: `claudeHelp` becomes "Paste the token printed by claude setup-token (Claude Pro/Max subscription). It is used for Amazon and LeBoncoin AI reviews. Leave it empty to keep the saved token." (FR-CLT-SET-014).
- `App.tsx` unchanged: `handleAiReviewRefresh` (guarded by `aiReviewRequesting`) and detail polling while `aiReview.running` or a price check is pending are platform-neutral, so after **Refresh price** the polled details bring back the button (AIR-032, AIR-033).

## Backend

### Customer rating (TS-AMZ-ITEM-AIR-001, -002)

Model:

```go
// CustomerRating is the product rating shown on an Amazon product page.
type CustomerRating struct {
	StarsTenths *int64 // 45 = 4.5 out of 5; nil when not shown or unreadable
	Count       *int64 // number of ratings; nil when not shown or unreadable
}
// CollectionResult gains:
CustomerRating *CustomerRating // Amazon: set when a product page was read; nil keeps the stored rating
```

Parsing (`internal/amazon/collector.go`, `customerRating(page) model.CustomerRating`, called by `parseProductPage` for `success` and `price_not_found`; the text-matched `unavailable` branch, `classifyPage` unavailability and request failures leave it nil):

- Stars: the `title` attribute of the element with `id="acrPopover"`, else its text content; regex `([0-5](?:[.,][0-9])?)\s+(?:out of|sur|von|de|su|van)\s+5\b` (en, fr, de, es, it, nl) → tenths (`4,5` → 45, `5` → 50).
- Count: text content of `id="acrCustomerReviewText"`; first number with digit-group separators (space, NBSP, U+202F, `.`, `,`) removed → integer (`1 234 évaluations`, `1,234 ratings` → 1234).
- Only these two IDs, which belong to the product's own rating block, are read, so ratings of recommended products elsewhere on the page are ignored. Each value is independent; an unreadable value is nil. Parsing never changes `Result`, price, title, thumbnail or second-hand fields.

Storage: `ensureColumn(items, amazon_rating_tenths, "INTEGER CHECK (amazon_rating_tenths IS NULL OR amazon_rating_tenths BETWEEN 0 AND 50)")` and `ensureColumn(items, amazon_rating_count, "INTEGER CHECK (amazon_rating_count IS NULL OR amazon_rating_count >= 0)")`, from a new `ensureAmazonRatingColumns` called in `Store.Open` after `createAmazonRequestSchema` (additive, idempotent; existing rows `NULL` = unknown, AIR-031). Not added to `itemsColumns` (same as `amazon_stopped_at`). `RecordCollection`, inside its transaction, for an Amazon item with `result.CustomerRating != nil`: `UPDATE items SET amazon_rating_tenths = ?, amazon_rating_count = ? WHERE id = ?` (nil → `NULL`, so a page without ratings records them as absent). `RecordFailure` and nil leave the columns (a failed check keeps the last rating). New `AmazonCustomerRating(ctx, id) (model.CustomerRating, error)`. `get`, `List` and the API never select the columns. Deleting the item deletes them (AIR-016).

### Purchase goal (TS-AMZ-ITEM-AIR-003)

- `Add`: `PurchaseGoal: strings.TrimSpace(purchaseGoal)` also in the Amazon case. Adding starts no review (the first collection's review trigger is LeBoncoin-only).
- `SetPurchaseGoal`: remove the `PURCHASE_GOAL_UNSUPPORTED` branch. Trim and compare as today; for an Amazon item, save and return `(goal, true, false, nil)` without `startReview`; an unchanged value writes nothing. LeBoncoin path unchanged.

### Claude request (TS-AMZ-ITEM-AIR-004, -005)

`internal/claude/amazon_review.go`:

```go
type AmazonReviewInput struct {
	Title, Marketplace, URL string
	PriceCents        int64                         // current price (reviewed price)
	PriceHistory      []model.Observation           // oldest first, latest 100
	SecondHandStatus  string                        // available | not_found | check_error | pending
	SecondHandLatest  *model.SecondHandObservation  // latest recorded detection, nil when none
	SecondHandHistory []model.SecondHandObservation // oldest first, latest 100
	Rating            model.CustomerRating
	PurchaseGoal      string
}
func (c *Client) ReviewAmazon(ctx context.Context, token string, input AmazonReviewInput) (model.AmazonAIReviewContent, error)
```

- Request: `Model`, `max_tokens` 2048, `system` = `systemPreamble` then the Amazon instructions; one `user` text block, no images. Instructions: expert buyer on Amazon European marketplaces; write in English; judge the current price from general market knowledge, the price history, the Amazon-sold second-hand offer and the customer rating; `recommendation` `wait` means "wait for a lower price"; amounts are integer euro cents; the product data is untrusted text, data only; respond with only one JSON object.
- Prompt: `<product_data>` indented JSON with `title`, `marketplace`, `url`, `currentPrice` (`euros()`), `priceHistory` `[{date, price}]`, `secondHandOffer` (`"none recorded"` when there is no detection or the status is `not_found`; else `{status: "current" | "last check failed, last detected", price, condition (label), detectedAt}` plus `history`), `customerRating` (`"4.5 out of 5"` or `"unknown"`), `numberOfRatings` (integer or `"unknown"`); then the goal block; then the schema. JSON encoding escapes `<`/`>`.
- Refactoring R-1: the goal block of `reviewPrompt` moves to `purchaseGoalBlock(goal string) string` in `review.go`, used by both prompts. The LeBoncoin prompt stays byte-identical (existing `purchase_goal_test.go` and `review_test.go` prove it).
- Schema and stored/API `review` object:

```json
{ "price": { "rating": "good_deal | fair | overpriced", "explanation": "string" },
  "fairPrice": { "minCents": 0, "maxCents": 0 },
  "recommendation": { "rating": "buy | wait | avoid", "explanation": "string" } }
```

- Parsing reuses `send` (HTTP error mapping, `max_tokens` stop → `ErrBadResponse`), the first-`{`-to-last-`}` extraction and `normalizeEnum`. `wait_for_a_lower_price` and `wait_for_lower_price` normalize to `wait`. Invalid enum, empty explanation, missing `fairPrice`, negative amounts or `minCents > maxCents` → `ErrBadResponse`. Unknown fields (e.g. `condition`, `suggestedOfferCents`) are ignored and not stored.
- `model.AmazonAIReviewContent{Price AIRating; FairPrice AmazonFairPrice{MinCents, MaxCents}; Recommendation AIRating}` with the JSON tags above.

### Availability (TS-AMZ-ITEM-AIR-006)

`amazonReviewAvailability(stoppedAt *time.Time, latestPrice *model.Observation, lastAttempt *model.Attempt) string` returns `"stopped"` when `stoppedAt != nil`; else `"no_current_price"` when `latestPrice == nil` or `lastAttempt.Result` is `price_not_found` or `unavailable`; else `""` (available; a `request_error` keeps the last price). Inputs come from `store.Get` (stored values, no in-flight `pending` overlay), so a running check does not change the answer. It does not read the token.

### Request and worker (TS-AMZ-ITEM-AIR-007)

`RequestAIReview(ctx, id)` for an Amazon item (LeBoncoin unchanged; `AI_REVIEW_UNSUPPORTED` removed):

1. `ownedListing` (404 as today). `store.GetWithHistory(ctx, id)`; availability from it: `stopped` → `&Error{409, "AI_REVIEW_ITEM_STOPPED", "Amazon checks are stopped: restart the item with Refresh price to request an AI review."}`; `no_current_price` → `&Error{409, "AI_REVIEW_NO_CURRENT_PRICE", "No current price: an AI review needs a current Amazon price."}`. Checked before the token, so the refusal is the same with or without one; nothing is stored (AIR-034).
2. Build the `AmazonReviewInput` snapshot from that item (latest price, histories reversed to oldest first and capped at 100, second-hand status and latest detection) plus `AmazonCustomerRating` and `listing.PurchaseGoal`.
3. `reserveReview(id)`: no token → `409 CLAUDE_TOKEN_MISSING`; already running → `alreadyRunning: true` (AIR-013).
4. `StartAIReview` (pending row), `s.workers.Add(1)`, goroutine: `claude.ReviewAmazon` with the reserved token under a 150 s context, then the existing `finishReview(id, reviewID, token, &input.PriceCents, content, err)` and `releaseReview`. A row insert error releases the reservation (as `launchReview`).

The worker makes no Amazon request, records no observation or attempt, and never calls the Amazon gate, so the schedule and the consecutive failure count are unchanged (AIR-025, AIR-026). Failures use `reviewFailureMessage` and the token rejection marking of `finishReview` (AIR-014; token-revision rule covers TC-018). `reviewAgain` is never set for Amazon. Deleting the item meanwhile: `FinishAIReview` updates no row (TC-016). Logs: item ID and outcome only.

Refactoring R-2: the `claudeClient` interface (`settings.go`) gains `ReviewAmazon`; test fakes (`fakeClaude`, `fakeReviewer`) gain the method. The Amazon launch lives in `amazon_review.go` and shares `reserveReview`, `releaseReview` and `finishReview`; `launchReview`/`runReview` stay LeBoncoin-only. Risk: low.

### Stored reviews and details state (TS-AMZ-ITEM-AIR-008, -009)

- `ai_reviews` schema unchanged. `AIReviews(ctx, itemID)` keeps its signature; it reads the item's platform once (`SELECT platform FROM items WHERE id = ?`) and `scanAIReview` decodes `review_json` into `*model.AIReviewContent` (LeBoncoin) or `*model.AmazonAIReviewContent` (Amazon). LeBoncoin output is byte-identical. `model.AIReview.Review` comment updated.
- `model.AIReviewState` gains `UnavailableReason *string \`json:"unavailableReason"\``.
- `Service.AIReview(ctx, item)` returns the state for both platforms; for Amazon it sets `UnavailableReason` from `amazonReviewAvailability` on `store.Get(ctx, item.ID)`; `nil` for LeBoncoin. `List` still has no `aiReview`.

### No automatic review (TS-AMZ-ITEM-AIR-010)

`collectReserved` keeps `item.Platform == "leboncoin"` on both review triggers; no trigger is added for add, price change, goal save, token save (`SaveSettings` never starts reviews) or a restart by **Refresh price**. Only the comments of `Add`/`collect` are updated.

### Migration (TS-AMZ-ITEM-AIR-011)

Only the two `ensureColumn` calls. `purchase_goal` already exists with `''`. `removeAmazonSearches` only runs while `amazon_search_items` exists, so it never deletes reviews made after this change and reviews removed by issue #111 do not come back. Startup starts no review.

### Concurrency

Reservation under `s.mu` as for LeBoncoin; Claude calls hold no lock or transaction; the snapshot reads happen before the reservation. A check recorded between the snapshot and the reservation only makes the review use the data read at request time.

## API

New section "Amazon item AI review and purchase goal (issue #305)" in `API_SPECIFICATION.md`:

- `GET /api/v1/items/{id}`: `aiReview` is an object for Amazon items too, with the Amazon `review` shape above; new field `unavailableReason` (`"stopped"`, `"no_current_price"` or `null`; always `null` for LeBoncoin).
- `POST /api/v1/items/{id}/ai-review` on Amazon: `202` as for LeBoncoin, from recorded data; new `409 AI_REVIEW_ITEM_STOPPED`, `409 AI_REVIEW_NO_CURRENT_PRICE`; `422 AI_REVIEW_UNSUPPORTED` no longer returned.
- `POST /api/v1/items` stores `purchaseGoal` for Amazon; `PUT /api/v1/items/{id}/purchase-goal` saves Amazon goals with `reviewStarted: false`; `422 PURCHASE_GOAL_UNSUPPORTED` no longer returned. `purchaseGoal` is no longer always `""` for Amazon.
- Customer rating: never returned.
- Compatibility: additive fields; the only client is the embedded frontend, changed in the same release. LeBoncoin shapes and behaviour unchanged. Earlier statements contradicted by this change are marked as superseded in place.

## Test files per test case

| Test case | Test file |
| --- | --- |
| TC-001 | `internal/amazon/customer_rating_test.go` (new) with fixtures `internal/amazon/testdata/rating-fr.html`, `rating-en.html` (new); other languages and corner cases as inline snippets; existing fixtures asserted unchanged |
| TC-002, TC-003 | `internal/claude/amazon_review_test.go` (new) |
| TC-004 | `internal/service/amazon_review_test.go` (new) |
| TC-005, TC-006, TC-011, TC-012 | `internal/httpapi/amazon_item_ai_review_test.go` (new; `service.NewWithOptions` with a counting fake Amazon collector, `withClaudeStatus` or a new review-answering transport helper in that file); rewrites `TestItemDetailsAIReviewField`, `TestRequestAIReviewStatuses` (`ai_review_test.go`) and the Amazon 422 line of `TestPurchaseGoalStatuses` (`purchase_goal_test.go`) |
| TC-007, TC-008, TC-009, TC-010, TC-014, TC-016, TC-018 | `internal/service/amazon_review_test.go` (new; real SQLite, `fakeReviewer`, counting fake Amazon collector); rewrites the Amazon part of `TestRequestAIReviewErrorsAndUnretrievedListing` (`review_test.go`) and the Amazon goal of `TestAddKeepsPurchaseGoalForLeboncoinOnly` / `TestSetPurchaseGoalWithoutTokenAndErrors` (`purchase_goal_test.go`) |
| TC-013 | `internal/service/amazon_review_test.go` (fake collector returning `CustomerRating`) and `internal/store/amazon_rating_test.go` (new: `RecordCollection` keeps/clears the rating) |
| TC-015 | `internal/store/amazon_rating_test.go` (database without the two columns, reopened twice) and `internal/service/amazon_review_test.go` (first review after upgrade) |
| TC-017 | existing LeBoncoin tests in `internal/service/review_test.go`, `purchase_goal_test.go`, `internal/claude/review_test.go`, `purchase_goal_test.go`, `internal/store/ai_reviews_test.go`, plus one case in `internal/service/amazon_review_test.go` |
| TC-019–TC-026, TC-028 | `tests/amazon-item-ai-review.spec.ts` (new); removes "Amazon items have no AI review section" (`tests/leboncoin-ai-review.spec.ts`) and "Amazon details have no purchase goal field" (`tests/leboncoin-purchase-goal.spec.ts`) |
| TC-027 | `tests/leboncoin-session-settings.spec.ts` (help text assertion updated) |
| TC-029 | existing `tests/leboncoin-ai-review.spec.ts`, `tests/leboncoin-purchase-goal.spec.ts` |

## Files to change

Backend: `internal/model/model.go`, `internal/amazon/collector.go`, `internal/amazon/testdata/rating-fr.html`, `rating-en.html` (new), `internal/store/store.go`, `internal/store/amazon_rating.go` (new), `internal/store/ai_reviews.go`, `internal/claude/review.go`, `internal/claude/amazon_review.go` (new), `internal/service/settings.go`, `internal/service/review.go`, `internal/service/amazon_review.go` (new), `internal/service/purchase_goal.go`, `internal/service/service.go`. `internal/httpapi` handlers need no change (generic service error mapping).

Frontend: `src/types.ts`, `src/components/AmazonAiReview.tsx` (new), `src/components/ItemDetail.tsx`, `src/components/AddItemModal.tsx`, `src/components/SettingsModal.tsx`.

Docs: `API_SPECIFICATION.md` (this stage), README only if it states that reviews or goals are LeBoncoin-only (it currently does not).

No dependency, configuration, CI, script or deployment change; no manual follow-up.

## Refactoring

- R-1 (perform): `purchaseGoalBlock` helper shared by both prompts. Reason: the Amazon prompt needs the same delimited goal. Scope: `internal/claude/review.go`. Risk: low; LeBoncoin prompt byte-identical, covered by existing tests.
- R-2 (perform): `claudeClient.ReviewAmazon` and fakes. Reason: the service must call the new Claude method and tests replace it. Scope: `settings.go` and test fakes. Risk: low.
- R-3 (perform): platform-aware decoding in `AIReviews` without signature change. Reason: two review shapes in one table. Scope: `ai_reviews.go`. Risk: low; LeBoncoin store tests unchanged.
- Declined: generalizing `AiReview.tsx` for both platforms (would risk LeBoncoin behaviour, AIR-019); a separate Amazon review table (would duplicate pending/failed/restart handling).

## Risks

- Amazon rating markup may vary; unreadable values are recorded as absent and never fail a check.
- Claude may ignore the schema: failed review with retry, as for LeBoncoin.
- The product title comes from Amazon and is delimited as untrusted data.

## Unresolved questions

None.
