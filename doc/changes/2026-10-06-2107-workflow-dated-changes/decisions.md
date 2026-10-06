# Review decisions: workflow-dated-changes

Reviewed specification revisions: staged [functional](../../specifications/workflow-dated-changes/functional.md) and [technical](../../specifications/workflow-dated-changes/technical.md) on `docs/workflow-dated-changes`.
Reviewed code revision: staged index (`git diff --cached -M`) on `docs/workflow-dated-changes`, 2026-10-06; `git diff` empty at adjudication.
Reviewer: independent reviewer agent ([review](review.md)).
Adjudicator: independent review adjudicator agent (separate invocation; did not implement, specify or review this change).

## Findings and decisions

### REV-001: link rewrites outside `doc/changes/` are not staged

- Evidence: at review time 19 files (`API_SPECIFICATION.md`, `doc/deprecated/leboncoin-session-file.md`, `doc/specifications/**`, `doc/todo/modal-focus-containment.md`) carried the rewrites only in the working tree. At adjudication `git diff --name-only` is empty and these paths appear in `git diff --cached`.
- Impact and scenario: committing the index alone would have left about 36 links to old folder names, violating FR-WORKFLOW-DATE-004.
- Criticality: non-critical, because it is a staging omission in documentation links, caught before any commit; no data or runtime behavior is affected. It would have blocked acceptance of FR-WORKFLOW-DATE-004 if committed.
- Disposition: fix.
- Reason: the files contain only the required rewrites; staging them is the specified state.
- Specification decision: not applicable.
- Resolution: fixed; verified by adjudicator (`git diff` empty, rewrites staged). Coordinator must commit from this index.
- Follow-up: none.

### REV-002: historical records rewritten despite "remain unchanged" constraint

- Evidence: FR-LBC-RM-006 / TS-LBC-RM-005 say `doc/changes/leboncoin-session/` and `leboncoin-403-investigation/` remain unchanged; this change makes link-only edits in `doc/changes/2026-10-03-1213-leboncoin-session/index.md` and `doc/changes/2026-10-03-0854-leboncoin-403-investigation/*`, and in those specification rows.
- Impact and scenario: none functionally. The older constraint scoped the leboncoin-session-file-removal change (do not alter historical content there); it did not forbid later path maintenance. FR-WORKFLOW-DATE-004 explicitly requires links to keep working, which is impossible after the renames without editing relative links in these records.
- Criticality: non-critical, because only link targets change, content and conclusions of the records are preserved.
- Disposition: reject (no code change); the rewrites are kept.
- Reason: reverting would break links and violate FR-WORKFLOW-DATE-004 and its acceptance criterion "No Markdown link or path points to an old folder name". The newer requirement supersedes the older constraint for paths only. This record documents the conflict and its resolution, as the reviewer suggested. Remaining risk: none beyond the historical files now showing a 2026-10-06 diff, which `git log` attributes to this change.
- Specification decision: not applicable (technical precedence of FR-WORKFLOW-DATE-004 over an old change-scoped constraint; no functional choice).
- Resolution: rejected as a defect; decision recorded here.
- Follow-up: none.

### REV-003: path rewritten inside a historical checksum list

- Evidence: `doc/changes/2026-10-03-1741-leboncoin-session-settings/review.md` from line 175, SHA-256 evidence block: paths rewritten, hashes unchanged.
- Impact and scenario: the hashes already did not match current files (snapshot of 2026-10-03), so no verification depends on them. Rewriting keeps the paths resolvable; leaving old paths would leave paths pointing to old folder names, contrary to the acceptance criterion.
- Criticality: non-critical, because the evidence is a historical snapshot whose identity is the file name plus hash; the path prefix is a location, and the old location no longer exists.
- Disposition: reject (keep the path-only update).
- Reason: FR-WORKFLOW-DATE-004 acceptance requires no path to an old folder name anywhere in the repository, without an exception for evidence blocks. The old path remains recoverable from git history (rename detected by `git diff -M`). Remaining risk: a reader re-hashing the 2026-10-03 snapshot must check out the pre-rename commit, which was already necessary.
- Specification decision: not applicable.
- Resolution: rejected as a defect; decision recorded here.
- Follow-up: none.

### REV-004: AGENTS.md does not mention omitting `issue-<n>`

- Evidence: `AGENTS.md` stage 8 gives the full pattern and refers to the workflow, where WORKFLOW.md states the omission without a work item.
- Impact and scenario: an agent reading only AGENTS.md might add an empty issue segment; unlikely, since AGENTS.md directs agents to the workflow.
- Criticality: non-critical (suggestion), because the authoritative rule is in WORKFLOW.md and is unambiguous.
- Disposition: reject.
- Reason: duplicating the rule in AGENTS.md adds a second source to keep in sync; the reference suffices. Remaining risk: low.
- Specification decision: not applicable.
- Resolution: rejected.
- Follow-up: none.

## Decision summary (published, informational)

| Finding | Criticality | Disposition | What was done |
|---|---|---|---|
| REV-001 link rewrites not staged | non-critical | fix | Rewrites staged with the renames; verified `git diff` empty |
| REV-002 historical records rewritten | non-critical | reject | Link-only edits kept: FR-WORKFLOW-DATE-004 requires working links and supersedes the old change-scoped constraint for paths |
| REV-003 path in historical checksum list | non-critical | reject | Path-only update kept; hashes were already a snapshot; acceptance forbids old paths |
| REV-004 AGENTS.md omits `issue-<n>` omission | non-critical | reject | Suggestion; WORKFLOW.md is the single authoritative rule and AGENTS.md references it |

Published as an issue comment: no originating issue — requested in a local session. Included in the final run report: yes.

## Release readiness

No critical or blocking findings remain. REV-001 is fixed in the staged index; the commit must be made from that index. QA is documentation consistency only (no application behavior changed).
