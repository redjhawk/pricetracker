# Blocking review comments (round 3)

## R3-01 `doc/changes/2026-11-25-0944-issue-200-align-ts-responsive-017-wording-with-the/no-tests.md:13`

Blocking comment R2-01 is still not fixed. The round-2 development commit (ee44efa) only deleted `blocking.md` and updated `state.json`. `no-tests.md` line 13 and `testcases.md` lines 59 and 64 are unchanged. On this branch, `git diff --name-status origin/master...HEAD -- 'doc/changes/*issue-140-*' 'doc/changes/*issue-165-*' 'doc/changes/*issue-203-*' doc/todo/` still prints `A	doc/todo/issue-200-r1-02.md`, so TC-003 fails on the correct change. Fix, as triage confirmed: check `doc/todo/` with `git diff --name-status --diff-filter=DMRT <base> -- doc/todo/`, or exclude `doc/todo/issue-200-r*.md`. Change both no-tests.md line 13 and testcases.md lines 59 and 64. In line 64, change "every file under `doc/todo/` ... are unchanged" to "every existing file under `doc/todo/`".

Triage: Confirmed: commit ee44efa only touched `blocking.md` and `state.json`; no-tests.md line 13 and testcases.md lines 59 and 64 are unchanged, and `git diff --name-status master -- 'doc/changes/*issue-140-*' 'doc/changes/*issue-165-*' 'doc/changes/*issue-203-*' doc/todo/` still prints `A doc/todo/issue-200-r1-02.md`, so TC-003 fails on the correct change. `git diff --name-status --diff-filter=DMRT master -- doc/todo/` prints nothing, so scoping the `doc/todo/` check to existing files (and saying "every existing file under `doc/todo/`" in line 64) fixes it.

## R3-02 `doc/specifications/responsive-layout/technical.md:231`

TS-RESPONSIVE-029 says `doc/todo/` is unchanged for issue #200. The branch adds `doc/todo/issue-200-r1-02.md`, the follow-up record from round-1 triage, so the technical specification contradicts the change it describes. This is the same scoping problem as R3-01. Fix: say "the existing files under `doc/todo/`", matching FR-RESPONSIVE-030 ("the files under `doc/todo/` are unchanged"), so new follow-up records for this issue are allowed.

Triage: Confirmed: TS-RESPONSIVE-029 (technical.md line 231) says `doc/todo/` is unchanged for issue #200, but the branch adds `doc/todo/issue-200-r1-02.md`, so the technical specification contradicts the change it describes and the TC-003 check built on it. The root cause is the same as R3-01; reword to "the existing files under `doc/todo/`" in the same fix, matching FR-RESPONSIVE-030.

