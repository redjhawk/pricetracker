# UC-COOKIE-002 — Log in over plain HTTP in development

**Primary actor:** Person logging in (regular user or administrator), typically a developer
**Supporting systems:** Browser, PriceFollower application (no reverse proxy)
**Trigger:** The actor submits the login form on a plain `http://` address, for example `http://localhost`.
**Origin:** Issue #268: plain HTTP access in development must keep working after the cookie becomes `Secure` behind the proxy.
**Covers:** FR-COOKIE-002, FR-COOKIE-004, FR-COOKIE-006

## Preconditions

- The application is in protected mode and reached directly over plain HTTP.
- The requests carry no `X-Forwarded-Proto` header, or one that does not state HTTPS.
- The actor has an account and is logged out; the login screen is displayed.

## Main flow

1. The actor enters a valid username and password and submits the login form.
2. The application logs the actor in and answers a `Set-Cookie` for the session cookie without `Secure` (FR-COOKIE-004).
3. The cookie's other attributes are the same as for a login through the HTTPS proxy: name, `HttpOnly`, `SameSite=Lax`, path `/`, 30-day lifetime (FR-COOKIE-006).
4. The browser stores the cookie and sends it with the next plain HTTP requests.
5. The actor reloads the page; the application is displayed, not the login screen.

## Alternatives and errors

- **Header not stating HTTPS:** the request carries `X-Forwarded-Proto` with `http`, an empty value, `ftp` or `http, https`; the outcome is the same as the main flow (FR-COOKIE-002).
- **Header stating HTTPS over plain HTTP:** a client sends `X-Forwarded-Proto: https` directly to the application over plain HTTP; it gets a `Secure` cookie that its browser does not send back over HTTP, so that client stays logged out (FR-COOKIE-001, FR-COOKIE-002).
- **Wrong credentials or login refused:** no session cookie is set; login and login attempt limits behave as today.

## Postconditions

- The actor is logged in over plain HTTP and stays logged in across reloads.
- No screen, message or setting changed.
