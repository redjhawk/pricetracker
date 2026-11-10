# Review decisions: issue #111 – delete Amazon searches, per-item Amazon stop

Reviewed specification revisions: `doc/specifications/amazon-searches-removal/{functional,technical}.md`, `doc/specifications/amazon-request-stop/{functional,technical}.md` as of `fdd2e69`
Reviewed code revision: `fdd2e693212509f756c2b264e84dd853e84b0be6` (`ai-dev/issue-111-item-stop-frontend`), full change `git diff ai-dev/issue-111-feature HEAD`
Reviewer: independent reviewer agent ([review.md](review.md))
Adjudicator: independent review adjudicator agent (different from developer and reviewer)

## Fix placement

Preference: commit the fixes as **new commits on the stack tip** (`ai-dev/issue-111-item-stop-frontend`, or a new part branch stacked on it if the tip PR would exceed the size limit). Rewriting earlier part branches would need rebasing every later part, and nothing deploys before the final feature→`master` merge, so a tip commit is simpler and equally safe. If the coordinator prefers part-local fixes: REV-001, REV-003, REV-004 belong to the item-stop-backend part (`2c1347e`, "stop Amazon checks per item after five failures"); REV-002 belongs to the item-stop-frontend part (`fdd2e69`).

## Findings and decisions

### REV-001: Refresh price on a stopped item is silently ignored while a Refresh all is pending

- Evidence: verified in code. `RefreshAll` (`internal/service/service.go` ~182–206) reserves every id in `inFlight`, including stopped Amazon items, and starts `runQueuedCollection(id, false)`. `RefreshItem` (~208–227) sees `active` and returns `202` without starting anything. The queued run reaches `amazonGate.do` with `manual = false` and is skipped (`if stopped && !manual`, `internal/service/amazon_gate.go:47`).
- Impact and scenario: user with stopped Amazon items clicks Refresh all prices, then Refresh price on a stopped item while runs are queued behind `refreshSlots`: no request, no attempt record, the item stays stopped, the UI received `202`. Refresh price is the only restart path (FR-AMZ-STOP-006), so the user is misled into thinking Amazon still fails.
- Criticality: **critical**, because it violates a specified functional requirement (FR-AMZ-STOP-006 "the item is checked although stopped", TS-AMZ-STOP-006) on the sole recovery action, silently (success response, no failure shown). The trigger is a normal user sequence (Refresh all then Refresh price), not an exotic race. No data loss, but the core feature's restart path is unreliable.
- Disposition: **fix**.
- Reason: required behavior is unambiguous; no functional decision needed.
- Required change: do not reserve stopped Amazon items in `RefreshAll` (they would be skipped by the gate anyway; keep the returned `itemsQueued` count unchanged as the API states), so a later `RefreshItem` reserves the item and runs it as manual. Alternative accepted: record a per-id manual flag that the queued run reads before calling the gate. Either way, a stopped item that is already running a manual check must stay single-flight.
- Specification decision: not applicable.
- Resolution: pending (developer).
- Owner: developer; recheck: reviewer; QA.
- Critical fix confirmation (required before completion):
  - Use case: add `doc/use-cases/price-collection/UC-PC-02-restart-stopped-amazon-item-during-refresh-all.md` (stopped Amazon items; Refresh all queued; Refresh price on a stopped item → item checked, on answer restarted, others stay stopped), linked from `doc/use-cases/README.md`.
  - Regression test: service test in `internal/service` (e.g. `TestRefreshItemRestartsStoppedItemWhileRefreshAllQueued`): fill `refreshSlots`/use a blocking fake collector, call `RefreshAll` with several stopped Amazon items, call `RefreshItem` on one stopped item, release, assert the fake Amazon request ran for that item, its stop is cleared, the others remain stopped, and `RefreshAll` still returns the full count. Must fail on `fdd2e69` and pass after the fix; `go vet ./... && go test ./...` green.

### REV-002: Stop date shown in two different time zones/formats

