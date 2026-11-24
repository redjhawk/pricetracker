# Test cases: Amazon thumbnails only from Amazon image hosts over HTTPS (issue #275)

Sources: functional specification [amazon-thumbnail-urls](../../specifications/amazon-thumbnail-urls/functional.md) (FR-AMZ-THUMB-001 to FR-AMZ-THUMB-005) and use cases [UC-AMZ-THUMB-001](../../use-cases/amazon-thumbnail-urls/UC-AMZ-THUMB-001-add-amazon-item-with-amazon-image.md), [UC-AMZ-THUMB-002](../../use-cases/amazon-thumbnail-urls/UC-AMZ-THUMB-002-add-amazon-item-without-accepted-image.md) and [UC-AMZ-THUMB-003](../../use-cases/amazon-thumbnail-urls/UC-AMZ-THUMB-003-recheck-amazon-item-with-stored-thumbnail.md).

Conventions:

- **Accepted** means the URL starts exactly with `https://m.media-amazon.com/` or `https://images-eu.ssl-images-amazon.com/`.
- **Candidates**, in order: `landingImage` (its `data-old-hires`, else its `src`), then the `src` of the first `<img>` with `data-a-dynamic-image`, then `og:image`.
- **Unit tests** (`internal/amazon`, package-internal) call `parseProductPage` (or `productImage`) on small HTML pages built in the test, each with a valid euro price in `corePriceDisplay_desktop_feature_div` so the result is `success`, and on the `testdata/` fixtures.
- **Integration tests** (`internal/httpapi`) follow `amazon_browser_retrieval_test.go`: a real `BrowserCollector` (Chromium from `amazontest.Browser`, skipped when absent) with `Origin` set to a local fixture server that serves the product page per ASIN (`retrievalFixture.pages`) and `aod-none.html` as used-offer answer, a real service and SQLite database, open mode. Items are added with `POST /api/v1/items`, re-checked with `POST /api/v1/items/{id}/refresh`, and read with `GET /api/v1/items` and `GET /api/v1/items/{id}` once the check is recorded.
- **Image replaced by X** (fixture copies): the `data-old-hires` value of `landingImage` is set to X; its `src` stays `/static/pixel.png`, so no candidate is accepted unless stated. **Image removed:** the `landingImage` tag is removed.
- **Stored before this change:** an item thumbnail written directly in the SQLite `items.thumbnail_url` column of the test database before the re-check.
- **E2E** (`tests/amazon-thumbnail-urls.spec.ts`, Playwright) mocks the API with `page.route`, as the other specs do; it proves what the list shows for the `thumbnailUrl` the server returns. The server side is proven by the integration cases.
- Existing Amazon tests that expect `https://m.media-amazon.com/images/I/price-found.jpg` or `…/mentions-captcha.jpg` as thumbnail stay unchanged and keep passing. LeBoncoin thumbnail tests stay unchanged.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-AMZ-THUMB-001 | TC-001, TC-002, TC-006, TC-008, TC-010 |
| FR-AMZ-THUMB-002 | TC-003, TC-004, TC-007, TC-008 |
| FR-AMZ-THUMB-003 | TC-004, TC-005, TC-007, TC-009, TC-010 |
| FR-AMZ-THUMB-004 | TC-005, TC-007, TC-009 |
| FR-AMZ-THUMB-005 | TC-008, TC-009 |

| Use case | Test cases |
| --- | --- |
| UC-AMZ-THUMB-001 | TC-001, TC-002, TC-003, TC-006, TC-010 |
| UC-AMZ-THUMB-002 | TC-004, TC-005, TC-007, TC-010 |
| UC-AMZ-THUMB-003 | TC-008, TC-009 |

## Cases

### TC-001 — Fixture pages keep their Amazon image

- Covers: FR-AMZ-THUMB-001; UC-AMZ-THUMB-001
- Level: unit
- Given the fixtures `price-found.html`, `mentions-captcha.html` and `consent-banner.html`
- When each page is parsed
- Then the thumbnail is `https://m.media-amazon.com/images/I/price-found.jpg`, `https://m.media-amazon.com/images/I/mentions-captcha.jpg` and `https://m.media-amazon.com/images/I/mentions-captcha.jpg`; `script-price.html` (only `src="/static/pixel.png"`, a relative path) gives no thumbnail.

### TC-002 — Accepted URLs are kept exactly as found, from either host and any candidate

