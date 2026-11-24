# UC-SECRET-007 — Save with a long whitespace-only input

**Primary actor:** Operator
**Supporting systems:** PriceFollower application, Settings modal
**Trigger:** The operator saves Settings while an input holds only whitespace (spaces, tabs, line breaks), longer than that entry's maximum input length, for example after pasting a large blank selection.
**Origin:** Issue #286: a whitespace-only input must keep the saved secret whatever its length, and must never be rejected as too long.
**Covers:** FR-SECRET-014, FR-SECRET-004, FR-SECRET-007, FR-SECRET-008, FR-LBC-SET-006, FR-CLT-SET-004

## Preconditions

- The operator has a saved LeBoncoin session S (with its latest attempt and warning if any) and a saved Claude token T (with its rejection warning if any).
- Settings is open and loaded, both inputs empty (UC-SECRET-001).

## Main flow

1. The operator fills the LeBoncoin input with more than 8192 spaces and no other character, leaves the Claude input empty and requests no removal.
2. The operator selects Save.
3. The LeBoncoin entry is untouched (FR-SECRET-014, FR-SECRET-004): no "too long" or other message is shown, S keeps its value, renewals, expiry or revocation state, latest attempt and warning.
4. The Claude entry is untouched: T is not verified again and keeps its warning (FR-SECRET-004).
5. The modal closes and the typed input is discarded.
6. The operator reopens Settings: both inputs are empty and the hints and warnings of S and T are unchanged. LeBoncoin checks keep using S.

## Alternatives and errors

- **Long whitespace-only Claude input:** the Claude input holds more than 1024 spaces and no other character; the save succeeds without a "too long" message, T is unchanged and not verified again (FR-SECRET-014).
- **Both inputs whitespace only:** Save changes nothing and closes the modal, as with both inputs empty (FR-SECRET-004).
- **Whitespace-only LeBoncoin input with a new Claude token:** the token is verified and replaces T as in UC-SECRET-003; S is untouched (FR-SECRET-004, FR-SECRET-005).
- **Session changed since the modal loaded:** S was renewed to Y after the modal opened; the save with the whitespace-only LeBoncoin input still succeeds without the "session changed" warning and the session stays Y (FR-SECRET-008).
- **Another modal opened before this save:** it can still replace S afterwards without the "session changed" warning, since S did not change (FR-SECRET-014).
- **Too long and not whitespace only:** an input longer than the maximum that contains any other character is still refused as before, with a message on the entry; nothing is saved and the input is kept (FR-LBC-SET-006, FR-CLT-SET-004).
- **Save request too large to be accepted at all:** the save is refused and nothing is saved, as for any other input (FR-SECRET-014).
- **Cancel:** the typed input is discarded; S and T stay saved.

## Postconditions

- The LeBoncoin session is still S with the same state; the Claude token is still T with the same warning.
