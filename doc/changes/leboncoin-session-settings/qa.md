# QA execution: leboncoin-session-settings

Status: executed and retested. QA-SET-F01 and QA-SET-F02 are fixed and pass the retest (see Retest). The retest found one new finding, QA-SET-F03, which is recorded for adjudication.\
Tested code/specification revisions: commit `509460f` plus the uncommitted working tree of 2026-10-03 at about 18:50 +02:00. That tree has 24 modified tracked files and the untracked files of this change. SHA-256 prefixes: `src/components/SettingsModal.tsx` `1060235a0350`, `src/App.tsx` `61e316575873`. Specifications: `doc/specifications/{app-header-menu,leboncoin-session-settings,leboncoin-session-collection,leboncoin-session-capture,leboncoin-session-file-removal}/` and `API_SPECIFICATION.md` (LeBoncoin session settings section, approved 2026-10-03).\
Environment: Linux. Go 1.25.0. Node 22.23.3. Playwright 1.63.0 driving Google Chrome 149.0.7827.53 (`/usr/bin/google-chrome`): headless for the application UI, and headed on display `:1` for the capture helper.\
Executed at: 2026-10-03, 18:50–19:02 +02:00\
Tester: independent QA agent (stage 7)

## Setup

- Frontend: `npm run build` produced `dist/`. It was copied into the ignored `web/static/dist` only for the Go builds and removed afterwards.
- Real application: `cmd/pricefollower`, built to the scratchpad, run with `PORT=3517 HOST=127.0.0.1 PRICEFOLLOWER_DATA_DIR=<scratchpad>/qa2/data-a PRICEFOLLOWER_CHECK_TIMES=03:00`. Base URL: `http://127.0.0.1:3517/`. It was used for the restart and persistence case.
- Fixture harness: a QA-only `main` kept in the scratchpad and compiled with `go build -overlay` as a virtual `cmd/qaharness/main.go`, so no file was added to the repository. It is derived from the harness of `doc/changes/leboncoin-session/qa.md`, with the removed `LeboncoinSessionFile` reference dropped. It uses the real `config.Load`, `store`, `service` and `httpapi`, and does not start the scheduler.
  - It replaces `http.DefaultTransport` with a fixture driven by a control file: status, price, expected `datadome` value (any other value gets a 403), `Set-Cookie` headers, listing ID and delay.
  - For each upstream request it records the host, the path, whether a `datadome` cookie was sent, and a 4-byte SHA-256 prefix of the cookie. Values are never recorded.
  - It served the same embedded frontend on the same base URL, `http://127.0.0.1:3517/`.
  - In live mode (`QA_REAL=1`) it wraps the real transport instead. It then records only the response status and the cookie and attribute **names** of each `Set-Cookie` header.
- Data: disposable scratchpad data directories (`data-a`, `data-b`, `data-live`). Session values used in the tests were synthetic (`QAsynth…`, 4096×`x`, `Q`+3000×`a`). The one real captured value is described under "Live cases"; it was never printed or recorded.
- Application routes visited: only `http://127.0.0.1:3517/`, where the menu and the modal live. The modal has no route. API: `http://127.0.0.1:3517/api/v1/settings/leboncoin-session`, `/api/v1/items`, `/api/v1/items/{id}/refresh`.
- Listing inputs: `https://www.leboncoin.fr/ad/voitures/3245888872`, served by the fixture in fixture cases and live in L-02 and L-03; `https://www.leboncoin.fr/ad/voitures/1` (live, nonexistent listing); `https://example.com/ad/voitures/1` (rejected locally, never requested).

## Exploratory session

Generator: `qa2/explore.mjs` (in the scratchpad). It uses a 32-bit LCG (`s = s*1664525 + 1013904223 mod 2^32`) and runs 40 steps per seed through Playwright against the fixture harness (`data-b`).

- The first random number picks the viewport: 400 px or 1280 px wide.
- Available actions:
  - With the modal closed: open Settings through the menu, change the session through the API (a simulated other browser), or reload the page.
  - With the modal open: save a random input (raw, `datadome=`, `Cookie:` form, whitespace only, `bad value`, `a=1`), Escape, Cancel, change the session through the API, or save unchanged.
