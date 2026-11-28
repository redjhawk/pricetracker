# Test cases: JSON content type and same-origin checks on state-changing API requests (issue #267)

Sources: functional specification [api-request-checks](../../specifications/api-request-checks/functional.md) (FR-API-REQ-001 to FR-API-REQ-009) and use cases [UC-API-REQ-001](../../use-cases/api-request-checks/UC-API-REQ-001-another-site-tries-to-change-data.md), [UC-API-REQ-002](../../use-cases/api-request-checks/UC-API-REQ-002-use-own-pages-behind-proxy-and-in-development.md), [UC-API-REQ-003](../../use-cases/api-request-checks/UC-API-REQ-003-call-the-api-from-a-script.md) and [UC-API-REQ-004](../../use-cases/api-request-checks/UC-API-REQ-004-another-site-submits-the-login.md).

Conventions:

- **HTTP handler tests** (`internal/httpapi`, package-internal) run `server.Handler()` on a real service and SQLite database: open mode as in `newSettingsServer`, protected mode with user `alice` and the administrator as in the existing auth and login attempt limit tests. Requests are built with `httptest.NewRequest`; `req.Host` is set explicitly (default `prices.example`), `Origin` and `Content-Type` are set only when a case says so. New Go test names use the prefix `TestAPIRequestChecks_TCnnn_`.
- **Refused** means the answer has the stated status and the standard error response (`{"error":{"code":…,"message":…}}`, checked with `assertError`) with the exact code and message: `403 ORIGIN_NOT_ALLOWED` "Requests from another site are not allowed." or `415 UNSUPPORTED_MEDIA_TYPE` "Request body must be JSON (Content-Type: application/json).".
- **No effect** is checked after each refusal with read requests (which pass both checks): `GET /api/v1/items` and `GET /api/v1/items/{id}` (items, statuses, next check times, purchase goals, AI review state), `GET /api/v1/settings` (secrets saved flags and revisions), `GET /api/v1/admin/users` (users and `lastLoginAt`), `GET /api/v1/auth/session`.
- **Valid add body** is `{"url":"https://www.leboncoin.fr/ad/test/<n>"}` with a distinct `<n>` per request.
- Existing Go test helpers that send a non-empty body to the handler (`request`, `callPath`, `call` and any other) set `Content-Type: application/json`, as the frontend does; the assertions of existing tests are unchanged and keep passing.
- **E2E:** the Playwright specs mock the API with `context.route`, so they cannot observe the Go checks; the backend rules and the user flows over real HTTP are proven by integration tests, and one Playwright case proves the frontend's requests satisfy the rules. The existing Playwright specs pass unchanged.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-API-REQ-001 | TC-001, TC-002, TC-004, TC-011, TC-012 |
| FR-API-REQ-002 | TC-002, TC-003, TC-013 |
| FR-API-REQ-003 | TC-006, TC-007, TC-011, TC-012 |
| FR-API-REQ-004 | TC-008, TC-013 |
| FR-API-REQ-005 | TC-009, TC-013 |
| FR-API-REQ-006 | TC-010 |
| FR-API-REQ-007 | TC-011, TC-012 |
| FR-API-REQ-008 | TC-013, TC-014 |
| FR-API-REQ-009 | TC-015 |

| Use case | Test cases |
| --- | --- |
| UC-API-REQ-001 | TC-001, TC-002, TC-004, TC-005, TC-010, TC-011 |
| UC-API-REQ-002 | TC-003, TC-008, TC-009, TC-013, TC-014 |
| UC-API-REQ-003 | TC-003, TC-006, TC-007, TC-008, TC-009, TC-011, TC-015 |
| UC-API-REQ-004 | TC-010, TC-012 |

## Origin check

### TC-001 — Adding an item from another site is refused and adds nothing

- Covers: FR-API-REQ-001; UC-API-REQ-001 (main flow)
- Level: integration (HTTP handler, real SQLite, open mode)
- Given open mode and no item
- When `POST /api/v1/items` with `Host: prices.example`, `Origin: https://evil.example`, `Content-Type: application/json` and a valid add body
- Then refused `403 ORIGIN_NOT_ALLOWED`, and `GET /api/v1/items` answers an empty list.
- Corner cases: the same request sent twice answers `403` both times and still adds nothing; the same body then sent with `Origin: https://prices.example` answers `201` (the refusal was due only to the origin).

