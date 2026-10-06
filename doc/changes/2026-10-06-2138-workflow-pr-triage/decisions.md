# Review decisions: workflow PR comment triage

Reviewed specification revisions: `doc/specifications/workflow-pr-triage/functional.md`, `technical.md` (working tree, 2026-10-06)
Reviewed code revision: uncommitted working tree on `docs/workflow-pr-reviewer` (WORKFLOW.md, AGENTS.md, `.agents/roles/pr-review-triage.md`, `.agents/roles/pr-reviewer.md`, `doc/workflow/templates/pr-review.md`, `doc/workflow/templates/pr-triage.md`, `.github/workflows/ai-dev.yml`)
Reviewer: independent reviewer agent ([review.md](review.md))
Adjudicator: independent review adjudicator agent (separate from developer and reviewer)

## Findings and decisions

### REV-001: Stacked merge order merges later parts into the wrong branch

- Evidence: part PR k≥2 has base `ai-dev/issue-<n>-<part k-1>`. Without branch deletion GitHub does not retarget it, so `gh pr merge` would merge part 2 into part 1's branch and the final PR would ship only part 1.
- Impact and scenario: any split change. The final squash to `master` would silently omit parts 2..N, and those parts would be deployed as missing while the issue closes.
- Criticality: critical, because incomplete work would reach production silently and the issue would close.
- Disposition: fix.
- Reason: this is a correctness defect in the merge procedure.
- Specification decision: not applicable (technical; TS-WORKFLOW-TRIAGE-005 updated).
- Resolution: fixed. The fix has three layers, so it does not depend on GitHub's retargeting alone:
  1. Every merge uses `--delete-branch`. When the base branch is deleted, GitHub retargets dependent PRs to the merged PR's base (the feature branch). See WORKFLOW.md stage 10 "Merge" (line 168), role lines 19–20, the ai-dev prompt, and TS-005.
  2. Before each part merge the agent checks that the part targets the feature branch and retargets it if not. `gh pr edit` is allowed, and WORKFLOW line 141 documents `gh pr edit --base`.
  3. Before the final merge the agent checks `git merge-base --is-ancestor` for every part. `Bash(git merge-base:*)` and `git fetch origin:*` are in allowedTools, so the remote refs can be fetched first.
  A failed merge stops the run, so the agent cannot proceed to the final merge with parts missing. I confirmed all four documents state the same commands.
- Follow-up: none.
- Critical fix confirmation: the evidence is the consistent text in WORKFLOW.md:168, `pr-review-triage.md`:19–21, ai-dev.yml:86–87 (prompt and allowedTools), and technical.md TS-005. No use case or regression test was added. The rule exists to verify application behavior, and this change has none: it only changes agent instructions and CI prompt text. The repository has no harness that could run a GitHub stacked-merge scenario, and an invented test would verify nothing. The ancestor check is the runtime regression guard: it executes on every real split run and blocks the final merge if a part is missing. The first real split ai-dev run must confirm that the feature branch contains every part before the final merge, and that confirmation must be recorded in its `pr-triage.md`. Recorded as a justified non-application, not a waiver of a functional requirement.

### REV-002: ai-dev.yml header still says the user merges

- Evidence: ai-dev.yml lines 2–4 now say Claude merges after review/triage and starts ci-deploy, because merges made with `GITHUB_TOKEN` trigger no workflows.
- Impact and scenario: a maintainer reading the file would be misled.
- Criticality: non-critical, because the comment does not affect behavior.
- Disposition: fix.
- Reason: consistency criterion of FR-WORKFLOW-TRIAGE-006.
- Specification decision: not applicable.
- Resolution: fixed (verified on disk).
- Follow-up: none.

### REV-003: pr-review template has stale "Blocking bugs reported to the user" and "@claude"-only re-reviews

- Evidence: the template headings are now `Possible bugs` and `Re-reviews after fixes (stage 10 blocking fixes or @claude fixes)`. The reporting section has been removed.
- Impact and scenario: the stale template would have reintroduced a stage 9 reporting duty.
- Criticality: non-critical, because it is template wording only.
- Disposition: fix.
- Reason: consistency with stages 9 and 10.
- Specification decision: not applicable.
- Resolution: fixed (verified on disk).
- Follow-up: none.

