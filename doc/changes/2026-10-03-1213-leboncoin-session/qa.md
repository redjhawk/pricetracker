# QA execution: leboncoin-session

Status: executed. Offline/fixture scope complete. QA-LBC-F01 retested and **passed** (2026-10-03). QA-LIVE-001 and QA-LIVE-002 executed and **passed**. QA-LIVE-003 (Raspberry Pi) is blocked. See [Retest and live run](#retest-and-live-run-2026-10-03).
Tested code/specification revisions: uncommitted working tree on `bd5b209` (`git diff | sha256sum` prefix `89ff17e4957f`). File SHA-256 prefixes: `internal/leboncoin/session.go` 01e42bf1549e, `internal/leboncoin/collector.go` 234b9aaab3c9, `scripts/capture-leboncoin-session.mjs` 1651127e8638, `config/config.go` a23f5f685abf, `internal/service/service.go` 0c81edcf16c4. Specifications: ready working-tree versions of [capture functional](../../specifications/leboncoin-session-capture/functional.md), [capture technical](../../specifications/leboncoin-session-capture/technical.md), [collection functional](../../specifications/leboncoin-session-collection/functional.md), [collection technical](../../specifications/leboncoin-session-collection/technical.md); [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) unchanged.
Environment:
- Application base URL: `http://127.0.0.1:3417` (production mode, embedded frontend).
- Server: a QA-only `main` (derived from the coordinator's `.tmp-session-qa/main.go`, kept in the scratchpad and compiled with `go build -overlay`, nothing added to the repository) that uses the real `config.Load`, `store`, `service` and `httpapi`, and replaces `http.DefaultTransport` with a fixture. The fixture reads a scratchpad control file (status, price, expected `datadome` value, `Set-Cookie` headers, delay, ad status, listing ID, redirect). It records each upstream request's host, path and whether a `datadome` cookie was present, plus a 4-byte SHA-256 prefix of the value; values are never recorded. The frontend is `dist/` built 2026-10-03 13:20 +02:00, temporarily copied to the ignored `web/static/dist` for the build and removed afterwards. `src/` is unchanged from HEAD. The scheduler is not started by the harness.
- Per-run isolated `PRICEFOLLOWER_DATA_DIR`, `PORT=3417`, `HOST=127.0.0.1`, `PRICEFOLLOWER_CHECK_TIMES=03:00`. `LEBONCOIN_SESSION_FILE` points to scratchpad session directories (mode 0700, file 0600 unless the case says otherwise).
- Browser: Playwright 1.63.0 (repository `node_modules`) driving `/usr/bin/google-chrome` 149.0.7827.53 headless, at a 1280×900 viewport. Node 22.23.3 (`/tmp/pricefollower-node22`). Go 1.25.0. Debian, linux/amd64, display `:1` for the capture helper.
- Session values are synthetic strings with a `QAsynth` prefix. In this report they are referred to by label (A, B, C), never by value. Live LeBoncoin was not used for collection.
Executed at: 2026-10-03 13:51–13:59 +02:00
Tester: independent QA tester agent (not an implementer, reviewer or adjudicator)

Supplementary, not counted as interface tests: `go test ./...` passed (config, leboncoin, service), and `node --test scripts/capture-leboncoin-session.test.mjs` passed 13/13.

## Exploratory session

The generator is the seeded mulberry32 PRNG in the scratchpad `explore.mjs`. Each run takes 40 steps, choosing uniformly among: add (one of 5 listing IDs), API refresh-all, two concurrent API refreshes of one random item, UI "Refresh all prices", UI detail "Refresh price", rotation (`Set-Cookie` with a new synthetic value), operator import replacement (fresh `capturedAt`), import corruption, 403 toggle, server restart, and delete.

The model tracks the expected effective cookie. The fixture's `Expected` is set to it, so wrong cookies return 403. After every step these invariants are checked:
- `GET /api/v1/items` returns 200.
- No API response contains `QAsynth`.
- No item is `unavailable`.
- Price history never shrinks and a latest price is never lost.
- Touched items are `success` when the session is valid and 403 is off, and `request_error` otherwise.
- No temporary files remain in the session directory and the sidecar mode is 0600.

Timing (fixture delay 50 ms) is not seeded, so a replay reproduces the same action order but not the exact interleaving.

| Seed | Steps | Ordered actions (abridged; full log in run output) | Violations |
| --- | --- | --- | --- |
| 20261003 | 40 | restart, corrupt×2, replace(gen1), 403 on, refreshAll, corrupt×2, …, replace(gen2), add 2222222222, rotate×2 (under 403, correctly not adopted), refreshAll, uiRefreshAll, refreshAll, add 1111111111, refreshOne×2, restart×2, uiRefreshAll, add 3333333333, refreshAll, corrupt, refreshOne×2, replace(gen6), corrupt, delete, replace(gen7), refreshOne×2, 403 off, uiDetail+refresh ×2, uiRefreshAll, delete | none |
| 7 | 40 | add/delete 3245888872, replace(gen3), 403 on, refreshAll, 403 off, restart, refreshAll×2, replace(gen6), 403 on, add 4444444444, replace(gen7, gen8), corrupt, delete, restart, add 3245888872, uiRefreshAll, uiDetail+refresh, add 3333333333, uiDetail+refresh, rotate(gen9), delete | none |
| 424242 | 40 | refreshAll, corrupt, replace(gen1), add 2222222222, delete, replace(gen2), corrupt, refreshAll, 403 on, add 1111111111, corrupt, uiDetail+refresh, refreshOne×2, refreshAll, uiRefreshAll, uiDetail+refresh×2, rotate(gen3, under 403), 403 off, uiRefreshAll, delete, refreshAll, add 4444444444, delete, add 3245888872, **rotate(gen4) → restart → uiRefreshAll/refreshAll succeed with rotated value**, 403 on, replace(gen5), restart | none |

The interface URLs exercised were `http://127.0.0.1:3417/` and `http://127.0.0.1:3417/items/{id}` (ids generated by the run, e.g. `/items/a9d929708b5c087b8f9128b531af7f9f` for seed 424242). The source listing inputs were `https://www.leboncoin.fr/ad/voitures/{3245888872,1111111111,2222222222,3333333333,4444444444}`. All were served by the fixture, and none reached LeBoncoin.

## Executed cases

URL legend: **App** = application route actually visited; **Listing** = listing URL submitted as input (fixture-served, not fetched from LeBoncoin).

### Session collection (fixture transport)

| Case | Requirement / corner case | Actual interface URL | Actual source listing URL, if relevant | Preconditions and exact actions | Expected | Observed | Result / evidence / finding |
| --- | --- | --- | --- | --- | --- | --- | --- |
| QA-COL-001 | FR-LBC-COL-002, COL-007; normal journey | App `http://127.0.0.1:3417/` → `http://127.0.0.1:3417/items/2dcf8cc6b3a069029f6ee16659f61a8c` | Listing `https://www.leboncoin.fr/ad/voitures/3245888872` | Valid session A. Fixture expects A, price 1234500. Browser: LeBoncoin tab → "Add item" → fill "Listing URL" → dialog "Add item" → open the item | Success; price shown; cookie sent only to LeBoncoin | List: "QA Renault Symbioz … 12.345,00 € … Active". Detail shows latest price and 1 history row. The fixture saw 1 request to `www.leboncoin.fr/ad/voitures/3245888872` with `datadome` matching A | Pass. Screenshots `ui1-list.png` and `ui1-detail.png` (scratchpad) |
| QA-COL-002 | COL-008; challenge/403 with existing price | App `/items/2dcf8cc6…` | same | After QA-COL-001, fixture returns 403. Click "Refresh price" | `request_error`; price kept; not unavailable | UI: "Retrieval error — The latest collection attempt failed. The last successful price is preserved. Retries are continuing." Latest price is still 12.345,00 €. API: `status: retrieval_error`, `lastAttempt.result: request_error`, message "LeBoncoin session collection failed. Check the private session files or renew the session.", `priceHistory` length 1 | Pass. `ui1-detail-403.png` |
| QA-COL-003 | Deep link/refresh | App `http://127.0.0.1:3417/items/2dcf8cc6b3a069029f6ee16659f61a8c` (page reload) | — | Reload the detail page | Same detail rendered | Same detail and error state rendered | Pass |
| QA-COL-004 | COL-003, COL-001/002; missing file; Amazon unaffected | API `POST http://127.0.0.1:3417/api/v1/items` | Listing `https://www.leboncoin.fr/ad/voitures/3245888872`; Amazon `https://www.amazon.de/dp/B09XS7JWHH` | Env points to a nonexistent `session.json` in a 0700 directory. Add the LeBoncoin item, then the Amazon item | LBC `request_error` with a safe message, no upstream request; backend up; Amazon uses its normal path without the cookie | LBC: `request_error` with the safe message, **0** LeBoncoin upstream requests, list 200. Log: "LeBoncoin session files are missing, unreadable or unsafe; check private ownership and permissions". Amazon made 3 requests to `www.amazon.de`, none with a Cookie header, and returned the normal Amazon `price_not_found` (the fixture serves no Amazon price) | Pass. A real Amazon price was not testable offline |
| QA-COL-005 | COL-003; malformed JSON | API `POST /api/v1/items` | Listing as above | `session.json` = `{"version":1,` (0600) | As QA-COL-004 | `request_error` with the safe message, 0 upstream requests, list 200. Log: "LeBoncoin session format is invalid; renew the session" | Pass |
| QA-COL-006 | COL-003; unknown field | same | same | Valid import plus `"extra":true` | Rejected | Same as QA-COL-005 | Pass |
| QA-COL-007 | COL-003; disallowed domain | same | same | Cookie domain `.example.com` | Rejected | Same as QA-COL-005 | Pass |
| QA-COL-008 | COL-003; expired | same | same | `expiresAt` `2026-01-01T00:00:00Z` | Rejected, no send | `request_error`, 0 upstream requests. Log: "LeBoncoin session is expired or revoked; renew the session" | Pass |
| QA-COL-009 | COL-003; world-readable file | same | same | `session.json` mode 0644 | Rejected | `request_error`, 0 requests, "missing, unreadable or unsafe" diagnostic | Pass |
| QA-COL-010 | COL-003; directory 0755 | same | same | Session directory 0755, file 0600 | Rejected | Same as QA-COL-009 | Pass |
| QA-COL-011 | COL-003; symlinked file | same | same | `session.json` → symlink to `real.json` (0600) | Rejected | Same as QA-COL-009 | Pass |
| QA-COL-012 | COL-003; symlinked parent directory | same | same | `link/session.json` where `link` → real 0700 directory | Rejected | Same as QA-COL-009 | Pass |
| QA-COL-013 | COL-002 control; Amazon unaffected with a valid session | API `POST /api/v1/items` | LBC as above; Amazon `https://www.amazon.de/dp/B09XS7JWHH` | Valid session A. Add both | LBC success; Amazon gets no cookie | LBC success 990000. Amazon's 3 requests had no Cookie header | Pass |
| QA-COL-014 | COL-001; sessionless | API `POST /api/v1/items`, `POST /api/v1/items/{id}/refresh` | Listing as above | No `LEBONCOIN_SESSION_FILE`. Sequence: 403 → 200 → 403 | Behaves as before | 1st: `request_error` "LeBoncoin could not be reached for a price check." (the pre-existing message). 2nd: success 500000. 3rd: `request_error`, price 500000 kept. No Cookie header on any request. The log keeps the pre-existing URL/status lines | Pass |
| QA-COL-015 | COL-004; rotation persisted | API add/refresh | Listing as above | Import A. Response sets `datadome`=B (Max-Age 3600) and an unrelated `tracker` cookie | Sidecar stores B only; import untouched; next request sends B | Sidecar `session.json.state.json` mode 0600 with keys `version, importFingerprint, cookie`. Contains B, no `tracker`. Import still A. The next refresh (fixture expects B) succeeded | Pass |
| QA-COL-016 | COL-004; survives restart | same | same | Stop/start server; refresh, fixture expects B | Uses B | Success 120000 | Pass |
| QA-COL-017 | COL-007; wrong listing with Set-Cookie | same | same | Fixture returns ad `list_id` 999 plus `Set-Cookie` C | `request_error`; C not committed | `request_error`, price 120000 kept; sidecar does not contain C | Pass |
| QA-COL-018 | COL-004; deletion not resurrected | same | same | Response `datadome=; Max-Age=0` → then refresh → restart → refresh | Revocation respected and persisted | The revoking response itself recorded success 130000. The next two attempts (before and after restart) were `request_error` with "expired or revoked" and **no upstream request** | Pass |
| QA-COL-019 | COL-005; replacement without restart | same | same | While running, atomically install a new import C; fixture expects C | Next attempt uses C | Success 150000 with C | Pass |
| QA-COL-020 | COL-005; in-flight replacement | same | same | Request in flight (2.5 s delay, response rotates to B). At 0.8 s the operator installs import A. Then refresh, fixture expects A | Newer import wins; stale state is only tagged with the old fingerprint | In-flight attempt success 160000. Next attempt sent A (hash matched A) → success 170000. B exists only in the sidecar under the superseded fingerprint and is ignored, as designed (TS-LBC-COL-003) | Pass |
| QA-COL-021 | COL-002; redirects | same | Listing as above | (a) 302 to `https://evil.example/steal`; (b) 302 to `https://www.leboncoin.fr/redirected/ad/voitures/3245888872` | (a) secret not sent off-host; (b) allowed | (a) `request_error`. The fixture received **no** request for `evil.example`. (b) Success; cookie sent only to `www.leboncoin.fr` | Pass |
| QA-COL-022 | COL-006; sidecar cannot be saved | same | same | Session directory mode 0500 (still private). Response rotates to B. Refresh; restart; refresh | Price kept; sanitized diagnostic; import intact; no truncated file | 1st success 200000 (price recorded). Log: "LeBoncoin session state could not be saved; the current process retains the update, but restart may require renewal". 2nd (same process) used B → success. After restart, A was used → 403 `request_error` (expected per design). Directory contains only `session.json`, unchanged | Pass |
| QA-COL-023 | COL-004 fail-closed; damaged sidecar | same | same | Truncated sidecar beside valid import A; add item; then operator removes the sidecar and installs a fresh import | Fail closed, no send; repair works | `request_error`, 0 upstream requests, "format is invalid" diagnostic. After repair: success 100000 | Pass |
| QA-COL-024 | COL-003/005, repeated and concurrent actions; UI | App `http://127.0.0.1:3417/` (LeBoncoin tab) | Listings `…/ad/voitures/3245888872`, `1111111111`, `2222222222`, `3333333333` | Session file missing; add 4 items → UI. Install session A without restart. Double-click "Refresh all prices" plus 2× `POST /api/v1/items/refresh` plus 2× each per-item refresh concurrently (response rotates to B) | UI shows the error state; then all succeed; no corruption | Before: 4 rows "Title unavailable · Awaiting first price · Retrieval error", 0 upstream requests. All 10 concurrent POSTs returned 202. Only 4 upstream requests were made (deduplicated); the first sent A, later ones sent B. All 4 items were active at 3.000,00 € with 1 history entry. The session directory contains only `session.json` and a valid sidecar | Pass. `ui-missing.png`, `ui-burst.png` |
| QA-COL-025 | COL-009, CAP-005; secret leakage | All of the above | — | Grep all server stdout/stderr logs for `QAsynth`. Scan every API response body seen by the browser and the exploratory runs | No value anywhere | 0 matches in logs. 0 browser-observed API responses and 0 exploratory responses contained a synthetic value. Session logs omit the URL (only listing ID/status) | Pass |
| QA-COL-026 | Exploratory randomized | See Exploratory session | See above | 3 seeds × 40 steps | No invariant violation | No violations | Pass |

### Capture helper CLI (`node scripts/capture-leboncoin-session.mjs`, run from the repository root)

Listing inputs below were only parsed. They were opened in a browser only in QA-CAP-020 to QA-CAP-023. Outputs were written under scratchpad directories.

| Case | Requirement / corner case | Actual interface URL | Actual source listing URL, if relevant | Preconditions and exact actions | Expected | Observed | Result / evidence / finding |
| --- | --- | --- | --- | --- | --- | --- | --- |
| QA-CAP-001 | Help | CLI | — | `--help` | Usage, exit 0 | Usage printed, exit 0 | Pass |
| QA-CAP-002 | Missing args | CLI | `https://www.leboncoin.fr/ad/voitures/3245888872` | No args; `--url` only | Usage, nonzero | Usage, exit 1 (both) | Pass |
| QA-CAP-003 | FR-LBC-CAP-001 invalid scheme | CLI | `http://www.leboncoin.fr/ad/voitures/3245888872` | Valid 0700 output | Clear rejection before browser | "Use a supported HTTPS LeBoncoin listing URL.", exit 1 | Pass |
| QA-CAP-004 | CAP-001 other host | CLI | `https://www.leboncoin.com/ad/voitures/3245888872`, `https://evil.example/ad/voitures/3245888872` | — | Rejected | Same message, exit 1 | Pass |
| QA-CAP-005 | CAP-001 non-listing path | CLI | `https://www.leboncoin.fr/recherche?category=2`, `https://www.leboncoin.fr/ad/voitures/abc` | — | Rejected | Same message, exit 1 | Pass |
| QA-CAP-006 | CAP-001 userinfo | CLI | `https://user@www.leboncoin.fr/ad/voitures/3245888872` | — | Rejected | Same message, exit 1 | Pass |
| QA-CAP-007 | CAP-001 uppercase path (REV-LBC-001 retest) | CLI | `https://www.leboncoin.fr/AD/Voitures/3245888872` | Output to a 0755 directory (QA-CAP-007a); then a valid directory with the QA wrapper (QA-CAP-021) | URL accepted | (a) URL passed; stopped at the output check "Output directory must be owned by you with mode 0700." (b) reached "Waiting for the requested active listing…" | Pass. Interface-level confirmation of the REV-LBC-001 correction |
| QA-CAP-008 | CAP-001 trailing comma, query/fragment | CLI | `https://www.leboncoin.fr/ad/voitures/3245888872,` and `…/3245888872?utm=x#frag` | 0755 output | URL accepted | Both reached the output-directory rejection (URL accepted) | Pass |
| QA-CAP-009 | CAP-005/006 output inside repo | CLI | listing 3245888872 | `--output /home/redjhawk/src/pricefollower/session.json` | Rejected | "Keep the session file outside the repository.", exit 1, no file created | Pass |
| QA-CAP-010 | CAP-006 directory not 0700 | CLI | same | Output directory 0755 | Rejected | "Output directory must be owned by you with mode 0700.", exit 1 | Pass |
| QA-CAP-011 | CAP-006 symlinked parent | CLI | same | `outlink/s.json`, `outlink` → 0700 directory | Rejected | "Output parent directories must be real directories, without symlinks.", exit 1 | Pass |
| QA-CAP-012 | CAP-006 symlink output | CLI | same | `out/link.json` symlink (dangling) | Rejected | "Existing output must be an owned private regular file, without symlinks.", exit 1; symlink untouched | Pass |
| QA-CAP-013 | CAP-006 existing non-private output | CLI | same | Existing `wr.json` mode 0644 | Rejected | Same message as QA-CAP-012; file unchanged | Pass |
| QA-CAP-014 | CAP-006 nonexistent parent | CLI | same | `nonexistentdir/s.json` | Clear failure | "Capture failed. Check the browser setup and private output permissions, then retry.", exit 1 | Pass with note: generic wording, but it names output permissions and fails before the browser starts |
| QA-CAP-015 | Argument validation | CLI | same | `--bogus x`; duplicate `--url` | Usage | Usage, exit 1 | Pass |
| QA-CAP-016 | CAP-003 timeout bounds | CLI | same | `--timeout-seconds` 0, 3601, abc, 1.5, -1 | Rejected | "Timeout must be an integer from 1 through 3600 seconds.", exit 1 (all 5) | Pass |
| QA-CAP-017 | Browser unavailable | CLI | same | `--browser /nonexistent/chrome --timeout-seconds 5` | Clear failure | "Browser executable unavailable. Run npx playwright install chromium or pass --browser <executable-path>.", exit 1 | Pass |
| QA-CAP-018 | CAP-002/003/007 real Chrome, short timeout | CLI; visible Chrome on `:1` | `https://www.leboncoin.fr/ad/voitures/3245888872` (also `/AD/Voitures/…`) | `--browser /usr/bin/google-chrome --timeout-seconds 5`, run 4×; once with `--timeout-seconds 1`; once with `--browser /opt/google/chrome/chrome`. Existing private export `prev.json` hash recorded first | Waits until the 5 s timeout ("Capture timed out…"), nonzero, no output, cleanup | **Every run failed after 2.7–3.7 s with "Browser debugging port collision; retry capture."** and exit 1, before the listing was visited. Cleanup correct: 0 `pricefollower-capture-*` profiles, 0 helper Chrome processes, no new output, `prev.json` hash unchanged | **Fail → QA-LBC-F01.** The cleanup and preservation parts pass |
| QA-CAP-019 | CAP-003 cancel during startup (real Chrome) | CLI | same | `--browser /usr/bin/google-chrome --timeout-seconds 5`. SIGINT at 0.6 s (4 Chrome processes and 1 profile present) | Cancelled, cleanup | "Capture cancelled.", exit 130, 0 profiles, 0 processes | Pass |
| QA-CAP-020 | CAP-003/006/007 timeout in the waiting stage (**QA test double**) | CLI; visible Chrome on `:1` | `https://www.leboncoin.fr/AD/Voitures/3245888872` | `--browser <scratchpad>/chrome-marker-wrapper.sh` (real Chrome; the wrapper opens the helper's marker page through DevTools `/json/new` and exposes the port only afterwards, working around QA-LBC-F01) `--timeout-seconds 5 --output out/prev.json` (existing private export) | Timeout, nonzero, prior export preserved, cleanup | Output: "Opening an isolated visible browser…" then "Waiting for the requested active listing…". Stderr: "Capture timed out. Complete verification and try again.", exit 1 at 7.0 s. During the run the temp profile was mode 700 with 21 processes. Afterwards: 0 profiles, 0 processes (including wrapper/socat), `prev.json` hash unchanged, no `.session-*.tmp` | Pass for timeout/cleanup logic. **Not** evidence for unmodified Chrome startup. The dedicated browser may have loaded live LeBoncoin; no cookies were read or recorded |
| QA-CAP-021 | CAP-003 Ctrl+C in waiting stage (test double) | CLI | `https://www.leboncoin.fr/ad/voitures/3245888872` | As QA-CAP-020, `--timeout-seconds 60`, SIGINT at 4.5 s | Cancelled, cleanup | "Capture cancelled.", exit 130, 0 profiles/processes, no output written | Pass (test double caveat) |
| QA-CAP-022 | SIGTERM in waiting stage (test double) | CLI | same | SIGTERM at 4.5 s | Cancelled, cleanup | "Capture cancelled.", exit 143, clean | Pass (test double caveat) |

### Live and platform cases (not executed in this run)

| Case | Requirement | Actual interface URL | Actual source listing URL | Preconditions and actions | Expected | Observed | Result |
| --- | --- | --- | --- | --- | --- | --- | --- |
| QA-LIVE-001 | FR-LBC-CAP-002..007 | CLI; visible Chrome on `:1` | Listing `https://www.leboncoin.fr/ad/voitures/3245888872` | Live capture with `--browser /usr/bin/google-chrome --timeout-seconds 120` | Private 0600 export of `datadome` only | Executed in the retest: success with no challenge | **Pass.** See RT-LIVE-001 |
| QA-LIVE-002 | FR-LBC-COL-002, COL-007, COL-010 | App `http://127.0.0.1:3417/`, `/items/8e82cf66c68552f5a4c2855b9f2280c9` | Listing `https://www.leboncoin.fr/ad/voitures/3245888872` (live) | Backend with the captured session; collector and full app against live LeBoncoin | Success or a documented normal `request_error` | Executed in the retest: session success at 26.900,00 €; sessionless comparison returned 403 | **Pass.** See RT-LIVE-002a–c |
| QA-LIVE-003 | FR-LBC-CAP-008/009, COL-010 | — | — | Transfer the session to a Raspberry Pi and collect there | Server-side result observed | Not run | **Blocked/untested:** no Raspberry Pi target is available |

## Failures

### QA-LBC-F01: capture helper cannot start with installed Google Chrome 149

- Requirements: FR-LBC-CAP-002/003 (TS-LBC-CAP startup), and blocks QA-LIVE-001.
- Reproduction (desktop, `DISPLAY=:1`, repository root):
  `node scripts/capture-leboncoin-session.mjs --url https://www.leboncoin.fr/ad/voitures/3245888872 --output <0700-dir>/new.json --browser /usr/bin/google-chrome --timeout-seconds 5`
  → after about 3 s, stderr "Browser debugging port collision; retry capture.", exit 1. This happened in 4 of 4 runs with `/usr/bin/google-chrome` and 1 of 1 with `/opt/google/chrome/chrome`, which is the path shown in `doc/leboncoin-session.md`.
- Evidence of cause (direct probe, isolated temp profile, same flags): when Chrome 149.0.7827.53 is started with the URL argument `about:blank#pricefollower-x`, DevTools `/json/list` shows only a `chrome://newtab/` page. With `about:blank` or a `data:text/html,…` URL, the page shows exactly that URL. In `openBrowser` (`scripts/capture-leboncoin-session.mjs`), the first successful `/json/list` response without the `about:blank#pricefollower-<uuid>` marker is treated as a port collision and fails immediately; the code does not retry. Capture therefore never reaches the listing with this browser. When the marker page is created through DevTools instead (QA-CAP-020), the rest of the flow behaves correctly.
- Not tested: the Playwright bundled Chromium, which is not installed on this machine.
- Impact: the operator cannot capture with the installed Chrome, and the diagnostic ("port collision; retry") is misleading. No secret exposure, data loss or leftover processes were observed.
- Disposition: for the review adjudicator. The coordinator should route this to `decisions.md` and development. Retest QA-CAP-018 and QA-LIVE-001 after a fix.
- **Retest 2026-10-03: resolved at the interface.** With the corrected marker `data:text/plain,pricefollower-<uuid>`, unmodified Chrome on both paths reaches the waiting stage, times out cleanly (RT-CAP-018a–d), and completes a live capture (RT-LIVE-001).

## Coverage and limitations

- Covered:
  - empty/pending/error states in the UI;
  - invalid and unsupported listing inputs for the helper;
  - missing, malformed, unknown-field, wrong-domain, expired, unsafe-permission, symlinked file/directory and damaged sidecar cases;
  - 403 with an existing price;
  - wrong listing identity;
  - rotation, deletion and restart persistence;
  - live import replacement and in-flight replacement;
  - redirects;
  - sidecar write failure;
  - repeated and concurrent refresh, deletion and restart in exploratory runs;
  - deep-link reload;
  - secret leakage in logs and API responses.
- **Scheduled collection** was not exercised: the harness does not start `RunScheduler`, so only immediate (add) and manual (refresh) attempts were covered. The collector path is shared, but scheduled reuse is untested at interface level.
- The fixture replaces all network I/O. Real TLS, real DataDome behaviour, real Amazon pricing and cross-device acceptance are untested; see the QA-LIVE cases.
- The application ran from a QA `main` instead of `cmd/pricefollower`. It uses the real config, store, service and httpapi packages, but `cmd` startup, signal handling and development seeding were not exercised.
- QA-CAP-020 to QA-CAP-022 depend on a QA test-double browser wrapper. They verify the helper's timeout, cancel and cleanup logic, not an unmodified Chrome startup.
- No new UI was introduced, so keyboard navigation and narrow-screen layout were not separately retested. Helper cancellation by keyboard was exercised as SIGINT (the Ctrl+C equivalent).
- In the initial run, the successful capture/export path (FR-LBC-CAP-004/005) was not executed. It was later executed live in RT-LIVE-001; no challenge appeared, so capture after a human solves a slider remains unexercised.
- Multiple backend processes sharing one session directory are out of scope.

## Handoff and cleanup

- Failure QA-LBC-F01 needs adjudication and a fix, then a retest of QA-CAP-018 and QA-LIVE-001. REV-LBC-001 (uppercase path) was observed corrected at the interface (QA-CAP-007); its recheck status remains with the reviewer and adjudicator.
- Cleanup:
  - All QA servers and Chrome/helper/wrapper/socat processes were stopped and verified absent.
  - No `/tmp/pricefollower-capture-*` or probe profiles remain.
  - The temporary `web/static/dist` copy was removed.
  - No repository files were added except this report.
  - `.tmp-session-qa/` and `/tmp/pricefollower-session-qa/` were left untouched for the coordinator.
- Test data, logs, screenshots and harness sources remain only in the session scratchpad. They contain synthetic values only.

## Retest and live run (2026-10-03)

- Executed: 2026-10-03 14:07–14:10 +02:00, by the same independent QA tester.
- Revision: same working tree, except that `scripts/capture-leboncoin-session.mjs` now has SHA-256 prefix 8ec4e0fd493b (QA-LBC-F01 correction; the startup marker is `data:text/plain,pricefollower-<uuid>`). Go files are unchanged (collector 234b9aaab3c9, session 01e42bf1549e).
- Context: [decisions.md](decisions.md), the "QA-LBC-F01 correction" section of [implementation-capture.md](implementation-capture.md), and "Recheck 2" in [review.md](review.md).
- **No wrapper or test double was used.** Live LeBoncoin requests were deliberately minimal:
  - capture: 1 run;
  - direct collector: 1 with the session, 1 without;
  - full app: 1 add.
- No cookie values were printed or recorded. The value checks used `grep -F -f` against a tmpfs file that was shredded immediately.

| Case | Requirement / corner case | Actual interface URL | Actual source listing URL | Preconditions and exact actions | Expected | Observed | Result |
| --- | --- | --- | --- | --- | --- | --- | --- |
| RT-CAP-018a | QA-LBC-F01 retest; CAP-002/003/006/007 | CLI; real Chrome visible on `:1` | `https://www.leboncoin.fr/ad/voitures/1` (cannot verify) | `--browser /usr/bin/google-chrome --timeout-seconds 8 --output <0700 dir>/prev.json`. `prev.json` is an existing synthetic 0600 export whose hash was recorded first | Waiting stage reached, then timeout exit 1; export byte-identical; cleanup | Output: "Opening an isolated visible browser…" then "Waiting for the requested active listing and its verified session…". Stderr: "Capture timed out. Complete verification and try again.", exit 1 at 10.5 s. At 5 s: 13 helper Chrome processes and profile mode 700. Afterwards: 0 profiles, 0 processes, `prev.json` sha256 OK | **Pass** |
| RT-CAP-018b | same | same | same | As 018a, but `--output <dir>/new.json` (no existing file) | No output written | Same messages and exit 1; `new.json` absent; clean | **Pass** |
| RT-CAP-018c | same | same | same | As 018a with `--browser /opt/google/chrome/chrome` | As 018a | Identical outcome; `prev.json` sha256 OK | **Pass** |
| RT-CAP-018d | same | same | same | As 018b with `/opt/google/chrome/chrome` | As 018b | Identical outcome; no `new.json` | **Pass** |
| RT-CAP-019 | CAP-003 Ctrl+C in waiting stage, unmodified Chrome | same | same | `/usr/bin/google-chrome --timeout-seconds 60`, SIGINT at 5 s | Exit 130, cleanup, no output | "Capture cancelled.", exit 130, 0 profiles/processes, only `prev.json` present | **Pass** |
| RT-LIVE-001 | QA-LIVE-001; FR-LBC-CAP-002..007 | CLI; real Chrome visible on `:1` | Listing `https://www.leboncoin.fr/ad/voitures/3245888872` (live, 2026-10-03 14:09:03 +02:00) | `--browser /usr/bin/google-chrome --timeout-seconds 120 --output <scratchpad 0700 dir>/session.json`. No interaction | Verified export, mode 0600, `datadome` only | No slider or challenge appeared; exit 0 after 9.8 s. Output: "Session saved privately to <path>. Server acceptance must be checked separately." Stderr empty. File: mode 0600; keys `version, capturedAt, cookie`; `version` 1; `capturedAt` UTC; cookie keys `name, value, domain, path, secure, expiresAt`; `name` datadome; `domain` `.leboncoin.fr`; `path` `/`; `secure` true; `expiresAt` UTC, about one year ahead; value length 128 with a valid charset (value not recorded). The value does not appear in stdout or stderr. 0 profiles and processes left | **Pass** |
| RT-LIVE-002a | QA-LIVE-002; COL-002, COL-004, COL-007 | — (direct collector, the `QA_LIVE=1` mode of `.tmp-session-qa/main.go`, built to the scratchpad) | Listing as above (live, 14:09:28) | `LEBONCOIN_SESSION_FILE`=captured file; `NewCollectorWithSession(...).Collect` | Success | `result: success`, `amountCents` 2690000, title "Renault Symbioz 1.8 E-Tech full hybrid 160ch Evolution - 25". Log: "LeBoncoin session response … status=200 duration=554ms". The response rotated the cookie: sidecar `session.json.state.json` created, mode 0600, keys `version, importFingerprint, cookie`; cookie keys complete; `expiresAt` UTC; value differs from the import (not recorded). Import untouched | **Pass** |
| RT-LIVE-002b | COL-001 comparison (sessionless) | — (same binary, no env) | Listing as above (live, 14:09:32) | No `LEBONCOIN_SESSION_FILE` | Recorded honestly | `result: request_error`, `amountCents` 0, title null. Log: "status=403 content_type=\"text/html;charset=utf-8\"", "LeBoncoin returned HTTP 403" | Observed: **403 / request_error** without a session. This confirms that the session is what made 002a succeed |
| RT-LIVE-002c | QA-LIVE-002 full app; COL-002, COL-009 | App `http://127.0.0.1:3417/` (LeBoncoin tab, Add item dialog) → `http://127.0.0.1:3417/items/8e82cf66c68552f5a4c2855b9f2280c9` | Listing as above (live, 14:09:45) | QA server with the real transport (`QA_REAL=1`, no fixture), isolated data directory, session = captured import plus the sidecar from 002a. Add the listing through the UI | Price shown | Item `active`, `lastAttempt.result: success`. List: "Renault Symbioz 1.8 E-Tech full hybrid 160ch Evolution - 25 · leboncoin.fr · 3245888872 · 26.900,00 € · Active". Detail: latest price 26.900,00 € and 1 history row. Server log: "LeBoncoin session response … status=200 duration=847ms". Neither the import value nor the rotated value appears in the server log or in any of the 7 API responses captured by the browser | **Pass.** Screenshots `live-list.png`, `live-detail.png` (scratchpad) |
| QA-LIVE-003 | Raspberry Pi portability | — | — | — | — | Not run | **Blocked:** no Raspberry Pi target. Desktop success does not prove Pi acceptance (FR-LBC-COL-010) |

Retest conclusions:
- No new failures.
- QA-LBC-F01 is resolved at the interface; its closure is recorded in `decisions.md` by the adjudicator.
- Live results are a point-in-time observation from this desktop's network on 2026-10-03; LeBoncoin behaviour may change.

Retest cleanup:
- The captured `session.json` and `session.json.state.json` were removed with `shred -u`, and the temporary value file in `/dev/shm` was shredded.
- The live data directories were deleted.
- QA servers and Chrome processes were stopped; no `pricefollower-capture-*` profile remains.
- The temporary `web/static/dist` copy used for the rebuild was removed.
- The QA binaries remain only in the scratchpad.