- Covers: FR-AMZ-THUMB-001; UC-AMZ-THUMB-001
- Level: unit (table test)
- Given a page with a price whose only image candidate is, per row:
  - `landingImage` `data-old-hires` `https://images-eu.ssl-images-amazon.com/images/I/x.jpg`;
  - `landingImage` `src` `https://m.media-amazon.com/images/I/src.jpg` (no `data-old-hires`);
  - dynamic-image `<img>` `src` `https://m.media-amazon.com/images/I/dyn._AC_SX300_.jpg`;
  - `og:image` `https://images-eu.ssl-images-amazon.com/images/I/og.jpg`;
  - `landingImage` `https://m.media-amazon.com/images/I/q.jpg?v=1&w=500#f` (path, query and fragment)
- When the page is parsed
- Then the thumbnail is that exact string, unchanged.
- Corner cases: both hosts, every candidate position, query string kept.

### TC-003 — A rejected candidate is skipped for the next accepted one

- Covers: FR-AMZ-THUMB-002; UC-AMZ-THUMB-001
- Level: unit (table test)
- Given a page with a price and, per row:
  - `landingImage` `http://m.media-amazon.com/images/I/a.jpg` and `og:image` `https://m.media-amazon.com/images/I/b.jpg` → `…/b.jpg`;
  - `landingImage` `https://example.com/a.jpg`, dynamic image `https://m.media-amazon.com/images/I/d.jpg`, `og:image` `https://m.media-amazon.com/images/I/b.jpg` → `…/d.jpg` (first accepted wins);
  - `landingImage` `data:image/png;base64,AAAA`, dynamic image `https://example.com/d.jpg`, `og:image` `https://images-eu.ssl-images-amazon.com/images/I/og.jpg` → `…/og.jpg`;
  - `landingImage` with `data-old-hires` `https://example.com/h.jpg` and `src` `/static/pixel.png`, `og:image` `https://m.media-amazon.com/images/I/b.jpg` → `…/b.jpg`
- When the page is parsed
- Then the thumbnail is the expected accepted URL and never a rejected one.
- Corner cases: rejection at the first and second positions, two rejections before an accepted one, several accepted candidates (order kept).

### TC-004 — Rejected URLs give no thumbnail

- Covers: FR-AMZ-THUMB-002, FR-AMZ-THUMB-003; UC-AMZ-THUMB-002
- Level: unit (table test)
- Given a page with a price whose only image candidate (`landingImage`, then in a second sub-case `og:image`) is, per row:
  `http://m.media-amazon.com/images/I/a.jpg`; `https://example.com/a.jpg`; `https://m.media-amazon.com.example.com/a.jpg`; `https://evil.com/https://m.media-amazon.com/a.jpg`; `https://M.MEDIA-AMAZON.COM/images/I/a.jpg`; `HTTPS://m.media-amazon.com/images/I/a.jpg`; `https://m.media-amazon.com:443/images/I/a.jpg`; `https://m.media-amazon.com:8443/a.jpg`; `https://user@m.media-amazon.com/a.jpg`; `https://images-na.ssl-images-amazon.com/images/I/a.jpg`; `https://m.media-amazon.com` (no trailing `/`); `//m.media-amazon.com/images/I/a.jpg`; `/images/I/a.jpg`; `data:image/jpeg;base64,AAAA`; `javascript:alert(1)`; ` https://m.media-amazon.com/images/I/a.jpg` (leading space); empty value
- When the page is parsed
- Then the thumbnail is `nil`.
- And a page with a price and no image at all also gives `nil` (same result).
- Corner cases: wrong scheme, other host, look-alike host, letter case, ports (including 443), user information, protocol-relative and relative paths, `data:` and `javascript:` URLs, blank value.

### TC-005 — The rest of the result is unaffected by a rejected image

- Covers: FR-AMZ-THUMB-003, FR-AMZ-THUMB-004; UC-AMZ-THUMB-002
- Level: unit
- Given `price-found.html` with its image changed to `https://example.com/a.jpg`, and the unchanged `price-found.html`
- When both are parsed
- Then the two results are equal except `ThumbnailURL` (`nil` for the first): same `Result` `success`, amount 12999, title `Price found product`, empty `Message` and `FailureCause`.

### TC-006 — Adding an item whose page has an accepted image stores and returns it

- Covers: FR-AMZ-THUMB-001; UC-AMZ-THUMB-001
- Level: integration (browser collector, HTTP API, real SQLite)
- Given the fixture server serving `price-found.html` for one ASIN and a copy whose `landingImage` is `https://images-eu.ssl-images-amazon.com/images/I/x.jpg` for a second ASIN
- When both listings are added with `POST /api/v1/items` and their first checks are recorded
- Then `GET /api/v1/items` returns `thumbnailUrl` `https://m.media-amazon.com/images/I/price-found.jpg` and `https://images-eu.ssl-images-amazon.com/images/I/x.jpg`, price 129.99, status `active`, last attempt `success`.