### TC-002 — `Origin` values that do not name the own origin

- Covers: FR-API-REQ-001, FR-API-REQ-002; UC-API-REQ-001 (alternatives "`Origin: null`, empty or not a valid origin", "Same host, other port", "Default port written in `Origin`")
- Level: integration (HTTP handler, real SQLite, open mode; table test, one `POST /api/v1/items` with `Content-Type: application/json` and a valid add body per row)
- Given, per row (`Host` → `Origin`):
  - `prices.example` → `https://evil.example`, `http://evil.example`, `https://sub.prices.example`, `https://prices.example.evil.example`, `null`, empty value, `prices.example` (no scheme), `not an origin`, `https://prices.example:443`, `http://prices.example:8080`;
  - `prices.example:8443` → `https://prices.example`;
  - `localhost:5173` → `http://localhost:3001`, `http://localhost`;
  - `127.0.0.1:4173` → `http://localhost:4173`
- Then each row is refused `403 ORIGIN_NOT_ALLOWED` and the item list stays empty.
- Corner cases: other host, other port, port only on one side, default port written, sub- and super-domain, `null`, empty and malformed values.

### TC-003 — Own origin and missing `Origin` pass the check

- Covers: FR-API-REQ-002; UC-API-REQ-002 (alternatives "Scheme differs", "Browser sends no `Origin`", "Development"), UC-API-REQ-003 (main flow step 2)
- Level: integration (HTTP handler, real SQLite, open mode; table test, one `POST /api/v1/items` with `Content-Type: application/json` and a valid add body per row)
- Given, per row (`Host` → `Origin`):
  - `prices.example` → `https://prices.example`, `http://prices.example`, `http://PRICES.example`, `HTTPS://Prices.Example`, no `Origin` header;
  - `PRICES.EXAMPLE` → `https://prices.example`;
  - `localhost:5173` → `http://localhost:5173`;
  - `127.0.0.1:4173` → `http://127.0.0.1:4173`;
  - `[::1]:5173` → `http://[::1]:5173`
- Then each row answers `201` with the added item, and the list holds one item per row.
- Corner cases: scheme not compared, letter case ignored on both sides, ports equal, IPv6 host, no header (scripts).

### TC-004 — Every state-changing route refuses another site and changes nothing

- Covers: FR-API-REQ-001; UC-API-REQ-001 (alternative "Other state-changing route")
- Level: integration (HTTP handler, real SQLite, open mode)
- Given open mode with a LeBoncoin item `lbc` having a current price and purchase goal `Office`, a saved Claude token, and the current settings read with `GET /api/v1/settings`
- When each of these requests is sent with `Origin: https://evil.example` (bodies with `Content-Type: application/json`):
  - `DELETE /api/v1/items/lbc`;
  - `POST /api/v1/items/refresh` and `POST /api/v1/items/lbc/refresh`, no body;
  - `POST /api/v1/items/lbc/ai-review`, no body;
  - `PUT /api/v1/items/lbc/purchase-goal` with `{"purchaseGoal":"Gaming"}`;
  - `PUT /api/v1/settings` removing the Claude token;
  - `PATCH /api/v1/items/lbc` with `{}`
- Then each is refused `403 ORIGIN_NOT_ALLOWED`; afterwards `lbc` still exists with the same status and next check time (no check queued), purchase goal `Office`, no AI review running or added, and `GET /api/v1/settings` answers the same as before.
- Corner cases: `PATCH` (not otherwise allowed on the route) gets `403`, not `405`.

### TC-005 — Read requests and non-API paths are unchanged

- Covers: FR-API-REQ-001, FR-API-REQ-002 (scope: only state-changing API requests); UC-API-REQ-001 (alternative "Read request")
- Level: integration (HTTP handler, real SQLite, open mode, one item)
- Given open mode with one item
- When `GET /api/v1/items`, `GET /api/v1/items/{id}`, `GET /api/v1/settings`, `GET /api/v1/auth/session`, `HEAD /api/v1/items` and `OPTIONS /api/v1/items` are each sent twice: once with `Origin: https://evil.example` and `Content-Type: text/plain`, once with neither
- Then both answers of each pair have the same status and body, and none is `403 ORIGIN_NOT_ALLOWED` or `415 UNSUPPORTED_MEDIA_TYPE`.
- And `POST /` (a non-API path) with `Origin: https://evil.example` and a `text/plain` body answers the same as without them.
- Corner cases: no answer of this case or of TC-001 (refusal) carries an `Access-Control-*` header (no CORS added).

