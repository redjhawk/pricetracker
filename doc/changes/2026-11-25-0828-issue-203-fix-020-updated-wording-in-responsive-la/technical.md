# Technical specification: fix "020 updated" wording in responsive-layout references (issue #203)

Status: ready
Functional specification: [responsive-layout](../../specifications/responsive-layout/functional.md), FR-RESPONSIVE-025 to 027 (issue #203, maintenance, no behaviour change).
Test cases: [testcases.md](testcases.md). Use cases: none, see [no-use-cases.md](no-use-cases.md).

## Where this specification lives

The subject's technical specification `doc/specifications/responsive-layout/technical.md` already exists. It is **not** updated by this change. FR-RESPONSIVE-027 and TC-004 require that file to stay unchanged, and they require the `doc/` diff to list only `doc/FUNCTIONAL_SPECIFICATIONS.md`, `doc/specifications/responsive-layout/functional.md` and files of this change folder. So the technical specification for issue #203 is this file, in the change folder. The factory's techspec gate only checks that the subject's `technical.md` exists, which it does.

Technical IDs TS-RESPONSIVE-024 to 026 are defined here. The subject file ends at TS-RESPONSIVE-023. A later change that edits the subject's `technical.md` must start at TS-RESPONSIVE-027 and may copy the mapping below into it.

## Current state (inspected)

The functional specification stage already made both wording fixes on the branch:

- `doc/FUNCTIONAL_SPECIFICATIONS.md`, the Responsive layout row: `FR-RESPONSIVE-018 to 024 specified, 016/017 updated (issue #165); FR-RESPONSIVE-025–027 specified (issue #203, …)`. The numstat against `master` is `1 1`.
- `doc/specifications/responsive-layout/functional.md`: owner answer (4) in the issue #165 Traceability bullet ends `FR-RESPONSIVE-016/017 updated.`. The bullet's only change is the removal of `/020`. The issue #203 section (FR-RESPONSIVE-025 to 027) and the issue #203 Traceability bullet are added.
- `grep -n '016/017/020'` on both files finds only the FR-RESPONSIVE-025 and 026 definition lines in `functional.md`.
- `git diff --name-status master -- . ':!doc'` is empty.

## Requirement mapping

| Functional ID | Technical solution | Files | Verification |
| --- | --- | --- | --- |
| FR-RESPONSIVE-025 | TS-RESPONSIVE-024 | `doc/FUNCTIONAL_SPECIFICATIONS.md` (done) | TC-001, TC-003 |
| FR-RESPONSIVE-026 | TS-RESPONSIVE-025 | `doc/specifications/responsive-layout/functional.md` (done) | TC-002, TC-003 |
| FR-RESPONSIVE-027 | TS-RESPONSIVE-026 | none | TC-004, TC-005 |

- **TS-RESPONSIVE-024 Index row wording.** In the Responsive layout row of `doc/FUNCTIONAL_SPECIFICATIONS.md`, the issue #165 entry reads `FR-RESPONSIVE-018 to 024 specified, 016/017 updated (issue #165)`. The row keeps its three cells, the `specified (issue #140)` start and the link to `specifications/responsive-layout/functional.md`, and gets the issue #203 entry. Nothing else in the file changes.
- **TS-RESPONSIVE-025 Traceability wording.** In `functional.md`, owner answer (4) of the issue #165 bullet reads `→ FR-RESPONSIVE-022, 023, FR-RESPONSIVE-016/017 updated.`. The rest of the bullet is byte-identical to `master`. The issue #203 section and the issue #203 Traceability bullet describe the fix without the string `016/017/020`, except the FR-RESPONSIVE-025 and 026 definitions, which quote it as their check.
- **TS-RESPONSIVE-026 No other change.** The rows of FR-RESPONSIVE-001 to 024, the issue #140 and #165 change folders, `doc/todo/` (including `issue-165-r1-05.md`), `doc/specifications/responsive-layout/technical.md`, `API_SPECIFICATION.md`, and every file outside `doc/` are unchanged. The existing Go and Playwright suites run unchanged and pass.

## Frontend

Not affected. No component, style, state or test file changes. The responsive layout behaviour of FR-RESPONSIVE-001 to 024 stays as specified in the subject's `technical.md`.

## Backend

Not affected. No handler, service, validation, persistence, migration or concurrency change.

## API

Not affected. No endpoint, field, status or error changes. `API_SPECIFICATION.md` is unchanged.

## Files to change

None beyond those already changed on the branch: `doc/FUNCTIONAL_SPECIFICATIONS.md` and `doc/specifications/responsive-layout/functional.md`, plus files of this change folder. The developer stage must not edit any other file. In particular it must not edit the subject's `technical.md` or any file outside `doc/`. If a check of TC-001 to TC-004 fails, the only allowed fix is in those two files.

## Tests

FR-RESPONSIVE-027 forbids any change outside `doc/`, so no test code is written. Every test case is a repository check or a run of the existing suites. The test stage records this in `no-tests.md` in this change folder, so the TDD red gate is skipped. A red phase is not possible anyway, because the wording is already fixed.

| Test case | How it is run | TS IDs |
| --- | --- | --- |
| TC-001 | `grep -n '^\| Responsive layout \|' doc/FUNCTIONAL_SPECIFICATIONS.md`: one line, which contains `FR-RESPONSIVE-018 to 024 specified, 016/017 updated (issue #165)` and not `016/017/020` | 024 |
| TC-002 | `grep -n '^- Issue #165 (owner):' doc/specifications/responsive-layout/functional.md`: one line, with answer (4) as in TS-RESPONSIVE-025; `git diff master -- doc/specifications/responsive-layout/functional.md` shows only `/020` removed from that bullet | 025 |
| TC-003 | `grep -n '016/017/020' doc/FUNCTIONAL_SPECIFICATIONS.md doc/specifications/responsive-layout/functional.md`: only the FR-RESPONSIVE-025 and 026 lines | 024, 025 |
| TC-004 | The four `git diff` commands of TC-004 against `master` | 026 |
| TC-005 | The existing gates: `go vet ./...`, `go test -count=1 ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test` (including `tests/responsive-layout.spec.ts`), unchanged and passing | 026 |

## Refactoring

None.

## Risks

- Technical IDs recorded outside the subject's `technical.md` could be reused by mistake. The note in "Where this specification lives" and a repository search for `TS-RESPONSIVE-02` prevent that.

## Manual follow-ups

None.
