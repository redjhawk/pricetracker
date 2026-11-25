# Test cases: close issue #238 as completed after PR #255 merges (issue #256)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-065 to FR-TEST-HYG-067, which settle FR-TEST-HYG-019). No use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **Closing comment:** "Done by PR #236; its follow-up record doc/todo/issue-233-r1-03.md is removed by the pull request of #241." (FR-TEST-HYG-065).
- **State before the change:** the state and reason of issue #238 when the change starts, read from `gh issue view 238 --json state,stateReason,comments` and the issue's timeline (`gh api repos/{owner}/{repo}/issues/238/timeline`). Only one of the three branches of FR-TEST-HYG-065 happens in the real run; TC-002 checks the branch that applies and the corner cases state what the other branches must give.
- The change is repository maintenance: it changes the state of GitHub issue #238 and one status line entry. No production code, API, database or user interface changes, so no new Go or Playwright test code is written; the cases are repository checks, GitHub state checks and runs of the existing suites. Narrow screens, keyboard and screen states do not apply.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-TEST-HYG-065 | TC-001, TC-002 |
| FR-TEST-HYG-066 | TC-003 |
| FR-TEST-HYG-067 | TC-004, TC-005 |

## Cases

### TC-001 — Issue #238 is closed as completed

- Covers: FR-TEST-HYG-065 (and FR-TEST-HYG-019)
- Level: repository check (GitHub issue state, `gh issue view 238 --json state,stateReason`)
- Given the change applied
- When the state of issue #238 is read
- Then `state` is `CLOSED` and `stateReason` is `COMPLETED`
- Corner cases:
  - `stateReason` `NOT_PLANNED` or empty fails the case, since PR #236 did the work;
  - issue #238 is not deleted, transferred or locked, so it stays readable;
  - reading the state twice gives the same result.

### TC-002 — Issue #238 gets the closing comment only when the change closes it

- Covers: FR-TEST-HYG-065
- Level: repository check (GitHub issue comments and timeline)
- Given the state of issue #238 before the change
- When the comments and the timeline events of issue #238 added since the change started are listed
- Then:
  - if issue #238 was open: one `closed` event with reason completed and exactly one new comment whose body is the closing comment;
  - if it was already closed as completed (for example by the merge of PR #280): no `reopened` or `closed` event and no new comment;
  - if it was closed with another reason (for example "not planned"): one `reopened` event, then one `closed` event with reason completed, and exactly one new comment whose body is the closing comment
- Corner cases:
  - the comment text matches exactly, including `doc/todo/issue-233-r1-03.md` and `#241`;
  - repeated action: running the close step again on an issue already closed as completed adds no comment and no event, so the comment is never posted twice;
  - an issue already closed as completed is never reopened, even briefly.

### TC-003 — The status line gives FR-TEST-HYG-019 as implemented

- Covers: FR-TEST-HYG-066
- Level: repository check (`doc/specifications/test-suite-hygiene/functional.md` and `git diff master`)
- Given the change applied and TC-001 passing
- When the `Status:` line of the test-suite hygiene functional specification is read and compared with `master`
- Then it gives FR-TEST-HYG-019 as implemented, does not contain "FR-TEST-HYG-019 closed by the merge" or "closed by the merge of the issue #238 change", and every other entry keeps its requirement IDs, status and issue number
- Corner cases:
  - FR-TEST-HYG-019 appears in exactly one entry of the status line (no leftover "closed by the merge" entry next to the new one);
  - the entry "FR-TEST-HYG-027–029 implemented (issue #238 …)" still names FR-TEST-HYG-027–029 as implemented by issue #238;
  - the text of FR-TEST-HYG-018–021 and FR-TEST-HYG-027–029 is unchanged;
  - the line is not changed if TC-001 fails (FR-TEST-HYG-066 holds only once FR-TEST-HYG-065 does).

### TC-004 — The diff stays in `doc/`, outside `doc/todo/`, and the gates pass

- Covers: FR-TEST-HYG-067
- Level: repository check and suite run (deterministic gates: `go vet ./...`, `go test ./...`, and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-status master` is listed and the gates run
- Then every changed path starts with `doc/`, none starts with `doc/todo/`, and `go vet ./...` and `go test ./...` pass
- Corner cases:
  - `doc/todo/issue-241-r1-01.md` (this issue's own record) and `doc/todo/issue-233-r1-03.md`'s absence are unchanged;
  - nothing under `scripts/factory/` changes (the optional extra `Closes #N` lines are excluded);
  - no production Go code, `_test.go`, `.spec.ts`, frontend source, `go.mod`, `go.sum`, `package.json` or `API_SPECIFICATION.md` changes, so the operator observes no change in the application;
  - the issue #241 and #238 change folders and technical specifications are unchanged;
  - the gates give the same result with `-count=1` and when run twice.

### TC-005 — No GitHub issue other than #238 and #256 changes state

- Covers: FR-TEST-HYG-067
- Level: repository check (GitHub issue state, `gh issue list --state all --json number,state,stateReason,updatedAt`)
- Given the states of all repository issues before the change
- When the states are read again after the change
- Then only issues #238 and #256 may differ; every other issue, in particular #233 and #241, has the same `state` and `stateReason`
- Corner cases:
  - issue #241 stays closed as it was after PR #255;
  - no issue is opened by the change;
  - the change's pull request body closes only #256.
