---
name: frontend-architecture
description: Choose, preserve, or evolve the structure and data flow of this application's React frontend. Use for frontend architecture decisions and cross-feature organization, not routine isolated component styling.
---

# Frontend architecture

Keep the frontend a small React 19 single-page application. It renders the server's data using IBM Carbon and does not own persistence, price collection, or domain rules.

## Structure

Organize code by the tracked-items feature, with a small shared app shell:

```text
src/
  app/                 # App entry and page/view selection
  features/items/      # tracked list, add-item form, item details, item UI
  lib/api/             # shared HTTP client and API error handling
```

Adapt these names to the repository's existing conventions. Do not add directories that would contain only a single unnecessary wrapper or introduce a second structure alongside an established one.

For v1, the app needs the tracked-items view and item-details view. Keep page-level orchestration in the page/view components. Keep item-specific rendering and controls in small feature components. Use Carbon components for common controls, data display, feedback, and layout.

## Data flow

```text
Page/view -> items API module -> backend API
     ^              |
     +-- props/state <- response
```

- Keep network requests in the API module rather than scattering `fetch` calls through presentational components.
- Pages own the loading, error, and server data state they need. Pass item data and event callbacks down through props.
- After add or delete succeeds, refresh or update the page's server-derived data. Treat backend responses as authoritative.
- Do not add a global state library, client-side database, or elaborate cache for this application unless actual requirements demonstrate that local page state and the API module are insufficient.
- Do not put scraping, scheduling, SQLite access, or price interpretation logic in the browser.

## Navigation and scope

Keep navigation minimal. Reuse an existing router if the project already has one. If not, use the simplest navigation supported by the app; do not add a routing dependency just to switch between the tracked-items view and item details. Keep URL/deep-link behavior consistent if a router is introduced later.

V1 is a shared no-login interface for European Amazon listings. Show euro item prices, latest observation time, up to three recent detections, and collection status. Do not add frontend user identity, trend calculations, alerts, or other marketplaces unless requested.

## API-first coordination

When a requested feature requires both frontend and backend changes, define the API contract first in `API_SPECIFICATION.md`; it needs no user confirmation (see AGENTS.md). Do not implement either tier before the contract is defined. The contract should state endpoint/method, request and response shapes, validation/errors, and states the UI needs to render. Then implement the frontend against that contract.

## Keep it simple

- Follow the existing build setup and package versions; avoid adding dependencies without a concrete need.
- Keep components focused on UI and orchestration. Avoid speculative abstraction, generic component frameworks, and duplicate state.
- Ensure pending, stale, unavailable, and error states remain distinguishable in the interface.
- Do not change the backend contract from the frontend. Propose a contract change through the API-first process instead.
