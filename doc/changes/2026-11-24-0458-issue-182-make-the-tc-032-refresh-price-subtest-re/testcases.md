# Test cases: TC-032 Refresh price subtest and restart after stop (issue #182)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-033 to FR-TEST-HYG-036). No use cases: see [no-use-cases.md](no-use-cases.md). The application behaviour under test is already described by UC-AMZ-HCHK-001 and UC-AMZ-HCHK-003 (FR-AMZ-HCHK-007, FR-AMZ-HCHK-008, FR-AMZ-STOP-004, FR-AMZ-STOP-006). These cases cite those use cases as context and do not change them.

Conventions:

- **Times** are in the schedule timezone (`paris`) on the test day `scheduleDay` (2030-03-05). The Amazon windows are morning 06:00–09:00 and evening 19:00–23:00.
- **Service test** (level `unit`): a Go test in `internal/service/` that uses the existing helpers of `amazon_human_checks_test.go`: a real SQLite store (`openHumanStore`), the controlled clock (`newClock`), the fake Amazon collector `amazonRecorder` (it counts calls and can block with `started`/`block`), and `startHuman`, `insertAt`, `nextCheck`, `windowOf`, `localDate`, `stopped`, `waitStarted` and `waitIdle`. No real Amazon request, browser, HTTP handler or network is used.
- **Mutation check**: a temporary edit of production code in a scratch worktree, never committed. The named test must fail with the edit and pass again once the edit is reverted.
- This change only edits and adds Go test code. No API, database, user interface or production code changes, so it has no new user flow, and no new integration or e2e case is needed. Narrow screens and the keyboard do not apply. The existing suites (`go test ./...`, `npx playwright test`) show that the operator sees no change.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-TEST-HYG-033 | TC-001, TC-002 |
| FR-TEST-HYG-034 | TC-003 |
| FR-TEST-HYG-035 | TC-004, TC-005 |
| FR-TEST-HYG-036 | TC-006 |

## Cases

### TC-001 — "Refresh price running" sees the item due while its check is in flight

- Covers: FR-TEST-HYG-033 (context: UC-AMZ-HCHK-001, UC-AMZ-HCHK-003, UC-ID-02)
- Level: unit (service test; subtest "Refresh price running" of `TestAmazonHumanChecks_TC032_NoDuplicateCheckWhileOneIsRunning`)
- Given a tracked, not stopped Amazon item `B000000321` due at 07:00, the clock at 07:05, and an `amazonRecorder` that blocks each check until it is released
- When `RefreshItem` is called, the test waits until the Amazon request has started (`waitStarted`), then sets the item's due time to 07:04 (`SetNextCheck(now − 1 min)`, a past time of the morning window), then runs `collectDue` while the check is still blocked, and then releases the check and waits until the service is idle
- Then the recorder counts exactly one check of the item (the Refresh price), and the item's next scheduled check is after 07:05 and inside a window (`windowOf ≥ 0`)
- Corner cases:
  - the due time is set only after the request has started and before `collectDue`. If it were set before `RefreshItem`, the redraw of an overdue Amazon time would hide it, as in the old version of the subtest;
  - `collectDue` runs while the Refresh price is still in flight (the block is not yet released), so `skipScheduledCheck` sees the item due and in flight;
  - the item's due time after the run is in the future: a past due time left in place fails the subtest;
  - a second check by the scheduled run (call count 2) fails the subtest;
  - the stored next check is read after `waitIdle`, so the end of the Refresh price cannot move it back to a past time.

### TC-002 — The subtest fails when the in-flight branch is removed

- Covers: FR-TEST-HYG-033
- Level: mutation check on the TC-001 subtest (`go test -count=1 -run 'TestAmazonHumanChecks_TC032_NoDuplicateCheckWhileOneIsRunning/Refresh_price_running' ./internal/service/`)
- Given the changed subtest of TC-001, in a scratch worktree
- When `skipScheduledCheck` is changed so that an Amazon item in flight is skipped without a new next check (it returns `true` before `SetNextCheck` when `active`), the subtest runs, then the change is reverted and the subtest runs again
- Then the subtest fails with the change (the item keeps the past due time 07:04 after the Refresh price ends, because a manual Amazon check does not change the schedule) and passes after the revert
- Corner cases:
  - on `master` (the old subtest, without the `SetNextCheck` step), the same change does not make the subtest fail. This shows that the old subtest did not reach the branch, as review R3-02 found;
  - the in-flight guard of `collect` stays in place. Removing it is not used as a mutation: `collectDue` calls `Collect` synchronously, so a second check would block on the held fake request, and the test would hang instead of failing with a clear message;
  - the "first check after add running" subtest is not used as proof here: it is checked by TC-003.

### TC-003 — "first check after add running" keeps its case and pass condition

