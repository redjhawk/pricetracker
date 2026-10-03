# LeBoncoin access experiments

Date: 2026-10-03. Target: <https://www.leboncoin.fr/ad/voitures/3245888872>.

## Result

**A manually verified browser session's `datadome` cookie worked with the existing Go collector.** On this host/network, the collector returned HTTP 200 and parsed the correct listing and price, including with its current default Chrome/154 headers. Without the cookie it returned HTTP 403. This is a demonstrated session-reuse method, not an unattended solution from a fresh session, a permanent bypass, or an implemented collector fix.

Observed data: ID `3245888872`, status `active`, title `Renault Symbioz 1.8 E-Tech full hybrid 160ch Evolution - 25`, `price_cents: 2690000` (€26,900). Price is the observed listing item price, not a guarantee of future availability or extra charges.

## Existing implementation and API assessment

The separate technical assessment inspected [the collector](../../../internal/leboncoin/collector.go), [configuration](../../../config/config.go), [service](../../../internal/service/service.go), and ARMv6 release/deployment scripts.

- The Go collector already sends Chrome-style User-Agent, Accept, language, client hints and navigation headers. Its default identity is Chrome/154, sourced from `AMAZON_USER_AGENT`.
- Its `http.Client` has no cookie jar and no custom transport. It performs one GET with a 15-second timeout; the enclosing service allows 31 seconds. It neither executes JavaScript nor retains challenge cookies.
- A 403 exits before parsing the body. Successful HTML must contain `__NEXT_DATA__.props.pageProps.ad` with the matching listing ID and active status. The successful live page still matches that format.
- Reordering `Header.Set` calls does not reproduce browser wire order: the inspected Go HTTP/1 serializer sorts headers. A Chrome header string also does not replace Go's TLS/HTTP behavior.
- ARMv6 currently deploys a single pure-Go binary; no browser is provisioned. A cookie jar can use Go's standard library; a browser-dependent production design would need separate deployment assessment.
- Frontend and [canonical API](../../../API_SPECIFICATION.md) are unchanged. The contract already treats DataDome/non-success responses as `request_error`; this investigation does not change error semantics, scheduling, storage or existing price retention.

## Environment and method

Experiments used the local x86_64 host: curl 8.14.1, wget, Lynx, Chrome 149.0.7827.53, Playwright 1.63.0 with temporary Node 22, Go 1.25.0, and temporary `curl_cffi` 0.16.3. Its `chrome` fingerprint alias resolved to `chrome150`, not the installed Chrome149. The library's numeric `http_version: 3` means HTTP/2, not HTTP/3.

All navigation used fresh, task-owned temporary profiles rather than personal browser profiles. No account login was requested. HTTP requests were bounded; browser requests included the page's normal challenge scripts/subframes. Cookie values, challenge identifiers, raw profiles and network IP addresses are excluded from repository records. The cookie export was created with mode 0600 inside a mode-0700 directory.

Initial restricted DNS access failed; the actual live runs used approved scoped network/browser execution. A search/browser tool also returned a previously crawled rendering of the listing, but that cache was **not** accepted as proof of current collector access or used to fill application prices.

## Executed cases

| ID | Client / sequence | Observed result |
| --- | --- | --- |
| LBC-01 | Plain curl, negotiated HTTP/2 | 403, `x-datadome: protected`, 771-byte challenge HTML, no listing JSON. |
| LBC-02 | wget, one attempt, save error body | 403; same enable-JavaScript challenge. |
| LBC-03 | Lynx with automatic cookie acceptance and cookie read/save files; two sequential runs | Both 403 with `Please enable JS and disable any ad blocker`. The reported Lynx continuation did not become listing access here. |
| LBC-04 | curl with current collector's Chrome154 headers, cookies retained, HTTP/2; repeat same session | Both 403; server sets a `datadome` cookie, but that unverified cookie alone does not grant access. |
| LBC-05 | Same curl session/headers forced to HTTP/1.1 | 403; changing HTTP version did not resolve the challenge. |
| LBC-06 | Fresh headless Chrome, JavaScript enabled, normal subresources, 15-second observation then reload with retained cookies | Listing 403 on both navigations. Challenge scripts loaded with 200, but iframe displayed temporary restriction. No ad data. |
| LBC-07 | Visible Chrome launched through Playwright, automation-banner flag omitted | Still exposed `navigator.webdriver: true`; automatic device check ran, then restriction/403. This was not an uninstrumented browser control. |
| LBC-08 | Separately spawned visible Chrome, fresh profile and fixed nonzero local debugging port; connect via CDP; homepage first, then listing | `navigator.webdriver: false`; JavaScript device check led to interactive verification. Both page requests remained 403 before human verification. CDP instrumentation remained present. |
| LBC-09 | `curl_cffi` Chrome150 TLS/HTTP profile with cookie-preserving session, fresh request and repeat | Both 403; fingerprint imitation alone did not grant access. This client does not run JavaScript. |
| LBC-10 | Isolated visible Chrome profile; user completed the interactive human verification | Listing response became 200 and the saved snapshot contains the correct ID, active status, title and `2690000` cents. A subsequent screenshot timeout interrupted the remaining export/reload workflow and closed the browser. LBC-11 separately establishes restart/reload success and cookie export. |
| LBC-11 | Reopen that same persisted profile, extract data, then reload | Both listing requests 200 without another user action. Correct ID, active status, title and `2690000` cents in `__NEXT_DATA__`. Challenge-resource list empty for this run. |
| LBC-12 | Actual Go collector, default Chrome154 headers, **no cookie**, after browser verification | At 08:43:15 UTC: HTTP/2 403, `request_error`. The browser verification did not simply unblock every request from this host. |
| LBC-13 | Actual Go collector, Chrome149 identity, inject only the validated `datadome` cookie | At 08:43:24 UTC: HTTP/2 200, `success`, correct title and `2690000` cents. No browser execution during this request. |
| LBC-14 | Actual Go collector, current default Chrome154 identity, same validated cookie | At 08:43:34 UTC: HTTP/2 200, `success`, same listing price. A production User-Agent change was not necessary for this observed success. |
| LBC-15 | Actual Go parser through an in-memory HTTP response containing saved successful browser HTML | At 08:43:35 UTC: `success`, `2690000` cents. Offline parser check, not another live HTTP success. |

