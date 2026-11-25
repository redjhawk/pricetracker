# Test cases: align TS-RESPONSIVE-017 wording with the growing item cell (issue #200)

Sources: functional specification [responsive-layout](../../specifications/responsive-layout/functional.md) (FR-RESPONSIVE-028 to FR-RESPONSIVE-030). No use cases: see [no-use-cases.md](no-use-cases.md). The responsive-layout specification defines no new `UC-` IDs, and its behaviour does not change.

Conventions:

- **Base:** `origin/master` (or `master`), compared with the change applied on top of it.
- **The entry:** the TS-RESPONSIVE-017 entry of `doc/specifications/responsive-layout/technical.md`, from the line starting `- **TS-RESPONSIVE-017` up to, not including, the line starting `- **TS-RESPONSIVE-018`.
- **The old wording:** the texts `Cells do not grow` and `the free space left on a line is always smaller than the next cell's width`.
- The change only edits documentation. FR-RESPONSIVE-030 allows no file change outside `doc/`, so no new Go or Playwright test code is written: the cases are repository checks (`grep`, `git diff`) and runs of the existing suites. Invalid input, empty results, keyboard and screen states do not apply; no API, database or user interface changes. Narrow screens are covered by the existing responsive-layout e2e suite (TC-005).

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-RESPONSIVE-028 | TC-001, TC-003 |
| FR-RESPONSIVE-029 | TC-002 |
| FR-RESPONSIVE-030 | TC-003, TC-004, TC-005 |

## Cases

### TC-001 — The TS-RESPONSIVE-017 entry states the cell growth rules and refers to TS-RESPONSIVE-023

- Covers: FR-RESPONSIVE-028
- Level: repository check (file content)
- Given the change applied on top of the base
- When the entry is extracted, for example with `awk '/^- \*\*TS-RESPONSIVE-017/{p=1} /^- \*\*TS-RESPONSIVE-018/{p=0} p' doc/specifications/responsive-layout/technical.md`
- Then the entry:
  - states that value cells do not grow (for example "Value cells do not grow");
  - states that only the item cell grows into the space left on its line;
  - states that the actions cell is pushed to the line end;
  - contains `TS-RESPONSIVE-023`, in the sentence that carries these rules
- Corner cases:
  - exactly one line of `technical.md` starts `- **TS-RESPONSIVE-017` (the entry is not duplicated or renumbered);
  - the rest of the entry is unchanged from the base: the `tr` and `td` flex rules, the line-breaking sentence citing FR-RESPONSIVE-024, the long-cell, DOM-order and `nowrap` paragraph; within the entry, `git diff <base> -- doc/specifications/responsive-layout/technical.md` touches only the sentence that held the old wording (lines outside the entry are covered by TC-003);
  - the stated rules match the shipped CSS: `src/index.css` still contains `td:has(> .item-cell)` with `flex: 1 1 14rem` and `td:has(> .row-actions)` with `margin-inline-start: auto`;
  - on the base, the entry contains none of the three statements (the check fails before the change).

### TC-002 — The old wording is gone from technical.md

- Covers: FR-RESPONSIVE-029
- Level: repository check (file content)
- Given the change applied on top of the base
- When `grep -nF -e 'Cells do not grow' -e "the free space left on a line is always smaller than the next cell's width" doc/specifications/responsive-layout/technical.md` runs
- Then nothing matches
- Corner cases:
  - the check is case-sensitive, so the new "Value cells do not grow" does not match `Cells do not grow`; a case-insensitive search for `cells do not grow` matches only that new sentence in the entry;
  - the old wording elsewhere in the repository (the FR-RESPONSIVE-029 definition and the issue #200 Traceability bullet in `functional.md`, `doc/todo/issue-165-r1-02.md` and this change folder, which record the problem) is out of this check;
  - on the base, line 117 matches both texts (the check fails before the change).

### TC-003 — Nothing else changes in the specifications, change records and todo files

- Covers: FR-RESPONSIVE-030, FR-RESPONSIVE-028
- Level: repository check (`git diff`)
- Given the change applied on top of the base
- When these commands run:
  - `git diff --name-status <base> -- . ':!doc'`;
  - `git diff <base> -- doc/specifications/responsive-layout/functional.md doc/specifications/responsive-layout/technical.md`;
  - `git diff --name-status <base> -- doc/changes/*issue-140-* doc/changes/*issue-165-* doc/changes/*issue-203-*`;
  - `git diff --name-status --diff-filter=DMRT <base> -- doc/todo/`
- Then:
  - the first command prints nothing (no file added, deleted, modified or renamed outside `doc/`);
  - in `functional.md`, no line of FR-RESPONSIVE-001 to FR-RESPONSIVE-027 is added, removed or modified; the only changes are the issue #200 section (FR-RESPONSIVE-028 to 030) and the issue #200 Traceability bullet;
  - in `technical.md`, the TS-RESPONSIVE-023 entry (from `- **TS-RESPONSIVE-023` to the next top-level entry) is unchanged, and no line outside the TS-RESPONSIVE-017 entry changes, apart from traceability lines that cite issue #200 or FR-RESPONSIVE-028 to 030;
  - the third and fourth commands print nothing: the change folders `2026-11-11-1142-issue-140-responsive-layout`, `2026-11-11-1513-issue-140-full-width-list`, `2026-11-19-2053-issue-165-use-all-space-to-show-item-list-when-the` and `2026-11-25-0828-issue-203-fix-020-updated-wording-in-responsive-la`, and every existing file under `doc/todo/` (including `issue-165-r1-02.md`), are unchanged; new follow-up records for issue #200 (`doc/todo/issue-200-r*.md`) are allowed
- Corner cases:
  - no `_test.go`, frontend source, `src/index.css`, `.spec.ts`, `go.mod`, `go.sum`, `package.json` or `API_SPECIFICATION.md` change;
  - no TS-RESPONSIVE or FR-RESPONSIVE ID is renumbered or removed (the sorted lists of IDs defined in both files equal the base lists plus FR-RESPONSIVE-028 to 030).

### TC-004 — The Go checks pass

- Covers: FR-RESPONSIVE-030
- Level: suite run (`go vet`, `go test`, unit and integration)
- Given the change applied on top of the base, with the existing tests unchanged
- When `go vet ./...` and `go test -count=1 ./...` run
- Then both pass, with the same test list as on the base
- Corner cases:
  - repeated runs give the same result (no cached results with `-count=1`).

### TC-005 — The application behaves as before

- Covers: FR-RESPONSIVE-030
- Level: e2e (existing Playwright suite) and deterministic gates
- Given the change applied on top of the base, with the existing tests unchanged
- When the other gates run: `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`, including `tests/responsive-layout.spec.ts`
- Then they pass as on the base; in particular the responsive-layout tests for card values side by side (TC-005 of issue #165), Status and actions kept on screen at narrow widths (QA-F001) and the desktop layout (TC-012 of issue #165) pass unchanged
- Corner cases:
  - narrow screens (320 to 1055 px) and the 1056 px desktop switch behave as on the base;
  - the user observes no change in the application (no API, database, setting or screen change: follows from TC-003).