- Covers: FR-TEST-HYG-034
- Level: unit (service test; subtest "first check after add running" of `TestAmazonHumanChecks_TC032_NoDuplicateCheckWhileOneIsRunning`) and repository check (`git diff master -- internal/service/amazon_human_checks_test.go`)
- Given the change applied on top of `master`
- When the subtest runs, and its body is compared with its `master` version
- Then it passes: the item added from `https://www.amazon.fr/dp/B000000322` at 07:05 gets its first check, its due time is set to 07:04 while the check is blocked, `collectDue` runs, and exactly one check is counted, with a next scheduled check after 07:05 inside a window. The diff changes none of its steps or assertions (the shared `assertScheduled` helper keeps its condition)
- Corner cases:
  - the subtest name is unchanged, so `-run '.../first_check_after_add_running'` still selects it;
  - the subtest still uses its own store, clock and seed (33), so it does not depend on the "Refresh price running" subtest, and running it alone or after that subtest gives the same result.

### TC-004 — A stopped item restarted by Refresh price at 07:30 is not checked again at 07:31

- Covers: FR-TEST-HYG-035 (context: UC-AMZ-HCHK-001, UC-AMZ-HCHK-003; FR-AMZ-STOP-004, FR-AMZ-STOP-006)
- Level: unit (new service test in `internal/service/`, for example in `amazon_human_checks_test.go`)
- Given a tracked Amazon item due at 07:00 and stopped at 07:00 (`StopAmazonItems`), a clock at 07:00, and an `amazonRecorder` that returns an Amazon answer (a success with a price, the default `succeeded()`)
- When the scheduled checks run at 07:00 (the stopped item is skipped), then the clock moves to 07:30, `RefreshItem` is called and the test waits until the service is idle, then the clock moves to 07:31 and `collectDue` runs
- Then:
  - after the Refresh price, the item is no longer stopped (`stopped` is false);
  - the recorder counts exactly one check of the item in total, the Refresh price at 07:30, so the 07:31 run makes none;
  - after the 07:31 run, the item's next scheduled check is in the evening window (`windowOf == evening`, 19:00–23:00) of the same day (`localDate == "2030-03-05"`)
- Corner cases:
  - the 07:00 run makes no check of the stopped item (zero calls before the Refresh price), and the item stays stopped until the Refresh price;
  - the 07:31 time is still in the morning window. Without the redraw by Refresh price, the overdue 07:00 time would be checked at 07:31, and the test would count two checks;
  - an item that stays stopped after the Refresh price fails the test;
  - a next check in the morning window, on the next day, or in the past fails the test;
  - a second `collectDue` at 07:31 also makes no check (repeated run).

### TC-005 — The restart-after-stop test fails if Refresh price leaves an overdue time

- Covers: FR-TEST-HYG-035
- Level: mutation check on the TC-004 test (`go test -count=1 -run <TC-004 test name> ./internal/service/`)
- Given the TC-004 test, in a scratch worktree
- When `RefreshItem` is changed so that it leaves the overdue due time of an Amazon item unchanged (the call to `rescheduleOverdueAmazonCheck` is removed, or the function returns `nil` at once), the test runs, then the change is reverted and the test runs again
- Then the test fails with the change (the 07:31 run makes a second check of the item) and passes after the revert
- Corner cases:
  - a second mutation, in which the Refresh price does not restart the stopped item (manual checks follow the stop), also makes the test fail;
  - the other TC-032 subtests and TC-022 (stopped item skipped at its time) are not needed for this proof.

### TC-006 — The tests run in the unit gate, the code is formatted, and only service test files change

- Covers: FR-TEST-HYG-036
- Level: repository check and suite run (deterministic gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`, and the change's diff)
- Given the change applied on top of `master`
- When `go vet ./...` and `go test ./...` run, `go test -count=1 -v -run 'TestAmazonHumanChecks_TC032|<TC-004 test name>' ./internal/service/` runs, `gofmt -l internal/service` runs, and `git diff --name-status master -- . ':!doc'` is listed
- Then `go vet` and `go test` pass; the verbose run lists the two TC-032 subtests and the TC-004 test as run and passed; `gofmt -l internal/service` prints nothing; and every path in the diff outside `doc/` matches `internal/service/*_test.go`
- Corner cases:
  - the tests need no build tag, environment variable or separate command, and they give the same result without cached results (`-count=1`) and when the gates run twice;
  - no production Go file, `go.mod`, `go.sum`, frontend source, Playwright spec or `API_SPECIFICATION.md` changes, so the operator sees no change in the application. The Playwright suite runs the same test list as on `master`;
  - no other test of `internal/service/` is removed or skipped (the `go test -v` list of the package keeps every test name that `master` has).
