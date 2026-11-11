# Review decisions: issue #140 responsive layout

Reviewed specification revisions: `doc/specifications/responsive-layout/functional.md` (ready, FR-RESPONSIVE-001 to 013), `doc/specifications/responsive-layout/technical.md` (ready, TS-RESPONSIVE-001 to 009), working tree
Reviewed code revision: uncommitted working tree on `ai-dev/issue-140-20261111-1142` (base `f6ed820`), including untracked `tests/responsive-layout.spec.ts`
Reviewer: independent reviewer agent ([review.md](review.md))
Adjudicator: independent review adjudicator agent (separate from developer and reviewer)

No API or backend change; `API_SPECIFICATION.md` is unaffected.

## Findings and decisions

### REV-001: Intermediate widths (672–1055 px) are not verified, and the table can still overflow there

- Evidence: `src/index.css` stacks the tables only at `max-width: 671.98px`; the former `min-width: 66rem` / `overflow-x: auto` guard is removed, and `.latest-price` / `.price-history-list li` keep `white-space: nowrap` outside the phone query. `tests/responsive-layout.spec.ts` checks 320, 360 and 1056 px only. Functional spec, "States and corner cases": "Widths between 320 px and desktop must also satisfy FR-RESPONSIVE-001 to 003". TS-RESPONSIVE-003 states the table "must fit" at 672–1055 px and defines the fallback: stack up to 1055.98px if QA finds overflow.
- Impact and scenario: a tablet or landscape-phone user (about 700–1000 px) on the Amazon tab, with a long title and a price history, could get sideways page scrolling. Because the inner scroll container was removed, any overflow now widens the whole page, which is a regression from the current release. Overflow is plausible (five columns, non-wrapping price lines, cell padding) but not observed.
- Criticality: pending measurement, treated as blocking until measured. If overflow is measured at any width from 672 to 1055 px, it is critical: it violates a stated requirement (FR-RESPONSIVE-001 over the full range) and regresses current behavior for a realistic device class. If no overflow is measured, the finding is non-critical (a test-coverage gap only). Lack of a test is not proof of safety, so release may not rely on the unverified state.
- Disposition: fix.
- Reason: the requirement explicitly covers this range, and the specification already gives the remedy, so no functional decision is needed. The fix has two parts. (1) The developer adds the `scrollWidth <= clientWidth` check to `tests/responsive-layout.spec.ts` at 700 px and 1000 px for the tracked-items list (Leboncoin and Amazon tabs, long title, price history) and the admin page. (2) If either check fails, the developer applies the TS-RESPONSIVE-003 fallback (stacked layout up to `max-width: 1055.98px`) and records it in `implementation.md`. This is a technical decision, already allowed by the spec.
- Specification decision: not applicable (covered by TS-RESPONSIVE-003 fallback).
- Resolution history:
  1. First recheck: the 700/1000 px tests passed and REV-001 was closed as non-critical. Those tests asserted only document-level scroll.
  2. QA ([qa.md QA-F001](qa.md), QA-003/QA-004) then measured real overflow. From 672 to about 1006 px the table (761 px) was wider than its container. Carbon's inner `.cds--data-table-content` scroller hid it from the document check, so Status and the row actions were clipped and reachable only by scrolling the table sideways. This happened on the Amazon tab, and on LeBoncoin at 672–800 px.
  3. **Final criticality: critical.** A specified requirement was violated: FR-RESPONSIVE-001 to 003 must hold across 320 px to desktop, and FR-RESPONSIVE-003 requires actions reachable without sideways scrolling. Status and the actions were unreachable without scrolling for every tablet-width user of the main screen. This was observed, not hypothetical.
- Resolution: **fixed and confirmed.** The developer applied the specified TS-RESPONSIVE-003 fallback: the stacked-table rules in `src/index.css` now apply at `@media (max-width: 1055.98px)` to both `.responsive-table` tables. The toolbar and StructuredList rules stay at 671.98px, and desktop (≥1056 px) is unchanged. See [implementation.md, "QA fix QA-F001"](implementation.md).
- Follow-up: none for REV-001. The admin test gap is tracked separately as REV-005.
- Critical fix confirmation:
  - Corrected code: `src/index.css` stacked-table block moved to `max-width: 1055.98px`. Verified in [review.md, "Reviewer recheck 2"](review.md).
  - Use case: [UC-TI-05 — Review tracked items at tablet width](../../use-cases/tracked-items/UC-TI-05-review-tracked-items-at-tablet-width.md), linked from `doc/use-cases/README.md`.
  - Regression test: `QA-F001 <width>px tables do not scroll inside and keep Status and actions on screen` (700, 800, 1000 px) in `tests/responsive-layout.spec.ts`. On both tabs it asserts `scrollWidth <= clientWidth` on `.cds--data-table-content` and `.tracked-table-container`, and that Status and `.row-actions` end within the viewport.
  - Fails without the fix: with the 671.98px breakpoint, 3 of 3 tests failed (developer report; the reviewer did not re-run this).
  - Passes with the fix: the reviewer independently ran `npx playwright test tests/responsive-layout.spec.ts` and got 23 passed in Chromium. The coordinator's count of 21 is a counting difference; there were no failures.
  - Desktop check still passes: "desktop keeps the tracked table header row" (1056 px).
  - Validation condition before commit: QA retests QA-002/003/004 at 672, 800, 1000 and 1055 px, measuring the inner-scroller `scrollWidth` against `clientWidth`, and re-runs the QA-015 desktop comparison at 1056 px, as requested in qa.md. A failure reopens this finding.

