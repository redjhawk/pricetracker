# Review decisions: full-width tracked-items list on phones (issue #140 follow-up)

Reviewed specification revisions: working-tree `doc/specifications/responsive-layout/functional.md` (FR-RESPONSIVE-014 to 017) and `technical.md` (TS-RESPONSIVE-011 to 014)
Reviewed code revision: HEAD `114430f` plus uncommitted working-tree diff (`src/index.css`, `tests/responsive-layout.spec.ts`, specs)
Reviewer: independent reviewer agent ([review.md](review.md))
Adjudicator: independent review adjudicator agent (separate from developer and reviewer)

## Independent verification

- `src/components/TrackedItemsPage.tsx` lines 122-175: `<TableContainer className="tracked-table-container">` (line 175) is rendered inside `<TabPanels><TabPanel>` (lines 122/124). The `.tracked-table-search` row (line 165) is also inside the TabPanel, outside `TableContainer`.
- `node_modules/@carbon/styles/css/styles.css` line 29962: `.cds--tab-content { padding: var(--cds-layout-density-padding-inline-local); }`; line 29366 defines the local value as `clamp(min, var(--cds-layout-density-padding-inline, normal), max)`. At `:root` (line 2803-2804) `normal = 1rem`, `min = 0px`; the app sets no density class and no `.cds--tab-content` override (grep of `src/*.css`, `src/**/*.tsx`: none). Resolved padding: **1rem on every side**.
- `.cds--content { padding: 2rem }` (styles.css 32192); `src/index.css` `.app-content` only zeroes `padding-block-start` (line 32-34). `.page-shell` at ≤ 1055.98 px: `width: min(100% - 2 * spacing-05, 96rem)`, centered (index.css 390-391) = 1rem per side.
- Total lateral inset at phone width = 2rem + 1rem + 1rem = 4rem = 64 px per side. The new rule (index.css 637-639) removes `2rem + spacing-05` = 48 px. At 320 px: container left = 16, right = 304. The reviewer's computation is correct.
- Tests could not be executed (see Environment limitation); verification is by code and CSS reading only.

## Findings and decisions

### REV-001: Negative margin omits the Carbon TabPanel padding; list keeps 16 px lateral margins

- Evidence: see Independent verification. `src/index.css` line 638 offsets 48 px against a 64 px inset. `technical.md` TS-RESPONSIVE-011 lists only two gap sources (3rem) and omits `.cds--tab-content` padding.
- Impact and scenario: every phone user (≤ 671 px, both tabs) still sees a 16 px page margin on each side of the item cards. FR-RESPONSIVE-014 ("left edge at x = 0, right edge at viewport width, no page background visible beside the cards") is not met. The new 320/360 px Playwright tests would fail.
- Criticality: **critical**, because the change exists solely to deliver FR-RESPONSIVE-014 (owner request "que la lista ocupe todo el ancho en móvil, sin márgenes laterales") and with this defect the requested feature does not work at all — it only narrows the margin from 64 px to 16 px. Likelihood 100% on all phone widths; certainty high (values derived from shipped Carbon CSS, no override in the app).
- Disposition: **fix**.
- Reason: the requested functionality is not delivered; no deferral is acceptable.
- Required change (developer):
  1. `src/index.css`, phone query: include the TabPanel padding in the offset, preferably by inheriting Carbon's token so a density change follows automatically:
     `.tracked-table-container { margin-inline: calc(-1 * (2rem + var(--cds-spacing-05) + var(--cds-layout-density-padding-inline-local, 1rem))); }`
     (the property is defined on the ancestor `.cds--tab-content`, so it inherits). A fixed `calc(-1 * (3rem + var(--cds-spacing-05)))` is acceptable if the comment states the three sources. Update the CSS comment to list all three sources.
  2. `doc/specifications/responsive-layout/technical.md`: TS-RESPONSIVE-011 gap breakdown = three sources, 4rem = 64 px per side; TS-RESPONSIVE-014 risk note must also name the `.cds--tab-content` padding / density token.
  3. Use case: add `doc/use-cases/tracked-items/UC-TI-06-review-tracked-items-full-width-on-phone.md` (actor: user on a phone at 320/360 px; main flow: open tracked items, view Amazon then LeBoncoin tab; postconditions: cards edge to edge with no lateral page margin, text inset by card padding, no horizontal scroll; header/tabs/search unchanged; references FR-RESPONSIVE-014 to 017), and link it from `doc/use-cases/README.md` under "Tracked items". Reference it from TS-RESPONSIVE-011 and the change `index.md`.
  4. Regression test: the existing tests `320px tracked-items list spans the full width` and `360px tracked-items list spans the full width` in `tests/responsive-layout.spec.ts` serve as the Playwright regression test for UC-TI-06; add a comment naming UC-TI-06 and REV-001. They must assert, on both tabs, `|left| ≤ 1`, `|right − clientWidth| ≤ 1`, text inset ≥ 4 px, and no horizontal scroll (already present).
