# Issue #290 Replace the vacuous progress check in the long whitespace TC-005 token e2e test

**Problem:** In tests/claude-token-settings.spec.ts:101, `await expect(page.getByText(/too long|Saving settings…/i)).toHaveCount(0)` runs after `await expect(dialog).toBeHidden()`. The `Saving settings…` progress is only rendered inside the open modal (src/components/SettingsModal.tsx:211), so that part can never match.

**Suggested work:** The frontend sends no request when both entries are untouched (TS-SECRET-019), so assert `expect(api.puts).toEqual([])` instead of the loop and the progress text. Alternatively, check for the progress text before the modal closes. Keep the `/too long/` check.

From review comment R1-03 on #287 (`doc/todo/issue-286-r1-03.md`).

## Comments by trusted authors