### TC-007 — Adding an item whose page has only rejected images: no thumbnail, check otherwise succeeds

- Covers: FR-AMZ-THUMB-002, FR-AMZ-THUMB-003, FR-AMZ-THUMB-004; UC-AMZ-THUMB-002
- Level: integration (browser collector, HTTP API, real SQLite)
- Given the fixture server serving, per ASIN, `price-found.html` with:
  - its image replaced by `https://example.com/a.jpg`;
  - its image replaced by `http://m.media-amazon.com/images/I/a.jpg` and an `og:image` `data:image/png;base64,AAAA`;
  - its image removed (no candidate)
- When each listing is added and its first check is recorded
- Then for every item `thumbnailUrl` is `null` in `GET /api/v1/items` and `GET /api/v1/items/{id}` (raw JSON `"thumbnailUrl":null`), and the title is `Price found product`, the latest price 129.99, status `active` (not `retrieval_error`), last attempt `success` with `message` `null`, the second-hand offer `not_found`, and `nextCheckAt` set, the same as for the unchanged page.
- And no Amazon failure is counted or logged (`assertNoRetrievalFailure`).
- Corner cases: one and several rejected candidates, page without any image.

### TC-008 — Re-check replaces the stored thumbnail with the first accepted image

- Covers: FR-AMZ-THUMB-001, FR-AMZ-THUMB-002, FR-AMZ-THUMB-005; UC-AMZ-THUMB-003
- Level: integration (browser collector, HTTP API, real SQLite; table test, one item per row)
- Given an added Amazon item whose stored thumbnail is, per row: `https://m.media-amazon.com/images/I/old.jpg`; `http://example.com/old.jpg` (stored before this change)
- When the fixture page of its ASIN is switched to `price-found.html` with, per sub-case:
  - `landingImage` `https://m.media-amazon.com/images/I/new.jpg`;
  - `landingImage` `https://example.com/a.jpg` and `og:image` `https://m.media-amazon.com/images/I/new.jpg`
- and the item is re-checked with `POST /api/v1/items/{id}/refresh` (202), then the check is recorded
- Then `GET /api/v1/items` and `GET /api/v1/items/{id}` return `thumbnailUrl` `https://m.media-amazon.com/images/I/new.jpg`.
- Corner cases: a second refresh with the same page keeps `…/new.jpg`; a scheduled check (`runScheduler`) instead of **Refresh price** gives the same result for one row.

### TC-009 — Re-check with no accepted image keeps the stored thumbnail

- Covers: FR-AMZ-THUMB-003, FR-AMZ-THUMB-004, FR-AMZ-THUMB-005; UC-AMZ-THUMB-003
- Level: integration (browser collector, HTTP API, real SQLite; table test, one item per row)
- Given an added Amazon item whose stored thumbnail is, per row: `https://m.media-amazon.com/images/I/old.jpg`; `http://example.com/old.jpg` (stored before this change)
- When its fixture page is switched to `price-found.html` with the image replaced by `https://example.com/a.jpg`, and the item is refreshed and the check recorded
- Then `thumbnailUrl` is still the stored value (`…/old.jpg`, resp. `http://example.com/old.jpg`); the new check records price 129.99, status `active`, last attempt `success` with `message` `null`.
- Corner cases: page without any image gives the same result; a failed check (fixture answering HTTP 503) leaves the stored thumbnail unchanged; a later refresh with an accepted image then replaces it (`…/new.jpg`).

### TC-010 — The list shows the accepted image or the placeholder

- Covers: FR-AMZ-THUMB-001, FR-AMZ-THUMB-003; UC-AMZ-THUMB-001, UC-AMZ-THUMB-002
- Level: e2e (Playwright, API mocked with `page.route`)
- Given the **Amazon** tab lists two items: A with `thumbnailUrl` `https://m.media-amazon.com/images/I/price-found.jpg` (image request routed to a 1×1 PNG), B with `thumbnailUrl` `null`
- When the tracked-items list is displayed and item B's details are opened
- Then A's row shows an `img.item-thumbnail` whose `src` is that URL; B's row shows the existing thumbnail placeholder and no `img.item-thumbnail`; B's details show no `img.detail-thumbnail`; no image request other than A's is made.
- Corner cases: the same at a 320 px wide viewport (placeholder visible, no horizontal page overflow); reloading the page gives the same display.
