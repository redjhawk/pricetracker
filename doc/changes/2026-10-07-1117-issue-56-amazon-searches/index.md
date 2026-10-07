# Amazon searches with AI review (issue #56)

Stage: implementation (next)
State: functional, technical and API specifications ready; no open functional question.

User request (GitHub issue #56, redjhawk):

> Black Friday are approaching and it is difficult to review all interesting material. I'd need this application to do it for me.
> Application should be able to receive, in a new tab, a filter coming from amazon (ex: https://www.amazon.fr/joursprime/?_encoding=UTF8&...&discounts-widget=...&promotionsSearchLastSeenAsin=B0HBWZ42P9&promotionsSearchStartIndex=0&promotionsSearchPageSize=60) This returns a list of elements.
> You can do this on a new tab and you can keep track on each research. For each research, you also keep track of the first 30 items (so you keep the link and the title and the price). Each item information is shown as like the others items. Price and history of price.
> And for each item, you perform an AI analysis using claude and the token the user has added to do review on leboncoin. So you also have to change the description of the text to say that it will be used to do amazon review.
> Review of the item must consider price.
> You should simulate human behavior (so take your time to click on an item and review all data, and then, close the link). This order IS REALLY IMPORTANT. There's no hurry.
> Request must be done between 10PM and 01.00AM paris time. And from 6AM to 8AM.
> If more than 5 request start failing, stop requesting Amazon and tell the user. IF there is this problem, log it. Log the error return from the server.

## Scope

New Amazon searches tab with frozen first-30 items, price tracking and history, sharing with tracked Amazon items, move to tracked list, deletion; human-like request pacing within time windows with a global Amazon stop after failures; Claude AI review of search items; Claude token help text update. Cross-tier (frontend, backend, API).

## User functional decisions (redjhawk)

- D-1: "Search items is another level on the interface: an "Amazon searchs" tab. Inside, the list of added searches. Clicking a search shows its first 30 items."
- D-2: Each item keeps link, title and price and is shown like other items: price and price history.
- D-3: "Those 30 items are not dynamic: they are the first 30 items first got. Refresh is done like for the other amazon items."
- D-4: "Always the 30 first elements, no parameter, no other filter."
- D-5: "If an item drops, show it as "unreachable", as done for amazon/leboncoin items."
- D-6: "If the same item appears in multiple searches, the item is shared. Same if it is added on the amazon item-by-item tab."
- D-7: "User can move an element from the search list to the list of tracked amazon elements. No need for the other way around. A moved item stays visible on the search too."
- D-8: "Once added, can only delete it (and so all related items, unless the item has been moved to the amazon tracked list). No rename, no edit."
- D-9: "What to judge: the price regarding characteristics. No goal for now."
- D-10: "Store detected price and keep a history (as other amazon items)."
- D-11: "When to review: as soon as information is got from Amazon. Act as a human: do the search, take time to open an item link, take time to review, go to next item."
- D-12: "Token description: say it will be used for amazon and leboncoin reviews."
- D-13: "Review language and format: English."
- D-14: "Review again: when price changed, just inform that the price has changed and the last AI review was done with the older price (indicate that price)."
- D-15: "Users: searches belong to each user; also usable in open mode (as other items)."
- D-16: "Time window only applies to items added as search items (10PM-01AM and 6AM-8AM Paris time)."
- D-17: "Pauses between 30 seconds and 2 minutes, random." / "5 failures then stop ALL requests to Amazon, even those for individual tracked elements."
- D-18: "If there's a success, continue; failures are kept to be tested again once all the others have been done."
- D-19: "After a stop: show a refresh button by a search item on the list of searches. Inform with an information line below the search item; it disappears if a refresh fixes the problem."
- D-20: "Restart (refresh after stop): starts checking the items again, review included; price history must not be lost."
- D-21: "Tracked amazon tabs: agent's choice" — decision: keep the existing tracked Amazon tab unchanged.

Recorded interpretation: the 5-failure limit counts consecutive failed Amazon requests; a success resets the count (D-18).

## Subjects

1. Amazon searches — [functional](../../specifications/amazon-searches/functional.md) FR-AMZ-SEARCH-001–016; ready.
2. Amazon human browsing — [functional](../../specifications/amazon-human-browsing/functional.md) FR-AMZ-HUMAN-001–012; ready.
3. Amazon AI review — [functional](../../specifications/amazon-ai-review/functional.md) FR-AMZ-AIR-001–010; ready.
4. Claude token settings (amendment) — [functional](../../specifications/claude-token-settings/functional.md) FR-CLT-SET-012; ready.

Global summary: [FUNCTIONAL_SPECIFICATIONS.md](../../FUNCTIONAL_SPECIFICATIONS.md) §5.5, §5.6b, FR-32–FR-34, §15.

## Stage results

1. Functional specification — ready (2026-10-07).
2. Technical specification — ready (2026-10-07): [Amazon searches](../../specifications/amazon-searches/technical.md) (TS-AMZ-SEARCH-001–009), [Amazon human browsing](../../specifications/amazon-human-browsing/technical.md) (TS-AMZ-HUMAN-001–007), [Amazon AI review](../../specifications/amazon-ai-review/technical.md) (TS-AMZ-AIR-001–006), [Claude token settings](../../specifications/claude-token-settings/technical.md) (TS-CLT-SET-008 added).
3. API contract — ready (2026-10-07): [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) section "Amazon searches (2026-10-07)".
4–10. Not started.

## Technical decisions

- Search items are rows of the existing `items` table with a new `tracked` flag; `UNIQUE(owner_id, canonical_url)` gives sharing between searches and the tracked list for free (D-6). New tables `amazon_searches`, `amazon_search_items`, `amazon_request_state`.
- One global `amazonGate` (mutex held for each Amazon request) serializes all Amazon traffic, so the 5th consecutive failure and the stop are one atomic step that also blocks tracked-item checks (D-17). Stop state persists in SQLite.
- One search worker goroutine, Europe/Paris windows, random 30–120 s pauses through an injectable clock/sleeper; one pass per search per window; failed actions retried once at the end of the pass (D-18), not unboundedly.
- Amazon reviews reuse `ai_reviews`, reservations and failure handling of the LeBoncoin review; a separate small Claude prompt returns only a price rating and explanation.

## Refactoring decisions

| ID | Decision | Reason | Scope | Part |
| --- | --- | --- | --- | --- |
| R-1 | Perform | Reuse the items table: list/IDs/due/duplicate queries filter `tracked = 1`; tracked delete untracks an item still in a search | `internal/store/store.go` | 3 |
| R-2 | Perform | All Amazon requests must stop together: tracked Amazon collection goes through the gate | `internal/service/service.go` | 5 |
| R-3 | Perform | `AIReview.Review` holds either review shape; decoded by platform; LeBoncoin JSON unchanged | `internal/model`, `internal/store/ai_reviews.go` | 7 |
| R-4 | Perform | Extract the finish step of `runReview` to share it with Amazon reviews | `internal/service/review.go` | 7 |
| — | Decline | Generic scheduler abstraction or a second item table: not needed | — | — |

## API contract changes (stage 3) and rationale

- New: `GET/POST /api/v1/amazon-searches`, `GET/DELETE /api/v1/amazon-searches/{id}`, `POST /api/v1/amazon-searches/{id}/items/{itemId}/track`, `POST /api/v1/amazon-searches/{id}/refresh`, `GET /api/v1/amazon/requests` — FR-AMZ-SEARCH-002–014, FR-AMZ-HUMAN-009–011.
- Item responses add `tracked` (search items are untracked items); list and refresh-all keep their meaning by covering tracked items only — D-6, D-7, D-21.
- `POST /api/v1/items` on a product already in a search makes it tracked instead of `409` — D-6 ("same if it is added on the amazon item-by-item tab").
- `DELETE /api/v1/items/{id}` untracks an item still in a search — FR-AMZ-SEARCH-012 corner case.
- `POST /api/v1/items/{id}/refresh` adds `409 ITEM_NOT_TRACKED` (windows, D-16) and `409 AMAZON_REQUESTS_STOPPED` (stop of all Amazon requests, D-17, FR-AMZ-HUMAN-009).
- `aiReview` becomes non-null for Amazon search items with an Amazon review shape (`price` only); `aiReviewSummary` in search items — FR-AMZ-AIR-003–005. LeBoncoin contracts unchanged.

## Delivery split plan

Stacked on `ai-dev/issue-56-feature` (created from `master`); estimates count docs, code and tests. Each part builds and passes `go vet ./...`, `go test ./...` and `npm run build` alone; no part exposes a half-working UI (the tab arrives in part 9).

| Part | Content | Main files | Estimate |
| --- | --- | --- | --- |
| 1 | Functional specifications (existing WIP commit) | `doc/specifications/amazon-*/functional.md`, `claude-token-settings/functional.md`, `doc/FUNCTIONAL_SPECIFICATIONS.md`, this index | ~240 |
| 2 | Technical specifications, API contract, index update | `doc/specifications/*/technical.md`, `API_SPECIFICATION.md`, this index | ~430 (measured) |
| 3 | Store: `tracked` column + R-1, search tables, request-state table, search store functions, tests | `internal/store/store.go`, `internal/store/searches.go`, `internal/store/amazon_requests.go`, tests | ~450 |
| 4 | Amazon adapter: `ParseSearchURL`, `FetchSearch` parser, `CollectProduct` (no second-hand request, product details), failure diagnostics, fixtures/tests | `internal/amazon/search.go`, `internal/amazon/collector.go`, `internal/model/model.go`, tests | ~400 |
| 5 | Gate (R-2), windows, refresh `409`s, tests incl. `-race` | `internal/service/amazon_gate.go`, `search_window.go`, `service.go`, tests | ~400 |
| 6 | Search worker (pass order, pauses, resume, retry once, stop), started in `main.go`; review hook as a no-op; tests with fake clock | `internal/service/search_worker.go`, `cmd/pricefollower/main.go`, tests | ~450 |
| 7 | Amazon AI review: Claude prompt, R-3, R-4, hook implementation, tests | `internal/claude/amazon_review.go`, `internal/service/amazon_review.go`, `review.go`, `internal/store/ai_reviews.go`, tests | ~450 |
| 8 | Search use cases and HTTP handlers (list/add/get/delete/track/refresh/status), item `tracked`, ownership tests | `internal/service/searches.go`, `internal/httpapi/searches.go`, `server.go`, tests | ~480 |
| 9 | Frontend: API module, types, third tab, searches list, search items page, refresh/stop line, Playwright spec | `src/api/searches.ts`, `src/types.ts`, `src/components/AmazonSearches*.tsx`, `TrackedItemsPage.tsx`, `App.tsx`, `tests/amazon-searches.spec.ts` | ~480 |
| 10 | Frontend: Amazon AI review section, untracked item detail, Amazon tab stop notification, Settings text, Playwright | `src/components/AmazonAiReview.tsx`, `ItemDetail.tsx`, `TrackedItemsPage.tsx`, `SettingsModal.tsx`, tests | ~300 |

If a part measured with `scripts/pr-size.sh` exceeds 500 lines, split it at a file boundary (e.g. part 8: service then handlers; part 9: list page then items page) before opening the PR. Workflow evidence (review, decisions, QA, commit step) goes with the last part.
