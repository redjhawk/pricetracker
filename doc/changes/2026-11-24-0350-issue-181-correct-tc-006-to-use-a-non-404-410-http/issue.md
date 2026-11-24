# Issue #181 Correct TC-006 to use a non-404/410 HTTP status

**Problem:** `doc/changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n/testcases.md:91` (TC-006) lists "HTTP 404 for the product page" as a failure. TS-AMZ-HCHK-003 step 7 and `classifyPage` map 404/410 to `unavailable`. The test in `internal/amazon/browser_test.go:335` uses 403 instead.

**Suggested work:** In TC-006, replace "HTTP 404" with a non-404/410 status such as 403. Add "HTTP 404/410 → unavailable" as a corner case of TC-004 so the test case document matches the specification and the tests.

From review comment R1-04 on #179 (`doc/todo/issue-168-r1-04.md`).

## Comments by trusted authors

