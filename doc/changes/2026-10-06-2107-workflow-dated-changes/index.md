# Workflow dated changes

Stage: commit.
User functional source: request of 2026-10-06 and the user's answers (rename all existing folders; datetime = start of the change).
Technical scope: `WORKFLOW.md`, `AGENTS.md`, ai-dev prompt (folder naming, plus `date -u` allowed), rename of all 20 existing change folders and rewrite of the links to them. No refactoring; API unchanged. `FUNCTIONAL_SPECIFICATIONS.md` is not updated because no product behavior changes.

## Artifacts

- [Functional](../../specifications/workflow-dated-changes/functional.md), [technical](../../specifications/workflow-dated-changes/technical.md)
- [Review](review.md), [decisions](decisions.md), [commit step](commit-step.md)

## Verification so far

- `ls doc/changes` sorts chronologically.
- A grep finds no old folder names.
- The relative-link check over all Markdown reports 1 broken link before the change and the same 1 after: `doc/specifications/leboncoin-session-file-removal/technical.md -> deprecated/README.md`, which is pre-existing and unrelated.
