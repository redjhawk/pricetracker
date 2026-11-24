# UC-HTTP-TMO-004 — Save the Claude token while Claude is slow to verify it

**Primary actor:** Operator
**Supporting systems:** Browser, PriceFollower application, Settings modal, Claude (token verification, up to 20 s)
**Trigger:** The operator pastes a Claude token into Settings and saves while Claude is slow to answer the token check.
**Origin:** Issue #273: the new response time limit must leave room for the Claude token verification, which waits up to 20 s.
**Covers:** FR-HTTP-TMO-005, FR-HTTP-TMO-002

## Preconditions

- The operator is logged in and Settings is open and loaded (UC-SECRET-001).
- Claude answers the token check only after about 19 s.

## Main flow

1. The operator pastes a valid token into the Claude input and selects Save; Save shows progress while the token is verified.
2. Claude accepts the token after about 19 s.
3. The application saves the token and answers its usual success response; the 60 s response limit does not interrupt it (FR-HTTP-TMO-005, FR-HTTP-TMO-002).
4. The modal closes; on reopening, Settings shows the Claude token as saved with its masked hint (UC-SECRET-003).

## Alternatives and errors

- **Claude does not answer:** after about 20 s the application answers its usual verification failure, "The token could not be verified because Claude could not be reached. Try again."; nothing is saved and the modal stays open with the input (FR-HTTP-TMO-005).
- **Claude refuses the token:** the usual "Claude refused this token. Create a new one with claude setup-token." message is shown; nothing is saved (FR-HTTP-TMO-005).
- **Claude answers quickly:** the save completes as in UC-SECRET-003.

## Postconditions

- The save ended with its normal outcome (saved, or the existing rejection or failure message); no time limit cut it off.
- The Settings screen and its messages are the same as before this change.
