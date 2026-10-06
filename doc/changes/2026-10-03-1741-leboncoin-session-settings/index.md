# LeBoncoin session settings

Stage: complete (commit stage recorded in [commit-step.md](commit-step.md)).\
Change slug: `leboncoin-session-settings`. Started 2026-10-03.

## Scope

Replace the file-based LeBoncoin session (change [leboncoin-session](../2026-10-03-1213-leboncoin-session/index.md)) with a session value stored in the application's SQLite database and set from the interface: a profile-style header icon opens a menu with **Settings**, which opens a modal with one form entry, “LeBonCoin session”. Keep automatic renewal of the cookie, show the last failure of the session in the modal, clear the session by saving an empty value, keep the desktop capture helper but print the captured `datadome=...` string, fix the helper's reported failure to visibly open a browser with progress messages, remove `LEBONCOIN_SESSION_FILE` and the file/sidecar code and current documentation references, and record the removed functionality in a new `doc/deprecated/` folder. This is a cross-tier change: the API contract must be proposed and confirmed by the user before implementing either tier.

## User request (verbatim, 2026-10-03)

> instead of using a file for storing the session, why not storing it in the db, and let the user to set it from the interface ? on the top right part of the screen, you should show something similar to a profile icon that will open a menu. on this menu, you should show a settings link. when clicking on it, a modal will appear. On this modal, you will have a form. on this form, it will be one entry. "LeBonCoin session". Using the entry of this form, the user will be able to update the session. This way, no file is required and application is autonomous. Delete all references to old system. use workflow on doc/workflow

## User decisions (answers to coordinator questions, 2026-10-03)

These are explicit user requirements, recorded as provided by the coordinator:

1. **Input:** accept a full cookie string — either the raw datadome value or a `datadome=...` string copied from a request header/cookie string; the app extracts the datadome value.
2. **Display:** the modal shows the FULL saved value in the field so it can be edited (the user accepted that anyone on the network can read it; the app has no accounts).
3. **Keep auto-updated cookie:** when LeBoncoin returns a renewed datadome cookie on a successful verified page, save it in the DB automatically (as the file version did).
4. **Show rejection hint:** in the modal, show when the last LeBoncoin attempt using the session failed (e.g. rejected/403), so the user knows to renew it.
5. **No separate “Clear” button:** saving the form with an EMPTY field clears the session and collection returns to sessionless mode.
6. **Removal:** remove the `LEBONCOIN_SESSION_FILE` environment variable and the file/sidecar session code. KEEP the desktop capture script that opens a browser, but instead of writing a file it PRINTS the captured `datadome=...` string in the terminal so the user pastes it into the Settings modal. Create a new folder under `doc/` to keep a trace of deprecated functionalities, with a proper file describing the removed file-based session functionality (what it was, why removed, what replaces it, links to history/commits by message).
7. **Bug to cover:** the user reports that running the capture script “doesn't open a browser” and prints no message. Coordinator observations: on this host (GNOME/Wayland session, `DISPLAY=:1`, `WAYLAND_DISPLAY=wayland-0`, system node v20.19.2, Chrome 149 at `/usr/bin/google-chrome`) a coordinator run showed Chrome processes starting with the temp profile, then the helper printed “Browser closed before a verified listing was captured.” and exited 1 early (before its timeout) — it is unknown whether a window was visible. With a missing output directory the helper failed before launching with a misleading generic message. The user-observable requirement is specified; root cause is technical work.

## User answers to functional questions (2026-10-03)

- **Q-SET-1:** “Warn and reload”. If the saved session changed after the modal was opened (auto-renewal, or a save from another browser), Save must detect it, change nothing, and ask the operator to reload the modal before saving. Applied in FR-LBC-SET-017.
- **History:** keep history and mark superseded rows. Superseded spec rows stay marked superseded, `doc/changes/2026-10-03-1213-leboncoin-session/` stays as history, and `doc/deprecated/` explains the removal. Applied in FR-LBC-RM-004 to 006 and the revision 2 collection and capture specifications.

API contract approval for this change was given later the same day; see “User decisions at the API gate (2026-10-03)” below and [api-step.md](api-step.md).

## Subject specifications

