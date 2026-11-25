# Technical specification: responsive layout

Status: ready
Functional specification: [functional.md](functional.md) (FR-RESPONSIVE-001 to 030, status ready)

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

### Full-width tracked-items list on phones (FR-RESPONSIVE-014 to 017)

| Functional ID | Technical solution | Files | Verification |
| --- | --- | --- | --- |
| FR-RESPONSIVE-014 | TS-RESPONSIVE-011 | `src/index.css` | Playwright: at 320/360 px the `.tracked-table-container` box has left ≈ 0 and right ≈ viewport width, both tabs |
| FR-RESPONSIVE-015 | TS-RESPONSIVE-012 | — (existing cell padding) | Playwright: leftmost cell text at x ≥ 4 |
| FR-RESPONSIVE-016 | TS-RESPONSIVE-013 | `src/index.css` | Playwright: `expectNoHorizontalScroll` at 320/360 px; existing phone tests unchanged |
| FR-RESPONSIVE-017 | TS-RESPONSIVE-014 | `src/index.css` | Playwright: at 700 px the container's left/right edges equal those of the `.tracked-table-search` row (the tab panel content box, REV-002); existing desktop/tablet tests |

- **TS-RESPONSIVE-011 Edge-to-edge list.** The lateral gap at phone width is the sum of three sources: Carbon's `.cds--content` padding (`2rem` each side; the app only overrides its block-start padding in `.app-content`), the `.page-shell` inset (`width: min(100% - 2 * spacing-05, 96rem)`, centered, so `spacing-05` = 1rem each side), and the Carbon tab panel `.cds--tab-content` padding (`var(--cds-layout-density-padding-inline-local)`, 1rem at normal density): 4rem = 64 px per side (REV-001). Inside `@media (max-width: 671.98px)`, `.tracked-table-container { margin-inline: calc(-1 * (2rem + var(--cds-spacing-05) + var(--cds-layout-density-padding-inline-local, 1rem))) }`; the token is inherited from the ancestor `.cds--tab-content`, so a density change follows automatically. Use case: [UC-TI-06](../../use-cases/tracked-items/UC-TI-06-review-tracked-items-full-width-on-phone.md). The class is used only by the tracked-items `TableContainer` in `TrackedItemsPage.tsx`; the admin table (`.responsive-table-container`) is not affected. As a block with `width: auto`, negative margins widen it to exactly the viewport.
- **TS-RESPONSIVE-012 Inner padding.** Kept from TS-RESPONSIVE-003: each card `td` has `padding-inline: var(--cds-spacing-05)` (16 px), so text and actions stay inset.
- **TS-RESPONSIVE-013 No horizontal scroll; other elements unchanged.** The container's border box equals the viewport width, so the document does not widen. The heading, tabs, Amazon-stop warning, empty states and the search field (`.tracked-table-search`) keep their current page margins: the search is a form field, not a card, and FR-RESPONSIVE-016 keeps surrounding content as before; it stays aligned with the heading and tabs.
- **TS-RESPONSIVE-014 Scope.** Single rule in the phone query; nothing changes at ≥ 672 px or on other screens. Risk: if `.cds--content` padding, the `.page-shell` inset, or the `.cds--tab-content` padding / `--cds-layout-density-padding-inline-local` token change, the calc must follow (noted in the CSS comment).

Backend and API: not affected; `API_SPECIFICATION.md` unchanged.

Issue #165 replaces TS-RESPONSIVE-011's calc and supersedes TS-RESPONSIVE-013 (search and surrounding content keep their margins) and TS-RESPONSIVE-014 (nothing changes at ≥ 672 px). TS-RESPONSIVE-012 still applies. See the next section.

### Full width below desktop (issue #165, FR-RESPONSIVE-014 to 024)

Change folder: [issue 165](../../changes/2026-11-19-2053-issue-165-use-all-space-to-show-item-list-when-the/issue.md). Use cases: UC-TI-05, UC-TI-06, UC-TI-07, UC-RL-01, UC-RL-02.

#### Current gaps (inspected)

