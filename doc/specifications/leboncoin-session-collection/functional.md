# Functional specification: LeBoncoin session collection

Status: ready (revision 2, 2026-10-03; FR-LBC-COL-011 to FR-LBC-COL-020 added; file-based requirements superseded)\
Owner: functional specification agent\
User decision/reference (revision 1): The user answered “Yes, manual verification is acceptable” to desktop verification, secure transfer to the Raspberry Pi, and repeating verification when rejected. This subject specifies reuse of that operator-supplied session within the existing collection behavior.\
User decision/reference (revision 2): user request and decisions 1–6 of 2026-10-03 for change [leboncoin-session-settings](../../changes/2026-10-03-1741-leboncoin-session-settings/index.md): store the session in the database, set it from the interface, keep automatic cookie renewal, show the last rejection, clear by saving an empty value, and remove the file-based session (`LEBONCOIN_SESSION_FILE`).

## Revision 2 summary

The session source changes from an operator-installed protected file (with a collector-owned sidecar) to a value stored in the application's database and edited through the [settings modal](../leboncoin-session-settings/functional.md). Requirements that depend on the file are marked **superseded** below and kept for history; the [deprecation record](../../deprecated/leboncoin-session-file.md) describes the removed behavior. FR-LBC-COL-007 and FR-LBC-COL-008 remain in force unchanged. The revision 1 purpose, scope, and corner cases below are retained as history; the revision 2 sections take precedence where they differ.

## Purpose and scope (revision 1, history)

Allow PriceFollower's existing Go collector to reuse a privately installed, manually verified LeBoncoin session for immediate, scheduled, and manual collection attempts. Preserve the current item API, interface, price interpretation, schedule, and observation retention.

Actors are the operator, the running backend, and LeBoncoin. Session-assisted operation requires an operator-configured protected session file produced by the [capture workflow](../leboncoin-session-capture/functional.md). Deployments that do not opt in continue to work as before. The collector continues running on the existing Raspberry Pi deployment without a browser daemon.

Excluded: a new session-management API or UI, automatic CAPTCHA solving, unattended establishment of a verified session, account authentication, altered collection frequency, new retry policies, and guarantees of remote acceptance or session duration.

## Requirements (revision 1)

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-LBC-COL-001 — **superseded** by FR-LBC-COL-011 | No session file is configured. | Preserve existing sessionless collection behavior for every platform. | Startup requires no browser or session file. LeBoncoin attempts use the existing path and existing error outcomes; Amazon behavior is unchanged. |
| FR-LBC-COL-002 — **superseded** by FR-LBC-COL-012 | A protected valid session file is configured. | Use only the applicable LeBoncoin `datadome` session for supported LeBoncoin collection requests. | Immediate, scheduled, and manual attempts can reuse the session. The secret is not sent to Amazon or unrelated hosts, including redirect destinations. No account cookies are imported. |
| FR-LBC-COL-003 — **superseded** by FR-LBC-COL-020 | The configured file is missing, malformed, expired, unreadable, or insufficiently protected. | Reject unusable session material safely and make the configuration/renewal problem understandable to the operator without exposing its contents. Keep the backend available. | Unsafe cookie data is never sent. A collection that cannot proceed records the existing `request_error` outcome; previous observations and unrelated collection continue to function. Diagnostics identify the condition without printing secrets. |
| FR-LBC-COL-004 — **superseded** by FR-LBC-COL-014, FR-LBC-COL-015 | A legitimate response updates or expires the LeBoncoin session cookie. | Retain the applicable update safely for later requests and across backend restarts; preserve private file protection. | A later attempt and a restart use the updated session state. A removed/expired cookie is not silently resurrected from an older saved value. Unrelated response cookies are not persisted. |
| FR-LBC-COL-005 — **superseded** by FR-LBC-COL-013 | The operator installs a renewed valid session file while the backend is running. | Load the replacement for subsequent collection attempts without requiring a server restart. | After safe replacement, a new attempt uses the replacement session. An earlier in-flight attempt does not overwrite the operator's newer session with stale state. |
| FR-LBC-COL-006 — **superseded** by FR-LBC-COL-016 | Session state cannot be saved. | Report the persistence problem safely and leave the installed session file intact; do not invent or discard a successfully parsed price because of a session-file write problem. | The operator receives a sanitized diagnostic. A complete previous export remains readable, and a successful listing response can still create its normal observation. No truncated session file is installed. |
| FR-LBC-COL-007 | LeBoncoin accepts a session-assisted request. | Apply the existing listing-identity, availability, and price-validation rules before recording success. | Only a valid response for the requested listing creates its normal observation. A cookie is not proof of a correct listing or price, and collection retains existing euro-cent semantics. |
| FR-LBC-COL-008 | LeBoncoin returns a challenge, 403, or another existing request-error condition. | Record `request_error`, preserve the latest successful price and all prior observations, and continue existing retries. | The listing is not marked free or unavailable merely because verification is requested. Existing API and UI show the established collection error/stale-price behavior. Operator guidance points to manual renewal where appropriate. |
| FR-LBC-COL-009 — **superseded** by FR-LBC-COL-018 | Collection, restart, or operator troubleshooting occurs. | Keep session secrets out of application responses, logs, repository content, and diagnostic evidence. | Existing API responses expose no session-file contents or cookie values. Diagnostics can explain load, rejection, renewal, or persistence failure without disclosing the secret. |
| FR-LBC-COL-010 — **superseded** by FR-LBC-COL-019 | A desktop session is imported on the Raspberry Pi. | Treat a successful server response as the evidence of usable collection; do not treat import alone as proof of acceptance. | The operator can check an attempt using existing controls. A cross-network rejection remains a normal collection failure with documented manual renewal; neither capture nor import claims guaranteed remote success. |

