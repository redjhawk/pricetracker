# UC-API-REQ-004 — A page of another site submits the login

**Primary actor:** Page of another site loaded in a browser
**Supporting systems:** Browser, PriceFollower application, login attempt limits
**Trigger:** The other site's page sends `POST /api/v1/auth/login` to the application's address.
**Origin:** Issue #267: a refused request must not log anyone in nor be counted as a failed login.
**Covers:** FR-API-REQ-001, FR-API-REQ-006, FR-API-REQ-007

## Preconditions

- The application is in protected mode at its own address (for example `prices.example`).
- An account exists for the username the page sends.

## Main flow

1. The other site's page makes the browser send `POST /api/v1/auth/login` with the right username and password, `Host: prices.example` and `Origin: https://evil.example`.
2. The application refuses the request before any login handling: `403 ORIGIN_NOT_ALLOWED`, message "Requests from another site are not allowed." (FR-API-REQ-001, FR-API-REQ-007).
3. No password check is made and no session cookie is set.

## Alternatives and errors

- **Wrong password, repeated:** the page sends six such logins with a wrong password; each answers `403 ORIGIN_NOT_ALLOWED`; none counts toward the login attempt limits nor is answered by them (no lockout answer). The next login from the application's own page with the right password succeeds (FR-API-REQ-007).
- **Body not declared as JSON as well:** the answer is still `403 ORIGIN_NOT_ALLOWED` (FR-API-REQ-006).
- **Same login from the application's own page:** it passes both checks and the login and its attempt limits behave as before (see UC-API-REQ-002).

## Postconditions

- No session is created, no password is checked, and no failed login is counted for the username or the client.
