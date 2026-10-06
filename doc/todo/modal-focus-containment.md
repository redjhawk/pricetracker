# Modal focus containment in Add item and Delete modals

Status: todo (deferred by the user on 2026-10-03; not scheduled)

## Problem

In the existing **Add item** and **Delete** modals, keyboard focus can leave the open dialog when Tab or Shift+Tab is pressed very quickly. QA observed focus escaping after 11–14 of 20 rapid presses. It landed on Carbon focus-sentinel elements, the page body, or header controls, including the header Add item button and the Application menu. With about 200 ms between presses, focus stayed inside. No second modal was opened in testing, and no data was affected.

Cause: Carbon's default focus wrapping uses sentinel elements and a delayed correction. The Settings modal avoids this with `<FeatureFlags enableFocusWrapWithoutSentinels>` (see `src/components/SettingsModal.tsx`).

## Reproduce

1. Open Add item, or Delete from an item's details page.
2. Press Tab 10 times, then Shift+Tab 10 times, with no delay.
3. `document.activeElement` is outside `[role=dialog]`.

## Suggested work

- Apply the same Carbon flag (or an equivalent Carbon-supported mechanism) to `AddItemModal.tsx` and `DeleteModal.tsx`.
- Add Playwright regression tests for rapid Tab/Shift+Tab containment.
- Verify the Application menu cannot open Settings over another modal (FR-APP-MENU-006).
- Run it through the staged workflow (doc/workflow/WORKFLOW.md).

## References

- Finding QA-SET-F03 in [QA report](../changes/2026-10-03-1741-leboncoin-session-settings/qa.md) and its decision in [decisions](../changes/2026-10-03-1741-leboncoin-session-settings/decisions.md).
