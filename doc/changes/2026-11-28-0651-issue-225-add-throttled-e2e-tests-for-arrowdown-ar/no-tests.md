# Red phase cannot fail (issue #225)

## Why

Issue #225 only adds e2e tests. The behaviour they cover, the ArrowDown/ArrowUp branch of `handleTriggerKeyDown` in `src/components/AppMenu.tsx`, has been on `master` since issue #215. The technical specification (TS-APP-MENU-007) lists no production change. The new tests therefore pass on the base branch, so the red gate cannot fail. Each check is instead shown to fail by a temporary fault injection, which was reverted afterwards: `git diff master -- src` is empty.

## Tests written

- `tests/leboncoin-session-settings.spec.ts`: TC-001, TC-003 and TC-005 (#225), plus the helper `arrowKeyBeforeFocusMoves`.
- `tests/app-information.spec.ts`: TC-002 and TC-004 (#225), the helpers `userMenuReady` and `arrowKeyBeforeFocusMoves`, and the two Settings routes in `mockApi`.

TC-001 – TC-004 throttle the CPU at rate 6 and run 5 rounds. The opening Enter and the closing Enter (TC-001, TC-002) are `page.keyboard.press` calls with no wait between them. Before each round, a one-shot `blur` listener on the trigger dispatches the arrow `keydown` while focus leaves the trigger for Carbon's move into the open, placed menu. This is the only moment when the menu is open and the key still reaches the trigger, so `focusEntryFromTrigger` runs (tests-note.md, round 1). That answers test-problem.md (R1-01, R1-02).

## Proof per test case

Base run (real handler): `npx playwright test tests/leboncoin-session-settings.spec.ts tests/app-information.spec.ts` gives 73 passed. `--grep "#225" --repeat-each=3` gives 15 passed.

Fault 1: `handleTriggerKeyDown` reduced to its Escape line (the arrow keys no longer call `focusEntryFromTrigger`). `--grep "#225"` gives 4 failed, 1 passed:

| Case | Result with fault 1 |
| --- | --- |
| TC-001 (#225) | fails: "Settings opened in round 1", the Settings dialog is not found (Information opens instead) |
| TC-002 (#225) | fails: "Settings opened in round 1", the Settings dialog is not found |
| TC-003 (#225) | fails: "Settings focused in round 1", the `menuitem` Settings is not focused |
| TC-004 (#225) | fails: "Log out focused in round 1", the `menuitem` Log out is not focused |
| TC-005 (#225) | passes, as expected: it presses the arrow key after focus is on Information, so the trigger branch is not involved |

Fault 2, for TC-005 (an arrow key handled twice): a temporary `document` `keydown` listener in `AppMenu` moved focus one more step whenever an arrow key was pressed on a menu entry. TC-005 (#225) fails at line 1187: after Enter, wait, ArrowDown, the `menuitem` Settings is not focused, because focus moved two steps and wrapped back to Information.

Both faults were reverted. `git diff master -- src` is empty, and the #225 tests pass again (15 of 15 over 3 repeats).
