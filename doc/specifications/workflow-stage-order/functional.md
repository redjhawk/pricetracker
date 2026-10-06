# Workflow stage order: functional specification

Status: ready
Source: user request of 2026-10-06 (update `FUNCTIONAL_SPECIFICATIONS.md` in stage 1; run QA before commits and pull requests).
Scope: development workflow documents only. Application behavior is not affected.

## Requirements

- **FR-WORKFLOW-ORDER-001** In stage 1, the functional specifier updates `doc/FUNCTIONAL_SPECIFICATIONS.md` with a summary of new or changed behavior and a link to each subject file, in addition to writing the subject files.
- **FR-WORKFLOW-ORDER-002** QA runs before any commit, push or pull request. Commits and pull requests are created only after review decisions and QA pass.
- **FR-WORKFLOW-ORDER-003** Accepted QA fixes go through review and decisions, then are retested, before committing.

## Acceptance criteria

- `AGENTS.md`, `WORKFLOW.md`, the commit template, and the functional-specifier and QA roles state FR-WORKFLOW-ORDER-001..003 consistently.
- No workflow document still says that PRs are opened before QA or that QA fixes are pushed to existing PR branches.
- FR-WORKFLOW-PR-001 refers to this subject for QA ordering.