- Validation condition: reviewer recheck confirms the computed offset equals the 64 px inset (container left = 0, right = viewport at 320 and 360 px); the two Playwright tests above pass in an environment where Chromium launches; existing responsive tests (desktop header row, tablet stacking, no horizontal scroll) still pass; QA visually confirms at 320/360 px both tabs, and no change at ≥ 672 px.
- Specification decision: not applicable (technical defect; functional spec is clear).
- Resolution: **fixed; verified from code; regression test pending execution.** `src/index.css` phone query now uses `calc(-1 * (2rem + var(--cds-spacing-05) + var(--cds-layout-density-padding-inline-local, 1rem)))` = 64 px, matching the independently computed inset; the CSS comment names the three sources. `technical.md` TS-RESPONSIVE-011 (three sources, 64 px, UC-TI-06 link) and the TS-RESPONSIVE-014 risk note are updated. Reviewer recheck ([review.md](review.md), Recheck summary): resolved, no new findings. QA ([qa.md](qa.md)) is blocked: `npm run build` passes and the built CSS has the rule only inside `@media (max-width:671.98px)`, but no browser test ran.
- Follow-up: coordinator runs `npx playwright test tests/responsive-layout.spec.ts` in CI (or any environment where Chromium launches) and records the result here and in qa.md before merge.
- Critical fix confirmation: **NOT confirmed — pending execution.** Present: corrected code; reviewer recheck; use case [UC-TI-06](../../use-cases/tracked-items/UC-TI-06-review-tracked-items-full-width-on-phone.md) linked from `doc/use-cases/README.md`; regression tests `320px/360px tracked-items list spans the full width` in `tests/responsive-layout.spec.ts` (comment at line 270 names UC-TI-06/REV-001). Missing: a passing run. Confirmation condition: both tests, and the rest of `responsive-layout.spec.ts`, pass on Chromium. If they fail, REV-001 reopens as critical.

### REV-002: 700 px test compares the list to `.page-shell`, which is wrong by the TabPanel padding

- Evidence: at 700 px the container sits inside `.cds--tab-content` (1rem padding), so `box.left = shell.left + 16`; `toBeCloseTo(shell.left, 0)` (tolerance < 0.5 px) fails on the correct, unchanged layout. TS-RESPONSIVE-014 / the FR-RESPONSIVE-017 verification row states the same wrong reference.
- Impact and scenario: a false-red regression test for FR-RESPONSIVE-017; it would block CI or, worse, push someone to "fix" a correct tablet layout. No user-facing defect.
- Criticality: **non-critical**, because the product behavior at ≥ 672 px is correct and unchanged; the defect is confined to test verification. It must still be fixed before commit because a knowingly failing test cannot be merged and FR-RESPONSIVE-017 would be unverified.
- Disposition: **fix**.
- Reason: cheap, local, and required for a meaningful FR-RESPONSIVE-017 check.
- Required change (developer): in `tests/responsive-layout.spec.ts` "700px tracked-items list keeps its page margins", compare the container with the enclosing `.cds--tab-content` content box (its rect inset by its computed `padding-left`/`padding-right`) or with the `.tracked-table-search` row's left/right edges, keeping `box.left > 8`. Update the FR-RESPONSIVE-017 verification cell in `technical.md` accordingly.
- Validation condition: reviewer recheck that the reference matches the actual DOM chain; test passes at 700 px once Chromium runs.
- Specification decision: not applicable.
- Resolution: fixed; verified from code. The test now compares the container with `.tracked-table-search` (same tab-panel content box; comment at line 294) and keeps the `> 8` guard; the `technical.md` FR-RESPONSIVE-017 verification cell is updated. Reviewer recheck: resolved. Not executed (no browser).
- Follow-up: same CI run as REV-001.

## Environment limitation

Chromium cannot launch in this runner (`libnspr4.so` missing, no sudo to install system libraries), so no Playwright test — including the regression tests required for REV-001 — can be executed here. Consequence: the REV-001 critical fix can be verified only by code reading and computed geometry in this run. The "passing regression test" condition is therefore **unmet** here; it must be satisfied by CI (or another environment with browser dependencies) before the critical fix is recorded as confirmed. The coordinator must report this as a limitation and must not claim the tests passed.

## Decision summary (published, informational)

See [decision-summary.md](decision-summary.md).

| Finding | Criticality | Disposition | What was done |
|---|---|---|---|
| REV-001 TabPanel padding omitted, 16 px margins remain | critical | fix | Fixed (64 px offset including the tab-panel token), reviewer-rechecked from code, UC-TI-06 added; regression tests written but **not executed — confirmation pending a green CI run** |
| REV-002 700 px test uses wrong reference | non-critical | fix | Fixed (compares with `.tracked-table-search`), reviewer-rechecked; not executed |

Published as an issue comment: by the coordinator on issue #140. Included in the final run report: yes.

## Release readiness

Both findings are fixed and reviewer-rechecked from code; review has no open findings. Not fully confirmed: REV-001 (critical) has no passing regression run, and browser QA is blocked. The coordinator may commit and open the PR, but must report this limitation, must not claim tests passed, and must not merge until CI runs `tests/responsive-layout.spec.ts` green on Chromium. A failing result reopens REV-001.
