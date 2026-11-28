# No new automated tests (issue #223)

The change only edits documentation under `doc/`: two Status cells of the subject index in `doc/FUNCTIONAL_SPECIFICATIONS.md` and the Status line of `doc/specifications/app-information/functional.md`. FR-APP-INFO-017 forbids any file change outside `doc/`, so no `_test.go` or `tests/*.spec.ts` file may be added or changed, and no documentation edit could make a test go red. The technical specification ([app-information/technical.md](../../specifications/app-information/technical.md#tests), section "Issue #223") asks this stage for this file instead of tests.

The documentation edits were already made by the functional specification stage (commit `068c29d`). Where each test case is proved:

| Test case | Proved by | Result at this stage |
| --- | --- | --- |
| TC-001 | `grep -n '^\| Header menu \|' doc/FUNCTIONAL_SPECIFICATIONS.md` prints one line inside section 15, equal to `\| Header menu \| [app-header-menu](specifications/app-header-menu/functional.md) \| implemented \|`. | holds (line 341) |
| TC-002 | `grep -n '^\| Application information \|' doc/FUNCTIONAL_SPECIFICATIONS.md` prints one line, equal to `\| Application information \| [app-information](specifications/app-information/functional.md) \| implemented \|`, directly after the TC-001 line. | holds (line 342) |
| TC-003 | `grep -n` of `FR-APP-MENU-007 specified`, `specified (issue #115)` and `FR-APP-INFO-014 specified` on `doc/FUNCTIONAL_SPECIFICATIONS.md` each print nothing. | holds |
| TC-004 | `sed -n 3p doc/specifications/app-information/functional.md` starts with `Status: implemented (issue #115); FR-APP-INFO-014 implemented (issue #215); FR-APP-INFO-015–017 implemented (issue #223`, contains no `ready` and keeps the trailing `\`; `grep -c '^Status:'` on the file prints `1`. | holds |
| TC-005 | `git diff --name-status master -- . ':!doc'` prints nothing; `git diff --numstat master -- doc/FUNCTIONAL_SPECIFICATIONS.md` is `2 2`; `git diff master` of the two subject `functional.md` files shows only the Status lines, the new requirement rows and the issue #223 Traceability bullets; `git diff --name-status master -- doc/changes ':!doc/changes/2026-11-28-0613-issue-223-*' doc/todo/` prints nothing. | holds |
| TC-006 | Existing, unchanged suites through the deterministic gates: `go vet ./...`, `go test -count=1 ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test` (including `tests/app-information.spec.ts`, `tests/leboncoin-session-settings.spec.ts` and `tests/responsive-layout.spec.ts`). | run by the gates and the development stage |