- Invariants checked after every step:
  - On open, the modal shows exactly the stored value.
  - A valid save closes the modal and stores the extracted value. An empty save clears.
  - An invalid input keeps the modal open and leaves the store unchanged.
  - A save after an API change leaves the store unchanged and keeps the modal open (409 path).

| Seed | Steps | Failures | Notable sequence |
| --- | --- | --- | --- |
| 20261003 | 40 | 0 | 28 apiChange → 29 apiChange → 30 save `QAsynthX30` gives a 409, store kept → 32 save `a=1` (invalid) → 33/34 save again (still invalid, store unchanged) |
| 777 | 40 | 0 | four consecutive API changes (21–24) → reload → open shows the latest value → save unchanged succeeds |
| 4242 | 40 | 0 | 3 open → 4 apiChange → 5 save unchanged gives a 409; repeated at 14/15 and 36/37 |

The first run of seed 20261003 reported two failures at steps 33 and 34. These were a generator error (it expected an invalid `a=1` re-save to succeed), not an application defect. The generator was fixed and all three seeds were rerun. The full logs are in `qa2/explore-<seed>.json`.

## Executed cases

URL key: **A** = `http://127.0.0.1:3517/` (served by the fixture harness, unless the case says the real binary); **API** = `http://127.0.0.1:3517/api/v1/settings/leboncoin-session`.

### Header menu

| Case | Requirement | URL | Listing input | Actions | Expected | Observed | Result |
| --- | --- | --- | --- | --- | --- | --- | --- |
| UI-MENU-01 | FR-APP-MENU-001, 004 | A | – | Load the list page at 1280 px | Profile icon at the right end of the header, after **Add item**; collapsed state exposed | Button named "Application menu" at x=1232 (Add item at x=1094); `aria-expanded=false` | Pass |
| UI-MENU-02 | FR-APP-MENU-002, 003 | A | – | Focus the trigger, press Enter | Menu opens with only "Settings" | One `menuitem`, "Settings"; `aria-expanded=true` | Pass |
| UI-MENU-03 | FR-APP-MENU-002, 004 | A | – | Press Escape with the menu open | Menu closes; focus returns to the trigger | Menu closed; the active element is the `app-menu` overflow button. The automated check first failed because it read `aria-label` on the wrong element; diagnosed with `focus.mjs`. | Pass |
| UI-MENU-04 | FR-APP-MENU-002 | A | – | Open the menu, click outside | Menu closes, nothing else happens | Closed | Pass |
| UI-MENU-05 | FR-APP-MENU-003 | A | – | Keyboard: Enter on the trigger, Enter on "Settings" | Menu closes; Settings modal opens | Modal "Settings" opened | Pass |
| UI-MENU-06 | FR-APP-MENU-006 | A | – | Open Settings with the mouse, press Tab twice (focus reaches the header **Add item** behind the modal), press Enter | The header cannot be reached behind the modal; no stacked modal | **Two modals visible at once**: "Add tracked item" and "Settings". Tab 3 also reaches the Application menu trigger. Evidence: `qa2/stacked-modals.png`, `focus3.mjs` | **Fail: QA-SET-F02** |

### Settings modal

