# Technical specification: platform tabs

Status: ready
Functional specification: [platform tabs](functional.md), FR-PLATFORM-TABS-001–008.
Owner: distinct technical specification agent.

## Requirement mapping

| Technical ID | Functional IDs | Design and expected files | Verification |
| --- | --- | --- | --- |
| TS-PLATFORM-TABS-001 | FR-PLATFORM-TABS-001, 006 | Controlled Carbon tabs in `src/components/TrackedItemsPage.tsx`; platform state in `src/App.tsx`, initially `amazon`, survives list unmount while viewing details. | Mixed-platform rows, order, initial selection, details/back and reload. |
| TS-PLATFORM-TABS-002 | FR-PLATFORM-TABS-002, 007 | Keep existing row rendering; conditionally render Amazon offer header/cell only in Amazon panel. Five/four columns and matching empty-result colspan. | Column count, Amazon offer variants, LeBoncoin absence, price/status regression fixtures. |
| TS-PLATFORM-TABS-003 | FR-PLATFORM-TABS-003 | Existing component-local search state; derive platform items then text matches without server calls or query resets on tab change. | Trimmed/case-insensitive matches across all existing fields, cross-platform search, clearing. |
| TS-PLATFORM-TABS-004 | FR-PLATFORM-TABS-004, 007 | Always render the two tabs; preserve loading and errors before displaying global/platform/search empty states. | Empty collection, empty platform, unmatched query, loading, failure and retry. |
| TS-PLATFORM-TABS-005 | FR-PLATFORM-TABS-005, 006 | Existing global count, refresh handler and mutations; change refresh label to `Refresh all prices`. Selection never changes as a side effect of data updates. | Assert global POST, queued count, enabled refresh on empty platform, add/delete/polling. |
| TS-PLATFORM-TABS-006 | FR-PLATFORM-TABS-008 | Carbon tablist/panels provide semantics and keyboard handling; preserve scrollable identity layout, optional minimal `src/index.css` adjustments. | Selected state and panel association, arrows/Home/End, visible focus, action reachability and narrow viewport. |

## Frontend design

