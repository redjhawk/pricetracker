# UC-APP-INFO-001 — Keep keyboard focus inside the Information modal

**Primary actor:** Keyboard user (regular user, administrator, or operator in open mode)
**Trigger:** The actor moves through the open Information modal with Tab or Shift+Tab, at any speed.
**Origin:** Issue #215: after Tab from the last control, focus briefly left the modal before coming back.
**Covers:** FR-APP-INFO-011, FR-APP-INFO-012, FR-APP-INFO-014

## Preconditions

- The actor opened the header menu with the keyboard and selected **Information** with Enter or Space.
- The Information modal is open and focus is on a control inside it.

## Main flow

1. The actor presses Tab several times in a row, with no wait between presses.
2. Right after each press, focus is on the next control inside the modal.
3. From the last control, Tab moves focus directly to the first control of the modal.
4. The actor presses Shift+Tab on the first control; focus moves directly to the last control of the modal.
5. The actor closes the modal with Escape, the Close button or the close icon; focus returns to the header menu control.

## Alternatives and errors

- **Keys pressed slowly:** the same result as in steps 2–4.
- **Version still loading or not loaded** (loading indicator or error message): focus stays inside the modal in the same way (FR-APP-INFO-009, FR-APP-INFO-010).
- **Open mode** (no username shown): focus stays inside the modal in the same way.

## Postconditions

- At no moment while the modal was open was focus on an element outside it.
- After closing, the modal is gone, nothing else changed, and focus is on the header menu control (FR-APP-INFO-011).
