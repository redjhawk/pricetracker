# Workflow PR reviewer: technical specification

Status: ready
Requirements: [functional](functional.md).

- **TS-WORKFLOW-PRR-001** Add the role `.agents/roles/pr-reviewer.md`, stage 9 in `WORKFLOW.md` (overview row, stage list `... pull-request, pr-review, complete`, section, Completion), and step 9 in `AGENTS.md`.
- **TS-WORKFLOW-PRR-002** Posting: `gh api repos/<owner>/<repo>/pulls/<n>/reviews --method POST --input <json>`, with inline `comments` and labels `[bug]`, `[refactor-pr]`, `[simplify]`, `[readability]`, `[test]`, `[nit]`. The review URLs and label counts are recorded in `doc/changes/<change>/pr-review.md`.
- **TS-WORKFLOW-PRR-003** Blocking: when there are `[bug]` comments, the event is `REQUEST_CHANGES`. GitHub rejects REQUEST_CHANGES from a PR's own author, and the PRs are opened by the same account or token, so the fallback is `COMMENT` with a body starting `CHANGES REQUESTED`. The user is informed through `gh issue comment` when there is an issue, and always in the final run report. This limitation is recorded rather than worked around with a second account.
- **TS-WORKFLOW-PRR-004** `ai-dev.yml`: allow `Bash(gh api repos/redjhawk/pricetracker/pulls/*/reviews --method POST --input *)` (narrowed, see TS-WORKFLOW-PRR-005), and add prompt text to run the agent after the PRs are opened and not to fix its comments without `@claude`. The job already has `pull-requests: write`.
- **TS-WORKFLOW-PRR-005** The allowed tool is narrowed to `gh api repos/redjhawk/pricetracker/pulls/*/reviews --method POST --input *`, so the run cannot merge, close, or dismiss. Inline comments must be on lines inside the diff (`side` `RIGHT`/`LEFT`). A 422 refusal of REQUEST_CHANGES triggers a COMMENT resubmission. After `@claude` fixes are pushed, the PR is reviewed again. `templates/pr-review.md` records the reviews.
- Frontend, backend and API: not affected (workflow only).
- Verification: a YAML parse, `git diff --check`, and a consistency grep. The role is first exercised on this change's own PR, so it reviews this PR.
