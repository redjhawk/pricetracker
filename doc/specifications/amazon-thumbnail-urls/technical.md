# Technical specification: Amazon thumbnails only from Amazon image hosts over HTTPS

Status: ready (issue #275)
Functional specification: [functional.md](functional.md) (FR-AMZ-THUMB-001..005).
Test cases: [testcases.md](../../changes/2026-11-24-0248-issue-275-amazon-store-only-https-thumbnail-urls-f/testcases.md) (TC-001..TC-010).
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "Item response", `thumbnailUrl` field note.

## Requirement mapping

| Technical ID | Functional IDs | Technical solution | Files |
| --- | --- | --- | --- |
| TS-AMZ-THUMB-001 | FR-AMZ-THUMB-001 | `acceptedImage(url)`: `true` when `url` starts with `https://m.media-amazon.com/` or `https://images-eu.ssl-images-amazon.com/` (`strings.HasPrefix`, case-sensitive, no trimming or parsing). An accepted URL is returned unchanged. | `internal/amazon/collector.go` |
| TS-AMZ-THUMB-002 | FR-AMZ-THUMB-001, -002 | `productImage` collects the three candidates in today's order and returns the first accepted one: `imageByID(page, "landingImage")` (`data-old-hires`, else `src`), then the `src` of the first `<img>` with `data-a-dynamic-image`, then the raw `og:image` `content`. | `internal/amazon/collector.go` |
| TS-AMZ-THUMB-003 | FR-AMZ-THUMB-003 | No accepted candidate → `productImage` returns `nil`, the same value as for a page without any image. | `internal/amazon/collector.go` |
| TS-AMZ-THUMB-004 | FR-AMZ-THUMB-004 | `parseProductPage` is otherwise unchanged: `Result`, `Message`, `FailureCause`, `AmountCents`, `Title` and the second-hand check do not depend on the image; nothing is logged for a rejected image. | `internal/amazon/collector.go` (no extra change) |
| TS-AMZ-THUMB-005 | FR-AMZ-THUMB-003, -005 | Existing store rule kept: `RecordCollection` (success only) runs `thumbnail_url = COALESCE(?, thumbnail_url)`, so `nil` keeps the stored value and an accepted URL replaces it. No migration rewrites stored values. | `internal/store/store.go` (no change) |

## Backend

### Amazon parser (`internal/amazon/collector.go`)

```go
// amazonImagePrefixes are the only thumbnail URL starts kept from an Amazon page (issue #275).
var amazonImagePrefixes = []string{"https://m.media-amazon.com/", "https://images-eu.ssl-images-amazon.com/"}

func acceptedImage(url string) bool {
	for _, prefix := range amazonImagePrefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}
	return false
}

func productImage(page string) *string {
	candidates := []string{imageByID(page, "landingImage"), dynamicImage(page), metaAttribute(page, "og:image")}
	for _, image := range candidates {
		if acceptedImage(image) {
			return &image
		}
	}
	return nil
}
```

- `dynamicImage(page)`: the existing loop (the `src` of the first `<img>` carrying `data-a-dynamic-image`), extracted. Only the first such `<img>` is a candidate, as today.
- `landingImage` is one candidate: `data-old-hires` when non-empty, else `src` (`imageByID` unchanged). A rejected `data-old-hires` does not fall back to its `src` (TC-003 row 4: its `src` is a relative path anyway).
- `og:image` must be the raw attribute value: `metaContent` passes it through `textContent`, which trims and collapses spaces and unescapes a second time, so ` https://m.media-amazon.com/…` would become accepted (TC-004 leading-space row). See Refactoring.
- `attribute` already HTML-unescapes once (`&amp;` in a query becomes `&`), as today; the stored URL is that value.
- Rules derived from the prefix check (TC-004): `http://`, `HTTPS://`, upper-case host, `:443`/`:8443`, `user@`, look-alike hosts (`m.media-amazon.com.example.com`), other Amazon hosts (`images-na.ssl-images-amazon.com`), missing trailing `/`, protocol-relative, relative, `data:`, `javascript:`, leading space and empty values are all rejected.

### Validation, persistence, migrations, concurrency

- No validation error: a rejected image only means "no candidate". The check result is unchanged otherwise.
- No schema change and no migration; thumbnails stored before this change stay until a successful check finds an accepted URL (FR-AMZ-THUMB-005). Failed checks never touch `thumbnail_url` (existing `if result.Result == "success"` branch).
- `parseProductPage` is used by the browser collector (`browser.go`) for every entry point (addition, scheduler, single and bulk refresh), so one change covers them all. Pure functions; no shared state.
- LeBoncoin (`firstImage`) and development seed data are unchanged.

## API

- No endpoint, request, response shape or error change. `thumbnailUrl` of Amazon items in list, add and details responses is now only a URL starting with one of the two prefixes, or `null`, or a value stored before this change. Documented in `API_SPECIFICATION.md` ("Item response", fields).
- Compatibility: clients already handle `thumbnailUrl: null`.

## Frontend

Not affected: `TrackedItemsPage.tsx` already shows `img.item-thumbnail` for a non-null `thumbnailUrl` and the `item-thumbnail-placeholder` otherwise; `ItemDetail.tsx` shows `img.detail-thumbnail` only for a non-null value. No new state, message or layout. TC-010 only proves this existing behaviour for the values the server now returns.

## Files to change

- `internal/amazon/collector.go`: `amazonImagePrefixes`, `acceptedImage`, `dynamicImage`, `metaAttribute`, `metaContent` (uses `metaAttribute`), `productImage`.
- Tests listed below.
- Already done in this stage: `API_SPECIFICATION.md`.

## Tests

| Test case | File | Notes |
| --- | --- | --- |
| TC-001, TC-002, TC-003, TC-004, TC-005 | `internal/amazon/thumbnail_urls_test.go` (package `amazon`, new) | Names `TestAmazonThumbnail_TC0NN_...`. A helper builds a page with `<span id="productTitle">`, a `corePriceDisplay_desktop_feature_div` block holding `<span class="a-price"><span class="a-offscreen">129,99 €</span></span>` and the row's image tags, and asserts `parseProductPage(model.CollectionResult{}, page)` gives `success`. TC-001 reads `testdata/` with `os.ReadFile`; TC-004 runs every value as `landingImage` `data-old-hires` and as `og:image` `content`; TC-005 replaces the `data-old-hires` value of `price-found.html` with `strings.Replace` and compares both results with `reflect.DeepEqual` after clearing `ThumbnailURL`. |
| TC-006, TC-007, TC-008, TC-009 | `internal/httpapi/amazon_thumbnail_urls_test.go` (package `httpapi`, new) | Names `TestAmazonThumbnailURLs_TC0NN_...`. Reuse `retrievalFixture` (`pages` per ASIN, `used: aod-none.html`), `retrievalCollector`, `openHumanStore`, `openBrowserProfileAPI`, `humanClock`/`humanTime`, `addItem`, `refresh`, `waitBrowserCheck`, `listItems`, `itemDetails`, `runScheduler`, `captureRetrievalLog`, `assertNoRetrievalFailure`; pages come from `amazontest.Fixture(t, "price-found.html")` edited with `strings.Replace` (image replaced/removed, `og:image` meta inserted in `<head>`). Pages switch by writing `fixture.pages[asin]` under `fixture.mu`. "Stored before this change": after the first check, `UPDATE items SET thumbnail_url = ? WHERE id = ?` through a second `sql.Open("sqlite", filepath.Join(dir, "pricefollower.sqlite"))`, as `internal/store/amazon_human_checks_test.go` does (no new store API). Raw JSON for TC-007: `callPath(server, GET, "/api/v1/items/{id}", "")` body contains `"thumbnailUrl":null`. TC-009 503 corner case: a wrapping `http.Handler` in the test file answers `503` for the item's `/dp/` path while a flag is set, delegating to `retrievalFixture` otherwise. |
| TC-010 | `tests/amazon-thumbnail-urls.spec.ts` (Playwright, new) | Mock API with `context.route` as `item-refresh.spec.ts` does (session, items list, item details); route `https://m.media-amazon.com/images/I/price-found.jpg` to a 1×1 PNG and record every other non-local request as unexpected; assert `img.item-thumbnail` `src`, placeholder `.item-thumbnail-placeholder`, no `img.detail-thumbnail` in B's details; repeat at `page.setViewportSize({ width: 320, height: 800 })` checking `document.documentElement.scrollWidth <= 320`, and after `page.reload()`. |

Existing tests expecting `…/price-found.jpg` or `…/mentions-captcha.jpg` (`internal/amazon/browser_test.go`, `amazon_browser_retrieval_test.go`) and LeBoncoin thumbnail tests stay unchanged.

## Refactoring

- `metaAttribute(page, property)`: extracted from `metaContent`, returns the raw `content` attribute of the first matching `<meta>`; `metaContent` becomes `textContent(metaAttribute(page, property))`, so `og:title` is unchanged. Reason: the URL check needs the value as found (TS-AMZ-THUMB-002). Scope: `collector.go` only. Risk: low; same matching loop, covered by existing title tests.
- `dynamicImage(page)`: extraction of the existing loop from `productImage`. Reason: build the candidate list. Risk: none.

## Manual follow-ups

None.
