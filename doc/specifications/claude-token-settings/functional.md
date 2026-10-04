# Functional specification: Claude token settings

Status: needs-clarification
Owner: functional specification agent
User decision/reference: GitHub issue #4 "IA review of LeBoncoin items" (2026-10-04), quoted in the [change index](../../changes/leboncoin-ai-review/index.md). No functional decision beyond the request is recorded yet; see open questions.

## Purpose and scope

Let the operator provide the Claude credential used for AI reviews of LeBoncoin items ([LeBoncoin AI review](../leboncoin-ai-review/functional.md)) from the existing **Settings** modal, the "same place where I can set the leboncoin token" (user request). Actor: the operator of the shared, no-login interface.

Included: a second entry in the existing Settings modal ([LeBoncoin session settings](../leboncoin-session-settings/functional.md)); saving, replacing and removing the credential; durable server-side storage; error states.

Excluded: login flows inside the app, multiple credentials, billing/usage display, changes to the existing "LeBonCoin session" entry behavior.

Term: the **Claude credential** is the "login token" named in the request; its exact kind is unresolved (Q-CLT-1).

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-CLT-SET-001 | The operator opens **Settings** (FR-LBC-SET-001). | The modal contains, in addition to "LeBonCoin session", one entry for the Claude credential (label depends on Q-CLT-1) with help text stating what to paste and how to obtain it. This amends the single-entry rule of FR-LBC-SET-001. | The modal shows two labelled entries; the LeBonCoin session entry otherwise behaves per FR-LBC-SET-001–019. |
| FR-CLT-SET-002 | The modal is loading or failed to load (FR-LBC-SET-002/003). | The Claude entry follows the same rules: unusable while loading or after a load failure, so an unloaded field can never remove a saved credential. | During loading or after a load error, the Claude entry and Save cannot be used. |
| FR-CLT-SET-003 | Settings are loaded. | Show the credential's current state. Clear vs. masked display is unresolved (Q-CLT-3). | Given no saved credential, the entry is empty and the help states AI reviews are unavailable until one is saved. |
| FR-CLT-SET-004 | The operator saves a non-empty Claude entry. | Trim surrounding whitespace and store the credential server-side, replacing any previous one. Input containing internal whitespace or control characters is rejected with a message on the entry and nothing changes. Verification against Claude on save is unresolved (Q-CLT-4). | After a successful save the modal closes; AI reviews started afterwards use the new credential without restart. |
| FR-CLT-SET-005 | The operator wants to remove the credential. | Removal mechanism unresolved (Q-CLT-3). | Pending Q-CLT-3. |
| FR-CLT-SET-006 | Save in progress, save failure, cancel/Escape. | Same rules as FR-LBC-SET-009/010/011. One Save applies both entries; a validation error in either entry saves neither. | Given an invalid Claude entry and a valid LeBoncoin change, Save changes nothing and marks the Claude entry invalid. |
| FR-CLT-SET-007 | The application restarts with its existing database. | The credential persists in application storage; no file, environment variable or shell access is needed. | After restart the modal shows the same state and reviews keep working. |
| FR-CLT-SET-008 | Any use of the application. | The credential never appears outside the Settings modal: not in URLs, item responses, review texts, error messages, logs or diagnostics, nor in browser storage after the modal closes. | Item responses, review errors and server logs never contain the credential. |
| FR-CLT-SET-009 | The latest AI review attempt failed because Claude rejected the credential (invalid, expired, revoked, quota/usage limit). | Show a warning next to the Claude entry with the failure date/time and advice to replace the credential. A newly saved credential starts without the warning. | Given a rejected credential, reopening Settings shows the warning; after saving a new credential it is gone. |
| FR-CLT-SET-010 | Keyboard, assistive technology, ~400 px viewport. | FR-LBC-SET-019 accessibility rules apply to the new entry. | Visible label; errors announced and associated with the entry; usable at ~400 px without horizontal scrolling. |

## States and corner cases

- Loading, load failed, no credential, credential saved, rejected-credential warning, saving, save failed, validation error.
- Concurrent change from another browser: whether FR-LBC-SET-017 "warn and reload" also applies is part of Q-CLT-3.
- Shared application: one global credential used by all reviews.

## Open questions

- Q-CLT-1 (blocking): kind of "login token" — (a) Claude Pro/Max subscription OAuth token from `claude setup-token`; (b) Anthropic API key (`sk-ant-api…`, pay-per-use); (c) accept either.
- Q-CLT-3 (blocking): visibility and removal — (a) like the LeBoncoin session: full value in clear, empty save removes it, warn-and-reload on concurrent change; (b) masked ("configured", last 4 characters), replace by typing a new value, separate Remove action.
- Q-CLT-4: verify on save — (a) Save checks the credential with Claude and refuses an invalid one; (b) store as-is, rejection reported on the next review (FR-CLT-SET-009).

## Traceability

- Request: [change index](../../changes/leboncoin-ai-review/index.md).
- [LeBoncoin session settings](../leboncoin-session-settings/functional.md) FR-LBC-SET-001–019.
- [LeBoncoin AI review](../leboncoin-ai-review/functional.md).
- Technical handoff target: `doc/specifications/claude-token-settings/technical.md` (after questions are resolved).
