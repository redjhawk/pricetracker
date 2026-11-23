# Test cases: close the finished TC-005 interruption-check todo (issue #241)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-018 to FR-TEST-HYG-021). No use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **Issue #233 TC-005:** `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile` in `internal/httpapi/amazon_browser_interrupted_test.go`, added by PR #236. It is not a case of this catalogue; the IDs below are this change's own.
- **Follow-up record:** `doc/todo/issue-233-r1-03.md` (`Status: todo`, links issue #238).
- The change is repository maintenance: it only deletes the follow-up record and closes issue #238. No production code, API, database or user interface changes, so no new Go or Playwright test code is written; the cases are repository checks and runs of the existing suites. Narrow screens, keyboard and screen states do not apply.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-TEST-HYG-018 | TC-001 |
| FR-TEST-HYG-019 | TC-002 |
| FR-TEST-HYG-020 | TC-003 |
| FR-TEST-HYG-021 | TC-001, TC-004 |

## Cases

### TC-001 — The finished follow-up record is deleted, and only that one

- Covers: FR-TEST-HYG-018, FR-TEST-HYG-021
- Level: repository check (file system and `git diff`)
- Given the change applied on top of `master`
- When `doc/todo/issue-233-r1-03.md` is looked up and `git diff --name-status master -- doc/todo` is listed
- Then the file does not exist, and the only entry under `doc/todo/` is `D doc/todo/issue-233-r1-03.md`
- Corner cases:
  - the file is deleted, not kept with `Status: done` (FR-TEST-HYG-018 requires it to no longer exist);
  - the other issue #233 records (`issue-233-r1-02.md`, `issue-233-r1-04.md`, `issue-233-r2-02.md`, `issue-233-r3-01.md`, `issue-233-r3-02.md`) still exist and are unchanged, including `issue-233-r3-01.md`, the review comment this issue comes from;
  - no file is added to `doc/todo/`;
  - no other file in the repository still refers to the deleted record as open work (a reference in history documents such as change folders is allowed).

### TC-002 — Issue #238 is closed as completed

- Covers: FR-TEST-HYG-019
- Level: repository check (GitHub issue state, `gh issue view 238 --json state,stateReason`)
- Given the change merged (or the issue closed by the change's pull request)
- When the state of issue #238 is read
- Then `state` is `CLOSED` and `stateReason` is `COMPLETED`
- Corner cases:
  - the reason is not `NOT_PLANNED`, since PR #236 did the work;
  - closing an issue that is already closed as completed leaves it closed as completed (repeated action);
  - issue #238 is not deleted or transferred, so it stays readable.

### TC-003 — Issue #233 TC-005 still detects a missed interruption point

- Covers: FR-TEST-HYG-020
- Level: repository check plus integration run (existing Go test with a real SQLite database and HTTP handlers)
- Given the change applied on top of `master`
- When `git diff master -- internal/httpapi/amazon_browser_interrupted_test.go` is listed and `go test -count=1 -run TestAmazonBrowserInterrupted_TC005 ./internal/httpapi/` runs
- Then the diff is empty; the test file still holds, after the wait on `api.fixture.flushed` and before `cancel()`, a non-blocking `select` on `stopped` that calls `t.Fatal("the check ended before the server stop: the interruption point was missed")`; and the test passes
- Corner cases:
  - the message is exactly the one of FR-TEST-HYG-020 (the draft wording in the follow-up record, "interruption point missed", is not the one required);
  - the `select` has a `default` branch, so it does not wait when the check is still running;
  - the test gives the same result without cached results (`-count=1`).

### TC-004 — Nothing outside `doc/` changes and the gates pass

- Covers: FR-TEST-HYG-021
- Level: repository check and suite run (deterministic gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`, and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-status master -- . ':!doc'` is listed and the gates run
- Then the diff outside `doc/` is empty; `go vet ./...` and `go test ./...` pass, and so do the other gates; the Playwright suite runs the same test list as on `master`
- Corner cases:
  - no production Go code, frontend source, `go.mod`, `go.sum`, `package.json` or `API_SPECIFICATION.md` changes, so the operator observes no change in the application;
  - no `_test.go` or `.spec.ts` file is added, changed or deleted;
  - the gates give the same result when run twice.
