# Issue #182 Make the TC-032 Refresh price subtest reach the in-flight branch and test restart after stop

**Problem:** In `internal/service/amazon_human_checks_test.go:746` (TC-032, subtest "Refresh price running"), the item is due at 07:00 and the clock is at 07:05. `RefreshItem` now calls `rescheduleOverdueAmazonCheck` (`internal/service/service.go:325-331`), which moves the overdue time to the evening window before the check starts. So `collectDue` finds no due item, and the in-flight branch of `skipScheduledCheck` never runs for a Refresh price. The subtest would still pass if that branch were removed. The R2-01 scenario has no test either.

**Suggested work:**
- In the Refresh price subtest, call `database.SetNextCheck(ctx, id, now.Now().Add(-time.Minute))` after `waitStarted` and before `service.collectDue(ctx)`, as the add subtest does.
- Add a service test for restart after stop: the item is due at 07:00 and stopped; Refresh price at 07:30 restarts it; `collectDue` at 07:31 gives exactly one check in total, and the next check is in the evening window.

From review comment R3-02 on #179 (`doc/todo/issue-168-r3-02.md`).

## Comments by trusted authors

