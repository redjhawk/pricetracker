# Backend implementation: Claude token settings and LeBoncoin AI review

Stage: 4 (developer), backend only. Frontend (`src/`) is out of scope for this record. Not committed.

Sources: [Claude token technical](../../specifications/claude-token-settings/technical.md), [AI review technical](../../specifications/leboncoin-ai-review/technical.md), [API contract](../../../API_SPECIFICATION.md) section "Claude token settings and AI reviews (2026-10-04)".

## Requirement-to-change mapping

| Technical ID | Change | Files |
| --- | --- | --- |
| TS-CLT-SET-001 | `claude_token` single-row table (idempotent `CREATE TABLE IF NOT EXISTS` + `INSERT OR IGNORE`), `model.ClaudeToken` | `internal/store/store.go`, `internal/model/model.go` |
| TS-CLT-SET-002 | `internal/claude` client: `Verify` (`max_tokens: 1`, 20 s), OAuth headers, preamble first system block, status mapping 401/403 → `ErrRejected`, 429 → `ErrUsageLimit`, network/5xx → `ErrUnreachable`; logs only operation and status | `internal/claude/client.go` |
| TS-CLT-SET-003 | `Service.ClaudeToken`, `Service.SaveSettings` (session validation → token format → verification outside a transaction → one store transaction); unchanged token not verified; empty clears without verification | `internal/service/settings.go`, `internal/store/claude_token.go` |
| TS-CLT-SET-004 | `GET /api/v1/settings/claude-token` (`Allow: GET`), `PUT /api/v1/settings` (`Allow: PUT`, 32 768-byte limit, single JSON value, `INVALID_REQUEST` rules) | `internal/httpapi/server.go`, `internal/httpapi/settings.go` |
| TS-CLT-SET-005 | `MarkClaudeTokenRejected` / `ClearClaudeTokenRejected` conditional on revision; any token save clears `last_rejected_at` | `internal/store/claude_token.go`, `internal/service/review.go` |
| TS-CLT-SET-007 | Token only in the two settings responses and the `Authorization` header; tests assert it is absent from logs, errors, item list and details | tests below |
| R-CLT-1 | `SaveLeboncoinSession` body extracted into `saveLeboncoinSessionTx`, reused by `SaveSettings`; existing session tests unchanged and passing | `internal/store/store.go` |
| TS-LBC-AIR-001 | Collector fills `CollectionResult.Listing` (`model.ListingDetails`) for active verified ads (success and `price_not_found`): description (≤ 20 000 chars), attributes (≤ 100, label/value fallbacks), all `img.leboncoin.fr` URLs (`urls_large` preferred, deduplicated), location, category, publication date, seller type. Not persisted. Attributes/location/owner decoded leniently so an unexpected shape never breaks price parsing | `internal/leboncoin/collector.go`, `internal/leboncoin/details.go`, `internal/model/model.go` |
| TS-LBC-AIR-002 | `Client.Review`: `max_tokens: 4096`, system = preamble + reviewer instructions (English), one text block (listing JSON, price history oldest first, photo note, schema) + up to 10 URL image blocks | `internal/claude/review.go` |
| TS-LBC-AIR-003 | Parser: concatenated text blocks, `stop_reason == max_tokens` rejected, first `{` to last `}` (covers fences/prose), enum normalisation, validation of ratings, explanations, fair price; nil lists → empty; any failure → `ErrBadResponse`; raw text never stored/logged | `internal/claude/review.go` |
| TS-LBC-AIR-004 | `ai_reviews` table + index, FK cascade; `StartAIReview`, `FinishAIReview` (0 rows not an error), `AIReviews` (latest attempt + ≤ 50 succeeded newest first), `FailInterruptedAIReviews` at `Open`, `LatestPriceCents`, `PriceHistory` | `internal/store/ai_reviews.go`, `internal/store/store.go` |
| TS-LBC-AIR-005 / R-AIR-1 | `collectReserved(ctx, id, newItem)`; `Add` uses internal `collect(ctx, id, true)`, `Collect(ctx, id)` keeps its signature. LeBoncoin: previous price read before recording; review started for a new item's first collection (fails with "The listing could not be retrieved from LeBoncoin." if no details) or when a successful price differs from the previous recorded price | `internal/service/service.go` |
| TS-LBC-AIR-006 | `startReview` (no token → no row; one per item via `reviewing` set; token and revision captured at start), worker with fresh fetch for manual requests (`collectLeboncoin`, no price/attempt recorded), 150 s Claude timeout, failure messages from the contract, rejection marking/clearing, shutdown → interrupted message; `RequestAIReview` (404/422/409/`alreadyRunning`) | `internal/service/review.go` |
| TS-LBC-AIR-007 | `aiReview` in `GET /api/v1/items/{id}` only (`model.ItemDetails`, `null` for Amazon); `POST /api/v1/items/{id}/ai-review` → 202 `{requestedAt, alreadyRunning}`, 404, 409, 422, 405 `Allow: POST` | `internal/httpapi/server.go`, `internal/httpapi/ai_review.go`, `internal/model/model.go` |

