# QA execution: issue #111 — delete Amazon searches, per-item Amazon stop

Status: blocked (interface/browser QA); API and served-source checks executed
Tested code/specification revisions: branch `ai-dev/issue-111-review-fixes` at `e478188` (base `ai-dev/issue-111-feature`); specs `doc/specifications/amazon-searches-removal/`, `doc/specifications/amazon-request-stop/`, `doc/specifications/claude-token-settings/` (FR-CLT-SET-013), `API_SPECIFICATION.md` "Amazon per-item stop (issue #111)", use case UC-PC-02.
Environment: `npm run dev` (Vite 6.4.3 at `http://localhost:5173`, Go API at `http://127.0.0.1:3001`, `PRICEFOLLOWER_ENV=development`, open mode, seeded 6 sample items into `.data/pricefollower.sqlite`, disposable local DB). Node v22.23.4 (`fetch` and `node:sqlite` used as the HTTP client and DB editor; `curl` was not permitted in this session). Outbound requests to Amazon fail in this sandbox (`request_error`, "Amazon could not be reached for a price check.").
Executed at: 2026-11-10 02:12–02:17 UTC
Tester: QA tester agent (independent of implementation)

## Browser prerequisite (blocked)

- `npx playwright test tests/platform-tabs.spec.ts` (Playwright 1.63.0): all 8 tests fail at `browserType.launch`; `~/.cache/ms-playwright/chromium_headless_shell-1243/.../chrome-headless-shell` exits 127 with `error while loading shared libraries: libnspr4.so: cannot open shared object file`.
- Firefox and WebKit: executables not installed (`firefox-1543/firefox/firefox`, `webkit-2359/pw_run.sh` missing).
- Missing prerequisite: system library `libnspr4.so` (package `libnspr4`, plus likely `libnss3`) for Chromium; no system packages may be installed. Consequently **no interface case was executed**; the cases below are API requests and checks of the source modules served by Vite, which do not count as interface tests.

## Exploratory session

No browser session possible. Action order executed against the API (no seed; fixed order):
1. Removed routes probed. 2. Item list inspected. 3. Three Amazon items stopped by DB edit (`amazon_stopped_at='2026-11-10T01:00:00Z'` for sony, kindle, samsung). 4. Refresh all, then immediately Refresh price on sony twice (UC-PC-02). 5. Waited for results. 6. Refresh price on lego (5th consecutive failure). 7. Added an Amazon item. 8. Refresh all with all prior Amazon items stopped.
Total real Amazon requests: 7 (all failed at network level).

## Executed cases

