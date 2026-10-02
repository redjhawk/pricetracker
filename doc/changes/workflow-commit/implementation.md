# Workflow commit stage implementation

Status: implemented; awaiting independent documentation review and verification
Role: expert developer

Functional source: the user's follow-up requests proper commits and a mandatory commit stage after all workflow steps pass, as relayed by the coordinator. The requested behavior is to commit completed, verified work before reporting completion.

Technical scope: documentation only. Frontend, backend and API are unaffected because this change defines repository delivery procedure and adds no application behavior or HTTP contract. All four project skills were read for the initial implementation handoff; no refactoring is needed.

Changed `AGENTS.md` to add coordinator stage 8. Changed `doc/workflow/WORKFLOW.md` to add the `commit` state, final-diff gates, explicit scoped staging, staged whitespace checking, descriptive messages, failure reporting, a no-commit user override, and completion requiring successful commits. Added `doc/workflow/templates/commit.md` for gate evidence, staged scope, checks and outcome reporting. The template avoids self-referential commit hashes and permits actual success to be reported in the final handoff after Git confirms the commit.

No application files changed for this workflow addition. No commit, amend or push was performed by the implementation role. The coordinator must verify the final staged scope after independent review/adjudication and documentation QA, then execute the authorized commits. Push remains separately gated by an explicit user request.
