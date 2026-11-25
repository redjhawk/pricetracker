# Issue #260 Assert that the upgrade redraw draws a different time for each Amazon item

**Problem:** `TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce` (`internal/service/amazon_human_checks_issue185_test.go:118-125`) lists FR-AMZ-HCHK-005 (independent draws per item) as covered behaviour, and the TC-002 title says each item gets "its own" window time. But the test only checks that each item is in the evening window. A regression that draws one time and writes it to every item (for example `nextCheckAt` moved out of the loop in `scheduleUnscheduledAmazonItems`) would still pass.

**Suggested work:** The random source is seeded (1851), so the draws are deterministic. After the loop, fail if any two values in `drawn` are equal, e.g. `drawn[ids[0]].Equal(drawn[ids[1]]) || drawn[ids[1]].Equal(drawn[ids[2]]) || drawn[ids[0]].Equal(drawn[ids[2]])`. Update the TC-002 Then clauses in the change's `testcases.md` to match.

From review comment R1-01 on #259 (`doc/todo/issue-185-r1-01.md`).

## Comments by trusted authors

