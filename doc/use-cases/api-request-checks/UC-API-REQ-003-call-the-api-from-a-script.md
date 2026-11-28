# UC-API-REQ-003 — Call the API from a script or tool

**Primary actor:** API client other than a browser (script, `curl`, tool) and the person who writes it
**Supporting systems:** PriceFollower application, `API_SPECIFICATION.md`
**Trigger:** The client sends a state-changing API request.
**Origin:** Issue #267: requests with a body must declare `Content-Type: application/json`.
**Covers:** FR-API-REQ-002, FR-API-REQ-003, FR-API-REQ-004, FR-API-REQ-005, FR-API-REQ-007, FR-API-REQ-009

## Preconditions

- The person writing the client has read the conventions of `API_SPECIFICATION.md`, which state both checks, their order, status codes, error codes, messages and the requests they apply to (FR-API-REQ-009).
- In protected mode, the client has a valid session.

## Main flow

1. The client sends `PUT /api/v1/settings` with `Content-Type: application/json`, a valid JSON body and no `Origin` header.
2. The request carries no `Origin`, so the origin check lets it through (FR-API-REQ-002).
3. The body is declared as JSON, so the content-type check lets it through (FR-API-REQ-004).
4. The settings are saved and the answer is the same as before.

## Alternatives and errors

- **Body not declared as JSON:** the client sends the same valid JSON body with `Content-Type: text/plain`, `application/x-www-form-urlencoded`, `multipart/form-data`, `application/merge-patch+json` or no `Content-Type` (for example `curl -d` without `-H 'Content-Type: application/json'`); the answer is `415` with the standard error response, code `UNSUPPORTED_MEDIA_TYPE`, message "Request body must be JSON (Content-Type: application/json)."; the settings are unchanged (FR-API-REQ-003). The person adds the header and sends again.
- **Non-empty body on a route that needs none:** `POST /api/v1/items/refresh` with a non-empty `text/plain` body answers `415` and no check starts (FR-API-REQ-003).
- **No body:** `POST /api/v1/items/refresh`, `POST /api/v1/items/{id}/refresh`, `POST /api/v1/items/{id}/ai-review`, `DELETE /api/v1/items/{id}` or `POST /api/v1/auth/logout` without body and without `Content-Type` answer as before (FR-API-REQ-005).
- **Protected mode, not logged in, body not JSON:** the answer is `415`, not `401`, since the checks come before authentication (FR-API-REQ-003, FR-API-REQ-007).
- **Invalid JSON with the JSON content type:** the answer is `400 INVALID_JSON` as before (FR-API-REQ-004).
- **Client sends an `Origin`:** it is judged as for a browser (see UC-API-REQ-001 and UC-API-REQ-002).
- **Read request:** `GET` and `HEAD` requests need no `Content-Type` and are unchanged.

## Postconditions

- A request that passes both checks has the same effect as before.
- A request refused with `415` has no effect: nothing is created, changed, started or counted.
