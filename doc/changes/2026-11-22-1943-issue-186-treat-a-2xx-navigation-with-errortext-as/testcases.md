# Test cases: success response the browser does not open (issue #186)

Sources: [functional specification](../../specifications/amazon-human-checks/functional.md) (FR-AMZ-HCHK-011, FR-AMZ-HCHK-012), use case [UC-AMZ-HCHK-005](../../use-cases/amazon-human-checks/UC-AMZ-HCHK-005-amazon-check-answered-with-page-the-browser-does-not-open.md).

Conventions:

- **Browser tests** run the real Chromium (`amazontest.Browser`) against a local `amazontest.Server` serving Amazon-like fixture responses, as in `internal/amazon/browser_test.go`. No test contacts amazon.*.
- A **failure** result is `Result: "request_error"`, `Message` = the Amazon check failed message, `SecondHandStatus: "check_error"`, `RequestURL` ending in `/dp/<ASIN>`.
- Integration tests through the API use a real SQLite database in `t.TempDir()`, the HTTP handlers (`GET /api/v1/items`, `GET /api/v1/items/{id}`, `POST /api/v1/items/{id}/refresh`, `POST /api/v1/items/refresh`) and the real browser collector, as in `internal/httpapi/amazon_browser_profile_test.go`; scheduled checks are triggered directly through the service with an injected clock.
- The UI does not change. No new Playwright case: the user-visible states (retrieval issue, second-hand check error, stop warning, refresh failed) are existing ones already covered by `tests/item-refresh.spec.ts` and `tests/platform-tabs.spec.ts`, which must keep passing. Narrow screens and keyboard are therefore not relevant.
- Existing browser tests of issue #168 (`TestBrowserCollector_TC004`, `TC006`, `TC007`) must keep passing: a success response opened as a page still gives price, price not found or listing unavailable.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-AMZ-HCHK-011 | TC-001, TC-002, TC-004, TC-005 |
| FR-AMZ-HCHK-012 | TC-003, TC-006 |

| Use case | Test cases |
| --- | --- |
| UC-AMZ-HCHK-005 | TC-001–TC-006 |

## Browser collector

### TC-001 — A 200 page offered as a download is a failure, not "price not found"

- Covers: FR-AMZ-HCHK-011; UC-AMZ-HCHK-005 (main flow steps 1–4)
- Level: integration (browser test, `internal/amazon`)
- Given a fixture server answering `/dp/B0PRICE001` with HTTP 200, `Content-Type: text/html; charset=utf-8`, `Content-Disposition: attachment; filename="page.html"` and the body of the `price-found.html` fixture (a page whose price the parser reads)
- When the browser collector checks the item
- Then the result is a failure (not `price_not_found`, not `success`, not `unavailable`), and the captured log contains the request URL and `Amazon page could not be opened: net::` followed by the browser's error text
- Corner cases:
  - the body contains a readable price, proving the price is not taken from a page the browser did not open;
  - repeated action: a second check with the same collector after the server switches to a normal 200 `text/html` page (no `Content-Disposition`) succeeds with the fixture price, so the failed download leaves the browser usable;
  - `Content-Disposition: inline` with the same body is opened as a page and gives `success` with the fixture price (only an attachment is a failure).

### TC-002 — A 204 or 205 response without content is a failure

- Covers: FR-AMZ-HCHK-011; UC-AMZ-HCHK-005 (main flow step 2, "no content")
- Level: integration (browser test, `internal/amazon`)
- Given a fixture server answering `/dp/B0PRICE001` with HTTP 204 (subtest) or HTTP 205 (subtest) and an empty body
- When the browser collector checks the item
- Then the result is a failure, never `price_not_found` or `unavailable`, and the captured log contains the request URL and the browser's error
- Corner cases: empty body; a status at each end of the "no content" set (204, 205).

### TC-003 — A non-success response with an empty body is still classified by its status

