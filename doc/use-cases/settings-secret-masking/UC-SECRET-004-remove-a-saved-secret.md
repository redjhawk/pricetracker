# UC-SECRET-004 — Remove a saved secret

**Primary actor:** Operator
**Supporting systems:** PriceFollower application, Settings modal
**Trigger:** The operator wants to stop using the saved Claude token or LeBoncoin session.
**Origin:** Issue #265: an empty input now keeps the secret, so removal becomes an explicit action that cannot be confused with "unchanged".
**Covers:** FR-SECRET-004, FR-SECRET-006, FR-SECRET-013

## Preconditions

- The operator has a saved Claude token T and a saved LeBoncoin session S.
- Settings is open and loaded (UC-SECRET-001).

## Main flow

1. The operator selects "Remove token" on the Claude entry.
2. The entry states in text that the token will be removed on Save; its input cannot be used and an undo action is offered (FR-SECRET-006).
3. The operator leaves the LeBoncoin input empty and selects Save.
4. T is removed without any Claude verification; S is untouched (FR-SECRET-004, FR-SECRET-006).
5. The modal closes.
6. The operator reopens Settings: the Claude entry states that no token is saved and has no remove action; the LeBoncoin hint is unchanged.
7. Items show "Configure a Claude token in Settings" (FR-LBC-AIR-008).

## Alternatives and errors

- **Remove the session:** "Remove session" then Save removes S; LeBoncoin collection becomes sessionless (FR-LBC-COL-011) and reopening states that no session is saved. The "session changed" check applies (see UC-SECRET-005).
- **Undo before saving:** the operator chooses "Remove session", undoes it and saves with an empty input; S is unchanged (FR-SECRET-006).
- **Cancel after requesting removal:** the removal is discarded; the secret stays saved (FR-SECRET-006, FR-LBC-SET-011).
- **Empty input:** emptying an input never removes a secret; only the remove action does (FR-SECRET-006).
- **No secret saved:** the entry has no remove action (FR-SECRET-006).
- **Save fails** (other entry invalid, Claude refusal, "session changed", server error): nothing is saved and the requested removal is kept so the operator can retry.
- **Keyboard and screen reader:** the remove action and its undo are reachable by keyboard and their accessible names name the secret; the requested removal is stated in text, not only by colour (FR-SECRET-013).

## Postconditions

- The removed secret is no longer stored; the other secret is unchanged.
