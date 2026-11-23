# UC-SECRET-001 — See which secrets are saved in Settings

**Primary actor:** Operator (logged-in regular user in protected mode, anyone in open mode)
**Supporting systems:** PriceFollower application, Settings modal
**Trigger:** The operator opens **Settings** from the header menu.
**Origin:** Issue #265: Settings showed the saved Claude token and LeBoncoin session in full, and the server sent them back.
**Covers:** FR-SECRET-001, FR-SECRET-002, FR-SECRET-003, FR-SECRET-009, FR-SECRET-010, FR-SECRET-011, FR-SECRET-013

## Preconditions

- The operator has a saved Claude token T ending in `wxyz` and a saved LeBoncoin session S ending in `9f3a`, both longer than the hint threshold. They may have been saved before this change (FR-SECRET-010).

## Main flow

1. The operator selects **Settings** in the header menu; the modal loads the settings from the server.
2. The server answers without T or S; it sends for each secret only whether it is saved and its hint (FR-SECRET-001).
3. The Claude entry states, in text next to its input, that a token is saved ending in `wxyz`, with the other characters masked (FR-SECRET-002).
4. The LeBoncoin entry states that a session is saved ending in `9f3a` (FR-SECRET-002).
5. Both inputs are empty. Each help text states what may be pasted and that leaving the input empty keeps the saved secret; neither says that an empty save removes it (FR-SECRET-003).
6. Each entry offers its remove action ("Remove token", "Remove session") (FR-SECRET-006, see UC-SECRET-004).
7. The operator closes the modal with Cancel.
8. The page and browser storage hold no full secret (FR-SECRET-011).

## Alternatives and errors

- **No secret saved:** the entry states that no token (or no session) is saved, its input is empty and it has no remove action (FR-SECRET-002, FR-SECRET-006).
- **Short secret:** a secret shorter than the threshold is shown as saved without any of its characters (FR-SECRET-002).
- **Session renewed, expired or revoked, or a secret changed from another browser:** reopening Settings shows the hint and state of the currently stored secret; an expired or revoked session's warning advises capturing a new session or removing it, with the hint of the stored value if one remains (FR-SECRET-009).
- **Warnings:** a rejected-token or rejected/failed-session warning shows next to its entry as before, beside the hint.
- **Load failure:** no hint, input, remove action or Save is usable; Cancel stays available (FR-LBC-SET-003, FR-CLT-SET-002).
- **Keyboard, screen reader, about 400 px:** each hint is text associated with its entry and read by assistive technology; hints and actions are usable without horizontal scrolling (FR-SECRET-013).

## Postconditions

- No server response contained T or S; the operator learnt only whether each is saved and its last 4 characters.
- Nothing changed on the server.
