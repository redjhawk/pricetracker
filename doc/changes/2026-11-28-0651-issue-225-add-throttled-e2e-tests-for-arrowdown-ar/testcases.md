# Test cases: ArrowDown and ArrowUp pressed at once after opening the header menu (issue #225)

Sources: functional specification [app-header-menu](../../specifications/app-header-menu/functional.md) (FR-APP-MENU-009, with FR-APP-MENU-003 and FR-APP-MENU-004); use cases [UC-APP-MENU-002](../../use-cases/app-header-menu/UC-APP-MENU-002-open-settings-with-arrowdown-right-after-opening.md) and [UC-APP-MENU-003](../../use-cases/app-header-menu/UC-APP-MENU-003-focus-last-entry-with-arrowup-right-after-opening.md).

Conventions:

- All cases are **e2e** Playwright tests in `tests/`, run against the Vite dev server with every API call mocked (`mockApi(context, …)`). The change is frontend-only and adds tests only: no Go unit or integration case applies.
- Open mode cases go in `tests/leboncoin-session-settings.spec.ts` (helpers `listReady`, `menuTrigger`, `throttleCpu`, `expectMenuClosed`); regular-user cases go where the protected-mode session mock already exists (`tests/app-information.spec.ts`, `userSession`), or reuse an equivalent mock.
- **No wait** means consecutive `page.keyboard.press` calls with no `expect`, poll or timeout between them. Assertions are made only after the last key of the sequence.
- **Slow device**: Chromium CPU throttling at rate 6 through a CDP session (`Emulation.setCPUThrottlingRate`, the existing `throttleCpu` helper), applied once the page is ready, and the sequence repeated 5 times in the same page.
- Menu control: `button` named "Application menu"; collapsed/expanded state read from `aria-expanded`; menu entries are `menuitem`s; the Settings modal is the `dialog` named "Settings".
- **Fails without the fix**: each case is checked once by hand with `handleTriggerKeyDown` in `src/components/AppMenu.tsx` reduced to the Escape branch (the arrow keys no longer call `focusEntryFromTrigger`); the throttled case must then fail. The check is not committed. The implementer reports the result in the pull request.
- The gate is the whole suite: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build` and `npx playwright test`. The existing header-menu tests (`TC-001`–`TC-003 (#215)`, "header menu: …") must keep passing.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-APP-MENU-009 | TC-001, TC-002, TC-003, TC-004 |
| FR-APP-MENU-004 | TC-001, TC-002, TC-003, TC-004, TC-005 |
| FR-APP-MENU-003 | TC-001, TC-002, TC-005 |

| Use case | Test cases |
| --- | --- |
| UC-APP-MENU-002 | TC-001 (main flow, slow device, open mode), TC-002 (regular user), TC-005 (ArrowDown after a pause) |
| UC-APP-MENU-003 | TC-003 (main flow, slow device, open mode), TC-004 (regular user), TC-005 (ArrowUp after a pause) |

## ArrowDown

### TC-001 — Slow device, open mode: Enter, ArrowDown, Enter opens Settings in each of 5 rounds

- Covers: FR-APP-MENU-009 (a), FR-APP-MENU-004, FR-APP-MENU-003; UC-APP-MENU-002 (main flow steps 1–6, alternative "slow device")
- Level: e2e (`tests/leboncoin-session-settings.spec.ts`)
- Given open mode, the list page ready (entries **Information**, **Settings**), no modal open, the CPU throttled at rate 6, the "Application menu" control focused and the menu closed
- When the operator presses Enter, ArrowDown, Enter with no wait, then closes the Settings modal (Escape or **Cancel**), and repeats this 5 times
- Then in each round the Settings dialog is visible and no `menuitem` is shown; after it closes the dialog is hidden, focus is on the control and `aria-expanded="false"`
- Corner cases:
  - repeated action: the 5th round gives the same result as the 1st (no state left over from a previous round);
  - the URL is unchanged (`/`) after the 5 rounds; the only requests are the Settings modal's own loads (one per opening, FR-LBC-SET-002), none made by the menu itself;
  - fails without the fix (see conventions): with only Escape handled, at least one round does not open the Settings dialog.

### TC-002 — Slow device, regular user: Enter, ArrowDown, Enter opens Settings in each of 5 rounds

- Covers: FR-APP-MENU-009 (a), FR-APP-MENU-004, FR-APP-MENU-003; UC-APP-MENU-002 (actor "regular user", alternative "slow device")
- Level: e2e (`tests/app-information.spec.ts` or the spec holding the protected-mode session mock)
- Given protected mode with a logged-in regular user (entries **Information**, **Settings**, **Log out**), the CPU throttled at rate 6, the control focused and the menu closed
- When the operator presses Enter, ArrowDown, Enter with no wait, closes the Settings modal, and repeats this 5 times
- Then in each round the Settings dialog opens (not the Information modal, no logout request); after it closes focus is on the control and `aria-expanded="false"`
- Corner cases:
  - no logout request is made in any round and the user stays logged in (the **Log out** entry is still shown when the menu is reopened);
  - the URL is unchanged.

## ArrowUp

### TC-003 — Slow device, open mode: Enter, ArrowUp focuses Settings in each of 5 rounds

- Covers: FR-APP-MENU-009 (b), FR-APP-MENU-004; UC-APP-MENU-003 (main flow steps 1–4, alternative "slow device")
- Level: e2e (`tests/leboncoin-session-settings.spec.ts`)
- Given open mode, the list page ready (entries **Information**, **Settings**), the CPU throttled at rate 6, the control focused and the menu closed
- When the operator presses Enter, ArrowUp with no wait; then Escape; and repeats this 5 times
- Then after ArrowUp in each round the menu is open (`aria-expanded="true"`) and the `menuitem` "Settings" (the last entry) is focused; after Escape no `menuitem` is shown, focus is on the control and `aria-expanded="false"`
- Corner cases:
  - no entry is activated: no dialog is open after any round;
  - repeated action: same result in each of the 5 rounds; no request is made by opening and closing the menu (request counter unchanged, `api.gets` 0); the URL is unchanged;
  - fails without the fix: with only Escape handled, at least one round leaves focus on **Information** (or on the control) instead of **Settings**.

### TC-004 — Slow device, regular user: Enter, ArrowUp focuses Log out in each of 5 rounds

- Covers: FR-APP-MENU-009 (b), FR-APP-MENU-004; UC-APP-MENU-003 (actor "regular user", alternative "slow device")
- Level: e2e (`tests/app-information.spec.ts` or the spec holding the protected-mode session mock)
- Given protected mode with a logged-in regular user (entries **Information**, **Settings**, **Log out**), the CPU throttled at rate 6, the control focused and the menu closed
- When the operator presses Enter, ArrowUp with no wait; then Escape; and repeats this 5 times
- Then after ArrowUp in each round the `menuitem` "Log out" is focused; after Escape the menu is closed, focus is on the control and `aria-expanded="false"`
- Corner cases:
  - Log out is focused but not activated: no logout request is made and the user stays logged in after the 5 rounds;
  - fails without the fix: with only Escape handled, at least one round leaves focus elsewhere than **Log out**.

## Arrow keys after a pause

### TC-005 — ArrowDown and ArrowUp pressed once focus is on the first entry are unchanged

- Covers: FR-APP-MENU-004, FR-APP-MENU-003, FR-APP-MENU-009 (reference behaviour the fast keys must match); UC-APP-MENU-002 and UC-APP-MENU-003 (alternatives "after a pause")
- Level: e2e (`tests/leboncoin-session-settings.spec.ts`; existing TC-003 (#215) and "header menu: …" tests may already prove part of it)
- Given open mode, no CPU throttling, the control focused and the menu closed
- When the operator presses Enter, waits until the `menuitem` "Information" is focused, then presses ArrowDown; and in a second sequence presses Enter, waits until "Information" is focused, then presses ArrowUp
- Then after ArrowDown "Settings" is focused and Enter opens the Settings dialog; after ArrowUp "Settings" (the last entry) is focused and Escape closes the menu with focus on the control
- Corner cases:
  - the final state of each slow sequence equals the final state of the matching fast sequence in TC-001 and TC-003 (same focused entry, same dialog);
  - the arrow key reaches the focused entry, not the trigger: it is handled once, so focus moves by exactly one step (not two).