R-AIR-2 declined as specified. No other refactoring. No new dependency, no config change, Amazon collection unchanged.

## Files by stacked PR layer

Line counts are added/removed lines (new files counted in full).

**PR 2: Claude token (≈ 860 lines, about 445 production and 415 test)**
- `internal/store/store.go` (+20/−3: `claude_token` schema, R-CLT-1)
- `internal/model/model.go` (+8: `ClaudeToken`)
- `internal/store/claude_token.go` (+91), `internal/store/claude_token_test.go` (+96)
- `internal/claude/client.go` (+116), `internal/claude/client_test.go` (+93)
- `internal/service/service.go` (+4/−2: `claude` field, import, `New`)
- `internal/service/settings.go` (+104), `internal/service/settings_test.go` (+108)
- `internal/httpapi/server.go` (+8: routes), `internal/httpapi/settings.go` (+91), `internal/httpapi/claude_settings_test.go` (+117)

**PR 3: listing details + review prompt/parser (≈ 595 lines)**
- `internal/model/model.go` (+≈55: `CollectionResult.Listing`, `ListingDetails`, `ListingAttribute`, `AIReviewContent` and parts)
- `internal/leboncoin/collector.go` (+10), `internal/leboncoin/details.go` (+104), `internal/leboncoin/details_test.go` (+58)
- `internal/claude/review.go` (+233), `internal/claude/review_test.go` (+135)

**PR 4: reviews (≈ 805 lines)**
- `internal/model/model.go` (+≈27: `AIReview`, `AIReviewState`, `ItemDetails`)
- `internal/store/store.go` (+8: schema and interrupted-review cleanup in `Open`), `internal/store/ai_reviews.go` (+155), `internal/store/ai_reviews_test.go` (+88)
- `internal/service/service.go` (+≈26/−5: `reviewing`, R-AIR-1, triggers), `internal/service/settings.go` (+1: `Review` in `claudeClient`), `internal/service/review.go` (+174), `internal/service/review_test.go` (+206)
- `internal/httpapi/server.go` (+17/−1), `internal/httpapi/ai_review.go` (+25), `internal/httpapi/ai_review_test.go` (+77)

### Size limitation and proposed sub-split

With tests, each layer goes over the ≈ 480-line target. The files are separated so each layer can be split again into stacked PRs that build and pass tests on their own:

| PR | Content | ≈ lines |
| --- | --- | --- |
| 2a | `claude_token` schema + R-CLT-1 + `ClaudeToken` model + store token file and test + `internal/claude/client.go` and test | 425 |
| 2b | Service `SaveSettings` (+ `service.go` wiring) + handlers + their tests | 435 |
| 3a | `ListingDetails` model + collector + `details.go` and test | 230 |
| 3b | `AIReviewContent` model + `review.go` and test | 400 |
| 4a | `AIReview` model + `ai_reviews` store, `Open` hook, store test | 280 |
| 4b | `AIReviewState` model + service worker/triggers + `claudeClient.Review` + service test | 420 |
| 4c | `ItemDetails` model + HTTP details field and POST route + handler test | 125 |

