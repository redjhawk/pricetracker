# LeBoncoin AI review

Stage: implementation (next)
State: ready — functional and technical specifications and the API contract are complete.

User request (GitHub issue #4, "IA review of LeBoncoin items"):

> i'd like to have the ia review of each leboncoin item added. Review can be done by claude by using a login token. Claude token token can be added on the settings modal. SAme place where I can set the leboncoin token. For each new item added, a review must be done by the ia: caracteristics, price, photos.... All information must be sent to the ia. I expect to have a review on: price (is it interesting ? is it too expensive ? ); Condition (check on the photos). And other things you consider interesting. Review is shown on the details page of the leboncoin item. A IA review refresh button must be available on the detials page of the item.

## Subjects

1. Claude token settings — [functional](../../specifications/claude-token-settings/functional.md) FR-CLT-SET-001–011; [technical](../../specifications/claude-token-settings/technical.md) TS-CLT-SET-001–007; ready.
2. LeBoncoin AI review — [functional](../../specifications/leboncoin-ai-review/functional.md) FR-LBC-AIR-001–014; [technical](../../specifications/leboncoin-ai-review/technical.md) TS-LBC-AIR-001–008; ready.

## Stage results

1. Functional specification — ready.
2. Technical specification — ready (2026-10-04).
3. API contract — ready (2026-10-04): [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) section "Claude token settings and AI reviews (2026-10-04)".
4. Implementation — next.
5–8. Not started.

## API contract change and rationale

Additive; existing contracts unchanged.

- `GET /api/v1/settings/claude-token`: the modal needs the saved token in full (D-2) and the rejected-token warning time (FR-CLT-SET-009). A separate resource keeps the existing LeBoncoin session endpoint untouched.
- `PUT /api/v1/settings`: one Save must apply both entries or neither (FR-CLT-SET-006, FR-CLT-SET-011). Two sequential PUTs could leave one entry saved when the other fails (e.g. Claude verification after a stored session), so the server validates both, verifies the token, then writes both in one transaction. `PUT /api/v1/settings/leboncoin-session` is kept for compatibility.
- `aiReview` in `GET /api/v1/items/{id}` only: the details page already polls this endpoint; the list does not show reviews (functional exclusion), so it is not burdened.
- `POST /api/v1/items/{id}/ai-review` (202, asynchronous, `alreadyRunning`): manual refresh independent of price refresh (D-5) and idempotent while running (FR-LBC-AIR-006).
- Error codes `INVALID_CLAUDE_TOKEN` 400, `CLAUDE_TOKEN_REJECTED` 422, `CLAUDE_UNREACHABLE` 502, `CLAUDE_TOKEN_MISSING` 409, `AI_REVIEW_UNSUPPORTED` 422 let the UI attach errors to the right entry/state.

## Technical decisions (summary)

- Claude subscription OAuth token via plain `net/http` to `POST https://api.anthropic.com/v1/messages` (`anthropic-beta: oauth-2025-04-20`, system preamble block first), model constant `claude-sonnet-5-5`; verification = `max_tokens: 1` call.
- Listing details are not persisted: automatic reviews use the triggering collection's in-memory result; manual refresh re-fetches the listing (covers pre-existing items).
- Up to 10 photos sent as URL image blocks; structured JSON output parsed and validated server-side.
- All review attempts kept in `ai_reviews` (cascade on delete); at most one running review per item (in memory); interrupted reviews failed at startup.
- Refactoring: R-CLT-1 (extract session save into a transaction helper) perform; R-AIR-1 (`newItem` flag on the collection worker) perform; R-AIR-2 declined.

## Proposed PR split (stacked, ≤ 500 changed lines each, generated files excluded)

| PR | Content | Estimated changed lines |
| --- | --- | --- |
| 1 | Specifications, API contract, change index (docs only) | ~450 |
| 2 | Backend Claude token: `claude_token` table, R-CLT-1, `internal/claude` client + `Verify`, service `SaveSettings`, settings handlers, tests | ~480 |
| 3 | Backend listing details in the collector + `claude.Review` prompt and parser, tests | ~450 |
| 4 | Backend reviews: `ai_reviews` table, triggers (R-AIR-1), review worker, `aiReview` field and `POST …/ai-review`, tests | ~480 |
| 5 | Frontend Claude token entry in `SettingsModal` + `src/api/settings.ts`, Playwright spec | ~250 |
| 6 | Frontend AI review section (`AiReview.tsx`, `ItemDetail`, `App` polling, types/API), Playwright spec | ~450 |

If PR 1 exceeds 500 lines, split the two subjects' specifications into separate docs PRs.

## User decisions

- D-1 Token kind: Claude Pro/Max subscription token from `claude setup-token` (Q-CLT-1).
- D-2 Token shown in full like the LeBoncoin session; saving an empty field removes it (Q-CLT-3).
- D-3 Token verified with Claude on save; refused if it does not work (Q-CLT-4).
- D-4 Review language: English (Q-AIR-1).
- D-5 Re-review: refresh button always available and automatic re-review every time the recorded price changes. The user selected both options (a) and (b) of Q-AIR-2; coordinator interpretation: their union — manual refresh plus automatic re-review on each price change.
- D-6/7 Format: rating per aspect (Price: Good deal / Fair / Overpriced; Condition: Excellent / Good / Fair / Poor) with explanation, plus overall recommendation (Buy / Negotiate / Avoid) (Q-AIR-3).
- D-8 Extra aspects, all: estimated fair price range and suggested offer; scam/risk signs; missing accessories/information and questions to ask the seller; description/photo consistency (Q-AIR-4).
- D-9 No token: no review; details page says "Configure a Claude token in Settings"; refresh disabled (Q-AIR-5).
- D-10 Items added before the feature: no automatic review; refresh to get one (Q-AIR-6).
- D-11 History: all reviews kept and viewable on the details page (Q-AIR-7).
- D-12 Price opinion based on Claude's general market knowledge plus the item's recorded price history (Q-AIR-8).
- All listing information (title, price, description, attributes, all photos, location, etc.) is sent to the AI; the collector currently keeps fewer fields.
