# Test cases: keyboard races in the header menu and the Information modal (issue #215)

Sources: functional specifications [app-header-menu](../../specifications/app-header-menu/functional.md) (FR-APP-MENU-007) and [app-information](../../specifications/app-information/functional.md) (FR-APP-INFO-014); use cases [UC-APP-MENU-001](../../use-cases/app-header-menu/UC-APP-MENU-001-close-menu-with-escape-right-after-opening.md) and [UC-APP-INFO-001](../../use-cases/app-information/UC-APP-INFO-001-keep-focus-inside-information-modal.md).

Conventions:

- All cases are **e2e** Playwright tests in `tests/`, run against the Vite dev server with every API call mocked (`mockApi(context, …)` as in `tests/app-information.spec.ts` and `tests/leboncoin-session-settings.spec.ts`). The change is frontend-only: no Go unit or integration case applies.
- **No wait** means consecutive `page.keyboard.press` calls with no `expect`, poll or timeout between them. Assertions are made only after the last key of the sequence, except where a case says "right after each press": then the check is a single synchronous `evaluate` (no `expect.poll`, no auto-retrying assertion), so a focus that is only restored later fails the case.
- **Slow machine**: to make the races reproducible instead of load-dependent, the cases marked with it repeat their check with Chromium CPU throttling (`Emulation.setCPUThrottlingRate`, rate 6, through a CDP session), and loop the key sequence 5 times in the same page.
- **Focus outside the modal**: a `focusin` listener installed on `document` once the modal is open records every focused element that is not inside the `dialog` element; the list must stay empty. This proves "not even momentarily", which an end-of-sequence check cannot.
- Menu control: `button` named "Application menu"; collapsed/expanded state read from `aria-expanded`; menu entries are `menuitem`s.
- The existing tests named in the issue (`tests/leboncoin-session-settings.spec.ts` "header menu: …", `tests/app-information.spec.ts` "keyboard opens the entry and focus stays inside the modal") must pass in the full `npx playwright test` run. The gate is the whole suite: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build` and `npx playwright test`.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-APP-MENU-007 | TC-001, TC-002, TC-003 |
| FR-APP-INFO-014 | TC-004, TC-005, TC-006, TC-007 |

| Use case | Test cases |
| --- | --- |
| UC-APP-MENU-001 | TC-001 (main flow), TC-002 (quick succession), TC-003 (Escape after a pause, pointer then Escape) |
| UC-APP-INFO-001 | TC-004 (steps 1–3), TC-005 (step 4), TC-006 (loading, error, open mode), TC-007 (step 5, postconditions) |

## Header menu

### TC-001 — Escape pressed at once after Enter closes the menu

- Covers: FR-APP-MENU-007, FR-APP-MENU-004; UC-APP-MENU-001 (main flow steps 1–5)
- Level: e2e (`tests/leboncoin-session-settings.spec.ts` or a new header-menu spec)
- Given the list page in open mode, no modal open, the menu closed and the "Application menu" control focused
- When the operator presses Enter then Escape with no wait
- Then no `menuitem` is shown, the control is focused and has `aria-expanded="false"`; then Enter on the control opens the menu again with "Information" and "Settings", the first entry reachable with ArrowDown/focus; Escape closes it again
- Corner cases:
  - Space instead of Enter to open, then Escape with no wait: same result;
  - slow machine (CPU throttling) and the Enter–Escape pair repeated 5 times: the menu is closed after each pair;
  - repeated action: no session, items or settings request is made by opening and closing the menu (request counter unchanged);
  - the page and URL are unchanged after the sequence;
  - protected mode as a regular user (entries "Information", "Settings", "Log out"): same result.

### TC-002 — Keys pressed in quick succession give the same result as slow keys

- Covers: FR-APP-MENU-007; UC-APP-MENU-001 (alternative "keys pressed in quick succession")
- Level: e2e
- Given the menu control focused and the menu closed
- When the operator presses Enter, Escape, Enter with no wait
- Then the menu is open (entries visible), `aria-expanded="true"`; after Escape the menu is closed and the control focused
- Corner cases:
  - Enter, Escape, Space with no wait: the menu is open;
  - Enter, Escape, Enter, Escape with no wait: the menu is closed, the control focused with `aria-expanded="false"`;
  - each sequence is also run slowly (asserting the state after each key) and gives the same final state;
  - slow machine (CPU throttling).

### TC-003 — Existing ways to close the menu are unchanged

- Covers: FR-APP-MENU-007 (no regression of the surrounding behaviour), FR-APP-MENU-002, FR-APP-MENU-004; UC-APP-MENU-001 (alternatives "Escape after a pause" and "menu opened with the pointer, then Escape")
- Level: e2e (existing "header menu: …" test in `tests/leboncoin-session-settings.spec.ts`, which must keep passing)
- Given the menu control on the list page
- When the menu is opened with Enter and Escape is pressed only once the first entry is focused; and when the menu is opened with a click and Escape is pressed
- Then each time the menu closes, focus is on the control, `aria-expanded="false"`, and nothing else changes
- Corner cases:
  - click on the control while open closes it; click outside the menu closes it;
  - Enter, ArrowDown, Enter on the item-details page still opens the Settings modal (selecting an entry is not affected by the Escape handling);
  - Escape while the menu is closed has no effect (the menu stays closed, no modal opens).

## Information modal

### TC-004 — Tab pressed repeatedly with no wait keeps focus inside the modal and wraps to the first control

- Covers: FR-APP-INFO-014, FR-APP-INFO-012; UC-APP-INFO-001 (main flow steps 1–3)
- Level: e2e (`tests/app-information.spec.ts`)
- Given open mode, the menu opened with Enter, **Information** selected with Enter, the version shown and focus inside the modal; the modal's tabbable controls (close icon, **Close** button) are counted as N
- When the operator presses Tab 2×N + 1 times with no wait
- Then right after each press the focused element is inside the dialog; the focused controls follow the modal's tab order and, after the last control, the next Tab focuses the first control directly; the `focusin` record of elements outside the modal is empty
- Corner cases:
  - slow machine (CPU throttling): same result;
  - protected mode as a regular user (username shown) and as the administrator: same result;
  - viewport 400 px wide: same result;
  - repeated action: close and reopen the modal, then repeat the Tab sequence: same result.

### TC-005 — Shift+Tab from the first control goes directly to the last control

- Covers: FR-APP-INFO-014, FR-APP-INFO-012; UC-APP-INFO-001 (main flow step 4)
- Level: e2e
- Given the Information modal open with focus on its first tabbable control
- When the operator presses Shift+Tab, then Shift+Tab N more times with no wait
- Then right after the first press focus is on the last control of the modal; right after each further press focus is on a control inside the modal; the `focusin` record of elements outside the modal is empty
- Corner cases:
  - mixed sequence Tab, Tab, Shift+Tab, Shift+Tab, Shift+Tab with no wait: focus inside the modal after each press;
  - slow machine (CPU throttling).

### TC-006 — Focus stays inside while the version is loading or failed, and in open mode

- Covers: FR-APP-INFO-014; UC-APP-INFO-001 (alternatives "version still loading or not loaded", "open mode", "keys pressed slowly")
- Level: e2e
- Given the Information modal open with the version reply delayed (loading indicator "Loading version" visible), then in a second run with the version reply failing ("The version could not be loaded." visible)
- When the operator presses Tab several times then Shift+Tab several times, with no wait
- Then right after each press focus is inside the modal, and the `focusin` record of elements outside the modal is empty
- Corner cases:
  - the version arrives while the operator is tabbing (delay shorter than the key sequence): focus stays inside the modal and the version replaces the loading indicator;
  - the same sequence pressed slowly (assertion after each key with auto-retry) gives the same focused controls as the fast sequence.

### TC-007 — Closing after fast tabbing returns focus to the menu control

- Covers: FR-APP-INFO-014, FR-APP-INFO-011; UC-APP-INFO-001 (main flow step 5, postconditions)
- Level: e2e
- Given the Information modal opened from the keyboard and Tab pressed N + 1 times with no wait
- When the operator presses Escape at once; then reopens the modal, tabs again, and closes it with the **Close** button; then with the close icon
- Then each time the dialog is hidden, focus is on the "Application menu" control, the page and URL are unchanged, and no `menuitem` is shown
- Corner cases:
  - Escape pressed with no wait after the last Tab still closes the modal (no key is lost);
  - repeated open, tab and close 3 times: same result, and the `focusin` record of elements outside the modal (cleared at each opening) is empty while the modal is open.