| Case | Requirement | URL | Listing input | Actions | Expected | Observed | Result |
| --- | --- | --- | --- | --- | --- | --- | --- |
| UI-SET-01 | FR-LBC-SET-001, 004 | A | – | Store 4096×`x` through the API, open Settings | Title "Settings", one labelled field, help text mentions datadome and empty-save removal, full value shown | One field "LeBonCoin session"; help text as specified; field length 4096, equal to the stored value | Pass |
| UI-SET-19a | FR-LBC-SET-019 | A | – | Open the modal by keyboard or mouse; check focus; press Tab | Focus moves into the modal and stays there | **Focus stays on `BODY`** (checked at 100, 600 and 1500 ms, in headless and headed Chrome). The first Tab goes to the header brand link, then Add item, the menu trigger, the page Add item and the platform tabs, all outside the dialog. The **Add item** modal, by contrast, focuses its primary button. Eight Tabs after focus had been placed inside the modal did stay inside. | **Fail: QA-SET-F01** |
| UI-SET-11a | FR-LBC-SET-011, 019 | A | – | Edit the field, press Escape | Closes, discards the edit, focus returns to the trigger, no server change | As expected; the stored value and revision are unchanged | Pass |
| UI-SET-11b | FR-LBC-SET-011 | A | – | Reopen | Shows the saved value, not the edit | Saved value shown | Pass |
| UI-SET-11c | FR-LBC-SET-011, FR-APP-MENU-004 | A | – | Edit the field, click Cancel | Discards the edit; focus returns to the trigger | As expected | Pass |
| UI-SET-05a | FR-LBC-SET-005, 007 | A | – | Save `␠␠QAsynthRaw1⏎` | Stores `QAsynthRaw1`; modal closes; reopening shows it | Stored; revision 2; reopened field shows the same value | Pass |
| UI-SET-05b | FR-LBC-SET-005 | A | – | Save `datadome=QAsynthB2` | `QAsynthB2` | As expected | Pass |
| UI-SET-05c | FR-LBC-SET-005 | A | – | Save `Cookie: a=1; datadome=QAsynthC3; b=2` | Only `QAsynthC3` | As expected; the reopened field shows only the value | Pass |
| UI-SET-05d | FR-LBC-SET-005 | A | – | Save `set-cookie: datadome=QAsynthD4; Max-Age=31536000; Domain=.leboncoin.fr; Path=/; Secure` | `QAsynthD4` | As expected | Pass |
| UI-SET-05e | FR-LBC-SET-005 (duplicate identical pairs) | A | – | Save `datadome=QAsynthE5; x=1; datadome=QAsynthE5` | `QAsynthE5` | As expected | Pass |
| UI-SET-06a–h | FR-LBC-SET-006 | A | – | Save each input: `a=1; b=2`; `datadome=`; `datadome=x1; datadome=y2`; `has space`; `qa"quote`; `qa,comma`; 4097×`x`; `qa\back` | Field marked invalid with a message; modal stays open; input kept; revision unchanged | `aria-invalid=true` and `aria-errormessage` set every time. Messages: "No datadome cookie was found in the pasted text."; "The datadome cookie value is empty."; "The pasted text contains different datadome values; paste only one."; the invalid-characters message (4 cases); "The datadome value is too long (maximum 4096 characters)." | Pass |
| UI-SET-08 | FR-LBC-SET-008 | A | – | Save `␠␠␠`, reopen, save the empty field again | First save clears (`value:null`, `status:none`); reopened field is empty; second save closes with no change | Clear moved revision 6 → 7; the no-op clear kept revision 7 | Pass |
| UI-SET-17a | FR-LBC-SET-017 | A | – | Open the modal, PUT `QAsynthOTHER` through the API, type `QAsynthMINE`, Save | Warning, input kept, store stays Y | "Session changed" warning; field keeps `QAsynthMINE`; store `QAsynthOTHER` | Pass |
| UI-SET-17b | FR-LBC-SET-017 | A | – | Escape, reopen, save `QAsynthMINE` | Reopened field shows Y; save succeeds | As expected | Pass |
| UI-SET-17c | FR-LBC-SET-017 | A | – | Open, change through the API, save an empty field | Refused; store keeps the new value | Refused with the warning; store `QAsynthOTHER2` | Pass |
| UI-SET-03a | FR-LBC-SET-003 | A | – | Browser route fulfils GET with 500 `INTERNAL_ERROR`; open the modal | Error shown; save and field unavailable; Cancel available | "Could not load settings"; Save disabled; field not rendered; Cancel closed the modal | Pass. The failure was injected in the browser; the server was not broken. |
| UI-SET-03b | FR-LBC-SET-003 | A | – | Remove the route, reopen | Retries and loads | Stored value shown | Pass |
| UI-SET-02 | FR-LBC-SET-002 | A | – | GET delayed 1.5 s through a browser route | Loading indicator; Save disabled | "Loading settings…"; Save disabled | Pass |
| UI-SET-10 | FR-LBC-SET-010 | A | – | Browser route aborts the PUT; type `QAsynthFAIL`, Save | Error; input kept; store unchanged | "Could not save settings"; input kept; store unchanged | Pass |
| UI-SET-09 | FR-LBC-SET-009 | A | – | PUT delayed 1.5 s; Save, force-click Save again, press Escape | Progress shown; one PUT; no close while saving | "Saving…" and "Saving the session…" shown; 1 PUT; modal open and field disabled during the save; closed after success | Pass |
| UI-SET-19b | FR-LBC-SET-019, FR-APP-MENU-005 | A | – | 400×800 viewport, reload; open the menu; store `Q`+3000×`a`; open Settings | Trigger visible and not overlapping; menu on screen; modal usable with no horizontal scroll | Trigger at x=352, w=48; menu item at x=240–400; `scrollWidth=400`; Save within 400 px; field holds 3001 characters. Evidence: `qa2/narrow-400.png` | Pass |
| UI-SET-18a | FR-LBC-SET-018 | A | – | After the tests, read `localStorage`, `sessionStorage` and the URL | No value | None found; URL `http://127.0.0.1:3517/` | Pass |
| UI-SET-18b | FR-LBC-SET-018 | A | `…/ad/voitures/3245888872` | Page HTML and list text after collection with a session | No value | `QAsynth` absent from the page | Pass |

