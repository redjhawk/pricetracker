# Platform tabs implementation

Status: ready for independent review. Date: 2026-10-03.
Role: distinct developer agent, using `.agents/roles/developer.md`, the staged workflow, and all four mandatory project skills.

## Scope and requirement mapping

Implemented the [ready functional specification](../../specifications/platform-tabs/functional.md) and [technical specification](../../specifications/platform-tabs/technical.md), preserving the [approved API](../../../API_SPECIFICATION.md) as recorded in [api-step.md](api-step.md). No backend changes or refactoring.

| Requirements | Implemented behavior | Verification |
| --- | --- | --- |
| FR-PLATFORM-TABS-001, 006 | App owns the Amazon-default platform; controlled Carbon tabs filter the existing collection in its original order and preserve selection across details and mutations. | Core rows/order, details/back, add/delete, reload tests. |
| FR-PLATFORM-TABS-002 | Amazon keeps five columns and all offer states; LeBoncoin has four columns with no offer header, cell, or placeholder. No-match colspan follows the selected schema. | Core column/cell counts; search colspan assertions; offer-state fixtures. |
| FR-PLATFORM-TABS-003 | Search remains local and shared across switches, trimming and folding case while retaining all previous matching fields. | Search cases and assertion that filtering sends no mutation or additional endpoint requests. |
| FR-PLATFORM-TABS-004, 007 | Stable tabs/panels surround loading, retrieval error/retry, global empty, platform empty, and no-match content in the specified order. Existing prices/statuses and offer states retain their rendering. | Empty, loading/error/retry, stale/pending/unavailable/error, free-price and missing-title fixtures. |
| FR-PLATFORM-TABS-005 | Count and refresh still use all items; button label is `Refresh all prices`. | Global POST/body assertion, two-item queued progress, polling and refresh-error test; refresh enabled on empty platform. |
| FR-PLATFORM-TABS-008 | Carbon tab semantics, keyboard activation, focus and associated panels; existing horizontal scrolling retained. | Arrow/Home/End, panel association, keyboard search/row access, focused outline and 390px viewport checks. |

Application changes: `src/App.tsx`, `src/components/TrackedItemsPage.tsx`. Existing styles were sufficient, so `src/index.css` was not edited. Only the selected panel mounts its content; both associated Carbon panels remain stable, and the shared search input therefore has no duplicate ID.

Verification support: exact `@playwright/test` version `1.63.0` in `package.json`/`package-lock.json`; `test:platform-tabs` npm script; `playwright.config.ts`; `tests/platform-tabs.spec.ts`; generated-result ignore entries in `.gitignore`. The focused suite contains eight cases and fixture routing specific to these flows. It starts Vite only on strict port 4173 without reusing existing servers. Every API request is intercepted in memory; unexpected API requests fail verification, and all external HTTP requests are aborted, including Carbon font requests. No Go process, persistent database, live marketplace request or real mutation is involved.

Documentation changes: the existing `README.md` describes browser setup and the command; `doc/use-cases/tracked-items/UC-TI-01-review-tracked-items.md` and `UC-TI-02-search-tracked-items.md` describe platform selection, column differences and search scope. This report is the developer's only workflow artifact edit.

## Actual verification and test-first evidence

Environment: Node 22 from `/tmp/pricefollower-node22/node_modules/node/bin`, local Chrome `/opt/google/chrome/chrome` selected through the optional `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH`, application URL `http://127.0.0.1:4173/`, fixture detail route `http://127.0.0.1:4173/items/leboncoin-one`. Default config remains portable and uses Playwright-managed Chromium unless that environment override is supplied.

Before any UI edits, ran:

```bash
PATH=/tmp/pricefollower-node22/node_modules/node/bin:$PATH PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/opt/google/chrome/chrome npm run test:platform-tabs -- --grep 'core:'
```

RED: one executed browser test failed in 6.4 seconds at `getByRole('tab', { name: 'Amazon', exact: true }).toBeVisible()`. Playwright reported `element(s) not found` after its 5000ms assertion timeout. Vite and Chrome had launched, and fixture rows loaded; the missing tab was the expected behavior failure, not an infrastructure failure. The coordinator received this evidence before UI implementation began.

The first installation attempt under the network sandbox failed with `EAI_AGAIN`; the approved scoped escalation installed the dependency successfully (four packages added, zero vulnerabilities reported). Browser runs used approved local-server/browser escalation.

During GREEN verification, the core UI assertions passed but an initial external-request assertion incorrectly treated blocked Carbon fonts as a failure; the test guard now aborts all external requests and specifically fails unexpected API calls. The first full run passed six cases and exposed test selector/tab-order assumptions. Carbon's existing overflow trigger is named `Options`, its tab panel adds no separate Tab stop, and table scrolling is provided by its inner `cds--data-table-content`. Corrected those test expectations using installed component implementation and actual browser output; no application behavior change was needed for them.

Final checks:

- `PATH=/tmp/pricefollower-node22/node_modules/node/bin:$PATH PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/opt/google/chrome/chrome npm run test:platform-tabs`: GREEN, all eight Chromium cases passed in 20.7 seconds.
- `PATH=/tmp/pricefollower-node22/node_modules/node/bin:$PATH npm run build`: passed; TypeScript build and Vite production bundle, 947 modules, Vite build 4.91s.
- `PATH=/tmp/pricefollower-node22/node_modules/node/bin:$PATH node node_modules/typescript/bin/tsc --noEmit --target ES2022 --module NodeNext --moduleResolution NodeNext --skipLibCheck tests/platform-tabs.spec.ts playwright.config.ts`: passed, covering verification TypeScript outside the app build.
- `git diff --check`: passed.

## Limitations and handoff

Independent review, adjudication, final exploratory QA and commits belong to subsequent roles and are not claimed here. Browser tests use fixtures and blocked font downloads, so no live collection/backend behavior or external listing availability was verified. No commit was created by this implementation agent.
