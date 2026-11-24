# UC-SECRET-002 — Replace the LeBoncoin session without touching the Claude token

**Primary actor:** Operator
**Supporting systems:** PriceFollower application, Settings modal
**Trigger:** The operator pastes a newly captured LeBoncoin session into Settings and saves.
**Origin:** Issue #265: saving one secret must not change the other, and the masked hint must never be saved as a secret.
**Covers:** FR-SECRET-001, FR-SECRET-004, FR-SECRET-005, FR-SECRET-007, FR-SECRET-011

## Preconditions

- The operator has a saved Claude token T (with or without a rejection warning) and a saved LeBoncoin session S.
- Settings is open and loaded, both inputs empty (UC-SECRET-001).

## Main flow

1. The operator pastes `datadome=<new value N>` into the LeBoncoin input and leaves the Claude input empty.
2. The operator selects Save.
3. The application extracts N and replaces S with it under the existing rules (FR-SECRET-005, FR-LBC-SET-005/007); the warnings of S are cleared.
4. The Claude entry is untouched: T is not sent, not verified with Claude, and keeps its value and its rejection warning (FR-SECRET-004). The hint is not sent back as a secret (FR-SECRET-007).
5. The save response contains neither T nor N (FR-SECRET-001); the modal closes and the typed input is discarded (FR-SECRET-011).
6. The operator reopens Settings: both inputs are empty, the LeBoncoin hint shows N's last 4 characters, and the Claude hint and warning are unchanged.

## Alternatives and errors

- **Claude cannot be reached:** the save still succeeds, since T is not verified (FR-SECRET-004).
- **Same value pasted again:** a non-empty input is always a replacement; the session's warnings are cleared (FR-SECRET-005).
- **Unusable input:** for example `a=1; b=2` without `datadome`; the save is refused with a message on the entry, nothing is saved and the input is kept (FR-LBC-SET-006).
- **Both inputs empty and no removal requested:** Save changes nothing and closes the modal; reopening shows the same hints and warnings (FR-SECRET-004, FR-SECRET-007).
- **Replace one and remove the other in one Save:** allowed; an error in either saves neither (FR-CLT-SET-006, see UC-SECRET-004).
- **Server or network error:** the modal stays open with the input; nothing is saved (FR-LBC-SET-010).
- **Cancel:** the typed input is discarded; S stays saved (FR-LBC-SET-011).

## Postconditions

- The LeBoncoin session is N; LeBoncoin checks starting afterwards use it.
- The Claude token is still T, with the same warning; AI reviews keep using it.
