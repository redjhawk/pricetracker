# Functional specification handoff

Status: ready (all subjects; Q-SET-1 resolved 2026-10-03)\
Date: 2026-10-03\
Role: independent functional specification agent

## Subjects and requirement IDs

1. [Application header menu](../../specifications/app-header-menu/functional.md) — FR-APP-MENU-001 to 006: profile-style icon at the top right of the header on every page, menu with exactly one **Settings** entry opening the settings modal, keyboard/Carbon accessibility, narrow screens, no account semantics.
2. [LeBoncoin session settings](../../specifications/leboncoin-session-settings/functional.md) — FR-LBC-SET-001 to 019: modal “Settings” with one entry “LeBonCoin session”; load/load-failure (saving disabled so a failed load cannot clear); full unmasked value displayed; raw value or cookie string accepted and `datadome` extracted; validation errors; save replaces; empty save clears; pending/failed save; cancel discards; database persistence across restarts; effect without restart and no check triggered; rejection/failure hint; expired/revoked status; renewed value shown on reopen; exposure limited to the modal; accessibility. FR-LBC-SET-017: a save after the stored session changed since the modal loaded is refused with a reload request.
3. [LeBoncoin session collection, revision 2](../../specifications/leboncoin-session-collection/functional.md) — new FR-LBC-COL-011 to 020 (sessionless when none saved, LeBoncoin-only use, save/clear effect and in-flight ordering, automatic renewal stored in DB, expiry/revocation, renewal write failure, recording of session-attempt outcome for the hint, secret handling, no acceptance guarantee, unreadable stored session). Superseded and kept: 001–006, 009, 010. Unchanged: 007, 008.
4. [LeBoncoin session capture, revision 2](../../specifications/leboncoin-session-capture/functional.md) — new FR-LBC-CAP-010 to 018 (URL-only input, one printed `datadome=<value>` line with paste instruction, no file, operator guide for the Settings workflow, reliably visible browser window, immediate progress messages, no false “browser closed” early exit, distinct failure messages, prerequisite checks). Superseded and kept: 001, 005–009. Unchanged: 002–004.
5. [File-based session removal](../../specifications/leboncoin-session-file-removal/functional.md) — FR-LBC-RM-001 to 007: `LEBONCOIN_SESSION_FILE` ignored/removed, old files neither read nor imported nor deleted, no file output in the helper, no live references in current docs/source, new `doc/deprecated/` folder with index and `leboncoin-session-file.md` record (contents enumerated, commits cited by message), historical records preserved, no item/API/Amazon behavior change.

## Decisions applied

User decisions 1–7 of 2026-10-03 (quoted in [index.md](index.md)) are applied as explicit requirements. Defaults taken from existing approved specifications rather than invented: an expired or LeBoncoin-revoked cookie is not sent and attempts record `request_error` (revision 1 FR-LBC-COL-004 semantics); an unreadable session yields `request_error` (revision 1 FR-LBC-COL-003); a successful save closes the modal (existing Add item modal behavior); dates follow the existing presentation. Deliberate exclusions, because not requested: menu entries other than Settings, a separate Clear button, masking, triggering a check on save, importing the old session file, a startup warning for the removed variable. The form label uses the user's spelling “LeBonCoin session”.

## User answers (2026-10-03)

- **Q-SET-1:** “Warn and reload”. If the saved session changed after the modal was opened (auto-renewal, or a save from another browser), Save must detect it, change nothing, and ask the operator to reload the modal before saving. Written into FR-LBC-SET-017; the settings subject is now ready.
- **History:** keep history and mark superseded rows. Superseded spec rows stay marked superseded, `doc/changes/2026-10-03-1213-leboncoin-session/` stays as history, and `doc/deprecated/` explains the removal. Matches FR-LBC-RM-004 to 006 as written.

No open questions remain.

## Notes for the technical stage

- The cross-tier change needs a new settings API contract (read with value, last session-attempt outcome and expired/revoked status; save/clear) confirmed by the user before implementation.
- `package.json` declares `node >=22` and the old guide required Node.js 22, while the reporting host's system Node.js is 20.19.2; FR-LBC-CAP-018 requires a clear prerequisite message. Root-cause analysis of the early “Browser closed…” exit belongs to technical work (FR-LBC-CAP-014/016).
- `doc/FUNCTIONAL_SPECIFICATIONS.md` (sections 7 and 9) does not yet mention the settings modal; the coordinator may decide whether to add a cross-reference. It was not modified by this role.

## Verification of this stage

Read `AGENTS.md`, the four project skills, the workflow (section 1), the functional-specifier role and template, the general functional specifications, the API contract headings, both revision 1 session specifications, the leboncoin-session change index/decisions/QA and operator guide, and the current React shell (`src/App.tsx`, `src/components/AddItemModal.tsx`). Each subject has scope, actors, exclusions, stable IDs, testable acceptance criteria, corner cases, open questions and traceability. Superseded IDs are preserved and marked. No application code, API contract or technical design was written; no application checks were run.
