# Implementation: full-width tracked-items list on phones

## Mapping

| Requirement | Technical | Change |
| --- | --- | --- |
| FR-RESPONSIVE-014 | TS-RESPONSIVE-011 | `src/index.css`: in `@media (max-width: 671.98px)`, `.tracked-table-container { margin-inline: calc(-1 * (2rem + var(--cds-spacing-05))) }` |
| FR-RESPONSIVE-015 | TS-RESPONSIVE-012 | none; existing `td` inline padding (spacing-05) kept |
| FR-RESPONSIVE-016 | TS-RESPONSIVE-013 | container width equals viewport; search, heading, tabs untouched |
| FR-RESPONSIVE-017 | TS-RESPONSIVE-014 | rule only in the phone query; admin table class not targeted |

## Files

- `src/index.css`: one scoped rule with comment.
- `tests/responsive-layout.spec.ts`: helper `listBox`; tests "320px/360px tracked-items list spans the full width" (both tabs: left ≈ 0, right ≈ viewport width within 1 px, cell text left ≥ 4 px, no horizontal scroll) and "700px tracked-items list keeps its page margins" (aligned with `.page-shell`, left > 8 px).
- `doc/specifications/responsive-layout/technical.md`, change `index.md`.

## Gap measurement

Not measured in a browser (see limitations). Derived from CSS: Carbon `.cds--content { padding: 2rem }` (`node_modules/@carbon/styles/css/styles.css`; `.app-content` overrides only `padding-block-start`) plus `.page-shell` inset of `spacing-05` (1rem) below 1056 px = 48 px per side at 360 px. The new tests assert the result.

## Commands and results

- `npm run build` (tsc -b && vite build): passed.
- `npx playwright test tests/responsive-layout.spec.ts -g "full width|700px tracked"`: 3 failed at browser launch, `chrome-headless-shell: error while loading shared libraries: libnspr4.so` (missing system library; installing it needs sudo, not permitted). No test assertion ran.

## Review fixes (decisions.md)

- REV-001 (critical): the gap also includes the Carbon tab panel padding (`.cds--tab-content`, 1rem), so 64 px per side. Rule is now `margin-inline: calc(-1 * (2rem + var(--cds-spacing-05) + var(--cds-layout-density-padding-inline-local, 1rem)))`, comment updated; TS-RESPONSIVE-011/014 updated; use case `doc/use-cases/tracked-items/UC-TI-06-review-tracked-items-full-width-on-phone.md` added and linked from `doc/use-cases/README.md`, TS-RESPONSIVE-011 and the change index; the 320/360 px tests carry a comment naming UC-TI-06 and REV-001 as their regression test.
- REV-002: the 700 px test now compares the container edges with the `.tracked-table-search` row (the tab panel content box) instead of `.page-shell`; FR-RESPONSIVE-017 verification row updated.
- `npm run build` after the fixes: passed. Playwright still cannot launch Chromium here (`libnspr4.so` missing), so the regression tests have not run.

## Limitations

The Playwright tests (new and existing) must be run by QA or CI in an environment with Chromium dependencies. No dev server was left running (Playwright's `webServer` manages it).
