# Issue #187 Skip used-offer requests when the product page is Chromium's error page

**Problem:** When the product page is an empty-body 404/410 that reaches `load` through the `errorText` branch (`internal/amazon/browser.go:352-357`), the tab shows `chrome-error://chromewebdata/`. `Collect` still calls `tab.fetchAll` (`internal/amazon/browser.go:116`). The two `fetch` calls (`credentials: "include"`, header `X-Requested-With`) then come from an opaque origin. They cannot succeed, may send Amazon a CORS preflight with `Origin: null`, and log a misleading used-offer failure.

**Suggested work:** Add an `errorPage bool` field to `loadedPage`, set it in the `errorText` branch of `load`, and in `Collect` return `SecondHandStatus = "check_error"` without calling `fetchAll` when it is set. Document this in step 8 of `doc/specifications/amazon-human-checks/technical.md`. Add a fixture test: empty-body 404 → `unavailable`, and the fixture server gets no used-offer request (this can share a fixture with the empty-body 404/410/503/429 follow-up).

From review comment R5-02 on #179 (`doc/todo/issue-168-r5-02.md`).

## Comments by trusted authors

