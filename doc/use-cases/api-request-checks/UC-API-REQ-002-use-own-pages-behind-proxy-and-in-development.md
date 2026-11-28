# UC-API-REQ-002 — Use the application's own pages behind the proxy and in development

**Primary actor:** Person using the application (any user, or the visitor in open mode)
**Supporting systems:** Browser, Cloudflare reverse proxy (production) or frontend dev server (development), PriceFollower application
**Trigger:** The actor performs an action that changes data from the application's own pages.
**Origin:** Issue #267: the new checks must not break the application's own requests.
**Covers:** FR-API-REQ-002, FR-API-REQ-004, FR-API-REQ-005, FR-API-REQ-008

## Preconditions

- The application is open in the browser at its own address: `https://prices.example` through the reverse proxy, or `http://localhost:5173` through the frontend dev server in development.
- In protected mode, the actor is logged in (or logs in during the flow).

## Main flow

1. The actor adds a listing from the add form.
2. The browser sends `POST /api/v1/items` with `Host: prices.example`, `Origin: https://prices.example`, `Content-Type: application/json` and the JSON body.
3. The proxy ends HTTPS and forwards the request with the same `Host`.
4. The application finds that the `Origin` host and port equal the `Host` and lets the request through (FR-API-REQ-002); the body is declared as JSON, so the content-type check lets it through (FR-API-REQ-004).
5. The item is added exactly as before; the interface shows the same result and no new message (FR-API-REQ-008).

## Alternatives and errors

- **Other actions with a body:** saving Settings, changing an item's purchase goal, logging in and creating a user as administrator send `Content-Type: application/json` and work as before (FR-API-REQ-004, FR-API-REQ-008).
- **Actions without a body:** Refresh all, refreshing one item, requesting an AI review, deleting an item and logging out send no body; no `Content-Type` is required and they work as before (FR-API-REQ-005, FR-API-REQ-008).
- **Development:** the page at `http://localhost:5173` sends requests through the dev server with `Host: localhost:5173` and `Origin: http://localhost:5173`; they pass both checks and adding an item and saving Settings succeed (FR-API-REQ-002, FR-API-REQ-008).
- **Scheme differs from the `Origin`:** HTTPS ends at the proxy; an `Origin` with the same host and port but another scheme, or another letter case (`http://PRICES.example`), passes the check (FR-API-REQ-002).
- **Browser sends no `Origin`:** the request passes the origin check (FR-API-REQ-002).
- **Content type with parameters or other case:** `application/json; charset=utf-8` or `Application/JSON; charset=UTF-8` passes the check (FR-API-REQ-004).
- **Invalid data:** an invalid JSON body or a field the route rejects gives the same error as before (for example `400 INVALID_JSON`), shown as today (FR-API-REQ-004).
- **Unexpected refusal:** if the interface ever received `403 ORIGIN_NOT_ALLOWED` or `415 UNSUPPORTED_MEDIA_TYPE`, it would show the server message like any other error; in normal use this does not happen (FR-API-REQ-008).

## Postconditions

- Every interface action has the same effect and display as before the change.
- No new screen, message, setting or loading state.
