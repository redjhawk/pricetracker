# Review decisions: issue #140 responsive layout

Reviewed specification revisions: `doc/specifications/responsive-layout/functional.md` (ready), `technical.md` (ready), working tree
Reviewed code revision: uncommitted working tree on `ai-dev/issue-140-20261111-1142` (base `f6ed820`)
Reviewer: independent reviewer agent
Adjudicator: <different agent>

Scope reviewed: `src/index.css`, `src/components/TrackedItemsPage.tsx`, `src/components/AdminPage.tsx`, `playwright.config.ts`, `tests/platform-tabs.spec.ts`, `tests/responsive-layout.spec.ts`, `doc/FUNCTIONAL_SPECIFICATIONS.md`, spec and change files. No backend or API change, as specified. The `platform-tabs.spec.ts` edit only inverts the assertion that checked the removed sideways scrolling. It is necessary and recorded in `implementation.md`. No unrelated changes found.

## Findings and decisions

### REV-001: Intermediate widths (672–1055 px) are not verified, and the table can still overflow there

- Evidence: `src/index.css` stacks the table only at `max-width: 671.98px`. Between 672 and 1055 px the Amazon tab keeps a 5-column table. `.latest-price` and `.price-history-list li` keep `white-space: nowrap` outside the phone query, and the 66rem/`overflow-x: auto` guard was removed. `tests/responsive-layout.spec.ts` only tests 320, 360 and 1056 px. FR-RESPONSIVE-001 and the "States and corner cases" section require widths between 320 px and desktop to have no sideways scroll. TS-RESPONSIVE-003 asks for QA at this range, with a fallback (stack up to 1055.98px).
- Impact and scenario: on a tablet or a landscape phone (for example 700–800 px) with the Amazon tab, the non-wrapping price and offer columns, the row actions and the cell padding may be wider than the viewport. That would bring back sideways page scrolling. The table container no longer scrolls on its own, so overflow would now widen the page. This is unverified, not observed.
- Criticality (provisional): non-critical, pending QA measurement. It becomes a defect against FR-RESPONSIVE-001 if overflow is observed.
- Suggested action: add a check at about 700 px and 1000 px (Amazon tab, long title), or apply the documented fallback.
- Disposition / Reason / Resolution / Follow-up: for the adjudicator.

### REV-002: Stacked tables may lose table semantics in some screen readers; the Item and actions cells have no label

- Evidence: in the phone query, `.responsive-table`, `tr` and `td` get `display: block`. WebKit/Safari VoiceOver is known to drop table roles when table elements are restyled with `display: block`, so the visually hidden `thead` is then not announced. The visible `::before` labels (`data-label`) give the label for Prices, Amazon second-hand offer and Status (and the Username and Last login cells on the admin page). The Item and actions cells have no label, as TS-RESPONSIVE-003 specifies. In Chromium, where table semantics stay, each value is announced twice ("Prices" from the column header plus the `::before` text). FR-RESPONSIVE-012 asks that "screen readers announce each value with its label". The automated tests do not cover this; the manual check is left to QA.
- Impact and scenario: an iOS VoiceOver user on the list may hear values without column context, except where the `::before` text gives it. Chromium users hear labels twice. Both are minor.
- Criticality (provisional): non-critical. The labels stay present as generated content, which most screen readers read, and the design follows the approved technical specification.
- Suggested action: QA to do the manual screen-reader check required by FR-RESPONSIVE-012 and record the result. Optionally, restore the roles explicitly (`role="table"/"row"/"cell"` are already given by Carbon markup only implicitly).

### REV-003: Some wrapping rules outside the media queries also apply on desktop

- Evidence: `overflow-wrap: anywhere` on `.responsive-table td`, `.item-title.cds--btn`, `.cds--modal-content`, headings and others, plus a global `img { max-width: 100% }`, sit outside any media query. TS-RESPONSIVE-009 explicitly allows this. With `overflow-wrap: anywhere`, the min-content width of table cells changes, so at ≥1056 px a column with long unbroken text (for example an 80-character username) now wraps where it used to widen the table. Normal data renders the same.
- Impact and scenario: there is a desktop rendering difference only for pathological long unbroken strings, and it is an improvement (it removes overflow). FR-RESPONSIVE-011 says "unchanged". The only automated desktop check covers the `thead` display.
- Criticality (provisional): non-critical. Specified, and harmless for realistic data.
- Suggested action: QA desktop visual comparison as planned. No code change needed unless the comparison shows a difference with realistic data.

