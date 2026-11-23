# UC-APP-MENU-001 — Close the header menu with Escape right after opening it

**Primary actor:** Keyboard user (regular user, administrator, or operator in open mode)
**Trigger:** The actor opens the header menu with the keyboard and presses Escape at once.
**Origin:** Issue #215: an Escape pressed before focus moved into the menu was ignored and the menu stayed open.
**Covers:** FR-APP-MENU-002, FR-APP-MENU-004, FR-APP-MENU-007

## Preconditions

- Any application page is displayed and no modal is open.
- The header menu is closed and its control has keyboard focus.

## Main flow

1. The actor presses Enter (or Space) on the menu control.
2. The menu opens and the control reports the expanded state.
3. The actor presses Escape at once, without waiting, possibly before focus has moved into the menu.
4. The menu closes, focus is on the control and the control reports the collapsed state.
5. The actor presses Enter or Space on the control again; the menu opens again with its entries (for example **Information** and **Settings**) reachable with the keyboard.

## Alternatives and errors

- **Escape after a pause** (focus already inside the menu): the menu closes and focus returns to the control, as in steps 4–5 (FR-APP-MENU-004).
- **Keys pressed in quick succession** (Enter, Escape, Enter/Space): the result is the same as with the same keys pressed slowly; each key acts on the state left by the previous one (FR-APP-MENU-007).
- **Menu opened with the pointer, then Escape:** the menu closes without other effect (FR-APP-MENU-002).

## Postconditions

- The menu is closed, the control has focus and reports the collapsed state.
- The page, its data and the URL are unchanged; no server request was made.
