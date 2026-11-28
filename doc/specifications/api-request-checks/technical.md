# Technical specification: JSON content type and same-origin checks on state-changing API requests

Status: ready (issue #267)
Functional specification: [functional.md](functional.md) (FR-API-REQ-001..009).
Test cases: [testcases.md](../../changes/2026-11-28-1729-issue-267-api-require-json-content-type-and-same-o/testcases.md) (TC-001..TC-015).
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "Conventions" (bullet "State-changing requests").

## Requirement mapping

| Technical ID | Functional IDs | Technical solution | Files |
| --- | --- | --- | --- |
| TS-API-REQ-001 | FR-API-REQ-001, -003, -007 | One `checkRequest(w, r) bool` called at the top of `handleAPI`, after `Cache-Control: no-store` and before the auth, admin, session and data routing; skipped for `GET`, `HEAD`, `OPTIONS`. Non-API paths never reach it. | `internal/httpapi/auth.go`, `internal/httpapi/request_checks.go` (new) |
| TS-API-REQ-002 | FR-API-REQ-001, -002 | `sameOrigin(r)`: no `Origin` header passes; otherwise exactly one value, parsed with `url.Parse`, scheme `http`/`https`, equal (ignoring case) to `scheme + "://" + u.Host`, and `u.Host` equal to `r.Host` ignoring case. | `internal/httpapi/request_checks.go` |
| TS-API-REQ-003 | FR-API-REQ-003, -005 | `requestHasBody(r)`: `ContentLength > 0` → body; `0` → none; unknown (`-1`) → read one byte and put it back. | `internal/httpapi/request_checks.go` |
| TS-API-REQ-004 | FR-API-REQ-003, -004 | `jsonContentType(r)`: media type before the first `;` of `Content-Type`, trimmed, equal to `application/json` ignoring case. | `internal/httpapi/request_checks.go` |
| TS-API-REQ-005 | FR-API-REQ-006 | Origin check first, content type second; the first failure answers. | `internal/httpapi/request_checks.go` |
| TS-API-REQ-006 | FR-API-REQ-007 | Because the check runs before `handleAuth`, a refused login never reaches `decodeCredentials` or `Service.Login`: no password check, no failure recorded, no cookie. | `internal/httpapi/auth.go` |
| TS-API-REQ-007 | FR-API-REQ-008 | No frontend change: `request()` in `src/api/client.ts` already sends `Content-Type: application/json` only with a body, and the bodyless actions send none; browsers send the page's own `Origin`, and the Vite proxy keeps `Host` (no `changeOrigin`). Deploy note that the proxy must forward `Host` unchanged. Existing Go test helpers send the JSON content type with a body. | `deploy/README.md`, Go test helpers (see Refactoring) |
| TS-API-REQ-008 | FR-API-REQ-009 | Both rules, their order, scope, codes and messages in the `## Conventions` section of `API_SPECIFICATION.md` (done in this stage). | `API_SPECIFICATION.md` |

## Backend

### Placement (TS-API-REQ-001, TS-API-REQ-006)

In `handleAPI` (`internal/httpapi/auth.go`):

```go
w.Header().Set("Cache-Control", "no-store")
if !checkRequest(w, r) {
	return
}
path := r.URL.Path
...
```

So the refusal comes before `/api/v1/auth/` routing (login, logout), the session lookup, the admin and protected-mode checks, `handleData` routing, `ROUTE_NOT_FOUND` and every `METHOD_NOT_ALLOWED`. The handler comment of `handleAPI` gains "after the request checks of API_SPECIFICATION.md". `handleFrontend` (non-API paths) is not changed. `securityHeaders` still adds `X-Content-Type-Options`; no `Access-Control-*` header is set anywhere.

### New file `internal/httpapi/request_checks.go`

```go
// checkRequest applies the checks of state-changing API requests (API_SPECIFICATION.md, Conventions):
// the origin first, then the JSON content type of a non-empty body. It answers the refusal itself.
func checkRequest(w http.ResponseWriter, r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "ORIGIN_NOT_ALLOWED", "Requests from another site are not allowed.")
		return false
	}
	if requestHasBody(r) && !jsonContentType(r) {
		writeError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Request body must be JSON (Content-Type: application/json).")
		return false
	}
	return true
}
```

- `sameOrigin(r *http.Request) bool` (TS-API-REQ-002):
  - `values, present := r.Header["Origin"]`; not present → `true`. `len(values) != 1` → `false` (an empty value is present and fails below).
  - `u, err := url.Parse(values[0])`; refuse when `err != nil`, `u.Scheme` is not `http` or `https` (`url.Parse` lowercases the scheme), `u.Host == ""`, or `!strings.EqualFold(values[0], u.Scheme+"://"+u.Host)`. The last test rejects anything after the host and port (path, `/`, `?`, `#`), user info, and opaque forms; `null`, `prices.example` and `not an origin` fail on scheme or host.
  - Accept when `strings.EqualFold(u.Host, r.Host)`. `u.Host` keeps the port as written and IPv6 brackets (`[::1]:5173`), so ports are compared as written and a port on one side only fails. `r.Host` is the `Host` header (HTTP/1.1) or `:authority` (HTTP/2) as received; forwarded headers are not read.
- `requestHasBody(r *http.Request) bool` (TS-API-REQ-003):
  - `r.ContentLength > 0` → `true`; `r.ContentLength == 0` or `r.Body == nil` or `r.Body == http.NoBody` → `false` (Go's server sets `ContentLength` 0 and `NoBody` for `Content-Length: 0` or no body).
  - Unknown length (`-1`, chunked): `n, err := io.ReadFull(r.Body, one[:])` with a one-byte buffer. `n == 1` → put it back with `r.Body = struct{ io.Reader; io.Closer }{io.MultiReader(bytes.NewReader(one[:]), r.Body), r.Body}` and return `true`. `n == 0` → return `!errors.Is(err, io.EOF)` (an empty chunked body has none; a read error counts as a body, so a non-JSON request with a broken body is refused rather than run).
  - Called only after the origin check passed and only for state-changing API requests, so read requests and refused origins never touch the body. The later `http.MaxBytesReader` of each route wraps the restored body, so size limits are unchanged.
- `jsonContentType(r *http.Request) bool` (TS-API-REQ-004): `mediaType, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")`; `strings.EqualFold(strings.TrimSpace(mediaType), "application/json")`. Parameters are ignored without parsing them; a missing or empty header fails; `application/merge-patch+json`, `text/json` and `application/jsonx` fail.

Responses use the existing `writeError` (standard error JSON, `application/json; charset=utf-8`, `no-store` already set by `handleAPI`). The codes and messages are written once, in `checkRequest`.

### Validation, persistence, concurrency

- Each route's own body validation (`INVALID_JSON`, `REQUEST_TOO_LARGE`, `INVALID_URL`, credentials, settings rules) is unchanged and runs only after both checks pass; a `text/plain` body `{"url":` answers `415`, never `400`.
- No service, store, schema or migration change. The checks are stateless per request; no shared state, no locking.
- No logging of refusals (none is required; routes do not log client errors today).

## API

- New refusals on every state-changing API request: `403 ORIGIN_NOT_ALLOWED` "Requests from another site are not allowed." and `415 UNSUPPORTED_MEDIA_TYPE` "Request body must be JSON (Content-Type: application/json).", in the standard `{ "error": { "code", "message" } }` shape. Order, scope and exemptions are in `API_SPECIFICATION.md` "Conventions" (updated in this stage, with the status line).
- Request and success shapes of every endpoint are unchanged.
- Compatibility: the frontend already satisfies both rules. Scripts keep working when they send `Content-Type: application/json` with a body and no foreign `Origin`; `curl -d` without that header is now refused with `415` (intended, FR-API-REQ-003).

## Frontend

Not affected (TS-API-REQ-007). `request()` (`src/api/client.ts`) sets `Content-Type: application/json` exactly when `init.body` is set; `deleteItem`, `refreshAll`, `refreshItem`, `requestAIReview` and `logout` send no body; `login`, `createUser`, `addItem`, `updatePurchaseGoal` and `saveSettings` send `JSON.stringify` bodies. Same-origin `fetch` gets the page's own `Origin` from the browser. If a refusal ever reached the interface, the existing error display shows `ApiError.message` as for any error. No view, state, accessibility or UI-state change; only an e2e test is added.

## Deployment

`deploy/README.md` (developer stage, permitted path): one sentence next to the existing Cloudflare sentence under "Server setup": state-changing API requests are refused unless their `Origin` matches the `Host` header, so the reverse proxy must forward the browser's `Host` unchanged (Cloudflare does; a tunnel must not rewrite it, e.g. no `httpHostHeader`). No infrastructure change.

## Files to change

- `internal/httpapi/request_checks.go` (new): `checkRequest`, `sameOrigin`, `requestHasBody`, `jsonContentType`.
- `internal/httpapi/auth.go`: call `checkRequest` at the top of `handleAPI`; comment.
- Go test helpers (see Refactoring R-1): `internal/httpapi/auth_test.go` (`request`), `claude_settings_test.go` (`callPath`), `settings_test.go` (`call`), `login_attempt_limits_test.go` (`postLoginBody`), `session_cookie_security_test.go` (`sendAuth`, `browserClient.do`).
- `deploy/README.md`: the sentence above.
- `doc/specifications/api-request-checks/functional.md` status line and `doc/FUNCTIONAL_SPECIFICATIONS.md` status (developer marks implemented).
- Already done in this stage: `API_SPECIFICATION.md`.
- Not changed: `src/`, `vite.config.ts`, `playwright.config.ts`, `internal/service/`, `internal/store/`, dependencies.

## Tests

| Test case | Test file | Notes |
| --- | --- | --- |
| TC-001..TC-010 | `internal/httpapi/api_request_checks_test.go` (new) | `TestAPIRequestChecks_TCnnn_…`; `newSettingsServer` (open mode); a local helper builds `httptest.NewRequest` with `req.Host` (default `prices.example`), optional `Origin` and `Content-Type`; reuse `assertError`. TC-004/TC-007/TC-009 seed the LeBoncoin item `lbc` with `insertItem` (`ai_review_test.go`) plus a current price and the Claude token through the store, as the existing AI review tests do; "no check queued" reads `nextCheckAt`/status from `GET /api/v1/items/{id}`. TC-007 unknown length: `req.ContentLength = -1`. TC-009 logout and TC-010 login use protected mode as `newCookieServer`. |
| TC-011, TC-012 | `internal/httpapi/api_request_checks_test.go` | protected mode with `alice` (`newCookieServer`) and the administrator (`server.service.ResetAdminPassword`, then `login`, as in `auth_test.go`); TC-012 reuses `postLoginBody`-style requests with one `RemoteAddr`, `hasSessionCookie`, `expectHTTPInvalid`/`expectHTTPLocked`; `lastLoginAt` from `GET /api/v1/admin/users`. |
| TC-013 | `internal/httpapi/api_request_checks_test.go` | `httptest.NewServer(server.Handler())`; `http.Client` with `cookiejar`; proxy setup sets `req.Host = "prices.example"` and `X-Forwarded-Proto: https`; development setup puts `httputil.NewSingleHostReverseProxy` (keeps the incoming `Host`, like Vite without `changeOrigin`) in front, requests with `req.Host = "localhost:5173"`. Bodies with `Content-Type: application/json` only. Open and protected mode; also a run with no `Origin`. |
| TC-014 | `tests/item-refresh.spec.ts` | new test `TC-014 (#267) …` with its own stateful `context.route` mock (protected session: login, items, add, purchase goal, settings, refresh, refresh all, AI review, delete, logout; then an admin session with the users list and user creation) that records method, path, `request.headerValue("content-type")`, `request.headerValue("origin")` and `request.postData()` of each API request; corner case: add answers `403 ORIGIN_NOT_ALLOWED` and the add form shows the message. Existing specs unchanged. |
| TC-015 | `internal/httpapi/api_request_checks_test.go` | reads `../../API_SPECIFICATION.md`, cuts the text from `## Conventions` to the next `## `, checks the tokens and that `ORIGIN_NOT_ALLOWED` appears before `UNSUPPORTED_MEDIA_TYPE`; compares the documented messages with the bodies answered in the TC-001 and TC-006 requests. |

TC-014 goes into an existing spec because a new spec file needs an entry in `testMatch` of `playwright.config.ts`, which this pipeline cannot change; `item-refresh.spec.ts` is short and already covers item actions. Moving it to a dedicated `tests/api-request-checks.spec.ts` is an optional manual follow-up (add it to `testMatch` in its own PR).

All test cases describe new behaviour, so the tests fail on the base branch (no `403`/`415` today) and the TDD red gate needs no follow-up; the helper changes of R-1 keep the existing tests green before and after.

## Refactoring

- R-1 Go test helpers send `Content-Type: application/json` when their body is non-empty (`request`, `callPath`, `call`, `postLoginBody`, `sendAuth`, `browserClient.do`). Reason: after the change, every existing test that posts a JSON body through them would get `415`. Scope: one conditional header line per helper; no assertion changes (tests that send invalid JSON keep getting `400` because the header is JSON). Risk: low; a missed helper shows as a `415` in an existing test.

## Manual follow-ups

- None required. Optional: dedicated e2e spec file (see Tests).
