# Test cases: mark the header menu and application information subjects as implemented (issue #223)

Sources: functional specifications [app-header-menu](../../specifications/app-header-menu/functional.md) (FR-APP-MENU-008) and [app-information](../../specifications/app-information/functional.md) (FR-APP-INFO-015 to FR-APP-INFO-017). No use cases: see [no-use-cases.md](no-use-cases.md). The existing use cases UC-APP-MENU-001 and UC-APP-INFO-001 do not change; TC-006 checks that their behaviour still holds.

Conventions:

- **Base:** `origin/master` (or `master`), compared with the change applied on top of it.
- **The index:** the table of section 15 ("Subject specifications") of `doc/FUNCTIONAL_SPECIFICATIONS.md`.
- **The menu row** and **the information row:** the lines of the index whose first cell is `Header menu` and `Application information`.
- **Status cell:** the third cell of a row, trimmed of surrounding spaces.
- **The Status line:** line 3 of `doc/specifications/app-information/functional.md`.
- The change only edits documentation. FR-APP-INFO-017 allows no file change outside `doc/`, so no new Go or Playwright test code is written: the cases are repository checks (`grep`, `git diff`) and runs of the existing suites. Narrow screens, keyboard, empty results and screen states do not apply to the new requirements; no API, database or user interface changes.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-APP-MENU-008 | TC-001, TC-003 |
| FR-APP-INFO-015 | TC-002, TC-003 |
| FR-APP-INFO-016 | TC-004 |
| FR-APP-INFO-017 | TC-005, TC-006 |

| Use case | Test cases |
| --- | --- |
| none (UC-APP-MENU-001, UC-APP-INFO-001 unchanged, regression only) | TC-006 |

## Cases

### TC-001 — The Header menu row's Status cell is exactly `implemented`

- Covers: FR-APP-MENU-008
- Level: repository check (file content)
- Given the change applied on top of the base
- When `grep -n '^| Header menu |' doc/FUNCTIONAL_SPECIFICATIONS.md` runs
- Then exactly one line matches, inside section 15; it is `| Header menu | [app-header-menu](specifications/app-header-menu/functional.md) | implemented |`: its Status cell is `implemented` and nothing else
- Corner cases:
  - the row still links to `specifications/app-header-menu/functional.md` and has three cells (same number of `|` separators as the neighbouring rows);
  - `### Header menu and Settings modal` (section 7 heading) is not a table row and is not matched;
  - on the base, the Status cell is `implemented; FR-APP-MENU-007 specified (issue #215)` (the check fails before the change).

### TC-002 — The Application information row's Status cell is exactly `implemented`

- Covers: FR-APP-INFO-015
- Level: repository check (file content)
- Given the change applied on top of the base
- When `grep -n '^| Application information |' doc/FUNCTIONAL_SPECIFICATIONS.md` runs
- Then exactly one line matches, inside section 15; it is `| Application information | [app-information](specifications/app-information/functional.md) | implemented |`: its Status cell is `implemented` and nothing else
- Corner cases:
  - the row still links to `specifications/app-information/functional.md` and has three cells;
  - the menu row stays directly above it (row order unchanged);
  - on the base, the Status cell is `specified (issue #115); FR-APP-INFO-014 specified (issue #215)` (the check fails before the change).

### TC-003 — The stale "specified" wording is gone from the index file

- Covers: FR-APP-MENU-008, FR-APP-INFO-015
- Level: repository check (file content)
- Given the change applied on top of the base
- When these commands run on `doc/FUNCTIONAL_SPECIFICATIONS.md`:
  - `grep -n "FR-APP-MENU-007 specified"`;
  - `grep -n "specified (issue #115)"`;
  - `grep -n "FR-APP-INFO-014 specified"`
- Then each prints nothing
- Corner cases:
  - the same wording in the definition lines of FR-APP-MENU-008 and FR-APP-INFO-015 (which quote it) and in the issue #223 change folder and `doc/todo/issue-215-r1-03.md` (which record the problem) is out of this check;
  - on the base, each command prints one line (the check fails before the change).

