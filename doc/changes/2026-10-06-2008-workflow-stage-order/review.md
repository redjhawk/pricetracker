# Review: workflow stage order

Reviewer: independent reviewer agent
Reviewed revision: working tree on `docs/workflow-qa-before-pr` (uncommitted diff plus untracked `doc/specifications/workflow-stage-order/` and `doc/changes/2026-10-06-2008-workflow-stage-order/`).
Sources: [functional](../../specifications/workflow-stage-order/functional.md), [technical](../../specifications/workflow-stage-order/technical.md).
Docs checked: `AGENTS.md`, `doc/workflow/**`, `.agents/roles/**`, `doc/specifications/workflow-pull-requests/**`, `.github/workflows/ai-dev.yml`.

FR-WORKFLOW-ORDER-001..003 are stated consistently in `AGENTS.md`, `WORKFLOW.md` (stage list, stages 1, 7, 8, Completion), the commit template, and the functional-specifier and QA roles. `ai-dev.yml` defers ordering to `WORKFLOW.md` and has no contradicting text. `git diff --check` is clean.

## REV-001 — Stale QA-after-PR ordering in TS-WORKFLOW-PR-001

- Requirement: FR-WORKFLOW-ORDER-002; acceptance criterion "No workflow document still says that PRs are opened before QA or that QA fixes are pushed to existing PR branches".
- Location: `doc/specifications/workflow-pull-requests/technical.md:14`.
- Evidence: "after review decisions pass, commit, push the work branch, and open ready (non-draft) PRs ... QA follows and its fixes are pushed to the same branches ... Add the `pull-request` feature stage after `commit` and before `qa`".
- Expected: the technical spec is marked superseded for QA ordering (as FR-WORKFLOW-PR-001 was) or amended to the new order.
- Actual: unchanged; contradicts WORKFLOW.md and the amended FR-WORKFLOW-PR-001.
- Impact: an agent reading the PR technical spec gets the old order. Not used as the runtime instruction, but it is a workflow document within the acceptance scope.
- Suggested action: add a supersession note or rewrite the QA sentences in TS-WORKFLOW-PR-001.
- Provisional severity: medium.

## REV-002 — Commit template "Passed gates" omits QA

- Requirement: FR-WORKFLOW-ORDER-002 (commits only after review decisions and QA pass).
- Location: `doc/workflow/templates/commit.md`, "Passed gates" section (lines 6–10); the QA line remains under the pull-request section (line 33).
- Expected: QA pass recorded as a gate before committing, alongside review decisions.
- Actual: gates list only specification consistency, API contract, and review; QA result is listed after PR links, implying post-PR recording.
- Impact: the template does not prompt the coordinator to confirm QA before committing; minor inconsistency.
- Suggested action: move the QA line into "Passed gates".
- Provisional severity: low.

## Observations (no finding)

- `index.md` skips the `FUNCTIONAL_SPECIFICATIONS.md` update with a reason (no product behavior change), as WORKFLOW.md stage 1 now allows.
- QA for this change is documentation consistency only, which AGENTS.md permits for documentation-only workflow changes.

## Review of the "Overview: from issue to production" section (uncommitted diff)

Claims checked against WORKFLOW.md sections 1–8, AGENTS.md, `.github/workflows/ai-dev.yml`, and `.github/workflows/ci-deploy.yml`. Stage table rows 1–8, the functional gate, trigger events, branch prefix, safety-net PR, PR size check (open PRs, 500, generated files excluded), and the ci-deploy steps (vet, test, build, reachability, ARMv6 deploy) match the sources.

## REV-003 — Overview stage order contradicts AGENTS.md

- Requirement: FR-WORKFLOW-ORDER-002 (QA before commits and PRs).
- Location: overview table rows 7–8; `AGENTS.md` "Team workflow" step 8 ("without waiting for QA") and the paragraph "QA (7) runs after stage 8 pushes and opens the PRs".
- Expected: AGENTS.md, WORKFLOW.md section 7, and the overview give one order.
- Actual: overview and section 7 put QA before commits/PRs; AGENTS.md (loaded first by every run, per the ai-dev prompt) still says PRs are opened without waiting for QA.
- Impact: agents receive contradictory instructions; AGENTS.md is read as the higher-level instruction.
- Suggested action: update AGENTS.md in this change to match, or record why it is out of scope.
- Provisional severity: high.

## REV-004 — "Always" claim conflicts with the ai-dev system prompt

- Location: overview "Always" paragraph; `ai-dev.yml` line 85 `--append-system-prompt`.
- Expected: only functional questions go to the user (WORKFLOW "Autonomy and the functional gate", AGENTS.md).
- Actual: the ai-dev prompt says "When a product decision, API contract approval or refactoring decision is needed, ask it in the issue and stop".
- Impact: the overview describes behavior the CI prompt contradicts; CI runs may stop for API/refactoring approval.
- Suggested action: align the ai-dev prompt (separate change) or note the discrepancy.
- Provisional severity: medium (pre-existing in ai-dev.yml; surfaced by the new claim).

## REV-005 — "The last PR's `Closes #<n>` closes the issue" is inaccurate for stacked PRs

- Location: overview "After the merge".
- Evidence: GitHub closing keywords act only when the PR merges into the default branch. In a stack, the last PR targets the previous PR's branch (section 8 "Stacking"), so merging it does not close the issue unless it was retargeted to `master` first.
- Suggested action: qualify ("when it merges into `master`") or drop.
- Provisional severity: low.

## REV-006 — ci-deploy trigger description omits conditions

- Location: overview "After the merge".
- Evidence: `ci-deploy.yml` triggers on push to `master` or `main` and the job runs only when `github.repository == 'redjhawk/pricetracker'` (header: "Copy to pricetracker"); it runs on the `barcelona-deploy` runner.
- Impact: in this repository the deploy job is skipped; the overview implies every push to `master` deploys.
- Suggested action: mention the repository guard, or state that it applies to the pricetracker repository.
- Provisional severity: low.

## REV-007 — Unverifiable claim about CI on pull requests

- Location: overview "After the run": "No CI runs on pull requests (the repository is public, so fork code never runs on barcelona)".
- Evidence: neither workflow has a `pull_request` trigger (true), but the rationale (public repository, fork safety) is not stated in any source checked; ai-dev does run on barcelona-dev for any comment containing `@claude` on a PR, with no author filter in the job `if` (lines 26–27; permission checks rely on claude-code-action defaults).
- Suggested action: keep the factual part ("no workflow triggers on pull_request") and drop or source the rationale.
- Provisional severity: low.