### Collection with a session (fixture transport)

| Case | Requirement | URL | Listing input | Actions | Expected | Observed | Result |
| --- | --- | --- | --- | --- | --- | --- | --- |
| COL-01 | FR-LBC-COL-011, 014, FR-LBC-SET-016 | API, A | `https://www.leboncoin.fr/ad/voitures/3245888872` (fixture) | Save `QAsynthW1`; fixture expects it and returns 200 with listing 3245888872 and `Set-Cookie: datadome=QAsynthRENEW1; Max-Age=3600; Domain=.leboncoin.fr; Path=/; Secure`; POST the item | Price stored; renewal stored after the verified listing | Item active, 1 234 500 cents. Session `QAsynthRENEW1`, revision 15, `expiresAt` +1 h, `lastAttempt accepted`. The upstream request carried `datadome`. | Pass |
| COL-02 | FR-LBC-COL-014 (renewal only after a verified listing) | API | same | Fixture returns 200 for **listing ID 999** with a renewal cookie `QAsynthRENEW2`; refresh | No renewal; attempt failed; price kept | Value still `QAsynthRENEW1`, revision 15; `lastAttempt failed`; item `retrieval_error` with price kept at 1 234 500 | Pass |
| COL-03 | FR-LBC-SET-014, FR-LBC-COL (403) | API, A | same | Fixture returns 403 with a renewal cookie `QAsynthRENEW3`; refresh; open Settings | Rejected; no renewal; price kept; hint shown | Value unchanged; `lastAttempt rejected`; item message "LeBoncoin rejected the saved session. Capture a new session and save it in Settings."; price kept. Modal: "LeBoncoin rejected this session on 03 Oct 2026, 16:55 UTC. Capture a new session and save it here." (`hint-rejected.png`, repeated after the session was replaced) | Pass |
| COL-04 | FR-LBC-SET-014 (other failure) | A | same | Fixture returns 500 with a matching session; open Settings | Warning without claiming a rejection | "The last LeBoncoin check using this session failed on 03 Oct 2026, 16:55 UTC for a reason other than a rejection." (`hint-failed.png`) | Pass |
| COL-05 | FR-LBC-COL-015, FR-LBC-SET-015, 007 | API, A | same | Fixture returns 200 with `Set-Cookie: datadome=; Max-Age=0; Domain=.leboncoin.fr; Path=/; Secure`; refresh; open Settings; save `QAsynthAFTERREVOKE` | Revoked status with the value shown and a "not used" warning; a new save removes the warning | `status:revoked`; price stored (1 100 000). Modal shows the old value and "LeBoncoin revoked this session on 03 Oct 2026, 16:55 UTC. It is no longer used. …" (`hint-revoked.png`). After the save, no warning. | Pass |
| COL-06 | FR-LBC-SET-014 (accepted gives no hint) | A | same | Successful check with a matching session; open Settings | No hint | No hint (`hint-accepted.png`) | Pass |
| COL-07 | FR-LBC-SET-008, FR-LBC-COL-011 | API | same | Clear through the API; refresh | Sessionless request | Upstream request had no `Cookie` header; price stored | Pass |
| COL-08 | FR-LBC-SET-013 | API | – | Every save in this session | Saving starts no check | Fixture request count rose only on POST item or refresh | Pass |

