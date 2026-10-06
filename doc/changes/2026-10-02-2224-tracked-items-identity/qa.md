# Interface QA: tracked item identity

Status: executed; all eight final browser cases passed, no application findings.
Tester: independent `documentation_qa` invocation acting as QA tester.
Date: 2026-10-03, Europe/Paris (session date).
Revision: working-tree implementation over HEAD `097002199d70d428a139856fb53614796fc3a0be`, after [review](review.md) and [adjudication](decisions.md).
Environment: actual Vite application at `http://127.0.0.1:4173/`; headless Chrome 149.0.7827.53, Playwright Core, desktop 1440×1000 and narrow 390×844.

## Setup and isolation

Executed `node /tmp/pricefollower-ui-qa/identity-qa.cjs`. Chrome initially failed under sandbox socket restrictions; authorized tool escalation allowed execution. Browser route interception returned API-shaped fixture responses using integer `amountCents`, nullable fields, five collection statuses, and available/pending/not-found/check-error second-hand offers. All API requests were intercepted; nonlocal external requests were blocked. Deletion changed in-memory fixtures only. No shared database, collector or marketplace was contacted.

Fixtures used Amazon external IDs `B000000000` through `B000000004` except row 1, which used LeBoncoin ID `1234567890`. Row 1 had null title and image; row 2 had a 378-character repeated title. Row 0 had an inline image. Actual fixture source URL `https://www.amazon.fr/dp/B000000000` was supplied to the source-listing control; a replaced `window.open` captured the exact attempted URL with `_blank` and `noopener,noreferrer`, without external navigation. LeBoncoin fixture URL `https://www.leboncoin.fr/ad/collection/1234567890` supplied fixture identity/data; its source-navigation control was not separately executed. These are controlled source values, not verified live listings or submitted listing inputs.

## Executed case and real-URL catalog

| Case | Requirements / corner case | Actual interface URL | Exact actions and expected outcome | Observed |
| --- | --- | --- | --- | --- |
| QA-001 | FR-001,002,003,005; mixed platforms, missing title/image, prices/offers/status | `http://127.0.0.1:4173/` | Load five fixtures; count headers/rows; inspect each identity, fallback/image counts, price/offer text | Passed: five headers, five rows, no Marketplace header; all full identities `marketplace · listingId`; “Title unavailable”, four placeholders and one image; item €123.45 and offer €45.67 retained. All five collection states rendered without browser errors. |
| QA-002 | FR-004; all search fields and no-match span | `http://127.0.0.1:4173/` | Enter `leboncoin.fr`, `1234567890`, `Fixture 0`, `B000000000`, `amazon`, `no-such-fixture`, then clear | Passed: expected row counts 1,1,1,1,4; no-match message uses a cell with `colspan=5`; clearing restores rows. ASIN and external ID coincide for Amazon fixture, so their independent distinction is established by code review rather than this fixture. |
| QA-003 | FR-003,006; keyboard navigation and source access | `http://127.0.0.1:4173/` → `http://127.0.0.1:4173/items/qa-0` → `/` | Focus Fixture 0 title, confirm active element, press Enter, inspect details heading, return via breadcrumb, activate source control | Passed: keyboard opens correct details; return works; exact Amazon source URL attempt captured and blocked. |
| QA-004 | FR-006; long title and narrow-screen geometry/scroll | `http://127.0.0.1:4173/` | Resize to 390×844; compare each title bottom against identity top and metadata right against cell right; scroll actual Carbon content wrapper horizontally | Passed: all identities begin at title bottom with no overlap and remain within cells. Carbon content wrapper width 326px, scroll width 1056px; horizontal scrolling executed. Full identity text remains present/readable. |
| QA-005 | FR-005; empty/loading/error/retry | `http://127.0.0.1:4173/` | Reload with empty fixtures, failed list response, normal response after Retry, then 900ms delayed list response | Passed: empty heading and disabled refresh, failure notification, retry returns table, delayed request displays loading text before table. |
| QA-006 | FR-005; refresh progress and polling completion | `http://127.0.0.1:4173/` | Click Refresh prices; fixture POST queues five pending items; inspect progress/disabled button; replace fixtures with active states and await poll | Passed: “Refreshing prices for 5 items…” shown, refresh disabled during progress; disappears after completed-state poll. |
| QA-007 | FR-003; destructive operation failure/retry | `http://127.0.0.1:4173/` | Open first row overflow menu; Delete; return simulated deletion error; retry with successful intercepted DELETE | Passed: failure notification retains dialog; successful retry closes dialog and reduces rows from five to four; fixture-only mutation. |
| QA-008 | FR-003,004,006; seeded exploratory search/navigation repetition | `http://127.0.0.1:4173/`, `/items/qa-1`, `/items/qa-2` | Execute 15 generated searches; when matched open first title and return; clear search afterward | Passed: recorded navigation completes without browser exceptions. Exact sequence below. |