### REV-004: Size check and todo commits on PR branches

- Evidence: WORKFLOW stage 10, the role (line 14) and TS-003 now say a todo is committed on the PR branch, or on a separate documentation PR if it would exceed 500 lines. Commits that only add todo files need no re-review.
- Impact and scenario: without this, a todo commit could fail the size check or leave the merged SHA different from the reviewed one without explanation.
- Criticality: non-critical, because the run fails visibly instead of shipping wrong code.
- Disposition: fix.
- Reason: it removes the ambiguity cheaply.
- Specification decision: not applicable.
- Resolution: fixed (verified). Remaining risk: the end-of-run size check lists only open PRs, so it no longer measures PRs that are already merged. Parts are measured before they merge, so this is accepted.
- Follow-up: none.

### REV-005: Failed merge path unspecified

- Evidence: WORKFLOW:168 says that if a merge fails (branch protection, conflicts, a required review), the agent records the error, informs the user, and stops without merging the rest. The role (line 21), the prompt and TS-005 say the same.
- Impact and scenario: an unspecified failure path could have led to partial merges.
- Criticality: non-critical, because `gh pr merge` fails loudly.
- Disposition: fix.
- Reason: it also backs up REV-001.
- Specification decision: not applicable.
- Resolution: fixed (verified).
- Follow-up: none.

### REV-006: `@claude` follow-up runs now auto-merge

- Evidence: WORKFLOW "After the run" (line 40) now states that an `@claude` comment ends with the PR reviewer and triage, so a fix run can merge and deploy automatically once nothing blocks. TS-005 says the same.
- Impact and scenario: users are now told about the behavior explicitly.
- Criticality: non-critical. The behavior follows the user's stated requirement that the agent merges automatically, so it is not a functional gap.
- Disposition: fix (documentation).
- Reason: it makes the consequence explicit.
- Specification decision: not applicable; it is covered by FR-WORKFLOW-TRIAGE-006.
- Resolution: fixed (verified).
- Follow-up: none.

### REV-007: Todo naming vs. existing todo file

- Evidence: TS-003, the role and WORKFLOW now say "in the format of the existing todo files, plus a date prefix". The existing file is not renamed.
- Impact and scenario: the wording was inaccurate. Behavior is unaffected.
- Criticality: non-critical (nit).
- Disposition: fix.
- Reason: wording accuracy.
- Specification decision: not applicable.
- Resolution: fixed (verified).
- Follow-up: none.

## Decision summary (published, informational)

| Finding | Criticality | Disposition | What was done |
|---|---|---|---|
| REV-001 Stacked merges land in wrong branch | critical | fix | Every merge now uses `--delete-branch` (GitHub retargets the next part). The agent checks or retargets each part's base and verifies every part with `git merge-base --is-ancestor` before the final merge. A failed merge stops the run. Consistent in WORKFLOW, role, ai-dev prompt and allowedTools, and TS-005. No use case or regression test: this documentation-only change has no application behavior or test harness, and the ancestor check guards each real run |
| REV-002 ai-dev header says user merges | non-critical | fix | Header updated to say the agent merges and dispatches ci-deploy |
| REV-003 Stale pr-review template sections | non-critical | fix | Reporting section removed; re-reviews section generalized |
| REV-004 Todo commits vs size check / re-review | non-critical | fix | Todo goes to a separate docs PR if over 500 lines; todo-only commits need no re-review |
| REV-005 Failed merge path | non-critical | fix | A failed merge is recorded and reported, and stops the merging |
| REV-006 `@claude` runs auto-merge | non-critical | fix | Stated explicitly in WORKFLOW "After the run" |
| REV-007 Todo naming wording | non-critical | fix | Wording now says "existing format plus a date prefix" |

Published as an issue comment: no originating issue — requested in a local session. Included in the final run report: yes.

## Release readiness

All seven findings are fixed and verified on disk, and no critical finding remains open. REV-001's use case and regression test requirement is recorded as not applicable to this documentation-only change, with the reasoning above. No functional questions are pending. Application QA is not required (no application behavior changed); verify documentation consistency only.
