# Workflow stage order

Stage: commit.
User functional source: request of 2026-10-06.
Technical scope: documentation only; no refactoring; API contract unchanged. `FUNCTIONAL_SPECIFICATIONS.md` is not updated because product behavior does not change.

## Artifacts

- [Functional specification](../../specifications/workflow-stage-order/functional.md)
- [Technical specification](../../specifications/workflow-stage-order/technical.md)
- [Review](review.md), [decisions](decisions.md), [commit step](commit-step.md)
- QA: documentation consistency checks only (no application behavior changed); see commit step.

## Follow-up: overview section

At the user's request, `WORKFLOW.md` gained an informational "Overview: from issue to production" section. It summarizes stages 1–8 in the new order, the `ai-dev` trigger, and what happens after the run and after the merge (`ci-deploy`). The numbered sections still prevail.
