# Review: workflow PR reviewer

Reviewer: independent stage 5 agent. Scope: uncommitted diff of `AGENTS.md`, `doc/workflow/WORKFLOW.md`, `.github/workflows/ai-dev.yml`, new `.agents/roles/pr-reviewer.md`, specifications and change folder, against [functional](../../specifications/workflow-pr-reviewer/functional.md) and [technical](../../specifications/workflow-pr-reviewer/technical.md).

## Verified as consistent

- FR-001..007 and TS-001..004 are all covered: role file, overview row 9, stage list (`pull-request, pr-review, complete`), section 9, Completion, After the run, AGENTS.md step 9, ai-dev allowlist and prompt.
- Repository path `redjhawk/pricetracker` matches `git remote -v`.
- The allowlist entry `Bash(gh api repos/redjhawk/pricetracker/pulls/*)` is a prefix match, so it matches the documented `gh api repos/redjhawk/pricetracker/pulls/<n>/reviews --method POST --input <file>` including trailing arguments. `Write` (for the JSON input file) and `gh issue comment` are already allowed.
- API usage is correct: `POST /repos/{o}/{r}/pulls/{n}/reviews` accepts `event` (`APPROVE`/`REQUEST_CHANGES`/`COMMENT`), `body`, and `comments[]` with `path`, `line`, `side`, `body`. GitHub returns 422 for `REQUEST_CHANGES` on one's own PR, so the documented COMMENT fallback is correct.

## Findings

### REV-001 — Allowlist grants far more than posting reviews (provisional: major)

- Requirement: TS-WORKFLOW-PRR-004, FR-WORKFLOW-PRR-006 (comment only).
- Location: `.github/workflows/ai-dev.yml` line 84, `Bash(gh api repos/redjhawk/pricetracker/pulls/*)`.
- Evidence: the prefix pattern also matches `gh api repos/redjhawk/pricetracker/pulls/<n>/merge --method PUT`, `pulls/<n> --method PATCH -f state=closed`, `pulls/<n>/reviews/<id>/dismissals`, comment deletion, etc. Merging into `master` triggers deployment. Previously the run had no way to merge PRs.
- Expected: only review creation is permitted. Actual: any pulls API operation, including merge.
- Suggested action: narrow the pattern, e.g. `Bash(gh api repos/redjhawk/pricetracker/pulls/*/reviews --method POST --input *)` (and use exactly that argument order in the role/prompt), or accept and record the risk explicitly.

### REV-002 — Role omits failure handling for inline comment positions and the own-PR refusal (provisional: minor)

- Requirement: FR-WORKFLOW-PRR-005, FR-WORKFLOW-PRR-007; TS-WORKFLOW-PRR-003.
- Location: `.agents/roles/pr-reviewer.md`, "How to report".
- Evidence: GitHub rejects the whole review (422) if any comment's `line`/`side` is not part of the PR diff, and rejects `REQUEST_CHANGES` on own PRs with 422. The role says "GitHub refuses that ... so in that case" but does not say how to detect it (try then fall back on 422, or compare PR author with the token's user) nor that `side` must be `RIGHT` for added/context lines (`LEFT` for removed), nor `start_line` for ranges. In ai-dev runs the PR author is always the same token, so REQUEST_CHANGES will always fail first.
- Suggested action: state `side: "RIGHT"` for new lines, that lines must lie within diff hunks (otherwise put the finding in the body), and "submit REQUEST_CHANGES; on a 422 own-PR error resubmit as COMMENT with the `CHANGES REQUESTED` prefix".

### REV-003 — WORKFLOW.md introduction does not list the new role (provisional: minor)

- Requirement: TS-WORKFLOW-PRR-001 (consistency).
- Location: `doc/workflow/WORKFLOW.md` line 3: "uses separate expert agents for functional specification, ..., and QA."
- Expected: includes pull request review, and the independence sentence covers the PR reviewer. Actual: unchanged.
- Suggested action: add "and pull request review" to that sentence.

### REV-004 — Behavior on follow-up `@claude` fix pushes is unspecified (provisional: question)

- Requirement: unspecified.
- Location: section 9 and ai-dev prompt.
- Evidence: after the user requests fixes with `@claude` on a PR, the run pushes to the same branch. Nothing says whether stage 9 runs again (a new review on the updated diff) or not; a `REQUEST_CHANGES` review would otherwise stay as the latest state, and the `CHANGES REQUESTED` COMMENT fallback cannot be dismissed. Not a functional gap requiring the user unless re-review is considered product behavior; adjudicator should decide whether to record a default (e.g. re-run stage 9 on the fix diff).
- Suggested action: record the intended behavior in section 9, or record it as deliberately out of scope.

### REV-005 — No template for `pr-review.md` (provisional: suggestion)

- Location: `doc/workflow/templates/`; section 9 and role require `doc/changes/<change>/pr-review.md`.
- Evidence: other stage artifacts have templates or a referenced format; this one is defined only by "URL and counts per label". Acceptable as-is; noted for consistency only.

No other defects found. Frontend, backend and API are unaffected, as specified.
