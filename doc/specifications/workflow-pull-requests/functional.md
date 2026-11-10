# Workflow pull requests: functional specification

Status: ready
Source: user requirements confirmed in issue #3.
Scope: autonomous development workflow (`AGENTS.md`, `doc/workflow/WORKFLOW.md` stage 8, commit template). Application behavior is not affected.

## Requirements

- **FR-WORKFLOW-PR-001** In autonomous development, the agent creates pull requests from the work branch to `master` at the end of the workflow, once the review stage (review and review decisions) and QA have passed (QA ordering superseded by [workflow-stage-order](../workflow-stage-order/functional.md)).
- **FR-WORKFLOW-PR-002** Pull requests are opened ready for review, not as drafts.
- **FR-WORKFLOW-PR-003** The agent writes each pull request's description.
- **FR-WORKFLOW-PR-004** Removed: there is no changed-line limit or target per pull request; unrelated work is never combined in one PR.
- **FR-WORKFLOW-PR-005** Removed with FR-WORKFLOW-PR-004 (changed lines are no longer counted).
- **FR-WORKFLOW-PR-006** A change may be split into several pull requests when that makes it easier to review.
- **FR-WORKFLOW-PR-007** Split pull requests are stacked: PR 1 targets `master`, PR 2 targets PR 1's branch, and so on. (Base superseded by [workflow-feature-branch](../workflow-feature-branch/functional.md): PR 1 targets a feature branch, and a final PR merges that branch into `master`.)
- **FR-WORKFLOW-PR-008** Refactoring is always its own pull request, placed before the feature when better done first, or after it when found during or after development.
- **FR-WORKFLOW-PR-009** Dependency updates are always their own pull request.
- **FR-WORKFLOW-PR-010** Formatting changes are always their own pull request.
- **FR-WORKFLOW-PR-011** Documentation related to a feature goes with that feature's pull request; unrelated documentation updates get their own pull request.

## Acceptance criteria

- `AGENTS.md`, `WORKFLOW.md` and the commit template state FR-WORKFLOW-PR-001 to 011 consistently and no longer forbid pushing in autonomous runs.
- When run tooling cannot create the required branches or PRs, the planned split is recorded and reported as a limitation, never claimed as done.