### API checks

| Case | Requirement | URL | Actions | Expected | Observed | Result |
| --- | --- | --- | --- | --- | --- | --- |
| API-01 | API: GET, empty store | API | GET on a fresh data directory | The none object | Exact none object, `Cache-Control: no-store` | Pass |
| API-02 | API: 405 | API | DELETE | 405, `Allow: GET, PUT`, `METHOD_NOT_ALLOWED` | As expected | Pass |
| API-03 | API: invalid JSON | API | PUT `{"value":` and PUT two concatenated JSON values | 400 `INVALID_JSON` | Both gave `INVALID_JSON` ("…must be valid JSON." / "…must contain one JSON value.") | Pass |
| API-04 | API: `INVALID_REQUEST` | API | `value:1`; `revision:-1` | 400 `INVALID_REQUEST` | As expected | Pass |
| API-05 | API: 413 | API | A body of about 33 000 bytes | 413 `REQUEST_TOO_LARGE` | As expected | Pass |
| API-06 | API: length limits | API | 4097×`x`; 4096×`x` | 400 too long; 200 | As expected | Pass |
| API-07 | API: 409 | API | PUT with a stale revision 0 | 409 `SESSION_CHANGED` | As expected | Pass |

### Persistence and logs

| Case | Requirement | URL | Actions | Expected | Observed | Result |
| --- | --- | --- | --- | --- | --- | --- |
| PERS-01 | FR-LBC-SET-012 | API, then A on the **real `cmd/pricefollower`** | Save `datadome=QAsynthPERSIST`; restart the harness; then stop it and start the real binary on the same data directory; open Settings | Same value after the restart | The API showed `QAsynthPERSIST` revision 19 after the harness restart. With the real binary, the modal field showed `QAsynthPERSIST` (`restart-real.png`). | Pass |
| LOG-01 | FR-LBC-SET-018 | – | grep the server logs (`log-a`, `log-b`, `log-real`, `log-live`) for `QAsynth`, the long synthetic values, and the captured live value (fixed-string match, no output) | No value | 0 matches in every log. The logs contain only listing ID, status, duration and price. | Pass |

### Live cases

These cases used one real application request to LeBoncoin and about 7 page loads by the helper's browser.

