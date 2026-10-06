# Review decisions: item-refresh

Reviewed specification revisions: [functional.md](../../specifications/item-refresh/functional.md) (ready, FR-ITEM-REFRESH-001–008), [technical.md](../../specifications/item-refresh/technical.md) (ready, TS-ITEM-REFRESH-001–007), API_SPECIFICATION.md unchanged ("Refresh one item").
Reviewed code revision: HEAD `97c44f4` plus uncommitted working tree (`package.json`, `playwright.config.ts`, `src/App.tsx`, `src/components/TrackedItemsPage.tsx`, untracked `tests/item-refresh.spec.ts`).
Reviewer: independent reviewer agent ([review.md](review.md))
Adjudicator: independent review adjudicator agent (distinct from developer and reviewer)

The adjudicator read the diff independently. It confirmed the reviewer's summary claims that bear on these decisions. `handleRefreshListItem` uses functional `Set` updates, never touches the refresh-all state (`refreshing`, `refreshCount`, `refreshError`), and clears `itemRefreshError` at the start of each refresh. The menu item sits between "Open on …" and Delete, and its disabled condition is `status === "pending" || refreshingItemIds.has(id)`.

## Findings and decisions

### REV-001: Scope deviation, `playwright.config.ts` changed

- Evidence: `playwright.config.ts` `testMatch` gains `"item-refresh.spec.ts"`. The technical spec's permitted files do not include this file. The developer disclosed the change in index.md.
- Impact and scenario: Without this change, `tests/item-refresh.spec.ts` (a deliverable that the technical spec requires) would never run, because `testMatch` is an explicit allow-list. The change only appends one entry. It does not change the existing tests, workers, reporter, or base URL. No runtime or user effect.
- Criticality: non-critical. Configuration is touched only for tests. The change is the minimum needed to run a test the spec requires. It is not refactoring, and it does not change the product or the API.
- Disposition: reject (no code change).
- Reason: The technical spec requires the Playwright file and its execution (Verification section). The permitted-files list left out the config only by oversight. Reverting the change would stop the required verification from running, which contradicts the spec more seriously. The deviation is disclosed and narrow, so it is recorded here as accepted. Remaining risk: none beyond one more spec file running in the suite.
- Specification decision: not applicable (technical scope only; no functional question).
- Resolution: rejected; accepted as a recorded necessary deviation.
- Follow-up: none.

### REV-002: Deleted-item row is not removed after a 404 refresh

- Evidence: The `src/App.tsx` `handleRefreshListItem` catch branch sets only `itemRefreshError`. The list polling effect runs only while an item is pending or refresh-all is running. A row for an item deleted elsewhere therefore stays until some later reload.
- Impact and scenario: The operator refreshes an item that another session deleted. The operator sees "Could not refresh price: <item>: <not found detail>", and the stale row stays until the next reload (refresh-all, add, delete, page reload, or any pending-driven poll). Retrying shows the same error. No data is corrupted, no price is lost, and the operator is told what happened. This needs concurrent deletion, which is unlikely in a single-operator v1 app.
- Criticality: non-critical. FR-ITEM-REFRESH-005 (error naming the item, prices kept) is met. The corner case says "the next list reload removes the row" and does not require that the failure itself trigger a reload. The current behavior literally satisfies it, as the reviewer also notes.
- Disposition: reject.
- Reason: Making the 404 trigger an immediate reload would add behavior beyond the ready functional and technical specs. Both specs describe removal on "the next reload", and the technical spec explicitly lists no reload on failure. The adjudicator cannot invent requirements. The current behavior is consistent with the existing list, which does not reload after other failed actions either. The remaining risk is a cosmetic stale row with a clear error message, and any normal action clears it. If the user wants the row removed immediately, that is a new functional request (it would amend FR-ITEM-REFRESH-005), not a defect in this change.
- Specification decision: not required for release, because the specification is unambiguous as written. Immediate removal would need a user functional request.
- Resolution: rejected; implementation matches the specification.
- Follow-up: QA should run the "item deleted then refreshed" case (technical.md QA plan). It should confirm that the error appears and that the row disappears on the next reload.

