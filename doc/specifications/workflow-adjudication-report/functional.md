# Workflow adjudication report: functional specification

Status: ready
Source: user request of 2026-10-06, with answers to clarifying questions: show the summary as an issue comment and in the final run report; for a fixed critical finding, add a use case file and an automated test.
Scope: development workflow documents and the `ai-dev` CI prompt. Application behavior is not affected.

## Requirements

- **FR-WORKFLOW-ADJ-001** After review decisions are final, the adjudicator's decisions are published as an informational summary: for each finding, its ID, criticality, disposition, and what was done (or why not).
- **FR-WORKFLOW-ADJ-002** The summary is posted as a comment on the originating issue and included in the final run report.
- **FR-WORKFLOW-ADJ-003** Publishing the summary does not block the workflow; it does not wait for a user reply. Critical findings still block completion until they are fixed.
- **FR-WORKFLOW-ADJ-004** For each fixed critical (blocking) finding, the adjudicator confirms the fix with evidence, and the change adds a use case under `doc/use-cases/` and an automated regression test that exercises it. QA executes that use case.

## Acceptance criteria

- `AGENTS.md`, `WORKFLOW.md` (stage 6 and the overview), the adjudicator, developer and QA roles, and the decisions template state FR-WORKFLOW-ADJ-001..004 consistently.
- The `ai-dev` run is allowed to comment on the issue, and its prompt tells Claude to post the summary without stopping.
