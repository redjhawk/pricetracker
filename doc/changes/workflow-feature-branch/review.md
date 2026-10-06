# Review: workflow-feature-branch

Reviewer: independent reviewer agent. Scope: uncommitted diff (AGENTS.md, WORKFLOW.md, templates/commit.md, workflow-pull-requests/functional.md, .github/workflows/ai-dev.yml) and new specs, checked against FR-WORKFLOW-FB-001..003 and TS-WORKFLOW-FB-001..004.

FR-WORKFLOW-FB-001..003 are covered consistently in AGENTS.md stage 8, the WORKFLOW.md overview and stage 8, the commit template and the ai-dev prompt. The size-check jq filter is correct: the second `select` drops only the exact head `ai-dev/issue-<n>-feature`, and `ai-dev/issue-<n>-featurex` parts are still checked. `ci-deploy.yml` triggers only on pushes to `master`/`main`, so merges into the feature branch do not deploy (TS note is correct). The roles in `.agents/roles/` do not mention PR bases, so they need no change. The allowedTools already cover `git checkout -b ai-dev/*`, `git push -u origin HEAD`, `gh pr create --base` and `gh pr edit --base`.

## REV-001 (low) ai-dev prompt: "from master" may not resolve in CI
- Requirement: FR-WORKFLOW-FB-001, TS-WORKFLOW-FB-001.
- Location: `.github/workflows/ai-dev.yml` `--append-system-prompt`, "first create `ai-dev/issue-<n>-feature` from master with `git checkout -b`".
- Evidence: the run is on the action's work branch. `git checkout master` is not in allowedTools, and the prompt does not give a start point. A bare `git checkout -b ai-dev/issue-<n>-feature` branches from the current HEAD, which may already hold the run's commits, so the result would not be a copy of master. A local `master` ref may also be stale or missing in the checkout.
- Suggested action: say explicitly `git fetch origin master` then `git checkout -b ai-dev/issue-<n>-feature origin/master`. Both are allowed by the existing `git fetch origin:*` and `git checkout -b ai-dev/*` patterns.

## REV-002 (low) Safety-net PR step ignores the feature branch
- Requirement: FR-WORKFLOW-FB-002/003 (unspecified for the safety net).
- Location: `.github/workflows/ai-dev.yml` step "Open pull request".
- Evidence: if the run's branch has commits but no PR, the step always opens it against the default branch with `Closes #<n>`. In a split run where Claude left the work branch without a PR, this would create a PR that bypasses the feature branch and deploys on merge. The technical spec does not mention this step.
- Suggested action: either record in the technical spec that the safety net is a fallback for single-PR runs, or skip it when `ai-dev/issue-<n>-feature` exists on the remote. Accept-as-is is reasonable because the step is a fallback.

## REV-003 (low) Leftover "pull requests to `master`" wording
- Location: `doc/workflow/WORKFLOW.md` line 123 ("open pull requests to `master`"); AGENTS.md stage 8 ("open ready (non-draft) pull requests to `master`"); `doc/specifications/workflow-pull-requests/technical.md` TS-WORKFLOW-PR-001/002 ("stacked bases", PRs to `master`) have no supersession note, unlike FR-WORKFLOW-PR-007.
- Evidence: the sentences right after now say part PRs target the feature branch, so readers get two bases.
- Suggested action: change it to "to `master` (directly, or through a feature branch when split)", and add a supersession pointer to TS-WORKFLOW-PR-002.

## REV-004 (low) Size measurement for the final PR is ambiguous
- Location: WORKFLOW.md stage 8, Size bullet ("Measure each PR with `scripts/pr-size.sh <base>` before opening it ... do not open or grow a PR it rejects") versus the Feature branch bullet (final PR exempt).
- Evidence: an agent following the Size bullet literally would refuse to open the final PR when the feature is over 500 lines.
- Suggested action: add "except the final feature-branch PR" to the Size bullet.

No critical or high findings.

## Overview diagram review (WORKFLOW.md "Split changes (feature branch)")

Checked against WORKFLOW.md stage 8, technical.md TS-WORKFLOW-FB-001..005 and `.github/workflows/ai-dev.yml`. Verified as accurate: single-PR changes target `master`; feature branch pushed first; PR 1 targets the feature branch and PR 2 targets PR 1 (diagram arrows); retargeting after a merge; final PR lists parts in merge order, is merged last, carries `Closes #<n>` and is exempt from the size check (ai-dev jq filter skips the exact `-feature` head); part PRs merge without deploying (`ci-deploy` runs only on `master`/`main` pushes); safety net uses `Part of #<n>` and base `ai-dev/issue-<n>-feature` when that branch exists.

## REV-005 (low) Diagram shows only the ai-dev branch name and origin/master start point
- Requirement: TS-WORKFLOW-FB-001.
- Location: WORKFLOW.md overview diagram, `ai-dev/issue-<n>-feature (exact copy of origin/master, pushed first)`.
- Evidence: stage 8 and TS-WORKFLOW-FB-001 use `feature/<change>` outside ai-dev runs; the `origin/master` start point is stated only for ai-dev (TS-WORKFLOW-FB-005, ai-dev prompt). The overview is informational and refers to stage 8, so it is not contradictory, only narrower.
- Suggested action: optionally annotate "(`feature/<change>` outside ai-dev)". Accept-as-is is reasonable.

## REV-006 (low) Safety-net bullet slightly overstates "whenever that branch exists"
- Location: WORKFLOW.md overview, third bullet.
- Evidence: ai-dev.yml applies the feature base only when `$BRANCH != ai-dev/issue-<n>-feature`. If the run's branch is the feature branch itself, the fallback PR targets `master` with `Closes #<n>`; that is the intended final PR, so `master` is not reached early by a part, but the wording "whenever that branch exists" omits this exception.
- Suggested action: say "for a part branch, whenever the feature branch exists". Accept-as-is is reasonable.

No critical or high findings for the overview diagram.
