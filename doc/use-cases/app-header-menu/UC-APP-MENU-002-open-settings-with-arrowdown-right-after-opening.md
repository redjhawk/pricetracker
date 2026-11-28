# UC-APP-MENU-002 — Open Settings with ArrowDown right after opening the header menu

**Primary actor:** Keyboard user (regular user, or operator in open mode), possibly on a slow device
**Trigger:** The actor opens the header menu with Enter and presses ArrowDown, then Enter, at once.
**Origin:** Issue #225 (review R2-01 on PR #221): no test pressed ArrowDown right after opening the menu on a slow device.
**Covers:** FR-APP-MENU-003, FR-APP-MENU-004, FR-APP-MENU-009

## Preconditions

- Any application page is displayed and no modal is open.
- The header menu is closed and its control has keyboard focus.
- The menu entries are, in order, **Information**, **Settings**, then **Log out** for a regular user.

## Main flow

1. The actor presses Enter on the menu control; the menu opens.
2. The actor presses ArrowDown at once, without waiting, possibly before focus has moved into the menu.
3. Focus is on **Settings**, the entry after the first.
4. The actor presses Enter at once.
5. The menu closes and the Settings modal opens.
6. The actor closes the Settings modal; focus returns to the menu control.

## Alternatives and errors

- **Slow device** (about six times slower): the result is the same as on a normal device, and the same in each of 5 consecutive rounds of steps 1–6 (FR-APP-MENU-009).
- **ArrowDown after a pause** (focus already on the first entry): focus moves to **Settings** and steps 4–6 apply (FR-APP-MENU-004).

## Postconditions

- The Settings modal was opened in every round; after it closes, the menu is closed and the control has focus.
- The page underneath, its data and the URL are unchanged.
