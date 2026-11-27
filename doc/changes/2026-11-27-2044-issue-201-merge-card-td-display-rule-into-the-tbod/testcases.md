# Test cases: merge the card td display rule into the tbody td block (issue #201)

Sources: functional specification [responsive-layout](../../specifications/responsive-layout/functional.md) (FR-RESPONSIVE-034 to FR-RESPONSIVE-036). No use cases: see [no-use-cases.md](no-use-cases.md). The card layout below 1056 px that must not change is described by UC-TI-07 and UC-RL-02, which this change does not modify. No new `UC-` IDs are defined.

Conventions:

- **Base:** `origin/master` (or `master`), compared with the change applied on top of it.
- **The media block:** the `@media (max-width: 1055.98px)` block of `src/index.css` that holds the card table rules (`.responsive-table tr`, `.responsive-table tbody tr td`, `.responsive-table td[data-label]::before`).
- **The cell rule:** the rule in the media block whose selector is `.responsive-table tbody tr td` alone (not the `:has(> .item-cell)` or `:has(> .row-actions)` variants).
- **The standalone rule:** a rule whose whole selector list is `.responsive-table td`.
- The change only moves one declaration between two stylesheet rules. FR-RESPONSIVE-036 allows no other file change outside `doc/`, so no new Go or Playwright test code is committed: the cases are repository checks (`grep`, `git diff`, reading `src/index.css`), runs of the existing suites, and one uncommitted cell measurement (TC-003). No API, database or user interface changes; invalid input, empty results and repeated actions do not apply beyond the corner cases listed. Narrow screens and keyboard use are covered by the existing responsive-layout e2e suite (TC-004).

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-RESPONSIVE-034 | TC-001, TC-002 |
| FR-RESPONSIVE-035 | TC-003, TC-004 |
| FR-RESPONSIVE-036 | TC-002, TC-004, TC-005 |

## Cases

### TC-001 — `display: block` for card cells is in the cell rule and the standalone rule is gone

- Covers: FR-RESPONSIVE-034
- Level: repository check (file content of `src/index.css`)
- Given the change applied on top of the base
- When the media block is read (for example `grep -n -A10 'responsive-table' src/index.css`)
- Then:
  - the cell rule contains the declaration `display: block`, exactly once;
  - the media block has no standalone rule (no rule whose selector list is exactly `.responsive-table td`, with or without surrounding whitespace);
  - the cell rule keeps its other declarations as on the base: `flex: 0 1 auto`, `width: auto`, `min-width: 0`, `max-width: 100%`, `height: auto`, `padding: var(--cds-spacing-03) var(--cds-spacing-05)`, `border: 0`
- Corner cases:
  - `display: block` is not merely added to another rule (for example `.responsive-table, .responsive-table tbody` or a `:has(...)` variant) instead of the cell rule;
  - no other media query of `src/index.css` gains a standalone rule (the `max-width: 671.98px` blocks included);
  - on the base, the check fails: the standalone rule exists (`.responsive-table td { display: block; }`) and the cell rule has no `display` declaration.

### TC-002 — The neighbouring `td` rules are unchanged and only the two card-cell rules differ

- Covers: FR-RESPONSIVE-034, FR-RESPONSIVE-036
- Level: repository check (`git diff`)
- Given the change applied on top of the base
- When `git diff --name-status <base> -- . ':!doc'` and `git diff -U0 <base> -- src/index.css` run
- Then:
  - the first command lists only `M src/index.css`;
  - the hunks of `src/index.css` only remove the standalone rule (its selector, `display: block` and braces, plus at most one adjacent blank line) and add `display: block` inside the cell rule;
  - the `.responsive-table td[data-label]::before` rule in the media block is unchanged (`content: attr(data-label)`, `display: block`, margin, colour, font size and weight);
  - the top-level `overflow-wrap: anywhere` rule outside the media queries still lists `.responsive-table td` in its selector list, unchanged
- Corner cases:
  - no `.tsx`, `.ts`, `.spec.ts`, `.go`, `_test.go`, `go.mod`, `go.sum`, `package.json`, `package-lock.json` or `API_SPECIFICATION.md` file changes;
  - no other rule of `src/index.css` changes, including `.responsive-table tr`, `.responsive-table thead`, the `:has(> .item-cell)` and `:has(> .row-actions)` rules and the TS-RESPONSIVE comments;
  - no formatting-only change elsewhere in `src/index.css` (whitespace, reordering of declarations in other rules).

### TC-003 — Every card and table cell keeps its rendered position and size

