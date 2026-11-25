# Test cases: mark login attempt limits as implemented in the specification index (issue #284)

Sources: functional specification [login-attempt-limits](../../specifications/login-attempt-limits/functional.md) (FR-LOGIN-LIM-012 to FR-LOGIN-LIM-014). No new use cases: see [no-use-cases.md](no-use-cases.md); the unchanged behaviour is that of UC-LOGIN-LIM-001 to UC-LOGIN-LIM-003.

Conventions:

- **Base:** `origin/master` (or `master`), compared with the change applied on top of it.
- **Index:** `doc/FUNCTIONAL_SPECIFICATIONS.md`, Subject specifications table.
- **The row:** the index line starting `| Login attempt limits | [login-attempt-limits](specifications/login-attempt-limits/functional.md) |`.
- **Status cell:** the third cell of a row, between the third and fourth `|`, with surrounding spaces trimmed.
- **Existing tests:** `TestLoginLimits_TC001` to `TestLoginLimits_TC013_*` (`internal/service/login_attempt_limits_test.go` and `internal/httpapi/login_attempt_limits_test.go`, issue #266).
- The change only edits documentation. FR-LOGIN-LIM-014 allows no file change outside `doc/`, so no new Go or Playwright test code is written: the cases are repository checks (`grep`, `git diff`) and runs of the existing suites. Narrow screens, keyboard, empty results and screen states do not apply; no API, database or user interface changes.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-LOGIN-LIM-012 | TC-001 |
| FR-LOGIN-LIM-013 | TC-002 |
| FR-LOGIN-LIM-014 | TC-003, TC-004 |

| Use case (unchanged behaviour) | Test cases |
| --- | --- |
| UC-LOGIN-LIM-001, UC-LOGIN-LIM-002, UC-LOGIN-LIM-003 | TC-004 |

## Cases

### TC-001 — The Login attempt limits row says implemented

- Covers: FR-LOGIN-LIM-012
- Level: repository check (index content)
- Given the change applied on top of the base
- When the lines of the index starting with `| Login attempt limits |` are read and the status cell of each is taken
- Then exactly one such line exists; its status cell begins with `implemented (issue #266)`; the line does not contain `specified (issue #266)`
- Corner cases:
  - the row keeps its subject name and its link to `specifications/login-attempt-limits/functional.md`, and still has four `|` separators (a valid table row);
  - text after `implemented (issue #266)` in the cell is allowed (for example the issue #284 note), but `specified (issue #266)` is not, anywhere in the line;
  - on the base, the status cell is `specified (issue #266)` (the check fails before the change).

### TC-002 — Only the Login attempt limits row of the index changes

- Covers: FR-LOGIN-LIM-013
- Level: repository check (`git diff`)
- Given the change applied on top of the base
- When `git diff --numstat <base> -- doc/FUNCTIONAL_SPECIFICATIONS.md` and `git diff -U0 <base> -- doc/FUNCTIONAL_SPECIFICATIONS.md` run
- Then the numstat is `1 1`; the only removed line is the base row (`... | specified (issue #266) |`) and the only added line is the row of TC-001
- Corner cases:
  - the Multiple users and login row, ending `FR-AUTH-005 superseded by FR-LOGIN-LIM-* (issue #266) |`, is byte-for-byte unchanged;
  - every other row whose status cell holds `specified (issue #NNN)` (for example Application information, Masked secrets in Settings, Amazon searches removal, Responsive layout) is unchanged, and so is the count of lines containing `specified (issue #` other than the Login attempt limits row;
  - the number of table lines of the index (`^| `) is the same as on the base (no row added, removed or moved);
  - text outside the table (headings, introduction) is unchanged.

### TC-003 — Nothing else changes in the specification, change records or todo files

- Covers: FR-LOGIN-LIM-014
- Level: repository check (`git diff`)
- Given the change applied on top of the base
- When `git diff --name-status <base> -- . ':!doc'`, `git diff <base> -- doc/specifications/login-attempt-limits/technical.md doc/changes/2026-11-23-2151-issue-266-login-limit-failed-attempts-by-client-ad doc/todo` and `git diff -U0 <base> -- doc/specifications/login-attempt-limits/functional.md` run
- Then the first two commands print nothing; in the `functional.md` diff, no line of the FR-LOGIN-LIM-001 to FR-LOGIN-LIM-011 table is removed or changed (only the status line and the new issue #284 section with FR-LOGIN-LIM-012–014 are added or changed)
- Corner cases:
  - no file is added, deleted or renamed outside `doc/` (no `_test.go`, Go source, frontend source, `.spec.ts`, `go.mod`, `go.sum`, `package.json` or `API_SPECIFICATION.md` change);
  - `doc/todo/issue-266-r1-01.md`, the review comment this issue comes from, still exists unchanged;
  - no file under `doc/use-cases/login-attempt-limits/` changes (the existing use cases stay valid);
  - the only files changed under `doc/` are the index, `doc/specifications/login-attempt-limits/functional.md` and the files of this change folder.

### TC-004 — Login attempt limits behave as before and the gates pass

- Covers: FR-LOGIN-LIM-014; UC-LOGIN-LIM-001, UC-LOGIN-LIM-002, UC-LOGIN-LIM-003 (regression)
- Level: integration (existing service and HTTP handler tests with real SQLite) and suite run (deterministic gates)
- Given the change applied on top of the base, with the existing tests unchanged
- When `go test -count=1 -run 'TestLoginLimits_' ./internal/service/ ./internal/httpapi/` runs, then `go vet ./...` and `go test ./...`, and the other gates (`node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`)
- Then every named test passes, as on the base: failures are counted per client address and username, a locked pair is refused with 429 without a password check while other clients still log in, and logins are refused as busy only when the maximum number of password checks are waiting; `go vet ./...` and `go test ./...` pass, and so do the other gates, with the same test list as on the base
- Corner cases:
  - the results are the same without cached results (`-count=1`) and when the gates run twice;
  - the user observes no change in the application (no API, database, setting or screen change: follows from TC-003).