- Below 1056 px the page content is inset by 48 px per side: Carbon's `.cds--content` has `padding: 2rem` (the app overrides only the block-start padding in `.app-content`), and the `.page-shell` inset is `spacing-05`. On the tracked items page, the Carbon tab panel `.cds--tab-content` adds 16 px more, so the cards there are inset by 64 px per side. Only the phone query cancels this gap, and only for `.tracked-table-container`, with a calc.
- The stacked cards (`max-width: 1055.98px`) set every `td` to `display: block; width: 100%`, so each value takes a whole line and the right part of the card stays empty.
- From 672 to 1055 px, the search field keeps its 24 rem width. The tabs keep their content width. Carbon's `.cds--inline-notification` has a breakpoint `max-inline-size` (608 px from md, 736 px from lg). The login form is capped at `min(100%, 24rem)`. So the space to the right of these elements stays empty.
- The admin `.responsive-table-container` (toolbar with Add user, then the users table) keeps the page margins at every width.

#### Requirement mapping

| Functional ID | Technical solution | Files | Verification |
| --- | --- | --- | --- |
| FR-RESPONSIVE-014, 018 | TS-RESPONSIVE-015, TS-RESPONSIVE-016 | `src/index.css` | TC-001 |
| FR-RESPONSIVE-015, 020 | TS-RESPONSIVE-016 (kept TS-RESPONSIVE-012), TS-RESPONSIVE-017 | `src/index.css` | TC-002, TC-007, TC-008 |
| FR-RESPONSIVE-016 | TS-RESPONSIVE-015, TS-RESPONSIVE-018 | `src/index.css` | TC-004, TC-007 |
| FR-RESPONSIVE-017 | TS-RESPONSIVE-021 | `src/index.css` | TC-003, TC-012, TC-013 |
| FR-RESPONSIVE-019 | TS-RESPONSIVE-016, TS-RESPONSIVE-021 | `src/index.css` | TC-003, TC-012 |
| FR-RESPONSIVE-021 | TS-RESPONSIVE-016, TS-RESPONSIVE-017 | `src/index.css` | TC-004 |
| FR-RESPONSIVE-022 | TS-RESPONSIVE-015, TS-RESPONSIVE-018, TS-RESPONSIVE-019, TS-RESPONSIVE-020 | `src/index.css` | TC-004, TC-007 to TC-011 |
| FR-RESPONSIVE-023 | TS-RESPONSIVE-016 | `src/index.css` | TC-010 |
| FR-RESPONSIVE-024 | TS-RESPONSIVE-017 | `src/index.css` | TC-005, TC-006, TC-010 |
| Kept FR-RESPONSIVE-001 to 013, FR-APP-INFO-013 | TS-RESPONSIVE-021, TS-RESPONSIVE-022 | — | TC-001 to TC-013 |

#### Frontend

CSS only, in `src/index.css`. There are no component, markup, state or dependency changes. Every new or moved rule is inside `@media (max-width: 1055.98px)`, the query that already switches the tables to cards, unless stated otherwise. Spacing uses Carbon tokens. The page gutter is `var(--cds-spacing-05)` (16 px). This is both the maximum gap that FR-RESPONSIVE-022 allows and the card inner padding.

- **TS-RESPONSIVE-015 Page gutter.** Add `.app-content { padding-inline: 0 }`. This applies to the tracker and admin apps, and removes Carbon's 2 rem. Keep the existing `.page-shell` width `min(100% - 2 * var(--cds-spacing-05), 96rem)`, so page content sits 16 px from each edge. Add `.tracked-page .cds--tab-content { padding-inline: 0 }`. Its block padding is kept, and it is scoped to the tracked page, the only `Tabs` user. Result: the heading, notifications, tab bar, search, empty and loading states, and item details content all sit at the 16 px gutter.
- **TS-RESPONSIVE-016 Edge-to-edge card lists.** Add `.tracked-table-container, .responsive-table-container { margin-inline: calc(-1 * var(--cds-spacing-05)) }`. As blocks with `width: auto`, they widen by exactly the gutter to span the viewport.
  - Delete the phone-only calc rule (TS-RESPONSIVE-011) and its comment. Replace it with a comment that names the gutter it cancels.
  - The admin container also holds the `TableToolbar`. Add `.responsive-table-container .cds--table-toolbar { padding-inline: var(--cds-spacing-05) }` so Add user stays 16 px from the edge.
  - The admin loading and empty `.empty-state` boxes inside the container become edge-to-edge bordered boxes. Their own padding keeps their text inset.
  - Card content keeps `padding-inline: var(--cds-spacing-05)` on each `td` (TS-RESPONSIVE-012). This includes the `colSpan` cell holding “No items match your search.” (FR-RESPONSIVE-021).
  - The cards and the full width share one media query, so they start and stop together at 1056 px (FR-RESPONSIVE-019).
