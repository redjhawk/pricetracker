# Review: workflow adjudication report

Reviewer: independent reviewer agent. Scope: uncommitted diff on `docs/workflow-adjudication-report` (AGENTS.md, WORKFLOW.md, templates/review-decisions.md, developer/qa-tester/review-adjudicator roles, ai-dev.yml) against [functional](../../specifications/workflow-adjudication-report/functional.md) and [technical](../../specifications/workflow-adjudication-report/technical.md).

Checks run: `git diff --check` (clean). YAML parse not possible here (`python3 -m yaml` unavailable); visual inspection shows the changed lines are single quoted-string arguments with no new quotes or colons breaking structure. `issues: write` is present in the job permissions (line 34), so TS-WORKFLOW-ADJ-004 holds. `doc/use-cases/README.md` exists.

All of FR-WORKFLOW-ADJ-001..004 are covered in AGENTS.md step 6, WORKFLOW.md overview row 6 and stage 6, the three roles, the decisions template and the ai-dev prompt/allowedTools. No unspecified behavior found. Findings below are non-blocking consistency gaps.

## REV-001 Stage 7 and QA template do not mention executing critical-fix use cases

- Requirement: FR-WORKFLOW-ADJ-004 ("QA executes that use case"), TS-WORKFLOW-ADJ-002.
- Location: `doc/workflow/WORKFLOW.md` section 7; `doc/workflow/templates/qa.md`.
- Evidence: only stage 6 ("QA executes the use case in stage 7") and `.agents/roles/qa-tester.md` state it; the stage 7 text, its handoff, and the QA template have no corresponding duty or field.
- Impact: a QA agent following WORKFLOW.md stage 7 or the template alone may skip the use case. Low (the role file covers it).
- Suggested action: add one sentence to stage 7 (and optionally its handoff) and a QA template line for critical-fix use cases.
- Provisional severity: non-critical.

## REV-002 Pre-existing stage 6 sentences moved into the "Decision report" paragraph

- Requirement: unspecified (readability / scope).
- Location: `doc/workflow/WORKFLOW.md` stage 6, "Decision report (informational, never a gate): ..." paragraph.
- Evidence: the existing sentences "The decision agent cannot override user-defined functionality ... refactoring returns to the technical agent." previously followed the critical-finding sentence; they now end the paragraph labelled "Decision report (informational, never a gate)".
- Impact: these are hard constraints on the decision agent; placing them under an "informational, never a gate" label can be read as weakening them. Low.
- Suggested action: move those sentences back to the critical-finding paragraph (or their own paragraph).
- Provisional severity: non-critical.

## REV-003 Completion section omits the new stage 6 handoff conditions

- Requirement: FR-WORKFLOW-ADJ-001/002/004 (consistency acceptance criterion mentions stage 6 and overview only).
- Location: `doc/workflow/WORKFLOW.md` "Completion".
- Evidence: Completion lists review outcomes, no critical blocker, QA, commits, PRs, but not "decision summary published" or "fixed critical findings have use case and regression test", which stage 6 handoff now requires.
- Impact: minor inconsistency; stage 6 handoff already gates it. Also note Completion's final user summary could mention the decision summary per FR-WORKFLOW-ADJ-002 ("included in the final run report").
- Suggested action: optionally add both conditions to Completion and mention the decision summary in the final summary sentence.
- Provisional severity: non-critical.

## Notes (not findings)

- AGENTS.md step 7 on disk says QA runs "before any commit or pull request", consistent with WORKFLOW.md and the stacked base `docs/workflow-qa-before-pr`; the loaded session copy of AGENTS.md differs, but the disk file is authoritative for this branch.
- The ai-dev prompt still asks to stop for API contract/refactoring approval; out of scope per technical spec (pre-existing REV-004 of workflow-stage-order).
