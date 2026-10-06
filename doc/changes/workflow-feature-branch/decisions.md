# Review decisions: workflow-feature-branch

Reviewed specification revisions: `doc/specifications/workflow-feature-branch/` (FR-WORKFLOW-FB-001..003, TS-WORKFLOW-FB-001..005), `doc/specifications/workflow-pull-requests/` (working tree)
Reviewed code revision: uncommitted working-tree snapshot on `docs/workflow-feature-branch` (base `ef0ba30`)
Reviewer: independent reviewer agent ([review.md](review.md))
Adjudicator: independent review adjudicator agent (separate invocation)

## Findings and decisions

### REV-001: ai-dev prompt "from master" may not resolve in CI

- Evidence: `.github/workflows/ai-dev.yml` prompt originally said to create the feature branch "from master" without a start point; a bare `git checkout -b` would branch from the run's HEAD.
- Impact and scenario: a split ai-dev run could create a feature branch already containing run commits, so the final feature→`master` PR would not aggregate only reviewed parts.
- Criticality: non-critical, because it affects a documentation/automation instruction for split runs only, has no data or user-facing product impact, and is caught by PR review of the final PR.
- Disposition: fix.
- Reason: cheap, precise fix using already-allowed commands.
- Specification decision: not applicable.
- Resolution: fixed. Prompt now says "first run `git fetch origin master` and `git checkout -b ai-dev/issue-<n>-feature origin/master` (an exact copy of master, before any run commit)"; TS-WORKFLOW-FB-005 records it. Both commands match existing allowedTools patterns.
- Follow-up: none.

### REV-002: Safety-net PR step ignores the feature branch

- Evidence: `.github/workflows/ai-dev.yml` step "Open pull request" always targeted the default branch with `Closes #<n>`.
- Impact and scenario: in a split run where a part branch was left without a PR, the fallback would open a PR to `master` that bypasses the feature branch and deploys on merge.
- Criticality: non-critical, because the step is a fallback, the PR still requires human review/merge, and nothing deploys without that merge.
- Disposition: fix.
- Reason: keeps FR-WORKFLOW-FB-002/003 (only the final merge reaches `master`) true for the fallback too.
- Specification decision: not applicable (technical; recorded as TS-WORKFLOW-FB-005).
- Resolution: fixed. Lines 100–107: `base` defaults to the default branch; when `$BRANCH` is not `ai-dev/issue-$ISSUE_NUMBER-feature` and that branch exists (`api "$repo_api/branches/$feature"`, `curl -f` fails on 404), `base` becomes the feature branch and the body says `Part of #$ISSUE_NUMBER`. The ahead-check and POST then use `$base`. Bash reviewed: quoting, `[ ... ] && cmd` inside `if`, and fallback `|| echo 0` are correct; when the run branch is the feature branch itself it still targets `master` with `Closes`, which is the intended final PR.
- Follow-up: none.

### REV-003: Leftover "pull requests to `master`" wording

- Evidence: WORKFLOW.md stage 8, AGENTS.md stage 8, and workflow-pull-requests/technical.md stated PRs go to `master` without the feature-branch alternative.
- Impact and scenario: agents could read two conflicting bases and open part PRs against `master`.
- Criticality: non-critical, because it is a wording inconsistency resolved by adjacent text; no runtime effect.
- Disposition: fix.
- Reason: consistency across instruction files is required for autonomous agents.
- Specification decision: not applicable.
- Resolution: fixed. WORKFLOW.md line 123 and AGENTS.md stage 8 now say "to `master` for a single PR, or to a feature branch when split"; workflow-pull-requests/technical.md line 14 adds a supersession note pointing to workflow-feature-branch/technical.md.
- Follow-up: none.

### REV-004: Size measurement for the final PR is ambiguous

- Evidence: WORKFLOW.md Size bullet said not to open any PR that `scripts/pr-size.sh` rejects, contradicting the final-PR exemption.
- Impact and scenario: an agent could refuse to open the final feature→`master` PR for a large feature, blocking delivery.
- Criticality: non-critical, because it blocks progress visibly rather than causing silent harm.
- Disposition: fix.
- Reason: one-clause clarification removes the contradiction.
- Specification decision: not applicable.
- Resolution: fixed. WORKFLOW.md line 125 now adds "except the final feature-branch→`master` PR, which is exempt"; CI size check (jq filter) already excludes the feature head.
- Follow-up: none.

## Decision summary (published, informational)

| Finding | Criticality | Disposition | What was done |
|---|---|---|---|
| REV-001 ai-dev prompt "from master" may not resolve in CI | non-critical | fix | Prompt now fetches master and branches from `origin/master`; recorded in TS-WORKFLOW-FB-005 |
| REV-002 Safety-net PR step ignores the feature branch | non-critical | fix | Fallback step targets `ai-dev/issue-<n>-feature` with `Part of #<n>` when it exists; bash verified |
| REV-003 Leftover "pull requests to `master`" wording | non-critical | fix | WORKFLOW.md and AGENTS.md mention both bases; supersession note added to workflow-pull-requests technical spec |
| REV-004 Size measurement for the final PR is ambiguous | non-critical | fix | Size bullet exempts the final feature-branch→`master` PR |

Published as an issue comment: no originating issue — requested in a local session. Included in the final run report: yes.

## Release readiness

No critical findings; all four non-critical findings are fixed and verified against the working tree. No open functional questions. Documentation-only change plus CI workflow script; application QA not applicable beyond documentation/role consistency checks.
