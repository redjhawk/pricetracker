# Test cases: no used-offer request from the browser's error page (issue #187)

Sources: [functional specification](../../specifications/amazon-human-checks/functional.md) (FR-AMZ-HCHK-013, with FR-AMZ-HCHK-012), use case [UC-AMZ-HCHK-006](../../use-cases/amazon-human-checks/UC-AMZ-HCHK-006-unavailable-listing-page-the-browser-does-not-open-sends-no-used-offer-request.md).

Conventions:

- **Browser tests** run the real Chromium (`amazontest.Browser`) against a local `amazontest.Server` serving Amazon-like fixture responses, as in `internal/amazon/browser_test.go`. No test contacts amazon.*.
- A **used-offer request** is any request the fixture server receives on the path `/gp/aod/ajax/ref=tmm_pap_used_aod_0` (both used-offer URLs), counted with `Server.Count`; an `OPTIONS` request on that path (CORS preflight) also counts.
- A **used-offer failure log** is a log line starting with `Amazon used-offer check failed for`.
- Integration tests through the API use a real SQLite database in `t.TempDir()`, the HTTP handlers (`GET /api/v1/items`, `GET /api/v1/items/{id}`, `POST /api/v1/items/{id}/refresh`, `POST /api/v1/items/refresh`) and the real browser collector, as in `internal/httpapi/amazon_browser_navigation_test.go`; scheduled checks are triggered directly through the service with an injected clock.
- The UI does not change. No new Playwright case: **Listing unavailable** and the second-hand check error are existing states already covered by `tests/item-refresh.spec.ts` and `tests/platform-tabs.spec.ts`, which must keep passing. Narrow screens and keyboard are therefore not relevant.
- Existing tests of issues #168 and #186 (`TestBrowserCollector_TC004`, `TC006`, `TC007`, `TestAmazonBrowserNavigation_TC003`–`TC006`) must keep passing: FR-AMZ-HCHK-012 is unchanged.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-AMZ-HCHK-013 | TC-001, TC-002, TC-003, TC-004 |
| FR-AMZ-HCHK-012 | TC-001, TC-003, TC-004 |

| Use case | Test cases |
| --- | --- |
| UC-AMZ-HCHK-006 | TC-001–TC-004 |

## Browser collector

### TC-001 — An empty-body 404 or 410 product page is unavailable and sends no used-offer request

- Covers: FR-AMZ-HCHK-013, FR-AMZ-HCHK-012; UC-AMZ-HCHK-006 (main flow steps 1–5)
- Level: integration (browser test, `internal/amazon`)
- Given a fixture server answering `/dp/B0PRICE001` with an empty body and, per subtest, HTTP 404 or HTTP 410, and answering the used-offer path with a valid used-offer fixture (so a request, if sent, could be detected and would succeed)
- When the browser collector checks the item
- Then the result is `Result: "unavailable"` with the listing-unavailable message, `HTTPStatus` is the page status, `SecondHandStatus` is `"check_error"` with no second-hand amount or condition; the fixture server received 0 used-offer requests (including no `OPTIONS` preflight); the captured log has no used-offer failure line
- Corner cases:
  - both unavailable statuses (404, 410) with an empty body (Chromium shows its error page);
  - the used-offer fixture is valid, proving the `check_error` comes from skipping the requests, not from a failed request;
  - repeated action: a second check with the same collector after the server switches `/dp/B0PRICE001` to the normal `price-found.html` page without a second-hand offer sends the used-offer requests again (2 requests on the used-offer path) and reads the used offer, so the skip does not leak into the next check.

### TC-002 — A 404 or 410 product page with content still gets its used-offer requests

- Covers: FR-AMZ-HCHK-013 (regression case of the acceptance criterion); UC-AMZ-HCHK-006 (alternative "404 or 410 page that the browser opens")
- Level: integration (browser test, `internal/amazon`)
- Given a fixture server answering `/dp/B0PRICE001` with an HTML body (an Amazon-like "page not found" page) and, per subtest, HTTP 404 or HTTP 410, and the used-offer path with a valid used-offer fixture
- When the browser collector checks the item
- Then the result is `Result: "unavailable"`, the fixture server received the used-offer requests as before (count > 0 on the used-offer path), and the second-hand result is the offer parsed from the used-offer fixture, not `check_error`
- Corner cases: used-offer fixture answered with HTTP 503 instead (subtest): the second-hand result is `check_error` and the used-offer failure line is logged, proving that log line still appears when a request was really made.

## Stored result and user-visible state

### TC-003 — A scheduled check answered with an empty-body 404 keeps the last second-hand offer and resets the count

- Covers: FR-AMZ-HCHK-013, FR-AMZ-HCHK-012; UC-AMZ-HCHK-006 (main flow steps 3–7, postconditions)
- Level: integration (real SQLite, HTTP handlers, real browser collector, `internal/httpapi`)
- Given a tracked Amazon item with a recorded last successful price (1999 cents) and a last detected available second-hand offer, 4 consecutive failures already recorded by the gate, and a fixture server answering `/dp/<ASIN>` with HTTP 404 and an empty body
- When the item's scheduled check runs
- Then `GET /api/v1/items/{id}` and `GET /api/v1/items` show the item as listing unavailable with second-hand status `check_error`; the item's last detected second-hand offer is unchanged; the consecutive failure count is 0 (a following 503 failure does not stop any item); the fixture server received only the product page request for this check (0 used-offer requests); the log has no used-offer failure line and no `Amazon request failed` line for this check
- Corner cases:
  - item without an earlier second-hand offer: the check records the check error and no offer appears;
  - HTTP 410 with an empty body behaves the same (subtest);
  - protected mode with two items owned by two users, both answered with an empty-body 404: neither sends a used-offer request and both show listing unavailable.

### TC-004 — Refresh price and Refresh all prices answered with an empty-body 404 send no used-offer request

- Covers: FR-AMZ-HCHK-013, FR-AMZ-HCHK-012; UC-AMZ-HCHK-006 (trigger "Refresh price", "Refresh all prices"; alternative "Refresh price on a stopped item")
- Level: integration (real SQLite, HTTP handlers, real browser collector, `internal/httpapi`)
- Given a tracked Amazon item with a last detected second-hand offer, and a fixture server answering `/dp/<ASIN>` with HTTP 404 and an empty body
- When the user calls `POST /api/v1/items/{id}/refresh`, then `POST /api/v1/items/refresh`
- Then after each call `GET /api/v1/items/{id}` shows the item as listing unavailable with second-hand status `check_error` and the last second-hand offer unchanged; the fixture server received 0 used-offer requests in total; the log has no used-offer failure line
- Corner cases:
  - Refresh price on an item stopped by 5 failures: the empty-body 404 is an Amazon answer, so that item is restarted (no longer stopped), still without any used-offer request; other stopped items stay stopped;
  - repeated Refresh price: each attempt gives the same result and never sends a used-offer request.
