# Workflow dated changes: technical specification

Status: ready
Requirements: [functional](functional.md).

- **TS-WORKFLOW-DATE-001** Format: `<YYYY-MM-DD-HHMM>-issue-<n>-<slug>`. The `-issue-<n>` segment is omitted without a work item. Times are UTC and there are no colons, so the name is valid on every filesystem and a plain lexical sort is chronological. `WORKFLOW.md` (Documents and traceability), `AGENTS.md` and the ai-dev prompt state this rule, and ai-dev may run `date -u`.
- **TS-WORKFLOW-DATE-002** Existing folders: the datetime is the earliest commit time that added a file to the folder, across all refs (`git log --all --diff-filter=A`, UTC). Squash merges reset commit times on `master`, so the original branch commits are used where they still exist. The issue number comes from the folder's index or commit subjects, and only issues are used, not PR numbers: amazon-installment-price #45, item-refresh #1, leboncoin-ai-review #4, leboncoin-old-price #42, leboncoin-purchase-goal #10, multi-user-login-phase1 #15, workflow-pull-requests #3. Folders are renamed with `git mv`, and references (`changes/<old>` anywhere, `../<old>` inside `doc/changes/`) are rewritten with name-boundary matching. Generic placeholders such as `<change>` are kept.
- Frontend, backend and API: not affected.
- Verification: a sorted `ls` is chronological; a grep finds no old names; a link checker reports no broken relative links under doc/.