### TC-004 — The application-information Status line says implemented, not `ready`

- Covers: FR-APP-INFO-016
- Level: repository check (file content)
- Given the change applied on top of the base
- When `sed -n 3p doc/specifications/app-information/functional.md` runs
- Then the line starts with `Status: implemented (issue #115); FR-APP-INFO-014 implemented (issue #215); FR-APP-INFO-015–017 implemented (issue #223` and does not contain `ready`
- Corner cases:
  - the line is the only `Status:` line of the file (`grep -c '^Status:'` prints `1`), and lines 1, 2 and 4 (title, blank, `Owner:`) are unchanged from the base;
  - the line keeps its trailing `\` hard line break, so `Owner:` and `User decision/reference:` still render on separate lines;
  - the range uses an en dash (`015–017`) as the requirement states; a hyphen does not satisfy the prefix check;
  - on the base, line 3 is `Status: ready\` (the check fails before the change).

### TC-005 — Nothing else changes in the repository, the requirements, the index, earlier change records or todo files

- Covers: FR-APP-INFO-017
- Level: repository check (`git diff`)
- Given the change applied on top of the base
- When these commands run:
  - `git diff --name-status <base> -- . ':!doc'`;
  - `git diff <base> -- doc/FUNCTIONAL_SPECIFICATIONS.md`;
  - `git diff <base> -- doc/specifications/app-information/functional.md doc/specifications/app-header-menu/functional.md`;
  - `git diff --name-status <base> -- doc/changes ':!doc/changes/2026-11-28-0613-issue-223-*' doc/todo/`
- Then:
  - the first command prints nothing (no file added, deleted, modified or renamed outside `doc/`);
  - in `doc/FUNCTIONAL_SPECIFICATIONS.md`, only the menu row and the information row change (numstat `2 2`); every other row of the index and every other line of the file is unchanged;
  - in the two subject files, no line of FR-APP-INFO-001 to FR-APP-INFO-014 or FR-APP-MENU-001 to FR-APP-MENU-007 is added, removed or modified; the only changes are the Status lines, the new rows FR-APP-INFO-015 to 017 and FR-APP-MENU-008, and the new issue #223 Traceability bullets;
  - the fourth command prints nothing: the change folders of earlier issues (including `2026-11-10-0541-issue-115-app-information` and `2026-11-23-0804-issue-215-tests-fail-on-master-blocks-204`) and every file under `doc/todo/` (including `issue-215-r1-03.md`) are unchanged
- Corner cases:
  - no `_test.go`, Go source, `src/` file (`AppMenu.tsx`, `InformationModal.tsx`), `.spec.ts`, `go.mod`, `go.sum`, `package.json` or `API_SPECIFICATION.md` change;
  - the use case files under `doc/use-cases/` for UC-APP-MENU-001 and UC-APP-INFO-001 are unchanged.

### TC-006 — The application behaves as before and the gates pass

- Covers: FR-APP-INFO-017; regression of UC-APP-MENU-001 and UC-APP-INFO-001
- Level: suite run (deterministic gates), including the existing e2e header-menu and application-information suites
- Given the change applied on top of the base, with the existing tests unchanged
- When `go vet ./...` and `go test ./...` run, then the other gates (`node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`, including `tests/app-information.spec.ts`, `tests/leboncoin-session-settings.spec.ts` and `tests/responsive-layout.spec.ts`)
- Then every gate passes with the same test list as on the base; the header-menu and Information-modal Playwright tests (menu keyboard use and Escape of FR-APP-MENU-007, modal focus trap of FR-APP-INFO-014, about 400 px width) pass as on the base
- Corner cases:
  - the results are the same without cached results (`go test -count=1 ./...`);
  - the operator observes no change in the application (no API, database, setting or screen change: follows from TC-005).
