# Commit step result

Status: prepared / committed / blocked / waived by explicit user request
Coordinator:

## Passed gates

- Specification and final diff consistency:
- Required user/API approval evidence or unchanged approved contract:
- Review report and decisions; no unresolved critical findings:
- Required QA report and passed results:

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

Record preparation and passed gates before committing. Report success only after Git confirms it. If the committed preparation record cannot include its own result, report the actual result and Git reference in the coordinator's final handoff. A failed commit blocks completion. Do not amend existing commits or push without an explicit user request.
