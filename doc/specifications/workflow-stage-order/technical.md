# Workflow stage order: technical specification

Status: ready
Requirements: [functional](functional.md).

- **TS-WORKFLOW-ORDER-001** (FR-WORKFLOW-ORDER-001) Add the `FUNCTIONAL_SPECIFICATIONS.md` update to stage 1 in `WORKFLOW.md` and its handoff, to `AGENTS.md` step 1, and to `.agents/roles/functional-specifier.md`. Skip it only when no product behavior changes, and record why.
- **TS-WORKFLOW-ORDER-002** (FR-WORKFLOW-ORDER-002, 003) Reorder the feature stages to `... review-decisions, qa, commit, pull-request, complete`. Update stage 7, stage 8, `AGENTS.md` steps 7–8 and the ordering paragraph, the commit template, and the QA role. Amend FR-WORKFLOW-PR-001.
- Frontend, backend and API: not affected, because this changes documentation only. `.github/workflows/ai-dev.yml` is unchanged because its prompt defers to `WORKFLOW.md` for stage order.
- Verification: grep the docs for leftover "after PR"/"do not wait for QA" wording and run `git diff --check`.
