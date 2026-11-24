# Functional specification: Secure session cookie behind an HTTPS reverse proxy

Status: FR-COOKIE-001–006 implemented (issue #268); FR-COOKIE-007–009 implemented (issue #293, maintenance: no behaviour change)
Owner: functional specification agent
User decision/reference: GitHub issue #268 ("Session cookie: set Secure when the application is served over HTTPS by a reverse proxy"); GitHub issue #293 ("Restore the clientAddress doc comment in auth.go", from review comment R1-01 on PR #292).

## Purpose and scope

The login session cookie is marked `Secure` (sent by the browser only over HTTPS) only when the application itself receives the HTTPS connection. In production HTTPS ends at the reverse proxy (Cloudflare), which forwards plain HTTP to the application, so the cookie is never `Secure`. The user asks that the cookie be `Secure` when the request reached the proxy over HTTPS, as stated by the proxy's `X-Forwarded-Proto: https` header, while plain HTTP access in development keeps working.

Issue #293 (maintenance): the issue #268 change inserted `secureRequest` (the function that decides whether a request is an HTTPS request) and its comment between the doc comment of `clientAddress` (the client address of login limits, in `internal/httpapi/auth.go`) and `func clientAddress`. The doc comment of `secureRequest` now starts with the `clientAddress` text, and `clientAddress` has none. The comment lines are moved back; no behaviour changes.

Actors: a person logging in or out (regular user or administrator), the browser, and the reverse proxy in front of the application.

Included: the `Secure` attribute of the session cookie the application sets at login and clears at logout.

Excluded: the cookie's other attributes (name, `HttpOnly`, `SameSite=Lax`, path `/`, 30-day lifetime), login, logout, sessions, open and protected modes ([multi-user-login](../multi-user-login/functional.md)) and login attempt limits ([login-attempt-limits](../login-attempt-limits/functional.md)), which stay unchanged; redirecting HTTP to HTTPS; HSTS or other security headers; any other forwarded header (`Forwarded`, `X-Forwarded-Host`, `X-Forwarded-For`); any screen change.

Terms:

- **HTTPS request:** a request the application receives over TLS itself, or a request whose `X-Forwarded-Proto` header states `https` (FR-COOKIE-002).
- **Session cookie:** the cookie the application sets at a successful login and clears at logout.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-COOKIE-001 | A login succeeds on a plain HTTP request carrying `X-Forwarded-Proto: https`. | The session cookie set in the response is `Secure`. | A successful login over plain HTTP with `X-Forwarded-Proto: https` answers a `Set-Cookie` for the session cookie that includes `Secure`. |
| FR-COOKIE-002 | The `X-Forwarded-Proto` header is present. | It states HTTPS when its value, or its first comma-separated entry when it lists several, is `https` ignoring letter case and surrounding spaces. Any other value (`http`, empty, unknown text) does not state HTTPS. | `HTTPS`, ` https `, and `https, http` give a `Secure` cookie; `http`, an empty value, `ftp`, and `http, https` do not. |
| FR-COOKIE-003 | A login succeeds on a request the application receives over TLS itself. | The session cookie is `Secure`, whatever the `X-Forwarded-Proto` header says (unchanged behaviour). | A successful login over TLS, with no header or with `X-Forwarded-Proto: http`, answers a `Secure` session cookie. |
| FR-COOKIE-004 | A login succeeds on a plain HTTP request that is not an HTTPS request (no `X-Forwarded-Proto`, or one that does not state HTTPS), for example in development. | The session cookie is not `Secure`, so the browser keeps and sends it over plain HTTP and the person stays logged in. | In development over `http://`, after logging in, reloading the page shows the application, not the login screen; the `Set-Cookie` has no `Secure`. |
| FR-COOKIE-005 | A logged-in person logs out. | The response that clears the session cookie is `Secure` under the same rule as login (FR-COOKIE-001 to 004), so the browser removes the cookie both over HTTPS behind the proxy and over plain HTTP. | Logout with `X-Forwarded-Proto: https` answers a clearing `Set-Cookie` with `Secure`; logout over plain HTTP without the header answers one without `Secure`; in both cases reloading shows the login screen (protected mode). |
| FR-COOKIE-006 | Any session cookie the application sets or clears. | Only the `Secure` attribute depends on the request; name, value, `HttpOnly`, `SameSite=Lax`, path `/` and lifetime (30 days at login, removal at logout) are as today. | The `Set-Cookie` of a login over plain HTTP and of a login with `X-Forwarded-Proto: https` differ only by `Secure`. |

Issue #293 (maintenance, no behaviour change):

- **FR-COOKIE-007** In `internal/httpapi/auth.go`, the two comment lines describing `clientAddress` ("clientAddress returns the client address of login limits: …" and "… else the connection host.") stand directly above `func clientAddress`, with no blank line between them and the function, so they are its doc comment.
- **FR-COOKIE-008** The doc comment of `secureRequest` holds only its own two lines ("secureRequest reports whether the request reached the application over HTTPS: …" and "… (trusted like CF-Connecting-IP)."), and one blank line separates the end of `secureRequest` from the `clientAddress` doc comment.
- **FR-COOKIE-009** The change's diff outside `doc/` only moves those comment lines in `internal/httpapi/auth.go`: no code line, comment wording or other file changes. FR-COOKIE-001–006 and the login attempt limits behave as before, the unit-test gate (`go vet ./...`, `go test ./...`) passes, and the operator observes no change in the application.

## States and corner cases

- The header is trusted as sent: the application is reachable in production only through Cloudflare. A client reaching the application directly and sending `X-Forwarded-Proto: https` over plain HTTP only gets a `Secure` cookie its browser will not send back over HTTP, so it harms only that client.
- Sessions logged in before the change keep working with their existing cookie until it expires or the person logs out; the cookie becomes `Secure` at the next login over HTTPS.
- Requests with several `X-Forwarded-Proto` headers are judged by the first header's value.
- No new screen, message, setting or loading state.

## Acceptance criteria summary

- Behind the HTTPS reverse proxy (`X-Forwarded-Proto: https`), the session cookie set at login and cleared at logout is `Secure`.
- Direct TLS still gives a `Secure` cookie.
- Plain HTTP without a header stating HTTPS gives a cookie without `Secure`, and login keeps working in development.
- No other cookie attribute changes.

Issue #293:

- `go doc -u ./internal/httpapi clientAddress` shows the doc comment starting "clientAddress returns the client address of login limits" (FR-COOKIE-007).
- The doc comment directly above `func secureRequest` starts with "secureRequest reports" and does not contain "clientAddress" (FR-COOKIE-008).
- `git diff origin/master -- internal/httpapi/auth.go` only moves the two `clientAddress` comment lines and changes no code line; `git diff --name-only origin/master` outside `doc/` lists only `internal/httpapi/auth.go`; `gofmt -l internal/httpapi` prints nothing; `go vet ./...` and `go test ./...` succeed (FR-COOKIE-009).

## Open questions

None. Issue #293 states the expected result (each function keeps only its own doc comment) and asks for no behaviour change.
