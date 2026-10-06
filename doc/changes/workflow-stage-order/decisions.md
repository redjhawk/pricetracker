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

## Second review round: REV-003 to REV-007

## REV-003: Overview stage order contradicts AGENTS.md

- Criticality: would be critical if true, because AGENTS.md is the top-level instruction. On this branch it is not true.
- Disposition: reject. The finding does not match the branch.
- Evidence: `AGENTS.md:22` says QA runs "before any commit or pull request". `AGENTS.md:23` says "after review decisions and QA pass for the final diff, create focused commits". `AGENTS.md:25` says "QA (7) runs before stage 8 commits and opens the PRs". `grep "without waiting for QA" AGENTS.md` finds no match. The reviewer quoted the pre-change text, which is also the text in the session's injected context snapshot. Commit `c37cc7f` already updated the file. AGENTS.md, WORKFLOW.md section 7, and the overview give the same order.
- Status: closed. No action needed.

## REV-004: "Always" claim conflicts with the ai-dev system prompt

- Criticality: noncritical. The conflict exists before this change. Its effect is that CI runs might stop to ask for approvals they do not need. That is conservative behavior, with no risk of data loss or incorrect code.
- Disposition: defer.
- Evidence: `ai-dev.yml:85` still says "When a product decision, API contract approval or refactoring decision is needed, ask it in the issue and stop". AGENTS.md and WORKFLOW.md "Autonomy and the functional gate" allow only functional questions to the user. The overview "Always" paragraph repeats AGENTS.md correctly. The divergent source is the CI prompt, which is a workflow-configuration file outside this documentation change's scope (FR-WORKFLOW-ORDER).
- Follow-up: a separate change should align the `--append-system-prompt` text in `.github/workflows/ai-dev.yml` with the functional-only gate.
- Status: deferred, not fixed.

## REV-005: Stacked PR `Closes #<n>` claim

- Criticality: noncritical (informational overview text).
- Disposition: fix.
- Evidence: the coordinator reworded the overview after the review. It now says: "GitHub closes the issue automatically only when that PR merges into `master`; a stacked PR merged into another branch does not close it." This matches GitHub's behavior: closing keywords take effect only on merges into the default branch.
- Status: fixed and verified.

## REV-006: ci-deploy trigger conditions omitted

- Criticality: noncritical.
- Disposition: reject.
- Evidence: the reviewer's impact claim ("in this repository the deploy job is skipped") is wrong. `git remote get-url origin` returns `https://github.com/redjhawk/pricetracker.git`, so the guard `github.repository == 'redjhawk/pricetracker'` (`ci-deploy.yml:22`) is true for this repository. The `master` trigger (`ci-deploy.yml:9`) is the one that applies, because master is the default branch. The overview's "a push to `master` triggers `ci-deploy`" is therefore accurate here. The overview is declared informational, and the numbered sections and workflow file prevail, so the runner and guard details are not needed.
- Status: closed. No change.

## REV-007: Unverifiable rationale for no CI on pull requests

- Criticality: noncritical.
- Disposition: reject.
- Evidence: the rationale has a source. `ci-deploy.yml:11` says: "No pull_request trigger: the repo is public and must never run fork code on barcelona." Neither workflow declares `pull_request`. `ai-dev.yml` uses `pull_request_review_comment`, which is a comment event, so the sentence "No CI runs on pull requests" stands. The point that `@claude` comments can trigger ai-dev on barcelona-dev is a separate security question about `ai-dev.yml`. It falls outside this change and is not contradicted by the overview.
- Status: closed. No change.

## Second-round outcome

No critical findings remain unresolved. REV-005 is fixed. REV-003, REV-006, and REV-007 are rejected on evidence. REV-004 is deferred to a separate ai-dev.yml change. Every finding the adjudicator decided to fix is fixed. No functional questions are pending.