## Content-type check

### TC-006 — Saving settings with a body not declared as JSON is refused

- Covers: FR-API-REQ-003; UC-API-REQ-003 (alternative "Body not declared as JSON")
- Level: integration (HTTP handler, real SQLite, open mode; table test)
- Given open mode with no saved secret and no `Origin` header
- When `PUT /api/v1/settings` with the valid body `{"leboncoinSession":{"value":"Cookie: a=1; datadome=synthetic; b=2","revision":0}}` and, per row, `Content-Type`: `text/plain`, `text/plain; charset=UTF-8`, `application/x-www-form-urlencoded`, `multipart/form-data; boundary=x`, `application/merge-patch+json`, `text/json`, `application/jsonx`, empty value, no header
- Then each row is refused `415 UNSUPPORTED_MEDIA_TYPE`, and `GET /api/v1/settings` still shows no saved LeBoncoin session at revision 0.
- And the same body with `Content-Type: application/json` then saves the session (revision 1).
- Corner cases: the `curl -d` default (`application/x-www-form-urlencoded`), JSON-like but other media types, missing and empty header; repeated refused saves do not move the revision.

### TC-007 — A non-empty body on any route needs the JSON content type

- Covers: FR-API-REQ-003; UC-API-REQ-003 (alternative "Non-empty body on a route that needs none")
- Level: integration (HTTP handler, real SQLite, open mode, item `lbc` with a current price and a saved Claude token)
- Given no `Origin` header
- When:
  - `POST /api/v1/items/refresh` with body `x` and `Content-Type: text/plain`;
  - `POST /api/v1/items/lbc/refresh` with body `{}` and no `Content-Type`;
  - `POST /api/v1/items/lbc/ai-review` with body `x` and `Content-Type: text/plain`;
  - `DELETE /api/v1/items/lbc` with body `a=1` and `Content-Type: application/x-www-form-urlencoded`;
  - `POST /api/v1/items` with a valid add body, `Content-Type: text/plain` and an unknown length (`req.ContentLength = -1`, chunked)
- Then each is refused `415 UNSUPPORTED_MEDIA_TYPE`; `lbc` still exists with the same status and next check time, no AI review starts, and no item is added.
- Corner cases: body of one byte, `{}` body without header, body of unknown length.

### TC-008 — A JSON body is handled by the route as before

- Covers: FR-API-REQ-004; UC-API-REQ-002 (alternatives "Content type with parameters or other case", "Invalid data"), UC-API-REQ-003 (main flow, alternative "Invalid JSON with the JSON content type")
- Level: integration (HTTP handler, real SQLite, open mode)
- Given open mode and `Origin: https://prices.example` or no `Origin` (both rows run)
- When `POST /api/v1/items` with a valid add body and, per row, `Content-Type`: `application/json`, `application/json; charset=utf-8`, `Application/JSON; charset=UTF-8`, `APPLICATION/JSON`
- Then each answers `201` with the item, as before.
- And with `Content-Type: application/json`, as before: body `{"url":` answers `400 INVALID_JSON`; body `{}` answers `400 INVALID_URL`; a body over the size limit answers `413 REQUEST_TOO_LARGE`; `PUT /api/v1/settings` with a valid body saves the settings and with `{"leboncoinSession":` answers `400 INVALID_JSON`; `PUT /api/v1/items/{id}/purchase-goal` with `{"purchaseGoal":"Gaming"}` answers `200` with the goal saved.
- Corner cases: `{"url":` with `Content-Type: text/plain` answers `415`, not `400` (the check comes before the body is read).

### TC-009 — Requests without a body need no content type

