# Tests stage note (issue #225)

Tests: TC-001, TC-003, TC-005 in `tests/leboncoin-session-settings.spec.ts`; TC-002, TC-004 and the two Settings mock routes in `tests/app-information.spec.ts`. All pass on the base branch (no production change, as the technical specification expects). Both spec files pass in full (73 tests), and the #225 tests passed 3 repeated runs.

## Round 1 (test-problem.md, R1-01, R1-02)

A probe showed that the review's suggested technique cannot work: after `trigger.click()`, the menu is not yet rendered when code runs synchronously. After one microtask, React has already committed the open menu and Carbon has already focused **Information**. React commits the menu, places it and runs Carbon's focus effect in a single microtask. No microtask, timeout or animation frame falls between "menu open" and "focus moved".

The only moment when the menu is open and placed but focus is still on the trigger is when the trigger loses focus to Carbon's `focus()` call. In the throttled tests (TC-001 – TC-004), each round now arms a one-shot `blur` listener on the trigger (`arrowKeyBeforeFocusMoves`). The listener dispatches the arrow `keydown` on the trigger at that moment. Chromium then cancels Carbon's outer focus change because a blur handler moved focus. The opening Enter and the closing Enter (TC-001, TC-002) are still `page.keyboard.press` calls with no wait between them. CPU rate 6 and the 5 rounds are kept, and assertions come after the last key of each round.

## "Fails without the fix" check: met

With `handleTriggerKeyDown` reduced to its Escape line (not committed, source restored, `git diff -- src` empty), the results were:

- TC-001 and TC-002 fail: in round 1, the Settings dialog is not visible because Information is focused and opens instead.
- TC-003 fails: Settings is not focused in round 1.
- TC-004 fails: Log out is not focused in round 1. The arrow key now reaches the trigger, not the focused entry, so Carbon's own wrap-around no longer hides the missing branch.
- TC-005 passes, as expected.

Checked again in attempt 2: with the Escape-only handler, `npx playwright test tests/leboncoin-session-settings.spec.ts tests/app-information.spec.ts` gives 4 failed (TC-001 – TC-004 (#225), each on its round 1 assertion), 69 passed. With the real handler, the whole suite passes (the attempt 1 gate run: 213 passed, 1 skipped). `git diff -- src` is empty.

## Red gate: no-tests.md (attempt 3)

The arrow-key branch has been on `master` since issue #215, so the suite passes before any implementation and the red phase cannot fail. `no-tests.md` records this, with a fault injection per test case: the Escape-only handler for TC-001 – TC-004, and an arrow key handled twice for TC-005. The tests are unchanged from attempt 2.

For the development stage: technical.md "Manual follow-up (pipeline)" and "Open" (TS-APP-MENU-007) are out of date. The regression check is met and no owner action is needed for the red gate.
