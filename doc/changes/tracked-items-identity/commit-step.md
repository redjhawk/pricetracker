# Commit handoff: tracked item identity

Coordinator: `/root`.
Authorization: subsequent user request to create proper commits after all workflow stages passed.

## Readiness and scope

Functional and technical specifications are ready; the API is unchanged. Independent review found no issues, adjudication recorded no unresolved findings, and all eight final browser QA groups passed. Production build and whitespace checks passed. No application edits occurred after review/QA.

Stage only `src/components/TrackedItemsPage.tsx`, the updated UC-TI-01 use case, `doc/specifications/tracked-items-identity/`, and `doc/changes/tracked-items-identity/`. The separate commit-workflow documentation change is excluded.

## Commit

Message: `Consolidate marketplace and listing ID in the Item column`.
Before committing, inspect the staged diff and run `git diff --cached --check`. Record the actual commit identifier in the coordinator's final response and Git history; this file is included in that commit and does not embed its own hash. A Git failure keeps the commit stage pending and must be reported; this handoff alone is not proof of commit success.

Push is not included in the current request.