- Covers: FR-RESPONSIVE-035
- Level: e2e (Playwright measurement, run against the base build and the changed build; not committed, as FR-RESPONSIVE-036 allows no test file change)
- Given the mocked API of `tests/responsive-layout.spec.ts`, with Amazon and LeBoncoin tracked items (including a long title) and an administrator session with at least one user (including a long username), built once from the base and once from the change
- When, at 320 px, 360 px, 775 px, 1055 px and 1280 px wide, on the tracked items page (Amazon tab, then LeBoncoin tab) and on the administrator page, the bounding box (left, top, width, height) and computed `display` of every `td` inside `.responsive-table` are read
- Then:
  - at each width and on each page or tab, the list of cells and each cell's bounding box are equal on the base and on the change (difference at most 0.5 px);
  - at 320, 360, 775 and 1055 px, every cell's computed `display` is `block` on both builds;
  - at 1280 px, every cell's computed `display` is `table-cell` on both builds (the desktop table is unaffected)
- Corner cases:
  - the tracked items no-match row (search text with no match: the single `colSpan` cell) has the same box on both builds, at 360 and 1055 px;
  - every `td` inside `.responsive-table` is inside a `tbody tr` on both builds (no `thead` or `tfoot` cell relied on the standalone rule), which is why removing it cannot change any cell;
  - the `data-label` captions (`::before`) still render above their value at 360 px (cell heights equal on both builds);
  - 1055 px versus 1056 px: at 1055 px the cards apply, at 1056 px the desktop table of TC-012 (issue #165) applies, on both builds.

### TC-004 — The existing suites and checks pass

- Covers: FR-RESPONSIVE-035, FR-RESPONSIVE-036
- Level: unit, integration and e2e (existing Go tests and Playwright suite) and frontend checks
- Given the change applied on top of the base, with the existing tests unchanged
- When `go vet ./...`, `go test -count=1 ./...`, `node --test scripts/*.test.mjs`, `npm run build` and `npx playwright test` run
- Then all pass, with the same test list as on the base; in particular in `tests/responsive-layout.spec.ts`:
  - `tracked items list on both tabs` and `administrator page with a long username` at 320 px and 360 px (FR-RESPONSIVE-009: no horizontal scroll, content on screen);
  - `QA-F001 … tables do not scroll inside and keep Status and actions on screen` and the `list on both tabs` / `admin page` width tests;
  - `TC-001` to `TC-010` of issue #165 (FR-RESPONSIVE-018 to 024: cards edge to edge, content inset, card layout and full width switch at 1056 px, no-match message, values side by side, reading and focus order, administrator list edge to edge with a long username wrapping);
  - `TC-012 desktop tracked items layout unchanged` and `TC-012 desktop administrator page layout unchanged`
- Corner cases:
  - narrow screens (320 to 1055 px) and the 1056 px desktop switch behave as on the base;
  - keyboard: `TC-006 phone card values in reading and focus order` still passes (Tab order and focused cells fully on screen);
  - repeated runs give the same result (`-count=1`, no cached Go results).

### TC-005 — Nothing else changes in the specifications, change records and todo files

- Covers: FR-RESPONSIVE-036
- Level: repository check (`git diff`)
- Given the change applied on top of the base
- When these commands run:
  - `git diff <base> -- doc/specifications/responsive-layout/functional.md`;
  - `git diff --name-status <base> -- doc/changes/*issue-140-* doc/changes/*issue-165-* doc/changes/*issue-200-* doc/changes/*issue-202-* doc/changes/*issue-203-*`;
  - `git diff --name-status --diff-filter=DMRT <base> -- doc/todo/`
- Then:
  - in `functional.md`, no line of FR-RESPONSIVE-001 to FR-RESPONSIVE-033 is added, removed or modified; the only changes are the issue #201 section (FR-RESPONSIVE-034 to 036) and the issue #201 Traceability bullet;
  - the second and third commands print nothing: the change folders of issues #140, #165, #200, #202 and #203 and every existing file under `doc/todo/` (including `issue-165-r1-03.md`) are unchanged; new follow-up records for issue #201 (`doc/todo/issue-201-r*.md`) are allowed
- Corner cases:
  - no FR-RESPONSIVE, UC-TI or UC-RL ID is renumbered or removed (the sorted list of FR-RESPONSIVE IDs defined in `functional.md` equals the base list plus FR-RESPONSIVE-034 to 036);
  - `doc/use-cases/tracked-items/UC-TI-07-review-tracked-items-edge-to-edge-below-desktop.md` and `doc/use-cases/responsive-layout/UC-RL-02-review-users-list-edge-to-edge-below-desktop.md` are unchanged;
  - the user observes no change in the application: no API, database, setting or screen change (follows from TC-002, TC-003 and TC-004).
