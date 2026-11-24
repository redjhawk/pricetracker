# UC-SECRET-003 — Replace the Claude token without touching the LeBoncoin session

**Primary actor:** Operator
**Supporting systems:** PriceFollower application, Settings modal, Claude (token verification)
**Trigger:** The operator pastes a new Claude token into Settings and saves.
**Origin:** Issue #265: saving the Claude token must leave the LeBoncoin session exactly as it is.
**Covers:** FR-SECRET-001, FR-SECRET-004, FR-SECRET-005, FR-SECRET-007, FR-SECRET-008, FR-SECRET-011

## Preconditions

- The operator has a saved Claude token T and a saved LeBoncoin session S, with its latest attempt and warning if any.
- Settings is open and loaded, both inputs empty (UC-SECRET-001).

## Main flow

1. The operator pastes a new token U into the Claude input and leaves the LeBoncoin input empty.
2. The operator selects Save; Save shows progress while U is verified.
3. Claude accepts U; it replaces T and the rejection warning of T is cleared (FR-SECRET-005, FR-CLT-SET-004/009).
4. The LeBoncoin entry is untouched: S keeps its value, renewals, expiry or revocation state, latest attempt and warning (FR-SECRET-004). The hint is not sent back as a secret (FR-SECRET-007).
5. The save response contains neither U nor S (FR-SECRET-001); the modal closes and the typed input is discarded (FR-SECRET-011).
6. The operator reopens Settings: both inputs are empty, the Claude hint shows U's last 4 characters, and the LeBoncoin hint and warning are unchanged.

## Alternatives and errors

- **Session changed since the modal loaded:** S was renewed to Y after the modal opened; the save still succeeds without the "session changed" warning and the session stays Y (FR-SECRET-008).
- **Another modal opened before this save:** it can still replace S afterwards without the "session changed" warning, since S did not change (FR-SECRET-004).
- **Same token pasted again:** it is verified again and its rejection warning is cleared (FR-SECRET-005).
- **Token refused or Claude unreachable:** nothing is saved (neither entry), the modal stays open with the input and a message on the Claude entry (FR-CLT-SET-011).
- **Whitespace inside the token:** the save is refused with a message on the entry (FR-CLT-SET-004).
- **Cancel:** the typed input is discarded; T stays saved.

## Postconditions

- The Claude token is U; AI reviews started afterwards use it.
- The LeBoncoin session and its state are unchanged.