- Covers: FR-API-REQ-005; UC-API-REQ-002 (alternative "Actions without a body"), UC-API-REQ-003 (alternative "No body")
- Level: integration (HTTP handler, real SQLite; open mode for the item routes, protected mode for logout)
- Given an item `lbc` with a current price and a saved Claude token (open mode), and `alice` logged in (protected mode)
- When, with no body and no `Content-Type`: `POST /api/v1/items/refresh`, `POST /api/v1/items/lbc/refresh`, `POST /api/v1/items/lbc/ai-review`, `DELETE /api/v1/items/lbc`, and `POST /api/v1/auth/logout` with `alice`'s session cookie
- Then they answer as before: `202`, `202`, `202`, `204`, `204` with the clearing session cookie (and the session is ended).
- Corner cases: the same requests with an empty body and `Content-Type: text/plain`, or with an explicit `Content-Length: 0`, answer the same; a repeated `DELETE /api/v1/items/lbc` answers `404 ITEM_NOT_FOUND` as before.

## Order and scope

### TC-010 — When both checks fail, the origin refusal is given

- Covers: FR-API-REQ-006; UC-API-REQ-001 (alternative "Body not declared as JSON as well"), UC-API-REQ-004 (alternative "Body not declared as JSON as well")
- Level: integration (HTTP handler, real SQLite; open mode, and protected mode for the login row)
- Given `Origin: https://evil.example`
- When `POST /api/v1/items` with a valid add body and `Content-Type: text/plain`; `PUT /api/v1/settings` with `a=1` and `Content-Type: application/x-www-form-urlencoded`; `POST /api/v1/auth/login` with `alice`'s credentials as JSON and no `Content-Type`
- Then each is refused `403 ORIGIN_NOT_ALLOWED`, not `415`; no item is added, the settings are unchanged, and the login sets no session cookie.

### TC-011 — Refusals come before authentication, routing and method handling

- Covers: FR-API-REQ-007, FR-API-REQ-001, FR-API-REQ-003; UC-API-REQ-001 (alternatives "Protected mode, visitor logged out/logged in", "Unknown API path or method not allowed"), UC-API-REQ-003 (alternative "Protected mode, not logged in, body not JSON")
- Level: integration (HTTP handler, real SQLite, protected mode with `alice` and the administrator)
- Given protected mode
- When:
  - logged out, `POST /api/v1/items` with a valid add body as JSON and `Origin: https://evil.example`;
  - logged out, no `Origin`, the same body with `Content-Type: text/plain`;
  - `alice` logged in (session cookie sent), `POST /api/v1/items` with `Origin: https://evil.example`;
  - `alice` logged in, `POST /api/v1/auth/logout` with `Origin: https://evil.example`;
  - logged out, and then logged in as the administrator, `POST /api/v1/admin/users` with `{"username":"bob","password":"long enough pw"}` and `Origin: https://evil.example`; the administrator, the same with `Content-Type: text/plain` and no `Origin`;
  - `alice` logged in, `POST /api/v1/admin/users` with `Origin: https://evil.example`;
  - `POST /api/v1/unknown` with `Origin: https://evil.example`;
  - `PATCH /api/v1/items/refresh` with body `x` and `Content-Type: text/plain`
- Then the answers are, in order: `403 ORIGIN_NOT_ALLOWED` (not `401`); `415 UNSUPPORTED_MEDIA_TYPE` (not `401`); `403 ORIGIN_NOT_ALLOWED` and `alice` has no item; `403 ORIGIN_NOT_ALLOWED` and `GET /api/v1/auth/session` still answers `alice` (session not ended); `403 ORIGIN_NOT_ALLOWED` twice (not `401`) and `415`, with no user `bob` in `GET /api/v1/admin/users`; `403 ORIGIN_NOT_ALLOWED` (not `FORBIDDEN`); `403 ORIGIN_NOT_ALLOWED` (not `404`); `415` (not `405`).

### TC-012 — A refused login neither logs in nor counts as a failed login

- Covers: FR-API-REQ-007, FR-API-REQ-001, FR-API-REQ-003; UC-API-REQ-004 (main flow, alternative "Wrong password, repeated", postconditions)
- Level: integration (HTTP handler, real SQLite, protected mode, one client address)
- Given protected mode and `alice` who never logged in
- When `POST /api/v1/auth/login` with `alice`'s right password as JSON and `Origin: https://evil.example`
- Then refused `403 ORIGIN_NOT_ALLOWED` with no `Set-Cookie`, and `GET /api/v1/admin/users` (as the administrator) shows `alice` with `lastLoginAt` null.
- And six logins for `alice` with a wrong password and `Origin: https://evil.example` each answer `403 ORIGIN_NOT_ALLOWED` (never the login attempt limit answer); six with a wrong password as a `text/plain` body and no `Origin` each answer `415`; then a login with the right password, `Origin: https://prices.example` and `Content-Type: application/json` answers `200` with a session cookie.
- Corner cases (fresh server): four real failed logins (same-origin, JSON, wrong password), then the twelve refused ones, then a fifth real failure: it answers `401 INVALID_CREDENTIALS`, not the limit answer (the refused requests were not counted), and the next login is refused by the limit as before.

