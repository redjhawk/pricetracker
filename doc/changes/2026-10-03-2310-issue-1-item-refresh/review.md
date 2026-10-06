# Independent review: item refresh from the list

Reviewer: independent reviewer agent (not the implementer).
Revision reviewed: HEAD `97c44f4` plus uncommitted diff (`package.json`, `playwright.config.ts`, `src/App.tsx`, `src/components/TrackedItemsPage.tsx`, untracked `tests/item-refresh.spec.ts`).
Inputs: functional.md (FR-ITEM-REFRESH-001–008), technical.md (TS-ITEM-REFRESH-001–007), index.md, API_SPECIFICATION.md "Refresh one item", surrounding `App.tsx` (`refreshItems`, list polling effect).

## Summary

Implementation matches the technical specification closely. Menu item order, per-item disabled state, quiet reload after 202, reuse of existing polling, separate error notification naming the item, and isolation from refresh-all state all match. No API contract change; no backend change. Carbon `OverflowMenuItem` and `InlineNotification` are used, so keyboard and accessibility semantics follow the existing menu. Tests cover each TS item. No critical or high findings.

## Findings

### REV-001 — Scope deviation: `playwright.config.ts` changed (provisional severity: low)

- Requirement: technical.md "Scope and refactoring" (permitted files).
- Location: `playwright.config.ts` `testMatch`.
- Evidence: `"item-refresh.spec.ts"` appended to `testMatch`; file not in permitted list.
- Expected/actual: spec lists only App.tsx, TrackedItemsPage.tsx, the new test, and package.json; config also changed.
- Impact: none functionally; it is required for the new spec to run, and it is already disclosed in index.md.
- Suggested action: accept and record it as a necessary deviation.

### REV-002 — Deleted-item row is not removed after a 404 refresh (provisional severity: low)

- Requirement: FR-ITEM-REFRESH-005 corner case "Item deleted elsewhere ... the next list reload removes the row"; technical.md Frontend "the next reload removes the row".
- Location: `src/App.tsx` `handleRefreshListItem` catch branch.
- Evidence: on error, no list reload is triggered. The polling effect runs only while an item is pending or refresh-all is running. So the stale row remains until a later manual action (refresh-all, page reload, add, or delete) causes a reload.
- Expected/actual: the spec wording says "the next list reload", which is literally satisfied. Users may expect the row to disappear, but that behavior is not specified as an immediate reload. The error notification is shown correctly.
- Impact: the stale row stays visible, and repeated refreshes produce the same 404 error.
- Suggested action: optional. Either accept as specified, or call `refreshItems(true)` on 404/after failure. That would be a behavior addition and would need a spec update.

### REV-003 — Test does not cover the in-flight disabled state independently of the pending status (provisional severity: info)

- Location: `tests/item-refresh.spec.ts` "action is disabled while its request is in flight and works by keyboard".
- Evidence: while the gate is held, the mock has not yet set `pending`, so the disabled assertion does exercise `refreshingItemIds`. That is correct. The test does not assert that the item becomes enabled again after a failed in-flight request (the `finally` removal).
- Impact: minor coverage gap; the code path is simple and was reviewed manually.
- Suggested action: optional extra assertion. No code change is required.

## Other checks (no issues)

- Correctness: `setRefreshingItemIds` uses functional updates, so concurrent refreshes of different items do not clobber each other. `refreshItems(true)` swallows its own errors, so a reload failure after a 202 does not produce a false "Could not refresh price" error. This is consistent with the existing quiet polling.
- Refresh-all isolation (FR-007): `refreshing`, `refreshCount`, and `refreshError` are untouched.
- Details page code is unchanged. There are no unrelated changes in the diff.
- Accessibility: the item is reachable inside the existing labelled `OverflowMenu` and disabled through the Carbon `disabled` prop. The error uses Carbon `InlineNotification`, which has the existing role and live semantics. The notification is not dismissible, which matches the existing refresh-all notification (pre-existing pattern, not attributed to this change).
- Security: no new inputs. The error text is rendered as React text, not HTML.

## Verification evidence

Developer-reported results are in index.md: build passed, 6/6 new Playwright tests passed, 30 regression tests passed, Go checks passed. The reviewer did not re-run them; they are not independently verified here.
