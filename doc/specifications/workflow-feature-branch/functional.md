# Workflow feature branch: functional specification

Status: ready
Source: user request of 2026-10-06.
Scope: development workflow and the `ai-dev` CI prompt and size check. Application behavior is not affected.

## Requirements

- **FR-WORKFLOW-FB-001** When a change is split into stacked PRs, the agent first creates a feature branch as a copy of `master`.
- **FR-WORKFLOW-FB-002** The part PRs merge into the feature branch, not into `master`.
- **FR-WORKFLOW-FB-003** A single final PR merges the feature branch into `master`. It is merged last, so the deploy runner starts only once per feature.

## Acceptance criteria

- `AGENTS.md`, `WORKFLOW.md` (overview and stage 8), the commit template and the ai-dev prompt describe FR-WORKFLOW-FB-001..003 consistently.
- The ai-dev size check does not fail because of the aggregate final PR.