- **TS-RESPONSIVE-017 Values side by side in a card.** In the existing stacked-table block:
  - `.responsive-table tr` changes from `display: block` to `display: flex; flex-wrap: wrap; align-items: flex-start`. `width: 100%`, `height: auto`, the padding and the bottom border are kept.
  - `.responsive-table tbody tr td` gets `width: auto; flex: 0 1 auto; min-width: 0; max-width: 100%`, replacing `width: 100%`. `display: block` and the padding are kept.
  - `table` and `tbody` stay `display: block; width: 100%`. The visually hidden `thead`, the `td[data-label]::before` labels and `.row-actions { justify-content: flex-start }` are unchanged.

  Flex line breaking puts each cell on the current line when its max-content width fits, and otherwise starts a new line. This is the rule of FR-RESPONSIVE-024. Value cells do not grow; only the item cell grows into the space left on its line, and the actions cell is pushed to the line end (TS-RESPONSIVE-023).

  A cell wider than the card, such as a long title or an 80-character username, takes its own line, is clamped to the card width, and wraps (`overflow-wrap: anywhere` on `.responsive-table td`, already present). No `order` or reversed direction is used, so visual order equals DOM order: Item, Prices, Amazon second-hand offer, Status, actions; or Username, Last login. Reading and Tab order follow it (FR-RESPONSIVE-012). Below 672 px the existing phone rules still remove `white-space: nowrap` from `.latest-price` and `.price-history-list li`. From 672 to 1055 px they keep `nowrap`, which keeps each value's max-content width compact.
- **TS-RESPONSIVE-018 Full-width blocks above and around the list.**
  - **Search field.** Move `.tracked-table-search .tracked-search { flex-basis: 100%; width: 100% }` from the two phone rules to the 1055.98 px query.
  - **Tab bar.** Move the tab rule from the phone query to the 1055.98 px query, scoped as `.tracked-page .cds--tabs .cds--tabs__nav-item { min-width: 0; flex: 1 1 auto }`. Add `.tracked-page .cds--tabs .cds--tab--list { inline-size: 100% }`, because Carbon sets it to `auto`. The two tabs and their bottom borders then span the content width.
  - **Notifications.** Add `.page-feedback .cds--inline-notification, .login-form .cds--inline-notification { min-inline-size: 0; max-inline-size: none }`. This covers the list error, the admin list error, the item-details error, the session error and the login error, and lifts Carbon's breakpoint cap. `.detail-notification` (Amazon-stop warning, refresh errors, admin success, purchase-goal messages) already has `max-width: none`. `.page-feedback` and `.login-form` never contain a modal, so modal notifications are untouched.
  - **Heading actions.** In the phone query only, add `.page-heading-actions .cds--btn { flex: 1 1 auto }`. The stacked heading's button row then reaches the right gutter. From 672 to 1055 px the actions are already right-aligned at the gutter, and the title at the left gutter.
  - **Empty and loading states.** `.empty-state` is a block and already fills the content width.