- Evidence: `TrackedItemsPage.tsx` `stopDate` (Europe/Paris, no year/zone) vs `ItemDetail.tsx` `dateTime` (UTC, labeled) for the same `amazonStoppedAt`.
- Impact and scenario: user sees e.g. 21:00 in the tab and 20:00 UTC on the details page for one stop; confusing but each value is correct for its own (labeled or prior) format.
- Criticality: non-critical, because no wrong data, no blocked action; cosmetic inconsistency. TS-AMZ-STOP-005 "formatted as today" is met by each screen keeping its existing format.
- Disposition: **fix** (cheap, improves consistency).
- Reason: one-line change; leaving it invites support confusion.
- Required change: use the details page's labeled `dateTime` formatter (UTC with zone label) for the tab warning too, or label the tab's zone; record the choice in `implementation.md`.
- Specification decision: not applicable (technical formatting choice).
- Resolution: pending. Owner: developer. Validation: `npm run build`; QA sees identical time on tab and details page.

### REV-003: Stopped items are shown as refreshing after Refresh all

- Evidence: `RefreshAll` reserves stopped items, so `withPending` shows them refreshing until the gate skips them.
- Impact and scenario: brief misleading refreshing state on items that send no request (FR-AMZ-STOP-004).
- Criticality: non-critical, because transient display only, status returns to the previous value as the API states.
- Disposition: **fix** — resolved by the REV-001 fix (not reserving stopped items); no separate work if that option is chosen.
- Reason: same root cause; fixing REV-001 by the first option removes it for free.
- Resolution: pending (with REV-001). Owner: developer. Validation: REV-001 test additionally asserts stopped items are not in `inFlight` after `RefreshAll`; QA: stopped items do not show refreshing after Refresh all. If the alternative REV-001 fix is chosen, this is deferred to `doc/todo/` with this rationale.

### REV-004: Persistence failure of the stop resets the counter

- Evidence: `amazon_gate.go:60–64`: `StopAmazonItems` error is logged, then `failures = 0` and "Amazon checks stopped" is logged anyway.
- Impact and scenario: only on SQLite write failure (rare, typically disk full/corruption); requests continue for 5 more failures and the log falsely claims a stop.
- Criticality: non-critical, because very low likelihood, no data loss, and the error itself is logged (TS-AMZ-STOP-003 requirement met); the effect is extra Amazon requests, the state before this feature anyway.
- Disposition: **fix** (small).
- Required change: on `StopAmazonItems` error, keep the counter at the limit (so the next failure retries the stop) and log the stop success only when the update succeeded.
- Resolution: pending. Owner: developer. Validation: gate unit test with a failing store stop: no "stopped" outcome, counter stays ≥ limit, next failure retries the stop.

### REV-005: Per-part build not independently re-verified

- Evidence: reviewer could not create worktrees; developer reports each part passing; Playwright not run (`libnspr4.so` missing).
- Impact: a part PR could fail CI on its own; no runtime effect since only the final merge deploys.
- Criticality: non-critical (verification gap, not a defect), but it must not be treated as resolved.
- Disposition: **fix** (verification action, no code change).
- Required change: coordinator/QA runs `go vet ./... && go test ./... && npm run build` on every part branch, and Playwright specs where the browser runs; record results in `commit-step.md`/QA report. If Playwright cannot run, record it as a limitation.
- Resolution: pending. Owner: coordinator/QA.

## Summary

| ID | Title | Criticality | Disposition | Action |
|---|---|---|---|---|
| REV-001 | Refresh price on stopped item ignored while Refresh all pending | critical | fix | Don't reserve stopped Amazon items in `RefreshAll` (or carry a manual flag); add UC-PC-02 use case and service regression test |
| REV-002 | Stop date in two time zones/formats | non-critical | fix | Use one labeled formatter in tab and details page |
| REV-003 | Stopped items shown refreshing after Refresh all | non-critical | fix | Resolved by REV-001 fix; assert in its test |
| REV-004 | Stop persistence failure resets counter, misleading log | non-critical | fix | Keep counter at limit on error; log stop only on success; gate test |
| REV-005 | Per-part build not re-verified | non-critical | fix (verification) | Run vet/test/build per part branch and Playwright where possible; record |

Fixes as new commits on the stack tip (preferred). Published as an issue comment on #111: by the coordinator. Included in the final run report: yes.

## Release readiness

Blocked: REV-001 (critical) is open until fixed, rechecked by the reviewer, its use case added and its regression test passing. REV-002–005 are open fix items. QA not yet run.
