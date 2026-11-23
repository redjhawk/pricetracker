# UC-SECRET-006 — Follow the operator guide to renew or remove the session

**Primary actor:** Operator
**Supporting systems:** Operator guide, Settings modal
**Trigger:** The operator reads the guide to save, check, renew or clear the LeBoncoin session.
**Origin:** Issue #265: the guide told operators to clear the session by saving an empty field, which now keeps it.
**Covers:** FR-SECRET-012

## Preconditions

- The operator has the operator guide and access to Settings.

## Main flow

1. The operator reads the guide's Settings instructions for the LeBoncoin session.
2. The guide explains that Settings shows a masked hint (last 4 characters) of the saved session, not the session itself.
3. The guide explains that leaving the input empty keeps the saved session, and pasting a new value replaces it.
4. The guide explains that the session is cleared with "Remove session" followed by Save.
5. The operator follows these steps in Settings (UC-SECRET-002, UC-SECRET-004).

## Alternatives and errors

- **Looking for the old instruction:** the guide no longer says to clear the session by saving an empty field (FR-SECRET-012).

## Postconditions

- The operator knows how to keep, replace and remove the session with the masked Settings.
