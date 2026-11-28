# UC-APP-MENU-003 — Focus the last menu entry with ArrowUp right after opening the header menu

**Primary actor:** Keyboard user (regular user, or operator in open mode), possibly on a slow device
**Trigger:** The actor opens the header menu with Enter and presses ArrowUp at once.
**Origin:** Issue #225 (review R2-01 on PR #221): no test pressed ArrowUp right after opening the menu on a slow device.
**Covers:** FR-APP-MENU-004, FR-APP-MENU-009

## Preconditions

- Any application page is displayed and no modal is open.
- The header menu is closed and its control has keyboard focus.
- The menu entries are, in order, **Information**, **Settings**, then **Log out** for a regular user.

## Main flow

1. The actor presses Enter on the menu control; the menu opens.
2. The actor presses ArrowUp at once, without waiting, possibly before focus has moved into the menu.
3. Focus is on the last entry: **Log out** for a logged-in regular user, **Settings** in open mode.
4. The actor presses Escape; the menu closes and focus returns to the control.

## Alternatives and errors

- **Slow device** (about six times slower): the result is the same as on a normal device, and the same in each of 5 consecutive rounds of steps 1–4 (FR-APP-MENU-009).
- **ArrowUp after a pause** (focus already on the first entry): focus moves to the last entry and step 4 applies (FR-APP-MENU-004).

## Postconditions

- The menu is closed, the control has focus and reports the collapsed state.
- No entry was activated; the page, its data and the URL are unchanged.
