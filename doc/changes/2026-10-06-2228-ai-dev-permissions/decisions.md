# Review decisions: ai-dev permissions

Reviewed specification revisions: [functional](../../specifications/ai-dev-permissions/functional.md), [technical](../../specifications/ai-dev-permissions/technical.md) (uncommitted working tree, 2026-10-07)
Reviewed code revision: uncommitted working-tree snapshot of `.github/workflows/ai-dev.yml` and `doc/workflow/WORKFLOW.md` on `docs/ai-dev-permissions`
Reviewer: independent stage 5 reviewer ([review.md](review.md))
Adjudicator: independent stage 6 review adjudicator (separate agent)

## Findings and decisions

### REV-001: `include_comments_by_actor` is outside the specifications

- Evidence: `.github/workflows/ai-dev.yml:84` adds `include_comments_by_actor: redjhawk,github-actions[bot]`; at review time no specification mentioned it. The technical spec now has a row (technical.md line 16) describing it and stating it is the user's change, included at their request on 2026-10-07.
- Impact and scenario: filters which issue/PR comments reach Claude's context on a public repository; without the spec row, implemented behavior was unspecified.
- Criticality: non-critical, because it narrows (hardens) input to Claude and does not break any workflow command; the only defect was a documentation gap.
- Disposition: fix.
- Reason: the user explicitly asked to include this edit in this PR, so it stays; specifying it closes the mismatch.
- Specification decision: not applicable (user's own instruction to include the change; no open functional question).
- Resolution: fixed. technical.md line 16 records the setting, its effect and its origin. Verified by reading the working tree.
- Follow-up: none.

### REV-002: WORKFLOW.md stage 8 still prescribed `gh pr edit --base`

- Evidence: `doc/workflow/WORKFLOW.md:141` now says to retarget "through the REST API (see stage 10) ... and `gh pr edit` fails in this repository"; it no longer prescribes `gh pr edit --base`. Consistent with stage 10 (line 168), the prompt and technical.md line 21.
- Impact and scenario: an agent following stage 8 literally would have failed once on `gh pr edit`.
- Criticality: non-critical, because the failure is recoverable and stage 10 already gave the correct command.
- Disposition: fix.
- Reason: one-sentence fix within the documentation scope of this change; cheaper than a follow-up.
- Specification decision: not applicable.
- Resolution: fixed in WORKFLOW.md stage 8; recorded in technical.md line 21.
- Follow-up: none.

### REV-003: heredoc commit messages via `$(cat <<EOF ...)` may be denied

- Evidence: `cat` is not in `--allowedTools`; the prompt (`ai-dev.yml:90`) now says "Write commit messages with `git commit -m` or `git commit -F <file>` (not `$(cat <<EOF ...)`, because `cat` is not allowed)".
- Impact and scenario: a non-interactive run could lose one attempt on a denied commit command.
- Criticality: non-critical, because the agent can retry with another form; no right is missing.
- Disposition: fix.
- Reason: a short prompt instruction avoids the wasted attempt without widening the allowlist.
- Specification decision: not applicable.
- Resolution: fixed in the `--append-system-prompt` text.
- Follow-up: none.

### REV-004: new `gh api` globs broader than their documented use

- Evidence: the PATCH entry and the `pulls/*/comments` / `pulls/*/reviews` read entries use `*`, which matches spaces and extra flags. technical.md lines 11 and 21 now state the globs are not strict and reference #74. Issue #74 received a comment on 2026-10-06T22:31:34Z covering the read entries.
- Impact and scenario: a run could, in principle, change any PR field or insert extra `gh api` flags; limited by `gh api` taking one endpoint, by the job token's scope to this repository, and by the prompt restricting instructions to the owner (now also enforced by `include_comments_by_actor`).
- Criticality: non-critical, because exploitation requires the run itself to deviate from its prompt, and the token's existing `pull-requests: write` already allows these actions by other allowed commands.
- Disposition: defer.
- Reason: hardening the globs needs exact-argument patterns that Claude Code's matcher does not express cleanly; the documented uses work now, and the residual risk is accepted and tracked.
- Specification decision: not applicable.
- Resolution: deferred; spec corrected to stop claiming "no extra arguments".
- Follow-up: repository owner, issue #74 (comment added for the read entries).

## Decision summary (published, informational)

| Finding | Criticality | Disposition | What was done |
|---|---|---|---|
| REV-001 `include_comments_by_actor` unspecified | non-critical | fix | Kept at the user's explicit request; specified in technical.md with its effect and origin |
| REV-002 WORKFLOW stage 8 prescribed `gh pr edit --base` | non-critical | fix | Stage 8 now points to the REST PATCH retarget (stage 10) |
| REV-003 heredoc commit messages need `cat` | non-critical | fix | Prompt tells the run to use `git commit -m` or `-F <file>` |
| REV-004 over-broad `gh api` globs | non-critical | defer | Spec corrected; tracked in issue #74 (comment added); residual risk bounded by token scope and owner-only comments |

Published as an issue comment: no originating issue — requested in a local session. Included in the final run report: yes.

## Release readiness

No critical findings and no unresolved functional questions. REV-001 to REV-003 are fixed in the working tree; REV-004 is deferred to issue #74 with recorded reasons. Application QA is not applicable (workflow and documentation change only); verification is a YAML parse and `git diff --check`. Ready for stage 8 commit and PR.
