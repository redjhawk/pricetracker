# Review decisions: workflow stage order

Adjudicator: independent review adjudicator agent (separate from developer and reviewer).
Inputs: [review](review.md), [functional](../../specifications/workflow-stage-order/functional.md), [technical](../../specifications/workflow-stage-order/technical.md), working-tree diff on `docs/workflow-qa-before-pr`.
API contract: not affected (documentation-only change).

## REV-001: Stale QA-after-PR ordering in TS-WORKFLOW-PR-001

- Type: defect (specification inconsistency).
- Criticality: noncritical. It affects no users, data, or runtime behavior. Agents take stage order from `AGENTS.md` and `WORKFLOW.md`, which are correct. Still, it broke an explicit acceptance criterion of FR-WORKFLOW-ORDER-002 ("no workflow document still says PRs are opened before QA..."), so a fix was required before completion. Likelihood of misleading an agent was low to moderate, since agents do read the specifications.
- Disposition: fix.
- Action: TS-WORKFLOW-PR-001 now says its QA ordering is superseded by TS-WORKFLOW-ORDER-002 and gives the new stage order `qa`, `commit`, `pull-request`. The old "QA follows and its fixes are pushed to the same branches" and "before `qa`" text was removed.
- Owner: developer (applied); adjudicator (verified).
- Status: fixed and independently verified.
- Recheck evidence: diff of `doc/specifications/workflow-pull-requests/technical.md:14`. A grep for `wait for QA|after PR|same (PR )?branches|QA follows` across `AGENTS.md`, `doc/workflow`, `.agents` and `doc/specifications/workflow-*` finds only the verification instruction in the new technical spec, which is not a hit on stale wording.
- Remaining risk: none material. The superseded sentence keeps the history readable.

## REV-002: Commit template "Passed gates" omits QA

- Type: defect (template inconsistency with FR-WORKFLOW-ORDER-002).
- Criticality: noncritical. Without the fix, the template would not prompt the coordinator to confirm QA before committing. `WORKFLOW.md` stage 8 already requires that check, so the gap was a weaker prompt, not a missing rule. It had no effect on data or users.
- Disposition: fix.
- Action: the QA line is now under "Passed gates" in `doc/workflow/templates/commit.md`. Its old post-PR line, which included "fixes pushed to the PR branches", was removed.
- Owner: developer (applied); adjudicator (verified).
- Status: fixed and independently verified.
- Recheck evidence: template diff (line added at gates, line removed from the pull-request section). `git diff --check` is clean.
- Remaining risk: none.

## Additional adjudicator checks

- `WORKFLOW.md` Completion still lists "required QA passes" alongside commits and PRs. It does not state an order, so it is consistent with the new order and needs no change.
- QA for this change is documentation consistency only. AGENTS.md permits this for documentation-only workflow changes.

## Outcome

Both findings are noncritical, fixed, and verified. No critical or unresolved findings remain, and no functional questions are pending.
