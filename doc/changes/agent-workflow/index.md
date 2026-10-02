# Agent workflow setup

Scope: repository instructions, six expert role prompts, staged workflow, and reusable specification/review/QA templates. The user request is the functional source for this documentation-only change; no product functionality or API changes are introduced.

Status: complete. Five review findings corrected and independently rechecked; documentation QA passed.

## Handoffs

- Functional workflow writer: `workflow_specifier`; [workflow](../../workflow/WORKFLOW.md).
- Technical role designer: `technical_roles`; [role prompts](../../../.agents/roles/).
- Documentation integration: coordinating agent.
- Independent reviewer: `workflow_reviewer`; [report](review.md).
- Independent adjudicator: `workflow_adjudicator`; [decisions](decisions.md).
- Independent documentation QA: `documentation_qa`; [verification](qa.md). Application interface testing is not applicable to this documentation-only change.

## User decisions

The request authorizes this setup and distinct specialist agents. The existing API is preserved; no new contract confirmation or refactoring decision is needed.

## Verification

Check role and workflow consistency, required stages and handoffs, and local Markdown links. Review corrections must be independently rechecked before completion. Role prompts require explicit agent invocation; they are not runtime registration.
