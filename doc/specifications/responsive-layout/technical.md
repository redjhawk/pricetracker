# Technical specification: responsive layout

Status: ready
Functional specification: [functional.md](functional.md) (FR-RESPONSIVE-001 to 013, status ready)

## Current overflow sources (inspected)

- `src/index.css` `@media (max-width: 960px)` forces `.tracked-table-container table { min-width: 66rem }` with `overflow-x: auto`, so the tracked-items table scrolls sideways (FR-RESPONSIVE-002/004 forbid sideways scrolling of any area).
- `.item-cell { min-width: 14rem }`, `.item-title.cds--btn { max-width: 22rem }`, `.latest-price` and `.price-history-list li` use `white-space: nowrap`.
- Long titles, listing IDs, URLs, usernames, purchase goals, AI review text and notification messages have no `overflow-wrap` (only `.information-list dd` has it).
- Admin page uses a Carbon `Table` (Username, Last login) inside `TableContainer` with a `TableToolbar`; long usernames can widen it.
- Item details: `.detail-heading` and `.purchase-goal-actions` flex rows without wrap; Carbon `StructuredList` histories (`.detections-list`) have intrinsic table layout; detail `Tag`s with long listing IDs.
- Existing breakpoints are 960px/640px, not Carbon breakpoints.

## Requirement mapping

| Functional ID | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- |
| FR-RESPONSIVE-001 | TS-RESPONSIVE-001 global guards + all rules below | `src/index.css` | Playwright `scrollWidth <= clientWidth` at 320/360 on every screen and modal |
| FR-RESPONSIVE-002, 010 | TS-RESPONSIVE-002 wrapping of long content | `src/index.css` | Long-title/URL fixture at 320 px |
| FR-RESPONSIVE-003, 012 | TS-RESPONSIVE-008 accessibility and targets | `src/index.css`, `src/components/TrackedItemsPage.tsx` | Visibility/click assertions; manual screen-reader/Tab QA |
| FR-RESPONSIVE-004 | TS-RESPONSIVE-003 stacked tracked-items cards | `src/index.css`, `src/components/TrackedItemsPage.tsx` | Labels "Prices", "Amazon second-hand offer", "Status" visible per item at 360 px; title opens details |
| FR-RESPONSIVE-005 | TS-RESPONSIVE-004 heading, tabs, search wrap | `src/index.css` | Tabs, search, Add item, refresh all in viewport at 320 px; search filters |
| FR-RESPONSIVE-006 | TS-RESPONSIVE-005 details page | `src/index.css` | Details at 320/360 with histories, purchase goal, AI review |
| FR-RESPONSIVE-007 | TS-RESPONSIVE-006 admin users table | `src/index.css`, `src/components/AdminPage.tsx` | Admin at 320 px, long username |
| FR-RESPONSIVE-008, 009 | TS-RESPONSIVE-007 header, menu, login, modals | `src/index.css` | Each modal/menu/login at 320 px |
| FR-RESPONSIVE-011 | TS-RESPONSIVE-009 all narrow rules inside `max-width` media queries | `src/index.css` | Existing desktop specs unchanged and passing; desktop screenshots compared in QA |
| FR-RESPONSIVE-013 | TS-RESPONSIVE-010 CSS-only adaptation, no remount | — | Resize 360→800→360 keeps tab, search text, modal input |

## Frontend

Breakpoints follow Carbon: `md` = 672px (`max-width: 671.98px` for phone layout), `lg` = 1056px (desktop unchanged at and above 1056px). The existing 960px and 640px queries are retargeted to `max-width: 1055.98px` and `max-width: 671.98px` respectively (inside the non-desktop range, so FR-RESPONSIVE-011 holds). No JavaScript viewport detection, no new components, no new dependencies.

