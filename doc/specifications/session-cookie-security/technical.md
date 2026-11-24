# Technical specification: Secure session cookie behind an HTTPS reverse proxy

Status: ready (issue #268)
Functional specification: [functional.md](functional.md) (FR-COOKIE-001..006).
Test cases: [testcases.md](../../changes/2026-11-24-0159-issue-268-session-cookie-set-secure-when-the-appli/testcases.md) (TC-001..TC-011).
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "Users and login", "Modes, session and access".

## Requirement mapping

| Technical ID | Functional IDs | Technical solution | Files |
| --- | --- | --- | --- |
| TS-COOKIE-001 | FR-COOKIE-001, -003, -004 | `secureRequest(r)`: `true` when `r.TLS != nil` or the forwarded protocol states HTTPS; used for the cookie's `Secure` field. | `internal/httpapi/auth.go` |
| TS-COOKIE-002 | FR-COOKIE-002 | Forwarded protocol rule: first `X-Forwarded-Proto` header value (`r.Header.Get`), its first comma-separated entry, trimmed, compared with `https` by `strings.EqualFold`. | `internal/httpapi/auth.go` |
| TS-COOKIE-003 | FR-COOKIE-005 | Logout keeps calling `setSessionCookie(w, r, "", -1)`, so the clearing cookie follows the same rule. | `internal/httpapi/auth.go` (no extra change) |
| TS-COOKIE-004 | FR-COOKIE-006 | Only the `Secure` expression of the `http.Cookie` literal changes; name, `Path`, `MaxAge`, `HttpOnly`, `SameSite` stay. | `internal/httpapi/auth.go` |
| TS-COOKIE-005 | FR-COOKIE-001 (trust) | Header trusted as sent, like `CF-Connecting-IP`; no proxy allow-list. Documented in `deploy/README.md`. | `deploy/README.md` |

## Backend

### HTTP (`internal/httpapi/auth.go`)

- New helper, next to `clientAddress`:
  ```go
  // secureRequest reports whether the request reached the application over HTTPS: directly over TLS,
  // or through the reverse proxy, whose first X-Forwarded-Proto entry is https (trusted like CF-Connecting-IP).
  func secureRequest(r *http.Request) bool {
  	if r.TLS != nil {
  		return true
  	}
  	proto, _, _ := strings.Cut(r.Header.Get("X-Forwarded-Proto"), ",")
  	return strings.EqualFold(strings.TrimSpace(proto), "https")
  }
  ```
- `setSessionCookie`: `Secure: secureRequest(r)` instead of `Secure: r.TLS != nil`. Login (success only) and logout already call it; error paths (`400 INVALID_JSON`, `401`, `429`) never reach it, so they still set no cookie (TC-011).
- Rule outcomes (TS-COOKIE-002): `r.Header.Get` returns the first header only, so with several headers only the first counts (TC-007). `HTTPS`, ` https `, `https, http`, ` HTTPS , http` → Secure; missing, empty, blank, `http`, `ftp`, `http, https`, `httpsx`, `https;`, `xhttps` → not Secure. `strings.TrimSpace` also trims tabs; the functional "surrounding spaces" is satisfied.
- Direct TLS wins whatever the header says (FR-COOKIE-003).

### Validation, persistence, migrations, concurrency

- No validation error: an unrecognised header value only means "not HTTPS".
- No schema change, migration or stored state; existing sessions keep their cookie until it expires or the person logs out.
- Pure function of the request; no shared state.

## API

- No endpoint, request, response body or error changes. Only the `Secure` attribute of the `pricefollower_session` `Set-Cookie` of `POST /api/v1/auth/login` (200) and `POST /api/v1/auth/logout` (204) depends on the request. Updated in `API_SPECIFICATION.md` ("Modes, session and access").
- Compatibility: development over plain HTTP without the header is unchanged; browsers behind Cloudflare receive a `Secure` cookie, which they send over HTTPS as before.

## Frontend

Not affected: the frontend never reads the cookie (HttpOnly) and has no screen, message or state change.

## Files to change

- `internal/httpapi/auth.go`: `secureRequest`, `setSessionCookie`.
- `deploy/README.md` ("Server setup"): extend the Cloudflare sentence: the session cookie is marked `Secure` when the request carries `X-Forwarded-Proto: https` (set by Cloudflare), which is another reason the server must only be reachable through Cloudflare.
- Already done in this stage: `API_SPECIFICATION.md`.

## Tests

All in a new file `internal/httpapi/session_cookie_security_test.go` (package `httpapi`), test names prefixed `TestSessionCookie_TC0NN_...`.

| Test case | Notes |
| --- | --- |
| TC-001, TC-002, TC-003, TC-004, TC-005, TC-007, TC-011 | `newSettingsServer` plus a regular user `alice` created through the service (protected mode); `httptest.NewRequest` login with `req.TLS = &tls.ConnectionState{}` when stated and `req.Header.Add("X-Forwarded-Proto", v)` per header; a helper returns the session `*http.Cookie` and its raw `Set-Cookie` line from `recorder.Result()`; TC-011 reuses `assertError` and `hasSessionCookie` (any `Set-Cookie` named `sessionCookie` counts as a failure). |
| TC-006 | each row logs in, then posts logout with the cookie (`req.AddCookie`) and the row's header/TLS; checks empty value, `MaxAge < 0` parsed / `Max-Age=0` raw, `HttpOnly`, `SameSite=Lax`, `Path=/`, `Secure`; corner cases: logout without cookie, repeated logout. |
| TC-008, TC-009, TC-010 | `httptest.NewServer(server.Handler())` with `http.Client{Jar: cookiejar.New(nil)}`; requests built with `http.NewRequest` to set `X-Forwarded-Proto`; reload is `GET /api/v1/auth/session` decoded with the `user` field. TC-010 proxy setup uses a client without jar and replays the cookie manually with `req.AddCookie`, dropping it when the logout response clears it. |

Existing tests (`auth_test.go`, `login_attempt_limits_test.go`) stay unchanged: they send no header over plain HTTP, so the cookie stays not Secure. No e2e test (see test cases).

## Refactoring

None.

## Manual follow-ups

None.
