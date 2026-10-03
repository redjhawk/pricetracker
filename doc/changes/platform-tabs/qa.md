# Platform tabs independent interface QA

Status: executed, passed. Date: 2026-10-03; supplemental session started at 06:44:40 UTC (08:44:40 Europe/Paris).
Tester: `/root/platform_tabs_qa`, independent of specification, implementation, review and adjudication.
Revision: frozen working tree based on `1b18badb711be2721444f4c4246f7866dee443ea`. All thirteen SHA-256 fingerprints in [review.md](review.md#frozen-revision-fingerprints) were recomputed after browser QA and match. No application, dependency, specification or durable-test file was edited by QA.

Read AGENTS.md, the workflow, QA role, all four required project skills, [functional](../../specifications/platform-tabs/functional.md) and [technical](../../specifications/platform-tabs/technical.md) specifications, [approved API](../../../API_SPECIFICATION.md), implementation, review and decisions before execution. Requirement numbers below abbreviate FR-PLATFORM-TABS-001–008.

## Environment and commands

Node 22 from `/tmp/pricefollower-node22/node_modules/node/bin`; installed Playwright 1.63.0; Chrome `149.0.7827.53` from `/opt/google/chrome/chrome`. Durable cases used the config's Desktop Chrome viewport (1280×720), except its 390×844 keyboard case. Supplemental cases used 1440×844 and 390×844.

Independently executed:

```bash
PATH=/tmp/pricefollower-node22/node_modules/node/bin:$PATH PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/opt/google/chrome/chrome npm run test:platform-tabs
```

Initial sandbox attempt exited 1: `Process from config.webServer was not able to start. Exit code: 1`. Repeated with scoped local-server/browser escalation: **8 passed (20.9s)**. This is a startup limitation resolved by the authorized run, not a failed application assertion. Config launched its own strict-port Vite on `http://127.0.0.1:4173/` without server reuse.

Supplemental commands, also with scoped local-server/browser escalation:

```bash
PATH=/tmp/pricefollower-node22/node_modules/node/bin:$PATH npm run dev:web -- --host 127.0.0.1 --port 4174 --strictPort
PATH=/tmp/pricefollower-node22/node_modules/node/bin:$PATH node /tmp/platform-tabs-qa.cjs
```

Supplemental script exited 0 and reported four `PASS layout+keyboard` combinations and `PASS seeded exploration 24 steps; all API calls GET; no unexpected API requests`. It uses Playwright assertions and Node strict assertions; a failed assertion writes failure evidence and exits nonzero. Harness and machine-readable evidence remain temporary at `/tmp/platform-tabs-qa.cjs` and `/tmp/platform-tabs-qa-results.json`. Actual results are also preserved in this report; generated evidence is not committed.

Both runs intercepted every `/api/` request with isolated in-memory fixtures and failed unexpected API requests. All external HTTP requests were blocked, including Carbon font downloads. The supplemental harness intercepted `window.open` and asserted the exact URL, `_blank` and `noopener,noreferrer` arguments; this verifies source-opening intent without visiting a marketplace. No Go server, SQLite file, live marketplace request or persistent data mutation was used.

## Actual URL and listing-input catalog

| Kind | Actual URL / input | Executed purpose |
| --- | --- | --- |
| Durable application list | `http://127.0.0.1:4173/` | All eight durable cases; reload and tab switching remain on this route. |
| Durable application details | `http://127.0.0.1:4173/items/leboncoin-one` | Open fixture bicycle details, return to list and check retained LeBoncoin selection. |
| Supplemental application list | `http://127.0.0.1:4174/` | Four layout/keyboard combinations and seeded exploration. |
| Supplemental application details | `http://127.0.0.1:4174/items/qa-amazon` | Amazon title/menu keyboard actions and exploratory round trips. |
| Supplemental application details | `http://127.0.0.1:4174/items/qa-leboncoin` | LeBoncoin title/menu keyboard actions and exploratory round trips. |
| External URL stored in fixture; source-opening intent | `https://www.amazon.de/dp/B09XS7JWHH` | Amazon identity and both keyboard source actions. Not submitted in the add form; no external navigation performed. |
| External URL stored in fixture; source-opening intent | `https://www.leboncoin.fr/ad/velos/1234567890` | LeBoncoin identity and both keyboard source actions. Not submitted in the add form; no external navigation performed. |
| Actual add-form input, fixture response | `https://www.leboncoin.fr/ad/velos/1234567891` | Durable add case types this exact value; intercepted POST asserts `{url: ...}` and returns pending fixture `leboncoin-added`. No live validation or collection claim. |

These listing addresses come from the executed fixtures. Listing existence/availability was not checked and is not inferred from their shape.

## Executed durable cases

All cases below ran on 2026-10-03 in the environment above. Exact executable actions and fixtures are in [tests/platform-tabs.spec.ts](../../../tests/platform-tabs.spec.ts); case titles map one-to-one to that suite. Every row passed independent browser assertions.

| Case | Requirements / corner case | Actual interface URL | Setup and actions | Expected and actual outcome |
| --- | --- | --- | --- | --- |
| QA-PLATFORM-TABS-001 | 001, 002; mixed platforms/order | `http://127.0.0.1:4173/` | Fixture order Amazon headphones, LeBoncoin bicycle, Second Amazon item. Load page; inspect default Amazon; select LeBoncoin. | Amazon default; two Amazon rows in incoming order and five columns; LeBoncoin alone with four cells/columns, no Amazon offer header or Amazon row. Passed. |
| QA-PLATFORM-TABS-002 | 003, 005; whitespace/case, cross-platform misses | `http://127.0.0.1:4173/` | Two fixtures. On Amazon fill `  HEADPHONES `, `B09XS7JWHH`, `amazon.de`, `AMAZON`, then ` BICYCLE `; switch LeBoncoin; fill `B09XS7JWHH`, `1234567890`, then empty. | Existing fields match; wrong-platform query gives no-match span 5/4; switch retains exact query; clearing restores row; global count remains 2. Requests are only list GETs. Passed. |
| QA-PLATFORM-TABS-003 | 001, 004, 005; empty platform/collection | `http://127.0.0.1:4173/` | Begin with LeBoncoin only; load Amazon; select LeBoncoin; replace fixture items with empty array; reload; select LeBoncoin. | Named Amazon empty feedback and enabled global refresh; LeBoncoin table; then global empty guidance, both tabs, disabled refresh and Add your first item. Passed. |
| QA-PLATFORM-TABS-004 | 004, 007; loading, retrieval failure, retry | `http://127.0.0.1:4173/` | Hold list response; switch LeBoncoin; release as fixture HTTP 500; replace fixture with successful bicycle data; click Retry. | Loading then retrieval error, neither reported as successful emptiness; Retry restores bicycle and selected LeBoncoin. Passed. |
| QA-PLATFORM-TABS-005 | 005–007; refresh failure/retry, polling | `http://127.0.0.1:4173/` | Select LeBoncoin; search `bicycle`; first refresh returns fixture 500; second queues both items; update fixture bicycle title and wait for poll. | Visible refresh error; two bodyless POSTs to `/api/v1/items/refresh`; progress says 2 items; polling shows Updated bicycle, retains tab/query, restores enabled refresh. Passed. |
| QA-PLATFORM-TABS-006 | 006; cross-platform add, details/back, delete last, reload | `http://127.0.0.1:4173/`; `http://127.0.0.1:4173/items/leboncoin-one` | On Amazon add exact listing input ending `1234567891` above; select LeBoncoin; open bicycle details and return; menu-delete and confirm Added bicycle then LeBoncoin bicycle; reload. | Add does not switch Amazon; new item appears under LeBoncoin; details/back retains LeBoncoin; deleting last item produces named platform-empty state; reload defaults Amazon. Only fixtures mutated. Passed. |
| QA-PLATFORM-TABS-007 | 002, 007; missing title, stale/pending/unavailable/error/free | `http://127.0.0.1:4173/` | Amazon fixtures with stale item, pending/null title/null price, unavailable/not_found offer, retrieval_error/check_error with retained offer, check_error without prior offer; LeBoncoin zero-price fixture. Inspect Amazon then switch LeBoncoin. | Bon état, checking/no-offer/failure/retained-price feedback, Title unavailable, Awaiting first price and all status labels present; failed item retains 12,99; LeBoncoin displays Gratuit in four cells. Passed. |
| QA-PLATFORM-TABS-008 | 008; Carbon keyboard/390px | `http://127.0.0.1:4173/` | At 390×844 focus Amazon; ArrowRight, Home, End; Tab to search, Tab to bicycle title; open Options. | Expected focus/selected state and tab-panel association; visible tab outline; search/title focus; full identity width fits; table scrolls; View details is visible. Passed. |

## Supplemental layout and row-action cases

Supplemental data: one active Amazon and one active LeBoncoin item with IDs `qa-amazon` and `qa-leboncoin`, the two exact source URLs above, null thumbnails, €12.99 regular price and Amazon €8.99 good-condition offer. Titles are `QA Amazon headphones with a deliberately long descriptive title` and `QA LeBoncoin bicycle with a deliberately long descriptive title`. Timestamps are `2026-10-03T08:00:00Z`; no pending polling occurs.

The following sequence was executed for each of the four cases below:

1. Select platform; assert title and 5/4 columns. Measure marketplace/ID text bounds within its Item cell and before the Prices cell; assert no clipped identity width. At 390px assert the table exceeds its scroll container. Capture screenshot.
2. Focus selected tab, Tab to search, Tab to title, Enter; assert actual detail URL. Click Tracked items; assert same selected tab.
3. Focus title, Tab to source button, Enter; assert exact intercepted source URL and window-opening arguments. Tab to Options; assert focus and, at 390px, positive table horizontal scroll. Enter; assert View details focused; Enter; assert detail URL; return and assert tab retention.
4. Focus Options, Enter, ArrowDown; assert Open on Amazon/LeBoncoin focused; Enter; assert source intent again.
5. Focus Options, Enter, ArrowDown twice; assert Delete focused; Enter; assert dialog. Tab until Cancel (at most six presses), assert focus, Enter; assert dialog closed and item still visible. Assert no mutation request was sent.

| Case | Requirements | Actual routes | Expected / actual result |
| --- | --- | --- | --- |
| QA-PLATFORM-TABS-009 | 002, 006, 008; desktop Amazon | `http://127.0.0.1:4174/`; `http://127.0.0.1:4174/items/qa-amazon` | 1440×844; sequence passed. Table/container both 1344px, identity text 149.594px fits 150px client width. Title/source/menu/details/delete cancel keyboard actions succeeded. |
| QA-PLATFORM-TABS-010 | 002, 006, 008; desktop LeBoncoin | `http://127.0.0.1:4174/`; `http://127.0.0.1:4174/items/qa-leboncoin` | 1440×844; sequence passed. Table/container both 1344px, identity text 149.875px fits 150px client width. All keyboard actions succeeded. |
| QA-PLATFORM-TABS-011 | 002, 006, 008; narrow Amazon | `http://127.0.0.1:4174/`; `http://127.0.0.1:4174/items/qa-amazon` | 390×844; sequence passed. 1056px table in 294px scrolling container; full `amazon.de · B09XS7JWHH` visible without overlapping Prices. Keyboard source/Options access scrolls table. All actions succeeded. |
| QA-PLATFORM-TABS-012 | 002, 006, 008; narrow LeBoncoin | `http://127.0.0.1:4174/`; `http://127.0.0.1:4174/items/qa-leboncoin` | 390×844; sequence passed. 1056px table in 294px container; full `leboncoin.fr · 1234567890` visible without overlapping Prices. Keyboard source/Options access scrolls table. All actions succeeded. |

Inspected all four screenshots: `/tmp/platform-tabs-1440-amazon.png`, `/tmp/platform-tabs-1440-leboncoin.png`, `/tmp/platform-tabs-390-amazon.png`, `/tmp/platform-tabs-390-leboncoin.png`. Desktop cells remain separate; narrow screenshots show the left part of the horizontally scrollable table with complete marketplace/ID. Long title text extends into the scrollable portion of its own cell; it does not overlap the next cell. Horizontal keyboard access was verified separately by the sequence above. Screenshots use fallback fonts because external font downloads were deliberately blocked.

## Reproducible exploratory session

**QA-PLATFORM-TABS-013 — passed**, requirements 001–003, 006, 008. At 390×844 started a fresh `http://127.0.0.1:4174/` session with Amazon selected and the supplemental fixtures. Actual detail routes were `/items/qa-amazon` and `/items/qa-leboncoin` on that same origin.

Generator: unsigned 32-bit LCG, initial seed `20261003`, `state = (Math.imul(state, 1664525) + 1013904223) >>> 0`, selection `state % n`. Action options were switch platform, fill search, clear search/open details/return; 24 actions executed. The full ordered sequence is retained here even if temporary evidence expires:

| Step (zero-based) | Actual action and observed result |
| --- | --- |
| 0 | Amazon search `B09XS7JWHH`: matches. |
| 1 | Amazon search ` QA `: matches after trim/case fold. |
| 2 | Repeat ` QA `: still matches. |
| 3 | Clear search; Amazon details/back: Amazon retained. |
| 4 | Select Amazon again: empty query retained. |
| 5 | Amazon search `leboncoin.fr`: no match, colspan 5. |
| 6 | Amazon search `no-match-qa`: no match, colspan 5. |
| 7 | Clear; Amazon details/back: Amazon retained. |
| 8 | Select LeBoncoin: empty query retained. |
| 9 | LeBoncoin search `amazon.de`: no match, colspan 4. |
| 10 | Select LeBoncoin again: `amazon.de` retained. |
| 11 | Repeat LeBoncoin selection: `amazon.de` retained. |
| 12–15 | Four separate clear/LeBoncoin details/back round trips: LeBoncoin retained every time. |
| 16 | LeBoncoin search empty string: row matches. |
| 17 | Clear; LeBoncoin details/back: LeBoncoin retained. |
| 18 | Select Amazon: empty query retained. |
| 19–21 | Three separate clear/Amazon details/back round trips: Amazon retained every time. |
| 22 | Select LeBoncoin: empty query retained. |
| 23 | Clear; LeBoncoin details/back: LeBoncoin retained. |

Every step asserted its visible result, selected platform or retained input as applicable. Supplemental run recorded only GET API calls, zero unexpected API calls and no mutation calls, including after every delete cancellation.

## Coverage, limits and handoff

No defect was found; no new finding requires review/adjudication. Required tab behaviors, platform schemas, search, empty/loading/error/retry, global refresh/polling, add/delete/details/reload, relevant price/offer states, keyboard actions and both-platform narrow layout were executed successfully. This report supplements, rather than substitutes for, the independent review and decision records.

Browser verification is Chromium-only and fixture-backed. Live collection, backend validation, database persistence, external listing availability, downloaded Carbon font rendering and other browser/screen-reader combinations were not tested. Malformed/unsupported listing inputs, duplicate rejection, missing-detail errors, delete failure and unrelated backend/concurrency behavior were not exercised: this change introduces no validation, detail-page, deletion-contract or backend changes. No pass is claimed for these boundaries. Source actions prove URL-opening intent, not completed external navigation. Detail links were exercised through interface navigation; direct entry into a detail route was not a separate case.

`git diff --check` passed after execution; all thirteen reviewed fingerprints matched. Supplemental browser context was closed; in-memory fixtures were discarded. The dedicated supplemental Vite server was stopped after verification. No persistent test data was created. QA writes only this report; the coordinator owns final documentation consistency, explicit staging and the required focused commit.