- **TS-RESPONSIVE-019 Login page.** Add `.login-form, .login-form .cds--btn[type="submit"] { width: 100%; max-width: none }`. Carbon caps buttons at 20 rem, so this makes the Log in button span the form too. `.login-page` already has `padding-inline: var(--cds-spacing-05)` at every width, so at 775 px the fields, the button and the error notification span the viewport minus 16 px per side. The phone-only `.login-page` padding rule is now redundant but harmless. Leave it.
- **TS-RESPONSIVE-020 Item details.** No dedicated rule is expected. TS-RESPONSIVE-015 moves the breadcrumb, heading, `.detail-actions` (right-aligned from 672 px, left-aligned below), summary tiles, histories, purchase goal `TextArea`, AI review `Accordion` and notifications to the 16 px gutter. These Carbon blocks are full width. The loading state is a Carbon `InlineLoading`, whose `.cds--inline-loading` box is `inline-size: 100%`. TC-008 and TC-009 measure that box, and the left gap of its text. If TC-009 finds a block narrower than the content width at 775 or 1055 px, give it `inline-size: 100%` in the same query. That is a developer decision; record it in the review.
- **TS-RESPONSIVE-021 Desktop, header and modals unchanged.** At ≥ 1056 px no new rule applies, so the margins, table header rows and search alignment equal the baseline (FR-RESPONSIVE-011, 017).
  - The Carbon `Header` is outside `.app-content`, and none of its selectors change.
  - Modals are Carbon fixed-position containers. Ancestor padding does not affect them, including the admin Add user `Modal` rendered inside `.page-shell`. The new notification selectors never match them.
  - FR-APP-INFO-013 is kept.
- **TS-RESPONSIVE-022 Resize.** CSS only, with no remount. The tab, search text and modal inputs survive resizing across 1056 px (FR-RESPONSIVE-013).
- **TS-RESPONSIVE-023 Supporting rules found during development.** All are in the 1055.98 px query:
  - **Spacing scale.** No rule is needed. Master commit 248ae54 declares `--cds-spacing-*` on `:root`, so the tokens used by TS-RESPONSIVE-015 to 018 resolve everywhere, and modals keep their current look.
  - **Tab bar.** `.tracked-page .cds--tabs .cds--tabs__nav-item--close--hidden { display: none }`. Carbon's 1 px close-button placeholder after each tab otherwise stops the last tab 1 px short of the gutter.
  - **Card cells.** `.responsive-table tbody tr td:has(> .item-cell) { flex: 1 1 14rem }` lets the item cell take the space the values leave on its line, and `td:has(> .row-actions) { margin-inline-start: auto }` ends the line with the actions, so the flipped Options menu opens on screen. `.item-cell { min-width: 0 }` lets a long title wrap inside the card.
  - **Breadcrumb.** The `.detail-breadcrumb` wrapping rules (REV-004) move from the phone query to the base rules, outside any media query. A long title in the breadcrumb then wraps at every width, including desktop, where it otherwise widened the page (FR-RESPONSIVE-001, 010; TC-012). Short titles keep their current layout.
  - **Item details.** `.detail-summary` uses two columns from 672 to 1055 px; the phone query keeps one column.
  - **Page top.** `.page-shell` gets `padding-block-start: var(--cds-spacing-05)` instead of `spacing-07`, so the top gap matches the 16 px side gutter.

Accessibility: no markup change. The table semantics, visible `data-label` captions, focus styles and Carbon target sizes are as in TS-RESPONSIVE-008. The stretched tabs keep their full labels and the Carbon tab keyboard behaviour.

#### Backend

Not affected. The change is presentation only: no handler, service, validation, persistence, migration or concurrency change.

#### API

Not affected. No endpoint, field, status or error changes. `API_SPECIFICATION.md` is unchanged.

#### Files to change

- `src/index.css`: TS-RESPONSIVE-015 to 020 and 023. Remove the TS-RESPONSIVE-011 calc rule.
- `tests/responsive-layout.spec.ts`: the new cases below. Remove `700px tracked-items list keeps its page margins`, whose behaviour FR-RESPONSIVE-018 replaces.

`playwright.config.ts` already lists the spec. No `.tsx`, Go or API file changes.

#### Tests

All cases are e2e Playwright tests in `tests/responsive-layout.spec.ts`, using the existing `mockApi` and `expectNoHorizontalScroll`. `mockApi` gets options for:

- sample-like items: a short-title Amazon item with three prices, an available second-hand offer and success status; an Amazon item without an offer; a short-title LeBoncoin item;
- the users list (short or 80-character username);
- an items response held unanswered, or returned with HTTP 500;
- an item details response held, or returned with HTTP 500;
- a login API returning 401.

Shared helpers: `edgeGaps(locator)` (left and right gap of a box or text range to the viewport edges) and `expectEdgeToEdge(locator)` (|left| ≤ 1, |right − width| ≤ 1). Baseline constants (TC-003, TC-012, TC-013) are measured on `master` before `src/index.css` changes and recorded in the test. For TC-005, a value is the `td` border box.

| Test case | Test (in `tests/responsive-layout.spec.ts`) | TS IDs |
| --- | --- | --- |
| TC-001 | `TC-001 tracked-item cards run edge to edge` (widths 320 to 1055, both tabs) | 015, 016 |
| TC-002 | `TC-002 card content stays inset and readable` | 016, 017 |
| TC-003 | `TC-003 card layout and full width switch together at 1056px` | 016, 021, 022 |
| TC-004 | `TC-004 no-match message inside the full-width list` | 016, 018 |
| TC-005 | `TC-005 card values side by side` | 017 |
| TC-006 | `TC-006 phone card values in reading and focus order` | 017 |
| TC-007 | `TC-007 elements above the list use the full width` | 015, 018 |
| TC-008 | `TC-008 tracked items empty, loading and error states use the full width` | 015, 018 |
| TC-009 | `TC-009 item details use the full width` | 015, 018, 020 |
| TC-010 | `TC-010 administrator users list edge to edge` | 016, 017 |
| TC-011 | `TC-011 login page uses the full width` | 018, 019 |
| TC-012 | `TC-012 desktop layout unchanged` (1056, 1440) | 021 |
| TC-013 | `TC-013 header and modals unchanged below desktop` | 021 |

The existing 320 px and 360 px `tracked-items list spans the full width` tests and the issue #140 tests stay.

#### Refactoring

Moving the search-width and tab-stretch rules from the phone query to the 1055.98 px query, and replacing the TS-RESPONSIVE-011 calc with the gutter rule, are needed by FR-RESPONSIVE-018 and 022.
- **Scope:** `src/index.css` only.
- **Risk:** low. Rules apply only below 1056 px, and TC-012 guards the desktop layout.

No other refactoring.

#### Risks

- The design relies on Carbon internals: `.cds--content` padding, the `.cds--tab-content` padding, the `.cds--tab--list` width, the `.cds--table-toolbar` layout and the `.cds--inline-notification` breakpoint widths. A Carbon upgrade may change them. TC-001, TC-007 and TC-010 detect it.
- Stretched tabs and phone heading buttons are a visible change below 1056 px. FR-RESPONSIVE-022 requires them.

### Specification wording (issue #203, FR-RESPONSIVE-025 to 027)

Change folder: [issue 203](../../changes/2026-11-25-0828-issue-203-fix-020-updated-wording-in-responsive-la/issue.md). Maintenance only: there is no code change. The functional specification stage made both wording fixes. No use cases.

| Functional ID | Technical solution | Files | Verification |
| --- | --- | --- | --- |
| FR-RESPONSIVE-025 | TS-RESPONSIVE-024 | `doc/FUNCTIONAL_SPECIFICATIONS.md` | TC-001, TC-003 |
| FR-RESPONSIVE-026 | TS-RESPONSIVE-025 | `doc/specifications/responsive-layout/functional.md` | TC-002, TC-003 |
| FR-RESPONSIVE-027 | TS-RESPONSIVE-026 | none | TC-004, TC-005 |

