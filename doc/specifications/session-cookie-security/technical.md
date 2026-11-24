# Technical specification: Secure session cookie behind an HTTPS reverse proxy

Status: TS-COOKIE-001–005 implemented (issue #268); TS-COOKIE-006–008 ready (issue #293, maintenance: no behaviour change)
Functional specification: [functional.md](functional.md) (FR-COOKIE-001..009).
Test cases: issue #268 [testcases.md](../../changes/2026-11-24-0159-issue-268-session-cookie-set-secure-when-the-appli/testcases.md) (TC-001..TC-011); issue #293 [testcases.md](../../changes/2026-11-24-0841-issue-293-restore-the-clientaddress-doc-comment-in/testcases.md) (TC-001..TC-004).
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "Users and login", "Modes, session and access".

## Requirement mapping

| Technical ID | Functional IDs | Technical solution | Files |
| --- | --- | --- | --- |
| TS-COOKIE-001 | FR-COOKIE-001, -003, -004 | `secureRequest(r)`: `true` when `r.TLS != nil` or the forwarded protocol states HTTPS; used for the cookie's `Secure` field. | `internal/httpapi/auth.go` |
| TS-COOKIE-002 | FR-COOKIE-002 | Forwarded protocol rule: first `X-Forwarded-Proto` header value (`r.Header.Get`), its first comma-separated entry, trimmed, compared with `https` by `strings.EqualFold`. | `internal/httpapi/auth.go` |
| TS-COOKIE-003 | FR-COOKIE-005 | Logout keeps calling `setSessionCookie(w, r, "", -1)`, so the clearing cookie follows the same rule. | `internal/httpapi/auth.go` (no extra change) |
| TS-COOKIE-004 | FR-COOKIE-006 | Only the `Secure` expression of the `http.Cookie` literal changes; name, `Path`, `MaxAge`, `HttpOnly`, `SameSite` stay. | `internal/httpapi/auth.go` |
| TS-COOKIE-005 | FR-COOKIE-001 (trust) | Header trusted as sent, like `CF-Connecting-IP`; no proxy allow-list. Documented in `deploy/README.md`. | `deploy/README.md` |
| TS-COOKIE-006 | FR-COOKIE-007 | The two `clientAddress` comment lines move, unchanged, to directly above `func clientAddress` (issue #293). | `internal/httpapi/auth.go` |
| TS-COOKIE-007 | FR-COOKIE-008 | The comment block above `func secureRequest` keeps only its own two lines; one blank line separates the closing `}` of `secureRequest` from the `clientAddress` comment (issue #293). | `internal/httpapi/auth.go` |
| TS-COOKIE-008 | FR-COOKIE-009 | No code line, other comment or other file outside `doc/` changes; no test file is added (proved by repository checks and the existing suites, issue #293). | none |

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

## Issue #293: restore the `clientAddress` doc comment

Maintenance only: two source comment lines move; no behaviour changes.

### Backend (`internal/httpapi/auth.go`)

On `master`, between `allowMethod` and `func clientAddress`, the file holds: a blank line, the two `clientAddress` lines, the two `secureRequest` lines, `func secureRequest … }`, a blank line. Go takes all four comment lines as the doc comment of `secureRequest`, and `clientAddress` has none.

Cut the two `clientAddress` lines and insert them, byte for byte, between the blank line after `secureRequest` and `func clientAddress`:

```go
// secureRequest reports whether the request reached the application over HTTPS: directly over TLS,
// or through the reverse proxy, whose first X-Forwarded-Proto entry is https (trusted like CF-Connecting-IP).
func secureRequest(r *http.Request) bool {
	…unchanged…
}

// clientAddress returns the client address of login limits: the CF-Connecting-IP header when it is
// a bare IP address (the application is only reachable through Cloudflare), else the connection host.
func clientAddress(r *http.Request) string {
```

- Function order stays `allowMethod`, `secureRequest`, `clientAddress`, `decodeCredentials`. No code line changes, so `git diff --numstat` is `2 2` and `gofmt -l internal/httpapi` stays empty.
- Validation, persistence, migrations, concurrency: not affected (no code change).

### API

Not affected: no endpoint, header, cookie or error changes; `API_SPECIFICATION.md` is unchanged.

### Frontend

Not affected: no frontend file changes.

### Files to change

- `internal/httpapi/auth.go` (development stage): the move above.
- `doc/specifications/session-cookie-security/functional.md` and this file (development stage): statuses of FR-COOKIE-007–009 and TS-COOKIE-006–008 set to implemented.

### Tests

FR-COOKIE-009 allows no change outside `doc/` other than `auth.go`, so no test file is added or changed, and the change (a comment) cannot make a test go red. The **tests stage** writes `<change folder>/no-tests.md` giving that reason and where each TC is proved (table below); the red gate is then skipped. The **development stage** makes the move, runs the checks below and the gates, and restores any other file outside `doc/` that differs from `master`.

| Test case | Proved by |
| --- | --- |
| TC-001 | `go doc -u ./internal/httpapi clientAddress` shows the comment "clientAddress returns the client address of login limits … else the connection host." (none on `master`); `grep -c 'clientAddress returns the client address' internal/httpapi/auth.go` is `1`; the line above the two lines is blank and the line after them is `func clientAddress(r *http.Request) string {`. |
| TC-002 | `go doc -u ./internal/httpapi secureRequest` shows only the two `secureRequest` lines (no "clientAddress returns"); the line above them is blank; exactly one blank line follows the closing `}` of `secureRequest`. |
| TC-003 | `git diff --name-status master -- . ':!doc'` lists only `M internal/httpapi/auth.go`; `git diff --numstat master -- internal/httpapi/auth.go` is `2 2`, with the same two lines removed and added; `gofmt -l internal/httpapi` prints nothing. |
| TC-004 | Existing, unchanged tests in `internal/httpapi/session_cookie_security_test.go` (`TestSessionCookie_TC001`–`TC011`), `internal/httpapi/login_attempt_limits_test.go` and `internal/service/login_attempt_limits_test.go` (`TestLoginLimits_*`): `go test -count=1 -run 'TestSessionCookie_|TestLoginLimits_' ./internal/httpapi/ ./internal/service/`; then the gates `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`. |

### Refactoring

None (the move is the change itself).

### Manual follow-ups

None.
