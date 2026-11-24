# Test cases: Secure session cookie behind an HTTPS reverse proxy (issue #268)

Sources: functional specification [session-cookie-security](../../specifications/session-cookie-security/functional.md) (FR-COOKIE-001 to FR-COOKIE-006) and use cases [UC-COOKIE-001](../../use-cases/session-cookie-security/UC-COOKIE-001-log-in-through-https-reverse-proxy.md), [UC-COOKIE-002](../../use-cases/session-cookie-security/UC-COOKIE-002-log-in-over-plain-http-in-development.md) and [UC-COOKIE-003](../../use-cases/session-cookie-security/UC-COOKIE-003-log-out-behind-proxy-or-over-plain-http.md).

Conventions:

- **HTTP handler tests** (`internal/httpapi`, package-internal) run `server.Handler()` on a real service and SQLite database (`newSettingsServer`) in protected mode, with user `alice` / `correct horse battery`. They send requests with `httptest.NewRequest` (plain HTTP, `req.TLS == nil`) and, when stated, set `req.TLS = &tls.ConnectionState{}` (direct TLS) and the `X-Forwarded-Proto` header (`Header.Add` once per header for multi-header cases).
- **Session cookie** is the `Set-Cookie` with the name `sessionCookie`. "Secure" means the parsed cookie has `Secure == true` and the raw header contains `; Secure`; "not Secure" means neither.
- **Browser-like flow tests** (`internal/httpapi`) run `httptest.NewServer(server.Handler())` (plain HTTP) with an `http.Client` using `net/http/cookiejar`, which, like a browser, does not send a `Secure` cookie over plain HTTP. "Reload" means `GET /api/v1/auth/session` with that client: it answers `user` set (application displayed) or `user: null` (login screen in protected mode).
- **E2E:** no Playwright case. The change has no screen change, and the existing specs mock the API with `page.route`, so they cannot observe the cookie attributes set by the Go server; the browser-like flow tests prove each user flow end to end over real HTTP.
- Existing login, logout and login attempt limit tests stay unchanged and keep passing (no `Secure` over plain HTTP without the header).

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-COOKIE-001 | TC-001, TC-007, TC-009 |
| FR-COOKIE-002 | TC-002, TC-003, TC-006, TC-009 |
| FR-COOKIE-003 | TC-004, TC-006 |
| FR-COOKIE-004 | TC-003, TC-008 |
| FR-COOKIE-005 | TC-006, TC-010 |
| FR-COOKIE-006 | TC-005, TC-006 |

| Use case | Test cases |
| --- | --- |
| UC-COOKIE-001 | TC-001, TC-002, TC-003, TC-004, TC-005, TC-007, TC-011 |
| UC-COOKIE-002 | TC-003, TC-005, TC-008, TC-009, TC-011 |
| UC-COOKIE-003 | TC-006, TC-010 |

## Cases

### TC-001 — Login over plain HTTP with `X-Forwarded-Proto: https` sets a Secure cookie

- Covers: FR-COOKIE-001; UC-COOKIE-001
- Level: integration (HTTP handler, real SQLite)
- Given protected mode and a plain HTTP request (`req.TLS == nil`)
- When `POST /api/v1/auth/login` with valid credentials and `X-Forwarded-Proto: https`
- Then 200, and the session cookie has a non-empty value and is Secure.

### TC-002 — Header values that state HTTPS

- Covers: FR-COOKIE-002; UC-COOKIE-001
- Level: integration (HTTP handler, real SQLite; table test, one login per row)
- Given a plain HTTP login request with valid credentials
- When `X-Forwarded-Proto` is, per row: `HTTPS`, `Https`, ` https ` (surrounding spaces), `https, http`, `https,http`, ` HTTPS , http`
- Then each login answers 200 with a Secure session cookie.
- Corner cases: letter case, surrounding spaces, several comma-separated entries with `https` first.

### TC-003 — Header values that do not state HTTPS, and no header

- Covers: FR-COOKIE-002, FR-COOKIE-004; UC-COOKIE-001, UC-COOKIE-002
- Level: integration (HTTP handler, real SQLite; table test, one login per row)
- Given a plain HTTP login request with valid credentials
- When, per row: no `X-Forwarded-Proto` header; `http`; `HTTP`; empty value; only spaces; `ftp`; `http, https`; `httpsx`; `https;` ; `xhttps`
- Then each login answers 200 with a session cookie that is not Secure.
- Corner cases: missing header, empty and blank values, unknown text, `https` only in a later entry, values that merely contain `https`.

### TC-004 — Direct TLS gives a Secure cookie whatever the header says

- Covers: FR-COOKIE-003; UC-COOKIE-001
- Level: integration (HTTP handler, real SQLite; table test)
- Given a login request with `req.TLS` set and valid credentials
- When, per row: no `X-Forwarded-Proto` header; `X-Forwarded-Proto: http`; `X-Forwarded-Proto: https`; empty value
- Then each login answers 200 with a Secure session cookie.

### TC-005 — Only `Secure` differs between plain HTTP and proxied HTTPS login

