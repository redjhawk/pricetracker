# Test cases: fix "020 updated" wording in responsive-layout references (issue #203)

Sources: functional specification [responsive-layout](../../specifications/responsive-layout/functional.md) (FR-RESPONSIVE-025 to FR-RESPONSIVE-027). No use cases: see [no-use-cases.md](no-use-cases.md). The responsive-layout specification defines no `UC-` IDs, and its behaviour does not change.

Conventions:

- **Base:** `origin/master` (or `master`), compared with the change applied on top of it.
- **The row:** the line of `doc/FUNCTIONAL_SPECIFICATIONS.md` whose first cell is `Responsive layout`.
- **The bullet:** the line of the Traceability section of `doc/specifications/responsive-layout/functional.md` that starts `- Issue #165 (owner):`.
- **The old wording:** the text `016/017/020`.
- The change only edits documentation. FR-RESPONSIVE-027 allows no file change outside `doc/`, so no new Go or Playwright test code is written: the cases are repository checks (`grep`, `git diff`) and runs of the existing suites. Narrow screens, keyboard, empty results and screen states do not apply; no API, database or user interface changes.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-RESPONSIVE-025 | TC-001, TC-003 |
| FR-RESPONSIVE-026 | TC-002, TC-003 |
| FR-RESPONSIVE-027 | TC-004, TC-005 |

## Cases

### TC-001 — The Responsive layout row says "016/017 updated (issue #165)"

- Covers: FR-RESPONSIVE-025
- Level: repository check (file content)
- Given the change applied on top of the base
- When `grep -n '^| Responsive layout |' doc/FUNCTIONAL_SPECIFICATIONS.md` runs
- Then exactly one line matches; it contains `FR-RESPONSIVE-018 to 024 specified, 016/017 updated (issue #165)` and does not contain the old wording
- Corner cases:
  - the row still starts with `specified (issue #140)` and still links to `specifications/responsive-layout/functional.md` (only the issue #165 wording changes, plus the issue #203 entry);
  - the row is still a valid three-cell table row (same number of `|` separators as the neighbouring rows);
  - on the base, the row contains `016/017/020 updated (issue #165)` (the check fails before the change).

### TC-002 — The issue #165 Traceability bullet says "FR-RESPONSIVE-016/017 updated"

- Covers: FR-RESPONSIVE-026
- Level: repository check (file content)
- Given the change applied on top of the base
- When `grep -n '^- Issue #165 (owner):' doc/specifications/responsive-layout/functional.md` runs
- Then exactly one line matches; owner answer (4) in it reads `(4) “all screen should use available space” → FR-RESPONSIVE-022, 023, FR-RESPONSIVE-016/017 updated.`; the line does not contain the old wording
- Corner cases:
  - the rest of the bullet (answers (1) to (3), the owner quotes and the link to the issue #165 change folder) is unchanged from the base: `git diff <base> -- doc/specifications/responsive-layout/functional.md` shows, for this bullet, only `/020` removed;
  - on the base, the bullet contains `FR-RESPONSIVE-016/017/020 updated` (the check fails before the change).

### TC-003 — The old wording is gone from both files except in the requirement lines

- Covers: FR-RESPONSIVE-026, FR-RESPONSIVE-025
- Level: repository check (file content)
- Given the change applied on top of the base
- When `grep -n '016/017/020' doc/FUNCTIONAL_SPECIFICATIONS.md doc/specifications/responsive-layout/functional.md` runs
- Then the only matching lines are the definition lines of FR-RESPONSIVE-025 and FR-RESPONSIVE-026 in `functional.md` (they quote the old wording in their checks); `doc/FUNCTIONAL_SPECIFICATIONS.md` has no match
- Corner cases:
  - the issue #203 Traceability bullet and the issue #203 entry of the row describe the fix without containing the old wording;
  - the old wording elsewhere in the repository (the issue #203 change folder and `doc/todo/issue-165-r1-05.md`, which record the problem) is out of this check.

### TC-004 — Nothing else changes in the specification, change records and todo files

- Covers: FR-RESPONSIVE-027
- Level: repository check (`git diff`)
- Given the change applied on top of the base
- When these commands run:
  - `git diff --name-status <base> -- . ':!doc'`;
  - `git diff <base> -- doc/specifications/responsive-layout/functional.md`;
  - `git diff --name-status <base> -- doc/changes/*issue-140-* doc/changes/*issue-165-* doc/todo/`;
  - `git diff --name-status <base> -- doc/`
- Then:
  - the first command prints nothing (no file added, deleted, modified or renamed outside `doc/`);
  - in `functional.md`, no line of the requirement tables FR-RESPONSIVE-001 to FR-RESPONSIVE-024 is added, removed or modified; the only changes are the issue #203 section (FR-RESPONSIVE-025 to 027), the `/020` removal in the issue #165 bullet and the new issue #203 Traceability bullet;
  - the third command prints nothing: the change folders `2026-11-11-1142-issue-140-responsive-layout`, `2026-11-11-1513-issue-140-full-width-list` and `2026-11-19-2053-issue-165-use-all-space-to-show-item-list-when-the`, and every file under `doc/todo/` (including `issue-165-r1-05.md`), are unchanged;
  - the last command lists only `doc/FUNCTIONAL_SPECIFICATIONS.md`, `doc/specifications/responsive-layout/functional.md` and files of the issue #203 change folder
- Corner cases:
  - no `_test.go`, frontend source, `.spec.ts`, `go.mod`, `go.sum`, `package.json` or `API_SPECIFICATION.md` change;
  - `doc/specifications/responsive-layout/technical.md` is unchanged;
  - in `doc/FUNCTIONAL_SPECIFICATIONS.md`, only the Responsive layout row changes (numstat `1 1`).

### TC-005 — The application behaves as before and the gates pass

- Covers: FR-RESPONSIVE-027
- Level: suite run (deterministic gates), including the existing e2e responsive-layout suite
- Given the change applied on top of the base, with the existing tests unchanged
- When `go vet ./...` and `go test ./...` run, then the other gates (`node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`, including `tests/responsive-layout.spec.ts`)
- Then `go vet ./...` and `go test ./...` pass, and so do the other gates, with the same test list as on the base; the responsive-layout Playwright tests pass as on the base
- Corner cases:
  - the results are the same without cached results (`go test -count=1 ./...`);
  - the user observes no change in the application (no API, database, setting or screen change: follows from TC-004).
