# Review: workflow PR comment triage

Reviewer: independent reviewer agent (not the developer or adjudicator), 2026-10-06.
Scope: uncommitted diff of AGENTS.md, WORKFLOW.md, `.agents/roles/pr-reviewer.md`, new `.agents/roles/pr-review-triage.md`, `doc/workflow/templates/pr-triage.md`, `.github/workflows/ai-dev.yml`, `doc/specifications/workflow-pr-reviewer/functional.md`, against `doc/specifications/workflow-pr-triage/{functional,technical}.md`.
Checks run: `git diff`, reading `ci-deploy.yml` (has `workflow_dispatch`, so `gh workflow run` works), the ai-dev safety-net and size-check steps, `doc/todo/`, and `templates/pr-review.md`.

FR-WORKFLOW-TRIAGE-001..006 are each described in AGENTS.md step 10, the WORKFLOW overview row and stage 10, the role, and the ai-dev prompt. FR-WORKFLOW-PRR-006/007 are marked superseded. `actions: write` and the three new allowed tools match the documented commands exactly (`gh issue create:*`, `gh pr merge:*`, `gh workflow run ci-deploy.yml --ref master`).

## REV-001 Stacked merge order merges later parts into the wrong branch

- Requirement: FR-WORKFLOW-TRIAGE-006, TS-WORKFLOW-TRIAGE-005.
- Location: WORKFLOW.md stage 10 "Merge"; `pr-review-triage.md` merge bullet; ai-dev.yml prompt.
- Evidence: in the documented stack, part PR k (k ≥ 2) has `--base ai-dev/issue-<n>-<part k-1>`, not the feature branch. `gh pr merge <part1> --merge` merges part 1 into the feature branch, but PR 2's base stays part 1's branch. GitHub only retargets dependent PRs automatically when the merged head branch is deleted (`--delete-branch`), which is not documented. Then `gh pr merge <part2> --merge` merges part 2 into the part-1 branch, never into the feature branch; the final feature→master PR ships only part 1.
- Expected: each part lands in the feature branch before the final PR merges.
- Suggested action: either merge with `gh pr merge <n> --merge --delete-branch` (GitHub then retargets the next part to the feature branch), or `gh pr edit <next> --base ai-dev/issue-<n>-feature` before each next merge; and verify the feature branch contains all parts before merging the final PR. Document the chosen command and keep allowedTools consistent.
- Provisional severity: critical (silently ships incomplete work to production).

## REV-002 ai-dev.yml header comment still says the user merges

- Requirement: FR-WORKFLOW-TRIAGE-006 (consistency acceptance criterion).
- Location: `.github/workflows/ai-dev.yml` lines 2–4: "You review and merge the PR; merging to master (the default branch) triggers ci-deploy."
- Expected: the agent merges, and its merges do not trigger ci-deploy (it dispatches it).
- Suggested action: update the comment.
- Provisional severity: minor.

## REV-003 pr-review template still has "Blocking bugs reported to the user" and "@claude"-only re-reviews

- Location: `doc/workflow/templates/pr-review.md` sections "Blocking bugs reported to the user" and "Re-reviews after @claude fixes".
- Evidence: the diff removed the reporting duty from stage 9 and the role, and re-reviews now follow triage fix rounds too.
- Suggested action: drop or rename the first section (reporting is stage 10), and rename the second to "Re-reviews after fixes".
- Provisional severity: minor.

## REV-004 Size check and todo commits on PR branches

- Location: WORKFLOW stage 10 non-blocking bullet; ai-dev "Check pull request sizes" step.
- Evidence: todo files are committed on the PR branch, adding lines to a part PR already near the 500-line ceiling; the size check runs at the end of the run and would fail the run (or, if PRs are already merged, measures nothing since it lists only open PRs). Each todo commit also changes the PR head after its stage 9 review, so the reviewed SHA differs from the merged one.
- Suggested action: state that todo commits are documentation and either put them on the final/feature PR or exclude `doc/todo/` from the size count; state that a todo-only commit needs no re-review.
- Provisional severity: minor.

## REV-005 Merge with an open REQUEST_CHANGES review / branch protection not addressed

- Location: stage 10 "Merge".
- Evidence: if a REQUEST_CHANGES review was accepted (not refused) on a PR, or the repository requires approvals, `gh pr merge` fails. In ai-dev runs the 422 fallback makes this unlikely, but the failure path (merge refused) is not specified: stop or report?
- Suggested action: say that a failed merge is reported to the user and stops the run, recording the error in `pr-triage.md`.
- Provisional severity: minor.

## REV-006 `@claude` follow-up runs now auto-merge

- Location: ai-dev prompt "After pushing @claude fixes to a PR, run the PR reviewer and the triage again on that PR"; WORKFLOW "After the run".
- Evidence: triage ends in merge, so any user `@claude` comment on a still-open PR can lead to an automatic merge and deploy. This is consistent with the user's "agent merges automatically", but it is not stated explicitly; a user commenting a question with `@claude` may not expect a merge.
- Suggested action: state explicitly in WORKFLOW "After the run" that an `@claude` fix run ends with triage and merge when nothing blocks. Question for adjudication, not a defect.
- Provisional severity: question.

## REV-007 Todo naming differs from the existing todo file

- Location: TS-WORKFLOW-TRIAGE-003; existing `doc/todo/modal-focus-containment.md` has no date prefix and a `Reproduce` section.
- Evidence: "in the existing todo format" with a new `<YYYY-MM-DD>-` prefix; harmless, but the spec calls it existing.
- Suggested action: say "new files use a date prefix"; no rename of the existing file.
- Provisional severity: nit.

No other spec mismatches or unspecified behavior found. The `GITHUB_TOKEN` rationale (merges trigger no workflows; `workflow_dispatch` exists on ci-deploy; `actions: write` needed) is correct.