| Case | Requirement | Interface | Listing input | Actions | Expected | Observed | Result |
| --- | --- | --- | --- | --- | --- | --- | --- |
| L-01 | FR-LBC-CAP-001, 011, 014 | Capture helper, display `:1` | `https://www.leboncoin.fr/ad/voitures/3245888872` | `node scripts/capture-leboncoin-session.mjs --url … --timeout-seconds 300`; stdout to a 0600 scratchpad file, stderr separate | Window reported open; exactly one line on stdout, `^datadome=\S+$`; no value on stderr | Run 18:58:35–18:58:44 +02:00, exit 0. stderr: "A new browser window is open…", three "Page loaded; checking the listing…", "Session captured…". stdout: 1 line matching the format; stderr: 0 `datadome=` matches. No challenge appeared. The run finished in 9 s with no human action, so I could not observe whether the window had focus. The helper reports only "window is open"; it raises the window on a best-effort basis. | Pass, with the focus limitation |
| L-02 | FR-LBC-SET-005, 007, FR-LBC-COL-011 | A (harness in live mode, `data-live`) | – | Playwright: menu → Settings → fill the captured line → Save | Stored value equals the captured value | Equality checked in the script (`true`); status active; value 128 characters. The value was never printed. | Pass |
| L-03 | FR-LBC-COL-011, 014, REV-SET-004 | A | `https://www.leboncoin.fr/ad/voitures/3245888872` | Add item through the UI (Add item → "Listing URL" → Add item); wait for the check | Success with a price; renewal stored | Item active, "Renault Symbioz 1.8 E-Tech full hybrid 160ch Evolution - 25", 26 900,00 €, 17:00 UTC. One upstream request with `datadome` sent, status 200. The session was renewed (value changed, `expiresAt` set, revision 1 → 2, `lastAttempt accepted`). `Set-Cookie` **names only**: `cnfdVisitorId; Path; Domain; Max-Age; SameSite; Secure`, `__Secure-Install; Path; Domain; Max-Age; SameSite; Secure`, `datadome; Max-Age; Domain; Path; Secure; SameSite`. The `datadome` header carried no attribute beyond those already modelled. Evidence: `live-list.png` | Pass |
| L-04 | FR-LBC-CAP (`--output` removal), FR-LBC-RM | Helper | `…/3245888872` | `--output x.json` | Rejected, nothing written | Exit 1, "--output is no longer supported. …"; stdout empty; no `x.json` | Pass |
| L-05 | FR-LBC-CAP (URL validation) | Helper | `https://example.com/ad/voitures/1` | Run | Rejected before any browser starts | Exit 1, "Use a supported HTTPS LeBoncoin listing URL."; stdout empty | Pass |
| L-06 | FR-LBC-CAP (timeout, cleanup) | Helper, `:1` | `https://www.leboncoin.fr/ad/voitures/1` | `--timeout-seconds 40` | Times out with a reason; no output; cleanup | 19:00:24–19:01:06, exit 1, "Capture timed out after 40 seconds: the page was not the requested active listing…"; stdout empty; no `/tmp/pricefollower-capture-*` directory left; no helper or Chrome process left | Pass |

After L-03, the captured stdout and stderr files and every file in `data-live` (which held the renewed value) were removed with `shred -u`.

## Failures

### QA-SET-F01: Settings modal does not move focus into the dialog, and Tab leaves it (FR-LBC-SET-019)

Status: **fixed; retest passed on 2026-10-03** (R-UI-SET-19a).

Reproduction (fixture harness or real binary, any data):

1. Open `http://127.0.0.1:3517/`.
2. Click "Application menu", then "Settings" (or use Enter twice from the focused trigger).
3. Wait at least 1.5 s for the settings to load.
4. `document.activeElement` is `BODY`.
5. Press Tab. Focus goes to the header brand link "Price follower", then to Add item, the Application menu trigger, the page Add item and the Amazon tab, all outside `[role=dialog]`.

Expected: focus moves into the modal on open and stays inside while the modal is open. Reproduced in headless and headed Chrome 149. The **Add item** modal focuses its primary button, so the problem is specific to the Settings modal.

A likely cause, not verified: the `TextArea` is disabled during loading, so the modal has nothing to focus when it opens, and it receives no focus afterwards.

Escape and focus return still work (UI-SET-11a/c).

### QA-SET-F02: An Add item modal can be stacked over the Settings modal (FR-APP-MENU-006)

Status: **fixed; retest passed on 2026-10-03** (R-UI-MENU-06).

Reproduction:

1. With the Settings modal open as in F01, press Tab twice. Focus is on the header **Add item** button.
2. Press Enter.

The "Add tracked item" and "Settings" modals are then both visible (`stacked-modals.png`). Tab 3 also reaches the Application menu trigger behind the modal.

This follows from F01. Fixing initial focus will probably also fix it, but a retest is needed.

## Coverage and limitations