`model.go`, `store.go`, `server.go` and `service.go` hunks must be staged per layer (for example with `git apply --cached` of per-layer patches), because interactive staging is unavailable.

## Verification (actually run)

- `go build ./...`: success.
- `go vet ./...`: no findings.
- `go test -count=1 -race ./...`: `internal/claude`, `internal/httpapi`, `internal/leboncoin`, `internal/service`, `internal/store` all `ok`. Packages without tests: `cmd/pricefollower`, `config`, `internal/amazon`, `internal/model`, `web`.
- `gofmt -l` clean on the changed Go files.
- Tests never call the real Anthropic API. Client tests use `httptest` with the unexported `url` override. Handler tests replace `http.DefaultTransport`, which the Claude client uses. Service tests use fake Claude and LeBoncoin collectors.
- Per-layer build and test isolation was not run. Each layer only adds files and hunks used by later layers. The one compile dependency is that 4b adds `Review` to `claudeClient` together with `fakeClaude.Review` in `review_test.go`.

## Accepted review fixes (Go only, per [decisions.md](decisions.md))

| Finding | Fix | Files | Test |
| --- | --- | --- | --- |
| REV-001 | `collectReserved` reserves `reviewing[id]` (via `reserveReview`) **before** `RecordCollection` when the item is LeBoncoin, a token is saved and (new item or price changed); after recording, `launchReview` inserts the pending row and starts the worker; a failed record or failed `StartAIReview` releases the reservation | `internal/service/service.go`, `internal/service/review.go` | `TestReviewRunningWhenCollectionIsRecorded`: a concurrent reader never sees the changed price stored while `running` is false |
| REV-003 | `startReview` returns `reviewStarted` / `reviewAlreadyRunning` / `reviewNoToken`; the running check and token check are both decided under `s.mu` in `reserveReview`; `RequestAIReview` maps the outcome directly (no second `reviewRunning` check) | `internal/service/review.go` | `TestStartReviewOutcomes`; existing request tests still pass |
| REV-007 | System instructions state that the content between `<listing_data>` and `</listing_data>` is untrusted seller text, to be treated as data only. Attempts to give instructions are reported as a risk sign. The listing JSON is wrapped in these delimiters, and JSON HTML escaping stops seller text from closing them | `internal/claude/review.go` | `TestReviewPromptWithoutPhotos` asserts single delimiters with a hostile description and the untrusted instruction |

Layer placement: REV-001 and REV-003 belong to PR 4 (4b); REV-007 belongs to PR 3 (3b). Added lines: ~+60 in `review.go` (service), ~+20 in `service.go`, ~+70 test lines, ~+5 in `claude/review.go`, ~+8 in `claude/review_test.go`.

Re-verification after the fixes: `go build ./...` ok, `go vet ./...` clean, `go test -count=1 -race ./...` all packages `ok`, `gofmt -l` clean.

REV-009: `reserveReview` now reads the Claude token before taking `s.mu`. A missing token or a read error returns `reviewNoToken` (with the error); only the `reviewing[id]` check-and-set is under the lock. Outcome mapping is unchanged. Re-verified with `go build`, `go vet` and `go test -count=1 -race ./...`, all `ok`.

## Limitations and notes

- No test calls the real Anthropic API. The OAuth beta header, the required preamble and the `claude-sonnet-5-5` model are constants in `internal/claude/client.go` and have not been checked against live Anthropic.
- `Verify` treats a 2xx response with a body that cannot be decoded as an error, so the save returns `CLAUDE_UNREACHABLE`. The spec says "2xx → nil". This is unlikely in practice.
- If reading the price history fails, the review continues without history and the failure is logged. The contract has no failure message for this case.
- The final review write uses a fresh 5 s context, so an outcome caused by shutdown is still stored before `Service.Close` returns.
- Frontend work (PR 5/6) and Playwright specs are not part of this record.