### REV-003: Test does not assert re-enable after a failed in-flight request

- Evidence: `tests/item-refresh.spec.ts` "action is disabled while its request is in flight and works by keyboard" covers the disabled state while the request is in flight. No test asserts that **Refresh price** becomes enabled again after a failed request (the `finally` removal from `refreshingItemIds`).
- Impact and scenario: A future regression in the `finally` cleanup would leave an item's action permanently disabled after one failure, and no automated test would catch it. The current code is correct: `finally` always deletes the ID through a functional update, as inspected.
- Criticality: non-critical. The finding is a test-coverage suggestion, not a defect. The behavior is correct today, and FR-ITEM-REFRESH-004 obligations are met.
- Disposition: defer.
- Reason: Delivered behavior is correct, so adding the assertion now would mean another implementation, review, and QA cycle with no change to the product. QA (stage 7) already covers retry-after-failure scenarios, which exercise this path in the running interface. Remaining risk: a possible future regression in cleanup that only manual testing would catch.
- Specification decision: not applicable.
- Resolution: deferred. QA must execute "refresh fails (mocked or deleted item) → retry the same item's **Refresh price** is enabled" and record the result. If QA finds it disabled, the deferral becomes a critical fix.
- Follow-up: owner is the QA tester for this change. Add the automated assertion the next time `tests/item-refresh.spec.ts` is modified.

## Release readiness

- Open blocking (critical) findings: none.
- Fix items routed to the developer: none.
- Unresolved functional questions: none.
- Developer-reported verification (build, 6 new and 30 regression Playwright tests, Go checks) has not been independently re-run by reviewer or adjudicator. QA must execute it.
- Next stage: QA (stage 7). It must include the REV-002 deleted-item case and the REV-003 retry-after-failure case. Completion still requires QA to pass and a commit (stage 8).

## Post-QA adjudication

### QA-OBS-001: Row overflow menu trigger is announced as "Options" rather than "Actions for <item>"

- Evidence: QA (`qa.md`) found that the row menu trigger's accessible name is "Options". `git show HEAD:src/components/TrackedItemsPage.tsx` (line 225) already contains the same `<OverflowMenu flipped size="sm" aria-label={`Actions for ${item.title ?? item.listingId}`}>`. The working tree (line 237) has it unchanged. So the `aria-label` not reaching the Carbon trigger button is pre-existing and was not introduced by this change. Existing regression tests already locate the trigger by "Options".
- Impact and scenario: A screen-reader user moving through the table hears "Options" for every row's menu button. The row context (the cell is in the item's table row) gives an indirect cue only. Keyboard operation works: QA confirmed that the menu opens, **Refresh price** can be selected, and status and error feedback can be perceived. No data or functional effect.
- Criticality: non-critical. FR-ITEM-REFRESH-008's acceptance criterion (a keyboard-only operator can open the menu, select **Refresh price**, and perceive the result) is met. The phrase "existing labelled 'Actions for <item>' menu" describes the menu the action was added to. It does not require this change to fix that menu's labelling.
- Disposition: defer.
- Reason: The defect predates this change and affects all row menu actions (View details, Open, Delete), not only Refresh price. Fixing it means changing how the label is passed to Carbon's `OverflowMenu` trigger (for example `iconDescription`/`aria-label` handling for the installed Carbon version). It also means updating existing regression tests that select by "Options". That widens scope beyond the narrow-change rule for this feature and needs its own spec, review, and QA. Rejecting is inappropriate because it is a real accessibility gap, so it is deferred rather than dismissed.
- Specification decision: no amendment to the item-refresh specs. FR-008 is satisfied as written.
- Resolution: deferred; does not block release of item-refresh.
- Follow-up: open a separate accessibility change. Make the row overflow trigger's accessible name "Actions for <item>", then update the tests that locate it by "Options".
