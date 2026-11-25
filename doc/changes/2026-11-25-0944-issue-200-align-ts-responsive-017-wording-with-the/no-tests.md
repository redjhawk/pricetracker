# No automated tests for issue #200

This change only corrects documentation wording: in the TS-RESPONSIVE-017 entry of `doc/specifications/responsive-layout/technical.md`, the sentence that said no card cell grows becomes "Value cells do not grow; only the item cell grows into the space left on its line, and the actions cell is pushed to the line end (TS-RESPONSIVE-023)." No production code, API, database, user interface or runtime behaviour changes.

FR-RESPONSIVE-030 (checked by TC-003) allows no file change outside `doc/`. That rules out any new or changed `_test.go` or `tests/*.spec.ts` file. The technical specification ([technical.md](../../specifications/responsive-layout/technical.md), section "TS-RESPONSIVE-017 wording (issue #200, FR-RESPONSIVE-028 to 030)") says the change has no test code.

The test cases are run as repository checks and with the existing suites:

| Test case | Check |
| --- | --- |
| TC-001 | `awk '/^- \*\*TS-RESPONSIVE-017/{p=1} /^- \*\*TS-RESPONSIVE-018/{p=0} p' doc/specifications/responsive-layout/technical.md` prints an entry stating that value cells do not grow, that only the item cell grows into the space left on its line and that the actions cell is pushed to the line end, citing `TS-RESPONSIVE-023`. Exactly one line starts `- **TS-RESPONSIVE-017`. Within the TS-RESPONSIVE-017 entry, `git diff master` touches only that sentence (lines outside the entry are covered by TC-003). `src/index.css` still has `td:has(> .item-cell)` with `flex: 1 1 14rem` and `td:has(> .row-actions)` with `margin-inline-start: auto`. |
| TC-002 | `grep -nF -e 'Cells do not grow' -e "the free space left on a line is always smaller than the next cell's width" doc/specifications/responsive-layout/technical.md` matches nothing. |
| TC-003 | `git diff --name-status master -- . ':!doc'` and `git diff --name-status master -- doc/changes/*issue-140-* doc/changes/*issue-165-* doc/changes/*issue-203-*` and `git diff --name-status --diff-filter=DMRT master -- doc/todo/` print nothing (new follow-up records `doc/todo/issue-200-r*.md` are allowed). FR-RESPONSIVE-001 to 027 and the TS-RESPONSIVE-023 entry are unchanged, and no ID is renumbered or removed. |
| TC-004 | `go vet ./...` and `go test -count=1 ./...` pass with the same test list as on `master`. |
| TC-005 | The existing gates (`node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test` including `tests/responsive-layout.spec.ts`) pass unchanged. |

When this stage ran, the change was not yet applied: TC-001 and TC-002 failed as expected (line 117 of `technical.md` still holds both old texts), and the diff against `master` outside `doc/` was empty.
