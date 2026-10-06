# Frontend implementation: Claude token settings and LeBoncoin AI review

Stage 4 (Developer), 2026-10-04. Sources: `doc/specifications/claude-token-settings/{functional,technical}.md`, `doc/specifications/leboncoin-ai-review/{functional,technical}.md`, `API_SPECIFICATION.md` section "Claude token settings and AI reviews (2026-10-04)". JSON shapes checked against `internal/model/model.go`. No Go code changed. Not committed.

## Layer PR-A: Claude token in Settings (TS-CLT-SET-006)

| File | Change | +/- |
| --- | --- | --- |
| `src/api/settings.ts` | `ClaudeTokenSettings`, `getClaudeToken()`, `saveSettings()` (`PUT /api/v1/settings`); `saveLeboncoinSession` kept | +21/-0 |
| `src/components/SettingsModal.tsx` | Loads both entries with `Promise.all`; shared loading/load-error; Claude `TextInput` (unmasked, `autoComplete="off"`, no spellcheck) with helper text and "AI reviews are unavailable…" when no token; `lastRejectedAt` warning above the entry; save always sends `leboncoinSession`, adds `claudeToken` only when the trimmed entry differs from the loaded value; `INVALID_CLAUDE_TOKEN`/`CLAUDE_TOKEN_REJECTED`/`CLAUDE_UNREACHABLE` shown as `invalidText` on the Claude entry, cleared on edit; "Saving settings…" | +64/-10 |
| `src/index.css` | `.settings-claude-token` spacing (hunk after `.modal-notification`) | +4/-0 |
| `tests/leboncoin-session-settings.spec.ts` | Mock adds `GET /api/v1/settings/claude-token` and moves PUT to `/api/v1/settings` (records `leboncoinSession` part so existing assertions hold); textbox count 1 → 2 | +12/-5 |

Total about 116 changed lines.

## Layer PR-B: AI review on item details (TS-LBC-AIR-008)

| File | Change | +/- |
| --- | --- | --- |
| `src/types.ts` | `AiReviewContent`, `AiReview`, `AiReviewState`; `TrackedItem.aiReview?` | +30/-0 |
| `src/api/items.ts` | `aiReview` passed through `mapItem`; `requestAiReview(id)` | +6/-1 |
| `src/components/AiReview.tsx` (new) | Section "AI review": refresh button (disabled without token, while running/requesting); info "Configure a Claude token in Settings"; `aria-live` progress; request error; last-failed notification with date and message; "No AI review yet."; caption with reviewed price and stale-price warning; three text-labelled Tags with explanations; `<dl>` details; history accordion of previous reviews | +198 |
| `src/components/ItemDetail.tsx` | Renders `AiReview` for LeBoncoin items with `aiReview`, before price history | +25/-1 |
| `src/App.tsx` | `aiReviewRequesting`/`aiReviewError` (reset on item change), `handleAiReviewRefresh`, detail polling (3 s) also while `aiReview.running` | +34/-4 |
| `src/index.css` | AI review layout classes (appended at end of file) | +35/-0 |

Total about 334 changed lines. Layer B does not depend on layer A; each builds alone (the two `index.css` hunks are in different places).

## Checks executed

- `npm run build` (`tsc -b && vite build`): passed. No separate lint/typecheck script exists.
- `npx playwright test tests/`: 36 passed (includes the updated settings spec, item refresh and platform tabs).

## Review fixes (decisions.md)

- REV-002 (PR-A): `SettingsModal.save()` omits `leboncoinSession` when the session entry is unchanged and only the Claude token changed, so the saved session's revision and warnings are not reset. When nothing changed, the session is still sent (existing save behavior). `saveSettings` input makes `leboncoinSession` optional. Covered by `tests/claude-token-settings.spec.ts` ("saving a token sends only the Claude part" asserts the PUT body and that the revoked-session warning remains after reopening).
- REV-006 (PR-B): the older-price warning in `AiReview.tsx` requires both `latest.priceCents` and the current price to be known and different. Covered by "unknown reviewed price shows no older price warning".
- REV-008 (PR-B): restored spacing in `src/App.tsx` (`[refreshingItemIds, setRefreshingItemIds]`) and `src/api/items.ts` (`ItemStatus, PriceObservation`).

## Added tests

- `tests/claude-token-settings.spec.ts` (PR-A, 3 tests): no-token helper, Claude-only save body (REV-002), unchanged token not sent, rejected-token warning, server error on the Claude entry cleared by editing.
- `tests/leboncoin-ai-review.spec.ts` (PR-B, 5 tests): no-token message and disabled button, refresh with progress, polling and review/history rendering, failed attempt with previous review and older-price warning, unknown price without warning (REV-006), no section for Amazon.
- `playwright.config.ts`: both files added to `testMatch` (A and B each add their own entry).

## Checks after fixes

- `npm run build`: passed.
- `npx playwright test`: 44 passed (36 existing + 8 new).

## Limitations

- Not exercised against the real backend or Claude; screen-reader and 400 px checks left to QA.