- **Load failure, save failure and slow responses (UI-SET-02, 03, 09, 10):** these were injected with Playwright request routing in the browser, not by breaking the server. Server-side storage failures (500) were not exercised against the UI.
- **Seeding warnings:** done through the fixture transport (403, 500, `Max-Age=0`), not by writing to the database; `sqlite3` is not installed. An expired status (`expiresAt` in the past) was not shown in the UI. Only revoked was shown.
- **Keyboard checks** covered the menu, Escape, Cancel and focus return. Screen-reader announcement was not tested with an assistive technology; only `aria-invalid` and `aria-errormessage` were checked.
- **Window focus (L-01):** whether the capture window had focus was not observable. The capture finished in 9 s with no challenge, and the helper does not report focus. A human-attended run is needed to confirm visibility and focus on a desktop with focus-stealing prevention. No verification challenge appeared, so the challenge path was not exercised. The policy was not to interact with a challenge.
- **Real browser close path:** not tested. The window closed itself on success and on timeout.
- **Live budget:** one application request to LeBoncoin; two helper runs with a few page loads each. The live listing may change. The observation time was 2026-10-03 17:00 UTC.
- **Exploratory runs** covered settings actions only. Item add/delete was not mixed in.
- **Fixture boundaries:** the harness replaces the network layer only; the collector, store, service and HTTP layers are the real code. The scheduler was not run; collection was triggered by add or refresh.

## Handoff and cleanup

- QA-SET-F01 and QA-SET-F02 go to the review adjudicator. Both require a frontend fix and a retest of UI-SET-19a and UI-MENU-06. The live L-03 observation is the supplementary REV-SET-004 evidence: only modelled attributes were seen, and the renewal was stored.
- Cleanup:
  - All QA servers, Chrome instances and helper runs were stopped.
  - The captured value files and the live database were shredded.
  - `web/static/dist` was removed after the builds; it is ignored, and another process had also removed it during the session.
  - No repository files were added except this report. `dist/` (ignored) remains from `npm run build`.
  - Scratchpad artifacts (synthetic data only) are under `qa2/`.

## Retest (2026-10-03, about 19:25–19:39 +02:00)

**Inputs.** The fixes are described under "QA fixes" in `implementation-frontend.md` and checked under "Recheck 2" in `review.md`.
- `src/components/SettingsModal.tsx` SHA-256 prefix `c53193851244`.
- The fix sets `selectorPrimaryFocus` to Cancel and enables Carbon `enableFocusWrapWithoutSentinels` for the Settings modal only.

**Setup.**
- Rebuilt the frontend (`npm run build`), the fixture harness (`go build -overlay`, as above) and `cmd/pricefollower`.
- The harness ran on `http://127.0.0.1:3517/` with a fresh data directory `data-r`.
- One fixture LeBoncoin item, `https://www.leboncoin.fr/ad/voitures/3245888872`, was served by the fixture only. No live LeBoncoin requests were made.
- Browser: Playwright with Chrome 149, headless, at 1280 px and at 400 px.
- Scripts: `qa2/retest.mjs` and `qa2/retest2.mjs` in the scratchpad.

