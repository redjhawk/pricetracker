# Test cases: downloads refused by the Amazon browser (issue #217)

Sources: [functional specification](../../specifications/amazon-human-checks/functional.md) (FR-AMZ-HCHK-014, with FR-AMZ-HCHK-011 for the check result), use case [UC-AMZ-HCHK-007](../../use-cases/amazon-human-checks/UC-AMZ-HCHK-007-amazon-answer-offered-as-download-is-not-saved.md).

Conventions:

- **Browser tests** run the real Chromium (`amazontest.Browser`) against a local `amazontest.Server` serving Amazon-like fixture responses, as in `internal/amazon/browser_navigation_test.go`. No test contacts amazon.*.
- **Isolated home:** each test sets `HOME` and `XDG_CONFIG_HOME` (`t.Setenv`) to directories under `t.TempDir()`, so the browser's default Downloads folder is `<HOME>/Downloads` and no `user-dirs.dirs` of the machine redirects it. The browser process inherits this environment.
- **Download response:** HTTP 200, `Content-Type: text/html; charset=utf-8`, `Content-Disposition: attachment; filename="page.html"`, body = the `price-found.html` fixture with an added unique marker comment (for example `<!-- download-marker-217 -->`) that the normal page does not contain.
- **No file written** means, checked after the check has returned (the browser is closed at the end of each check): no file named `page.html`, no partial download (`*.crdownload`, `.com.google.Chrome.*`, `Unconfirmed *.crdownload`) and no regular file whose content contains the marker, anywhere under the profile directory or under `HOME`; `<HOME>/Downloads` does not exist when it did not exist before, and is still empty when it existed empty.
- A **failure** result is `Result: "request_error"`, `Message` = the Amazon check failed message, `SecondHandStatus: "check_error"` (as asserted by `assertNavigationFailure`).
- Integration tests through the API use a real SQLite database in `t.TempDir()`, the HTTP handlers (`GET /api/v1/items/{id}`, `POST /api/v1/items/{id}/refresh`, `POST /api/v1/items/refresh`) and the real browser collector with its persistent profile in the data directory (`<data>/amazon-browser`, as `internal/service/service.go`), as in `internal/httpapi/amazon_browser_navigation_test.go`; scheduled checks are triggered directly through the service.
- The UI does not change. No new Playwright case: the user-visible states of a failed check are those of issue #186, already covered by `tests/item-refresh.spec.ts` and `tests/platform-tabs.spec.ts`, which must keep passing. Narrow screens and keyboard are therefore not relevant.
- Existing tests of issue #186 (`TestBrowserNavigation_TC001`–`TC003`, `TestAmazonBrowserNavigation_TC004`–`TC006`) keep their assertions: the check result is unchanged (FR-AMZ-HCHK-011, FR-AMZ-HCHK-012).

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-AMZ-HCHK-014 | TC-001, TC-002, TC-003, TC-004 |

| Use case | Test cases |
| --- | --- |
| UC-AMZ-HCHK-007 | TC-001 (main flow 1–5, alternatives "same check repeated", "next check answered with the normal page"), TC-002 (main flow 3–4), TC-003 (main flow, alternatives "same check repeated", "next check answered with the normal page"), TC-004 (alternatives "fifth consecutive failure", "protected mode") |

## Browser collector

### TC-001 — A product page offered as a download leaves no file in the profile or the Downloads folder

- Covers: FR-AMZ-HCHK-014 (with FR-AMZ-HCHK-011); UC-AMZ-HCHK-007 (main flow steps 1–5, alternatives "same check repeated" and "next check answered with the normal page")
- Level: integration (browser test, `internal/amazon`; extends `TestBrowserNavigation_TC001_AttachmentIsAFailure`, subtest "attachment then a normal page")
- Given an isolated home without a `Downloads` folder, a persistent profile directory under `t.TempDir()`, and a fixture server answering `/dp/B0PRICE001` with the download response
- When the browser collector checks the item, then checks it a second time with the same answer, then the server switches to the normal 200 `text/html` page (no `Content-Disposition`) and the item is checked a third time
- Then:
  - the first and second checks are failures logged with the request URL and the browser's error (as today), and the server received the product page request each time;
  - after each of the first two checks, no file is written under the profile directory or under `HOME` (see "No file written"), and `<HOME>/Downloads` does not exist;
  - the third check is `success` with the fixture price (12999 cents), and still no file with the marker exists in the profile or under `HOME`