- **TS-RESPONSIVE-001 Global guards.** Keep `min-width: 320px` on `html, body, #root`. Add `min-width: 0` to flex/grid children that hold text (`.page-heading > div`, `.detail-heading-copy`, `.item-identity`, `.summary-tile`). Images get `max-width: 100%`. No `overflow-x: hidden` on `body` (it would hide content instead of fitting it, violating FR-RESPONSIVE-002).
- **TS-RESPONSIVE-002 Long content.** Add `overflow-wrap: anywhere` to: `.page-heading h1`, `.detail-heading h1`, `.item-title.cds--btn`, `.item-asin`, `.detail-tags .cds--tag` (and allow tags `max-width: 100%`, `white-space: normal`), `.summary-footnote`, `.purchase-goal`, `.ai-review`, `.page-description`, `.empty-state`, `.detail-notification`, `.modal-notification`, `.delete-copy`, `.modal-intro`, admin username cells, `.cds--modal-content`. Within `max-width: 671.98px` remove `white-space: nowrap` from `.latest-price` and `.price-history-list li` (allow `flex-wrap: wrap`) and remove `.item-title` `max-width: 22rem` in favour of `max-width: 100%`.
- **TS-RESPONSIVE-003 Tracked-items stacked layout.** Remove the `min-width: 66rem` / `overflow-x: auto` rule. At `max-width: 671.98px`, the Carbon `Table` keeps its semantic markup but is displayed as cards: `table, tbody, tr, td { display: block; width: 100% }`; `thead` is visually hidden with the standard visually-hidden pattern (clip, 1px), not `display:none`, so table semantics remain; each `tr` gets padding and a `border-block-end: 1px solid var(--cds-border-subtle)`; each `td` gets `height: auto`, padding `var(--cds-spacing-03) var(--cds-spacing-05)`. Labels: in `TrackedItemsPage.tsx` add `data-label="Prices"`, `data-label="Amazon second-hand offer"`, `data-label="Status"` to the matching `TableCell`s (Item and actions cells get no label; the title identifies the item and actions have accessible names). CSS `td[data-label]::before { content: attr(data-label); display: block; color: var(--cds-text-secondary); font-size: 0.75rem; font-weight: 600; margin-block-end: var(--cds-spacing-02) }`. `.item-cell` `min-width: 0`. `.row-actions` `justify-content: flex-start` on phone. The "No items match your search." row's `colSpan` cell displays as a block normally. Between 672px and 1055px the table stays a table; it must fit because cells wrap (TS-RESPONSIVE-002, `.item-cell` min-width reduced to `0`); if QA finds overflow there, apply the stacked layout up to 1055.98px instead (developer decision, recorded in review).
- **TS-RESPONSIVE-004 Heading, tabs, search.** `.page-heading` already stacks at phone width; `.page-heading-actions` keeps `flex-wrap: wrap`, buttons may take `max-width: 100%`. Carbon `Tabs` (two tabs) must fit at 320px: set `.cds--tabs .cds--tab--list { flex-wrap: wrap }` or allow the tabs to shrink (`.cds--tabs__nav-item { min-width: 0; flex: 1 1 auto }`) within the phone query; tab labels stay full text. Search: `.tracked-table-search .tracked-search { flex-basis: 100%; width: 100% }`. Amazon-stop warning and empty states use `.detail-notification`/`.empty-state` wrapping; `.empty-state` padding drops to `var(--cds-spacing-05)` on phone.
- **TS-RESPONSIVE-005 Item details.** `.detail-heading` stacks (existing rule, moved to 671.98px); `.detail-thumbnail` stays 5rem. `.detail-actions` wraps (existing). `.detail-summary` one column (existing). StructuredList histories: in the phone query set `.detections-list.cds--structured-list { display: block; }`, rows `display: flex; flex-wrap: wrap; gap`, cells `display: block; padding: var(--cds-spacing-03)`; header row stays visible as column captions, or if it cannot align, each cell keeps its order date / price / condition (values remain identifiable by position and the visible header). `.purchase-goal-actions { flex-wrap: wrap }`. AI review: `.ai-review > .history-heading` already wraps; `.ai-review-details dd { margin-inline-start: 0 }` on phone; Accordion content gets `padding-inline-end: var(--cds-spacing-05)` overriding Carbon's 25% right padding at phone width. Source link uses `overflow-wrap: anywhere`.
- **TS-RESPONSIVE-006 Admin.** Users table stacks like TS-RESPONSIVE-003 with `data-label="Username"` and `data-label="Last login"` on the two `TableCell`s in `AdminPage.tsx`, sharing CSS through a common class `responsive-table` added to both `Table`s (tracked items and users) via `className`. `TableToolbar` content wraps (`.cds--toolbar-content { flex-wrap: wrap; }` scoped to `.responsive-table-container` if needed) so Add user stays visible.
- **TS-RESPONSIVE-007 Header, menu, login, modals.** Carbon `Header` is fixed 3rem height: on phone, `.brand-mark` may shrink with `min-width: 0` and text `overflow-wrap: anywhere`/ellipsis must not hide text — use smaller `font-size` (0.875rem) so "Price follower" plus menu icon and Log in link fit at 320px. The `AppMenu` OverflowMenu is Carbon-positioned with `flipped`; verify it stays on screen. Login: `.login-page` padding drops to `var(--cds-spacing-05)` horizontally (already `min(100%, 24rem)` form). Modals: Carbon `ComposedModal`/`Modal` is full-width below md by default; add `.cds--modal-content { overflow-wrap: anywhere }` and ensure footers with two buttons stay in viewport (Carbon default). No modal markup changes.
- **TS-RESPONSIVE-008 Accessibility.** Table semantics preserved (no `role` overrides beyond what `display:block` causes; acceptable as table headers remain in DOM and each cell has a visible `::before` label; additionally the visible label text equals the column header). DOM order unchanged, so reading and Tab order follow visual order. Carbon sizes for buttons/tabs untouched; focus outlines untouched. No `user-scalable=no` in `index.html`.
- **TS-RESPONSIVE-009 Desktop unchanged.** Every new rule except pure wrapping (`overflow-wrap: anywhere`, `min-width: 0`, which do not alter desktop rendering of normal data) lives inside `max-width` queries below 1056px.
- **TS-RESPONSIVE-010 Resize.** Layout is CSS-only; React state (tab, search, modal inputs) is untouched on resize.