| Case | Requirement | Actual interface URL | Actions | Expected | Observed | Result |
| --- | --- | --- | --- | --- | --- | --- |
| R-UI-SET-19a | FR-LBC-SET-019 (QA-SET-F01) | `http://127.0.0.1:3517/` | Open Settings with the GET delayed 1.2 s; read focus while loading and after loading; press Tab 12 times and Shift+Tab 12 times with no delay | Focus is in the dialog in both states and never leaves it | Focus on **Cancel** inside the dialog while loading and when loaded; 0 of 24 fast key presses left the dialog; same at 400 px | Pass |
| R-UI-MENU-06 | FR-APP-MENU-006 (QA-SET-F02) | same | With Settings open: Tab ×6 (checking whether focus reaches `header`), then Enter | Header unreachable; no stacked modal | Header never reached; at most one dialog visible (Enter on the focused Cancel closed it); same at 400 px | Pass |
| R-UI-SET-11a | FR-LBC-SET-011, 019 | same | Open, edit, press Escape | Closes; focus on the trigger; no change | Focus on the `app-menu` trigger; revision unchanged | Pass |
| R-UI-SET-11c | FR-LBC-SET-011, FR-APP-MENU-004 | same | Open, edit, click Cancel | Same as 11a | Same as 11a | Pass |
| R-KBD-01 | FR-LBC-SET-019, 007, FR-APP-MENU-002/003 | same | Keyboard only: focus the trigger, Enter, Enter; Shift+Tab to the field; Ctrl+A, type `datadome=QAsynthKEY1`; Tab to Save; Enter | Initial focus in the dialog; saves `QAsynthKEY1`; closes; focus returns to the trigger | Initial focus on Cancel; stored `QAsynthKEY1`; modal closed; focus on the trigger | Pass |
| R-KBD-02 | FR-LBC-SET-011, 019 | same | Keyboard only: Enter on the trigger, Enter; read the field; Enter on the focused Cancel | Reopened field shows the saved value; Cancel closes; focus returns | As expected | Pass |
| R-REC-ADD | Record only (Add item modal containment) | same | Open **Add item**; Tab ×10 and Shift+Tab ×10 with no delay; then Tab ×6 with 200 ms between presses | Recorded | Initial focus on the primary button inside the dialog. Fast presses: focus was outside the dialog after 11–12 of 20 presses (a "Focus sentinel" span, `BODY`, the header "Price follower" link, …). Slow presses: 0 of 6 outside. | Recorded: **QA-SET-F03** |
| R-REC-DEL | Record only (Delete modal containment) | `http://127.0.0.1:3517/items/b03e5a19289782de13017f4873e64024` (fixture item details) | Click Delete; same key sequence as R-REC-ADD | Recorded | Dialog "Delete tracked item"; initial focus on Cancel. Fast presses: 14 of 20 outside (sentinel, `BODY`, header link). Slow presses: 0 of 6 outside. | Recorded: **QA-SET-F03** |
| R-REC-STACK | Record only (FR-APP-MENU-006 for other modals) | `http://127.0.0.1:3517/` | Add item open; fast Tab until the header **Add item** has focus; Enter | Recorded | Header Add item reached behind the modal; Enter left one dialog ("Add tracked item"); no second modal | Recorded: header reachable, no stacking observed |
| R-FLAKE | Watch for the platform-tabs flake ("Recheck 2") | Vite dev server `http://127.0.0.1:4173` started by the repository Playwright config | `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/usr/bin/google-chrome npx playwright test`, twice | Both runs green | 30 passed (57.3 s); 30 passed (1.0 min); the flake did not reproduce. A first attempt without the executable variable could not launch the bundled headless shell (not installed); that is environment-only, and those tests did not run. | Pass |

### QA-SET-F03 (new, recorded only): Add item and Delete modals let fast Tab/Shift+Tab focus escape (FR-APP-MENU-006 / Carbon modal pattern)

Reproduction:
1. Open `http://127.0.0.1:3517/` and click **Add item**. Or, on an item details page, click **Delete**.
2. Press Tab 10 times, then Shift+Tab 10 times, with no delay between presses (Playwright `keyboard.press`).
3. After many of those presses, `document.activeElement` is outside `[role=dialog]`: Carbon's "Focus sentinel" span, `BODY`, or header controls including the header **Add item**.

With 200 ms between presses, focus stayed inside.

This is the same Carbon sentinel/`setTimeout` behaviour that the Settings fix bypasses with `enableFocusWrapWithoutSentinels`. These modals were not changed by this change, so this is probably pre-existing behaviour.

Impact observed: header controls are reachable behind the modal, but Enter on the header Add item did not stack a second modal. Whether a human typing speed triggers it was not established. It goes to the adjudicator to decide on criticality and scope.

### Retest cleanup

- The QA server was stopped.
- Orphaned `chrome_crashpad_handler` processes from my Chrome runs were stopped.
- `web/static/dist`, `test-results/` and `playwright-report/` were removed.
- No repository files were changed other than this report.
