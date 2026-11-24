# UC-SECRET-005 — Change the LeBoncoin session after it changed since Settings opened

**Primary actor:** Operator
**Supporting systems:** PriceFollower application, Settings modal, LeBoncoin collector (renews, expires or revokes the session)
**Trigger:** The operator replaces or removes the LeBoncoin session from a modal opened before the stored session changed.
**Origin:** Issue #265: the "session changed" check keeps working, but only when the session is being changed.
**Covers:** FR-SECRET-008, FR-SECRET-009

## Preconditions

- Modal A was opened and loaded while the stored session was X.
- The stored session then changed to Y: renewed by LeBoncoin (FR-LBC-COL-014), expired or revoked (FR-LBC-COL-015), or saved or removed from another browser.

## Main flow

1. In modal A, the operator pastes a new session N into the LeBoncoin input (or chooses "Remove session").
2. The operator selects Save.
3. The save is refused: nothing is saved (neither entry), the stored session stays Y, and the modal stays open with the input (or the requested removal) and a warning that the session changed and Settings must be reloaded (FR-SECRET-008).
4. The operator closes and reopens Settings.
5. The LeBoncoin hint shows Y's last characters, or its expired/revoked state (FR-SECRET-009).
6. The operator pastes N again (or chooses "Remove session") and saves; the save succeeds.

## Alternatives and errors

- **LeBoncoin entry untouched:** from modal A the operator saves only a new Claude token, or nothing; the save is not refused and the session stays Y (FR-SECRET-008, see UC-SECRET-003).
- **No change since loading:** the replacement or removal succeeds at once (UC-SECRET-002, UC-SECRET-004).

## Postconditions

- The stored session was never overwritten by a modal that did not see its latest state.
