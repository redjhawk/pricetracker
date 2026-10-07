# Functional specification: Claude token settings

Status: ready
Owner: functional specification agent
User decision/reference: GitHub issue #4 "IA review of LeBoncoin items" (2026-10-04) and user decisions D-1–D-3 recorded in the [change index](../../changes/2026-10-04-0710-issue-4-leboncoin-ai-review/index.md).

## Purpose and scope

Let the operator provide the Claude token used for AI reviews of LeBoncoin items ([LeBoncoin AI review](../leboncoin-ai-review/functional.md)) and Amazon search items ([Amazon AI review](../amazon-ai-review/functional.md), issue #56) from the existing **Settings** modal, the "same place where I can set the leboncoin token" (user request). Actor: the operator of the shared, no-login interface.

Included: a second entry in the existing Settings modal ([LeBoncoin session settings](../leboncoin-session-settings/functional.md)); saving with verification, replacing and removing the token; durable server-side storage; error states.

Excluded: login flows inside the app, Anthropic API keys, multiple tokens, billing/usage display, changes to the existing "LeBonCoin session" entry behavior.

Term: the **Claude token** is a Claude Pro/Max subscription token produced by the `claude setup-token` command (D-1).

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-CLT-SET-001 | The operator opens **Settings** (FR-LBC-SET-001). | The modal contains, in addition to "LeBonCoin session", a "Claude token" entry with help text stating to paste the token printed by `claude setup-token` (Claude Pro/Max subscription). This amends the single-entry rule of FR-LBC-SET-001. | The modal shows two labelled entries; the LeBonCoin session entry otherwise behaves per FR-LBC-SET-001–019. |
| FR-CLT-SET-002 | The modal is loading or failed to load (FR-LBC-SET-002/003). | The Claude entry is unusable while loading or after a load failure, so an unloaded field can never remove a saved token. | During loading or after a load error, the Claude entry and Save cannot be used. |
| FR-CLT-SET-003 | Settings are loaded. | Show the saved token in full, in clear, like the LeBoncoin session (D-2). | Given a saved token, the entry shows its exact value. Given none, the entry is empty and the help states AI reviews are unavailable until one is saved. |
| FR-CLT-SET-004 | The operator saves a non-empty Claude entry that differs from the saved value. | Trim surrounding whitespace; reject internal whitespace or control characters with a message on the entry. Otherwise verify the token with Claude before storing (D-3); store it server-side, replacing any previous one, only if Claude accepts it. | After a successful save the modal closes; AI reviews started afterwards use the new token without restart. |
| FR-CLT-SET-011 | Verification on save fails (Claude rejects the token, or Claude cannot be reached). | Refuse the save: nothing is stored (neither entry), the modal stays open with the operator's input, and a message on the Claude entry states the token was refused or could not be verified. | Given an invalid token, Save shows the error and reopening Settings shows the previous state. |
| FR-CLT-SET-005 | The operator empties the Claude entry and saves. | Remove the saved token (D-2); no verification needed. | Afterwards the entry is empty and items show "Configure a Claude token in Settings" (FR-LBC-AIR-008). |
| FR-CLT-SET-006 | Save in progress, save failure, cancel/Escape, concurrent change. | Same rules as FR-LBC-SET-009/010/011/017. Save shows progress during verification. One Save applies both entries; an error in either entry saves neither. | Given an invalid Claude entry and a valid LeBoncoin change, Save changes nothing and marks the Claude entry invalid. |
| FR-CLT-SET-007 | The application restarts with its existing database. | The token persists in application storage; no file, environment variable or shell access is needed. | After restart the modal shows the same token and reviews keep working. |
| FR-CLT-SET-008 | Any use of the application. | The token never appears outside the Settings modal: not in URLs, item responses, review texts, error messages, logs or diagnostics, nor in browser storage after the modal closes. | Item responses, review errors and server logs never contain the token. |
| FR-CLT-SET-009 | The latest AI review attempt failed because Claude rejected the token (invalid, expired, revoked, usage limit). | Show a warning next to the Claude entry with the failure date/time and advice to replace the token. A newly saved token starts without the warning. | Given a rejected token, reopening Settings shows the warning; after saving a new token it is gone. |
| FR-CLT-SET-012 | Settings are shown (issue #56, D-12). | The Claude token help text states that the token is used for Amazon and LeBoncoin AI reviews. | The help text mentions both Amazon and LeBoncoin reviews. |
| FR-CLT-SET-010 | Keyboard, assistive technology, ~400 px viewport. | FR-LBC-SET-019 accessibility rules apply to the new entry. | Visible label; errors announced and associated with the entry; usable at ~400 px without horizontal scrolling. |

## States and corner cases

- Loading, load failed, no token, token saved, rejected-token warning, saving/verifying, verification refused, save failed, validation error.
- Saving an unchanged token does not require re-verification.
- Shared application: one global token used by all reviews.

## Open questions

None.

## Traceability

- Request and decisions: [change index](../../changes/2026-10-04-0710-issue-4-leboncoin-ai-review/index.md) D-1–D-3.
- [LeBoncoin session settings](../leboncoin-session-settings/functional.md) FR-LBC-SET-001–019.
- [LeBoncoin AI review](../leboncoin-ai-review/functional.md).
- Issue #56 amendment (FR-CLT-SET-012): [change index](../../changes/2026-10-07-1117-issue-56-amazon-searches/index.md).
- Technical handoff target: `doc/specifications/claude-token-settings/technical.md`.
