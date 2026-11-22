# Test cases: Amazon browser retrieval (issue #204)

Sources: [functional specification](../../specifications/amazon-browser-retrieval/functional.md), use cases [UC-AMZ-BRET-001](../../use-cases/amazon-browser-retrieval/UC-AMZ-BRET-001-amazon-check-of-available-listing-returns-price.md), [UC-AMZ-BRET-002](../../use-cases/amazon-browser-retrieval/UC-AMZ-BRET-002-understand-failed-amazon-check-from-server-log.md), [UC-AMZ-BRET-003](../../use-cases/amazon-browser-retrieval/UC-AMZ-BRET-003-install-server-for-amazon-checks.md), [UC-AMZ-BRET-004](../../use-cases/amazon-browser-retrieval/UC-AMZ-BRET-004-restart-amazon-item-stopped-by-retrieval-errors.md).

Conventions:

- **Browser tests** run the real browser (`amazontest.Browser(t)`) against a local fixture server (`amazontest.NewServer`, Amazon URLs sent to it through `Origin`), with a profile in `t.TempDir()` and no reading pause. No test contacts amazon.*.
- **Mentioning page** is a new fixture in `internal/amazon/testdata/`: a complete product page (title, image, euro price 129,99 € in the usual price block) shaped like a real Amazon product page, whose content mentions the words without being a robot check: an inline script with `captcha`, `Captcha` and `validateCaptcha` strings and a `robot` string, a footer link whose URL contains `captcha`, a product title "Robot aspirateur …", an image `alt` with "robot", and a customer review sentence containing "robot check". Some mentions are in the first 200 kB of the page, some after.
- **Consent page** is the mentioning page with a cookie/consent banner (overlay form with an accept button, as Amazon shows it) placed over the product.
- **Robot pages** are the robot-check pages: the English one of issue #168 TC-006 (`Enter the characters you see below`, form `/errors/validateCaptcha`), a French one (`Saisissez les caractères que vous voyez ci-dessous`, captcha image, same form), both with HTTP 200, title `Amazon.fr` and no product.
- **Secret fixture**: the fixture server sets a `session-token` cookie with a known value; "no secret" means that value never appears in the result, its excerpt or the log.
- Integration tests use a real SQLite database in `t.TempDir()`, the service harness of `internal/service/amazon_human_checks_test.go` (fake clock, captured log) and, when stated, the HTTP handlers (`POST /api/v1/items`, `POST /api/v1/items/refresh`, `POST /api/v1/items/{id}/refresh`, `GET /api/v1/items`).
- New Go test names use the prefix `TestBrowserRetrieval_TCnnn_` (package `amazon`) or `TestAmazonBrowserRetrieval_TCnnn_` (service, HTTP and documentation tests) so they do not clash with the `TCnnn` names of issues #168 and #180.
- No interface change: there is no e2e case. The browser tests of issues #168 and #180 and the Playwright tests in `tests/` must keep passing.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-AMZ-BRET-001 | TC-001, TC-002, TC-004, TC-009, TC-012 |
| FR-AMZ-BRET-002 | TC-004, TC-005, TC-009 |
| FR-AMZ-BRET-003 | TC-001, TC-002, TC-005, TC-008, TC-009 |
| FR-AMZ-BRET-004 | TC-003, TC-008, TC-010 |
| FR-AMZ-BRET-005 | TC-003, TC-006, TC-007, TC-012 |
| FR-AMZ-BRET-006 | TC-011, TC-012, TC-013 |
| FR-AMZ-BRET-007 | TC-010 |

| Use case | Test cases |
| --- | --- |
| UC-AMZ-BRET-001 | TC-001, TC-002, TC-003, TC-004, TC-005, TC-008, TC-009 |
| UC-AMZ-BRET-002 | TC-003, TC-006, TC-007, TC-008, TC-013 |
| UC-AMZ-BRET-003 | TC-011, TC-012, TC-013 |
| UC-AMZ-BRET-004 | TC-010 |

## Reading the product page

### TC-001 — Mentions of "captcha" or "robot" do not make a robot check

