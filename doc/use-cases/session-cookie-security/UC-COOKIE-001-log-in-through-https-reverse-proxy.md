# UC-COOKIE-001 — Log in through the HTTPS reverse proxy

**Primary actor:** Person logging in (regular user or administrator)
**Supporting systems:** Browser, Cloudflare reverse proxy (ends HTTPS, forwards plain HTTP with `X-Forwarded-Proto`), PriceFollower application
**Trigger:** The actor submits the login form on the production address `https://…`.
**Origin:** Issue #268: behind the proxy the session cookie was never `Secure` because the application itself receives plain HTTP.
**Covers:** FR-COOKIE-001, FR-COOKIE-002, FR-COOKIE-003, FR-COOKIE-006

## Preconditions

- The application is in protected mode and reached through the reverse proxy over HTTPS.
- The proxy forwards each request to the application over plain HTTP with `X-Forwarded-Proto: https`.
- The actor has an account and is logged out; the login screen is displayed.

## Main flow

1. The actor enters a valid username and password and submits the login form.
2. The browser sends the login over HTTPS; the proxy forwards it over plain HTTP with `X-Forwarded-Proto: https`.
3. The application logs the actor in and answers a `Set-Cookie` for the session cookie that includes `Secure` (FR-COOKIE-001).
4. The cookie keeps its other attributes unchanged: name, `HttpOnly`, `SameSite=Lax`, path `/`, 30-day lifetime (FR-COOKIE-006).
5. The browser stores the cookie and sends it with the next HTTPS requests; the application is displayed and stays displayed after a reload.

## Alternatives and errors

- **Header value in another form:** `X-Forwarded-Proto` is `HTTPS`, ` https ` or `https, http`; the cookie is `Secure` as in step 3 (FR-COOKIE-002).
- **Header not stating HTTPS:** `X-Forwarded-Proto` is `http`, empty, `ftp` or `http, https`; the request is not an HTTPS request and the cookie is not `Secure` (FR-COOKIE-002, see UC-COOKIE-002).
- **Several `X-Forwarded-Proto` headers:** only the first header's value is judged (FR-COOKIE-002).
- **Application receives TLS itself:** the login reaches the application over TLS, with no header or with `X-Forwarded-Proto: http`; the cookie is `Secure` as before (FR-COOKIE-003).
- **Wrong credentials or login refused:** no session cookie is set; login and login attempt limits behave as today.
- **Session logged in before the change:** the existing cookie keeps working until it expires or the actor logs out; the next login through the proxy gives a `Secure` cookie.

## Postconditions

- The actor is logged in with a session cookie marked `Secure`, which the browser sends only over HTTPS.
- No screen, message or setting changed.