### REV-002: Stacked tables may lose table semantics in some screen readers; the Item and actions cells have no label

- Evidence: the phone query sets `display: block` on `.responsive-table` table, `tr` and `td`; `thead` is visually hidden (not `display: none`), as specified. WebKit is known to drop implicit table roles in that case. `td[data-label]::before` shows "Prices", "Amazon second-hand offer", "Status", "Username" and "Last login". The Item and actions cells have no label, as TS-RESPONSIVE-003 specifies. FR-RESPONSIVE-012: "At 360 px, screen readers announce each value with its label".
- Impact and scenario: an iOS VoiceOver user on a phone may hear values without the column header, but the `::before` label text is part of the content and is read before each labelled value, so the label is still announced. The unlabelled cells are the item title (self-identifying link button) and the action buttons (which have accessible names). In Chromium, labels may be read twice (header plus `::before`): verbose, not wrong.
- Criticality: non-critical. The requirement ("value announced with its label") is met by the generated label text in both the semantics-kept and semantics-lost cases. The design is the approved TS-RESPONSIVE-003. No information becomes unavailable to the user; the worst case is verbosity or the loss of table navigation shortcuts. However, FR-RESPONSIVE-012 is not covered by automated tests, so it must be verified.
- Disposition: fix (verification only, no code change unless the check fails).
- Reason: adding explicit `role="table"/"row"/"cell"/"columnheader"` attributes would change markup beyond the specification and could cause other double announcements. That is not justified without observed failure. A manual check is cheap and is required by FR-RESPONSIVE-012 anyway.
- Specification decision: not applicable.
- Resolution: **closed, validated** by [QA-014](qa.md). At 360 px, Playwright `ariaSnapshot()` and the CDP full accessibility tree were taken for "Tracked items" (both tabs) and "Users". Chromium keeps table semantics. Every labelled cell is exposed with its label first, for example `cell "Prices 279,00 € …"`, `cell "Status Active"`, `cell "Last login Never"`. Title buttons are named after the item, and action buttons have names such as "Open amazon listing in a new tab" and "Options". No code change was needed.
- Follow-up: the check covered Chromium only; VoiceOver/WebKit was not tested. That remaining risk is low and accepted: even if WebKit drops the table roles, the generated `::before` label stays in the content in front of each value. No issue is required.

### REV-003: Some wrapping rules outside the media queries also apply on desktop

- Evidence: `overflow-wrap: anywhere` on `.responsive-table td`, `.item-title.cds--btn`, `.cds--modal-content` and headings, and a global `img { max-width: 100% }`, are outside any media query. TS-RESPONSIVE-009 explicitly allows pure wrapping rules to apply on all widths. The only effect at ≥1056 px is that long unbroken strings (for example an 80-character username) wrap instead of widening the table.
- Impact and scenario: desktop rendering of realistic data is unchanged; only pathological unbroken strings render differently, and they no longer overflow, which is better. FR-RESPONSIVE-011 ("unchanged") is about layout, columns, positions and actions, which stay the same.
- Criticality: non-critical. It follows the approved technical specification and does not cause loss of function or data.
- Disposition: reject (no code change).
- Reason: moving these rules into media queries would bring back desktop overflow for long strings and contradict TS-RESPONSIVE-009. `img { max-width: 100% }` does not affect images that are smaller than their container. The remaining risk is a visual difference with realistic data, which the planned QA desktop comparison covers.
- Specification decision: not applicable.
- Resolution: **rejected; closed, validated** by [QA-015](qa.md). At 1056 and 1440 px, QA rendered the list, both detail pages, admin and the Add item, Settings and Information modals with the branch CSS and with `master:src/index.css`. There were 0 geometry differences on every screen and modal (for example, 303 of 303 elements identical on the Amazon list). This QA-015 run predates the QA-F001 fix, so the desktop comparison is repeated in the REV-001 validation condition.
- Follow-up: none.

### REV-005: Admin 700/1000 px tests check only document-level scroll, not Carbon's inner table scroller

