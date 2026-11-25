# Technical specification: mark login attempt limits as implemented in the specification index (issue #284)

Status: ready (issue #284, maintenance: no behaviour change)
Functional specification: [login-attempt-limits/functional.md](../../specifications/login-attempt-limits/functional.md), section "Specification index status (issue #284, maintenance)" (FR-LOGIN-LIM-012..014).
Test cases: [testcases.md](testcases.md) (TC-001..TC-004). Use cases: none ([no-use-cases.md](no-use-cases.md)).

## Where this specification lives

FR-LOGIN-LIM-014 requires `doc/specifications/login-attempt-limits/technical.md` to stay unchanged, and TC-003 checks that its diff is empty. This change's technical specification is therefore this file in the change folder, not an update of the subject's `technical.md`. The subject's `technical.md` (TS-LOGIN-LIM-001..) still exists and still describes the delivered implementation of FR-LOGIN-LIM-001..011.

## Requirement mapping

| Technical ID | Functional IDs | Technical solution | Files |
| --- | --- | --- | --- |
| TS-LOGIN-LIM-284-001 | FR-LOGIN-LIM-012 | The status cell of the Login attempt limits row begins with `implemented (issue #266)` and the line does not contain `specified (issue #266)`; the subject name, link and four `\|` separators stay. | `doc/FUNCTIONAL_SPECIFICATIONS.md` |
| TS-LOGIN-LIM-284-002 | FR-LOGIN-LIM-013 | Only that one line of the index changes (numstat `1 1` against the base); no other row, heading or text is touched. | `doc/FUNCTIONAL_SPECIFICATIONS.md` |
| TS-LOGIN-LIM-284-003 | FR-LOGIN-LIM-014 | No file outside `doc/` changes; under `doc/`, only the index, `doc/specifications/login-attempt-limits/functional.md` (status line and issue #284 section) and this change folder change. No new test code: proved by repository checks and the existing suites. | none |

The `TS-LOGIN-LIM-284-NNN` prefix keeps these IDs apart from TS-LOGIN-LIM-001.. of the subject's `technical.md`, which must not change.

## Frontend

Not affected: no screen, component, state or text of the application changes (FR-LOGIN-LIM-014).

## Backend

Not affected: no service, handler, validation, persistence, migration or concurrency change. The login attempt limits stay as implemented by issue #266.

## API

Not affected: no request, response or error changes; `API_SPECIFICATION.md` is unchanged.

## Documentation changes

The functional specification stage already changed the index row on this branch. Its status cell now reads:

`implemented (issue #266); FR-LOGIN-LIM-012–014 specified (issue #284, maintenance: this row says that FR-LOGIN-LIM-001–011 are implemented; no behaviour change)`

This already satisfies TS-LOGIN-LIM-284-001 and TS-LOGIN-LIM-284-002. When the work is delivered, the development stage:

1. In `doc/FUNCTIONAL_SPECIFICATIONS.md`, changes `FR-LOGIN-LIM-012–014 specified (issue #284` to `FR-LOGIN-LIM-012–014 implemented (issue #284` in the same row, and changes nothing else. The cell still begins with `implemented (issue #266)` and the diff against the base stays one line.
2. In `doc/specifications/login-attempt-limits/functional.md`, changes the status line from `FR-LOGIN-LIM-012–014 specified (issue #284, …)` to `FR-LOGIN-LIM-012–014 implemented (issue #284, …)`. The FR-LOGIN-LIM-001..011 table and the other sections stay unchanged.
3. Does not edit `doc/specifications/login-attempt-limits/technical.md`, `API_SPECIFICATION.md`, `doc/use-cases/login-attempt-limits/`, `doc/changes/2026-11-23-2151-issue-266-login-limit-failed-attempts-by-client-ad/` or `doc/todo/` (including `doc/todo/issue-266-r1-01.md`).

Files to change: `doc/FUNCTIONAL_SPECIFICATIONS.md` and `doc/specifications/login-attempt-limits/functional.md`, plus this change folder.

## Tests

FR-LOGIN-LIM-014 forbids any change outside `doc/`, so no `_test.go`, `.spec.ts` or other test file is added or changed. Also, the index row already says implemented on this branch, so no test could fail before development and pass after it. The tests stage writes `no-tests.md` in this change folder, listing how each test case is proved:

| Test case | Proved by |
| --- | --- |
| TC-001 | `grep -c '^| Login attempt limits |' doc/FUNCTIONAL_SPECIFICATIONS.md` is `1`; the status cell of that line (third cell) begins with `implemented (issue #266)`; `grep '^| Login attempt limits |' doc/FUNCTIONAL_SPECIFICATIONS.md \| grep -c 'specified (issue #266)'` is `0`; the line keeps the link `specifications/login-attempt-limits/functional.md` and four `\|`. |
| TC-002 | `git diff --numstat master -- doc/FUNCTIONAL_SPECIFICATIONS.md` is `1 1`; `git diff -U0` shows only the base row removed and the TC-001 row added; the count of `^\| ` lines and of other `specified (issue #` lines is the same as on `master`; the Multiple users and login row is unchanged. |
| TC-003 | `git diff --name-status master -- . ':!doc'` prints nothing; `git diff master -- doc/specifications/login-attempt-limits/technical.md doc/changes/2026-11-23-2151-issue-266-login-limit-failed-attempts-by-client-ad doc/todo doc/use-cases/login-attempt-limits` prints nothing; `git diff -U0 master -- doc/specifications/login-attempt-limits/functional.md` changes only the status line and adds the issue #284 section. |
| TC-004 | Existing, unchanged tests `TestLoginLimits_TC001`..`TestLoginLimits_TC013_*` in `internal/service/login_attempt_limits_test.go` and `internal/httpapi/login_attempt_limits_test.go`, run with `go test -count=1 -run 'TestLoginLimits_' ./internal/service/ ./internal/httpapi/`; then the gates `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`. |

## Refactoring

None.

## Risks

- A wider edit of the index (for example fixing the other `specified (issue #NNN)` rows that the issue mentions as optional) would break FR-LOGIN-LIM-013 and TC-002. Those rows are out of scope.
- Editing the subject's `technical.md` to record this change would break FR-LOGIN-LIM-014 and TC-003. That is why this specification is in the change folder.

## Manual follow-ups

None.
