# Technical specification: Amazon browser retrieval

Status: ready (issue #204)\
Functional specification: [functional.md](functional.md), FR-AMZ-BRET-001 – 007\
Test cases: [testcases.md](../../changes/2026-11-22-2139-issue-204-retrieval-error/testcases.md), TC-001 – TC-013\
Companions: [Amazon human-like checks](../amazon-human-checks/technical.md) (browser collector, TS-AMZ-HCHK-003 classification amended here), [Amazon browser profile recovery](../amazon-browser-profile/technical.md), [Amazon request stop](../amazon-request-stop/technical.md) (count, stop and restart, unchanged)

## Summary

Root cause: `BrowserCollector.classifyPage` matches `(?i)captcha|robot check|enter the characters you see below` against the first 200 kB of the **rendered DOM** (`outerHTML`), which includes the page's inline scripts and links. Real Amazon product pages mention `captcha` in their scripts and URLs (for example `validateCaptcha`), so every product page is classified as "Amazon blocked the product request", a `request_error`. After five checks every Amazon item is stopped.

Fix:

1. Detect a robot check by the page's structure: there is no product block, and there is a robot-check form or visible robot-check text. Scripts and styles are ignored.
2. Give each failure a cause text that the operator can act on. The gate's log line carries that cause next to the URL and time. A browser that cannot start reports its exit status and the end of its error output.
3. Document the browser that the server needs, and check the documentation against the names the code reads.

Price and second-hand parsing, the failure count, stop and restart rules, the UI and the API do not change.

## Requirement mapping

`BRET-*` abbreviates `FR-AMZ-BRET-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-AMZ-BRET-001 | BRET-001, BRET-003 | New `isRobotCheck(html)` replaces `robotCheckPattern` in `classifyPage`. A page with a product block is never a robot check. Otherwise, with scripts and styles removed, the page is a robot check if it has a `<form>` whose `action` contains `/errors/validateCaptcha`, or if its visible text contains a robot-check phrase. | `internal/amazon/browser.go`, `internal/amazon/collector.go` | TC-001, TC-002, TC-004, TC-009, TC-012 |
| TS-AMZ-BRET-002 | BRET-001, BRET-002 | No change to `parseProductPage`, `secondHandOfferFromProductPage`, `usedOfferURLs`, `parseUsedOffers` or the in-page `fetch`. Once a page passes classification, browser results equal parser results. A consent overlay is only extra DOM and does not affect the price regions. | none (verified only) | TC-002, TC-004, TC-005, TC-009 |
| TS-AMZ-BRET-003 | BRET-004 | A robot check, non-2xx status other than 404/410, wrong content type, unsupported redirect, navigation error or timeout, or browser start failure is still a `request_error` with `RequestURL`, `HTTPStatus` and `ResponseExcerpt`. `amazonGate.do` counts, stops and restarts exactly as before. | `internal/amazon/browser.go` (cause text only) | TC-003, TC-008, TC-010 |
| TS-AMZ-BRET-004 | BRET-005 | New diagnostic field `CollectionResult.FailureCause`, filled by the collector for every `request_error` and `price_not_found` result. The text is cleaned: no control characters, one line, at most 1,024 bytes. The gate's failure line logs `cause=` instead of the generic user message. A new line logs `price_not_found` with its URL, time and cause, without counting it. | `internal/model/model.go`, `internal/amazon/browser.go`, `internal/amazon/collector.go`, `internal/service/amazon_gate.go` | TC-003, TC-006, TC-007, TC-012 |
| TS-AMZ-BRET-005 | BRET-005 | `cdpBrowser` keeps the last 2 KB of the browser's stderr and its exit status. An error caused by the browser exiting becomes `the browser exited (<exit status>): <stderr tail>`, so a start failure logs `browser could not start: the browser exited (exit status 1): Running as root without --no-sandbox …`. | `internal/amazon/cdp.go` | TC-006, TC-007, TC-012, TC-013 |
| TS-AMZ-BRET-006 | BRET-006 | New `Server setup` section in `deploy/README.md`: browser install commands, lookup order (`PRICEFOLLOWER_BROWSER_PATH`, then executable names on `PATH`), `PRICEFOLLOWER_BROWSER_NO_SANDBOX=true` for containers and root, the profile directory and the related log lines. `README.md` links to it. The names are exported from code so a documentation test can compare both ways. The start log line for a missing browser stays as it is. | `deploy/README.md`, `README.md`, `internal/amazon/browser.go`, `internal/service/service.go` | TC-011, TC-012, TC-013 |
| TS-AMZ-BRET-007 | BRET-007 | No change. A stopped item is still skipped by the scheduler, and **Refresh price** (`manual = true`) still runs it. A non-failure result clears its stop (`ClearAmazonStop`) and resets the count. | none (verified only) | TC-010 |

## Frontend

Not affected. There is no new screen, state, label or message, and item states and the retrieval-issue display stay the same. The functional spec says nothing about the cause is added to the UI. The Playwright tests in `tests/` must keep passing. No file in `src/` changes.

## Backend

### Robot-check detection (TS-AMZ-BRET-001)

`internal/amazon/collector.go`:

- Move the region list of `productPrice` to a package variable `productPriceRegions` (same order and values). `productPrice` iterates over it. This is a pure extraction with no behaviour change.

`internal/amazon/browser.go`:

- Remove `robotCheckPattern`.
- Add ``robotCheckText = regexp.MustCompile(`(?i)enter the characters you see below|saisissez les caractères que vous voyez ci-dessous`)`` and ``robotCheckForm = regexp.MustCompile(`(?is)<form\b[^>]*\baction\s*=\s*["']?[^"'>]*/errors/validatecaptcha`)``. The form action is the marker that does not depend on the language. The two phrases are those of the robot pages in the test cases.
- Add `isRobotCheck(html string) bool`:
  1. `markup := stylePattern.ReplaceAllString(scriptPattern.ReplaceAllString(html, " "), " ")`. Script content can never count, even when it contains a `<form …validateCaptcha>` string.
  2. If `markup` contains `id="productTitle"` or `id='productTitle'`, or `id="<region>"` / `id='<region>'` for any `productPriceRegions` entry, return `false`. A product page is read as a product page whatever words it contains (BRET-003).
  3. Return `robotCheckForm.MatchString(markup) || robotCheckText.MatchString(textContent(markup))`.
- In `classifyPage`, replace the robot step with `if isRobotCheck(page.html) { return false, errors.New("Amazon showed a robot check instead of the product page") }`. It scans the whole rendered page, which is already capped at 5 MB. The earlier 200 kB slice is not needed because there is no longer a free-text match over scripts. The order of the other checks does not change.
- Results: an empty 200 page has no product, form or text, so it is not a robot check and `parseProductPage` gives `price_not_found` as today. A robot page wrapped in a larger page without a product block is still a robot check. Links, `alt` texts, titles and reviews that contain "robot" or "captcha" do not match either rule.

### Failure causes (TS-AMZ-BRET-004)

`internal/model/model.go`: add `FailureCause string` to the "Failure diagnostics for logs (Amazon); never persisted nor returned by the API" group of `CollectionResult`. The store and the HTTP layer never read it.

`internal/amazon/browser.go` / `collector.go`:

- New `logText(reason string) string`. It replaces every control character (`unicode.IsControl`, which includes `\n` and `\r`) with a space, collapses runs of whitespace, trims, drops invalid UTF-8, and cuts the result to 1,024 bytes on a rune boundary, adding `…` when it cuts.
- `requestFailure(item, result, reason)`: `cause := logText(reason)`, then `printRequestError(item, cause)` (the existing `Amazon collection failed for <marketplace>/<ASIN>: <cause>` line) and `result.FailureCause = cause`.
- `parseProductPage`: for `price_not_found`, set `result.FailureCause = "Amazon page showed no detectable euro price"`. The user message does not change.
- Cause texts. The prefixes of (a) to (f) are pairwise different, as TC-007 requires:

| Cause (functional) | `FailureCause` | Source |
| --- | --- | --- |
| (a) browser could not start | `browser could not start: <reason>`. The reason is the `exec` error (it names the missing file), `no Chromium or Google Chrome found on PATH`, a profile error, or `the browser exited (<status>): <stderr tail>` | `open`, existing prefix |
| (b) page could not be opened | `Amazon page could not be opened: <net::ERR_…>` | `load`, existing |
| (c) page did not finish loading | `Amazon page did not finish loading` | `loadError`, existing |
| (d) non-success status | `Amazon returned HTTP <n>` | `classifyPage`, existing |
| (e) robot check | `Amazon showed a robot check instead of the product page` | `classifyPage`, new text |
| (f) no detectable price | `Amazon page showed no detectable euro price` | `parseProductPage`, new |
| other failures | `redirected to an unsupported host`, `Amazon returned an unexpected content type`, `page script failed: …`, `Amazon page loaded without a response` | existing |

`internal/service/amazon_gate.go`, in `do`, after `now` is computed:

- Let `cause := result.FailureCause`, falling back to `result.Message` when the field is empty (other collectors and test doubles).
- For `request_error`: `log.Printf("Amazon request failed url=%s at=%s status=%d cause=%s response=%s", result.RequestURL, now.Format(time.RFC3339), result.HTTPStatus, cause, result.ResponseExcerpt)`. The prefix stays the same, so existing counts of `Amazon request failed` still hold.
- For `price_not_found`: `log.Printf("Amazon check found no price url=%s at=%s status=%d cause=%s", …)`. This line does not contain `failed`, and the count is reset as today.
- `success` and `unavailable` log nothing new.

Secrets (FR-AMZ-HUMAN-012): causes come from error strings (exec, CDP `errorText`, status code, browser stderr) and fixed texts. Cookies and headers are never formatted into them. The excerpt is the rendered HTML as today, and cookie values are not part of the DOM text.

### Browser exit reason (TS-AMZ-BRET-005)

`internal/amazon/cdp.go`:

- New type `tailBuffer`, an `io.Writer` guarded by a mutex that keeps only the last 2,048 bytes written. It has a `String()` method.
- `launchBrowser`: set `cmd.Stderr = &tailBuffer{}` and `cmd.WaitDelay = time.Second`. Child processes that keep the stderr pipe open then delay `Wait` by at most one second after the main process exits. The wait goroutine closes `b.exited` once `cmd.Wait()` returns, so `cmd.ProcessState` is set. Reads happen after `<-b.exited`, so no extra lock is needed.
- New `(b *cdpBrowser) exitError() error`. It waits up to 2 s for `b.exited`. If the browser exited, it returns `fmt.Errorf("%w (%s): %s", errBrowserExited, status, tail)`. Otherwise it returns `errBrowserExited`. The status is `b.cmd.ProcessState.String()` (for example `exit status 1` or `signal: killed`), and the tail is the trimmed stderr tail. The `: tail` part is left out when the tail is empty. The tail keeps at most its last 768 bytes (`lastBytes`, cut on a rune boundary, with a leading `…` when the start is dropped), so that with the cause prefixes it fits in the 1,024 bytes `logText` keeps, and the browser's last line (usually its fatal reason) is never cut.
- `call`: the two `return errBrowserExited` paths (write error, and `b.read` closed) return `b.exitError()`. `errors.Is(err, errBrowserExited)` still holds. `open` already wraps these errors as `browser could not start: …` during `setUp`, so a stub that prints and exits gives the cause `browser could not start: the browser exited (exit status 1): Running as root without --no-sandbox is not supported. See https://crbug.com/638180.`
- `close()` does not change. Its `Browser.close` call happens after `b.read` closes, and `exitError` returns at once because `b.exited` is already closed or about to close. TC-006 requires the check to end within 20 s, and the worst case is 2 s for `exitError` plus the existing 2 s + 5 s of `close`.
- Memory is bounded at 2 KB per browser. A browser that runs normally writes warnings to stderr, and these are never logged.

### Installation and lookup names (TS-AMZ-BRET-006)

`internal/amazon/browser.go`:

- `const BrowserPathVariable = "PRICEFOLLOWER_BROWSER_PATH"` and `var BrowserNames = []string{"chromium", "chromium-browser", "google-chrome-stable", "google-chrome"}`. `FindBrowser` uses both, with the same order and behaviour.

`internal/service/service.go`:

- `const browserNoSandboxVariable = "PRICEFOLLOWER_BROWSER_NO_SANDBOX"`, read by `newBrowserCollector` (`== "true"`, unchanged).
- New `Options.AmazonBrowserOrigin string // tests only: BrowserOptions.Origin of the default browser collector`. `newBrowserCollector(cfg, origin)` passes it on. Production leaves it empty. TC-012 and TC-013 use it to test the collector the service builds from configuration and environment without contacting Amazon.
- The start line `Amazon browser not found; Amazon checks will fail until Chromium is installed or PRICEFOLLOWER_BROWSER_PATH is set` stays as it is and is logged once per `NewWithOptions` when `FindBrowser` fails. Each check still resolves the executable again, so a browser installed after start is used without a restart.

`deploy/README.md`, new section `## Server setup` placed before `## Administrator account`:

- The data directory (`PRICEFOLLOWER_DATA_DIR`), which the service user must be able to write. It contains the SQLite database and the browser profile `amazon-browser/`.
- Subsection `### Browser for Amazon checks`:
  - Install Chromium or Google Chrome. On Debian: `sudo apt-get install -y chromium`. On Ubuntu and other amd64 systems: `wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb && sudo apt-get install -y ./google-chrome-stable_current_amd64.deb`.
  - How the application finds it: `PRICEFOLLOWER_BROWSER_PATH` (full path to the executable), otherwise the first of `chromium`, `chromium-browser`, `google-chrome-stable`, `google-chrome` found on the service's `PATH`. List the names in backticks, in this order, in one sentence.
  - Containers or a service running as root: set `PRICEFOLLOWER_BROWSER_NO_SANDBOX=true`. In Docker, also give the container enough shared memory (`--shm-size=1g`).
  - What the log shows: the start line when no browser is found, and `Amazon request failed … cause=…` for each failed check (see the cause list).
- `README.md`, "Production deployment" bullet under "Technology": add a link to [deploy/README.md#server-setup](../../../deploy/README.md#server-setup) saying the server also needs Chromium or Google Chrome for Amazon checks.

### Persistence, validation and concurrency

- Persistence and migrations: none. `FailureCause` is never stored, and the schema, failure count, stops and attempts do not change.
- Validation: none added. The browser path and the no-sandbox flag come from the environment as before.
- Concurrency: unchanged. Checks are serialized by `BrowserCollector.mu` and `amazonGate.request`. `tailBuffer` has its own mutex because `exec`'s copy goroutine writes to it. `cmd.ProcessState` is published by closing `exited`.

## API

Not affected. There is no new endpoint, field, status or error. `lastAttempt.result`, `lastAttempt.message`, `amazonStoppedAt` and `POST /api/v1/items/{id}/refresh` (`202`) keep their contract. `FailureCause`, like the other diagnostic fields, is never serialized. `API_SPECIFICATION.md` does not change.

## Files to change

| File | Change |
| --- | --- |
| `internal/amazon/browser.go` | TS-AMZ-BRET-001, -003, -004, -006: `isRobotCheck`, robot cause text, `logText`, `FailureCause` in `requestFailure`, `BrowserPathVariable`, `BrowserNames` |
| `internal/amazon/collector.go` | `productPriceRegions` extraction; `FailureCause` for `price_not_found` (TS-AMZ-BRET-001, -004) |
| `internal/amazon/cdp.go` | TS-AMZ-BRET-005: `tailBuffer`, `WaitDelay`, `exitError` |
| `internal/model/model.go` | `CollectionResult.FailureCause` |
| `internal/service/amazon_gate.go` | `cause=` in the failure line; new `price_not_found` line |
| `internal/service/service.go` | `browserNoSandboxVariable`, `Options.AmazonBrowserOrigin`, `newBrowserCollector(cfg, origin)` |
| `deploy/README.md`, `README.md` | TS-AMZ-BRET-006 documentation and link |
| `internal/amazon/amazontest/amazontest.go` | `ExpandedFixture(t, name)`: `Fixture` with `<!--PADDING-->` replaced by about 210 kB of neutral product-description markup, so that mentions fall both before and after 200 kB |
| `internal/amazon/testdata/mentions-captcha.html` (new) | Mentioning page (test-case conventions), price `129,99 €` in `corePriceDisplay_desktop_feature_div`, with `<!--PADDING-->` before the review mention |
| `internal/amazon/testdata/consent-banner.html` (new) | Mentioning page with a consent overlay form and accept button over the product |
| `internal/amazon/testdata/robot-check-en.html`, `robot-check-fr.html` (new) | Robot pages: title `Amazon.fr`, captcha image, form `/errors/validateCaptcha`, English or French instruction, no product |
| `internal/amazon/testdata/aod-amazon-used.html` (new) | Used-offer answer with one `aod-offer` sold by Amazon (used, condition, price) |
| Test files below | New tests |

## Test implementation

Test names: `TestBrowserRetrieval_TCnnn_` in package `amazon`, and `TestAmazonBrowserRetrieval_TCnnn_` in `service` and `httpapi`. Browser tests use `amazontest.Browser(t)`, `amazontest.NewServer`, `Origin`, a profile in `t.TempDir()` and `Pause` returning 0. The secret fixture sets the cookie `session-token=fixture-secret-token-204`.

| Test case | File | Notes |
| --- | --- | --- |
| TC-001 | `internal/amazon/browser_retrieval_test.go` | No browser. Calls `(&BrowserCollector{}).classifyPage(loadedPage{status: 200, mimeType: "text/html", url: "https://www.amazon.fr/dp/…", html: …})`, then `parseProductPage`. Subtests cover each single mention in lower and upper case, the two robot pages (raw and wrapped in a page without a product block), an empty body, and all existing fixtures. |
| TC-002 | `internal/amazon/browser_retrieval_test.go` | Local handler `retrievalHandler(t)` serves the new fixtures by ASIN (`B0MENTION1`, `B0CONSENT1`, `B0DOMROBOT`, robot ASINs) plus the existing `fixturePages`, and answers `aod-none.html` for used offers. `B0DOMROBOT` is the mentioning page with an inline script that appends a `captcha`/`robot` text node on load. Uses the existing `captureOutput`. |
| TC-003 | `internal/amazon/browser_retrieval_test.go` | Robot pages with the secret cookie, plus a robot body with status 503. Checks `FailureCause` and the logged text contain `robot check` (or `HTTP 503`), checks `server.Count("/gp/aod/ajax…") == 0`, and checks that the secret is absent. |
| TC-004 | `internal/amazon/browser_retrieval_test.go` | Expected values come from a test helper `parsedResult(item, html, usedPages)` that composes `parseProductPage`, `secondHandOfferFromProductPage` and `parseUsedOffers` on the fixture HTML with the same used-offer answers. 404 and 410 ASINs are included. |
| TC-005 | `internal/amazon/browser_retrieval_test.go` | Per-ASIN used-offer answers (`aod-amazon-used.html`, `aod-other-seller.html`, `aod-none.html`, 503, a closed listener) on pages carrying the mentions. Checks the request `Cookie` header on `/gp/aod/ajax` in `server.Requests()`. |
| TC-006 | `internal/amazon/browser_retrieval_test.go` | `/bin/sh` stubs in `t.TempDir()` (message plus `exit 1`, `exit 3`, multi-line output longer than 1 KB) and a missing path, each with a 20 s context. Uses the existing `assertNothingLeft`. The "no browser" case uses `t.Setenv(BrowserPathVariable, "")`, `t.Setenv("PATH", t.TempDir())` and `ExecPath: ""`. The cause must be a single line of at most 1,024 bytes. |
| TC-007 | `internal/service/amazon_browser_retrieval_test.go` | Harness of `amazon_human_checks_test.go` (`openHumanStore`, `startHuman`, `newClock`, `insertAt`, `amazonFailures`). The collector is a small dispatcher (`AmazonCollector`) that routes each item ID to its own `amazon.BrowserCollector`. Logs are captured with `log.SetOutput`, plus stdout for the collector line. The test parses `url=`, `at=` and `cause=` from the gate lines and compares the causes after removing URL and time. |
| TC-008 | `internal/service/amazon_browser_retrieval_test.go` | Fixture server with a switchable handler (robot page, then mentioning page, 503, closed listener). Uses `amazonFailures` and the store's stopped state. |
| TC-012 | `internal/service/amazon_browser_retrieval_test.go` | `NewWithOptions(cfg, db, Options{AmazonBrowserOrigin: server.URL, Now: …})` with no `Amazon` set, so the production `newBrowserCollector` runs. Uses `t.Setenv` for `PRICEFOLLOWER_BROWSER_PATH` and `PRICEFOLLOWER_BROWSER_NO_SANDBOX`, and a `PATH` directory containing only a `chromium` symlink to `amazontest.Browser(t)`. The root case is skipped unless `os.Geteuid() == 0`. Not parallel. |
| TC-013 | `internal/service/amazon_browser_retrieval_test.go` | Same construction with an empty `PATH` directory. Counts the start line once. Then symlinks `chromium` into that directory and checks that the next Refresh price succeeds without restarting. |
| TC-011 | `internal/service/amazon_browser_retrieval_docs_test.go` | Reads `../../deploy/README.md` and `../../README.md`. Extracts the `## Server setup` section. Compares the set of `PRICEFOLLOWER_BROWSER_[A-Z_]+` tokens with `{amazon.BrowserPathVariable, browserNoSandboxVariable}`. Checks that every `amazon.BrowserNames` entry appears in backticks and that every backticked `chromium*`/`google-chrome*` token is in the list. Checks for an install command (`apt-get install`) and the `README.md` link to `deploy/README.md#server-setup`. |
| TC-009 | `internal/httpapi/amazon_browser_retrieval_test.go` | Harness of `amazon_human_checks_test.go` (`openHumanStore`, `humanClock`, `listItems`, `waitChecked`, `refresh`, `addItem`). `service.NewWithOptions` gets `Amazon: amazon.NewBrowserCollector(...)` on the consent page with an `aod-amazon-used.html` answer. Protected mode uses two owners as in the existing ownership tests. |
| TC-010 | `internal/httpapi/amazon_browser_retrieval_test.go` | Items stopped by five robot failures through the real gate, then the handler switches to the mentioning page. Checks `amazonStoppedAt` in `GET /api/v1/items` and the `202` from refresh. |

The existing tests must keep passing: `internal/amazon/browser_test.go` (#168 TC-003, TC-006, TC-007), `browser_profile_test.go` (#180), `collector_test.go`, `internal/service/*`, `internal/httpapi/*`, and the Playwright tests in `tests/`. There is no e2e case because the UI does not change.

## Refactoring needed by the feature

| Refactoring | Reason | Scope | Risk |
| --- | --- | --- | --- |
| Extract `productPriceRegions` from `productPrice` | `isRobotCheck` must recognize the same product blocks as the price parser | `collector.go`, one variable | None: same list and order, covered by the existing parser tests and TC-004 |
| Export `BrowserPathVariable` and `BrowserNames`, and name `browserNoSandboxVariable` | TC-011 must fail when the code and the documentation diverge | `browser.go`, `service.go` | None: same values. `amazontest.Browser` keeps its own copy because importing `amazon` from it would create an import cycle in the `amazon` tests |
| `newBrowserCollector(cfg, origin)` and `Options.AmazonBrowserOrigin` | TC-012 and TC-013 must run the production collector construction without contacting Amazon | `service.go` | Low: empty in production, so behaviour is the same |
| `cdpBrowser` stderr tail and exit error | BRET-005 requires the browser's own reason | `cdp.go` | Low: `errors.Is(…, errBrowserExited)` is kept and `WaitDelay` bounds `Wait`. Covered by #168 TC-007, #180 tests and TC-006 |

Out of scope: changing price or second-hand parsing, retrying or restarting stopped items automatically, extra Chromium flags, packaging the browser, and any UI change.

## Verification

`go build ./...`, `go vet ./...`, `go test ./...`, and `go test -race ./internal/amazon/... ./internal/service/... ./internal/httpapi/...` with Chromium available (`CI` set, so browser tests cannot skip). Also `npm run build` and the Playwright suite (tests unchanged).

No open question.
