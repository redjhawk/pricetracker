# QA execution: leboncoin-old-price

Status: executed (live old-price source verification blocked)
Tested code/specification revisions: branch `ai-dev/issue-42-old-price-frontend` at `28d2302`; [functional.md](../../specifications/leboncoin-old-price/functional.md), [technical.md](../../specifications/leboncoin-old-price/technical.md), [decisions.md](decisions.md).
Environment: `npm run dev` (Vite UI `http://localhost:5173`, Go API `http://127.0.0.1:3001`, `PRICEFOLLOWER_ENV=development`, open mode), Playwright 1.63.0 headless Chromium, viewport 1280x900 and 360x800. Isolated dev DB `.data/pricefollower.sqlite`, freshly seeded with 8 sample items at startup; removed after QA.
Executed at: 2026-10-05, ~20:43–21:00 UTC
Tester: QA tester agent (independent of implementation)

## Exploratory session

No seeded randomness; scripted Playwright actions in this order:

1. Live probe of LeBoncoin from Chromium (REV-004).
2. Inserted test rows into the dev DB (node `node:sqlite`), mimicking what `RecordCollection` writes: item `sample-lbc-pool` gets old price 129,00 € with `observed_at=''`/`is_old_price=1` (undated), old price 119,00 € dated `2026-09-20T10:00:00.000Z`/`is_old_price=1`, collected price 109,00 € dated `2026-10-01T10:00:00.000Z`; item `sample-lbc-donation` (current price "Gratuit") gets undated old price 5,00 €.
3. List page, LeBoncoin tab; detail pages; API JSON; LeBoncoin add via UI; item refresh; narrow viewport; deletions; DB checks.

## Executed cases

| Case | Requirement / corner case | Actual interface URL | Actual source listing URL, if relevant | Preconditions and exact actions | Expected | Observed | Result / evidence / finding |
| --- | --- | --- | --- | --- | --- | --- | --- |
| QA-01 | REV-004 real `old_price` shape | n/a (direct Chromium navigation) | `https://www.leboncoin.fr/recherche?text=ps5` | Playwright `goto` search page to find a listing with a crossed-out price | Page with `__NEXT_DATA__` | HTTP 403 (anti-bot), no `__NEXT_DATA__`, no ad links | Blocked; shape/unit/date remain unverified, REV-004 stays open |
| QA-02 | FR-LBC-OLD-PRICE-001/007, live add | `http://localhost:5173/` | `https://www.leboncoin.fr/ad/consoles/3080123456` (arbitrary ID, not a known listing) | Add item dialog, paste URL, submit, wait 8 s | Item accepted; failed first collection records no old price | Item created (9 items); DB: 0 price observations, 1 attempt `request_error` "LeBoncoin could not be reached for a price check." | Pass for FR-007 (failure path). Success path with real old price not executable (403) |
| QA-03 | FR-004/005 details, undated + dated + ordering | `http://localhost:5173/items/sample-lbc-pool` | — | Open detail after inserting rows (step 2) | Newest first; dated old price by date; undated last with "Old price" | Rows: 1 99,00 € 05 Oct 2026 16:43 UTC; 2 109,00 € 01 Oct; 3 119,00 € 20 Sept 2026 10:00 UTC; 4 129,00 € "Old price". Latest price stays 99,00 € | Pass |
| QA-04 | FR-004 accessibility | same as QA-03 | — | Inspect DOM of "Old price" | Plain text, no `<time>` | `<div class="cds--structured-list-td" role="cell">Old price</div>` | Pass |
| QA-05 | FR-003/005 API contract | `http://127.0.0.1:3001/api/v1/items/sample-lbc-pool` (API check, supporting only) | — | GET item | `oldPrice` flag; `timestamp:null` only for undated | priceHistory last entry `{"amountCents":12900,"timestamp":null,"oldPrice":true}`, dated `oldPrice:true` with timestamp; collected `oldPrice:false` | Pass |
| QA-06 | FR-005 list price periods, no new column | `http://localhost:5173/` LeBoncoin tab | — | Open list, LeBoncoin tab | Old price within price periods; no new column | Columns Item/Prices/Status only. Pool: 99,00 €, 109,00 €, 119,00 € 20 Sept (last three; undated 129 € is 4th, not shown by design). Donation: "Gratuit 05 Oct, 12:43" then "5,00 € Old price" | Pass |
| QA-07 | Corner: old price higher than free current price | `http://localhost:5173/items/sample-lbc-donation` | — | Open detail | Gratuit latest, 5,00 € "Old price" below | As expected | Pass |
| QA-08 | FR-002 refresh keeps old price unchanged | `http://localhost:5173/items/sample-lbc-donation` | `https://www.leboncoin.fr/ad/bricolage/3277184962` (seed URL) | Click "Refresh price", wait 10 s, reload | Old-price entry unchanged, none added | History unchanged (Gratuit; 5,00 € Old price). Live refresh itself failed (403), so a successful refresh with old price could not be exercised | Pass (limited) |
| QA-09 | FR-009 Amazon unaffected | `http://localhost:5173/items/sample-kindle-paperwhite` | — | Open detail | No "Old price" | 3 dated rows, no "Old price"; Amazon second-hand offer shown as before | Pass |
| QA-10 | Narrow screen | `http://localhost:5173/items/sample-lbc-pool`, `http://localhost:5173/` at 360x800 | — | Open pages, screenshot, measure scrollWidth | "Old price" readable | Row "4 129,00 € Old price" renders cleanly. List scrollWidth 360. Detail scrollWidth 477 (Amazon detail also 411; caused by long breadcrumb/title, not by this change) | Pass; pre-existing detail overflow noted as QA-OBS-01 (out of scope) |
| QA-11 | FR-008 delete cascade | `http://localhost:5173/items/sample-lbc-donation`, then `http://localhost:5173/items/f7c4fd16d408e86896cb2af4ffc77bca` | — | Delete, confirm in dialog; query DB | Item and all observations (incl. old price) gone | Redirect to `/`; DB counts: items 0, price_observations 0, attempts 0 for both | Pass |

Console errors during all UI runs: none.

## Coverage and limitations

- REV-004 not resolved: LeBoncoin returns 403 to headless Chromium from this environment, so the real `old_price` shape, unit (euros vs cents) and any date field could not be observed. The add-time success path (FR-001, FR-006 with a real listing) was therefore verified only via DB-inserted rows equivalent to what the store writes, not end-to-end. Recommend manual verification on a reachable network with a listing showing a crossed-out price.
- Dated old price was only displayed from inserted data; the collector currently never produces a date (documented limitation).
- Not covered: protected (logged-in) mode, keyboard-only navigation, refresh-all.

## Handoff and cleanup

- Defects in this change: none found.
- QA-OBS-01 (non-blocking, pre-existing): detail page horizontally overflows at 360 px on any item; not caused by issue #42.
- REV-004 remains open (blocked by 403), to be kept as a known limitation in decisions.
- Cleanup: dev server stopped; the dev DB directory `.data/` (created and seeded by this QA run) and the temporary Playwright script were removed; working tree clean except this file.
