# Review decisions: workflow-adjudication-report

Reviewed specification revisions: [functional](../../specifications/workflow-adjudication-report/functional.md), [technical](../../specifications/workflow-adjudication-report/technical.md) (working tree)
Reviewed code revision: uncommitted working tree on `docs/workflow-adjudication-report` (base `c5bedc4`)
Reviewer: independent reviewer agent ([review](review.md))
Adjudicator: independent review adjudicator agent (separate from developer and reviewer)

## Findings and decisions

### REV-001: Stage 7 and QA template do not mention executing critical-fix use cases

- Evidence: originally only stage 6 and `.agents/roles/qa-tester.md` required QA to execute critical-fix use cases. Now `doc/workflow/WORKFLOW.md` stage 7 (line 109) covers "every use case created for a fixed critical finding in stage 6", and `doc/workflow/templates/qa.md` line 20 requires "a case for every use case created for a fixed critical finding".
- Impact and scenario: a QA agent following only WORKFLOW.md or the template could skip the regression use case, so a critical fix would go unchecked at the interface level. The role file already reduced the likelihood.
- Criticality: non-critical, because the role file already required it and stage 6 named the duty. This was a consistency gap, not a missing obligation.
- Disposition: fix.
- Reason: a one-sentence change makes FR-WORKFLOW-ADJ-004 / TS-WORKFLOW-ADJ-002 consistent across every QA entry point.
- Specification decision: not applicable.
- Resolution: fixed. I checked the on-disk WORKFLOW.md stage 7 and templates/qa.md.
- Follow-up: none.

### REV-002: Pre-existing stage 6 constraints sat under the "Decision report (informational, never a gate)" paragraph

- Evidence: in `doc/workflow/WORKFLOW.md` stage 6, the sentences "The decision agent cannot override user-defined functionality ... refactoring returns to the technical agent." now end the critical-finding paragraph (line 99). The "Decision report" paragraph (line 101) contains only reporting rules.
- Impact and scenario: under the informational label, readers could have treated hard constraints on the decision agent as optional.
- Criticality: non-critical, because the wording stayed the same and the role file restates the constraints. The risk was misreading, not a change in rules.
- Disposition: fix.
- Reason: this restores the original placement at no cost, and it removes any implication that the constraints were weakened.
- Specification decision: not applicable.
- Resolution: fixed. I checked the paragraph structure on disk.
- Follow-up: none.

### REV-003: Completion section omits the new stage 6 handoff conditions

- Evidence: `doc/workflow/WORKFLOW.md` Completion (line 137) now requires "every fixed critical finding has a passing use case and regression test" and "the decision summary is published". The closing user-summary sentence does not mention the decision summary by name. Stage 6 (line 101) already states that the summary is "included in the final run report".
- Impact and scenario: without these conditions, Completion could be read as allowing completion before the stage 6 gates are met. That remaining risk is now gone.
- Criticality: non-critical, because the stage 6 handoff already gated these conditions.
- Disposition: fix (the optional final-summary wording was not added).
- Reason: the gating conditions were added. The final-report inclusion of the decision summary is already a normative stage 6 statement, so repeating it in the closing sentence would only duplicate it. Leaving it out has no gating or behavioral effect.
- Specification decision: not applicable.
- Resolution: fixed. I checked Completion on disk.
- Follow-up: none.

## Decision summary (published, informational)

| Finding | Criticality | Disposition | What was done |
|---|---|---|---|
| REV-001 Stage 7 / QA template missing critical-fix use cases | non-critical | fix | Stage 7 and templates/qa.md now require executing every use case created for a fixed critical finding; verified on disk |
| REV-002 Stage 6 constraints under "informational" paragraph | non-critical | fix | Constraints moved back to the critical-finding paragraph; the Decision report paragraph now holds only reporting rules; verified on disk |
| REV-003 Completion omits stage 6 handoff conditions | non-critical | fix | Completion now requires a passing use case and regression test for fixed critical findings, and a published decision summary; the optional summary-sentence wording was not added because stage 6 already says the summary goes in the final report |

Published as an issue comment: no originating issue — requested in a local session. Included in the final run report: yes.

## Release readiness

No critical findings were raised and none are open. All three non-critical findings are fixed and verified against the working tree. There are no pending functional questions. Application QA is not applicable to this documentation-only change; verification is document and role consistency.
