# Issue #223 Mark the header menu and application information subjects as implemented in the subject index

**Problem:** In `doc/FUNCTIONAL_SPECIFICATIONS.md` lines 324-325, the subject index lists the header menu as "implemented; FR-APP-MENU-007 specified (issue #215)" and application information as "specified (issue #115); FR-APP-INFO-014 specified (issue #215)". Both subjects are implemented on master: `src/components/AppMenu.tsx` and `src/components/InformationModal.tsx`, both covered by e2e tests.

**Suggested work:** Set both rows to `implemented`. Drop the stale "specified (issue #115)". Check whether the `Status:` line in `doc/specifications/app-information/functional.md` should also move from `ready` to implemented.

From review comment R1-03 on #221 (`doc/todo/issue-215-r1-03.md`).

## Comments by trusted authors

