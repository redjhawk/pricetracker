# Issue #225 Add throttled e2e tests for ArrowDown/ArrowUp pressed right after opening the header menu

**Problem:** `handleTriggerKeyDown` in `src/components/AppMenu.tsx` (lines 13-31, `focusEntryFromTrigger`) moves focus into the menu when ArrowDown or ArrowUp reaches the trigger before Carbon has focused the first entry. No test exercises this path. `grep ArrowUp tests/` finds nothing. ArrowDown is pressed only in unthrottled Enter, ArrowDown, Enter sequences (`tests/leboncoin-session-settings.spec.ts` lines 164, 432, 623). Those sequences can pass even without the fix when Carbon has already focused the menu. FR-APP-MENU-007's corner case says that keys pressed in quick succession give the same result as the same keys pressed slowly.

**Suggested work:** In `tests/leboncoin-session-settings.spec.ts`, add a test that throttles the CPU (`Emulation.setCPUThrottlingRate` rate 6 through a CDP session, as TC-001 does at line 458) and repeats each of these sequences 5 times with no wait:
- Enter, ArrowDown, Enter: expect the Settings dialog. Close it before the next round.
- Enter, ArrowUp: expect the last entry to be focused. That is Settings in open mode and Log out as a regular user in protected mode. Press Escape before the next round.

Check that each new test fails when `handleTriggerKeyDown` handles only Escape.

From review comment R2-01 on #221 (`doc/todo/issue-215-r2-01.md`).

## Comments by trusted authors