| Case | Requirement / corner case | Actual interface URL | Actual source listing URL, if relevant | Preconditions and exact actions | Expected | Observed | Result / evidence / finding |
| --- | --- | --- | --- | --- | --- | --- | --- |
| QA-001 | Search routes removed (API spec, amazon-searches-removal) | `http://localhost:5173/api/v1/amazon-searches` (GET, POST), `/api/v1/amazon-searches/1/refresh` (POST), `/api/v1/amazon-searches/1` (DELETE) | POST body URL `https://www.amazon.fr/s?k=x` | Fresh dev server | `404 ROUTE_NOT_FOUND` | All four: `404 {"error":{"code":"ROUTE_NOT_FOUND",...}}` | Pass (API) |
| QA-002 | `GET /api/v1/amazon/requests` removed | `http://localhost:5173/api/v1/amazon/requests` | – | – | `404 ROUTE_NOT_FOUND` | `404 ROUTE_NOT_FOUND` | Pass (API) |
| QA-003 | Item field `amazonStoppedAt` present, `tracked` gone | `http://localhost:5173/api/v1/items` | – | Seed data | Every item has `amazonStoppedAt` (null), no `tracked` | 8 items, all `amazonStoppedAt: null`, no `tracked` key; LeBoncoin `null` | Pass (API) |
| QA-004 | Frontend no longer references searches | `http://localhost:5173/src/**` (21 served `.ts/.tsx/.css` modules) | – | Fetched every `git ls-files src` module from Vite | No `amazon-searches`/`AmazonSearch`/"Amazon searches" text | No match; all modules 200 | Pass (static, not interface) |
| QA-005 | FR-CLT-SET-013 token help text | `http://localhost:5173/src/components/SettingsModal.tsx` | – | Served module | Mentions LeBoncoin AI reviews only | "It is used for LeBoncoin AI reviews." – no Amazon mention | Pass (static, not interface) |
| QA-006 | Stop warnings present in UI source (FR-AMZ-STOP tab warning, item notice) | `/src/components/TrackedItemsPage.tsx`, `/src/components/ItemDetail.tsx` | – | Served modules | Tab warning while any listed Amazon item is stopped; item notice with restart hint | "Amazon checks were stopped on … after repeated failures." (latest stop); "Amazon checks of this item are stopped since …. Refresh price restarts it." | Static only; rendering **Blocked** (no browser) |
| QA-007 | UC-PC-02 main flow steps 1–3; FR-AMZ-STOP-006 failure alternative | `POST /api/v1/items/refresh`, `POST /api/v1/items/sample-sony-headphones/refresh` | `https://amazon.de/dp/B09XS7JWHH` (seed) | sony, kindle, samsung stopped by DB edit; Refresh all then Refresh price on sony at once | Refresh all `202` counting all items; stopped items not reserved; sony check runs despite stop; on failure sony stays stopped and attempt recorded; kindle/samsung get no request | Refresh all `202 itemsQueued 8`; sony `202`; attempts: sony 1→2 (`request_error` at 02:13:09), kindle 1→1, samsung 1→1, non-stopped Amazon/LeBoncoin +1; sony still `amazonStoppedAt 2026-11-10T01:00:00Z` | Pass (API) for the failure path |
| QA-008 | UC-PC-02 step 4: an Amazon answer restarts only that item | – | – | Needs a reachable Amazon answer | sony `amazonStoppedAt` → null, others stay stopped | Not executable live (Amazon unreachable). Regression test `TestRefreshItemRestartsStoppedItemWhileRefreshAllQueued` passed (`go test ./internal/service/ -count=1`) | **Blocked** live; covered by automated test only |
| QA-009 | One check per item at a time | `POST /api/v1/items/sample-sony-headphones/refresh` (2nd, 7 ms after 1st) | – | First check in progress | Second starts nothing | `202 itemsQueued 1`, but only one new attempt recorded | Pass (API) – response still reports 1 queued; consistent with existing refresh contract |
| QA-010 | 5th consecutive failure stops every non-stopped Amazon item (FR-AMZ-STOP-003) | `POST /api/v1/items/sample-lego-land-rover/refresh` | `amazon` seed lego item | 4 consecutive failures already counted | All Amazon items stopped, LeBoncoin untouched, log line | lego, nike, title-unavailable `amazonStoppedAt 2026-11-10T02:15:47.465Z`; earlier stops kept; LeBoncoin null; log "Amazon checks stopped after 5 consecutive failures at 2026-11-10T02:15:47Z" | Pass (API) |
| QA-011 | Details response carries `amazonStoppedAt` | `GET /api/v1/items/sample-kindle-paperwhite` | – | kindle stopped | Field present | `200`, `2026-11-10T01:00:00Z` | Pass (API) |
| QA-012 | Add Amazon item while others stopped | `POST /api/v1/items` | `https://www.amazon.fr/dp/B0BSHF7WHW` | All Amazon items stopped | `201`, `amazonStoppedAt: null` | `201`, `amazonStoppedAt: null`, collected immediately (1 attempt) | Pass (API) |
| QA-013 | Refresh all skips stopped items | `POST /api/v1/items/refresh` | – | 6 seed Amazon items stopped, 1 new not stopped | `202` counting all; no attempts for stopped items | `202 itemsQueued 9`; attempts unchanged for all 6 stopped; +1 for new item and each LeBoncoin item | Pass (API) |
| QA-014 | Existing browser specs | – | – | `npx playwright test tests/platform-tabs.spec.ts` | Run | Browser launch fails (`libnspr4.so`) | **Blocked** |
| QA-015 | Automated suites | – | – | `go test ./... -count=1`, `npx tsc -b --noEmit` | Pass | All Go packages ok; tsc no errors | Pass (non-interface) |

## Coverage and limitations

- No interface (browser) case was executed: tab warning rendering, item notice, removal of the searches tab, settings help text display, keyboard navigation and polling UI are **not verified in a browser**.
- The success path of a restart (Amazon answer clears the stop) could not be observed live because Amazon is unreachable from this sandbox; it relies on the regression test.
- The DB edit of `consecutive_failures=5` was not picked up because the gate loads the count at startup; the count observed was the real in-memory count (verified by reading `internal/service/amazon_gate.go`). Not a defect.
- Upgrade deletion of saved searches was not exercised against an old DB (only the fresh seed DB).

## Handoff and cleanup

- Defects found: none.
- Dev server stopped. The dev DB `.data/pricefollower.sqlite` contains QA mutations (stopped items, one added item `83b1c9a118dce36d6a608bd6f968bb44`); it is local and disposable — delete `.data/` to reseed.
- To unblock interface QA: provide `libnspr4`/`libnss3` (or a runner image with Playwright dependencies) and rerun QA-006, QA-008 (with a controlled Amazon fixture) and QA-014.
