# Tests stage note (issue #225)

Tests added: TC-001, TC-003, TC-005 in `tests/leboncoin-session-settings.spec.ts`; TC-002, TC-004 and the two Settings mock routes in `tests/app-information.spec.ts`. All pass on the base branch, as the technical specification expects (no production change). Both spec files pass in full (73 tests).

## "Fails without the fix" check: not met

With the arrow-key branch of `handleTriggerKeyDown` disabled (Escape only), TC-001 – TC-004 still pass. The source was restored afterwards.

Cause: a temporary counter in `handleTriggerKeyDown` showed that across the 5 throttled rounds of TC-003, no ArrowUp reached the trigger. Playwright's `page.keyboard.press` returns only after the renderer has handled the key. React handles the Enter keydown as a discrete event and runs Carbon's placement and focus effects at once, so focus is already on **Information** when the arrow key arrives, even at CPU rate 6. Through Playwright's keyboard, the `focusEntryFromTrigger` path cannot be reached in Chromium.

As the technical specification requires, the rate (6) and the round count (5) were kept. The tests still check the required behaviour (FR-APP-MENU-009), but they do not catch a regression in the trigger's arrow-key branch. The owner decides whether a different technique is wanted, for example dispatching the arrow keydown on the trigger in the same task as the opening.