Use existing `Tabs`, `TabList`, `Tab`, `TabPanels`, and `TabPanel` exports from `@carbon/react`. The installed `lib/components/Tabs/Tabs.d.ts` confirms controlled `selectedIndex` and `onChange({ selectedIndex })`; the installed implementation supplies tab/panel IDs, `aria-controls`, `aria-labelledby`, selected state, hidden inactive panels, and automatic arrow/Home/End activation. Give `TabList` a descriptive accessible name such as `Listing platforms`. Amazon and LeBoncoin are separate content groups with different table schemas, as requested. This agrees with the [official Carbon tabs guidance](https://www.carbondesignsystem.com/building-blocks/core/components/tabs/guidelines) for related groups of content; its distinction from same-content filtering does not require a content switcher here.

Keep `selectedPlatform` typed as `TrackedItem["platform"]` in App, initialized to `amazon`, and pass it with a setter callback to TrackedItemsPage. Map the two fixed tabs to indexes 0/1. Preserve existing routes and navigation; no router, storage, query parameter, or new API state. Keep search in TrackedItemsPage so it survives tab changes with its existing detail-round-trip lifecycle.

Render two stable associated panels. Keep one straightforward row/table rendering path within the existing component, with conditional Amazon offer markup; do not extract an unrelated table framework. Derive arrays using `filter` so server order is retained. Share the search value across panels; render either one shared input within the selected panel or distinct platform-specific input IDs bound to the same value. Avoid duplicate DOM IDs. Preserve matching fields, trimming and case folding from the existing implementation.

For the active panel, resolve feedback in this order: existing loading, existing retrieval error (with Retry), entirely empty collection with existing first-item guidance, empty selected platform with its display name, then table with search and either matching rows or the existing no-match row. Search remains available for populated platforms. Tabs remain selectable in all these states, and add controls remain available. Existing refresh progress/error UI stays global. Count, refresh disabling and queued count use the full collection, never a filtered array. The refresh action label explicitly states its global scope.

Retain existing price, UTC date, latest successful observation, missing-title, thumbnail, status, source-link and overflow action rendering. Amazon offer states stay intact. LeBoncoin omits the offer cell altogether, including its placeholder. Existing component props provide all necessary data. Minimal Carbon-token spacing around panels and a platform-specific narrow-screen table width are allowed only when required by actual layout; preserve readable marketplace and listing ID and horizontal table scrolling.

## Backend and API

Backend: not affected. No handler, scheduler, collector, validation, persistence or migration changes. The backend already returns all items and their platform discriminator.

API: no contract change. Preserve the approved [canonical contract](../../../API_SPECIFICATION.md), whose latest listed approval is consecutive price periods on 2026-10-02. `GET /api/v1/items` returns `200 { items: [...] }` in existing order; `platform` is `amazon` or `leboncoin`, and `secondHandOffer` is nullable for LeBoncoin. No platform query parameter or separate fetch is introduced. `POST /api/v1/items/refresh` stays bodyless and returns `202 { requestedAt, itemsQueued }` for the entire collection. Details GET, add POST, delete DELETE, existing error envelopes/status codes, pending collection and polling are unchanged. Existing frontend euro conversion is unchanged. The API preservation assessment requires no new confirmation.

## Scope and refactoring

Permitted application edits: `src/App.tsx`, `src/components/TrackedItemsPage.tsx`, and optional focused `src/index.css` layout rules. Preserve existing types and API module. No refactoring is proposed or required.

Permitted verification support: one exact-pinned `@playwright/test` development dependency in `package.json`/`package-lock.json`, one focused npm test script, `playwright.config.ts`, `tests/platform-tabs.spec.ts`, and `.gitignore` entries for generated Playwright reports/results. This dependency provides durable browser tests for the requested test-first process and keyboard/layout behavior; no Vitest or general test-suite migration. Update the existing README with the command and browser prerequisite, and update only the relevant tracked-items use cases UC-TI-01/02 to describe tabs, column differences and selected-platform search. Subject specifications and this change's workflow evidence are also in scope.

## Verification plan and handoff

Configure a Chromium Playwright project with a dedicated Vite-only server at `http://127.0.0.1:4173`, strict port binding and no reuse of an unrelated server. Use `npm run dev:web` with explicit host/port/strict-port arguments. No Go server or persistent database is needed. API routes are intercepted with contract-shaped, in-memory fixture responses; reject unexpected API requests and block real external requests, including source-listing navigation. Keep fixtures explicit and small. Detail navigation may request fixture-backed `/api/v1/items/{id}` at the actual app route `/items/{id}`. Test mutations alter only fixture data.

Support optional `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH` in config while defaulting to Playwright-managed Chromium for portability. The current environment can use `/opt/google/chrome/chrome`. Use Node 22 as required by `package.json`; the coordinator has a temporary installation at `/tmp/pricefollower-node22/node_modules/node/bin`. These local paths are execution options, not hard-coded repository defaults.

Before modifying UI, execute a focused core browser test proving both platform tabs, platform-only rows, five/four columns and absent LeBoncoin offers. Record its expected failure because current UI lacks tabs (RED); infrastructure failures do not qualify. Implement the scoped UI, execute the focused suite (GREEN), and run `npm run build`. Cover the requirement mapping above with fixtures including mixed data, one/zero populated platforms, stale/pending/error/free/missing-title states, search retention, add/delete, global refresh and details/back. Use accessible role/text assertions and actual input actions rather than mirroring implementation.

Independent QA follows review/adjudication and executes the final interface at recorded actual URLs, including desktop/narrow widths, keyboard/focus, loading/retry, repeated switching and a seeded or fully recorded exploratory action sequence. Record actual commands, actions and limitations. No live scraping, deployment or backend test campaign is needed for this frontend change. No unresolved product, API or refactoring decision blocks this handoff.
