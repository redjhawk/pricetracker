# Workflow PR comment triage: functional specification

Status: ready
Source: user request of 2026-10-06, with the answer to a clarifying question: the agent merges automatically.
Scope: development workflow, a new agent role, and the ai-dev CI configuration. Application behavior is not affected.

## Requirements

- **FR-WORKFLOW-TRIAGE-001** After the PR review, a new agent decides for each comment whether it is blocking.
- **FR-WORKFLOW-TRIAGE-002** A non-blocking comment is added to the `doc/todo/` folder.
- **FR-WORKFLOW-TRIAGE-003** For each non-blocking comment, a new issue is created on the project when possible. If creating the issue fails, the user is informed and the workflow does not stop.
- **FR-WORKFLOW-TRIAGE-004** Blocking comments return to the developer agent.
- **FR-WORKFLOW-TRIAGE-005** Review and triage are repeated until no blocking comment remains.
- **FR-WORKFLOW-TRIAGE-006** When no blocking comment remains, the agent merges the PRs automatically.

## Acceptance criteria

- `WORKFLOW.md`, `AGENTS.md`, the triage role and the ai-dev prompt describe FR-WORKFLOW-TRIAGE-001..006 consistently, and FR-WORKFLOW-PRR-006/007 are marked as superseded.
- An ai-dev run is allowed to create issues, merge PRs, and start the deploy after merging.
