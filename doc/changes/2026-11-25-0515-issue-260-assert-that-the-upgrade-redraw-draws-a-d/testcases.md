# Test cases: assert that the upgrade redraw draws a different time for each Amazon item (issue #260)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-078 to FR-TEST-HYG-081). There are no use cases: see [no-use-cases.md](no-use-cases.md). The behaviour the test proves is FR-AMZ-HCHK-005 of [Amazon human-like checks](../../specifications/amazon-human-checks/functional.md), which this change does not alter.

Conventions:

- **The test file:** `internal/service/amazon_human_checks_issue185_test.go`.
- **The redraw test:** `TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce`, with its items `B000001853`, `B000001854` and `B000001855`, its random seed 1851 for the first start and 1852 for the second.
- **The run:** `go test -count=1 -run '^TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce$' -v ./internal/service/`.
- **The issue #185 test cases:** `doc/changes/2026-11-23-1706-issue-185-tests-missing-from-168-amazon-human-like/testcases.md`.
- **Fault injection:** a temporary edit of `scheduleUnscheduledAmazonItems` in `internal/service/service.go`, made only for the run of a case, reverted afterwards and never committed. Every injected time is still drawn by `nextCheckAt`, so it stays in the evening window: the window check passes and only the new check can catch it.
- The change only touches test code and test documentation. No production code, API, database or user interface changes, and there is no user flow, so there are no e2e cases. The redraw test itself is integration level (real SQLite database, `service.New`, injected clock and random source). The IDs below are this change's own, not those in the test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-078 | TC-001, TC-002, TC-003 | — |
| FR-TEST-HYG-079 | TC-001, TC-004 | — |
| FR-TEST-HYG-080 | TC-005 | — |
| FR-TEST-HYG-081 | TC-006 | — |

## Cases

### TC-001: With its fixed seed, the redraw test passes and the three drawn times differ

- Covers: FR-TEST-HYG-078, FR-TEST-HYG-079
- Level: integration (Go, `internal/service`, real SQLite database, injected clock and random source)
- Given the change applied and no fault injection
- When the run executes
- Then the redraw test passes; its three items' next scheduled checks after the first start are in the evening window of 2030-03-05 and pairwise different
- Corner cases:
  - repeated runs (`-count=3`) give the same result, since the seed is fixed;
  - the test passes in `go test ./internal/service/...` with no extra flag or environment.

### TC-002: One time given to every item fails the test with a message naming two items and the time

- Covers: FR-TEST-HYG-078
- Level: integration (Go, `internal/service`; fault injection)
- Given the change applied, and `scheduleUnscheduledAmazonItems` temporarily edited to call `s.nextCheckAt("amazon", s.now())` once before the loop and store that one time for every item
- When the run executes
- Then the redraw test fails after the first start, and the failure message names two of the three item IDs and their shared time
- And, without the fault injection, the run passes
- And, with the same fault injection on `master` (the test file before the change), the run passes, which shows the gap the change closes
- Corner cases:
  - the failure comes from the new check, not from the window check: the shared time is in the evening window of 2030-03-05 and is not the former 08:00 time;
  - the redraw mark is still cleared, so the failure is not the mark check either;
  - the times are compared as instants (`time.Time.Equal`), so the same time read back from the database in another location or monotonic state still counts as equal.

### TC-003: The same time for only two of the three items fails the test, for every pair

- Covers: FR-TEST-HYG-078
- Level: integration (Go, `internal/service`; fault injection)
- Given the change applied, and one of these temporary edits at a time, each drawing one shared time before the loop, storing it for the two named items only and drawing a time of its own for the third:
  1. `B000001853` and `B000001854`;
  2. `B000001854` and `B000001855`;
  3. `B000001853` and `B000001855`
- When the run executes
- Then the redraw test fails each time, with a message naming the two items of the edit and their shared time
- Corner cases:
  - edit 3 catches a check that compares only neighbouring items (first with second, second with third): such a check would let that run pass;
  - the edits do not depend on the order in which the store returns the items.

### TC-004: The redraw test keeps its name and its other checks and messages

- Covers: FR-TEST-HYG-079
- Level: code check of the diff of the test file, and the `-v` output of the run on `master` and with the change
- Given the change applied
- When the diff of the test file is read and both runs are compared
- Then:
  - the test name is unchanged and both runs report it as `PASS`;
  - the checks before the start (mark 1, every item at the shared former time), after the first start (evening window of 2030-03-05, not the former time, mark 0) and after the second start (every item keeps the time of the first start, mark 0) are kept with their messages;
  - the seeds, clocks, item IDs and the former time are unchanged;
  - the diff only adds the pairwise check after the first start's loop.
- Corner cases:
  - no other test in the test file changes;
  - helpers the test uses (`insertAt`, `nextCheck`, `redrawMark`, `setRedrawMark`, `startHuman`, `windowOf`, `localDate`) are unchanged.

### TC-005: TC-002 of the issue #185 test cases states the pairwise different times

- Covers: FR-TEST-HYG-080
- Level: document check of the issue #185 test cases
- Given the change applied
- When `git diff master -- <the issue #185 test cases>` is read
- Then:
  - the Then clauses of the first start of TC-002 also state that the three Amazon items' next scheduled checks are pairwise different;
  - its corner cases state that one time drawn and given to every item makes the test fail;
  - the Covers line, Level, Given, the other Then clauses, the second start and the other corner cases of TC-002 are unchanged;
  - no other line of the document changes.
- Corner cases:
  - the coverage summary and TC-001, TC-003, TC-004 and TC-005 of that document are unchanged.

### TC-006: Only the test file changes outside `doc/`, and the gates pass

- Covers: FR-TEST-HYG-081
- Level: repository check and suite run (deterministic gates: `gofmt -l internal/service`, `go vet ./...`, `go test ./...` and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` and `git diff --name-only master -- doc/changes/2026-11-23-1706-issue-185-tests-missing-from-168-amazon-human-like/` are listed and the gates run
- Then the only file listed outside `doc/` is `internal/service/amazon_human_checks_issue185_test.go`, the only file listed in the issue #185 change folder is its `testcases.md` (TC-005 checks it changes only TC-002), `gofmt -l internal/service` prints nothing, and `go vet ./...` and `go test ./...` pass
- Corner cases:
  - production Go code (including `internal/service/service.go` and `internal/service/amazon_schedule.go`), frontend source, `go.mod`, `go.sum`, `package.json`, `tests/`, `API_SPECIFICATION.md` and the Amazon human-like checks specifications are unchanged, so the operator observes no change in the application;
  - the issue #185 `tests.lock` and the follow-up record `doc/todo/issue-185-r1-01.md` are unchanged;
  - no fault injection from TC-002 or TC-003 remains;
  - the gates give the same result when run twice.
