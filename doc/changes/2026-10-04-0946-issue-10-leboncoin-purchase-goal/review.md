# Review: LeBoncoin purchase goal

Stage: 5 (independent reviewer). Revision reviewed: HEAD `784058749bf4c899549f1e9dbb5bdabc0049ba3f` plus the uncommitted working tree (18 modified files, 8 untracked files listed in `implementation.md`).
Inputs: functional and technical specifications (ready), API_SPECIFICATION.md "LeBoncoin purchase goal (2026-10-04)", implementation record. Tests were not re-run by the reviewer; developer evidence is recorded in `implementation.md`.

Overall: requirements FR-PURCHASE-GOAL-001–009 are implemented as specified. Prompt injection handling (JSON-encoded goal inside `<purchase_goal>`, block omitted when empty) matches TS-005/006. The rerun flag is set and cleared under `s.mu`, and the developer's re-check closes the race where the running review ends between `startReview` and setting the flag. No critical findings.

## Findings

### REV-001 Stale rerun flag when a reservation is released without running (low)

- Requirement: TS-PURCHASE-GOAL-004 ("rerun flag must not loop"; set only by a goal change and cleared before restart).
- Location: `internal/service/review.go:136` (`launchReview` failure) and `internal/service/service.go:295` call `releaseReview`, which deletes `reviewing[id]` but not `reviewAgain[id]`; `internal/service/purchase_goal.go` sets `reviewAgain[id]` whenever `reviewing[id]` is true.
- Evidence: `reviewing[id]` is true from reservation onward, before launch. If `SetPurchaseGoal` sets the flag while another caller holds a reservation that then fails to launch (e.g. `StartAIReview` storage error) or is released at service.go:295, the flag remains.
- Expected/actual: expected the flag to be consumed or cleared with the reservation; actually it survives and causes one extra, unrequested review after the next unrelated review of that item finishes. In that same case the goal-change review is also never run.
- Impact: rare (requires a concurrent reservation failure); one unexpected Claude call, or a missed review for the new goal.
- Suggested action: clear `reviewAgain` in `releaseReview` too (or have it honour the flag the same way as `releaseReviewAndRerun`).

### REV-002 Goal saved but request reports 500 when the review start fails (low)

- Requirement: functional "Save failure: ... the stored goal and reviews are unchanged"; FR-PURCHASE-GOAL-009.
- Location: `internal/service/purchase_goal.go` (`startReview` error after `store.SetPurchaseGoal`).
- Evidence: the goal is written, then a `startReview` storage error returns `goal, true, false, err`; the handler maps it to 500. Frontend shows an error, keeps the draft dirty, and does not reload, while the stored goal has changed. Recorded by the developer as a limitation.
- Impact: the UI says the save failed although it succeeded; a retry returns `changed: false` and starts no review. Storage-failure edge case only.
- Suggested action: either return 200 with `reviewStarted: false` and log the review error, or accept and document the limitation.

### REV-003 Unlisted file change: `src/index.css` (informational)

- Requirement: technical specification "Allowed files" (scope).
- Location: `src/index.css` (`.purchase-goal`, `.purchase-goal-actions`).
- Evidence: not in the technical file list; disclosed in `implementation.md`. Two layout rules using Carbon spacing tokens, mirroring `.ai-review`.
- Impact: none functionally; scope deviation only. `playwright.config.ts` and existing spec fixture updates are likewise outside the list but necessary consequences (new spec registration, now-required `purchaseGoal` field).
- Suggested action: accept and record in the adjudication.

### REV-004 Error notification persists after the operator edits the draft (low, preference)

- Requirement: FR-PURCHASE-GOAL-009 (save result announced).
- Location: `src/components/PurchaseGoal.tsx` `onChange` clears `success` but not `error`.
- Impact: a previous error remains visible while the operator corrects the text; harmless, cleared on next save.
- Suggested action: optional; clear `error` on edit for consistency.

## Checked without findings

- API: PUT route, 200 body, 400 (invalid JSON, missing/non-string field, trailing value), 404, 405 with `Allow: PUT`, 413, 422; POST optional `purchaseGoal`, 1 MiB cap; `purchaseGoal` always present, `""` for Amazon.
- Migration: idempotent `ensureItemColumn` with `DEFAULT ''`.
- Unchanged goal: no write, no review (`changed: false`).
- No token: goal saved, `reviewStarted: false`.
- Review reads the goal at worker time, so the newest goal is used.
- Accessibility: labelled `TextArea`, native Carbon `Button`, `InlineLoading`, Carbon success notification (`role=status`), error keeps draft; field only for LeBoncoin; no goal in history.
- Shutdown: rerun is started before `workers.Done()` of the finishing worker, so `workers.Add` does not race with a completed `Wait`.

## Re-review of fixes (REV-001, REV-002)

Revision: HEAD `784058749bf4c899549f1e9dbb5bdabc0049ba3f` plus the uncommitted working tree after the developer's fixes. Reviewer ran `go test -race ./internal/service/`: pass.

- REV-001: verified. `releaseReview` (`internal/service/review.go`) is now the single release path: under `s.mu` it reads and deletes `reviewAgain[id]` with `reviewing[id]`, then starts one review if flagged. It covers the `launchReview` failure path, the `service.go` collection-save failure path, and `runReview`. Because the flag is deleted before the restart, the restart cannot loop. Test: `TestReleasedReservationRunsRequestedReview`.
- REV-002: verified. Both `startReview` error sites in `SetPurchaseGoal` now log the error and return `goal, true, false, nil`, so the API returns 200 with `reviewStarted: false`. Test: `TestReviewStartFailureAfterSaveIsNotAnError`. The limitation has been removed from `implementation.md`.
- New findings: none.