Loading/error/empty states reuse the same classes and are covered by TS-RESPONSIVE-002.

### Additional selectors found during development (REV-004)

All three live inside the phone query (`max-width: 671.98px`), and the new Playwright spec found each one at 320 px:

- `.detail-notification.cds--inline-notification, .modal-notification.cds--inline-notification { min-inline-size: 0 }`: Carbon's InlineNotification has an 18rem minimum width. The Amazon-stop warning overflowed 320 px.
- `.cds--header__name { min-width: 0; padding-inline: spacing-05 spacing-03 }` and `.brand-mark { line-height: 1.1; white-space: normal }`: with Log in, Add item and the menu, the header was wider than 320 px. It pushed the Application menu trigger and its flipped menu off screen (FR-RESPONSIVE-009).
- `.detail-breadcrumb .cds--breadcrumb-item { min-width: 0; max-width: 100% }` and `.detail-breadcrumb .cds--link { overflow-wrap: anywhere; white-space: normal }`: the current breadcrumb item repeats the item title, and the 200-character title overflowed (FR-RESPONSIVE-010).

Intermediate widths (REV-001, QA-F001): the page itself did not scroll at 700 px and 1000 px. QA then found that between 672 px and about 1005 px the tracked-items table overflowed inside Carbon's `.cds--data-table-content` scroller, which cut off Status and the row actions. **Decision: the TS-RESPONSIVE-003 fallback is applied.** The stacked-table rules (`.responsive-table` blocks, visually hidden `thead`, `td[data-label]::before`, `.row-actions` start-aligned) now live in `@media (max-width: 1055.98px)` for both tables (tracked items and admin users). The other phone rules stay at 671.98px. Desktop (≥1056 px) is unchanged. Use case: [UC-TI-05](../../use-cases/tracked-items/UC-TI-05-review-tracked-items-at-tablet-width.md).

## Backend

No change. The issue is purely presentational; no handler, service, persistence, or validation behavior is involved.

## API

No contract change. `API_SPECIFICATION.md` remains unchanged; no endpoint, field or status is added or modified. Recorded in the change index.

## Scope and refactoring

Permitted files: `src/index.css`, `src/components/TrackedItemsPage.tsx` (`data-label` attributes, `responsive-table` class), `src/components/AdminPage.tsx` (same), `tests/responsive-layout.spec.ts` (new), `playwright.config.ts` (add the spec to `testMatch`). Other components only if QA proves a CSS-only fix impossible; record the reason.

Refactoring: retargeting the existing 960px/640px queries to Carbon breakpoints 1055.98px/671.98px — needed to align with the Carbon grid and the spec's reference widths; low risk (affects only widths between 640–672 and 960–1056, both non-desktop); performed with the feature. No other refactoring.

Risks: Carbon internal class names (`cds--tab--list`, `cds--structured-list`, `cds--toolbar-content`) may change on upgrade; keep selectors scoped and few.

## Verification and unresolved questions

New Playwright spec `tests/responsive-layout.spec.ts`, added to `testMatch` in `playwright.config.ts`. Follow `tests/platform-tabs.spec.ts` and `tests/app-information.spec.ts`: mock all `/api/v1/*` via `context.route` (session `/api/v1/auth/session` open mode and admin protected mode, `/api/v1/items` list, item details, `/api/v1/admin/users`, settings endpoints), abort non-local origins. Fixtures include an Amazon item with second-hand offer, a LeBoncoin item with purchase goal and AI review, a 200-character title without spaces, a long listing URL, a long username.

Helper: `expectNoHorizontalScroll(page)` asserting `await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)` is true.

For viewports `{ width: 320, height: 640 }` and `{ width: 360, height: 740 }`:
1. List, Amazon tab and LeBoncoin tab (FR-001/004/005/010): no horizontal scroll; labels "Prices", "Status", "Amazon second-hand offer" visible; tabs, search, Add item, refresh-all `toBeInViewport()`; typing in search filters.
2. Empty list and Amazon-stop warning states (FR-005).
3. Item details for both items (FR-006): no horizontal scroll; history entries and AI review visible; refresh/delete/purchase-goal buttons in viewport.
4. Admin page with long username (FR-007).
5. Login page in protected unauthenticated mode (FR-009); header menu opened, entries in viewport.
6. Modals Add item, Delete, Settings, Information, Add user (FR-008): no horizontal scroll, primary/secondary buttons in viewport.
7. Resize 360→1056→360 with search text and open Add item modal input preserved (FR-013).
8. At 1056 px, the tracked table header row is visible (`thead` displayed as table header) (FR-011).

Build checks: `npm run build` (tsc + vite), `npx playwright test`. QA: manual screen-reader/Tab spot check at 360 px (FR-012) and desktop visual comparison (FR-011).

Unresolved questions: none.
