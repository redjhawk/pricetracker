# Technical specification: tracked item identity in the list

Status: ready
Owner: distinct technical specification agent
Functional specification: [tracked item identity](functional.md), FR-TI-IDENTITY-001 through 006.

## Requirement mapping

| Technical ID | Functional ID | Technical solution | Verification |
| --- | --- | --- | --- |
| TS-TI-IDENTITY-001 | FR-TI-IDENTITY-001 | Render `item.marketplace`, a middle-dot separator with spaces, and `item.listingId` in the existing `item-asin` span below the title. | Amazon and LeBoncoin rows show `marketplace · listingId`, with the full external ID. |
| TS-TI-IDENTITY-002 | FR-TI-IDENTITY-002 | Remove Marketplace header and its row cell; remove unused Carbon `Tag` import; change no-match `colSpan` from 6 to 5. | Five headers/cells remain; no standalone Marketplace header; no-match cell spans five columns. |
| TS-TI-IDENTITY-003 | FR-TI-IDENTITY-003 | Preserve existing title Button, callbacks, fallback, images, source links, and menus. | Execute title navigation, keyboard activation, source access and deletion against isolated fixture data. |
| TS-TI-IDENTITY-004 | FR-TI-IDENTITY-004 | Preserve current memoized local search and all five searchable fields. | Search marketplace, listing ID, title, ASIN and platform; test zero matches and clearing the search. |
| TS-TI-IDENTITY-005 | FR-TI-IDENTITY-005 | Preserve existing loading/error/empty/refresh branches and price/offer/status rendering. | Exercise loading, empty collection, failure/retry, refresh progress and representative statuses/offers. |
| TS-TI-IDENTITY-006 | FR-TI-IDENTITY-006 | Reuse existing identity flex column, title wrapping, secondary text color/type and narrow-screen scroll container. | At 390px and desktop widths inspect long title/full identity visibility, absence of overlap, scroll reachability and keyboard focus. |

## Frontend

Only `src/components/TrackedItemsPage.tsx` needs an application change. It already receives `TrackedItem` props, with nonnullable string `marketplace` and `listingId`. No added fetching, state, mapping, dependency or helper component is required. The identity span renders the existing values directly as React text: `marketplace · listingId`. The middle dot separates the two values without changing their meaning. Do not substitute `platform`, `asin`, or the internal `id`.

Keep existing Carbon Table, TableHead, TableHeader, TableBody, TableRow and TableCell, including the accessible Actions header. Installed `@carbon/react` TableCell declarations support numeric `colSpan`; no new Carbon APIs are required. A plain metadata span is the established presentation for listing identity; remove the standalone marketplace Tag.

Keep semantic row/header alignment at five columns: Item, Prices, Amazon second-hand offer, Status, Actions. Preserve focusable title and menu controls, accessible action names, decorative image treatment, title fallback, and all event callbacks. No new interactive element is introduced. Loading/errors and navigation remain owned by the existing page/app flow.

Preserve `src/index.css`: `.item-identity` already lays out the title and metadata vertically, `.item-title` wraps, and `.item-asin` uses existing secondary text styling. The existing rule below 960px uses horizontal overflow with a `66rem` table minimum. Removing the standalone column satisfies the requested consolidation; this change does not redesign responsive table sizing. QA must verify complete readable metadata with scrolling. If execution reveals a defect requiring CSS changes, return it through review/decisions rather than broadening scope silently.

## Backend

Not affected. Existing Go HTTP/service/store/collector layers already supply marketplace and external listing ID. No handler, validation, persistence, migration, scheduling, concurrency, cancellation, or scraping change is needed. No database or upstream marketplace request is required to verify this presentation change.

## API

No contract change. Preserve the approved [canonical API specification](../../../API_SPECIFICATION.md), including the contract revisions through 2026-10-02. `GET /api/v1/items` still returns `200` with `{ "items": [...] }`; each item already contains string `marketplace` and `listingId`, nullable `title`/`thumbnailUrl`, and existing prices/status/offer fields. Marketplace is the host (for example `amazon.fr`, `leboncoin.fr`); listing ID is the external ASIN or numeric ad identifier. Existing safe JSON errors, asynchronous refresh semantics, detail/add/delete endpoints and integer euro-cent mapping remain unchanged. No request field or endpoint is added. An unchanged approved single-tier contract needs no new user confirmation.

## Scope and refactoring

Application scope: `src/components/TrackedItemsPage.tsx` only. Documentation scope: these subject specifications, per-stage reports under `doc/changes/tracked-items-identity/`, and the relevant main-flow statement in `doc/use-cases/tracked-items/UC-TI-01-review-tracked-items.md`. The developer must update use-case step 3 to state that marketplace precedes listing ID below the title in Item and no standalone Marketplace column remains. Item details is a separate, unaffected subject.

No refactoring is necessary or proposed. Do not rename existing CSS classes, reorganize components, change types/API clients, modify backend files, add dependencies, or alter unrelated workflows. Preserve existing user changes.

## Verification and unresolved questions

Run `npm run build` for TypeScript and production bundling. Independent reviewer checks every mapping above and exact scope. QA runs the actual application with Vite at `http://localhost:4173/`, Chrome `/opt/google/chrome/chrome`, and installed `playwright-core` under `/tmp/pricefollower-ui-qa/node_modules/`. Browser API interception supplies temporary contract-shaped fixtures; do not mutate the shared database or contact marketplaces. These are planned URLs/setup, not execution claims; QA must record actual URLs and actions used.

Cover both platforms, absent title/thumbnail, long title, no matches, empty/loading/failure/retry/refresh, representative price and offer states, keyboard navigation, narrow-screen scrolling, repeated and randomized actions with a recorded seed or complete ordered sequence. Source-listing URLs come from fixture or existing contract values and are recorded as external inputs, distinct from the application URL; block external navigation in QA and verify the attempted URL. Tests of existing mutation controls use isolated intercepted responses only. Record results and limitations in QA Markdown.

No unresolved functional or API decision blocks implementation.
