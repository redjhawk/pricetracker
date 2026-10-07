# Technical specification: Amazon human browsing

Status: ready
Functional specification: [functional.md](functional.md), FR-AMZ-HUMAN-001–012 (ready, 2026-10-07)
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "Amazon searches (2026-10-07)" (request status, refresh after stop)
Companions: [Amazon searches technical](../amazon-searches/technical.md), [Amazon AI review technical](../amazon-ai-review/technical.md)

## Requirement mapping

`HUMAN-*` abbreviates `FR-AMZ-HUMAN-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-AMZ-HUMAN-001 | 001, 002 | Pure functions `inSearchWindow(t)` / `nextSearchWindowStart(t)` on `Europe/Paris`; only the search worker uses them | `internal/service/search_window.go` (new) | Table tests incl. 21:59, 22:00, 00:59, 01:00, 05:59, 07:59, 08:00 and both DST change days |
| TS-AMZ-HUMAN-002 | 003, 004, 005, 011 | One search worker goroutine processes due searches one at a time, in the fixed action order, with random pauses from an injectable `browsingClock` | `internal/service/search_worker.go` (new), `cmd/pricefollower/main.go` | Fake clock/sleeper test asserting the exact action sequence and pause bounds |
| TS-AMZ-HUMAN-003 | 006, 007, 008 | One global `amazonGate` serializes every Amazon request (search work and tracked items), counts consecutive failures and stops atomically at 5; state persisted | `internal/service/amazon_gate.go` (new), `internal/service/service.go`, `internal/store/amazon_requests.go` (new) | Concurrency test (`-race`): parallel tracked + search requests, 5th failure blocks the next caller; restart persists stop |
| TS-AMZ-HUMAN-004 | 006 | Failed action put aside; retried once at the end of the pass | `search_worker.go` | Test: item 3 fails, retry happens after item 30 |
| TS-AMZ-HUMAN-005 | 006, 012 | Failure log line with URL, time, HTTP status, response excerpt; no token, no cookies | `internal/amazon/collector.go`, `internal/amazon/search.go`, gate | Test captures `log` output: contains status and body excerpt, never the Claude token |
| TS-AMZ-HUMAN-006 | 008, 009, 010, 011 | Restart endpoint clears the block and makes the search due; `stoppedAt` kept until the next Amazon success | store, service, `internal/httpapi/searches.go`, frontend | Handler + service tests; Playwright for the Refresh button and info line |
| TS-AMZ-HUMAN-007 | 009 | Tracked Amazon checks while blocked send nothing, record nothing, log a skip; single refresh returns `409 AMAZON_REQUESTS_STOPPED`; the Amazon tab shows a stop notification | `service.go`, `TrackedItemsPage.tsx` | Service test: blocked gate → no collector call, no attempt row |

## Backend

### Request windows (`search_window.go`)

`var parisLocation = mustLoad("Europe/Paris")` (tzdata is already embedded by `main.go`). Windows are local `[22:00, 01:00)` (crossing midnight) and `[06:00, 08:00)`. `inSearchWindow(t)` converts `t` to Paris time and compares minutes since midnight: `>= 22:00 || < 01:00 || (>= 06:00 && < 08:00)`. `nextSearchWindowStart(t)` returns the first 06:00 or 22:00 Paris instant strictly after `t` (UTC result); computed with `time.Date` in the location so DST is handled by Go.

### Amazon gate (`amazon_gate.go`)

```go
type amazonGate struct {
    mu      sync.Mutex      // held for the whole Amazon request: serializes all Amazon traffic
    store   *store.Store
    failures int            // consecutive failures, mirrors the store
    blocked  bool
}
var errAmazonStopped = errors.New("Amazon requests are stopped")
// do runs request under the lock unless blocked; request reports whether the
// Amazon request failed. Count, block and persistence happen under the same lock.
func (g *amazonGate) do(ctx context.Context, request func() amazonOutcome) (amazonOutcome, error)
func (g *amazonGate) restart(ctx context.Context) error // blocked=false, failures=0 (stoppedAt kept)
```

- The mutex is held during the network call, which is intended: it makes Amazon traffic strictly sequential and makes the 5th failure and the block one atomic step, so no other request (tracked or search) can start between them (HUMAN-008). It is never held with a DB transaction open across the network call; persistence happens after the call, still under the gate lock, in a short write.
- Failure = `request_error` result (network error, non-2xx other than 404/410, unexpected content type, captcha/robot check) or, for the results page, an unparsable page (no product found). `unavailable` (404/410 or "no longer available") and `price_not_found` are Amazon answers, not failures, and reset the count like a success. AI review errors never pass through the gate (FR-AMZ-AIR-007).
- On reaching 5: `blocked = true`, store `stopped_at = now`, log `Amazon requests stopped after 5 consecutive failures at <time>`.
- On success: `failures = 0`; if `stopped_at` is set and not blocked, clear it (information line disappears after a successful refresh, D-19).
- Persistence: table `amazon_request_state (id INTEGER PRIMARY KEY CHECK (id = 1), consecutive_failures INTEGER NOT NULL DEFAULT 0, blocked INTEGER NOT NULL DEFAULT 0, stopped_at TEXT)`, one row created at open; loaded into the gate by `service.New`. The stop survives a process restart.
- Tracked items: `collectReserved` wraps the Amazon `collector.Collect` call (including its second-hand request) in `gate.do`. `errAmazonStopped` → no attempt recorded, log `Amazon requests stopped; check of item <id> skipped`; the scheduler has already moved `next_check_at`. `RefreshItem` on an Amazon item while blocked → `409 AMAZON_REQUESTS_STOPPED`; `RefreshAll` keeps its contract and skips Amazon items silently in the worker, the Amazon tab notification explains it. With the gate, the existing 2 refresh slots still run LeBoncoin in parallel; Amazon requests queue on the gate (accepted: "there's no hurry"). LeBoncoin never uses the gate.

### Search worker (`search_worker.go`)

```go
type browsingClock interface {
    Now() time.Time
    Sleep(ctx context.Context, d time.Duration) bool // false when ctx is cancelled
}
type searchWorker struct { service *Service; clock browsingClock; pause func() time.Duration }
// default pause: 30s + rand.Intn(91)s  → 30..120 s inclusive (HUMAN-003)
```

`Service.RunSearchWorker(ctx)` is started by `main.go` next to `RunScheduler` and joined on shutdown. Loop: if not in a window, or the gate is blocked, sleep until `min(nextSearchWindowStart, now+1 min)` and re-check; otherwise take the earliest due search (`DueSearches`: `next_run_at <= now ORDER BY next_run_at, added_at`, all owners, one at a time — HUMAN-004) and run one pass; if none is due, sleep 1 min.

One pass of search S (every step first checks: context, window, gate not blocked, S still exists; otherwise return and keep the persisted position):

1. If `pass_position == 0`: open the results page through the gate (`FetchSearch`). Success with no capture yet → `CaptureSearch`; success with capture → nothing stored (the set is frozen, the request only follows the human order). Failure → `SetSearchError`, put "results" aside. Set `pass_position = 1`.
2. For each item at position `p ≥ pass_position` in captured order: pause; open the item (`collector.CollectProduct` through the gate); on success record immediately (`store.RecordCollection`) and call the AI review hook (Amazon AI review technical); pause ("reading"); log `close item <asin>`; pause before the next item happens at the top of the loop; persist `pass_position = p + 1`. A failure is logged, recorded as a failed attempt (keeps the last price) and the item is put aside.
3. After the last item: retry each put-aside action once, in order, with a pause before each (D-18). Remaining failures wait for the next pass.
4. Complete: `pass_position = 0`, `next_run_at = nextSearchWindowStart(now)` — one pass per window, i.e. twice a day like tracked items.

The put-aside list is in memory: after a process restart mid-pass the pass resumes at `pass_position` and earlier failures are retried at the next pass. With no capture (results failed twice) the pass ends at step 1 and the next window tries again.

`CollectProduct` is the existing product-page parse without the extra second-hand offer request (one request per item, HUMAN-004); second-hand status comes from the product page only (`check_error` when absent). It also returns product details for the review (Amazon AI review technical).

### Logs (HUMAN-006, 012)

`model.CollectionResult` gains in-memory fields `RequestURL string`, `HTTPStatus int` (0 for network errors), `ResponseExcerpt string` (first 2 KB of the body, control characters replaced). The gate logs on every failure: `Amazon request failed url=<url> at=<RFC3339> status=<n> error=<message> response=<excerpt>`. Requests to Amazon carry no token or cookie; the Claude token is never passed to this code.

### Restart (HUMAN-010, 011)

`Service.RestartAmazonSearch(ctx, searchID)`: owned search, else 404; if the gate is neither blocked nor has `stopped_at` → `409 AMAZON_NOT_STOPPED`; otherwise `gate.restart` and set the search `pass_position = 0`, `next_run_at = now`. Work starts immediately in a window, else at the next window (search shows `waiting`). Other searches resume at their own `next_run_at`. Price history and reviews are untouched.

## API

API_SPECIFICATION.md "Amazon searches (2026-10-07)": `amazonRequests` object (`stopped`, `stoppedAt`, `consecutiveFailures`) in `GET /api/v1/amazon-searches` and in `GET /api/v1/amazon/requests`; `POST /api/v1/amazon-searches/{id}/refresh`; `409 AMAZON_REQUESTS_STOPPED` on `POST /api/v1/items/{id}/refresh` for Amazon items. Other item endpoints unchanged.

## Frontend

- Searches list: when `amazonRequests.stoppedAt` is not null, each row in state `stopped` (or `waiting` after a restart while `stoppedAt` remains) shows a `Button` kind tertiary size sm "Refresh" (only while `stopped`) and, in the row below, an `InlineNotification` kind warning lowContrast hideCloseButton: "Amazon requests were stopped after repeated failures on <date time>." After a refresh: reload the list; the line disappears when the server clears `stoppedAt`.
- Amazon tab (`TrackedItemsPage`): fetch `GET /api/v1/amazon/requests` with the item list; when `stopped`, show a warning notification "Amazon requests are stopped since <date time> after repeated failures. Restart them from the Amazon searches tab." Item refresh `409 AMAZON_REQUESTS_STOPPED` shows the API message in the existing refresh error slot.
- Dates via the existing `formatDate`; notifications have text, not colour only.

## Scope and refactoring

- Refactoring R-2 (performed, part 5): route the Amazon branch of `collectReserved` through the gate. Reason: HUMAN-008 requires a single stop for all Amazon traffic. Risk: tracked checks become sequential; acceptable at about 100 items.
- Declined: a generic scheduler abstraction; the existing `RunScheduler` stays as is.

## Verification and unresolved questions

- Unit: window table tests; gate (`go test -race`): 4 failures + success + 4 failures not blocked; 5 consecutive → blocked, next `do` returns `errAmazonStopped` without calling the request; concurrent callers never exceed one in flight; persistence across reopen.
- Worker with fake clock and fake collector: action order `results, pause, item1, pause, close1, pause, item2…`; pauses within [30 s, 120 s]; window end stops and the next window resumes at the saved position; deleted search stops the pass; failed item retried once after the last item; 5 failures stop tracked checks too.
- Log test: failure log contains status and excerpt, not the Claude token.
- QA: real windows cannot be waited for interactively; QA checks the `waiting` state and `waitingUntil` outside windows and the logs, relies on the fake-clock tests for in-window behavior, and records the limitation. No test-only configuration is added. No functional question is open.
