# Technical specification: HTTP server read and write timeouts and header size limit

Status: ready (issue #273)
Functional specification: [functional.md](functional.md) (FR-HTTP-TMO-001..007).
Use cases: [UC-HTTP-TMO-001..005](../../use-cases/http-server-timeouts/).
Test cases: [testcases.md](../../changes/2026-11-24-0712-issue-273-http-server-add-read-and-write-timeouts/testcases.md) (TC-001..TC-013).
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "HTTP server limits (issue #273)".

## Requirement mapping

| Technical ID | Functional IDs | Technical solution | Files |
| --- | --- | --- | --- |
| TS-HTTP-TMO-001 | FR-HTTP-TMO-001, -002, -003, -004, -007 | Server constructor `httpapi.NewHTTPServer(cfg, handler)` returns the one `*http.Server` with `ReadHeaderTimeout` 5 s, `ReadTimeout` 15 s, `WriteTimeout` 60 s, `IdleTimeout` 60 s, `MaxHeaderBytes` 64 KiB. | `internal/httpapi/http_server.go` (new) |
| TS-HTTP-TMO-002 | FR-HTTP-TMO-001 | A request body whose read passes the read deadline aborts the handler with `http.ErrAbortHandler`, so no response is written and nothing is handled. Handlers that never read the body are not affected. | `internal/httpapi/http_server.go`, `internal/httpapi/server.go` |
| TS-HTTP-TMO-003 | FR-HTTP-TMO-001..007 | `cmd/pricefollower/main.go` builds its server only with `httpapi.NewHTTPServer`; no `http.Server{` literal is left in `cmd/`. | `cmd/pricefollower/main.go` |
| TS-HTTP-TMO-004 | FR-HTTP-TMO-003, -004 | Header size: Go reads at most `MaxHeaderBytes` + 4096 = 69,632 bytes of request line and headers and answers `431` itself above that. | (net/http behaviour, no code) |
| TS-HTTP-TMO-005 | FR-HTTP-TMO-005, -006 | Handler duration check: no handler runs longer than 60 s; the longest is `PUT /api/v1/settings` (Claude verification bounded at 20 s). Long work stays in background workers. No code change. | (review, no code) |
| TS-HTTP-TMO-006 | FR-HTTP-TMO-005, -006 | Frontend unchanged: `fetch` has no time limit of its own, so Settings waits for a slow save. | (none) |

## Backend

### Server constructor (`internal/httpapi/http_server.go`, new)

```go
// Limits of the application's HTTP server (issue #273).
const (
	serverReadHeaderTimeout = 5 * time.Second
	serverReadTimeout       = 15 * time.Second // whole request: headers and body
	serverWriteTimeout      = 60 * time.Second // from the received headers; Settings save waits up to 20 s for Claude
	serverIdleTimeout       = 60 * time.Second
	serverMaxHeaderBytes    = 64 << 10
)

// NewHTTPServer returns the application's HTTP server: handler on the configured host and port with the
// request limits above. One server serves development and production mode.
func NewHTTPServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Handler:           handler,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
		MaxHeaderBytes:    serverMaxHeaderBytes,
	}
}
```

- It lives in `httpapi` because `config/` is outside the developer's paths and `cmd/` cannot be imported by tests. `Handler` is exactly the given handler (TC-001 compares it), so the abort of TS-HTTP-TMO-002 sits in `httpapi.New`, not here.
- `cmd/pricefollower/main.go`: replace the `&http.Server{...}` literal with `server := httpapi.NewHTTPServer(cfg, api.Handler())`; drop the `net` import, which becomes unused (`time` stays for the shutdown timeout). Start-up, logging and the 5 s shutdown are unchanged.

### Go 1.25 `net/http` behaviour relied on (read in `net/http/server.go`)

- `readRequest` takes `t0 = time.Now()` and sets the read deadline to `t0 + 5 s` for the headers, then `t0 + 15 s` once the headers are read; the write deadline is set to *headers received* + 60 s. This gives FR-HTTP-TMO-001, -002 and -007.
- On a kept-alive connection the server first waits with the idle deadline (60 s) for the first 4 bytes of the next request (`Peek(4)`), then starts `t0`. So the 15 s count of a second request starts when its bytes arrive, not when the previous response was sent (TC-003: client pauses 3 s, close ≈ 18 s after the first response, inside 15–20 s). An idle connection is closed only by the 60 s idle limit (TC-012).
- A header read that times out is a "common net read error": the server closes without any reply. When part of the request line has arrived, `bufio.Reader.ReadLine` returns it with a nil error at the deadline, so the malformed line is answered with Go's plain-text `400 Bad Request` before the close (TC-012 accepts both).
- `net/http` calls the handler as soon as the headers arrive. Only handlers that read the body (login, create user, add item, purchase goal, settings) can be stopped by the 15 s limit (TS-HTTP-TMO-002). A handler that never reads the body (refreshes, AI review, `DELETE`, `GET`) runs at once; afterwards `net/http` tries to discard the unread body before it writes the header, hits the read deadline, and still sends the response with `Connection: close`.
- Once the body is fully read (or for a request without body), `startBackgroundRead` clears the read deadline. A handler that runs past 15 s (the 19 s Claude verification) therefore keeps its request context; the 15 s limit does not cancel it (TC-009).
- A write after the write deadline fails; the server closes the connection when the handler returns (TC-005, TC-006). Other connections are not affected.
- Header size: the connection reader is limited to `MaxHeaderBytes + 4096` (= 69,632) bytes until the headers are parsed. Headers (request line, header lines and final empty line, CRLFs included) of at most 69,632 bytes are parsed; more answers `HTTP/1.1 431 Request Header Fields Too Large` with `Content-Type: text/plain; charset=utf-8`, `Connection: close`, body `431 Request Header Fields Too Large`, then `CloseWrite` and a 500 ms wait before the close. This happens before routing, so session, method and path do not matter (TC-007, TC-008).

### Abort a request whose body arrives too late (TS-HTTP-TMO-002)

Without this, a body read that passes the 15 s deadline returns a timeout error to the handler's JSON decoder, which answers `400 INVALID_JSON`; that small response is still sent (the write deadline is 60 s), contrary to FR-HTTP-TMO-001 ("no API response").

In `internal/httpapi/http_server.go`:

```go
// abortLateBodies ends the request without a response when its body is not received within the server's
// read limit (FR-HTTP-TMO-001); the handler would otherwise answer 400 INVALID_JSON.
func abortLateBodies(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.Body != http.NoBody {
			r.Body = lateBody{r.Body}
		}
		next.ServeHTTP(w, r)
	})
}

type lateBody struct{ io.ReadCloser }

func (b lateBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if errors.Is(err, os.ErrDeadlineExceeded) {
		panic(http.ErrAbortHandler) // net/http closes the connection, writes nothing and logs nothing
	}
	return n, err
}
```

- `httpapi.New`: `s.handler = securityHeaders(abortLateBodies(mux))`.
- The timeout error from the connection (`*net.OpError` wrapping `os.ErrDeadlineExceeded`) reaches the body reader unchanged through `bufio.Reader` and `net/http`'s `body`; `http.MaxBytesReader`, set by handlers on top of `lateBody`, passes it through too.
- The panic unwinds the handler while it decodes the body, before any service call, so nothing is saved or started. The handler has written nothing, so nothing is sent; `net/http` recovers `ErrAbortHandler` silently and closes the connection at once (≈15 s, TC-002).
- Constraint: request bodies must be read in the handler goroutine (true for every handler today: `auth.go`, `server.go`, `purchase_goal.go`, `settings.go`); a panic in another goroutine would stop the process. No handler code uses `recover`.
- Other body errors (client gone, too large, malformed) are unchanged. `httptest` recorder tests never hit a deadline, so existing tests are unaffected.

### Handler durations (TS-HTTP-TMO-005)

- `PUT /api/v1/settings`: `claude.Client.Verify` bounds the verification with `context.WithTimeout(ctx, 20*time.Second)`; the LeBoncoin session part is parsed locally. Total well under 60 s; a 19 s acceptance answers 200, a silent Claude answers `502 CLAUDE_UNREACHABLE` at ≈20 s.
- `POST /api/v1/items`, `POST /api/v1/items/{id}/refresh`, `POST /api/v1/items/refresh`, `POST /api/v1/items/{id}/ai-review` start their work on service workers and answer at once (unchanged). Login waits at most for a few PBKDF2 checks (bounded queue). Static files are small.
- No handler change is needed.

### Validation, persistence, migrations, concurrency

- No validation rule, schema, migration or stored state changes. The limits are per connection; `net/http` serves each connection in its own goroutine, so a slow client never blocks others.
- Values are constants, not configuration (the issue fixes them; no setting is added).

## API

- No endpoint, request, response or JSON error change. New section "HTTP server limits (issue #273)" in `API_SPECIFICATION.md` (done in this stage) documents the limits and the plain-text `431`, the only response not in the JSON error shape.
- Compatibility: normal browsers, the Vite development proxy and Cloudflare send headers far below 64 KiB and bodies far faster than 15 s; nothing changes for them.

## Frontend

Not affected: no screen, message, state or request change. `src/` has no request time limit (no `AbortSignal.timeout`/timer around `fetch`), so Save keeps its progress state for the whole 19 s wait; a closed connection reaches the existing network-error handling. TC-010 only adds an e2e test.

## Files to change

- `internal/httpapi/http_server.go` (new): constants, `NewHTTPServer`, `abortLateBodies`, `lateBody`.
- `internal/httpapi/server.go`: `New` wraps the mux with `abortLateBodies`.
- `cmd/pricefollower/main.go`: use `httpapi.NewHTTPServer`.
- Already done in this stage: `API_SPECIFICATION.md`.

## Tests

### Go: `internal/httpapi/http_server_limits_test.go` (new, package `httpapi`)

Names `TestHTTPServerLimits_TC0NN_...`. Shared helpers in the file:

- `limitsNet`: one fake `http.DefaultTransport` (type `claudeTransport` of `claude_settings_test.go`) for all live cases. It dispatches by host and value so parallel cases do not interfere: `api.anthropic.com` by bearer token (accept after 19 s; block until `r.Context().Done()` and return its error; 401 at once; for reviews, block until released then answer a valid review) and LeBoncoin by listing ID in the URL path (count calls; answer `maskedListingHTML`-like page; or block until released or the request context ends). Each case uses its own tokens and listing IDs.
- `openLimitsServer(t)`: store in `t.TempDir()` with the config of `openSecretServer`, `protectedUsers` for alice's session cookie, items inserted with `database.Insert` and `OwnerID: aliceID` (LeBoncoin listings, IDs per case), then `srv := NewHTTPServer(config.Config{Host: "127.0.0.1"}, api.Handler())`, `net.Listen("tcp", "127.0.0.1:0")`, `go srv.Serve(listener)`, cleanup `srv.Close()`. The scheduler is not started.
- `serveTestHandler(t, handler)`: same constructor and listener around a test handler (TC-005, TC-006).
- Raw client: `net.Dial`, requests written by hand (`Host`, `Cookie: pricefollower_session=...`, `Content-Type`, `Content-Length`), a `time.Now()` start per case; "closed" = a read returns `io.EOF` or `syscall.ECONNRESET` (`errors.Is`). To measure a close time, read in a loop with a generous `SetReadDeadline` and record when the read ends; zero response bytes means that read returned no data.
- Standard clients use `&http.Client{Transport: &http.Transport{}}`: never `http.DefaultTransport`, which is the fake.

Structure: TC-001 is its own serial test. All live and long cases (TC-002..TC-009, TC-011, TC-012) are subtests of one serial parent `TestHTTPServerLimits_Live` that installs `limitsNet` and restores `http.DefaultTransport` in its `t.Cleanup` (runs after all subtests); each long subtest calls `t.Parallel()`. No other test of the package runs meanwhile, so swapping `DefaultTransport` is safe; the parent takes about 70 s (TC-011's 70 s block is the longest).

| Test case | Implementation notes |
| --- | --- |
| TC-001 | `NewHTTPServer` with `config.Config{Host: "127.0.0.1", Port: 8081, Development: d}` for both `d`; handler `http.NewServeMux()` (pointer, comparable with `==`); assert the five limits, `MaxHeaderBytes == 65536`, `Addr == "127.0.0.1:8081"`, `Handler == mux`. The "no `http.Server{` literal in `cmd/`" corner is a review check, optionally a test reading `../../cmd/pricefollower/main.go` and asserting it does not contain `http.Server{`. |
| TC-002 | Three parallel subtests; headers at once, body bytes every 1 s from a goroutine stopped by cleanup; assert open at 14 s, closed in [15 s, 20 s], zero bytes read; then `GET /api/v1/items`, `GET /api/v1/settings/leboncoin-session` (`saved: false`), `GET /api/v1/items/I1` (no goal) through a standard client, and `limitsNet` LeBoncoin count 0. |
| TC-003 | One connection: full `GET /api/v1/items` read with `http.ReadResponse(bufio.Reader)`, 3 s pause, slow purchase-goal `PUT`; expected close ≈ 18 s after the first response (see kept-alive note); assert open at 14 s, closed in [15 s, 20 s], no second status line, goal unchanged. |
| TC-004 | Standard client; body `{"purchaseGoal":"..."}` padded with spaces to 1,048,000 bytes → usual purchase-goal success; details show the goal; 1,048,577-byte body → `413 REQUEST_TOO_LARGE` (via `assertError`-like decode of the response). Assert elapsed < 15 s. |
| TC-005 | `serveTestHandler`: `GET /large` sets `Content-Length: 268435456`, writes 64 KiB chunks, records request time and first failed write time, returns at the first error. Raw client: `(*net.TCPConn).SetReadBuffer(4096)`, sends the request, sleeps 62 s, then reads to the end. Assert the first failure in [60 s, 65 s] after the request time, bytes read < 268,435,456, end is EOF or reset. |
| TC-006 | `serveTestHandler`: `PUT /slow` reads the body, sets an `atomic.Bool`, sleeps 61 s, writes 200; `GET /ping` answers 200. Second connection at 30 s gets 200 at once. First client: zero bytes, closed in [60 s, 65 s] after the send; the flag stays set. |
| TC-007 | Raw client; `X-Padding` of 61,440 bytes; exact boundary: compute the padding so request line + header lines + final CRLF = 65,536 bytes; a second 60 KiB cookie; `GET /` gets its usual frontend answer (`index.html`, or the "not embedded" 404 when the test binary has no build), asserted not 431. Responses parsed with `http.ReadResponse`. |
| TC-008 | Raw client writes the whole request, then reads with `http.ReadResponse`: 431, body has no `I1`, then EOF/reset. Boundary 69,633 bytes; without cookie; 100 KiB cookie; `POST /api/v1/items` with valid body (item count unchanged afterwards); `GET /`; three in a row then a normal `GET /api/v1/items` → 200. |
| TC-009 | Three parallel subtests, standard client without timeout, `PUT /api/v1/settings` `{"claudeToken":{"value":"<per-subtest token>"}}`: 200 in [19 s, 21 s] with `claudeToken.saved` and hint = last 4 characters, then `GET /api/v1/settings/claude-token` saved; 502 `CLAUDE_UNREACHABLE` with the API message in [19.5 s, 25 s], not saved; 422 `CLAUDE_TOKEN_REJECTED` at once, not saved. Bodies decode as complete JSON. |
| TC-010 | See e2e below. |
| TC-011 | Claude token saved with `storeSecrets` (or a fast-accept token); LeBoncoin and review fakes block on a per-subtest release channel. Each of the four requests answers its usual status (201, 202, 202, 202) within 5 s with a complete body; a second Refresh price answers its usual response at once. Release (one subtest after a few seconds, one after 70 s) and poll `GET /api/v1/items/{id}` until the check and the review are no longer pending. The LeBoncoin client's own 15 s timeout may end a 70 s block first: the subtest then expects the usual finished-check outcome, not a pending one. |
| TC-012 | Three parallel subtests: silent dial → closed in [5 s, 7 s]; headers one byte per second → closed in [5 s, 7 s], zero bytes; full request and response, then silent: open at 16 s, closed in [60 s, 65 s]; then a new connection `GET /api/v1/items` → 200. |
| TC-013 | Existing suites (`go test ./...`, `npx playwright test`) unchanged. |

### e2e: `tests/claude-token-settings.spec.ts`

TC-010: new test using `mockApi(context, noToken)` with `api.putGate = new Promise((r) => setTimeout(r, 19_000))`; `test.setTimeout(60_000)`. Fill the token, Save; assert `Saving…` disabled and "Saving settings…" visible, and that no error notification appears while waiting (for example at 10 s and 18 s); `await expect(dialog).toBeHidden({ timeout: 30_000 })`; reopen and assert the masked hint text, as in the existing TC-013 (#265) test. No change to `playwright.config.ts`.

## Refactoring

- Moving the `http.Server` construction from `cmd/pricefollower/main.go` to `httpapi.NewHTTPServer`. Reason: tests must build the exact production server (TC-001..TC-012) and `cmd/` cannot be imported. Scope: one function, constants and three imports. Risk: low; same fields plus the three new ones.

## Manual follow-ups

None. `go test ./...` grows by about 70 s of wall time, inside the default 10 min package timeout of the gates.
