# Independent review: issue #111

- Reviewer: independent reviewer agent (did not write the code)
- Revision reviewed: `fdd2e693212509f756c2b264e84dd853e84b0be6` (`ai-dev/issue-111-item-stop-frontend`), full diff from `ai-dev/issue-111-feature` (= `master` 41905bd), 12 stacked parts
- Inputs: `doc/specifications/amazon-searches-removal/{functional,technical}.md`, `doc/specifications/amazon-request-stop/{functional,technical}.md`, `doc/specifications/claude-token-settings` (FR-CLT-SET-013, TS-CLT-SET-009), `API_SPECIFICATION.md`, `index.md`, `implementation.md`
- Severities are provisional; the adjudicator decides.

## Checked without finding

- Migration `removeAmazonSearches` (`internal/store/store.go`): one transaction, runs only when `amazon_search_items` exists, so it is idempotent; it deletes only `tracked = 0` rows (only search-only items had this value) and Amazon `ai_reviews`; tracked Amazon items (also those moved from or still in a search) and LeBoncoin items and reviews are untouched. `PRAGMA foreign_keys=ON` is set on the single connection (`SetMaxOpenConns(1)`), so child rows cascade. It runs after `createAIReviewSchema`, so `ai_reviews` exists. Covered by `internal/store/search_removal_test.go` (incl. reopen and fresh DB).
- Global stop conversion `convertGlobalAmazonStop`: one transaction, guarded by `blocked = 1`, uses `COALESCE(stopped_at, now)`, Amazon items only; idempotent; tested.
- Gate: counter shared across items, reset by any Amazon answer, stop of every owner's Amazon items at 5, counter persisted; manual check of a stopped item allowed; success clears only that item; failure keeps the stop and counts. Mutex covers check + request + stop, so no lost stop between items. `DueIDs` excludes stopped items; Refresh all skips them in the gate without an attempt record.
- Removal leftovers: no remaining references to searches, `tracked` JSON field, `ITEM_NOT_TRACKED`, `AMAZON_REQUESTS_STOPPED`, `/amazon/requests`, `AmazonAiReview`, `productDetails` in Go, `src/` or `tests/` (only the kept DB column `tracked` and the `tracked-search` CSS of the item filter, both as specified). Removed routes tested to return 404.
- API_SPECIFICATION.md matches the implementation (`amazonStoppedAt`, removed routes, `202` on stopped item refresh).
- Frontend: two tabs, warning derived from the items list with the title/subtitle of TS-AMZ-STOP-005, detail notice with Refresh price enabled, Carbon `InlineNotification` (text, role semantics).

## Findings

### REV-001 Refresh price on a stopped item is silently ignored while a Refresh all is pending

- Type: defect. Provisional severity: major.
- Requirement: FR-AMZ-STOP-006, TS-AMZ-STOP-006.
- Location: `internal/service/service.go` `RefreshItem` (`_, active := s.inFlight[id]` … `if !active { go s.runQueuedCollection(id, true) }`) with `RefreshAll` (`runQueuedCollection(id, false)`); `internal/service/amazon_gate.go` `do` (`if stopped && !manual { return errAmazonStopped }`).
- Evidence: `RefreshAll` reserves every tracked item, including stopped ones, in `inFlight` and waits for `refreshSlots`. If the user then chooses Refresh price on a stopped item that is still queued (or about to be skipped), `RefreshItem` sees it active, returns `202` and starts nothing; the queued run carries `manual = false`, so the gate skips it.
- Expected: the item is checked although stopped. Actual: no request, no attempt, item stays stopped, and the UI got `202` with no failure shown.
- Reproduction: stop all Amazon items, have several items so Refresh all queues beyond the slot count, click Refresh all prices, open a stopped item, click Refresh price.
- Impact: the only restart path appears to work but does nothing; the user may conclude Amazon still fails.
- Suggested resolution: let a manual request upgrade the reservation (e.g. store a manual flag per in-flight id that the queued run reads before calling the gate), or do not reserve stopped Amazon items in `RefreshAll`; add a service test for this sequence.

### REV-002 Stop date shown in two different time zones/formats

- Type: defect (minor) / consistency. Provisional severity: minor.
- Requirement: TS-AMZ-STOP-005 ("formatted as today"), FR-AMZ-STOP-005.
- Location: `src/components/TrackedItemsPage.tsx` `stopDate` (Europe/Paris, no year, no zone label); `src/components/ItemDetail.tsx` reuses `dateTime` (UTC, with year and "UTC").
- Evidence: the same `amazonStoppedAt` is shown e.g. "10 Nov, 21:00" in the tab and "10 Nov 2026, 20:00 UTC" on the details page.
- Impact: user sees two different hours for one stop; the tab value has no time-zone label.
- Suggested resolution: use one formatter for both notices (the tab one matches the former warning; or the detail one, which is labeled), and record the choice.

### REV-003 Stopped items are shown as refreshing after Refresh all

- Type: question (behavior specified only implicitly). Provisional severity: minor.
- Requirement: TS-AMZ-STOP-004, API "stopped ones end without a new attempt, so their status returns to the previous value".
- Location: `internal/service/service.go` `RefreshAll` reserves stopped items in `inFlight` (pending state via `withPending`).
- Evidence: stopped Amazon items appear pending/refreshing until their queued run reaches the gate and is skipped, and `itemsQueued` counts them, although no request is sent (FR-AMZ-STOP-004).
- Impact: brief misleading "refreshing" state; the spec accepts the count but not explicitly the transient pending display. Related to REV-001: not reserving stopped items would solve both (keeping `itemsQueued` unchanged).
- Suggested resolution: adjudicator to confirm acceptable, or skip reservation of stopped Amazon items.

### REV-004 Persistence failure of the stop resets the counter

- Type: question. Provisional severity: minor.
- Requirement: FR-AMZ-STOP-003/010, TS-AMZ-STOP-003 ("Persistence errors are logged").
- Location: `internal/service/amazon_gate.go` (`StopAmazonItems` error logged, then `g.failures = 0` and "Amazon checks stopped" logged).
- Evidence: if the `UPDATE` fails, nothing is stopped but the counter is reset and the log claims the stop happened; Amazon requests continue.
- Impact: low (SQLite write failure is rare), but the log is misleading.
- Suggested resolution: on error keep the counter at the limit (retry the stop on next failure) or at least do not log the stop as done.

### REV-005 Per-part build not independently re-verified

- Type: question / verification limitation. Provisional severity: informational.
- Requirement: technical implementation plan ("each part must pass `go vet`, `go test`, `npm run build` on its own").
- Evidence: `implementation.md` states every part passed; this reviewer could not create a worktree in this environment to rebuild each part. Diff inspection found no dangling reference across parts (re-slicing recorded in `implementation.md`); between parts 11 and 12 the frontend calls the removed `/api/v1/amazon/requests`, caught as planned. Playwright specs were not run by the developer (missing `libnspr4.so`).
- Suggested resolution: QA/coordinator run `go vet ./... && go test ./... && npm run build` per part branch and the Playwright specs where the browser runs.

## Summary

No data-loss defect found in the migration or the conversion. One major defect (REV-001: manual restart lost while a Refresh all is pending), three minor items (REV-002–004) and one verification limitation (REV-005).
