# UC-HTTP-TMO-001 — A client that sends its request too slowly is dropped

**Primary actor:** HTTP client of the application (browser, reverse proxy, script) that sends its request very slowly
**Supporting systems:** PriceFollower application
**Trigger:** The client opens a connection and starts sending a request, then sends the rest of it slowly.
**Origin:** Issue #273: the server limited only the time to receive request headers, so a client sending its body slowly kept its connection open without limit.
**Covers:** FR-HTTP-TMO-001, FR-HTTP-TMO-007

## Preconditions

- The application is running (development or production mode).
- The client can reach the application.

## Main flow

1. The client opens a connection; the request start is the moment the server starts waiting for a request on it.
2. Within 5 s the client sends complete request headers announcing a JSON body, for example a `PUT` saving a setting or a `POST` adding an item.
3. The client sends the body one byte per second.
4. Between 15 s and 20 s after the request start, the request is still not fully received; the server stops reading and closes the connection (FR-HTTP-TMO-001).
5. The client receives no API response.
6. Nothing the request asked for is saved or started: the item, setting or goal it tried to save is unchanged (FR-HTTP-TMO-001).

## Alternatives and errors

- **No complete headers within 5 s:** the client opens a connection and sends nothing, or sends its headers too slowly; the server closes the connection after about 5 s, with no response or, when part of the request line has arrived, after a plain `400 Bad Request`, as before (FR-HTTP-TMO-007).
- **Endpoint without a request body:** the request goes to an endpoint that takes no body (for example Refresh price, Refresh all, a new AI review, a delete) and announces a body that it sends slowly; the request is handled once its headers arrive, as today, and its response is sent before the connection is closed (FR-HTTP-TMO-001).
- **Kept-alive connection:** after a response on a kept-alive connection, the next request starts when that response was sent; the same 15 s limit applies to it (FR-HTTP-TMO-001).
- **Idle kept-alive connection:** the client sends no new request; the server closes the connection after about 60 s, as before (FR-HTTP-TMO-007).
- **Whole request received in time:** a request, including its largest accepted body (1 MiB), sent at normal speed is fully received well within 15 s and handled as usual (UC-HTTP-TMO-005).
- **Browser interface:** if a browser request is dropped this way, the interface shows its existing network error state; there is no new message.

## Postconditions

- The connection is closed and the request was not handled.
- No data changed.
