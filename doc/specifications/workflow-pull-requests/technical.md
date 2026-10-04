# Workflow pull requests: technical specification

Status: ready
Functional source: [functional.md](functional.md)

## Tiers

- Frontend: not affected (documentation-only workflow change).
- Backend: not affected (documentation-only workflow change).
- API: not affected; `API_SPECIFICATION.md` unchanged.

## Design

- **TS-WORKFLOW-PR-001** (FR-WORKFLOW-PR-001..003) Rewrite `doc/workflow/WORKFLOW.md` stage 8 as "Coordinator commits and pull requests": after review decisions pass, commit, push the work branch, and open ready (non-draft) PRs to `master` with agent-written descriptions; QA follows and its fixes are pushed to the same branches. Agent-written descriptions contain purpose, requirement IDs, scope, verification and limitations, and, for stacked PRs, position in the stack and base. Add the `pull-request` feature stage after `commit` and before `qa`, and update Completion.
- **TS-WORKFLOW-PR-002** (FR-WORKFLOW-PR-004..011) Add sizing and splitting rules to stage 8: count added + deleted lines with `git diff --numstat`, excluding generated files; target 450±50, ceiling 500; stacked bases; separate PRs for refactoring, dependency updates, formatting and unrelated documentation.
- **TS-WORKFLOW-PR-003** Tooling limitation: `.github/workflows/ai-dev.yml` lets the agent commit and run only `git push origin HEAD`, forbids creating or switching branches, and a CI step opens a single PR for the issue branch after the agent finishes. When tooling prevents the required branches/PRs, the coordinator records the planned split (order, branch and base, changed-line counts, descriptions) in `commit-step.md` and reports it as a limitation. `.github/` is not changed by this change.
- **TS-WORKFLOW-PR-004** Add a pull-request section to `doc/workflow/templates/commit.md` and replace its no-push sentence; align `AGENTS.md` stage 8.

Refactoring: none.

## Verification

Read-through consistency of `AGENTS.md`, `WORKFLOW.md` and the template; `git diff --check`; changed-line count of this change.
