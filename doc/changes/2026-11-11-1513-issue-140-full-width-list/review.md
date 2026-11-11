# Review: full-width tracked-items list on phones (issue #140 follow-up)

Reviewer: independent reviewer agent (`.agents/roles/reviewer.md`)
Revision reviewed: HEAD `114430f` plus the uncommitted working-tree diff (`doc/FUNCTIONAL_SPECIFICATIONS.md`, `doc/specifications/responsive-layout/functional.md`, `doc/specifications/responsive-layout/technical.md`, `src/index.css`, `tests/responsive-layout.spec.ts`).
Method: code reading only; browsers cannot launch in this environment (`libnspr4` missing), so the geometry below is computed from the CSS.

## Findings

### REV-001 Lateral gap miscounted: Carbon TabPanel padding ignored, list stays 16 px from each edge

- **Location:** `src/index.css` phone query (`.tracked-table-container { margin-inline: calc(-1 * (2rem + var(--cds-spacing-05))) }`); `technical.md` TS-RESPONSIVE-011.
- **Requirement:** FR-RESPONSIVE-014 (card left edge at x = 0, right edge at viewport width, at 320/360 px on both tabs).
- **Evidence:** the `TableContainer` is rendered inside a Carbon `<TabPanel>` (`src/components/TrackedItemsPage.tsx` lines 122-175), which gets `.cds--tab-content { padding: var(--cds-layout-density-padding-inline-local) }` (`node_modules/@carbon/styles/css/styles.css` ~29962). That local value is `clamp(0px, var(--cds-layout-density-padding-inline, 1rem), 999999999px)` = **1rem**; the app does not override `.cds--tab-content` or the density token. The real inset at phone width is therefore: `.cds--content` padding 2rem (styles.css 32192; `.app-content` only zeroes block-start) + `.page-shell` inset 1rem (`width: min(100% - 2 * spacing-05, 96rem)` at ≤1055.98 px) + TabPanel padding 1rem = **64 px per side**. The rule removes only 48 px. At 320 px: container left = 16, right = 304 (width 288), not 0/320. The new Playwright tests (`Math.abs(box.left) <= 1`) would fail.
- **Impact:** FR-RESPONSIVE-014 not met; the cards still show a 16 px page margin on each side. The new tests are red once browsers run.
- **Suggestion:** include the tab panel padding in the offset, e.g. `margin-inline: calc(-1 * (2rem + var(--cds-spacing-05) + var(--cds-layout-density-padding-inline-local, 1rem)))` (the custom property is defined on `.cds--tab-content`, an ancestor, so it inherits), or a fixed `calc(-1 * (3rem + var(--cds-spacing-05)))` with an updated comment. Update TS-RESPONSIVE-011's gap breakdown (64 px, three sources) and the TS-RESPONSIVE-014 risk note to list the TabPanel padding too.

- **Recheck (developer fix):** resolved. The rule is now `calc(-1 * (2rem + var(--cds-spacing-05) + var(--cds-layout-density-padding-inline-local, 1rem)))`. The custom property is set on `.cds--tabs, .cds--tab-content` (styles.css 29365-29367) and is not redefined on `.cds--data-table-container` or on any element between the panel and the container, so it inherits as 1rem. At 320 px: shell 224 px, panel content 192 px, plus 2 × 64 = 320 px, so left = 0 and right = 320. At 360 px it works out the same way. No overflow. The CSS comment names all three sources. The 320/360 tests are unchanged and now match the expected geometry.

## Checks with no finding

- **Overflow / box model:** once the offset matches the full inset, a `display: block`, `width: auto` container with negative margins gets a border box exactly equal to the viewport; no padding or border-inline on `.cds--data-table-container` (only `padding-block-start`), so `box-sizing` is irrelevant. The table (`width: 100%`, stacked as blocks at ≤1055.98 px) fills it; `.cds--data-table-content` has `overflow-x: auto` but content is not wider. No ancestor has `overflow: hidden`, so nothing is clipped. Mobile viewports have overlay scrollbars; in desktop emulation `document.documentElement.clientWidth` (used by the test) already excludes a classic scrollbar, and the inset chain is percentage-based on the same width, so the result stays consistent. Note: if the offset overshoots (e.g. a future smaller padding), the overflow would be horizontal page scroll — `expectNoHorizontalScroll` covers that.
- **Scope:** `.tracked-table-container` is used only in `TrackedItemsPage.tsx` line 175; the admin table uses `.responsive-table-container`. The rule is inside `@media (max-width: 671.98px)` only (FR-RESPONSIVE-017).
- **Inner padding (FR-RESPONSIVE-015):** existing phone `td` inline padding keeps text inset; the test's `textLeft >= 4` check via `Range.getBoundingClientRect` is sound (empty `td` would yield a 0-width rect at its padding edge, still ≥ 4).
- **Search row:** keeping `.tracked-table-search` at page margins is consistent with FR-RESPONSIVE-014 ("any element inside the same list container") since it sits outside `TableContainer`, and with FR-RESPONSIVE-016.
- **Tests:** 320/360 tests switch both tabs, wait for the long-title row, and also assert no horizontal scroll — sound, and they correctly detect REV-001. The 700 px test compares against `.page-shell`; since the TabPanel padding also applies there, `box.left` = shell.left + 16, so `toBeCloseTo(shell.left, 0)` (tolerance 0.5 px) **will fail** at 700 px even though the layout is correct (see REV-002).

### REV-002 700 px test compares to the wrong reference box

- **Location:** `tests/responsive-layout.spec.ts`, test "700px tracked-items list keeps its page margins".
- **Requirement:** FR-RESPONSIVE-017 / TS-RESPONSIVE-014 verification.
- **Evidence:** the container sits inside `.cds--tab-content` with 1rem inline padding, so its left edge is `.page-shell` left + 16 px and right edge is shell right − 16 px; `toBeCloseTo(shell.left, 0)` requires a difference < 0.5 px.
- **Impact:** false failure of a correct (unchanged) tablet layout; the test does not verify what TS-RESPONSIVE-014 states.
- **Suggestion:** compare against the enclosing `.cds--tab-content` content box (or the `.tracked-table-search` row left edge), or assert the container left/right equal those of the same element measured before the change; update the technical verification column accordingly.
- **Recheck (developer fix):** resolved in the test. The test now compares the container with `.tracked-table-search`. Both are block children of the same tab panel and have the same inline extent at 700 px, so `toBeCloseTo(..., 0)` holds. The `> 8` guard still detects any phone rule that leaks into this width. I did not re-verify the wording of the TS-RESPONSIVE-014 verification column in `technical.md` (it is documentation only).

## Recheck summary

Revision rechecked: HEAD `114430f` plus the updated working-tree diff. REV-001 and REV-002 are resolved. No new findings.

## Summary

2 findings. REV-001 (blocking for FR-RESPONSIVE-014): the negative margin omits the 1rem Carbon TabPanel padding, leaving 16 px margins. REV-002: the 700 px regression test uses the wrong reference and would fail on correct layout.
