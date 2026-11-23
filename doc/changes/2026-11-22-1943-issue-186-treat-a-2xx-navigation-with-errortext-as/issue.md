# Issue #186 Treat a 2xx navigation with errorText as a failed Amazon check

**Problem:** In `internal/amazon/browser.go:352-357` (`browserTab.load`), when `Page.navigate` returns an `errorText`, any recorded Document response of the navigated frame becomes the loaded page, including a 2xx one. The documented reason for this path (technical.md step 4) is `net::ERR_HTTP_RESPONSE_CODE_FAILURE` for a non-2xx answer with an empty body. A 2xx response with an `errorText` (for example `net::ERR_ABORTED` for a download, or a 204/205) leaves the frame on `about:blank` or an error page, which `parseProductPage` then turns into `price_not_found`. That resets the Amazon failure count.

**Suggested work:** In the `errorText` branch, return the recorded response only when its status is outside 200-299. Otherwise return the failure `Amazon page could not be opened: <errorText>`. Update step 4 of `doc/specifications/amazon-human-checks/technical.md` to match. Add a fixture test: 200 `text/html` with `Content-Disposition: attachment` → `request_error`.

From review comment R5-01 on #179 (`doc/todo/issue-168-r5-01.md`).

## Comments by trusted authors

### redjhawk at 2026-11-23T06:30:44Z

@claude The factory now resolves merge conflicts with a merge agent (#212). Resume at `tests`.