| Subject | Functional file | Requirement IDs | Status |
| --- | --- | --- | --- |
| Application header menu | [functional](../../specifications/app-header-menu/functional.md) | FR-APP-MENU-001 – 006 | ready |
| LeBoncoin session settings | [functional](../../specifications/leboncoin-session-settings/functional.md) | FR-LBC-SET-001 – 019 | ready |
| LeBoncoin session collection (revision 2) | [functional](../../specifications/leboncoin-session-collection/functional.md) | new FR-LBC-COL-011 – 020; superseded 001–006, 009, 010; 007, 008 unchanged | ready |
| LeBoncoin session capture (revision 2) | [functional](../../specifications/leboncoin-session-capture/functional.md) | new FR-LBC-CAP-010 – 018; superseded 001, 005–009; 002–004 unchanged | ready |
| File-based session removal | [functional](../../specifications/leboncoin-session-file-removal/functional.md) | FR-LBC-RM-001 – 007 | ready |

## Technical specifications

| Subject | Technical file | Technical IDs | Status |
| --- | --- | --- | --- |
| Application header menu | [technical](../../specifications/app-header-menu/technical.md) | TS-APP-MENU-001 – 004 | ready |
| LeBoncoin session settings | [technical](../../specifications/leboncoin-session-settings/technical.md) | TS-LBC-SET-001 – 007 | ready (API pending) |
| LeBoncoin session collection (revision 2) | [technical](../../specifications/leboncoin-session-collection/technical.md) | new TS-LBC-COL-006 – 011; superseded 001, 003, 004; 002 amended; 005 retained | ready |
| LeBoncoin session capture (revision 2) | [technical](../../specifications/leboncoin-session-capture/technical.md) | new TS-LBC-CAP-006 – 012; superseded 004, 005, shared import format; 001–003 retained | ready |
| File-based session removal | [technical](../../specifications/leboncoin-session-file-removal/technical.md) | TS-LBC-RM-001 – 006 | ready |

## Stage records

1. [Functional handoff](functional-step.md).
2. [Technical handoff](technical-step.md): ready.
3. [API contract proposal](api-step.md): proposed in [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) (section “LeBoncoin session settings (approved 2026-10-03)”, proposal revision 1); **approved by the user on 2026-10-03**.
4. Implementation, 5. review, 6. review decisions, 7. QA, 8. commit: pending.

## Unresolved questions

Functional: none. Q-SET-1 and the history question were answered by the user on 2026-10-03 (see above).

Pending user decisions from the technical stage (see [technical-step.md](technical-step.md)):

1. **Blocking:** confirm (or amend) the proposed settings API contract.
2. Optional refactor R-1 (shared `src/api/client.ts`); recommendation: decline or defer.
3. Non-blocking clarification: natural expiry does not increment `revision` (no `409` from expiry alone); revocation and renewals do.
4. For information: capture helper minimum Node.js 20.19 (engines for the build stay ≥ 22).

## Handoffs

- Functional specifier (separate agent, 2026-10-03): five functional files written; see [functional-step.md](functional-step.md).
- Technical specifier (separate agent, 2026-10-03): five technical files, pending API proposal and capture defect analysis; see [technical-step.md](technical-step.md) and [api-step.md](api-step.md).

## User decisions at the API gate (2026-10-03)

- API contract: **"Approve"** (proposal revision 1 unchanged).
- Refactoring R-1 (move `request`/`ApiError` into shared `src/api/client.ts`): **"Do it first"**, so it is performed before the feature as a separate step.
- Expiry clarification: **"Yes, as proposed"**: expiry alone does not bump `revision`.

## Completion (2026-10-03)

Review findings REV-SET-001–008 are resolved (REV-SET-006 at commit). QA failures QA-SET-F01/F02 (critical) were fixed, rechecked and retested. QA-SET-F03 (focus escape in the pre-existing Add item/Delete modals at machine-speed key presses) is pre-existing, noncritical and deferred: a separate change if the user wants it. Live QA on the desktop host: capture helper output pasted into Settings, and the listing was collected with a price. Raspberry Pi acceptance remains untested. See [review](review.md), [decisions](decisions.md), [QA](qa.md), [commits](commit-step.md).
