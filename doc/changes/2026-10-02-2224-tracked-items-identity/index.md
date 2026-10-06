# Tracked item identity layout

Stage: complete. Build, independent review, adjudication, and eight executed browser QA groups passed. No application findings or unresolved decisions.
Scope: move marketplace into the tracked list's Item column, before the article listing ID; remove the dedicated Marketplace column.
User source: current request to execute every WORKFLOW.md stage and write each stage result in Markdown.

## Stage results

1. [Functional specification](../../specifications/tracked-items-identity/functional.md) and [handoff](functional-step.md): `marketplace_functional`.
2. [Technical specification](../../specifications/tracked-items-identity/technical.md) and [handoff](technical-step.md): `marketplace_technical`.
3. [API contract result](api-step.md): `marketplace_technical`; existing approved API preserved.
4. [Implementation](implementation.md): `marketplace_developer`.
5. [Review](review.md): `workflow_adjudicator`, serving only as independent reviewer for this change.
6. [Decisions](decisions.md): `/root`, independent adjudicator; distinct from component developer and reviewer.
7. [QA](qa.md): `documentation_qa`, independent interface tester; actual URLs and exploratory cases.
8. [Commit handoff](commit-step.md): coordinating agent, after all preceding checks passed.

## Decisions and blockers

No refactoring requested. No API confirmation is needed if this remains a frontend presentation change using the existing marketplace and listingId fields. Track any newly discovered blocker here before advancing dependent work.

Agent allocation: the runtime refused additional threads after the specification and implementation stages. Available distinct agents are reused for review and QA; the coordinating agent adjudicates independently of the implementation and review agents. Historical agent names do not change their assigned role for this change.

Final result: marketplace and external listing ID now share the secondary identity line below the existing clickable title; the table has five columns. No API change or refactoring was required. All workflow stage results are linked above. The subsequent user request authorizes committing this completed change; pushing is not requested in that instruction.
