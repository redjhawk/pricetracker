# UC-LOGIN-LIM-003 — Log in when too many password checks are waiting

**Primary actor:** Person logging in (regular user or administrator)
**Supporting systems:** PriceFollower application, login screen
**Trigger:** The actor submits the login form while many other logins are being checked, for example during a flood of attempts.
**Origin:** Issue #266: password checks run one at a time with no bound on how many wait, so a flood of attempts made every other login wait.
**Covers:** FR-LOGIN-LIM-009, FR-LOGIN-LIM-010, FR-LOGIN-LIM-011

## Preconditions

- The application is in protected mode and the login screen is displayed.
- The actor's pair (client address and username) is not locked.
- A password check is running and the maximum number of password checks are already waiting to run.

## Main flow

1. The actor submits the username with the right password.
2. The application answers at once, without a password check and without waiting, with HTTP 429 (FR-LOGIN-LIM-009).
3. The login screen shows "Too many login attempts are being checked. Try again in a moment." in its "Could not log in" error, and the actor stays logged out (FR-LOGIN-LIM-011).
4. The refusal is not counted as a failure for the actor's pair (FR-LOGIN-LIM-009).
5. Once the waiting checks have finished, the actor submits the same login again.
6. The password is checked and the actor is logged in.

## Alternatives and errors

- **A slot to wait is free:** with fewer than the maximum number of checks waiting (for example one running and none waiting), the login waits its turn, is checked, and succeeds after the running check ends; it is never refused as busy (FR-LOGIN-LIM-010).
- **Repeated busy refusals:** none of them count toward a lock; the actor can retry as often as needed.
- **Wrong password once checked:** the actor sees "Incorrect username or password" and the failure counts for the pair (UC-LOGIN-LIM-001).
- **Pair locked:** the attempt is refused with the locked message instead, before any check or wait (UC-LOGIN-LIM-001); the two refusals are told apart by their messages.
- **While waiting its turn:** the login button behaves as today while a login is being checked.

## Postconditions

- A busy refusal created no session and ran no password check.
- The actor's failure count is unchanged by busy refusals.
