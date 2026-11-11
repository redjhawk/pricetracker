# Implementation: responsive layout (issue #140)

Sources: [functional](../../specifications/responsive-layout/functional.md), [technical](../../specifications/responsive-layout/technical.md). API unchanged.

## Requirement mapping

| Requirement | Change |
| --- | --- |
| FR-RESPONSIVE-001, 002, 010 (TS-001, 002) | `src/index.css`: `min-width: 0` on `.page-heading > div` and `.summary-tile`; `img { max-width: 100% }`; `overflow-wrap: anywhere` on headings, item title/ID, tags, footnotes, purchase goal, AI review, descriptions, empty states, notifications, modal copy, `.responsive-table td`, `.cds--modal-content`. Phone query: `.latest-price`/`.price-history-list li` wrap, `.item-title` `max-width: 100%`, detail tags wrap. No `overflow-x: hidden` on `body`. |
| FR-RESPONSIVE-004 (TS-003) | Removed the `66rem` table min-width and `overflow-x: auto`. Phone query (`max-width: 671.98px`): `.responsive-table` and its `tbody`/`tr`/`td` display as blocks; `thead` visually hidden (absolute, 1px, clipped; kept in DOM); `td[data-label]::before` shows the label. `TrackedItemsPage.tsx`: `className="responsive-table"` and `data-label` "Prices", "Amazon second-hand offer", "Status". `.item-cell` `min-width: 0` below 1056px; `.row-actions` start-aligned on phone. After QA-F001 the stacked layout applies up to 1055.98px (see below). |
| FR-RESPONSIVE-005 (TS-004) | Phone: search `flex-basis: 100%`; tabs `.cds--tabs__nav-item { min-width: 0; flex: 1 1 auto }`; heading buttons `max-width: 100%`; `.empty-state` padding `spacing-05`; `.detail-notification`/`.modal-notification` `min-inline-size: 0` (Carbon's 18rem minimum overflowed 320px, found by the new spec). |
| FR-RESPONSIVE-006 (TS-005) | Phone: `.detections-list` StructuredList as block/flex-wrapped rows; `.purchase-goal-actions` wraps; `.ai-review-details dd` no indent; AI review accordion content `padding-inline-end: spacing-05`; detail breadcrumb current item wraps (the 200-char title overflowed there, found by the new spec). |
| FR-RESPONSIVE-007 (TS-006) | `AdminPage.tsx`: `responsive-table` class, `data-label` "Username"/"Last login", `responsive-table-container` on `TableContainer`; toolbar content wraps on phone. |
| FR-RESPONSIVE-008, 009 (TS-007) | Phone: `.cds--header__name` `min-width: 0` and smaller inline padding, `.brand-mark` 0.875rem and may wrap (the header overflowed at 320px, pushing the menu off screen); login padding `spacing-05`; modal content wraps. No markup change. |
| FR-RESPONSIVE-011 (TS-009) | Narrow rules inside `max-width: 1055.98px` / `671.98px` (retargeted from 960/640px, recorded refactoring in the technical spec). Outside queries only wrap/`min-width: 0`/`img max-width` guards. |
| FR-RESPONSIVE-012, 013 (TS-008, 010) | CSS-only; DOM order and table semantics unchanged; no JS viewport detection. |

## Files

- `src/index.css`, `src/components/TrackedItemsPage.tsx`, `src/components/AdminPage.tsx`
- `tests/responsive-layout.spec.ts` (new, 14 tests), `playwright.config.ts` (`testMatch`)
- `tests/platform-tabs.spec.ts` (outside the permitted list, necessary): the test "Carbon keyboard navigation and narrow table scrolling preserve access" asserted the removed behavior (table container scrolls sideways at 390px). Renamed to "... the narrow stacked table preserve access" and the assertion inverted to `scrollWidth <= clientWidth` (FR-RESPONSIVE-004).

## Checks executed

- Chromium could not start initially (`libnspr4.so` missing); ran `npx playwright install-deps chromium`, after which it launched.
- `npm run build`: passed.
- `npx playwright test tests/responsive-layout.spec.ts`: 14 passed (after fixes for the notification min-width, header, breadcrumb, and test locators).
- `npx playwright test tests/platform-tabs.spec.ts -g "narrow stacked"`: 1 passed.
- Full `npx playwright test`: 63 passed, 8 failed before the platform-tabs update. Final run: 64 passed, 7 failed:
  - `app-information.spec.ts`: REV-002 (Application menu never appears), "failure shows an error..." (same), "Close button..." and "at 400 px..." (strict mode: `.cds--modal-close` / `.cds--modal-container` match 3 modals), "keyboard opens the entry..." (focus assertion).
  - `platform-tabs.spec.ts`: "both tabs remain available..." expects 3 tabs, the app renders 2.
  - `leboncoin-session-settings.spec.ts`: "header menu: visible on every page..." (`toHaveCount`), QA-SET-F01 (element not found).
  These failures involve locators, mock data and DOM counts, not layout. The coordinator confirmed they fail identically on unmodified `master` (stash run: the same 7 failed, 31 passed in those 3 files).

## Review fixes

- REV-001: added no-horizontal-scroll tests at 700 px and 1000 px (list on both tabs with the long fixtures, and the admin page with the long username). All 4 passed without changes, so the TS-RESPONSIVE-003 fallback (stacking up to 1055.98px) was not applied. No use case was added because no fix was needed.
- REV-004: the extra selectors (notification min-width, header name/brand, breadcrumb) and their reasons are listed in [technical.md](../../specifications/responsive-layout/technical.md) under "Additional selectors found during development".
- After the fixes: `npm run build` passed; `npx playwright test tests/responsive-layout.spec.ts` 18 passed.

## QA fix QA-F001 (REV-001 reopened, critical)

- Problem: between 672 px and about 1005 px the tracked-items table overflowed inside `.cds--data-table-content`, which hid Status and the row actions.
- Fix (`src/index.css`): the stacked-table rules moved from the 671.98px query to a `max-width: 1055.98px` query, which applies to both `.responsive-table` tables. Desktop (≥1056 px) is unchanged; `technical.md` records the decision.
- Regression test: `QA-F001 {700,800,1000}px tables do not scroll inside and keep Status and actions on screen`. At each width and on both tabs, it checks that `.cds--data-table-content` and `.tracked-table-container` have `scrollWidth <= clientWidth` and that the Status cells and `.row-actions` end within the viewport. The 800 px no-scroll tests were added as well.
- Fails without the fix: with the old 671.98px breakpoint, all 3 QA-F001 tests failed (`3 failed`).
- Passes with the fix: `npm run build` passed; `npx playwright test tests/responsive-layout.spec.ts tests/platform-tabs.spec.ts` gave 30 passed, 1 failed. That failure is the known pre-existing "both tabs remain available…", which also fails on `master`. The responsive spec on its own: 21/21 passed.
- Use case: [UC-TI-05](../../use-cases/tracked-items/UC-TI-05-review-tracked-items-at-tablet-width.md), linked from `doc/use-cases/README.md`.

## Limitations

- Manual screen-reader/Tab check (FR-RESPONSIVE-012) and desktop visual comparison (FR-RESPONSIVE-011) left to QA. The automated desktop check only covers the tracked-table header at 1056px.
- Row OverflowMenu at phone width opens below the row and may need vertical scrolling; the page has no horizontal scroll with it open.