## User flows

### TC-013 — The application's requests work behind the proxy and in development

- Covers: FR-API-REQ-008, FR-API-REQ-002, FR-API-REQ-004, FR-API-REQ-005; UC-API-REQ-002 (main flow, alternatives "Other actions with a body", "Actions without a body", "Development")
- Level: integration (browser-like flow: `httptest.NewServer(server.Handler())` over real HTTP, `http.Client` with a cookie jar, real SQLite; each request carries the headers the frontend sends: `Content-Type: application/json` only with a body)
- Given two setups, each in protected mode and in open mode:
  - proxy: every request carries `Host: prices.example`, `Origin: https://prices.example` on state-changing requests, and `X-Forwarded-Proto: https`;
  - development: requests go through a reverse proxy in front of the test server that keeps the `Host`, as the Vite dev server proxy does (`vite.config.ts`, no `changeOrigin`), with `Host: localhost:5173` and `Origin: http://localhost:5173` on state-changing requests
- When the client (protected mode: logs in as `alice` first) adds an item, sets its purchase goal, saves Settings (Claude token), refreshes the item, refreshes all, requests an AI review, deletes the item and (protected mode) logs out; and the administrator logs in and creates a user
- Then every answer has the same status and body shape as today (`200`/`201`/`202`/`204`) and none is `403` or `415`.
- Corner cases: the same flow with no `Origin` on any request (a script) gives the same answers.

### TC-014 — The interface sends requests that pass both checks

- Covers: FR-API-REQ-008; UC-API-REQ-002 (main flow, alternatives "Other actions with a body", "Actions without a body", "Unexpected refusal")
- Level: e2e (Playwright, new spec in `tests/` added to `testMatch`, API mocked with `context.route`, protected mode for login, logout and user creation)
- Given the application at `http://127.0.0.1:4173` with the API mocked and every API request recorded with its method, path, `Content-Type`, `Origin` and body (`request.headerValue`, `request.postData()`)
- When the user, using the interface: logs in, adds an item, changes its purchase goal, saves Settings, refreshes the item, uses Refresh all, requests an AI review, deletes the item and logs out; and the administrator logs in and creates a user
- Then every recorded request with a body has `Content-Type` `application/json` and a body that parses as JSON; every refresh, AI review, delete and logout request has no body; every recorded `Origin`, when present, is `http://127.0.0.1:4173`.
- And the existing Playwright specs pass unchanged.
- Corner cases: when the mock answers adding an item with `403` `ORIGIN_NOT_ALLOWED` and message "Requests from another site are not allowed.", the add form shows that message in its usual error display, with no other new message.

## API contract

### TC-015 — `API_SPECIFICATION.md` documents both rules

- Covers: FR-API-REQ-009; UC-API-REQ-003 (preconditions)
- Level: unit (documentation test on the real `API_SPECIFICATION.md`)
- Given the `## Conventions` section of `API_SPECIFICATION.md`
- When the test reads it
- Then it contains `ORIGIN_NOT_ALLOWED` with `403` and the exact message "Requests from another site are not allowed.", `UNSUPPORTED_MEDIA_TYPE` with `415` and the exact message "Request body must be JSON (Content-Type: application/json).", `application/json`, `Origin`, `Host`, and names `GET`, `HEAD` and `OPTIONS` as the exempt methods; the origin rule is stated before the content-type rule.
- And the codes and messages it documents are equal to those the server answers in TC-001 and TC-006.
- Corner cases: the tokens are searched only in the `## Conventions` section (a mention elsewhere does not pass); the statements that the checks come before authentication, apply to `/api/` paths only, exempt requests without a body, and compare host and port ignoring case and scheme are checked by review against FR-API-REQ-001 to FR-API-REQ-007.