- Covers: FR-AMZ-HCHK-012; UC-AMZ-HCHK-005 (alternative "non-success status not opened as a page")
- Level: integration (browser test, `internal/amazon`)
- Given a fixture server answering `/dp/B0PRICE001` with an empty body and, per subtest, HTTP 404, 410, 503 or 403
- When the browser collector checks the item
- Then 404 and 410 give `Result: "unavailable"` with the listing-unavailable message; 503 and 403 give a failure whose `HTTPStatus` is that status (the gate's `status=<code>` log line is asserted in TC-006)
- Corner cases: empty body (Chromium shows its own error page and the navigation ends with an error text); 403 and 503 cover both a client and a server error.

## Failure count, stop and user-visible result

### TC-004 — A download response is counted and does not reset the consecutive failure count

- Covers: FR-AMZ-HCHK-011; UC-AMZ-HCHK-005 (main flow steps 3–5, alternative "fifth consecutive failure")
- Level: integration (real SQLite, HTTP handlers, real browser collector, `internal/httpapi`)
- Given two tracked Amazon items, each with a recorded last successful price (1999 cents) and an available second-hand offer, and a fixture server whose `/dp/<ASIN>` answer is switched per check
- When five scheduled checks run in sequence with answers: HTTP 503 page, HTTP 503 page, HTTP 200 `text/html` with `Content-Disposition: attachment`, HTTP 503 page, HTTP 503 page
- Then after checks 1–4 no item is stopped; after the 5th check every tracked Amazon item is stopped (`amazonStoppedAt` set in `GET /api/v1/items`); throughout, each item's latest price stays 1999 cents and its last second-hand offer is kept; no stored collection for the attachment check has result `price_not_found`; the log contains 5 `Amazon request failed` lines, the third with the request URL and the browser's error
- Corner cases:
  - the attachment check is in the middle of the sequence, so a reset by it would prevent the stop;
  - the attachment check as the 5th failure (4 × 503 then the attachment) also stops every item;
  - protected mode with the two items owned by two users: both are stopped.

### TC-005 — Refresh price and Refresh all prices answered with a download show a failed refresh

- Covers: FR-AMZ-HCHK-011; UC-AMZ-HCHK-005 (main flow step 6, alternative "Refresh price")
- Level: integration (real SQLite, HTTP handlers, real browser collector, `internal/httpapi`)
- Given a tracked Amazon item with a last successful price and second-hand offer, and a fixture server answering HTTP 200 `text/html` with `Content-Disposition: attachment`
- When the user calls `POST /api/v1/items/{id}/refresh`, then `POST /api/v1/items/refresh`
- Then `GET /api/v1/items/{id}` shows the item in the retrieval error state with last attempt `request_error` and second-hand status `check_error`, not "price not found"; the latest price and last second-hand offer are unchanged; the item list shows the same retrieval issue
- Corner cases:
  - Refresh price on a stopped item answered with the download: the refresh fails and the item stays stopped (no Amazon answer was received);
  - repeated Refresh price: each attempt is a separate counted failure, never a reset.

### TC-006 — A non-success response with an empty body keeps its effect on the failure count

- Covers: FR-AMZ-HCHK-012; UC-AMZ-HCHK-005 (alternative "non-success status not opened as a page")
- Level: integration (real SQLite, HTTP handlers, real browser collector, `internal/httpapi`)
- Given two tracked Amazon items and 4 consecutive failures already recorded by the gate (4 checks answered HTTP 503 with a page)
- When the next scheduled check is answered HTTP 404 with an empty body, followed by 4 more checks answered HTTP 503 with an empty body, then one more HTTP 503 with an empty body
- Then the 404 check shows the item as listing unavailable and resets the count (no item is stopped after the following 4 failures); the next failure (the 5th after the reset) stops every Amazon item; each 503 failure is logged with `status=503`
- Corner cases: empty body for both the resetting status (404) and the failing status (503); 410 with an empty body behaves as 404 (subtest).
