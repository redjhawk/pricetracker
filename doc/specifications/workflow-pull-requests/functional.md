# Workflow pull requests: functional specification

Status: ready
Source: user requirements confirmed in issue #3.
Scope: autonomous development workflow (`AGENTS.md`, `doc/workflow/WORKFLOW.md` stage 8, commit template). Application behavior is not affected.

## Requirements

- **FR-WORKFLOW-PR-001** In autonomous development, the agent creates pull requests from the work branch to `master` at the end of the workflow, once the review stage (review and review decisions) and QA have passed (QA ordering superseded by [workflow-stage-order](../workflow-stage-order/functional.md)).
- **FR-WORKFLOW-PR-002** Pull requests are opened ready for review, not as drafts.
- **FR-WORKFLOW-PR-003** The agent writes each pull request's description.
- **FR-WORKFLOW-PR-004** Each pull request targets 450 changed lines with a ±50 margin when splitting; 500 changed lines is the only enforced limit. Smaller PRs are acceptable when the change or a separate-kind PR is smaller; unrelated work is never combined to reach the target.
- **FR-WORKFLOW-PR-005** Changed lines are added plus deleted lines. Specifications and documentation count; generated files (for example `package-lock.json`) do not count.
- **FR-WORKFLOW-PR-006** A change exceeding 500 changed lines is split into several pull requests.
- **FR-WORKFLOW-PR-007** Split pull requests are stacked: PR 1 targets `master`, PR 2 targets PR 1's branch, and so on.
- **FR-WORKFLOW-PR-008** Refactoring is always its own pull request, placed before the feature when better done first, or after it when found during or after development.
- **FR-WORKFLOW-PR-009** Dependency updates are always their own pull request.
- **FR-WORKFLOW-PR-010** Formatting changes are always their own pull request.
- **FR-WORKFLOW-PR-011** Documentation related to a feature goes with that feature's pull request; unrelated documentation updates get their own pull request.

## Acceptance criteria

- `AGENTS.md`, `WORKFLOW.md` and the commit template state FR-WORKFLOW-PR-001 to 011 consistently and no longer forbid pushing in autonomous runs.
- When run tooling cannot create the required branches or PRs, the planned split is recorded and reported as a limitation, never claimed as done.
