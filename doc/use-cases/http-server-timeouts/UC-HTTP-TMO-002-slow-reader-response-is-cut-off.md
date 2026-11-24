# UC-HTTP-TMO-002 — A client that reads its response too slowly is cut off

**Primary actor:** HTTP client of the application that reads the response very slowly
**Supporting systems:** PriceFollower application
**Trigger:** The client sends a request at normal speed, then reads the response more slowly than the server sends it.
**Origin:** Issue #273: the server had no limit on sending a response, so a client reading slowly kept its connection open without limit.
**Covers:** FR-HTTP-TMO-002

## Preconditions

- The application is running (development or production mode).
- The client sends a request whose response takes long to send at the client's reading speed, for example a large asset or item list.

## Main flow

1. The client sends its request headers; the server receives them.
2. The server handles the request and starts sending the response.
3. The client reads the response so slowly that sending it would take over 60 s.
4. Between 60 s and 65 s after the request headers were received, the server stops sending and closes the connection (FR-HTTP-TMO-002).
5. The client does not receive the rest of the response.

## Alternatives and errors

- **Request already handled:** the 60 s count from the received headers also covers receiving the body and handling the request; if the request changed something (for example saved a setting), the change stays done although the client lost the response. Reloading shows the result.
- **Response sent in time:** a client reading at normal speed receives the whole response (UC-HTTP-TMO-005).
- **Browser interface:** if a browser response is cut off this way, the interface shows its existing network error state; there is no new message.

## Postconditions

- The connection is closed; the server no longer holds it open for the slow client.