- Corner cases:
  - repeated action: the second download response adds no file;
  - the Downloads folder is absent before the check and is not created;
  - partial files: no `*.crdownload` or Chrome temporary download file remains;
  - the next check behaves as if the download had never been offered (no download prompt or leftover state prevents opening the page).

### TC-002 — An existing Downloads folder stays empty, whatever the attachment's file name

- Covers: FR-AMZ-HCHK-014; UC-AMZ-HCHK-007 (main flow steps 3–4, precondition "the Downloads folder may or may not exist")
- Level: integration (browser test, `internal/amazon`; subtests of `TestBrowserNavigation_TC001_AttachmentIsAFailure`)
- Given an isolated home whose `Downloads` folder already exists and is empty, and a fixture server answering `/dp/B0PRICE001` with the download response, per subtest: `Content-Disposition: attachment; filename="page.html"`; `Content-Disposition: attachment` without a file name; `Content-Type: application/octet-stream` with `Content-Disposition: attachment; filename="B0PRICE001.bin"`
- When the browser collector checks the item
- Then the check is a failure (never `price_not_found`, `success` or `unavailable`); `<HOME>/Downloads` still exists and contains no entry; the profile directory has no file named after the attachment (`page.html`, `B0PRICE001.bin`, `B0PRICE001`, `download*`), no partial download and no file containing the marker
- Corner cases: Downloads folder present (the browser's default target); attachment without a file name (Chromium derives one from the URL); a non-HTML content type, which Chromium always downloads.

## Checks through the application

### TC-003 — Refresh price and Refresh all prices answered with a download write nothing in the data directory profile

- Covers: FR-AMZ-HCHK-014 (with FR-AMZ-HCHK-011); UC-AMZ-HCHK-007 (main flow, alternatives "same check repeated" and "next check answered with the normal page")
- Level: integration (real SQLite, HTTP handlers, real browser collector with its persistent profile in `<data>/amazon-browser`, `internal/httpapi`)
- Given an isolated home without a `Downloads` folder, a tracked Amazon item with a recorded last successful price (1999 cents) and second-hand offer, and a fixture server answering the item's `/dp/<ASIN>` with the download response
- When the user calls `POST /api/v1/items/{id}/refresh`, then `POST /api/v1/items/refresh`, then the server switches to the normal page and the user calls `POST /api/v1/items/{id}/refresh` again
- Then:
  - after the first two refreshes, `GET /api/v1/items/{id}` shows the item in the retrieval error state with last attempt `request_error` and second-hand status `check_error`, latest price 1999 cents and the last second-hand offer kept (FR-AMZ-HCHK-011);
  - after each refresh, no file is written under `<data>/amazon-browser` or under `HOME` (see "No file written") and `<HOME>/Downloads` does not exist; the data directory holds no new file outside `amazon-browser` other than the SQLite database files;
  - after the third refresh, the item's latest price is the fixture page price
- Corner cases: repeated action through two different refresh entry points; the browser profile persists between checks, so a file left by one check would be visible to the next; the normal page after the downloads records its price.

### TC-004 — A scheduled check answered with a download stops items as before and writes no file

- Covers: FR-AMZ-HCHK-014 (with FR-AMZ-HCHK-011); UC-AMZ-HCHK-007 (alternatives "fifth consecutive failure" and "protected mode")
- Level: integration (real SQLite, HTTP handlers, real browser collector with its persistent profile, `internal/httpapi`; extends `TestAmazonBrowserNavigation_TC004_DownloadIsACountedFailure`)
- Given an isolated home without a `Downloads` folder, two tracked Amazon items owned by two users (protected mode), 4 consecutive failures already recorded (4 scheduled checks answered HTTP 503 with a page)
- When the next scheduled check is answered with the download response
- Then every tracked Amazon item is stopped (`amazonStoppedAt` set in `GET /api/v1/items` for each user), each item's latest price is kept, and no file is written under the profile directory or under `HOME` (see "No file written"); `<HOME>/Downloads` does not exist
- Corner cases: the download as the 5th consecutive failure; items of several users in protected mode; the stop path closes the browser without leaving a partial download.
