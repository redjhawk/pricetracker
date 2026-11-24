# Test cases: HTTP server read and write timeouts and header size limit (issue #273)

Sources: functional specification [http-server-timeouts](../../specifications/http-server-timeouts/functional.md) (FR-HTTP-TMO-001 to FR-HTTP-TMO-007) and use cases [UC-HTTP-TMO-001 to UC-HTTP-TMO-005](../../use-cases/http-server-timeouts/).

Conventions:

- **Server constructor**: the one function that builds the application's `*http.Server` and that `cmd/pricefollower/main.go` uses (the technical specification names and places it). Tests never build their own `http.Server` literal for the application.
- **Live server** (level `integration`): a Go test that builds the server with the server constructor, around the real `httpapi` handler on a real SQLite store in `t.TempDir()`, and serves it with `Serve` on a `127.0.0.1:0` listener. Unless a case says otherwise, the handler is the real one and the limits are the production values (5 s headers, 15 s read, 60 s write, 60 s idle, 64 KiB headers). Marketplace collectors and Claude are fakes (no network beyond loopback).
- **Raw client**: a `net.Dial` TCP connection that writes the request bytes itself, so the test controls the speed and the exact header size. "Closed" means a read on the connection returns `io.EOF` or a connection reset. Times are measured with the test's monotonic clock from the moment named in the case.
- **Session**: a logged-in session cookie (`login` helper of `auth_test.go`) in protected mode; "no session" means the request has no cookie.
- Long cases (15 s to 65 s) call `t.Parallel()` so they run at the same time; the package stays under about 70 s of extra wall time.
- The e2e suite mocks the API with `context.route` on the Vite preview server, so it cannot see the Go server's limits. The server behaviour is proved by integration cases; the only e2e case proves that the Settings interface waits for a slow save without its own limit.
- No screen, message or setting changes, so narrow screens and keyboard use do not apply beyond the unchanged existing e2e suite (TC-013).

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-HTTP-TMO-001 | TC-001, TC-002, TC-003, TC-004 |
| FR-HTTP-TMO-002 | TC-001, TC-005, TC-006, TC-009 |
| FR-HTTP-TMO-003 | TC-001, TC-007 |
| FR-HTTP-TMO-004 | TC-001, TC-008 |
| FR-HTTP-TMO-005 | TC-009, TC-010 |
| FR-HTTP-TMO-006 | TC-004, TC-011, TC-013 |
| FR-HTTP-TMO-007 | TC-001, TC-012 |

| Use case | Test cases |
| --- | --- |
| UC-HTTP-TMO-001 | TC-002, TC-003, TC-004, TC-012 |
| UC-HTTP-TMO-002 | TC-005, TC-006 |
| UC-HTTP-TMO-003 | TC-007, TC-008 |
| UC-HTTP-TMO-004 | TC-009, TC-010 |
| UC-HTTP-TMO-005 | TC-011, TC-012, TC-013 |

## Cases

### TC-001 — The server constructor sets every limit

