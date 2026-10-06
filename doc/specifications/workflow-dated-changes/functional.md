# Workflow dated changes: functional specification

Status: ready
Source: user request of 2026-10-06, with answers to clarifying questions: rename all existing folders; the datetime is when the change started.
Scope: `doc/changes/` naming and the workflow documents. Application behavior is not affected.

## Requirements

- **FR-WORKFLOW-DATE-001** Each change folder under `doc/changes/` starts with the datetime at which the change started, so that listing the folders orders them chronologically.
- **FR-WORKFLOW-DATE-002** When the change has a work item (GitHub issue), its ID follows the datetime in the folder name.
- **FR-WORKFLOW-DATE-003** The datetime records the start of the change and does not change afterwards.
- **FR-WORKFLOW-DATE-004** All existing change folders are renamed to this format, and links to them keep working.

## Acceptance criteria

- `ls doc/changes` lists the changes in chronological order.
- Every folder whose change has a known issue includes `issue-<n>`.
- No Markdown link or path in the repository points to an old folder name.