### REV-004: Small unspecified selectors added

- Evidence: `.detail-breadcrumb` wrapping, `.detail-notification`/`.modal-notification` `min-inline-size: 0` and `.cds--header__name` padding are not listed in the technical specification. `implementation.md` gives their reasons: spec failures at 320 px. They serve FR-RESPONSIVE-001/009/010, and no new behavior is added.
- Criticality (provisional): non-critical. Informational: the technical specification could list them for traceability.

## Verification notes

- The 7 remaining Playwright failures (app-information, leboncoin-session-settings, platform-tabs "both tabs remain available…") are stated by the coordinator to fail on unmodified `master` too. The reviewer did not reproduce them. They concern locators and mock counts, not layout.
- Resize (FR-013) and desktop `thead` (FR-011) are covered by tests. DOM order is unchanged, so Tab order follows visual order.

## Decision summary (published, informational)

| Finding | Criticality | Disposition | What was done |
|---|---|---|---|
| REV-001 Intermediate widths unverified | non-critical (provisional) | for adjudicator | — |
| REV-002 Stacked table semantics / labels | non-critical (provisional) | for adjudicator | — |
| REV-003 Desktop-wide wrapping rules | non-critical (provisional) | for adjudicator | — |
| REV-004 Unspecified small selectors | non-critical (provisional) | for adjudicator | — |

Published as an issue comment: pending (adjudicator). Included in the final run report: pending.

## Release readiness

No critical findings were observed. REV-001 needs QA measurement at 672–1055 px before release. REV-002 needs the manual screen-reader check.

## Reviewer recheck (after decisions.md)

- REV-001: verified. `tests/responsive-layout.spec.ts` (lines 189–208) adds 700 px and 1000 px tests. They cover the tracked list on the Amazon tab (5 columns, 200-character title, offers) and the LeBoncoin tab, plus the admin page with an 80-character username, and assert no horizontal page scroll. The reviewer ran them independently (`npx playwright test tests/responsive-layout.spec.ts -g "700px|1000px"`): 4 passed in Chromium. No CSS change since the review. The overflow risk is not observed, and this resolves the reviewer's concern. Residual: modals and the details page are not tested at intermediate widths. Low risk, because Carbon modals are width-bounded and the details page only uses wrapping rules.
- REV-004: verified. `technical.md` now has a section "Additional selectors found during development (REV-004)" that lists the notification `min-inline-size`, header-name/brand-mark, and breadcrumb selectors, each with its reason. It matches `src/index.css`. Resolved.
- REV-002 and REV-003: no code change. Their status follows the adjudicator's decisions and QA (manual screen-reader check, desktop visual comparison). No new findings.

## Reviewer recheck 2 (QA-F001, REV-001 reopened as critical)

- Diff: in `src/index.css` the stacked-table rules (`.responsive-table` block display, visually hidden `thead`, row and cell padding, `::before` labels, `.row-actions` alignment) are now in `@media (max-width: 1055.98px)`. They apply to both the tracked-items and users tables. The toolbar wrap and the StructuredList rules stay at 671.98px. Desktop (≥1056 px) is unaffected, and the test "desktop keeps the tracked table header row" still asserts `table-header-group` at 1056 px.
- Regression test: "QA-F001 {700,1000}px tables do not scroll inside and keep Status and actions on screen" checks `.cds--data-table-content` and `.tracked-table-container` for `scrollWidth <= clientWidth` on both tabs. It also checks that the Status cells and row actions end within the viewport. This covers the gap that the earlier REV-001 tests missed: they only checked document scroll, and Carbon's inner scroller hid the overflow.
- Use case `doc/use-cases/tracked-items/UC-TI-05-review-tracked-items-at-tablet-width.md` exists.
- Reviewer run: `npx playwright test tests/responsive-layout.spec.ts` gave 23 passed (Chromium). The coordinator reported 21. The difference is probably counting; all tests pass. The reviewer did not re-run the failing-on-old-breakpoint check, so that rests on the developer's report.
- Outcome: the REV-001 fix is verified. My earlier recheck, which accepted the document-scroll-only tests, was insufficient; this corrects it.
- New finding (non-critical, informational), REV-005: the admin tests at 700/1000 px still assert only document scroll, not the inner scroller. Since the users table now stacks at these widths, the risk is low. Optionally extend the QA-F001 scroller assertion to the admin page.