FR IDs above abbreviate `FR-TI-IDENTITY-001` through `006` from the [functional specification](../../specifications/tracked-items-identity/functional.md).

## Exploratory reproducibility

Seed `20261003`, unsigned 32-bit LCG `seed = (1664525 * seed + 1013904223) mod 2^32`, query selected by `seed % 5` from `[amazon, leboncoin.fr, 1234567890, unmatched, empty string]`. Run after fixture deletion of qa-0. Matched first-row title navigation always followed by the Tracked items breadcrumb return.

```text
seeded 0: search "leboncoin.fr"
seeded 0: details http://127.0.0.1:4173/items/qa-1
seeded 1: search "unmatched"
seeded 2: search "leboncoin.fr"
seeded 2: details http://127.0.0.1:4173/items/qa-1
seeded 3: search "unmatched"
seeded 4: search "amazon"
seeded 4: details http://127.0.0.1:4173/items/qa-2
seeded 5: search "unmatched"
seeded 6: search "unmatched"
seeded 7: search "1234567890"
seeded 7: details http://127.0.0.1:4173/items/qa-1
seeded 8: search "1234567890"
seeded 8: details http://127.0.0.1:4173/items/qa-1
seeded 9: search "unmatched"
seeded 10: search "unmatched"
seeded 11: search "leboncoin.fr"
seeded 11: details http://127.0.0.1:4173/items/qa-1
seeded 12: search "amazon"
seeded 12: details http://127.0.0.1:4173/items/qa-2
seeded 13: search "unmatched"
seeded 14: search "leboncoin.fr"
seeded 14: details http://127.0.0.1:4173/items/qa-1
```

## Evidence, corrections and limits

Temporary execution artifacts: `/tmp/pricefollower-ui-qa/identity-qa.cjs`, `identity-results.json` (fixtures, requests, actions and numeric geometry), `identity-narrow.png` and `identity-desktop.png`. Zero page errors recorded. Temporary files are local evidence and may be removed later; this report retains actual URLs, inputs, actions and outcomes.

The first run produced two harness failures: an assertion incorrectly inspected the outer table container instead of Carbon's scrolling content wrapper; the overflow menu selector incorrectly assumed a button role. Correcting only the harness and rerunning all eight cases passed. Application code was unchanged; these are not product findings.

This is focused frontend QA with API fixtures, not live backend/collector integration. No real external listing availability, add-item validation, duplicate creation, independent LeBoncoin source activation, deep-link reload, full accessibility audit or server refresh-failure scenario was executed. Focus/Enter navigation was executed; detailed visual focus styling was not separately asserted. Those limits do not block this column-consolidation scope. No test data cleanup was required: browser closed and fixtures existed only in its isolated route handler.

## Handoff

No new QA findings or unresolved feature blockers. All required identity, table alignment, filtering, narrow-screen and preserved-flow checks passed. Coordinator/adjudicator may record final readiness based on this report.
