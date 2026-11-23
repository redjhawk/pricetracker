# UC-LOGIN-LIM-001 — Locked out after repeated failed logins from one client

**Primary actor:** Person logging in (regular user, administrator, or anyone sending login attempts)
**Supporting systems:** Cloudflare (sends `CF-Connecting-IP`), PriceFollower application, login screen
**Trigger:** The actor submits the login form with a wrong password several times from the same client address.
**Origin:** Issue #266: failed logins locked the username for everyone; now only the failing client is refused, without a password check.
**Covers:** FR-LOGIN-LIM-001, FR-LOGIN-LIM-002, FR-LOGIN-LIM-004, FR-LOGIN-LIM-005, FR-LOGIN-LIM-006, FR-LOGIN-LIM-011

## Preconditions

- The application is in protected mode and the login screen is displayed.
- The actor's client address (A) has no recent failure and no active lock for the username (for example `alice`).

## Main flow

1. The actor submits `alice` with a wrong password from address A.
2. The login screen shows "Incorrect username or password"; the failure is counted for A/`alice` only (FR-LOGIN-LIM-001).
3. The actor repeats steps 1–2 until A/`alice` has 5 consecutive failures.
4. The actor submits `alice` again from A within 1 minute of the 5th failure, with the right password.
5. The application refuses the attempt at once, without a password check, with HTTP 429 (FR-LOGIN-LIM-002, FR-LOGIN-LIM-004).
6. The login screen shows "Too many failed attempts. Wait one minute and try again." in its "Could not log in" error, and the actor stays logged out (FR-LOGIN-LIM-011).
7. The actor waits until 1 minute after the 5th failure, then submits `alice` with the right password from A.
8. The actor is logged in; A/`alice` failures are forgotten (FR-LOGIN-LIM-006).

## Alternatives and errors

- **Unknown username:** failures for a username that does not exist are counted and lock its pair the same way (FR-LOGIN-LIM-001).
- **Same username with different letter case:** `Alice` and `alice` are the same pair; their failures add up.
- **Success before the 5th failure:** after 4 failures, a login with the right password succeeds and the count of A/`alice` restarts at zero; 4 more failures do not lock it (FR-LOGIN-LIM-005).
- **Further attempts while locked:** each one is refused as in steps 5–6, is not counted as a failure and does not extend the lock (FR-LOGIN-LIM-004).
- **Locked attempt while another login is being checked:** the refusal is answered before that other password check ends and does not delay it (FR-LOGIN-LIM-004).
- **Attempt already waiting when the pair becomes locked:** it is still checked; a wrong password leaves the lock unchanged, the right one logs in and restarts the count.
- **Failures after the lock ends:** 5 new consecutive failures are needed to lock A/`alice` again; 4 failures then the right password log in (FR-LOGIN-LIM-006).
- **Old failures:** failures of A/`alice` followed by no other failure for 1 hour no longer count; 4 old and 4 new failures do not lock the pair (FR-LOGIN-LIM-006).
- **Login cancelled while waiting for its password check:** it is not counted as a failure.

## Postconditions

- Only A/`alice` was refused; no other pair's count changed.
- No password check ran for refused attempts.
- Failures are kept only for pairs with a recent failure or an active lock.