- Covers: FR-AMZ-BRET-001, FR-AMZ-BRET-003; UC-AMZ-BRET-001
- Level: unit
- Given the HTML of the mentioning page as a loaded page (HTTP 200, `text/html`, supported URL)
- When the page is classified and read as a product page
- Then it is not a failure and not unavailable; the result is `success` with 12999 cents, the fixture title and image
- Corner cases:
  - each mention alone (script string, link URL, title, image `alt`, review text), in lower and upper case (`CAPTCHA`, `Robot`): still `success`;
  - the two robot pages: still classified as a robot check (failure);
  - the robot pages wrapped in a larger page with no product block: still a robot check;
  - empty body with HTTP 200: not a robot check; read as `price_not_found` as today;
  - the existing fixtures (`price-found`, `price-not-found`, `unavailable`, `script-price`, `used-*`) give the same classification and result as before the change.

### TC-002 — Browser check of a mentioning page records its price

- Covers: FR-AMZ-BRET-001, FR-AMZ-BRET-003; UC-AMZ-BRET-001
- Level: integration (browser test)
- Given the fixture server serving the mentioning page at `/dp/<ASIN>` and no used offer at `/gp/aod/ajax`, and the log captured
- When the browser collector checks the item
- Then the result is `success` with 12999 cents, the fixture title and image, HTTP status 200, no message; the second-hand status is `not_found`; the log has no `Amazon collection failed` line and no line naming a robot check or a block
- Corner cases:
  - the consent page: same result (the banner does not prevent reading the price);
  - a page whose script adds a `captcha` / `robot` text to the DOM after load (present only in the rendered page): same result;
  - two checks in a row on the same profile: both `success`;
  - the browser is started headless, no window, no credentials (existing #168 TC-003 keeps passing).

### TC-003 — Robot checks stay failures, logged as robot checks

- Covers: FR-AMZ-BRET-004, FR-AMZ-BRET-005; UC-AMZ-BRET-001, UC-AMZ-BRET-002
- Level: integration (browser test)
- Given the fixture server serving one of the robot pages at `/dp/<ASIN>` with the secret fixture, and the log captured
- When the browser collector checks the item
- Then the result is `request_error` with message `Amazon could not be reached for a price check.`, second-hand status `check_error`, HTTP status 200, a non-empty response excerpt; the logged cause names a robot check; no used-offer request is made; no secret
- Corner cases: English and French robot pages; a robot page answered with HTTP 503 is a failure logged with the non-success status 503; existing #168 TC-006 (403, 503, robot check) keeps passing.

### TC-004 — Browser results equal the results before the browser change

- Covers: FR-AMZ-BRET-001, FR-AMZ-BRET-002; UC-AMZ-BRET-001
- Level: integration (browser test)
- Given every product fixture (`price-found`, `script-price`, `price-not-found` (instalment only), `unavailable`, `used-amazon`, `used-no-price`, `used-other-seller`, mentioning page, consent page) and `/dp/B0GONE0001` answering 404 and another ASIN answering 410
- When each is checked by the browser collector, and its HTML is read by the product page parser with the same used-offer answers
- Then for each fixture the browser result equals the parser result on: result, amount, title, image URL, message, second-hand status, amount, condition and condition label
- Corner cases: instalment-only page gives `price_not_found` (FR-AMAZON-PRICE-003); unavailable text, 404 and 410 give `unavailable` with `The listing is no longer available.`, never `request_error`.

### TC-005 — Second-hand offer result is unchanged

- Covers: FR-AMZ-BRET-002, FR-AMZ-BRET-003; UC-AMZ-BRET-001
- Level: integration (browser test)
- Given product pages that also carry the mentions of the mentioning page, and the used-offer endpoint answering per ASIN
- When the browser collector checks each item
- Then:
  - page with a used offer sold by Amazon (`used-amazon` content): second-hand `available` with its amount, condition and label; no `/gp/aod/ajax` request;
  - page without one, used-offer answer with an offer sold by Amazon (new fixture `aod-amazon-used.html`): `available` with that offer's amount and condition; the used-offer request is made from inside the page and carries the page's cookies;
  - used-offer answer with another seller's offer only: `not_found`;
  - used-offer answer with no offer: `not_found`
- Corner cases: used-offer endpoint answering 503 or refusing the connection: second-hand `check_error` while the price result stays `success`; used-offer answer whose content mentions `captcha`: read as offers, not as a failure.

## Failure causes in the log

### TC-006 — A browser that cannot start is logged with its own reason

- Covers: FR-AMZ-BRET-005; UC-AMZ-BRET-002
- Level: unit
- Given a collector whose executable is (a) a missing file, (b) a stub script that writes `Running as root without --no-sandbox is not supported. See https://crbug.com/638180.` to its error output and exits 1, (c) a stub that exits 3 without output; and the log captured
- When an item is checked
- Then each check is a `request_error` ending within 20 s with no browser process or goroutine left; the logged cause says the browser could not start and includes the reason: (a) the missing file, (b) the stub's message, (c) the exit status
- Corner cases: stub writing several lines: the reason is logged without breaking the entry over several log lines and is bounded in length; `PRICEFOLLOWER_BROWSER_PATH` unset and no browser on `PATH`: the cause says no Chromium or Chrome was found.

### TC-007 — Each failure cause has its own log text, with URL and time

- Covers: FR-AMZ-BRET-005; UC-AMZ-BRET-002
- Level: integration (SQLite + browser test)
- Given the service with a fake clock and one Amazon item per cause, each checked by a browser collector set up for that cause: (a) missing executable, (b) connection refused, (c) page that never finishes loading (navigation timeout 2 s), (d) HTTP 503, (e) robot page, (f) page without a detectable price (`price-not-found`), all with the secret fixture where a server answers; the log captured
- When **Refresh price** is requested for each item in turn
- Then for each cause the log has an entry for that check containing the request URL, the check time (RFC 3339) and the cause: (a) browser could not start with its reason, (b) the page could not be opened, (c) the page did not finish loading, (d) non-success status with `503`, (e) robot check, (f) no detectable price; the cause texts of (a), (b) or (c), (d), (e), (f) are pairwise different once URL and time are removed; no secret
- Corner cases:
  - failures (a)–(e) increase the consecutive failure count by one each; (f) is logged but, as today, resets the count to 0 and is not a failure of FR-AMZ-STOP;
  - every item shows a retrieval issue and keeps its last successful price;
  - an unavailable listing (404) logs no failure;
  - a successful check logs no failure.

### TC-008 — Failures are counted and stop items as before; mentions are not

- Covers: FR-AMZ-BRET-004, FR-AMZ-BRET-003; UC-AMZ-BRET-001, UC-AMZ-BRET-002
- Level: integration (SQLite + browser test)
- Given two Amazon items with a stored successful price and failure count 0, and a fixture server answering the robot page
- When five checks run in a row (Refresh price and scheduled checks at the due time of the fake clock)
- Then each check shows a retrieval issue and logs a robot check; the count goes 1 to 5; the items are not stopped after failures 1–4; after the 5th both items are stopped and the stop is logged; both items still show their last successful price
- Corner cases: after 4 robot failures, the server switches to the mentioning page: the next check succeeds, records the price and resets the count to 0, and no block is counted; HTTP 503 and connection refused count the same way as the robot page.

## User flows

### TC-009 — Every Amazon check of an available listing records its price

- Covers: FR-AMZ-BRET-001, FR-AMZ-BRET-002, FR-AMZ-BRET-003; UC-AMZ-BRET-001
- Level: integration (SQLite + HTTP + browser test)
- Given the service and HTTP handlers with a browser collector on the fixture server serving the consent page (mentions included) and a used-offer answer with an Amazon used offer, and a fake clock
- When an Amazon item is added with `POST /api/v1/items` (first check), then its scheduled check runs after its due time, then `POST /api/v1/items/refresh` (Refresh all prices), then `POST /api/v1/items/{id}/refresh` (Refresh price)
- Then after each check `GET /api/v1/items` shows the fixture price, title, image and the second-hand offer, a new check time, a status that is not a retrieval issue and a latest attempt without error message; the failure count is 0; no failure is logged
- Corner cases: protected mode with Amazon items of two owners: both owners' items get their prices; a second Refresh price after the first finished also succeeds; an item whose page shows only an instalment amount shows "price not found" and an unavailable listing shows **Listing unavailable**, not a retrieval issue.

### TC-010 — Refresh price restarts an item stopped by the retrieval errors

- Covers: FR-AMZ-BRET-007, FR-AMZ-BRET-004; UC-AMZ-BRET-004
- Level: integration (SQLite + HTTP + browser test)
- Given three Amazon items stopped by five robot-check failures (the issue #204 situation), each with a last successful price, and the fixture server now serving the mentioning page
- When the scheduler runs after the items' due times, then `POST /api/v1/items/{id}/refresh` is called for the first item
- Then the scheduler sends no request for the stopped items; the refresh answers `202`, records the fixture price and check time, and `GET /api/v1/items` shows the first item with `amazonStoppedAt: null`; the other two items stay stopped; the failure count is 0
- Corner cases: Refresh price on a stopped item while the server answers the robot page: no price recorded, the item stays stopped, the failure is logged as a robot check and counted; Refresh price repeated on the restarted item: succeeds again; an item not stopped gets its next scheduled check without user action.

## Installation

### TC-011 — The installation documentation states the browser needs

- Covers: FR-AMZ-BRET-006; UC-AMZ-BRET-003
- Level: unit (documentation test)
- Given the repository's installation documentation (the server setup section of `deploy/README.md`, linked from `README.md`)
- When the test reads it
- Then it names the browser to install (Chromium or Google Chrome) with an install command; states how the application finds it: `PRICEFOLLOWER_BROWSER_PATH`, otherwise one of the executable names looked up on `PATH`; and states the setting for a container or root service: `PRICEFOLLOWER_BROWSER_NO_SANDBOX=true`
- Corner cases: every executable name and environment variable the documentation lists is one that `amazon.FindBrowser` and the service actually read, and every name they read is documented (the test fails when code and documentation diverge); `README.md` links to the section.

### TC-012 — A server set up as documented checks Amazon listings

- Covers: FR-AMZ-BRET-006, FR-AMZ-BRET-001, FR-AMZ-BRET-005; UC-AMZ-BRET-003
- Level: integration (SQLite + browser test)
- Given the service with the collector it builds from its configuration and environment (only the test origin added), the environment set as the documentation says: `PRICEFOLLOWER_BROWSER_PATH` pointing to the test browser, `PRICEFOLLOWER_BROWSER_NO_SANDBOX=true`, `PATH` without any browser, profile in the data directory, and the mentioning page served
- When Refresh price is requested for an Amazon item
- Then the price is recorded and no failure is logged
- Corner cases:
  - `PRICEFOLLOWER_BROWSER_PATH` unset and a `chromium` link to the test browser in a directory on `PATH`: same success;
  - `PRICEFOLLOWER_BROWSER_PATH` pointing to a missing file: the check fails with "browser could not start" and the missing-file reason in the log;
  - when the test runs as root, `PRICEFOLLOWER_BROWSER_NO_SANDBOX` unset: the check fails and the log carries the browser's own reason (skipped when not root).

### TC-013 — A missing browser is reported at start

- Covers: FR-AMZ-BRET-006; UC-AMZ-BRET-002, UC-AMZ-BRET-003
- Level: integration (SQLite)
- Given `PRICEFOLLOWER_BROWSER_PATH` unset, `PATH` without any browser, and the log captured
- When the service starts with its default collector
- Then the application starts, the log has the line saying the Amazon browser was not found, once; a following Amazon check fails and logs "browser could not start" with its reason
- Corner cases: with a browser on `PATH` or `PRICEFOLLOWER_BROWSER_PATH` set, no such start line; a browser installed after start is used by the next check without restart.
