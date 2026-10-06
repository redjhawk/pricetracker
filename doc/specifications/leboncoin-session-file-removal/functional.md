# Functional specification: file-based LeBoncoin session removal

Status: ready\
Owner: functional specification agent\
User decision/reference: user request of 2026-10-03 (“Delete all references to old system”) and user decision 6 (remove `LEBONCOIN_SESSION_FILE` and the file/sidecar session code; keep the capture helper but print instead of writing a file; create a new folder under `doc/` that keeps a trace of deprecated functionality, with a file describing the removed file-based session). Quoted in the [change index](../../changes/2026-10-03-1741-leboncoin-session-settings/index.md).

## Purpose and scope

Remove the file-based LeBoncoin session mechanism delivered by change [leboncoin-session](../../changes/2026-10-03-1213-leboncoin-session/index.md), now replaced by the [settings modal](../leboncoin-session-settings/functional.md) and database storage, and keep a durable, discoverable record of what was removed. Actors: the operator (deploying, upgrading and reading documentation) and future maintainers.

Included: the `LEBONCOIN_SESSION_FILE` setting, the session import file and its collector-owned `.state.json` sidecar, file export by the capture helper, the Raspberry Pi transfer/installation and systemd drop-in instructions, current documentation references, and a new deprecation folder and record.

Excluded: rewriting historical change records and investigation reports; deleting files on existing servers automatically; importing the old file's session into the database; any change to Amazon collection or item behavior.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-LBC-RM-001 | The application starts, with or without `LEBONCOIN_SESSION_FILE` in its environment. | The application no longer recognizes `LEBONCOIN_SESSION_FILE`; its presence has no effect on startup or collection. | With the variable unset, set to a valid file, or set to a missing file, the application starts normally and LeBoncoin collection depends only on the session saved in Settings. The configuration documentation no longer lists the variable. |
| FR-LBC-RM-002 | An existing deployment has a revision 1 session file and sidecar on disk. | The application neither reads, writes, imports nor deletes these files. | After upgrade, the old files are untouched and unused; the settings modal shows no session until the operator saves one. The deprecation record tells the operator how to remove the old files and drop-in. |
| FR-LBC-RM-003 | The capture helper runs. | The helper no longer writes or renews a session file (see FR-LBC-CAP-010 to FR-LBC-CAP-012). | No file output option exists; the session is only printed. |
| FR-LBC-RM-004 | The operator or a maintainer reads current documentation (README, operator guide, deployment and configuration documentation). | Current documentation contains no instructions or references for the file-based session, its environment variable, sidecar, transfer to the Raspberry Pi, or drop-in; it describes the Settings workflow instead. Where current documentation previously pointed to the file workflow, it points to the Settings workflow and, for history, to the deprecation record. | A search of current (non-historical) documentation and application/configuration source for `LEBONCOIN_SESSION_FILE`, `session.json`, `.state.json` sidecar or `leboncoin-session.conf` finds no live instruction; only the deprecation record, superseded specification rows, and historical change records mention them. |
| FR-LBC-RM-005 | The change is delivered. | Create a new folder `doc/deprecated/` containing an index and the record `doc/deprecated/leboncoin-session-file.md`. The record states: what the file-based session was (desktop capture to a private file, secure transfer to the Raspberry Pi, `LEBONCOIN_SESSION_FILE` systemd drop-in, collector-owned `.state.json` sidecar with automatic cookie renewal, renewal without restart); when and why it was removed (user request of 2026-10-03: store the session in the database and set it from the interface so no file is required and the application is autonomous); what replaces it (settings modal, database storage, printed capture string, with links to their specifications and the operator guide); clean-up steps for an existing deployment (remove the drop-in, reload systemd and restart the service, then delete the old session file, sidecar and any staging directory, and paste a fresh session in Settings); and history links: the commits by message — “feat(leboncoin): reuse a manually verified session in the collector”, “feat(leboncoin): add desktop session capture helper and operator guide”, “docs(leboncoin): record session specifications, review, QA and decisions”, “docs(leboncoin): replace trailing-space line breaks in session records” — with how to find them via `git log`, plus links to the superseded specifications and the [leboncoin-session change records](../../changes/2026-10-03-1213-leboncoin-session/index.md), and the commit(s) of this removal by message. | The folder and both files exist; the record contains each listed element; all its links resolve; the documentation index (`doc/README.md`) links to `doc/deprecated/`. |
| FR-LBC-RM-006 | Historical material exists. | Preserve history: change records under `doc/changes/2026-10-03-1213-leboncoin-session/` and `doc/changes/2026-10-03-0854-leboncoin-403-investigation/` remain unchanged; superseded requirement rows stay in their specifications, marked superseded with links to their replacements and to the deprecation record. | Those change records are byte-identical before and after the change; each superseded requirement ID still exists and is marked. |
| FR-LBC-RM-007 | The removal is delivered. | The removal does not change the item API responses, item outcomes, price history, schedule, or Amazon collection. | Existing item list/detail/add/delete/refresh behavior and Amazon collection behave as before. |

## States and corner cases

- An upgraded server whose systemd drop-in still sets the removed variable runs normally (FR-LBC-RM-001); LeBoncoin collection is sessionless until a session is saved in Settings.
- A deployment that never used the file workflow is unaffected.
- Session renewals previously stored in the sidecar are lost on upgrade by design (no import); the operator captures and saves a fresh session.
- The previous operator guide `doc/leboncoin-session.md` is either rewritten for the Settings workflow or replaced, with README links updated accordingly (FR-LBC-RM-004); its old content survives in Git history and is summarized in the deprecation record.

## Open questions

None affecting this handoff. The absence of an automatic import and of a startup warning for the removed variable follows from the instruction to delete all references to the old system. If the user prefers a one-time import or a warning, that would be a new requirement.

## Traceability

- Superseded: [session collection](../leboncoin-session-collection/functional.md) FR-LBC-COL-001 to 006, 009, 010; [session capture](../leboncoin-session-capture/functional.md) FR-LBC-CAP-001, 005 to 009.
- Replacement: [session settings](../leboncoin-session-settings/functional.md), [header menu](../app-header-menu/functional.md), collection FR-LBC-COL-011 to 020, capture FR-LBC-CAP-010 to 018.
- History: [leboncoin-session change](../../changes/2026-10-03-1213-leboncoin-session/index.md), [403 investigation](../../changes/2026-10-03-0854-leboncoin-403-investigation/report.md).
- Technical handoff target: `doc/specifications/leboncoin-session-file-removal/technical.md`.