- Covers: FR-HTTP-TMO-001, FR-HTTP-TMO-002, FR-HTTP-TMO-003, FR-HTTP-TMO-004, FR-HTTP-TMO-007
- Level: unit
- Given a configuration with host `127.0.0.1` and port `8081`, in development mode and in production mode, and a test handler
- When the server constructor builds the server
- Then `ReadTimeout` is 15 s, `WriteTimeout` is 60 s, `MaxHeaderBytes` is 65,536, `ReadHeaderTimeout` is 5 s, `IdleTimeout` is 60 s, `Addr` is `127.0.0.1:8081` and `Handler` is the given handler
- Corner cases:
  - both modes give the same limits (one server serves both);
  - `MaxHeaderBytes` is not 0 (0 would fall back to Go's 1 MiB default);
  - `cmd/pricefollower/main.go` builds its server only through the constructor (review check: no `http.Server{` literal remains in `cmd/`).

### TC-002 — A request body sent one byte per second is dropped and changes nothing

- Covers: FR-HTTP-TMO-001 (UC-HTTP-TMO-001 main flow)
- Level: integration (live server, raw client, session)
- Given a live server with one tracked LeBoncoin item `I1` without a purchase goal, no saved LeBoncoin session, and a fake LeBoncoin collector that counts calls
- When, for each subtest in parallel, the raw client dials (start of the measure), sends complete request headers at once with `Content-Type: application/json` and a `Content-Length` of a valid body, then sends that body one byte per second:
  - `POST /api/v1/items` with a valid LeBoncoin URL;
  - `PUT /api/v1/settings` with a valid LeBoncoin session;
  - `PUT /api/v1/items/I1/purchase-goal` with a valid goal
- Then the connection is closed between 15 s and 20 s after the dial, the client has received zero response bytes (no status line, no API response), and afterwards `GET /api/v1/items` lists only `I1`, the LeBoncoin session is still not saved, `I1` still has no goal, and the fake collector was never called
- Corner cases:
  - the connection is still open at 14 s (the limit is not shorter);
  - the body is never complete: its last byte would arrive after 20 s;
  - the headers arrive within 5 s, so the 5 s header limit is not what closes the connection.

### TC-003 — The 15 s limit restarts for the next request on a kept-alive connection

- Covers: FR-HTTP-TMO-001 (UC-HTTP-TMO-001 alternative "Kept-alive connection")
- Level: integration (live server, raw client, session)
- Given a live server with one tracked item `I1` and a raw client connection
- When the client sends `GET /api/v1/items` at normal speed and reads its whole response (the end of the read starts the measure), waits 3 s, then on the same connection sends `PUT /api/v1/items/I1/purchase-goal` headers with a valid goal `Content-Length` and the body one byte per second
- Then the first response is a complete 200 with the item list; the connection is closed between 15 s and 20 s after the first response was read, with no second response, and `I1` still has no goal
- Corner cases:
  - the connection is still open 14 s after the first response (the 3 s pause does not consume part of a limit counted from the dial);
  - the first request on the connection is not affected by the later slow one.

### TC-004 — A 1 MiB body sent at normal speed is received and handled

- Covers: FR-HTTP-TMO-001, FR-HTTP-TMO-006 (UC-HTTP-TMO-001 alternative "Whole request received in time")
- Level: integration (live server, standard `http.Client`, session)
- Given a live server with one tracked LeBoncoin item `I1` without a goal
- When the client sends `PUT /api/v1/items/I1/purchase-goal` whose body is a valid goal padded with JSON whitespace to 1,048,000 bytes (just under the 1 MiB body limit)
- Then the answer is the usual success response of the goal endpoint, and `GET /api/v1/items/I1` shows the goal saved
- Corner cases:
  - a body of 1 MiB + 1 byte still gets the existing too-large error, not a closed connection (the body limits are unchanged);
  - the whole exchange takes well under 15 s.

### TC-005 — A client that does not read a large response is cut off at 60 s

- Covers: FR-HTTP-TMO-002 (UC-HTTP-TMO-002 main flow)
- Level: integration (server constructor around a test handler, raw client)
- Given a server built by the server constructor around a test handler that answers `GET /large` with `Content-Length: 268435456` (256 MiB) and writes it in 64 KiB chunks, recording the time it received the request and the time a write first fails; the raw client sets a 4 KiB receive buffer
- When the client sends the request headers at once, reads nothing for 62 s, then reads as fast as it can until the connection ends
- Then the handler's first failed write happens between 60 s and 65 s after it received the request, the client reads fewer than 268,435,456 body bytes and the connection ends with `io.EOF` or a reset
- Corner cases:
  - no write fails before 59 s (the limit is not shorter);
  - the response is far larger than the loopback buffers, so the server is really blocked by the slow client;
  - the test handler stops at the first failed write, so the test does not leak a goroutine.

### TC-006 — A response not started within 60 s of the request headers is never received

- Covers: FR-HTTP-TMO-002 (UC-HTTP-TMO-002 alternative "Request already handled")
- Level: integration (server constructor around a test handler, raw client)
- Given a server built by the server constructor around a test handler for `PUT /slow` that reads the small request body, records a change in a test variable, waits 61 s and then writes a 200 response
- When the raw client sends the whole request at once (the end of the send starts the measure) and waits for the answer
- Then the client receives zero response bytes and the connection is closed between 60 s and 65 s after the request was sent; the change recorded by the handler stays done
- Corner cases:
  - the counting starts from the request headers, so the time spent in the handler counts towards the 60 s;
  - the server keeps serving other connections during the wait: a `GET /ping` on a second connection at 30 s answers 200 at once.

### TC-007 — Request headers up to 64 KiB are handled as usual

- Covers: FR-HTTP-TMO-003 (UC-HTTP-TMO-003 main flow)
- Level: integration (live server, raw client, session)
- Given a live server with one tracked item `I1`
- When the raw client sends `GET /api/v1/items` with the session cookie and an `X-Padding` header of 60 KiB (61,440 characters)
- Then the answer is 200 with the item list containing `I1`
- Corner cases:
  - boundary: the same request with `X-Padding` sized so the request line and all header lines total exactly 65,536 bytes answers 200;
  - the large part is a cookie: a second 60 KiB cookie next to the session cookie gives 200 with the item list;
  - a page path (`GET /`) with a 60 KiB header gets its usual answer, not 431.

### TC-008 — Request headers over 68 KiB are refused with 431

- Covers: FR-HTTP-TMO-004 (UC-HTTP-TMO-003 alternative "Headers over 68 KiB")
- Level: integration (live server, raw client)
- Given a live server with one tracked item `I1`
- When the raw client sends `GET /api/v1/items` with the session cookie and an `X-Padding` header of 100 KiB (102,400 characters)
- Then the status is 431 (Request Header Fields Too Large), the body contains no item list (no `I1`), and the server closes the connection after the answer
- Corner cases:
  - boundary: headers totalling 69,633 bytes (one byte over 68 KiB) answer 431;
  - without a session, the same request answers 431, not 401;
  - the large part is a 100 KiB cookie: 431;
  - a write request (`POST /api/v1/items` with a valid body and a 100 KiB header) answers 431 and adds no item;
  - a page path (`GET /`) with a 100 KiB header answers 431;
  - repeated: three such requests in a row on new connections each answer 431, and a normal `GET /api/v1/items` afterwards answers 200.

### TC-009 — Saving the Claude token through the live server waits for a 19 s verification

- Covers: FR-HTTP-TMO-005, FR-HTTP-TMO-002 (UC-HTTP-TMO-004 main flow and alternatives)
- Level: integration (live server, standard `http.Client` without timeout, session; Claude faked through `http.DefaultTransport`, as in `claude_settings_test.go`)
- Given a live server with no Claude token saved
- When the client sends `PUT /api/v1/settings` with a new Claude token, and the fake Claude, for each subtest in parallel:
  - accepts the token after 19 s;
  - never answers (blocks until the outgoing request's context ends);
  - refuses the token (401) at once
- Then respectively:
  - the answer is the usual 200 success response with the Claude token saved and its hint (last 4 characters), received between 19 s and 21 s, and `GET /api/v1/settings/claude-token` then shows it saved with that hint;
  - the answer is the usual 502 `CLAUDE_UNREACHABLE` "The token could not be verified because Claude could not be reached. Try again.", received between 19.5 s and 25 s, and no token is saved;
  - the answer is the usual 422 `CLAUDE_TOKEN_REJECTED` at once, and no token is saved
- Corner cases:
  - the connection is not closed by the server before the answer in any subtest;
  - the response body is complete (valid JSON), not cut off.

### TC-010 — Settings shows the token as saved after a 19 s save

- Covers: FR-HTTP-TMO-005 (UC-HTTP-TMO-004 main flow)
- Level: e2e (Playwright, `tests/claude-token-settings.spec.ts`, mocked API)
- Given Settings open with no Claude token saved, and the mocked `PUT /api/v1/settings` answering its usual success response after 19 s
- When the operator pastes a token into the Claude input and selects Save
- Then Save shows progress for the whole wait, the modal then closes without an error notification, and on reopening Settings shows the Claude token as saved with its masked hint
- Corner cases:
  - no error or network error state appears during the 19 s wait (the interface has no shorter limit of its own);
  - the test sets its own Playwright timeout above the wait (for example 60 s).

### TC-011 — Add item, Refresh price, Refresh all and AI review answer at once while the work runs

- Covers: FR-HTTP-TMO-006 (UC-HTTP-TMO-005 main flow steps 2 to 5)
- Level: integration (live server, standard `http.Client`, session)
- Given a live server with a tracked LeBoncoin item `I1`, a fake LeBoncoin collector and a fake Claude review that both block until the test releases them, and a saved Claude token
- When the client, one after the other, sends `POST /api/v1/items` with a new LeBoncoin URL, `POST /api/v1/items/I1/refresh`, `POST /api/v1/items/refresh` and `POST /api/v1/items/I1/ai-review`
- Then each request answers its usual success status within 5 s while the related check or review is still blocked; after the test releases the fakes, the checks and the review complete as today (results visible in `GET /api/v1/items/{id}`)
- Corner cases:
  - the fakes stay blocked for 70 s in one subtest, longer than the 60 s write limit, and the requests still answer within 5 s with complete bodies;
  - repeated: a second Refresh price while the first is blocked answers its usual response at once.

### TC-012 — The 5 s header limit and 60 s idle limit are unchanged

- Covers: FR-HTTP-TMO-007 (UC-HTTP-TMO-001 alternatives "No complete headers within 5 s" and "Idle kept-alive connection"; UC-HTTP-TMO-005 step 6)
- Level: integration (live server, raw client)
- Given a live server
- When, for each subtest in parallel, the raw client:
  - dials and sends nothing;
  - dials and sends the headers of `GET /api/v1/items` one byte per second;
  - sends `GET /api/v1/items` at normal speed, reads the whole response, then sends nothing
- Then respectively:
  - the connection is closed between 5 s and 7 s after the dial;
  - the connection is closed between 5 s and 7 s after the dial, with no response;
  - the response is a complete answer, and the connection stays open at 16 s after the response and is closed between 60 s and 65 s after it
- Corner cases:
  - the idle connection is not closed by the 15 s read limit;
  - after the idle close, a new connection with `GET /api/v1/items` answers 200 (the next action works).

### TC-013 — Existing suites pass unchanged

- Covers: FR-HTTP-TMO-006 (UC-HTTP-TMO-005 postconditions)
- Level: integration and e2e (existing suites: `go test ./...`, `npx playwright test`)
- Given the change applied on top of `master`
- When both suites run
- Then they pass, and `git diff master` changes no assertion of an existing test (only new tests and the server constructor change are added)
- Corner cases:
  - the responsive layout and keyboard e2e cases (narrow screens) still pass, since no screen changes.
