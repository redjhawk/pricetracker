# Implementation (issue #111)

Stage: development done (next: review)
Developer: expert implementation agent. Sources: [removal technical](../../specifications/amazon-searches-removal/technical.md), [request stop technical](../../specifications/amazon-request-stop/technical.md), TS-CLT-SET-009, [API_SPECIFICATION.md](../../../API_SPECIFICATION.md).

## Requirement → change

| Technical ID | Change |
| --- | --- |
| TS-AMZ-SRCH-DEL-001 | Deleted `AmazonSearchesPage.tsx`, `AmazonSearchItems.tsx`; `/searches/…` route removed (an old URL shows the main page, Amazon tab); two tabs in `TrackedItemsPage.tsx`; search types and `tracked` removed from `types.ts`, `api/items.ts` |
| TS-AMZ-SRCH-DEL-002 | Deleted `httpapi/searches.go`, `service/searches.go`, `service/search_worker.go`, `service/search_window.go`, `amazon/search.go`, `model/searches.go` and their tests; worker start removed from `main.go`; routes now `404 ROUTE_NOT_FOUND` (`httpapi/removed_routes_test.go`) |
| TS-AMZ-SRCH-DEL-003 | Deleted `store/searches.go`; `removeAmazonSearches` in `store/store.go` (one transaction, only when `amazon_search_items` exists); `store/search_removal_test.go` |
| TS-AMZ-SRCH-DEL-004 | `Store.Delete` deletes again; `Add` no longer reuses search items; `ITEM_NOT_TRACKED` refusal removed |
| TS-AMZ-SRCH-DEL-005 | Deleted `service/amazon_review.go`, `claude/amazon_review.go`(+tests), `AmazonAiReview.tsx`; `ReviewAmazon` removed from the Claude interface and fakes; `Service.AIReview` returns `nil` for Amazon; `store.AIReviews` decodes LeBoncoin reviews only (platform parameter dropped); `productDetails`/`CollectProduct` and `model.ProductDetails` removed; the migration deletes Amazon `ai_reviews` rows |
| TS-AMZ-STOP-001/002 | Failure log line kept in `amazonGate.do`; global count kept in `amazon_request_state.consecutive_failures` |
| TS-AMZ-STOP-003 | Column `items.amazon_stopped_at`; `StopAmazonItems` at the 5th failure, count reset, stop logged |
| TS-AMZ-STOP-004 | `DueIDs` excludes stopped items; the gate skips a stopped item without request or attempt (Refresh all prices, schedule, running batch) |
| TS-AMZ-STOP-005 | Amazon tab warning from `amazonStoppedAt` (most recent stop); details notice; `src/api/searches.ts` deleted |
| TS-AMZ-STOP-006 | `manual` threaded through `runQueuedCollection`/`collectReserved`; answer clears only that item; failure keeps the stop and is counted |
| TS-AMZ-STOP-007 | `convertGlobalAmazonStop` in `store/amazon_requests.go` (one transaction, idempotent) |
| TS-AMZ-STOP-008 | `GET /api/v1/amazon/requests`, `model.AmazonRequests`, `Service.AmazonRequests`, `409 AMAZON_REQUESTS_STOPPED` removed; `amazonStoppedAt` on items |
| TS-CLT-SET-009 | `claudeHelper` text in `SettingsModal.tsx` |

## Deviations and recorded decisions

- `removeAmazonSearches` runs in `store.Open` right after `createAIReviewSchema` instead of inside `createSchema`, so `ai_reviews` exists when its Amazon rows are deleted. Same effect, startup order only.
- The Playwright scenario `tests/amazon-searches.spec.ts` (and its entry in `playwright.config.ts`) was deleted with the feature; `platform-tabs.spec.ts` keyboard test now ends on the LeBoncoin tab; dead `/api/v1/amazon/requests` mocks removed from three specs.
- No HTTP test sends `POST /items/{id}/refresh` on a stopped item, because it would start a real Amazon request; the acceptance (no 409) is covered by the service test `TestRefreshPriceRestartsOnlyThatItem`.

## Parts (stacked, each built and tested on its own)

| # | Branch | Base | Changed lines (`scripts/pr-size.sh <base>`) |
| --- | --- | --- | --- |
| 1 | `ai-dev/issue-111-docs` | `ai-dev/issue-111-feature` (= `origin/master` 41905bd) | 331 |
| 2 | `ai-dev/issue-111-frontend-search-items` | part 1 | 279 |
| 3 | `ai-dev/issue-111-frontend-tab` | part 2 | 414 |
| 4 | `ai-dev/issue-111-backend-http` | part 3 | 291 |
| 5 | `ai-dev/issue-111-service-searches` | part 4 | 400 |
| 6 | `ai-dev/issue-111-search-worker` | part 5 | 483 |
| 7 | `ai-dev/issue-111-amazon-review` | part 6 | 275 |
| 8 | `ai-dev/issue-111-amazon-search-model` | part 7 | 416 |
| 9 | `ai-dev/issue-111-store-searches` | part 8 | 486 |
| 10 | `ai-dev/issue-111-search-migration` | part 9 | 118 |
| 11 | `ai-dev/issue-111-item-stop-backend` | part 10 | 449 |
| 12 | `ai-dev/issue-111-item-stop-frontend` | part 11 | see commit-step |

Re-slicing against the plan: `service/amazon_review_test.go` moved to part 5 (it depends on the worker test fakes); `amazon/search.go` and `productDetails` moved to part 8 and the `tracked` field to part 9 (their tests depend on each other).

## Checks executed

Every part: `go vet ./...` (no output), `go test ./...` (`amazon`, `claude`, `httpapi`, `leboncoin`, `service`, `store`: ok), `npm run build` (`tsc -b && vite build`: built). Playwright could not run: the Chromium headless shell fails to start (`libnspr4.so: cannot open shared object file`); QA must run the specs in an environment with the browser libraries.
