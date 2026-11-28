# UC-API-REQ-001 — A page of another site tries to change data through the visitor's browser

**Primary actor:** Page of another site loaded in the visitor's browser
**Supporting systems:** Browser, Cloudflare reverse proxy (production), PriceFollower application
**Trigger:** While the visitor has the other site open, its page sends a state-changing API request (`POST`, `PUT`, `DELETE` or any method other than `GET`, `HEAD`, `OPTIONS`) to the application's address.
**Origin:** Issue #267: in open mode nothing tied a state-changing request to the application's own pages.
**Covers:** FR-API-REQ-001, FR-API-REQ-006, FR-API-REQ-007

## Preconditions

- The application is reachable at its own address (for example `prices.example`), in open or protected mode.
- The visitor's browser has a page of another site open (for example `https://evil.example`).

## Main flow

1. The other site's page makes the browser send `POST /api/v1/items` to `prices.example` with a valid JSON body to add a listing.
2. The browser sets `Host: prices.example` and `Origin: https://evil.example` itself.
3. The application compares the `Origin` host and port with the `Host`; they differ.
4. The application answers `403` with the standard error response, code `ORIGIN_NOT_ALLOWED`, message "Requests from another site are not allowed." (FR-API-REQ-001).
5. No item is added and no check is started.

## Alternatives and errors

- **Other state-changing route:** the page sends `DELETE /api/v1/items/{id}`, `POST /api/v1/items/refresh`, `POST /api/v1/items/{id}/refresh`, `POST /api/v1/items/{id}/ai-review`, `PUT /api/v1/settings` or a purchase goal change with a foreign `Origin`; the answer is `403 ORIGIN_NOT_ALLOWED` and nothing is changed: the item still exists, no check or AI review starts, settings and goals are unchanged (FR-API-REQ-001).
- **`Origin: null`, empty or not a valid origin** (for example from a sandboxed frame or a local file): refused with `403 ORIGIN_NOT_ALLOWED` as in step 4 (FR-API-REQ-001).
- **Same host, other port:** `Host: localhost:5173` with `Origin: http://localhost:3001`; refused with `403` (FR-API-REQ-001, FR-API-REQ-002).
- **Default port written in `Origin`:** `Origin: https://prices.example:443` with `Host: prices.example`; refused with `403`, since ports are compared as written (FR-API-REQ-001).
- **Body not declared as JSON as well:** the page sends a form-like request with `Content-Type: text/plain` or `application/x-www-form-urlencoded`; the origin check applies first and the answer is `403 ORIGIN_NOT_ALLOWED`, not `415` (FR-API-REQ-006).
- **Protected mode, visitor logged out:** the answer is `403 ORIGIN_NOT_ALLOWED`, not `401` (FR-API-REQ-007).
- **Protected mode, visitor logged in:** the browser may send the session cookie, but the answer is still `403 ORIGIN_NOT_ALLOWED` and nothing is changed (FR-API-REQ-007).
- **Unknown API path or method not allowed on the route:** the refusal comes before route and method handling; the answer is `403 ORIGIN_NOT_ALLOWED` (FR-API-REQ-007).
- **Login form submitted by the other site:** see UC-API-REQ-004.
- **Read request:** a `GET`, `HEAD` or `OPTIONS` request with a foreign `Origin` is handled as before; no check applies and no CORS header is added.

## Postconditions

- Nothing is created, changed, started or counted: no item, setting, goal, user, session, check or AI review.
- The visitor's data is as before the other site's page was opened; the application's screens show nothing new.
