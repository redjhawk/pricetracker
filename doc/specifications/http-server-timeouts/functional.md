# Functional specification: HTTP server read and write timeouts and header size limit

Status: ready (issue #273)
Owner: functional specification agent
User decision/reference: GitHub issue #273 ("HTTP server: add read and write timeouts").

## Purpose and scope

The application's HTTP server limits only how long it waits for request headers (5 s) and how long an idle connection stays open (60 s). A client that sends its request body very slowly, or reads the response very slowly, keeps its connection open for as long as it likes. The user asks for limits on reading the whole request (15 s), on sending the response (60 s, because saving the Claude token in Settings waits up to 20 s for Claude to accept it) and on the size of the request headers. Every normal request keeps working.

Actors: any HTTP client of the application (browser, reverse proxy, script), and the operator running the server.

Included: the time the server allows to receive a whole request, the time it allows to send a response, and the largest request headers it accepts, in development and production mode alike (one server serves both).

Excluded: the existing 5 s limit for receiving request headers and the 60 s limit for an idle kept-alive connection, which stay unchanged; the request body size limits of the API endpoints, which stay unchanged; the time limits of background work (marketplace checks, AI reviews), which do not run inside a request; any screen, message or setting change; limits on the number of connections; outgoing requests the server sends to Amazon, LeBoncoin, Claude or GitHub.

Terms:

- **Request start:** the moment the server starts waiting for a request on a connection (the connection is accepted, or the previous response on a kept-alive connection was sent).
- **Request headers:** the request line and all header lines of a request, including cookies.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-HTTP-TMO-001 | A client has not finished sending its whole request (headers and body) 15 s after the request start. | The server stops reading the request and closes the connection. The request is not handled: nothing it asked for is saved or started, and no API response is sent. | A client that sends request headers announcing a JSON body and then sends the body one byte per second has its connection closed by the server between 15 s and 20 s after the request start, without an API response; the item, setting or goal it tried to save is unchanged. |
| FR-HTTP-TMO-002 | The server has not finished sending a response 60 s after it received the request headers. | The server stops sending and closes the connection; the client does not receive the rest of the response. | A client that reads a response more slowly than the server sends it, so that sending takes over 60 s, has its connection closed by the server between 60 s and 65 s after its request headers were received. |
| FR-HTTP-TMO-003 | A request whose request headers are at most 64 KiB (65,536 bytes) arrives. | It is handled as today. | A `GET /api/v1/items` with a 60 KiB extra header and a valid session answers 200 with the item list. |
| FR-HTTP-TMO-004 | A request whose request headers exceed 68 KiB (69,632 bytes) arrives. | The server answers HTTP 431 (Request Header Fields Too Large), does not handle the request, and closes the connection. Sizes between 64 KiB and 68 KiB may be accepted or refused. | A `GET /api/v1/items` with a 100 KiB extra header answers 431 and no item list. |
| FR-HTTP-TMO-005 | A person saves a Claude token in Settings and Claude takes up to 20 s to accept or reject it. | The save completes with its normal outcome (saved, or the existing rejection message); the time limits do not interrupt it. | With Claude answering the token check after 19 s, saving the token answers the usual success response and Settings shows the token as saved; with Claude not answering, the usual verification failure is shown after about 20 s. |
| FR-HTTP-TMO-006 | Any request from a client that sends its request and reads its response at normal speed. | Every page, asset and API request completes with its usual response. No request waits for a marketplace check or AI review: adding an item, Refresh price, Refresh all and a new AI review answer at once while the work runs in the background, as today. | The existing integration and e2e tests pass unchanged; adding an item, Refresh price and starting an AI review answer within 5 s each while the related check or review takes longer. |
| FR-HTTP-TMO-007 | A client sends no complete request headers within 5 s, or leaves a kept-alive connection idle for 60 s. | Unchanged: the server closes the connection. | A client that opens a connection and sends nothing has it closed after about 5 s; an idle kept-alive connection is closed after about 60 s. |

## States and corner cases

- The 15 s limit (FR-HTTP-TMO-001) covers the request headers too, so the 5 s header limit (FR-HTTP-TMO-007) still ends a connection that sends no headers sooner.
- The 60 s limit (FR-HTTP-TMO-002) counts from the received request headers, so it also covers the time to receive the body and to handle the request. A client too slow to read a response may lose the connection after the server has already done what the request asked; reloading shows the result.
- The largest request bodies the API accepts (1 MiB) are received well within 15 s at normal speeds; a body slower than that is refused under FR-HTTP-TMO-001.
- A connection closed by these limits is shown by the browser interface as its existing network error state; there is no new message.
- No new screen, message, setting or loading state.

## Acceptance criteria summary

- A request not fully received within 15 s of its start is dropped without being handled.
- A response not fully sent within 60 s of the received request headers is cut off and the connection closed.
- Request headers up to 64 KiB are accepted; headers over 68 KiB are answered 431.
- Saving the Claude token, with its verification of up to 20 s, and every other normal request keep working; the 5 s header and 60 s idle limits are unchanged.