## Reproduction and evidence

Temporary artifacts are under `/tmp/pricefollower-leboncoin-diagnostic/`: `http-tests.py`, `http-results.json`, `browser.cjs`, `direct.json`, `headed-native.json`, `native-home-first.json`, `manual-initial.json`, `manual.json`, `tls-tests.py`, `tls-results.json`, and private HTML/screenshots/profile data. These temporary files may expire and are not application dependencies.

The Go harness source is retained there as `go-probe.go`. For reproduction, it must be built from a temporary directory inside the project module so Go permits its `internal/` imports; it is not a standalone project file or production patch.

The developer's temporary Go harness calls `leboncoin.NewCollector(...).Collect(...)` directly, without service/database activity. It wraps `http.DefaultTransport`, clones each outgoing request and adds only the test cookie for exact HTTPS host `www.leboncoin.fr`. It rejects other destinations and restores the default transport on exit. It prints sanitized status/result data and never prints cookie values. The offline mode returns a local HTML fixture instead of making a network request.

The harness was compiled with Go 1.25.0, using a temporary build cache. A nonfatal read-only module-cache warning occurred; build exited successfully. The developer's initial forced-local-toolchain build attempt used Go1.24.4 and could not satisfy the module requirement; the coordinator's actual build used Go1.25.0. Executed binary commands:

```bash
/tmp/pricefollower-leboncoin-diagnostic/go-probe baseline
/tmp/pricefollower-leboncoin-diagnostic/go-probe cookie149
/tmp/pricefollower-leboncoin-diagnostic/go-probe cookie154
/tmp/pricefollower-leboncoin-diagnostic/go-probe offline
```

`go-results.json` is a labeled coordinator transcription of those observed stdout/stderr results, not a raw capture. The probe's process exit code only indicates that it emitted a report; the report's HTTP status, result, ID and amount determine acquisition success. Independent verification must assert those values explicitly.

The same harness and existing collector also cross-compiled successfully with `GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0`, producing a statically linked ARM executable. This verifies build feasibility only; it was not executed on the remote ARMv6 device and does not establish cookie portability between devices or network addresses.

## Interpretation and next implementation boundary

The successful difference in the Go cases is the validated session cookie. The experiment does not isolate every DataDome signal or prove the exact rule that caused the initial block. Executing JavaScript and storing a newly issued, unverified cookie were insufficient; the user-completed verification mattered in this session. “Enable JS” refers to JavaScript, not Java.

A narrow implementation candidate is secure cookie-file configuration plus a Go cookie jar to load a browser-validated session and retain legitimate response updates. That would let the ARMv6 collector continue using Go for price requests. It must handle missing/expired/rejected cookies as normal collection failures, preserve last successful prices, avoid secret logging, and make renewal needs understandable. These are proposed requirements, **not implemented or approved new behavior**.

No claim is made about session lifetime, automatic renewal, future CAPTCHA frequency, simultaneous workers, transfer to another IP/device, other listing categories, or remote ARMv6 success. Browser restart and short-term reuse were tested locally; scheduled collection over hours/days was not. A fresh unattended setup still fails, and a human may need to renew verification. Adding a browser service, paid scraping service, proxy, or CAPTCHA-solving service was not tested or selected.

Primary references supporting the distinction between headers and other signals: [DataDome detection models](https://docs.datadome.co/docs/threat-detection), [DataDome JavaScript integration](https://docs.datadome.co/docs/javascript-tag), and [curl_cffi impersonation documentation](https://curl-cffi.readthedocs.io/en/latest/impersonate/_index.html). These references explain possible mechanisms; the executed cases above establish the observed results for this URL.