- Covers: FR-COOKIE-006; UC-COOKIE-001, UC-COOKIE-002
- Level: integration (HTTP handler, real SQLite)
- Given two successful logins for `alice`: one plain HTTP without the header, one plain HTTP with `X-Forwarded-Proto: https`
- When both session cookies are parsed
- Then both have the same name, a non-empty value, `HttpOnly`, `SameSite=Lax`, `Path=/` and `Max-Age` equal to 30 days (2592000 s); the first is not Secure and the second is Secure.
- And the raw `Set-Cookie` headers are equal once the value and `; Secure` are removed.

### TC-006 — Logout clearing cookie follows the same Secure rule

- Covers: FR-COOKIE-005, FR-COOKIE-002, FR-COOKIE-003, FR-COOKIE-006; UC-COOKIE-003
- Level: integration (HTTP handler, real SQLite; table test, each row logs in first, then logs out with the session cookie)
- Given a logged-in session for `alice`
- When `POST /api/v1/auth/logout` with, per row:
  - plain HTTP, `X-Forwarded-Proto: https` → Secure;
  - plain HTTP, `X-Forwarded-Proto: HTTPS , http` → Secure;
  - plain HTTP, no header → not Secure;
  - plain HTTP, `X-Forwarded-Proto: http, https` → not Secure;
  - plain HTTP, empty `X-Forwarded-Proto` → not Secure;
  - TLS, `X-Forwarded-Proto: http` → Secure
- Then 204, and the clearing session cookie has an empty value, `Max-Age` ≤ 0 (raw header `Max-Age=0`), `HttpOnly`, `SameSite=Lax`, `Path=/`, and the expected `Secure` attribute.
- Corner cases: logout without a session cookie (plain HTTP with `X-Forwarded-Proto: https`) still answers 204 with a Secure clearing cookie; repeated logout of the same session answers the same clearing cookie.

### TC-007 — Several `X-Forwarded-Proto` headers: only the first is judged

- Covers: FR-COOKIE-001, FR-COOKIE-002; UC-COOKIE-001
- Level: integration (HTTP handler, real SQLite)
- Given a plain HTTP login request with valid credentials
- When it carries two `X-Forwarded-Proto` headers, first `https` then `http`; and, in a second login, first `http` then `https`
- Then the first login's session cookie is Secure and the second's is not.

### TC-008 — Development flow: login over plain HTTP stays logged in after reload

- Covers: FR-COOKIE-004; UC-COOKIE-002
- Level: integration (browser-like flow: `httptest.NewServer`, cookie jar, real SQLite)
- Given protected mode, the application reached over plain `http://` with no `X-Forwarded-Proto` header, and the client logged out (`GET /api/v1/auth/session` answers `user: null`)
- When the client logs in as `alice` and then reloads (`GET /api/v1/auth/session`)
- Then the login `Set-Cookie` has no `Secure`, and the reload answers `user` `alice` (application displayed, not the login screen).
- Corner cases: a second reload still answers `alice`; with `X-Forwarded-Proto: http` on every request the outcome is the same; wrong password answers 401 `INVALID_CREDENTIALS`, sets no session cookie, and the reload answers `user: null`.

### TC-009 — Header stating HTTPS sent directly over plain HTTP harms only that client

- Covers: FR-COOKIE-001, FR-COOKIE-002; UC-COOKIE-002
- Level: integration (browser-like flow: `httptest.NewServer`, cookie jar, real SQLite)
- Given protected mode and the application reached directly over plain `http://`
- When the client logs in as `alice` with `X-Forwarded-Proto: https` and then reloads over plain HTTP
- Then the login answers 200 with a Secure session cookie, the jar does not send it back, and the reload answers `user: null`.
- And a second client logging in over plain HTTP without the header stays logged in after reload (the first client's request does not affect it).

### TC-010 — Logout flow removes the cookie behind the proxy and over plain HTTP

- Covers: FR-COOKIE-005; UC-COOKIE-003
- Level: integration (browser-like flow: `httptest.NewServer`, real SQLite)
- Given protected mode and `alice` logged in
- When:
  - development: over plain HTTP without the header, using a cookie jar, the client logs out and reloads;
  - proxy: every request carries `X-Forwarded-Proto: https` and the client replays the session cookie manually (as a browser on HTTPS would), logs out, then reloads with whatever cookie the logout response left (none once cleared)
- Then in both setups logout answers 204 with a clearing session cookie (Secure in the proxy setup only), the jar or replay holds no session cookie afterwards, and the reload answers `user: null` (login screen).
- Corner cases: replaying the old session token after logout answers `user: null` (session ended server side).

### TC-011 — Refused login sets no cookie, whatever the header

- Covers: UC-COOKIE-001, UC-COOKIE-002 (alternative "wrong credentials or login refused")
- Level: integration (HTTP handler, real SQLite)
- Given protected mode
- When a plain HTTP login with `X-Forwarded-Proto: https` sends a wrong password; and a request with an invalid JSON body is sent with the same header
- Then the answers are 401 `INVALID_CREDENTIALS` and 400 `INVALID_JSON` as today, with no session `Set-Cookie`.
