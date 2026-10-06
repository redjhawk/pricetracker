# PR comment triage: 2026-10-06-2117-workflow-pr-reviewer / 2026-10-06-2138-workflow-pr-triage

Triage agent: independent PR comment triage agent (separate Task invocation; not the developer or PR reviewer)

## Rounds

### Round 1 (2026-10-06, PR #55 at 6d069ea, PR #57 at b217738)

Verification notes: `origin/feature/workflow-pr-review` is 0 commits ahead of `origin/master` and there is no feature->master PR (confirms G1). The ai-dev allowlist on `docs/workflow-pr-triage` contains `Bash(gh pr merge:*)` and `Bash(gh api repos/redjhawk/pricetracker/pulls/*/reviews --method POST --input *)` (confirms G2/G3). The ai-dev prompt has no instruction to reuse an existing `doc/changes/*-issue-<n>-*` folder (confirms #54 bug). `pr-review.md` exists only on #57's branch, `pr-triage.md` exists on neither.

#### Grouped blocking defects

- **G1. Final feature->master PR cannot be opened at stage 8** (4200750002, 4200750004, 4200750373, 4200750384). Same defect, four places.
  Required fix (on #55 for stage 8 text and prompt, #57 for merge text): stage 8 must not open the final PR for split changes. In the merge step, after every part is merged into the feature branch and the `git merge-base --is-ancestor` check passes, create the final PR (`gh pr create --base master --head ai-dev/issue-<n>-feature`, body with `Closes #<n>`), then `gh pr merge <n> --squash --delete-branch`. Update WORKFLOW.md stage 8 row, the stacking section, the Completion gate ("pull requests exist" -> parts exist; final PR created at merge), the merge bullet (line 168), the ai-dev prompt (stage 8 and merge text), pr-review-triage.md, and the matching TS (TS-WORKFLOW-TRIAGE-005 and stage-8 TS). Remove the now-false "final PR" limitation from commit-step records.
- **G2. `Bash(gh pr merge:*)` is unrestricted and contradicts TS-WORKFLOW-PRR-005** (4200750378).
  Required fix (#57): replace with `Bash(gh pr merge * --merge --delete-branch)` and `Bash(gh pr merge * --squash --delete-branch)`; update TS-WORKFLOW-PRR-005 (it currently says the run "cannot merge") and the #55/#57 PR descriptions to state that merging is allowed only in those two forms and that globs are not a security boundary (see G3).
- **G3. Review-posting glob claimed as a guarantee** (4200574885).
  Required fix (#55): reword TS-WORKFLOW-PRR-005 so it does not claim the pattern prevents merge/close/dismiss; state the `*` glob can match extra arguments and that safety rests on token scope and the procedure. (A numeric-checked wrapper script is optional hardening; tracked as todo T-A rather than required here.)
- **G4. `pr-review.md` is lost in ai-dev runs** (4200574892).
  Required fix (#55, ai-dev prompt + pr-reviewer role): after posting reviews, commit `doc/changes/<change>/pr-review.md` to the last PR branch with `Refs: #<n>` and push; record the reviewed head SHA (not the post-commit head), and state that a commit that only adds/updates `pr-review.md`/`pr-triage.md`/todo files needs no re-review. Apply the same rule to `pr-triage.md` in #57.

| Comment (link) | Label | Decision (blocking / non-blocking) | Reason | Outcome |
|---|---|---|---|---|
| https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200574885 | [bug] | blocking (G3) | Verified: middle `*` matches arbitrary text; TS-PRR-005 claims a guarantee the allowlist cannot give. Spec/contract inaccuracy on a security control. Fix is a wording change; the wrapper is optional (todo T-A). | pending developer fix |
| https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200574892 | [bug] | blocking (G4) | Verified: no instruction to commit `pr-review.md`; the ai-dev workspace is discarded, so the record the Completion gate depends on is lost (data-integrity of workflow records). | pending developer fix |
| https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200574899 | [simplify] | non-blocking | Simplification; the fallback path is correct, only wasteful. | todo T-B |
| https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200574906 | [nit] | non-blocking | Coordinator stages explicitly, so stray JSON is unlikely to be committed; still worth naming a temp path. | todo T-C |
| https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200574912 | [nit] | non-blocking | Grammar only. | todo T-D (doc nits) |
| https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200574919 | [nit] | non-blocking | Record clarity; the review body prefix still distinguishes blocking COMMENTs. | todo T-E |
| https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200574926 | [nit] | non-blocking | Verified AGENTS.md:27 lists only reviewer and adjudicator; role file already states PR reviewer independence, so no behavior gap. | todo T-D (doc nits) |
| https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200750002 | [bug] | blocking (G1) | Verified: feature branch 0 commits ahead of master, no final PR exists; stage 8 instruction can never succeed. | pending developer fix |
| https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200750004 | [bug] | blocking (G1) | Same defect as 4200750002 in WORKFLOW.md stage 8 row. | pending developer fix |
| https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200750014 | [nit] | non-blocking | Valid observation about intermediate state of part 1, but parts land together on the feature branch and only the final squash reaches master, so master never sees part 1 alone. | todo T-F |
| https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200750027 | [nit] | non-blocking | Stale record text; record accuracy, no behavior. Can be fixed cheaply alongside G1 commit-step updates. | todo T-G (stale change records) |
| https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200750039 | [nit] | non-blocking | Verified: `pr-review.md` only on #57's branch; link resolves once the stack is merged (feature branch is the unit that reaches master). | todo T-G |
| https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200750373 | [bug] | blocking (G1) | Same defect: merge step expects a final PR that does not exist. | pending developer fix |
| https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200750378 | [bug] | blocking (G2) | Verified: `Bash(gh pr merge:*)` in allowlist; master unprotected so any merge deploys; contradicts unchanged TS-WORKFLOW-PRR-005. Security risk + spec mismatch. | pending developer fix |
| https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200750384 | [bug] | blocking (G1) | Same defect in the ai-dev prompt merge text and stage 8 text. | pending developer fix |
| https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200750388 | [readability] | non-blocking | Wording inconsistency; all three variants stop at 5 rounds. Behaviorally equivalent enough; define "round" once. | todo T-H |
| https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200750391 | [nit] | non-blocking | Verified WORKFLOW.md:167 list omits "broken build or test"; the role, which the triage agent actually loads, includes it. | todo T-D (doc nits) |
| https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200750398 | [test] | non-blocking | Process concern about the adjudicator's waiver of the use case/regression test for REV-001. Not a code defect; the missed defect it cites is covered by G1. Documentation-only change has no test harness; adding a written use case is suggested work. Not a functional question requiring the user. | todo T-I |
| https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200750403 | [nit] | non-blocking | Stale record text. | todo T-G |
| https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200750409 | [nit] | non-blocking | `pr-triage.md` will be created by this stage (coordinator writes this record into the folder), which resolves the link. | todo T-G (resolved if this record is committed) |
| https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200750412 | [nit] | non-blocking | Formatting only. | todo T-D (doc nits) |
| https://github.com/redjhawk/pricetracker/pull/54#discussion_r4200575319 | [bug] | non-blocking for this feature | Verified still open: the prompt has no "reuse existing `doc/changes/*-issue-<n>-*` folder" rule. It is a real pre-existing defect from merged #54, not introduced by #55/#57; folding it into this feature would mix unrelated work (itself a blocking criterion). Track and fix in its own small PR promptly. | todo T-J |

#### Todos for non-blocking comments (coordinator creates `doc/todo/2026-10-06-<slug>.md` + issue)

- **T-A** `pr-review-post-wrapper` - "Post PR reviews through a numeric-checked wrapper". Problem: allowlist glob `pulls/*/reviews ... --input *` matches arbitrary arguments. Suggested work: add `scripts/post-pr-review.sh <n> <file>` validating `^[0-9]+$` and allow only `Bash(scripts/post-pr-review.sh:*)`. Source: 4200574885.
- **T-B** `pr-review-direct-comment-in-ai-dev` - "Submit COMMENT directly in ai-dev runs". Problem: ai-dev always tries REQUEST_CHANGES, which always 422s. Suggested work: in ai-dev submit `COMMENT` with `CHANGES REQUESTED:` prefix directly; elsewhere try REQUEST_CHANGES and fall back on 422. Source: 4200574899.
- **T-C** `pr-review-json-temp-path` - "Write review JSON outside the work tree". Problem: no location given for `--input` file; risk of stray `review-*.json` commits. Suggested work: specify `$RUNNER_TEMP` (or an ignored path) in role and prompt. Source: 4200574906.
- **T-D** `workflow-doc-nits-pr-review` - "Fix wording nits in PR review/triage workflow docs". Problem: double "and" (WORKFLOW.md:55), AGENTS.md independence sentence omits PR reviewer/triage, WORKFLOW.md:167 blocking list omits "broken build or test", ai-dev.yml:4 over-long header line. Suggested work: apply the four edits; keep the blocking list only in the role and link to it. Sources: 4200574912, 4200574926, 4200750391, 4200750412.
- **T-E** `pr-review-event-column-values` - "Distinguish blocking COMMENT in pr-review template". Problem: Event column can't tell fallback blocking COMMENT from plain COMMENT. Suggested work: values `REQUEST_CHANGES / COMMENT (CHANGES REQUESTED) / COMMENT`. Source: 4200574919.
- **T-F** `stacked-parts-coherent-intermediate` - "Write stack parts so each part is a coherent state". Problem: part 1 introduces text that part 2 rewrites (user decides -> triage decides). Suggested work: add workflow guidance that split parts use neutral wording for behavior a later part replaces. Source: 4200750014.
- **T-G** `pr-reviewer-triage-stale-records` - "Correct stale commit-step records and pending links". Problem: commit-step.md of both change folders describe wrong PRs/bases; index.md links to files added later. Suggested work: record real stack (#55 -> feature/workflow-pr-review, #57 -> docs/workflow-pr-reviewer), mark links pending until files exist. Sources: 4200750027, 4200750039, 4200750403, 4200750409.
- **T-H** `triage-loop-guard-wording` - "Define a triage round and one stop rule". Problem: loop guard worded three ways; "same comment" not trackable across reviews. Suggested work: define round = fix push + re-review + triage; use "stop after 5 rounds if any blocking comment remains" in role, WORKFLOW.md, TS-WORKFLOW-TRIAGE-004 and prompt. Source: 4200750388.
- **T-I** `split-stack-merge-use-case` - "Add a use case walking stage 8-10 for a 2-part split". Problem: REV-001 critical-fix waiver left no use case/regression check; the missing final PR slipped through. Suggested work: write `doc/use-cases/` walkthrough of a 2-part split through merge, or a scripted dry run against a scratch repo. Source: 4200750398.
- **T-J** `ai-dev-reuse-change-folder-on-resume` - "Reuse the issue's change folder when ai-dev resumes". Problem: a resumed ai-dev run can create a second `doc/changes/*-issue-<n>-*` folder, breaking FR-WORKFLOW-DATE-003. Suggested work: add to the ai-dev prompt "if a folder `doc/changes/*-issue-<n>-*` already exists, keep using it and never create a second one", in a separate PR. Source: https://github.com/redjhawk/pricetracker/pull/54#discussion_r4200575319.

## Merge

- No blocking comments remain: no (round 1: G1-G4 blocking, 7 comments)
- Merges in order (PR, base, method, result): not started
- Deploy: not started
- User informed: pending coordinator

### Round 2 (2026-10-06, PR #57 at 9c0a89f22f4ca3eb27cf063d286979b47a6e4c10; PR #55 re-review: 0 comments)

| Comment (link) | Label | Decision (blocking / non-blocking) | Reason | Outcome |
|---|---|---|---|---|
| https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200798855 | [bug] | blocking (G5) | Verified WORKFLOW.md:170 handoff requires `pr-triage.md` (incl. merges and deploy) committed to the last PR branch, but stage 10 merges with `--delete-branch` and squashes the feature branch into `master` before the deploy exists; the record can then only reach `master` by a direct push, which the workflow never authorizes. The handoff is unexecutable and the merge record is lost. | pending developer fix |
| https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200798863 | [bug] | blocking (G5) | Same defect in `templates/pr-triage.md:17` `## Merge` section (merges, deploy, user informed fields) and in the role's commit instruction. | pending developer fix |
| https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200798870 | [readability] | non-blocking | Verified TS-WORKFLOW-PRR-004 still says "not to fix its comments without `@claude`", superseded by stage 10 (triage drives fixes). Stale spec wording; role, WORKFLOW.md and the prompt carry the actual behavior, so no runtime effect. | todo T-K |

#### Required fix for G5 (developer, on PR #57 branch)

In `doc/workflow/WORKFLOW.md` (stage 10 Merge bullet + Handoff), `.agents/roles/pr-review-triage.md`, `doc/workflow/templates/pr-triage.md`, the ai-dev prompt, and the triage functional/technical spec if they state it: split the record in two moments. (1) Before the first merge, the triage agent fills `## Merge` with "No blocking comments remain: yes, at round k" and the planned merge order (PR, base, method), and the coordinator commits and pushes `pr-triage.md` to the last PR branch (record-only, no re-review), so it travels into `master` with the squash. (2) Actual merge results, merge failures, the deploy run link, todo/issue outcomes created after that point are reported only on the originating issue (`gh issue comment`) and in the final run report, never committed after branches are deleted. Rename the template fields accordingly (e.g. "Planned merge order"; "Merge results and deploy: reported on <issue comment / final run report>"). Then the PR reviewer re-reviews #57.

#### Todos for non-blocking comments

- **T-K** `prr-004-stale-fix-wording` - "Align TS-WORKFLOW-PRR-004 with stage 10 fix loop". Problem: `doc/specifications/workflow-pr-reviewer/technical.md:9` still says review comments are not fixed without `@claude`, contradicting stage 10 where blocking comments are fixed automatically. Suggested work: reword to "stage 10 triage decides; blocking comments are fixed by the developer agent, non-blocking become todos (see TS-WORKFLOW-TRIAGE-*)" (can be folded into the G5 commit cheaply). Source: https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200798870.

## Merge (round 2)

- No blocking comments remain: no (round 2: G5 blocking, 2 comments)
- Merges: not started

### Round 3 (2026-10-06, PR #57 at 73c463ec092cdcaf34f5d0b860c9c29dda1d9136; PR #55 at 2acc3e7 re-review https://github.com/redjhawk/pricetracker/pull/55#pullrequestreview-5434973956: 0 comments)

| Comment (link) | Label | Decision (blocking / non-blocking) | Reason | Outcome |
|---|---|---|---|---|
| https://github.com/redjhawk/pricetracker/pull/57#pullrequestreview-5434993904 (G5) | - | resolved | Verified at origin/docs/workflow-pr-triage: role lines 19 (record `## Merge` before first merge, commit to last PR branch, nothing committed after merges start) and 24 (post-merge results only on issue comment and run report) implement the two-moment split. | closed |
| https://github.com/redjhawk/pricetracker/pull/57#pullrequestreview-5434993904 (decisions.md:24 stale refs) | [nit] | non-blocking | Verified: `doc/changes/2026-10-06-2138-workflow-pr-triage/decisions.md:24` cites `pr-review-triage.md`:19–21 for the retarget/ancestor-check evidence; line 19 is now the `## Merge` record bullet, the cited content is at lines 20–21. Documentation cross-reference only, no behavior effect. | todo T-L |

#### Todos for non-blocking comments

- **T-L** `triage-decisions-stale-line-refs` - "Fix stale line references in workflow-pr-triage decisions.md". Problem: decisions.md:24 cites `pr-review-triage.md`:19–21; the retarget and ancestor-check text is now at lines 20–21. Suggested work: change the reference to `pr-review-triage.md`:20–21 (or cite by bullet text instead of line numbers). Source: https://github.com/redjhawk/pricetracker/pull/57#pullrequestreview-5434993904.

## Merge

- No blocking comments remain: yes, at round 3 (G1-G4 resolved per #55 re-review 5434973956, G5 resolved per #57 re-review 5434993904; 0 bugs; only non-blocking nit T-L)
- Planned merge order (PR, base, method):
  1. #55 (part 1) -> `feature/workflow-pr-review`, `gh pr merge 55 --merge --delete-branch`
  2. #57 (part 2) -> `feature/workflow-pr-review` (retarget with `gh pr edit 57 --base feature/workflow-pr-review` if it still targets the deleted part-1 branch), `gh pr merge 57 --merge --delete-branch`
  3. Part 3, the todo files PR (docs/workflow-pr-todos) -> `feature/workflow-pr-review` after retarget, `gh pr merge <n> --merge --delete-branch`
4. Ancestor check (`git merge-base --is-ancestor` for both part heads in `origin/feature/workflow-pr-review`), then create the final PR `gh pr create --base master --head feature/workflow-pr-review` (body lists #55, #57 in merge order with `Closes #<n>`), merge with `gh pr merge <n> --squash --delete-branch`
- Merge results and deploy: reported on the originating issue comment and the final run report (not committed)

## Todo files and issues created

All 12 non-blocking groups (T-A..T-L) are in `doc/todo/2026-10-06-*.md`, with issues #58-#69. The todo files are in a separate documentation PR (part 3), because adding them to #57 would take it over 500 changed lines. All issue creations succeeded.

## Follow-up: part 4 (PR #71, retarget before merge)

During the merges, GitHub closed #57 when its base branch was deleted at the merge of #55. #57's commits reached the feature branch through #70 (ancestor check passed). PR #71 fixes the procedure.

### Round 1 (PR #71 at 3d9681a, review https://github.com/redjhawk/pricetracker/pull/71#pullrequestreview-5435024588)

| Comment (link) | Label | Decision | Reason | Outcome |
|---|---|---|---|---|
| https://github.com/redjhawk/pricetracker/pull/71#discussion_r4200844245 | [readability] | non-blocking | Edge cases are wording only. The existing "if any merge fails, stop" rule covers the failure case. | `doc/todo/2026-10-06-merge-retarget-edge-cases.md` |
| https://github.com/redjhawk/pricetracker/pull/71#discussion_r4200844251 | [readability] | non-blocking | The base is guaranteed by the explicit retarget before each merge. A stale base never reaches master. | `doc/todo/2026-10-06-merge-base-precheck.md` |
| https://github.com/redjhawk/pricetracker/pull/71#discussion_r4200844256 | [nit] | non-blocking | It adds no new path to master or a deploy. `gh pr edit:*` is already allowed. | `doc/todo/2026-10-06-strict-gh-api-allowlist.md` |

Merge: no blocking comments remain. Order: #71 into `feature/workflow-pr-review` (`--merge --delete-branch`; last part, no next part); ancestor check; create the final PR into `master`; `--squash --delete-branch`; then the deploy.
