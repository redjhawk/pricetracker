# Workflow pull requests (issue #3)

Stage: review-decisions; fixes for REV-001..005 applied per [decisions](decisions.md), reviewer recheck pending, then documentation QA and commit. See [review](review.md).
User functional source: issue #3 (PRs after review, ready not draft, agent-written descriptions, 450±50 changed lines with a 500 ceiling, stacked splits, separate refactoring/dependency/formatting/unrelated-docs PRs).
Technical scope: documentation only; frontend, backend and API unaffected. No refactoring. `.github/` unchanged.

## Artifacts

- [Functional specification](../../specifications/workflow-pull-requests/functional.md)
- [Technical specification](../../specifications/workflow-pull-requests/technical.md)
- Changed: `AGENTS.md` (stage 8), `doc/workflow/WORKFLOW.md` (feature stages, stage 8, Completion), `doc/workflow/templates/commit.md`.

## Known limitation

`.github/workflows/ai-dev.yml` permits only `git push origin HEAD`, forbids new branches, and opens one PR via CI. Stacked splits in such runs are recorded as a planned split in `commit-step.md` and reported, not claimed as created.
