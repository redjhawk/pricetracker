# Technical specification: Amazon browser retrieval

Status: ready (issue #204); TS-AMZ-BRET-008 ready (issue #226, see [Issue #226](#issue-226-ubuntu-chromium-snap-pitfall)); TS-AMZ-BRET-009 ready (issue #210, see [Issue #210](#issue-210-response-field-of-the-failure-log-line))\
Functional specification: [functional.md](functional.md), FR-AMZ-BRET-001 – 009\
Test cases: [#204 testcases.md](../../changes/2026-11-22-2139-issue-204-retrieval-error/testcases.md), TC-001 – TC-013; [#226 testcases.md](../../changes/2026-11-23-1020-issue-226-document-the-ubuntu-chromium-snap-pitfal/testcases.md), TC-001 – TC-004; [#210 testcases.md](../../changes/2026-11-27-2140-issue-210-document-the-response-field-of-the-amazo/testcases.md), TC-001 – TC-005\
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
| TS-AMZ-BRET-008 | BRET-008, BRET-006 | Issue #226. One new paragraph, the snap sentence, in `### Browser for Amazon checks` of `deploy/README.md`, right after the lookup paragraph. No code change: `FindBrowser`, `BrowserNames` and `BrowserPathVariable` stay as they are. | `deploy/README.md` | #226 TC-001 – TC-004, #204 TC-011 |
| TS-AMZ-BRET-009 | BRET-009, BRET-005 | Issue #210. In the log list of `### Browser for Amazon checks` of `deploy/README.md`, the failed-check line gets `response=…` after `cause=…`, and the item gets one sentence on `response`. No code change: `amazonGate.do` already logs `… cause=%s response=%s`. | `deploy/README.md` | #210 TC-001 – TC-005 |

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

## Issue #226: Ubuntu Chromium snap pitfall

FR-AMZ-BRET-008, TS-AMZ-BRET-008. Test case numbers in this section are those of the [#226 test cases](../../changes/2026-11-23-1020-issue-226-document-the-ubuntu-chromium-snap-pitfal/testcases.md).

### Summary

On Ubuntu, `chromium` and `chromium-browser` usually lead to the snap Chromium, which `FindBrowser` finds before `google-chrome-stable`. Snap confinement can stop it from starting for the service. The fix is one documentation sentence that tells the operator to set `PRICEFOLLOWER_BROWSER_PATH` to Google Chrome. The lookup order does not change (the functional specification excludes it).

### Frontend

Not affected: no screen, state or message changes. No file in `src/` or `web/` changes. The Playwright tests in `tests/` must keep passing.

### Backend

Not affected: no code change. `amazon.FindBrowser`, `amazon.BrowserNames` (`chromium`, `chromium-browser`, `google-chrome-stable`, `google-chrome`) and `amazon.BrowserPathVariable` stay as they are. `FindBrowser` still resolves the executable at each check, so a value of `PRICEFOLLOWER_BROWSER_PATH` set later is used without a restart. No persistence, migration, validation or concurrency change. New tests only pin the existing behaviour (TC-003, TC-004).

### API

Not affected: no endpoint, field or error changes. `API_SPECIFICATION.md` does not change.

### Documentation (TS-AMZ-BRET-008)

`deploy/README.md`, subsection `### Browser for Amazon checks` of `## Server setup`: add this paragraph directly after the paragraph that starts "The application uses `PRICEFOLLOWER_BROWSER_PATH`" and before the container paragraph:

> When a Chromium snap is installed (usual on Ubuntu, where `chromium` and `chromium-browser` lead to it), snap confinement can stop it from starting for the service and every Amazon check then fails with "browser could not start", so set `PRICEFOLLOWER_BROWSER_PATH` to the Google Chrome executable, for example `/usr/bin/google-chrome-stable`.

Rules for the text, checked by the tests:

- It is one sentence. It has no `.`, `!` or `?` followed by whitespace before its end, and no `e.g.`.
- The example path is written only as the full path `/usr/bin/google-chrome-stable`. It is never a bare name such as `google-chrome-stable` in its own backtick token, so the documented looked-up names stay those of the code. The repeated `chromium` and `chromium-browser` tokens come after the lookup list, so they do not change its order.
- `PRICEFOLLOWER_BROWSER_PATH` is the only setting named, so the #204 TC-011 variable check still passes.
- No other part of `deploy/README.md` or `README.md` changes.

### Test implementation

Test names use the prefix `TestAmazonBrowserSnapDocs_TCnnn_`.

| Test case | File | Notes |
| --- | --- | --- |
| TC-001 | `internal/service/amazon_browser_snap_docs_test.go` (new) | Reuses `serverSetupSection`, `readRepositoryFile` and `fencedCodePattern` from `amazon_browser_retrieval_docs_test.go`. New helpers: `browserSubsection(section)` returns the text from `\n### Browser for Amazon checks\n` to the next `\n### ` or `\n## `, or ""; `sentences(text)` removes fenced code, then splits on ``[.!?](\s+|$)`` and on blank lines, and returns each sentence with its offset. `assertSnapSentence(t, deployment)` requires one sentence that contains `snap`, `ubuntu` and `google chrome` (case-insensitive), and `amazon.BrowserPathVariable` and `/usr/bin/google-chrome-stable` (exact). The sentence must start after the first `` `google-chrome` `` token of the subsection, which ends the lookup list. It runs on the real file and on fixtures built from `specifiedSnapSetup`: `specifiedServerSetup` with the snap sentence inserted as a new paragraph after the lookup paragraph. Fixtures that must fail: no sentence; no setting; no path; words split over two sentences; sentence only under `## Administrator account`; sentence only in a fenced block. The specified fixture must pass. Each failing fixture is checked with a helper that returns an error instead of calling `t.Fatal`. |
| TC-002 | `internal/service/amazon_browser_snap_docs_test.go` (new) | `lookedUpNames(subsection)`: backtick tokens of the prose (fenced code removed) that match `browserNamePattern`, in reading order, without repeats. It must equal `amazon.BrowserNames` (`slices.Equal`). It runs on the real file, on `specifiedSnapSetup`, and on fixtures: names in another order and an extra bare name (`google-chrome-unstable`, `chromium-snap`) fail; `chromium` repeated in the snap sentence passes; the backticked `/usr/bin/google-chrome-stable` is never collected. Also calls `assertInstallationDocumentation` on `specifiedSnapSetup` and the real files, so #204 TC-011 is checked on the changed section (the existing #204 TC-011 tests stay unchanged). |
| TC-003 | `internal/amazon/browser_snap_test.go` (new, package `amazon`) | Writes executable `#!/bin/sh` stubs (mode `0o755`) in `t.TempDir()`, then sets `t.Setenv("PATH", dir)` and `t.Setenv(BrowserPathVariable, …)`. Checks `FindBrowser` for each case of the test case: `chromium` and `google-chrome-stable` give `chromium`; only `chromium-browser` and `google-chrome` give `chromium-browser`; only `google-chrome-stable` gives it; the setting set to the `google-chrome-stable` stub wins over `chromium`; an empty setting acts as unset; nothing gives an error. Asserts that `BrowserNames` equals the four names in order. Not parallel (`t.Setenv`). |
| TC-004 | `internal/service/amazon_browser_snap_test.go` (new) | Reuses the #204 helpers: `amazontest.Browser(t)` (called before `PATH` changes), `newAmazonFixture` with `page(amazontest.ExpandedFixture(t, "mentions-captcha.html"))` (price 129.99 €), `openHumanStore`, `insertPricedAmazonItem` (last price 19.99 €), `captureRetrievalLog`, `documentedService`, `newClock`, `refreshPrice`, `storedItem`, `amazonFailures`, `checkLogLine`, `logCause`. Snap stub: `#!/bin/sh` script named `chromium` in a `PATH` directory that writes `cannot create user data directory: Permission denied` to stderr and exits 1. `PRICEFOLLOWER_BROWSER_NO_SANDBOX=true` as in #204 TC-012. Step (a), setting empty: `request_error`, latest price still 1999 cents, `amazonFailures == 1`, and the logged cause has `browser could not start` and the stub's message. Step (b): `t.Setenv(amazon.BrowserPathVariable, browser)` on the same service, clock moved forward, Refresh price again: `success`, price 12999, `amazonFailures == 0`, no new `Amazon request failed` line. Subtests cover the corner cases: only a `google-chrome-stable` link to the test browser on `PATH` with the setting empty; the setting already set with the snap stub on `PATH`; a repeated refresh after (b); the setting pointing to a missing file, where the cause has `browser could not start` and the missing path. Not parallel. |

No e2e test: the interface does not change. The whole suite must keep passing: `go test ./...` (including #204 TC-011, TC-012 and TC-013) and the Playwright tests in `tests/`.

### Files to change

| File | Change |
| --- | --- |
| `deploy/README.md` | Snap sentence (TS-AMZ-BRET-008) |
| `internal/service/amazon_browser_snap_docs_test.go` (new) | TC-001, TC-002 |
| `internal/amazon/browser_snap_test.go` (new) | TC-003 |
| `internal/service/amazon_browser_snap_test.go` (new) | TC-004 |
| `internal/service/amazon_browser_retrieval_test.go` | Formatting only, required by `gofmt`: the one-line `B0CAUSEG01` func literal of #204 TC-007 is wrapped over three lines. The master form is not gofmt-clean (`git show master:internal/service/amazon_browser_retrieval_test.go \| gofmt -l` prints `<standard input>`, because `gofmt` keeps a func literal body on one line only when it is short). The red gate checks `gofmt -l` on the changed test files, the implementation stage needs `gofmt -l .` empty, and `repair.sh` checks `gofmt -l cmd internal`. Reverting the hunk would break those checks. The test does not change in any other way. |

### Refactoring

None. The new tests reuse the unexported helpers of the #204 tests in the same packages, so no helper needs to move.

### Verification

`go vet ./...`, `go test ./...` with Chromium available (`CI` set, so browser tests cannot skip), and the Playwright suite (tests unchanged). No manual follow-up outside the developer's paths. No open question.

## Issue #210: response field of the failure log line

FR-AMZ-BRET-009, TS-AMZ-BRET-009. Test case numbers in this section are those of the [#210 test cases](../../changes/2026-11-27-2140-issue-210-document-the-response-field-of-the-amazo/testcases.md).

### Summary

`amazonGate.do` logs `Amazon request failed url=%s at=%s status=%d cause=%s response=%s`, where `response` is `result.ResponseExcerpt`: the first 2 KB of the rendered page with control characters replaced by spaces, so the line stays one line. It is empty when no page was loaded. `deploy/README.md` shows the line without `response=…`. The fix is documentation only: the documented line gets the field, and one sentence says what it holds.

### Frontend

Not affected: no screen, state or message changes. No file in `src/` or `web/` changes. The Playwright tests in `tests/` must keep passing.

### Backend

Not affected: no code change. The failure line format in `internal/service/amazon_gate.go`, the no-price line (`Amazon check found no price url=%s at=%s status=%d cause=%s`, no `response`), `excerpt` in `internal/amazon/collector.go`, and the failure count, stop and restart rules stay as they are. No persistence, migration, validation or concurrency change. The new tests only pin the existing log format against the documentation (TC-004, TC-005).

### API

Not affected: no endpoint, field or error changes. `ResponseExcerpt` and `FailureCause` are never serialized. `API_SPECIFICATION.md` does not change.

### Documentation (TS-AMZ-BRET-009)

`deploy/README.md`, list "What the server log shows:" in `### Browser for Amazon checks` of `## Server setup`. Replace only the failed-check item with:

> - For each failed Amazon check: `Amazon request failed url=… at=… status=… cause=… response=…`. The cause says whether the browser could not start (with the browser's own reason), the page could not be opened or did not finish loading, Amazon returned a non-success HTTP status, or Amazon showed a robot check. `response` is the start of the page Amazon returned; it can be empty, for example when the browser could not start.

Rules for the text, checked by the tests:

- The inline code token is the only one in the browser subsection that starts with `Amazon request failed`. Its fields are `url`, `at`, `status`, `cause`, `response`, in the order of the gate's format, and `…` (U+2026) stands for each value.
- The `response` sentence counts as one sentence for `sentences`: no `.`, `!` or `?` followed by whitespace before its end, and no `e.g.`. It comes after the token and contains `response`, `start of the page` and `empty`.
- The causes sentence, the start item and the no-price item stay word for word, and the list keeps its three items. No setting or browser name is added, so #204 TC-011 and #226 TC-001 – TC-002 still pass.
- No other part of `deploy/README.md` or `README.md` changes.

### Test implementation

Test names use the prefix `TestAmazonFailureLogDocs_TCnnn_`, all in package `service`. They reuse the #204 and #226 helpers of the same package: `readRepositoryFile`, `serverSetupSection`, `inlineCodePattern`, `prose`, `sentences`, `browserSubsection`, `specifiedSnapSetup`, `assertInstallationDocumentation`, `snapSentenceError`, `lookedUpNamesError`. From the service tests they reuse `openHumanStore`, `startHuman`, `newClock`, `scheduleDay`, `insertPricedAmazonItem`, `captureRetrievalLog`, `refreshPrice`, `checkLogLine`, `storedItem`, `amazonFailures`, `shorten`, `fakeAmazon`, `succeeded`, `newAmazonFixture`, `page`, `retrievalBrowser`, `collectorByItem` and `retrievalSecret`. Each check that has failing fixtures is a function that returns an `error`, so fixtures are checked without `t.Fatal`, as in the #226 tests.

New helpers in `amazon_failure_log_docs_test.go`:

- `failedLinePrefix = "Amazon request failed"`, `noPriceLinePrefix = "Amazon check found no price"`, `formerFailedLine = "Amazon request failed url=… at=… status=… cause=…"`.
- `logListItems(deployment) ([]string, error)`: in `prose(browserSubsection(serverSetupSection(deployment)))`, it finds the line `What the server log shows:` and skips blank lines. It returns the text after `- ` of each following line that starts with `- `, up to the first line that does not. It returns an error when the subsection or the list is missing.
- `logItem(items, label)`: the item that starts with `label` (`At start, when no browser is found:`, `For each failed Amazon check:` or `For a page without a detectable price:`).
- `documentedLine(deployment, prefix) (line, item string, end int, err error)`: requires exactly one inline code token starting with `prefix` in the browser subsection prose. It returns the token, its log list item, and the offset in the item where the token ends. It returns an error when the token is missing, repeated, or not in a log list item.
- `fieldNames(line)`: the names captured by ``(?:^| )([a-z]+)=``, in reading order.
- `documentedPattern(line) *regexp.Regexp`: splits the documented line on `…` and quotes each part with `regexp.QuoteMeta`. It joins the parts with `(.*?)`, except that the last `…` becomes `(.*)`, and anchors the result with `^` and `$`.
- `specifiedLogList`: the list as the README has it after the change (the start item, the failed-check item quoted above and the unchanged no-price item), preceded by `What the server log shows:\n\n` and followed by a blank line. `specifiedFailureLogSetup` is `specifiedSnapSetup` with `specifiedLogList` inserted before `## Administrator account`.

| Test case | File | Notes |
| --- | --- | --- |
| TC-001 | `internal/service/amazon_failure_log_docs_test.go` (new) | `failedLineError(deployment)`: the token from `documentedLine(…, failedLinePrefix)` must be in the failed-check item, equal `Amazon request failed url=… at=… status=… cause=… response=…`, and have `fieldNames` equal to `url, at, status, cause, response`. Runs on the real file and on `specifiedFailureLogSetup`, which pass. Fixtures that must fail: the former line; `response=…` before `cause=…`; `formerFailedLine` added as a second token in the same item; the list moved under `## Administrator account`; the list only in a fenced block. |
| TC-002 | `internal/service/amazon_failure_log_docs_test.go` (new) | `responseSentenceError(deployment)`: needs one sentence of `sentences(<failed-check item>)` that starts at or after the token's end and contains `response`, `start of the page` and `empty` (case-insensitive). Runs on the real file and on `specifiedFailureLogSetup`, which pass. Fixtures that must fail: no sentence; no "empty"; no "start of the page"; "start" and "page" only from the causes ("could not start", "the page did not load"); the words split over two sentences; the sentence only in the no-price item; the sentence only under `## Administrator account`. |
| TC-003 | `internal/service/amazon_failure_log_docs_test.go` (new) | `logListError(deployment)` requires exactly three items. The start item's token is exactly `Amazon browser not found; Amazon checks will fail until Chromium is installed or …`. The failed-check item contains the causes sentence word for word. The token from `documentedLine(…, noPriceLinePrefix)` is exactly `Amazon check found no price url=… at=… status=… cause=…`, and its item contains `It is not counted as a failed request.` after it. Fixtures that must fail: `response=…` added to the no-price line; the causes sentence removed; the no-price item removed. The test also runs `assertInstallationDocumentation`, `snapSentenceError` and `lookedUpNamesError` on `specifiedFailureLogSetup`. The #204 and #226 tests already check the real file. |
| TC-004 | `internal/service/amazon_failure_log_test.go` (new) | Each case gets its own store and service: `openHumanStore`, `insertPricedAmazonItem`, `startHuman(t, database, dir, now, fakeAmazon(…), 210)`, `captureRetrievalLog`, `refreshPrice`. The `request_error` result has `RequestURL`, `HTTPStatus: 503`, `Message` and `FailureCause` set to `Amazon returned HTTP 503`, and `ResponseExcerpt` set to `<html><body>Service Unavailable fixture</body></html>`. The logged line is the text from `Amazon request failed` to the end of its log line, with the logger prefix dropped, and there must be exactly one. The test checks that `fieldNames` of the logged line equal those of the documented line read from `deploy/README.md`. It also checks that `documentedPattern` matches the logged line, with capture 4 (cause) and capture 5 (response) equal to the fake's cause and excerpt. Subtests: an empty excerpt (the line ends with `response=`, and the captures are the cause and ""); the excerpt `<p>cause=fake status=999</p>` (exact captures; field names are not compared because the excerpt holds `=`); `price_not_found` with `FailureCause` set to `Amazon page showed no detectable euro price` (no failure line; the `Amazon check found no price` line has the `fieldNames` of the documented no-price line, without `response`); `succeeded()` (no failure line). Not parallel, because the logger is global. |
| TC-005 | `internal/service/amazon_failure_log_test.go` (new) | Uses the harness of #204 TC-007: `amazontest.Browser(t)`, `newAmazonFixture(t, page(amazontest.Fixture(t, "robot-check-en.html")))`, and `collectorByItem` with `B0RESPR210` → `retrievalBrowser(t, browser, server.URL, 30*time.Second)` and `B0RESPA210` → `retrievalBrowser(t, <missing path in t.TempDir()>, server.URL, 30*time.Second)`. Then `insertPricedAmazonItem`, `startHuman(…, 210)` and `captureRetrievalLog`. Refresh price runs for (a), then (b), then (a) again, each at its own `scheduleDay(6, m)`. Each line is found with `checkLogLine`, cut at `Amazon request failed`, and must match `documentedPattern` of the documented line. (a): the cause contains `robot check` (case-insensitive) and none of `<`, `Enter the characters` or `fixture-amzn-204`; the response is non-empty, at most 2,048 bytes, starts with `<html` and contains `<title dir="ltr">Amazon.fr</title>`. (b): the cause contains `browser could not start` and the missing path, and the response is "". After the three checks, `amazonFailures` is 3, both items are `retrieval_error` with a latest price of 1999 cents and are not stopped, and `retrievalSecret` is not in the log. |

No e2e test: the interface does not change. The whole suite must keep passing: `go test ./...` (including #204 TC-007 and TC-011 and #226 TC-001 – TC-004) and the Playwright tests in `tests/`.

### Files to change

| File | Change |
| --- | --- |
| `deploy/README.md` | Failed-check item of the log list (TS-AMZ-BRET-009) |
| `internal/service/amazon_failure_log_docs_test.go` (new) | TC-001, TC-002, TC-003 |
| `internal/service/amazon_failure_log_test.go` (new) | TC-004, TC-005 |

### Refactoring

None. The new tests reuse the unexported helpers of the #204 and #226 tests in the same package.

### Verification

`go vet ./...`, `gofmt -l .`, `go test ./...` with Chromium available (`CI` set, so browser tests cannot skip), and the Playwright suite (tests unchanged). No manual follow-up outside the developer's paths. No open question.
