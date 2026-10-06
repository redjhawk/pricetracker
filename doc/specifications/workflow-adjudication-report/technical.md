# Workflow adjudication report: technical specification

Status: ready
Requirements: [functional](functional.md).

- **TS-WORKFLOW-ADJ-001** (FR-WORKFLOW-ADJ-001..004) `WORKFLOW.md`: stage 6 adds the requirement for a critical fix confirmation (use case + regression test), the "Decision report" paragraph and the handoff; the overview row 6 is updated. `AGENTS.md` step 6 is aligned.
- **TS-WORKFLOW-ADJ-002** Roles: the adjudicator confirms critical fixes and writes the summary; the developer writes the use case and the regression test; QA executes the use case.
- **TS-WORKFLOW-ADJ-003** `templates/review-decisions.md`: a per-finding "Critical fix confirmation" line and a "Decision summary" table.
- **TS-WORKFLOW-ADJ-004** `.github/workflows/ai-dev.yml`: allow `Bash(gh issue comment:*)` and tell Claude to post the summary after the decisions without stopping. The job token already has `issues: write`.
- Frontend, backend and API: not affected (workflow only). The ai-dev prompt's request for API or refactoring approval (REV-004 of workflow-stage-order) is out of scope.
- Verification: consistency grep, `git diff --check`, and a YAML parse of ai-dev.yml.