- Evidence: [review.md, "Reviewer recheck 2"](review.md). The admin tests at 700/1000 px in `tests/responsive-layout.spec.ts` assert document `scrollWidth`, the same weak check that hid QA-F001. The QA-F001 inner-scroller assertion covers only the tracked-items table.
- Impact and scenario: the admin users table could overflow inside its scroller at tablet width without a test failing. That is unlikely now: since the REV-001 fix, the users table shares the `.responsive-table` stacked rules up to 1055.98px, and the QA-F001 tests exercise those same rules. The table has only two short columns (Username, Last login) and wraps long usernames (`overflow-wrap: anywhere`).
- Criticality: non-critical. This is a test-coverage gap with no observed defect. The CSS path it would guard is already covered by the tracked-items regression test, and the admin page reaches a small number of administrators.
- Disposition: defer.
- Reason: adding the assertion is cheap, but it would reopen the developer, review and QA cycle for a defect that has not been observed, and whose CSS is already protected by the QA-F001 test on the shared class. Remaining risk: a future admin-only style change could reintroduce inner overflow unnoticed. A follow-up test is the right place for that, outside this change.
- Specification decision: not applicable.
- Resolution: deferred. Validation for this change: QA's REV-001 retest also measures the admin `.cds--data-table-content` scrollWidth against clientWidth at 700 and 1000 px. If it overflows, this finding becomes critical and is fixed in this change.
- Follow-up: the coordinator records a `doc/todo/` item and creates a GitHub issue if possible: "Extend QA-F001 inner-scroller assertion to the admin users table at 700/800/1000 px".

### REV-004: Small unspecified selectors added

- Evidence: `.detail-breadcrumb` wrapping, `min-inline-size: 0` on `.detail-notification` and `.modal-notification`, and `.cds--header__name` padding are not listed in `technical.md`. `implementation.md` gives their reason: failing 320 px checks for FR-RESPONSIVE-001/009/010.
- Impact and scenario: these are CSS-only fit fixes and add no new behavior. They are a traceability gap, not a scope deviation.
- Criticality: non-critical. They implement specified requirements and add no functionality.
- Disposition: fix (documentation only).
- Reason: the workflow asks for technical decisions to be traceable. A one-line addition to `technical.md` (TS-RESPONSIVE-001/009 section) listing these selectors and their reason is cheap and closes the gap. No code change.
- Specification decision: not applicable.
- Resolution: fixed. `technical.md` now has the section "Additional selectors found during development (REV-004)", which matches `src/index.css`. The reviewer verified it. Validation condition: the 320 px Playwright checks that motivated the selectors keep passing.
- Follow-up: none.

## Additional note (not a finding)

The 7 Playwright failures in `app-information`, `leboncoin-session-settings` and platform-tabs "both tabs remain available…" fail identically on unmodified `master`. The coordinator reproduced this with a stash run. They are pre-existing, not caused by this change, and do not block. Any new failure would block the commit.

## Decision summary (published, informational)

| Finding | Criticality | Disposition | What was done |
|---|---|---|---|
| REV-001 Tables cut off at 672–1055 px (QA-F001) | critical | fix | Stacked tables extended to `max-width: 1055.98px` (the specified fallback). Use case [UC-TI-05](../../use-cases/tracked-items/UC-TI-05-review-tracked-items-at-tablet-width.md). Regression test "QA-F001 …" at 700/800/1000 px fails on the old breakpoint (developer) and passes now (reviewer run: 23/23 in the spec file). Confirmed; QA retest still due. |
| REV-002 Stacked table may lose table semantics in screen readers | non-critical | fix (check only) | Closed: QA-014 passed; each value is exposed with its label (Chromium only, VoiceOver not tested). |
| REV-003 Wrapping rules also apply on desktop | non-critical | reject | Closed: QA-015 found 0 geometry differences against `master` at 1056 and 1440 px. |
| REV-004 Small CSS selectors not in the technical spec | non-critical | fix (docs only) | Listed in `technical.md`; reviewer verified. |
| REV-005 Admin tablet tests check only page scroll | non-critical | defer | The shared CSS is covered by the QA-F001 test. Follow-up `doc/todo/` item and issue to extend the inner-scroller assertion to admin; QA measures admin in this run. |

Published as an issue comment: by the coordinator on issue #140. Included in the final run report: yes.

## Release readiness

The only critical finding is REV-001, which reopened through QA-F001. Its fix is confirmed: corrected code, reviewer recheck, use case UC-TI-05, and a passing regression test that failed before the fix. REV-002, REV-003 and REV-004 are closed. REV-005 is deferred with a follow-up. The 7 Playwright failures are pre-existing on `master` (reproduced).

Remaining gate before commit: the QA retest for REV-001 (QA-002/003/004 at 672, 800, 1000 and 1055 px with inner-scroller measurement, including admin for REV-005, and the QA-015 desktop comparison at 1056 px). Once it passes, no blocking findings remain. There are no open functional questions.
