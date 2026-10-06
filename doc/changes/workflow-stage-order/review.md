# Review: workflow stage order

Reviewer: independent reviewer agent
Reviewed revision: working tree on `docs/workflow-qa-before-pr` (uncommitted diff plus untracked `doc/specifications/workflow-stage-order/` and `doc/changes/workflow-stage-order/`).
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
