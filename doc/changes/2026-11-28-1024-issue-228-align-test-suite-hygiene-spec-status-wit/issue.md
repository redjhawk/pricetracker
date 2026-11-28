# Issue #228 Align test-suite-hygiene spec status with the index and convention

**Problem:** The status of the test-suite-hygiene subject is inconsistent. `doc/specifications/test-suite-hygiene/functional.md` line 3 says `Status: implemented`, `doc/specifications/test-suite-hygiene/technical.md` line 3 says `Status: ready`, and the index row in `doc/FUNCTIONAL_SPECIFICATIONS.md` line 347 says `specified (issue #208, ...)`. Every other spec uses `Status: ready`, sometimes with the issue number, e.g. `Status: ready (issue #204)`.

**Suggested work:** Choose one status and use it everywhere. Following the convention, set `Status: ready (issue #208)` in `functional.md` (line 3) and `technical.md` (line 3), and update the Status column of the index row in `doc/FUNCTIONAL_SPECIFICATIONS.md` (line 347) to match. Also drop the optional `Status: implemented` step from the file table in `technical.md` (line 34), or document `implemented` as an accepted status value.

From review comment R1-01 on #227 (`doc/todo/issue-208-r1-01.md`).

## Comments by trusted authors

