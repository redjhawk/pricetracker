# UC-LOGIN-LIM-002 — Log in while another client is locked out for the same username

**Primary actor:** Real owner of the account (regular user or administrator)
**Supporting systems:** Cloudflare (sends `CF-Connecting-IP`), PriceFollower application, login screen
**Trigger:** The actor logs in while another client keeps sending wrong passwords for the actor's username.
**Origin:** Issue #266: anyone typing wrong passwords for a username kept its real user, or the administrator, out.
**Covers:** FR-LOGIN-LIM-001, FR-LOGIN-LIM-003, FR-LOGIN-LIM-007, FR-LOGIN-LIM-008

## Preconditions

- The application is in protected mode.
- Another client, at address A, has made 5 consecutive failed logins for `alice` within the last minute, so A/`alice` is locked.
- The actor uses a different client address, B, and knows the right password for `alice`.

## Main flow

1. The actor opens the login screen from address B.
2. The actor submits `alice` with the right password.
3. Cloudflare passes the request with `CF-Connecting-IP` B; the application takes B as the client address (FR-LOGIN-LIM-007).
4. The pair B/`alice` is not locked, so the password is checked and the actor is logged in (FR-LOGIN-LIM-003).
5. Attempts from A for `alice` keep being refused until A's lock ends; they never count toward B/`alice` (FR-LOGIN-LIM-001).

## Alternatives and errors

- **Other username from the locked address:** while A/`alice` is locked, `bob` logs in from address A with the right password (FR-LOGIN-LIM-003).
- **Same connection, different header addresses:** two requests through the same connection or proxy with `CF-Connecting-IP` 203.0.113.1 and 203.0.113.2 are different clients; after 5 failures for `alice` with the first address, `alice` logs in with the second (FR-LOGIN-LIM-007).
- **No usable header:** when the request has no `CF-Connecting-IP`, or its value is empty or not a valid IP address (for example `not-an-ip`), the connection's address is the client address; a request whose connection address is locked for `alice` is refused as locked, whatever invalid header it carries (FR-LOGIN-LIM-008).
- **Wrong password from B:** the actor sees "Incorrect username or password"; the failure counts for B/`alice` only (FR-LOGIN-LIM-001; see UC-LOGIN-LIM-001).

## Postconditions

- The actor is logged in from B; A/`alice` stays locked until its minute ends.
- No other pair's failure count changed.
