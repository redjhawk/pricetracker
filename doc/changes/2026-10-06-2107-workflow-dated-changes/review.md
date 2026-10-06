# Review: workflow dated changes

Reviewer: independent reviewer agent, 2026-10-06. Scope: staged diff (`git diff --cached -M`) plus working-tree changes on `docs/workflow-dated-changes`, against [functional](../../specifications/workflow-dated-changes/functional.md) and [technical](../../specifications/workflow-dated-changes/technical.md).

## Checks performed

- Naming rule: WORKFLOW.md (Documents and traceability), AGENTS.md stage 8 and the ai-dev prompt all state `<YYYY-MM-DD-HHMM>-issue-<n>-<slug>`, UTC, start time. WORKFLOW.md states the omission of `issue-<n>` without a work item; ai-dev always has an issue. `Bash(date -u:*)` is in allowedTools and the prompt gives `date -u +%Y-%m-%d-%H%M`. Consistent and unambiguous.
- Datetimes: for all 20 renamed folders, the earliest `git log --all --diff-filter=A` time (UTC) of the old path equals the folder prefix. `ls doc/changes` is chronological.
- Issue numbers: #1, #3, #4, #10, #15, #42, #45 match the folders' index.md files. Other folders mention only PR numbers in commit subjects (#40, #50, #51, #52) or none, and correctly have no `issue-` segment. No missing issue number found.
- References: no remaining `changes/<old>` or `../<old>` reference anywhere in the repository. `doc/specifications/<subject>` paths (including `leboncoin-session-settings`, `leboncoin-ai-review`, `tracked-items-identity`, ...) are unchanged; `leboncoin-session` and `leboncoin-session-settings` are distinguished correctly (25 and 2 rewrites respectively, all to the right targets). Generic `<change>` placeholders are kept.

## Findings

### REV-001: link rewrites outside `doc/changes/` are not staged

- Requirement: FR-WORKFLOW-DATE-004, acceptance "No Markdown link or path points to an old folder name".
- Location: 19 working-tree-only modifications: `API_SPECIFICATION.md`, `doc/deprecated/leboncoin-session-file.md`, 15 files under `doc/specifications/`, `doc/todo/modal-focus-containment.md` (`git status`: ` M`).
- Evidence: the renames are staged but these rewrites are not; a commit of the index alone would leave about 36 broken links to old folder names.
- Expected: rewrites staged together with the renames. Actual: unstaged.
- Suggested action: stage these files explicitly (they contain only the rewrites; diff checked) before committing.
- Provisional severity: major (blocking if committed as-is).

### REV-002: historical records rewritten despite "remain unchanged" constraint

- Requirement: pre-existing FR-LBC-RM-006 / TS-LBC-RM-005 (`doc/specifications/leboncoin-session-file-removal/`) say `doc/changes/leboncoin-session/` and `leboncoin-403-investigation/` remain unchanged; TS-WORKFLOW-DATE-002 requires rewriting references in them.
- Location: `doc/changes/2026-10-03-1213-leboncoin-session/index.md`, `doc/changes/2026-10-03-0854-leboncoin-403-investigation/*` (link-only edits), and the same specification rows, which were themselves rewritten to the new paths.
- Impact: the old constraint was meant for that change's diff; the new spec supersedes it for paths. No functional impact, but the conflict is not recorded.
- Suggested action: accept, and note in decisions that link-only edits to historical records are authorized by FR-WORKFLOW-DATE-004.
- Provisional severity: minor.

### REV-003: path rewritten inside a historical checksum list

- Location: `doc/changes/2026-10-03-1741-leboncoin-session-settings/review.md` lines 175 onwards (`sha256` evidence block).
- Evidence: the path for `api-step.md` and others was rewritten next to an unchanged hash; the hashes already did not match current files (evidence snapshot), so this does not break anything, but it alters recorded evidence.
- Suggested action: accept as a path-only update, or leave the block untouched; either way record it.
- Provisional severity: trivial.

### REV-004: AGENTS.md does not mention omitting `issue-<n>`

- Location: `AGENTS.md` stage 8 parenthesis.
- Evidence: states the full pattern only and defers to the workflow, which covers omission. Not ambiguous in practice because it says "see the workflow".
- Suggested action: optional; none required.
- Provisional severity: trivial (suggestion).

No unspecified behavior found beyond WORKFLOW.md's sentence "Follow-ups to the same change stay in its folder", which follows from FR-WORKFLOW-DATE-003 (datetime never changes) and is acceptable.
