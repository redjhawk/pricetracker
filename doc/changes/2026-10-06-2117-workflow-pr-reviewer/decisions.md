# Review decisions: 2026-10-06-2117-workflow-pr-reviewer

Reviewed specification revisions: [functional](../../specifications/workflow-pr-reviewer/functional.md), [technical](../../specifications/workflow-pr-reviewer/technical.md) (TS-WORKFLOW-PRR-001..005), working tree
Reviewed code revision: uncommitted working-tree snapshot on `docs/workflow-pr-reviewer` (base `fb36cf6`)
Reviewer: independent stage 5 reviewer agent ([review](review.md))
Adjudicator: independent review adjudicator agent (distinct from developer and reviewer)

## Findings and decisions

### REV-001: Allowlist grants far more than posting reviews

- Evidence: the reviewed `ai-dev.yml` allowed `Bash(gh api repos/redjhawk/pricetracker/pulls/*)`, which prefix-matches merge (`pulls/<n>/merge --method PUT`), close, and dismissal calls. FR-WORKFLOW-PRR-006 requires comment-only behavior.
- Impact and scenario: an ai-dev run could merge a PR into `master`, which deploys, or close/dismiss reviews, bypassing the user's decision.
- Criticality: non-critical for this documentation/configuration change, because it was caught before commit and no run has used it; it would have been a significant permission over-grant had it shipped.
- Disposition: fix.
- Reason: narrowing costs nothing and enforces the comment-only requirement technically, not only by prompt.
- Specification decision: not applicable (technical; recorded as TS-WORKFLOW-PRR-005).
- Resolution: fixed. `ai-dev.yml` line 84 now allows only `Bash(gh api repos/redjhawk/pricetracker/pulls/*/reviews --method POST --input *)`; the role and the system prompt use exactly that argument order. Remaining note: TS-WORKFLOW-PRR-004 still quotes the old broad pattern, but TS-WORKFLOW-PRR-005 explicitly narrows it; acceptable, cosmetic.
- Follow-up: none.

### REV-002: Role omits failure handling for inline positions and own-PR refusal

- Evidence: `.agents/roles/pr-reviewer.md` previously lacked `side` guidance, the in-diff line constraint, and a concrete 422 fallback procedure.
- Impact and scenario: in every ai-dev run the token authors the PRs, so REQUEST_CHANGES always gets 422; an out-of-diff line also rejects the whole review. Without guidance the review might never be posted.
- Criticality: non-critical, because the failure is visible (no review URL) and completion requires a posted review, so it cannot silently pass; still worth fixing for reliability.
- Disposition: fix.
- Reason: makes FR-WORKFLOW-PRR-005/007 achievable on the first attempt.
- Specification decision: not applicable.
- Resolution: fixed. "How to report" now specifies `side` `RIGHT`/`LEFT`, that commented lines must be in the diff or go in the body, and on 422 to resubmit as `COMMENT` with the `CHANGES REQUESTED: ...` prefix. Reflected in TS-WORKFLOW-PRR-005 and the ai-dev prompt.
- Follow-up: none.

### REV-003: WORKFLOW.md introduction does not list the new role

- Evidence: `doc/workflow/WORKFLOW.md` line 3 omitted pull request review.
- Impact and scenario: readers of the introduction miss the new independent role; documentation inconsistency only.
- Criticality: non-critical, documentation consistency.
- Disposition: fix.
- Reason: one-phrase fix.
- Specification decision: not applicable.
- Resolution: fixed. Line 3 now ends "..., QA, and pull request review."
- Follow-up: none.

### REV-004: Behavior on follow-up `@claude` fix pushes was unspecified

- Evidence: section 9 and the ai-dev prompt did not say whether the PR reviewer runs again after `@claude` fixes; a `CHANGES REQUESTED` COMMENT review cannot be dismissed, so it would remain the latest blocking signal.
- Impact and scenario: after fixes, the PR would still display an unresolved "do not merge" review with no record of resolution.
- Criticality: non-critical, unspecified behavior with a stale-signal consequence, no data or deployment risk.
- Disposition: fix (technical default recorded).
- Reason: the user's functional answer was "comment only; user decides fixes". Re-running the reviewer on the updated diff stays within that answer: the re-review still only comments, edits nothing, pushes nothing, and triggers no automatic fixes; the user still decides any further fixes via `@claude`. It only refreshes the review state the user relies on, which is needed because the COMMENT fallback cannot be dismissed. It is therefore a technical completion of the posting mechanism, not a new functional decision, and needs no user question.
- Specification decision: not applicable; recorded in TS-WORKFLOW-PRR-005.
- Resolution: fixed. WORKFLOW.md section 9, the role ("After the user's `@claude` fixes are pushed, review that PR again in full ... states which earlier comments are resolved and which remain"), the ai-dev prompt, and the `pr-review.md` template ("Re-reviews after @claude fixes") all state it consistently. The prompt still says "Do not fix its comments unless the user asks with @claude".
- Follow-up: none.

### REV-005: No template for `pr-review.md`

- Evidence: section 9 and the role required `doc/changes/<change>/pr-review.md` without a template.
- Impact and scenario: inconsistent records across changes.
- Criticality: non-critical, suggestion.
- Disposition: fix.
- Reason: cheap and consistent with other stage artifacts.
- Specification decision: not applicable.
- Resolution: fixed. `doc/workflow/templates/pr-review.md` added (per-PR revision, URL, event, label counts, blocking bugs, re-reviews) and linked from the role.
- Follow-up: none.

## Decision summary (published, informational)

| Finding | Criticality | Disposition | What was done |
|---|---|---|---|
| REV-001 Allowlist grants merge/close via pulls API | non-critical | fix | Allowlist narrowed to `pulls/*/reviews --method POST --input *`; role and prompt use that exact form |
| REV-002 Missing inline-position and 422 fallback handling | non-critical | fix | Role specifies `side`, in-diff lines, and COMMENT resubmission with `CHANGES REQUESTED` prefix on 422 |
| REV-003 WORKFLOW.md intro omits new role | non-critical | fix | Intro now lists pull request review |
| REV-004 Re-review after `@claude` fixes unspecified | non-critical | fix | PR reviewer re-runs (comment only) after fix pushes and records resolved/remaining comments; within "comment only; user decides fixes" |
| REV-005 No `pr-review.md` template | non-critical | fix | Template added and linked from the role |

Published as an issue comment: no originating issue — requested in a local session. Included in the final run report: yes.

## Release readiness

No critical findings; all five findings are fixed and verified against the working tree by the adjudicator. No functional questions pending. Cosmetic residue: TS-WORKFLOW-PRR-004 quotes the superseded broad allowlist pattern (superseded by TS-WORKFLOW-PRR-005). Documentation-only change: application QA not applicable; stage 8 commit/PR and stage 9 remain for the coordinator.
