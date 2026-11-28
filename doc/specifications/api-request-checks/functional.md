# Functional specification: JSON content type and same-origin checks on state-changing API requests

Status: specified (issue #267)
Owner: functional specification agent
User decision/reference: GitHub issue #267 ("API: require JSON content type and same-origin requests on state-changing routes").

## Purpose and scope

The API's state-changing routes (`POST`, `PUT`, `DELETE`) read their body as JSON whatever its `Content-Type`, and never look at the `Origin` header. In open mode there is no session cookie, so nothing ties such a request to the application's own pages: a page on another site can make the visitor's browser add, delete or refresh items, or change settings. The user asks that the API refuse a state-changing request whose body is not declared as JSON (`415`) and a state-changing request sent from another origin (`403`). The application runs behind a reverse proxy (Cloudflare), so the request's own origin is the `Host` the request was sent to.

Actors: the application's own pages in a browser, a page of another site loaded in the same browser, and other API clients (scripts, tools).

Included: every API request (path starting with `/api/`) with a state-changing method, in open mode and in protected mode, in production and in development.

Excluded: `GET`, `HEAD` and `OPTIONS` requests, which are unchanged; non-API paths (the frontend pages); CORS headers or preflight answers (none are added); the session cookie and its attributes ([session-cookie-security](../session-cookie-security/functional.md)); the `Referer` header and any forwarded header (`Forwarded`, `X-Forwarded-Host`, `X-Forwarded-Proto`); any screen change. Login, sessions, modes ([multi-user-login](../multi-user-login/functional.md)), login attempt limits ([login-attempt-limits](../login-attempt-limits/functional.md)) and every route's own validation stay unchanged for requests that pass both checks.

Terms:

- **State-changing request:** an API request whose method is not `GET`, `HEAD` or `OPTIONS` (today `POST`, `PUT`, `DELETE`; also `PATCH` and any other method).
- **Request with a body:** a request that carries a non-empty body. A request with no body or an empty body (for example Refresh, Delete, Request an AI review, Log out) has none.
- **JSON content type:** a `Content-Type` header whose media type is `application/json`, ignoring letter case and any parameters (for example `application/json; charset=utf-8`).
- **Own origin:** the `Origin` header names the application's own origin when its host and port (no port when the `Origin` has none) are equal to the request's `Host` header, ignoring letter case. The `Origin` scheme (`http` or `https`) is not compared, since HTTPS ends at the reverse proxy.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-API-REQ-001 | A state-changing request carries an `Origin` header that does not name the own origin (other host, other port, `null`, empty, or not a valid origin). | The request is refused with `403` and the standard error response, code `ORIGIN_NOT_ALLOWED`, message "Requests from another site are not allowed."; nothing is changed, stored, started or counted. | `POST /api/v1/items` with `Host: prices.example`, `Origin: https://evil.example` and a valid JSON body answers `403 ORIGIN_NOT_ALLOWED` and no item is added; the same with `Origin: null` answers `403`; `DELETE /api/v1/items/{id}` with a foreign `Origin` answers `403` and the item still exists. |
| FR-API-REQ-002 | A state-changing request carries an `Origin` header that names the own origin, or carries no `Origin` header. | The origin check lets the request through. | `Host: prices.example` with `Origin: https://prices.example`, with `Origin: http://PRICES.example`, or with no `Origin` passes the check; `Host: localhost:5173` with `Origin: http://localhost:5173` passes; `Host: localhost:5173` with `Origin: http://localhost:3001` answers `403`. |
| FR-API-REQ-003 | A state-changing request with a body has no `Content-Type` header, or one that is not the JSON content type. | The request is refused with `415` and the standard error response, code `UNSUPPORTED_MEDIA_TYPE`, message "Request body must be JSON (Content-Type: application/json)."; nothing is changed, stored, started or counted. | `PUT /api/v1/settings` with a valid JSON body and `Content-Type: text/plain` answers `415 UNSUPPORTED_MEDIA_TYPE` and the settings are unchanged; the same with `application/x-www-form-urlencoded`, `multipart/form-data`, `application/merge-patch+json` or no `Content-Type` answers `415`; `POST /api/v1/items/refresh` with a non-empty `text/plain` body answers `415` and no check starts. |
| FR-API-REQ-004 | A state-changing request with a body has the JSON content type. | The content-type check lets the request through; the route then handles the body as today (invalid JSON, size limits and field rules unchanged). | `POST /api/v1/items` with `Content-Type: application/json` or `Application/JSON; charset=UTF-8` and a valid body adds the item as before; with `Content-Type: application/json` and an invalid body it answers `400 INVALID_JSON` as before. |
| FR-API-REQ-005 | A state-changing request without a body. | No `Content-Type` is required; the content-type check lets it through, whatever `Content-Type` it carries. | `POST /api/v1/items/refresh`, `POST /api/v1/items/{id}/refresh`, `POST /api/v1/items/{id}/ai-review`, `DELETE /api/v1/items/{id}` and `POST /api/v1/auth/logout` with no body and no `Content-Type` answer as before. |
| FR-API-REQ-006 | A state-changing request fails both checks. | The origin check applies first: the answer is `403 ORIGIN_NOT_ALLOWED`. | A `POST /api/v1/items` with a foreign `Origin` and `Content-Type: text/plain` answers `403`. |
| FR-API-REQ-007 | A state-changing request fails a check. | The refusal comes before authentication, method and route handling: it is given in open and protected mode, whether or not the caller is logged in, and on any API path. A refused login is not a failed login: it neither counts toward nor is answered by the login attempt limits, and no password check is made. | In protected mode, a logged-out `POST /api/v1/items` with a foreign `Origin` answers `403`, not `401`; a `POST /api/v1/auth/login` with a foreign `Origin` and the right password answers `403`, sets no session cookie, and six such requests with a wrong password leave the next same-origin login with the right password succeeding. |
| FR-API-REQ-008 | The application's own pages are used, in production behind the reverse proxy and in development. | Every action of the interface (add, delete, refresh one or all items, AI review, purchase goal, Settings save, log in, log out, administrator user creation) works as before; the interface shows no new message. | The existing end-to-end tests pass unchanged; in development (frontend dev server and backend together) adding an item and saving Settings succeed. |
| FR-API-REQ-009 | The API contract. | `API_SPECIFICATION.md` states both rules, their order, their status codes, error codes and messages, and the requests they apply to. | The conventions of `API_SPECIFICATION.md` describe FR-API-REQ-001 to 007. |

## States and corner cases

- Clients other than browsers (scripts, `curl`) usually send no `Origin`; they keep working as long as a request with a body declares `Content-Type: application/json`. `curl -d` without `-H 'Content-Type: application/json'` is refused with `415`.
- The `Host` header is trusted as sent: in production the application is reachable only through Cloudflare, which keeps the browser's `Host`. A client reaching the application directly can choose both headers, but only a browser page of another site is the threat addressed here, and the browser sets `Origin` and `Host` itself.
- An `Origin` with the same host and port but another scheme (`http` instead of `https`) is accepted (FR-API-REQ-002), since HTTPS ends at the reverse proxy.
- A port is compared as written: `Origin: https://prices.example:443` with `Host: prices.example` is refused. Browsers do not write the default port in `Origin`.
- Refused requests have no effect: no item, setting, goal, user, session, check, AI review or login failure is created, changed or counted.
- No new screen, message, setting, loading state or empty state. The interface never receives `403 ORIGIN_NOT_ALLOWED` or `415 UNSUPPORTED_MEDIA_TYPE` in normal use; if it did, it would show the server message like any other error.

## Acceptance criteria summary

- A state-changing API request with a foreign `Origin` is refused with `403 ORIGIN_NOT_ALLOWED` and has no effect (FR-API-REQ-001, 006, 007).
- A state-changing API request with a body not declared as `application/json` is refused with `415 UNSUPPORTED_MEDIA_TYPE` and has no effect (FR-API-REQ-003, 007).
- Requests with the own origin or no `Origin`, with a JSON body or no body, behave as before; `GET` requests are unchanged (FR-API-REQ-002, 004, 005).
- The application works as before in production and development (FR-API-REQ-008), and `API_SPECIFICATION.md` documents the rules (FR-API-REQ-009).

## Open questions

None. The issue states both rules, their status codes and the comparison with `Host`; the error codes, messages, check order and the methods covered are API contract choices made here.
