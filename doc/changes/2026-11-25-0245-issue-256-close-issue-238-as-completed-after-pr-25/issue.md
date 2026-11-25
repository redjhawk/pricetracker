# Issue #256 Close issue #238 as completed after PR #255 merges

**Problem:** FR-TEST-HYG-019 (doc/specifications/test-suite-hygiene/functional.md, status line 3) requires issue #238 to be closed as completed. PR #255 closes only #241 (`Closes #$ISSUE` in scripts/factory/factory.sh:448), so #238 stays open after the merge.

**Suggested work:**
- After PR #255 merges, run `gh issue close 238 --reason completed --comment "Done by PR #236; its follow-up record doc/todo/issue-233-r1-03.md is removed by the pull request of #241."` (TS-TEST-HYG-022 in doc/specifications/test-suite-hygiene/technical.md).
- Verify TC-002 with `gh issue view 238 --json state,stateReason` (expect `CLOSED` / `COMPLETED`).
- Set FR-TEST-HYG-019 to implemented in the status line of doc/specifications/test-suite-hygiene/functional.md.
- Optionally: let the factory add extra `Closes #N` lines to the PR body when a change closes other issues as well.

From review comment R1-01 on #255 (`doc/todo/issue-241-r1-01.md`).

## Comments by trusted authors

