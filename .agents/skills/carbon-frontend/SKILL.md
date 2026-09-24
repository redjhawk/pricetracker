---
name: carbon-frontend
description: Build or update this application's React 19 frontend using IBM Carbon Design System. Use when implementing UI, forms, tables, navigation, or visual states in the frontend.
---

# Carbon frontend developer

Implement the frontend in this repository with React 19 and IBM Carbon Design System. Follow existing project structure and installed package versions. Keep the frontend as a presentation client of the backend: it must not connect directly to SQLite or scrape marketplace pages.

## Carbon-first workflow

- Inspect the app's package manifest, installed Carbon packages, existing components, and styling conventions before choosing imports or APIs. Reuse existing project choices.
- When Carbon MCP tools are available, consult `code_search` for component APIs and `docs_search` for usage, tokens, and accessibility before coding. For charts, consult the Carbon chart tool/documentation first. Do not invent components, props, variants, or tokens.
- If Carbon MCP is unavailable, use the installed package's exports/types and current official Carbon documentation as the source of truth. State uncertainty rather than guessing at version-specific APIs.
- Use Carbon React components and Carbon's layout, typography, spacing, color, and theme systems. Prefer Carbon Grid/Column for layout where appropriate. Include required Carbon styles and theme setup using the project's supported package version.
- Do not introduce Tailwind, another utility styling framework, hard-coded brand colors, arbitrary spacing values, or inline token strings. Use Carbon tokens and project stylesheets; keep custom styling focused on application-specific layout.
- Use semantic HTML and Carbon patterns. Ensure keyboard operation, visible focus, sensible heading structure, accessible names, labels and validation feedback, sufficient contrast, and reduced-motion support where motion is used.

## Product context

This frontend is for a simple price tracking service. V1 displays the server's shared collection of European Amazon listings, prices in euros, latest detection time, up to three recent detections, and collection status. It supports adding a listing URL and deleting an item. There is no login. Do not add price trends, notifications, or non-Amazon platform flows unless requested.

Treat server data as potentially pending, stale, unavailable, or partially loaded. Make these states understandable and preserve the distinction between last successful price and collection errors. Present item prices only; do not imply shipping or tax inclusion.

## Implementation guidance

- Before making UI changes, identify the relevant files and the data contract/API already present. Coordinate with actual backend contracts instead of fabricating response fields.
- For any feature that requires changes to both frontend and backend, use API-first development: draft the API contract before implementation and present it to the user for confirmation. Do not implement either tier until the user confirms the contract. Once confirmed, implement both tiers against that contract. A change confined to the frontend that uses an already-confirmed API does not require a new confirmation.
- The proposed contract should define the endpoint/method, request and response shapes, validation and error behavior, and any loading/status semantics the UI depends on. Keep it focused on the requested feature.
- For a new screen request, clarify its purpose, content, component set, layout, representative data, and files from the request/context; do not expand scope with unrelated screens or files.
- Use Carbon components for interactive controls, data display, notifications, and loading/empty/error states when suitable components exist.
- Keep data fetching and mutation behavior explicit, handle loading and failures, and refresh or update displayed data after successful add/delete operations.
- Format euro amounts and dates for the application's locale without changing the stored numeric values.
- Do not add tests, README files, or unrelated project files unless requested. Do not claim that code compiles or was tested unless that was actually verified.

## Reference

Carbon's guidance for prompts and code generation: <https://carbondesignsystem.com/developing/carbon-mcp/prompts/>. It emphasizes precise UI requirements, consulting Carbon MCP before coding when available, Carbon fidelity, accessibility, and avoiding invented APIs or ad-hoc styling.
