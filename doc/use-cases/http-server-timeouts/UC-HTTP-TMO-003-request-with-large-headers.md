# UC-HTTP-TMO-003 — Send a request with large request headers

**Primary actor:** HTTP client of the application (browser, reverse proxy, script)
**Supporting systems:** PriceFollower application
**Trigger:** The client sends a request whose request line and header lines, cookies included, are large.
**Origin:** Issue #273: the server did not set its own limit on the size of request headers.
**Covers:** FR-HTTP-TMO-003, FR-HTTP-TMO-004

## Preconditions

- The application is running (development or production mode).
- The client has a valid session.

## Main flow

1. The client sends `GET /api/v1/items` with its session cookie and an extra header of 60 KiB, so the request headers total at most 64 KiB (65,536 bytes).
2. The server handles the request as before and answers 200 with the item list (FR-HTTP-TMO-003).

## Alternatives and errors

- **Headers over 68 KiB:** the client sends `GET /api/v1/items` with an extra header of 100 KiB; the server answers HTTP 431 (Request Header Fields Too Large), does not handle the request, sends no item list and closes the connection (FR-HTTP-TMO-004).
- **Headers between 64 KiB and 68 KiB:** the request may be handled or answered 431 (FR-HTTP-TMO-004).
- **Other requests:** the same limits apply to every page, asset and API request, with or without a session.

## Postconditions

- A request with headers up to 64 KiB was handled as usual.
- A request with headers over 68 KiB was refused with 431 and changed nothing.
