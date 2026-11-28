# Issue #234 Align backend step 4 of the Amazon human-checks spec with TS-AMZ-HCHK-017

**Problem:** In `doc/specifications/amazon-human-checks/technical.md`, Backend `Collect` step 4 (around line 69) says the HTTP cache is cleared "with a 2xx response". `browserTab.load` (`internal/amazon/browser.go:441-451`) also clears it when no response was recorded within the 1 s wait, as the TS-AMZ-HCHK-017 table row (line 30) and the bullet around line 360 already say.

**Suggested work:** Change the clause to "with a 2xx response or none (a response may come after the 1 s wait), the HTTP cache is cleared first (issue #217, TS-AMZ-HCHK-017)".

From review comment R2-02 on #231 (`doc/todo/issue-217-r2-02.md`).

## Comments by trusted authors

