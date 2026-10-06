# Implementation: LeBoncoin purchase goal

Stage: 4 (developer). Specifications: [functional](../../specifications/leboncoin-purchase-goal/functional.md), [technical](../../specifications/leboncoin-purchase-goal/technical.md), API "LeBoncoin purchase goal (2026-10-04)". Not committed.

## Requirement to change mapping

| Requirement | Change |
| --- | --- |
| FR-PURCHASE-GOAL-001 / TS-001 | `AddItemModal` `TextArea` "Purchase goal (optional)" (cleared on close/success, disabled while submitting); `App.handleAdd` and `api.addItem(url, purchaseGoal)` send `{ url, purchaseGoal }`; `POST /api/v1/items` decodes `purchaseGoal`, body limit 1 MiB; `Service.Add(ctx, url, goal)` stores the trimmed goal for LeBoncoin only; `Store.Insert` writes `purchase_goal`. |
| FR-002 / TS-002 | `model.Item.PurchaseGoal` (`purchaseGoal`, `""` when none), selected in `store.get` (list, add, details); `PurchaseGoal.tsx` (labelled `TextArea`, "Save goal") rendered in `ItemDetail` above the AI review for LeBoncoin only. |
| FR-003 / TS-003 | `PUT /api/v1/items/{id}/purchase-goal` (`httpapi/purchase_goal.go`); `Service.SetPurchaseGoal` trims, returns `changed: false` without writing when equal; `Store.SetPurchaseGoal` replaces the value. |
| FR-004 / TS-004 | On change, `startReview(id, nil, true)`; if a review is running, `reviewAgain[id]` is set under `s.mu` (re-checked atomically; if the review ended meanwhile a new one is started); `releaseReview` (used by every release path, see REV-001) clears `reviewing` and `reviewAgain` under one lock and starts one more review when flagged. No token: saved, `reviewStarted: false`. Frontend reloads details after save; existing polling shows the running review. |
| FR-005 / TS-005 | `reviewListing` passes `listing.PurchaseGoal` (read when the worker runs) as `claude.ReviewInput.PurchaseGoal`; `reviewPrompt` adds the instruction line and a JSON-encoded `<purchase_goal>` block after `</listing_data>`. Schema and parsing unchanged. |
| FR-006 / TS-006 | Service trims; empty/whitespace goal omits the block, leaving the prompt identical to before. |
| FR-007 / TS-007 | No length validation; `TEXT` column; 1 MiB transport cap (`maxGoalBodyBytes`). |
| FR-008 / TS-008 | Field only for `platform === "leboncoin"`; PUT on Amazon → `422 PURCHASE_GOAL_UNSUPPORTED`; review objects unchanged. |
| FR-009 / TS-009 | Visible label, native Carbon `Button`, `InlineLoading` while saving, success/error `InlineNotification`; draft kept on error; save disabled when unchanged. |

Migration: `{"purchase_goal", "TEXT NOT NULL DEFAULT ''"}` added to the existing idempotent `ensureItemColumn` list; existing rows get `''`.

## Files

Backend: `internal/model/model.go` (field additions; gofmt realigned the `Listing` struct), `internal/store/store.go`, `internal/service/service.go`, `internal/service/review.go`, `internal/service/purchase_goal.go` (new), `internal/claude/review.go`, `internal/httpapi/server.go`, `internal/httpapi/purchase_goal.go` (new).
Frontend: `src/types.ts`, `src/api/items.ts`, `src/App.tsx`, `src/components/AddItemModal.tsx`, `src/components/ItemDetail.tsx`, `src/components/PurchaseGoal.tsx` (new), `src/index.css` (two small layout rules with Carbon spacing tokens; not listed in the technical file but needed for spacing of the new section).
Tests: `internal/claude/purchase_goal_test.go`, `internal/service/purchase_goal_test.go`, `internal/httpapi/purchase_goal_test.go`, `tests/leboncoin-purchase-goal.spec.ts` (new); `internal/service/review_test.go` (new `Add` parameter); `playwright.config.ts` (register new spec); existing Playwright fixtures `tests/leboncoin-ai-review.spec.ts`, `tests/leboncoin-session-settings.spec.ts`, `tests/platform-tabs.spec.ts` gain `purchaseGoal: ""` (now a required contract field) and the add-request assertion in `platform-tabs.spec.ts` expects `purchaseGoal: ""`.

No refactoring performed.

## Checks executed

- `go vet ./...` — pass.
- `go test -race ./...` and `go test ./...` — pass (claude, httpapi, leboncoin, service, store).
- `npm run build` — pass (tsc + Vite).
- `npx playwright test` (all 48 tests, including 4 new) — final runs: 47 passed, 1 failed. The failing test differed between runs and is unrelated to this change: `leboncoin-session-settings.spec.ts:124` (header menu) failed once, then passed 3/3 with `--repeat-each=3`; it passed in the preceding full run. New spec passed in every run.

## Review fixes (decisions.md)

- REV-001: `releaseReview` (`internal/service/review.go`) now holds the former `releaseReviewAndRerun` logic: under one lock it deletes `reviewing[id]` and `reviewAgain[id]`, and if the flag was set it calls `startReview(id, nil, true)` (errors logged). All release paths (`runReview` defer, `launchReview` failure, `collectReserved` failure) therefore honour and clear the flag; `releaseReviewAndRerun` was removed. Test: `TestReleasedReservationRunsRequestedReview` (reserve, save goal, release without running → flag cleared, one review with the goal).
- REV-002: `SetPurchaseGoal` (`internal/service/purchase_goal.go`) logs a `startReview` error after the goal is saved and returns `goal, true, false, nil` (200, `reviewStarted: false`) at both call sites. Test: `TestReviewStartFailureAfterSaveIsNotAnError` (SQLite trigger makes `ai_reviews` inserts fail; no error, `reviewStarted == false`, goal stored).
- Checks after fixes: `go vet ./...` pass; `go test ./...` pass; `go test -race -count=3 ./internal/service/` pass.

## Limitations

- Not verified against the real Claude API or LeBoncoin; prompt content is verified by unit test only.
- The intermittent header-menu Playwright test is pre-existing flakiness, not investigated.