## States and corner cases (revision 1, history)

The existing application states remain pending collection, successful observation, unavailable listing, price not found, and request error according to the approved contract. Session capture/import does not create new client-visible fields or status values. Last successful prices remain available when session-assisted attempts fail.

Session-file absence when no file is configured is ordinary operation; a configured but unusable file is an operator-visible configuration/renewal problem. A renewal takes effect on a subsequent attempt, not retroactively on an already completed request. Concurrent attempts and file replacement must not corrupt the session file or restore superseded session data. Session updates do not change SQLite price history or scheduling rules.

## Purpose and scope (revision 2)

Allow the existing Go collector to reuse an operator-supplied LeBoncoin `datadome` session value stored in the application's database for immediate, scheduled, and manual LeBoncoin collection attempts; keep that value renewed when LeBoncoin legitimately renews it; and record the outcome of session-assisted attempts so the settings modal can warn the operator. Preserve the existing item API, interface, price interpretation, schedule, and observation retention.

Actors: the operator (through the settings modal), the running backend, and LeBoncoin. No session file, environment variable, browser on the server, or restart is involved. Excluded: automatic challenge solving, unattended establishment of a session, account authentication, altered collection frequency, new retry policies, automatic fallback between sessions, and guarantees of acceptance or session lifetime.

