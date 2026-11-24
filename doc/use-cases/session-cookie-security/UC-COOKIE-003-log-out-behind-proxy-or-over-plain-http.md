# UC-COOKIE-003 — Log out behind the HTTPS proxy or over plain HTTP

**Primary actor:** Logged-in person (regular user or administrator)
**Supporting systems:** Browser, Cloudflare reverse proxy (production only), PriceFollower application
**Trigger:** The actor logs out from the application header menu.
**Origin:** Issue #268: the response that clears the session cookie must follow the same `Secure` rule as login so the browser removes the cookie in both setups.
**Covers:** FR-COOKIE-005, FR-COOKIE-006

## Preconditions

- The application is in protected mode.
- The actor is logged in with a session cookie set as in UC-COOKIE-001 (through the HTTPS proxy) or UC-COOKIE-002 (over plain HTTP).

## Main flow

1. The actor chooses to log out.
2. The browser sends the logout over HTTPS; the proxy forwards it over plain HTTP with `X-Forwarded-Proto: https`.
3. The application ends the session and answers a `Set-Cookie` that clears the session cookie and includes `Secure` (FR-COOKIE-005).
4. The clearing cookie keeps the same name, `HttpOnly`, `SameSite=Lax` and path `/` as at login (FR-COOKIE-006).
5. The browser removes the session cookie; the login screen is displayed and stays displayed after a reload.

## Alternatives and errors

- **Plain HTTP in development:** the logout reaches the application over plain HTTP without a header stating HTTPS; the clearing `Set-Cookie` has no `Secure`; the browser removes the cookie and reloading shows the login screen (FR-COOKIE-005).
- **Application receives TLS itself:** the clearing `Set-Cookie` includes `Secure`, whatever `X-Forwarded-Proto` says (FR-COOKIE-005, FR-COOKIE-003).
- **Header in another form:** the header is judged as at login (`HTTPS`, ` https `, `https, http` state HTTPS; `http`, empty, `ftp`, `http, https` do not) (FR-COOKIE-005, FR-COOKIE-002).

## Postconditions

- The session is ended and the browser no longer holds the session cookie.
- The actor sees the login screen.
