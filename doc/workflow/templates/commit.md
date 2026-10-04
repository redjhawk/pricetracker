# Commit step result

Status: prepared / committed / blocked / waived by explicit user request
Coordinator:

## Passed gates

- Specification and final diff consistency:
- API contract changes and recorded rationale, or unchanged contract:
- Review report and decisions; no unresolved critical findings:

## Staged scope and checks

- Explicit paths or hunks staged:
- Unrelated working-tree changes preserved:
- Staged diff inspection:
- `git diff --cached --check` actual result:
- Other verification commands and actual results:

## Commit outcome

- Descriptive commit message(s):
- Actual outcome or error:
- Commit reference: resolve the message using `git log`; do not put the commit's own hash in its committed files.
- If committing was waived, cite the explicit user request:

## Pull requests

- Changed lines per PR (added + deleted, generated files excluded; target 450±50, ceiling 500):
- PR order, branch, base (stacked), and kind (feature / refactoring / dependencies / formatting / unrelated docs):
- Agent-written descriptions:
- Ready (not draft) PR links, or planned split and the tooling limitation preventing it:
- QA report and passed results (after PR creation; fixes pushed to the PR branches; required for completion):

Record preparation and passed gates before committing. Report success only after Git confirms it. If the committed preparation record cannot include its own result, report the actual result and Git reference in the coordinator's final handoff. A failed commit blocks completion. Do not amend existing commits. Push and open pull requests per workflow stage 8; never claim a PR exists unless the tooling confirms it.