## Requirements (revision 2)

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-LBC-COL-011 | No session is saved (never set, or cleared by saving an empty value, FR-LBC-SET-008). | Use the existing sessionless collection behavior for every platform. | Startup requires no file, environment variable or browser. LeBoncoin attempts send no session cookie and keep their existing outcomes; Amazon behavior is unchanged. |
| FR-LBC-COL-012 | A usable session value is saved. | Send only that `datadome` session, and only on supported LeBoncoin collection requests to LeBoncoin's own hosts. | Immediate, scheduled and manual LeBoncoin attempts send the saved value. It is never sent to Amazon, to a non-LeBoncoin redirect destination, or to any unrelated host. No other cookie is added. |
| FR-LBC-COL-013 | The operator saves or clears the session while the backend runs. | Apply the new state to attempts that start after the save, without a restart. An attempt that started earlier must not overwrite or resurrect state the operator saved later. | After a save, the next attempt uses the new value; after a clear, the next attempt is sessionless. A renewal received by an in-flight attempt that began with the older value does not replace the operator's newer value and does not restore a cleared session. |
| FR-LBC-COL-014 | A LeBoncoin response that passes the existing listing verification (the requested listing was returned) renews the `datadome` cookie (user decision 3). | Store the renewed value in the database in place of the saved value, for later attempts, after restarts, and for display in the settings modal (FR-LBC-SET-016). | After such a response, the next attempt and an attempt after restart send the renewed value, and the modal shows it. A cookie set on a challenge, HTTP 403, wrong listing or other unverified response does not replace a saved value. Unrelated response cookies are never stored. |
| FR-LBC-COL-015 | LeBoncoin explicitly deletes the saved cookie, or a known expiry of the saved value has passed. | Stop sending the value; LeBoncoin attempts record the existing `request_error` outcome with renewal guidance and send no stale value. The expired/revoked state is shown in the settings modal (FR-LBC-SET-015) until the operator saves a new value or an empty value. | After a deletion or expiry, no later attempt (including after a restart) sends the old value; attempts record `request_error`; the previous successful prices remain. Saving a new value resumes session-assisted collection; saving an empty value resumes sessionless collection. A value pasted by the operator has no known expiry unless LeBoncoin later supplies one. |
| FR-LBC-COL-016 | A renewed value cannot be stored (database write failure). | Keep the parsed collection result, keep the previously stored session intact, and report the problem in a sanitized server diagnostic. | The successful listing response still creates its normal observation. The stored session is not emptied or corrupted. The diagnostic contains no cookie value. |
| FR-LBC-COL-017 | A session-assisted attempt finishes. | Record, for the currently saved value, the time and outcome of its latest session-assisted attempt: accepted (LeBoncoin returned the requested listing), rejected (verification challenge or HTTP 403), or failed for another reason. Recording does not change item outcomes. | The settings modal can show FR-LBC-SET-014 from this record. A record made with an older value is not shown for a newly saved value. Sessionless attempts are not reported as session rejections. |
| FR-LBC-COL-018 | Collection, restart or troubleshooting occurs. | Keep the session value out of item API responses, collection error messages, logs and diagnostics. The settings modal is the only place it is shown. | Item list/detail responses and recorded attempt messages contain no session value. Logs and diagnostics describe rejection, expiry, storage failure and similar conditions without the value. |
| FR-LBC-COL-019 | A session is saved from the interface. | Treat a successful LeBoncoin response, not the save itself, as evidence that the session works; acceptance from the server's device/network and session lifetime are not guaranteed. | The operator checks the result with existing refresh controls and the settings hint. A rejection is a normal collection failure with the documented renewal procedure; nothing claims guaranteed acceptance. |
| FR-LBC-COL-020 | The saved session cannot be read from the database when an attempt starts. | Do not proceed with an unknown session state: record `request_error`, send no cookie, keep the backend and other collection running, and report a sanitized diagnostic. | The LeBoncoin attempt records `request_error`; previous observations remain; Amazon collection continues; no value appears in the diagnostic. |

FR-LBC-COL-007 (existing listing, availability and price validation) and FR-LBC-COL-008 (challenge/403 recorded as `request_error`, prices retained, retries continue) apply unchanged to session-assisted attempts in revision 2; for FR-LBC-COL-008, the operator renewal guidance now points to the settings modal.

## States and corner cases (revision 2)

- Item states remain pending, successful observation, unavailable listing, price not found and request error; no new item-visible field or status is introduced by the session.
- Session states seen by the operator (through the settings modal): none saved; saved and unused; saved and last accepted; saved and last rejected/failed; expired or revoked.
- Concurrent attempts may receive renewals: the stored value must always be one complete value received from LeBoncoin or saved by the operator, never a mix or a truncated value. The newest legitimate renewal for the current operator-saved value wins; an operator save always takes precedence over renewals from attempts started before it (FR-LBC-COL-013).
- Saving or renewing the session never changes price history, observations or scheduling.
- Data already stored in a session file by revision 1 is not imported (see [file-based session removal](../leboncoin-session-file-removal/functional.md), FR-LBC-RM-002).

## Open questions

None for this subject. Revision 1: none. Revision 2: none; the stale-modal question Q-SET-1 was resolved by the user (“Warn and reload”, FR-LBC-SET-017) and does not change the collector rules above.

## Traceability

- [Existing functional specifications](../../FUNCTIONAL_SPECIFICATIONS.md), section 6 and FR-04/FR-07/FR-14/FR-18.
- [Canonical API contract](../../../API_SPECIFICATION.md): preserve existing collection and error semantics.
- [Investigation](../../changes/2026-10-03-0854-leboncoin-403-investigation/report.md): verified local cookie reuse succeeded, remote portability remains unproven.
- [Capture subject](../leboncoin-session-capture/functional.md).
- Technical handoff target: `doc/specifications/leboncoin-session-collection/technical.md`.
- Revision 2: [LeBoncoin session settings](../leboncoin-session-settings/functional.md), [file-based session removal](../leboncoin-session-file-removal/functional.md), [deprecation record](../../deprecated/leboncoin-session-file.md), change [leboncoin-session-settings](../../changes/2026-10-03-1741-leboncoin-session-settings/index.md).
