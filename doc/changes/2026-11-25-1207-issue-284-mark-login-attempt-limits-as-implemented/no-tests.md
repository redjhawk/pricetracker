# No new tests (issue #284)

This change only edits documentation: the status cell of the Login attempt limits row in `doc/FUNCTIONAL_SPECIFICATIONS.md` and the status line of `doc/specifications/login-attempt-limits/functional.md`. No automated test can be written for it:

- FR-LOGIN-LIM-014 (and TC-003) forbid any file change outside `doc/`, so no `_test.go`, `.spec.ts` or other test file may be added or changed.
- The functional specification stage already changed the index row on this branch, so no test could fail before development and pass after it (no TDD red is possible).
- No behaviour, API, database or screen changes: the existing `TestLoginLimits_*` tests already cover the unchanged behaviour.

The test cases are proved by repository checks and the existing suites, as listed in [technical.md](technical.md#tests):

| Test case | Proved by |
| --- | --- |
| TC-001 | `grep -c '^\| Login attempt limits \|' doc/FUNCTIONAL_SPECIFICATIONS.md` is `1`; the third cell of that line begins with `implemented (issue #266)`; the line does not contain `specified (issue #266)`; it keeps the link `specifications/login-attempt-limits/functional.md` and four `\|`. |
| TC-002 | `git diff --numstat master -- doc/FUNCTIONAL_SPECIFICATIONS.md` is `1 1`; `git diff -U0` shows only the base row removed and the TC-001 row added; the counts of `^\| ` lines and of other `specified (issue #` lines equal those on `master`; the Multiple users and login row is unchanged. |
| TC-003 | `git diff --name-status master -- . ':!doc'` prints nothing; `git diff master -- doc/specifications/login-attempt-limits/technical.md doc/changes/2026-11-23-2151-issue-266-login-limit-failed-attempts-by-client-ad doc/todo doc/use-cases/login-attempt-limits` prints nothing; `git diff -U0 master -- doc/specifications/login-attempt-limits/functional.md` changes only the status line and adds the issue #284 section. |
| TC-004 | Existing, unchanged `TestLoginLimits_TC001`..`TestLoginLimits_TC013_*` in `internal/service/login_attempt_limits_test.go` and `internal/httpapi/login_attempt_limits_test.go`, run with `go test -count=1 -run 'TestLoginLimits_' ./internal/service/ ./internal/httpapi/`; then the gates `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`. |

State at this stage: TC-001 to TC-003 checks pass on this branch, and `go test -count=1 -run 'TestLoginLimits_' ./internal/service/ ./internal/httpapi/` passes.
