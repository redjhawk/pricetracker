# Review decisions: leboncoin-purchase-goal

Reviewed specification revisions: `doc/specifications/leboncoin-purchase-goal/functional.md`, `technical.md`, API_SPECIFICATION.md "LeBoncoin purchase goal (2026-10-04)" at HEAD `7840587`
Reviewed code revision: HEAD `7840587` plus uncommitted working tree (snapshot 2026-10-04)
Reviewer: stage 5 independent reviewer (`review.md`)
Adjudicator: stage 6 independent adjudicator (separate agent; did not write or review the code)

## Findings and decisions

### REV-001: Stale rerun flag when a reservation is released without running

- Evidence: `internal/service/review.go:110-114` `releaseReview` deletes `reviewing[id]` only; it is called from `launchReview` on `StartAIReview` failure (`review.go:136`) and from `collect` on `RecordCollection` failure (`service.go:295`). `SetPurchaseGoal` (`purchase_goal.go`) sets `reviewAgain[id]` whenever `reviewing[id]` is true, which includes the reserved-but-not-launched window. Confirmed by code reading.
- Impact and scenario: a goal is saved while a collection-triggered reservation exists; that reservation is then released on a storage error. The goal-change review never runs (violating the functional rule that a changed goal requests a new review), and the leftover flag later causes one unrequested review after the next unrelated review finishes.
- Criticality: non-critical, because it requires a concurrent storage failure during a short window, affects only one item, causes at most one extra Claude call or one missed review, and corrupts no data. The operator can recover by re-saving or requesting a review.
- Disposition: fix.
- Reason: the fix is a few lines, removes a real spec deviation (TS-PURCHASE-GOAL-004 flag lifecycle; changed goal must trigger a review), and only clearing the flag would still leave the goal review missed, so the flag must be honoured. No loop risk: the flag is cleared before restarting, and is only set by a goal change.
- Required fix: make every release path honour and clear the flag. Replace the body of `releaseReview` with the logic of `releaseReviewAndRerun` (or have both `review.go:136` and `service.go:295` call `releaseReviewAndRerun`), so `reviewAgain[id]` is deleted together with `reviewing[id]` and, if it was set, `startReview(id, nil, true)` is invoked (errors logged). Add a service unit test: set `reviewAgain` during a reservation, release it via the failure path, assert the flag is cleared and one review is started.
- Specification decision: not applicable.
- Resolution: resolved (developer fix; reviewer verified, see review.md "Re-review of fixes"; `go test -race ./internal/service/` pass).
- Follow-up: owner developer; validation: new unit test passes, `go test ./...` green, reviewer recheck of all `releaseReview` call sites.

### REV-002: Goal saved but request reports 500 when the review start fails

- Evidence: `internal/service/purchase_goal.go` writes the goal via `store.SetPurchaseGoal`, then returns `goal, true, false, err` on `startReview` error; `internal/httpapi/purchase_goal.go:43-45` maps any error to 500. Functional spec "Save failure: ... the stored goal and reviews are unchanged"; technical spec "review failures follow existing AI review rules" (review failures are not save failures).
- Impact and scenario: on a storage error while starting the review, the UI shows a save error and keeps the draft dirty although the goal was stored; a retry returns `changed: false` and starts no review, so the operator is misinformed and the review for the new goal is silently lost.
- Criticality: non-critical, because it needs a storage failure after a successful write, no data is lost or corrupted, and the stored goal is the one the operator entered.
- Disposition: fix.
- Reason: current behaviour contradicts the functional statement that a reported save failure leaves the stored goal unchanged; the technical spec already classifies review failures separately from save failures. Returning success for the committed save is the accurate, spec-aligned outcome and is a small change. Accepting it as a limitation would knowingly ship a spec mismatch with a cheap fix available.
- Required fix: in `SetPurchaseGoal`, after the goal is stored, do not return the `startReview` error (both call sites): log it (`log.Printf("start AI review for item %s: %v", id, err)`) and return `goal, true, false, nil`, so the API responds 200 with `reviewStarted: false`. Add/adjust a service test asserting no error and `reviewStarted == false` when review start fails after the save. Remove the corresponding limitation from `implementation.md`.
- Specification decision: not applicable (no API contract change: 200 with `reviewStarted: false` is already a documented response).
- Resolution: resolved (developer fix; reviewer verified, see review.md "Re-review of fixes"; `go test -race ./internal/service/` pass).
- Follow-up: owner developer; validation: test passes, reviewer recheck.

### REV-003: Unlisted file change `src/index.css` (and `playwright.config.ts`, existing spec fixtures)

- Evidence: `src/index.css` adds `.purchase-goal`, `.purchase-goal-actions` using Carbon spacing tokens; disclosed in `implementation.md`. `playwright.config.ts` registers the new spec; existing Playwright fixtures add the now-required `purchaseGoal` field.
- Impact and scenario: none functionally; scope deviation from the technical "Allowed files" list only.
- Criticality: non-critical, because these are minimal, necessary consequences of the specified UI and API contract (layout of the specified field, registration of the specified test, fixtures matching the required response field).
- Disposition: reject (no code change).
- Reason: reverting would break layout consistency with `.ai-review`, leave the specified Playwright test unregistered, or make existing fixtures violate the contract. The deviation is disclosed and follows the Carbon frontend skill (tokens, no hard-coded values). Remaining risk: none; recorded here as accepted scope.
- Specification decision: not applicable.
- Resolution: rejected (accepted as-is).
- Follow-up: none.

### REV-004: Error notification persists after the operator edits the draft

- Evidence: `src/components/PurchaseGoal.tsx` `onChange` clears `success` but not `error`.
- Impact and scenario: after a failed save, the error stays visible while the operator edits; it is replaced on the next save attempt.
- Criticality: non-critical, because FR-PURCHASE-GOAL-009 requires that the result be announced and the text kept, which holds; the stale message is accurate about the last attempt and does not block any action.
- Disposition: defer.
- Reason: it is a UX preference, not a defect against the specification; the spec does not define when the error is dismissed, and changing it now adds unspecified behaviour and test churn for no requirement gain. Remaining risk: minor cosmetic inconsistency between success and error clearing.
- Specification decision: not applicable (could be specified in a future UX pass).
- Resolution: deferred.
- Follow-up: owner functional specifier, optional future UX refinement; no tracking issue required.

## Release readiness

No critical findings. REV-001 and REV-002 are non-critical but accepted for fix; they must be fixed, unit-tested, and rechecked by the reviewer before the commit step. REV-003 accepted as-is; REV-004 deferred with rationale. No pending functional questions. QA has not yet run.
