# Review: workflow pull requests (issue #3)

Reviewer: independent reviewer agent (role `.agents/roles/reviewer.md`).
Revision reviewed: uncommitted working tree on `ai-dev/issue-3-20261004-0616` (base `3cbd3a6`), files `AGENTS.md`, `doc/workflow/WORKFLOW.md`, `doc/workflow/templates/commit.md`, `doc/specifications/workflow-pull-requests/{functional,technical}.md`, `doc/changes/workflow-pull-requests/index.md`.
Sources: issue #3 user requirements, functional and technical specifications above.

Overall: FR-WORKFLOW-PR-001..011 are each stated in `AGENTS.md` stage 8 and `WORKFLOW.md` stage 8; no requirement is missing. The findings below are ordering inconsistencies and small additions beyond requirements. Adjudication is performed separately by the independent decision agent; severities are provisional.

## REV-001 Feature stage order contradicts "PR before QA"

- Requirement: FR-WORKFLOW-PR-001.
- Location: `doc/workflow/WORKFLOW.md` line 21 (stage lifecycle).
- Evidence: lifecycle lists `... review-decisions, qa, commit, pull-request, complete`, while stage 8 says commits and PRs happen once review decisions pass, without waiting for QA.
- Expected: lifecycle order reflecting commit and pull-request before (or independent of) `qa`, e.g. `review-decisions, commit, pull-request, qa, complete`.
- Actual: a reader following the lifecycle would wait for QA before creating PRs.
- Impact: contradictory guidance on the core user requirement.
- Suggested action: reorder the lifecycle list (and optionally note in stage 7 that QA runs after PR creation).
- Provisional severity: major.

## REV-002 Stage numbering/handoff still places QA (7) before commits (8)

- Requirement: FR-WORKFLOW-PR-001.
- Location: `AGENTS.md` lines 22-23 ("8. Coordinator: after review decisions pass ..."); `WORKFLOW.md` stage 7 ("after review corrections") and stage 8 heading; AGENTS.md "Run stages in dependency order".
- Evidence: stage 8 now depends only on stage 6, but numbering and "run stages in dependency order" still imply stage 7 precedes stage 8. Stage 8 text resolves it explicitly, so this is a clarity issue rather than a hard contradiction.
- Suggested action: add a short note in stage 7 or AGENTS.md that QA runs after PR creation and its fixes are pushed to the PR branches.
- Provisional severity: minor.

## REV-003 Commit template gate list still treats QA as a pre-commit gate

- Requirement: FR-WORKFLOW-PR-001.
- Location: `doc/workflow/templates/commit.md` line 11, under the gates "Record preparation and passed gates before committing".
- Evidence: "Required QA report and passed results (QA may run after PR creation)" remains in the pre-commit gate list; the parenthetical softens but leaves it ambiguous whether it must be filled before committing.
- Suggested action: state that this item is completed after QA (later commit/push) or move it to a post-PR section.
- Provisional severity: minor.

## REV-004 PR description content list not in requirements

- Requirement: unspecified (FR-WORKFLOW-PR-003 only says the agent writes descriptions).
- Location: `doc/workflow/WORKFLOW.md` stage 8, "(purpose, requirement IDs, scope, verification, limitations)".
- Evidence: prescribed description contents are an addition beyond the user requirement and are not in the functional or technical specification.
- Impact: low; reasonable guidance, but unspecified behavior per reviewer rules.
- Suggested action: either record it in TS-WORKFLOW-PR-001 as a technical decision or drop it.
- Provisional severity: minor.

## REV-005 Hard ceiling of 500 vs. "450±50" lower bound not addressed

- Requirement: FR-WORKFLOW-PR-004/006.
- Location: `WORKFLOW.md` stage 8 Size bullet; functional spec FR-004.
- Evidence: rules state target and ceiling but not whether PRs below 400 (e.g. a small refactoring PR or the whole change being small) are acceptable. Separate-PR rules (FR-008..011) necessarily create small PRs, so 400 cannot be a minimum.
- Suggested action: clarify that 450±50 is a target for split sizing and that only 500 is enforced; smaller PRs are fine. This is wording, not a new functional rule (the user said "exceeding 500 must be split").
- Provisional severity: minor (question/clarity).

## Checked without findings

- No-push sentences removed consistently from AGENTS.md, WORKFLOW.md stage 8 and the template.
- Stacking (FR-007), separate refactoring/dependency/formatting/unrelated docs PRs (FR-008..011), counting rule incl. specs and excluding generated files (FR-005) are consistent across files.
- Tooling-limitation handling (TS-WORKFLOW-PR-003) is consistent and never claims unconfirmed PRs; `.github/` untouched; no application, API or refactoring changes.
- Change size: well under 450 changed lines.

## Re-review after fixes

Revision: uncommitted working tree after developer fixes per [decisions.md](decisions.md).

- REV-001: resolved. `WORKFLOW.md` line 21 now orders `review-decisions, commit, pull-request, qa, complete`; consistent with stage 8, `AGENTS.md`, and TS-WORKFLOW-PR-001.
- REV-002: resolved. `WORKFLOW.md` stage 7 and the `AGENTS.md` "dependency order" sentence state QA runs after stage 8 opens PRs and fixes go to the same PR branches; numbering kept.
- REV-003: resolved. QA line removed from the pre-commit gate list in `templates/commit.md` and added to the Pull requests section as a post-PR item required for completion.
- REV-004: resolved. TS-WORKFLOW-PR-001 now records the description contents. See REV-007 for a residual mismatch.
- REV-005: resolved in `WORKFLOW.md` (500 is the only enforced limit; smaller PRs acceptable). See REV-006 for a residual in `AGENTS.md`.

## New findings

### REV-006 `AGENTS.md` still reads 450±50 as a mandatory range

- Requirement: FR-WORKFLOW-PR-004/006; follow-up of REV-005.
- Location: `AGENTS.md` stage 8, "Keep each PR at 450±50 changed lines (hard ceiling 500, ...)".
- Evidence: `WORKFLOW.md` now says 450±50 is the split target, 500 the only enforced limit, and smaller PRs are acceptable; `AGENTS.md` still says "keep each PR at 450±50", implying a 400 minimum, which separate-kind PRs cannot meet.
- Suggested action: reword, e.g. "target 450±50 changed lines when splitting; 500 is the hard ceiling".
- Provisional severity: minor.

### REV-007 Stacked-PR description content missing from `WORKFLOW.md`

- Requirement: TS-WORKFLOW-PR-001 (unspecified at functional level).
- Location: `WORKFLOW.md` stage 8, "(purpose, requirement IDs, scope, verification, limitations)".
- Evidence: TS-WORKFLOW-PR-001 adds "for stacked PRs, position in the stack and base"; the workflow text omits it.
- Suggested action: add that item to the `WORKFLOW.md` list, or drop it from the technical specification.
- Provisional severity: minor.

## Re-review of REV-006 and REV-007

- REV-006: resolved. `AGENTS.md` stage 8 now reads "Target 450±50 changed lines per PR when splitting; 500 is the hard ceiling", consistent with `WORKFLOW.md` stage 8 and the commit template.
- REV-007: resolved. `WORKFLOW.md` stage 8 description list now includes "for stacked PRs their position in the stack and base", matching TS-WORKFLOW-PR-001.

New findings: none.
