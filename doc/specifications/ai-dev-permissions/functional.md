# ai-dev permissions: functional specification

Status: ready
Source: user request of 2026-10-07: check that the ai-dev pipeline has every right the agents need to complete the workflow, and fix what is missing.
Scope: `.github/workflows/ai-dev.yml`. Application behavior is not affected.

## Requirements

- **FR-AIDEV-PERM-001** An ai-dev run can execute every command that the workflow (`AGENTS.md`, `WORKFLOW.md`, `.agents/roles/`) requires for stages 1–10.
- **FR-AIDEV-PERM-002** A right that the pipeline cannot be given is recorded as a limitation, and the run reports it to the user instead of failing silently.

## Acceptance criteria

- Every documented command has a matching allowlist entry or a documented working alternative.
- The `.github/workflows` push limitation is stated in the workflow file and in the prompt.
