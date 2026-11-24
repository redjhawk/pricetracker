# Red phase skipped for issue #182

The tests of this change are written, but no red phase is possible. They test behaviour that issue #168 (PR #179) already shipped: `RefreshItem` redraws an overdue Amazon check time, `skipScheduledCheck` gives an in-flight Amazon item a new time, and a Refresh price restarts a stopped item. The technical specification (`doc/specifications/test-suite-hygiene/technical.md`, "Scheduled checks around a Refresh price") says no production code changes. Both tests pass on the `master` code, so the red gate would always fail. This file skips it.

The tests and the mutation checks below show that the tests can fail.

## Tests written

| Test case | Test |
| --- | --- |
| TC-001 | `internal/service/amazon_human_checks_test.go`, subtest `Refresh price running` of `TestAmazonHumanChecks_TC032_NoDuplicateCheckWhileOneIsRunning`. Added `database.SetNextCheck(ctx, id, now.Now().Add(-time.Minute))` after `waitStarted` and before `collectDue`. This is the only diff in that file. |
| TC-003 | Subtest `first check after add running` in the same test, unchanged |
| TC-004 | `internal/service/amazon_human_checks_issue182_test.go`: `TestAmazonRestartAfterStop_Issue182_TC004_NoScheduledCheckAfterRefreshPrice` (new) |
| TC-002, TC-005 | Mutation checks below |
| TC-006 | Gates and repository checks below |

## Mutation checks (TC-002, TC-005)

This stage may not edit production files, so each mutation was applied with `go test -overlay`. The overlay replaced `internal/service/service.go` or `internal/service/amazon_gate.go` with a scratch copy that contained the mutation. The scratch files are listed in `delete.txt`. No production file changed: `git diff --name-status master -- . ':!doc'` lists only the two test files.

1. **TC-002, mutation 1.** In `skipScheduledCheck`, after `now := s.now()`: `if active && item.Platform == "amazon" { return true }`.
   - Command: `go test -count=1 -run 'TestAmazonHumanChecks_TC032_NoDuplicateCheckWhileOneIsRunning/Refresh_price_running' ./internal/service/`
   - Changed subtest: **FAIL** `amazon_human_checks_test.go:764: next check of B000000321 at 2030-03-05 07:04:00 +0100 CET, want a later window`
   - `master` subtest (the `SetNextCheck` step removed for the run, then restored): **ok**. The old subtest did not reach the in-flight branch, as review R3-02 found.
2. **TC-005 a, mutation 2.** In `RefreshItem`, replace `err = s.rescheduleOverdueAmazonCheck(ctx, id, requestedAt)` with `err = nil`.
   - Command: `go test -count=1 -run Issue182 ./internal/service/`
   - **FAIL** `amazon_human_checks_issue182_test.go:60: 2 checks of the item (2 in total) after the 07:31 runs, want 1`
3. **TC-005 b, mutation 3.** In `amazonGate.do`, remove the `if stopped { ClearAmazonStop … }` block.
   - Command: `go test -count=1 -run Issue182 ./internal/service/`
   - **FAIL** `amazon_human_checks_issue182_test.go:50: the item is still stopped after the Refresh price`
4. **Revert.** With the mutations off, and without the overlay, `go test -count=1 -run 'TestAmazonHumanChecks_TC032|Issue182' ./internal/service/` passes.

## Gates and repository checks (TC-003, TC-006)

- `go test -count=1 -v -run 'TestAmazonHumanChecks_TC032|Issue182' ./internal/service/`: `Refresh_price_running`, `first_check_after_add_running` and `TestAmazonRestartAfterStop_Issue182_TC004_NoScheduledCheckAfterRefreshPrice` all `PASS`.
- `gofmt -l internal/service`: empty. `go vet ./...` and `go test -count=1 ./...`: pass.
- `git diff master -- internal/service/amazon_human_checks_test.go`: only the three lines of the `SetNextCheck` step inside `Refresh price running`. The `first check after add running` subtest and `assertScheduled` are unchanged.
- The tests use no build tag, skip, environment variable or `t.Parallel`. No test was removed.