- **TS-RESPONSIVE-024 Index row wording.** The issue #165 entry of the Responsive layout row of `doc/FUNCTIONAL_SPECIFICATIONS.md` reads `FR-RESPONSIVE-018 to 024 specified, 016/017 updated (issue #165)`. The row gets the issue #203 entry. Nothing else in that file changes.
- **TS-RESPONSIVE-025 Traceability wording.** Owner answer (4) of the issue #165 Traceability bullet in `functional.md` reads `→ FR-RESPONSIVE-022, 023, FR-RESPONSIVE-016/017 updated.`. Only `/020` is removed from that bullet. Only the FR-RESPONSIVE-025 and 026 definitions quote `016/017/020`, as their check.
- **TS-RESPONSIVE-026 No other change.** The text of FR-RESPONSIVE-001 to 024, the issue #140 and #165 change records, `doc/todo/` and every file outside `doc/` are unchanged. The existing Go and Playwright suites run unchanged and pass.

Frontend, backend and API: not affected. `API_SPECIFICATION.md` is unchanged.

Tests: no test code. FR-RESPONSIVE-027 allows no change outside `doc/` (see the change folder's `no-tests.md`). TC-001 to TC-004 are `grep` and `git diff` checks against `master`. TC-005 runs the existing gates: `go vet ./...`, `go test -count=1 ./...`, `node --test scripts/*.test.mjs`, `npm run build` and `npx playwright test`.

### TS-RESPONSIVE-017 wording (issue #200, FR-RESPONSIVE-028 to 030)

Change folder for issue #200: [issue 200](../../changes/2026-11-25-0944-issue-200-align-ts-responsive-017-wording-with-the/issue.md). Issue #200 is maintenance only, with no code change and no use cases (FR-RESPONSIVE-028 to 030).

| Functional ID | Technical solution | Files | Verification |
| --- | --- | --- | --- |
| FR-RESPONSIVE-028 | TS-RESPONSIVE-027 | `doc/specifications/responsive-layout/technical.md` | TC-001, TC-003 |
| FR-RESPONSIVE-029 | TS-RESPONSIVE-028 | `doc/specifications/responsive-layout/technical.md` | TC-002 |
| FR-RESPONSIVE-030 | TS-RESPONSIVE-029 | none | TC-003, TC-004, TC-005 |

- **TS-RESPONSIVE-027 Growth rules in TS-RESPONSIVE-017 (FR-RESPONSIVE-028).** In the TS-RESPONSIVE-017 entry, the developer replaces only the last sentence of the line-breaking paragraph (the one after the FR-RESPONSIVE-024 sentence, which says that no cell grows) with `Value cells do not grow; only the item cell grows into the space left on its line, and the actions cell is pushed to the line end (TS-RESPONSIVE-023).` The rest of that paragraph and of the entry stays as it is. This matches the shipped `src/index.css`, which does not change: `.responsive-table tbody tr td { flex: 0 1 auto }`, `td:has(> .item-cell) { flex: 1 1 14rem }` and `td:has(> .row-actions) { margin-inline-start: auto }`.
- **TS-RESPONSIVE-028 Old wording removed (FR-RESPONSIVE-029).** After TS-RESPONSIVE-027, `technical.md` contains neither of the two texts listed in FR-RESPONSIVE-029. This issue #200 section must not quote them either; the new sentence's lower-case "cells do not grow" does not match the case-sensitive check.
- **TS-RESPONSIVE-029 No other change (FR-RESPONSIVE-030).** For issue #200, `functional.md` (already updated by the functional stage), the TS-RESPONSIVE-023 entry, the issue #140, #165 and #203 change records, `doc/todo/` and every file outside `doc/` are unchanged. No ID is renumbered.

Issue #200 frontend, backend and API: not affected, as only `technical.md` text changes (FR-RESPONSIVE-030). `API_SPECIFICATION.md` is unchanged. No refactoring for issue #200.

Issue #200 tests: no test code, as FR-RESPONSIVE-030 allows no change outside `doc/`. TC-001 to TC-003 are `grep`, `awk` and `git diff` checks against `master`. TC-004 runs `go vet ./...` and `go test -count=1 ./...`. TC-005 runs the existing gates `node --test scripts/*.test.mjs`, `npm run build` and `npx playwright test`, including `tests/responsive-layout.spec.ts`.

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
