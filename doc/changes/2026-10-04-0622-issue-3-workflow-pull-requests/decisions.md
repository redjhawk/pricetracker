# Review decisions: workflow pull requests (issue #3)

Reviewed specification revisions: [functional.md](../../specifications/workflow-pull-requests/functional.md), [technical.md](../../specifications/workflow-pull-requests/technical.md) (working tree, status ready)
Reviewed code revision: uncommitted working tree on `ai-dev/issue-3-20261004-0616`, base `3cbd3a6`
Reviewer: independent reviewer agent ([review.md](review.md))
Adjudicator: independent review adjudicator agent (separate from developer and reviewer)

User requirements (issue #3): PRs created after review without waiting for QA; ready, not draft; agent-written descriptions; 450±50 changed lines (added + deleted, specifications count, generated files excluded); over 500 must be split; stacked splits; refactoring, dependency updates, formatting and unrelated documentation in separate PRs.

## Findings and decisions

### REV-001: Feature stage order contradicts "PR before QA"

- Evidence: `doc/workflow/WORKFLOW.md` line 21 lists `review-decisions, qa, commit, pull-request, complete`; stage 8 of the same file and `AGENTS.md` stage 8 say PRs are opened once review decisions pass, without waiting for QA. FR-WORKFLOW-PR-001 requires PR creation not to wait for QA.
- Impact and scenario: an agent or coordinator tracking progress by the lifecycle states (the canonical status list used in change indexes) will advance `qa` before `commit`/`pull-request`, i.e. wait for QA before opening PRs. This directly violates the first and central user requirement of issue #3. Likelihood is high because the lifecycle is the explicit ordered state machine, and two documents in the same file then disagree, so the outcome depends on which sentence the agent reads.
- Criticality: critical, because the delivered workflow text contradicts an explicit user functional requirement (FR-WORKFLOW-PR-001) on the core behavior the change exists to introduce; the documentation is the deliverable, so an internal contradiction on that point is a functional defect, not a style issue.
- Disposition: fix.
- Reason: a one-line reorder removes the contradiction; there is no tradeoff.
- Required change: in `WORKFLOW.md` line 21 change the order to `... review, review-decisions, commit, pull-request, qa, complete`. Completion already requires QA to pass, so `complete` stays last. Ensure TS-WORKFLOW-PR-001 ("Add the `pull-request` feature stage") states the position (after `commit`, before `qa`).
- Specification decision: not applicable; the user requirement is unambiguous.
- Resolution: resolved; reviewer recheck confirms `WORKFLOW.md` line 21 orders `review-decisions, commit, pull-request, qa, complete`, consistent with stage 8, `AGENTS.md` and TS-WORKFLOW-PR-001.
- Follow-up: none after fix. Owner: developer.

### REV-002: Stage numbering/handoff still places QA (7) before commits (8)

- Evidence: `AGENTS.md` lists QA as 7 and coordinator commits/PRs as 8 and says "Run stages in dependency order"; `WORKFLOW.md` stage 7 says QA runs "after review corrections". Stage 8 text in both files explicitly says not to wait for QA.
- Impact and scenario: a reader inferring order from numbering could run QA first. Stage 8 explicitly resolves the order and "dependency order" is satisfied (stage 8 depends on stage 6 only), so misreading is less likely than REV-001; once REV-001 is fixed the lifecycle also states the order.
- Criticality: non-critical, because the governing stage 8 text is explicit and correct and, after REV-001, the lifecycle agrees; the residual issue is clarity, not a contradiction of the requirement.
- Disposition: fix.
- Reason: cheap, removes residual ambiguity on the core requirement. Renumbering stages is rejected as unnecessary churn (role files and prior change records reference stage numbers); a note suffices.
- Required change: add one sentence to `WORKFLOW.md` stage 7 (and a short clause in `AGENTS.md` item 7 or the "Run stages in dependency order" sentence) stating that QA runs after stage 8 pushes and opens the PRs, and that accepted QA fixes are pushed to the same PR branches. Keep numbering.
- Specification decision: not applicable.
- Resolution: resolved; reviewer recheck confirms `WORKFLOW.md` stage 7 and the `AGENTS.md` dependency-order sentence state QA runs after PRs open; numbering kept.
- Follow-up: none after fix. Owner: developer.

### REV-003: Commit template gate list still treats QA as a pre-commit gate

- Evidence: `doc/workflow/templates/commit.md` line 11 keeps "Required QA report and passed results (QA may run after PR creation)" in the list introduced as gates to record and pass before committing.
- Impact and scenario: a coordinator filling the template may block the commit/PR until QA passes, contradicting FR-WORKFLOW-PR-001; the parenthetical reduces but does not remove the ambiguity.
- Criticality: non-critical, because the parenthetical and stage 8 indicate QA may follow, so the likely reading is correct; however the template is the operational checklist and should not suggest otherwise.
- Disposition: fix.
- Reason: small edit that aligns the checklist with the requirement.
- Required change: remove the QA line from the pre-commit gate list and add it to (or after) the "Pull requests" section as a post-PR item, e.g. "QA report and passed results (after PR creation; fixes pushed to the PR branches)", which remains required for completion.
- Specification decision: not applicable.
- Resolution: resolved; reviewer recheck confirms the QA line moved from the pre-commit gates to a post-PR item in the template.
- Follow-up: none after fix. Owner: developer.

### REV-004: PR description content list not in requirements

- Evidence: `WORKFLOW.md` stage 8 prescribes description contents "(purpose, requirement IDs, scope, verification, limitations)"; FR-WORKFLOW-PR-003 only requires that the agent writes descriptions, and TS-WORKFLOW-PR-001 does not mention the list.
- Impact and scenario: unspecified behavior; it does not conflict with any requirement and improves description quality. Risk is only traceability (behavior not backed by a specification).
- Criticality: non-critical, because it is additive guidance with no conflict with user requirements and no adverse effect.
- Disposition: fix (specification update, not removal).
- Reason: description content is a technical/process decision the agents may make autonomously; keeping it is beneficial, but it must be recorded so implemented behavior is specified. Dropping it would lose useful guidance without benefit.
- Required change: add to TS-WORKFLOW-PR-001 that agent-written descriptions contain purpose, requirement IDs, scope, verification and limitations (and, for stacked PRs, position in the stack and base). No change to `WORKFLOW.md` required.
- Specification decision: not applicable (technical decision, no user input needed).
- Resolution: resolved; reviewer recheck confirms TS-WORKFLOW-PR-001 records the description contents. Residual workflow-text mismatch tracked as REV-007.
- Follow-up: REV-007. Owner: technical specifier / developer.

### REV-005: Hard ceiling of 500 vs. "450±50" lower bound not addressed

- Evidence: `WORKFLOW.md` stage 8 Size bullet and FR-WORKFLOW-PR-004 state target 450±50 and ceiling 500, without saying whether PRs under 400 are allowed.
- Impact and scenario: an agent could treat 400 as a minimum and merge unrelated work, pad changes, or refuse to open a small PR; that would conflict with FR-WORKFLOW-PR-008..011, which require separate (often small) PRs, and with small whole changes such as this one. The user said only that changes over 500 must be split, so 400 cannot be a hard minimum.
- Criticality: non-critical, because the most plausible reading is correct and the separate-PR rules already force small PRs; the ambiguity is wording.
- Disposition: fix.
- Reason: clarification follows necessarily from the user's stated rules (only exceeding 500 triggers a split; separate kinds always get their own PR), so it is not a new functional decision and needs no user question.
- Required change: in the `WORKFLOW.md` Size bullet (and FR-WORKFLOW-PR-004 wording), state that 450±50 is the sizing target when splitting, 500 is the only enforced limit, and smaller PRs are acceptable when the change or a separate-kind PR (refactoring, dependencies, formatting, unrelated docs) is smaller. Never combine unrelated work to reach the target.
- Specification decision: not applicable; derived from issue #3 statements.
- Resolution: resolved in `WORKFLOW.md` per reviewer recheck (500 is the only enforced limit; smaller PRs acceptable). Residual `AGENTS.md` wording tracked as REV-006.
- Follow-up: REV-006. Owner: developer (with functional spec wording).

### REV-006: `AGENTS.md` still reads 450±50 as a mandatory range

- Evidence: `AGENTS.md` stage 8 says "Keep each PR at 450±50 changed lines (hard ceiling 500, ...)"; `WORKFLOW.md` after REV-005 says 450±50 is the split target, 500 the only enforced limit, and smaller PRs are acceptable.
- Impact and scenario: an agent reading only `AGENTS.md` may treat 400 as a minimum, bundling unrelated work or refusing small PRs, which conflicts with FR-WORKFLOW-PR-008..011 and with the user's rule that only changes over 500 must be split.
- Criticality: non-critical, because `WORKFLOW.md`, the detailed stage text `AGENTS.md` directs agents to follow, is correct; the residual is an inconsistent summary phrase.
- Disposition: fix.
- Reason: one-phrase edit completing the intent of REV-005; leaving two entry documents inconsistent on sizing invites misreading.
- Required change: reword `AGENTS.md` stage 8 to "target 450±50 changed lines when splitting; 500 is the hard ceiling (generated files excluded)" or equivalent.
- Specification decision: not applicable.
- Resolution: resolved; reviewer recheck confirms `AGENTS.md` stage 8 reads "Target 450±50 changed lines per PR when splitting; 500 is the hard ceiling", consistent with `WORKFLOW.md` and the commit template.
- Follow-up: none. Owner: developer.

### REV-007: Stacked-PR description content missing from `WORKFLOW.md`

- Evidence: TS-WORKFLOW-PR-001 lists "for stacked PRs, position in the stack and base" among description contents; `WORKFLOW.md` stage 8 omits it.
- Impact and scenario: specification and workflow text disagree; reviewers of a stacked PR may lack its position and base, making merge order less clear. No user requirement is violated (FR-WORKFLOW-PR-003 only requires agent-written descriptions).
- Criticality: non-critical, because it is a low-consequence technical-guidance mismatch with no conflict with user requirements.
- Disposition: fix (add to `WORKFLOW.md`; keep the specification).
- Reason: the item supports FR-WORKFLOW-PR-007 stacking by making merge order explicit; adding it is preferable to dropping useful, already-specified guidance.
- Required change: extend the `WORKFLOW.md` stage 8 list to "(purpose, requirement IDs, scope, verification, limitations, and for stacked PRs their position in the stack and base)".
- Specification decision: not applicable.
- Resolution: resolved; reviewer recheck confirms the `WORKFLOW.md` stage 8 description list includes "for stacked PRs their position in the stack and base", matching TS-WORKFLOW-PR-001.
- Follow-up: none. Owner: developer.

## Release readiness

- All findings REV-001..007 are resolved per reviewer recheck; no critical or open finding remains.
- Next: documentation QA (consistency, `git diff --check`, changed-line count), then stage 8.
- No functional questions for the user; no API or refactoring routing required.
